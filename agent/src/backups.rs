//! Backups, run by the agent itself: on the schedules in config files, and
//! when the hub asks (Back up now). This task outlives sessions, so backups
//! keep running and their results queue while the hub is unreachable. One
//! backup runs at a time. See "Back up" in docs/PROTOCOL.md.

use std::path::{Path, PathBuf};
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use tokio::sync::{mpsc, watch};
use tokio::task::JoinHandle;
use tokio::time::MissedTickBehavior;
use tracing::{info, warn};

use crate::backupchild::Upload;
use crate::backuprun;
use crate::discovery;
use crate::project::ProjectConfig;
use crate::protocol::{
    BackupResult, BackupRun, BackupStarted, MSG_BACKUP_RESULT, MSG_BACKUP_STARTED, Msg,
};
use crate::schedule::LastRuns;

/// Messages kept while no session takes them: about 50 backups' worth.
const QUEUE: usize = 100;
const CHECK_EVERY: Duration = Duration::from_secs(60);

pub struct Settings {
    /// This binary, run again as the backup child.
    pub exe: PathBuf,
    pub timeout: Duration,
    pub state_path: PathBuf,
    pub project_dirs: Vec<PathBuf>,
    pub allow_any_project_path: bool,
}

/// What sessions use to talk to the backups task.
pub struct Handle {
    requests: mpsc::Sender<BackupRun>,
    projects: watch::Sender<Vec<ProjectConfig>>,
}

impl Handle {
    /// Back up now, from `backup_run`.
    pub fn request(&self, run: BackupRun) {
        if self.requests.try_send(run).is_err() {
            warn!("too many backup requests; dropping one");
        }
    }

    /// The projects from the latest search.
    pub fn projects(&self, projects: &[ProjectConfig]) {
        self.projects.send_replace(projects.to_vec());
    }
}

/// Starts the backups task. Its messages for the hub arrive on the
/// receiver; they wait there while no session reads it.
pub fn spawn(settings: Settings) -> (Handle, mpsc::Receiver<Msg>) {
    let (out, outbox) = mpsc::channel(QUEUE);
    let (requests, inbox) = mpsc::channel(8);
    let (projects, projects_rx) = watch::channel(Vec::new());
    let last = LastRuns::load(&settings.state_path);
    let service = Service {
        settings,
        out,
        projects: projects_rx,
        last,
    };
    tokio::spawn(service.run(inbox, projects.clone()));
    (Handle { requests, projects }, outbox)
}

struct Service {
    settings: Settings,
    out: mpsc::Sender<Msg>,
    projects: watch::Receiver<Vec<ProjectConfig>>,
    last: LastRuns,
}

impl Service {
    async fn run(
        mut self,
        mut inbox: mpsc::Receiver<BackupRun>,
        projects: watch::Sender<Vec<ProjectConfig>>,
    ) {
        // Until a session reports projects, search the project directories
        // here, so schedules run even if the hub can't be reached.
        let (dirs, any) = (
            self.settings.project_dirs.clone(),
            self.settings.allow_any_project_path,
        );
        if let Ok(found) =
            tokio::task::spawn_blocking(move || discovery::discover(&dirs, &[], any)).await
        {
            projects.send_if_modified(|p| {
                p.is_empty() && !found.projects.is_empty() && {
                    *p = found.projects;
                    true
                }
            });
        }

        let mut running: Option<JoinHandle<()>> = None;
        let mut tick = tokio::time::interval(CHECK_EVERY);
        tick.set_missed_tick_behavior(MissedTickBehavior::Skip);
        loop {
            let busy = |r: &Option<JoinHandle<()>>| r.as_ref().is_some_and(|h| !h.is_finished());
            tokio::select! {
                Some(req) = inbox.recv() => {
                    if busy(&running) {
                        let path = req.project_path.to_string_lossy().into_owned();
                        let result = rejected(req.run_id, path, "manual", "a backup is already running on this node");
                        send(&self.out, MSG_BACKUP_RESULT, &result);
                    } else {
                        running = Some(self.start(req.run_id, req.project_path, "manual"));
                    }
                }
                _ = tick.tick() => {
                    // A due backup waits while another runs.
                    if !busy(&running) && let Some(path) = self.next_due(now()) {
                        running = Some(self.start(new_id(), path, "schedule"));
                    }
                }
            }
        }
    }

    /// A project whose scheduled backup is due, marked as run.
    fn next_due(&mut self, now: u64) -> Option<PathBuf> {
        let projects = self.projects.borrow().clone();
        for p in projects {
            let Some(b) = &p.backups else { continue };
            let Some(schedule) = b.schedule else { continue };
            if let Some(slot) = self.last.due(&p.path, schedule, b.time_of_day(), now) {
                // Marked before it runs, so a backup that fails or is
                // rejected isn't retried every minute.
                self.last.mark(&p.path, slot);
                return Some(PathBuf::from(&p.path));
            }
        }
        None
    }

    fn start(&self, run_id: String, path: PathBuf, trigger: &'static str) -> JoinHandle<()> {
        let known = self
            .projects
            .borrow()
            .iter()
            .any(|p| Path::new(&p.path) == path);
        let (exe, limit, out) = (
            self.settings.exe.clone(),
            self.settings.timeout,
            self.out.clone(),
        );
        tokio::spawn(async move {
            let result = backup(&exe, limit, &out, run_id, &path, trigger, known).await;
            let size = result.size;
            info!(project = %path.display(), trigger, status = result.status, size, error = %result.error, "backup finished");
            send(&out, MSG_BACKUP_RESULT, &result);
        })
    }
}

async fn backup(
    exe: &Path,
    limit: Duration,
    out: &mpsc::Sender<Msg>,
    run_id: String,
    path: &Path,
    trigger: &'static str,
    known: bool,
) -> BackupResult {
    let project_path = path.to_string_lossy().into_owned();
    let (job, who) = match backuprun::prepare(known, path).await {
        Ok(prepared) => prepared,
        Err(e) => return rejected(run_id, project_path, trigger, &e),
    };
    let (started_at, clock) = (now(), Instant::now());
    let started = BackupStarted {
        run_id: run_id.clone(),
        project_path: project_path.clone(),
        trigger,
        started_at,
        run_as: who.name.clone(),
    };
    send(out, MSG_BACKUP_STARTED, &started);
    let mut result = BackupResult {
        run_id,
        project_path,
        trigger,
        started_at,
        ..BackupResult::default()
    };
    match backuprun::run(exe, &job, &who, limit).await {
        Ok(done) => {
            result.status = "ok";
            (result.file, result.local_path, result.size) = (done.file, done.local_path, done.size);
            (result.database, result.upload) = (done.database, done.upload);
        }
        Err(e) => {
            result.status = "failed";
            result.upload = skipped();
            result.error = e;
        }
    }
    result.duration_ms = clock.elapsed().as_millis() as u64;
    result
}

fn rejected(
    run_id: String,
    project_path: String,
    trigger: &'static str,
    error: &str,
) -> BackupResult {
    BackupResult {
        run_id,
        project_path,
        trigger,
        status: "rejected",
        started_at: now(),
        upload: skipped(),
        error: error.into(),
        ..BackupResult::default()
    }
}

fn skipped() -> Upload {
    Upload {
        status: "skipped".into(),
        ..Upload::default()
    }
}

/// Queues a message for the hub, or drops it if the queue is full.
fn send<T: serde::Serialize>(out: &mpsc::Sender<Msg>, kind: &str, data: &T) {
    if let Ok(msg) = Msg::new(kind).with_data(data)
        && out.try_send(msg).is_err()
    {
        warn!("too many backup messages waiting for the hub; dropping {kind}");
    }
}

fn now() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_or(0, |d| d.as_secs())
}

/// A random UUID (version 4) for a scheduled backup's run_id.
fn new_id() -> String {
    let mut b = [0u8; 16];
    if let Ok(mut f) = std::fs::File::open("/dev/urandom") {
        let _ = std::io::Read::read_exact(&mut f, &mut b);
    }
    b[6] = (b[6] & 0x0f) | 0x40;
    b[8] = (b[8] & 0x3f) | 0x80;
    let h = hex::encode(b);
    format!(
        "{}-{}-{}-{}-{}",
        &h[..8],
        &h[8..12],
        &h[12..16],
        &h[16..20],
        &h[20..]
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use std::os::unix::fs::PermissionsExt;

    async fn next(rx: &mut mpsc::Receiver<Msg>) -> (String, serde_json::Value) {
        let msg = tokio::time::timeout(Duration::from_secs(10), rx.recv())
            .await
            .unwrap()
            .unwrap();
        (msg.kind, msg.data.unwrap())
    }

    #[tokio::test]
    async fn runs_one_backup_at_a_time() {
        let dir = tempfile::tempdir().unwrap();
        let project = fs::canonicalize(dir.path()).unwrap();
        let config = project.join("config.vpspilot.json");
        fs::write(
            &config,
            r#"{"name": "shop", "backups": {"local_dir": "b"}}"#,
        )
        .unwrap();
        fs::set_permissions(&config, fs::Permissions::from_mode(0o644)).unwrap();
        // A fake agent binary standing in for the backup child.
        let exe = project.join("agent");
        let done = r#"{"file":"f.zip","local_path":"/b/f.zip","size":3,"database":"","upload":{"status":"skipped","key":"","error":""}}"#;
        fs::write(
            &exe,
            format!("#!/bin/sh\ncat >/dev/null; sleep 1; echo '{done}'\n"),
        )
        .unwrap();
        fs::set_permissions(&exe, fs::Permissions::from_mode(0o755)).unwrap();

        let (handle, mut rx) = spawn(Settings {
            exe,
            timeout: Duration::from_secs(10),
            state_path: project.join("state.json"),
            project_dirs: vec![],
            allow_any_project_path: false,
        });
        handle.projects(&[ProjectConfig::load(&config).unwrap()]);
        let run = |id: &str, path: &Path| BackupRun {
            run_id: id.into(),
            project_path: path.into(),
        };

        handle.request(run("unknown", Path::new("/srv/other")));
        let (kind, data) = next(&mut rx).await;
        assert_eq!(
            (kind.as_str(), data["status"].as_str()),
            ("backup_result", Some("rejected"))
        );

        handle.request(run("a", &project));
        let (kind, data) = next(&mut rx).await;
        assert_eq!(
            (kind.as_str(), data["run_id"].as_str()),
            ("backup_started", Some("a"))
        );
        handle.request(run("b", &project));
        let (_, data) = next(&mut rx).await;
        assert_eq!(
            (data["run_id"].as_str(), data["status"].as_str()),
            (Some("b"), Some("rejected"))
        );
        assert_eq!(data["error"], "a backup is already running on this node");

        let (kind, data) = next(&mut rx).await;
        assert_eq!(
            (kind.as_str(), data["status"].as_str()),
            ("backup_result", Some("ok"))
        );
        assert_eq!(
            (data["trigger"].as_str(), data["size"].as_u64()),
            (Some("manual"), Some(3))
        );
    }

    #[test]
    fn makes_uuids() {
        let (a, b) = (new_id(), new_id());
        assert_ne!(a, b);
        assert_eq!((a.len(), &a[14..15]), (36, "4"));
    }
}
