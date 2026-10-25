---
title: Introduction
linkTitle: Introduction
description: What VPS Pilot is, how the hub and agents fit together, and what works today.
---

VPS Pilot is one self-hosted dashboard for the handful of Linux servers a
freelancer, a small agency or a solo SaaS owner looks after. You run one hub
and put a small agent on each server. The agents report their health to the
hub, and you watch everything from one place in the browser.

## How it fits together

**The hub** is a single Go binary. It stores everything in two SQLite
databases, serves the dashboard (built into the binary), and listens on two
ports:

- `8080`: the dashboard and its API, for your browser.
- `55001`: the agent channel, TLS 1.3 only, for your servers.

**The agent** (`vps-pilot-agent`) is a small Rust program that runs on each
server. On its first run it enrolls with a one-time join token from the
dashboard. After that it reconnects on its own with the credentials it saved,
and it checks the hub's certificate against the fingerprint it pinned at
enrollment.

There are no other moving parts: no external database, no message queue, no
cloud account.

## What works today

- **Monitoring.** CPU (per core), memory, disk and network traffic for each
  server, charted over 5 minutes up to 7 days, and whether its agent is online.
- **[Alerts](/docs/alerts/).** CPU, memory, disk and network thresholds, sent to Discord, Slack or email.
- **[Projects](/docs/projects/).** The agent finds every `config.vpspilot.json` on the server and
  the dashboard lists those projects, with per-server scan settings.
- **[Run commands](/docs/projects/#run-a-command).** Run a command a project
  declares, by name, with live output and a history of every run.
- **Nodes.** Add a server with a join token, re-enroll it on a new machine
  without losing its history, or revoke its key.

{{< callout soon >}}
Tailing a project's logs, backups and cron jobs are planned. The dashboard
already shows what a project declares for them.
{{< /callout >}}

## The rule that shapes everything

The server owner stays in control. Anything the dashboard will ever be able to
run on a server has to be written in that project's `config.vpspilot.json`
first, on the server itself. The hub only sends the *name* of a declared
command, never a command string, so even a compromised hub can't run anything
the server owner didn't write down. [Security](/docs/security/) has the
details.

## Where to go next

- [Quick start](/docs/quick-start/): build the hub, log in and connect your
  first server.
- [Agent](/docs/agent/): flags, environment variables and exit codes.
- [Configuration](/docs/configuration/): the `.env` file and the hub's
  command-line flags.
- [Security](/docs/security/): how agents authenticate and what a hub can do.
- [Scope](/docs/scope/): what VPS Pilot will and won't do.
