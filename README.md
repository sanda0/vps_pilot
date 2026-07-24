# VPS Pilot

VPS Pilot is a **server monitoring and management platform** designed for private VPS servers.  
It provides real-time monitoring, alerting, project management, and (future) cron job automation — all from a single dashboard.

---

## ✨ Features

### 📊 Real-time Monitoring
- The C++ agent in [`agent/`](agent/) is installed on each node.
- Agents send system metrics to the central server via TCP:
  - **CPU usage**
  - **Memory usage**
  - **Network statistics**
  - **Disk usage**
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
- Flexible alert conditions (CPU, Memory, Disk, Network)
- Alert history and tracking

---

### 🚀 Projects Management (Coming Soon)
- Each node can have multiple projects
- Projects require a `config.vpspilot.json` file
- Agents scan disks for project config files and send project metadata to the central server
- Central server will display available projects and allow:
  - Running predefined commands
  - Managing project logs
  - Backing up project directories and databases

**Sample `config.vpspilot.json`:**
```vps_pilot/docs/readme_draft.md#L1-1
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
| **Agent**          | C++20 + Conan     |
| **Central Server** | Golang            |
| **Dashboard**      | React + Vite      |
| **Database**       | SQLite (dual DB)  |
| **Deployment**     | Single executable |

### Architecture
- **Operational DB**: Users, nodes, alerts, projects
- **Timeseries DB**: Metrics data (CPU, Memory, Network stats)
- **Embedded UI**: React app bundled into Go binary
- **TCP Server**: Receives metrics from agents (default port `55001`)
- **HTTP/WebSocket**: REST API + real-time data streaming (default port `8080`)

---

## 📦 Quick Start

### Prerequisites
- **Go** 1.21+ ([install](https://go.dev/doc/install))
- **Node.js** 18+ or **Bun** ([install node](https://nodejs.org/) or [install bun](https://bun.sh/))
- **Git**

---

### 1. Clone the Repository
```vps_pilot/README.md#L1-1
git clone https://github.com/sanda0/vps_pilot.git
cd vps_pilot
```

---

### 2. Build Single Executable (Recommended)

The `build.sh` script handles everything: installs frontend dependencies, builds the React UI, and compiles it into the Go binary.

```vps_pilot/README.md#L1-1
chmod +x build.sh
./build.sh
```

What it does:
1. Installs Node.js dependencies (if needed)
2. Builds the React frontend (`client/dist/`)
3. Copies the built UI into `server/cmd/app/dist/`
4. Compiles the Go binary with the UI embedded

**Output:** `server/vps_pilot`

---

### 3. Configure Environment

```vps_pilot/README.md#L1-1
cd server
cp .env.example .env
```

Then open `.env` and fill in your values:

```vps_pilot/README.md#L1-1
# Database directory
DB_PATH=./data

# JWT settings
TOKEN_LIFESPAN=60
TOKEN_SECRET=your-secret-key-min-32-chars

# TCP server (receives metrics from agents)
TCP_SERVER_PORT=55001
AGENT_TOKEN=use-the-same-long-random-value-on-every-agent

# Email alerts (optional)
MAIL_HOST=smtp.gmail.com
MAIL_PORT=587
MAIL_USERNAME=your-email@gmail.com
MAIL_PASSWORD=your-app-password
MAIL_FROM_ADDRESS=noreply@vpspilot.com
```

> **Note:** `TOKEN_SECRET` must be at least 32 characters long.
Set the same `AGENT_TOKEN` value in each agent's `config.json`.

---

### 4. Run Database Migrations & Create Superuser

Migrations run automatically on startup, but you can also run them manually:

```vps_pilot/README.md#L1-1
cd server
./vps_pilot -migrate
```

Then create your admin account:

```vps_pilot/README.md#L1-1
./vps_pilot -create-superuser
```

Follow the interactive prompts to set a username and password.

---

### 5. Start the Server

```vps_pilot/README.md#L1-1
cd server
./vps_pilot
```

By default the server listens on port **8080**. To use a different port:

```vps_pilot/README.md#L1-1
./vps_pilot -port 9090
```

---

### 6. Access the Dashboard

Open your browser: **http://localhost:8080**

Log in with the credentials you created in step 4.

---

## 🚩 CLI Flags

All flags are passed directly to the `vps_pilot` binary:

| Flag                | Default  | Description                             |
|---------------------|----------|-----------------------------------------|
| `-port`             | `8080`   | HTTP server port                        |
| `-migrate`          | —        | Run database migrations and exit        |
| `-create-superuser` | —        | Create an admin user interactively and exit |
| `-create-makefile`  | —        | Generate a `Makefile` in `server/` and exit |

**Examples:**

```vps_pilot/README.md#L1-1
# Run on a custom port
./vps_pilot -port 3000

# Only run migrations (no server started)
./vps_pilot -migrate

# Create an admin user
./vps_pilot -create-superuser

# Generate Makefile helper
./vps_pilot -create-makefile
```

---

## 🔧 Development Mode

For development with hot reload:

### Backend (Terminal 1)
```vps_pilot/README.md#L1-1
cd server
go run main.go
```

Or with [air](https://github.com/air-verse/air) for hot reload:
```vps_pilot/README.md#L1-1
cd server
air
```

### Frontend (Terminal 2)
```vps_pilot/README.md#L1-1
cd client
npm install   # or: bun install
npm run dev   # or: bun run dev
```

**Access:**
- Frontend (hot reload): http://localhost:5173
- Backend API: http://localhost:8080

---

## 📋 Makefile Commands

Generate the Makefile first if you don't have one:

```vps_pilot/README.md#L1-1
cd server
./vps_pilot -create-makefile
```

Then use:

```vps_pilot/README.md#L1-1
cd server

# Migrations
make migrate              # Run database migrations
make db-info              # Show database info
make db-reset             # Reset databases

# Building
make build                # Build server only
make build-full           # Build with embedded UI (runs ../build.sh)
make sqlc                 # Generate SQLC code

# Running
make run                  # Run server
make dev                  # Run with hot reload (requires air)

# User Management
make create-superuser     # Create admin user

# Testing
make test                 # Run tests
make test-coverage        # Run tests with coverage

# Maintenance
make backup               # Backup databases
make clean                # Clean build artifacts
```

---

## ⚙️ Configuration

### Email Alerts
Configure in `.env`:
```vps_pilot/README.md#L1-1
MAIL_HOST=smtp.gmail.com
MAIL_PORT=587
MAIL_USERNAME=your-email@gmail.com
MAIL_PASSWORD=your-app-password
MAIL_FROM_ADDRESS=noreply@vpspilot.com
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

```vps_pilot/README.md#L1-1
vps_pilot/
├── client/                  # React + Vite frontend
│   ├── src/
│   │   ├── components/      # Reusable UI components
│   │   ├── pages/           # Page components
│   │   ├── hooks/           # Custom React hooks
│   │   └── lib/             # Utilities and API client
│   └── dist/                # Built frontend (generated, gitignored)
├── server/                  # Go backend
│   ├── cmd/
│   │   ├── app/             # HTTP server + embedded UI
│   │   │   └── dist/        # Embedded UI files (generated, gitignored)
│   │   └── cli/             # CLI tools (migrations, superuser, makefile)
│   ├── internal/
│   │   ├── db/              # Database layer (SQLC + migrations)
│   │   ├── handlers/        # HTTP handlers
│   │   ├── services/        # Business logic
│   │   ├── middleware/       # HTTP middleware
│   │   ├── tcpserver/       # TCP server for agent metrics
│   │   └── utils/           # Utilities
│   ├── data/                # SQLite databases (gitignored)
│   ├── main.go              # Entry point
│   └── vps_pilot            # Compiled binary (gitignored)
├── docs/                    # Documentation
├── build.sh                 # Full build script
└── README.md
```

---

## 🐳 Docker Deployment (Coming Soon)

Docker Compose setup will be available in a future release.

---

## 🔐 Security Notes

- Change default admin credentials immediately after first login
- Use a strong `TOKEN_SECRET` (minimum 32 characters)
- Keep the `data/` directory in a secure, backed-up location
- Use HTTPS in production (put a reverse proxy like nginx or Caddy in front)
- Restrict access to TCP port `55001` to trusted agent IPs via firewall rules

---

## 🐛 Troubleshooting

**Build fails:**
```vps_pilot/README.md#L1-1
# Check the frontend builds cleanly
cd client && npm run build

# Check the Go code compiles
cd server && go build .
```

**Server won't start:**
```vps_pilot/README.md#L1-1
# Check if port 8080 is already in use
lsof -i :8080

# Check database directory permissions
ls -la server/data/

# Verify .env exists
ls -la server/.env
```

**UI doesn't load after build:**
```vps_pilot/README.md#L1-1
# Check that dist was copied into the server
ls server/cmd/app/dist/

# If missing, rebuild
./build.sh
```

**Metrics not showing up:**
- Ensure the agent is installed and running on the target node
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
- [ ] Project management via `config.vpspilot.json`
- [ ] Remote command execution for projects
- [ ] Project backups (database + directories)
- [ ] Remote cron job creation and management
- [ ] Docker Compose deployment
- [ ] Multi-user support with roles
- [ ] API documentation (Swagger)
- [ ] Mobile app

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## 🧑‍💻 Author

Made with ❤️ by [Sandakelum](https://github.com/sanda0)

---

## 📜 License

This project is licensed under the MIT License.

---

## 📸 Screenshots

### Dashboard
![Dashboard](https://github.com/user-attachments/assets/fff1c368-9c8e-4bb6-9720-f9a7f46a2910)

### Monitoring
![Metrics View](https://github.com/user-attachments/assets/fff1c368-9c8e-4bb6-9720-f9a7f46a2910)

---

## 🔗 Links

- **C++ Agent**: [`agent/`](agent/)
- **Issues**: https://github.com/sanda0/vps_pilot/issues
- **Documentation**: [docs/](docs/)

---

**⭐ Star this repo if you find it useful!**
