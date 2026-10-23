---
title: VPS Pilot
description: One self-hosted dashboard for the handful of Linux VPSs you run. A single Go binary with SQLite and the UI built in, plus a small Rust agent on each server. Free and open source.
pitch: One self-hosted dashboard for the handful of Linux servers you run. A single Go binary with the UI built in, plus a tiny Rust agent on each server. You stay in control of what it can do.

# Every "available" item is implemented end to end in the code today. Check
# the code before moving anything from "soon" to "available".
available:
  - title: Live metrics
    text: CPU per core, memory and network traffic for every server, charted from the last 5 minutes to the last 7 days. Charts refresh every 10 seconds.
  - title: Threshold alerts
    text: Get a Discord, Slack or email message when CPU, memory or network traffic goes over your limit, with a wait you choose between repeats.
  - title: Project discovery
    text: The agent finds every config.vpspilot.json on the server. Scan settings per server, rescan on demand, and manual paths the agent checks itself.
  - title: Safe enrollment
    text: One-time join tokens, TLS 1.3 with a pinned certificate, and a key per server that you can revoke. Re-enroll a server without losing its history.
  - title: One binary hub
    text: Go, SQLite and the dashboard in a single executable. Seven days of metrics, no external database or service.
  - title: Tiny agent
    text: One Rust binary per server. It dials out to the hub, so the server needs no open ports.

soon:
  - Run the commands a project declares, by name, with live output and an audit log
  - Tail the log folders a project declares
  - Backups of declared folders, the env file and the database
  - Cron jobs declared per project
  - Disk usage charts and alerts (the agent already reports disk usage)
  - Install script and systemd unit for the agent
  - Docker Compose setup
  - Multiple users with roles

notList:
  - Deploying from Git, push-to-deploy, GitHub or GitLab integration
  - Provisioning servers
  - Orchestrating containers or Kubernetes
  - Editing files on the server
  - An open shell, or running arbitrary commands
---
