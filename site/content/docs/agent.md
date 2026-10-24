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

When it connects, it also sends the hostname, OS, distribution and version,
kernel version, CPU count and total memory. The hostname names the node the
first time it enrolls. While the agent is connected, the dashboard shows the
node as **Online**; otherwise it shows **Offline** and when it was last seen.

## Install

### With the install script

In the dashboard, open **Nodes → Add node**, copy the install command and run
it on the server:

```sh
curl -fsSL https://vpspilot.dev/install-agent.sh \
  | sudo VPS_PILOT_TOKEN=vpsp1.… sh -s -- --hub hub.example.com:55001 --version v0.2.0
```

The script:

1. downloads the agent release for the server's architecture (amd64 or
   arm64) and checks it against the release's `SHA256SUMS`;
2. installs `/usr/local/bin/vps-pilot-agent` and the
   [systemd unit](#run-it-with-systemd);
3. saves the hub address in `/etc/vps-pilot-agent/agent.env` (mode `0600`);
4. starts the service, waits for the agent to enroll, then removes the token
   from `agent.env`.

| Option | Default | Description |
| --- | --- | --- |
| `--hub` | required on the first install | Hub address, `host[:port]`. |
| `--version` | latest release | Release to install. The command from the dashboard pins the hub's version. |
| `--projects-dir` | `/var/www,/home` | Comma-separated directories to search for `config.vpspilot.json`. |
| `--token` | `VPS_PILOT_TOKEN` | Join token. Prefer the environment variable, so it doesn't show up in `ps`. |

To upgrade, run the script again, with `--version` for a specific release.
It keeps the enrollment and the saved settings; options you pass replace the
saved ones.

### From the .deb

Each release has `vps-pilot-agent_<version>_amd64.deb` and `_arm64.deb`.

```sh
sudo apt install ./vps-pilot-agent_0.2.0_amd64.deb
```

The package installs `/usr/bin/vps-pilot-agent` and the systemd unit, and
creates an empty `/etc/vps-pilot-agent/agent.env`. To enroll, put the hub
address and the join token from **Nodes → Add node** in that file:

```sh
VPS_PILOT_HUB=hub.example.com:55001
VPS_PILOT_TOKEN=vpsp1.…
```

Then run `sudo systemctl enable --now vps-pilot-agent`. Once the node shows
as online, delete the `VPS_PILOT_TOKEN` line. Installing a newer `.deb`
upgrades the agent and restarts it.

### By hand

1. In the dashboard, open **Nodes → Add node** and copy the command under
   **Agent already installed?**.
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

The install script and the `.deb` install this unit. It is also in every
agent release archive, as `vps-pilot-agent.service`. The `.deb` runs
`/usr/bin/vps-pilot-agent` instead.

```ini
# /etc/systemd/system/vps-pilot-agent.service
[Unit]
Description=VPS Pilot agent
Documentation=https://vpspilot.dev/docs/agent/
Wants=network-online.target
After=network-online.target

[Service]
# VPS_PILOT_HUB, and VPS_PILOT_TOKEN until the agent has enrolled.
EnvironmentFile=/etc/vps-pilot-agent/agent.env
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

Settings go in `/etc/vps-pilot-agent/agent.env`, one `NAME=value` per line,
using the [environment variables](#options) above. After changing it, run
`sudo systemctl restart vps-pilot-agent`.

To use the unit with an agent installed by hand, copy it to
`/etc/systemd/system/`, create `agent.env` with at least `VPS_PILOT_HUB`,
enroll once (step 3 above), then:

```sh
systemctl daemon-reload
systemctl enable --now vps-pilot-agent
journalctl -u vps-pilot-agent -f
```

## Uninstall

- **`.deb`:** `sudo apt remove vps-pilot-agent` stops the agent and keeps its
  enrollment. `sudo apt purge vps-pilot-agent` also deletes
  `/etc/vps-pilot-agent` and the credentials in `/var/lib/vps-pilot-agent`.
- **Install script or by hand:**

  ```sh
  sudo systemctl disable --now vps-pilot-agent
  sudo rm -rf /usr/local/bin/vps-pilot-agent \
    /etc/systemd/system/vps-pilot-agent.service \
    /etc/vps-pilot-agent /var/lib/vps-pilot-agent
  sudo systemctl daemon-reload
  ```

The node and its history stay in the hub. Use **Revoke agent key** on the
node so a copy of the old credentials can't connect.

## Re-enroll or revoke

Open the node in the dashboard:

- **Re-enroll agent** creates a join token for that node. Use it to move the
  node to a new machine or reinstall the agent. The node keeps its ID and its
  history, and the old key stops working. On a machine that is already
  enrolled, delete `/var/lib/vps-pilot-agent/state.json` first: the agent
  ignores a token while it has credentials.
- **Revoke agent key** disconnects the agent straight away. It exits with code
  `3` the next time it tries to connect, and it can only come back with a new
  join token.

Projects found under `--projects-dir` are covered in
[Projects](/docs/projects/).

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

The agent talks to the hub over a documented [protocol](/docs/protocol/), so
you can also write your own.
