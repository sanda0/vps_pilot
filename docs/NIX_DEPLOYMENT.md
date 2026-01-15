# Nix-Based Deployment Architecture

This document describes VPS Pilot's **template-based Nix deployment system** for running multiple projects on the same server without conflicts.

## 🎯 Key Concept: Heroku-Like Simplicity

**Users don't need to know Nix!** Just configure `config.vpspilot.json` with project type and versions, and VPS Pilot handles the rest.

```json
{
  "name": "My App",
  "type": "laravel",
  "runtime": { "php": "8.2", "node": "20" },
  "build": { "commands": ["composer install", "npm run build"] },
  "start": { "command": "php artisan serve" }
}
```

VPS Pilot automatically:
1. Selects the appropriate Nix template
2. Injects your specified versions
3. Generates `flake.nix` dynamically
4. Builds isolated environment
5. Deploys your app

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Built-in Templates](#built-in-templates)
3. [Architecture](#architecture)
4. [Folder Structure](#folder-structure)
5. [Command Flow](#command-flow)
6. [Flake Examples](#flake-examples)
7. [Deployment Strategies](#deployment-strategies)
8. [Rollback Procedures](#rollback-procedures)
9. [Best Practices](#best-practices)
10. [Troubleshooting](#troubleshooting)

---

## Overview

### Why Nix with Templates?

Traditional deployment approaches face several challenges:
- **Version conflicts**: Multiple projects need different Node/PHP/Python versions
- **Complexity**: Users need to learn Docker/Nix/Kubernetes
- **Dependency hell**: System-wide packages conflict between projects
- **Reproducibility**: "Works on my machine" syndrome

VPS Pilot's template-based Nix approach solves these:
- **Simplicity**: Users just set versions in JSON (like Heroku)
- **Isolation**: Each project has its own complete dependency tree
- **No Conflicts**: Multiple PHP/Node/Go versions coexist
- **Reproducibility**: Builds are deterministic and reproducible
- **Atomicity**: Upgrades are atomic; system is never in broken state
- **Rollback**: Instant rollback to any previous generation
- **No Learning Curve**: If you can configure JSON, you can deploy

---

## Built-in Templates

VPS Pilot includes pre-built Nix templates for common project types. **Users never need to write Nix code!**

### Available Templates

| Project Type | Template | Configuration Example |
|-------------|----------|----------------------|
| **Laravel** | `laravel.nix.tmpl` | `{ "type": "laravel", "runtime": { "php": "8.2", "node": "20" } }` |
| **Node.js** | `nodejs.nix.tmpl` | `{ "type": "nodejs", "runtime": { "node": "20" } }` |
| **Go** | `go.nix.tmpl` | `{ "type": "go", "runtime": { "go": "1.21" } }` |
| **Python** | `python.nix.tmpl` | `{ "type": "python", "runtime": { "python": "3.11" } }` |
| **Django** | `django.nix.tmpl` | `{ "type": "django", "runtime": { "python": "3.11" } }` |
| **Static** | `static.nix.tmpl` | `{ "type": "static", "runtime": { "node": "20" } }` |
| **Ruby** | `ruby.nix.tmpl` | `{ "type": "ruby", "runtime": { "ruby": "3.2" } }` |
| **Custom** | User's `flake.nix` | `{ "type": "custom", "nix": { "flake_file": "./flake.nix" } }` |

### How Templates Work

1. **User specifies project type** in `config.vpspilot.json`
2. **Agent selects matching template** (e.g., `laravel.nix.tmpl`)
3. **Agent injects variables** from config (PHP version, Node version, commands)
4. **Agent generates `flake.nix`** dynamically in project directory
5. **Nix builds** using the generated flake
6. **App deploys** with isolated dependencies

### Example: Simple Laravel Setup

**User's `config.vpspilot.json`:**
```json
{
  "name": "My Laravel App",
  "type": "laravel",
  "runtime": {
    "php": "8.2",
    "node": "20"
  },
  "build": {
    "commands": [
      "composer install --no-dev",
      "npm install",
      "npm run build"
    ]
  },
  "start": {
    "command": "php artisan serve --host=0.0.0.0 --port=$PORT"
  }
}
```

That's it! **User never writes Nix code.** The agent handles everything automatically.

For complete template reference and examples, see [NIX_TEMPLATES.md](NIX_TEMPLATES.md).

---

### Key Concepts

```
┌─────────────────────────────────────────────────────┐
│                   VPS Pilot Server                   │
│  ┌────────────────────────────────────────────┐    │
│  │         Deployment Manager                  │    │
│  │  • Receives GitHub webhooks                │    │
│  │  • Sends deploy commands to agents         │    │
│  │  • Tracks deployment status                │    │
│  └──────────────┬──────────────────────────────┘    │
└─────────────────┼────────────────────────────────────┘
                  │ TCP/gRPC
                  ▼
┌─────────────────────────────────────────────────────┐
│                    Node Agents                       │
│  ┌────────────────────────────────────────────┐    │
│  │          Project Orchestrator               │    │
│  │  • Clones repositories                      │    │
│  │  • Validates flake.nix                      │    │
│  │  • Builds with Nix                          │    │
│  │  • Manages systemd services                 │    │
│  └────────────────────────────────────────────┘    │
│                                                      │
│  /opt/vpspilot/projects/                            │
│  ├── project-1/  (Node.js 20, PHP 8.2)             │
│  ├── project-2/  (Node.js 18, Python 3.11)         │
│  └── project-3/  (Go 1.21, Ruby 3.2)               │
└─────────────────────────────────────────────────────┘
```

---

## Architecture

### Component Overview

```
┌───────────────────────────────────────────────────────────┐
│                     Nix Store                              │
│             /nix/store/                                    │
│                                                            │
│  ┌──────────────────┐  ┌──────────────────┐             │
│  │  nodejs-20.10.0  │  │  nodejs-18.19.0  │  ← Shared   │
│  └──────────────────┘  └──────────────────┘             │
│  ┌──────────────────┐  ┌──────────────────┐             │
│  │    php-8.2.14    │  │    php-7.4.33    │             │
│  └──────────────────┘  └──────────────────┘             │
│                                                            │
└───────────────┬───────────────────────────────────────────┘
                │
                │ Symlinks to specific versions
                │
    ┌───────────┴───────────────────┐
    │                               │
    ▼                               ▼
┌─────────────────┐         ┌─────────────────┐
│   Project A     │         │   Project B     │
│  Node.js 20     │         │  Node.js 18     │
│  PHP 8.2        │         │  PHP 7.4        │
└─────────────────┘         └─────────────────┘
```

### Process Flow

1. **Deployment Request**: Central server sends deployment command
2. **Repository Clone**: Agent clones specified Git repository
3. **Environment Setup**: Nix reads `flake.nix` and builds environment
4. **Dependency Resolution**: Nix downloads and builds all dependencies
5. **Build**: Project is built within Nix environment
6. **Environment Variables**: Agent creates `.env` file from dashboard secrets
7. **Service Creation**: Systemd service is created/updated
8. **Health Check**: Application is verified to be running
9. **Status Report**: Agent reports back to central server

---

## Environment Variables & Secrets

### Placeholder System

VPS Pilot uses placeholders in `config.vpspilot.json` for sensitive values:

```json
{
  "env": {
    "APP_ENV": "production",
    "DB_PASSWORD": "{{DB_PASSWORD}}",
    "API_KEY": "{{API_KEY}}"
  }
}
```

### Dashboard Management

1. **User sets actual values** in web dashboard
2. **Values encrypted** in operational database (AES-256)
3. **Central server sends** encrypted values to agent during deployment
4. **Agent replaces placeholders** and creates `.env` file

### Security Flow

```
┌─────────────────────────────────────────────────────┐
│           VPS Pilot Dashboard (Web UI)              │
│                                                      │
│  User Sets Environment Variables:                   │
│  ┌────────────────────────────────────────────┐    │
│  │ DB_PASSWORD = super_secret_123 🔒         │    │
│  │ API_KEY     = sk_live_abc123... 🔒        │    │
│  └────────────────────────────────────────────┘    │
│                      │                              │
│                      │ Encrypt with AES-256         │
│                      ▼                              │
│  ┌────────────────────────────────────────────┐    │
│  │      Operational Database (Encrypted)      │    │
│  │                                            │    │
│  │  project_env_vars:                         │    │
│  │  | key         | value (encrypted)      |  │    │
│  │  | DB_PASSWORD | U2FsdGVkX1... (cipher)|  │    │
│  │  | API_KEY     | U2FsdGVkX1... (cipher)|  │    │
│  └────────────────────────────────────────────┘    │
└──────────────────────┬──────────────────────────────┘
                       │
                       │ Deployment Command
                       │ (Decrypted + TLS)
                       ▼
┌─────────────────────────────────────────────────────┐
│                  Node Agent                         │
│                                                      │
│  Receives:                                          │
│  {                                                  │
│    "config": {                                      │
│      "env": {                                       │
│        "DB_PASSWORD": "{{DB_PASSWORD}}"             │
│      }                                              │
│    },                                               │
│    "env_vars": {                                    │
│      "DB_PASSWORD": "super_secret_123"  ← Actual    │
│    }                                                │
│  }                                                  │
│                      │                              │
│                      ▼                              │
│  ┌────────────────────────────────────────────┐    │
│  │  Create .env file:                         │    │
│  │                                            │    │
│  │  DB_PASSWORD=super_secret_123              │    │
│  │  API_KEY=sk_live_abc123...                 │    │
│  │                                            │    │
│  │  Permissions: 600 (owner read/write only)  │    │
│  └────────────────────────────────────────────┘    │
│                      │                              │
│                      ▼                              │
│  ┌────────────────────────────────────────────┐    │
│  │  Nix environment loads .env                │    │
│  │  Application starts with secrets           │    │
│  └────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘
```

### Benefits

- ✅ **Git-safe**: Placeholders in repo, secrets in database
- ✅ **Encrypted**: AES-256 at rest, TLS in transit
- ✅ **Centralized**: Manage all secrets from one place
- ✅ **Per-node**: Different secrets for dev/staging/prod
- ✅ **Audit trail**: Track all changes to env vars
- ✅ **Access control**: Only authorized users can view/edit

For complete details, see [NIX_TEMPLATES.md - Environment Variables](NIX_TEMPLATES.md#environment-variables--secrets-management).

---

## Folder Structure

### On Each Node

```
/opt/vpspilot/
├── projects/
│   ├── project-123-myapp/
│   │   ├── flake.nix                    # Nix environment definition
│   │   ├── flake.lock                   # Locked dependency versions
│   │   ├── config.vpspilot.json         # VPS Pilot configuration
│   │   ├── .env                         # Environment variables
│   │   ├── .git/                        # Git repository
│   │   ├── .nix-profile/                # Nix profile (symlinks)
│   │   └── src/                         # Application source
│   │
│   ├── project-456-api/
│   │   ├── flake.nix
│   │   ├── flake.lock
│   │   ├── config.vpspilot.json
│   │   └── ...
│   │
│   └── current/                         # Symlinks for blue-green
│       ├── project-123 -> project-123-myapp-v5/
│       └── project-456 -> project-456-api-v3/
│
├── backups/                             # Project backups
│   ├── project-123-2026-01-15.tar.gz
│   └── project-456-2026-01-14.tar.gz
│
├── logs/                                # Centralized logs
│   ├── project-123.log
│   └── project-456.log
│
└── cache/                               # Build caches
    └── nix/
```

### Systemd Service Files

```
/etc/systemd/system/
├── vpspilot-agent.service               # Main agent
├── vpspilot-project-123.service         # Project services
├── vpspilot-project-456.service
└── vpspilot-project-789.service
```

---

## Command Flow

### Deployment Sequence

```bash
# 1. Agent receives deployment command from server
{
  "action": "deploy",
  "project_id": 123,
  "github_url": "https://github.com/user/repo.git",
  "branch": "main",
  "commit": "abc1234",
  "env_vars": {
    "DATABASE_URL": "postgres://...",
    "API_KEY": "secret"
  }
}

# 2. Create project directory
mkdir -p /opt/vpspilot/projects/project-123-myapp
cd /opt/vpspilot/projects/project-123-myapp

# 3. Clone repository
git clone --depth 1 --branch main https://github.com/user/repo.git .
git checkout abc1234

# 4. Validate flake.nix exists
if [ ! -f flake.nix ]; then
  echo "ERROR: No flake.nix found"
  exit 1
fi

# 5. Create .env file from env_vars
cat > .env << EOF
DATABASE_URL=postgres://...
API_KEY=secret
EOF

# 6. Build with Nix
nix build .#default \
  --log-format bar-with-logs \
  --show-trace \
  2>&1 | tee build.log

# 7. Run tests (if configured)
if grep -q "test" config.vpspilot.json; then
  nix develop --command npm test
fi

# 8. Create/update systemd service
cat > /etc/systemd/system/vpspilot-project-123.service << EOF
[Unit]
Description=VPS Pilot Project 123
After=network.target

[Service]
Type=simple
User=vpspilot
WorkingDirectory=/opt/vpspilot/projects/project-123-myapp
ExecStart=/bin/sh -c 'nix run .#default'
Restart=always
RestartSec=10
StandardOutput=append:/opt/vpspilot/logs/project-123.log
StandardError=append:/opt/vpspilot/logs/project-123.log

[Install]
WantedBy=multi-user.target
EOF

# 9. Reload systemd and start service
systemctl daemon-reload
systemctl enable vpspilot-project-123
systemctl restart vpspilot-project-123

# 10. Wait for health check
for i in {1..30}; do
  if curl -f http://localhost:3000/health; then
    echo "Health check passed"
    break
  fi
  sleep 2
done

# 11. Report success to central server
curl -X POST http://central-server:8000/api/projects/123/status \
  -H "Content-Type: application/json" \
  -d '{
    "status": "running",
    "version": "abc1234",
    "pid": 12345,
    "port": 3000
  }'
```

### Update/Upgrade Flow

```bash
# 1. Backup current state
tar -czf /opt/vpspilot/backups/project-123-$(date +%Y-%m-%d).tar.gz \
  /opt/vpspilot/projects/project-123-myapp

# 2. Pull latest changes
cd /opt/vpspilot/projects/project-123-myapp
git fetch origin
git checkout $NEW_COMMIT

# 3. Update flake.lock (optional, for dependency updates)
nix flake update

# 4. Build new version
nix build .#default

# 5. Test new version (in isolated environment)
nix develop --command npm test

# 6. If tests pass, restart service
systemctl restart vpspilot-project-123

# 7. Monitor for 60 seconds
sleep 60
if ! systemctl is-active vpspilot-project-123; then
  # Rollback on failure
  git checkout $OLD_COMMIT
  nix build .#default
  systemctl restart vpspilot-project-123
fi
```

---

## Flake Examples

### Node.js + TypeScript + PostgreSQL

```nix
{
  description = "Node.js API with TypeScript";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        # Development environment
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            nodejs_20
            nodePackages.typescript
            nodePackages.npm
            postgresql_15
          ];
          
          shellHook = ''
            echo "Node.js $(node --version)"
            echo "PostgreSQL $(postgres --version)"
            export DATABASE_URL="postgresql://localhost/mydb"
            npm install
          '';
        };

        # Production build
        packages.default = pkgs.buildNpmPackage {
          pname = "myapi";
          version = "1.0.0";
          src = ./.;
          
          npmDepsHash = "sha256-xyz...";  # Run with wrong hash to get real hash
          
          buildPhase = ''
            npm run build
          '';
          
          installPhase = ''
            mkdir -p $out/bin
            mkdir -p $out/lib
            cp -r dist/* $out/lib/
            cp package.json $out/lib/
            
            cat > $out/bin/myapi << EOF
            #!/usr/bin/env bash
            cd $out/lib
            ${pkgs.nodejs_20}/bin/node index.js
            EOF
            chmod +x $out/bin/myapi
          '';
        };

        # Run command
        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/myapi";
        };
      }
    );
}
```

### Python + Django + Redis

```nix
{
  description = "Django Application";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      python = pkgs.python311;
      pythonPackages = python.pkgs;
    in {
      devShells.default = pkgs.mkShell {
        buildInputs = [
          python
          pythonPackages.django
          pythonPackages.psycopg2
          pythonPackages.redis
          pythonPackages.celery
          pkgs.postgresql
          pkgs.redis
        ];
        
        shellHook = ''
          export DJANGO_SETTINGS_MODULE=myapp.settings
          python manage.py migrate
          echo "Django development environment ready"
        '';
      };

      packages.default = pythonPackages.buildPythonApplication {
        pname = "django-app";
        version = "1.0.0";
        src = ./.;
        
        propagatedBuildInputs = [
          pythonPackages.django
          pythonPackages.psycopg2
          pythonPackages.redis
          pythonPackages.celery
          pythonPackages.gunicorn
        ];
        
        installPhase = ''
          mkdir -p $out/lib/python
          cp -r * $out/lib/python/
          
          mkdir -p $out/bin
          cat > $out/bin/django-app << EOF
          #!/usr/bin/env bash
          cd $out/lib/python
          ${pythonPackages.gunicorn}/bin/gunicorn myapp.wsgi:application \
            --bind 0.0.0.0:8000 \
            --workers 4
          EOF
          chmod +x $out/bin/django-app
        '';
      };
    };
}
```

### PHP + Laravel + MySQL

```nix
{
  description = "Laravel Application";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      php = pkgs.php82.buildEnv {
        extensions = { enabled, all }: enabled ++ (with all; [
          mysqli
          pdo
          pdo_mysql
          mbstring
          xml
          curl
          zip
          gd
        ]);
        extraConfig = ''
          memory_limit = 256M
          upload_max_filesize = 50M
          post_max_size = 50M
        '';
      };
    in {
      devShells.default = pkgs.mkShell {
        buildInputs = [
          php
          pkgs.php82Packages.composer
          pkgs.nodejs_18
          pkgs.mysql80
        ];
        
        shellHook = ''
          export PATH="$PWD/vendor/bin:$PATH"
          composer install
          php artisan --version
          npm install
        '';
      };

      packages.default = pkgs.stdenv.mkDerivation {
        pname = "laravel-app";
        version = "1.0.0";
        src = ./.;
        
        buildInputs = [ php pkgs.nodejs_18 ];
        
        buildPhase = ''
          composer install --no-dev --optimize-autoloader
          npm run build
        '';
        
        installPhase = ''
          mkdir -p $out
          cp -r * $out/
          
          mkdir -p $out/bin
          cat > $out/bin/laravel-serve << EOF
          #!/usr/bin/env bash
          cd $out
          ${php}/bin/php artisan serve --host=0.0.0.0 --port=8000
          EOF
          chmod +x $out/bin/laravel-serve
        '';
      };
    };
}
```

### Go Microservice

```nix
{
  description = "Go Microservice";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
    in {
      packages.default = pkgs.buildGoModule {
        pname = "myservice";
        version = "1.0.0";
        src = ./.;
        
        # To get the correct vendorHash:
        # 1. Set to empty string or wrong hash
        # 2. Run nix build
        # 3. Copy the hash from error message
        vendorHash = "sha256-abc123...";
        
        # Build flags
        ldflags = [
          "-s" "-w"
          "-X main.Version=1.0.0"
          "-X main.BuildTime=${builtins.currentTime}"
        ];
        
        # Optional: run tests during build
        checkPhase = ''
          go test ./...
        '';
      };

      devShells.default = pkgs.mkShell {
        buildInputs = with pkgs; [
          go_1_21
          gopls
          go-tools
          delve
        ];
        
        shellHook = ''
          echo "Go $(go version)"
        '';
      };

      apps.default = {
        type = "app";
        program = "${self.packages.${system}.default}/bin/myservice";
      };
    };
}
```

### Full-Stack: Next.js + Go API + PostgreSQL

```nix
{
  description = "Full-Stack Application";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        packages = {
          # Frontend
          frontend = pkgs.buildNpmPackage {
            pname = "frontend";
            version = "1.0.0";
            src = ./frontend;
            npmDepsHash = "sha256-...";
            
            buildPhase = ''
              npm run build
            '';
            
            installPhase = ''
              mkdir -p $out
              cp -r .next/* $out/
              cp -r public $out/
              cp package.json $out/
              
              mkdir -p $out/bin
              cat > $out/bin/frontend << EOF
              #!/usr/bin/env bash
              cd $out
              ${pkgs.nodejs_20}/bin/node server.js
              EOF
              chmod +x $out/bin/frontend
            '';
          };

          # Backend API
          backend = pkgs.buildGoModule {
            pname = "backend";
            version = "1.0.0";
            src = ./backend;
            vendorHash = "sha256-...";
          };

          # Combined package
          default = pkgs.writeShellScriptBin "fullstack-app" ''
            # Start database
            ${pkgs.postgresql}/bin/postgres -D /var/lib/postgresql/data &
            POSTGRES_PID=$!
            
            # Start backend
            ${self.packages.${system}.backend}/bin/backend &
            BACKEND_PID=$!
            
            # Start frontend
            ${self.packages.${system}.frontend}/bin/frontend &
            FRONTEND_PID=$!
            
            # Trap exit signals
            trap "kill $POSTGRES_PID $BACKEND_PID $FRONTEND_PID" EXIT
            
            # Wait for any process to exit
            wait -n
          '';
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            nodejs_20
            go_1_21
            postgresql_15
            postgresql15Packages.pgvector
          ];
          
          shellHook = ''
            echo "Development environment ready"
            export DATABASE_URL="postgresql://localhost/mydb"
            export API_URL="http://localhost:8080"
          '';
        };
      }
    );
}
```

---

## Deployment Strategies

### Strategy 1: Blue-Green Deployment

Recommended for production. Zero-downtime deployments.

```bash
#!/bin/bash
# deploy-blue-green.sh

PROJECT_ID=$1
NEW_VERSION=$2
PROJECT_DIR="/opt/vpspilot/projects"
CURRENT_DIR="$PROJECT_DIR/project-$PROJECT_ID-current"
BLUE_DIR="$PROJECT_DIR/project-$PROJECT_ID-blue"
GREEN_DIR="$PROJECT_DIR/project-$PROJECT_ID-green"

# Determine which environment is currently active
if [ -L "$CURRENT_DIR" ]; then
  ACTIVE=$(readlink "$CURRENT_DIR")
  if [[ "$ACTIVE" == *"blue"* ]]; then
    DEPLOY_TO="green"
    DEPLOY_DIR="$GREEN_DIR"
  else
    DEPLOY_TO="blue"
    DEPLOY_DIR="$BLUE_DIR"
  fi
else
  DEPLOY_TO="blue"
  DEPLOY_DIR="$BLUE_DIR"
fi

echo "Deploying to: $DEPLOY_TO"

# 1. Clone to inactive environment
rm -rf "$DEPLOY_DIR"
git clone $REPO_URL "$DEPLOY_DIR"
cd "$DEPLOY_DIR"
git checkout "$NEW_VERSION"

# 2. Build with Nix
nix build .#default

# 3. Run tests
if ! nix develop --command npm test; then
  echo "Tests failed, aborting deployment"
  exit 1
fi

# 4. Start new version
nix run .#default &
NEW_PID=$!

# 5. Health check
for i in {1..30}; do
  if curl -f http://localhost:$PORT/health; then
    echo "New version healthy"
    break
  fi
  sleep 2
done

# 6. Switch traffic (atomic symlink swap)
ln -sfn "$DEPLOY_DIR" "$CURRENT_DIR.tmp"
mv -Tf "$CURRENT_DIR.tmp" "$CURRENT_DIR"

# 7. Stop old version
if [ ! -z "$OLD_PID" ]; then
  kill "$OLD_PID"
fi

# 8. Clean old environment after 5 minutes (grace period)
(sleep 300 && rm -rf "$OLD_DIR") &

echo "Deployment complete: $DEPLOY_TO is now active"
```

### Strategy 2: Rolling Update

For projects that can handle gradual updates.

```bash
#!/bin/bash
# deploy-rolling.sh

PROJECT_DIR="/opt/vpspilot/projects/project-$PROJECT_ID"

cd "$PROJECT_DIR"

# 1. Backup current state
cp flake.lock "flake.lock.backup-$(date +%s)"

# 2. Pull changes
git pull origin main

# 3. Update dependencies
nix flake update

# 4. Build new version (doesn't affect running version)
if ! nix build .#default; then
  echo "Build failed, rolling back"
  git reset --hard HEAD@{1}
  exit 1
fi

# 5. Run tests
if ! nix develop --command npm test; then
  echo "Tests failed, rolling back"
  git reset --hard HEAD@{1}
  exit 1
fi

# 6. Graceful restart
systemctl reload vpspilot-project-$PROJECT_ID || \
systemctl restart vpspilot-project-$PROJECT_ID

# 7. Verify new version
sleep 5
if ! systemctl is-active vpspilot-project-$PROJECT_ID; then
  echo "Service failed to start, rolling back"
  git reset --hard HEAD@{1}
  nix build .#default
  systemctl restart vpspilot-project-$PROJECT_ID
  exit 1
fi

echo "Rolling update complete"
```

### Strategy 3: Canary Deployment

Deploy to subset of instances first.

```bash
#!/bin/bash
# deploy-canary.sh

CANARY_PERCENT=10

# 1. Deploy to canary instances (10% of traffic)
for instance in $CANARY_INSTANCES; do
  ssh "$instance" "cd /opt/vpspilot/projects/project-$PROJECT_ID && \
    git pull && \
    nix build && \
    systemctl restart vpspilot-project-$PROJECT_ID"
done

# 2. Monitor canary metrics for 10 minutes
echo "Monitoring canary deployment..."
sleep 600

# 3. Check error rates
ERROR_RATE=$(curl -s "http://metrics/api/error-rate?project=$PROJECT_ID")

if [ "$ERROR_RATE" -lt 1 ]; then
  echo "Canary successful, deploying to all instances"
  
  # 4. Deploy to remaining instances
  for instance in $REMAINING_INSTANCES; do
    ssh "$instance" "cd /opt/vpspilot/projects/project-$PROJECT_ID && \
      git pull && \
      nix build && \
      systemctl restart vpspilot-project-$PROJECT_ID"
  done
else
  echo "Canary failed, rolling back"
  
  # Rollback canary instances
  for instance in $CANARY_INSTANCES; do
    ssh "$instance" "cd /opt/vpspilot/projects/project-$PROJECT_ID && \
      git reset --hard HEAD@{1} && \
      nix build && \
      systemctl restart vpspilot-project-$PROJECT_ID"
  done
fi
```

---

## Rollback Procedures

### Instant Rollback (Using Nix Generations)

```bash
# List all generations
nix profile history --profile /nix/var/nix/profiles/project-123

# Output:
# Version 1: 2026-01-10 → Node.js 18.0.0
# Version 2: 2026-01-12 → Node.js 20.0.0  ← Current
# Version 3: 2026-01-15 → Node.js 20.1.0

# Rollback to previous generation
nix profile rollback --profile /nix/var/nix/profiles/project-123

# Rollback to specific generation
nix profile rollback --to 1 --profile /nix/var/nix/profiles/project-123

# Restart service
systemctl restart vpspilot-project-123
```

### Git-Based Rollback

```bash
#!/bin/bash
# rollback.sh

PROJECT_ID=$1
TARGET_COMMIT=$2
PROJECT_DIR="/opt/vpspilot/projects/project-$PROJECT_ID"

cd "$PROJECT_DIR"

# 1. Stop current version
systemctl stop vpspilot-project-$PROJECT_ID

# 2. Checkout target commit
git checkout "$TARGET_COMMIT"

# 3. Rebuild with Nix
nix build .#default

# 4. Start service
systemctl start vpspilot-project-$PROJECT_ID

# 5. Verify
sleep 5
if systemctl is-active vpspilot-project-$PROJECT_ID; then
  echo "Rollback successful to commit $TARGET_COMMIT"
else
  echo "Rollback failed!"
  exit 1
fi
```

### Blue-Green Rollback

```bash
#!/bin/bash
# rollback-blue-green.sh

PROJECT_ID=$1
CURRENT_DIR="/opt/vpspilot/projects/project-$PROJECT_ID-current"

# Get current active environment
ACTIVE=$(readlink "$CURRENT_DIR")

# Determine previous environment
if [[ "$ACTIVE" == *"blue"* ]]; then
  PREVIOUS="/opt/vpspilot/projects/project-$PROJECT_ID-green"
else
  PREVIOUS="/opt/vpspilot/projects/project-$PROJECT_ID-blue"
fi

# Atomic switch back
ln -sfn "$PREVIOUS" "$CURRENT_DIR.tmp"
mv -Tf "$CURRENT_DIR.tmp" "$CURRENT_DIR"

# Restart service (systemd follows symlink)
systemctl restart vpspilot-project-$PROJECT_ID

echo "Rolled back to: $PREVIOUS"
```

### Emergency Rollback Script

```bash
#!/bin/bash
# emergency-rollback.sh

PROJECT_ID=$1

echo "EMERGENCY ROLLBACK for project $PROJECT_ID"

# 1. Stop service immediately
systemctl stop vpspilot-project-$PROJECT_ID

# 2. Restore from latest backup
LATEST_BACKUP=$(ls -t /opt/vpspilot/backups/project-$PROJECT_ID-*.tar.gz | head -1)
echo "Restoring from: $LATEST_BACKUP"

tar -xzf "$LATEST_BACKUP" -C /opt/vpspilot/projects/

# 3. Start service
systemctl start vpspilot-project-$PROJECT_ID

# 4. Verify
if systemctl is-active vpspilot-project-$PROJECT_ID; then
  echo "Emergency rollback successful"
  
  # Send alert to admins
  curl -X POST http://vpspilot-server/api/alerts \
    -d "Emergency rollback performed on project $PROJECT_ID"
else
  echo "CRITICAL: Emergency rollback failed!"
  exit 1
fi
```

---

## Best Practices

### 1. Flake.nix Organization

```nix
{
  description = "Project Name";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    # Pin to specific commit for production
    # nixpkgs.url = "github:NixOS/nixpkgs?rev=abc123...";
  };

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      
      # Version management
      version = "1.0.0";
      gitRevision = self.rev or "dirty";
      
      # Shared configuration
      commonBuildInputs = with pkgs; [
        nodejs_20
        postgresql
      ];
      
    in {
      # Development shell
      devShells.default = pkgs.mkShell {
        buildInputs = commonBuildInputs ++ [ pkgs.nodePackages.typescript ];
      };
      
      # Production package
      packages.default = pkgs.buildNpmPackage {
        pname = "myapp";
        inherit version;
        src = ./.;
        # ... rest of config
      };
      
      # Tests
      checks.default = pkgs.runCommand "test-myapp" {
        buildInputs = commonBuildInputs;
      } ''
        cd ${self}
        npm test
        touch $out
      '';
    };
}
```

### 2. Dependency Pinning

```bash
# Always commit flake.lock
git add flake.lock
git commit -m "Lock dependencies"

# Update specific input
nix flake update nixpkgs

# Update all inputs
nix flake update

# Check what will be updated
nix flake update --dry-run
```

### 3. Build Caching

```nix
# Use binary cache for faster builds
{
  outputs = { self, nixpkgs }:
    let
      pkgs = import nixpkgs {
        system = "x86_64-linux";
        config = {
          # Enable unfree packages if needed
          allowUnfree = true;
        };
      };
    in {
      # Configure binary caches
      nixConfig = {
        extra-substituters = [
          "https://cache.nixos.org"
          "https://your-company-cache.s3.amazonaws.com"
        ];
        extra-trusted-public-keys = [
          "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="
          "your-key:..."
        ];
      };
      
      packages.default = # ...
    };
}
```

### 4. Environment Variable Management

```bash
# Never commit .env files
echo ".env" >> .gitignore

# Use .env.example as template
cat > .env.example << EOF
DATABASE_URL=postgresql://user:pass@localhost/db
API_KEY=your-api-key-here
PORT=3000
EOF

# Agent injects actual secrets during deployment
# from VPS Pilot's encrypted secret storage
```

### 5. Health Checks

```json
{
  "deployment": {
    "healthcheck_url": "http://localhost:3000/health",
    "healthcheck_interval": 10,
    "healthcheck_timeout": 5,
    "healthcheck_retries": 3
  }
}
```

Implement health endpoint in your app:

```javascript
// Node.js example
app.get('/health', (req, res) => {
  // Check database connection
  db.ping()
    .then(() => res.json({ status: 'healthy' }))
    .catch(err => res.status(500).json({ status: 'unhealthy', error: err.message }));
});
```

### 6. Logging

```nix
# Configure logging in systemd service
StandardOutput=append:/opt/vpspilot/logs/project-${PROJECT_ID}.log
StandardError=append:/opt/vpspilot/logs/project-${PROJECT_ID}.error.log
```

### 7. Resource Limits

```ini
# /etc/systemd/system/vpspilot-project-123.service
[Service]
# Memory limit
MemoryMax=1G
MemoryHigh=800M

# CPU limit (50% of one core)
CPUQuota=50%

# File descriptor limit
LimitNOFILE=4096

# Restart policy
Restart=always
RestartSec=10
StartLimitBurst=3
StartLimitIntervalSec=60
```

---

## Troubleshooting

### Build Failures

```bash
# Check build log
nix build .#default --show-trace 2>&1 | tee build.log

# Common issues:

# 1. Wrong hash
# Error: hash mismatch in fixed-output derivation
# Solution: Copy correct hash from error message

# 2. Missing dependencies
# Error: command not found: gcc
# Solution: Add to buildInputs

# 3. Network issues during build
# Error: unable to download
# Solution: Use --impure flag for development
nix build .#default --impure
```

### Runtime Errors

```bash
# Check service status
systemctl status vpspilot-project-123

# View logs
journalctl -u vpspilot-project-123 -f

# Check if port is in use
lsof -i :3000

# Verify Nix environment
nix develop --command env
```

### Disk Space Issues

```bash
# Nix store can grow large
du -sh /nix/store

# Clean old generations
nix-collect-garbage --delete-older-than 30d

# Clean specific project
nix profile wipe-history --older-than 30d \
  --profile /nix/var/nix/profiles/project-123

# Optimize store (deduplicate)
nix-store --optimise
```

### Slow Builds

```bash
# Enable parallel builds
nix build --cores 4 --max-jobs 4

# Use binary cache
nix build --option binary-caches "https://cache.nixos.org"

# Check what's being built
nix build --dry-run --json | jq
```

### Dependency Conflicts

```bash
# This shouldn't happen with Nix, but if it does:

# Check which version is actually used
nix-store --query --tree result

# Lock to specific version
nix flake update --update-input nixpkgs \
  --override-input nixpkgs github:NixOS/nixpkgs/rev-abc123
```

### Service Won't Start

```bash
# 1. Check systemd service file
systemctl cat vpspilot-project-123

# 2. Verify executable exists
ls -la /nix/store/*/bin/myapp

# 3. Check permissions
stat /opt/vpspilot/projects/project-123-myapp

# 4. Run manually for debugging
cd /opt/vpspilot/projects/project-123-myapp
nix run .#default

# 5. Check environment variables
systemctl show vpspilot-project-123 --property=Environment
```

---

## Additional Resources

- **Nix Manual**: https://nixos.org/manual/nix/stable/
- **Nix Flakes**: https://nixos.wiki/wiki/Flakes
- **NixOS Packages**: https://search.nixos.org/packages
- **Nix Pills**: https://nixos.org/guides/nix-pills/
- **VPS Pilot Agent**: https://github.com/sanda0/vps_pilot_agent

---

**Last Updated**: January 2026
