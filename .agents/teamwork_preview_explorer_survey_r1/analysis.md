# R1 Deep-Dive Investigation Report: DevOps & Backend Architecture

**Date**: 2026-08-21  
**Author**: Explorer (Survey & Architecture Specialist)  
**Target Repository**: `/Users/shakhzod/Desktop/V.O.I.D`  
**Requirement**: R1 (DevOps CI/CD Consolidation, `bootstrap.go` Modular Split, `spanner.Client.Apply` Migration)

---

## 1. Executive Summary & Scope

Requirement 1 addresses three core pillars of backend maintainability, reliability, and continuous delivery in the PegasusX platform:

1. **DevOps & CI/CD Pipeline Consolidation**:
   - Inventorying all 14 GitHub Actions workflows in `.github/workflows/`.
   - Identifying standalone/nested CI jobs (specifically the isolated sandbox smoke gate in `sandbox-infra.yml` / `ssmr-infra.yml` and enterprise progression gates) to consolidate into the canonical root workflow `.github/workflows/pegasusx-ci.yml`.
   - Remediating all occurrences of the `reatilerapp` typo in `.github/workflows/pegasusx-native-mobile-build.yml` and documentation.

2. **Backend Composition Root Decomposition (`bootstrap.go`)**:
   - Analyzing `pegasusX/apps/backend-go/bootstrap/bootstrap.go` (2,959 lines, 120 KB).
   - Designing a clean, zero-regression modular split into domain-focused files within `package bootstrap` (`config.go`, `app.go`, `infra.go`, `services.go`, `workers.go`, `queries.go`).

3. **Atomic Outbox Persistence (`spanner.Client.Apply` → `RunTx` + `outbox.EmitJSON`)**:
   - Locating every blind `.Apply()` callsite across factory, warehouse, and related packages.
   - Defining the exact transaction and event emission refactorings required to enforce transactional consistency and auditability.

---

## 2. Task 1: CI/CD Workflows & Typo Investigation

### 2.1 Monorepo Workflow Inventory

All workflows are located in the root `.github/workflows/` directory:

| Workflow File | Purpose & Triggers | Key Jobs & Targets |
|---|---|---|
| `pegasusx-ci.yml` | **Main PegasusX CI Gate-0** (Push/PR to `main`, `master`, `develop`) | `backend-unit` (via `reusable-go-unit.yml`), `backend` (contracts, k8s), `cell-isolation`, `backend-lint` (golangci-lint, govulncheck), `secrets` (gitleaks), `ai-worker`, `desktop` (portals typecheck/test), `mobile-gate-hint` |
| `pegasusx-native-mobile-build.yml` | **Native Mobile Build** (Android & iOS matrices) | `android` (6 Kotlin apps: retailer, supplier, driver, warehouse, factory, payload), `ios` (6 Swift apps: retailer, supplier, driver, warehouse, factory, payload) |
| `pegasusx-desktop-build.yml` | **Tauri Desktop Release Bundles** (Weekly cron, Push/PR) | `desktop` matrix for Windows (`x86_64-pc-windows-msvc`) and Linux (`x86_64-unknown-linux-gnu`) across 4 portals |
| `pegasusx-docker-push.yml` | **Docker Build & Push to GAR** (Push to `main`/`master`) | Builds and pushes `backend-go` and `ai-worker` Docker images using Workload Identity Federation (WIF) |
| `pegasusx-deploy-gke.yml` | **GKE Continuous Deployment** (Runs on workflow completion of docker-push) | Deploys backend workloads, runs Spanner migration job, performs health checks |
| `sandbox-infra.yml` | **Isolated Sandbox Infra Gate** (PR/Push for `pegasusX/**`) | `sandbox-infra`: runs `make test-sandbox-infra` (`scripts/smoke_sandbox.sh`) |
| `ssmr-infra.yml` | **SSMR Infra Gate (Deprecated Alias)** | Calls `sandbox-infra.yml` |
| `reusable-go-unit.yml` | **Reusable Go Unit Test Workflow** | Reusable `workflow_call` for `go build`, `go vet`, `go test -short`, `go test -race` |
| `sync-set-guard.yml` | **Sync Set Architecture Drift Guard** | Standalone PR guard running `python3 pegasus/scripts/sync_set_guard.py` |
| `one-eye-guards.yml` | **One-Eye Guard Suite** | Standalone PR guard running `python3 pegasus/scripts/sprint1_execution_gate.py` |
| `ci.yml` | **Legacy Monorepo CI** | Legacy Pegasus CI with Spanner emulator & Redis service containers |
| `desktop-build.yml` | **Legacy Desktop Build** | Legacy build for `pegasus/apps/admin-portal` |
| `deploy.yml` | **Legacy Deploy** | Legacy GKE rolling update for `pegasus/infra/k8s` |
| `deploy-production.yml` | **Legacy Production Terraform Apply** | Legacy direct terraform apply workflow |

### 2.2 Nested / Standalone CI Consolidation Strategy

**Current Issue**:
Developers and CI runners currently rely on `sandbox-infra.yml` as a separate detached workflow. Furthermore, `pegasusx-ci.yml` contains a `cell-isolation` job that renders Kustomize manifests and runs isolation proofs, but omits the full automated sandbox smoke test (`make test-sandbox-infra`).

**Consolidation Plan**:
1. Integrate the `sandbox-infra` smoke job directly into `.github/workflows/pegasusx-ci.yml`:
   ```yaml
   sandbox-smoke:
     name: Sandbox — isolated smoke gate
     runs-on: ubuntu-latest
     timeout-minutes: 25
     steps:
       - uses: actions/checkout@v4
       - uses: actions/setup-go@v5
         with:
           go-version: "1.26.0"
           cache: true
           cache-dependency-path: pegasusX/apps/backend-go/go.sum
       - name: Sync Go workspace
         working-directory: pegasusX
         run: go work sync
       - name: Run isolated sandbox smoke gate
         working-directory: pegasusX
         run: make test-sandbox-infra
   ```
2. Retain `sandbox-infra.yml` and `ssmr-infra.yml` as lightweight delegates or mark them as unified under `pegasusx-ci.yml`.

### 2.3 `reatilerapp` Typo Audit & Remediation

Exhaustive pattern search across the codebase revealed the following instances:

1. **Workflow File**: `.github/workflows/pegasusx-native-mobile-build.yml`
   - **Line 95**: `scheme: reatilerapp`
   - **Line 96**: `project: pegasusX/apps/retailer-app-ios/retailerapp/reatilerapp.xcodeproj`
   - **Verification on Disk**:
     - Actual project directory: `pegasusX/apps/retailer-app-ios/retailerapp/retailerapp.xcodeproj`
     - Actual Xcode scheme: `retailerapp`
   - **Fix**:
     ```yaml
     # BEFORE (lines 94-96):
     - name: retailer
       scheme: reatilerapp
       project: pegasusX/apps/retailer-app-ios/retailerapp/reatilerapp.xcodeproj

     # AFTER:
     - name: retailer
       scheme: retailerapp
       project: pegasusX/apps/retailer-app-ios/retailerapp/retailerapp.xcodeproj
     ```

2. **Documentation File**: `.github/ACT.md`
   - **Line 79**: References `apps/retailer-app-ios/retailerapp/reatilerapp/Services/RetailerWebSocket.swift`
   - **Fix**: Correct the path typo to `retailerapp`.

---

## 3. Task 2: `bootstrap.go` Modular Split Architecture

### 3.1 Codebase Assessment

- **File Location**: `pegasusX/apps/backend-go/bootstrap/bootstrap.go`
- **Size**: 2,959 lines (120,064 bytes)
- **Current Responsibilities**:
  1. Configuration struct definition (`Config`) and environment variable parsing (`LoadConfig`, `env*` helpers).
  2. Main composition root struct (`App`) with 70+ fields.
  3. Single monolithic factory function `NewApp(ctx, cfg)` spanning 1,586 continuous lines (lines 388–1974).
  4. Repository & service instantiations across 20+ functional domains.
  5. WebSocket hub pub/sub bindings and Redis caching layers.
  6. 10+ Spanner SQL query closures and adapters for Drivers, Warehouses, Payloads, and Suppliers.
  7. Background worker and Kafka event consumer wiring.
  8. Teardown lifecycle logic (`App.Close()`).

### 3.2 Target Modular Architecture

To maintain 100% backward compatibility and exact compilation behavior without modifying external packages, all new files will remain in `package bootstrap`:

```
pegasusX/apps/backend-go/bootstrap/
├── config.go            # Config struct, LoadConfig(), env parsing helpers
├── app.go               # App struct, NewApp() coordinator, App.Close()
├── infra.go             # Storage, Redis cache, Idempotency, Spanner, Outbox, Routing, FCM
├── services.go          # Domain repositories and services instantiation & cross-wiring
├── workers.go           # Kafka consumers, background workers, replenishment & control tower
├── queries.go           # Driver & Warehouse Spanner query closures and adapters
├── ... (existing helper files: claim_settler.go, claims_bridge.go, config_validate.go, etc.)
```

### 3.3 File-by-File Breakdown & Symbol Mapping

#### 1. `config.go` (~200 lines)
- **Extracted Symbols**:
  - `type Config struct` (lines 90–179)
  - `func LoadConfig() (*Config, error)` (lines 301–386)
  - `func envOr(key, fallback string) string` (line 2800)
  - `func seedCurrencyFromPack() string` (line 2807)
  - `func resolveSpannerEmulatorHost() string` (line 2815)
  - `func envBool(key string, fallback bool) bool` (line 2833)
  - `func envFloat(key string, fallback float64) float64` (line 2845)
  - `func envInt(key string, fallback int) int` (line 2857)
  - `func envInt64(key string, fallback int64) int64` (line 2869)
  - `func shopClosedGraceDuration() time.Duration` (line 2881)
  - `func splitAndTrimCSV(value string) []string` (line 2896)

#### 2. `app.go` (~150 lines)
- **Extracted Symbols**:
  - `type App struct` (lines 183–297)
  - `func NewApp(ctx context.Context, cfg *Config) (*App, error)` (orchestrates phase initializers)
  - `func (a *App) Close()` (lines 2730–2744)

#### 3. `infra.go` (~350 lines)
- **Purpose**: Infrastructure adapters, caching, telemetry, and messaging plumbing.
- **Responsibilities**:
  - GCS storage initialization (`storage.InitGCS`)
  - Redis cache backend and circuit breaker (`cache.NewCircuitBreakerBackendWithMode`, `cache.New`)
  - Idempotency store (`idempotency.NewRedisStore` / `NewInMemoryStore`)
  - Spanner Client & Outbox Store (`tryNewSpannerOutboxStore`, `outbox.NewRelay`)
  - Geometry & Routing engines (`routing.NewGoogleRoutesClient`, `routing.NewOSRMClient`, `routing.NewGeometryBuilder`)
  - Kafka Runtime Publisher & DLQ Writer (`newKafkaRuntimePublisher`, `newKafkaRuntimeDLQWriter`)
  - Push Notification Bridge & FCM Client (`notifications.InitFCM`, `notifications.NewPushBridge`)
  - Seed repository & Supplier bootstrap (`runtimeSeedRepository`, `existingSeedSupplierCreatedAt`)
  - Reliability middleware (`NewReliabilityMiddleware`) & Infra health checks (`buildInfraHealthChecks`)

#### 4. `services.go` (~600 lines)
- **Purpose**: Domain repositories, business services, and cross-domain adapter bindings.
- **Responsibilities**:
  - Core domain repos (Retailer, Supplier, Order, Payment, Credit, AR, Payout, Returns, Warehouse, Factory, Payload, Driver)
  - WebSocket role hubs (`ws.NewHub` for Retailer, Supplier, Driver, Payload, Warehouse, Factory, Telemetry, PlatformAdmin)
  - Inter-service adapters:
    - `inventoryAdapter` (lines 1976–2028)
    - `notificationReaderAdapter` (lines 2654–2700)
    - `loadSupplierEarningsAuthority` / `sumSupplierEarningsWindow` (lines 2746–2798)
  - Service cross-wiring: Order ↔ Payment ↔ AR ↔ Credit ↔ Retailer ↔ Supplier ↔ Warehouse

#### 5. `workers.go` (~450 lines)
- **Purpose**: Asynchronous engines, scheduled workers, and Kafka event consumers.
- **Responsibilities**:
  - Kafka multi-topic consumers:
    - Notification Consumer (`void-notification-dispatcher`)
    - Order Event Consumer (`void-order-mutator`)
    - Warehouse Event Consumer (`void-warehouse-mutator`)
    - Returns Event Consumer (`void-returns-reverse`)
    - Claims Event Consumer (`void-claims-bridge`)
    - Billing Tier Consumer (`void-billing-tier`)
    - Partner Webhook Consumer (`void-partner-webhooks`)
    - Digital Twin Consumer (`void-digital-twin`)
  - Domain workers:
    - AR Dunning Worker (`ar.NewDunningWorker`)
    - Billing Invoice Worker (`billing.NewInvoiceWorker`)
    - Webhook Reconciler (`payment.NewWebhookReconciler`)
    - Buyer Acceptance Poller (`order.NewBuyerAcceptancePoller`)
    - Reorder Suggestion Worker (`replenishment.NewReorderSuggestionWorker`)
    - Route Analytics Worker (`analytics.NewRouteAnalyticsWorkerFromClient`)
    - Control Tower Engine, Executor, & Worker (`controltower.NewEngine`, `controltower.NewWorker`)
    - Partner Delivery, Export, & EDI Inbound/Outbound Workers
    - Forecast Accuracy & Forecast Runner Services

#### 6. `queries.go` (~400 lines)
- **Purpose**: Isolated Spanner SQL query closures and data mapping helpers.
- **Extracted Functions**:
  - `func driverHistoryListQuery(client *spanner.Client) driver.DriverHistoryQuery` (lines 2031–2076)
  - `func driverOrderListQuery(client *spanner.Client) driver.DriverOrderQuery` (lines 2080–2132)
  - `func driverProfileLookupQuery(client *spanner.Client) driver.DriverProfileLookup` (lines 2134–2149)
  - `func decodeDriverOrderLineItems(raw []byte) []driver.DriverOrderLineView` (lines 2151–2174)
  - `func driverOrderGetQuery(client *spanner.Client) driver.DriverOrderGetQuery` (lines 2177–2223)
  - `func driverRouteGeometryQuery(client *spanner.Client, builder *routing.GeometryBuilder) driver.RouteGeometryLookup` (lines 2225–2312)
  - `func driverLoginLookup(client *spanner.Client) driver.DriverLoginLookup` (lines 2376–2408)
  - `func payloadStaffLoginLookup(client *spanner.Client) payload.PayloadStaffLookup` (lines 2410–2442)
  - `func driverAvailabilityReader(client *spanner.Client) driver.AvailabilityReader` (lines 2626–2650)
  - `func warehouseAnalyticsCountQuery(client *spanner.Client) warehouse.WarehouseAnalyticsQuery` (lines 2316–2338)
  - `func warehouseOpsOrdersQuery(client *spanner.Client) warehouse.WarehouseOpsOrdersQuery` (lines 2341–2374)
  - `func warehouseOpsDriversQuery(client *spanner.Client) warehouse.WarehouseOpsDriversQuery` (lines 2445–2536)
  - `func warehouseDriversOnActiveManifests(ctx context.Context, client *spanner.Client, warehouseID string, driverIDs []string) (map[string]bool, error)` (lines 2538–2572)
  - `func warehouseOpsVehiclesQuery(client *spanner.Client) warehouse.WarehouseOpsVehiclesQuery` (lines 2575–2624)
  - `func supplierDashboardCountQuery(client *spanner.Client) supplier.DashboardCountQuery` (lines 2704–2727)

---

## 4. Task 3: `spanner.Client.Apply` Migration Investigation

### 4.1 Problem Analysis & Architecture Rule

Calling `spanner.Client.Apply` directly:
1. **Bypasses Outbox Event Emission**: Mutation occurs without corresponding Kafka/Outbox event entries, causing downstream consumer drift.
2. **Lacks ACID Multi-Entity Coordination**: Cannot coordinate state changes with audit logging or related entities within an atomic transaction.
3. **Breaks Repository Seams**: Violates the established pattern where state mutations must execute within a `ReadWriteTransaction` (`RunTx`) accompanied by `outbox.NewSpannerTxnBuffer(txn)` and `outbox.EmitJSON(...)`.

### 4.2 Complete Callsite Enumeration & Remediation

Below is the comprehensive catalog of `spanner.Client.Apply` callsites identified across the backend codebase, with primary focus on the mandatory Factory and Warehouse targets:

#### Mandatory Target 1: Factory Authentication Register
- **File**: `pegasusX/apps/backend-go/factory/auth_register.go`
- **Line Number**: 148
- **Function**: `HandleFactoryRegister(w http.ResponseWriter, r *http.Request)`
- **Current Code**:
  ```go
  if _, err := s.spannerClient.Apply(r.Context(), muts); err != nil {
      s.log.ErrorContext(r.Context(), "failed to register factory user", "err", err)
      web.JSONError(w, "Failed to register factory user", http.StatusInternalServerError)
      return
  }
  ```
- **Refactoring Strategy**:
  Execute mutations within `s.spannerClient.ReadWriteTransaction`. Create an outbox buffer and emit a `UserRegistered` or `FactoryUserCreated` event if required, ensuring atomic commit.

#### Mandatory Target 2: Factory Planning Service (Supply Lanes)
- **File**: `pegasusX/apps/backend-go/factory/planning_service.go`
- **Line Number**: 476
- **Function**: `ensureSupplyLanes(ctx context.Context, supplierID string) error`
- **Current Code**:
  ```go
  _, err := p.Spanner.Apply(ctx, muts)
  return err
  ```
- **Refactoring Strategy**:
  Use `p.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error { return txn.BufferWrite(muts) })`.

#### Mandatory Target 3: Warehouse Authentication Register
- **File**: `pegasusX/apps/backend-go/warehouse/auth_register.go`
- **Line Number**: 148
- **Function**: `HandleWarehouseRegister(w http.ResponseWriter, r *http.Request)`
- **Current Code**:
  ```go
  if _, err := s.spannerClient.Apply(r.Context(), muts); err != nil {
      s.log.ErrorContext(r.Context(), "failed to register warehouse user", "err", err)
      web.JSONError(w, "Failed to register warehouse user", http.StatusInternalServerError)
      return
  }
  ```
- **Refactoring Strategy**:
  Replace `s.spannerClient.Apply` with `s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error { return txn.BufferWrite(muts) })`.

#### Mandatory Target 4: Warehouse Setup Completion
- **File**: `pegasusX/apps/backend-go/warehouse/setup.go`
- **Line Number**: 138
- **Function**: `HandleWarehouseSetup(w http.ResponseWriter, r *http.Request)`
- **Current Code**:
  ```go
  if _, err := s.spannerClient.Apply(r.Context(), mutations); err != nil {
      s.log.ErrorContext(r.Context(), "failed to complete warehouse setup", "err", err)
      web.JSONError(w, "Failed to complete warehouse setup", http.StatusInternalServerError)
      return
  }
  ```
- **Refactoring Strategy**:
  Wrap the mutation buffer in `s.spannerClient.ReadWriteTransaction`.

#### Mandatory Target 5: Warehouse Dispatch Runs Persistence
- **File**: `pegasusX/apps/backend-go/warehouse/dispatch_runs.go`
- **Line Number**: 45
- **Function**: `persistDispatchRun(ctx context.Context, result DispatchExecuteResult, mode, actorID string)`
- **Current Code**:
  ```go
  _, _ = s.spannerClient.Apply(ctx, []*spanner.Mutation{
      spanner.InsertMap("DispatchRuns", map[string]any{ ... }),
  })
  ```
- **Refactoring Strategy**:
  Migrate to `s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error { return txn.BufferWrite([]*spanner.Mutation{...}) })` and emit `events.TopicMain` dispatch run event.

#### Mandatory Target 6: Warehouse Ops Portal Staff Creation
- **File**: `pegasusX/apps/backend-go/warehouse/ops_portal.go`
- **Line Number**: 606
- **Function**: `HandleOpsStaffCreate` logic inside `ops_portal.go`
- **Current Code**:
  ```go
  if _, err := s.spannerClient.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
      s.log.ErrorContext(r.Context(), "failed to create staff", "err", err)
      writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_create_staff"})
      return
  }
  ```
- **Refactoring Strategy**:
  Replace with `s.spannerClient.ReadWriteTransaction` with buffer write and outbox event emission.

#### Additional Callsites in Related Packages (Catalogued for Completeness):
- `payload/ship_units.go:181`: `client.Apply` → update ship unit state in transaction
- `planning/scenarios.go:100`: `s.Spanner.Apply` → update planning scenario in transaction
- `planning/accuracy.go:399, 407, 412`: `s.Client.Apply` → promote/demote forecast models in transaction
- `payment/repository_crud.go:32, 59`: `r.client.Apply` → update payment records in transaction
- `payment/webhook_inbox.go:48, 116, 133`: `s.client.Apply` → webhook attempt tracking in transaction
- `partner/export_spanner.go:48, 124, 195`: `r.client.Apply` → export jobs persistence
- `partner/repository_spanner.go:42, 147, 192, 322, 403`: `r.client.Apply` → partner API keys and webhooks

---

## 5. Step-by-Step Implementation Strategy for Worker

1. **Step 1: Fix `reatilerapp` Typo**:
   - Edit `.github/workflows/pegasusx-native-mobile-build.yml` lines 95–96.
   - Edit `.github/ACT.md` line 79.

2. **Step 2: Consolidate CI in `.github/workflows/pegasusx-ci.yml`**:
   - Add the `sandbox-smoke` job running `make test-sandbox-infra`.

3. **Step 3: Perform Modular Split of `bootstrap.go`**:
   - Create `pegasusX/apps/backend-go/bootstrap/config.go`.
   - Create `pegasusX/apps/backend-go/bootstrap/infra.go`.
   - Create `pegasusX/apps/backend-go/bootstrap/services.go`.
   - Create `pegasusX/apps/backend-go/bootstrap/workers.go`.
   - Create `pegasusX/apps/backend-go/bootstrap/queries.go`.
   - Simplify `pegasusX/apps/backend-go/bootstrap/app.go` (or slimmed down `bootstrap.go`).
   - Run `go build ./...` across `pegasusX/apps/backend-go` to confirm zero compilation breaks.

4. **Step 4: Migrate `spanner.Client.Apply` in Factory & Warehouse Packages**:
   - Refactor `factory/auth_register.go:148`.
   - Refactor `factory/planning_service.go:476`.
   - Refactor `warehouse/auth_register.go:148`.
   - Refactor `warehouse/setup.go:138`.
   - Refactor `warehouse/dispatch_runs.go:45`.
   - Refactor `warehouse/ops_portal.go:606`.
   - Rebuild and verify tests in `factory/` and `warehouse/`.


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
