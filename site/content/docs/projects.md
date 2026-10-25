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
    "dir": ["storage/app"]
  }
}
```

Only `name` is required. `{ "name": "blog" }` is a valid config.

| Field | Type | Description |
| --- | --- | --- |
| `name` | string | Required. Shown in the dashboard. |
| `tech` | list of strings | Free-form tags, for example `"laravel"` or `"node"`. |
| `commands` | list of `{ name, command }` | The only commands the dashboard can run, by `name` (see [Run a command](#run-a-command)). Each needs both fields, and names must be unique. |
| `logs` | list of paths | Log folders, relative to the project directory. |
| `backups.env_file` | path | The project's env file. |
| `backups.zip_file_name` | string | Base name for backup archives. |
| `backups.database` | object | The *names* of the variables in `env_file` that hold the database settings, not the values. |
| `backups.dir` | list of paths | Folders to include in a backup. |

A file larger than 64 KiB, invalid JSON, a missing `name`, an incomplete
command or a duplicate command name makes the config unusable. The agent
reports it as a problem instead of a project.

{{< callout soon >}}
Tailing the declared log folders, and backups (the env file, the folders and
a database dump) are planned. Today the project page shows what the file
declares for them.
{{< /callout >}}

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
