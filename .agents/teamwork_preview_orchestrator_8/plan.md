# Execution Plan: Supplier Onboarding & Fleet Hub in pegasus.x

## Context & Objectives
Implement end-to-end Supplier Sign-Up, Sign-In, Non-Bypassable Phased Onboarding (Catalog MXIK/Tiyins, Cash & Global Pay B2B), and Post-Onboarding Warehouse/Fleet Management Hub in `pegasus.x` using pure PostgreSQL 16 (`pgxpool`) and Redis 7.

## Architecture Boundaries
- `pegasus.x` is SOVEREIGN LEAN SINGLE-TENANT: PostgreSQL 16 + Redis 7 ONLY.
- Zero Spanner, zero Kafka.
- Strict 64-bit integer tiyins for money.
- Purge all mock/memory repositories.

## Execution Phases

### Phase 0: Survey & Codebase Exploration (Parallel)
- Spawn 3 Explorers:
  - Explorer 1: Focus on `pegasus.x/backend/internal/supplier/`, auth routes, existing router, repository, models, and mock data purge requirements.
  - Explorer 2: Focus on database migrations in `pegasus.x/backend/migrations/`, schema patterns, existing tables (suppliers, products, warehouses, trucks, payloaders), and migration 069 requirements.
  - Explorer 3: Focus on middleware, auth JWT tokens, outbox pattern/Redis events in `pegasus.x`, Global Pay gateway requirements, and fleet/dock logistics endpoints.
- Synthesize into `PROJECT.md` with Feature Inventory and Interface Contracts.

### Phase 1: E2E Testing Track (Parallel)
- Dispatch `teamwork_preview_test_writer` to design comprehensive tests covering all 4 tiers (Feature, Boundary/Corner, Cross-Feature, Real-world).
- Generate `TEST_INFRA.md` and test suite in `pegasus.x/backend`.

### Phase 2: Milestone 1 — PostgreSQL 16 Migration 069 & Repository Purge
- Create `069_supplier_onboarding_and_globalpay.sql` covering suppliers, onboarding status, products (MXIK, tiyins, VAT), payment gateways, warehouses (lat/lon), trucks, payloaders.
- Purge `MemoryRepository` and hardcoded seeds from `pegasus.x/backend/internal/supplier/repository.go`.
- Wire `pgxpool` queries.
- Run tests and review.

### Phase 3: Milestone 2 — Supplier Sign-Up & Sign-In with STIR Deduplication
- Implement `POST /v1/auth/supplier/register` (company_name, tax_id 9-digit STIR, phone +998, bcrypt password). Enforce STIR uniqueness in PG with HTTP 409. Default `onboarding_status = 'PENDING'`.
- Implement `POST /v1/auth/supplier/login` (verify bcrypt, issue JWT with claims, `onboarding_status`, `next_step: "/onboarding/products"`).
- Run tests and review.

### Phase 4: Milestone 3 — Non-Bypassable Onboarding Gate & Phased Wizard
- Middleware `RequireSupplierOnboardingCompleted` (HTTP 428 Precondition Required if != 'COMPLETED', whitelist auth and onboarding).
- Step 1: Catalog (`POST /v1/supplier/onboarding/products` - EAN-13, 17-digit MXIK, package code, units_per_case, unit_price_tiyin, 12% VAT).
- Step 2: Payment (`POST /v1/supplier/onboarding/payment` - Cash, Global Pay B2B BIN validation).
- Step 3: Complete (`POST /v1/supplier/onboarding/complete` - status 'COMPLETED', outbox/ws event).
- Run tests and review.

### Phase 5: Milestone 4 — Warehouse & Fleet Management Hub
- `POST/GET/PUT/DELETE /v1/supplier/warehouses` (lat/lon `DOUBLE PRECISION`, Redis proximity cache invalidation, `warehouse.relocated` event, 409 deletion guard on stock/orders).
- `POST/GET /v1/supplier/warehouses/{id}/trucks` (license plate, capacity kg/m³, fuel type).
- `POST/GET /v1/supplier/warehouses/{id}/payloaders` (name, phone, warehouse_id).
- Run tests and review.

### Phase 6: Final Milestone — E2E Test Suite Pass & Hardening
- Run full test suite: `go test -v -race ./...` in `pegasus.x/backend`.
- Dispatch Reviewers to verify all acceptance criteria and code hygiene.
- Generate final handoff report.
