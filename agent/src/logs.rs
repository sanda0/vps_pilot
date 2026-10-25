//! Log requests in this session: checks each `log_list` against "Read logs"
//! in docs/PROTOCOL.md, then runs the log child process (logchild.rs) as the
//! config file's owner to list the files.

use std::collections::HashMap;
use std::path::{Path, PathBuf};
use std::process::Stdio;
use std::sync::Arc;
use std::time::Duration;

use tokio::sync::{Notify, mpsc};
use tokio::task::{Id, JoinSet};
use tokio::time::timeout;

use crate::commands::{RunAs, owner_command};
use crate::logchild::{self, Job, Listing};
use crate::project::ProjectConfig;
use crate::protocol::{LogFiles, LogList, MSG_LOG_FILES, Msg};

const MAX_STREAMS: usize = 8;
const LIST_TIMEOUT: Duration = Duration::from_secs(10);

pub struct Logs {
    /// The agent binary, run again as the log child.
    exe: PathBuf,
    follow_limit: Duration,
    out: mpsc::Sender<Msg>,
    tasks: JoinSet<()>,
    streams: HashMap<String, (Id, Arc<Notify>)>,
}

impl Logs {
    pub fn new(exe: PathBuf, follow_limit: Duration, out: mpsc::Sender<Msg>) -> Self {
        let (tasks, streams) = (JoinSet::new(), HashMap::new());
        Logs {
            exe,
            follow_limit,
            out,
            tasks,
            streams,
        }
    }

    /// Lists the project's log files and sends `log_files`.
    pub fn list(&mut self, req: LogList, projects: &[ProjectConfig]) {
        let (exe, out) = (self.exe.clone(), self.out.clone());
        let known = is_known(projects, &req.project_path);
        self.tasks.spawn(async move {
            let files = match prepare(known, &req.project_path).await {
                Ok((logs, who)) => list(&exe, &req.project_path, &logs, &who).await,
                Err(e) => Err(e),
            };
            let (files, error) = match files {
                Ok(listing) if listing.truncated => (
                    listing.files,
                    "more than 1000 log files; showing the first 1000".into(),
                ),
                Ok(listing) => (listing.files, String::new()),
                Err(e) => (Vec::new(), e),
            };
            send(
                &out,
                MSG_LOG_FILES,
                &LogFiles {
                    request_id: req.request_id,
                    files,
                    error,
                },
            )
            .await;
        });
    }
}

fn is_known(projects: &[ProjectConfig], path: &Path) -> bool {
    projects.iter().any(|p| Path::new(&p.path) == path)
}

/// The project's declared log folders and the user to read them as, after
/// re-reading its config with the same checks as for commands.
async fn prepare(known: bool, project: &Path) -> Result<(Vec<String>, RunAs), String> {
    if !known {
        return Err("not a project found by this agent".into());
    }
    let dir = project.to_path_buf();
    let loaded = tokio::task::spawn_blocking(move || ProjectConfig::load_for_run(&dir)).await;
    let (config, uid) = loaded
        .map_err(|e| e.to_string())?
        .map_err(|e| format!("{e:#}"))?;
    Ok((config.logs, RunAs::owner(uid)?))
}

async fn list(exe: &Path, project: &Path, logs: &[String], who: &RunAs) -> Result<Listing, String> {
    let mut cmd = owner_command(exe, project, who);
    cmd.kill_on_drop(true)
        .args(logchild::args(project, logs, &Job::List))
        .stdout(Stdio::piped())
        .stderr(Stdio::piped());
    let child = cmd
        .spawn()
        .map_err(|e| format!("start the log reader: {e}"))?;
    let output = timeout(LIST_TIMEOUT, child.wait_with_output())
        .await
        .map_err(|_| "listing log files took too long".to_string())?
        .map_err(|e| e.to_string())?;
    if !output.status.success() {
        return Err(String::from_utf8_lossy(&output.stderr).trim().to_string());
    }
    serde_json::from_slice(&output.stdout).map_err(|e| format!("log reader: {e}"))
}

async fn send<T: serde::Serialize>(out: &mpsc::Sender<Msg>, kind: &str, data: &T) {
    if let Ok(msg) = Msg::new(kind).with_data(data) {
        let _ = out.send(msg).await;
    }
}
