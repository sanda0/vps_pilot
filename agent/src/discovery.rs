//! Finds `config.vpspilot.json` files under the project directories.

use std::fs;
use std::io::ErrorKind;
use std::path::{Path, PathBuf};

use tracing::{debug, warn};

use crate::project::{CONFIG_FILE, ProjectConfig};

/// How many directory levels below each root are searched
/// (/home/<user>/apps/<client>/<project> is 4).
const MAX_DEPTH: usize = 4;

/// Dependency and build directories that never hold a project of their own.
/// Hidden directories (.git, .cache, ...) are skipped too.
const SKIP_DIRS: &[&str] = &["node_modules", "vendor", "target"];

/// Loads every valid config under `roots`, sorted by path. Invalid configs
/// are logged and skipped, as are roots that don't exist. Symlinked
/// directories are not followed.
pub fn discover(roots: &[PathBuf]) -> Vec<ProjectConfig> {
    let mut found = Vec::new();
    for root in roots {
        walk(root, 0, &mut found);
    }
    found.sort_by(|a, b| a.path.cmp(&b.path));
    // Overlapping roots (e.g. /home and /home/app) find the same project twice.
    found.dedup_by(|a, b| a.path == b.path);
    found
}

fn walk(dir: &Path, depth: usize, found: &mut Vec<ProjectConfig>) {
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

    let file = dir.join(CONFIG_FILE);
    if file.is_file() {
        match ProjectConfig::load(&file) {
            Ok(config) => found.push(config),
            Err(e) => warn!("skipping project: {e:#}"),
        }
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

        let found = discover(&[root.to_path_buf()]);
        assert_eq!(names(&found), ["root", "a", "nested", "deepest"]);
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
        assert_eq!(names(&discover(&roots)), ["a"]);
    }

    #[cfg(unix)]
    #[test]
    fn does_not_follow_symlinked_dirs() {
        let tmp = tempfile::tempdir().unwrap();
        let outside = tempfile::tempdir().unwrap();
        add_project(&outside.path().join("p"), "outside");
        std::os::unix::fs::symlink(outside.path(), tmp.path().join("link")).unwrap();
        assert!(discover(&[tmp.path().to_path_buf()]).is_empty());
    }
}
