//! Runs one project command as its config file's owner and streams the
//! output. See "Run commands" in docs/PROTOCOL.md.

use std::ffi::CString;
use std::path::{Path, PathBuf};
use std::process::Stdio;
use std::sync::Arc;
use std::time::Duration;

use nix::sys::signal::{Signal, killpg};
use nix::unistd::{Gid, Pid, Uid, User};
use tokio::io::AsyncReadExt;
use tokio::process::{Child, Command};
use tokio::sync::{Notify, mpsc};
use tokio::time::{Instant, sleep, sleep_until};

use crate::output::Emitter;
use crate::protocol::{CommandExit, ExitStatus, Msg, Stream};

const PATH: &str = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin";
const CHUNK: usize = 32 * 1024;
/// How long to wait after SIGTERM before SIGKILL.
const KILL_GRACE: Duration = Duration::from_secs(5);
/// How long output is still read after the command exits, for children it
/// left running that hold stdout or stderr open.
const DRAIN: Duration = Duration::from_secs(2);

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

/// Runs `command` with `/bin/sh -c` in `dir` and sends its output to `out`.
/// Returns the `command_exit` payload; the caller sends it.
pub async fn run(
    run_id: &str,
    dir: &Path,
    command: &str,
    run_as: RunAs,
    limits: &Limits,
    cancel: Arc<Notify>,
    out: mpsc::Sender<Msg>,
) -> CommandExit {
    let mut cmd = Command::new("/bin/sh");
    cmd.arg("-c")
        .arg(command)
        .current_dir(dir)
        .env_clear()
        .env("HOME", &run_as.home)
        .env("USER", &run_as.name)
        .env("LOGNAME", &run_as.name)
        .env("SHELL", &run_as.shell)
        .env("PATH", PATH)
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .process_group(0);
    if let Some((uid, gid, groups)) = run_as.switch.clone() {
        // SAFETY: runs between fork and exec, and only makes the
        // async-signal-safe setgroups, setgid and setuid calls.
        unsafe {
            cmd.pre_exec(move || {
                nix::unistd::setgroups(&groups)?;
                nix::unistd::setgid(gid)?;
                nix::unistd::setuid(uid)?;
                Ok(())
            });
        }
    }

    let started = Instant::now();
    let mut child = match cmd.spawn() {
        Ok(child) => child,
        Err(e) => {
            return CommandExit::not_started(run_id, ExitStatus::Failed, format!("start: {e}"));
        }
    };
    let mut group = Group {
        pgid: Pid::from_raw(child.id().unwrap_or_default() as i32),
        armed: true,
    };
    let (mut stdout, mut stderr) = (child.stdout.take().unwrap(), child.stderr.take().unwrap());
    let (mut out_buf, mut err_buf) = (vec![0u8; CHUNK], vec![0u8; CHUNK]);
    let (mut out_open, mut err_open) = (true, true);
    let mut emitter = Emitter::new(run_id, out, CHUNK, limits.max_output);
    let deadline = started + limits.timeout;
    let mut exited = None;
    let mut drain_until = deadline;

    let status = loop {
        if exited.is_some() && !out_open && !err_open {
            break ExitStatus::Exited;
        }
        tokio::select! {
            n = stdout.read(&mut out_buf), if out_open => {
                let n = n.unwrap_or(0);
                out_open = n > 0;
                emitter.push(Stream::Stdout, &out_buf[..n], !out_open).await;
            }
            n = stderr.read(&mut err_buf), if err_open => {
                let n = n.unwrap_or(0);
                err_open = n > 0;
                emitter.push(Stream::Stderr, &err_buf[..n], !err_open).await;
            }
            st = child.wait(), if exited.is_none() => {
                exited = Some(st);
                drain_until = (Instant::now() + DRAIN).min(deadline);
            }
            _ = sleep_until(drain_until), if exited.is_some() => break ExitStatus::Exited,
            _ = sleep_until(deadline), if exited.is_none() => break ExitStatus::Timeout,
            _ = cancel.notified() => break ExitStatus::Cancelled,
        }
    };

    let exit_code = match status {
        ExitStatus::Exited => {
            // Children it left running keep going, as they would from a shell.
            group.armed = false;
            match exited {
                Some(Ok(st)) => exit_code(st),
                Some(Err(e)) => {
                    let mut exit =
                        CommandExit::not_started(run_id, ExitStatus::Failed, e.to_string());
                    exit.run_as = run_as.name;
                    return exit;
                }
                None => None,
            }
        }
        _ => {
            group.kill(&mut child).await;
            None
        }
    };
    CommandExit {
        run_id: run_id.to_string(),
        status,
        exit_code,
        run_as: run_as.name,
        duration_ms: started.elapsed().as_millis() as u64,
        truncated: emitter.truncated,
        error: String::new(),
    }
}

/// The exit code, or 128 + n when killed by signal n (as a shell reports it).
fn exit_code(st: std::process::ExitStatus) -> Option<i32> {
    use std::os::unix::process::ExitStatusExt;
    st.code().or_else(|| st.signal().map(|s| 128 + s))
}

#[cfg(test)]
mod tests {
    use super::*;

    async fn run_sh(command: &str, limits: Limits, cancel: Arc<Notify>) -> (CommandExit, String) {
        let dir = tempfile::tempdir().unwrap();
        let me = RunAs::owner(nix::unistd::getuid().as_raw()).unwrap();
        let (tx, mut rx) = mpsc::channel(64);
        let exit = run("r1", dir.path(), command, me, &limits, cancel, tx).await;
        let mut text = String::new();
        while let Ok(msg) = rx.try_recv() {
            text.push_str(msg.data.unwrap()["data"].as_str().unwrap());
        }
        (exit, text)
    }

    fn limits(timeout_ms: u64) -> Limits {
        Limits {
            timeout: Duration::from_millis(timeout_ms),
            max_output: 1 << 20,
        }
    }

    #[test]
    fn owner_lookup() {
        let me = RunAs::owner(nix::unistd::getuid().as_raw()).unwrap();
        assert!(me.switch.is_none() && !me.name.is_empty());
        let err = RunAs::owner(u32::MAX - 7).unwrap_err();
        assert!(err.contains("no passwd entry"), "{err}");
    }

    #[tokio::test]
    async fn streams_output_and_exit_code() {
        let cmd = r#"echo "out $PWD"; echo err >&2; test -z "$VPS_PILOT_TOKEN"; exit 3"#;
        let (exit, text) = run_sh(cmd, limits(10_000), Arc::default()).await;
        assert_eq!(exit.status, ExitStatus::Exited);
        assert_eq!(exit.exit_code, Some(3));
        assert!(text.contains("out /") && text.contains("err\n"), "{text:?}");
        assert!(!exit.run_as.is_empty() && !exit.truncated);
    }

    #[tokio::test]
    async fn kills_on_timeout() {
        let (exit, _) = run_sh("sleep 30", limits(200), Arc::default()).await;
        assert_eq!(exit.status, ExitStatus::Timeout);
        assert_eq!(exit.exit_code, None);
        assert!(exit.duration_ms < 5_000);
    }

    #[tokio::test]
    async fn cancels() {
        let cancel = Arc::new(Notify::new());
        cancel.notify_one();
        let (exit, _) = run_sh("sleep 30", limits(10_000), cancel).await;
        assert_eq!(exit.status, ExitStatus::Cancelled);
    }

    #[tokio::test]
    async fn reports_signal_exit_and_truncation() {
        let cmd = "head -c 100000 /dev/zero | tr '\\0' x; kill -9 $$";
        let mut l = limits(10_000);
        l.max_output = 1000;
        let (exit, text) = run_sh(cmd, l, Arc::default()).await;
        assert_eq!(exit.exit_code, Some(137));
        assert!(exit.truncated && text.len() < 100_000);
    }
}
