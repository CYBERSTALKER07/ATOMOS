# Comprehensive Ecosystem Deep Audit & Parity Plan: pegasus vs. pegasusX vs. pegasus.x

> **Architectural Law:** Two-Tier Verification Gate (Automated Bash/CodeGraph Radar + Targeted Raw Reading & Manual Editing).  
> **Strict System Boundary:** Zero Spanner/Kafka in `pegasus.x`; zero single-tenant PostgreSQL in `pegasusX`.  
> **Honesty Rule:** No claim is marked "Wired" or "Complete" without exact `file:line` citations and test suite pass verification.

**Goal:** Execute a compiler-grade, deep architectural audit across `pegasus`, `pegasusX`, and `pegasus.x` using automated bash scripts for radar scanning and targeted raw codebase reading and manual code editing for maximum precision. Detect all feature parity gaps, state machine inconsistencies, concurrency risks, and data flow bottlenecks, producing verified synchronizations.

**Tech Stack:** 
- `pegasusX`: Go 1.23 (Chi), Google Cloud Spanner, Apache Kafka, Redis 7, Google OR-Tools (Python), Next.js 15, Kotlin Android, Swift iOS.
- `pegasus.x`: Go 1.23 (Chi, pgx/v5), PostgreSQL 16 (Timescale/PostGIS container), Redis 7 (Streams/PubSub), Python 3.12 (FastAPI, Croston-SBA, 2-Opt), Tauri v2 Desktop, Telegram Grammy Bot, React 19 MiniApp.

---

## Phase 1: Automated Static & Graph Radar (Bash Scripts & Tooling)

Automated scripts provide macro-level discovery across the ~220 packages, 3,749 lines of Spanner DDL, and 69 PostgreSQL migrations.

### Task 1.1: Automated Database Schema & Migration Diff
**Purpose:** Map every table, column, index, and constraint between `pegasusX/apps/backend-go/schema/spanner.ddl` and `pegasus.x/backend/migrations/*.sql`.

**Step 1: Execute Schema Extraction Script**
```bash
bash -c '
echo "=== AUDIT 1.1: SPANNER DDL VS POSTGRES MIGRATIONS ==="
mkdir -p scratch/audit

# Extract Spanner tables & primary keys
grep -E "CREATE TABLE [A-Za-z0-9_]+" pegasusX/apps/backend-go/schema/spanner.ddl | awk "{print \$3}" | sort -u > scratch/audit/spanner_tables.txt

# Extract PostgreSQL tables from 69 migrations
grep -h -E "CREATE TABLE (IF NOT EXISTS )?[A-Za-z0-9_]+" pegasus.x/backend/migrations/*.sql | awk "{print \$NF}" | tr -d "();" | sort -u > scratch/audit/postgres_tables.txt

echo "Spanner tables count: $(wc -l < scratch/audit/spanner_tables.txt)"
echo "Postgres tables count: $(wc -l < scratch/audit/postgres_tables.txt)"
'
```

**Step 2: Detect Interleaved Child Tables in Spanner**
```bash
grep -n "INTERLEAVE IN PARENT" pegasusX/apps/backend-go/schema/spanner.ddl > scratch/audit/spanner_interleaved.txt
```

---

### Task 1.2: Messaging & Topic/Stream Topology Audit
**Purpose:** Extract all Kafka topics, partition balancers, and Redis Stream keys across both systems to verify event contracts.

**Step 1: Execute Event Bus Scanner**
```bash
bash -c '
echo "=== AUDIT 1.2: MESSAGING TOPOLOGY ==="
# Kafka topics in pegasusX
grep -rn "Topic[A-Za-z0-9_]* =" pegasusX/apps/backend-go/events/ > scratch/audit/kafka_topics.txt

# Redis Streams and PubSub channels in pegasus.x
grep -rnE "(XAdd|PublishEvent|stream:)" pegasus.x/backend/internal/ > scratch/audit/redis_messaging.txt
'
```

---

### Task 1.3: HTTP Route Surface & Endpoint Matrix
**Purpose:** Generate a unified route matrix comparing registered Chi router endpoints between `pegasusX` (136 packages) and `pegasus.x` (82 packages).

**Step 1: Run Route Discovery Script**
```bash
bash -c '
echo "=== AUDIT 1.3: ROUTE REGISTRY SCAN ==="
# pegasusX routes
grep -rnE "r\.(Get|Post|Put|Patch|Delete)\(" pegasusX/apps/backend-go/ | awk -F: "{print \$1 \":\" \$2 \" \" \$3}" > scratch/audit/pegasusX_routes.txt

# pegasus.x routes
grep -rnE "r\.(Get|Post|Put|Patch|Delete)\(" pegasus.x/backend/internal/api/ > scratch/audit/pegasus_x_routes.txt

echo "pegasusX routes count: $(wc -l < scratch/audit/pegasusX_routes.txt)"
echo "pegasus.x routes count: $(wc -l < scratch/audit/pegasus_x_routes.txt)"
'
```

---

## Phase 2: Targeted Raw Codebase Reading (Manual Inspection - Microscope)

Automated scripts identify where code exists; targeted raw reading verifies runtime semantics, transaction boundaries, error handling, and mathematical invariants.

### Task 2.1: Transaction Boundaries & Outbox Atomicity Inspection
**Files to raw-read:**
- `pegasusX/apps/backend-go/outbox/spanner_txn_buffer.go:1-60`
- `pegasusX/apps/backend-go/outbox/relay.go:60-150`
- `pegasus.x/backend/internal/outbox/emitter.go:1-40`
- `pegasus.x/backend/internal/outbox/relay.go:50-140`

**Manual Inspection Checklist:**
1. Verify `SpannerTxnBuffer`: Does `buf.Flush(ctx)` commit mutations and `OutboxEvents` in the exact same Spanner transaction with `spanner.CommitTimestamp`?
2. Verify `pegasus.x` Outbox Emitter: Does `outbox.Emit` receive `pgx.Tx` directly? Is there any path where an entity is committed without an outbox record?
3. Verify concurrency locking: Does `pegasus.x` use `FOR UPDATE SKIP LOCKED`? Does `pegasusX` outbox rely on distributed lease timeout (`ClaimedUntil < @now`)?
4. Inspect poison pill isolation: Does an event with 20 failures move cleanly to `OutboxDeadLetters` without panicking or wedging the relay ticker?

---

### Task 2.2: Connection Pooling & Resource Ergonomics Inspection
**Files to raw-read:**
- `pegasusX/apps/backend-go/bootstrap/infra.go:60-160`
- `pegasusX/apps/backend-go/bootstrap/runtime_adapters.go:40-85`
- `pegasus.x/backend/internal/db/postgres.go:1-70`
- `pegasus.x/docker/docker-compose.yml:20-60`

**Manual Inspection Checklist:**
1. Verify `pgxpool` configuration in `pegasus.x`: Inspect `cfg.MaxConns = 25`, `cfg.MinConns = 5`, `cfg.MaxConnLifetime = 1h`. Verify `RunInTx` recovery handler (`recover()` -> `tx.Rollback`).
2. Verify Spanner client session pool in `pegasusX`: Check `spanner.Client` session pool settings. Verify gRPC HTTP/2 channel multiplexing.
3. Verify Redis connection settings: Does `pegasusX` operate in fail-closed mode (`circuit_fail_closed`) in production? Does `pegasus.x` enforce Redis memory ceiling (`--maxmemory 512mb --maxmemory-policy volatile-lru`)?

---

### Task 2.3: Financial Accounting & Mathematical Precision Inspection
**Files to raw-read:**
- `pegasusX/apps/backend-go/payment/execution.go:1-120`
- `pegasusX/apps/backend-go/payment/double_entry.go:1-90`
- `pegasus.x/backend/internal/payment/handover.go:1-245`
- `pegasus.x/backend/internal/fiscal/calculator.go:1-80`

**Manual Inspection Checklist:**
1. Verify $\sum \text{Debits} == \sum \text{Credits}$ invariance assertion prior to committing journal entries.
2. Confirm zero floating-point arithmetic (`float32`/`float64`) in currency handling; verify strict `int64` minor units (Uzbekistan tiyins: $1\text{ UZS} = 100\text{ tiyins}$).
3. Inspect Uzbekistan statutory compliance in `pegasus.x`: Verify 12% VAT calculations (`DefaultVatRateBps = 1200`), half-up commercial rounding, and statutory B2B cash limit of 25,000,000 UZS (`ValidateB2BCashLimit`).

---

### Task 2.4: Real-Time WebSockets & Subterranean Reconnect Inspection
**Files to raw-read:**
- `pegasusX/apps/backend-go/ws/hub.go:240-340`
- `pegasus.x/backend/internal/ws/hub.go:100-165`
- `pegasus.x/apps/driver-app-android/app/src/main/java/.../OfflineDeliveryQueue.kt`

**Manual Inspection Checklist:**
1. Verify `pegasusX` ring buffer: Does `recordHistory` maintain 256 events per room? Does `ReplaySince` properly deliver missed messages without blocking other subscribers?
2. Verify `pegasus.x` monotonic sequencer: Does `BroadcastEnvelope` increment `atomic.AddInt64(&h.seq, 1)`? Does `GetEventsSince` return `fullResync: true` when a client requests a sequence older than the 2,000-event ring buffer?
3. Verify mobile offline behavior: Inspect driver subterranean signing (`HMAC-SHA256`) and local Room SQLite queue.

---

### Task 2.5: Algorithmic Optimization & S&OP Inspection
**Files to raw-read:**
- `pegasusX/apps/dispatch-optimizer-py/main.py:1-120`
- `pegasusX/services/optimizer-core/server/contract_solver.py:50-180`
- `pegasus.x/planning/main.py:1-100`
- `pegasus.x/planning/engine/croston.py:1-60`
- `pegasus.x/planning/engine/meio.py:1-70`

**Manual Inspection Checklist:**
1. Inspect Google OR-Tools in `pegasusX`: Verify `PATH_CHEAPEST_ARC` first solution heuristic, `GUIDED_LOCAL_SEARCH` metaheuristics, volumetric capacity constraints, and multi-wave virtual truck cloning.
2. Inspect Croston's SBA in `pegasus.x`: Verify formula $z_{t} = (1 - \frac{\alpha}{2}) \frac{z'_t}{p'_t}$ correcting standard Croston positive bias for intermittent demand.
3. Inspect MEIO in `pegasus.x`: Verify dynamic safety stock equation factoring lead time jitter and service level normal inverse CDF ($Z_{\alpha}$).

---

## Phase 3: Critical Domain Gaps & Feature Parity Plan

Scan both codebases to synchronize missing capabilities into `pegasus.x` from `pegasusX` (and vice-versa where `pegasus.x` sovereign capabilities exceed `pegasusX`).

### Parity Target 1: Fleet Management & Driver Shift Lifecycle in `pegasus.x`
**Gap:** `pegasusX` has rich vehicle registry and manifest assignment domain logic, while `pegasus.x` initially lacked complete driver shift pairing and DVIR pre-trip vehicle inspections.
**Action Plan:**
1. **Schema Check:** Inspect migration `025_fleet_and_driver_lifecycle_management.sql` in `pegasus.x` (`vehicles`, `driver_vehicle_assignments`, `dvir_inspections`).
2. **Backend Services:** Verify `backend/internal/fleet/` implementation for:
   - Dynamic daily shift clock-in and driver-vehicle pairing.
   - Pre-trip DVIR walkaround inspection with grounding locks.
   - Mid-shift hot-swapping for broken-down trucks on active routes.
3. **Desktop & Mobile UI:** Verify Tauri WMS desktop Fleet tab and Driver Android/iOS `PreTripDVIRDialog`.

### Parity Target 2: Universal Mutation Protocol (UMP) vs. Spanner Immutability
**Gap:** `pegasus.x` implements post-dispatch order immutability via append-only `entity_adjustments` (UMP), while `pegasusX` uses Spanner `OrderShopClosedLog` and `Claims`.
**Action Plan:**
1. Audit post-dispatch order mutations across `pegasusX/apps/backend-go/order/` and `pegasus.x/backend/internal/ump/`.
2. Ensure neither system allows destructive SQL updates to orders once in `IN_TRANSIT`.

---

## Phase 4: Manual Code Editing & Verification Gates

All edits must follow the Red-Green-Refactor TDD cycle and pass compiler-grade tests.

### Step 1: Write Focused Unit Tests First
```bash
# In pegasusX backend
cd pegasusX/apps/backend-go && go test -v -race ./outbox/... ./kafka/... ./ws/...

# In pegasus.x backend
cd pegasus.x/backend && go test -v -race ./internal/outbox/... ./internal/ws/... ./internal/payment/...
```

### Step 2: Perform Contiguous Manual Edits
- Use `replace_file_content` for surgical, line-bounded edits.
- Never make bulk blind replacements.
- Re-read every edited file immediately to verify line numbers and syntax integrity.

### Step 3: Run Full Verification Suite
```bash
# pegasusX full build & test
cd pegasusX/apps/backend-go && go build ./... && go test ./...

# pegasus.x full build & test
cd pegasus.x/backend && go build ./... && go test ./...

# Python planning engines
cd pegasus.x/planning && pytest -v
cd pegasusX/services/optimizer-core && pytest -v
```

---

## Phase 5: Handoff & Execution Choice

This audit plan is saved to `docs/plans/2026-09-16-ecosystem-deep-audit-plan.md` and registered in `task_plan.md`.

Execution options:
1. **Subagent-Driven (this session):** Dispatch autonomous subagents per audit phase with targeted raw code reading and manual validation between tasks.
2. **Teamwork Preview Multi-Agent Delegation:** Launch via `teamwork_preview` using the approved `prompt_draft.md`.
