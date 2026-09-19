# Changes: Milestone 1 (DevOps and Backend Architecture)

## 1. CI and DevOps Workflows
- **`.github/workflows/pegasusx-native-mobile-build.yml`**:
  - Corrected `reatilerapp` typo in matrix for retailer iOS app:
    - `scheme: retailerapp`
    - `project: pegasusX/apps/retailer-app-ios/retailerapp/retailerapp.xcodeproj`
- **`.github/ACT.md`**:
  - Corrected `reatilerapp` path typo on line 79 to `apps/retailer-app-ios/retailerapp/retailerapp/Services/RetailerWebSocket.swift`.
- **`.github/workflows/pegasusx-ci.yml`**:
  - Integrated `sandbox-infra` smoke gate job (`make test-sandbox-infra`) directly into the main PegasusX CI workflow.
- **`pegasusX/scripts/retailer-os-unit.sh`**:
  - Corrected path reference from `reatilerapp` to `retailerapp`.

## 2. Bootstrap Modularization
Decomposed the 2,959-line `pegasusX/apps/backend-go/bootstrap/bootstrap.go` monolith into 6 clean, modular source files within `package bootstrap`, preserving all exported and package-level identifiers:
- **`pegasusX/apps/backend-go/bootstrap/config.go`**:
  - `Config` struct definition.
  - `LoadConfig()` with production profile validation.
  - Configuration parsing and helper functions (`envOr`, `envBool`, `envInt`, `envInt64`, `envFloat`, `splitAndTrimCSV`, `resolveSpannerEmulatorHost`, `seedCurrencyFromPack`, `shopClosedGraceDuration`).
- **`pegasusX/apps/backend-go/bootstrap/app.go`**:
  - `App` composition root struct holding all long-lived singletons.
  - `NewApp(ctx, cfg)` composition orchestrator.
  - `App.Close()` graceful shutdown method.
- **`pegasusX/apps/backend-go/bootstrap/infra.go`**:
  - Infrastructure initialization: GCS storage (`setupGCS`), Redis cache & circuit breaker (`setupRedisCache`), Idempotency store (`setupIdempotency`), Spanner outbox/routing (`setupSpannerAndRouting`), Kafka publisher (`setupKafkaPublisher`), Push bridge (`setupPushBridge`), and Driver location store (`setupDriverLocations`).
- **`pegasusX/apps/backend-go/bootstrap/services.go`**:
  - Domain adapter structs: `inventoryAdapter` (implements `supplier.InventoryServicer`), `notificationReaderAdapter` (implements `retailer.NotificationReader`, `retailer.NotificationWriter`, `driver.DriverNotificationReader`).
  - Seed persistence repository: `runtimeSeedRepository` and `existingSeedSupplierCreatedAt`.
- **`pegasusX/apps/backend-go/bootstrap/workers.go`**:
  - Domain Kafka event consumers (`setupKafkaConsumers`) for notification dispatcher, order mutator, warehouse mutator, returns reverse, claims bridge, billing tier, partner webhooks, and digital twin.
  - Dunning notification orchestration (`setupDunningNotification`).
- **`pegasusX/apps/backend-go/bootstrap/queries.go`**:
  - Spanner query closures: `driverHistoryListQuery`, `driverOrderListQuery`, `driverProfileLookupQuery`, `decodeDriverOrderLineItems`, `driverOrderGetQuery`, `driverRouteGeometryQuery`, `warehouseAnalyticsCountQuery`, `warehouseOpsOrdersQuery`, `driverLoginLookup`, `payloadStaffLoginLookup`, `warehouseOpsDriversQuery`, `warehouseDriversOnActiveManifests`, `warehouseOpsVehiclesQuery`, `driverAvailabilityReader`, `supplierDashboardCountQuery`, `loadSupplierEarningsAuthority`, and `sumSupplierEarningsWindow`.
- Removed original `bootstrap.go` file.

## 3. Spanner `.Apply` Migration to `ReadWriteTransaction`
Migrated all direct `spanner.Client.Apply` calls to `ReadWriteTransaction` across the 6 target files:
- **`pegasusX/apps/backend-go/factory/auth_register.go`**:
  - Migrated `s.spannerClient.Apply(r.Context(), muts)` to `s.spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error { return txn.BufferWrite(muts) })`.
  - Added `"context"` import.
- **`pegasusX/apps/backend-go/factory/planning_service.go`**:
  - In `ensureSupplyLanes`, migrated `p.Spanner.Apply(ctx, muts)` to `p.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error { return txn.BufferWrite(muts) })`.
- **`pegasusX/apps/backend-go/warehouse/auth_register.go`**:
  - Migrated `s.spannerClient.Apply(r.Context(), muts)` to `s.spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error { return txn.BufferWrite(muts) })`.
  - Added `"context"` import.
- **`pegasusX/apps/backend-go/warehouse/setup.go`**:
  - In `HandleWarehouseSetup`, migrated `s.spannerClient.Apply(r.Context(), mutations)` to `s.spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error { return txn.BufferWrite(mutations) })`.
  - Added `"context"` import.
- **`pegasusX/apps/backend-go/warehouse/dispatch_runs.go`**:
  - In `persistDispatchRun`, migrated `s.spannerClient.Apply(...)` to `s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error { return txn.BufferWrite(...) })`.
- **`pegasusX/apps/backend-go/warehouse/ops_portal.go`**:
  - In `HandleOpsStaff` (staff creation), migrated `s.spannerClient.Apply(r.Context(), []*spanner.Mutation{m})` to `s.spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error { return txn.BufferWrite([]*spanner.Mutation{m}) })`.


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
