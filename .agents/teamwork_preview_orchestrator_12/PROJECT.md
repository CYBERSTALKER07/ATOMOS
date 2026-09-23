# Project: pegasus.x Codebase Hardening & Enterprise Doctrine Enforcement

## Architecture
- Codebase: /Users/shakhzod/Desktop/V.O.I.D/pegasus.x
- Sovereign Single-Tenant Core: PostgreSQL 16 (pgx/v5) + TimescaleDB + PostGIS, Redis 7 Streams, Caddy 2 reverse proxy, Next.js 15 / Tauri v2 Desktop clients.
- Strict Two-System Architectural Boundary: Strictly PostgreSQL 16 + Redis 7 Streams. NO Spanner, NO Kafka.
- Strict Financial Arithmetic: 64-bit integer tiyin minor units (int64). Zero floating-point math for pricing, invoices, VAT, discounts, or ledger balances.
- Persistence Rigor: Production constructors fail closed if *db.Pool is nil. Zero in-memory repository fallbacks or fake seeds in production packages.
- Real-Time Monotonic Pipeline: Entity state update and outbox event committed in the exact same pgx.Tx; outbox relay polls via `FOR UPDATE SKIP LOCKED` and publishes to Redis 7 Streams (`XADD`); WebSocket Hub broadcasts monotonic `RealtimeEnvelope` frames (`seq`, dual `event_type` and `type` fields); Desktop clients invalidate and refresh reactively.

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | R2 Purge In-Memory Fallbacks | Purge in-memory fallback repos in consignment, rebate, payout, wmsops; fail closed constructors | M1 | ORIGINAL_REQUEST |
| 2 | R1 Currency & Tiyin Minor Units | Audit and eliminate floating-point currency math; enforce int64 tiyins across all domain models & handlers | M2 | ORIGINAL_REQUEST |
| 3 | R1 Domain State Machine Purity | Audit mutations for domain state machines, concurrency checks, validation guards | M2 | ORIGINAL_REQUEST |
| 4 | R3 Outbox Atomic Pairing | Mutating state transitions commit entity mutation & outbox event in same pgx.Tx | M3 | ORIGINAL_REQUEST |
| 5 | R3 Outbox Relay & Redis Streams | Relay polls with FOR UPDATE SKIP LOCKED, publishes to Redis 7 Streams with aggregate root keys | M3 | ORIGINAL_REQUEST |
| 6 | R3 WebSocket & Desktop Invalidation | Monotonic RealtimeEnvelope (seq, dual event_type/type), Tauri/Desktop reactive UI listeners | M3 | ORIGINAL_REQUEST |
| 7 | R4 Full Test Suite & Race Detection | Execute go test -v -race ./... with zero failures, zero race conditions, passing benchmarks | M4 | ORIGINAL_REQUEST |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| 0 | Codebase Survey & In-Depth Audit | 3 parallel Explorers surveying R1, R2, R3 across pegasus.x | none | DONE |
| 1 | Purge In-Memory Fallbacks & Fail-Closed Constructors | cmd/server/main.go, internal/consignment, internal/rebate, internal/payout, internal/wmsops | M0 | DONE |
| 2 | Currency Arithmetic & Domain State Machine Purity | soliq/efactura.go, rebate/rebate.go, fscm/dunning.go, copa/copa.go, ar/dunning.go, matching/matching.go, consignment/consignment.go, supplier/service.go, payout/calculator.go, api/handlers_supplier.go, api/handlers_fleet_driver.go, epod/repository.go, wmsops/repository.go, internal/ewm | M1 | DONE |
| 3 | Cross-Role Real-Time Monotonic Pipeline Parity | outbox/relay.go, ws/hub.go, warehouse/service.go, rebate/service.go, consignment/service.go, matching/service.go, warehouse-desktop/lib/fleet-ws-events.ts, supplier-desktop | M1 | READY |
| 4 | Comprehensive Automated Test Suite & Race Verification | go test -v -race ./... across all backend packages & scale benchmarks | M1, M2, M3 | PLANNED |

## Code Layout
- Target: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`
  - `backend/`
    - `cmd/`
    - `internal/`
      - `consignment/`
      - `rebate/`
      - `payout/`
      - `wmsops/`
      - `outbox/`
      - `realtime/`
      - `supplier/`
      - `warehouse/`
      - `order/`
      - `driver/`
      - `finance/`
      - `...`
    - `database/migrations/`
  - Desktop / Web clients in `apps/` or root
