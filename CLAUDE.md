# VPS Pilot

Monorepo: `server/` (Go hub), `client/` (dashboard), `agent/` (Rust agent).
Read `SCOPE.md` before adding a feature. `make help` lists every task.

## Pull requests and commits: keep them small

- Every PR must stay **under 1000 changed lines** and every commit **under
  200** (additions + deletions; `git diff --stat main...HEAD` for a PR,
  `git show --stat <commit>` for a commit). Tests count. Lockfiles and
  sqlc-generated code don't.
- Split a PR into commits that each build and pass tests on their own, so it
  can be reviewed commit by commit.
- If a change will be bigger than one PR, plan it as a stack of PRs that each
  build and work on their own (e.g. agent discovery → agent wiring → hub →
  dashboard), and list that stack before writing code.
- One concern per PR. No drive-by refactors or reformatting.
- Check the sizes before opening a PR. If either is over the limit, split it.

## Running locally

1. One-time: `cp server/.env.example server/.env` and set `TOKEN_SECRET`
   (`openssl rand -hex 32`), then `make install migrate superuser`.
2. `make dev`: hub on :8080 and dashboard on http://localhost:5173.
3. Agent: in the dashboard, go to Nodes → Add node and copy the join token, then
   `make agent VPS_PILOT_TOKEN=vpsp1...`. Later runs are just `make agent`.
   `make agent-reset` forgets the enrollment. `PROJECT_DIRS=~/some/dir` (comma-separated
   for several) limits where it looks for `config.vpspilot.json`; the default is
   `/var/www` and `/home`.

Before pushing: `make lint test`.
