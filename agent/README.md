# vps-pilot-agent

The VPS Pilot agent runs on each server. It reports health to the hub over TLS. It's written in Rust, ships as one static binary, and speaks the protocol in [docs/PROTOCOL.md](../docs/PROTOCOL.md).

## Install

From v0.2.0, **Nodes → Add node** in the dashboard gives a one-line command
that runs [install-agent.sh](../site/static/install-agent.sh) and sets the
agent up as a systemd service. Each release also has a `.deb` for amd64 and
arm64. The systemd unit is in [packaging/](packaging/). See the
[agent docs](https://vpspilot.dev/docs/agent/) for all three ways to install.

By hand:

1. In the dashboard, open **Nodes → Add node** and copy the command under
   **Agent already installed?**.
2. Run it on the server:

```sh
VPS_PILOT_TOKEN=vpsp1.… vps-pilot-agent --hub hub.example.com:55001
```

On first run the agent enrolls and saves its credentials to `--state` (default `/var/lib/vps-pilot-agent/state.json`, mode `0600`). After that it no longer needs the token. Restarts reconnect using the saved state.

To move an existing node to a new machine or reinstall the agent, open the node and choose **Re-enroll agent**. The node keeps its history.

## Options

| Flag | Env | Default | |
| --- | --- | --- | --- |
| `--hub` | `VPS_PILOT_HUB` | (required) | Hub address `host[:port]`; port defaults to `55001` |
| `--token` | `VPS_PILOT_TOKEN` | | One-time join token (first run only) |
| `--state` | `VPS_PILOT_STATE` | `/var/lib/vps-pilot-agent/state.json` | Credentials file |
| `--interval` | | `5` | Seconds between stat reports |
| `--backoff` | | `10` | Seconds before reconnecting |
| `--projects-dir` | `VPS_PILOT_PROJECT_DIRS` (comma-separated) | `/var/www`, `/home` | Where to look for `config.vpspilot.json`, up to 4 levels deep. Repeatable. Hidden directories, `node_modules`, `vendor`, `target` and symlinked directories are skipped |
| `--allow-any-project-path` | `VPS_PILOT_ALLOW_ANY_PROJECT_PATH` | off | Accept project paths added in the dashboard anywhere on disk. Off: they must be inside a `--projects-dir` |
| `--command-timeout` | `VPS_PILOT_COMMAND_TIMEOUT` | `600` | Seconds a project command may run before it is killed |

Log level is controlled by `RUST_LOG` (e.g. `RUST_LOG=debug`).

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Stopped (Ctrl-C / SIGINT) |
| `1` | Malformed join token |
| `2` | Invalid command-line arguments |
| `3` | The hub rejected the agent's credentials (key revoked, token used or expired). Restarting won't help; re-enroll. Under systemd, set `RestartPreventExitStatus=3`. |

Network errors never exit. The agent reconnects after `--backoff`.

## Develop

```sh
cargo run -- --hub 127.0.0.1:55001 --state ./tmp/state.json --token vpsp1.…
cargo fmt && cargo clippy --all-targets -- -D warnings && cargo test
```

Static release builds use musl via `cargo zigbuild --release --target x86_64-unknown-linux-musl` (see `.github/workflows/release.yml`).

## Roadmap

- Project discovery (`config.vpspilot.json`) and named command execution (Phase 2)
- Optional [osquery](https://osquery.io) integration for cron, listening ports, processes and packages
