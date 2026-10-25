//! Runs one project command as its config file's owner and streams the
//! output. See "Run commands" in docs/PROTOCOL.md.

use std::ffi::CString;
use std::path::PathBuf;
use std::time::Duration;

use nix::sys::signal::{Signal, killpg};
use nix::unistd::{Gid, Pid, Uid, User};
use tokio::process::Child;
use tokio::time::sleep;

/// How long to wait after SIGTERM before SIGKILL.
const KILL_GRACE: Duration = Duration::from_secs(5);

pub struct Limits {
    pub timeout: Duration,
    pub max_output: usize,
}

/// The user a command runs as.
#[derive(Debug, Clone)]
pub struct RunAs {
    pub name: String,
    home: PathBuf,
    shell: PathBuf,
    /// uid, gid and supplementary groups to switch to; None when the agent
    /// already runs as this user.
    switch: Option<(Uid, Gid, Vec<Gid>)>,
}

impl RunAs {
    /// The owner of a config file, if this agent can run commands as them:
    /// it runs as root, or already as that user.
    pub fn owner(uid: u32) -> Result<RunAs, String> {
        let uid = Uid::from_raw(uid);
        let user = User::from_uid(uid)
            .ok()
            .flatten()
            .ok_or_else(|| format!("owner uid {uid} has no passwd entry"))?;
        let me = nix::unistd::geteuid();
        let switch = if me == uid {
            None
        } else if me.is_root() {
            let name = CString::new(user.name.as_str()).map_err(|e| e.to_string())?;
            let groups = nix::unistd::getgrouplist(&name, user.gid)
                .map_err(|e| format!("groups of {}: {e}", user.name))?;
            Some((uid, user.gid, groups))
        } else {
            return Err(format!(
                "the agent isn't root, so it can't run commands as {}",
                user.name
            ));
        };
        Ok(RunAs {
            name: user.name,
            home: user.dir,
            shell: user.shell,
            switch,
        })
    }
}

/// Kills the command's process group if the run ends early, including when
/// its task is aborted because the session ended.
struct Group {
    pgid: Pid,
    armed: bool,
}

impl Group {
    async fn kill(&mut self, child: &mut Child) {
        let _ = killpg(self.pgid, Signal::SIGTERM);
        if tokio::time::timeout(KILL_GRACE, child.wait())
            .await
            .is_err()
        {
            let _ = killpg(self.pgid, Signal::SIGKILL);
        }
        self.armed = false;
    }
}

impl Drop for Group {
    fn drop(&mut self) {
        if !self.armed {
            return;
        }
        let _ = killpg(self.pgid, Signal::SIGTERM);
        let pgid = self.pgid;
        match tokio::runtime::Handle::try_current() {
            Ok(rt) => drop(rt.spawn(async move {
                sleep(KILL_GRACE).await;
                let _ = killpg(pgid, Signal::SIGKILL);
            })),
            Err(_) => drop(killpg(pgid, Signal::SIGKILL)),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn owner_lookup() {
        let me = RunAs::owner(nix::unistd::getuid().as_raw()).unwrap();
        assert!(me.switch.is_none() && !me.name.is_empty());
        let err = RunAs::owner(u32::MAX - 7).unwrap_err();
        assert!(err.contains("no passwd entry"), "{err}");
    }
}
