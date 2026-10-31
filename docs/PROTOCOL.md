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

## Run commands

The hub asks the agent to run a command by **name**. It never sends a command string. The agent looks the name up in the project's `config.vpspilot.json` and runs what it finds there, so a hub can only trigger commands the server owner wrote into a file.

```
hub → agent   command_run      data = { run_id, project_path, command }
agent → hub   command_output   data = { run_id, seq, stream, data }      (0 or more)
agent → hub   command_exit     data = { run_id, status, … }              (exactly one)
```

An agent supports this only if its `SystemInfo.features` contains `"commands"`. Agents from v0.2 and earlier don't send `features`. The hub must not send `command_run` to them, because they would ignore it and the hub would wait for a `command_exit` that never comes.

### Before running

The agent answers with `command_exit` and status `rejected` (and runs nothing) unless all of these hold:

1. `project_path` is a project from the agent's latest search: a directory under its project directories, or an accepted manual path.
2. The agent re-reads `<project_path>/config.vpspilot.json` at run time and it parses. The file must not be a symlink and must not be group- or world-writable.
3. `command` matches the `name` of an entry in its `commands`.
4. The agent can run as the file's owner: it runs as root, or it already runs as that user.
5. The same command in the same project is not already running, and fewer than 4 commands are running on the node.

### Running

- The command runs as the **owner of `config.vpspilot.json`** (user and group), never automatically as the agent's user. That way, a user who can write a config under `/home` can't get commands run as root.
- It runs `/bin/sh -c <command>` in `project_path`. Stdin is `/dev/null`. The environment is only `HOME`, `USER`, `LOGNAME`, `SHELL` (from the owner's passwd entry) and `PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin`.
- The command runs in its own process group. Timeout, cancel and a dropped connection kill the whole group: `SIGTERM`, then `SIGKILL` after 5 seconds.
- Timeout is 10 minutes. The agent's `--command-timeout` (seconds) changes it.
- When the session ends, the agent kills its running commands. The hub marks runs that never got a `command_exit` as `lost` when the node disconnects.

### Output

Stdout and stderr are sent as they arrive, in chunks of up to 32 KiB of UTF-8 text. Invalid UTF-8 is replaced with U+FFFD. `seq` starts at 0 and counts up across both streams, so the hub can order and de-duplicate them. After 4 MiB of output, the agent stops sending chunks but keeps reading so the command doesn't block. It then sets `truncated` in `command_exit`.

### Statuses

| `status` | Meaning | `exit_code` |
| --- | --- | --- |
| `exited` | The command ran and exited | its exit code (`128 + n` if killed by signal `n`) |
| `timeout` | Killed after the timeout | `null` |
| `cancelled` | Killed after `command_cancel` | `null` |
| `rejected` | Not started, see [Before running](#before-running) | `null` |
| `failed` | Couldn't be started (e.g. spawn or `setuid` failed) | `null` |

`command_cancel` for a run that has already finished, or is unknown, is ignored.

## Read logs

The dashboard can list and read the files in the log folders a project declares (`logs` in `config.vpspilot.json`), and nothing else.

```
hub → agent   log_list    data = { request_id, project_path }
agent → hub   log_files   data = { request_id, files, error }

hub → agent   log_open    data = { stream_id, project_path, file, before, max_bytes, follow }
agent → hub   log_data    data = { stream_id, seq, offset, data, … }   (0 or more)
agent → hub   log_end     data = { stream_id, reason, error }          (exactly one)
hub → agent   log_close   data = { stream_id }                         (optional)
```

An agent supports this only if its `SystemInfo.features` contains `"logs"`. The hub must not send `log_list` or `log_open` to other agents.

### Which files

1. `project_path` must be a project from the agent's latest search. The agent re-reads its `config.vpspilot.json` for every request, with the same checks as for [running commands](#before-running): not a symlink, not group- or world-writable.
2. Each entry of `logs` is a folder relative to the project directory. It is resolved (`..` and symlinks) and must stay inside the project directory. Folders that don't resolve inside it, or don't exist, are skipped. `storage/logs` pointing at `/var/log/app` is skipped.
3. A file is readable if it is a regular file directly in a declared folder, or in a folder directly below it (for daily folders like `logs/2026-10-25/app.log`), and it still resolves inside that folder after symlinks.
4. `file` in `log_open` is the path relative to the project directory, exactly as `log_files` returned it. Anything else gets `log_end` with reason `rejected`.
5. Compressed files (`.gz`, `.bz2`, `.xz`, `.zst`, `.zip`) are listed but can't be opened.

**The agent reads as the config file's owner.** Listing and reading run in a child process with the owner's user and groups, the same switch as for commands. A file the owner can't read can't be read from the dashboard, even when the agent runs as root.

`log_files` lists at most 1000 files, sorted by path. If there are more, `error` says so.

### Reading a file

`log_open` asks for one window of the file and, optionally, what is written after it:

- `before` is a byte offset; the agent reads up to `max_bytes` that end there. Omit it (or send `null`) for the end of the file. `max_bytes` defaults to 64 KiB and is clamped to `1 ..= 1048576`.
- The window starts at the beginning of a line: the agent drops the partial line it starts in, unless the window starts at offset 0.
- `follow: true` is only allowed when `before` is omitted. After the window, the agent keeps sending what is appended to the file, checking at least once a second. If the file is truncated or replaced (log rotation), the agent reopens it from the start and sets `rotated` on the next `log_data`.

`log_data.offset` is the byte offset in the file where `data` starts, so the dashboard can ask for the window before the first one it has ("load earlier"). `data` is UTF-8 text in chunks of up to 32 KiB. Invalid UTF-8 is replaced with U+FFFD. `seq` counts up from 0 per stream.

A stream ends with exactly one `log_end`:

| `reason` | Meaning |
| --- | --- |
| `done` | The window was sent and `follow` was false |
| `closed` | The hub sent `log_close` |
| `rejected` | Not allowed, see [Which files](#which-files); `error` says why |
| `error` | Reading failed, e.g. the file was deleted; `error` says why |
| `limit` | A followed stream reached 30 minutes. Open it again to keep following |

At most 8 streams are open at once on a node; more get `rejected`. When the session ends, the agent closes all its streams. The hub doesn't store log contents.

## Back up

The agent backs up what a project declares in `backups` in `config.vpspilot.json`: the folders in `dir`, the `env_file` and a dump of the database. It writes a zip into `local_dir` on the server and, if `s3` is set, uploads it to an S3-compatible bucket (Linode Object Storage, AWS S3, MinIO…). The hub never receives the backup itself, only its result.

```
hub → agent   backup_run       data = { run_id, project_path }                    (Back up now)
agent → hub   backup_started   data = { run_id, project_path, trigger, … }
agent → hub   backup_result    data = { run_id, project_path, status, … }         (exactly one per backup_started)
```

Backups start two ways: on the schedule the config file declares, run by the agent itself even when the hub is unreachable, or from `backup_run` (**Back up now** in the dashboard). For a scheduled backup, the agent chooses the `run_id`.

An agent supports this only if its `SystemInfo.features` contains `"backups"`. The hub must not send `backup_run` to other agents.

### What the config declares

```json
"backups": {
  "env_file": ".env",
  "zip_file_name": "shop_backup",
  "database": {
    "connection": "DB_CONNECTION", "host": "DB_HOST", "port": "DB_PORT",
    "username": "DB_USERNAME", "password": "DB_PASSWORD", "database_name": "DB_DATABASE"
  },
  "dir": ["storage/app"],
  "schedule": "daily",
  "at": "03:00",
  "local_dir": "../backups/shop",
  "retention_days": 7,
  "s3": {
    "access_key": "S3_ACCESS_KEY",
    "secret_key": "S3_SECRET_KEY",
    "bucket": "shop-backups",
    "region": "us-east-1",
    "endpoint": "https://us-east-1.linodeobjects.com",
    "prefix": "shop/",
    "path_style": false,
    "retention_days": 30
  }
}
```

| Field | Default | Meaning |
| --- | --- | --- |
| `env_file` | none | The project's env file, relative to the project directory. It goes into the zip, and the `database` and `s3` variables are read from it. |
| `zip_file_name` | the project's `name`, with other characters replaced by `_` | Base name of the zip. Letters, digits, `.`, `-` and `_` only. |
| `database` | none | The **names** of the variables in `env_file` that hold the database settings. Needs `env_file`. |
| `dir` | none | Folders to include, relative to the project directory. |
| `schedule` | none | `daily` or `weekly` (Sundays). Without it, backups only run from **Back up now**. |
| `at` | `03:00` | Time of day for `schedule`, `HH:MM` in the server's local time zone. |
| `local_dir` | none; a backup fails without it | Where the zips are written: relative to the project directory, or absolute. It must not be inside a folder in `dir`. |
| `retention_days` | `7` | Zips in `local_dir` older than this are deleted. `1 ..= 3650`. |
| `s3.access_key`, `s3.secret_key` | required with `s3` | The **names** of the variables in `env_file` that hold the keys, never the keys themselves. Needs `env_file`. |
| `s3.bucket`, `s3.region`, `s3.endpoint` | required with `s3` | The bucket, its region and the `https://` endpoint. |
| `s3.prefix` | `""` | Put in front of each object key. |
| `s3.path_style` | `false` | Use `endpoint/bucket/key` URLs instead of `bucket.endpoint/key` (MinIO needs this). |
| `s3.retention_days` | `30` | Objects under `prefix` older than this are deleted. `1 ..= 3650`. |

The fields added in v0.5 are checked when the config is read: a `schedule` other than `daily` or `weekly`, an invalid `at` or `retention_days`, or an incomplete `s3` makes the config unusable, like an incomplete command, and the agent reports it as a problem instead of a project. The older fields were accepted before backups ran, so they are checked only when a backup runs: no `local_dir`, an invalid `zip_file_name` or a `database` without `env_file` fails that backup. Because `database` and `s3` hold only variable names, the hub and the dashboard never see a password or a key.

### Before backing up

The agent sends `backup_result` with status `rejected` (and backs up nothing) unless all of these hold:

1. `project_path` is a project from the agent's latest search, and its re-read `config.vpspilot.json` passes the same checks as for [running commands](#before-running) and declares `backups`.
2. The agent can run as the file's owner: it runs as root, or it already runs as that user.
3. No other backup is running on the node. A scheduled backup waits for the running one instead; a `backup_run` is rejected.

The agent then checks the older fields (see above) and the paths, as the owner. Each entry of `dir`, `env_file` and a SQLite database file must resolve (`..` and symlinks) inside the project directory. A failed check ends the backup with status `failed` and says why.

### Backing up

- **As the config file's owner.** The backup runs in a child process with the owner's user and groups, the same switch as for commands, so it reads only what the owner can read and writes only where the owner can write.
- **Env file.** Read as `KEY=VALUE` lines. Blank lines, `#` comments and an `export ` prefix are ignored, and single or double quotes around a value are removed. An unquoted value ends at ` #`. There is no `${…}` expansion.
- **Database.** The value of the `connection` variable picks the tool. The tool must be installed on the server; the agent doesn't bundle one.

  | `connection` | Dump | Password |
  | --- | --- | --- |
  | `mysql`, `mariadb` | `mysqldump --single-transaction --quick --routines --triggers` | in a temporary `--defaults-extra-file`, mode `0600` |
  | `pgsql`, `postgres`, `postgresql` | `pg_dump --no-owner` (plain SQL) | `PGPASSWORD` |
  | `sqlite` | `sqlite3 <file> ".backup …"` | none; `database_name` is the file, relative to the project directory or absolute |

  Passwords never appear on a command line. Any other `connection` fails the backup.
- **Zip.** `<zip_file_name>-<UTC time>.zip`, e.g. `shop_backup-20261031T030000Z.zip`, written as `.part` and renamed when complete. It holds:

  ```
  manifest.json            project, created_at, agent_version, database type
  files/<path>             each file of `dir` and `env_file`, at its path in the project
  database/dump.sql        or database/database.sqlite
  ```

  Symlinks inside `dir` are not followed or stored. The zip is mode `0600` and `local_dir` is created with mode `0700` if it doesn't exist, because the zip holds the env file.
- **Retention.** After a successful backup, the agent deletes zips in `local_dir` named `<zip_file_name>-<UTC time>.zip` whose time is older than `retention_days`. It never deletes the newest one, nor other files.
- **Upload.** With `s3`, the zip is uploaded to `<prefix><file name>`, signed with AWS Signature Version 4, in parts of 16 MiB. The keys need permission to put, list and delete objects. After a successful upload, the agent deletes objects under `<prefix><zip_file_name>-` older than `s3.retention_days`. A failed upload doesn't remove the local zip: the backup still counts as `ok`, and `upload` says what went wrong.
- **Timeout.** One hour. The agent's `--backup-timeout` (seconds) changes it. The timeout kills the child and its tools. The next backup of the project removes the `.part` file and dump folder they leave behind.

A backup keeps running if the session with the hub ends. The agent keeps up to 50 `backup_result`s it couldn't send and sends them after it reconnects; they are lost if the agent restarts. The hub de-duplicates by `run_id`.

### Schedule

The agent checks the schedule every minute. It remembers when each project was last backed up by schedule in its state file, so a restart doesn't back up twice. If the agent wasn't running at the scheduled time, it runs the missed backup once when it's back.

### Statuses

| `status` | Meaning |
| --- | --- |
| `ok` | The zip was written to `local_dir`. See `upload` for the bucket |
| `failed` | No zip was kept: a path check, the env file, the dump, writing the zip or the timeout failed. `error` says which |
| `rejected` | Not started, see [Before backing up](#before-backing-up). No `backup_started` is sent first |

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
  "total_memory": 8221474816,
  "agent_version": "v0.4.0",
  "features": ["commands", "logs", "backups"]
}
```

`hostname` names the node on first enrollment. `total_memory` is in bytes. `agent_version` and `features` were added in v0.3. When they are missing, the hub treats the agent as v0.2 with no features. `features` lists what the agent can do beyond streaming stats and projects. The values are `commands` ([Run commands](#run-commands), v0.3), `logs` ([Read logs](#read-logs), v0.4) and `backups` ([Back up](#back-up), v0.5).

`CommandRun`

```json
{ "run_id": "0b6c3c3e-6a8f-4b7e-9d0a-2f3c1e7a9b10", "project_path": "/var/www/shop", "command": "build" }
```

`run_id` is a UUID chosen by the hub. `project_path` is the project's `path` from `ProjectSyncPayload`. `command` is a command **name**.

`CommandOutput`

```json
{ "run_id": "0b6c3c3e-…", "seq": 0, "stream": "stdout", "data": "> shop@1.0.0 build\n" }
```

`stream` is `stdout` or `stderr`.

`CommandExit`

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

`run_as` is the user the command ran as. It's empty if the command never started. `duration_ms` runs from start to exit, and is 0 if the command never started. `error` is a short reason for `rejected` and `failed` (e.g. `unknown command "deploy"`, `config.vpspilot.json is world-writable`), and empty otherwise.

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

`LogList`

```json
{ "request_id": "4f1d…", "project_path": "/var/www/shop" }
```

`LogFiles`

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

`path` is relative to the project directory. `size` is in bytes and `modified` is Unix seconds. `error` is set, and `files` empty, when the request was refused (e.g. `not a project found by this agent`), or when the list was cut at 1000 files.

`LogOpen`

```json
{ "stream_id": "9a2e…", "project_path": "/var/www/shop", "file": "storage/logs/laravel.log", "before": null, "max_bytes": 65536, "follow": true }
```

`stream_id` is a UUID chosen by the hub.

`LogData`

```json
{ "stream_id": "9a2e…", "seq": 0, "offset": 983040, "data": "[2026-10-25 08:00:01] production.INFO: …\n", "size": 1048576, "rotated": false }
```

`size` is the file's size when the chunk was read, so the dashboard can show how much of the file it has.

`LogEnd`

```json
{ "stream_id": "9a2e…", "reason": "rejected", "error": "not a file in this project's log folders" }
```

`BackupRun`

```json
{ "run_id": "c41a…", "project_path": "/var/www/shop" }
```

`run_id` is a UUID chosen by the hub.

`BackupStarted`

```json
{ "run_id": "c41a…", "project_path": "/var/www/shop", "trigger": "schedule", "started_at": 1793156400, "run_as": "deploy" }
```

`trigger` is `manual` (from `backup_run`) or `schedule`. `started_at` is Unix seconds.

`BackupResult`

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

`file`, `local_path` and `size` (bytes) are set when `status` is `ok`. `database` is the dump type, or empty without `database`. `upload.status` is `ok`, `failed` or `skipped` (no `s3`, or the backup didn't succeed). `error` is a short reason for `failed` and `rejected` (e.g. `mysqldump not found`, `DB_PASSWORD is not set in .env`), and empty otherwise.
