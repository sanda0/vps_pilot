# VPS Pilot — development tasks. Run `make help` for the list.

# Hub settings (DB_PATH, TOKEN_SECRET, ...) live in server/.env; the hub
# loads that file itself. It's read here only for the agent port.
-include server/.env

HUB_PORT    ?= 8080
AGENT_PORT  ?= $(or $(TCP_SERVER_PORT),55001)
AGENT_STATE ?= $(CURDIR)/agent/.dev/state.json
SQLC        := go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0

export HUB_PORT

.DEFAULT_GOAL := help
.PHONY: help dev hub ui agent agent-reset install superuser migrate sqlc \
        test lint fmt build clean ensure-dist ensure-bun

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

dev: ensure-dist ## Run hub (hot reload) + dashboard dev server together
	@echo "Dashboard: http://localhost:5173   Hub API: http://localhost:$(HUB_PORT)   Agent port: $(AGENT_PORT)"
	@$(MAKE) --no-print-directory -j2 hub ui

hub: ensure-dist ## Run the hub with hot reload (air), or go run without it
	@cd server && if command -v air >/dev/null 2>&1; then \
		air -- -port $(HUB_PORT) -agent-port $(AGENT_PORT); \
	else \
		go run . -port $(HUB_PORT) -agent-port $(AGENT_PORT); \
	fi

ui: ensure-bun ## Run the dashboard dev server (proxies /api to the hub)
	cd client && bun run dev

agent: ## Run the agent against the local hub (first run: VPS_PILOT_TOKEN=vpsp1...; optional PROJECT_DIRS=~/dir)
	cd agent && VPS_PILOT_TOKEN="$(VPS_PILOT_TOKEN)" cargo run -- \
		--hub 127.0.0.1:$(AGENT_PORT) --state $(AGENT_STATE) \
		$(if $(PROJECT_DIRS),--projects-dir $(PROJECT_DIRS))

agent-reset: ## Forget the dev agent's enrollment
	rm -f $(AGENT_STATE)

install: ensure-bun ## Install dashboard dependencies
	cd client && bun install

superuser: ## Create a dashboard login
	cd server && go run . -create-superuser

migrate: ## Apply database migrations
	cd server && go run . -migrate

sqlc: ## Regenerate Go code from SQL queries
	cd server && $(SQLC) generate

test: ensure-dist ensure-bun ## Run hub and agent tests, type-check the dashboard
	cd server && go test -race ./...
	cd agent && cargo test
	cd client && bun run build

lint: ensure-dist ## go vet, rustfmt check, clippy
	cd server && go vet ./...
	cd agent && cargo fmt --check && cargo clippy --all-targets -- -D warnings

fmt: ## Format Go and Rust code
	cd server && gofmt -w .
	cd agent && cargo fmt

build: ensure-bun ## Production build: hub with embedded UI + agent
	./build.sh

clean: ## Remove build output (keeps databases)
	rm -rf server/vps_pilot server/tmp server/cmd/app/dist agent/target

# go:embed needs server/cmd/app/dist to contain at least one file.
ensure-dist:
	@mkdir -p server/cmd/app/dist
	@[ -n "$$(ls -A server/cmd/app/dist)" ] || touch server/cmd/app/dist/.gitkeep

# The dashboard is built with bun (https://bun.sh).
ensure-bun:
	@command -v bun >/dev/null 2>&1 || { \
		echo "bun not found: install it with 'curl -fsSL https://bun.sh/install | bash'"; exit 1; }
