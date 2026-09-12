# PegasusX Track 5 Audit: Driver, Fleet, Dispatch & Routing Optimization
**Audit Date**: 2026-08-30  
**Scope**: `apps/backend-go` & `pegasusX/apps/backend-go` (Driver Management, Fleet Lifecycle, Dispatch Engine, Binpacking & Optimization, Telemetry & Geofencing, Routing & ETAs, Spanner Transaction Boundaries, WebSocket Relays)

---

## Executive Summary
A comprehensive, line-by-line source code audit was conducted on Track 5 of the PegasusX Go backend. The audit identified **19 critical and high-severity findings** spanning fatal Spanner schema mismatches, silent authentication wipes on driver updates, broken vehicle capacity computations, SQL compilation failures in route reordering and dynamic replanning, insecure delivery token leakage via telemetry, geofence bypass vulnerabilities, and phantom table references in the ETA recalculation engine.

---

## Detailed Findings by Area

### Area 1: Driver Management, Onboarding, Shifts & Vehicle Assignments

#### Finding 1.1: Driver PIN Wipe and Attribute Zeroing on REST Update
- **Location**: `pegasusX/apps/backend-go/driver/repository_crud.go:89` and `driver/crud_handlers.go:123-128`
- **Flaw**: `HandleUpdateDriver` unmarshals the JSON request body into `req Driver` and calls `r.repo.UpdateDriver(ctx, req, emit)`. `UpdateDriver` executes `spanner.UpdateStruct("Drivers", d)`. In Go struct `Driver`, `PinHash` is marked with `json:"-"`. As a result, `req.PinHash` is always `nil`. `spanner.UpdateStruct` mutates all struct fields in Spanner, setting `PinHash = NULL`, `CreatedAt = 0001-01-01T00:00:00Z`, and resetting boolean flags (`IsActive=false`, `OnShift=false`) to zero values if omitted from the update payload.
- **Blast Radius**: Any administrative edit to a driver's name, phone, or vehicle permanently wipes the driver's PIN hash from Spanner. The driver is immediately locked out and unable to log in via native Android/iOS apps (`POST /v1/auth/driver/login`).
- **Recommendation**: Replace `spanner.UpdateStruct` with `spanner.UpdateMap` containing only the explicitly supplied request fields, or read the existing `Driver` row within a transaction and merge updated fields while preserving `PinHash` and `CreatedAt`.

#### Finding 1.2: Vehicle Volume and Class Overwritten to Zero on Partial Update
- **Location**: `pegasusX/apps/backend-go/driver/repository_crud.go:180` and `driver/crud_handlers.go:294-320`
- **Flaw**: `HandleUpdateVehicle` calls `r.repo.UpdateVehicle(ctx, req, emit)` which executes `spanner.UpdateStruct("Vehicles", v)`. If an operator updates a vehicle's label or status without resending `max_volume_vu` and `vehicle_class`, `MaxVolumeVU` defaults to `0.0` and `VehicleClass` to `""`.
- **Blast Radius**: The vehicle's capacity in Spanner is overwritten to 0 VU. During warehouse dispatch and binpacking, `BuildAvailableFleet` resolves capacity to 0 VU, rendering the vehicle unusable for dispatch.
- **Recommendation**: Use column-selective mutations via `spanner.UpdateMap` for vehicle updates.

#### Finding 1.3: Inability to Set Driver PIN via REST Driver Creation
- **Location**: `pegasusX/apps/backend-go/driver/crud_handlers.go:29-63`
- **Flaw**: `HandleCreateDriver` (`POST /v1/drivers`) unmarshals into `Driver` where `PinHash` is tagged `json:"-"`. There is no PIN field in the request struct, no bcrypt hashing step, and no dedicated endpoint to set a PIN.
- **Blast Radius**: Drivers created via the standard REST CRUD API cannot log in with phone + PIN.
- **Recommendation**: Add a `PIN` string field to driver creation requests, bcrypt-hash it, and persist to `PinHash`.

#### Finding 1.4: Multi-Tenant Data Leak in Cash Reconciliation Listing
- **Location**: `pegasusX/apps/backend-go/driver/cash_bag.go:406-417` and `schema/spanner.ddl:1866-1880`
- **Flaw**: `HandleListCashReconciliations` executes `SELECT ... FROM CashReconciliations WHERE 1=1` without tenant filtering. Furthermore, the `CashReconciliations` table in `schema/spanner.ddl` completely lacks a `SupplierId` column.
- **Blast Radius**: Cash reconciliation records from all suppliers and warehouses are visible to any authenticated warehouse user across tenants.
- **Recommendation**: Add `SupplierId STRING(36) NOT NULL` to `CashReconciliations` table DDL, index by `(SupplierId, CreatedAt DESC)`, and enforce tenant scope filtering in `HandleListCashReconciliations`.

#### Finding 1.5: Duplicate Cash Reconciliation Records on Idempotent Retries
- **Location**: `pegasusX/apps/backend-go/driver/cash_bag.go:162-181`
- **Flaw**: `TurnInCashBag` creates a new UUID `reconID := uuid.NewString()` on every submission without checking if an existing turn-in was already recorded for `(DriverId, ShiftDate)`.
- **Blast Radius**: Network retries from driver mobile apps generate duplicate reconciliation entries for the same shift date.
- **Recommendation**: Enforce uniqueness on `(DriverId, ShiftDate)` or upsert the active shift's reconciliation row.

---

### Area 2: Dispatch Engine, Assignment Logic, Capacity Checks & Routing

#### Finding 2.1: Fatal Schema Mismatches in Rescue Service (100% Runtime Failure)
- **Location**: `pegasusX/apps/backend-go/driver/rescue.go:37, 47, 148, 170, 196, 210, 220`
- **Flaw**: `driver/rescue.go` contains numerous invalid SQL queries that reference non-existent columns and tables:
  1. Line 37: `UPDATE Drivers SET TruckStatus = 'NEEDS_RESCUE' WHERE Id = @id` (columns `TruckStatus` and `Id` do not exist on `Drivers`; PK is `DriverId`).
  2. Line 47 & 148: `SELECT SupplierId, AssignedWarehouseId FROM Drivers WHERE Id = @id` (column `AssignedWarehouseId` does not exist; columns are `HomeNodeType`, `HomeNodeId`).
  3. Line 170: `SELECT Id, RetailerId FROM Orders WHERE DriverId = @oldDriverId ...` (column `Id` does not exist on `Orders`; PK is `OrderId`).
  4. Line 210: `UPDATE Drivers SET TruckStatus = 'BROKEN_DOWN' WHERE Id = @oldDriverId` (invalid column names).
  5. Line 220: `SELECT LicensePlate FROM Drivers WHERE Id = @id` (`LicensePlate` is on `Vehicles`, not `Drivers`).
- **Blast Radius**: `POST /v1/driver/ops/rescue/request` and `POST /v1/driver/ops/rescue/respond` fail 100% of the time with Spanner query compilation errors.
- **Recommendation**: Align all queries to `schema/spanner.ddl` column definitions and join `Vehicles` for vehicle metadata.

#### Finding 2.2: Uncommitted WebSocket Broadcast Inside ReadWriteTransaction
- **Location**: `pegasusX/apps/backend-go/driver/rescue.go:79`
- **Flaw**: `s.driverHub.Broadcast(context.Background(), "fleet_broadcast", payload)` is invoked inside the `client.ReadWriteTransaction` closure before the transaction commits.
- **Blast Radius**: If the transaction fails, encounters lock contention, or retries, phantom WebSocket events are broadcast to connected drivers.
- **Recommendation**: Buffer events via `outbox.EmitJSON` to ensure broadcast occurs only upon commit, or trigger broadcasts after `ReadWriteTransaction` returns successfully.

#### Finding 2.3: Undeclared Struct Fields in Driver Service (`redisClient`, `warehouseHub`)
- **Location**: `pegasusX/apps/backend-go/driver/live_tracking.go:36, 49`, `driver/idempotency_guard.go:78`, `driver/rescue.go:64`
- **Flaw**: Methods reference `s.redisClient` and `s.warehouseHub`, but these fields were omitted from `driver.Service` struct definition in `driver/service.go`.
- **Blast Radius**: Compile-time / package build errors when building driver test packages.
- **Recommendation**: Declare `redisClient *redis.Client` and `warehouseHub *ws.Hub` in `driver.Service` and inject them in `bootstrap/app.go`.

#### Finding 2.4: Zombie Unauthenticated AI Dispatch Sidecar Path
- **Location**: `pegasusX/apps/backend-go/driver/ai_dispatch.go:39-71`
- **Flaw**: `OptimizeFleetRoutes` makes an unauthenticated HTTP call to `http://localhost:8000/fleet-route` using custom `FleetLocation` coordinates (`X`, `Y`), duplicating the canonical Phase 2 optimizer client in `dispatch/plan` and `dispatch/optimizerclient`.
- **Blast Radius**: Code divergence and confusion regarding OR-Tools solver routing.
- **Recommendation**: Remove `driver/ai_dispatch.go` and route all VRP optimization requests through `dispatch/plan/OptimizeAndValidate`.

#### Finding 2.5: Single-Order Oversized Orphan Inability to Split
- **Location**: `pegasusX/apps/backend-go/dispatch/binpack.go:180-214`
- **Flaw**: When `AllowRetailerSplit=true`, the splitting logic groups existing orders by volume. If a *single* order's `VolumeVU` exceeds `maxFleetCap`, it is placed into a chunk by itself. In subsequent vehicle fitting steps (lines 330, 346), no vehicle fits it, and the order is orphaned.
- **Blast Radius**: Bulk orders larger than the fleet's maximum vehicle class (e.g. > 400 VU) can never be dispatched automatically even when splitting is enabled.
- **Recommendation**: Implement line-item/SKU-level order splitting into sub-orders for single orders exceeding maximum truck volume.

---

### Area 3: Live Telemetry, Geofencing, Proof-of-Delivery & Offline Sync

#### Finding 3.1: Fatal SQL Column Mismatch in Route Reordering (`HandleFleetRouteReorder`)
- **Location**: `pegasusX/apps/backend-go/order/driver_edges.go:94`
- **Flaw**: Executes `UPDATE Orders SET SequenceIndex = @seq, UpdatedAt = PENDING_COMMIT_TIMESTAMP() WHERE OrderId = @oid AND RouteId = @rid AND DriverId = @did`. The `SequenceIndex` column does not exist on `Orders` table (`schema/spanner.ddl:167-206`); it exists on `ManifestOrders`.
- **Blast Radius**: When a driver manually reorders waypoints in their app (`POST /v1/fleet/route/reorder`), the request crashes with `Column SequenceIndex not found in table Orders`. Waypoints are never reordered.
- **Recommendation**: Update `ManifestOrders.SequenceIndex` instead of `Orders`.

#### Finding 3.2: Insecure QR Delivery Token Leakage via Telemetry Next-Stop Injection
- **Location**: `pegasusX/apps/backend-go/telemetryroutes/routes.go:161-193`
- **Flaw**: `handleLocation` accepts untrusted `NextStopRetailerID` and `NextStopOrderID` from the client. When within proximity, it resolves the secret QR delivery token via `d.DeliveryTokens.ResolveDeliveryToken(ctx, loc.NextStopOrderID)` and broadcasts `DRIVER_APPROACHING` containing `"delivery_token": deliveryToken` to `retailer:<NextStopRetailerID>`. It never validates that `NextStopOrderID` is actually assigned to `identity.DriverID` or belongs to `NextStopRetailerID`.
- **Blast Radius**: A malicious driver can inject arbitrary order IDs and retailer IDs to steal delivery tokens belonging to other orders and retailers.
- **Recommendation**: Validate ownership: verify in Spanner or cache that `NextStopOrderID` is assigned to `identity.DriverID` and associated with `NextStopRetailerID` before resolving or broadcasting tokens.

#### Finding 3.3: Geofence Verification Bypass by Omitting Coordinates in Delivery Submit
- **Location**: `pegasusX/apps/backend-go/order/service.go:1896-1903`
- **Flaw**: In `SubmitDelivery`, the check is: `if !req.BypassGeofence && (req.Latitude != 0 || req.Longitude != 0)`. If a driver sends `Latitude: 0, Longitude: 0`, geofence validation is skipped without requiring supervisor bypass authorization.
- **Blast Radius**: Drivers can submit fraudulent deliveries from any location by omitting GPS coordinates.
- **Recommendation**: Make coordinates mandatory for `SubmitDelivery` unless `BypassGeofence` is verified with an authentic supervisor override token.

#### Finding 3.4: Euclidean Distortion in Polyline Deviation Calculation
- **Location**: `pegasusX/apps/backend-go/routing/deviation.go:49-64`
- **Flaw**: `distancePointToSegmentMeters` projects GPS coordinates onto polyline segments using raw Euclidean degree math: `dx := end.Lng - start.Lng`, `dy := end.Lat - start.Lat`. Because 1° longitude equals `111 * cos(lat)` km while 1° latitude equals ~111 km, this introduces significant distortion (up to 25–40% at mid/high latitudes).
- **Blast Radius**: Inaccurate off-route detection, resulting in spurious replan triggers or missed route deviations.
- **Recommendation**: Scale longitude differences by `math.Cos(lat * math.Pi / 180)` prior to dot-product projection.

#### Finding 3.5: Unmanaged Background Goroutine in Reassign Handshake
- **Location**: `pegasusX/apps/backend-go/order/reassign_handshake.go:44`
- **Flaw**: Uses `go s.driverHub.Broadcast(context.Background(), "driver:"+sib, b)` to notify sibling drivers without error handling, context cancellation, or worker pool management.
- **Blast Radius**: Leaked goroutines under traffic spikes; dropped messages on container termination.
- **Recommendation**: Route notifications through the transactional outbox or a managed worker pool.

---

### Area 4: Spanner Transaction Boundaries, Lock Contention, Outbox & WebSockets

#### Finding 4.1: Fatal Column Mismatches in Route Replan Outbox Mutation
- **Location**: `pegasusX/apps/backend-go/routing/replan.go:58-67`
- **Flaw**: `outboxMutation` generates an `OutboxEvents` mutation with columns `"Topic": e.TopicName`, `"PayloadJson": string(e.Payload)`, and omits the required NOT NULL column `SupplierId`. `schema/spanner.ddl:679-691` defines columns `TopicName`, `Payload BYTES(MAX)`, and `SupplierId STRING(64) NOT NULL`.
- **Blast Radius**: Dynamic route replanning transactions fail 100% of the time upon commit.
- **Recommendation**: Use `outbox.EventRowMap(e)` to generate valid `OutboxEvents` mutations.

#### Finding 4.2: Phantom Tables and Columns in ETA Recalculation Service
- **Location**: `pegasusX/apps/backend-go/eta/service.go:152, 173, 193-196, 237`
- **Flaw**: `RecalculateRoute` executes queries against non-existent database objects:
  1. Line 152: `SELECT DriverId FROM Routes WHERE Id = @RouteId` (table `Routes` does not exist).
  2. Line 173: `SELECT StopsPerHour, TotalOrders FROM DriverScores ...` (column `TotalOrders` does not exist in `DriverScores`).
  3. Line 193: `SELECT ... FROM RouteStops rs JOIN Retailers r ON rs.RetailerId = r.Id` (table `RouteStops` does not exist; column `Retailers.Id` does not exist, PK is `RetailerId`).
  4. Line 237: `SELECT RetailerId, ShopClosedRate FROM RetailerScores ...` (table `RetailerScores` does not exist).
  5. Line 119-137: `txBuf.BufferOutbox` omits NOT NULL column `SupplierId` when writing to `OutboxEvents`.
- **Blast Radius**: Real-time ETA updates for routes crash with Spanner SQL compilation errors.
- **Recommendation**: Rewrite `RecalculateRoute` to read from `SupplierTruckManifests`, `ManifestOrders`, and `StopTwins`.

#### Finding 4.3: Nanosecond Timestamp Defeating Idempotency in Payment Leg Split
- **Location**: `pegasusX/apps/backend-go/order/driver_edges.go:974, 988`
- **Flaw**: `IdempotencyKey: fmt.Sprintf("split-%s-cash-%d", current.OrderID, now.UnixNano())`. If a Spanner read-write transaction retries due to lock contention, a new `now.UnixNano()` is generated, defeating payment leg deduplication.
- **Blast Radius**: Transaction retry loops or network duplicates can record multiple payment legs for the same delivery.
- **Recommendation**: Construct deterministic idempotency keys, e.g., `fmt.Sprintf("split-%s-%s", current.OrderID, method)`.

#### Finding 4.4: Undefined Status Constant in Order Package
- **Location**: `pegasusX/apps/backend-go/order/service.go:1321`
- **Flaw**: Uses `status = StatusDraft`, which is undefined in `order/service.go` constants.
- **Blast Radius**: Prevents Go backend compilation across touched packages.
- **Recommendation**: Define `StatusDraft Status = "DRAFT"` or use `StatusPending`.

---

## Architectural & Edge-Case Open Questions

1. **Multi-Tenant Fleet Cross-Utilization**:
   - *Question*: In a multi-supplier ecosystem where drivers or third-party logistics (3PL) fleets serve multiple suppliers, how should vehicle capacity and shift schedules be partitioned to prevent double-booking across independent tenant dispatch runs?
   - *Current Gap*: Fleet and vehicle records are strictly keyed by `SupplierId`, preventing shared 3PL capacity pooling across suppliers.

2. **Offline-to-Online State Reconciliation with Concurrent Online Edits**:
   - *Question*: If a driver completes deliveries offline and scans QR tokens while an online retailer cancels or reschedules the order in the portal, how should conflict resolution prioritize physical goods transfer vs financial authorization?
   - *Current Gap*: `HandleSyncBatch` checks `isTerminalMoneyStatus`, but if the order was transitioned to `CANCEL_REQUESTED` or `CANCELLED`, the offline physical delivery conflicts with order cancellation.

3. **High-Frequency Telemetry Ingestion Lock Contention at Scale**:
   - *Question*: With thousands of active drivers reporting GPS pings every 1–3 seconds, writing throttled updates to Spanner `OutboxEvents` (even at 5-second intervals) introduces massive lock contention on the `OutboxEvents` primary key and secondary indexes (`Idx_OutboxEvents_Unpublished`).
   - *Current Gap*: `SpannerLocationBusEmitter` writes standalone mutations directly to Spanner. At 10,000+ drivers, this will saturate Spanner commit slots.
   - *Proposed Architecture*: Telemetry pings should flow purely through Redis Pub/Sub / Kafka partitioned by DriverId, bypassing Spanner entirely for ephemeral GPS data.

4. **Dynamic Multi-Vehicle Rescue Allocations**:
   - *Question*: When a truck breaks down mid-route with 200 VU of cold-chain and dry goods, how should the rescue engine discover N nearby idle drivers, evaluate spare capacity and refrigeration capabilities, and partition the remaining manifest orders into sub-routes without manual dispatcher intervention?
   - *Current Gap*: `driver/rescue.go` assumes a single 1:1 rescue driver assignment and fails to account for vehicle class, temperature compartments, or route replanning.
