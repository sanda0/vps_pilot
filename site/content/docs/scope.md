---
title: Scope
weight: 90
description: What VPS Pilot does, what it will do, and what belongs in other tools.
---

VPS Pilot is one self-hosted dashboard for the small fleet of Linux servers a
freelancer, small agency or solo SaaS owner runs. It watches those servers and
lets you do the few things each project allows, and nothing more.

## What it does

| Capability | What it means | Status |
| --- | --- | --- |
| Monitor | CPU, memory, network and disk per server, live and over 5 minutes to 7 days | Available |
| Alert | Thresholds with notifications to Discord, Slack or email | Available |
| Run commands | Run a command *named* in `config.vpspilot.json` and stream its output | Available |
| Read logs | Tail the log folders a project declares | Available |
| Back up | Zip the declared folders, the env file and a database dump | Available |
| Schedule | Cron jobs declared per project, with the last run and its output | Coming soon |

## What it doesn't do

- Deploy from Git, push-to-deploy, or GitHub and GitLab integration
- Provision servers
- Orchestrate containers or Kubernetes
- Edit files on the server
- Give you an open shell or run arbitrary commands

Those jobs have good tools already. For Git deploys and containers, look at
[Coolify](https://coolify.io) or [Dokploy](https://dokploy.com). For
provisioning and configuration, [Ansible](https://www.ansible.com). For
everything else, plain SSH.

## Rules for contributors

1. **The hub sends command names, never command strings.** The agent looks
   the name up in its local `config.vpspilot.json` and runs that. A
   compromised hub can only trigger commands already written into a file on
   the server.
2. **No command execution over an unauthenticated agent channel.** Agents
   enroll with a one-time join token and connect over TLS.
3. **Nodes are identified by agent ID, not IP.**
4. **Keep it light.** One hub binary, SQLite, no external services required.
5. The agent lives in `agent/` and is written in Rust. The hub and the agent
   talk over the [agent protocol](/docs/protocol/).

The same rules are in
[SCOPE.md](https://github.com/sanda0/vps_pilot/blob/main/SCOPE.md) in the
repository. Read it before you propose a feature.
