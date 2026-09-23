# Remediation Plan — pegasus.x Victory Audit Hardening

## Objective
Remediate all findings in `/Users/shakhzod/Desktop/V.O.I.D/.agents/victory_auditor_orch_2/handoff.md` across `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`, ensuring complete elimination of disguised in-memory repositories, fail-closed constructors, strict tiyin arithmetic, atomic outbox pairing, and 100% passing tests and benchmarks.

---

## Phase 1: Exploration & Code Scope Analysis
- Verify exact lines and dependencies across all target packages:
  - `internal/cyclecount/repository.go`
  - `internal/transfer/repository.go`
  - `internal/empties/service.go`
  - `internal/qm/service.go` & `internal/qm/quarantine.go`
  - `internal/promotion/repository.go` & `internal/promotion/service.go`
  - `internal/commitments/repository.go` & `internal/commitments/service.go`
  - `internal/order/service.go`
  - `internal/credit/service.go`
  - `internal/retailer/repository.go`
  - `internal/warehouse/service.go`
  - `internal/api/handlers_soliq.go`
  - Constructors across `crossdock`, `controltower`, `claims`, `wms`, `manifest`, `dock`, `creditnote`, `cashrecon`.

## Phase 2: Implementation (Worker Subagents)
- **Work Package A: Purge Disguised In-Memory Repositories & Implement Genuine Persistence**:
  1. `cyclecount`: Delete `MemoryCycleCountRepo` & `memFallback` from `repository.go`. Move unit test mocks to `repository_test.go`.
  2. `transfer`: Delete `MemoryTransferRepo` & `memFallback` from `repository.go`. Move unit test mocks to `repository_test.go`.
  3. `empties`: Delete `MemoryEmptiesRepo` & silent fallback from `service.go`. Move mocks to `*_test.go`.
  4. `qm`: Delete `MemoryQMRepo` & `NewMemoryQMRepo` from `service.go`. Wire genuine postgres qm repo in `internal/api/router.go`.
  5. `promotion` & `commitments`: Implement PostgreSQL 16 persistence for promotions and commitments; delete `seedInMemory()` in production files.
- **Work Package B: Fail-Closed Constructors**:
  1. `order/service.go`: `NewService` panics if `pool == nil`. Delete `s.inMemoryOrders` and `s.initInMemory()`. Update unit tests if any passed nil pool.
  2. `credit/service.go`: `NewService` panics if `pool == nil`. Delete in-memory maps from production code.
  3. Enforce `if pool == nil { panic("...") }` across remaining constructors (`crossdock`, `controltower`, `promotion`, `commitments`, `claims`, `wms`, `manifest`, `dock`, `creditnote`, `cashrecon`).
- **Work Package C: Currency/VAT Arithmetic & Outbox Atomicity**:
  1. `retailer/repository.go`: Fix VAT integer floor division at line 2884 to statutory round-half-up: `vat := (tot * 12 + 56) / 112`.
  2. `qm/quarantine.go`: Eliminate `float64` conversion at line 79; enforce strict `int64` minor unit math.
  3. `warehouse/service.go`: Enforce `TxRepository` and eliminate non-atomic dual-transaction fallbacks in lines 119-130 and 398-412.
  4. `api/handlers_soliq.go`: Fix line 147 to check and return transaction error instead of discarding with `_ = s.pool.RunInTx(...)`.

## Phase 3: Automated Test Execution & Race Detection
- Compile binaries:
  - `go build -v ./cmd/server`
  - `go build -v ./cmd/smokecheck`
- Static Analysis:
  - `go vet ./...` (0 diagnostics)
- Full Race-Enabled Test Suite:
  - `go test -v -race ./...` (0 failures, 0 races)
- Scale & Domain Benchmarks:
  - 1,000-order H3 clustering (<100ms) with 0 abandoned orders
  - 100-order CVRP dispatch with 0 abandoned orders
  - 3L-CVRP longitudinal axle statics physics & supervisor override
  - Mid-shift breakdown rescue hot-swap

## Phase 4: Independent Dual Review & Challenger Verification
- Dispatch Reviewer 1 to audit code changes against the Universal Engineering Doctrine.
- Dispatch Reviewer 2 / Challenger to adversarially probe for hidden fallbacks, regression vectors, or test skips.
- Update `GATE_STATUS.md`.

## Phase 5: Synthesis & Sentinel Notification
- Synthesize all review artifacts.
- Prepare final handoff report.
- Send completion message to Sentinel parent for Victory Audit re-evaluation.
