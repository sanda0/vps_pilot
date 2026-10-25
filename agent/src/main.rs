mod client;
mod commands;
mod discovery;
mod metrics;
mod output;
mod project;
mod protocol;
mod runs;
mod scanner;
mod state;
mod tls;

use std::path::PathBuf;
use std::process::ExitCode;
use std::time::Duration;

use clap::Parser;
use tracing::{error, info, warn};
use tracing_subscriber::EnvFilter;

use client::{Config, JoinToken, SessionError};
use metrics::Collector;

/// Exit code when the hub rejects the agent's credentials. A service manager
/// should not restart on it (systemd: RestartPreventExitStatus=3). Code 2 is
/// clap's usage error.
const EXIT_AUTH_FAILED: u8 = 3;

const DEFAULT_AGENT_PORT: u16 = 55001;

/// The release tag when built by release.yml (VPS_PILOT_VERSION=v0.1.0),
/// otherwise the Cargo.toml version.
pub const VERSION: &str = match option_env!("VPS_PILOT_VERSION") {
    Some(v) => v,
    None => env!("CARGO_PKG_VERSION"),
};

/// VPS Pilot agent: reports this server's health to a VPS Pilot hub.
#[derive(Parser, Debug)]
#[command(version = VERSION)]
struct Args {
    /// Hub agent-channel address, host[:port]
    #[arg(long, env = "VPS_PILOT_HUB")]
    hub: String,

    /// One-time join token from the dashboard. Only needed until the agent
    /// has enrolled. Prefer the env var so it doesn't show in `ps`.
    #[arg(long, env = "VPS_PILOT_TOKEN", hide_env_values = true)]
    token: Option<String>,

    /// Where enrolled credentials are stored
    #[arg(
        long,
        env = "VPS_PILOT_STATE",
        default_value = "/var/lib/vps-pilot-agent/state.json"
    )]
    state: PathBuf,

    /// Seconds between stat reports
    #[arg(long, default_value_t = 5, value_parser = clap::value_parser!(u64).range(1..))]
    interval: u64,

    /// Seconds to wait before reconnecting
    #[arg(long, default_value_t = 10)]
    backoff: u64,

    /// Directory searched for config.vpspilot.json files (repeatable)
    #[arg(
        long = "projects-dir",
        env = "VPS_PILOT_PROJECT_DIRS",
        value_delimiter = ',',
        default_values = ["/var/www", "/home"]
    )]
    projects_dirs: Vec<PathBuf>,

    /// Accept project paths added from the dashboard anywhere on disk, not
    /// only inside --projects-dir
    #[arg(long, env = "VPS_PILOT_ALLOW_ANY_PROJECT_PATH")]
    allow_any_project_path: bool,

    /// Seconds a project command may run before it is killed
    #[arg(long, env = "VPS_PILOT_COMMAND_TIMEOUT", default_value_t = 600, value_parser = clap::value_parser!(u64).range(1..))]
    command_timeout: u64,
}

/// Appends the default agent port when the hub address has none.
fn hub_address(hub: &str) -> String {
    let has_port = match hub.rsplit_once(':') {
        Some((host, port)) => !host.ends_with(':') && port.parse::<u16>().is_ok(),
        None => false,
    };
    if has_port || hub.starts_with('[') && hub.contains("]:") {
        hub.to_string()
    } else {
        format!("{hub}:{DEFAULT_AGENT_PORT}")
    }
}

/// Resolves on SIGINT or SIGTERM. Returning from main then ends the session,
/// which stops running project commands; the default SIGTERM action would
/// kill the agent and leave them running.
async fn shutdown_signal() {
    use tokio::signal::unix::{SignalKind, signal};
    match signal(SignalKind::terminate()) {
        Ok(mut term) => tokio::select! {
            _ = tokio::signal::ctrl_c() => {}
            _ = term.recv() => {}
        },
        Err(e) => {
            warn!("can't handle SIGTERM: {e}");
            let _ = tokio::signal::ctrl_c().await;
        }
    }
}

#[tokio::main]
async fn main() -> ExitCode {
    tracing_subscriber::fmt()
        .with_env_filter(
            EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info")),
        )
        .init();

    let args = Args::parse();
    info!("VPS Pilot agent {VERSION}");
    // An empty token (e.g. VPS_PILOT_TOKEN= in an env file) means "no token".
    let token = args
        .token
        .as_deref()
        .map(str::trim)
        .filter(|t| !t.is_empty());
    let join = match token.map(JoinToken::parse).transpose() {
        Ok(join) => join,
        Err(e) => {
            error!("{e:#}");
            return ExitCode::FAILURE;
        }
    };
    if join.is_none() && !args.state.exists() {
        error!(
            state = %args.state.display(),
            "not enrolled yet: create a join token in the dashboard (Nodes → Add node) and pass it with --token or VPS_PILOT_TOKEN"
        );
        return ExitCode::FAILURE;
    }
    if join.is_some() && args.state.exists() {
        warn!(
            state = %args.state.display(),
            "already enrolled; ignoring --token (delete the state file to re-enroll)"
        );
    }

    let cfg = Config {
        hub: hub_address(&args.hub),
        state_path: args.state,
        join,
        interval: Duration::from_secs(args.interval),
        project_dirs: args.projects_dirs,
        allow_any_project_path: args.allow_any_project_path,
        command_timeout: Duration::from_secs(args.command_timeout),
    };
    let backoff = Duration::from_secs(args.backoff);
    let mut collector = Collector::new();

    loop {
        tokio::select! {
            result = client::run_session(&cfg, &mut collector) => match result {
                Err(SessionError::AuthFailed(reason)) => {
                    error!("hub rejected this agent: {reason}");
                    return ExitCode::from(EXIT_AUTH_FAILED);
                }
                Err(SessionError::Disconnected(e)) => {
                    warn!("disconnected: {e:#}; retrying in {backoff:?}");
                }
                Ok(()) => {}
            },
            _ = shutdown_signal() => {
                info!("shutting down");
                return ExitCode::SUCCESS;
            }
        }

        tokio::select! {
            _ = tokio::time::sleep(backoff) => {}
            _ = shutdown_signal() => return ExitCode::SUCCESS,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn hub_address_defaults_port() {
        assert_eq!(hub_address("hub.example.com"), "hub.example.com:55001");
        assert_eq!(hub_address("10.0.0.1:6000"), "10.0.0.1:6000");
        assert_eq!(hub_address("[::1]:6000"), "[::1]:6000");
    }
}
