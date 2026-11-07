---
title: Monitoring a VPS without opening a single inbound port
date: 2026-11-07
description: The VPS Pilot agent dials out to a hub you host, over TLS, so the servers you monitor accept no new connections. Here's how one pasted token authenticates both ends, what the connection can carry, and what it costs.
---

A lot of self-hosted monitoring still asks every server to listen. Prometheus
scrapes `node_exporter` on port 9100, Netdata serves its dashboard on 19999,
and a passive Zabbix agent waits on 10050. To watch a server, you open a port
on it and then spend effort making sure only the right machine can use it.

Dialing out instead isn't new. Datadog's agent, Netdata Cloud, Zabbix in
active mode and Grafana Alloy pushing with `remote_write` all send data out
rather than wait for a scrape. What I wanted for
[VPS Pilot](https://vpspilot.dev/) was that model with three extra
properties:

- the hub is **yours**: one Go binary with SQLite on a machine you run, no
  third-party cloud;
- enrolling a server takes **one pasted string**, with no certificate
  authority or domain name required;
- the same connection carries **requests back** from the hub (run a command,
  read a log, take a backup) without turning into a remote shell.

This post is about how those fit together, and what they cost.

## The cost of an agent port

An open port on each server is paid for on each server:

- It's reachable by anyone who can route to it, and the agent behind it
  often runs as root. Even a read-only metrics endpoint tells a stranger your
  kernel version and what you run.
- You keep a firewall rule on every machine allowing the monitoring server's
  address, and update all of them when that address changes.
- Each exporter needs its own TLS and authentication, which many leave off by
  default.
- A machine behind NAT or on a private network can't be scraped from outside
  at all.

None of this is hard on one server. It's the multiplication that hurts.

## Turning the connection around

The agent opens one TCP connection to the hub's agent port (`55001` by
default), does a TLS 1.3 handshake, and keeps the connection open:

```text
  server (no inbound ports)                    hub (one port open)
  +--------------------+                       +--------------------+
  | agent              |--- TCP + TLS 1.3 ---->| :55001             |
  |                    |   opened by agent     |                    |
  | stats every 5 s ---|---------------------->| store, chart,      |
  |                    |                       | alert              |
  | run "build" <------|-----------------------| requests ride the  |
  |                    |                       | same connection    |
  +--------------------+                       +--------------------+
```

Stats go up every five seconds; the hub's requests come down the same
connection. The hub never connects to a server, so a server's firewall can
be as strict as this:

```sh
sudo ufw default deny incoming
sudo ufw allow OpenSSH
sudo ufw enable
```

If the connection drops, the agent retries every 10 seconds. The hub
identifies a node by its agent ID, never its IP address, so a server that
changes address or sits behind NAT reconnects as itself.

## One string, two proofs

A connection opened by the server needs trust in both directions. The hub
must know the agent is one you added. The agent must know it reached your hub
and not something in the middle. The join token carries both:

```text
vpsp1.<secret>.<fingerprint>
```

The dashboard creates it (**Nodes → Add node**). It works once and expires
after an hour, and the hub stores only a SHA-256 hash of the secret. You
paste it into the install command on the server:

```sh
curl -fsSL https://vpspilot.dev/install-agent.sh \
  | sudo VPS_PILOT_TOKEN=vpsp1.… sh -s -- --hub hub.example.com:55001
```

### The secret proves the agent

On the first run, the agent sends only the secret part. The hub checks it,
marks it used, and issues long-term credentials:

```text
agent → hub   enroll      token = <secret>               data = system info
hub → agent   enrolled    data = { agent_id, node_key, node_id }
```

The node key is 32 random bytes. The agent writes it to a state file with
mode `0600`, and the install script deletes the token from the server. Every
later connection authenticates with the ID and key instead:

```text
agent → hub   connected   token = "<agent_id>:<node_key>"
```

The hub keeps only a hash of the key and compares it in constant time. A join
token that leaks after use, say from shell history, is worthless.

### The fingerprint proves the hub

The third part of the token is the SHA-256 fingerprint of the hub's TLS
certificate. By default the hub creates a self-signed ECDSA P-256 certificate
on first start.

The agent ignores certificate authorities and host names. It pins that
fingerprint, and the TLS handshake proves the hub holds the matching private
key. Someone who intercepts the connection, or takes over the hub's DNS
record, can't present that certificate. You can point agents at the hub by IP
address.

Why not Let's Encrypt? It would add a dependency on a domain name, renewals
and a public CA, for a channel where both ends are yours. Pinning is simpler,
with one sharp edge, covered under the costs below. If you'd rather use your
own certificate, the hub takes one, and the agent pins that instead.

## What a stranger gets from the hub's port

The hub's port is the one thing exposed, so it's built to give strangers
little:

- 10 seconds to authenticate, TLS handshake included, or the connection is
  closed. Until then the hub ignores everything except `enroll` and
  `connected`.
- Frames over 1 MiB drop the connection.
- Rejections say only `invalid credentials`, not which part was wrong.
- One live connection per node. Revoking a key in the dashboard disconnects
  that node at once. The agent then exits with code 3, and its systemd unit
  is set not to restart on that code, so it doesn't keep knocking.

## Dialing out isn't handing over a shell

The obvious worry: if requests come down the connection, a compromised hub
can do anything on every server. It can't, because the hub never sends a
command, only a name.

Each project on a server has a `config.vpspilot.json` written by the server's
owner. To run `build`, the hub sends the word `"build"`. The agent re-reads
the file, looks the name up, and runs what's written there, as the file's
owner rather than as the agent's user. It refuses a config that is a symlink
or writable by group or others. Log reads, backups and cron jobs follow the
same rule. The worst a compromised hub can do is trigger commands you already
wrote into a file on that server. The [security docs](/docs/security/) have
the details.

## Why not just use a VPN?

WireGuard or Tailscale solve reachability: put the hub and the servers on one
network and scraping works without public ports. That's a good setup, and
VPS Pilot runs fine over one; then even the hub's port needn't be public.

Dialing out means you don't need one. For a freelancer with five VPSs at
three providers, "install an agent, paste a token" is less to run than a mesh
VPN plus a scraper. If you already have the VPN, use it.

## What it costs

- **The hub must be reachable** from your servers, on one port. Limit 55001
  to your servers' addresses if your firewall allows, and keep the dashboard
  behind a reverse proxy with HTTPS.
- **The hub is now the valuable target.** It holds every node's key hash and
  can reach every agent. The name-only rule bounds the damage, but protect it
  like the one machine that matters.
- **If the hub is down, you see nothing live.** Stats from that window are
  lost; the agent doesn't buffer them. Cron jobs and backups keep running on
  the servers, and up to 50 of their results queue until the hub is back.
- **Back up the hub's `tls/` folder.** Lose the certificate and the
  fingerprint changes, and every agent refuses to connect until you
  re-enroll it. That's the sharp edge of pinning.

For a handful of servers, I think that trade is worth it: one port on one
machine you look after, instead of a port on every machine you run.

## Try it or read the protocol

The [quick start](/docs/quick-start/) gets a hub up and a server connected in
a few minutes. The [protocol](/docs/protocol/) is documented message by
message, if you'd rather read the wire format first. The code is on
[GitHub](https://github.com/sanda0/vps_pilot) under the AGPL.
