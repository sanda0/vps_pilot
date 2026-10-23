---
title: Building from source
weight: 60
description: Build the hub with the dashboard embedded, build the agent, and work on VPS Pilot locally.
---

## Requirements

- [Go](https://go.dev/doc/install) 1.25 or newer
- [Bun](https://bun.sh/) for the dashboard
- [Rust](https://rustup.rs/) stable for the agent

## Everything at once

```sh
./build.sh        # or: make build
```

This builds the dashboard, the hub with the dashboard embedded
(`server/vps_pilot`), and the agent (`agent/target/release/vps-pilot-agent`).

## Step by step

### 1. The dashboard

```sh
cd client
bun install
bun run build
```

Vite writes the build straight into `server/cmd/app/dist/` (see
`client/vite.config.ts`), which is where the hub embeds it from. There's no
copy step.

### 2. The hub

```sh
cd server
go build -ldflags "-s -w" -o vps_pilot .
```

The dashboard is embedded with `go:embed`, so build it first. The directory
must contain at least one file for the hub to compile at all: without a
dashboard build, `mkdir -p cmd/app/dist && touch cmd/app/dist/.gitkeep` gives
you a hub that serves the API but no dashboard.

The SQLite driver is pure Go, so the hub cross-compiles without a C
toolchain:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o vps_pilot .
```

### 3. The agent

```sh
cd agent
cargo build --release
```

For a static binary that runs on any Linux server, see
[Build a static binary](/docs/agent/#build-a-static-binary).

## Working on VPS Pilot

Run these from the repository root. `make help` lists every task.

```sh
cp server/.env.example server/.env   # then set TOKEN_SECRET
make install migrate superuser       # one-time setup
make dev                             # hub on :8080 + dashboard on http://localhost:5173
make agent VPS_PILOT_TOKEN=vpsp1...  # local agent; later runs are just `make agent`
make agent-reset                     # forget the local agent's enrollment
make lint test                       # before pushing
```

`make dev` runs the hub with hot reload when [air](https://github.com/air-verse/air)
is installed (plain `go run` otherwise) and the Vite dev server, which
forwards `/api` to the hub. Get the join token for `make agent` from
**Nodes → Add node** in the dashboard.

`make sqlc` regenerates the Go database code after you change a query in
`server/internal/db/sql/`.

Before you open a pull request, read [Scope](/docs/scope/) and keep the
change small: under 1000 changed lines per pull request and under 200 per
commit.
