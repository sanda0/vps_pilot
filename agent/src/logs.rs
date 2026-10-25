//! Log requests in this session: checks each `log_list` and `log_open`
//! against "Read logs" in docs/PROTOCOL.md, then runs the log child process
//! (logchild.rs) as the config file's owner to list or read. Dropping `Logs`
//! (the session ended) closes every stream.

use std::collections::HashMap;
use std::path::{Path, PathBuf};
use std::process::Stdio;
use std::sync::Arc;
use std::time::Duration;

use tokio::io::{AsyncBufReadExt, AsyncReadExt, BufReader};
use tokio::sync::{Notify, mpsc};
use tokio::task::{Id, JoinSet};
use tokio::time::timeout;
use tracing::warn;

use crate::commands::{Group, RunAs, owner_command};
use crate::logchild::{self, EXIT_REJECTED, Job, Listing};
use crate::logread::Chunk;
use crate::project::ProjectConfig;
use crate::protocol::{
    LogData, LogEnd, LogEndReason, LogFiles, LogList, LogOpen, MSG_LOG_DATA, MSG_LOG_END,
    MSG_LOG_FILES, Msg,
};

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

    /// Starts a stream, or sends `log_end` with reason `rejected`.
    pub async fn open(&mut self, req: LogOpen, projects: &[ProjectConfig]) {
        if req.stream_id.is_empty() || self.streams.contains_key(&req.stream_id) {
            warn!(
                stream_id = req.stream_id,
                "ignoring log_open with an empty or reused stream_id"
            );
            return;
        }
        let refused = if self.streams.len() >= MAX_STREAMS {
            Some(format!("{MAX_STREAMS} log streams are already open"))
        } else if req.follow && req.before.is_some() {
            Some("only the end of a file can be followed".to_string())
        } else {
            None
        };
        if let Some(error) = refused {
            let end = LogEnd {
                stream_id: req.stream_id,
                reason: LogEndReason::Rejected,
                error,
            };
            return send(&self.out, MSG_LOG_END, &end).await;
        }
        let close = Arc::new(Notify::new());
        let known = is_known(projects, &req.project_path);
        let stream = Stream {
            exe: self.exe.clone(),
            limit: self.follow_limit,
            out: self.out.clone(),
        };
        let id = req.stream_id.clone();
        let task = self.tasks.spawn(stream.run(req, known, close.clone()));
        self.streams.insert(id, (task.id(), close));
    }

    pub fn close(&self, stream_id: &str) {
        if let Some((_, close)) = self.streams.get(stream_id) {
            close.notify_one();
        }
    }

    /// Waits for a task to finish and forgets it. Never returns while none
    /// run, so it can sit in a `select!`.
    pub async fn reap(&mut self) {
        let id = match self.tasks.join_next_with_id().await {
            Some(Ok((id, ()))) => id,
            Some(Err(e)) => e.id(),
            None => return std::future::pending().await,
        };
        self.streams.retain(|_, (task, _)| *task != id);
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

struct Stream {
    exe: PathBuf,
    limit: Duration,
    out: mpsc::Sender<Msg>,
}

impl Stream {
    async fn run(self, req: LogOpen, known: bool, close: Arc<Notify>) {
        let (reason, error) = match self.read(&req, known, close).await {
            Ok(reason) => (reason, String::new()),
            Err((reason, error)) => (reason, error),
        };
        let end = LogEnd {
            stream_id: req.stream_id,
            reason,
            error,
        };
        send(&self.out, MSG_LOG_END, &end).await;
    }

    async fn read(
        &self,
        req: &LogOpen,
        known: bool,
        close: Arc<Notify>,
    ) -> Result<LogEndReason, (LogEndReason, String)> {
        let rejected = |e| (LogEndReason::Rejected, e);
        let failed = |e: std::io::Error| (LogEndReason::Error, e.to_string());
        let (logs, who) = prepare(known, &req.project_path).await.map_err(rejected)?;
        let job = Job::Read {
            file: req.file.clone(),
            before: req.before,
            max_bytes: req.max_bytes,
            follow: req.follow,
        };
        let mut cmd = owner_command(&self.exe, &req.project_path, &who);
        cmd.args(logchild::args(&req.project_path, &logs, &job))
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());
        let mut child = cmd.spawn().map_err(failed)?;
        let mut group = Group::new(&child);
        let mut lines = BufReader::new(child.stdout.take().unwrap()).lines();
        let limit = tokio::time::sleep(self.limit);
        tokio::pin!(limit);
        let mut seq = 0;
        loop {
            tokio::select! {
                line = lines.next_line() => match line.map_err(failed)? {
                    Some(line) => {
                        let c: Chunk = serde_json::from_str(&line).map_err(|e| failed(e.into()))?;
                        let data = LogData { stream_id: req.stream_id.clone(), seq, offset: c.offset, data: c.data, size: c.size, rotated: c.rotated };
                        seq += 1;
                        send(&self.out, MSG_LOG_DATA, &data).await;
                    }
                    None => break,
                },
                _ = close.notified() => {
                    group.kill(&mut child).await;
                    return Ok(LogEndReason::Closed);
                }
                _ = &mut limit, if req.follow => {
                    group.kill(&mut child).await;
                    return Ok(LogEndReason::Limit);
                }
            }
        }
        let status = child.wait().await.map_err(failed)?;
        group.disarm();
        if status.success() {
            return Ok(LogEndReason::Done);
        }
        let mut stderr = String::new();
        if let Some(mut err) = child.stderr.take() {
            let _ = err.read_to_string(&mut stderr).await;
        }
        let reason = if status.code() == Some(EXIT_REJECTED.into()) {
            LogEndReason::Rejected
        } else {
            LogEndReason::Error
        };
        Err((reason, stderr.trim().to_string()))
    }
}

async fn send<T: serde::Serialize>(out: &mpsc::Sender<Msg>, kind: &str, data: &T) {
    if let Ok(msg) = Msg::new(kind).with_data(data) {
        let _ = out.send(msg).await;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::PermissionsExt;

    /// Stands in for the log child: lists one file, or for `read` sends two
    /// chunks, refuses ("reject"), or sends one and keeps following.
    const FAKE: &str = r#"#!/bin/sh
case "$3" in
  list) printf '%s\n' '{"files":[{"path":"logs/a.log","size":2,"modified":1}],"truncated":false}' ;;
  read) case "$4" in
    reject) echo nope >&2; exit 2 ;;
    follow) printf '%s\n' '{"offset":0,"data":"a\n","size":2,"rotated":false}'; exec sleep 30 ;;
    *) printf '%s\n' '{"offset":0,"data":"a\n","size":2,"rotated":false}'
       printf '%s\n' '{"offset":2,"data":"b\n","size":4,"rotated":false}' ;;
  esac ;;
esac
"#;

    struct Setup {
        _dir: tempfile::TempDir,
        logs: Logs,
        rx: mpsc::Receiver<Msg>,
        projects: Vec<ProjectConfig>,
        path: PathBuf,
    }

    fn setup(follow_limit: Duration) -> Setup {
        let dir = tempfile::tempdir().unwrap();
        let exe = dir.path().join("fake-agent");
        std::fs::write(&exe, FAKE).unwrap();
        std::fs::set_permissions(&exe, std::fs::Permissions::from_mode(0o755)).unwrap();
        let project = dir.path().join("app");
        std::fs::create_dir(&project).unwrap();
        let config = project.join(crate::project::CONFIG_FILE);
        std::fs::write(&config, r#"{"name": "app", "logs": ["logs"]}"#).unwrap();
        std::fs::set_permissions(&config, std::fs::Permissions::from_mode(0o644)).unwrap();
        let (found, _) = ProjectConfig::load_for_run(&project).unwrap();
        let (tx, rx) = mpsc::channel(64);
        let path = PathBuf::from(&found.path);
        let logs = Logs::new(exe, follow_limit, tx);
        Setup {
            _dir: dir,
            logs,
            rx,
            projects: vec![found],
            path,
        }
    }

    fn open(id: &str, path: &Path, file: &str, follow: bool, before: Option<u64>) -> LogOpen {
        LogOpen {
            stream_id: id.into(),
            project_path: path.into(),
            file: file.into(),
            before,
            max_bytes: 100,
            follow,
        }
    }

    async fn next(rx: &mut mpsc::Receiver<Msg>) -> (String, serde_json::Value) {
        let msg = timeout(Duration::from_secs(5), rx.recv())
            .await
            .unwrap()
            .unwrap();
        (msg.kind, msg.data.unwrap())
    }

    #[tokio::test]
    async fn lists_and_reads() {
        let mut s = setup(Duration::from_secs(60));
        let req = LogList {
            request_id: "l1".into(),
            project_path: s.path.clone(),
        };
        s.logs.list(req, &s.projects);
        let (kind, files) = next(&mut s.rx).await;
        assert_eq!(
            (kind.as_str(), &files["files"][0]["path"]),
            ("log_files", &"logs/a.log".into())
        );

        s.logs
            .open(open("s1", &s.path, "logs/a.log", false, None), &s.projects)
            .await;
        for seq in 0..2 {
            let (kind, data) = next(&mut s.rx).await;
            assert_eq!(
                (kind.as_str(), data["seq"].as_u64()),
                ("log_data", Some(seq)),
                "{data}"
            );
        }
        let (kind, end) = next(&mut s.rx).await;
        assert_eq!((kind.as_str(), &end["reason"]), ("log_end", &"done".into()));

        let unknown = LogList {
            request_id: "l2".into(),
            project_path: "/nowhere".into(),
        };
        s.logs.list(unknown, &s.projects);
        let (_, files) = next(&mut s.rx).await;
        assert!(
            files["error"].as_str().unwrap().contains("not a project"),
            "{files}"
        );
    }

    #[tokio::test]
    async fn rejects_closes_and_limits() {
        let mut s = setup(Duration::from_millis(300));
        let end = async |logs: &mut Logs,
                         req: LogOpen,
                         projects: &[ProjectConfig],
                         rx: &mut mpsc::Receiver<Msg>| {
            logs.open(req, projects).await;
            loop {
                let (kind, data) = next(rx).await;
                if kind == "log_end" {
                    return (
                        data["reason"].as_str().unwrap().to_string(),
                        data["error"].as_str().unwrap().to_string(),
                    );
                }
            }
        };
        let r = end(
            &mut s.logs,
            open("a", &s.path, "reject", false, None),
            &s.projects,
            &mut s.rx,
        )
        .await;
        assert_eq!(r, ("rejected".into(), "nope".into()));
        let r = end(
            &mut s.logs,
            open("b", &s.path, "x", true, Some(10)),
            &s.projects,
            &mut s.rx,
        )
        .await;
        assert_eq!(r.0, "rejected");
        let r = end(
            &mut s.logs,
            open("c", &s.path, "follow", true, None),
            &s.projects,
            &mut s.rx,
        )
        .await;
        assert_eq!(r.0, "limit");

        s.logs
            .open(open("d", &s.path, "follow", true, None), &s.projects)
            .await;
        assert_eq!(next(&mut s.rx).await.0, "log_data");
        s.logs.close("d");
        let (kind, data) = next(&mut s.rx).await;
        assert_eq!(
            (kind.as_str(), &data["reason"]),
            ("log_end", &"closed".into())
        );
    }
}
