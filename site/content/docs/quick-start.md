---
title: Quick start
weight: 10
description: Build the hub, create your login and connect your first server in about ten minutes.
---

There are no prebuilt releases yet, so you build VPS Pilot from source. You
need:

- [Go](https://go.dev/doc/install) 1.25 or newer, for the hub
- [Bun](https://bun.sh/), for the dashboard
- [Rust](https://rustup.rs/) (stable), for the agent
- Git and OpenSSL

The agent runs on Linux. The hub runs anywhere Go does. More detail on each
step is in [Building from source](/docs/building/).

## 1. Clone the repository

```sh
git clone https://github.com/sanda0/vps_pilot.git
cd vps_pilot
```

## 2. Build

```sh
./build.sh
```

`make build` does the same. The script installs the dashboard's dependencies
if needed, builds the dashboard straight into `server/cmd/app/dist/`, compiles
the hub with the dashboard embedded, and builds the agent (skipped if `cargo`
isn't installed). You get two binaries:

- `server/vps_pilot`: the hub
- `agent/target/release/vps-pilot-agent`: the agent

## 3. Configure the hub

```sh
cd server
cp .env.example .env
openssl rand -hex 32
```

Paste the random string into `.env` as `TOKEN_SECRET`. It signs dashboard
logins.

{{< callout warning >}}
Don't skip `TOKEN_SECRET`. If it's empty, the hub falls back to a built-in
default secret that anyone can read in the source code, and logs a warning.
{{< /callout >}}

The hub reads `.env` from the directory you start it in, and keeps its data
in `./data` by default, so run the commands below from `server/`. Every
setting is described in [Configuration](/docs/configuration/).

## 4. Create your login

```sh
./vps_pilot -create-superuser
```

It asks for an email address and a password. You log in with the email. Note
that the command prints both values back once it's done, so run it in a
terminal nobody is watching.

The database migrations run on every start. `./vps_pilot -migrate` runs them
on their own and exits, if you want to do that first.

## 5. Start the hub

```sh
./vps_pilot
```

The dashboard listens on port `8080` and the agent channel on `55001`. Use
`./vps_pilot -port 9090` for another dashboard port.

## 6. Open the dashboard

Go to <http://localhost:8080> (or your server's address) and log in with the
email and password from step 4.

## 7. Connect a server

1. In the dashboard, open **Nodes → Add node**. It shows a ready-to-run
   command with a join token. The token works once and expires after an hour.
2. Copy the agent binary to the server, for example to
   `/usr/local/bin/vps-pilot-agent`.
3. Run the command on the server. It saves its credentials under `/var/lib`,
   so run it as root or add `--state` with a path you can write:

```sh
VPS_PILOT_TOKEN=vpsp1.… vps-pilot-agent --hub hub.example.com:55001
```

The host in the command is the one you opened the dashboard with. If that was
`localhost`, replace it with an address the server can reach.

The agent enrolls, saves its credentials to
`/var/lib/vps-pilot-agent/state.json`, and starts sending stats every five
seconds. It doesn't need the token again: later runs are just
`vps-pilot-agent --hub hub.example.com`. The server must be able to reach the
hub on port `55001`.

The new node appears under **Nodes**, named after the server's hostname.

## Next steps

- Keep the agent running with systemd: see [Agent](/docs/agent/#run-it-with-systemd).
- Put the dashboard behind HTTPS: see [Configuration](/docs/configuration/#run-the-hub-in-production).
- Add a `config.vpspilot.json` to your projects: see [Projects](/docs/projects/).
- Get a message when a server runs hot: see [Alerts](/docs/alerts/).
