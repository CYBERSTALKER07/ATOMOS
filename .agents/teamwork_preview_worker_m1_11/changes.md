# Milestone 1: Changes Report

**Worker**: `teamwork_preview_worker_m1_11`  
**Mission**: Milestone 1 — Requirement R2 & R1.3: Purge in-memory repository fallbacks and enforce fail-closed constructors across production packages in `pegasus.x/backend`.

---

## 1. Summary of Changes

In accordance with the Universal Enterprise Architecture & Engineering Doctrine (AGENTS.md & GEMINI.md) and Requirement R2 & R1.3 of the Master Modernization Plan:
- Purged all in-memory repository fallbacks (`MemoryRepository`) from production code across 4 core packages: `internal/consignment`, `internal/rebate`, `internal/payout`, `internal/wmsops`.
- Enforced strict fail-closed database constructors in each package: passing a nil `*db.Pool` panics with an explicit error message rather than silently falling back to mock storage.
- Enforced fail-closed behavior on server bootstrap in `cmd/server/main.go`: replaced the non-fatal warning on database connection failure with `log.Fatalf`, ensuring the production server process halts immediately if PostgreSQL is unreachable.
- Migrated all test doubles and mock implementations strictly into `*_test.go` files (`service_test.go` in consignment, `repository_test.go` in rebate, payout, and wmsops; plus `wmsops_mock_test.go` and `payout_mock_test.go` in `internal/api`). These are completely excluded from production binaries.
- Guarded `api.NewServer` service initializations and added testing overrides (`SetWmsOpsService`, `SetPayoutService`) so that isolated HTTP unit tests continue functioning without DB connections while production builds remain strictly bound to PostgreSQL 16.

---

## 2. File Modification Details

### 1. `backend/cmd/server/main.go`
- **Change**: Replaced warning log on database connection failure (lines 78-83) with `log.Fatalf("database connection failed: %v", err)`.
- **Rationale**: Enforces fail-closed server startup. A production server without a valid database connection must never proceed to serve traffic with nil or uninitialized pools.

### 2. `backend/internal/consignment/`
- **`service.go`**:
  - Purged in-memory `MemoryRepository` struct definition and all its methods (`SaveAgreement`, `GetAgreement`, `GetAgreementByPartyAndSKU`, `SaveSettlementVoucher`, `ListAgreementsByWarehouse`) from production source (~60 lines).
  - Removed unused `sync` package import.
  - Updated `NewService(repo, pool)` to fail closed with a panic (`"consignment: repository or database pool is required and cannot both be nil"`) if both `repo` and `pool` are nil.
- **`repository.go`**:
  - Enforced fail-closed guard in `NewPostgresRepository(pool *db.Pool)`: panics with `"consignment: database pool is required and cannot be nil"` if `pool == nil`.
- **`service_test.go`** (New File):
  - Houses the isolated `MemoryRepository` implementation and constructor `NewMemoryRepository()` strictly within test scope.
  - Implements unit tests `TestConsignmentMemoryRepository_CRUD`, `TestNewPostgresRepository_FailClosedOnNilPool`, and `TestNewService_FailClosedOnNilPool`.
- **`consignment_test.go`**:
  - Updated line 117 to instantiate `NewMemoryRepository()` and pass it to `NewService(repo, nil)`.

### 3. `backend/internal/rebate/`
- **`repository.go`**:
  - Purged in-memory `MemoryRepository` struct definition and methods (`GetContract`, `ListContractsByParties`, `SaveContract`, `RecordAccrual`) from production source (~55 lines).
  - Removed unused `sync` package import.
  - Enforced fail-closed guard in `NewPostgresRepository(pool *db.Pool)`: panics with `"rebate: database pool is required and cannot be nil"` if `pool == nil`.
- **`service.go`**:
  - Updated `NewService(repo, pool)` to fail closed with a panic (`"rebate: repository or database pool is required and cannot both be nil"`) if both `repo` and `pool` are nil.
- **`repository_test.go`** (New File):
  - Houses the isolated `MemoryRepository` implementation and constructor `NewMemoryRepository()`.
  - Implements unit tests `TestRebateMemoryRepository_CRUD`, `TestRebateService_WithMemoryRepository`, `TestNewPostgresRepository_FailClosedOnNilPool`, and `TestNewService_FailClosedOnNilPool`.

### 4. `backend/internal/payout/`
- **`repository.go`**:
  - Purged lines 291–398 containing `MemoryRepository` struct and methods (`GetPolicy`, `SavePolicy`, `CreateBatch`, `GetBatch`, `UpdateBatch`, `ListBatches`).
  - Removed unused `sync` package import.
  - Enforced fail-closed guards in `NewPostgresRepository(pool *db.Pool)` and `NewRepository(pool *db.Pool)`: both panic with `"payout: database pool is required and cannot be nil"` if `pool == nil`.
- **`repository_test.go`** (New File):
  - Houses the test-scoped `MemoryRepository` and constructor `NewMemoryRepository(_ *db.Pool)`.
  - Implements unit tests `TestPayoutMemoryRepository_CRUD`, `TestNewPostgresRepository_FailClosedOnNilPool`, and `TestNewRepository_FailClosedOnNilPool`.
- **`payout_test.go`**:
  - Seamlessly runs against `NewMemoryRepository(nil)` located in `repository_test.go`.

### 5. `backend/internal/wmsops/`
- **`repository.go`**:
  - Purged ~665 lines of `MemoryRepository` struct and all 11 of its in-memory methods (`GetOpsBoard`, `ListBroadcastTemplates`, `CreateBroadcastTemplate`, `DeleteBroadcastTemplate`, `SaveBroadcastDispatch`, `ListBroadcastDispatches`, `SaveInboundQCRecord`, `GetInboundQCRecord`, `ListInboundQCRecords`, `SavePerimeter`, `GetPerimeter`, `ListPerimeters`, `SaveGateEvent`, `ListGateEvents`, `SavePickerLocation`, `GetLiveHeatmap`, `GetExpressConfig`, `SaveExpressConfig`, `ListReplenishmentInsights`, `GetReplenishmentInsight`, `SaveReplenishmentInsight`, `UpdateReplenishmentInsightStatus`, `SaveInventoryPolicy`, `GetInventoryPolicy`, `SavePlanWarmRun`, `ListRecentPlanWarmRuns`, `GetResidualFleet`).
  - Removed unused `sort` package import.
  - Enforced fail-closed guard in `NewPostgresRepository(pool *db.Pool)`: panics with `"wmsops: database pool is required and cannot be nil"` if `pool == nil`.
- **`repository_test.go`** (New File):
  - Houses the full test-scoped `MemoryRepository` implementation and constructor `NewMemoryRepository()`.
  - Implements unit tests `TestWmsOpsMemoryRepository_CRUD` and `TestNewPostgresRepository_FailClosedOnNilPool`.
- **`wmsops_test.go`**:
  - Seamlessly runs against `NewMemoryRepository()` located in `repository_test.go`.

### 6. `backend/internal/api/`
- **`router.go`**:
  - Guarded `payoutSvc`, `wmsOpsSvc`, `rebateSvc`, and `consignmentSvc` initializations behind `if pool != nil` inside `NewServer(...)` (lines 219–275).
  - Added test override setters `SetWmsOpsService(svc *wmsops.Service)` and `SetPayoutService(svc *payout.Service)` matching existing patterns (`SetWarehouseService`, `SetDoorstepService`, `SetSupplierService`, `SetPayloadService`).
- **`wmsops_mock_test.go`** (New File):
  - Implements `testWmsOpsMockRepository` conforming to `wmsops.Repository` strictly for headless API tests.
- **`payout_mock_test.go`** (New File):
  - Implements `testPayoutMockRepository` conforming to `payout.Repository` strictly for headless API tests.
- **`retailer_e2e_test.go`**:
  - Added `payout` and `wmsops` imports.
  - In `setupTestServer(t)`: configured `payoutMock` and `wmsopsMock` services via `server.SetPayoutService` and `server.SetWmsOpsService`.

---

## 3. Verification Commands & Results

1. **Race-Detector Test Execution on Affected Packages**:
   ```bash
   cd pegasus.x/backend
   go test -v -race -count=1 ./internal/consignment/... ./internal/rebate/... ./internal/payout/... ./internal/wmsops/...
   ```
   **Result**: 100% PASS (28 unit tests passed, 0 failures, 0 race conditions).

2. **API E2E Test Suite Execution**:
   ```bash
   cd pegasus.x/backend
   go test -count=1 ./internal/api/...
   ```
   **Result**: 100% PASS (`ok github.com/pegasus-x/core/internal/api 7.250s`).

3. **Go Static Analysis**:
   ```bash
   cd pegasus.x/backend
   go vet ./internal/consignment/... ./internal/rebate/... ./internal/payout/... ./internal/wmsops/... ./cmd/server/... ./internal/api/...
   ```
   **Result**: 0 violations, clean exit code 0.

4. **Production Server Binary Compilation**:
   ```bash
   cd pegasus.x/backend
   go build -v ./cmd/server
   ```
   **Result**: Clean compilation, zero mock references in production binary.
