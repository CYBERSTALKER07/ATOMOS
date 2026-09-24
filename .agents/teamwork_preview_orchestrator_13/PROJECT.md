# Project: Pegasus Sovereign Core Full-Stack Modularization (pegasus.x)

## Architecture

### Backend Domain Subrouter Architecture
- Interface: `backend/internal/api/modules/module.go` defines `Module` interface (`RegisterRoutes(r chi.Router)`) and `Registry`.
- Domain Subrouters in `package api`:
  - `core.go`: Probes (`/health`, `/healthz`, `/ready`), client policy, metrics, JWKS, public auth, sync webhooks, onboarding lifecycle, media upload, users, notifications.
  - `logistics.go`: Telemetry, fleet, driver shifts & DVIR, dispatch solver & preview, manifests, epod, control tower, GS1, SOATO, routing, SMS POD.
  - `warehouse.go`: Warehouse onboarding, metrics, dock bay provisioning, inventory, stock adjustments, cycle counts, replenishments, pick waves, preorders, floor exceptions, tomorrow board, coverage, demand forecast, scheduling, crossdock, returns, bins, lots, empties, QM, EWM.
  - `commercial.go`: Catalog, products, orders, checkout, supplier profile/kyc/pricing/crm/promotions, retailer store POS/shifts/holds/local SKUs/team/pulse/auto-order, UMP, claims, voice ordering.
  - `finance.go`: Soliq OFD fiscalization, VAT verification, credit notes, cash ledger & reconciliation, payments & split-tender, credit limits & applications, AR, payouts, FX rates, seasonality, Global Pay webhooks, SoftPOS charge, COPA/FSCM/payroll.
- Router Monolith: `backend/internal/api/router.go` reduced from 2,448 lines to <900 lines by delegating route registrations to the 5 domain subrouter modules.
- DTO Unification: `backend/internal/api/dto.go` unifies anonymous inline DTOs (Refresh Token, Supervisor PIN, Operational Status, Dispatch Buffer, Action Reason, Supplier Attachment, etc.) preserving exact JSON tags.

### Frontend Shared Monorepo Architecture
- Single Source of Truth for Types: `@pegasusx/types` absorbs all types from `contracts/types.ts` and `contracts/regional_types.ts`, and local duplicate types (e.g. `apps/warehouse-desktop/lib/dispatch-types.ts`). `contracts/` re-exports `@pegasusx/types` for backward compatibility.
- Control Tower UI Primitives:
  - `@pegasusx/ui-kit`: Consolidates `NavigationRail`, `DetailDrawer` (`ContextInspectorDrawer`), and `KpiStatCard` / `KpiStatGrid` (`DensityMetricCard`).
  - `@pegasusx/pulse-ui`: Consolidates `NetworkPulsePanel` alongside `PulseTimeline`.
- Dependency Hygiene: Purge unused `firebase` dependency from `apps/warehouse-desktop/package.json` (and other desktop apps) and remove dead `lib/firebase.ts`.
- Build Tooling: Standardize `"build": "tsc --noEmit"` and `"typecheck"` scripts across packages.

### Infrastructure Gateway & Compose Architecture
- Base Service Definition: `docker-compose.base.yml` defining common services (`postgres`, `redis`, `backend`, `planning`, portals), healthchecks, restart policies, and base networks/volumes without seed volume leakage into production.
- Environment Overlays:
  - `docker-compose.dev.yml`: Exposes host ports (`5432`, `6379`, `8080`, `8000`, `3000-3002`), mounts seeds, sets development tuning.
  - `docker-compose.prod.yml`: Adds `caddy`, `retailer-telegram-miniapp`, `telegram-bot`, `prometheus`, `grafana`, production `postgres.conf`, and network hardening.
  - `docker-compose.yml`: Top-level composition including base and dev overlays, removing obsolete `version: '3.8'`.
- Caddy Gateway Modular Snippets:
  - `docker/caddy.d/api.caddy`: API gateway reverse proxy rules and headers.
  - `docker/caddy.d/ws.caddy`: Dedicated WebSocket proxying with `flush_interval -1`.
  - `docker/caddy.d/portal.caddy`: Portal reverse proxies for supplier, retailer, warehouse, and miniapp.
  - `docker/Caddyfile`: Root configuration importing `caddy.d/*.caddy` and mapping site blocks with 100% route retention.

---

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Domain Route Module Interface | Define `Module` interface with `RegisterRoutes(r chi.Router)` in `backend/internal/api/modules/module.go` | M1 | Survey 1 |
| 2 | Domain Subrouter Decomposition | Decompose `backend/internal/api/router.go` into `logistics.go`, `warehouse.go`, `commercial.go`, `finance.go`, `core.go` | M1 | Survey 1 |
| 3 | Unified API DTOs | Extract and unify repeated inline DTO structs into `backend/internal/api/dto.go` | M1 | Survey 1 |
| 4 | Router Line Reduction & Parity | Reduce `router.go` to <900 lines while maintaining 100% route contract parity across all 1,119 endpoints | M1 | Survey 1 |
| 5 | Contracts & Types Synchronization | Synchronize `contracts/` with `@pegasusx/types`, eliminating duplicate type declarations | M2 | Survey 2 |
| 6 | Shared Control Tower UI Primitives | Extract `NavigationRail`, `DetailDrawer`, and `KpiStatCard` into `@pegasusx/ui-kit` and `NetworkPulsePanel` into `@pegasusx/pulse-ui` | M2 | Survey 2 |
| 7 | Purge Stale Dependencies | Remove `firebase` from `apps/warehouse-desktop/package.json` and delete dead `lib/firebase.ts` | M2 | Survey 2 |
| 8 | Frontend Builds & Typechecks | Ensure clean TypeScript compilation and builds across `@pegasusx/types`, `@pegasusx/pulse-ui`, `@pegasusx/ui-kit`, and desktop apps | M2 | Survey 2 |
| 9 | Docker Compose Modular Overlays | Refactor into `docker-compose.base.yml`, `docker-compose.dev.yml`, and `docker-compose.prod.yml` | M3 | Survey 3 |
| 10 | Caddy Gateway Snippets | Split `docker/Caddyfile` into `caddy.d/api.caddy`, `caddy.d/ws.caddy`, `caddy.d/portal.caddy` with 100% route retention | M3 | Survey 3 |
| 11 | Backend Regression Suite | Verify `go vet ./...` (0 diagnostics) and `go test -v -race ./...` (100% pass) across backend Go workspace | M4 | Survey 1 & Plan |
| 12 | Frontend Monorepo Verification | Verify `pnpm --filter @pegasusx/* build` passes cleanly across all packages | M4 | Survey 2 & Plan |
| 13 | Infrastructure Verification | Validate `docker compose config` across standalone, dev, and prod overlay invocations | M4 | Survey 3 & Plan |

---

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| 1 | Backend Domain Subrouter & Route Module Decomposition (R1) | Tasks 1, 2, 3: Create `modules.Module`, subrouters (`logistics.go`, `warehouse.go`, `commercial.go`, `finance.go`, `core.go`), `dto.go`, reduce `router.go` to <900 lines | None | DONE |
| 2 | Frontend Shared Monorepo Package Consolidation (R2) | Tasks 4, 5, 6: Synchronize `@pegasusx/types` with `contracts/`, extract `NavigationRail`, `DetailDrawer`, `KpiStatCard` to `ui-kit`, `NetworkPulsePanel` to `pulse-ui`, purge `firebase` | None | DONE |
| 3 | Infrastructure Gateway & Compose Modularization (R3) | Tasks 7, 8: Modularize `docker-compose.base.yml`, `docker-compose.dev.yml`, `docker-compose.prod.yml`, `docker/caddy.d/*.caddy` | None | DONE |
| 4 | Comprehensive Full-Stack Verification & Zero-Regression Assurance (R4) | Task 9: Run full `go test -v -race ./...`, `pnpm build`, `docker compose config` verification gates | M1, M2, M3 | DONE |

---

## Interface Contracts

### Backend Router ↔ Domain Modules
```go
package modules

import "github.com/go-chi/chi/v5"

type Module interface {
    Name() string
    RegisterRoutes(r chi.Router)
}

type Registry struct {
    modules []Module
}

func NewRegistry() *Registry
func (reg *Registry) Register(m Module)
func (reg *Registry) MountAll(r chi.Router)
```

Each concrete subrouter (`LogisticsModule`, `WarehouseModule`, `CommercialModule`, `FinanceModule`, `CoreModule`) resides in `package api`, embeds `s *Server`, and implements `modules.Module`.

### Frontend Monorepo Packages
```typescript
// @pegasusx/types (Single Source of Truth)
// Re-exported by contracts/index.ts for backward compatibility:
export * from '@pegasusx/types';

// @pegasusx/ui-kit (Tactical Control Tower Primitives)
export { NavigationRail } from './desktop/NavigationRail';
export { DetailDrawer, ContextInspectorDrawer } from './desktop/DetailDrawer';
export { KpiStatCard, KpiStatGrid, DensityMetricCard } from './portal/KpiStat';

// @pegasusx/pulse-ui (Real-time Telemetry Primitives)
export { NetworkPulsePanel } from './NetworkPulsePanel';
export { PulseTimeline } from './PulseTimeline';
```

### Infrastructure Overlays
- Invocations:
  - Dev: `docker compose up -d` or `docker compose -f docker-compose.base.yml -f docker-compose.dev.yml up -d`
  - Prod: `docker compose -f docker-compose.base.yml -f docker-compose.prod.yml up -d`
  - Validation: `docker compose -f docker-compose.base.yml -f docker-compose.prod.yml config`

---

## Code Layout
- `backend/internal/api/modules/module.go`: Interface & Registry
- `backend/internal/api/logistics.go`: Logistics subrouter
- `backend/internal/api/warehouse.go`: Warehouse subrouter
- `backend/internal/api/commercial.go`: Commercial subrouter
- `backend/internal/api/finance.go`: Finance subrouter
- `backend/internal/api/core.go`: Core system subrouter
- `backend/internal/api/dto.go`: Unified API DTOs
- `backend/internal/api/router.go`: Lean Server & router orchestrator (<900 lines)
- `packages/types/src/`: Consolidated domain types and DTOs
- `contracts/index.ts`: Re-exporting `@pegasusx/types`
- `packages/ui-kit/src/desktop/`: NavigationRail, DetailDrawer
- `packages/ui-kit/src/portal/`: KpiStatCard
- `packages/pulse-ui/src/`: NetworkPulsePanel, PulseTimeline
- `apps/warehouse-desktop/package.json`: Purged of firebase
- `docker-compose.base.yml`: Common services definition
- `docker-compose.dev.yml`: Dev overlay
- `docker-compose.prod.yml`: Prod overlay
- `docker-compose.yml`: Top-level composition
- `docker/caddy.d/api.caddy`: API gateway snippet
- `docker/caddy.d/ws.caddy`: WebSocket gateway snippet
- `docker/caddy.d/portal.caddy`: Portal reverse proxy snippets
- `docker/Caddyfile`: Lean root Caddy configuration importing snippets
