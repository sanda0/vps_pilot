//! Which files the dashboard may read: the files in a project's declared
//! log folders. Runs in the log child process, as the config file's owner.
//! See "Which files" in docs/PROTOCOL.md.

use std::fs;
use std::path::{Path, PathBuf};
use std::time::UNIX_EPOCH;

use crate::protocol::LogFile;

/// Most files listed per project.
pub const MAX_FILES: usize = 1000;
const COMPRESSED: &[&str] = &["gz", "bz2", "xz", "zst", "zip"];

/// A declared log folder after resolving `..` and symlinks.
#[derive(Debug, Clone, PartialEq)]
pub struct Folder {
    real: PathBuf,
    /// Relative to the resolved project directory.
    rel: PathBuf,
}

/// Resolves the declared folders. Folders that don't exist, aren't
/// directories or resolve outside the project directory are skipped.
pub fn folders(project: &Path, logs: &[String]) -> Vec<Folder> {
    let Ok(project) = fs::canonicalize(project) else {
        return Vec::new();
    };
    let mut out: Vec<Folder> = Vec::new();
    for log in logs {
        let Ok(real) = fs::canonicalize(project.join(log)) else {
            continue;
        };
        if !real.starts_with(&project) || !real.is_dir() || out.iter().any(|f| f.real == real) {
            continue;
        }
        let rel = real
            .strip_prefix(&project)
            .unwrap_or(Path::new(""))
            .to_path_buf();
        out.push(Folder { real, rel });
    }
    out
}

/// The readable files in `folders`, sorted by path, and whether the list was
/// cut at MAX_FILES.
pub fn list(folders: &[Folder]) -> (Vec<LogFile>, bool) {
    let mut files = Vec::new();
    for folder in folders {
        for name in names(&folder.real) {
            let path = folder.real.join(&name);
            match fs::canonicalize(&path) {
                Ok(dir) if dir.is_dir() && inside(&dir, folder) => {
                    for nested in names(&dir) {
                        files.extend(log_file(&dir.join(&nested), folder, &name.join(&nested)));
                    }
                }
                Ok(_) => files.extend(log_file(&path, folder, &name)),
                Err(_) => {}
            }
        }
    }
    files.sort_by(|a, b| a.path.cmp(&b.path));
    files.dedup_by(|a, b| a.path == b.path);
    let truncated = files.len() > MAX_FILES;
    files.truncate(MAX_FILES);
    (files, truncated)
}

/// The resolved path of `file` (relative to the project, as `list` returns
/// it), if it is a readable log file.
pub fn resolve(project: &Path, folders: &[Folder], file: &str) -> Result<PathBuf, String> {
    let refused = || "not a file in this project's log folders".to_string();
    let rel = Path::new(file);
    if file.is_empty() || rel.is_absolute() {
        return Err(refused());
    }
    let project = fs::canonicalize(project).map_err(|_| refused())?;
    let real = fs::canonicalize(project.join(rel)).map_err(|_| refused())?;
    if !real.is_file() || !folders.iter().any(|f| inside(&real, f)) {
        return Err(refused());
    }
    if compressed(&real) {
        return Err("compressed log files can't be opened".into());
    }
    Ok(real)
}

/// Directly in the folder, or in a folder directly below it.
fn inside(real: &Path, folder: &Folder) -> bool {
    let parent = real.parent();
    parent == Some(&folder.real) || parent.and_then(Path::parent) == Some(&folder.real)
}

fn compressed(path: &Path) -> bool {
    let ext = path
        .extension()
        .and_then(|e| e.to_str())
        .unwrap_or_default();
    COMPRESSED.contains(&ext.to_ascii_lowercase().as_str())
}

fn names(dir: &Path) -> Vec<PathBuf> {
    let Ok(read) = fs::read_dir(dir) else {
        return Vec::new();
    };
    read.flatten()
        .map(|e| PathBuf::from(e.file_name()))
        .collect()
}

fn log_file(path: &Path, folder: &Folder, below: &Path) -> Option<LogFile> {
    let real = fs::canonicalize(path).ok()?;
    let meta = fs::metadata(&real).ok()?;
    if !meta.is_file() || !inside(&real, folder) {
        return None;
    }
    let modified = meta
        .modified()
        .ok()?
        .duration_since(UNIX_EPOCH)
        .ok()?
        .as_secs();
    Some(LogFile {
        path: folder.rel.join(below).to_string_lossy().into_owned(),
        size: meta.len(),
        modified: modified as i64,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::symlink;

    /// A project with storage/logs/{app.log, old.log.gz, 2026/day.log,
    /// 2026/deep/x.log}, a symlink out of it and a folder outside it.
    fn project() -> (tempfile::TempDir, tempfile::TempDir) {
        let dir = tempfile::tempdir().unwrap();
        let outside = tempfile::tempdir().unwrap();
        let logs = dir.path().join("storage/logs");
        fs::create_dir_all(logs.join("2026/deep")).unwrap();
        for f in ["app.log", "old.log.gz", "2026/day.log", "2026/deep/x.log"] {
            fs::write(logs.join(f), "line\n").unwrap();
        }
        fs::write(outside.path().join("secret"), "s").unwrap();
        symlink(outside.path().join("secret"), logs.join("secret.log")).unwrap();
        symlink(outside.path(), dir.path().join("elsewhere")).unwrap();
        (dir, outside)
    }

    fn declared(logs: &[&str]) -> Vec<String> {
        logs.iter().map(|s| s.to_string()).collect()
    }

    #[test]
    fn folders_must_resolve_inside_the_project() {
        let (dir, _outside) = project();
        let logs = declared(&[
            "storage/logs",
            "elsewhere",
            "missing",
            "/var/log",
            "storage/../storage/logs",
        ]);
        let got = folders(dir.path(), &logs);
        assert_eq!(got.len(), 1, "{got:?}");
        assert_eq!(got[0].rel, Path::new("storage/logs"));
    }

    #[test]
    fn lists_files_one_level_deep_inside_the_folder() {
        let (dir, _outside) = project();
        let (files, truncated) = list(&folders(dir.path(), &declared(&["storage/logs"])));
        let paths: Vec<_> = files.iter().map(|f| f.path.as_str()).collect();
        assert_eq!(
            paths,
            [
                "storage/logs/2026/day.log",
                "storage/logs/app.log",
                "storage/logs/old.log.gz"
            ]
        );
        assert!(!truncated && files[1].size == 5 && files[1].modified > 0);
    }

    #[test]
    fn resolves_only_listed_readable_files() {
        let (dir, _outside) = project();
        let f = folders(dir.path(), &declared(&["storage/logs"]));
        assert!(resolve(dir.path(), &f, "storage/logs/app.log").is_ok());
        assert!(resolve(dir.path(), &f, "storage/logs/2026/day.log").is_ok());
        for (bad, why) in [
            ("storage/logs/secret.log", "log folders"),
            ("storage/logs/2026/deep/x.log", "log folders"),
            ("storage/logs/../../elsewhere/secret", "log folders"),
            ("/etc/passwd", "log folders"),
            ("storage/logs", "log folders"),
            ("storage/logs/old.log.gz", "compressed"),
        ] {
            let err = resolve(dir.path(), &f, bad).unwrap_err();
            assert!(err.contains(why), "{bad}: {err}");
        }
    }
}
