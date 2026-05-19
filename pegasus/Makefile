# ═══════════════════════════════════════════════════════════════════════════════
# PEGASUS — LEVIATHAN OPERATIONS MAKEFILE
# Enterprise-grade local simulation, seeding, and E2E testing orchestrator.
# ═══════════════════════════════════════════════════════════════════════════════

SHELL := /bin/bash
.DEFAULT_GOAL := help

# ── Infrastructure Configuration ────────────────────────────────────────────
# Non-secret defaults for local emulators. Secrets (JWT_SECRET, VAULT_MASTER_KEY,
# INTERNAL_API_KEY, GEMINI_API_KEY, …) are NEVER hardcoded here — they must be
# provided via apps/backend-go/.env (see .env.example for the full schema).
export SPANNER_PROJECT        ?= pegasus-logistics
export SPANNER_INSTANCE       ?= pegasus-dev
export SPANNER_DATABASE       ?= pegasus-db
export SPANNER_EMULATOR_HOST  ?= localhost:9010
export KAFKA_BROKER_ADDRESS   ?= localhost:9092
export REDIS_ADDRESS          ?= localhost:6379
export BACKEND_PORT           ?= 8080

BACKEND_DIR   := apps/backend-go
ADMIN_DIR     := apps/admin-portal
DDL_FILE      := $(BACKEND_DIR)/schema/spanner.ddl
ENV_FILE      := $(BACKEND_DIR)/.env
DEFAULT_SPRINT1_BASE_SHA := $(shell /usr/bin/git merge-base HEAD origin/main 2>/dev/null || /usr/bin/git merge-base HEAD main 2>/dev/null || true)

# ── Secret Preflight ────────────────────────────────────────────────────────
# Every target that boots Go code depends on `require-env`, which refuses to
# run if apps/backend-go/.env is missing. godotenv inside the Go binary then
# loads the file; the binary itself panics if any required variable is empty.
.PHONY: require-env
require-env:
	@if [ ! -f $(ENV_FILE) ]; then \
		echo "═══ FATAL: $(ENV_FILE) is missing ═══"; \
		echo "Copy $(BACKEND_DIR)/.env.example to $(ENV_FILE) and populate it."; \
		exit 1; \
	fi

# ═══════════════════════════════════════════════════════════════════════════════
# TARGETS
# ═══════════════════════════════════════════════════════════════════════════════

.PHONY: help env-up env-down env-status optimizer-sim-up optimizer-sim-down optimizer-sim-bootstrap spanner-init seed simulate test-e2e \
	build clean nuke full-reset dev-ip dev-devices sprint1-gate \
	desktop-dev desktop-build desktop-build-mac desktop-build-win desktop-check desktop-clean

## ── help: Display available targets ────────────────────────────────────────
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

## ── dev-ip: Print this Mac's LAN IP for physical-device dev ──────────────
dev-ip: ## Print LAN IP to plug into client dev configs
	@ip=$$(ipconfig getifaddr en0 2>/dev/null); \
	 if [ -z "$$ip" ]; then ip=$$(ipconfig getifaddr en1 2>/dev/null); fi; \
	 if [ -z "$$ip" ]; then ip=$$(ifconfig 2>/dev/null | awk '/inet /{print $$2}' | grep -v '^127\.' | head -1); fi; \
	 if [ -z "$$ip" ]; then echo "Could not detect LAN IP. Join Wi-Fi and retry."; exit 1; fi; \
	 echo "LAN IP: $$ip"; \
	 echo ""; \
	 echo "Backend binds 0.0.0.0:$(BACKEND_PORT) — devices on the same Wi-Fi can reach http://$$ip:$(BACKEND_PORT)"

## ── dev-devices: Print the per-client recipe for physical-device testing ──
dev-devices: ## Show per-client setup for physical-device dev
	@ip=$$(ipconfig getifaddr en0 2>/dev/null); \
	 if [ -z "$$ip" ]; then ip=$$(ipconfig getifaddr en1 2>/dev/null); fi; \
	 if [ -z "$$ip" ]; then ip="<LAN-IP>"; fi; \
	 echo "═══ Physical-device dev recipe (Mac LAN IP: $$ip) ═══"; \
	 echo ""; \
	 echo "Next.js portals (admin / warehouse / factory / retailer-desktop):"; \
	 echo "  cp apps/<portal>/.env.local.example apps/<portal>/.env.local"; \
	 echo "  Set NEXT_PUBLIC_API_URL=http://$$ip:$(BACKEND_PORT) (and WS_URL)"; \
	 echo ""; \
	 echo "Expo payload-terminal:"; \
	 echo "  cp apps/payload-terminal/.env.example apps/payload-terminal/.env"; \
	 echo "  Set EXPO_PUBLIC_API_URL=http://$$ip:$(BACKEND_PORT)"; \
	 echo ""; \
	 echo "Android (driver / retailer / warehouse / factory):"; \
	 echo "  echo 'dev.host=$$ip' >> apps/<app>-android/local.properties"; \
	 echo "  Gradle re-reads on next build."; \
	 echo ""; \
	 echo "iOS (driverappios / retailer-app-ios / warehouse-app-ios / factory-app-ios):"; \
	 echo "  Xcode → Product → Scheme → Edit Scheme → Run → Arguments → Environment Variables"; \
	 echo "  Add: PEGASUS_DEV_HOST=$$ip   (or PEGASUS_DEV_HOST=$$ip:$(BACKEND_PORT))"; \
	 echo ""; \
	 echo "Backend: already binds 0.0.0.0 — no change needed."

## ── env-up: Start all emulators (Kafka KRaft, Redis, Spanner) ─────────────
env-up: ## Start docker-compose infrastructure
	@echo "═══ IGNITION: Starting emulator fleet ═══"
	docker compose up -d
	@echo "Waiting for healthchecks..."
	@sleep 3
	@docker compose ps
	@echo "═══ Emulators online ═══"

## ── env-down: Stop and destroy all emulator volumes ───────────────────────
env-down: ## Stop infrastructure and purge volumes
	@echo "═══ SHUTDOWN: Tearing down emulators ═══"
	docker compose down -v
	@echo "═══ Clean shutdown complete ═══"

## ── env-status: Show running container status ─────────────────────────────
env-status: ## Show container health
	docker compose ps

## ── optimizer-sim-up: Start OR-Tools local simulation profile ─────────────
optimizer-sim-up: ## Start optimizer sidecar + adapter worker on local emulators
	@echo "═══ OPTIMIZER SIM: Starting local Docker profile (no cloud API keys) ═══"
	docker compose --profile optimizer-sim up -d
	@echo "═══ Optimizer simulation profile started ═══"
	@echo "Tip: if this is the first run, initialize schema with 'make spanner-init'"

## ── optimizer-sim-down: Stop OR-Tools local simulation profile ────────────
optimizer-sim-down: ## Stop optimizer sidecar + adapter worker profile
	@echo "═══ OPTIMIZER SIM: Stopping profile services ═══"
	docker compose --profile optimizer-sim stop optimizer-core-rust optimizer-adapter-worker
	@echo "═══ Optimizer simulation profile stopped ═══"

## ── optimizer-sim-bootstrap: Bring up profile + initialize Spanner ────────
optimizer-sim-bootstrap: require-env optimizer-sim-up spanner-init ## First-run bootstrap for optimizer local simulation
	@echo "═══ Optimizer simulation bootstrap complete ═══"

## ── spanner-init: Create instance + database + apply DDL ──────────────────
spanner-init: require-env ## Initialize Spanner emulator (instance + database + full DDL)
	@echo "═══ SPANNER INIT: Provisioning instance and database ═══"
	cd $(BACKEND_DIR) && go run cmd/setup/main.go
	@echo "═══ Spanner schema applied ═══"

## ── seed: Run the deterministic E2E seeder ────────────────────────────────
seed: require-env ## Seed Spanner + Redis + Kafka topics with demo data
	@echo "═══ SEEDING: Injecting deterministic test matrix ═══"
	cd $(BACKEND_DIR) && go run cmd/seed/main.go
	@echo "═══ Seed complete ═══"

## ── build: Compile the backend ────────────────────────────────────────────
build: ## Build the Go backend
	cd $(BACKEND_DIR) && go build -o backend-go ./...

## ── run-backend: Run the backend with emulator ENV vars ───────────────────
run-backend: require-env ## Run the backend natively
	@echo "═══ STARTING BACKEND (Local Emulators) ═══"
	cd $(BACKEND_DIR) && go run .
	@echo "═══ SIMULATE: Full stack ignition ═══"
	@sleep 5
	@$(MAKE) spanner-init
	@$(MAKE) seed
	@echo "═══ Starting backend on :$(BACKEND_PORT) ═══"
	@trap 'kill 0' EXIT; \
                cd $(BACKEND_DIR) && go run . & \

## ── migrate: Run Spanner DDL migrations out-of-band (production pattern) ──
migrate: require-env ## Apply pending Spanner DDL via cmd/migrate (idempotent)
	@echo "═══ MIGRATE: Applying Spanner DDL out-of-band ═══"
	cd $(BACKEND_DIR) && go run ./cmd/migrate
	@echo "═══ Migrations applied ═══"

## ── nuke: Purge all Spanner data (uses admin endpoint) ────────────────────
nuke: require-env ## Truncate all Spanner tables via seed script
	cd $(BACKEND_DIR) && go run cmd/seed/main.go
	@echo "═══ Data nuked and re-seeded ═══"

## ── reset-db: Reset database schema and inject simulation ─────────────────
reset-db: require-env ## Truncate all tables and inject clear simulation
	@echo "═══ RESET: Clearing database and injecting clean simulation ═══"
	cd $(BACKEND_DIR) && go run cmd/simulate/*.go -clear -suppliers=5 -warehouses=2 -retailers=100
	@echo "═══ Database simulation reset complete ═══"

## ── full-reset: env-down → env-up → spanner-init → seed ──────────────────
full-reset: env-down env-up ## Full infrastructure reset + reseed
	@sleep 5
	@$(MAKE) spanner-init
	@$(MAKE) seed
	@echo "═══ FULL RESET COMPLETE ═══"

## ── sprint1-gate: Run Sprint-1 execution baseline guard suite ─────────────
sprint1-gate: ## Run Sprint-1 enterprise execution gate and emit report JSON
	@echo "═══ SPRINT-1 EXECUTION GATE ═══"
	@args="--repo-root .. --output-dir pegasus/.execution/sprint1 --with-enforce --changed-only --diff-mode working-tree"; \
	base="$(BASE_SHA)"; \
	if [ -z "$$base" ]; then base=$$(/usr/bin/git merge-base HEAD origin/main 2>/dev/null || /usr/bin/git merge-base HEAD main 2>/dev/null || true); fi; \
	head="$(HEAD_SHA)"; \
	if [ -z "$$head" ] && [ -n "$$base" ]; then head="HEAD"; fi; \
	if [ -n "$$base" ]; then args="$$args --base-sha $$base"; fi; \
	if [ -n "$$head" ]; then args="$$args --head-sha $$head"; fi; \
	/usr/bin/python3 scripts/sprint1_execution_gate.py $$args
	@echo "═══ Sprint-1 gate report written to pegasus/.execution/sprint1/gate-report.json ═══"

## ── clean: Remove build artifacts ─────────────────────────────────────────
clean: ## Clean build artifacts
	cd $(BACKEND_DIR) && rm -f backend-go main seed
	@echo "═══ Artifacts cleaned ═══"

# ═══════════════════════════════════════════════════════════════════════════════
# DESKTOP (Tauri) TARGETS
# ═══════════════════════════════════════════════════════════════════════════════

.PHONY: desktop-dev desktop-build desktop-build-mac desktop-build-win desktop-check desktop-clean

## ── desktop-dev: Launch Admin Portal in Tauri dev mode ────────────────────
desktop-dev: ## Start desktop dev (Tauri + Next.js)
	@echo "═══ DESKTOP DEV: Launching Tauri shell ═══"
	cd $(ADMIN_DIR) && npm run tauri:dev

## ── desktop-build: Production desktop build for current platform ──────────
desktop-build: ## Build desktop app for current OS
	@echo "═══ DESKTOP BUILD: Compiling release binary ═══"
	cd $(ADMIN_DIR) && npm run tauri:build

## ── desktop-build-mac: macOS universal binary (.app + .dmg) ───────────────
desktop-build-mac: ## Build macOS universal desktop app
	@echo "═══ DESKTOP BUILD: macOS universal ═══"
	cd $(ADMIN_DIR) && npm run tauri:build:mac

## ── desktop-build-win: Windows x64 (.exe + .msi) ─────────────────────────
desktop-build-win: ## Build Windows x64 desktop app
	@echo "═══ DESKTOP BUILD: Windows x64 ═══"
	cd $(ADMIN_DIR) && npm run tauri:build:win

## ── desktop-check: Verify Rust compiles ───────────────────────────────────
desktop-check: ## Cargo check the Tauri crate
	cd $(ADMIN_DIR)/src-tauri && cargo check
	@echo "═══ Rust check passed ═══"

## ── desktop-clean: Remove Tauri build artifacts ───────────────────────────
desktop-clean: ## Clean desktop build output
	cd $(ADMIN_DIR)/src-tauri && cargo clean
	rm -rf $(ADMIN_DIR)/out
	@echo "═══ Desktop artifacts cleaned ═══"
