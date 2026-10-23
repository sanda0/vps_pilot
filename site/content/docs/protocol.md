---
title: Agent protocol
weight: 80
description: The wire protocol between the hub and its agents. Any agent that follows it works with the hub.
---

The hub and an agent talk over one long-lived TCP connection to the hub's
agent port (`55001` by default, `-agent-port` or `TCP_SERVER_PORT`). This page
is the contract. The reference agent lives in `agent/`, and the hub's Go types
are in `server/internal/tcpserver/types.go`.

## Transport

- **TLS 1.3** only. The hub presents a self-signed certificate created on
  first start (`$DB_PATH/tls/agent-hub.crt` and `.key`), or one supplied with
  `-agent-tls-cert` and `-agent-tls-key`.
- The agent doesn't validate a CA chain or host name. It pins the **SHA-256
  fingerprint of the hub's leaf certificate (DER)**, taken from the join token.
  The handshake signature is still verified.
- If the hub's certificate is regenerated, the fingerprint changes and every
  enrolled agent refuses to connect until it's re-enrolled.

## Framing

Every message is one frame: a 4-byte big-endian length, then that many bytes
of JSON.

```text
+-----------------------+------------------------------+
| length: u32, big end. | JSON envelope (length bytes) |
+-----------------------+------------------------------+
```

The length must be between 1 and 1048576 (1 MiB). Anything else is a protocol
error and the connection is dropped.

```json
{ "type": "sys_stat", "node_id": 3, "token": "…", "data": { } }
```

| Field | Type | Notes |
| --- | --- | --- |
| `type` | string | Required. The message type. |
| `node_id` | int | Optional. Set by the hub. The hub ignores any value an agent sends and uses the authenticated node. |
| `token` | string | Optional. Credentials, only in `enroll` and `connected`. |
| `data` | object | Optional. The payload; its shape depends on `type`. |

Both sides ignore message types they don't know.

## Credentials

| Credential | Format | Lifetime | Stored by the hub as |
| --- | --- | --- | --- |
| Join token | `vpsp1.<secret>.<fingerprint hex>` | One use, 1 hour | SHA-256 of `<secret>` |
| Node key | random, base64url | Until revoked or re-enrolled | SHA-256 |
| Agent ID | UUID | Per enrollment | plain |

The dashboard creates join tokens (**Nodes → Add node**, or **Re-enroll agent**
on a node). The agent sends only `<secret>` to the hub; the fingerprint is for
its pin.

## Handshake

The hub closes any connection that hasn't authenticated within **10 seconds**,
TLS handshake included. Until then it drops every message except `enroll` and
`connected`.

**First run: enroll**

```text
agent → hub   enroll      token = <join secret>          data = SystemInfo
hub → agent   enrolled    node_id                        data = { agent_id, node_key, node_id }
hub → agent   sys_stat    node_id                        (start streaming)
```

The agent must save `agent_id`, `node_key`, `node_id` and the pinned
fingerprint before it streams. A join token created for an existing node
re-enrolls it: the node keeps its ID and history, and its old key stops
working.

**Later runs: connect**

```text
agent → hub   connected   token = "<agent_id>:<node_key>"   data = SystemInfo
hub → agent   sys_stat    node_id                           (start streaming)
```

**Rejection**

```text
hub → agent   auth_failed   data = { "reason": "…" }
```

The hub then closes the connection. Retrying with the same credentials won't
work, so the agent should stop; the reference agent exits with code `3`.

A node has at most one live connection: a second successful `connected`
closes the older one. Revoking a node's key closes its connection, and its
next `connected` gets `auth_failed`.

## After the handshake

| Type | Direction | `data` |
| --- | --- | --- |
| `sys_stat` | agent → hub | `SystemStat`, every interval (5 s by default) |
| `sys_info` | agent → hub | `SystemInfo`. Optional; the reference agent sends it only in the handshake. |
| `projects` | agent → hub | `ProjectSyncPayload` |
| `project_settings` | hub → agent | `ProjectSettings`, after the handshake and whenever they change |
| `rescan_projects` | hub → agent | none. Search for projects now. |

## Projects

The agent searches its project directories (default `/var/www` and `/home`)
up to 4 levels deep, skipping hidden directories, `node_modules`, `vendor`,
`target` and symlinked directories. It also reads each manual path from
`ProjectSettings.paths` directly, if the path resolves inside one of its
project directories (or the agent runs with `--allow-any-project-path`).

It searches when the session starts, when `project_settings` arrives with
different values, when `rescan_projects` arrives, and every `scan_interval`
seconds while `auto_scan` is true. Until it receives settings it uses
`auto_scan: true`, `scan_interval: 900` and no manual paths.

After a search it sends `projects` if the result differs from what it last
sent in this session. The first search of a session and a search requested by
`rescan_projects` are always sent. Each message is the complete state: the
hub replaces the node's projects and errors with it.

## Payloads

`SystemInfo`. `hostname` names the node on first enrollment; `total_memory`
is in bytes.

```json
{
  "hostname": "web-1",
  "os": "linux",
  "platform": "ubuntu",
  "platform_version": "24.04",
  "kernel_version": "6.8.0-45-generic",
  "cpus": 4,
  "total_memory": 8221474816
}
```

`SystemStat`. `cpu_usage` has one percentage per core; `mem_usage` and
`disk_usage` (root filesystem) are percentages; network values are bytes per
second across non-loopback interfaces.

```json
{
  "cpu_usage": [12.5, 3.1, 7.8, 4.0],
  "mem_usage": 41.2,
  "disk_usage": 63.0,
  "net_sent_ps": 10240,
  "net_recv_ps": 52011
}
```

`ProjectSettings`. `scan_interval` is in seconds and the agent clamps it to
60–86400; `paths` are absolute directories added in the dashboard.

```json
{ "auto_scan": true, "scan_interval": 900, "paths": ["/var/www/legacy/app"] }
```

`ProjectSyncPayload`. `projects` holds every valid config, with `path` set by
the agent; `errors` holds unusable configs and rejected manual paths, keyed by
directory.

```json
{
  "projects": [
    {
      "name": "shop",
      "path": "/var/www/shop",
      "tech": ["laravel"],
      "commands": [{ "name": "build", "command": "npm run build" }],
      "logs": ["storage/logs"],
      "backups": { "env_file": ".env", "dir": ["storage/app"] }
    }
  ],
  "errors": [
    { "path": "/opt/legacy-app", "error": "outside the agent's project directories (--projects-dir)" }
  ]
}
```
