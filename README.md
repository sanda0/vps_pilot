# VPS Pilot

VPS Pilot is a **server monitoring and management platform** designed for private VPS servers.  
It provides real-time monitoring, alerting, project management, and (future) cron job automation — all from a single dashboard.

![Screenshot from 2025-04-29 08-13-30](https://github.com/user-attachments/assets/fff1c368-9c8e-4bb6-9720-f9a7f46a2910)



## ✨ Features

### 📊 Monitoring
- Agents installed on each node (server).
- Agents send system metrics to the central server:
  - **CPU usage**
  - **Memory usage**
  - **Network statistics**
- Metrics are visualized in the dashboard with selectable time ranges:
  - 5 minutes, 15 minutes, 1 hour, 1 day, 2 days, 7 days.
---

### 🚨 Alerting
- Users can configure alerts based on metric thresholds.
- Notifications sent via:
  - **Discord** (✅ Implemented)
  - **Email** (🔜 Not yet implemented)
  - **Slack** (🔜 Not yet implemented)

---

### 🚀 Projects Management (TODO)
- Each node can have multiple projects.
- Projects require a `config.vpspilot.json` file.
- Agents scan disks for project config files and send project metadata to the central server.
- Central server will display available projects and allow:
  - Running predefined commands.
  - Managing project logs.
  - Backing up project directories and databases.

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
> **Note:** This feature is still under development.

---

### ⏲️ Cron Jobs Management (TODO)
- Plan to allow users to create and manage cron jobs on nodes remotely from the dashboard.
- Feature is not implemented yet.

---

## 🛠️ Tech Stack

| Component        | Technology |
|------------------|------------|
| Agent            | Golang     |
| Central Server   | Golang     |
| Dashboard        | React.js   |
| Database         | TimescaleDB |

---

## 📦 Installation

(Instructions will be added soon. Likely via Docker Compose or manual Go/React build.)

---

## 📅 Roadmap

- [x] Real-time metrics collection (CPU, Memory, Network)
- [x] Discord alert integration
- [ ] Email alert integration
- [ ] Slack alert integration
- [ ] Project management via `config.vpspilot.json`
- [ ] Remote command execution for projects
- [ ] Project backups (database + directories)
- [ ] Remote cron job creation and management

---

## 🧑‍💻 Author

Made with ❤️ by [Sandakelum](https://github.com/sanda0)

---

## 📜 License

This project is licensed under the [MIT License](LICENSE).

