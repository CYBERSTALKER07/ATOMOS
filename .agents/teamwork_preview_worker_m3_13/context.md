# Worker M3 Context: Infrastructure Gateway & Compose Modularization

Working Directory: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3_13`
Target Workspace Directory: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`
Authoritative Requirements: `/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md`
Explorer Findings: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_13_3/handoff.md`
Project Plan: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_13/PROJECT.md`

File Write Ownership (Exclusive):
- `docker-compose.base.yml`
- `docker-compose.dev.yml`
- `docker-compose.prod.yml`
- `docker-compose.yml`
- `docker/caddy.d/api.caddy`
- `docker/caddy.d/ws.caddy`
- `docker/caddy.d/portal.caddy`
- `docker/Caddyfile`

Mission & Tasks:
1. Decompose Docker Compose into modular overlays per Explorer 3 handoff:
   - Create `docker-compose.base.yml` defining common services (`postgres`, `redis`, `backend`, `planning`, `supplier-portal`, `retailer-portal`, `warehouse-portal`), healthchecks, restart policies, and base networks/volumes. Do NOT mount `./database/seeds` in base (prevents seed pollution in production).
   - Create `docker-compose.dev.yml` with `include: [docker-compose.base.yml]`, exposing host ports (`5432`, `6379`, `8080`, `8000`, `3000-3002`), mounting seeds, and setting dev tuning flags.
   - Create `docker-compose.prod.yml` with `include: [docker-compose.base.yml]`, adding `caddy`, `retailer-telegram-miniapp`, `telegram-bot`, `prometheus`, `grafana`, production configs (`postgres.conf`), and network hardening. Mount `./docker/Caddyfile` and `./docker/caddy.d`.
   - Update `docker-compose.yml` to use `include: [docker-compose.base.yml, docker-compose.dev.yml]` ensuring default `docker compose up -d` continues to work seamlessly, and remove obsolete `version: '3.8'`.
2. Decompose `docker/Caddyfile` into modular domain snippets:
   - Create `docker/caddy.d/api.caddy` (`(api_gateway)` snippet).
   - Create `docker/caddy.d/ws.caddy` (`(ws_gateway)` snippet with `flush_interval -1`).
   - Create `docker/caddy.d/portal.caddy` (portal proxy snippets).
   - Refactor `docker/Caddyfile` to import `caddy.d/*.caddy` and map site blocks (`:80`, `:3000`, `:3001`, `:3002`, `:3003`) with 100% route retention.
3. Verification:
   - Run `docker compose config`
   - Run `docker compose -f docker-compose.base.yml -f docker-compose.dev.yml config`
   - Run `docker compose -f docker-compose.base.yml -f docker-compose.prod.yml config`
   - Run `bash scripts/verify-staging.sh` if applicable.
4. Write detailed handoff report to `handoff.md` in your working directory.
