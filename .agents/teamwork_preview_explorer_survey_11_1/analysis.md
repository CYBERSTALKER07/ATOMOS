# Exhaustive Codebase Audit: In-Memory Repository Stubs, Fake Seeds, and Nil-Pool Fallbacks in `pegasus.x`

**Target Codebase**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Auditor**: `teamwork_preview_explorer_survey_11_1`  
**Date**: 2026-09-23  
**Status**: Complete  

---

## 1. Executive Summary

An exhaustive, line-by-line audit of all 83 Go packages in `pegasus.x/backend/internal/`, the HTTP router (`internal/api/router.go`), server entry points (`cmd/server/main.go`), and smoke check suites (`cmd/smokecheck/main.go`) was conducted to identify:
1. In-memory repository stubs (`MemoryRepository`, `memoryRepo`, `mockRepo`).
2. Hardcoded fake seeds in production packages (`seedInitialState`, `seedInMemory`, `seedDefaults`).
3. Silent fallbacks when the database connection pool (`*db.Pool`) is `nil` or when database queries fail.
4. Constructor signatures and initialization paths across the application.

### Key Audit Findings
- **4 Priority Target Packages (Milestone 1)**: `internal/consignment`, `internal/rebate`, `internal/payout`, and `internal/wmsops` all contain full `MemoryRepository` definitions directly in production Go files (`service.go` or `repository.go`). Their constructors silently fallback to in-memory repositories whenever `pool == nil`.
  - In `wmsops`, over **664 lines of production code** are dedicated solely to an in-memory repository implementation!
- **5 Additional Dual-Repository Packages**: `internal/fscm`, `internal/matching`, `internal/qm`, `internal/ewm`, and `internal/copa` contain `MemoryRepository` / `MemoryQMRepo` definitions in production files. `ewm` and `copa` currently have **zero PostgreSQL repository implementations**, running 100% in-memory!
- **3 Composite Fallback Packages**: `internal/transfer`, `internal/cyclecount`, and `internal/empties` implement `pgRepository` but embed a `memFallback` field (`*MemoryTransferRepo`, `*MemoryCycleCountRepo`, `*MemoryEmptiesRepo`) that silently catches `nil` pools and SQL query errors, delegating mutations to in-memory maps.
- **3 Production Mock Clients**: `internal/onec` (`StubODataClient`), `internal/payroll` (`StubGlobalPayPayoutClient`), and `internal/aiorder` (`StubAudioTranscriber`) declare mock clients in production files. In `router.go:275`, the production router actively initializes `payroll.NewStubGlobalPayPayoutClient(50000000000)` with 500M UZS of dummy money.
- **1 In-Memory Middleware Fallback**: `internal/api/middleware_idempotency.go` uses a global variable `memStore = &memoryIdempotencyStore{}` that activates whenever Redis is unreachable.
- **19 Hybrid Repositories with Embedded Fake Seeds**: `scheduling`, `notifications`, `returns`, `retailer`, `epod`, `loyalty`, `fxrates`, `crossdock`, `promotion`, `controltower`, `gs1core`, `coverage`, `fleet`, `forecasting`, `pickwave`, `seasonalcore`, `commitments`, `floorexception`, `ar` (plus `bins`, `doorstep`, `order`, `inventory`, and `onboarding`) declare internal `map[string]...` fields and execute seeding routines (`seedInitialState()`, `seedInMemory()`, `seedDefaults()`, `initBaseline...`) upon construction.
- **The Root Cause**: In `cmd/server/main.go:76-79`, `pool, err := db.Connect(...)` catches database connection failures, logs a `[Database WARNING]`, and continues booting the server with `pool = nil`. This intentional historical allowance of a "degraded in-memory demo mode" is what spawned the widespread anti-pattern of silent fallbacks.

---

## 2. Priority Target Packages (Milestone 1) Deep Dive

### 2.1 `internal/consignment`

#### Location & Structure
- **Production File**: `backend/internal/consignment/service.go`
  - Lines 22–26: Definition of `type MemoryRepository struct` with `sync.RWMutex`, `agreements map[string]*ConsignmentAgreement`, and `vouchers map[string]*ConsignmentSettlementVoucher`.
  - Lines 28–33: Constructor `func NewMemoryRepository() *MemoryRepository`.
  - Lines 35–80: 45 lines implementing `SaveAgreement`, `GetAgreement`, `GetAgreementByPartyAndSKU`, `SaveSettlementVoucher`, and `ListAgreementsByWarehouse`.
  - Lines 87–99:
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
- **Postgres Repository**: `backend/internal/consignment/repository.go`
  - Lines 18–20: `func NewPostgresRepository(pool *db.Pool) *PostgresRepository`: does not validate `pool == nil`. Methods (e.g. line 23) check `if r.pool == nil { return errors.New("consignment repository: database pool is nil") }`.

#### Callers & Production Initialization
- `backend/internal/api/router.go:272`:
  ```go
  consignmentSvc: consignment.NewService(nil, pool),
  ```
  If `pool == nil`, `consignment.NewService` silently initializes `NewMemoryRepository()` and continues running.

#### Test Usage
- `backend/internal/consignment/consignment_test.go:117`:
  ```go
  func TestConsignmentService_EndToEnd(t *testing.T) {
      ctx := context.Background()
      svc := NewService(nil, nil)
      ...
  }
  ```
  Unit tests call `NewService(nil, nil)` to test agreement registration, receiving, and ownership transfer against the in-memory repository.

---

### 2.2 `internal/rebate`

#### Location & Structure
- **Production File**: `backend/internal/rebate/repository.go`
  - Lines 13–18: Definition of `type MemoryRepository struct` with `mu sync.RWMutex`, `contracts map[string]*ConditionContract`, `accruals map[string][]*ConditionContractAccrual`, `postings map[string][]LedgerPosting`.
  - Lines 20–26: Constructor `func NewMemoryRepository() *MemoryRepository`.
  - Lines 28–64: In-memory implementations for `GetContract`, `ListContractsByParties`, `SaveContract`, `RecordAccrual`.
  - Lines 70–72: `func NewPostgresRepository(pool *db.Pool) *PostgresRepository`: does not validate `pool == nil`.
- **Production File**: `backend/internal/rebate/service.go`
  - Lines 24–36:
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

#### Callers & Production Initialization
- `backend/internal/api/router.go:268`:
  ```go
  rebateSvc: rebate.NewService(nil, pool),
  ```
  Silently falls back to `NewMemoryRepository()` if `pool == nil`.

#### Test Usage
- `backend/internal/rebate/rebate_test.go`:
  Tests pure calculation functions (`AccrueDeliveryRebate`, `SettleConditionContract`, `ReverseUnfulfilledAccrual`).
  Does not currently use `NewService` or `MemoryRepository` directly.

---

### 2.3 `internal/payout`

#### Location & Structure
- **Production File**: `backend/internal/payout/repository.go`
  - Lines 37–43:
    ```go
    // NewRepository returns the primary production PostgreSQL repository.
    func NewRepository(pool *db.Pool) Repository {
        if pool != nil {
            return NewPostgresRepository(pool)
        }
        return NewMemoryRepository(nil)
    }
    ```
  - Lines 292–305: Definition of `type MemoryRepository struct` with `policies`, `batches`, `items` maps, and constructor `func NewMemoryRepository(_ *db.Pool) *MemoryRepository`.
  - Lines 307–398: Over 90 lines of in-memory implementations.
- **Production File**: `backend/internal/payout/service.go`
  - Lines 27–41: `func NewService(repo Repository, rdb *redis.Client, wsHub WebSocketHub, logger *slog.Logger, pool ...*db.Pool) *Service`.
  - Lines 96–126: In `GenerateBatch`, if `items` is empty, it queries the DB only `if len(items) == 0 && s.pool != nil`. If `pool == nil`, it silently bypasses the database query and errors with "no delivered orders found" without notifying that the database pool is missing.

#### Callers & Production Initialization
- `backend/internal/api/router.go:219–220`:
  ```go
  payoutRepo := payout.NewRepository(pool)
  payoutSvc := payout.NewService(payoutRepo, rdb, wsHub, nil, pool)
  ```
  If `pool == nil`, `payout.NewRepository(pool)` returns `NewMemoryRepository(nil)`.

#### Test Usage
- `backend/internal/payout/payout_test.go:114`:
  ```go
  func TestPayoutService_FullLifecycle(t *testing.T) {
      repo := NewMemoryRepository(nil)
      svc := NewService(repo, nil, nil, nil)
      ...
  }
  ```
  Test explicitly instantiates `NewMemoryRepository(nil)` to run the 6-step lifecycle test suite without PostgreSQL.

---

### 2.4 `internal/wmsops`

#### Location & Structure
- **Production File**: `backend/internal/wmsops/repository.go`
  - Lines 118–130: Definition of `type MemoryRepository struct` with 10 in-memory fields: `templates`, `dispatches`, `qcRecords`, `perimeters`, `gateEvents`, `pickers`, `expressConfigs`, `insights`, `inventoryPolicies`, `planWarmRuns`.
  - Lines 132–155: Constructor `func NewMemoryRepository() *MemoryRepository`.
  - Lines 157–782: **625 lines of in-memory methods** (`GetOpsBoard`, `ListTemplates`, `SaveTemplate`, `RecordQCInspection`, `SavePerimeter`, `RecordGateCrossing`, `UpdatePickerLocation`, `SetExpressConfig`, `SaveReplenishmentInsight`, `ApplyInventoryPolicy`, `SavePlanWarmRun`, etc.).
  - Lines 791–799:
    ```go
    func NewPostgresRepository(pool *db.Pool) Repository {
        if pool == nil {
            return NewMemoryRepository()
        }
        return &PostgresRepository{
            pool:    pool,
            pickers: make(map[string]map[string]PickerLocation),
        }
    }
    ```
- **Production File**: `backend/internal/wmsops/service.go`
  - Lines 31–41: `func NewService(repo Repository, rdb *redis.Client, wsHub *ws.Hub, qmSvc *qm.QMService) *Service`.

#### Callers & Production Initialization
- `backend/internal/api/router.go:228–229`:
  ```go
  wmsOpsRepo := wmsops.NewPostgresRepository(pool)
  wmsOpsSvc := wmsops.NewService(wmsOpsRepo, rdb, wsHub, qmService)
  ```
  If `pool == nil`, `wmsops.NewPostgresRepository` returns `NewMemoryRepository()`.

#### Test Usage
- `backend/internal/wmsops/wmsops_test.go:14`:
  ```go
  package wmsops_test
  ...
  func TestWmsOpsSuite(t *testing.T) {
      ctx := context.Background()
      repo := wmsops.NewMemoryRepository()
      qmSvc := qm.NewQMService(qm.NewMemoryQMRepo(), nil)
      wsHub := ws.NewHub(nil)
      svc := wmsops.NewService(repo, nil, wsHub, qmSvc)
      ...
  }
  ```
  `wmsops_test` runs an extensive 9-part test suite completely against `wmsops.NewMemoryRepository()`.

---

## 3. Secondary Packages Audit (Comprehensive Inventory)

Beyond the 4 Milestone-1 packages, our codebase-wide AST and regex scan revealed 36 additional packages with memory stubs, composite fallbacks, or hybrid embedded map seeds.

### 3.1 Dual-Repository & Missing Postgres Packages

| Package | Files | Problem Description | Lines |
| :--- | :--- | :--- | :--- |
| `internal/ewm` | `service.go` | **No PostgresRepository exists**. `type MemoryRepository struct` and `NewMemoryRepository()` defined in `service.go`. `NewService(repo, pool)` always defaults to memory repo. | 22–82, 89–97 |
| `internal/copa` | `service.go` | **No PostgresRepository exists**. `type MemoryRepository struct` and `NewMemoryRepository()` defined in `service.go`. `NewService(repo, pool)` always defaults to memory repo. | 23–96, 103–111 |
| `internal/fscm` | `service.go`, `repository.go` | `service.go` defines `MemoryRepository` (lines 21–63). `NewService` checks `if pool != nil { repo = NewPostgresRepository(pool) } else { repo = NewMemoryRepository() }`. | 21–63, 70–83 |
| `internal/matching` | `repository.go`, `service.go` | `repository.go` defines `MemoryRepository` (lines 14–91). `NewService` checks `if pool != nil { repo = NewPostgresRepository(pool) } else { repo = NewMemoryRepository() }`. | 14–91, 24–36 |
| `internal/qm` | `service.go`, `repository.go` | `service.go` defines `MemoryQMRepo` (lines 21–94). `NewQMService` checks `if pool != nil { repo = NewPostgresQMRepo(pool) } else { repo = NewMemoryQMRepo() }`. | 21–94, 100–112 |

### 3.2 Composite Fallback Repositories (`pgRepository` + `memFallback`)

| Package | Files | Problem Description | Lines |
| :--- | :--- | :--- | :--- |
| `internal/transfer` | `repository.go` | `type pgRepository struct` contains `memFallback *MemoryTransferRepo`. Constructor `NewRepository(pool)` initializes `memFallback`. Every method checks `if r.pool == nil { return r.memFallback... }`. | 45–166, 169–184 |
| `internal/cyclecount` | `repository.go` | `type pgRepository struct` contains `memFallback *MemoryCycleCountRepo`. Every method checks `if r.pool == nil { return r.memFallback... }`. | 47–182, 184–199 |
| `internal/empties` | `service.go` | `type pgEmptiesRepo struct` contains `memFallback *MemoryEmptiesRepo`. Methods catch `if r.pool == nil` AND `errors.Is(err, pgx.ErrNoRows)` or query errors and delegate to `r.memFallback`. | 60–193, 196–224, 349, 370 |

### 3.3 Production Mock Clients & Stubs

| Package | Files | Problem Description | Lines |
| :--- | :--- | :--- | :--- |
| `internal/onec` | `odata.go` | `type StubODataClient struct` and `NewStubODataClient()` declared in production file `odata.go`. Called in `cmd/smokecheck/main.go:4683`. | 139–165 |
| `internal/payroll` | `client.go` | `type StubGlobalPayPayoutClient struct` and `NewStubGlobalPayPayoutClient(initialPoolMinor int64)` declared in production file `client.go`. Injected into production router `router.go:275` with 500M UZS fake balance! | 28–100, `router.go:275` |
| `internal/aiorder` | `voicenote.go` | `type StubAudioTranscriber struct` declared in production file `voicenote.go`. | 125–136 |
| `internal/api` | `middleware_idempotency.go` | `type memoryIdempotencyStore struct` and `var memStore` declared in `middleware_idempotency.go`. Used as fallback when Redis is absent. | 31–38, 170–218 |

### 3.4 Hybrid Repositories with Embedded Fallback Maps & Fake Seeds

The following 19 packages combine a PostgreSQL pool with internal mutex-guarded `map[string]...` fields and execute hardcoded seeding methods inside their production constructors:

| Package | Struct | Fallback Maps & Seed Methods | Hardcoded Sample Fixtures |
| :--- | :--- | :--- | :--- |
| `scheduling` | `Repository` | `policies`, `cutoffs`, `promises`; `seedInitialState()` | `"sup-pepsi-uz"`, `"ZONE-TAS-N"`, `"ZONE-TAS-E"`, `"ord-9001"` |
| `notifications` | `Repository` | `notifications`, `preferences`; `seedInitialState()` | `"@pepsico_tashkent_bot"`, `"+998901234567"` |
| `returns` | `Repository` | `returns`, `sessions`; `seedInitialState()` | `"RET-2026-0301"`, `"wh-tashkent-1"`, `"ord-7001"` |
| `retailer` | `pgRepository` | 20+ maps (`registers`, `shifts`, `drawers`, `cartItems`); `seedInitialState()` | `"reg_korzinka_till_01"`, `"ret_korzinka_001"` |
| `epod` | `Repository` | `stops`, `epods`, `manifest`; `seedInitialState()` | `"man-01"`, `"ord-7001"`, `"Korzinka Chilanzar Core"` |
| `loyalty` | `Repository` | `programs`, `accounts`, `ledger`; `seedInitialState()` | `"sup_pepsico_uz"`, `"PepsiCo Uzbekistan Retailer Advantage Club"` |
| `fxrates` | `Repository` | `rates`, `revaluations`; `seedInitialState()` | Hardcoded scaled CBU exchange rates |
| `crossdock` | `Repository` | `orders`, `scans`; `seedInitialState()` | `"xdo-tash-801"`, `"wh-tashkent-1"`, `"PO-2026-0301"` |
| `promotion` | `Repository` | `promotions`, `tiers`, `redemptions`; `seedInitialState()` | `"sku_pepsi_15l"`, `"cat_beverages"`, `"B2BBUNDLE"` |
| `controltower` | `Repository` | `playbooks`, `exceptions`, `runs`; `seedDefaults()` | Playbook `"10000000-0000-0000-0000-000000000001"` |
| `gs1core` | `Repository` | `prefixes`, `shipUnits`; `seedDefaults()` | Prefix `"4780012"` (Tashkent FMCG Logistics Hub) |
| `coverage` | `Repository` | `zones`; `seedInMemory()` | Zone `"z1111111-1111-1111-1111-111111111101"` (Tashkent South) |
| `fleet` | `Repository` | `vehicles`, `drivers`, `assignments`; `seedInitialData()` | `"veh_isuzu_01"`, `"01 772 AAA"`, `"veh_gazelle_02"` |
| `forecasting` | `Repository` | `forecasts`; `seedInMemory()` | `"f2222222-2222-2222-2222-222222222201"`, `"Coca-Cola Classic 1.5L"` |
| `pickwave` | `Repository` | `waves`, `tasks`; `seedInMemory()` | Waves `"a1111111-..."`, Bay `"BAY-01"`, Seal `"SL-UZ-992144"` |
| `seasonalcore` | `Repository` | `templates`, `overrides`, `profiles`; `seedDefaults()` | Built-in summer beverage surge overrides |
| `commitments` | `Repository` | `commitments`, `preorders`, `waves`; `seedInMemory()` | Commitment `"c1111111-1111-1111-1111-111111111101"` |
| `floorexception` | `Repository` | `exceptions`; `seedInMemory()` | Exception `"e1111111-1111-1111-1111-111111111101"` |
| `ar` | `Repository` | `invoices`, `dunningLogs`, `writeOffs`; `seedDefaults()` | Invoices `"INV-2026-001"`, `"INV-2026-002"` |
| `bins` | `repository` | `memBins`, `memLots`; pre-seeds `initBins` | Bin `"bin-01"`, `"LOC-A01-1A"`, Zone `"A"` |
| `doorstep` | `Repository` | `tokens`, `receipts`, `driverDrawer`, `offloads` | Fallback maps if `r.pool == nil` |
| `order` | `Service` | `inMemoryOrders`, `whPolicies`; `SeedOrderForTesting()` | Fallback maps in `CreateOrder`, `GetOrder`, etc. |
| `inventory` | `Service` | `inMemoryBalances`, `skus`; `initBaseline...()` | Fallback maps in `ReserveStock`, `DeductStock`, etc. |
| `onboarding` | `Service`, `Lifecycle` | `suppliers`, `retailers`, `lifecycles`; `initBaselineProfiles()` | In-memory maps only; `LifecycleManager` never uses pool! |

---

## 4. Root Cause Analysis: Server Boot Degradation

In `pegasus.x/backend/cmd/server/main.go:76–95`:
```go
// 2. Database Connection Pool
pool, err := db.Connect(ctx, cfg.DatabaseURL)
if err != nil {
    log.Printf("[Database WARNING] PostgreSQL connection deferred or failed: %v", err)
} else {
    log.Println("✓ PostgreSQL 16 connection pool established (max 25 conns)")
    defer pool.Close()
    ...
}
```
When `db.Connect` fails, instead of halting execution via `log.Fatalf("[DATABASE FATAL]...")`, the server prints a warning and proceeds to initialize all services with `pool = nil`.

Then in `internal/api/router.go:148–251`:
The constructor `NewServer` receives `pool = nil` and calls:
- `consignment.NewService(nil, pool)` -> falls back to `NewMemoryRepository()`
- `rebate.NewService(nil, pool)` -> falls back to `NewMemoryRepository()`
- `payout.NewRepository(pool)` -> falls back to `NewMemoryRepository(nil)`
- `wmsops.NewPostgresRepository(pool)` -> falls back to `NewMemoryRepository()`
- `fleet.NewRepository(pool)` -> runs `seedInitialData()`
- `retailer.NewRepository(pool)` -> runs `seedInitialState()`
- ... and 20 other services silently spin up in-memory data structures.

This architectural decision to support an in-memory demo mode created a systemic loophole where production services were designed to accommodate `pool == nil`. To achieve Google Principal Engineer caliber, this must be permanently eliminated: **the server must fail closed at boot if PostgreSQL 16 is unavailable, and all production constructors must reject `nil` database pools.**

---

## 5. Concrete Refactoring Plan

### 5.1 Architecture: The Fail-Closed Standard

1. **Production Repositories**:
   - Must only accept `*db.Pool`.
   - Must fail closed: if `pool == nil`, return `(*PostgresRepository, error)` with `errors.New("<package>: database pool is required")`.
   - Must contain ZERO `map[string]...` in-memory fallback fields.
   - Must contain ZERO `seedInitialState()`, `seedInMemory()`, or `seedDefaults()` methods in production files.
2. **Production Services**:
   - Constructor signature: `func NewService(repo Repository, pool *db.Pool) (*Service, error)`.
   - If `repo == nil`:
     - If `pool == nil`: return `nil, errors.New("<package>: database pool or repository is required")`.
     - Automatically construct `repo, err = NewPostgresRepository(pool)`.
   - Contain ZERO in-memory mock instantiation in non-test files.
3. **Mock Implementations (`*_test.go`)**:
   - All `MemoryRepository`, `memoryRepo`, `Stub...` types and their methods are moved to dedicated `mock_repository_test.go` files in the package.
   - For test suites running in `package <pkg>_test` (such as `wmsops_test`), export test helper functions (e.g. `NewMemoryRepository()`) within `mock_repository_test.go` so external tests can instantiate them without leaking into production binaries (`go build`).

---

### 5.2 Refactoring Milestone 1: The 4 Core Packages

#### Phase 1A: `internal/consignment`
1. **`internal/consignment/service.go`**:
   - Delete `type MemoryRepository struct` (lines 22–26).
   - Delete `func NewMemoryRepository()` (lines 28–33).
   - Delete all `(m *MemoryRepository)` methods (lines 35–80).
   - Update `NewService`:
     ```go
     func NewService(repo Repository, pool *db.Pool) (*Service, error) {
         if repo == nil {
             if pool == nil {
                 return nil, errors.New("consignment: database pool or repository is required")
             }
             var err error
             repo, err = NewPostgresRepository(pool)
             if err != nil {
                 return nil, fmt.Errorf("consignment: failed to init repository: %w", err)
             }
         }
         return &Service{
             repo: repo,
             pool: pool,
         }, nil
     }
     ```
2. **`internal/consignment/repository.go`**:
   - Update `NewPostgresRepository`:
     ```go
     func NewPostgresRepository(pool *db.Pool) (*PostgresRepository, error) {
         if pool == nil {
             return nil, errors.New("consignment: database pool is required")
         }
         return &PostgresRepository{pool: pool}, nil
     }
     ```
3. **`internal/consignment/mock_repository_test.go`** *(NEW FILE)*:
   - Move `MemoryRepository` and `NewMemoryRepository()` here, strictly scoped to unit tests.
4. **`internal/consignment/consignment_test.go`**:
   - Line 117: Change `svc := NewService(nil, nil)` to:
     ```go
     repo := NewMemoryRepository()
     svc, err := NewService(repo, nil)
     if err != nil {
         t.Fatalf("failed to create test service: %v", err)
     }
     ```
5. **`internal/api/router.go`**:
   - Line 272: Update service creation:
     ```go
     consignmentSvc, err := consignment.NewService(nil, pool)
     if err != nil {
         log.Fatalf("[FATAL] consignment service: %v", err)
     }
     ```

---

#### Phase 1B: `internal/rebate`
1. **`internal/rebate/repository.go`**:
   - Delete `type MemoryRepository struct` (lines 13–18).
   - Delete `func NewMemoryRepository()` (lines 20–26).
   - Delete all `(m *MemoryRepository)` methods (lines 28–64).
   - Update `NewPostgresRepository`:
     ```go
     func NewPostgresRepository(pool *db.Pool) (*PostgresRepository, error) {
         if pool == nil {
             return nil, errors.New("rebate: database pool is required")
         }
         return &PostgresRepository{pool: pool}, nil
     }
     ```
2. **`internal/rebate/service.go`**:
   - Update `NewService`:
     ```go
     func NewService(repo Repository, pool *db.Pool) (*Service, error) {
         if repo == nil {
             if pool == nil {
                 return nil, errors.New("rebate: database pool or repository is required")
             }
             var err error
             repo, err = NewPostgresRepository(pool)
             if err != nil {
                 return nil, fmt.Errorf("rebate: failed to init repository: %w", err)
             }
         }
         return &Service{
             repo: repo,
             pool: pool,
         }, nil
     }
     ```
3. **`internal/rebate/mock_repository_test.go`** *(NEW FILE)*:
   - Provide `MemoryRepository` and `NewMemoryRepository()` for unit testing.
4. **`internal/api/router.go`**:
   - Line 268: Update service creation:
     ```go
     rebateSvc, err := rebate.NewService(nil, pool)
     if err != nil {
         log.Fatalf("[FATAL] rebate service: %v", err)
     }
     ```

---

#### Phase 1C: `internal/payout`
1. **`internal/payout/repository.go`**:
   - Update `NewPostgresRepository`:
     ```go
     func NewPostgresRepository(pool *db.Pool) (*PostgresRepository, error) {
         if pool == nil {
             return nil, errors.New("payout: database pool is required")
         }
         return &PostgresRepository{pool: pool}, nil
     }
     ```
   - Update `NewRepository`:
     ```go
     func NewRepository(pool *db.Pool) (Repository, error) {
         return NewPostgresRepository(pool)
     }
     ```
   - Delete `type MemoryRepository struct` (lines 292–297).
   - Delete `func NewMemoryRepository()` (lines 299–305).
   - Delete all `(r *MemoryRepository)` methods (lines 307–398).
2. **`internal/payout/service.go`**:
   - Update `NewService`:
     ```go
     func NewService(repo Repository, rdb *redis.Client, wsHub WebSocketHub, logger *slog.Logger, pool ...*db.Pool) (*Service, error) {
         if repo == nil {
             return nil, errors.New("payout: repository is required")
         }
         ...
         return s, nil
     }
     ```
   - Line 96 in `GenerateBatch`:
     ```go
     if len(items) == 0 {
         if s.pool == nil {
             return nil, errors.New("payout: database pool is required to fetch delivered orders")
         }
         // execute query...
     }
     ```
3. **`internal/payout/mock_repository_test.go`** *(NEW FILE)*:
   - Move `MemoryRepository` and `NewMemoryRepository()` here.
4. **`internal/payout/payout_test.go`**:
   - Line 114:
     ```go
     repo := NewMemoryRepository(nil)
     svc, err := NewService(repo, nil, nil, nil)
     if err != nil {
         t.Fatalf("failed to create payout service: %v", err)
     }
     ```
5. **`internal/api/router.go`**:
   - Lines 219–220:
     ```go
     payoutRepo, err := payout.NewRepository(pool)
     if err != nil {
         log.Fatalf("[FATAL] payout repository: %v", err)
     }
     payoutSvc, err := payout.NewService(payoutRepo, rdb, wsHub, nil, pool)
     if err != nil {
         log.Fatalf("[FATAL] payout service: %v", err)
     }
     ```

---

#### Phase 1D: `internal/wmsops`
1. **`internal/wmsops/repository.go`**:
   - Delete `type MemoryRepository struct` (lines 118–130).
   - Delete `func NewMemoryRepository()` (lines 132–155).
   - Delete all `(m *MemoryRepository)` methods (lines 157–782) — 625 lines purged!
   - Update `NewPostgresRepository`:
     ```go
     func NewPostgresRepository(pool *db.Pool) (Repository, error) {
         if pool == nil {
             return nil, errors.New("wmsops: database pool is required")
         }
         return &PostgresRepository{
             pool:    pool,
             pickers: make(map[string]map[string]PickerLocation),
         }, nil
     }
     ```
2. **`internal/wmsops/service.go`**:
   - Update `NewService`:
     ```go
     func NewService(repo Repository, rdb *redis.Client, wsHub *ws.Hub, qmSvc *qm.QMService) (*Service, error) {
         if repo == nil {
             return nil, errors.New("wmsops: repository is required")
         }
         ...
         return &Service{...}, nil
     }
     ```
3. **`internal/wmsops/mock_repository_test.go`** *(NEW FILE)*:
   - Move `MemoryRepository` and `func NewMemoryRepository() *MemoryRepository` into `mock_repository_test.go` in package `wmsops`.
   - Because `mock_repository_test.go` is part of `package wmsops`, `NewMemoryRepository()` is exported during test compilation, allowing `wmsops_test` in `wmsops_test.go` to import and call `wmsops.NewMemoryRepository()`.
4. **`internal/wmsops/wmsops_test.go`**:
   - Lines 14–17:
     ```go
     repo := wmsops.NewMemoryRepository()
     qmSvc, _ := qm.NewQMService(qm.NewMemoryQMRepo(), nil)
     wsHub := ws.NewHub(nil)
     svc, err := wmsops.NewService(repo, nil, wsHub, qmSvc)
     if err != nil {
         t.Fatalf("failed to create wmsops service: %v", err)
     }
     ```
5. **`internal/api/router.go`**:
   - Lines 228–229:
     ```go
     wmsOpsRepo, err := wmsops.NewPostgresRepository(pool)
     if err != nil {
         log.Fatalf("[FATAL] wmsops repository: %v", err)
     }
     wmsOpsSvc, err := wmsops.NewService(wmsOpsRepo, rdb, wsHub, qmService)
     if err != nil {
         log.Fatalf("[FATAL] wmsops service: %v", err)
     }
     ```

---

### 5.3 Roadmap for Secondary Packages (Future Milestones)

1. **`internal/ewm` & `internal/copa`**:
   - Implement `PostgresRepository` mapped to tables created in migration `008_copa_profitability_and_slotting.sql`:
     - `ewm`: `sku_velocity_assignments`, `cross_dock_allocations`
     - `copa`: `copa_drop_profitability`, `retailer_margin_profiles`
   - Move existing `MemoryRepository` to `mock_repository_test.go`.
   - Update `NewService` to require `pool *db.Pool`.
2. **`internal/transfer`, `internal/cyclecount`, `internal/empties`**:
   - Remove `memFallback` struct field from `pgRepository`.
   - Move `MemoryTransferRepo`, `MemoryCycleCountRepo`, and `MemoryEmptiesRepo` to `mock_repository_test.go`.
   - Delete all fallback branches: let Postgres errors propagate directly to caller.
3. **`internal/onec`, `internal/payroll`, `internal/aiorder`**:
   - Move `StubODataClient`, `StubGlobalPayPayoutClient`, and `StubAudioTranscriber` to `*_test.go` files.
   - In `router.go:275`, wire a production `GlobalPayPayoutClient` (or real HTTP client configured from Vault/env) rather than hardcoded mock with 500M fake balance.
4. **Hybrid Repositories (`scheduling`, `fleet`, `retailer`, `returns`, etc.)**:
   - Extract `seedInitialState()` routines out of production files and into database seed files (`database/seeds/`) or `*_test.go` setup fixtures.
   - Remove in-memory fallback maps from repository structs.
   - Ensure all queries target PostgreSQL 16 via `pgxpool`.
5. **`cmd/server/main.go` Fail-Closed Hardening**:
   - Upgrade line 78 from `log.Printf("[Database WARNING]...")` to:
     ```go
     pool, err := db.Connect(ctx, cfg.DatabaseURL)
     if err != nil {
         log.Fatalf("[DATABASE FATAL] PostgreSQL 16 connection failed: %v", err)
     }
     ```
   - This permanently seals the degraded in-memory mode loophole.

---

## 6. Verification & Blast Radius Matrix

| Package | Files Modified | Dependent Packages | Verification Command | Expected Invariants |
| :--- | :--- | :--- | :--- | :--- |
| `consignment` | `service.go`<br>`repository.go`<br>`mock_repository_test.go`<br>`consignment_test.go` | `internal/api/router.go`<br>`cmd/server/main.go` | `go test -v -race ./internal/consignment/...` | 1. Zero `MemoryRepository` in `service.go`.<br>2. `NewService(nil, nil)` returns error.<br>3. All GL postings balance (Debits == Credits). |
| `rebate` | `repository.go`<br>`service.go`<br>`mock_repository_test.go`<br>`rebate_test.go` | `internal/api/router.go`<br>`cmd/server/main.go` | `go test -v -race ./internal/rebate/...` | 1. Zero `MemoryRepository` in `repository.go`.<br>2. `NewService(nil, nil)` returns error.<br>3. Continuous accrual and settlement math pass. |
| `payout` | `repository.go`<br>`service.go`<br>`mock_repository_test.go`<br>`payout_test.go` | `internal/api/router.go`<br>`cmd/server/main.go` | `go test -v -race ./internal/payout/...` | 1. Zero `MemoryRepository` in `repository.go`.<br>2. `NewRepository(nil)` returns error.<br>3. 6-stage lifecycle unit test passes cleanly. |
| `wmsops` | `repository.go`<br>`service.go`<br>`mock_repository_test.go`<br>`wmsops_test.go` | `internal/api/router.go`<br>`cmd/server/main.go` | `go test -v -race ./internal/wmsops/...` | 1. 625 lines of in-memory methods purged from `repository.go`.<br>2. `NewPostgresRepository(nil)` returns error.<br>3. All 9 ops board / QC tests pass in `wmsops_test`. |
| `api` | `internal/api/router.go` | `cmd/server/main.go`<br>`cmd/smokecheck/main.go` | `go test -v -race ./internal/api/...` | Router compiles cleanly with validated constructors. |
