//! Finds `config.vpspilot.json` files under the project directories.

use std::fs;
use std::io::ErrorKind;
use std::path::{Path, PathBuf};

use serde::Serialize;
use tracing::{debug, warn};

use crate::project::{CONFIG_FILE, ProjectConfig};

/// How many directory levels below each root are searched
/// (/home/<user>/apps/<client>/<project> is 4).
const MAX_DEPTH: usize = 4;

/// Dependency and build directories that never hold a project of their own.
/// Hidden directories (.git, .cache, ...) are skipped too.
const SKIP_DIRS: &[&str] = &["node_modules", "vendor", "target"];

/// A config that couldn't be used, or a rejected manual path, keyed by the
/// project directory. Reported to the hub in `projects`.
#[derive(Debug, Clone, PartialEq, Serialize)]
pub struct ProjectError {
    pub path: String,
    pub error: String,
}

#[derive(Debug, Default, PartialEq)]
pub struct Found {
    pub projects: Vec<ProjectConfig>,
    pub errors: Vec<ProjectError>,
}

/// Loads every config under `roots` plus the hub's manual `paths`, sorted by
/// path. Roots that don't exist are skipped and symlinked directories are not
/// followed. A manual path must resolve inside one of the roots unless
/// `allow_any_path` is set: the server owner decides where projects can live.
pub fn discover(roots: &[PathBuf], paths: &[PathBuf], allow_any_path: bool) -> Found {
    let mut found = Found::default();
    for root in roots {
        walk(root, 0, &mut found);
    }
    for path in paths {
        match manual_dir(path, roots, allow_any_path) {
            Ok(dir) => load(&dir, &mut found),
            Err(error) => found.errors.push(ProjectError {
                path: path.to_string_lossy().into_owned(),
                error,
            }),
        }
    }
    // Overlapping roots (e.g. /home and /home/app) and manual paths inside a
    // root find the same directory twice.
    found.projects.sort_by(|a, b| a.path.cmp(&b.path));
    found.projects.dedup_by(|a, b| a.path == b.path);
    found.errors.sort_by(|a, b| a.path.cmp(&b.path));
    found.errors.dedup_by(|a, b| a.path == b.path);
    found
}

/// Resolves a manual path and checks it's allowed.
fn manual_dir(path: &Path, roots: &[PathBuf], allow_any_path: bool) -> Result<PathBuf, String> {
    if !path.is_absolute() {
        return Err("must be an absolute path".into());
    }
    let dir = fs::canonicalize(path).map_err(|e| format!("can't open: {e}"))?;
    let inside = || {
        roots
            .iter()
            .filter_map(|root| fs::canonicalize(root).ok())
            .any(|root| dir.starts_with(root))
    };
    if !allow_any_path && !inside() {
        return Err("outside the agent's project directories (--projects-dir)".into());
    }
    Ok(dir)
}

fn load(dir: &Path, found: &mut Found) {
    match ProjectConfig::load(&dir.join(CONFIG_FILE)) {
        Ok(config) => found.projects.push(config),
        Err(e) => found.errors.push(ProjectError {
            path: dir.to_string_lossy().into_owned(),
            error: format!("{e:#}"),
        }),
    }
}

fn walk(dir: &Path, depth: usize, found: &mut Found) {
    let entries = match fs::read_dir(dir) {
        Ok(entries) => entries,
        Err(e) if depth == 0 && e.kind() != ErrorKind::NotFound => {
            warn!(dir = %dir.display(), "can't read project directory: {e}");
            return;
        }
        Err(e) => {
            debug!(dir = %dir.display(), "skipping: {e}");
            return;
        }
    };

    if dir.join(CONFIG_FILE).is_file() {
        load(dir, found);
    }
    if depth == MAX_DEPTH {
        return;
    }

    for entry in entries.flatten() {
        // file_type() does not follow symlinks.
        if !entry.file_type().is_ok_and(|t| t.is_dir()) {
            continue;
        }
        let name = entry.file_name();
        let name = name.to_string_lossy();
        if name.starts_with('.') || SKIP_DIRS.contains(&name.as_ref()) {
            continue;
        }
        walk(&entry.path(), depth + 1, found);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn add_project(dir: &Path, name: &str) {
        fs::create_dir_all(dir).unwrap();
        let json = format!(r#"{{"name": "{name}"}}"#);
        fs::write(dir.join(CONFIG_FILE), json).unwrap();
    }

    fn names(projects: &[ProjectConfig]) -> Vec<&str> {
        projects.iter().map(|p| p.name.as_str()).collect()
    }

    #[test]
    fn finds_projects_and_skips_the_rest() {
        let tmp = tempfile::tempdir().unwrap();
        let root = tmp.path();
        add_project(root, "root");
        add_project(&root.join("a"), "a");
        add_project(&root.join("a/nested"), "nested");
        add_project(&root.join("u/apps/client/b"), "deepest");
        add_project(&root.join("u/apps/client/b/too/deep"), "too deep");
        add_project(&root.join("app/node_modules/dep"), "dependency");
        add_project(&root.join(".cache/x"), "hidden");
        fs::create_dir_all(root.join("broken")).unwrap();
        fs::write(root.join("broken").join(CONFIG_FILE), "{").unwrap();

        let found = discover(&[root.to_path_buf()], &[], false);
        assert_eq!(names(&found.projects), ["root", "a", "nested", "deepest"]);
        assert_eq!(found.errors.len(), 1);
        assert!(found.errors[0].path.ends_with("broken"));
    }

    #[test]
    fn missing_roots_and_overlaps() {
        let tmp = tempfile::tempdir().unwrap();
        add_project(&tmp.path().join("a"), "a");
        let roots = [
            tmp.path().join("missing"),
            tmp.path().to_path_buf(),
            tmp.path().join("a"),
        ];
        assert_eq!(names(&discover(&roots, &[], false).projects), ["a"]);
    }

    #[cfg(unix)]
    #[test]
    fn does_not_follow_symlinked_dirs() {
        let tmp = tempfile::tempdir().unwrap();
        let outside = tempfile::tempdir().unwrap();
        add_project(&outside.path().join("p"), "outside");
        std::os::unix::fs::symlink(outside.path(), tmp.path().join("link")).unwrap();
        assert_eq!(
            discover(&[tmp.path().to_path_buf()], &[], false),
            Found::default()
        );
    }

    #[cfg(unix)]
    #[test]
    fn manual_paths_must_stay_inside_roots() {
        let tmp = tempfile::tempdir().unwrap();
        let (root, outside) = (tmp.path().join("root"), tmp.path().join("outside"));
        let deep = root.join("a/b/c/d/e/deep");
        add_project(&deep, "deep");
        add_project(&outside, "outside");
        std::os::unix::fs::symlink(&outside, root.join("link")).unwrap();
        let paths = [
            deep.clone(),
            outside.clone(),
            root.join("../outside"),
            root.join("link"),
            root.join("missing"),
            PathBuf::from("relative/dir"),
        ];

        let roots = [root];
        let found = discover(&roots, &paths, false);
        assert_eq!(names(&found.projects), ["deep"]);
        assert_eq!(found.errors.len(), 5, "{:?}", found.errors);

        let found = discover(&roots, &paths, true);
        assert_eq!(names(&found.projects), ["outside", "deep"]);
    }
}
