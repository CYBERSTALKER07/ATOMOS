# Handoff Report: Track 5 Codebase Audit (Driver, Fleet, Dispatch & Routing Optimization)

## 1. Observation
A comprehensive line-by-line inspection of the Track 5 Go backend codebase was performed across packages `driver/`, `dispatch/`, `manifest/`, `warehouse/`, `telemetry/`, `telemetryroutes/`, `order/`, `routing/`, `eta/`, `geolocation/`, `factory/`, `ws/`, and `schema/spanner.ddl`.

Key verbatim code observations:
1. **`driver/repository_crud.go:89`**:
   `mut := spanner.UpdateStruct("Drivers", d)`
   Called from `HandleUpdateDriver` (`driver/crud_handlers.go:123-128`) where `PinHash` in struct `Driver` is tagged `json:"-"` and therefore always `nil` on unmarshaled JSON. `UpdateStruct` mutates all columns in Spanner, setting `PinHash = NULL` and zeroing unsupplied fields.
2. **`driver/rescue.go:37, 47, 148, 170, 196, 210, 220`**:
   - Line 37: `UPDATE Drivers SET TruckStatus = 'NEEDS_RESCUE' WHERE Id = @id` (Spanner schema has `DriverId`, no `Id`; no `TruckStatus` column on `Drivers`).
   - Line 47: `SELECT SupplierId, AssignedWarehouseId FROM Drivers WHERE Id = @id` (No `AssignedWarehouseId` column; schema has `HomeNodeType`, `HomeNodeId`).
   - Line 170: `SELECT Id, RetailerId FROM Orders WHERE DriverId = @oldDriverId` (No `Id` column; PK is `OrderId`).
   - Line 220: `SELECT LicensePlate FROM Drivers WHERE Id = @id` (No `LicensePlate` on `Drivers`; column is on `Vehicles`).
3. **`order/driver_edges.go:94`**:
   `UPDATE Orders SET SequenceIndex = @seq, UpdatedAt = PENDING_COMMIT_TIMESTAMP() WHERE OrderId = @oid AND RouteId = @rid AND DriverId = @did`
   Table `Orders` in `schema/spanner.ddl:167-206` has no column named `SequenceIndex` (it is on `ManifestOrders:956`).
4. **`telemetryroutes/routes.go:161-185`**:
   Accepts untrusted `loc.NextStopRetailerID` and `loc.NextStopOrderID` from client payload without checking driver assignment, then resolves delivery token and broadcasts it to `retailer:<NextStopRetailerID>`.
5. **`routing/replan.go:58-67`**:
   `outboxMutation` maps columns `"Topic": e.TopicName`, `"PayloadJson": string(e.Payload)` and omits `SupplierId`. `schema/spanner.ddl:679-691` defines `TopicName`, `Payload BYTES(MAX)`, and `SupplierId STRING(64) NOT NULL`.
6. **`eta/service.go:152, 173, 193, 237`**:
   Executes queries against non-existent tables `Routes`, `RouteStops`, `RetailerScores` and non-existent column `DriverScores.TotalOrders`.

## 2. Logic Chain
1. *From Obs 1*: Because `spanner.UpdateStruct` mutates all table columns according to Go struct field values, any omitted field in a partial update request resets the corresponding database column to its Go zero-value. Since `PinHash` is omitted (`json:"-"`), driver PIN hashes are deleted on any profile update, instantly breaking driver login.
2. *From Obs 2*: In Spanner SQL, querying or updating non-existent columns (`Drivers.Id`, `Drivers.TruckStatus`, `Drivers.AssignedWarehouseId`, `Orders.Id`) produces fatal syntax/semantic errors (`Column not found`). Thus, all rescue endpoints in `driver/rescue.go` abort with HTTP 500.
3. *From Obs 3*: In `HandleFleetRouteReorder`, attempting to update `Orders.SequenceIndex` causes Spanner to abort the transaction because `SequenceIndex` belongs to `ManifestOrders`. Stop reordering never persists.
4. *From Obs 4*: Because telemetry endpoints trust arbitrary `NextStopOrderID` and `NextStopRetailerID` from client payloads without verifying driver-manifest assignment, an attacker can leak sensitive QR handoff tokens by submitting arbitrary order IDs.
5. *From Obs 5 & 6*: Dynamic replanning and ETA calculation services fail at runtime due to column name mismatches in `OutboxEvents` and queries against phantom tables (`Routes`, `RouteStops`, `RetailerScores`), preventing continuous routing optimization.

## 3. Caveats
- The audit focused specifically on Track 5 Go backend services and data plane contracts. Client-side Swift (iOS) and Kotlin (Android) apps were checked for wire contract parity against backend endpoints, but native OS-level background GPS battery optimization was not audited.
- External solver sidecars (`services/optimizer-core` Python OR-Tools) were evaluated at the interface/HTTP boundary (`optimizerclient`), not the internal C++ bindings.

## 4. Conclusion
Track 5 has a robust architectural foundation with H3 spatial indexing, Tetris buffer binpacking, OR-Tools integration, and multi-hub WebSockets. However, it contains **19 critical bugs and design defects** (including 7 fatal Spanner SQL/DDL mismatches, PIN wipe vulnerabilities, and delivery token leakage) that will cause 100% runtime failure across rescue operations, route reordering, dynamic replanning, and ETA calculations if deployed without remediation.

All 19 findings, exact code references, blast radiuses, and recommendations are documented in `/Users/shakhzod/Desktop/V.O.I.D/.agents/audit_track5_driver_routing/findings.md`.

## 5. Verification Method
1. Inspect the source lines cited in `findings.md`:
   - View `pegasusX/apps/backend-go/driver/repository_crud.go:89`
   - View `pegasusX/apps/backend-go/driver/rescue.go:37, 47, 148, 170, 220`
   - View `pegasusX/apps/backend-go/order/driver_edges.go:94`
   - View `pegasusX/apps/backend-go/telemetryroutes/routes.go:161-185`
   - View `pegasusX/apps/backend-go/routing/replan.go:58-67`
   - View `pegasusX/apps/backend-go/eta/service.go:152, 173, 193, 237`
2. Cross-reference with `schema/spanner.ddl`:
   - Inspect table `Drivers` (lines 388-406)
   - Inspect table `Vehicles` (lines 412-430)
   - Inspect table `Orders` (lines 167-206)
   - Inspect table `ManifestOrders` (lines 953-964)
   - Inspect table `OutboxEvents` (lines 679-691)
   - Inspect table `DriverScores` (lines 3026-3038)
