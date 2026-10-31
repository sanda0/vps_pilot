//! When scheduled backups are due. Times in config files are in the
//! server's local time zone. What has run is kept in a file next to the
//! agent's state, so a restart doesn't back up twice and a backup missed
//! while the agent was down runs once when it's back. See "Schedule" in
//! docs/PROTOCOL.md.

use std::collections::BTreeMap;
use std::fs::{self, File};
use std::io::Write;
use std::os::unix::fs::OpenOptionsExt;
use std::path::{Path, PathBuf};

use crate::project::BackupSchedule;

const DAY: u64 = 86_400;

/// The latest scheduled time at or before `now`, in Unix seconds.
pub fn last_slot(now: u64, schedule: BackupSchedule, at: (u32, u32)) -> u64 {
    let (secs_into_day, weekday) = local_time(now);
    slot(now, secs_into_day, weekday, schedule, at)
}

/// `last_slot` given the local time of day and weekday (0 = Sunday).
fn slot(
    now: u64,
    secs_into_day: u64,
    weekday: u32,
    schedule: BackupSchedule,
    at: (u32, u32),
) -> u64 {
    let today = now - secs_into_day + u64::from(at.0 * 3600 + at.1 * 60);
    match schedule {
        BackupSchedule::Daily if today > now => today - DAY,
        BackupSchedule::Daily => today,
        BackupSchedule::Weekly => {
            let sunday = today - u64::from(weekday) * DAY;
            if sunday > now {
                sunday - 7 * DAY
            } else {
                sunday
            }
        }
    }
}

/// Seconds since local midnight and the local weekday of `now`.
fn local_time(now: u64) -> (u64, u32) {
    let t = now as nix::libc::time_t;
    // SAFETY: localtime_r only writes the tm it is given.
    let mut tm: nix::libc::tm = unsafe { std::mem::zeroed() };
    if unsafe { nix::libc::localtime_r(&t, &mut tm) }.is_null() {
        return (now % DAY, ((now / DAY + 4) % 7) as u32); // UTC
    }
    let secs = tm.tm_hour * 3600 + tm.tm_min * 60 + tm.tm_sec;
    (secs as u64, tm.tm_wday as u32)
}

/// For each project directory, the scheduled time its last backup was for.
pub struct LastRuns {
    path: PathBuf,
    runs: BTreeMap<String, u64>,
}

impl LastRuns {
    /// The file next to the agent's state file. A missing or broken file
    /// starts empty.
    pub fn load(state_path: &Path) -> Self {
        let mut name = state_path.as_os_str().to_owned();
        name.push(".backups");
        let path = PathBuf::from(name);
        let runs = fs::read(&path)
            .ok()
            .and_then(|b| serde_json::from_slice(&b).ok())
            .unwrap_or_default();
        LastRuns { path, runs }
    }

    /// The slot that is due for `project`, if any. A project seen for the
    /// first time starts from `now`, so adding a schedule doesn't back up
    /// at once.
    pub fn due(
        &mut self,
        project: &str,
        schedule: BackupSchedule,
        at: (u32, u32),
        now: u64,
    ) -> Option<u64> {
        let slot = last_slot(now, schedule, at);
        match self.runs.get(project) {
            Some(&last) if last >= slot => None,
            Some(_) => Some(slot),
            None => {
                self.mark(project, now);
                None
            }
        }
    }

    /// Records that `project` was backed up for `slot`.
    pub fn mark(&mut self, project: &str, slot: u64) {
        self.runs.insert(project.to_string(), slot);
        if let Err(e) = self.save() {
            tracing::warn!("save {}: {e}", self.path.display());
        }
    }

    fn save(&self) -> std::io::Result<()> {
        let mut tmp = self.path.as_os_str().to_owned();
        tmp.push(".tmp");
        let tmp = PathBuf::from(tmp);
        let mut f = File::options()
            .write(true)
            .create(true)
            .truncate(true)
            .mode(0o600)
            .open(&tmp)?;
        f.write_all(&serde_json::to_vec(&self.runs)?)?;
        f.sync_all()?;
        fs::rename(&tmp, &self.path)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use BackupSchedule::{Daily, Weekly};

    // Saturday 2026-10-31 10:00 local time (any zone: slot() takes local
    // time of day and weekday directly).
    const NOW: u64 = 1_793_440_000;
    const SECS: u64 = 10 * 3600;
    const SATURDAY: u32 = 6;

    #[test]
    fn finds_the_latest_slot() {
        let midnight = NOW - SECS;
        assert_eq!(
            slot(NOW, SECS, SATURDAY, Daily, (3, 0)),
            midnight + 3 * 3600
        );
        assert_eq!(
            slot(NOW, SECS, SATURDAY, Daily, (23, 30)),
            midnight - DAY + 23 * 3600 + 1800
        );
        assert_eq!(slot(NOW, SECS, SATURDAY, Daily, (10, 0)), NOW);
        // Last Sunday, six days back.
        assert_eq!(
            slot(NOW, SECS, SATURDAY, Weekly, (3, 0)),
            midnight - 6 * DAY + 3 * 3600
        );
        // On a Sunday before the time: the Sunday a week before.
        assert_eq!(
            slot(NOW, SECS, 0, Weekly, (11, 0)),
            midnight - 7 * DAY + 11 * 3600
        );
    }

    #[test]
    fn remembers_what_ran() {
        let dir = tempfile::tempdir().unwrap();
        let state = dir.path().join("state.json");
        let mut runs = LastRuns::load(&state);
        // First seen: starts from now, nothing due.
        assert_eq!(runs.due("/srv/shop", Daily, (3, 0), NOW), None);
        assert_eq!(
            runs.due("/srv/shop", Daily, (3, 0), NOW + DAY),
            Some(last_slot(NOW + DAY, Daily, (3, 0)))
        );

        runs.mark("/srv/shop", last_slot(NOW + DAY, Daily, (3, 0)));
        let mut again = LastRuns::load(&state);
        assert_eq!(again.due("/srv/shop", Daily, (3, 0), NOW + DAY), None);
        assert!(dir.path().join("state.json.backups").exists());
    }
}
