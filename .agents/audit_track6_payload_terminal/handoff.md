# Track 6 Audit Handoff Report: Payload, Terminal, IoT & Hardware Domain

## 1. Observation
Directly observed code locations and behaviors in the codebase:
- `pegasusX/apps/backend-go/stocklots/coldchain.go:254-285`: `quarantineManifestLotsInTxn` queries `StockLots WHERE WarehouseId = @wid AND ProductId = @pid AND Status = 'AVAILABLE'` and updates them to `QUARANTINE` when an in-transit manifest experiences a temperature excursion.
- `pegasusX/apps/backend-go/payload/apply.go:15-79`: `apply` queries `ListManifests`, `ListManifestOrders` for every manifest, and `ListExceptions`, then mutates local service state, and writes back `SaveManifest`, `SaveManifestOrder`, and `SaveException` for every single entity of the supplier on every single mutation.
- `pegasusX/apps/backend-go/payload/exceptions.go:51-86`: `HandleDockDamage` executes SQL deleting from non-existent table `LoadLedger`, updates non-existent columns `Orders.TotalPrice` and `Orders.Id`, and inserts string into `OutboxEvents.Payload` (defined as `BYTES(MAX)`).
- `pegasusX/apps/backend-go/payload/fleet_compat.go:66-72`: `HandleFleetReassign` calls `tx.UpdateOrderAssignment` but fails to update `ManifestOrders` or recalculate manifest volumes/stop counts in `SupplierTruckManifests`.
- `pegasusX/apps/payload-terminal/api.ts:251-256`: `reportScanProgress` calls `POST /v1/payload/scan`, which is not registered on `payloaderoutes/routes.go`.
- `pegasusX/apps/payload-app-ios/payload-app-ios/Services/APIClient.swift:268-276`: `sealOrder` calls `POST v1/payload/seal` without passing `manifest_id`, causing backend rejection at `payload/service.go:1977-1983`.
- `pegasusX/apps/backend-go/payload/ship_units.go:181-193`: `insertShipUnit` uses non-transactional `client.Apply` without buffering an Outbox event into `OutboxEvents`.
- `pegasusX/apps/backend-go/telemetryroutes/routes.go:121-124`: Only `POST /v1/telemetry/location` is mounted; sensor telemetry (temperature, humidity, tilt, tamper, shock) has no ingestion routes.

## 2. Logic Chain
1. **Cold-Chain Quarantine Inversion:**
   - Manifest orders represent goods loaded on a truck in transit.
   - Originating warehouse `StockLots` with `Status = 'AVAILABLE'` represent goods physically stored in the warehouse.
   - Therefore, quarantining `AVAILABLE` lots in the warehouse upon a truck excursion falsely locks down warehouse inventory while leaving spoiled transit cargo unflagged.
2. **Apply Scalability & Concurrency Bottleneck:**
   - Spanner ReadWriteTransactions lock read and write key ranges.
   - Reading and rewriting all supplier manifests, orders, and exceptions on every small mutation locks the entire supplier partition and causes `AbortedDueToContention` for any concurrent worker.
   - Retries in Spanner execute the callback multiple times; modifying in-memory slices (`s.manifests`) before commit corrupts memory state on retry.
3. **Contract Desynchronization:**
   - Frontend and native apps call specific endpoint URLs and payloads.
   - The Go backend router mounts different path patterns (`/v1/payloader/manifests/{id}/load-ledger/scan` vs `/v1/payload/scan`).
   - Therefore, client features fail with HTTP 404 or HTTP 400 in production.

## 3. Caveats
- Hardware gateways and smart locker controllers were evaluated based on the existing Go backend codebase and client repositories in `pegasusX/`. Hardware firmware (C/C++/Rust embedded code on microcontrollers) is not part of this repository.
- Simulator integration tests (`simulator/ecosystem_e2e_logic_test.go`) were reviewed to confirm how cold-chain breaches are currently asserted.

## 4. Conclusion
Track 6 has strong foundational UI shells across Expo, Android, and iOS, along with comprehensive manifest state machines. However, the domain is compromised by 4 critical backend flaws: inverted cold-chain inventory quarantines, an unscalable full-table rewrite loop in `apply.go`, dead/broken dock damage code, and route mismatches across native clients. Addressing the 14 documented findings will bring Track 6 into full production reliability.

## 5. Verification Method
1. Inspect the referenced code paths directly:
   - `view_file` on `pegasusX/apps/backend-go/stocklots/coldchain.go` lines 254-285.
   - `view_file` on `pegasusX/apps/backend-go/payload/apply.go` lines 15-80.
   - `view_file` on `pegasusX/apps/backend-go/payload/exceptions.go` lines 50-86.
   - `view_file` on `pegasusX/apps/payload-terminal/api.ts` lines 250-262.
   - `view_file` on `pegasusX/apps/payload-app-ios/payload-app-ios/Services/APIClient.swift` lines 268-276.
2. Run backend test suite for payload and stocklots packages:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go && go test -v ./payload/... ./stocklots/... ./telemetryroutes/...
   ```
