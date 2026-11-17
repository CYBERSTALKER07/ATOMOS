# Pegasus Ecosystem AI Agent Governance & Engineering Instructions

> **Canonical System Mission**: The **Pegasus** platform is a hyperscale multi-tenant, multi-supplier logistics execution engine, predictive commerce operating system, and wholesale marketplace. It coordinates suppliers, regional fulfillment warehouses, manufacturing factories, independent retail shops, and commercial delivery fleets across complex regional networks.
>
> **Target Path**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/agents.md`  
> **Ecosystem Root**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus`

---

## 1. Monorepo Topology & Workspace Conventions

Every autonomous agent or human engineer working within `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus` must strictly adhere to the monorepo workspace boundaries:

- **Go 1.25 Multi-Module Workspace**: Declared at `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/go.work`. Includes 7 coordinated Go modules:
  - `apps/backend-go` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/go.mod`) — Core REST API, Chi router, WebSocket servers, Spanner client, and Outbox publisher.
  - `apps/ai-worker` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/ai-worker/go.mod`) — Empathy Engine, SKU demand forecasting, and Clarke-Wright dispatch heuristic.
  - `packages/config` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/config/go.mod`) — Fail-closed environment parser and runtime validator.
  - `packages/optimizer-contract` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/optimizer-contract/go.mod`) — Typed solver contracts for VRP and CP-SAT requests.
  - `packages/ai-bridge` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/ai-bridge/go.mod`) — Google Gemini LLM bridge for zero-shot catalog/invoice schema mapping.
  - `services/optimizer-core/adapters/go` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/optimizer-core/adapters/go/go.mod`) — Kafka-to-gRPC adapter tunneling jobs to the Rust solver.
  - `adyen-go-api-library-main` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/adyen-go-api-library-main/go.mod`) — Localized Adyen API payment client.
- **18 Application Subsystems (`apps/`)**:
  - `apps/backend-go`: High-concurrency Chi backend (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go`).
  - `apps/ai-worker`: Predictive forecast and bulk import worker (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/ai-worker`).
  - `apps/admin-portal`: Supplier & Global Admin Portal in Next.js 15, React 19, and Tauri 2 (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/admin-portal`).
  - `apps/factory-portal`: Factory production and replenishment desktop portal (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/factory-portal`).
  - `apps/warehouse-portal`: Warehouse intake, staging, and dispatch lock portal (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-portal`).
  - `apps/retailer-app-desktop`: Retailer POS terminal and order management portal (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-desktop`).
  - `apps/payload-terminal`: Expo 55 / React Native 0.83 dock barcode scanning terminal (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/payload-terminal`).
  - 5 Native Android Apps (Kotlin 2.x, Jetpack Compose, Hilt, Room): `driver-app-android` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/driver-app-android`), `retailer-app-android` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-android`), `factory-app-android` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/factory-app-android`), `warehouse-app-android` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-app-android`), `payload-app-android` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/payload-app-android`).
  - 5 Native iOS Apps (Swift 6, SwiftUI, SwiftData, CoreLocation): `driverappios` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/driverappios`), `retailer-app-ios` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-ios`), `factory-app-ios` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/factory-app-ios`), `warehouse-app-ios` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-app-ios`), `payload-app-ios` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/payload-app-ios`).
  - `apps/synthetic-tester`: High-throughput locust-style load generation suite (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/synthetic-tester`).
- **2 Autonomous Sidecars (`services/`)**:
  - `services/optimizer-core`: Tonic/Prost Rust gRPC sidecar executing VRP and CP-SAT solvers (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/optimizer-core`).
  - `services/deep-agents`: LangGraph multi-agent auditing and reasoning fleet (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/deep-agents`).
- **8 Shared Packages (`packages/`)**:
  - `packages/ai-bridge`: Gemini provider connector (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/ai-bridge`).
  - `packages/api-client`: Cross-portal HTTP client with retry and error interceptors (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/api-client`).
  - `packages/config`: Go environment loader (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/config`).
  - `packages/i18n`: Type-safe translations for EN, RU, UZ-Latn, UZ-Cyrl, TR, AR (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/i18n`).
  - `packages/optimizer-contract`: Solver payload definitions (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/optimizer-contract`).
  - `packages/types`: Universal TypeScript interfaces and WebSocket event envelopes (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/types`).
  - `packages/ui-kit`: Material 3 design system tokens and Tailwind CSS presets (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/ui-kit`).
  - `packages/validation`: Runtime schema checks (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/validation`).

---

## 2. Mandatory Honesty Rules ("Zero Theatre")

The Pegasus engineering doctrine strictly enforces the **Zero Theatre Mandate**. Violations will be flagged by automated guardrails and forensic review:

1. **Zero Hallucinated Endpoints**: Never fabricate mock or placeholder HTTP endpoints. Every route referenced in client code or documentation must exist as a registered Chi handler mounted in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/main.go` or its subrouters.
2. **Zero Mock Data in Production UI**: Web portals and mobile apps must never render fake fallback JSON payloads when backend requests fail. Missing or degraded backend communication must trigger explicit UI error states or degraded indicator banners.
3. **No Direct Event Publishing Bypassing Outbox**: It is strictly forbidden to call Kafka producers directly from HTTP handlers to publish business state events. All durable domain events MUST be written to `OutboxEvents` within the same Spanner transaction as the entity mutation (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/outbox/relay.go`).
4. **No Schema Changes Without DDL & Migrations**: Spanner schema modifications must be accompanied by idempotent DDL statements in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/schema/spanner.ddl` and registered in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/migrations/migrations.go`.
5. **No TODO Placeholders in Core Critical Paths**: Do not leave stubbed handlers returning `http.StatusOK` with empty bodies or comments like `// TODO: implement real logic`. Core paths must either implement real logic or return structured HTTP 501 / HTTP 422 errors.

---

## 3. Core Architectural Constraints & Invariants

All agents must honor the fundamental architectural constraints implemented in the codebase:

### 3.1 Google Cloud Spanner (94 Tables)
The transactional datastore is Google Cloud Spanner, defined in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/schema/spanner.ddl` (lines 15–2374).
- The schema consists of exactly 94 production tables.
- All transactional modifications must execute inside Spanner `ReadWriteTransaction` blocks.
- Multi-tenancy is enforced through `SupplierId` partitioning on every catalog, fleet, manifest, and order query.
- Queries referencing historical rows must leverage commit timestamps (`allow_commit_timestamp=true`).

### 3.2 Transactional Outbox Pattern
To prevent dual-write inconsistency between Cloud Spanner and Apache Kafka:
- Events are persisted to `OutboxEvents` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/schema/spanner.ddl:2151`).
- The `outbox.Relay` background daemon (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/outbox/relay.go:58`) polls pending events, shards them across goroutines via `FNV32(AggregateID) % numShards` to preserve ordering, writes to Kafka, and marks them `PUBLISHED` in Spanner via batch mutations.
- Unrecoverable failures route to `OutboxDLQ` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/schema/spanner.ddl:2171`).

### 3.3 Double-Entry Treasury & Anomaly Detection
- Every financial movement is tracked as an immutable debit/credit row in `LedgerEntries` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/schema/spanner.ddl:290`).
- Background reconciliation cron `StartReconciliationCron` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/admin/audit_cron.go:13`) runs hourly, comparing captured gateway balances against Spanner expected totals.
- Discrepancies are recorded in `LedgerAnomalies` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/schema/spanner.ddl:313`) for immediate administrative audit.
- Fee structures are locked at checkout into `MasterInvoices` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/schema/spanner.ddl:140`) and `InvoiceSettlementSlices` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/schema/spanner.ddl:188`).

### 3.4 Strict 64-Bit Integer Monetary Amounts
- Floating-point types (`FLOAT32`, `FLOAT64`) are **STRICTLY PROHIBITED** for currencies, prices, wallet balances, and settlements.
- All monetary amounts in database schemas, Go structs, TypeScript contracts, and solver inputs are expressed as 64-bit signed integers (`INT64` / `int64`) in the minor currency unit (e.g., Uzbek Tiyin: 1 UZS = 100 Tiyin; US Cents: 1 USD = 100 Cents).

### 3.5 Uber H3 Resolution-7 Geospatial Indexing
- Raw Cartesian and unindexed O(N) Haversine iterations in production query paths are banned.
- All spatial partitioning, territory validation, warehouse catchment polygons, and driver telemetry must utilize **Uber H3** hexagonal indexing at resolution 7 (~1.2 km² per cell), implemented in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/proximity/h3.go` and `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/proximity/engine.go`.
- Warehouse polygons are serialized as H3 string arrays (`Warehouses.H3Indexes`).
- Proximity queries execute via H3 `GridDisk` ring expansion with panic-safe Haversine fallbacks.

### 3.6 Single-Flight Redis Cache Coalescing
- High-read entities (Retailer profiles, supplier catalogs, driver profiles) are cached in Redis via `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/cache/cache.go`.
- To eliminate cache stampedes and dogpiling under high concurrency, cache misses must wrap Spanner reads with `golang.org/x/sync/singleflight.Group` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/bootstrap/app.go:74`), ensuring that concurrent requests for the same key result in exactly 1 database query.

### 3.7 6 Dedicated WebSocket Hubs & Command Handshakes
Live bidirectional events flow through 6 isolated WebSocket hubs constructed in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/bootstrap/app.go:115` and implemented in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/ws/`:
1. `RetailerHub` (`/ws/retailer`): Pushes cart updates, `DRIVER_APPROACHING`, and AI preorder confirmations.
2. `DriverHub` (`/v1/ws/driver`): Pushes stop sequences, dynamic reroutes, and cash collection acknowledgments.
3. `PayloaderHub` (`/v1/ws/payloader`): Pushes digital seal states and dock loading validations.
4. `SupplierHub` (`/ws/supplier`): Pushes live telemetry aggregates and solver completions.
5. `WarehouseHub` (`/ws/warehouse`): Broadcasts dispatch lock states and inbound transfer alerts.
6. `FactoryHub`: Pushes production transfer assignments and dock manifests.
- Command execution uses `CommandRegistry` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/ws/command_registry.go`) with Redis-backed client ACK validation.

---

## 4. Developer Workflows & Verification Commands

All modifications must be validated using the standard Pegasus developer toolchain defined in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/Makefile`:

### 4.1 Local Simulation Environment Setup
1. **Boot Local Emulators**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus
   make env-up
   ```
   Starts Kafka 7.7.0 (KRaft mode, port 9092), Kafka UI (port 8081), Redis Alpine (port 6379), Cloud Spanner Emulator (ports 9010/9020), Firebase Auth emulator (port 9099), and Global Pay WireMock (port 8085). Defined in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/docker-compose.yml`.

2. **Initialize Spanner Schema**:
   ```bash
   make spanner-init
   ```
   Executes `cmd/setup/main.go` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/cmd/setup/main.go`), provisioning instance `pegasus-dev`, database `pegasus-db`, and applying all 94 DDL statements.

3. **Inject Deterministic Test Seed**:
   ```bash
   make seed
   ```
   Executes `cmd/seed/main.go` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/cmd/seed/main.go`) to inject multi-role users, test catalog, warehouses, factories, and live orders.

4. **Run Native Backend**:
   ```bash
   make run-backend
   ```
   Binds `0.0.0.0:8080` with fail-closed configuration loaded from `apps/backend-go/.env`.

5. **Start Optimizer Sidecar Simulation**:
   ```bash
   make optimizer-sim-up
   ```
   Spins up the Rust Tonic solver (`pegasus-optimizer-core-rust` on port 50055) and the Go adapter worker (`pegasus-optimizer-adapter-worker` with metrics on port 8082).

### 4.2 Automated Testing & Enterprise Quality Gates
Before submitting any commit or code change, run the mandatory verification gates:

- **Enterprise Baseline Execution Gate**:
  ```bash
  make sprint1-gate
  ```
  Runs `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/scripts/sprint1_execution_gate.py` to verify architecture boundaries, versionscan compliance, contract drift, and security policies. Output report: `pegasus/.execution/sprint1/gate-report.json`.

- **Contract Synchronization Check**:
  ```bash
  make gen-contracts-gate
  ```
  Runs `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/scripts/parity/gen_contracts_gate.sh` to ensure `contracts/events.schema.json` matches backend code.

- **Unified Architectural Guard ("One-Eye")**:
  ```bash
  npm run guard:one-eye
  ```
  Defined in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/package.json:19`. Runs contract drift guard, architecture boundary guard, design token guard, production safety guard, visual test intelligence guard, and security guard.

- **Backend Go Unit & Integration Tests**:
  ```bash
  cd apps/backend-go && go test -v -race ./...
  cd apps/ai-worker && go test -v ./...
  cd services/optimizer-core/adapters/go && go test -v ./...
  ```

- **Rust Optimizer Core Tests**:
  ```bash
  cd services/optimizer-core/server-rust && cargo test
  ```

- **Web Portals Playwright E2E Test Suite**:
  ```bash
  npm run test:e2e
  npm run test:e2e:admin
  npm run test:e2e:retailer
  npm run test:e2e:cross
  ```

- **Physical Device LAN IP Setup**:
  ```bash
  make dev-ip
  make dev-devices
  ```
  Displays the local workstation IP and environment file configurations for physical iOS and Android test devices.


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
