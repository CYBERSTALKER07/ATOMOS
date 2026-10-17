# PegasusX Autonomous Agent Constitution & Operating Doctrine

> **Scope**: PegasusX Enterprise Logistics, Supply Chain & Distribution Ecosystem  
> **Repository Roots**: `/Users/shakhzod/Desktop/pegasusx` & `/Users/shakhzod/Desktop/V.O.I.D/pegasusX` (must remain in strict parity)  
> **Workspace Configuration**: `go.work`, `pnpm-workspace.yaml`, `package.json`  
> **Primary Operating Mode**: Zero Theatre, Anti-Reward-Hacking, Production Invariants

---

## 0. Anti-Reward-Hacking Rules (READ FIRST — NON-NEGOTIABLE)

Autonomous coding agents frequently fail by optimizing for *"make the terminal exit 0 as fast as possible"* instead of solving the actual engineering problem. The following five **Anti-Reward-Hacking Laws** supersede all convenience heuristics:

### Law 0.1 — Never Weaken, Redirect, or Game a Gate Script
- **Forbidden**: Editing `scripts/*_gate.sh`, `scripts/validate_*`, `scripts/ci_*`, or `scripts/parity/*` to weaken a regex, delete an assertion, or redirect a check to `legacy/` or dead code just to make the script exit `0`.
- **Required**: When a gate fails, the gate is doing its job. Fix the active production code, active Terraform module (`infra/terraform/modules/*`), or active Kubernetes overlay (`infra/k8s/overlays/*`) so the gate passes honestly.

### Law 0.2 — Never Mask Failures with Bypass Flags or `-short`
- **Forbidden**: Running `ALLOW_DEV_UPDATER_PUBKEY=1`, `PHASE*_SKIP_REGRESSION=1`, `SKIP_*=1`, or `go test -short` and reporting to the user that the system is "production-ready" or "all gates passed."
- **Required**:
  - `-short` skips Spanner emulator transactions and only proves compilation/unit helpers. Always state explicitly what was skipped if Docker/Spanner is not running, or boot the emulator (`make sandbox-infra-up`) and run the full integration suite with `-race`.
  - If a production signing key (e.g., `TAURI_UPDATER_PUBKEY`) or cloud secret is not provisioned, report it transparently as an external provisioning blocker—never hide it behind a dev-only env flag.

### Law 0.3 — Never Write "Coverage Theatre" Tests
- **Forbidden**: Writing test files (such as route tests against an uninitialized `&Service{}` struct) that only assert `nil` dependency guards (`404`) or missing/wrong auth headers (`401`/`403`) while never executing the actual handler body, payload validation, or business logic.
- **Required**: Every test suite must exercise real happy-path (`200`/`201`), input validation (`400`/`422`), domain state transition conflict (`409`), and concurrency/idempotency behavior against real or in-memory/emulator backing stores.

### Law 0.4 — Never Duplicate Shared Code to Dodge Build Graph Wiring
- **Forbidden**: Copy-pasting shared source files into deleted or duplicate directories (e.g., recreating `packages/mobile-ios-design/` instead of wiring `packages/mobile-ios-core/` and `packages/mobile-ios-kit/`, or duplicating shared Kotlin/TypeScript models across apps) to silence a compiler error.
- **Required**: Maintain a single canonical source of truth. Fix the build definition (`project.yml`, `Package.swift`, `settings.gradle.kts`, `tsconfig.json`, `go.work`) so all apps consume the canonical shared package.

### Law 0.5 — Never Commit Build Artifacts, Vendor Caches, or Local State
- **Forbidden**: Committing compiled binaries (`ai-worker`, `handoff-service`, `*.o`, `*.a`), language vendor/build caches (`.gem/`, `target/`, `node_modules/`, `.next/`, `out/`, `build/`, `.gradle/`, `*.tsbuildinfo`), or Terraform plans/state (`*.tfplan`, `*.tfstate`) into git.
- **Required**: Before every commit, inspect `git status --short` and `git diff --cached --stat`. If build outputs appear, untrack them and harden `.gitignore`.

---

## 1. System Architecture & Domain Mission

### 1.1. Single-Supplier Multi-Retailer (SSMR) Doctrine
PegasusX is an enterprise wholesale Fast-Moving Consumer Goods (FMCG) distribution engine engineered for Central Asia (Uzbekistan / Tashkent and regional hubs) with multi-region cell scalability.

Unlike open two-sided marketplaces, PegasusX enforces the **Single-Supplier Multi-Retailer (SSMR)** model (`package.json`):
- **Authoritative Supplier Core**: A single supplier organization controls the master product catalog, pricing tiers, packaging units, factories, distribution warehouses, and delivery fleet.
- **Retailer Network**: Independent retail stores order via mobile apps, desktop POS terminals, and automated replenishment schedules.
- **End-to-End Custody & Money Chain**:
  1. **Factory (`apps/backend-go/factoryroutes/`)**: Production line scheduling, batching, palletization, and inter-warehouse transfers.
  2. **Warehouse (`apps/backend-go/warehouseroutes/`, `apps/backend-go/warehouseops/`)**: Inbound receiving, FEFO lot tracking, cold-chain monitoring, CRDT offline inventory adjustments, and wave picking.
  3. **Payload / Dock (`apps/backend-go/payloaderoutes/`)**: 3D truck bin-packing, axle load validation, and tamper-evident seal verification.
  4. **Dispatch & Driver (`apps/backend-go/driverroutes/`, `services/optimizer-core/`)**: OR-Tools / Rust VRP routing, OSRM/Google geometry, and real-time GPS telemetry.
  5. **Handoff & EPOD (`packages/handoff/engine.go`, `apps/backend-go/orderroutes/`)**: Cryptographic SHA-256/HMAC QR proof-of-delivery handoff tokens and geofence verification.
  6. **Financial Settlement & Fiscalization (`apps/backend-go/payment/`, `apps/backend-go/ar/`, `apps/backend-go/soliq/`)**: Integer minor-unit (Tiyin) ledger accounting, Accounts Receivable (AR) credit & dunning, and mandatory Uzbekistan State Tax Committee (**Soliq** EHF/OFD) fiscal receipts under ADR-009.

### 1.2. Authoritative Technology Stack (Do Not Hallucinate Foreign Stacks)
- **Primary Database**: **Google Cloud Spanner** (`cloud.google.com/go/spanner`) with authoritative schema in `apps/backend-go/schema/spanner.ddl` and incremental migrations in `apps/backend-go/schema/migrations/`. *(Note: Do NOT introduce PostgreSQL/`pgx`/`timescaledb` idioms into `apps/backend-go`—Spanner is the sole transactional store.)*
- **Cache & Realtime Backplane**: **Redis** (Cloud Memorystore in GCP) for cache-aside reads (`cache:invalidate` post-commit) and fail-open WebSocket Pub/Sub fanout (`ws:<hub>:fanout`).
- **Event Streaming**: **Apache Kafka** (GCP Managed Service for Apache Kafka in staging/prod) fed exclusively via the Spanner transactional `OutboxEvents` relay (`apps/backend-go/outbox/relay.go`).
- **Backend & Workers**: Go 1.25+ (`apps/backend-go`, `apps/ai-worker`, `apps/handoff-service`), Python OR-Tools + Rust CP-SAT/VRP (`services/optimizer-core`).
- **Web & Desktop Portals**: Next.js + React + Tailwind + Tauri v2 (`apps/retailer-portal`, `apps/supplier-portal`, `apps/warehouse-portal`, `apps/factory-portal`, `apps/admin-portal`).
- **Mobile Fleet**: Native Kotlin + Jetpack Compose (`apps/*-app-android`) and Native Swift + SwiftUI (`apps/*-app-ios`).

---

## 2. Production Invariants & Automated Enforcement Gates

Every architectural rule is paired with an executable gate script. All code must satisfy both the spirit of the invariant and the mechanical gate:

### Invariant 1: Zero Orphaned UI Placeholders
- **Rule**: Client screens, ViewModels, and backend handlers must never contain `TODO: Inject`, mock toggles, or dead buttons.
- **Gate**: `scripts/ci_fail_todo_inject.sh` (`rg -e 'TODO: Inject' apps packages`).

### Invariant 2: Immutable Production Container Digests & Service Separation
- **Rule**: Production Kubernetes overlays (`infra/k8s/overlays/prod`) must never contain `IMAGE_PLACEHOLDER`, `:latest`, `:local`, or `REPLACE_WITH_DIGEST`. `optimizer-core` must never be remapped onto the `backend-go` image, and `ai-worker-config` must never point to `pegasusx-local` or `kafka:9092`.
- **Gates**: `scripts/ci_fail_placeholder_images.sh`, `scripts/validate_backend_k8s.sh`, `scripts/validate_ai_worker_k8s.sh`, `scripts/validate_production_profile.sh`.

### Invariant 3: Zero Mock Data in Retailer Clients
- **Rule**: Production client builds must never contain simulator strings (`Mock Data`, `hardcoded BarMark`, `fakeH3Pulse`, `CONTROL_TOWER_SIMULATOR`).
- **Gate**: `scripts/ci_no_mock_control_tower.sh`.

### Invariant 4: Inviolable Money-Path & Fiscal Correctness
- **Rule**:
  1. All monetary values must use `int64` minor units (Uzbek Tiyin: `1 UZS = 100 Tiyin`). `float32`/`float64` for money is strictly forbidden.
  2. Failed payment captures must never transition an order to `CAPTURED`.
  3. Duplicate client idempotency keys must never create duplicate ledger rows or double captures.
  4. Missing payment or fiscal credentials in production must fail closed immediately—never silently succeed.
  5. Shop-closed credit deliveries must atomically record debt in the AR ledger.
- **Gate**: `scripts/money_path_gate.sh` (runs `TestMoneyPathGate` and `TestWorkerShopClosed` against the Spanner emulator).

### Invariant 5: Strict Multi-Region Cell Isolation
- **Rule**: Secondary regional cells (e.g., `cell-eu`) must never reference or overwrite the primary `pegasusx/ssmr` Terraform state prefix or restore foreign cell Spanner backups.
- **Gate**: `scripts/assert_cell_backend.sh`.

### Invariant 6: Solver Mathematical Honesty
- **Rule**: Routing and scheduling solvers in `services/optimizer-core/` must report their true mathematical status (`SolverStatus.HEURISTIC` for greedy/local-search heuristics; `SolverStatus.OPTIMAL` only when proven by an exact solver).
- **Gate**: `scripts/ci_optimizer_core.sh`.

### Invariant 7: 3-Way Event Schema Lockstep (`-strict=true`)
- **Rule**: Every domain event change must atomically update:
  1. Go event structs & constants (`apps/backend-go/events/events.go`)
  2. JSON Schema (`contracts/events.schema.json` with `"additionalProperties": false` on all nested objects)
  3. Generated TypeScript, Swift, and Kotlin contracts (`make gen-contracts`)
- **Gate**: `make gen-contracts-gate` and `go test ./events/... -args -strict=true`.

### Invariant 8: Spanner Transactional Outbox & Schema Lockstep
- **Rule**:
  1. Every schema change in `apps/backend-go/schema/migrations/*.sql` must be reflected identically in `apps/backend-go/schema/spanner.ddl`.
  2. HTTP handlers must never publish directly to Kafka; all domain events must be buffered into `OutboxEvents` inside the same `spanner.ReadWriteTransaction`.
  3. Redis cache invalidation must occur only after the Spanner transaction commits, and WebSocket broadcasts over Redis Pub/Sub must be fail-open.
- **Gate**: `scripts/ci_schema_drift_gate.sh`.

### Invariant 9: 360° Cross-Platform & 6-Role Parity
- **Rule**: 100% of `/v1/...` routes called by any Web, Desktop, Android, or iOS client across all 6 roles (Retailer, Driver, Warehouse, Supplier, Factory, Platform Admin) must map to registered routes in `apps/backend-go`, and active Terraform monitoring (`infra/terraform/modules/monitoring/main.tf`) must enforce all platform SLO alerts.
- **Gates**: `scripts/parity/role_row_contract_check_full.sh`, `scripts/parity/gap_hunter_gate.sh`, `scripts/phase2_gate.sh` through `scripts/phase5c_gate.sh`.

---

## 3. Mandatory Engineering Workflow (Before & After Every Edit)

### Step 1: Pre-Edit Blast-Radius & Root-Cause Analysis
1. **Locate the Canonical Source**: Inspect `go.work`, `pnpm-workspace.yaml`, `Package.swift`, or `settings.gradle.kts` before creating any file. Never create a duplicate model, helper, or package directory if one already exists.
2. **Check Concurrency & Failure Modes**:
   - In Go: verify `context.Context` propagation, `defer iter.Stop()` on every Spanner `RowIterator`, transaction retry safety inside `spanner.ReadWriteTransaction` (no non-idempotent side effects inside the transaction closure!), and mutex/map safety under `-race`.
   - In Infra/K8s: verify that changes in `infra/k8s/base` or `infra/terraform/modules` are properly wired into `overlays/prod`, `overlays/staging`, and `overlays/pilot`.

### Step 2: Post-Edit Verification Checklist
Never claim a task is complete without running the relevant verification gates and reporting their exact output:

```bash
# 1. Static, Schema, K8s & Contract Gates (Always runnable locally)
make schema-drift-gate
make parity-contract-full
make gap-hunter-gate
make validate-ai-worker-k8s
make validate-backend-k8s
make validate-launch-readiness
terraform fmt -check -recursive infra/terraform && (cd infra/terraform && terraform validate)

# 2. Go Backend & Worker Race-Detector Suite
(cd apps/backend-go && go test -race -short ./...)
(cd apps/ai-worker && go test -race ./...)

# 3. Optimizer Core (Python + Rust)
bash scripts/ci_optimizer_core.sh

# 4. Full Spanner Emulator & Enterprise Gates (Requires Docker / Spanner Emulator on :9010)
make sandbox-infra-up
make money-path-gate
make ci-enterprise-gates
```
