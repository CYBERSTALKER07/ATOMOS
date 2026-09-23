# Handoff Report: In-Memory Repository Stubs & Silent Fallbacks Audit (R2 & R1.3)

**Agent**: `teamwork_preview_explorer_survey_11_1`  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_11_1`  
**Parent**: `d877571c-b5bd-4489-a1cb-441c991bb03d`  
**Target Codebase**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Date**: 2026-09-23  

---

## 1. Observation

### Observation 1.1: Core Priority Packages with Production In-Memory Repositories (Milestone 1)
- **`backend/internal/consignment/service.go`**:
  - Line 22: `type MemoryRepository struct { mu sync.RWMutex; agreements map[string]*ConsignmentAgreement; vouchers map[string]*ConsignmentSettlementVoucher }`
  - Line 28: `func NewMemoryRepository() *MemoryRepository`
  - Lines 35–80: In-memory CRUD methods on `*MemoryRepository`.
  - Line 87–94:
    ```go
    func NewService(repo Repository, pool *db.Pool) *Service {
        if repo == nil {
            if pool != nil {
                repo = NewPostgresRepository(pool)
            } else {
                repo = NewMemoryRepository()
            }
        }
        return &Service{repo: repo, pool: pool}
    }
    ```
- **`backend/internal/rebate/repository.go` & `service.go`**:
  - `repository.go:13`: `type MemoryRepository struct { mu sync.RWMutex; contracts map[string]*ConditionContract; accruals map[string][]*ConditionContractAccrual; postings map[string][]LedgerPosting }`
  - `repository.go:20`: `func NewMemoryRepository() *MemoryRepository`
  - `service.go:24–31`:
    ```go
    func NewService(repo Repository, pool *db.Pool) *Service {
        if repo == nil {
            if pool != nil {
                repo = NewPostgresRepository(pool)
            } else {
                repo = NewMemoryRepository()
            }
        }
        return &Service{repo: repo, pool: pool}
    }
    ```
- **`backend/internal/payout/repository.go` & `service.go`**:
  - `repository.go:37–43`:
    ```go
    func NewRepository(pool *db.Pool) Repository {
        if pool != nil {
            return NewPostgresRepository(pool)
        }
        return NewMemoryRepository(nil)
    }
    ```
  - `repository.go:292`: `type MemoryRepository struct { mu sync.RWMutex; policies map[string]*Policy; batches map[string]*Batch; items map[string][]BatchItem }`
  - `repository.go:299`: `func NewMemoryRepository(_ *db.Pool) *MemoryRepository`
  - `service.go:96`: `if len(items) == 0 && s.pool != nil { ... }` (silently ignores missing DB pool).
- **`backend/internal/wmsops/repository.go` & `service.go`**:
  - `repository.go:118`: `type MemoryRepository struct` (holding 10 maps/slices).
  - `repository.go:132`: `func NewMemoryRepository() *MemoryRepository`
  - `repository.go:157–782`: 625 lines of in-memory methods (`GetOpsBoard`, `ListTemplates`, `SaveTemplate`, etc.).
  - `repository.go:791–794`:
    ```go
    func NewPostgresRepository(pool *db.Pool) Repository {
        if pool == nil {
            return NewMemoryRepository()
        }
        return &PostgresRepository{pool: pool, pickers: make(map[string]map[string]PickerLocation)}
    }
    ```

### Observation 1.2: Other In-Memory Repositories in Production Code
- `backend/internal/ewm/service.go:22`: `type MemoryRepository struct` and `service.go:91`: `repo = NewMemoryRepository()`. (No PostgresRepository exists!).
- `backend/internal/copa/service.go:23`: `type MemoryRepository struct` and `service.go:105`: `repo = NewMemoryRepository()`. (No PostgresRepository exists!).
- `backend/internal/fscm/service.go:21`: `type MemoryRepository struct` and `service.go:75`: `repo = NewMemoryRepository()`.
- `backend/internal/matching/repository.go:14`: `type MemoryRepository struct` and `service.go:29`: `repo = NewMemoryRepository()`.
- `backend/internal/qm/service.go:21`: `type MemoryQMRepo struct` and `service.go:105`: `repo = NewMemoryQMRepo()`.

### Observation 1.3: Composite Fallback Repositories (`pgRepository` + `memFallback`)
- `backend/internal/transfer/repository.go:169–178`: `type pgRepository struct { pool *db.Pool; memFallback *MemoryTransferRepo }`. `NewRepository` creates `memFallback = NewMemoryTransferRepo()`. Every method has `if r.pool == nil { return r.memFallback... }` (lines 182, 212, 252, 280, 317, 344, 382, 408).
- `backend/internal/cyclecount/repository.go:184–194`: `type pgRepository struct { pool *db.Pool; memFallback *MemoryCycleCountRepo }`. Every method has `if r.pool == nil { return r.memFallback... }` (lines 197, 218, 256, 282, 308, 333, 354, 392, 418).
- `backend/internal/empties/service.go:196–206`: `type pgEmptiesRepo struct { pool *db.Pool; memFallback *MemoryEmptiesRepo }`. Methods fall back to `r.memFallback` if `pool == nil` or on query error / `pgx.ErrNoRows` (lines 222, 349, 370).

### Observation 1.4: Production Mock Clients & Stubs
- `backend/internal/onec/odata.go:139`: `type StubODataClient struct`
- `backend/internal/payroll/client.go:28`: `type StubGlobalPayPayoutClient struct`
- `backend/internal/aiorder/voicenote.go:125`: `type StubAudioTranscriber struct`
- `backend/internal/api/middleware_idempotency.go:31–38`: `var memStore = &memoryIdempotencyStore{}`
- `backend/internal/api/router.go:275`: `payrollSvc: payroll.NewPayrollService(payroll.NewStubGlobalPayPayoutClient(50000000000), pool)` (injects 500M UZS dummy balance into production router).

### Observation 1.5: 19 Hybrid Repositories with Embedded Fake Seeds
- The following repositories contain internal `map[string]...` caches and run seeding methods during constructor execution:
  - `scheduling/repository.go:33` (`repo.seedInitialState()`)
  - `notifications/repository.go:29` (`repo.seedInitialState()`)
  - `returns/repository.go:28` (`repo.seedInitialState()`)
  - `retailer/repository.go:258` (`repo.seedInitialState()`)
  - `epod/repository.go:36` (`repo.seedInitialState()`)
  - `loyalty/repository.go:31` (`repo.seedInitialState()`)
  - `fxrates/repository.go:27` (`repo.seedInitialState()`)
  - `crossdock/repository.go:30` (`repo.seedInitialState()`)
  - `promotion/repository.go:34` (`r.seedInitialState()`)
  - `controltower/repository.go:31` (`repo.seedDefaults()`)
  - `gs1core/repository.go:29` (`repo.seedDefaults()`)
  - `coverage/repository.go:34` (`repo.seedInMemory()`)
  - `fleet/repository.go:58` (`repo.seedInitialData()`)
  - `forecasting/repository.go:33` (`repo.seedInMemory()`)
  - `pickwave/repository.go:32` (`repo.seedInMemory()`)
  - `seasonalcore/repository.go:29` (`repo.seedDefaults()`)
  - `commitments/repository.go:31` (`repo.seedInMemory()`)
  - `floorexception/repository.go:32` (`repo.seedInMemory()`)
  - `ar/repository.go:30` (`repo.seedDefaults()`)
  - `bins/repository.go:40` (`initBins`)
  - `doorstep/repository.go:32` (in-memory maps)
  - `order/service.go:56` (`inMemoryOrders`)
  - `inventory/service.go:47` (`initBaseline...`)
  - `onboarding/service.go:115` (`initBaselineProfiles()`)

### Observation 1.6: Server Entry Point
- `backend/cmd/server/main.go:76–79`:
  ```go
  pool, err := db.Connect(ctx, cfg.DatabaseURL)
  if err != nil {
      log.Printf("[Database WARNING] PostgreSQL connection deferred or failed: %v", err)
  } else { ... }
  ```
  Failure to connect does NOT exit; the server proceeds with `pool = nil`.

---

## 2. Logic Chain

1. **Root Vulnerability**: `cmd/server/main.go:78` treats a database connection failure as a non-fatal warning, propagating `pool = nil` to `api.NewServer` (Observation 1.6).
2. **Cascading Silent Fallbacks**: Because `pool` can be `nil`, constructors in `consignment`, `rebate`, `payout`, and `wmsops` explicitly branch on `pool == nil` and instantiate in-memory repositories (`MemoryRepository`) rather than returning an error (Observation 1.1).
3. **Data Loss & Split-Brain Hazard**: In production, if the database pool disconnects or fails at boot:
   - Orders, vouchers, payouts, and ops board actions are written to ephemeral heap memory.
   - Any server restart permanently destroys all state.
   - Upstream and downstream roles desynchronize because transactional outbox events are never emitted to PostgreSQL (`pgx.Tx`), violating the Zero-Orphaned-Features Doctrine (V.O.I.D Doctrine §3).
4. **Mock Pollution in Production Code**: By leaving `MemoryRepository` (625 lines in `wmsops`), `StubODataClient`, and `StubGlobalPayPayoutClient` inside production packages, production binaries bundle unvetted mock code that can accidentally be activated in live routes (Observations 1.1, 1.4).
5. **Contract Inconsistency**: Unit tests across `consignment_test.go`, `payout_test.go`, and `wmsops_test.go` directly rely on calling `NewService(nil, nil)` or `NewMemoryRepository()` (Observations 1.1, 1.4). Moving these mocks to `*_test.go` preserves test coverage while guaranteeing zero mock data in production binaries.

---

## 3. Caveats

1. **Historical Smoke Check Suite**: `cmd/smokecheck/main.go` runs standalone verification without a live database for certain tests (e.g. `ewm.NewService(nil, nil)`, `onec.NewStubODataClient()`). Smoke check tests that instantiate services directly with `nil` will need test-scoped mock repositories or a local test database.
2. **Scope of Milestone 1 vs Full Hardening**: Milestone 1 focuses on `internal/consignment`, `internal/rebate`, `internal/payout`, and `internal/wmsops`. The 19 hybrid repositories and other packages (`ewm`, `copa`, `transfer`, etc.) should be scheduled in subsequent milestones (M1–M4) to prevent an excessively large single commit blast radius.
3. **No Database Migrations Required for M1**: Database migrations for consignment (`009`), rebate (`007`), payout (`051`), and wmsops (`056`, `057`) already exist in `database/migrations/`. No new DDL is needed to eliminate the memory fallbacks.

---

## 4. Conclusion

1. **Violations Identified**:
   - `internal/consignment/service.go`: Lines 22–80 contain `MemoryRepository`.
   - `internal/rebate/repository.go`: Lines 13–64 contain `MemoryRepository`.
   - `internal/payout/repository.go`: Lines 292–398 contain `MemoryRepository`.
   - `internal/wmsops/repository.go`: Lines 118–782 contain `MemoryRepository`.
   - Production constructors in all 4 packages silently fallback to in-memory repositories when `pool == nil`.
2. **Refactoring Scope**:
   - Purge all 4 `MemoryRepository` implementations from production Go files.
   - Relocate mocks into dedicated `mock_repository_test.go` files.
   - Enforce fail-closed constructors: return `(*Service, error)` and `(*PostgresRepository, error)` requiring a non-nil `*db.Pool`.
   - Update `router.go` to handle constructor errors.
   - Update unit tests in `*_test.go` to instantiate mocks directly.

---

## 5. Verification Method

### 5.1 Static Verification (AST & Grep Scans)
Run the following commands from `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`:

1. **Verify Zero MemoryRepository in Production Go Files for M1 Packages**:
   ```bash
   rg "type .*MemoryRepository struct" backend/internal/consignment/ backend/internal/rebate/ backend/internal/payout/ backend/internal/wmsops/ -g '!*_test.go'
   ```
   *Expected Result*: 0 matches.

2. **Verify Fail-Closed Constructors Reject Nil Pool**:
   Verify that calling `consignment.NewService(nil, nil)`, `rebate.NewService(nil, nil)`, `payout.NewRepository(nil)`, and `wmsops.NewPostgresRepository(nil)` returns a non-nil error.

3. **Verify Zero Compilation Regressions**:
   ```bash
   cd backend && go build ./...
   ```
   *Expected Result*: Exit code 0, clean build.

4. **Verify Automated Unit & Integration Tests**:
   ```bash
   cd backend && go test -v -race ./internal/consignment/... ./internal/rebate/... ./internal/payout/... ./internal/wmsops/...
   ```
   *Expected Result*: 100% PASS with 0 race conditions.

5. **Full Suite Regression Check**:
   ```bash
   cd backend && go test -race ./...
   ```
   *Expected Result*: All existing tests pass cleanly.
