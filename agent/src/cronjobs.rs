//! Cron jobs, run by the agent itself at the minutes their schedules
//! match. This task outlives sessions, so jobs run and their results queue
//! while the hub is unreachable. See "Cron jobs" in docs/PROTOCOL.md.

use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use tokio::sync::{mpsc, watch};
use tracing::{info, warn};

use crate::commands::{self, Limits, Output, RunAs};
use crate::cron::{self, CronSchedule, LocalMinute};
use crate::project::{ProjectConfig, ProjectCronJob};
use crate::protocol::{CommandExit, CronResult, ExitStatus, MSG_CRON_RESULT, Msg};
use crate::slots::Slots;

/// Results kept while no session takes them.
const QUEUE: usize = 50;
/// Output kept per run: the last 64 KiB.
const MAX_OUTPUT: usize = 64 * 1024;

#[allow(dead_code)] // Started from main, in the next change.
/// Starts the cron task. `projects` is the latest search; `slots` is
/// shared with Run. Results arrive on the receiver and wait there while no
/// session reads it.
pub fn spawn(
    timeout: Duration,
    slots: Slots,
    projects: watch::Receiver<Vec<ProjectConfig>>,
) -> mpsc::Receiver<Msg> {
    let (out, outbox) = mpsc::channel(QUEUE);
    let limits = Arc::new(Limits {
        timeout,
        max_output: MAX_OUTPUT,
    });
    tokio::spawn(async move {
        let mut last = None;
        loop {
            let wait = 60_000 - unix_now().as_millis() % 60_000;
            tokio::time::sleep(Duration::from_millis(wait as u64 + 5)).await;
            // The minute it is now, read after waking: minutes slept
            // through (a suspend, a clock jump) are skipped.
            let minute = unix_now().as_secs() / 60 * 60;
            let Some((key, local)) = cron::local_minute(minute) else {
                continue;
            };
            // Clocks went back: these minutes have run.
            if last.is_some_and(|l| key <= l) {
                continue;
            }
            last = Some(key);
            for (path, job) in due(&projects.borrow(), local) {
                let (slots, limits, out) = (slots.clone(), limits.clone(), out.clone());
                tokio::spawn(async move {
                    let result = execute(&path, &job, minute, &slots, &limits).await;
                    info!(project = %path.display(), command = job.command, status = ?result.status, exit_code = ?result.exit_code, error = result.error, "cron job finished");
                    let msg = Msg::new(MSG_CRON_RESULT).with_data(&result);
                    if msg.map(|m| out.try_send(m)).is_err() {
                        warn!("too many cron results waiting for the hub; dropping one");
                    }
                });
            }
        }
    });
    outbox
}

fn unix_now() -> Duration {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
}

/// The jobs whose schedule matches `local`.
fn due(projects: &[ProjectConfig], local: LocalMinute) -> Vec<(PathBuf, ProjectCronJob)> {
    let mut jobs = Vec::new();
    for p in projects {
        for job in &p.cron {
            if CronSchedule::parse(&job.schedule).is_ok_and(|s| s.matches(local)) {
                jobs.push((PathBuf::from(&p.path), job.clone()));
            }
        }
    }
    jobs
}

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

    #[test]
    fn finds_due_jobs() {
        let dir = tempfile::tempdir().unwrap();
        let p = project(
            dir.path(),
            r#"[{"command": "hello", "schedule": "30 2 * * *"}]"#,
        );
        let at = |minute, hour| LocalMinute {
            minute,
            hour,
            day: 7,
            month: 11,
            weekday: 6,
        };
        let jobs = due(std::slice::from_ref(&p), at(30, 2));
        assert_eq!(jobs.len(), 1);
        assert_eq!(jobs[0].1.command, "hello");
        assert!(due(&[p], at(31, 2)).is_empty());
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
