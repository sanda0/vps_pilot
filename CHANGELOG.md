# Changelog

User-facing changes to VPS Pilot. The format follows
[Keep a Changelog](https://keepachangelog.com), and versions follow
[Semantic Versioning](https://semver.org). See [RELEASING.md](RELEASING.md).

## Unreleased

### Upgrade notes
- Upgrading the hub adds the `backup_runs` table to the operational database
  on start.
- Backups need agent v0.5.0 or later. Older agents keep working for
  everything else, and the dashboard refuses Back up now on them.
- A `backups` block needs `local_dir` before backups run. Configs without it
  stay valid; their backups fail and say so.
- Dumping a database needs its tool on the server: `mysqldump`, `pg_dump` or
  `sqlite3`.

### Added
- Back up: the agent zips the folders, env file and database dump a
  project's `config.vpspilot.json` declares into `local_dir`, on the
  `schedule` in the file or from **Back up now** on the project page, and can
  upload each zip to S3-compatible storage (Linode, AWS, MinIO). Old zips are
  deleted after `retention_days`. The **Backups** card shows each backup's
  status, size, path and upload result. Backups run as the config file's
  owner, and the S3 keys stay in the env file on the server. See the docs for
  restoring.

### Changed
- The project page has one **Commands** card for the commands and their runs,
  and the run history is paginated.
- The project page's **Project info** card is a compact row across the page,
  with a link to the node and a button to copy the path.

## v0.4.0 - 2026-10-31

### Upgrade notes
- Reading logs needs agent v0.4.0 or later. Older agents keep working for
  everything else, and the dashboard asks you to upgrade them when you open
  a project's logs. The hub's databases don't change.

### Added
- Read logs: the **Logs** card on the project page lists the files in the
  log folders a project's `config.vpspilot.json` declares, and follows the
  one you pick live, with **Load earlier** for older lines. The agent reads
  them as the config file's owner, and only files that stay inside a
  declared folder after symlinks are resolved. Nothing is stored on the hub.

## v0.3.0 - 2026-10-25

### Upgrade notes
- Upgrading the hub adds the `command_runs` and `command_run_output` tables
  to the operational database on start.
- Running commands needs agent v0.3.0 or later. Agents from v0.2.x keep
  working for monitoring and projects, and the dashboard refuses to run
  commands on them.
- The agent refuses to run commands from a `config.vpspilot.json` that is a
  symlink or writable by its group or others. Ubuntu's default umask creates
  files writable by their group, so run `chmod 644 config.vpspilot.json`
  on existing configs.

### Added
- Run project commands: each command declared in a project's
  `config.vpspilot.json` gets a **Run** button on the project page. The hub
  sends only the command's name; the agent reads the file again, runs the
  command as the file's owner in the project directory, and streams its
  output live to the dashboard. Runs can be cancelled.
- Run history: every run is recorded with who started it, the command line,
  the user it ran as, how it ended, its exit code, its duration and its
  output (up to 4 MiB).
- Agent `--command-timeout` (`VPS_PILOT_COMMAND_TIMEOUT`): seconds a command
  may run before it is killed. The default is 600.

### Changed
- The agent shuts down the same way on SIGTERM as on SIGINT, so a plain
  `kill` or `docker stop` also stops the commands it is running.

## v0.2.0 - 2026-10-24

### Upgrade notes
- Upgrading the hub adds a `disk_stat` table to the timeseries database on
  start. Agents from v0.1.x keep working and already send disk usage.
- To move an agent you run by hand to the systemd service, stop it and run
  `curl -fsSL https://vpspilot.dev/install-agent.sh | sudo sh -s -- --hub <hub>:55001`.
  If it used the default state file (`/var/lib/vps-pilot-agent/state.json`),
  it keeps its enrollment and needs no token. Otherwise use the command from
  **Re-enroll agent**.

### Added
- Disk usage: the hub stores the root filesystem's percent used, the
  Metrics tab charts it, and alerts can watch it (**Disk** metric). Agents
  already sent it, so they don't need upgrading.
- Nodes show whether their agent is connected: **Online**, or **Offline**
  with when it was last seen, on the node page and the node list.
- One-line agent install: **Nodes → Add node** shows a command that runs
  `https://vpspilot.dev/install-agent.sh`. It installs the agent release
  matching the hub, checks it against `SHA256SUMS`, sets it up as a systemd
  service, enrolls it, and removes the token from disk. Running it again
  upgrades the agent.
- Agent `.deb` packages for amd64 and arm64, with the systemd unit.
- Agent release archives include the systemd unit,
  `vps-pilot-agent.service`.

### Changed
- The dashboard sidebar shows the VPS Pilot wordmark instead of a
  placeholder icon and "Enterprise".

## v0.1.1 - 2026-10-24

### Fixed
- Network alerts compare sent traffic with the send threshold and received
  traffic with the receive threshold. Before, they compared both with an
  unset field and fired on any traffic. A threshold left empty isn't checked.
- Network alert thresholds can be entered in B/s, KB/s or MB/s, and alert
  cards show them in a readable unit. Before, they had to be typed in bytes
  per second with no unit shown.
- Alert cards show whether an alert is enabled. Before, every alert showed as
  Inactive and the edit form opened with Enabled off, so saving an edit
  turned the alert off.

## v0.1.0 - 2026-10-23

### Added
- Project discovery: the agent finds every `config.vpspilot.json` under its
  project directories and reports it to the hub.
- Per-node scan settings in the dashboard: automatic scan, interval,
  **Rescan now**, and the problems from the last scan.
- Projects added by hand, accepted only inside the agent's project directories
  unless it runs with `--allow-any-project-path`.
- `vps_pilot -version` and `vps-pilot-agent --version` print the release
  tag, and both log it at startup.
- Releases include a `SHA256SUMS` file for checking the downloads.
