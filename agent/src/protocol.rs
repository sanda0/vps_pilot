//! Wire protocol shared with the hub. See docs/PROTOCOL.md.
//!
//! Each frame is a 4-byte big-endian length followed by a JSON envelope.

use std::path::PathBuf;
use std::time::Duration;

use anyhow::{Context, Result, bail};
use serde::{Deserialize, Serialize};
use tokio::io::{AsyncRead, AsyncReadExt, AsyncWrite, AsyncWriteExt};

use crate::discovery::ProjectError;
use crate::project::ProjectConfig;

/// Frames larger than this are rejected (matches the hub).
pub const MAX_FRAME_SIZE: u32 = 1 << 20;

pub const MSG_ENROLL: &str = "enroll";
pub const MSG_ENROLLED: &str = "enrolled";
pub const MSG_CONNECTED: &str = "connected";
pub const MSG_AUTH_FAILED: &str = "auth_failed";
pub const MSG_SYS_STAT: &str = "sys_stat";
pub const MSG_PROJECTS: &str = "projects";
pub const MSG_PROJECT_SETTINGS: &str = "project_settings";
pub const MSG_RESCAN_PROJECTS: &str = "rescan_projects";
pub const MSG_COMMAND_RUN: &str = "command_run";
pub const MSG_COMMAND_CANCEL: &str = "command_cancel";
pub const MSG_COMMAND_OUTPUT: &str = "command_output";
pub const MSG_COMMAND_EXIT: &str = "command_exit";
pub const MSG_LOG_LIST: &str = "log_list";
pub const MSG_LOG_FILES: &str = "log_files";
pub const MSG_LOG_OPEN: &str = "log_open";
pub const MSG_LOG_CLOSE: &str = "log_close";
pub const MSG_LOG_DATA: &str = "log_data";
pub const MSG_LOG_END: &str = "log_end";
pub const MSG_BACKUP_STARTED: &str = "backup_started";

/// Frame envelope.
#[derive(Debug, Default, Clone, PartialEq, Serialize, Deserialize)]
pub struct Msg {
    #[serde(rename = "type")]
    pub kind: String,
    #[serde(default, skip_serializing_if = "is_zero")]
    pub node_id: i64,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub token: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub data: Option<serde_json::Value>,
}

fn is_zero(n: &i64) -> bool {
    *n == 0
}

impl Msg {
    pub fn new(kind: &str) -> Self {
        Msg {
            kind: kind.to_string(),
            ..Default::default()
        }
    }

    pub fn with_token(mut self, token: impl Into<String>) -> Self {
        self.token = token.into();
        self
    }

    pub fn with_data<T: Serialize>(mut self, data: &T) -> Result<Self> {
        self.data = Some(serde_json::to_value(data)?);
        Ok(self)
    }

    /// Decodes the data payload into T.
    pub fn payload<T: for<'de> Deserialize<'de>>(&self) -> Result<T> {
        let data = self.data.clone().context("message has no data")?;
        Ok(serde_json::from_value(data)?)
    }
}

/// Sent by the hub in `enrolled`.
#[derive(Debug, Clone, Deserialize)]
pub struct EnrolledPayload {
    pub agent_id: String,
    pub node_key: String,
    pub node_id: i64,
}

/// Sent by the hub in `auth_failed`.
#[derive(Debug, Clone, Default, Deserialize)]
pub struct AuthFailedPayload {
    #[serde(default)]
    pub reason: String,
}

/// Sent by the hub in `project_settings`. Missing fields keep their defaults,
/// which are also used until the hub sends settings.
#[derive(Debug, Clone, PartialEq, Deserialize)]
#[serde(default)]
pub struct ProjectSettings {
    pub auto_scan: bool,
    /// Seconds between automatic searches; see `interval()`.
    pub scan_interval: u64,
    /// Project directories added from the dashboard.
    pub paths: Vec<PathBuf>,
}

impl Default for ProjectSettings {
    fn default() -> Self {
        ProjectSettings {
            auto_scan: true,
            scan_interval: 900,
            paths: Vec::new(),
        }
    }
}

impl ProjectSettings {
    /// The scan interval, clamped to 1 minute ..= 1 day.
    pub fn interval(&self) -> Duration {
        Duration::from_secs(self.scan_interval.clamp(60, 86_400))
    }
}

/// Sent by the agent in `projects`: every project on the node. The hub
/// removes projects missing from the list.
#[derive(Debug, Serialize)]
pub struct ProjectsPayload<'a> {
    pub projects: &'a [ProjectConfig],
    pub errors: &'a [ProjectError],
}

/// Sent by the hub in `command_run`. `command` is a command *name* from the
/// project's config, never a command string.
#[derive(Debug, Clone, PartialEq, Deserialize)]
pub struct CommandRun {
    pub run_id: String,
    pub project_path: PathBuf,
    pub command: String,
}

/// Sent by the hub in `command_cancel`.
#[derive(Debug, Clone, Deserialize)]
pub struct CommandCancel {
    pub run_id: String,
}

#[derive(Debug, Clone, Copy, PartialEq, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum Stream {
    Stdout,
    Stderr,
}

/// Sent by the agent in `command_output`.
#[derive(Debug, Clone, PartialEq, Serialize)]
pub struct CommandOutput {
    pub run_id: String,
    /// Counts up from 0 across both streams.
    pub seq: u64,
    pub stream: Stream,
    pub data: String,
}

#[derive(Debug, Clone, Copy, PartialEq, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum ExitStatus {
    Exited,
    Timeout,
    Cancelled,
    Rejected,
    Failed,
}

/// Sent by the agent in `command_exit`, once per `command_run`.
#[derive(Debug, Clone, PartialEq, Serialize)]
pub struct CommandExit {
    pub run_id: String,
    pub status: ExitStatus,
    pub exit_code: Option<i32>,
    pub run_as: String,
    pub duration_ms: u64,
    pub truncated: bool,
    pub error: String,
}

impl CommandExit {
    /// A run that never started.
    pub fn not_started(run_id: &str, status: ExitStatus, error: impl Into<String>) -> Self {
        CommandExit {
            run_id: run_id.to_string(),
            status,
            exit_code: None,
            run_as: String::new(),
            duration_ms: 0,
            truncated: false,
            error: error.into(),
        }
    }
}

/// Sent by the hub in `log_list`.
#[derive(Debug, Clone, Deserialize)]
pub struct LogList {
    pub request_id: String,
    pub project_path: PathBuf,
}

/// A file in a project's log folders, as listed in `log_files`.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct LogFile {
    /// Relative to the project directory.
    pub path: String,
    pub size: u64,
    /// Unix seconds.
    pub modified: i64,
}

/// Sent by the agent in `log_files`.
#[derive(Debug, Clone, PartialEq, Serialize)]
pub struct LogFiles {
    pub request_id: String,
    pub files: Vec<LogFile>,
    pub error: String,
}

/// Sent by the hub in `log_open`.
#[derive(Debug, Clone, PartialEq, Deserialize)]
pub struct LogOpen {
    pub stream_id: String,
    pub project_path: PathBuf,
    /// Relative to the project directory, as listed in `log_files`.
    pub file: String,
    /// Byte offset the window ends at; None for the end of the file.
    #[serde(default)]
    pub before: Option<u64>,
    #[serde(default = "default_max_bytes")]
    pub max_bytes: u64,
    #[serde(default)]
    pub follow: bool,
}

fn default_max_bytes() -> u64 {
    64 * 1024
}

/// Sent by the hub in `log_close`.
#[derive(Debug, Clone, Deserialize)]
pub struct LogClose {
    pub stream_id: String,
}

/// Sent by the agent in `log_data`.
#[derive(Debug, Clone, PartialEq, Serialize)]
pub struct LogData {
    pub stream_id: String,
    pub seq: u64,
    /// Byte offset in the file where `data` starts.
    pub offset: u64,
    pub data: String,
    /// The file's size when the chunk was read.
    pub size: u64,
    /// The file was truncated or replaced; `data` starts it again.
    pub rotated: bool,
}

#[derive(Debug, Clone, Copy, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum LogEndReason {
    Done,
    Closed,
    Rejected,
    Error,
    Limit,
}

/// Sent by the agent in `log_end`, once per `log_open`.
#[derive(Debug, Clone, PartialEq, Serialize)]
pub struct LogEnd {
    pub stream_id: String,
    pub reason: LogEndReason,
    pub error: String,
}

pub async fn read_frame<R: AsyncRead + Unpin>(r: &mut R) -> Result<Msg> {
    let size = r.read_u32().await?;
    if size == 0 || size > MAX_FRAME_SIZE {
        bail!("invalid frame size {size}");
    }
    let mut buf = vec![0u8; size as usize];
    r.read_exact(&mut buf).await?;
    serde_json::from_slice(&buf).context("decode frame")
}

pub async fn write_frame<W: AsyncWrite + Unpin>(w: &mut W, msg: &Msg) -> Result<()> {
    let body = serde_json::to_vec(msg)?;
    if body.len() > MAX_FRAME_SIZE as usize {
        bail!("frame too large: {} bytes", body.len());
    }
    let mut buf = Vec::with_capacity(4 + body.len());
    buf.extend_from_slice(&(body.len() as u32).to_be_bytes());
    buf.extend_from_slice(&body);
    w.write_all(&buf).await?;
    w.flush().await?;
    Ok(())
}

#[derive(Debug, Serialize)]
pub struct BackupStarted {
    pub run_id: String,
    pub project_path: String,
    /// `manual` or `schedule`.
    pub trigger: &'static str,
    pub started_at: u64,
    pub run_as: String,
}

#[derive(Debug, Default, Serialize)]
pub struct BackupResult {
    pub run_id: String,
    pub project_path: String,
    pub trigger: &'static str,
    /// `ok`, `failed` or `rejected`.
    pub status: &'static str,
    pub started_at: u64,
    pub duration_ms: u64,
    pub file: String,
    pub local_path: String,
    pub size: u64,
    pub database: String,
    pub upload: crate::backupchild::Upload,
    pub error: String,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn project_settings_defaults_and_clamp() {
        let parse = |json: &str| serde_json::from_str::<ProjectSettings>(json).unwrap();
        assert_eq!(parse("{}"), ProjectSettings::default());
        let s = parse(r#"{"auto_scan": false}"#);
        assert!(!s.auto_scan);
        assert_eq!(s.interval(), Duration::from_secs(900));
        assert_eq!(
            parse(r#"{"scan_interval": 5}"#).interval(),
            Duration::from_secs(60)
        );
        let week = parse(r#"{"scan_interval": 604800}"#);
        assert_eq!(week.interval(), Duration::from_secs(86_400));
    }

    #[tokio::test]
    async fn frame_round_trip() {
        let msg = Msg::new(MSG_CONNECTED)
            .with_token("id:key")
            .with_data(&serde_json::json!({"os": "linux"}))
            .unwrap();
        let mut buf = Vec::new();
        write_frame(&mut buf, &msg).await.unwrap();
        assert_eq!(&buf[..4], &((buf.len() - 4) as u32).to_be_bytes());

        let got = read_frame(&mut buf.as_slice()).await.unwrap();
        assert_eq!(got, msg);
    }

    #[tokio::test]
    async fn envelope_matches_hub_json() {
        let msg = Msg::new(MSG_SYS_STAT);
        let json = serde_json::to_string(&msg).unwrap();
        assert_eq!(json, r#"{"type":"sys_stat"}"#);

        let hub: Msg = serde_json::from_str(r#"{"type":"sys_stat","node_id":3}"#).unwrap();
        assert_eq!(hub.node_id, 3);
    }

    #[test]
    fn command_payloads_match_protocol_doc() {
        let run: CommandRun = serde_json::from_str(
            r#"{"run_id": "r1", "project_path": "/var/www/shop", "command": "build"}"#,
        )
        .unwrap();
        assert_eq!(run.project_path, PathBuf::from("/var/www/shop"));
        assert_eq!(run.command, "build");

        let out = CommandOutput {
            run_id: "r1".into(),
            seq: 0,
            stream: Stream::Stderr,
            data: "oops\n".into(),
        };
        assert_eq!(
            serde_json::to_value(&out).unwrap(),
            serde_json::json!({"run_id": "r1", "seq": 0, "stream": "stderr", "data": "oops\n"})
        );

        let exit = CommandExit::not_started("r1", ExitStatus::Rejected, "unknown command");
        assert_eq!(
            serde_json::to_value(&exit).unwrap(),
            serde_json::json!({
                "run_id": "r1", "status": "rejected", "exit_code": null, "run_as": "",
                "duration_ms": 0, "truncated": false, "error": "unknown command"
            })
        );
    }

    #[test]
    fn log_payloads_match_protocol_doc() {
        let open: LogOpen = serde_json::from_str(
            r#"{"stream_id": "s1", "project_path": "/var/www/shop", "file": "storage/logs/a.log"}"#,
        )
        .unwrap();
        assert_eq!(
            (open.before, open.max_bytes, open.follow),
            (None, 65536, false)
        );

        let end = LogEnd {
            stream_id: "s1".into(),
            reason: LogEndReason::Rejected,
            error: "no".into(),
        };
        assert_eq!(
            serde_json::to_value(&end).unwrap(),
            serde_json::json!({"stream_id": "s1", "reason": "rejected", "error": "no"})
        );
    }

    #[tokio::test]
    async fn rejects_bad_frame_sizes() {
        for size in [0u32, MAX_FRAME_SIZE + 1] {
            let buf = size.to_be_bytes();
            assert!(read_frame(&mut buf.as_slice()).await.is_err());
        }
    }
}
