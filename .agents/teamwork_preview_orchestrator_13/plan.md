# Full-Stack Modularization Execution Plan — Orchestrator 13

## Objectives
Execute the modularization plan for Pegasus Sovereign Core (`pegasus.x`) addressing Requirements R1, R2, R3, R4:
1. Backend Domain Subrouter & Route Module Decomposition (`backend/internal/api/router.go` from 2,448 to <950 lines)
2. Frontend Shared Monorepo Package Consolidation (`@pegasusx/types`, `@pegasusx/pulse-ui`, `@pegasusx/ui-kit`, purge firebase)
3. Infrastructure Gateway & Compose Modularization (`docker-compose.yml` overlays, `docker/Caddyfile` domain snippets)
4. Comprehensive Verification & Zero-Regression Assurance (`go test -v -race ./...`, `pnpm build`, `docker compose config`)

## Execution Phases
### Phase 0: Parallel Codebase Survey
- Explorer 1 (Backend): Map all routes, handlers, middleware, inline DTOs, and existing `internal/api/modules` or router abstractions in `backend/internal/api/router.go`.
- Explorer 2 (Frontend): Map `contracts/`, `packages/types/`, `packages/pulse-ui/`, `packages/ui-kit/`, and `apps/warehouse-desktop/package.json` for drift, missing control tower primitives, and unused dependencies.
- Explorer 3 (Infrastructure): Map `docker-compose.yml`, environment requirements (`base`, `dev`, `prod`), and `docker/Caddyfile` for domain route snippet extraction.

### Phase 1: Milestone 1 — Backend Modularization (R1)
- Worker: Implement `Module` interface in `backend/internal/api/modules/` or domain subrouters (`logistics.go`, `warehouse.go`, `commercial.go`, `finance.go`, `core.go`). Unify inline DTO structs. Reduce `router.go` to <950 lines while preserving 100% route contract parity.
- Reviewer 1 & 2: Verify `go vet ./...`, route parity, and lack of regression.
- Challenger & Auditor: Verify fail-closed architecture, zero mocks in production, and test passes.

### Phase 2: Milestone 2 — Frontend Consolidation (R2)
- Worker: Sync `contracts/` and `@pegasusx/types`. Extract NavRail, MetricCards, InspectorDrawer to shared packages. Purge `firebase` from `warehouse-desktop/package.json`.
- Reviewer 1 & 2: Check TypeScript compilation (`pnpm build`), token compliance.
- Challenger & Auditor: Verify zero contract drift.

### Phase 3: Milestone 3 — Infrastructure Modularization (R3)
- Worker: Refactor `docker-compose.yml` with overlays (`docker-compose.base.yml`, `docker-compose.dev.yml`, `docker-compose.prod.yml`). Modularize `docker/Caddyfile` into snippets (`caddy.d/api.caddy`, `caddy.d/ws.caddy`, `caddy.d/portal.caddy`).
- Reviewer: Validate with `docker compose config` and Caddy validation.

### Phase 4: Milestone 4 — Final Verification (R4)
- Run full suite: `go test -v -race ./...`, full frontend builds, docker compose config.
- Pass all quality gates.
