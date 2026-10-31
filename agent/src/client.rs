//! One connection to the hub: handshake (enroll or connect), then stream stats.

use std::path::PathBuf;
use std::sync::Arc;
use std::time::Duration;

use anyhow::{Context, anyhow, bail};
use tokio::io::{AsyncRead, ReadHalf};
use tokio::net::TcpStream;
use tokio::sync::{Notify, mpsc, watch};
use tokio::task::AbortHandle;
use tokio::time::{MissedTickBehavior, timeout};
use tracing::{debug, info, warn};

use crate::backups;
use crate::commands::Limits;
use crate::discovery::Found;
use crate::logs::Logs;
use crate::metrics::Collector;
use crate::project::ProjectConfig;
use crate::protocol::{
    self, AuthFailedPayload, EnrolledPayload, MAX_FRAME_SIZE, MSG_AUTH_FAILED, MSG_COMMAND_CANCEL,
    MSG_COMMAND_RUN, MSG_CONNECTED, MSG_ENROLL, MSG_ENROLLED, MSG_PROJECT_SETTINGS, MSG_PROJECTS,
    MSG_RESCAN_PROJECTS, MSG_SYS_STAT, Msg, ProjectSettings, ProjectsPayload,
};
use crate::protocol::{
    BackupRun, CommandCancel, CommandRun, LogClose, LogList, LogOpen, MSG_BACKUP_RUN,
    MSG_LOG_CLOSE, MSG_LOG_LIST, MSG_LOG_OPEN,
};
use crate::runs::Runs;
use crate::scanner;
use crate::state::State;
use crate::tls;

const CONNECT_TIMEOUT: Duration = Duration::from_secs(10);
const HANDSHAKE_TIMEOUT: Duration = Duration::from_secs(10);
/// Output sent per run before the rest is dropped (docs/PROTOCOL.md).
const MAX_COMMAND_OUTPUT: usize = 4 << 20;
/// How long a followed log stream stays open (docs/PROTOCOL.md).
const LOG_FOLLOW_LIMIT: Duration = Duration::from_secs(30 * 60);

/// Join token from the dashboard: `vpsp1.<secret>.<hub cert fingerprint>`.
#[derive(Debug, Clone, PartialEq)]
pub struct JoinToken {
    pub secret: String,
    pub fingerprint: String,
}

impl JoinToken {
    pub fn parse(token: &str) -> anyhow::Result<Self> {
        let parts: Vec<&str> = token.trim().split('.').collect();
        match parts.as_slice() {
            ["vpsp1", secret, fp] if !secret.is_empty() => Ok(JoinToken {
                secret: secret.to_string(),
                fingerprint: tls::parse_fingerprint(fp)?,
            }),
            _ => bail!("malformed join token (expected vpsp1.<secret>.<fingerprint>)"),
        }
    }
}

pub struct Config {
    pub hub: String,
    pub state_path: PathBuf,
    pub join: Option<JoinToken>,
    pub interval: Duration,
    pub project_dirs: Vec<PathBuf>,
    pub allow_any_project_path: bool,
    pub command_timeout: Duration,
}

#[derive(Debug, thiserror::Error)]
pub enum SessionError {
    /// The hub refused our credentials. Retrying won't help.
    #[error("hub rejected agent: {0}")]
    AuthFailed(String),
    /// Network or protocol failure. Reconnect after a backoff.
    #[error(transparent)]
    Disconnected(#[from] anyhow::Error),
}

/// Aborts the reader task when the session ends.
struct AbortOnDrop(AbortHandle);

impl Drop for AbortOnDrop {
    fn drop(&mut self) {
        self.0.abort();
    }
}

/// Runs one session. Returns only with an error: either the connection
/// dropped (retry) or the hub rejected us (stop).
pub async fn run_session(
    cfg: &Config,
    collector: &mut Collector,
    backups: &backups::Handle,
    backup_out: &mut mpsc::Receiver<Msg>,
) -> Result<(), SessionError> {
    let state = State::load(&cfg.state_path)?;
    let fingerprint = match (&state, &cfg.join) {
        (Some(s), _) => s.hub_fingerprint.clone(),
        (None, Some(join)) => join.fingerprint.clone(),
        (None, None) => {
            return Err(SessionError::AuthFailed(format!(
                "not enrolled: no state at {} and no --token given",
                cfg.state_path.display()
            )));
        }
    };

    let tcp = timeout(CONNECT_TIMEOUT, TcpStream::connect(&cfg.hub))
        .await
        .map_err(|_| anyhow!("timed out connecting to {}", cfg.hub))?
        .with_context(|| format!("connect to {}", cfg.hub))?;
    tcp.set_nodelay(true).ok();
    let stream = tls::connect(tcp, tls::client_config(&fingerprint)?).await?;
    let (mut rd, mut wr) = tokio::io::split(stream);

    let info = collector.system_info();
    let hello = match (&state, &cfg.join) {
        (Some(s), _) => Msg::new(MSG_CONNECTED).with_token(s.connect_token()),
        (None, Some(join)) => Msg::new(MSG_ENROLL).with_token(join.secret.clone()),
        (None, None) => unreachable!(),
    }
    .with_data(&info)?;
    protocol::write_frame(&mut wr, &hello).await?;

    let node_id = loop {
        let msg = timeout(HANDSHAKE_TIMEOUT, protocol::read_frame(&mut rd))
            .await
            .map_err(|_| anyhow!("hub did not answer the handshake"))??;
        match msg.kind.as_str() {
            MSG_ENROLLED => {
                let creds: EnrolledPayload = msg.payload()?;
                State {
                    agent_id: creds.agent_id,
                    node_key: creds.node_key,
                    node_id: creds.node_id,
                    hub_fingerprint: fingerprint.clone(),
                }
                .save(&cfg.state_path)?;
                info!(
                    node_id = creds.node_id,
                    state = %cfg.state_path.display(),
                    "enrolled; the join token is no longer needed"
                );
            }
            MSG_SYS_STAT => break msg.node_id,
            MSG_AUTH_FAILED => return Err(auth_failed(&msg)),
            other => debug!("ignoring {other:?} during handshake"),
        }
    };
    info!(node_id, hub = %cfg.hub, "connected; streaming stats");

    // Reads run in their own task: read_frame is not cancel-safe, so it
    // must not sit in the select! below.
    let (tx, mut inbox) = mpsc::channel(16);
    let reader = tokio::spawn(read_loop(rd, tx));
    let _reader_guard = AbortOnDrop(reader.abort_handle());

    // Project searches run in their own task (see scanner.rs). A result is
    // reported when it changed or when the scanner says it must be.
    let (settings_tx, settings_rx) = watch::channel(ProjectSettings::default());
    let rescan = Arc::new(Notify::new());
    let (projects_tx, mut projects_rx) = mpsc::channel(1);
    let scanner = tokio::spawn(scanner::scan_loop(
        cfg.project_dirs.clone(),
        cfg.allow_any_project_path,
        settings_rx,
        rescan.clone(),
        projects_tx,
    ));
    let _scanner_guard = AbortOnDrop(scanner.abort_handle());
    let mut reported: Option<Found> = None;

    // Commands and log streams run in their own tasks and queue their
    // messages here. Dropping `runs` and `logs` when the session ends kills
    // their processes. Logs are read by this binary run again as the config
    // file's owner; /proc/self/exe still works after an upgrade replaced it.
    let (out_tx, mut outbox) = mpsc::channel(64);
    let limits = Limits {
        timeout: cfg.command_timeout,
        max_output: MAX_COMMAND_OUTPUT,
    };
    let mut runs = Runs::new(limits, out_tx.clone());
    let mut logs = Logs::new("/proc/self/exe".into(), LOG_FOLLOW_LIMIT, out_tx);

    let mut ticker = tokio::time::interval(cfg.interval);
    ticker.set_missed_tick_behavior(MissedTickBehavior::Skip);
    loop {
        tokio::select! {
            msg = inbox.recv() => match msg {
                Some(Ok(msg)) if msg.kind == MSG_AUTH_FAILED => return Err(auth_failed(&msg)),
                Some(Ok(msg)) if msg.kind == MSG_PROJECT_SETTINGS => {
                    match msg.payload::<ProjectSettings>() {
                        Ok(new) => {
                            settings_tx.send_if_modified(|s| {
                                let changed = *s != new;
                                *s = new;
                                changed
                            });
                        }
                        Err(e) => warn!("ignoring invalid project_settings: {e:#}"),
                    }
                }
                Some(Ok(msg)) if msg.kind == MSG_RESCAN_PROJECTS => rescan.notify_one(),
                Some(Ok(msg)) if msg.kind == MSG_COMMAND_RUN => match msg.payload::<CommandRun>() {
                    Ok(run) => {
                        let projects = reported.as_ref().map(|f| f.projects.as_slice()).unwrap_or_default();
                        runs.start(run, projects).await;
                    }
                    Err(e) => warn!("ignoring invalid command_run: {e:#}"),
                },
                Some(Ok(msg)) if msg.kind == MSG_COMMAND_CANCEL => {
                    if let Ok(c) = msg.payload::<CommandCancel>() {
                        runs.cancel(&c.run_id);
                    }
                }
                Some(Ok(msg)) if msg.kind == MSG_BACKUP_RUN => match msg.payload::<BackupRun>() {
                    Ok(run) => backups.request(run),
                    Err(e) => warn!("ignoring invalid backup_run: {e:#}"),
                },
                Some(Ok(msg)) if msg.kind.starts_with("log_") => {
                    let projects = reported.as_ref().map(|f| f.projects.as_slice()).unwrap_or_default();
                    handle_log(&mut logs, &msg, projects).await;
                }
                Some(Ok(msg)) => debug!("ignoring {:?} from hub", msg.kind),
                Some(Err(e)) => return Err(e.into()),
                None => return Err(anyhow!("hub closed the connection").into()),
            },
            Some(msg) = outbox.recv() => protocol::write_frame(&mut wr, &msg).await?,
            Some(msg) = backup_out.recv() => protocol::write_frame(&mut wr, &msg).await?,
            () = runs.reap() => {}
            () = logs.reap() => {}
            _ = ticker.tick() => {
                let stat = collector.collect();
                protocol::write_frame(&mut wr, &Msg::new(MSG_SYS_STAT).with_data(&stat)?).await?;
            }
            Some(scanner::ScanResult { found, always_report }) = projects_rx.recv() => {
                backups.projects(&found.projects);
                if !always_report && reported.as_ref() == Some(&found) {
                    continue;
                }
                if let Some(msg) = projects_msg(&found)? {
                    protocol::write_frame(&mut wr, &msg).await?;
                    let (count, errors) = (found.projects.len(), found.errors.len());
                    info!(count, errors, "reported projects");
                }
                reported = Some(found);
            }
        }
    }
}

async fn handle_log(logs: &mut Logs, msg: &Msg, projects: &[ProjectConfig]) {
    let invalid = |e: anyhow::Error| warn!("ignoring invalid {}: {e:#}", msg.kind);
    match msg.kind.as_str() {
        MSG_LOG_LIST => match msg.payload::<LogList>() {
            Ok(req) => logs.list(req, projects),
            Err(e) => invalid(e),
        },
        MSG_LOG_OPEN => match msg.payload::<LogOpen>() {
            Ok(req) => logs.open(req, projects).await,
            Err(e) => invalid(e),
        },
        MSG_LOG_CLOSE => match msg.payload::<LogClose>() {
            Ok(req) => logs.close(&req.stream_id),
            Err(e) => invalid(e),
        },
        other => debug!("ignoring {other:?} from hub"),
    }
}

/// The `projects` message, or None (logged) when it wouldn't fit in a frame.
/// Sending an oversized frame would fail and drop the connection on every
/// reconnect, taking stats down with it.
fn projects_msg(found: &Found) -> anyhow::Result<Option<Msg>> {
    let payload = ProjectsPayload {
        projects: &found.projects,
        errors: &found.errors,
    };
    let msg = Msg::new(MSG_PROJECTS).with_data(&payload)?;
    let size = serde_json::to_vec(&msg)?.len();
    if size > MAX_FRAME_SIZE as usize {
        warn!(
            count = found.projects.len(),
            size, "project list is too large to report; not sending it"
        );
        return Ok(None);
    }
    Ok(Some(msg))
}

async fn read_loop<S: AsyncRead>(mut rd: ReadHalf<S>, tx: mpsc::Sender<anyhow::Result<Msg>>) {
    loop {
        let msg = protocol::read_frame(&mut rd).await;
        let failed = msg.is_err();
        if tx.send(msg).await.is_err() || failed {
            return;
        }
    }
}

fn auth_failed(msg: &Msg) -> SessionError {
    let reason = msg
        .payload::<AuthFailedPayload>()
        .map(|p| p.reason)
        .unwrap_or_default();
    if reason.is_empty() {
        warn!("hub sent auth_failed without a reason");
    }
    SessionError::AuthFailed(reason)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_join_token() {
        let fp = "ab".repeat(32);
        let token = JoinToken::parse(&format!("vpsp1.s3cret.{fp}\n")).unwrap();
        assert_eq!(token.secret, "s3cret");
        assert_eq!(token.fingerprint, fp);
    }

    #[test]
    fn rejects_malformed_join_tokens() {
        let fp = "ab".repeat(32);
        for bad in [
            String::new(),
            "vpsp1.s3cret".to_string(),
            format!("vpsp2.s3cret.{fp}"),
            format!("vpsp1..{fp}"),
            "vpsp1.s3cret.short".to_string(),
        ] {
            assert!(JoinToken::parse(&bad).is_err(), "{bad:?}");
        }
    }

    fn found(name: &str, tech: &str) -> Found {
        let project = crate::project::ProjectConfig {
            name: name.into(),
            path: format!("/srv/{name}"),
            tech: vec![tech.into()],
            commands: vec![],
            logs: vec![],
            backups: None,
        };
        let error = crate::discovery::ProjectError {
            path: "/opt/x".into(),
            error: "outside".into(),
        };
        Found {
            projects: vec![project],
            errors: vec![error],
        }
    }

    #[test]
    fn projects_message_shape() {
        let msg = projects_msg(&found("app", "go")).unwrap().unwrap();
        assert_eq!(msg.kind, "projects");
        let data = msg.data.unwrap();
        assert_eq!(data["projects"][0]["path"], "/srv/app");
        assert_eq!(data["errors"][0]["path"], "/opt/x");
        assert_eq!(data["errors"][0]["error"], "outside");
    }

    #[test]
    fn oversized_project_list_is_not_sent() {
        let big = "x".repeat(MAX_FRAME_SIZE as usize);
        assert!(projects_msg(&found("app", &big)).unwrap().is_none());
    }
}
