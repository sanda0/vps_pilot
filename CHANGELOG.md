# Changelog

User-facing changes to VPS Pilot. The format follows
[Keep a Changelog](https://keepachangelog.com), and versions follow
[Semantic Versioning](https://semver.org). See [RELEASING.md](RELEASING.md).

## Unreleased

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
