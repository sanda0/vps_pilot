# Agent protocol

The hub and the agent (`agent/`) talk over a single long-lived TCP connection on port `55001` (`-agent-port` / `TCP_SERVER_PORT`). This document is the contract. Any agent implementation that follows it works with the hub.

## Transport

- **TLS 1.3** only. The hub presents a self-signed certificate, generated on first start in `$DB_PATH/tls/agent-hub.{crt,key}`. You can supply your own with `-agent-tls-cert` / `-agent-tls-key`.
- The agent does **not** validate a CA chain or hostname. It pins the **SHA-256 fingerprint of the hub's leaf certificate (DER)**, which it learns from the join token. The TLS handshake signature is still verified, so only the holder of that certificate's private key can complete it.
- Keep the hub's cert and key files. If they are regenerated, the fingerprint changes and every enrolled agent refuses to connect until it is re-enrolled.

## Framing

Every message is one frame:

```
+----------------------+---------------------------+
| length: u32, big-end | JSON envelope (length B)  |
+----------------------+---------------------------+
```

- `length` is the byte length of the JSON body, `1 ..= 1048576` (1 MiB). Anything else is a protocol error and the connection is dropped.
- Envelope:

```json
{ "type": "sys_stat", "node_id": 3, "token": "…", "data": { … } }
```

| Field | Type | Notes |
| --- | --- | --- |
| `type` | string | Required. Message type, see below. |
| `node_id` | int | Optional. Set by the hub. The hub ignores any value the agent sends and uses the authenticated node. |
| `token` | string | Optional. Credentials, only on `enroll` / `connected`. |
| `data` | object | Optional. Payload, shape depends on `type`. |

## Credentials

| Credential | Format | Lifetime | Stored by hub as |
| --- | --- | --- | --- |
| Join token | `vpsp1.<secret>.<cert fingerprint hex>` | One use, 1 hour | SHA-256 of `<secret>` |
| Node key | random, base64url | Until revoked or re-enrolled | SHA-256 |
| Agent ID | UUID | Permanent per enrollment | plain |

The dashboard creates join tokens (**Nodes → Add node**, or **Re-enroll** on a node). The agent sends only the `<secret>` part to the hub. The fingerprint part is for the agent's pin.

## Handshake

The hub closes any connection that hasn't authenticated within **10 seconds** (this includes the TLS handshake). Until a connection authenticates, the hub drops every message other than `enroll` / `connected`.

### First run: enroll

```
agent → hub   enroll      token = <join secret>          data = SystemInfo
hub → agent   enrolled    node_id                        data = { agent_id, node_key, node_id }
hub → agent   sys_stat    node_id                        (start streaming)
```

The agent must persist `agent_id`, `node_key`, `node_id` and the pinned fingerprint before streaming. A join token created for an existing node re-enrolls that node: the node keeps its ID and history, and its previous key stops working.

### Later runs: connect

```
agent → hub   connected   token = "<agent_id>:<node_key>"   data = SystemInfo
hub → agent   sys_stat    node_id                           (start streaming)
```

### Rejection

```
hub → agent   auth_failed   data = { "reason": "…" }
```

The hub then closes the connection. Reasons are deliberately generic. Retrying with the same credentials won't succeed, so the agent should stop (the reference agent exits with code `3`).

A node can have only one live connection. A second successful `connected` closes the older connection. Revoking a node's key in the dashboard closes its connection, and the next `connected` gets `auth_failed`.

## Messages after the handshake

| Type | Direction | `data` |
| --- | --- | --- |
| `sys_stat` | agent → hub | `SystemStat`, every interval (default 5s) |
| `sys_info` | agent → hub | `SystemInfo`, whenever it changes |
| `projects` | agent → hub | `ProjectSyncPayload`, see [Projects](#projects) |
| `project_settings` | hub → agent | `ProjectSettings`, after the handshake and whenever they change |
| `rescan_projects` | hub → agent | none. Search for projects now |

Unknown types are ignored by both sides.

## Projects

Each project is a directory with a `config.vpspilot.json`. The agent finds them two ways:

- **Search**: walk the agent's project directories (`--projects-dir`, default `/var/www` and `/home`), up to 4 levels deep. Hidden directories, `node_modules`, `vendor`, `target` and symlinked directories are skipped.
- **Manual paths** from `ProjectSettings.paths`: the agent reads `<path>/config.vpspilot.json` directly, whatever its depth. A manual path must resolve (after `..` and symlinks) to a directory inside one of the agent's project directories, unless the agent runs with `--allow-any-project-path`. The server owner, not the dashboard, decides which parts of the disk can hold projects.

When the agent searches:

| Trigger | Search runs |
| --- | --- |
| Session starts | always |
| `project_settings` received with different values | always |
| `rescan_projects` received | always |
| Timer, every `scan_interval` seconds | only if `auto_scan` is true |

Until it receives `project_settings`, the agent uses `auto_scan: true`, `scan_interval: 900` and no manual paths. A hub that never sends settings gets that behaviour.

After a search, the agent sends `projects` if the result (projects or errors) differs from what it last sent in this session. The first search of a session and a search requested by `rescan_projects` are always sent, so the hub knows when the node was last scanned. Each `projects` message is the complete state: the hub replaces the node's projects and errors with it.

## Payloads

`SystemInfo`

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

`hostname` names the node on first enrollment. `total_memory` is in bytes.

`SystemStat`

```json
{
  "cpu_usage": [12.5, 3.1, 7.8, 4.0],
  "mem_usage": 41.2,
  "disk_usage": 63.0,
  "net_sent_ps": 10240,
  "net_recv_ps": 52011
}
```

`cpu_usage` holds one percentage per core. `mem_usage` and `disk_usage` (root filesystem) are percentages. Network values are bytes per second across non-loopback interfaces.

`ProjectSettings`

```json
{
  "auto_scan": true,
  "scan_interval": 900,
  "paths": ["/var/www/legacy/app"]
}
```

`scan_interval` is in seconds; the agent clamps it to `60 ..= 86400`. `paths` are absolute project directories added from the dashboard.

`ProjectSyncPayload`

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
    { "path": "/opt/legacy-app", "error": "outside the agent's project directories" },
    { "path": "/var/www/blog", "error": "parse config.vpspilot.json: missing field `name`" }
  ]
}
```

`projects` holds every valid config found, with `path` set by the agent to the absolute project directory. `errors` holds configs that couldn't be used and manual paths that were rejected, keyed by project directory. Both lists are complete each time. The Go types are in `server/internal/tcpserver/types.go`.
