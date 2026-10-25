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
