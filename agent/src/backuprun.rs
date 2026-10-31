//! Runs one backup: re-reads the project's config with the same checks as
//! for commands, then runs the backup child (backupchild.rs) as the config
//! file's owner, killing it after the timeout. See "Before backing up" in
//! docs/PROTOCOL.md.

use std::path::{Path, PathBuf};
use std::process::Stdio;
use std::time::Duration;

use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::time::timeout;

use crate::backupchild::{self, Done, Job};
use crate::commands::{Group, RunAs, owner_command};
use crate::project::ProjectConfig;

/// Stderr kept for the error message.
const MAX_ERROR: usize = 1000;

/// Re-reads the config of a project from the latest search and checks the
/// backup can run as its owner. An error means the backup is rejected.
pub async fn prepare(known: bool, project: &Path) -> Result<(Job, RunAs), String> {
    if !known {
        return Err("not a project found by this agent".into());
    }
    let dir = project.to_path_buf();
    let loaded = tokio::task::spawn_blocking(move || ProjectConfig::load_for_run(&dir)).await;
    let (config, uid) = loaded
        .map_err(|e| e.to_string())?
        .map_err(|e| format!("{e:#}"))?;
    let backups = config
        .backups
        .ok_or("config.vpspilot.json declares no backups")?;
    let job = Job {
        project: PathBuf::from(&config.path),
        name: config.name,
        agent_version: crate::VERSION.into(),
        backups,
    };
    Ok((job, RunAs::owner(uid)?))
}

/// Runs the backup child for `job` as `who`. Its whole process group,
/// dump tools included, is killed after `limit`.
pub async fn run(exe: &Path, job: &Job, who: &RunAs, limit: Duration) -> Result<Done, String> {
    let input = serde_json::to_vec(job).map_err(|e| e.to_string())?;
    let mut child = owner_command(exe, &job.project, who)
        .arg(backupchild::ARG)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .kill_on_drop(true)
        .spawn()
        .map_err(|e| format!("start the backup: {e}"))?;
    let mut group = Group::new(&child);
    let (mut stdin, mut stdout, mut stderr) = (
        child.stdin.take().unwrap(),
        child.stdout.take().unwrap(),
        child.stderr.take().unwrap(),
    );
    let io = async {
        stdin.write_all(&input).await?;
        drop(stdin);
        let (mut out, mut err) = (Vec::new(), Vec::new());
        tokio::try_join!(stdout.read_to_end(&mut out), stderr.read_to_end(&mut err))?;
        let status = child.wait().await?;
        Ok::<_, std::io::Error>((status, out, err))
    };
    let (status, out, err) = match timeout(limit, io).await {
        Ok(result) => result.map_err(|e| format!("backup: {e}"))?,
        Err(_) => {
            group.kill(&mut child).await;
            return Err(format!("the backup took longer than {}", minutes(limit)));
        }
    };
    group.disarm();
    if !status.success() {
        let err = String::from_utf8_lossy(&err);
        let err = err.trim();
        return Err(match err.char_indices().nth(MAX_ERROR) {
            _ if err.is_empty() => format!("the backup process failed ({status})"),
            Some((i, _)) => format!("{}…", &err[..i]),
            None => err.to_string(),
        });
    }
    serde_json::from_slice(&out).map_err(|e| format!("backup process: {e}"))
}

fn minutes(d: Duration) -> String {
    match d.as_secs() {
        s if s >= 60 && s % 60 == 0 => format!("{} minutes", s / 60),
        1 => "1 second".into(),
        s => format!("{s} seconds"),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    /// A fake agent binary: a shell script standing in for `__backup`.
    fn fake_exe(dir: &Path, script: &str) -> PathBuf {
        use std::os::unix::fs::PermissionsExt;
        let exe = dir.join("agent");
        fs::write(&exe, format!("#!/bin/sh\n{script}\n")).unwrap();
        fs::set_permissions(&exe, fs::Permissions::from_mode(0o755)).unwrap();
        exe
    }

    async fn job(dir: &Path) -> (Job, RunAs) {
        use std::os::unix::fs::PermissionsExt;
        let config = dir.join("config.vpspilot.json");
        fs::write(
            &config,
            r#"{"name": "shop", "backups": {"local_dir": "b"}}"#,
        )
        .unwrap();
        fs::set_permissions(&config, fs::Permissions::from_mode(0o644)).unwrap();
        prepare(true, dir).await.unwrap()
    }

    #[tokio::test]
    async fn prepares_from_the_config() {
        let dir = tempfile::tempdir().unwrap();
        assert_eq!(
            prepare(false, dir.path()).await.unwrap_err(),
            "not a project found by this agent"
        );
        let config = dir.path().join("config.vpspilot.json");
        fs::write(&config, r#"{"name": "shop"}"#).unwrap();
        use std::os::unix::fs::PermissionsExt;
        fs::set_permissions(&config, fs::Permissions::from_mode(0o644)).unwrap();
        let err = prepare(true, dir.path()).await.unwrap_err();
        assert_eq!(err, "config.vpspilot.json declares no backups");
        let (job, _) = job(dir.path()).await;
        assert_eq!(
            (job.name.as_str(), job.backups.local_dir.as_str()),
            ("shop", "b")
        );
    }

    #[tokio::test]
    async fn runs_the_child_and_reports_failures() {
        let dir = tempfile::tempdir().unwrap();
        let (job, who) = job(dir.path()).await;
        let limit = Duration::from_secs(10);

        // The child gets the job on stdin and answers with Done.
        let done = r#"{"file":"f.zip","local_path":"/b/f.zip","size":3,"database":"","upload":{"status":"skipped","key":"","error":""}}"#;
        let ok = fake_exe(
            dir.path(),
            &format!("grep -q '\"name\":\"shop\"' && echo '{done}'"),
        );
        assert_eq!(run(&ok, &job, &who, limit).await.unwrap().size, 3);

        let fails = fake_exe(
            dir.path(),
            "cat >/dev/null; echo 'mysqldump not found' >&2; exit 1",
        );
        assert_eq!(
            run(&fails, &job, &who, limit).await.unwrap_err(),
            "mysqldump not found"
        );

        // A grandchild sleeping in the same process group is killed too.
        let slow = fake_exe(dir.path(), "sleep 30 & wait");
        let started = std::time::Instant::now();
        let err = run(&slow, &job, &who, Duration::from_secs(1))
            .await
            .unwrap_err();
        assert_eq!(err, "the backup took longer than 1 second");
        assert!(
            started.elapsed() < Duration::from_secs(10),
            "waited for the grandchild"
        );
        assert_eq!(minutes(Duration::from_secs(3600)), "60 minutes");
    }
}
