# Scope

Read this before adding a feature. If a change doesn't fit here, it probably belongs in another tool.

## What VPS Pilot is

One self-hosted dashboard for the small fleet of Linux VPSs a freelancer, small agency or solo SaaS owner runs. A lightweight agent on each server reports health to a single hub binary (Go + SQLite, UI embedded).

Everything a project can do is declared in that project's own `config.vpspilot.json` on the server. The dashboard can never do more than the server owner allowed.

## What it does

| Capability | Description |
| --- | --- |
| Monitor | CPU, memory, disk and network per server, live and over 5m–7d |
| Alert | Thresholds with notifications to Discord, Slack or email |
| Run commands | Run a command *named* in `config.vpspilot.json` and stream its output |
| Read logs | Tail the log folders the project declares |
| Back up | Zip declared folders, the env file and a database dump |
| Schedule | Cron jobs declared per project, with last run and output |

## Out of scope

- Deploying from Git, git-push deploys, GitHub/GitLab integration
- Provisioning servers
- Containers or Kubernetes orchestration
- Editing files remotely
- An open shell or arbitrary command execution

Those belong to Coolify, Dokploy, Ansible or plain SSH.

## Rules for contributors (and AI agents)

1. **The hub sends command names, never command strings.** The agent looks the name up in its local `config.vpspilot.json` and runs that. A compromised hub can only trigger commands already written into a file on the server.
2. **No command execution over an unauthenticated agent channel.** Agents enroll with a one-time join token and connect over TLS.
3. **Nodes are identified by agent ID, not IP.**
4. **Keep it light.** One hub binary, SQLite, no external services required.
5. The agent lives in `agent/` and is written in Rust. Hub and agent talk over the protocol in `docs/PROTOCOL.md`.
