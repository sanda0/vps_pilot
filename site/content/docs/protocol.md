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
| `command_run` | hub → agent | `CommandRun`, see [Run commands](#run-commands) |
| `command_cancel` | hub → agent | `{ "run_id": "…" }` |
| `command_output` | agent → hub | `CommandOutput` |
| `command_exit` | agent → hub | `CommandExit` |
| `log_list` | hub → agent | `LogList`, see [Read logs](#read-logs) |
| `log_files` | agent → hub | `LogFiles` |
| `log_open` | hub → agent | `LogOpen` |
| `log_close` | hub → agent | `{ "stream_id": "…" }` |
| `log_data` | agent → hub | `LogData` |
| `log_end` | agent → hub | `LogEnd` |
| `backup_run` | hub → agent | `BackupRun`, see [Back up](#back-up) |
| `backup_started` | agent → hub | `BackupStarted` |
| `backup_result` | agent → hub | `BackupResult` |

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

## Run commands

The hub asks the agent to run a command by **name**, never by command string.
The agent looks the name up in the project's `config.vpspilot.json` and runs
what it finds there.

```text
hub → agent   command_run      { run_id, project_path, command }
agent → hub   command_output   { run_id, seq, stream, data }   (0 or more)
agent → hub   command_exit     { run_id, status, … }           (exactly one)
```

Only agents whose `SystemInfo.features` contains `"commands"` support this.
Agents from v0.2 and earlier send no `features`; they would ignore
`command_run`, so the hub never sends it to them.

Before running, the agent answers with status `rejected` unless all of these
hold:

1. `project_path` is a project from its latest search.
2. It reads `<project_path>/config.vpspilot.json` again and it parses. The
   file isn't a symlink and isn't writable by its group or others.
3. `command` is the `name` of one of its `commands`.
4. It can run as the file's owner: it runs as root, or already as that user.
5. The same command isn't already running in that project, and fewer than 4
   commands are running.

The command runs as the config file's owner (user, group and the user's
groups) with `/bin/sh -c` in `project_path`. Stdin is `/dev/null`, and the
environment is only `HOME`, `USER`, `LOGNAME`, `SHELL` and
`PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin`. It gets
its own process group. A timeout (10 minutes, `--command-timeout`), a
`command_cancel` or the end of the session kills the group: `SIGTERM`, then
`SIGKILL` 5 seconds later. The hub marks runs that never got a `command_exit`
as `lost` when the node disconnects.

Output is sent as it arrives, in chunks of up to 32 KiB of UTF-8 (invalid
bytes become U+FFFD). `seq` counts up from 0 across both streams. After 4 MiB
the agent stops sending output, keeps reading it, and sets `truncated`.

| `status` | Meaning | `exit_code` |
| --- | --- | --- |
| `exited` | Ran and exited | its exit code (`128 + n` if killed by signal `n`) |
| `timeout` | Killed after the timeout | `null` |
| `cancelled` | Killed after `command_cancel` | `null` |
| `rejected` | Not started; `error` says why | `null` |
| `failed` | Couldn't be started | `null` |

A `command_cancel` for a run that has finished, or that the agent doesn't
know, is ignored.

## Read logs

The hub can list and read the files in the log folders a project declares
(`logs` in `config.vpspilot.json`), and nothing else.

```text
hub → agent   log_list    { request_id, project_path }
agent → hub   log_files   { request_id, files, error }

hub → agent   log_open    { stream_id, project_path, file, before, max_bytes, follow }
agent → hub   log_data    { stream_id, seq, offset, data, … }   (0 or more)
agent → hub   log_end     { stream_id, reason, error }          (exactly one)
hub → agent   log_close   { stream_id }                         (optional)
```

Only agents whose `SystemInfo.features` contains `"logs"` support this. The
hub never sends `log_list` or `log_open` to other agents.

Which files can be read:

1. `project_path` is a project from the agent's latest search. The agent
   reads its `config.vpspilot.json` again for every request, with the same
   checks as for running commands: not a symlink, not writable by its group
   or others.
2. Each entry of `logs` is resolved (`..` and symlinks) and must stay inside
   the project directory. Folders that don't, or don't exist, are skipped.
3. A file is readable if it's a regular file directly in a declared folder or
   in a folder directly below it, and still resolves inside that folder.
4. `file` in `log_open` is the path relative to the project directory,
   exactly as `log_files` listed it. Anything else ends with `rejected`.
5. Compressed files (`.gz`, `.bz2`, `.xz`, `.zst`, `.zip`) are listed but
   can't be opened.

Listing and reading run in a child process with the config file owner's user
and groups, so a file that user can't read can't be read at all. `log_files`
lists at most 1000 files, sorted by path; if there are more, `error` says so.

`log_open` asks for one window of the file and, optionally, what is written
after it:

- `before` is a byte offset; the agent reads up to `max_bytes` ending there.
  `null` means the end of the file. `max_bytes` defaults to 64 KiB and is
  clamped to 1–1048576.
- The window starts at the beginning of a line, unless it starts at offset 0.
- `follow: true` is only allowed when `before` is `null`. The agent then
  sends what is appended, checking at least once a second. If the file is
  truncated or replaced, it reopens it from the start and sets `rotated` on
  the next `log_data`.

`data` is UTF-8 text in chunks of up to 32 KiB (invalid bytes become U+FFFD),
and `offset` is where it starts in the file. `seq` counts up from 0 per
stream.

| `reason` | Meaning |
| --- | --- |
| `done` | The window was sent and `follow` was false |
| `closed` | The hub sent `log_close` |
| `rejected` | Not allowed; `error` says why |
| `error` | Reading failed, e.g. the file was deleted |
| `limit` | A followed stream reached 30 minutes |

At most 8 streams are open at once on a node; more are `rejected`. When the
session ends, the agent closes its streams. The hub doesn't store log
contents.

## Back up

The agent backs up what a project declares in `backups`: the `dir` folders,
the `env_file` and a database dump, into a zip in `local_dir`, and uploads it
if `s3` is set. The hub never receives the backup, only its result.

```text
hub → agent   backup_run       { run_id, project_path }             (Back up now)
agent → hub   backup_started   { run_id, project_path, trigger, … }
agent → hub   backup_result    { run_id, project_path, status, … }  (one per backup)
```

Backups start on the schedule in the config file, run by the agent even when
the hub is unreachable (the agent picks the `run_id`), or from `backup_run`.
Only agents whose `SystemInfo.features` contains `"backups"` support this; the
hub never sends `backup_run` to other agents.

Before backing up, the agent checks, like for commands, that `project_path` is
a project from its latest search, re-reads the config (not a symlink, not
writable by group or others) and that it can run as the config file's owner.
One backup runs at a time on a node: a scheduled backup waits, and a
`backup_run` is rejected. A rejected backup sends only `backup_result`.

The backup runs as the config file's owner. `dir`, `env_file` and a SQLite
file must resolve inside the project directory. The S3 keys and database
password are read from the env file on the server; the config, and so the
hub, only has their variable names. See
[Back up](/docs/projects/#back-up) for the zip, the dump tools, retention and
the 1-hour timeout.

A backup keeps running if the session ends. The agent keeps up to 50 results
it couldn't send and sends them after it reconnects. The hub de-duplicates by
`run_id`, and marks a backup with no result after 24 hours as `lost`; a later
result still replaces that.

| `status` | Meaning |
| --- | --- |
| `ok` | The zip was written to `local_dir`; `upload` says how the upload went |
| `failed` | No zip was kept; `error` says why |
| `rejected` | Not started; `error` says why |

## Payloads

`SystemInfo`. `hostname` names the node on first enrollment; `total_memory`
is in bytes. `agent_version` and `features` were added in v0.3; when they're
missing, the hub treats the agent as having no features. The features are
`commands` (v0.3), `logs` (v0.4) and `backups` (v0.5).

```json
{
  "hostname": "web-1",
  "os": "linux",
  "platform": "ubuntu",
  "platform_version": "24.04",
  "kernel_version": "6.8.0-45-generic",
  "cpus": 4,
  "total_memory": 8221474816,
  "agent_version": "v0.5.0",
  "features": ["commands", "logs", "backups"]
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

`CommandRun`. `run_id` is a UUID chosen by the hub; `project_path` is the
project's `path` from `ProjectSyncPayload`; `command` is a command name.

```json
{ "run_id": "0b6c3c3e-6a8f-4b7e-9d0a-2f3c1e7a9b10", "project_path": "/var/www/shop", "command": "build" }
```

`CommandOutput`. `stream` is `stdout` or `stderr`.

```json
{ "run_id": "0b6c3c3e-…", "seq": 0, "stream": "stdout", "data": "> shop@1.0.0 build\n" }
```

`CommandExit`. `run_as` is the user it ran as, empty if it never started;
`error` is a short reason for `rejected` and `failed`.

```json
{
  "run_id": "0b6c3c3e-…",
  "status": "exited",
  "exit_code": 0,
  "run_as": "deploy",
  "duration_ms": 41230,
  "truncated": false,
  "error": ""
}
```

`LogList` and `LogFiles`. File paths are relative to the project directory,
`size` is in bytes and `modified` is Unix seconds. `error` is set, and `files`
empty, when the request was refused.

```json
{ "request_id": "4f1d…", "project_path": "/var/www/shop" }
```

```json
{
  "request_id": "4f1d…",
  "files": [
    { "path": "storage/logs/laravel.log", "size": 1048576, "modified": 1792915200 },
    { "path": "storage/logs/laravel-2026-10-24.log.gz", "size": 20480, "modified": 1792828800 }
  ],
  "error": ""
}
```

`LogOpen`. `stream_id` is a UUID chosen by the hub.

```json
{ "stream_id": "9a2e…", "project_path": "/var/www/shop", "file": "storage/logs/laravel.log", "before": null, "max_bytes": 65536, "follow": true }
```

`LogData`. `size` is the file's size when the chunk was read.

```json
{ "stream_id": "9a2e…", "seq": 0, "offset": 983040, "data": "[2026-10-25 08:00:01] production.INFO: …\n", "size": 1048576, "rotated": false }
```

`LogEnd`

```json
{ "stream_id": "9a2e…", "reason": "rejected", "error": "not a file in this project's log folders" }
```

`BackupRun`. `run_id` is a UUID chosen by the hub.

```json
{ "run_id": "c41a…", "project_path": "/var/www/shop" }
```

`BackupStarted`. `trigger` is `manual` or `schedule`; `started_at` is Unix
seconds.

```json
{ "run_id": "c41a…", "project_path": "/var/www/shop", "trigger": "schedule", "started_at": 1793156400, "run_as": "deploy" }
```

`BackupResult`. `file`, `local_path` and `size` are set when `status` is `ok`.
`upload.status` is `ok`, `failed` or `skipped`.

```json
{
  "run_id": "c41a…",
  "project_path": "/var/www/shop",
  "trigger": "schedule",
  "status": "ok",
  "started_at": 1793156400,
  "duration_ms": 84210,
  "file": "shop_backup-20261031T030000Z.zip",
  "local_path": "/var/www/backups/shop/shop_backup-20261031T030000Z.zip",
  "size": 52428800,
  "database": "mysql",
  "upload": { "status": "ok", "key": "shop/shop_backup-20261031T030000Z.zip", "error": "" },
  "error": ""
}
```
