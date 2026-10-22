//! Decides when to search for projects: at session start, when the hub's
//! settings change, when the hub asks for a rescan, and on a timer while
//! auto scan is on. See "Projects" in docs/PROTOCOL.md.

use std::path::PathBuf;
use std::sync::Arc;

use tokio::sync::{Notify, mpsc, watch};
use tokio::time::{Instant, sleep_until};

use crate::discovery::{self, Found};
use crate::protocol::ProjectSettings;

pub struct ScanResult {
    pub found: Found,
    /// The first search of a session or one the hub asked for: report it
    /// even if nothing changed.
    pub always_report: bool,
}

/// Runs searches until the session drops `tx`, `settings` or `rescan`'s owner.
/// Searches run on the blocking pool, so a slow disk never delays stats.
pub async fn scan_loop(
    dirs: Vec<PathBuf>,
    allow_any_path: bool,
    mut settings: watch::Receiver<ProjectSettings>,
    rescan: Arc<Notify>,
    tx: mpsc::Sender<ScanResult>,
) {
    let mut always_report = true;
    loop {
        let current = settings.borrow_and_update().clone();
        let scan_dirs = dirs.clone();
        let paths = current.paths.clone();
        let search = move || discovery::discover(&scan_dirs, &paths, allow_any_path);
        let Ok(found) = tokio::task::spawn_blocking(search).await else {
            return;
        };
        let result = ScanResult {
            found,
            always_report,
        };
        if tx.send(result).await.is_err() {
            return;
        }

        let next = Instant::now() + current.interval();
        always_report = tokio::select! {
            _ = sleep_until(next), if current.auto_scan => false,
            _ = rescan.notified() => true,
            changed = settings.changed() => match changed {
                Ok(()) => false,
                Err(_) => return,
            },
        };
    }
}

#[cfg(test)]
mod tests {
    use std::time::Duration;

    use super::*;

    /// The next result's `always_report`, or None if no search ran.
    async fn next(rx: &mut mpsc::Receiver<ScanResult>) -> Option<bool> {
        let result = tokio::time::timeout(Duration::from_millis(300), rx.recv()).await;
        result.ok().flatten().map(|r| r.always_report)
    }

    #[tokio::test]
    async fn scans_at_start_on_rescan_and_on_changed_settings() {
        let (settings, settings_rx) = watch::channel(ProjectSettings::default());
        let rescan = Arc::new(Notify::new());
        let (tx, mut rx) = mpsc::channel(1);
        let dirs = vec![PathBuf::from("/nonexistent/vps-pilot-test")];
        tokio::spawn(scan_loop(dirs, false, settings_rx, rescan.clone(), tx));

        assert_eq!(next(&mut rx).await, Some(true), "first scan");
        assert_eq!(next(&mut rx).await, None, "default interval is 15 min");
        rescan.notify_one();
        assert_eq!(next(&mut rx).await, Some(true), "rescan");
        settings.send_modify(|s| s.auto_scan = false);
        assert_eq!(next(&mut rx).await, Some(false), "settings changed");
        // The session only sends settings that differ, so an unchanged value
        // never wakes the scanner.
        settings.send_if_modified(|_| false);
        assert_eq!(next(&mut rx).await, None);
    }
}
