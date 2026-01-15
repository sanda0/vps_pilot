# VPS Pilot Documentation

Welcome to VPS Pilot documentation! This directory contains comprehensive guides for building, deploying, and using VPS Pilot.

---

## 📚 Documentation Index

### Getting Started
- **[README](../README.md)** - Project overview, features, and quick start
- **[QUICKSTART](../QUICKSTART.md)** - Fast setup guide to get running quickly

### Building & Deployment
- **[BUILDING](BUILDING.md)** - How to build VPS Pilot with embedded UI
- **[NIX_DEPLOYMENT](NIX_DEPLOYMENT.md)** - Complete guide to Nix-based project deployment
- **[NIX_TEMPLATES](NIX_TEMPLATES.md)** - Built-in Nix templates (no Nix knowledge required!)
- **[NIX_QUICK_REFERENCE](NIX_QUICK_REFERENCE.md)** - Quick reference for Nix commands and patterns

### Integration
- **[GITHUB_INTEGRATION](GITHUB_INTEGRATION.md)** - GitHub webhooks and auto-deployment
- **[EMBEDDED_UI_SUMMARY](EMBEDDED_UI_SUMMARY.md)** - Details on embedded UI architecture

### Other Resources
- **[readme_draft](readme_draft.md)** - Draft documentation and notes

---

## 🚀 Quick Links

### For Users
1. **New to VPS Pilot?** Start with [README](../README.md)
2. **Want to deploy quickly?** See [QUICKSTART](../QUICKSTART.md)
3. **Deploying a project?** Check [NIX_TEMPLATES](NIX_TEMPLATES.md) - no Nix knowledge needed!
4. **Setting up nodes?** Check [Node Setup](../README.md#-node-setup-for-project-deployment) in README

### For Developers
1. **Deploying projects?** Read [NIX_TEMPLATES](NIX_TEMPLATES.md) for Heroku-like simplicity
2. **Need architecture details?** See [NIX_DEPLOYMENT](NIX_DEPLOYMENT.md)
3. **Need quick Nix commands?** Use [NIX_QUICK_REFERENCE](NIX_QUICK_REFERENCE.md)
4. **Building from source?** Follow [BUILDING](BUILDING.md)
5. **GitHub integration?** See [GITHUB_INTEGRATION](GITHUB_INTEGRATION.md)

---

## 📖 Documentation Overview

### NIX_TEMPLATES.md
**Built-in Nix templates for zero-config deployment**

Topics covered:
- Complete template library (Laravel, Node.js, Go, Python, etc.)
- Template variable injection
- Template selection logic
- User experience walkthrough (no Nix knowledge needed!)
- Custom template override for advanced users

**Best for**: Understanding how VPS Pilot makes deployment as simple as Heroku

### NIX_DEPLOYMENT.md
**Complete Nix deployment architecture guide**

Topics covered:
- Why Nix for project deployment
- Architecture overview
- Folder structure on nodes
- Complete command flow
- Real-world flake.nix examples (Node.js, PHP, Python, Go)
- Deployment strategies (Blue-Green, Rolling, Canary)
- Rollback procedures
- Best practices
- Troubleshooting

**Best for**: Understanding the entire Nix-based deployment system

### NIX_QUICK_REFERENCE.md
**Quick reference and cheat sheet**

Topics covered:
- Essential Nix commands
- Common use cases with examples
- config.vpspilot.json templates
- Runtime version cheat sheet
- Troubleshooting cheat sheet
- Deployment workflows
- Pro tips

**Best for**: Quick lookups and copy-paste examples

### BUILDING.md
**Building VPS Pilot from source**

Topics covered:
- Frontend build process
- Go backend compilation
- Embedding UI in binary
- Development vs production builds
- Build scripts

**Best for**: Contributing to VPS Pilot or custom builds

### GITHUB_INTEGRATION.md
**GitHub webhooks and auto-deployment**

Topics covered:
- Setting up GitHub webhooks
- Automatic deployments on push
- Branch-based deployments
- Security considerations

**Best for**: Automating deployments from GitHub

---

## 🎯 Common Tasks

### Deploy a New Project

1. Create `config.vpspilot.json` in your repository:
   ```json
   {
     "name": "My App",
     "type": "laravel",
     "runtime": {
       "php": "8.2",
       "node": "20"
     },
     "build": {
       "commands": ["composer install", "npm run build"]
     },
     "start": {
       "command": "php artisan serve --host=0.0.0.0 --port=$PORT"
     }
   }
   ```

2. Push to GitHub

3. Deploy via VPS Pilot dashboard

**That's it! No Nix code needed!**

**See**: [NIX_TEMPLATES](NIX_TEMPLATES.md)

### Update a Running Project

1. Push changes to GitHub
2. In VPS Pilot dashboard: Projects → Select Project → Update
3. Choose deployment strategy
4. Monitor deployment logs

**See**: [NIX_DEPLOYMENT](NIX_DEPLOYMENT.md#deployment-strategies)

### Rollback a Deployment

**Instant rollback:**
```bash
nix profile rollback --profile /nix/var/nix/profiles/project-123
systemctl restart vpspilot-project-123
```

**See**: [NIX_DEPLOYMENT](NIX_DEPLOYMENT.md#rollback-procedures)

### Run Multiple Projects with Different Runtime Versions

Each project's `flake.nix` specifies its own runtime versions:

```
Node A:
├── project-1 → Node.js 18 + PHP 7.4
├── project-2 → Node.js 20 + PHP 8.2
└── project-3 → Go 1.21 + Python 3.11

No conflicts! ✅
```

**See**: [README - Multi-Version Support](../README.md#multi-version-support-example)

---

## 🏗️ Architecture Diagrams

### System Overview
```
┌─────────────────────────────────────┐
│       VPS Pilot Server              │
│   ┌──────────────────────────┐     │
│   │   Web Dashboard (React)   │     │
│   └──────────────────────────┘     │
│   ┌──────────────────────────┐     │
│   │   REST API (Go)          │     │
│   └──────────────────────────┘     │
│   ┌──────────────────────────┐     │
│   │   TCP Server (Metrics)   │     │
│   └──────────────────────────┘     │
│   ┌──────────────────────────┐     │
│   │   SQLite Databases       │     │
│   └──────────────────────────┘     │
└─────────┬───────────────────────────┘
          │ TCP/WebSocket
          ▼
┌─────────────────────────────────────┐
│         Node Agents                 │
│   ┌──────────────────────────┐     │
│   │   Metrics Collector      │     │
│   │   Project Manager (Nix)  │     │
│   │   Systemd Integration    │     │
│   └──────────────────────────┘     │
│                                     │
│   /opt/vpspilot/projects/          │
│   ├── project-1/ (Node 20)         │
│   ├── project-2/ (PHP 8.2)         │
│   └── project-3/ (Go 1.21)         │
└─────────────────────────────────────┘
```

### Nix Deployment Flow
```
GitHub Push
    │
    ▼
┌─────────────────┐
│ VPS Pilot       │
│ Webhook Handler │
└────────┬────────┘
         │ Deployment Command
         ▼
┌─────────────────┐
│ Node Agent      │
└────────┬────────┘
         │
         ├─► Clone Repository
         ├─► Validate flake.nix
         ├─► nix build
         ├─► nix run (or systemd service)
         ├─► Health Check
         └─► Report Status
```

---

## 🔍 Troubleshooting Guide

### Build Issues
- **Hash mismatch**: [NIX_QUICK_REFERENCE](NIX_QUICK_REFERENCE.md#build-errors)
- **Missing dependencies**: [NIX_DEPLOYMENT](NIX_DEPLOYMENT.md#troubleshooting)
- **Build script fails**: [BUILDING](BUILDING.md)

### Deployment Issues
- **Service won't start**: [NIX_DEPLOYMENT](NIX_DEPLOYMENT.md#service-wont-start)
- **Port conflicts**: [NIX_QUICK_REFERENCE](NIX_QUICK_REFERENCE.md#runtime-errors)
- **Permission errors**: [README - Security Notes](../README.md#-security-notes)

### Agent Issues
- **Agent disconnected**: Check systemd service status
- **Metrics not showing**: Verify TCP port 55001 is open
- **Project not detected**: Ensure `config.vpspilot.json` exists

---

## 🤝 Contributing

Want to improve VPS Pilot documentation?

1. Fork the repository
2. Edit documentation in `docs/`
3. Follow markdown best practices
4. Submit a pull request

**Documentation standards:**
- Use clear, concise language
- Include code examples
- Add diagrams where helpful
- Keep formatting consistent

---

## 📞 Getting Help

- **Issues**: https://github.com/sanda0/vps_pilot/issues
- **Discussions**: https://github.com/sanda0/vps_pilot/discussions
- **Nix Community**: https://discourse.nixos.org/

---

**Last Updated**: January 2026
