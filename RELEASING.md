# Releasing VPS Pilot

How VPS Pilot is versioned and released, and what that means for your pull
requests. Only maintainers tag releases, but every contributor helps by keeping
`main` releasable and the changelog current.

## Versions

- The hub, the agent and the dashboard share **one version** and are released
  together from one Git tag: `vMAJOR.MINOR.PATCH` (for example `v0.2.0`).
- We follow [Semantic Versioning](https://semver.org). While the version is
  `0.x`:
  - **MINOR** (`0.2.0` → `0.3.0`): new features, and anything that breaks
    users (a protocol change old agents can't speak, a renamed setting or flag,
    a migration that can't be undone).
  - **PATCH** (`0.2.0` → `0.2.1`): bug and security fixes only.
- `1.0.0` comes once the agent protocol, the `config.vpspilot.json` fields, the
  `.env` names and the hub flags are stable.
- Release candidates use the dotted form: `v0.2.0-rc.1`, `v0.2.0-rc.2`. The
  release workflow marks tags containing `alpha`, `beta` or `rc` as
  pre-releases.
- Tags are permanent. A pushed tag is never moved or deleted; a broken release
  is fixed by the next patch.
- `v0.0.1` to `v0.0.5` are milestone tags added after the fact. They have no
  published binaries.

## What this means for your pull request

1. **Work against `main`.** There are no `develop` or long-lived release
   branches. `main` must always be releasable: every PR builds and passes
   `make lint test`.
2. **Hide unfinished features.** A feature split over several PRs can land in
   pieces, but the dashboard must not offer it until it works end to end (the
   disabled "Coming soon" run buttons on the project page are the pattern).
3. **Add a changelog line.** If your PR changes what users see or do, add one
   line under `## Unreleased` in [CHANGELOG.md](CHANGELOG.md), in the right
   group: Added, Changed, Deprecated, Removed, Fixed or Security. Docs,
   website and CI changes don't need one.
4. **Say when something breaks.** If an upgrade needs more than replacing the
   binaries (all agents must be upgraded, a setting is renamed, a migration
   rewrites data), add it under `### Upgrade notes` in the `Unreleased`
   section and mention it in the PR description.
5. **Keep old agents working.** Users upgrade the hub first and their agents
   later. A new hub must work with agents from the previous minor release:
   - add new message types or optional fields instead of changing existing
     ones (both sides ignore types they don't know);
   - never rename or remove a field or message type an older agent uses; mark
     it deprecated in [docs/PROTOCOL.md](docs/PROTOCOL.md) and remove it one
     minor release later;
   - update `docs/PROTOCOL.md` in the same PR as the code.
6. **Keep config files working.** New `config.vpspilot.json` fields are
   optional with a default. A file that worked in one release keeps working in
   the next.
7. **Migrations go forward.** They run on every hub start. Write the
   `down.sql` too, but don't rely on it, and don't add destructive migrations
   in a patch release.
8. **Document unreleased features carefully.** The website deploys from `main`
   on every merge, so it describes `main`, not the last release. Mark docs for
   a feature that isn't released yet with `{{< callout soon >}}`.

## How a release happens (maintainers)

Minor releases ship when a milestone is done, about every 4 to 6 weeks. The
planned order is one theme per release (finish monitoring, run commands, logs,
backups, cron), then `1.0.0`.

1. **Release PR.** Rename `## Unreleased` in `CHANGELOG.md` to
   `## vX.Y.Z - YYYY-MM-DD` and start a new empty `## Unreleased` above it.
   Bring the website up to date with the release in the same PR:
   - set `version` in `site/hugo.toml` to `vX.Y.Z` (shown next to the logo);
   - for each feature in this release, remove its `{{< callout soon >}}`
     notes in `site/content/docs/`, its "Coming soon" status in
     `site/content/docs/scope.md`, and any "coming soon" text or badge on the landing page
     (`site/layouts/index.html`);
   - update any page that describes what changed (limits, install steps,
     screenshots).
2. **Release candidate.** After the PR is merged:

   ```sh
   git switch main && git pull
   make lint test
   git tag -a v0.2.0-rc.1 -m "v0.2.0-rc.1"
   git push origin v0.2.0-rc.1
   ```

3. **Test it on a real VPS.** Upgrade a hub that has existing data, check that
   an agent from the previous release still connects, enroll a new agent, and
   look at the charts, alerts and projects. During this short freeze, only
   fixes merge to `main`; tag `rc.2` and so on as needed.
4. **Release.** Tag the tested commit (`git tag -a v0.2.0 -m "v0.2.0"`) and
   push the tag.
5. **Release notes.** Replace the generated notes on the GitHub Release with
   the changelog section, upgrade notes first.

Pushing a `v*.*.*` tag runs
[.github/workflows/release.yml](.github/workflows/release.yml). It builds the
hub with the dashboard embedded and the static agent, both for Linux amd64
and arm64, then creates the GitHub Release with the archives.

### Patch releases

- **If `main` has only fixes since the release:** merge the fix, add its
  changelog line, and tag `vX.Y.(Z+1)` on `main`.
- **If `main` already holds unreleased features:** branch from the release
  tag (`git switch -c release/0.2 v0.2.0`), cherry-pick the fix from `main`
  (via a PR into `release/0.2`), and tag the patch on that branch. Only the
  latest minor release gets patches.
