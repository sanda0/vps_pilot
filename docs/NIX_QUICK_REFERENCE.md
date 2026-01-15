# Nix Quick Reference for VPS Pilot

Quick reference guide for developers deploying projects with VPS Pilot's Nix-based system.

---

## 📋 Quick Commands

### Essential Nix Commands

```bash
# Build your project
nix build

# Run your project
nix run

# Enter development environment
nix develop

# Run a command in dev environment
nix develop --command npm install

# Update dependencies
nix flake update

# Check what will be built
nix flake show

# Clean old builds
nix-collect-garbage -d
```

### Project Setup

```bash
# Initialize a new Nix flake
nix flake init

# Initialize with template
nix flake init -t templates#nodejs

# Validate flake.nix
nix flake check

# Format flake.nix
nix fmt
```

---

## 🚀 Common Use Cases

### Case 1: Simple Node.js App

**Project Structure:**
```
myapp/
├── flake.nix
├── package.json
├── src/
│   └── index.js
└── config.vpspilot.json
```

**Minimal flake.nix:**
```nix
{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  
  outputs = { nixpkgs, ... }: {
    packages.x86_64-linux.default = 
      let pkgs = nixpkgs.legacyPackages.x86_64-linux;
      in pkgs.buildNpmPackage {
        pname = "myapp";
        version = "1.0.0";
        src = ./.;
        npmDepsHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
      };
  };
}
```

**Getting the correct npmDepsHash:**
```bash
# 1. Build with wrong hash (will fail)
nix build

# 2. Copy the correct hash from error message:
# "got: sha256-xyz123..."
# 
# 3. Update flake.nix with correct hash
```

### Case 2: Python Web App

**flake.nix:**
```nix
{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  
  outputs = { nixpkgs, ... }:
    let
      pkgs = nixpkgs.legacyPackages.x86_64-linux;
      python = pkgs.python311;
    in {
      packages.x86_64-linux.default = python.pkgs.buildPythonApplication {
        pname = "myapp";
        version = "1.0.0";
        src = ./.;
        
        propagatedBuildInputs = with python.pkgs; [
          flask
          sqlalchemy
          requests
        ];
      };
      
      devShells.x86_64-linux.default = pkgs.mkShell {
        buildInputs = [ python python.pkgs.pip ];
        shellHook = "pip install -r requirements.txt";
      };
    };
}
```

### Case 3: Go API

**flake.nix:**
```nix
{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  
  outputs = { nixpkgs, ... }:
    let pkgs = nixpkgs.legacyPackages.x86_64-linux;
    in {
      packages.x86_64-linux.default = pkgs.buildGoModule {
        pname = "myapi";
        version = "1.0.0";
        src = ./.;
        vendorHash = null;  # or specific hash
      };
    };
}
```

### Case 4: Multi-Language Stack

**flake.nix:**
```nix
{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  
  outputs = { nixpkgs, ... }:
    let pkgs = nixpkgs.legacyPackages.x86_64-linux;
    in {
      devShells.x86_64-linux.default = pkgs.mkShell {
        buildInputs = with pkgs; [
          nodejs_20      # Frontend
          go_1_21        # Backend
          postgresql_15  # Database
          redis          # Cache
        ];
        
        shellHook = ''
          echo "Full-stack environment ready!"
          export DATABASE_URL="postgresql://localhost/mydb"
        '';
      };
    };
}
```

---

## 🔧 config.vpspilot.json Templates

### Minimal Configuration

```json
{
  "name": "My App",
  "tech": ["nodejs"],
  "deployment": {
    "port": 3000
  }
}
```

### Complete Configuration

```json
{
  "name": "Production App",
  "description": "Production-ready application",
  "tech": ["nodejs", "postgresql", "redis"],
  
  "nix": {
    "flake": "./flake.nix",
    "package": "default",
    "shell": "default"
  },
  
  "deployment": {
    "build_command": "nix build",
    "start_command": "nix run .#default",
    "stop_command": null,
    "restart_command": null,
    "healthcheck_url": "http://localhost:3000/health",
    "healthcheck_interval": 30,
    "healthcheck_timeout": 5,
    "healthcheck_retries": 3,
    "port": 3000,
    "environment": "production"
  },
  
  "resources": {
    "memory_limit": "1G",
    "cpu_limit": "1.0",
    "disk_limit": "10G"
  },
  
  "logs": [
    "/var/log/myapp/app.log",
    "/var/log/myapp/error.log"
  ],
  
  "commands": [
    {
      "name": "build",
      "command": "nix develop --command npm run build",
      "description": "Build the application"
    },
    {
      "name": "test",
      "command": "nix develop --command npm test",
      "description": "Run tests"
    },
    {
      "name": "migrate",
      "command": "nix develop --command npm run migrate",
      "description": "Run database migrations"
    },
    {
      "name": "seed",
      "command": "nix develop --command npm run seed",
      "description": "Seed database"
    }
  ],
  
  "backups": {
    "enabled": true,
    "schedule": "0 2 * * *",
    "retention_days": 30,
    "env_file": ".env",
    "zip_file_name": "backup",
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
      "public/uploads",
      "data"
    ]
  },
  
  "monitoring": {
    "enabled": true,
    "metrics_endpoint": "/metrics",
    "alert_on_failure": true
  }
}
```

---

## 📦 Runtime Version Cheat Sheet

### Node.js Versions

```nix
nodejs_16     # Node.js 16.x
nodejs_18     # Node.js 18.x LTS
nodejs_20     # Node.js 20.x LTS
nodejs_21     # Node.js 21.x
nodejs        # Latest stable (alias)
```

### PHP Versions

```nix
php74         # PHP 7.4
php80         # PHP 8.0
php81         # PHP 8.1
php82         # PHP 8.2
php83         # PHP 8.3
php           # Latest stable (alias)
```

### Python Versions

```nix
python39      # Python 3.9
python310     # Python 3.10
python311     # Python 3.11
python312     # Python 3.12
python3       # Latest stable (alias)
```

### Go Versions

```nix
go_1_19       # Go 1.19
go_1_20       # Go 1.20
go_1_21       # Go 1.21
go_1_22       # Go 1.22
go            # Latest stable (alias)
```

### Database Versions

```nix
postgresql_12   # PostgreSQL 12
postgresql_13   # PostgreSQL 13
postgresql_14   # PostgreSQL 14
postgresql_15   # PostgreSQL 15
postgresql_16   # PostgreSQL 16

mysql80         # MySQL 8.0
mysql84         # MySQL 8.4

mariadb         # Latest MariaDB
mariadb_106     # MariaDB 10.6
mariadb_1011    # MariaDB 10.11

redis           # Latest Redis
mongodb         # Latest MongoDB
```

---

## 🐛 Troubleshooting Cheat Sheet

### Build Errors

| Error | Solution |
|-------|----------|
| `hash mismatch` | Copy correct hash from error message |
| `attribute 'buildNpmPackage' missing` | Update nixpkgs: `nix flake update` |
| `command not found` | Add missing package to `buildInputs` |
| `network access blocked` | Use `--impure` for development |
| `infinite recursion` | Check for circular dependencies in flake |

### Runtime Errors

| Error | Solution |
|-------|----------|
| Port already in use | Check with `lsof -i :PORT`, kill process |
| Permission denied | Check file/directory permissions |
| Module not found | Run `npm install` in `shellHook` |
| Database connection failed | Ensure DB is running, check credentials |
| Nix store full | Run `nix-collect-garbage -d` |

### Commands to Check Issues

```bash
# Check if Nix is working
nix --version

# Verify flake.nix syntax
nix flake check

# See what will be built
nix build --dry-run

# Build with verbose output
nix build --show-trace --verbose

# Check system resources
df -h /nix/store
du -sh /nix/store

# View recent builds
nix profile history

# Check service logs
journalctl -u vpspilot-project-* -f

# Verify port is open
netstat -tulpn | grep :3000
```

---

## 🔄 Deployment Workflows

### Development Workflow

```bash
# 1. Clone your repo
git clone https://github.com/user/myapp.git
cd myapp

# 2. Create flake.nix
cat > flake.nix << 'EOF'
{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  outputs = { nixpkgs, ... }:
    let pkgs = nixpkgs.legacyPackages.x86_64-linux;
    in {
      devShells.x86_64-linux.default = pkgs.mkShell {
        buildInputs = [ pkgs.nodejs_20 ];
      };
    };
}
EOF

# 3. Enter dev environment
nix develop

# 4. Work on your code
npm install
npm run dev

# 5. Test build
nix build

# 6. Commit
git add flake.nix flake.lock
git commit -m "Add Nix support"
git push
```

### Production Deployment via VPS Pilot

```bash
# 1. Push code to GitHub
git push origin main

# 2. In VPS Pilot Dashboard:
#    - Go to Projects
#    - Click "New Project"
#    - Select Node
#    - Enter GitHub URL
#    - Click "Deploy"

# 3. Agent automatically:
#    - Clones repo
#    - Detects flake.nix
#    - Builds with Nix
#    - Starts service
#    - Reports status

# 4. Monitor deployment
#    - View logs in dashboard
#    - Check metrics
#    - Verify health checks
```

### Update Workflow

```bash
# 1. Make changes locally
git checkout -b feature/new-thing
# ... make changes ...
git commit -m "Add new feature"

# 2. Test with Nix
nix build
nix run .#default

# 3. Push changes
git push origin feature/new-thing

# 4. In VPS Pilot Dashboard:
#    - Go to Project
#    - Click "Update"
#    - Select branch/commit
#    - Choose deployment strategy:
#      - Rolling update (default)
#      - Blue-green (zero downtime)
#      - Canary (gradual rollout)

# 5. Agent performs update
#    - Pulls new code
#    - Builds new version
#    - Runs tests
#    - Switches to new version
#    - Keeps old version for rollback
```

---

## 💡 Pro Tips

### Tip 1: Speed Up Builds with Binary Cache

```nix
{
  nixConfig = {
    extra-substituters = [
      "https://cache.nixos.org"
      "https://nix-community.cachix.org"
    ];
    extra-trusted-public-keys = [
      "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="
      "nix-community.cachix.org-1:mB9FSh9qf2dCimDSUo8Zy7bkq5CX+/rkCWyvRCYg3Fs="
    ];
  };
}
```

### Tip 2: Pin Dependencies for Reproducibility

```nix
{
  inputs = {
    # Pin to specific commit
    nixpkgs.url = "github:NixOS/nixpkgs?rev=abc123def456...";
    
    # Or pin to specific release
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-23.11";
  };
}
```

### Tip 3: Share Development Environment

```bash
# Team members just run:
nix develop

# Everyone gets identical environment!
# No more "install Node 20.10 not 20.11"
```

### Tip 4: Use flake templates

```bash
# List available templates
nix flake show templates

# Initialize from template
nix flake init -t templates#nodejs
nix flake init -t templates#python
nix flake init -t templates#go
```

### Tip 5: Overlay for Custom Packages

```nix
{
  outputs = { nixpkgs, ... }:
    let
      overlay = final: prev: {
        myCustomPackage = prev.buildNpmPackage {
          # custom package definition
        };
      };
      
      pkgs = import nixpkgs {
        system = "x86_64-linux";
        overlays = [ overlay ];
      };
    in {
      packages.x86_64-linux.default = pkgs.myCustomPackage;
    };
}
```

### Tip 6: Multiple Outputs

```nix
{
  outputs = { nixpkgs, ... }:
    let pkgs = nixpkgs.legacyPackages.x86_64-linux;
    in {
      packages.x86_64-linux = {
        # Main application
        default = pkgs.buildNpmPackage { /* ... */ };
        
        # CLI tool
        cli = pkgs.buildNpmPackage { /* ... */ };
        
        # Documentation
        docs = pkgs.stdenv.mkDerivation { /* ... */ };
      };
    };
}

# Build specific output:
# nix build .#cli
# nix build .#docs
```

---

## 📚 Learn More

- **Nix Pills Tutorial**: https://nixos.org/guides/nix-pills/
- **Nix Language**: https://nixos.org/manual/nix/stable/language/
- **Search Packages**: https://search.nixos.org/packages
- **Nix Flakes Book**: https://nixos-and-flakes.thiscute.world/
- **VPS Pilot Docs**: [/docs/NIX_DEPLOYMENT.md](NIX_DEPLOYMENT.md)

---

## 🤝 Getting Help

1. **Check the logs**: `journalctl -u vpspilot-project-* -f`
2. **Verify flake**: `nix flake check`
3. **Build locally**: `nix build --show-trace`
4. **VPS Pilot Dashboard**: View deployment logs and metrics
5. **Community**: https://discourse.nixos.org/

---

**Quick Links:**
- [Full Deployment Guide](NIX_DEPLOYMENT.md)
- [VPS Pilot Main Repo](https://github.com/sanda0/vps_pilot)
- [Agent Repo](https://github.com/sanda0/vps_pilot_agent)
