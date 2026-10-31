//! Backups, run by the agent itself: on the schedules in config files, and
//! when the hub asks (Back up now). This task outlives sessions, so backups
//! keep running and their results queue while the hub is unreachable. One
//! backup runs at a time. See "Back up" in docs/PROTOCOL.md.

use std::path::Path;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use tokio::sync::mpsc;
use tracing::warn;

use crate::backupchild::Upload;
use crate::backuprun;
use crate::protocol::{BackupResult, BackupStarted, MSG_BACKUP_STARTED, Msg};

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

    #[test]
    fn makes_uuids() {
        let (a, b) = (new_id(), new_id());
        assert_ne!(a, b);
        assert_eq!((a.len(), &a[14..15]), (36, "4"));
    }
}
