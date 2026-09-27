# PegasusX Autonomous Agent Instructions & Operating Doctrine

> **Scope**: PegasusX Enterprise Logistics, Supply Chain & Distribution Ecosystem  
> **Repository Root**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX`  
> **Workspace Configuration**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/go.work`, `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/pnpm-workspace.yaml`, `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/package.json`  
> **Primary Operating Mode**: Zero Theatre, Absolute Code Grounding, Production Invariants

---

## 1. System Goal & Mission

### 1.1. Single-Supplier Multi-Retailer (SSMR) Doctrine
PegasusX is the enterprise logistics distribution engine engineered specifically for wholesale Fast-Moving Consumer Goods (FMCG) distribution in Central Asia (Uzbekistan / Tashkent metropolitan area and regional hubs), designed with multi-region cell scalability.

Unlike two-sided open marketplace architectures where disparate sellers compete for consumer carts, PegasusX implements the **Single-Supplier Multi-Retailer (SSMR)** doctrine (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/package.json:5`):
- **Authoritative Supplier Core**: A unified supplier organization commands the catalog, master pricing policies, bulk packaging units, warehouse distribution centers, factory manufacturing transfers, and company truck fleet.
- **Retailer Network**: Hundreds to thousands of independent retail stores (convenience stores, supermarkets, neighborhood grocers) interface with the supplier via dedicated ordering apps, desktop POS stations, scheduled replenishment feeds, and automated preorders.
- **Physical Logistics Chain**: The system manages every operational handoff across physical custody boundaries:
  1. Factory manufacturing, palletization, and regional inter-warehouse transfers (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/factoryroutes/routes.go`).
  2. Warehouse inbound receiving, lot tracking, cold-chain temperature thresholds, and batch wave picking (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/warehouseroutes/routes.go`).
  3. Dock loading, truck capacity bin-packing, tamper-evident security seal validation, and order injection (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/payloaderoutes/routes.go`).
  4. Turn-by-turn fleet dispatch, OSRM route geometry, and background driver GPS telemetry (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/driverroutes/routes.go`, `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/telemetryroutes/routes.go`).
  5. Cryptographically authenticated electronic proof of delivery (EPOD), dynamic HMAC QR handoff tokens (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/packages/handoff/engine.go:1-80`), and proximity geofencing.
  6. Financial settlement, integer Tiyin cash collection reconciliation, Accounts Receivable (AR) dunning, and mandatory fiscal registration with the State Tax Committee of Uzbekistan (**Soliq** EHF electronic invoices) under ADR-009 (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/soliq/client.go:1-60`).

---

## 2. Honesty Commandments ("Zero Theatre")

All autonomous agents and human engineers contributing to PegasusX are bound by the **Zero Theatre Doctrine**. Software must function genuinely against real database transactions, real message brokers, real WebSocket rooms, and genuine solver mathematics. Mock facades, stubbed screens, and fake data are treated as system faults.

### Commandment 1: Zero Tolerance for "TODO: Inject" Placeholders
- **Rule**: Client screens, ViewModels, and backend handlers must never carry `TODO: Inject` markers. Such placeholders signify orphaned UI components or fake implementations that silently fail to perform work.
- **Enforcement**: Automated CI gate `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/ci_fail_todo_inject.sh:1-12` scans all files in `apps/` and `packages/` using ripgrep (`rg -e 'TODO: Inject'`). If found, the build fails immediately.
- **Directive**: Either wire the genuine dependency to the live API client or delete the screen entirely.

### Commandment 2: Zero Placeholder or `:latest` Images in Production Overlays
- **Rule**: Production Kubernetes deployment manifests must never contain `IMAGE_PLACEHOLDER`, `:latest`, `:local`, or unpinned digest tags. Furthermore, `optimizer-core` must NEVER be remapped onto `backend-go` (a dangerous crash-loop risk).
- **Enforcement**: Gate `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/ci_fail_placeholder_images.sh:1-44` renders the production Kustomize overlay (`infra/k8s/overlays/prod`) and verifies image immutability and service separation (`lines 26-38`).

### Commandment 3: Zero Mock Strings in Retailer Clients
- **Rule**: Retailer desktop and mobile production builds must never contain demo strings such as `Mock Data`, `hardcoded BarMark`, `fakeH3Pulse`, or `CONTROL_TOWER_SIMULATOR`.
- **Enforcement**: Gate `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/ci_no_mock_control_tower.sh:1-15` audits `apps/retailer-app-desktop`, `apps/retailer-app-android`, and `apps/retailer-app-ios`.

### Commandment 4: Inviolable Money Path Correctness
- **Rule**: Financial transactions must never report false successes or risk duplicate captures. Specifically:
  1. Payment capture failures must never transition order status to `CAPTURED`.
  2. Duplicate client idempotency keys must never create duplicate ledger entries.
  3. Empty payment gateway credentials must produce immediate hard errors, never silent bypasses.
  4. Shop-closed credit debt must always be committed to the immutable AR ledger.
- **Enforcement**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/money_path_gate.sh:1-56` executes integration test suites `TestMoneyPathGate` and `TestWorkerShopClosed` against a live Google Cloud Spanner emulator (`apps/backend-go/order/`, `apps/backend-go/payment/`).

### Commandment 5: Strict Multi-Region Cell Isolation
- **Rule**: Secondary expansion cells (e.g., `europe-west1` / `cell-eu`) must NEVER be able to open or overwrite the primary live SSMR state prefix (`pegasusx/ssmr`). Workload identity namespaces must be dynamically derived per cell (`local.k8s_namespace`). Restoring an Uzbekistan Spanner backup onto a foreign cell is strictly prohibited.
- **Enforcement**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/assert_cell_backend.sh:1-156` inspects Terraform backend files, `infra/terraform/cell.tf`, and K8s overlays to verify complete cryptographic and state separation.

### Commandment 6: Solver Honesty
- **Rule**: Routing and capacity solvers (`services/optimizer-core/`) must strictly report their true mathematical state. Heuristic approximations (e.g. nearest-neighbor or greedy algorithms) MUST return `SolverStatus.HEURISTIC` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/services/optimizer-core/proto/optimizer_core.proto:14-16`). Never claim `OPTIMAL` unless mathematically proven by an exact solver (e.g. OR-Tools branch-and-bound / CP-SAT).

### Commandment 7: 3-Way Schema Synchronization Lockstep
- **Rule**: Whenever a domain event is added or modified, all three authoritative definitions must be updated simultaneously in the same atomic commit:
  1. Go backend constants: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/events/events.go:33-360`
  2. JSON Schema contract: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/contracts/events.schema.json`
  3. TypeScript types: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/packages/types/src/events.ts`
- **Enforcement**: CI target `make gen-contracts-gate` validates contract parity.

---

## 3. Core Architectural Constraints

Every engineer and AI agent operating in PegasusX must comply with these six architectural invariants:

### 3.1. Cloud Spanner as Authoritative Ledger
- The primary storage engine is Google Cloud Spanner, defined in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/schema/spanner.ddl` (220+ tables) and evolved through 125 migrations in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/schema/migrations/`.
- Every mutation must execute inside a `spanner.ReadWriteTransaction`.
- Schema drift is prohibited: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/ci_schema_drift_gate.sh:1-43` verifies that every migration table and column is reflected in `spanner.ddl`.

### 3.2. Atomic Transactional Outbox Pattern
- **Rule**: Direct publishing to Apache Kafka from HTTP request handlers is strictly forbidden.
- **Mechanism**: Every domain state mutation (e.g., order creation, manifest sealing, delivery confirmation) writes an event row into the `OutboxEvents` table (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/schema/spanner.ddl:685-702`) within the SAME Spanner ReadWrite transaction.
- **Relay Loop**: The background outbox relay worker (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/outbox/relay.go:14-65`) polls `OutboxEvents` every **250ms**, fetches batches of **100 events**, publishes them to Kafka with exponential backoff and jitter, and marks them published.
- **Dead-Letter Budget**: Events that fail more than 5 times in a single tick back off; if total attempts reach **20**, the event is moved to `OutboxDeadLetters` (`spanner.ddl:704-718`).

### 3.3. Post-Commit Invalidation & Fail-Open WebSocket Fanout
- **Post-Commit Invalidation**: Redis cache keys (`cache:invalidate`) must only be invalidated AFTER the Spanner transaction commits successfully (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/runtime_workers.go:34`).
- **Fail-Open Realtime**: WebSocket broadcasts over Redis Pub/Sub must be fail-open. A Redis relay glitch or network drop must log an error and increment a metric, but MUST NEVER fail the HTTP handler or crash the pod (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/ws/hub.go:10-13, 358-365`).

### 3.4. Source Suppression & Monotonic Reconnection Buffering
- **8 Dedicated WebSocket Hubs**: Constructed in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/ws/hub.go` and mounted in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/main.go:417-433`:
  `RetailerHub`, `SupplierHub`, `DriverHub`, `PayloadHub`, `WarehouseHub`, `FactoryHub`, `TelemetryHub`, `PlatformAdminHub`.
- **Source Suppression**: Senders attach an instance identifier in the relay envelope (`relayEnvelope{Source, Room, Payload}`). When a pod receives a broadcast from Redis on `ws:<hub>:fanout`, it drops the message if `Source == h.instance` (`ws/hub.go:395-397`), preventing infinite reflection loops.
- **Reconnect Ring Buffer**: Each hub maintains an in-memory ring buffer of the last **256 events** per room. Reconnecting clients pass `lastEventID` or `sinceSeq` to trigger `ReplaySince` (`ws/hub.go:310-341`), preventing dropped UI state during brief mobile handovers.

### 3.5. Integer Minor Currency Invariant
- Floating-point representations (`float32`, `float64`, `double`) are strictly prohibited for monetary values.
- All pricing, totals, payments, ledger debits/credits, refunds, and dunning balances are stored in **integer minor units** (e.g., Uzbek Tiyin: 1 UZS = 100 Tiyin) via `INT64 TotalMinor` in Spanner (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/schema/spanner.ddl:184-185, 580, 665`).

### 3.6. Full Role-Row Route Parity
- PegasusX guarantees that 100% of client API calls across all 6 role-rows map to real registered backend routes.
- **Parity Gate**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/parity/role_row_contract_check_full.sh:1-60` extracts all `/v1/...` API client strings from every web, desktop, and mobile app and verifies their presence in `apps/backend-go` route registrations. Orphaned endpoints fail the build.

---

## 4. Development & Verification Workflows

Before committing changes or submitting a pull request, execute the following standardized Makefile verification steps:

```bash
# 1. Local Infrastructure Bootstrap
make sandbox-infra-up        # Starts Spanner emulator, Redis, Kafka, Kafka-UI, optimizer-core
make test-sandbox-infra      # Executes full sandbox smoke test + PX_E2E marker gate

# 2. Local Unit & Contract Verification
make qa-gate                 # Runs all Go unit tests with race detection
make parity-contract-full    # Validates 100% client-to-backend route alignment
make gap-hunter-gate         # Audits event producer and consumer payload alignment

# 3. Enterprise & Safety Gates
make money-path-gate         # Validates payment capture idempotency and credit invariants
make schema-drift-gate       # Validates Spanner DDL against all migrations
make cell-backend-guard      # Verifies Terraform cell state isolation
make ci-enterprise-gates     # Chains Phase 2 through Phase 5c enterprise test gates

# 4. Pre-flight & Launch Validation
make wire-ready              # Comprehensive automated verification before GCP deployment
make p0-preflight            # Validates production profile, K8s manifests, and credentials
```

### Key Verification Scripts Index
- `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/smoke_sandbox.sh` — Sandbox infrastructure smoke testing.
- `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/smoke_sandbox_lifecycle.sh` — Full lifecycle vertical (order create → dispatch → load → seal → arrive → collect → fiscalize).
- `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/smoke_sandbox_fiscal.sh` — ADR-009 Soliq fiscal hard-gate testing.
- `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/ci_fail_todo_inject.sh` — Rejects orphaned UI placeholders.
- `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/ci_fail_placeholder_images.sh` — Rejects `:latest` or unpinned container images.
- `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/ci_no_mock_control_tower.sh` — Rejects fake simulation data in retailer apps.
- `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/money_path_gate.sh` — Rejects double-charge or unrecorded credit flaws.
- `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/assert_cell_backend.sh` — Rejects multi-region cell leakage.


## Core Engineering Doctrine (Applied from Master)

### Zero-Tolerance for Naive CRUD
- **Naive CRUD is strictly forbidden**: Never write dumb table mutations that lack domain state machines, concurrency checks, transactional outbox events, validation guards, or comprehensive telemetry.
- **Production-Grade Rigor**: Every feature, backend service, database table, UI view, and pipeline must be built with production depth, resilience, and mathematical elegance.
- **Fail-Closed & Defensive Architecture**: Systems must fail safely, recover gracefully, prevent cascade failures, and handle Byzantine faults, network drops, and corrupted inputs without crashing or corrupting state.

### The Three-Tier Sourcing Hierarchy (Best Practices & Algorithms)
1. **Tier 1 — Battle-Tested Big Tech & Open-Source Tools**:
   - If an industry-standard, high-performance open-source library, tool, or framework exists and fits the target stack (`pgx/v5`, `go-chi/chi/v5`, `redis-go`, `uber-go/zap`, `pydantic-v2`, `zod`, `tailwind v4`, `timescaledb`), evaluate, select, and integrate it rather than rolling naive custom implementations.
2. **Tier 2 — Algorithmic Reverse-Engineering & Native Adaptation**:
   - If no direct open-source package or cloud tool fits our sovereign single-tenant or global multi-tenant stack, dissect and extract the mathematical models, algorithmic logic, data structures, and state machines from world-class systems (Google OR-Tools CVRP, Maglev consistent hashing, Uber H3 spatial indexing, Stripe Idempotency, Amazon Shuffle Sharding, Netflix Concurrency Limiters) and implement them natively at enterprise grade.
3. **Tier 3 — First-Principles Novel Engineering**:
   - If no algorithm, library, or tool exists anywhere in the world to solve our exact problem, design a novel, mathematically sound, enterprise-grade algorithm from first principles. Formalize its invariants, prove its boundary conditions, test its edge cases, and implement it with zero compromises.

---

## 2. Mandatory Dual-Domain Pre-Edit & Post-Edit Brainstorming Protocol

Every code edit—regardless of size—must pass through two non-bypassable verification gates:

### Gate A: Pre-Edit Research & Brainstorming (BEFORE touching code)
1. **Mandatory Web & Codebase Research**:
   - Proactively search the web for industry gold standards, RFCs, statutory specifications, algorithmic benchmarks, battle-tested open-source libraries, and real-world failure postmortems.
   - Search the codebase using CodeGraph and Kythe to inspect existing patterns, data structures, and shared types.
2. **Dual-Domain Brainstorming (Technical + Non-Technical)**:
   - **Technical Edge Cases**:
     - Concurrency races, goroutine leaks, deadlocks, connection pool starvation (`pgxpool`).
     - Time-of-Check to Time-of-Use (TOCTOU) windows, reentrancy vulnerabilities.
     - Out-of-order event delivery, slow consumer head-of-line blocking, backpressure exhaustion.
     - Distributed partition hazards, split-brain conditions, clock skew, retry storms.
     - Memory bloat, memory leak under heavy continuous streaming, unhandled context cancellations.
   - **Non-Technical & Physical Logistics Edge Cases**:
     - Real-world road conditions, mountainous/rural terrain, cellular dropouts (offline-first sync).
     - Mid-shift vehicle breakdowns, dynamic hot-swap rescue transfers, driver shift timeouts.
     - Doorstep freight damage disputes, itemized returns, partial cancellations, carton rejections.
     - Uzbekistan statutory tax compliance: Soliq 12% VAT integer rounding discrepancies, STIR/MXIK commodity validations.
     - B2B cash payment transaction rules, corporate card webhook latency, driver cash drawer reconciliation.
     - Cargo physical properties: longitudinal axle weight static moments (11.5T single axle, 20% steer traction), temperature deviations in cold-chain reefers.

### Gate B: Post-Edit Verification & Pressure-Testing (AFTER editing code)
1. **Blast-Radius & Reverse Dependency Audit**:
   - Re-read every edit with Red Team adversarial eyes to identify unintentional contract drift, unhandled error branches, or regression vectors.
