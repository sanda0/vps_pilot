---
title: Agent
weight: 20
description: Install, configure and run vps-pilot-agent, the small Rust program that reports each server's health to the hub.
---

The agent runs on each server you want to watch. It's one Rust binary with no
runtime dependencies. It connects out to the hub over TLS, so the server
doesn't need any open ports of its own.

Every five seconds (`--interval`) it sends:

- CPU usage, one percentage per core
- memory usage, as a percentage
- disk usage of the root filesystem, as a percentage
- network traffic sent and received, in bytes per second, across all
  interfaces except loopback

{{< callout soon >}}
The hub doesn't store or chart disk usage yet. The agent already sends it, so
disk charts and disk alerts will work with agents you install today.
{{< /callout >}}

When it connects, it also sends the hostname, OS, distribution and version,
kernel version, CPU count and total memory. The hostname names the node the
first time it enrolls.

## Install

1. In the dashboard, open **Nodes → Add node** and copy the command.
2. Put `vps-pilot-agent` on the server, for example in `/usr/local/bin/`.
3. Run the command there:

```sh
VPS_PILOT_TOKEN=vpsp1.… vps-pilot-agent --hub hub.example.com:55001
```

On the first run the agent enrolls and writes its credentials to the state
file (`--state`, default `/var/lib/vps-pilot-agent/state.json`, mode `0600`,
in a `0700` directory). From then on it reconnects with that file and ignores
any token you pass. Delete the file to enroll again.

Pass the token in the environment rather than with `--token`, so it doesn't
show up in `ps`.

## Options

Every option can also come from an environment variable, except `--interval`
and `--backoff`.

| Flag | Environment variable | Default | Description |
| --- | --- | --- | --- |
| `--hub` | `VPS_PILOT_HUB` | required | Hub address, `host[:port]`. The port defaults to `55001`. |
| `--token` | `VPS_PILOT_TOKEN` | none | One-time join token. Only needed for the first run. |
| `--state` | `VPS_PILOT_STATE` | `/var/lib/vps-pilot-agent/state.json` | Where the credentials are saved. |
| `--interval` | none | `5` | Seconds between stat reports (at least 1). |
| `--backoff` | none | `10` | Seconds to wait before reconnecting. |
| `--projects-dir` | `VPS_PILOT_PROJECT_DIRS` | `/var/www`, `/home` | Where to look for `config.vpspilot.json`. Repeat the flag, or separate directories with commas in the variable. |
| `--allow-any-project-path` | `VPS_PILOT_ALLOW_ANY_PROJECT_PATH` | off | Accept project paths added in the dashboard anywhere on disk, not only inside a `--projects-dir`. |

Set `RUST_LOG` to change the log level, for example `RUST_LOG=debug`. The
default is `info`.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Stopped with Ctrl-C (SIGINT). |
| `1` | The join token is malformed, or the agent isn't enrolled and no token was given. |
| `2` | Invalid command-line arguments. |
| `3` | The hub rejected the agent's credentials: the key was revoked, or the token was already used or has expired. Restarting won't help. Re-enroll it. |

Network errors never stop the agent. It reconnects after `--backoff` seconds,
for as long as it takes.

## Run it with systemd

There's no packaged service unit yet. This one works with the current agent:

```ini
# /etc/systemd/system/vps-pilot-agent.service
[Unit]
Description=VPS Pilot agent
Wants=network-online.target
After=network-online.target

[Service]
Environment=VPS_PILOT_HUB=hub.example.com
ExecStart=/usr/local/bin/vps-pilot-agent
Restart=on-failure
RestartSec=10
# 3 means the hub rejected the credentials: re-enroll instead of retrying.
RestartPreventExitStatus=3
# The agent shuts down cleanly on SIGINT.
KillSignal=SIGINT

[Install]
WantedBy=multi-user.target
```

Enroll once by hand (step 3 above), then:

```sh
systemctl daemon-reload
systemctl enable --now vps-pilot-agent
journalctl -u vps-pilot-agent -f
```

{{< callout soon >}}
An install script and an official systemd unit for the agent are planned.
{{< /callout >}}

## Re-enroll or revoke

Open the node in the dashboard:

- **Re-enroll agent** creates a join token for that node. Use it to move the
  node to a new machine or reinstall the agent. The node keeps its ID and its
  history, and the old key stops working.
- **Revoke agent key** disconnects the agent straight away. It exits with code
  `3` the next time it tries to connect, and it can only come back with a new
  join token.

If the hub's TLS certificate changes (for example because its data directory
was lost), every agent refuses to connect, because the certificate no longer
matches the pinned fingerprint. Re-enroll each one.

## Build a static binary

`cargo build --release` produces a binary linked against the build machine's
glibc, which may not run on an older server. For a fully static binary, build
for musl the way releases are built:

```sh
pip install ziglang cargo-zigbuild
rustup target add x86_64-unknown-linux-musl
cd agent
cargo zigbuild --release --target x86_64-unknown-linux-musl
```

Use `aarch64-unknown-linux-musl` for ARM servers.
