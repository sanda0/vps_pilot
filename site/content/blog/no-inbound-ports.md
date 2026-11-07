---
title: Monitoring a VPS without opening a single inbound port
date: 2026-11-07
description: The VPS Pilot agent dials out to the hub over TLS, so the servers you monitor accept no new connections at all. Here's how enrollment works with a one-time join token, and why that beats exposing an agent port.
---

Most monitoring setups ask every server to listen. Prometheus's
`node_exporter` serves metrics on port 9100 and waits to be scraped. Netdata's
agent serves its dashboard on 19999. Zabbix's agent, in passive mode, waits
on 10050. To monitor a server, you open a port on it.

VPS Pilot works the other way round. The agent on each server opens one
connection *out* to the hub and keeps it. The servers you monitor accept no
new connections at all: their firewall can deny every inbound packet except
SSH, and monitoring keeps working.

This post explains how that works, how the hub and the agent prove who they
are to each other with a single pasted token, and what you give up.

## What's wrong with an agent port

An open port on every server is a cost you pay on every server:

- **It's attack surface.** Internet-wide scanners find a new listening port
  quickly.
  The agent behind it often runs as root, and a metrics endpoint tells a
  stranger your kernel version, your process names and how busy you are.
- **It needs firewall rules.** You allow the monitoring server's IP address on
  each machine. When that address changes, you update every server, and a
  rule you forget is a port left open to the world.
- **It needs its own security.** Many exporters serve plain HTTP with no
  authentication unless you add TLS and passwords to each one.
- **It doesn't reach everything.** A machine behind NAT, a home lab or a
  provider's private network can't be scraped from outside.

## Turn the connection around

The VPS Pilot agent connects to the hub's agent port (`55001` by default)
over TLS 1.3 and keeps that one connection open. Everything travels over it:

- the agent's stats every five seconds, and the projects it finds;
- the hub's requests: run a command a project declares, read a log file,
  back up now.

The hub never connects to a server. On each server, a firewall like this is
all you need:

```sh
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow OpenSSH
sudo ufw enable
```

If the connection drops, the agent tries again every 10 seconds. The hub
knows each node by its agent ID, not its IP address, so a server that changes
address, or sits behind NAT, just reconnects as itself.

Only one machine needs an open port: the hub.

## Proving who's who, with one string

A connection that starts from the server needs trust in both directions. The
hub has to know the agent is one you added, and the agent has to know it's
talking to your hub and not to someone in the middle. VPS Pilot does both
with a join token.

### The join token

In the dashboard, **Nodes → Add node** creates a token like this:

```text
vpsp1.<secret>.<fingerprint>
```

It works once and expires after an hour. The hub stores only a SHA-256 hash
of the secret. You paste it into the install command on the server:

```sh
curl -fsSL https://vpspilot.dev/install-agent.sh \
  | sudo VPS_PILOT_TOKEN=vpsp1.… sh -s -- --hub hub.example.com:55001
```

On its first run, the agent sends the secret to the hub. The hub checks it,
marks it used, and answers with an **agent ID** and a **node key** (32 random
bytes). The agent saves them in its state file, readable only by its owner,
and the install script removes the token from the server. From then on, the
agent authenticates with its ID and key, and the hub compares a hash of the
key in constant time.

The token is never needed again. One that leaks from your shell history after
it was used is worth nothing.

### Pinning the hub's certificate

The third part of the token is the SHA-256 fingerprint of the hub's TLS
certificate. The hub creates a self-signed certificate (ECDSA P-256) the
first time it starts, so there's no certificate authority and no domain name
involved.

The agent doesn't trust certificate authorities or check host names. It
pins that fingerprint, and the TLS handshake proves the hub holds the
matching private key. Someone who intercepts the connection, or takes over
the hub's DNS name, can't present that certificate, so the agent refuses to
talk to them. You can point agents at the hub by IP address if you like.

So the one string you paste carries both proofs: the secret proves the agent
to the hub, and the fingerprint proves the hub to the agent.

### Keeping the hub's port boring

The hub's port is open to the internet, so it's built to give strangers
nothing:

- A connection has 10 seconds to authenticate, TLS handshake included, or
  it's closed. Until then, the hub ignores every message except enrolling
  and connecting.
- A frame larger than 1 MiB drops the connection.
- Rejections say only `invalid credentials`, so they don't reveal which part
  was wrong.
- A node has one live connection at most. Revoking its key in the dashboard
  disconnects it at once. The agent then exits with code 3, and its systemd
  unit doesn't restart it, so it doesn't keep knocking.

## Dialing out isn't handing over a shell

A connection the server opened still carries requests from the hub, so the
next question is what the hub can ask for. The answer is: only what the
server's owner wrote down.

Each project on a server has a `config.vpspilot.json`. To run `build`, the
hub sends the agent the word `"build"`, never a command. The agent looks the
name up in the file on the server and runs what's written there, as the
file's owner. Logs, backups and cron jobs follow the same rule. Even someone
who took over your hub couldn't run anything that isn't already in a file you
wrote. The [security docs](/docs/security/) go through it in detail.

## What you give up

Dialing out isn't free:

- **The hub must be reachable** from your servers, on one port. If your
  firewall allows it, limit port 55001 to your servers' addresses, and put
  the dashboard behind a reverse proxy with HTTPS.
- **If the hub is down, you see nothing live.** Stats from that time are
  lost. The agents reconnect on their own, and cron jobs and backups keep
  running on the servers, with their results queued until the hub is back.
- **Back up the hub's `tls/` folder.** If its certificate is lost, the
  fingerprint changes, and every agent refuses to connect until you
  re-enroll it.

For a handful of VPSs, that trade is easy: one port on one machine you
protect, instead of a port on every machine you run.

## Try it

The [quick start](/docs/quick-start/) gets a hub running and your first server
connected in a few minutes, and the [agent docs](/docs/agent/) cover the
install script, the `.deb` and systemd. The [protocol](/docs/protocol/) is
documented too, if you'd like to see every message on the wire.
