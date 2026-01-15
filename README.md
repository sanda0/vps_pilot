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

### 🚀 Projects Management with Nix (Coming Soon)

VPS Pilot provides **Heroku-like simplicity** with Nix power under the hood:
- ✅ **No Nix knowledge required** - just specify project type and versions
- ✅ **Built-in templates** for Laravel, Node.js, Go, Python, and more
- ✅ **Isolated environments** per project (no version conflicts)
- ✅ **Reproducible builds** across all nodes
- ✅ **Different runtime versions** per project
- ✅ **Automatic rollback** capability

#### How It Works (Simple!)

1. **User configures `config.vpspilot.json`**:
   - Set project type: "laravel", "nodejs", "go", etc.
   - Specify versions: PHP 8.2, Node 20, Go 1.21
   - Define build/start commands
2. **Agent clones GitHub repo** to the node
3. **Agent auto-generates Nix environment** from template
4. **Project deploys automatically** with isolated dependencies

**No `flake.nix` needed!** VPS Pilot handles it for you.

#### Project Structure on Node

```
/opt/vpspilot/projects/
├── project-123-myapp/
│   ├── flake.nix              # Nix environment definition
│   ├── flake.lock             # Locked dependencies
│   ├── config.vpspilot.json   # VPS Pilot configuration
│   ├── .env                   # Environment variables
│   └── src/                   # Application code
├── project-456-api/
│   ├── flake.nix
│   ├── config.vpspilot.json
│   └── ...
└── project-789-frontend/
    ├── flake.nix
    ├── config.vpspilot.json
    └── ...
```

#### Simple `config.vpspilot.json` Examples

**Laravel Project (Auto-generated Nix):**
```json
{
  "name": "My Laravel App",
  "type": "laravel",
  "runtime": {
    "php": "8.2",
    "node": "20",
    "composer": "latest"
  },
  "build": {
    "commands": [
      "composer install --no-dev --optimize-autoloader",
      "npm install",
      "npm run build"
    ]
  },
  "start": {
    "command": "php artisan serve --host=0.0.0.0 --port=$PORT"
  },
  "env": {
    "APP_ENV": "production"
  },
  "database": {
    "type": "mysql",
    "version": "8.0"
  }
}
```

**Node.js API (Auto-generated Nix):**
```json
{
  "name": "Express API",
  "type": "nodejs",
  "runtime": {
    "node": "20",
    "npm": "latest"
  },
  "build": {
    "commands": [
      "npm install",
      "npm run build"
    ]
  },
  "start": {
    "command": "npm start"
  },
  "healthcheck": {
    "url": "/health",
    "interval": 30
  }
}
```

**Go Microservice (Auto-generated Nix):**
```json
{
  "name": "Go API",
  "type": "go",
  "runtime": {
    "go": "1.21"
  },
  "build": {
    "commands": [
      "go build -o app ."
    ]
  },
  "start": {
    "command": "./app"
  }
}
```

**Python/Django (Auto-generated Nix):**
```json
{
  "name": "Django App",
  "type": "python",
  "runtime": {
    "python": "3.11",
    "pip": "latest"
  },
  "build": {
    "commands": [
      "pip install -r requirements.txt",
      "python manage.py collectstatic --noinput"
    ]
  },
  "start": {
    "command": "gunicorn myapp.wsgi:application --bind 0.0.0.0:$PORT"
  }
}
```

**Static Site (Auto-generated Nix):**
```json
{
  "name": "React Frontend",
  "type": "static",
  "runtime": {
    "node": "20"
  },
  "build": {
    "commands": [
      "npm install",
      "npm run build"
    ],
    "output_dir": "dist"
  },
  "start": {
    "command": "npx serve -s dist -p $PORT"
  }
}
```

**Advanced: Custom Nix (Optional):**
```json
{
  "name": "Custom Project",
  "type": "custom",
  "nix": {
    "flake_file": "./custom-flake.nix"
  }
}
```
If `type: "custom"`, VPS Pilot uses your provided `flake.nix`.

#### Agent Command Flow

```
┌─────────────────────────────────────────────────────────┐
│ 1. Central Server sends deploy command                  │
│    { project_id, github_url, branch, commit }           │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 2. Agent clones repository                               │
│    /opt/vpspilot/projects/project-{id}-{name}/         │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 3. Parse config.vpspilot.json                           │
│    • Read project type: "laravel", "nodejs", etc.       │
│    • Read runtime versions: PHP 8.2, Node 20            │
│    • Read build/start commands                          │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 4. Select or generate Nix template                      │
│                                                          │
│    IF type == "custom" AND flake.nix exists:           │
│      → Use provided flake.nix                          │
│                                                          │
│    ELSE:                                                │
│      → Load built-in template for project type          │
│      → Inject runtime versions from config              │
│      → Generate flake.nix automatically                 │
│                                                          │
│    Templates available:                                 │
│      • laravel.nix.tmpl                                │
│      • nodejs.nix.tmpl                                 │
│      • go.nix.tmpl                                     │
│      • python.nix.tmpl                                 │
│      • static.nix.tmpl                                 │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 5. Build with Nix (creates isolated environment)        │
│    nix build .#default --log-format bar-with-logs       │
│                                                          │
│    Nix automatically handles:                            │
│    • Download PHP 8.2, Node 20, etc.                    │
│    • Create isolated environment                        │
│    • No conflicts with other projects                   │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 6. Run build commands from config                       │
│    nix develop --command composer install               │
│    nix develop --command npm install                    │
│    nix develop --command npm run build                  │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 7. Create systemd service                              │
│    ExecStart=nix develop --command {start.command}     │
│    - Auto-restart on failure                            │
│    - Logging to journald                                │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 8. Health check and report status                       │
│    { status: "running", pid, port, version }           │
└─────────────────────────────────────────────────────────┘
```

#### Upgrade Strategy

**Option 1: Blue-Green Deployment**
```bash
# Agent creates new directory with new version
/opt/vpspilot/projects/project-123-myapp-v2/

# Build in new environment
cd project-123-myapp-v2
nix build

# Switch symlink atomically
ln -sfn project-123-myapp-v2 project-123-myapp-current

# Start new version
nix run .#default

# Stop old version (kept for rollback)
```

**Option 2: In-Place Update with Rollback**
```bash
# Store current flake.lock as backup
cp flake.lock flake.lock.backup

# Pull new code
git pull origin main

# Update Nix dependencies
nix flake update

# Build and test
nix build
nix develop --command npm test

# If success: deploy
# If failure: rollback
git reset --hard HEAD@{1}
cp flake.lock.backup flake.lock
nix build  # Rebuild previous version
```

#### Rollback Strategy

```bash
# Agent keeps last N generations
nix profile history

# Rollback to previous generation
nix profile rollback

# Rollback to specific generation
nix profile rollback --to <generation-number>

# List all available generations
nix profile diff-closures

# Instant rollback: switch symlink back
ln -sfn project-123-myapp-v1 project-123-myapp-current
systemctl restart vpspilot-project-123
```

#### Multi-Version Support Example

```
Node A:
├── project-1 → Node.js 18 + PHP 7.4
├── project-2 → Node.js 20 + PHP 8.2
└── project-3 → Go 1.21 + Python 3.11

Node B:
├── project-4 → Node.js 16 + PHP 8.1
├── project-5 → Ruby 3.2 + PostgreSQL 15
└── project-6 → Rust + Node.js 20

No conflicts! Each project has isolated environment.
```

#### Built-in Templates

VPS Pilot includes pre-built Nix templates for common project types. Users just specify versions in `config.vpspilot.json`.

| Project Type | Template | Supported Versions |
|-------------|----------|-------------------|
| **Laravel** | `laravel.nix.tmpl` | PHP 7.4, 8.0, 8.1, 8.2, 8.3 |
| **Node.js** | `nodejs.nix.tmpl` | Node 16, 18, 20, 21 |
| **Go** | `go.nix.tmpl` | Go 1.19, 1.20, 1.21, 1.22 |
| **Python** | `python.nix.tmpl` | Python 3.9, 3.10, 3.11, 3.12 |
| **Django** | `django.nix.tmpl` | Python 3.9+ |
| **Static** | `static.nix.tmpl` | Node 16+ (for build tools) |
| **Ruby** | `ruby.nix.tmpl` | Ruby 2.7, 3.0, 3.1, 3.2 |
| **Custom** | User provides `flake.nix` | Any version |

#### Complete config.vpspilot.json Schema

```json
{
  "name": "My Application",
  "type": "laravel|nodejs|go|python|django|static|ruby|custom",
  
  "runtime": {
    "php": "8.2",
    "node": "20",
    "go": "1.21",
    "python": "3.11",
    "ruby": "3.2",
    "composer": "latest",
    "npm": "latest",
    "pip": "latest"
  },
  
  "database": {
    "type": "mysql|postgresql|sqlite|mongodb",
    "version": "8.0"
  },
  
  "cache": {
    "type": "redis|memcached",
    "version": "latest"
  },
  
  "build": {
    "commands": [
      "composer install",
      "npm install",
      "npm run build"
    ]
  },
  
  "start": {
    "command": "php artisan serve --host=0.0.0.0 --port=$PORT",
    "port": 3000
  },
  
  "healthcheck": {
    "url": "/health",
    "interval": 30,
    "timeout": 5,
    "retries": 3
  },
  
  "env": {
    "APP_ENV": "production",
    "LOG_LEVEL": "info"
  },
  
  "resources": {
    "memory_limit": "1G",
    "cpu_limit": "1.0"
  },
  
  "backups": {
    "enabled": true,
    "schedule": "0 2 * * *",
    "database": {
      "connection": "DB_CONNECTION",
      "host": "DB_HOST",
      "port": "DB_PORT",
      "username": "DB_USERNAME",
      "password": "DB_PASSWORD",
      "database_name": "DB_DATABASE"
    },
    "directories": [
      "storage/app",
      "public/uploads"
    ]
  },
  
  "nix": {
    "flake_file": "./custom-flake.nix"
  }
}
```

#### Benefits of Template-Based Nix Deployment

| Feature | Traditional | VPS Pilot (Nix Templates) |
|---------|------------|----------|
| **User Complexity** | ❌ Learn Docker/Nix | ✅ Just set versions in JSON |
| **Dependency Conflicts** | ❌ Frequent | ✅ Impossible |
| **Version Management** | ❌ Manual | ✅ Automatic |
| **Reproducibility** | ❌ "Works on my machine" | ✅ Bit-for-bit identical |
| **Rollback** | ❌ Complex | ✅ Instant |
| **Setup Time** | ❌ Hours | ✅ Minutes |
| **Learning Curve** | ❌ Steep | ✅ Heroku-like simplicity |

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
| **Project Isolation** | Nix Flakes    |

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
- **Nix** (for nodes running projects) ([install](https://nixos.org/download.html))

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

## 🖥️ Node Setup (For Project Deployment)

### Installing Nix on Nodes

On each node that will run projects, install Nix with flakes support:

```bash
# Install Nix (multi-user installation)
sh <(curl -L https://nixos.org/nix/install) --daemon

# Enable flakes and nix-command
mkdir -p ~/.config/nix
echo "experimental-features = nix-command flakes" >> ~/.config/nix/nix.conf

# Verify installation
nix --version
```

### Agent Installation on Nodes

1. **Install the VPS Pilot Agent** ([Agent Repository](https://github.com/sanda0/vps_pilot_agent))
   ```bash
   # Download and install agent
   wget https://github.com/sanda0/vps_pilot_agent/releases/latest/download/vps_pilot_agent
   chmod +x vps_pilot_agent
   sudo mv vps_pilot_agent /usr/local/bin/
   ```

2. **Configure the Agent**
   ```bash
   # Create config file
   sudo mkdir -p /etc/vpspilot
   sudo nano /etc/vpspilot/config.json
   ```
   
   ```json
   {
     "server": "your-vpspilot-server.com:55001",
     "node_name": "production-node-1",
     "project_dir": "/opt/vpspilot/projects",
     "use_nix": true
   }
   ```

3. **Create Systemd Service**
   ```bash
   sudo nano /etc/systemd/system/vpspilot-agent.service
   ```
   
   ```ini
   [Unit]
   Description=VPS Pilot Agent
   After=network.target
   
   [Service]
   Type=simple
   User=vpspilot
   ExecStart=/usr/local/bin/vps_pilot_agent
   Restart=always
   RestartSec=10
   
   [Install]
   WantedBy=multi-user.target
   ```

4. **Start the Agent**
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable vpspilot-agent
   sudo systemctl start vpspilot-agent
   sudo systemctl status vpspilot-agent
   ```

### Nix Garbage Collection (Optional)

To prevent disk space issues, set up automatic Nix garbage collection:

```bash
# Clean old generations older than 30 days
sudo nix-collect-garbage --delete-older-than 30d

# Add to crontab for weekly cleanup
(crontab -l 2>/dev/null; echo "0 3 * * 0 nix-collect-garbage --delete-older-than 30d") | crontab -
```

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
- [ ] **Nix-based project deployment** 🔥
  - [ ] Nix flake detection and validation
  - [ ] Isolated environment per project
  - [ ] Multi-version runtime support
  - [ ] Blue-green deployments
  - [ ] Instant rollbacks
- [ ] Project management via `config.vpspilot.json`
- [ ] Remote command execution for projects
- [ ] Project backups (database + directories)
- [ ] GitHub integration for auto-deployment
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
- **Nix Templates (Heroku-like!)**: [docs/NIX_TEMPLATES.md](docs/NIX_TEMPLATES.md)
- **Nix Deployment Guide**: [docs/NIX_DEPLOYMENT.md](docs/NIX_DEPLOYMENT.md)
- **Nix Quick Reference**: [docs/NIX_QUICK_REFERENCE.md](docs/NIX_QUICK_REFERENCE.md)

---

**⭐ Star this repo if you find it useful!**

