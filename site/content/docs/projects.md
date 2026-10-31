---
title: Projects & config.vpspilot.json
linkTitle: Projects
weight: 30
description: How the agent finds projects, how to control scans from the dashboard, and every field of config.vpspilot.json.
---

A project is a directory on a server that holds a `config.vpspilot.json`
file. The file says what the project is and, later, what the dashboard may do
with it. The agent finds these files and reports them to the hub, which lists
them under **Projects**.

## A sample config

```json
{
  "name": "shop",
  "tech": ["laravel", "mysql"],
  "commands": [
    { "name": "build", "command": "npm run build" },
    { "name": "migrate", "command": "php artisan migrate --force" }
  ],
  "logs": ["storage/logs"],
  "backups": {
    "env_file": ".env",
    "zip_file_name": "shop_backup",
    "database": {
      "connection": "DB_CONNECTION",
      "host": "DB_HOST",
      "port": "DB_PORT",
      "username": "DB_USERNAME",
      "password": "DB_PASSWORD",
      "database_name": "DB_DATABASE"
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
      "retention_days": 30
    }
  }
}
```

Only `name` is required. `{ "name": "blog" }` is a valid config.

| Field | Type | Description |
| --- | --- | --- |
| `name` | string | Required. Shown in the dashboard. |
| `tech` | list of strings | Free-form tags, for example `"laravel"` or `"node"`. |
| `commands` | list of `{ name, command }` | The only commands the dashboard can run, by `name` (see [Run a command](#run-a-command)). Each needs both fields, and names must be unique. |
| `logs` | list of paths | Log folders, relative to the project directory, that the dashboard can read (see [Read logs](#read-logs)). |
| `backups.env_file` | path | The project's env file. It goes into each backup, and the `database` and `s3` variables are read from it. |
| `backups.zip_file_name` | string | Base name of the zip: letters, digits, `.`, `-` and `_`. Defaults to the project's `name`, with other characters replaced by `_`. |
| `backups.database` | object | The *names* of the variables in `env_file` that hold the database settings, not the values. |
| `backups.dir` | list of paths | Folders to include in a backup. |
| `backups.schedule` | `daily` or `weekly` | When the agent backs up on its own; `weekly` is Sundays. Without it, backups only run from **Back up now**. |
| `backups.at` | `HH:MM` | Time of day for `schedule`, in the server's time zone. Defaults to `03:00`. |
| `backups.local_dir` | path | Where the zips are written, relative to the project directory or absolute. Required for backups to run, and it must not be inside a `dir` folder. |
| `backups.retention_days` | number | Zips in `local_dir` older than this are deleted. Defaults to 7. |
| `backups.s3` | object | Also upload each zip to S3-compatible storage. See [Upload to a bucket](#upload-to-a-bucket). |

A file larger than 64 KiB, invalid JSON, a missing `name`, an incomplete
command or a duplicate command name makes the config unusable. So do an
invalid `schedule`, `at` or `retention_days`, or an incomplete `s3`. The agent
reports it as a problem instead of a project.

## Run a command

Each command in `commands` has a **Run** button on the project page. After
you confirm, the hub sends the agent the command's *name*. It never sends the
command itself.

The agent then:

1. reads the project's `config.vpspilot.json` again, so it always runs what
   the file says now;
2. refuses to go on if the file is a symlink or writable by its group or by
   others, or if the name isn't in its `commands`;
3. runs the command with `/bin/sh -c` in the project directory, **as the
   owner of `config.vpspilot.json`**, with that user's groups;
4. streams stdout and stderr to the dashboard as they arrive.

The output appears live under **Runs** on the project page, with stderr in
red. **Cancel** stops a running command.

### Who the command runs as

A command runs as the user who owns the config file, not as the agent's user.
That way, someone who can only write files in their own home directory can't
get a command run as root. Run as root, the agent can switch to any owner. Run
as another user (for example with `make agent`), it only runs configs that
user owns.

The file must not be writable by anyone but its owner:

```sh
chmod 644 config.vpspilot.json
```

Ubuntu creates files writable by their group by default (umask `002`), so new
configs usually need this. If you forget, the run shows as **Rejected** with
the reason `config.vpspilot.json is group- or world-writable; run chmod 644
config.vpspilot.json`.

### How it runs

- The working directory is the project directory, and stdin is empty.
- The environment holds only `HOME`, `USER`, `LOGNAME`, `SHELL` (from the
  owner's account) and a standard `PATH`. Load anything else in the command,
  for example `set -a; . ./.env; set +a; php artisan migrate --force`.
- A command is killed after 10 minutes. Change that with the agent's
  `--command-timeout`.
- The same command can't run twice at once in a project, and at most 4
  commands run at once on a server.
- Output past 4 MiB is cut off. The command keeps running.
- When a command ends, processes it left running in the background keep
  running. They still belong to the agent's systemd service, so restarting
  the agent stops them. Start long-running services with systemd instead.

### Run history

Every run is recorded: who started it, the command line at the time, the
user it ran as, its status, exit code, duration and output. **Runs** on the
project page lists them, newest first. Click one to see its output.

| Status | Meaning |
| --- | --- |
| Exit 0, Exit *n* | The command ran and exited with that code. A command killed by a signal shows 128 + the signal number. |
| Timeout | Killed after `--command-timeout`. |
| Cancelled | Stopped with **Cancel**. |
| Rejected | The agent refused to start it. The reason is shown. |
| Failed | The command couldn't be started. |
| Lost | The agent disconnected, or the hub restarted, during the run. The agent stops its commands when its connection drops. |

Running commands needs agent v0.3.0 or later. On older agents, **Run** shows
an error asking you to upgrade.

## Read logs

The **Logs** card on the project page lists the files in the project's `logs`
folders. Pick one to see its last 64 KiB and follow it as lines are written.
**Load earlier** adds the 256 KiB before what you see. Following stops after
30 minutes; **Follow again** picks it up from the end of the file.

The agent reads the files **as the owner of `config.vpspilot.json`**, the same
way it runs commands, and refuses a config that is a symlink or writable by
its group or others. A file that user can't read can't be read from the
dashboard, even when the agent runs as root.

What can be read:

- files directly in a declared folder, or one folder below it (for daily
  folders like `storage/logs/2026-10-25/app.log`);
- only after `..` and symlinks are resolved: a declared folder must stay
  inside the project directory, and a file inside its folder. A
  `storage/logs` that links to `/var/log/app` is skipped;
- compressed files (`.gz`, `.bz2`, `.xz`, `.zst`, `.zip`) are listed but
  can't be opened;
- at most 1000 files are listed, and at most 8 files can be open at once on
  a server.

When a log is rotated (truncated or replaced), the view starts again from the
new file. The hub passes the lines on and doesn't store them.

Reading logs needs agent v0.4.0 or later. On older agents, the Logs card shows
an error asking you to upgrade.

## Back up

The agent backs up what `backups` declares into a zip: the `dir` folders, the
`env_file` and a dump of the database. It writes the zip to `local_dir` on the
server and, if `s3` is set, uploads it to a bucket. The hub only records how
each backup went; the backup itself never leaves the server and the bucket.

A backup runs:

- **on the schedule** in the config file (`schedule` and `at`). The agent runs
  it itself, so it happens even when the hub is down. A backup missed while
  the agent wasn't running runs once when it's back, and adding a schedule
  doesn't back up at once;
- **from Back up now** on the project page's **Backups** card.

The card shows what the project backs up and where, and its recent backups:
status, who started it (or "Scheduled"), size, the zip's path and how the
upload went. One backup runs at a time on a server; pressing Back up now while
another runs records it as rejected.

### What a backup does

The agent re-reads the config with the same checks as for
[running a command](#who-the-command-runs-as), then runs the backup **as the
owner of `config.vpspilot.json`**, so it only reads and writes what that user
can.

- **Paths.** Each `dir`, the `env_file` and a SQLite database file must stay
  inside the project directory after `..` and symlinks are resolved.
  Symlinks inside the folders are skipped.
- **Database.** The value of the `connection` variable picks the tool, which
  must be installed on the server: `mysql` or `mariadb` use `mysqldump`,
  `pgsql`, `postgres` or `postgresql` use `pg_dump`, and `sqlite` uses
  `sqlite3`, with `database_name` as the file. Passwords never go on a command
  line.
- **The zip** is named like `shop_backup-20261031T030000Z.zip` (UTC time) and
  holds `manifest.json`, the files under `files/` at their paths in the
  project, and the dump under `database/` (`dump.sql` or `database.sqlite`).
  It's readable only by its owner (mode `0600`), because it holds the env
  file, and `local_dir` is created with mode `0700`.
- **Retention.** After each backup, zips of this project older than
  `retention_days` are deleted from `local_dir`, and older than
  `s3.retention_days` from the bucket. The newest is always kept.
- **Timeout.** A backup is stopped after an hour (`--backup-timeout` on the
  agent, in seconds).

Backups need agent v0.5.0 or later. On older agents, Back up now shows an
error asking you to upgrade.

### Upload to a bucket

`s3` uploads each zip to Linode Object Storage, AWS S3, MinIO or any other
S3-compatible storage:

| Field | Description |
| --- | --- |
| `access_key`, `secret_key` | The **names** of the variables in `env_file` that hold the keys, never the keys themselves. The hub and the dashboard only ever see the names. |
| `bucket`, `region`, `endpoint` | The bucket, its region and its `https://` endpoint, e.g. `https://us-east-1.linodeobjects.com`. |
| `prefix` | Optional. Put in front of each object's name, e.g. `shop/`. |
| `path_style` | Optional. `true` for `endpoint/bucket/key` URLs, which MinIO needs. |
| `retention_days` | Optional. Zips in the bucket older than this are deleted. Defaults to 30. |

The keys need permission to put, list and delete objects in the bucket. If an
upload fails, the zip is still in `local_dir` and the backup counts as done;
the card shows why the upload failed.

### Restore a backup

There is no restore button: restoring overwrites files and a database, so you
do it on the server, as the project's owner.

1. Get the zip: from `local_dir`, or download it from the bucket (for example
   with the Linode Cloud Manager, `s3cmd get` or `aws s3 cp`).
2. Unzip it somewhere private:

   ```sh
   mkdir -m 700 /tmp/restore
   unzip shop_backup-20261031T030000Z.zip -d /tmp/restore
   ```

3. Put the files back. `files/` mirrors the project directory, env file
   included:

   ```sh
   rsync -a /tmp/restore/files/ /var/www/shop/
   ```

4. Load the database. The dump doesn't create the database, so create an
   empty one first if it's gone:
   - **MySQL / MariaDB**

     ```sh
     mysql -u shop -p -e 'CREATE DATABASE IF NOT EXISTS shop'
     mysql -u shop -p shop < /tmp/restore/database/dump.sql
     ```

   - **PostgreSQL** (the dump has no owner statements, so the user you load it
     with owns the tables)

     ```sh
     createdb -U shop shop
     psql -U shop -d shop -f /tmp/restore/database/dump.sql
     ```

   - **SQLite**: stop the app, then copy the file back over the database:

     ```sh
     cp /tmp/restore/database/database.sqlite /var/www/shop/database/database.sqlite
     ```

5. Delete `/tmp/restore`, which holds the env file.

Try a restore on a spare server before you need one.

## How the agent finds projects

The agent searches its project directories: `/var/www` and `/home` unless you
set `--projects-dir` (see [Agent options](/docs/agent/#options)). For each
one it:

- looks up to 4 levels deep, so `/home/<user>/apps/<client>/<project>` is
  found;
- skips hidden directories, `node_modules`, `vendor` and `target`;
- doesn't follow symlinked directories;
- skips project directories that don't exist.

A search runs:

| When | Runs |
| --- | --- |
| The agent connects | always |
| The scan settings change | always |
| You press **Rescan now** | always |
| Every scan interval | only while automatic scan is on |

Each report is the complete list, so a project whose file was deleted
disappears from the dashboard after the next search.

## Scan settings

Open **Projects → Scan settings** and pick the node. Each node has its own
settings:

- **Automatic scan**: on or off. On by default.
- **Interval**: from every minute to once a day. The default is every 15
  minutes.
- **Rescan now**: search straight away. This needs the agent to be connected;
  an offline agent searches when it reconnects.
- The problems from the last search: broken config files and rejected paths,
  each with the reason.

## Add a project by hand

Some projects live deeper than 4 levels, or in a directory the search skips.
Use **Projects → Add project** and enter the project directory's absolute
path. The agent then reads `<path>/config.vpspilot.json` directly.

The agent, not the dashboard, decides whether to accept the path. It resolves
`..` and symlinks first, and the result must be inside one of its project
directories. Anything else shows up as a problem in **Scan settings** with the
reason `outside the agent's project directories`.

To allow paths anywhere on the server, start the agent with
`--allow-any-project-path` (or `VPS_PILOT_ALLOW_ANY_PROJECT_PATH=true`). Only
do that if you trust everyone who can log in to the dashboard.

## Delete a project

**Delete** on a project page removes the hub's record of it. If the config
file is still on the server, the project comes back the next time the agent
reports: when something on the server changes, when the agent reconnects, or
when you press **Rescan now**. Delete the file on the server to remove the
project for good.
