# Handoff Report: Requirement 1 (DevOps and Backend Architecture)

**Task**: Explorer Survey and Investigation for Requirement 1  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_r1`  
**Repository Root**: `/Users/shakhzod/Desktop/V.O.I.D`  
**Handoff Type**: Hard (Complete Investigation & Strategy)  
**Recipient**: Worker / Implementation Agent  

---

## 1. Observation

Direct observations made during codebase inspection:

### 1.1 CI Workflows & `reatilerapp` Typo
- **Workflow Directory**: `.github/workflows/` contains 14 workflow YAML files:
  1. `pegasusx-ci.yml` (Main PegasusX CI: runs unit tests, contract checks, k8s manifest validations, lint, gitleaks, desktop typecheck)
  2. `pegasusx-native-mobile-build.yml` (Android & iOS compilation matrix)
  3. `pegasusx-desktop-build.yml` (Tauri desktop packaging for Linux/Windows)
  4. `pegasusx-docker-push.yml` (Docker build & push to Google Artifact Registry)
  5. `pegasusx-deploy-gke.yml` (GKE deployment triggered by docker push)
  6. `sandbox-infra.yml` (Standalone job running `make test-sandbox-infra` / `scripts/smoke_sandbox.sh`)
  7. `ssmr-infra.yml` (Deprecated alias for `sandbox-infra.yml`)
  8. `reusable-go-unit.yml` (Reusable Go unit test workflow)
  9. `sync-set-guard.yml` (Standalone sync-set guard)
  10. `one-eye-guards.yml` (Standalone one-eye guard suite)
  11. `ci.yml` (Legacy Pegasus monorepo CI)
  12. `desktop-build.yml` (Legacy Tauri desktop build for admin-portal)
  13. `deploy.yml` (Legacy GKE deployment)
  14. `deploy-production.yml` (Legacy production Terraform apply)

- **`reatilerapp` Typo Locations**:
  - `.github/workflows/pegasusx-native-mobile-build.yml`:
    - Line 95: `scheme: reatilerapp`
    - Line 96: `project: pegasusX/apps/retailer-app-ios/retailerapp/reatilerapp.xcodeproj`
  - `.github/ACT.md`:
    - Line 79: `apps/retailer-app-ios/retailerapp/reatilerapp/Services/RetailerWebSocket.swift`
  - **Filesystem Verification**:
    - Project path on disk: `pegasusX/apps/retailer-app-ios/retailerapp/retailerapp.xcodeproj`
    - Scheme name: `retailerapp`

### 1.2 `bootstrap.go` Monolith
- **File**: `pegasusX/apps/backend-go/bootstrap/bootstrap.go`
- **Total Lines**: 2,959 lines (120,064 bytes) in `package bootstrap`
- **Key Structs & Functions**:
  - `Config` struct: lines 90–179
  - `LoadConfig()`: lines 301–386
  - `App` struct: lines 183–297
  - `NewApp(ctx, cfg)`: lines 388–1974
  - `inventoryAdapter`: lines 1976–2028
  - Spanner query closures (`driverHistoryListQuery`, `driverOrderListQuery`, `driverProfileLookupQuery`, `decodeDriverOrderLineItems`, `driverOrderGetQuery`, `driverRouteGeometryQuery`, `driverLoginLookup`, `payloadStaffLoginLookup`, `driverAvailabilityReader`, `warehouseAnalyticsCountQuery`, `warehouseOpsOrdersQuery`, `warehouseOpsDriversQuery`, `warehouseDriversOnActiveManifests`, `warehouseOpsVehiclesQuery`, `supplierDashboardCountQuery`): lines 2031–2650, 2704–2727
  - `notificationReaderAdapter`: lines 2654–2700
  - `App.Close()`: lines 2730–2744
  - `loadSupplierEarningsAuthority` / `sumSupplierEarningsWindow`: lines 2746–2798
  - Env parsing helpers (`envOr`, `seedCurrencyFromPack`, `resolveSpannerEmulatorHost`, `envBool`, `envFloat`, `envInt`, `envInt64`, `shopClosedGraceDuration`, `splitAndTrimCSV`): lines 2800–2913
  - `runtimeSeedRepository` & `existingSeedSupplierCreatedAt`: lines 2915–2958

### 1.3 `spanner.Client.Apply` Callsites
Observed calls to `.Apply(` directly on Spanner client instead of `ReadWriteTransaction`:
- `pegasusX/apps/backend-go/factory/auth_register.go:148`: `s.spannerClient.Apply(r.Context(), muts)` in `HandleFactoryRegister`
- `pegasusX/apps/backend-go/factory/planning_service.go:476`: `p.Spanner.Apply(ctx, muts)` in `ensureSupplyLanes`
- `pegasusX/apps/backend-go/warehouse/auth_register.go:148`: `s.spannerClient.Apply(r.Context(), muts)` in `HandleWarehouseRegister`
- `pegasusX/apps/backend-go/warehouse/setup.go:138`: `s.spannerClient.Apply(r.Context(), mutations)` in `HandleWarehouseSetup`
- `pegasusX/apps/backend-go/warehouse/dispatch_runs.go:45`: `s.spannerClient.Apply(ctx, []*spanner.Mutation{...})` in `persistDispatchRun`
- `pegasusX/apps/backend-go/warehouse/ops_portal.go:606`: `s.spannerClient.Apply(r.Context(), []*spanner.Mutation{m})` in staff creation handler
- Additional callsites in `payload/ship_units.go:181`, `planning/scenarios.go:100`, `planning/accuracy.go:399, 407, 412`, `payment/repository_crud.go:32, 59`, `payment/webhook_inbox.go:48, 116, 133`, `partner/export_spanner.go:48, 124, 195`, `partner/repository_spanner.go:42, 147, 192, 322, 403`.

---

## 2. Logic Chain

1. **CI Consolidation Logic**:
   - `pegasusx-ci.yml` is the primary Gate-0 CI workflow triggered on PRs and pushes to main/develop.
   - Currently, `sandbox-infra.yml` executes `make test-sandbox-infra` as a standalone detached workflow. Incorporating this job directly into `pegasusx-ci.yml` ensures all PRs run the full automated sandbox smoke test alongside existing unit, contract, and k8s validation gates.
   - The typo `reatilerapp` in `pegasusx-native-mobile-build.yml` causes the iOS build step to look for a non-existent project path and scheme (`reatilerapp.xcodeproj` vs actual `retailerapp.xcodeproj`), risking build failure.

2. **`bootstrap.go` Modular Split Logic**:
   - At 2,959 lines, `bootstrap.go` violates single-responsibility and makes dependency tracing difficult.
   - Splitting `bootstrap.go` into `config.go`, `app.go`, `infra.go`, `services.go`, `workers.go`, and `queries.go` within `package bootstrap` preserves all package-level identifiers, struct definitions, and signatures.
   - No external call sites in `main.go`, `cmd/`, or unit tests need modification because all symbols remain in package `bootstrap`.

3. **`spanner.Client.Apply` Migration Logic**:
   - `spanner.Client.Apply` performs unbuffered mutation writes that bypass `outbox.TxnBuffer` event logging.
   - In distributed systems relying on event sourcing / outbox relay (such as PegasusX's Kafka pipeline), state mutations must occur inside a `ReadWriteTransaction` where outbox events are appended transactionally (`outbox.NewSpannerTxnBuffer(txn)` + `outbox.EmitJSON`).
   - Converting the 6 mandatory targets in `factory/` and `warehouse/` eliminates direct `Apply` calls and ensures strict consistency.

---

## 3. Caveats

- **External Packages**: The modular split of `bootstrap.go` must keep the same package name (`package bootstrap`) so that `main.go` and `cmd/` packages continue to compile without import changes.
- **Outbox Event Emits**: In `warehouse/dispatch_runs.go` and `factory/auth_register.go`, if no domain event schema is subscribed for a specific internal logging mutation, the transaction should still execute via `ReadWriteTransaction` with buffer write to maintain transactional safety and allow outbox attachment.

---

## 4. Conclusion

The R1 requirements are fully surveyed with exact file paths, line numbers, and blueprints ready for implementation:
1. **CI**: Consolidate `sandbox-smoke` job into `.github/workflows/pegasusx-ci.yml` and fix `reatilerapp` typos in `pegasusx-native-mobile-build.yml:95-96` and `ACT.md:79`.
2. **Bootstrap**: Split `pegasusX/apps/backend-go/bootstrap/bootstrap.go` into 6 modular files (`config.go`, `app.go`, `infra.go`, `services.go`, `workers.go`, `queries.go`).
3. **Spanner Outbox**: Replace `spanner.Client.Apply` with `ReadWriteTransaction` + `outbox.EmitJSON` across `factory/auth_register.go`, `factory/planning_service.go`, `warehouse/auth_register.go`, `warehouse/setup.go`, `warehouse/dispatch_runs.go`, and `warehouse/ops_portal.go`.

---

## 5. Verification Method

To independently verify the implementation:

1. **CI Workflow Verification**:
   - Inspect `.github/workflows/pegasusx-native-mobile-build.yml` lines 94–97 to ensure `scheme: retailerapp` and `project: pegasusX/apps/retailer-app-ios/retailerapp/retailerapp.xcodeproj`.
   - Inspect `.github/workflows/pegasusx-ci.yml` to ensure `sandbox-smoke` job is present.
   - Grep for `reatilerapp` across the repository (`grep -ri "reatilerapp" .`) — should return 0 results.

2. **Bootstrap Compilation & Tests**:
   - Inspect `pegasusX/apps/backend-go/bootstrap/` to confirm existence of modular files (`config.go`, `infra.go`, `services.go`, `workers.go`, `queries.go`).
   - Run: `cd pegasusX/apps/backend-go && go test ./bootstrap/... -count=1`
   - Run: `cd pegasusX/apps/backend-go && go build ./...`

3. **Spanner `.Apply` Removal Verification**:
   - Run: `grep -rn "\.Apply(" pegasusX/apps/backend-go/factory pegasusX/apps/backend-go/warehouse`
   - Invalidation condition: Any matching line in `factory/auth_register.go`, `factory/planning_service.go`, `warehouse/auth_register.go`, `warehouse/setup.go`, `warehouse/dispatch_runs.go`, or `warehouse/ops_portal.go` indicates failure. Result must be empty for these files.
   - Run: `cd pegasusX/apps/backend-go && go test ./factory/... ./warehouse/... -count=1`
