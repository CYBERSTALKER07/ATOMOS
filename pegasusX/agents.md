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

## 1. Enterprise Engineering, Algorithm & Research Doctrine (Mandatory for All Code)

Every function, algorithm, data structure, database query, and infrastructure component—whether authoring **new code** or touching **existing code**—must adhere to Big-Tech engineering standards (Google / AWS / Palantir / Stripe / Netflix caliber). No naive, toy, hacky, or amateur code is ever permitted in PegasusX.

### 1.1. The 4-Tier Solution Discovery Waterfall (Research Protocol)

Before writing any non-trivial logic, algorithm, pipeline, or service interface, the agent must execute the following 4-tier solution discovery waterfall:

```
[Tier 1: Deep Web & Literature Research]
       │ (State-of-the-art RFCs, ACM papers, production benchmarks)
       ▼
[Tier 2: Proven Open-Source & Commercial Tooling]
       │ (Paid/unpaid battle-tested libraries, battle-hardened algorithms)
       ▼
[Tier 3: Reverse-Engineer Big Tech Architecture]
       │ (Analyze how Google, AWS, Uber, Stripe, Palantir solve it at scale)
       ▼
[Tier 4: Enterprise-Grade Big Tech Innovation]
         (Design custom mathematically-proven, zero-contention algorithm)
```

1. **Tier 1 — Deep Research (Web, Literature & RFCs):**
   - Actively research state-of-the-art implementations, algorithms, and industry standards using web search and official documentation tools.
   - Do not guess algorithmic complexity, cryptographic handshakes, or distributed consensus patterns. Benchmark against established RFCs, academic literature, and official cloud architecture guides.

2. **Tier 2 — Leverage Battle-Tested Open Source & Tooling (Paid or Unpaid):**
   - If a problem is already solved by a world-class, battle-tested open-source or commercial tool/library (e.g., Google OR-Tools for VRP/routing, Uber H3 for hexagonal spatial indexing, RocksDB/Pebble for LSM storage, Roaring Bitmaps for high-throughput filtering, SIMD-accelerated parsers, CRDT libraries), **adopt and integrate it**.
   - Do not naively reinvent solved engineering wheels (e.g., writing custom heuristic 3D bin-packing or vehicle routing from scratch when proven CP-SAT/MILP solvers exist).

3. **Tier 3 — Reverse-Engineer Big Tech / Market Leaders:**
   - If no drop-in open-source library fits PegasusX constraints, research and reverse-engineer the architectural patterns, data structures, and algorithms used by industry leaders:
     - *Palantir Foundry*: Copy-on-Write scenario branching, ontology object blast-radius graphs, kinetic action contracts.
     - *Uber / DoorDash*: CVRPTW dynamic dispatching, geofence hysteresis, driver payout waterfalls.
     - *Amazon / Flexport*: Reverse-stop LIFO loading, 3D crushability bin-packing, SSCC-18 pallet serialization.
     - *Stripe / Adyen*: Dual-entry immutable ledgers, idempotent charge ratchets, split-tender waterfalls.
   - Reconstruct their data structures, state machine transitions, and concurrency guarantees cleanly within our Go, Rust, or Kotlin/Swift architectures.

4. **Tier 4 — Enterprise-Grade Big Tech Proprietary Innovation:**
   - If our unique operational requirements (e.g., offline-first Central Asian distribution, Soliq EHF PKCS#12 fiscal latency, Cloud Spanner multi-tenant partitioning) have no industry precedent, design and engineer our own **proprietary Big-Tech caliber approach**:
     - Formulate explicit mathematical complexity bounds ($O(1)$ lookups, $O(N \log N)$ pipelines, zero $O(N^2)$ loops).
     - Guarantee zero lock contention, transactional atomicity, and memory efficiency under high-concurrency profiling.
     - Document the formal algorithm specification in code docstrings with complexity receipts and invariants.

### 1.2. Pre-Implementation Brainstorming Protocol

Before typing a single line of code, the agent must brainstorm and evaluate:
1. **Algorithmic Complexity & Data Structures**: Time complexity ($O$), Space complexity, memory allocations (`pprof`), cache locality, and whether a specialized structure (Trie, Fenwick Tree, Ring Buffer, Circular Queue, Skip List, Inverted Index) outperforms naive slices/maps.
2. **Database & Network Roundtrip Minimization**: Never execute $N+1$ Spanner queries. Use batched mutations, interleaved table hierarchies, read-only replica routing, and Spanner transactions with deferred mutations.
3. **Concurrency & Failure Mode Analysis**: Thread safety under `-race`, goroutine lifecycle leakage, deadlocks, transaction retry loops (idempotent mutations only!), backpressure buffers, and network timeout budgets.
4. **Edge-Case & Chaos Hardening**: How does this code behave during network drops, malformed JSON, out-of-order Kafka events, clock drift, duplicate webhook deliveries, and database deadlocks?

### 1.3. Universal Applicability: The Boy Scout Modernization Standard

- **New Code**: Must strictly satisfy this entire doctrine from line 1.
- **Existing Code**: Whenever inspecting, modifying, or refactoring existing code, agents must **audit it against this standard**:
  - Detect and eliminate naive loops, $O(N^2)$ lookups, and unindexed filtering.
  - Replace string-concatenated SQL/Spanner queries with parameterized statements.
  - Eliminate unhandled errors, swallowed exceptions, or non-deterministic behavior.
  - Upgrade legacy code to enterprise-grade big-tech standards on every visit.

---

## 2. Architectural Boundaries & Authoritative System Map

> [!IMPORTANT]
> **Separation of Concerns**: `agents.md` is strictly the agent's **operating constitution and engineering doctrine**.  
> Detailed domain operations, business logic, role workflows, and state machines are maintained in dedicated documentation. Agents must consult the authoritative references below:

### 2.1. System Ecosystem & 9 Operational Roles
PegasusX enforces a Single-Supplier Multi-Retailer (SSMR) FMCG architecture across 9 operational roles. Consult the designated specifications for feature mechanics:
- **Master Role Index**: [`docs/roles/ROLES_MASTER_INDEX.md`](docs/roles/ROLES_MASTER_INDEX.md)
- **Detailed Role Operations & Features**: [`docs/roles/DETAILED_ROLE_OPERATIONS_AND_FEATURES.md`](docs/roles/DETAILED_ROLE_OPERATIONS_AND_FEATURES.md)
- **Palantir Foundry Ontology & Edge Cases**: [`docs/roles/FOUNDRY_ONTOLOGY_AND_EDGE_CASES.md`](docs/roles/FOUNDRY_ONTOLOGY_AND_EDGE_CASES.md)
- **Canonical Order Lifecycle & State Machine**: [`docs/roles/ORDER_LIFECYCLE.md`](docs/roles/ORDER_LIFECYCLE.md) & [`docs/canonical-order-state-machine.md`](docs/canonical-order-state-machine.md)
- **Individual Role Runbooks**:
  - Supplier: [`docs/roles/ROLE_SUPPLIER.md`](docs/roles/ROLE_SUPPLIER.md)
  - Factory: [`docs/roles/ROLE_FACTORY.md`](docs/roles/ROLE_FACTORY.md)
  - Warehouse: [`docs/roles/ROLE_WAREHOUSE.md`](docs/roles/ROLE_WAREHOUSE.md)
  - Payloader: [`docs/roles/ROLE_PAYLOAD.md`](docs/roles/ROLE_PAYLOAD.md)
  - Driver: [`docs/roles/ROLE_DRIVER.md`](docs/roles/ROLE_DRIVER.md)
  - Retailer: [`docs/roles/ROLE_RETAILER.md`](docs/roles/ROLE_RETAILER.md)
  - Depot Cashier & Vault: [`docs/roles/ROLE_DEPOT_CASHIER.md`](docs/roles/ROLE_DEPOT_CASHIER.md)
  - Platform Admin: [`docs/roles/ROLE_PLATFORM_ADMIN.md`](docs/roles/ROLE_PLATFORM_ADMIN.md)
  - Fleet & Yard Manager: [`docs/roles/ROLE_FLEET_YARD.md`](docs/roles/ROLE_FLEET_YARD.md)
  - SRE & FinOps: [`docs/roles/ROLE_SRE_FINOPS.md`](docs/roles/ROLE_SRE_FINOPS.md)

### 2.2. Authoritative Technology Stack (Do Not Hallucinate Foreign Stacks)
- **Primary Transactional Store**: **Google Cloud Spanner** (`cloud.google.com/go/spanner`) with authoritative schema in `apps/backend-go/schema/spanner.ddl` and incremental migrations in `apps/backend-go/schema/migrations/`. *(Note: PostgreSQL/`pgx`/`timescaledb` idioms are strictly forbidden in `apps/backend-go`—Spanner is the sole transactional store.)*
- **Cache & Realtime Backplane**: **Redis** (Cloud Memorystore in GCP) for cache-aside reads (`cache:invalidate` post-commit) and fail-open WebSocket Pub/Sub fanout (`ws:<hub>:fanout`).
- **Event Streaming**: **Apache Kafka** (GCP Managed Service for Apache Kafka in staging/prod) fed exclusively via the Spanner transactional `OutboxEvents` relay (`apps/backend-go/outbox/relay.go`).
- **Backend & Workers**: Go 1.25+ (`apps/backend-go`, `apps/ai-worker`, `apps/handoff-service`), Python OR-Tools + Rust CP-SAT/VRP (`services/optimizer-core`).
- **Web & Desktop Portals**: Next.js + React + Tailwind + Tauri v2 (`apps/retailer-portal`, `apps/supplier-portal`, `apps/warehouse-portal`, `apps/factory-portal`, `apps/admin-portal`).
- **Mobile Fleet**: Native Kotlin + Jetpack Compose (`apps/*-app-android`) and Native Swift + SwiftUI (`apps/*-app-ios`).

---

## 3. Production Invariants & Automated Enforcement Gates

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
- **Rule**: 100% of `/v1/...` routes called by any Web, Desktop, Android, or iOS client across all 9 operational roles must map to registered routes in `apps/backend-go`, and active Terraform monitoring (`infra/terraform/modules/monitoring/main.tf`) must enforce all platform SLO alerts.
- **Gates**: `scripts/parity/role_row_contract_check_full.sh`, `scripts/parity/gap_hunter_gate.sh`, `scripts/phase2_gate.sh` through `scripts/phase5c_gate.sh`.

### Invariant 10: Mandatory Pre-Execution Kinetic & 12 Edge-Case Guards
- **Rule**: All domain state mutations across the 10-step operational cascade must enforce the 12 kinetic edge-case pre-flight rubrics defined in [`docs/roles/FOUNDRY_ONTOLOGY_AND_EDGE_CASES.md`](docs/roles/FOUNDRY_ONTOLOGY_AND_EDGE_CASES.md). Handlers must never bypass ontology security markings or kinetic precondition checks.
- **Gate**: `(cd apps/backend-go && go test -race ./foundry/...)` and `make schema-drift-gate`.

---

## 4. Mandatory Engineering Workflow (Before & After Every Edit)

### Step 1: Pre-Edit Blast-Radius & Root-Cause Analysis
1. **Query the Code Graph**: Always query Graph RAG (`/graph-rag explore <symbol>`, `/graph-rag blast <symbol>`, or `codegraph_explore`) before modifying any function, route, or table to inspect callers, outbox events, and Spanner mutations.
2. **Locate the Canonical Source**: Inspect `go.work`, `pnpm-workspace.yaml`, `Package.swift`, or `settings.gradle.kts` before creating any file. Never create a duplicate model, helper, or package directory if one already exists.
3. **Check Concurrency & Failure Modes**:
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

---

## 5. Frontend & Mobile UI Engineering Doctrine (Mobbin & 21st.dev MCPs)

### 5.1. Zero Shared UI Packages (Absolute Invariant)
- **Law**: Never create, import, or wire shared cross-app UI libraries (e.g. `@pegasusx/ui-kit`, `:mobile-design`, `:mobile-kit`, or `packages/mobile-ios-design`).
- **Required**: Every application must own its UI components, design tokens, layouts, animations, and motion primitives natively and locally:
  - **Web & Desktop Portals** (`apps/*-portal`, `apps/retailer-app-desktop`): Local `components/ui/`, `app/styles/`, and `lib/enterprise-swr.ts`.
  - **iOS Applications** (`apps/*-app-ios`): Local `Theme/`, `Design/`, `Views/`, and `Services/NetworkLifecycleEngine.swift`.
  - **Android Applications** (`apps/*-app-android`): Local `ui/design/`, `ui/components/`, `ui/theme/`, and `network/NetworkLifecycleEngine.kt`.
- **Enforcement**: Mechanically verified by `UiIndependenceTest` on Android/vitest and `ci_anti_theatre_gate.sh`.

### 5.2. UI Design Tooling & MCP Integration
To guarantee that user interfaces remain production-grade without hallucinating amateur layouts or introducing prohibited shared packages, agents use two designated MCP design tools:

1. **Mobbin MCP (`https://api.mobbin.com/mcp`) — Mobile & Web Pattern Grounding**:
   - **Role**: Reference engine for real-world, battle-tested UX flows (e.g., Amazon Flex / Uber Driver delivery manifests, Square POS checkout, Flexport warehouse pallet receiving, DoorDash live tracking).
   - **Usage**: Query Mobbin (`search_screens`, `search_flows`, `search_sections`) to understand production UX structure, hierarchy, and edge-state displays before authoring screens.
   - **Policy**: Pure visual and structural reference only. Never import external code directly; reimplement all patterns natively in local SwiftUI, Jetpack Compose, or Tailwind React.

2. **21st.dev MCP (`https://21st.dev/api/mcp`) — Web & Desktop Component Code Generation**:
   - **Role**: Source code retrieval for modern React, Tailwind CSS, and Framer Motion components (Bento grids, count-up stats, command palettes, telemetry EKG strips, glassmorphic panels).
   - **Usage**: Retrieve raw component source code and place it directly into the target app's local `components/ui/` directory (following the copy-paste `shadcn/ui` model).
   - **Policy**: Strict adaptation to the **Pegasus Monochrome Enterprise Theme** (`bg-neutral-900`, `border-neutral-800`, `text-neutral-100`). Never install npm runtime dependencies from 21st.dev; embed source code locally to preserve zero shared package boundaries.
