//! The commands running on this node, from Run and from cron jobs, so
//! both keep to "Before running" in docs/PROTOCOL.md: the same command in
//! the same project runs once at a time, and at most 4 run at once.

use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex};

/// Most commands running at once on this node.
pub const MAX_RUNNING: usize = 4;

#[derive(Clone, Default)]
pub struct Slots(Arc<Mutex<Vec<(PathBuf, String)>>>);

/// A running command's place. Dropping it frees the place.
pub struct Slot {
    slots: Slots,
    project: PathBuf,
    command: String,
}

impl Slots {
    /// Takes a place for `command` in `project`, or says why there's none.
    pub fn take(&self, project: &Path, command: &str) -> Result<Slot, String> {
        let mut running = self.0.lock().unwrap_or_else(|e| e.into_inner());
        if running.iter().any(|(p, c)| p == project && c == command) {
            return Err(format!("{command:?} is already running"));
        }
        if running.len() >= MAX_RUNNING {
            return Err(format!("{MAX_RUNNING} commands are already running"));
        }
        running.push((project.to_path_buf(), command.to_string()));
        Ok(Slot {
            slots: self.clone(),
            project: project.to_path_buf(),
            command: command.to_string(),
        })
    }
}

impl Drop for Slot {
    fn drop(&mut self) {
        let mut running = self.slots.0.lock().unwrap_or_else(|e| e.into_inner());
        if let Some(i) = running
            .iter()
            .position(|(p, c)| *p == self.project && *c == self.command)
        {
            running.swap_remove(i);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn one_run_per_command_and_four_in_all() {
        let slots = Slots::default();
        let a = Path::new("/srv/a");
        let first = slots.take(a, "build").unwrap();
        let err = slots.take(a, "build").err().unwrap();
        assert_eq!(err, "\"build\" is already running");
        let others: Vec<_> = ["x", "y", "z"]
            .iter()
            .map(|c| slots.take(a, c).unwrap())
            .collect();
        let err = slots.take(Path::new("/srv/b"), "build").err().unwrap();
        assert_eq!(err, "4 commands are already running");
        drop(first);
        assert!(slots.take(a, "build").is_ok());
        drop(others);
        assert!(slots.0.lock().unwrap().is_empty());
    }
}
