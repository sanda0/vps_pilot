# VPS Pilot

VPS Pilot is a **server monitoring and management platform** designed for private VPS servers.  
It provides real-time monitoring, alerting, project management, and (future) cron job automation — all from a single dashboard.

---

## ✨ Features

### 📊 Real-time Monitoring
- Agents installed on each node (server). ([Agent repo](https://github.com/sanda0/vps_pilot_agent))
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

| Component        | Technology          |
|------------------|---------------------|
| **Agent**        | Golang             |
| **Central Server** | Golang           |
| **Dashboard**    | React + Vite       |
| **Database**     | SQLite (dual DB)   |
| **Deployment**   | Single executable  |

### Architecture
- **Operational DB**: Users, nodes, alerts, projects
- **Timeseries DB**: Metrics data (CPU, Memory, Network stats)
- **Embedded UI**: React app bundled into Go binary
- **TCP Server**: Receives metrics from agents (port 55001)
- **HTTP/WebSocket**: REST API + real-time data streaming

---

## 📦 Quick Start

### Prerequisites
- **Go** 1.21+ ([install](https://go.dev/doc/install))
- **Node.js** 18+ or **Bun** ([install node](https://nodejs.org/) or [install bun](https://bun.sh/))
- **Git**

### 1. Clone the Repository
```bash
git clone https://github.com/sanda0/vps_pilot.git
cd vps_pilot
```

### 2. Build Single Executable (Recommended)
```bash
# Build both frontend and backend into single binary
./build.sh

# Binary created at: server/vps_pilot
```

### 3. Configure Environment
```bash
cd server
cp .env.example .env
# Edit .env with your settings
```

**Required `.env` variables:**
```env
DB_PATH=./data
TOKEN_LIFESPAN=60
TOKEN_SECRET=your-secret-key-min-32-chars
TCP_SERVER_PORT=55001

# Email alerts (optional)
MAIL_HOST=smtp.gmail.com
MAIL_PORT=587
MAIL_USERNAME=your-email@gmail.com
MAIL_PASSWORD=your-app-password
MAIL_FROM_ADDRESS=noreply@vpspilot.com
```

### 4. Create Superuser
```bash
# Migrations run automatically on first start
./vps_pilot -create-superuser
```

### 5. Start the Server
```bash
./vps_pilot
```

### 6. Access Dashboard
Open your browser: **http://localhost:8000**

Login with the credentials you created in step 4.

---

## 🔧 Development Mode

For development with hot reload:

### Backend (Terminal 1)
```bash
cd server
go run main.go
```

### Frontend (Terminal 2)
```bash
cd client
npm install
npm run dev
```

**Access:**
- Frontend: http://localhost:5173
- Backend API: http://localhost:8000

---

## 📋 Available Commands

```bash
cd server

# Migrations
make migrate              # Run database migrations
make db-info             # Show database info
make db-reset            # Reset databases

# Building
make build               # Build server only
make build-full          # Build with embedded UI
make sqlc                # Generate SQLC code

# Running
make run                 # Run server
make dev                 # Run with hot reload (requires air)

# User Management
make create-superuser    # Create admin user

# Testing
make test               # Run tests
make test-coverage      # Run tests with coverage

# Maintenance
make backup             # Backup databases
make clean              # Clean build artifacts
```

---

## ⚙️ Configuration

### Email Alerts
Configure in `.env` for email notifications:
```env
MAIL_HOST=smtp.gmail.com
MAIL_PORT=587
MAIL_USERNAME=your-email@gmail.com
MAIL_PASSWORD=your-app-password
MAIL_FROM_ADDRESS=noreply@vpspilot.com
```

### Slack Alerts
1. Go to your Slack workspace
2. Navigate to Apps → Incoming Webhooks
3. Create a new webhook for your desired channel
4. Copy the webhook URL and paste it in the alert configuration

### Discord Alerts
1. Go to your Discord server settings
2. Navigate to Integrations → Webhooks
3. Create a new webhook for your desired channel
4. Copy the webhook URL and paste it in the alert configuration

---

## 🐳 Docker Deployment (Coming Soon)

Docker Compose setup will be available in future releases.

---

## 📂 Project Structure

```
vps_pilot/
├── client/              # React frontend
│   ├── src/
│   │   ├── components/  # Reusable UI components
│   │   ├── pages/       # Page components
│   │   ├── hooks/       # Custom React hooks
│   │   └── lib/         # Utilities and API client
│   └── dist/            # Built frontend (after build)
├── server/              # Go backend
│   ├── cmd/
│   │   ├── app/         # Main application
│   │   │   └── dist/    # Embedded UI (after build)
│   │   └── cli/         # CLI tools
│   ├── internal/
│   │   ├── db/          # Database layer
│   │   ├── handlers/    # HTTP handlers
│   │   ├── services/    # Business logic
│   │   ├── middleware/  # HTTP middleware
│   │   ├── tcpserver/   # TCP server for agents
│   │   └── utils/       # Utilities
│   └── data/            # SQLite databases
├── docs/                # Documentation
├── build.sh             # Build script
└── README.md
```

---

## 🔐 Security Notes

- Change default admin credentials immediately
- Use a strong `TOKEN_SECRET` (min 32 characters)
- Keep databases in a secure location
- Use HTTPS in production (reverse proxy recommended)
- Secure TCP port 55001 with firewall rules

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

## 🐛 Troubleshooting

**Build fails:**
```bash
# Check frontend builds
cd client && npm run build

# Check Go compiles
cd server && go build .
```

**Server won't start:**
```bash
# Check if port 8000 is in use
lsof -i :8000

# Check database permissions
ls -la data/
```

**Metrics not showing:**
- Ensure agent is installed and running on nodes
- Check TCP port 55001 is open
- Verify node is registered in dashboard

For more help, see [QUICKSTART.md](QUICKSTART.md) or [docs/BUILDING.md](docs/BUILDING.md)

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

- **Agent Repository**: https://github.com/sanda0/vps_pilot_agent
- **Issues**: https://github.com/sanda0/vps_pilot/issues
- **Documentation**: [docs/](docs/)

---

**⭐ Star this repo if you find it useful!**

