//! The log child process: the agent runs itself as `vps-pilot-agent __log
//! …` as a config file's owner, so listing and reading logs only ever use
//! that user's permissions (see logs.rs). It writes JSON to stdout.

use std::io::{self, Write};
use std::path::PathBuf;
use std::process::ExitCode;
use std::time::Duration;

use serde::{Deserialize, Serialize};

use crate::logfiles;
use crate::logread::{self, Request};
use crate::protocol::LogFile;

/// First argument that makes the agent binary act as the log child.
pub const ARG: &str = "__log";
/// Exit code for a request the rules refuse; stderr says why.
pub const EXIT_REJECTED: u8 = 2;
const POLL: Duration = Duration::from_secs(1);

#[derive(Debug, Clone, PartialEq)]
pub enum Job {
    List,
    Read {
        file: String,
        before: Option<u64>,
        max_bytes: u64,
        follow: bool,
    },
}

/// What `list` writes.
#[derive(Debug, Serialize, Deserialize)]
pub struct Listing {
    pub files: Vec<LogFile>,
    pub truncated: bool,
}

/// Parses `__log <project> list -- <folders…>` or
/// `__log <project> read <file> <before|-> <max_bytes> <0|1> -- <folders…>`.
fn parse(args: &[String]) -> Option<(PathBuf, Vec<String>, Job)> {
    let sep = args.iter().position(|a| a == "--")?;
    let (head, logs) = (&args[..sep], args[sep + 1..].to_vec());
    let job = match head {
        [_, _, list] if list == "list" => Job::List,
        [_, _, read, file, before, max, follow] if read == "read" => Job::Read {
            file: file.clone(),
            before: if before == "-" {
                None
            } else {
                Some(before.parse().ok()?)
            },
            max_bytes: max.parse().ok()?,
            follow: follow == "1",
        },
        _ => return None,
    };
    Some((PathBuf::from(&head[1]), logs, job))
}

/// Entry point when the first argument is ARG.
pub fn main() -> ExitCode {
    let args: Vec<String> = std::env::args().skip(1).collect();
    match run(&args, &mut io::stdout().lock()) {
        Ok(()) => ExitCode::SUCCESS,
        Err((code, msg)) => {
            eprintln!("{msg}");
            ExitCode::from(code)
        }
    }
}

fn run(args: &[String], out: &mut impl Write) -> Result<(), (u8, String)> {
    let (project, logs, job) = parse(args).ok_or((1, "bad arguments".to_string()))?;
    let folders = logfiles::folders(&project, &logs);
    let failed = |e: io::Error| (1, e.to_string());
    match job {
        Job::List => {
            let (files, truncated) = logfiles::list(&folders);
            serde_json::to_writer(&mut *out, &Listing { files, truncated })
                .map_err(|e| failed(e.into()))
        }
        Job::Read {
            file,
            before,
            max_bytes,
            follow,
        } => {
            logfiles::resolve(&project, &folders, &file).map_err(|e| (EXIT_REJECTED, e))?;
            let open = || logfiles::resolve(&project, &folders, &file).map_err(io::Error::other);
            let req = Request {
                before,
                max_bytes,
                follow,
                poll: POLL,
            };
            match logread::read(open, &req, out) {
                // The agent closed the pipe: the stream is over.
                Err(e) if e.kind() == io::ErrorKind::BrokenPipe => Ok(()),
                other => other.map_err(failed),
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    #[test]
    fn lists_and_rejects() {
        let dir = tempfile::tempdir().unwrap();
        fs::create_dir(dir.path().join("logs")).unwrap();
        fs::write(dir.path().join("logs/app.log"), "hi\n").unwrap();
        let project = dir.path().to_str().unwrap();
        let args = |job: &[&str]| -> Vec<String> {
            let mut a = vec![ARG, project];
            a.extend(job);
            a.extend(["--", "logs"]);
            a.into_iter().map(String::from).collect()
        };

        let mut out = Vec::new();
        run(&args(&["list"]), &mut out).unwrap();
        let listing: Listing = serde_json::from_slice(&out).unwrap();
        assert_eq!(listing.files[0].path, "logs/app.log");

        let read = ["read", "../etc/passwd", "-", "10", "0"];
        let err = run(&args(&read), &mut Vec::new()).unwrap_err();
        assert_eq!(err.0, EXIT_REJECTED, "{err:?}");
    }
}
