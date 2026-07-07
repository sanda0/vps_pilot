# VPS Pilot — Quick Start Guide

Get up and running in minutes.

---

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| **Go** | 1.21+ | https://go.dev/doc/install |
| **Node.js** | 18+ | https://nodejs.org/ |
| **Bun** *(optional, faster)* | latest | https://bun.sh/ |
| **Git** | any | https://git-scm.com/ |

---

## Step 1 — Clone the repository

```sh
git clone https://github.com/sanda0/vps_pilot.git
cd vps_pilot
```

---

## Step 2 — Build the binary

The `build.sh` script does everything in one shot:
- Installs frontend dependencies (if needed)
- Builds the React UI
- Embeds the UI into the Go binary

```sh
chmod +x build.sh
./build.sh
```

When it finishes you will see:

```
🎉 Build completed successfully!
📁 Binary location: server/vps_pilot
```

---

## Step 3 — Configure environment

```sh
cd server
cp .env.example .env
```

Open `.env` and fill in your values:

```env
# Path where SQLite databases will be stored
DB_PATH=./data

# JWT authentication
TOKEN_LIFESPAN=60
TOKEN_SECRET=change-me-to-a-random-string-32-chars-min

# Port the agent TCP server listens on
TCP_SERVER_PORT=55001

# Email alerts (optional)
MAIL_HOST=smtp.gmail.com
MAIL_PORT=587
MAIL_USERNAME=your-email@gmail.com
MAIL_PASSWORD=your-app-password
MAIL_FROM_ADDRESS=noreply@vpspilot.com
```

> **Important:** `TOKEN_SECRET` must be at least 32 characters long.

---

## Step 4 — Run migrations

Migrations run automatically when the server starts, but you can also run them manually:

```sh
cd server
./vps_pilot -migrate
```

---

## Step 5 — Create your admin account

```sh
cd server
./vps_pilot -create-superuser
```

Follow the interactive prompts to set your username and password.

---

## Step 6 — Start the server

```sh
cd server
./vps_pilot
```

The server starts on port **8080** by default.  
To use a different port, pass the `-port` flag:

```sh
./vps_pilot -port 9090
```

---

## Step 7 — Open the dashboard

Visit **http://localhost:8080** in your browser and log in with the credentials you created in Step 5.

---

## CLI Flags Reference

| Flag | Description |
|------|-------------|
| `-port <port>` | HTTP server port (default: `8080`) |
| `-migrate` | Run database migrations and exit |
| `-create-superuser` | Create an admin user interactively and exit |
| `-create-makefile` | Generate a `Makefile` in `server/` and exit |

---

## Development Mode (hot reload)

Run the backend and frontend separately so changes reflect instantly.

**Terminal 1 — Backend**
```sh
cd server
go run main.go
```

Or with [air](https://github.com/air-verse/air) for automatic reload on file changes:
```sh
cd server
air
```

**Terminal 2 — Frontend**
```sh
cd client
npm install   # or: bun install
npm run dev   # or: bun run dev
```

| Service | URL |
|---------|-----|
| Frontend (Vite) | http://localhost:5173 |
| Backend API | http://localhost:8080 |

---

## Troubleshooting

**Build fails**
```sh
# Test that the frontend builds on its own
cd client && npm run build

# Test that the Go code compiles on its own
cd server && go build .
```

**Server won't start**
```sh
# Check if port 8080 is already taken
lsof -i :8080

# Make sure the .env file exists
ls -la server/.env

# Check data directory permissions
ls -la server/data/
```

**UI shows a blank page or 404 after build**
```sh
# Check the dist folder was copied into the server
ls server/cmd/app/dist/

# If it is missing, run the build again
./build.sh
```

**Metrics not appearing in the dashboard**
- Make sure the [VPS Pilot Agent](https://github.com/sanda0/vps_pilot_agent) is installed and running on the target node
- Confirm TCP port `55001` is open on the server (check your firewall rules)
- Verify the node is registered in the dashboard

---

## What's Next?

- Install the agent on your nodes → [Agent Repository](https://github.com/sanda0/vps_pilot_agent)
- Read the full build guide → [docs/BUILDING.md](docs/BUILDING.md)
- Configure alerts (Email / Slack / Discord) → see the **Alerts** section in the dashboard