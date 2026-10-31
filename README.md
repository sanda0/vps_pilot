# VPS Pilot

VPS Pilot is a **server monitoring and management platform** designed for private VPS servers.  
It provides real-time monitoring, alerting, project management, and (future) cron job automation — all from a single dashboard.

---

## ✨ Features

### 📊 Real-time Monitoring
- A lightweight Rust agent ([agent/](agent/)) on each node (server), enrolled with a one-time join token
- Agents send system metrics to the central server over TLS ([protocol](docs/PROTOCOL.md)):
  - **CPU usage**
  - **Memory usage**
  - **Network statistics**
  - **Disk usage** of the root filesystem
- Metrics are visualized in the dashboard with selectable time ranges:
  - 5 minutes, 15 minutes, 1 hour, 1 day, 2 days, 7 days
- Real-time updates via WebSocket connection
- Interactive charts with historical data

---

### 🚨 Smart Alerting
- Configure alerts based on metric thresholds
- Multiple notification channels supported:
  - **Discord** ✅
  - **Email** ✅
  - **Slack** ✅
- Alert conditions on CPU, memory and network
- Alert history and tracking

---

### 🚀 Projects Management
- Each node can have multiple projects; each needs a `config.vpspilot.json` in its directory
- Agents find these files under their project directories (`--projects-dir`, default `/var/www` and `/home`, 4 levels deep) and report them to the hub
- Per node, in **Projects → Scan settings**: automatic scan on/off, scan interval, **Rescan now**, and the problems from the last scan (broken configs, rejected paths)
- **Projects → Add project** points a node's agent at a directory the search doesn't reach. The agent only accepts it inside its project directories unless started with `--allow-any-project-path`
- **Run** a command declared in the file from the project page, with live output and a history of every run. The agent runs it as the owner of `config.vpspilot.json`
- **Read** the files in the log folders the file declares from the project page, and follow one live. The agent reads them as the owner of `config.vpspilot.json`
- Planned, for what the file declares only:
  - Backing up project directories and databases

**Sample `config.vpspilot.json`:**
```json
{
  "name": "meta ads dashboard",
  "tech": ["laravel", "react", "mysql"],
  "logs": [],
  "commands": [
    { "name": "node build", "command": "npm run build" },
    { "name": "php build", "command": "composer install" }
  ],
  "backups": {
    "env_file": ".env",
    "zip_file_name": "project_backup",
    "database": {
      "connection": "DB_CONNECTION",
      "host": "DB_HOST",
      "port": "DB_PORT",
      "username": "DB_USERNAME",
      "password": "DB_PASSWORD",
      "database_name": "DB_DATABASE"
    },
    "dir": [
      "storage/app",
      "database/companies"
    ]
  }
}
```

---

### ⏲️ Cron Jobs Management (Planned)
- Remote cron job creation and management
- Schedule tasks across multiple nodes
- Monitor job execution and logs
- **Status:** Not yet implemented

---

## 🛠️ Tech Stack

| Component          | Technology        |
|--------------------|-------------------|
| **Agent**          | Rust              |
| **Central Server** | Golang            |
| **Dashboard**      | React + Vite      |
| **Database**       | SQLite (dual DB)  |
| **Deployment**     | Single executable |

### Architecture
- **Operational DB**: Users, nodes, alerts, projects
- **Timeseries DB**: Metrics data (CPU, Memory, Network stats)
- **Embedded UI**: React app bundled into Go binary
- **Agent channel**: TLS server that receives metrics from enrolled agents (default port `55001`)
- **HTTP/WebSocket**: REST API + real-time data streaming (default port `8080`)

---

## 📦 Quick Start

### Prerequisites
- **Go** 1.25+ ([install](https://go.dev/doc/install))
- **Bun** ([install](https://bun.sh/)) for the dashboard
- **Rust** stable ([install](https://rustup.rs/)) for the agent
- **Git**

---

### 1. Clone the Repository
```sh
git clone https://github.com/sanda0/vps_pilot.git
cd vps_pilot
```

---

### 2. Build Single Executable (Recommended)

The `build.sh` script (or `make build`) installs dashboard dependencies, builds the UI into the Go binary, and builds the agent.

```sh
chmod +x build.sh
./build.sh
```

What it does:
1. Installs dashboard dependencies with bun (if needed)
2. Builds the dashboard straight into `server/cmd/app/dist/`
3. Compiles the Go binary with the UI embedded
4. Builds the agent with cargo (skipped if cargo isn't installed)

**Output:** `server/vps_pilot` (hub) and `agent/target/release/vps-pilot-agent` (agent)

---

### 3. Configure Environment

```sh
cd server
cp .env.example .env
```

Then open `.env` and set at least `TOKEN_SECRET`:

```sh
openssl rand -hex 32   # paste the output as TOKEN_SECRET
```

The other settings (`DB_PATH`, `TOKEN_LIFESPAN`, `COOKIE_SECURE`, `TCP_SERVER_PORT`, SMTP for email alerts) are explained in [`server/.env.example`](server/.env.example).

---

### 4. Run Database Migrations & Create Superuser

Migrations run automatically on startup, but you can also run them manually:

```sh
cd server
./vps_pilot -migrate
```

Then create your admin account:

```sh
./vps_pilot -create-superuser
```

Follow the interactive prompts to set a username and password.

---

### 5. Start the Server

```sh
cd server
./vps_pilot
```

By default the server listens on port **8080**. To use a different port:

```sh
./vps_pilot -port 9090
```

---

### 6. Access the Dashboard

Open your browser: **http://localhost:8080**

Log in with the credentials you created in step 4.

---

### 7. Connect a Server (Agent)

1. In the dashboard, open **Nodes → Add node** and copy the join token.
2. Copy `vps-pilot-agent` to the server and run it once with the token:

```sh
VPS_PILOT_TOKEN=vpsp1.… ./vps-pilot-agent --hub your-hub.example.com
```

The agent enrolls, saves its credentials to `/var/lib/vps-pilot-agent/state.json`, and doesn't need the token again. The hub's agent port (`55001`) must be reachable from the server. See [agent/README.md](agent/README.md) for options and exit codes.

---

## 🚩 CLI Flags

All flags are passed directly to the `vps_pilot` binary:

| Flag                | Default  | Description                             |
|---------------------|----------|-----------------------------------------|
| `-port`             | `8080`   | HTTP server port                        |
| `-agent-port`       | `55001`  | Agent channel (TLS) port; also `TCP_SERVER_PORT` |
| `-agent-tls-cert` / `-agent-tls-key` | self-signed in `$DB_PATH/tls` | Your own certificate for the agent channel |
| `-migrate`          | —        | Run database migrations and exit        |
| `-create-superuser` | —        | Create an admin user interactively and exit |
| `-create-makefile`  | —        | Generate a `Makefile` in `server/` and exit |

**Examples:**

```sh
# Run on a custom port
./vps_pilot -port 3000

# Only run migrations (no server started)
./vps_pilot -migrate

# Create an admin user
./vps_pilot -create-superuser
```

---

## 🔧 Development

From the repo root (`make help` lists every task):

```sh
cp server/.env.example server/.env   # then set TOKEN_SECRET
make install migrate superuser       # one-time setup
make dev                             # hub on :8080 (hot reload with air) + dashboard on http://localhost:5173
make agent VPS_PILOT_TOKEN=vpsp1...  # run a local agent; later runs are just `make agent`
make agent PROJECT_DIRS=~/vpspilot-demo  # look for projects only there (default: /var/www,/home)
make agent-reset                     # forget the local agent's enrollment
make lint test                       # before pushing
```

### Working on the website

The website and docs at [vpspilot.dev](https://vpspilot.dev) live in [`site/`](site/) and are built with [Hugo](https://gohugo.io) **v0.167.0** (the standard edition is enough; no Node needed). On Ubuntu, install that exact version from the release:

```sh
curl -fsSLO https://github.com/gohugoio/hugo/releases/download/v0.167.0/hugo_0.167.0_linux-amd64.deb
sudo apt install ./hugo_0.167.0_linux-amd64.deb
```

Then `make site-dev` previews it at http://localhost:1313 and `make site-build` builds it into `site/public/`. Pushes to `main` that touch `site/` deploy it to GitHub Pages.

---

## ⚙️ Configuration

### Email Alerts
Configure in `.env`:
```sh
MAIL_HOST=smtp.gmail.com
MAIL_PORT=587
MAIL_USERNAME=your-email@gmail.com
MAIL_PASSWORD=your-app-password
MAIL_FROM_ADDRESS=noreply@example.com
```

### Slack Alerts
1. Go to your Slack workspace → **Apps → Incoming Webhooks**
2. Create a new webhook for your desired channel
3. Copy the webhook URL and paste it in the alert configuration in the dashboard

### Discord Alerts
1. Go to your Discord server settings → **Integrations → Webhooks**
2. Create a new webhook for your desired channel
3. Copy the webhook URL and paste it in the alert configuration in the dashboard

---

## 📂 Project Structure

```sh
vps_pilot/
├── agent/                   # Rust agent (one per server)
├── client/                  # React + Vite dashboard (built into server/cmd/app/dist)
│   └── src/
│       ├── components/      # Reusable UI components
│       ├── pages/           # Page components
│       ├── hooks/           # Custom React hooks
│       └── lib/             # Utilities and API client
├── server/                  # Go backend
│   ├── cmd/
│   │   ├── app/             # HTTP server + embedded UI
│   │   │   └── dist/        # Embedded UI files (generated, gitignored)
│   │   └── cli/             # CLI tools (migrations, superuser)
│   ├── internal/
│   │   ├── db/              # Database layer (SQLC + migrations)
│   │   ├── handlers/        # HTTP handlers
│   │   ├── services/        # Business logic
│   │   ├── middleware/       # HTTP middleware
│   │   ├── agentauth/       # Agent enrollment and keys
│   │   ├── tcpserver/       # Agent channel (TLS)
│   │   └── utils/           # Utilities
│   ├── data/                # SQLite databases (gitignored)
│   ├── main.go              # Entry point
│   └── vps_pilot            # Compiled binary (gitignored)
├── docs/                    # Documentation, agent protocol
├── build.sh                 # Full build script
├── Makefile                 # Development tasks
└── SCOPE.md                 # What VPS Pilot does and doesn't do
```

---

## 🐳 Docker Deployment (Coming Soon)

Docker Compose setup will be available in a future release.

---

## 🔐 Security Notes

- Use a strong random `TOKEN_SECRET` (`openssl rand -hex 32`)
- Keep the `data/` directory in a secure, backed-up location
- Use HTTPS in production (put a reverse proxy like nginx or Caddy in front)
- Agents authenticate with a per-node key issued at enrollment; join tokens are single-use and expire after 1 hour
- Keep `data/tls/` with the database: agents pin the hub certificate's fingerprint, so losing it means re-enrolling every agent
- Optionally restrict TCP port `55001` to your servers' IPs via firewall rules

---

## 🐛 Troubleshooting

**Build fails:**
```sh
# Check the frontend builds cleanly
cd client && bun run build

# Check the Go code compiles
cd server && go build .
```

**Server won't start:**
```sh
# Check if port 8080 is already in use
lsof -i :8080

# Check database directory permissions
ls -la server/data/

# Verify .env exists
ls -la server/.env
```

**UI doesn't load after build:**
```sh
# Check that dist was copied into the server
ls server/cmd/app/dist/

# If missing, rebuild
./build.sh
```

**Metrics not showing up:**
- Ensure the agent is installed, enrolled (**Nodes → Add node**) and running on the target node
- If the agent exits with code `3`, its credentials were rejected; re-enroll it from the node page
- Check that TCP port `55001` is open and reachable from the node
- Verify the node is registered in the dashboard

---

## 📅 Roadmap

- [x] Real-time metrics collection (CPU, Memory, Network, Disk)
- [x] WebSocket-based live updates
- [x] Discord alert integration
- [x] Email alert integration
- [x] Slack alert integration
- [x] SQLite dual-database architecture
- [x] Embedded UI in single binary
- [x] User authentication with JWT
- [x] Authenticated agent enrollment + TLS on the agent channel
- [x] Rust agent in this repo (`agent/`)
- [x] Project discovery via `config.vpspilot.json` (per-node scan settings, manual paths)
- [x] Run named project commands (declared in the config file only) with audit log
- [x] Log tailing for declared log folders
- [ ] Project backups (database + directories)
- [ ] Cron jobs declared per project
- [x] Install script + systemd unit for the agent
- [ ] Docker Compose deployment
- [ ] Multi-user support with roles
- [ ] API documentation (Swagger)
- [ ] Mobile app

---

## 🤝 Contributing

Contributions are welcome! Before you start:

- Read [SCOPE.md](SCOPE.md): it lists what VPS Pilot does and what belongs in other tools.
- Keep pull requests under 1000 changed lines and each commit under 200, with one concern per pull request.
- Run `make lint test` before pushing.
- If your change affects users, add a line to [CHANGELOG.md](CHANGELOG.md). [RELEASING.md](RELEASING.md) explains versions and releases.

---

## 🧑‍💻 Author

Made with ❤️ by [Sandakelum](https://github.com/sanda0)

---

## 📜 License

This project is licensed under the [GNU Affero General Public License v3.0](LICENSE) (AGPL-3.0-only). Versions released before this change remain available under the MIT License.

---

## 🔗 Links

- **Website**: https://vpspilot.dev
- **Agent**: [agent/](agent/) · [protocol](docs/PROTOCOL.md)
- **Issues**: https://github.com/sanda0/vps_pilot/issues
- **Documentation**: [docs/](docs/)

---

**⭐ Star this repo if you find it useful!**