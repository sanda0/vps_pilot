//! Cron jobs, run by the agent itself at the minutes their schedules
//! match. This task outlives sessions, so jobs run and their results queue
//! while the hub is unreachable. See "Cron jobs" in docs/PROTOCOL.md.

// The scheduler that calls this comes in the next change.
#![allow(dead_code)]

use std::path::Path;
use std::sync::Arc;

use tracing::info;

use crate::commands::{self, Limits, Output, RunAs};
use crate::project::{ProjectConfig, ProjectCronJob};
use crate::protocol::{CommandExit, CronResult, ExitStatus};
use crate::slots::Slots;

/// Results kept while no session takes them.
const QUEUE: usize = 50;
/// Output kept per run: the last 64 KiB.
const MAX_OUTPUT: usize = 64 * 1024;

/// Runs one job, after the checks in "Running" under "Cron jobs".
async fn execute(
    path: &Path,
    job: &ProjectCronJob,
    scheduled_at: u64,
    slots: &Slots,
    limits: &Limits,
) -> CronResult {
    let run_id = crate::backups::new_id();
    let (exit, output) = match slots.take(path, &job.command) {
        Err(reason) => (
            CommandExit::not_started(&run_id, ExitStatus::Skipped, reason),
            String::new(),
        ),
        Ok(_slot) => match prepare(path, job).await {
            Err(reason) => (
                CommandExit::not_started(&run_id, ExitStatus::Rejected, reason),
                String::new(),
            ),
            Ok((cmd, who)) => {
                info!(project = %path.display(), command = job.command, user = who.name, "running cron job");
                commands::run(
                    &run_id,
                    path,
                    &cmd,
                    who,
                    limits,
                    Arc::default(),
                    Output::Tail,
                )
                .await
            }
        },
    };
    CronResult {
        run_id,
        project_path: path.to_string_lossy().into_owned(),
        command: job.command.clone(),
        schedule: job.schedule.clone(),
        scheduled_at,
        status: exit.status,
        exit_code: exit.exit_code,
        run_as: exit.run_as,
        duration_ms: exit.duration_ms,
        output,
        truncated: exit.truncated,
        error: exit.error,
    }
}

/// Re-reads the config: the job must still be in `cron` and its command
/// in `commands`. Returns the command string and who runs it.
async fn prepare(path: &Path, job: &ProjectCronJob) -> Result<(String, RunAs), String> {
    let dir = path.to_path_buf();
    let (config, uid) = tokio::task::spawn_blocking(move || ProjectConfig::load_for_run(&dir))
        .await
        .map_err(|e| e.to_string())?
        .map_err(|e| format!("{e:#}"))?;
    if !config.cron.iter().any(|j| j.command == job.command) {
        return Err(format!("{:?} is no longer in cron", job.command));
    }
    let cmd = config
        .command(&job.command)
        .ok_or_else(|| format!("unknown command {:?}", job.command))?;
    Ok((cmd.to_string(), RunAs::owner(uid)?))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use std::os::unix::fs::PermissionsExt;
    use std::path::PathBuf;
    use std::time::Duration;

    fn project(dir: &Path, cron: &str) -> ProjectConfig {
        let json = format!(
            r#"{{"name": "app", "cron": {cron}, "commands": [
                {{"name": "hello", "command": "echo hi; echo oops >&2; exit 2"}}]}}"#
        );
        let file = dir.join(crate::project::CONFIG_FILE);
        fs::write(&file, json).unwrap();
        fs::set_permissions(&file, fs::Permissions::from_mode(0o644)).unwrap();
        ProjectConfig::load_for_run(dir).unwrap().0
    }

    #[tokio::test]
    async fn runs_skips_and_rejects() {
        let dir = tempfile::tempdir().unwrap();
        let p = project(
            dir.path(),
            r#"[{"command": "hello", "schedule": "@daily"}]"#,
        );
        let (path, job) = (PathBuf::from(&p.path), p.cron[0].clone());
        let limits = Limits {
            timeout: Duration::from_secs(10),
            max_output: MAX_OUTPUT,
        };
        let slots = Slots::default();

        let r = execute(&path, &job, 60, &slots, &limits).await;
        assert_eq!((r.status, r.exit_code), (ExitStatus::Exited, Some(2)));
        assert_eq!((r.output.as_str(), r.scheduled_at), ("hi\noops\n", 60));
        assert_eq!((r.schedule.as_str(), r.run_id.len()), ("@daily", 36));

        let busy = slots.take(&path, "hello").unwrap();
        let r = execute(&path, &job, 60, &slots, &limits).await;
        assert_eq!(r.status, ExitStatus::Skipped);
        assert_eq!(r.error, "\"hello\" is already running");
        drop(busy);

        project(dir.path(), "[]");
        let r = execute(&path, &job, 60, &slots, &limits).await;
        assert_eq!(r.status, ExitStatus::Rejected);
        assert!(r.error.contains("no longer in cron"), "{}", r.error);
    }
}
