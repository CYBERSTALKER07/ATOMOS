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

### 1.1. Single-Supplier Multi-Retailer (SSMR) & 9-Role Ecosystem Doctrine
PegasusX is an enterprise wholesale Fast-Moving Consumer Goods (FMCG) distribution and commercial operating system engineered for Central Asia (Uzbekistan / Tashkent and regional hubs) with multi-region cell scalability (`cell-uz`, `cell-eu`, `cell-kz`).

Unlike open two-sided marketplaces, PegasusX enforces the **Single-Supplier Multi-Retailer (SSMR)** model across **9 Operational Roles** and a **Palantir Foundry-Grade Semantic-Kinetic-Dynamic Architecture**:
- **Role 1 — Supplier / Brand Manufacturer (`apps/backend-go/supplierroutes/`, `apps/backend-go/pricing/`, `apps/backend-go/promotion/`, `apps/backend-go/payout/`)**: Commercial HQ, GTIN/GS1 & GlobalProducts catalog sovereignty, multi-tier price matrix & promo contract budgets, bilateral trade credit underwriting & dunning ladder, reverse factoring / immutable `InvoiceSettlementSlices` payouts, and S&OP demand forecasting.
- **Role 2 — Factory (`apps/backend-go/factoryroutes/`)**: BOM explosion, raw material lot locking, production line scheduling, in-line quality control, SSCC-18 pallet serialization, and inter-facility FTL transfers.
- **Role 3 — Warehouse (`apps/backend-go/warehouseroutes/`, `apps/backend-go/warehouseops/`, `apps/backend-go/stocklots/`)**: Regional DC inbound receiving, ABC velocity putaway, FEFO lot rotation, cold-chain monitoring, CRDT offline inventory adjustments, dynamic wave picking, and probabilistic cycle counts.
- **Role 4 — Payloader (`apps/backend-go/payloaderoutes/`)**: Reverse-stop LIFO loading, 3D volumetric & crushability bin-packing, front/rear axle weight scaling, returnable transport item (RTI) reverse-space accounting, and cryptographic bolt seal binding.
- **Role 5 — Driver (`apps/backend-go/driverroutes/`, `services/optimizer-core/`)**: Mandatory pre-trip DVIR clearance, dynamic CVRPTW turn-by-turn routing, 150m/50m doorstep geofence verification, cryptographic ePOD handshake, split-tender collection, and mid-route fleet rescue custody transfers.
- **Role 6 — Retailer (`apps/backend-go/retailerroutes/`)**: Store POS, Shelf AI gap detection, autonomous S&OP replenishment cart, multi-supplier `ParentOrders` checkout saga, stock receiving sessions, and 48-hour concealed damage claims tribunal.
- **Role 7 — Depot Cashier & Vault Ops (`apps/backend-go/cashreconroutes/`, `apps/backend-go/foundry/`)**: Tamper-evident cash bag ingestion, Dual-Blind Denomination Counting Ritual (expected total hidden until recount ratchet completes), RRN-verified tender reclassification, Perpetual Depot Vault Ledger, CIT armored bank courier handover, and uncapped multi-tender settlement tracking aggregated across Legal Entity `TIN/PINFL`.
- **Role 8 — Platform Admin & Global Control Tower (`apps/backend-go/platformadmin/`, `apps/backend-go/controltowerroutes/`)**: Multi-region cell federation & zero cross-cell leakage, declarative Market Packs (zero hardcoded currency literals), Soliq EHF/OFD PKCS#12 fiscal gateway, immutable `AuditEvents` & SHA-256 `EvidenceDossiers`.
- **Role 9 — Fleet, Yard & Maintenance Manager (`apps/backend-go/foundry/`, `apps/backend-go/driverroutes/`)**: ALPR gate check-in/out, dynamic yard bay slotting (`INBOUND_STAGING`, `CROSS_DOCK`, `OUTBOUND_LOADING`, `MECHANICAL_SERVICE`), mandatory 8-point Pre-Trip DVIR walkaround, automatic **Red-Tag Grounding Gate** (`OUT_OF_SERVICE`), telemetry-linked predictive maintenance holds, and post-trip preventative servicing.

### 1.1.1. Foundry-Grade Semantic-Kinetic-Dynamic Triad (Pre-Execution Architecture)
Every execution across PegasusX operates on three foundational layers (`apps/backend-go/foundry/` & `docs/roles/FOUNDRY_ONTOLOGY_AND_EDGE_CASES.md`):
1. **Semantic Layer (The Ontology Graph)**: Every domain entity (`RawMaterialLot`, `WorkOrder`, `SSCCPallet`, `PickWave`, `Tote`, `Vehicle`, `RouteManifest`, `Order`, `CashBag`, `InvoiceSettlementSlice`, `SoliqFactura`) is indexed as a strongly-typed Ontology Object connected by directed `OntologyGraphEdges` with propagating `SecurityMarkings` (`MARKING:AML_INVESTIGATION`, `MARKING:DISCREPANT_QUARANTINE`, `MARKING:THERMAL_QUARANTINE_SPOILED`), enabling millisecond multi-hop `SearchAround` blast-radius traversals.
2. **Kinetic Layer (Pre-Execution Action Engine)**: State transitions execute through validated **Kinetic Action Contracts** that enforce deterministic pre-condition rubrics, multi-party approvals, and atomic audit diff logging before committing to Cloud Spanner.
3. **Dynamic Layer (Copy-on-Write Scenario Sandbox / Digital Twin)**: Planners and operators fork live state into isolated `ScenarioId` overlays (`ResolvedState = BaseSpannerState ⊕ ScenarioPatches - ScenarioDeletions`), run deterministic solvers without triggering external fiscal/SMS side effects, and commit back to production via optimistic concurrency watermark checks (`BaseVersion`).

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

### Invariant 9: 360° Cross-Platform & 9-Role Parity
- **Rule**: 100% of `/v1/...` routes called by any Web, Desktop, Android, or iOS client across all 9 operational roles (Supplier, Factory, Warehouse, Payloader, Driver, Retailer, Depot Cashier, Platform Admin, Fleet & Yard Manager) must map to registered routes in `apps/backend-go`, and active Terraform monitoring (`infra/terraform/modules/monitoring/main.tf`) must enforce all platform SLO alerts.
- **Gates**: `scripts/parity/role_row_contract_check_full.sh`, `scripts/parity/gap_hunter_gate.sh`, `scripts/phase2_gate.sh` through `scripts/phase5c_gate.sh`.

### Invariant 10: Mandatory Pre-Execution Kinetic & 12 Edge-Case Guards
- **Rule**: Before any domain mutation executes across the 10-step operational cascade, the backend must evaluate and enforce the 12 cross-role edge-case pre-execution guards (`apps/backend-go/foundry/` & `docs/roles/FOUNDRY_ONTOLOGY_AND_EDGE_CASES.md`):
  1. **Split-Tender Doorstep Partial Rejection**: Fault-attributed volume-break tier protection (`DAMAGED_IN_TRANSIT`/`SHORT_SHIPPED` preserves original volume discount tier; `BUYER_REMORSE_REFUSAL` recalculates tier) + deterministic tender reduction waterfall (reduces `CREDIT` draw before `CASH`).
  2. **Multi-Tender Tracking & Uncapped Policy**: Aggregate daily settlements across the parent Legal Entity `TIN/PINFL` + `SupplierId` for transparent audit logging under an uncapped payment and credit policy (no artificial ceilings or credit limit blocks).
  3. **Two-Phase Soliq Fiscal Pre-Flight & Payout Gate**: Block pick-wave release if any SKU lacks a valid IKPU/MXIK code or if the Supplier's `.p12` signing certificate expires within `< 72h`; block `InvoiceSettlementSlices` from `BATCHED` unless `FiscalState == REGISTERED_SOLIQ`.
  4. **Mid-Route Breakdown Rescue & Cold-Chain Thermal Quarantine**: Split driver cash liability at the exact handover stop (`1..K` vs `K+1..N`), quarantine chilled/frozen totes if thermal excursion exceeds tolerance, and bind the new rescue bolt seal.
  5. **Zero-Destructive RRN Tender Reclassification**: Require a verified gateway RRN within `±15m` of the ePOD timestamp to post compensating ledger entries when reclassifying Cash to Card.
  6. **3D Volumetric Crushability & Returnable Container (`RTI`) Space Accounting**: Enforce `Ti/Hi` crushability rules and track reverse-logistics empty crate/pallet volume and deposits in `ReturnableContainerLedger`.
  7. **Scenario Optimistic Concurrency Watermark**: Verify `BaseVersion` on every object touched by a Copy-on-Write Scenario before committing to production; surface structured merge conflicts on drift.
  8. **Post-Settlement Clawback Slices & Rolling Reserve Escrow**: Never mutate `BATCHED` settlement slices; append negative `CONCEALED_DAMAGE_CLAIM_APPROVED` slices backed by rolling reserve holdbacks.
  9. **FX Volatility Corridor & Wave Price Freeze**: Enforce `LOCK_AT_CHECKOUT` vs `FLOAT_WITH_CORRIDOR_CAP` before wave release and hard-freeze minor unit prices once a pick wave is released.
  10. **Abandoned Pick-Wave Tote Heartbeat Takeover**: Detect `PICKER_IN_HAND` totes stalled `> 20m` without scan activity and allow atomic picker takeover without double-picking FEFO lots.
  11. **Dual-Blind Cashier Recount Ratchet & Tolerance-Skimming Detection**: Never expose `ExpectedCashMinor` to the Depot Cashier during input or first variance; require a blind Recount #2 before supervisor reveal, and flag systematic micro-skimming patterns.
  12. **Pre-Trip 8-Point DVIR Red-Tag Grounding Gate**: Automatically transition any vehicle with a critical DVIR defect to `OUT_OF_SERVICE`, uncouple driver-vehicle pairing, block Payloader/Dispatch selection, and issue a workshop maintenance order.
- **Gate**: `(cd apps/backend-go && go test -race ./foundry/...)` and `make schema-drift-gate`.

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
