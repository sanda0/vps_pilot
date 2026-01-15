# VPS Pilot Nix Templates

Built-in Nix templates for easy deployment without writing Nix code.

## Overview

VPS Pilot provides pre-built Nix templates for common project types. Users simply specify their project type and runtime versions in `config.vpspilot.json`, and the agent automatically generates the appropriate Nix environment.

**User Experience: Just like Heroku!**
- No Nix knowledge required
- No `flake.nix` to write
- Just specify versions and go!

---

## Template Variables

All templates support these variables that are injected from `config.vpspilot.json`:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{PHP_VERSION}}` | PHP version | `82` (for 8.2) |
| `{{NODE_VERSION}}` | Node.js version | `20` |
| `{{GO_VERSION}}` | Go version | `1_21` |
| `{{PYTHON_VERSION}}` | Python version | `311` (for 3.11) |
| `{{RUBY_VERSION}}` | Ruby version | `32` (for 3.2) |
| `{{BUILD_COMMANDS}}` | Build commands | From `build.commands` |
| `{{START_COMMAND}}` | Start command | From `start.command` |
| `{{PROJECT_NAME}}` | Project name | From `name` |

---

## Laravel Template

**File:** `templates/laravel.nix.tmpl`

```nix
{
  description = "{{PROJECT_NAME}} - Laravel Application";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      
      # Version mapping from config
      php = pkgs.php{{PHP_VERSION}}.buildEnv {
        extensions = { enabled, all }: enabled ++ (with all; [
          mysqli
          pdo
          pdo_mysql
          mbstring
          xml
          curl
          zip
          gd
          redis
          bcmath
          intl
        ]);
        extraConfig = ''
          memory_limit = 256M
          upload_max_filesize = 50M
          post_max_size = 50M
        '';
      };
      
      nodejs = pkgs.nodejs_{{NODE_VERSION}};
      
    in {
      devShells.default = pkgs.mkShell {
        buildInputs = [
          php
          pkgs.php{{PHP_VERSION}}Packages.composer
          nodejs
          pkgs.git
        ];
        
        shellHook = ''
          export PATH="$PWD/vendor/bin:$PATH"
          echo "Laravel Environment Ready!"
          echo "PHP: $(php --version | head -n 1)"
          echo "Node: $(node --version)"
          echo "Composer: $(composer --version)"
        '';
      };

      packages.default = pkgs.stdenv.mkDerivation {
        pname = "{{PROJECT_NAME}}";
        version = "1.0.0";
        src = ./.;
        
        buildInputs = [ php nodejs ];
        
        buildPhase = ''
          {{BUILD_COMMANDS}}
        '';
        
        installPhase = ''
          mkdir -p $out
          cp -r * $out/
          
          # Create start script
          mkdir -p $out/bin
          cat > $out/bin/start << 'EOF'
          #!/usr/bin/env bash
          cd $out
          ${php}/bin/php artisan config:cache
          ${php}/bin/php artisan route:cache
          ${php}/bin/php artisan view:cache
          {{START_COMMAND}}
          EOF
          chmod +x $out/bin/start
        '';
      };
    };
}
```

---

## Node.js Template

**File:** `templates/nodejs.nix.tmpl`

```nix
{
  description = "{{PROJECT_NAME}} - Node.js Application";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        nodejs = pkgs.nodejs_{{NODE_VERSION}};
      in {
        devShells.default = pkgs.mkShell {
          buildInputs = [
            nodejs
            pkgs.nodePackages.npm
            pkgs.nodePackages.typescript
            pkgs.git
          ];
          
          shellHook = ''
            echo "Node.js Environment Ready!"
            echo "Node: $(node --version)"
            echo "NPM: $(npm --version)"
          '';
        };

        packages.default = pkgs.buildNpmPackage {
          pname = "{{PROJECT_NAME}}";
          version = "1.0.0";
          src = ./.;
          
          # This will be computed on first build
          npmDepsHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
          
          buildPhase = ''
            {{BUILD_COMMANDS}}
          '';
          
          installPhase = ''
            mkdir -p $out/bin
            mkdir -p $out/lib
            
            # Copy built files
            if [ -d "dist" ]; then
              cp -r dist/* $out/lib/
            elif [ -d "build" ]; then
              cp -r build/* $out/lib/
            else
              cp -r * $out/lib/
            fi
            
            cp package.json $out/lib/ 2>/dev/null || true
            
            # Create start script
            cat > $out/bin/start << 'EOF'
            #!/usr/bin/env bash
            cd $out/lib
            export NODE_ENV=production
            {{START_COMMAND}}
            EOF
            chmod +x $out/bin/start
          '';
        };
      }
    );
}
```

---

## Go Template

**File:** `templates/go.nix.tmpl`

```nix
{
  description = "{{PROJECT_NAME}} - Go Application";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      go = pkgs.go_{{GO_VERSION}};
    in {
      devShells.default = pkgs.mkShell {
        buildInputs = [
          go
          pkgs.gopls
          pkgs.go-tools
          pkgs.git
        ];
        
        shellHook = ''
          echo "Go Environment Ready!"
          echo "Go: $(go version)"
        '';
      };

      packages.default = pkgs.buildGoModule {
        pname = "{{PROJECT_NAME}}";
        version = "1.0.0";
        src = ./.;
        
        # Set to null for projects using vendor directory
        # Otherwise will be computed on first build
        vendorHash = null;
        
        buildPhase = ''
          {{BUILD_COMMANDS}}
        '';
        
        installPhase = ''
          mkdir -p $out/bin
          
          # Find and copy the built binary
          if [ -f "./{{PROJECT_NAME}}" ]; then
            cp "./{{PROJECT_NAME}}" $out/bin/app
          elif [ -f "./app" ]; then
            cp "./app" $out/bin/app
          elif [ -f "./main" ]; then
            cp "./main" $out/bin/app
          else
            # Find any binary in current directory
            binary=$(find . -maxdepth 1 -type f -executable | head -n 1)
            if [ -n "$binary" ]; then
              cp "$binary" $out/bin/app
            else
              echo "Error: No binary found"
              exit 1
            fi
          fi
          
          chmod +x $out/bin/app
        '';
      };
    };
}
```

---

## Python Template

**File:** `templates/python.nix.tmpl`

```nix
{
  description = "{{PROJECT_NAME}} - Python Application";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      python = pkgs.python{{PYTHON_VERSION}};
      pythonPackages = python.pkgs;
    in {
      devShells.default = pkgs.mkShell {
        buildInputs = [
          python
          pythonPackages.pip
          pythonPackages.virtualenv
          pkgs.git
        ];
        
        shellHook = ''
          echo "Python Environment Ready!"
          echo "Python: $(python --version)"
          
          # Create virtual environment if needed
          if [ ! -d ".venv" ]; then
            python -m venv .venv
          fi
          source .venv/bin/activate
          
          # Install dependencies
          if [ -f "requirements.txt" ]; then
            pip install -r requirements.txt
          fi
        '';
      };

      packages.default = pkgs.stdenv.mkDerivation {
        pname = "{{PROJECT_NAME}}";
        version = "1.0.0";
        src = ./.;
        
        buildInputs = [ python pythonPackages.pip ];
        
        buildPhase = ''
          {{BUILD_COMMANDS}}
        '';
        
        installPhase = ''
          mkdir -p $out/lib/python
          cp -r * $out/lib/python/
          
          mkdir -p $out/bin
          cat > $out/bin/start << 'EOF'
          #!/usr/bin/env bash
          cd $out/lib/python
          export PYTHONPATH=$out/lib/python:$PYTHONPATH
          {{START_COMMAND}}
          EOF
          chmod +x $out/bin/start
        '';
      };
    };
}
```

---

## Django Template

**File:** `templates/django.nix.tmpl`

```nix
{
  description = "{{PROJECT_NAME}} - Django Application";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      python = pkgs.python{{PYTHON_VERSION}};
      pythonPackages = python.pkgs;
    in {
      devShells.default = pkgs.mkShell {
        buildInputs = [
          python
          pythonPackages.django
          pythonPackages.gunicorn
          pythonPackages.psycopg2
          pythonPackages.redis
          pythonPackages.celery
          pkgs.git
        ];
        
        shellHook = ''
          echo "Django Environment Ready!"
          echo "Python: $(python --version)"
          echo "Django: $(python -c 'import django; print(django.get_version())')"
          
          export DJANGO_SETTINGS_MODULE={{PROJECT_NAME}}.settings
        '';
      };

      packages.default = pythonPackages.buildPythonApplication {
        pname = "{{PROJECT_NAME}}";
        version = "1.0.0";
        src = ./.;
        
        propagatedBuildInputs = [
          pythonPackages.django
          pythonPackages.gunicorn
          pythonPackages.psycopg2
          pythonPackages.redis
          pythonPackages.celery
        ];
        
        buildPhase = ''
          {{BUILD_COMMANDS}}
        '';
        
        installPhase = ''
          mkdir -p $out/lib/python
          cp -r * $out/lib/python/
          
          mkdir -p $out/bin
          cat > $out/bin/start << 'EOF'
          #!/usr/bin/env bash
          cd $out/lib/python
          export DJANGO_SETTINGS_MODULE={{PROJECT_NAME}}.settings
          python manage.py migrate --noinput
          {{START_COMMAND}}
          EOF
          chmod +x $out/bin/start
        '';
      };
    };
}
```

---

## Static Site Template

**File:** `templates/static.nix.tmpl`

```nix
{
  description = "{{PROJECT_NAME}} - Static Site";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      nodejs = pkgs.nodejs_{{NODE_VERSION}};
    in {
      devShells.default = pkgs.mkShell {
        buildInputs = [
          nodejs
          pkgs.nodePackages.npm
          pkgs.git
        ];
        
        shellHook = ''
          echo "Static Site Environment Ready!"
          echo "Node: $(node --version)"
        '';
      };

      packages.default = pkgs.stdenv.mkDerivation {
        pname = "{{PROJECT_NAME}}";
        version = "1.0.0";
        src = ./.;
        
        buildInputs = [ nodejs ];
        
        buildPhase = ''
          {{BUILD_COMMANDS}}
        '';
        
        installPhase = ''
          mkdir -p $out/public
          
          # Copy built files from common output directories
          if [ -d "dist" ]; then
            cp -r dist/* $out/public/
          elif [ -d "build" ]; then
            cp -r build/* $out/public/
          elif [ -d "public" ]; then
            cp -r public/* $out/public/
          elif [ -d "_site" ]; then
            cp -r _site/* $out/public/
          elif [ -d "out" ]; then
            cp -r out/* $out/public/
          else
            echo "Error: No build output directory found"
            exit 1
          fi
          
          # Create start script with simple HTTP server
          mkdir -p $out/bin
          cat > $out/bin/start << 'EOF'
          #!/usr/bin/env bash
          cd $out/public
          ${nodejs}/bin/npx serve -s . -p ${PORT:-3000}
          EOF
          chmod +x $out/bin/start
        '';
      };
    };
}
```

---

## Ruby Template

**File:** `templates/ruby.nix.tmpl`

```nix
{
  description = "{{PROJECT_NAME}} - Ruby Application";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      ruby = pkgs.ruby_{{RUBY_VERSION}};
    in {
      devShells.default = pkgs.mkShell {
        buildInputs = [
          ruby
          pkgs.bundler
          pkgs.git
        ];
        
        shellHook = ''
          echo "Ruby Environment Ready!"
          echo "Ruby: $(ruby --version)"
          echo "Bundler: $(bundle --version)"
          
          # Install gems
          bundle config set --local path 'vendor/bundle'
          bundle install
        '';
      };

      packages.default = pkgs.stdenv.mkDerivation {
        pname = "{{PROJECT_NAME}}";
        version = "1.0.0";
        src = ./.;
        
        buildInputs = [ ruby pkgs.bundler ];
        
        buildPhase = ''
          export GEM_HOME=$out/lib/ruby/gems
          bundle config set --local deployment 'true'
          bundle config set --local path 'vendor/bundle'
          bundle install
          
          {{BUILD_COMMANDS}}
        '';
        
        installPhase = ''
          mkdir -p $out/lib/ruby
          cp -r * $out/lib/ruby/
          
          mkdir -p $out/bin
          cat > $out/bin/start << 'EOF'
          #!/usr/bin/env bash
          cd $out/lib/ruby
          export GEM_HOME=$out/lib/ruby/gems
          {{START_COMMAND}}
          EOF
          chmod +x $out/bin/start
        '';
      };
    };
}
```

---

## Template Selection Logic

The agent uses this logic to select the appropriate template:

```go
func SelectTemplate(config Config) (string, error) {
    // Check if user provided custom flake.nix
    if config.Type == "custom" {
        if config.Nix.FlakeFile != "" && fileExists(config.Nix.FlakeFile) {
            return config.Nix.FlakeFile, nil
        }
        if fileExists("flake.nix") {
            return "flake.nix", nil
        }
        return "", errors.New("custom type selected but no flake.nix found")
    }
    
    // Select built-in template
    templates := map[string]string{
        "laravel": "templates/laravel.nix.tmpl",
        "nodejs":  "templates/nodejs.nix.tmpl",
        "go":      "templates/go.nix.tmpl",
        "python":  "templates/python.nix.tmpl",
        "django":  "templates/django.nix.tmpl",
        "static":  "templates/static.nix.tmpl",
        "ruby":    "templates/ruby.nix.tmpl",
    }
    
    template, ok := templates[strings.ToLower(config.Type)]
    if !ok {
        return "", fmt.Errorf("unknown project type: %s", config.Type)
    }
    
    return template, nil
}
```

---

## Template Rendering Example

```go
func RenderTemplate(templatePath string, config Config) error {
    // Read template
    tmplContent, err := ioutil.ReadFile(templatePath)
    if err != nil {
        return err
    }
    
    // Create replacements map
    replacements := map[string]string{
        "{{PROJECT_NAME}}":    config.Name,
        "{{PHP_VERSION}}":     formatPHPVersion(config.Runtime.PHP),
        "{{NODE_VERSION}}":    config.Runtime.Node,
        "{{GO_VERSION}}":      formatGoVersion(config.Runtime.Go),
        "{{PYTHON_VERSION}}":  formatPythonVersion(config.Runtime.Python),
        "{{RUBY_VERSION}}":    formatRubyVersion(config.Runtime.Ruby),
        "{{BUILD_COMMANDS}}":  strings.Join(config.Build.Commands, "\n          "),
        "{{START_COMMAND}}":   config.Start.Command,
    }
    
    // Replace all placeholders
    result := string(tmplContent)
    for placeholder, value := range replacements {
        result = strings.ReplaceAll(result, placeholder, value)
    }
    
    // Write generated flake.nix
    return ioutil.WriteFile("flake.nix", []byte(result), 0644)
}

func formatPHPVersion(version string) string {
    // "8.2" -> "82"
    return strings.ReplaceAll(version, ".", "")
}

func formatGoVersion(version string) string {
    // "1.21" -> "1_21"
    return strings.ReplaceAll(version, ".", "_")
}

func formatPythonVersion(version string) string {
    // "3.11" -> "311"
    return strings.ReplaceAll(version, ".", "")
}
```

---

## User Experience Example

**1. User creates `config.vpspilot.json` in their Laravel project:**

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

**2. User pushes to GitHub**

**3. VPS Pilot deploys:**
- Agent clones repo
- Reads `config.vpspilot.json`
- Sees `type: "laravel"`
- Loads `laravel.nix.tmpl`
- Injects: PHP 8.2, Node 20
- Generates `flake.nix` automatically
- Builds with Nix
- Starts app

**4. Done! No Nix knowledge needed!**

---

## Advanced: Custom Template Override

If a user wants more control, they can provide their own template:

```json
{
  "name": "My Complex App",
  "type": "custom",
  "nix": {
    "flake_file": "./deployment/custom-flake.nix"
  }
}
```

The agent will use the user's custom `flake.nix` instead of a template.

---

## Benefits

✅ **Zero Nix knowledge required** for users  
✅ **Heroku-like experience** - just configure and deploy  
✅ **Full Nix power** under the hood  
✅ **Escape hatch** for advanced users (custom flakes)  
✅ **Consistent** environments across all nodes  
✅ **Version flexibility** - each project can use different versions  

---

**See also:**
- [NIX_DEPLOYMENT.md](NIX_DEPLOYMENT.md) - Complete deployment architecture
- [NIX_QUICK_REFERENCE.md](NIX_QUICK_REFERENCE.md) - Quick reference guide
- [README.md](../README.md) - Project overview
