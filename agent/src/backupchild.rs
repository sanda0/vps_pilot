//! The backup child process: the agent runs itself as `vps-pilot-agent
//! __backup` as a config file's owner, so a backup only reads and writes
//! what that user can. It reads a `Job` as JSON on stdin and writes `Done`
//! as JSON to stdout, or the reason it failed to stderr. See "Backing up"
//! in docs/PROTOCOL.md.

use std::fs::{self, DirBuilder};
use std::io::{self, Read};
use std::os::unix::fs::DirBuilderExt;
use std::path::{Path, PathBuf};
use std::process::ExitCode;
use std::time::{SystemTime, UNIX_EPOCH};

use serde::{Deserialize, Serialize};

use crate::backupzip::{self, Contents};
use crate::dump::Database;
use crate::envfile::EnvFile;
use crate::project::ProjectBackup;

/// First argument that makes the agent binary act as the backup child.
pub const ARG: &str = "__backup";

#[derive(Debug, Serialize, Deserialize)]
pub struct Job {
    /// The project directory.
    pub project: PathBuf,
    /// The project's `name`.
    pub name: String,
    pub agent_version: String,
    pub backups: ProjectBackup,
}

/// A backup that succeeded.
#[derive(Debug, Serialize, Deserialize)]
pub struct Done {
    pub file: String,
    pub local_path: String,
    pub size: u64,
    /// The dump type, or empty without `database`.
    pub database: String,
}

/// Entry point when the first argument is ARG.
pub fn main() -> ExitCode {
    let mut input = String::new();
    let result = io::stdin()
        .take(1024 * 1024)
        .read_to_string(&mut input)
        .map_err(|e| e.to_string())
        .and_then(|_| serde_json::from_str::<Job>(&input).map_err(|e| e.to_string()))
        .and_then(|job| backup(&job, now()));
    match result.and_then(|done| serde_json::to_string(&done).map_err(|e| e.to_string())) {
        Ok(json) => {
            println!("{json}");
            ExitCode::SUCCESS
        }
        Err(e) => {
            eprintln!("{e}");
            ExitCode::FAILURE
        }
    }
}

fn now() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_or(0, |d| d.as_secs())
}

pub fn backup(job: &Job, now: u64) -> Result<Done, String> {
    let b = &job.backups;
    let project = fs::canonicalize(&job.project).map_err(|e| format!("project directory: {e}"))?;
    let base = b.zip_base(&job.name).map_err(|e| e.to_string())?;
    let dirs = b
        .dirs
        .iter()
        .map(|d| backupzip::inside(&project, d))
        .collect::<Result<Vec<_>, _>>()?;
    let local = local_dir(&project, &b.local_dir, &dirs)?;

    let mut roots = dirs;
    let env = if b.env_file.is_empty() {
        None
    } else {
        let path = backupzip::inside(&project, &b.env_file)?;
        let env = EnvFile::load(&path, &b.env_file)?;
        roots.push(path);
        Some(env)
    };
    let database = match (&b.database, &env) {
        (None, _) => None,
        (Some(_), None) => return Err("backups.database needs backups.env_file".into()),
        (Some(names), Some(env)) => Some(Database::from_env(names, env, &project)?),
    };

    remove_leftovers(&local, &base);
    let stamp = backupzip::stamp(now);
    let tmp = Tmp(local.join(format!(".{base}-{stamp}.tmp")));
    DirBuilder::new()
        .mode(0o700)
        .create(&tmp.0)
        .map_err(|e| format!("create {}: {e}", tmp.0.display()))?;
    let dump = match &database {
        Some(db) => Some((db.dump(&tmp.0)?, db.file_name())),
        None => None,
    };

    let file = format!("{base}-{stamp}.zip");
    let part = local.join(format!("{file}.part"));
    let contents = Contents {
        project: &project,
        roots,
        database: dump,
        manifest: serde_json::json!({
            "project": job.name,
            "created_at": now,
            "agent_version": job.agent_version,
            "database": database.as_ref().map_or("", Database::kind),
        }),
    };
    let size = backupzip::write(&part, &contents).inspect_err(|_| {
        let _ = fs::remove_file(&part);
    })?;
    let path = local.join(&file);
    // A hard link, unlike rename, never replaces a zip that already exists.
    let linked = fs::hard_link(&part, &path).map_err(|e| format!("create {}: {e}", path.display()));
    let _ = fs::remove_file(&part);
    linked?;
    prune(&local, &base, now, b.retention_days());
    Ok(Done {
        file,
        local_path: path.to_string_lossy().into_owned(),
        size,
        database: database.as_ref().map_or("", Database::kind).into(),
    })
}

/// Resolves `local_dir`, creating it with mode 0700, and checks it isn't
/// inside a folder that is backed up.
fn local_dir(project: &Path, local_dir: &str, dirs: &[PathBuf]) -> Result<PathBuf, String> {
    if local_dir.is_empty() {
        return Err("backups.local_dir is not set".into());
    }
    let path = project.join(local_dir);
    DirBuilder::new()
        .recursive(true)
        .mode(0o700)
        .create(&path)
        .map_err(|e| format!("create {local_dir}: {e}"))?;
    let real = fs::canonicalize(&path).map_err(|e| format!("{local_dir}: {e}"))?;
    if dirs.iter().any(|d| real.starts_with(d)) {
        return Err(format!(
            "backups.local_dir {local_dir} is inside a backed-up folder"
        ));
    }
    Ok(real)
}

/// The zip named `<base>-<stamp>.zip`, as its time.
fn zip_time(name: &str, base: &str) -> Option<u64> {
    let stamp = name
        .strip_prefix(base)?
        .strip_prefix('-')?
        .strip_suffix(".zip")?;
    backupzip::parse_stamp(stamp)
}

/// Removes what a killed backup left behind: `<base>-<stamp>.zip.part`
/// files and `.<base>-<stamp>.tmp` folders.
fn remove_leftovers(local: &Path, base: &str) {
    for name in names(local) {
        if let Some(zip) = name.strip_suffix(".part")
            && zip_time(zip, base).is_some()
        {
            let _ = fs::remove_file(local.join(&name));
        } else if let Some(dir) = name.strip_prefix('.').and_then(|n| n.strip_suffix(".tmp"))
            && zip_time(&format!("{dir}.zip"), base).is_some()
        {
            let _ = fs::remove_dir_all(local.join(&name));
        }
    }
}

/// Deletes this project's zips older than `days`, except the newest.
fn prune(local: &Path, base: &str, now: u64, days: u32) {
    let mut zips: Vec<(u64, String)> = names(local)
        .into_iter()
        .filter_map(|n| Some((zip_time(&n, base)?, n)))
        .collect();
    zips.sort();
    zips.pop();
    let cutoff = now.saturating_sub(u64::from(days) * 86_400);
    for (time, name) in zips {
        if time < cutoff {
            let _ = fs::remove_file(local.join(name));
        }
    }
}

fn names(dir: &Path) -> Vec<String> {
    fs::read_dir(dir)
        .map(|d| {
            d.filter_map(|e| e.ok()?.file_name().into_string().ok())
                .collect()
        })
        .unwrap_or_default()
}

/// A private folder for the dump, removed when the backup ends.
struct Tmp(PathBuf);

impl Drop for Tmp {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::MetadataExt;

    const DAY: u64 = 86_400;

    fn project() -> (tempfile::TempDir, Job) {
        let dir = tempfile::tempdir().unwrap();
        fs::create_dir_all(dir.path().join("shop/storage/app")).unwrap();
        fs::write(dir.path().join("shop/storage/app/a.txt"), "a").unwrap();
        fs::write(dir.path().join("shop/.env"), "APP_KEY=x\n").unwrap();
        let backups: ProjectBackup = serde_json::from_value(serde_json::json!({
            "env_file": ".env", "dir": ["storage/app"], "local_dir": "../backups/shop",
            "retention_days": 2
        }))
        .unwrap();
        let job = Job {
            project: dir.path().join("shop"),
            name: "my shop".into(),
            agent_version: "v0.5.0".into(),
            backups,
        };
        (dir, job)
    }

    #[test]
    fn writes_the_zip_and_prunes_old_ones() {
        let (dir, job) = project();
        let local = dir.path().join("backups/shop");
        let start = 1_793_415_600;
        let first = backup(&job, start).unwrap();
        assert_eq!(first.file, "my_shop-20261031T030000Z.zip");
        assert_eq!(first.database, "");
        let meta = fs::metadata(&first.local_path).unwrap();
        assert_eq!((meta.len(), meta.mode() & 0o777), (first.size, 0o600));
        assert_eq!(fs::metadata(&local).unwrap().mode() & 0o777, 0o700);
        assert!(backup(&job, start).is_err(), "overwrote a zip");

        // Another project's zip and a leftover from a killed backup.
        fs::write(local.join("blog-20200101T000000Z.zip"), "").unwrap();
        fs::write(local.join("my_shop-20261101T000000Z.zip.part"), "").unwrap();
        fs::create_dir(local.join(".my_shop-20261101T000000Z.tmp")).unwrap();

        backup(&job, start + DAY).unwrap();
        let last = backup(&job, start + 3 * DAY).unwrap();
        let mut left = names(&local);
        left.sort();
        assert_eq!(
            left,
            [
                "blog-20200101T000000Z.zip",
                "my_shop-20261101T030000Z.zip",
                last.file.as_str()
            ]
        );
    }

    #[test]
    fn checks_paths_and_settings() {
        for (change, want) in [
            (
                serde_json::json!({"local_dir": ""}),
                "backups.local_dir is not set",
            ),
            (
                serde_json::json!({"local_dir": "storage/app/backups"}),
                "inside a backed-up folder",
            ),
            (
                serde_json::json!({"dir": ["../"]}),
                "outside the project directory",
            ),
            (serde_json::json!({"zip_file_name": "a b"}), "zip_file_name"),
            (
                serde_json::json!({"env_file": "", "database": {"connection": "DB_CONNECTION"}}),
                "needs backups.env_file",
            ),
        ] {
            let (_dir, mut job) = project();
            let mut b = serde_json::to_value(&job.backups).unwrap();
            for (k, v) in change.as_object().unwrap() {
                b[k] = v.clone();
            }
            job.backups = serde_json::from_value(b).unwrap();
            let err = backup(&job, 0).unwrap_err();
            assert!(err.contains(want), "{change}: {err}");
        }
    }
}
