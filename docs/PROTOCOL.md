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
| `projects` | agent → hub | `ProjectSyncPayload` (Phase 2) |

Unknown types are ignored by both sides.

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

`ProjectSyncPayload`: the full list of `config.vpspilot.json` files found on the node. See `server/internal/tcpserver/types.go`.
