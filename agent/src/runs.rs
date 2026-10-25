//! The commands running in this session: checks each `command_run` against
//! "Before running" in docs/PROTOCOL.md, starts it, and tracks it for
//! `command_cancel`. Dropping `Runs` (the session ended) kills every run.

use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::Arc;

use tokio::sync::{Notify, mpsc};
use tokio::task::{Id, JoinSet};
use tracing::{info, warn};

use crate::commands::{self, Limits, RunAs};
use crate::project::ProjectConfig;
use crate::protocol::{CommandExit, CommandRun, ExitStatus, MSG_COMMAND_EXIT, Msg};

/// Most commands running at once on this node.
const MAX_RUNNING: usize = 4;

struct Running {
    task: Id,
    project: PathBuf,
    command: String,
    cancel: Arc<Notify>,
}

pub struct Runs {
    limits: Arc<Limits>,
    out: mpsc::Sender<Msg>,
    tasks: JoinSet<()>,
    running: HashMap<String, Running>,
}

impl Runs {
    pub fn new(limits: Limits, out: mpsc::Sender<Msg>) -> Self {
        Runs {
            limits: Arc::new(limits),
            out,
            tasks: JoinSet::new(),
            running: HashMap::new(),
        }
    }

    /// Starts `run`, or sends `command_exit` with status `rejected`.
    /// `projects` is the agent's latest search.
    pub async fn start(&mut self, run: CommandRun, projects: &[ProjectConfig]) {
        if run.run_id.is_empty() || self.running.contains_key(&run.run_id) {
            warn!(
                run_id = run.run_id,
                "ignoring command_run with an empty or reused run_id"
            );
            return;
        }
        let known = projects
            .iter()
            .any(|p| std::path::Path::new(&p.path) == run.project_path);
        let busy = self
            .running
            .values()
            .any(|r| r.project == run.project_path && r.command == run.command);
        let reason = if !known {
            Some("not a project found by this agent".to_string())
        } else if busy {
            Some(format!("{:?} is already running", run.command))
        } else if self.running.len() >= MAX_RUNNING {
            Some(format!("{MAX_RUNNING} commands are already running"))
        } else {
            None
        };
        if let Some(reason) = reason {
            let exit = CommandExit::not_started(&run.run_id, ExitStatus::Rejected, reason);
            send_exit(&self.out, exit).await;
            return;
        }

        let cancel = Arc::new(Notify::new());
        let (limits, out) = (self.limits.clone(), self.out.clone());
        let task = self
            .tasks
            .spawn(execute(run.clone(), limits, cancel.clone(), out));
        self.running.insert(
            run.run_id,
            Running {
                task: task.id(),
                project: run.project_path,
                command: run.command,
                cancel,
            },
        );
    }

    pub fn cancel(&self, run_id: &str) {
        if let Some(r) = self.running.get(run_id) {
            r.cancel.notify_one();
        }
    }

    /// Waits for a run to finish and forgets it. Never returns while
    /// nothing is running, so it can sit in a `select!`.
    pub async fn reap(&mut self) {
        match self.tasks.join_next_with_id().await {
            Some(Ok((id, ()))) => self.running.retain(|_, r| r.task != id),
            Some(Err(e)) => self.running.retain(|_, r| r.task != e.id()),
            None => std::future::pending().await,
        }
    }
}

async fn execute(
    run: CommandRun,
    limits: Arc<Limits>,
    cancel: Arc<Notify>,
    out: mpsc::Sender<Msg>,
) {
    let dir = run.project_path.clone();
    let loaded = tokio::task::spawn_blocking(move || ProjectConfig::load_for_run(&dir)).await;
    let prepared = match loaded {
        Ok(Ok((config, uid))) => match config.command(&run.command) {
            Some(cmd) => RunAs::owner(uid).map(|who| (cmd.to_string(), who)),
            None => Err(format!("unknown command {:?}", run.command)),
        },
        Ok(Err(e)) => Err(format!("{e:#}")),
        Err(e) => Err(e.to_string()),
    };
    let exit = match prepared {
        Ok((cmd, who)) => {
            info!(run_id = run.run_id, project = %run.project_path.display(), command = run.command, user = who.name, "running command");
            commands::run(
                &run.run_id,
                &run.project_path,
                &cmd,
                who,
                &limits,
                cancel,
                out.clone(),
            )
            .await
        }
        Err(reason) => CommandExit::not_started(&run.run_id, ExitStatus::Rejected, reason),
    };
    info!(run_id = exit.run_id, status = ?exit.status, exit_code = ?exit.exit_code, error = exit.error, "command finished");
    send_exit(&out, exit).await;
}

async fn send_exit(out: &mpsc::Sender<Msg>, exit: CommandExit) {
    if let Ok(msg) = Msg::new(MSG_COMMAND_EXIT).with_data(&exit) {
        let _ = out.send(msg).await;
    }
}
