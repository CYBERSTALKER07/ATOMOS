# Review & Adversarial Audit Report: Milestones 2 & 3

**Reviewer**: Reviewer M2_M3_1 (`reviewer`, `critic`)  
**Date**: 2026-09-23T02:54:00+05:00 (UTC: 2026-09-22T21:54:00Z)  
**Target Monorepo**: `pegasus.x` (Sovereign Lean Single-Tenant Core)  
**Subject Under Review**: Worker M2 Handoff (`.agents/worker_m2/handoff.md`) & Worker M3 Handoff (`.agents/worker_m3/handoff.md`)  
**Verdict**: **REQUEST_CHANGES**

---

## Review Summary

**Verdict**: **REQUEST_CHANGES**

Both Worker M2 and Worker M3 demonstrated high engineering quality in the core domain algorithms and business workflows:
- Catch weight financial reconciliations calculate exact 64-bit integer tiyin deltas and emit transactional outbox events.
- E-Factura RFC 5652 CMS SignedData correctly implements standard ASN.1 structures with cryptographic RSA-SHA256 signature verification over encapsulated payloads.
- Order intake auto-vetting (> 600,000 UZS) correctly integrates `PENDING_APPROVAL` states, reserves inventory, and releases stock upon rejection.
- Damaged and returned goods strictly route to `WH-QUARANTINE-01` with `is_atp_excluded = true`.
- Blind receiving reconciles purchase order discrepancies without exposing quantities to operators, persisting shortage claims with integer tiyin totals.
- Payload dock persistence cleanly purged all in-memory mock fallback maps, migrating to direct PostgreSQL 16 `pgxpool.Pool` execution.
- 3L-CVRP longitudinal static moment formulas accurately compute front/rear axle loads, enforce the 11,500 kg statutory single axle limit, maintain $\ge 20\%$ steer tractive authority, and handle cantilever rear tail-lift moments ($x > L$).
- Supervisor override gates enforce 14-digit PINFL, non-empty reason code, bolt seal serial regex (`^SEAL-UZ-[0-9A-Z]{6}$`), and SHA-256 seal hashing.
- Fleet rescue service purged mock seeds (`RSC-2026-081`, etc.) in favor of direct PG16 persistence and dynamic stop transfers via Redis Streams without cancelling customer orders.
- Dispatch pre-flight gates enforce active driver-vehicle pairing, on-shift status, and passing same-day pre-trip DVIR inspections.
- Zero Spanner and zero Kafka references were verified in `pegasus.x/backend`.

**Reason for REQUEST_CHANGES**:
A critical compiler regression was detected during adversarial full-repository testing (`go test ./...`). When Worker M2 expanded `warehouse.Repository` in `backend/internal/warehouse/repository.go` by adding 7 new methods (`EnsureQuarantineBin`, `IsBinATPExcluded`, `SaveBlindScan`, `ListBlindScans`, `SaveShortageClaim`, `ListShortageClaims`, `GetPOExpectedQuantities`), downstream test mock `testWarehouseMockRepository` in `backend/internal/api/warehouse_mock_test.go` was not updated. As a result, `backend/internal/api` fails compilation with:
`internal/api/retailer_e2e_test.go:88:39: cannot use warehouseMock (variable of type *testWarehouseMockRepository) as warehouse.Repository value in argument to warehouse.NewService: *testWarehouseMockRepository does not implement warehouse.Repository (missing method EnsureQuarantineBin)`.

---

## 1. Observation

### Observation 1.1: Test Suite Execution of Owned Packages (PASS)
Executed command:
```bash
go test -count=1 -v -race ./internal/supplier/... ./internal/warehouse/... ./internal/order/... ./internal/qm/... ./internal/soliq/... ./internal/payload/... ./internal/dispatch/... ./internal/fleet/...
```
Verbatim result:
```
ok  	github.com/pegasus-x/core/internal/supplier 	2.784s
ok  	github.com/pegasus-x/core/internal/warehouse	3.958s
ok  	github.com/pegasus-x/core/internal/order    	1.206s
ok  	github.com/pegasus-x/core/internal/qm       	1.194s
ok  	github.com/pegasus-x/core/internal/soliq    	1.686s
ok  	github.com/pegasus-x/core/internal/payload  	2.290s
ok  	github.com/pegasus-x/core/internal/dispatch 	1.211s
ok  	github.com/pegasus-x/core/internal/fleet    	3.947s
```
All unit and race tests in the explicitly specified packages pass cleanly.

### Observation 1.2: Monorepo Full Test Execution (FAIL)
Executed command:
```bash
cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go test ./...
```
Verbatim failure:
```
# github.com/pegasus-x/core/internal/api_test [github.com/pegasus-x/core/internal/api.test]
internal/api/retailer_e2e_test.go:88:39: cannot use warehouseMock (variable of type *testWarehouseMockRepository) as warehouse.Repository value in argument to warehouse.NewService: *testWarehouseMockRepository does not implement warehouse.Repository (missing method EnsureQuarantineBin)
FAIL	github.com/pegasus-x/core/internal/api [build failed]
```

### Observation 1.3: Interface Contract Drift Location
- In `backend/internal/warehouse/repository.go` (lines 40–48), `warehouse.Repository` interface was extended:
  ```go
  type Repository interface {
      ...
      // Quarantine Segregation & ATP Exclusion
      EnsureQuarantineBin(ctx context.Context, warehouseID string) error
      IsBinATPExcluded(ctx context.Context, warehouseID, locationCode string) (bool, error)

      // Blind Receiving & Variance Reconciliation
      SaveBlindScan(ctx context.Context, scan BlindPalletScan) error
      ListBlindScans(ctx context.Context, poID string) ([]BlindPalletScan, error)
      SaveShortageClaim(ctx context.Context, claim ShortageClaim) error
      ListShortageClaims(ctx context.Context, poID string) ([]ShortageClaim, error)
      GetPOExpectedQuantities(ctx context.Context, poID string) (map[string]POItemExpectation, error)
  }
  ```
- In `backend/internal/api/warehouse_mock_test.go` (lines 16–303), `testWarehouseMockRepository` implements only the previous methods (`CreateWarehouse`, `GetWarehouseByID`, `GetWarehouseByTaxID`, `GetWarehouseByPhone`, `UpdateOnboardingStatus`, `UpdateApprovalSettings`, `SaveDockBays`, `ListDockBays`, `SaveWarehouseBins`, `ListWarehouseBins`, `SaveStockLots`, `ListStockLots`, `RecordAuditEvent`, `CountBays`, `CountBins`, `CountStockLots`). It completely lacks implementations for the 7 new methods.
- At `backend/internal/api/retailer_e2e_test.go:88`:
  ```go
  warehouseMock := newTestWarehouseMockRepository()
  warehouseSvc := warehouse.NewService(warehouseMock, nil) // Compile error: does not implement warehouse.Repository
  ```

### Observation 1.4: Zero Spanner and Zero Kafka Verification (PASS)
- Scanned `pegasus.x/backend` for `cloud.google.com/go/spanner` imports: 0 occurrences.
- Scanned `pegasus.x/backend` for Kafka driver imports (`sarama`, `kafka-go`, `confluent-kafka-go`): 0 occurrences.
- The string `spanner` only appears in `backend/internal/db/migration_074_test.go:319` (as a test assertion keyword verifying spanner is NOT present) and `cmd/smokecheck/main.go:2796` (in a trace span name).
- The string `kafka` only appears in `backend/internal/db/migration_074_test.go:320` (as a test assertion keyword verifying kafka is NOT present).

### Observation 1.5: Milestone 2 Code Inspection (PASS)
- Catch Weight:
  - `backend/internal/supplier/models.go:326-373`: `CalculateCatchWeightAdjustment` enforces positive nominal/actual weights, tolerance 0–100%, and computes exact integer tiyin adjustments (`round(actual * price) - round(expected * price)`).
  - `backend/internal/order/catch_weight.go:58-250`: `RecordOutboundCatchWeight` acquires `FOR UPDATE` row locks on `order_items` and `orders`, validates `is_catch_weight`, adjusts order total, and emits outbox event `order.catch_weight_adjusted` in the same `RunInTx` transaction.
- E-Factura RFC 5652 CMS SignedData:
  - `backend/internal/soliq/eimzo.go:265-594`: RFC 5652 ASN.1 structures (`CMSContentInfo`, `CMSSignedData`, `CMSEncapsulatedContentInfo`, `CMSSignerInfo`), valid DER encoding, certificate window checks, signer INN checks, and RSA PKCS#1 v1.5 SHA-256 signature verification.
- Warehouse Auto-Approval & Vetting:
  - `backend/internal/order/service.go:245-288, 510-605`: Evaluates `WarehouseVettingPolicy`. If `effectiveTotal > threshold` (default 60,000,000 tiyins = 600,000 UZS) or buyer is first-time or credit-blocked, sets `StatusPendingApproval` with `needs_vetting = true`. Stock is reserved.
  - `backend/internal/order/service.go:1175-1390`: `ApproveVettedOrder` transitions order to `CONFIRMED` and emits `order.vetting_approved`. `RejectVettedOrder` transitions order to `CANCELLED`, releases stock reservations from `stock_balances` via `GREATEST(0, reserved_qty - $1)`, and emits `order.vetting_rejected`.
- Quarantine Segregation:
  - `backend/internal/qm/quarantine.go:13`: `CanonicalQuarantineBin = "WH-QUARANTINE-01"`.
  - `backend/internal/qm/repository.go`: `PostgresQMRepo` persists lots in `qm_quarantine_lots` with `is_atp_excluded = true`.
  - `backend/internal/warehouse/service.go:467-481`: `EnsureQuarantineBin` and `IsBinATPExcluded` enforce quarantine segregation.
- Blind Receiving Variance Reconciliation:
  - `backend/internal/warehouse/service.go:483-616`: `RecordBlindPalletScan` and `ReconcileBlindReceiving` record scans into `inbound_blind_scans`, fetch PO items via `GetPOExpectedQuantities`, detect shortages, generate `ShortageClaim` records with 64-bit integer tiyin values, and emit outbox events `inbound.shortage_claim_created` and `inbound.blind_reconciled`.
- Migration 076:
  - `database/migrations/076_supplier_catch_weight_and_order_vetting.sql`: Valid PostgreSQL 16 schema adding `PENDING_APPROVAL` to `order_status` enum, `needs_vetting` to `orders`, catch weight columns to `products` and `order_items`, and creating `inbound_blind_scans` and `inbound_shortage_claims`.

### Observation 1.6: Milestone 3 Code Inspection (PASS)
- Zero Mock Purge in Payload:
  - `backend/internal/payload/repository.go`: Contains no in-memory maps or `seedInitialDockData()`. All queries execute against PostgreSQL 16 (`r.pool.QueryRow`, `r.pool.Query`, `r.pool.Exec`). Test mock isolated to `mock_repository_test.go` (strictly `_test.go`).
- 3L-CVRP Longitudinal Statics:
  - `backend/internal/payload/service.go:293-425`: Implements static moment equilibrium:
    $W_{front} = W_{curb,front} + \sum w_i (L - x_i)/L$, $W_{rear} = W_{curb,rear} + \sum w_i x_i / L$.
    Steer ratio $W_{front}/(W_{front} + W_{rear}) \ge 20\%$. Single axle maximum $\le 11,500$ kg.
    Correctly models cantilever rear load ($x > L$) as negative front axle load.
- Supervisor Override & Bolt Seal:
  - `backend/internal/payload/service.go:555-660`: Validates 14-digit PINFL regex `^[0-9]{14}$`, reason code non-empty, bolt seal regex `^SEAL-UZ-[0-9A-Z]{6}$`. Generates SHA-256 digital seal digest.
- Fleet Rescue Zero Mock Purge & Hot-Swap:
  - `backend/internal/dispatch/fleet_rescue_service.go`: Purged `rescueIncidents` map and `initRescueIncidents()`. Uses `RescueStore` interface backed by PostgreSQL 16 table `fleet_rescue_incidents`.
  - `backend/internal/dispatch/service.go:575-765`: `ExecuteRescue` cancels broken manifest, marks broken driver `OFFLINE`, releases vehicle to `MAINTENANCE` with `BREAKDOWN_MID_SHIFT`, creates rescue manifest, inserts transferred stops into `manifest_stops`, logs records into `manifest_stop_transfers`, reassigns orders (`LOADED`, `DISPATCHED`) with ZERO order cancellation, updates `fleet_rescue_incidents`, emits outbox event, and publishes durable event to Redis Streams (`XADD`) topic `events:fleet:rescue_dispatched`.
- Pre-trip DVIR Gating:
  - `backend/internal/dispatch/service.go:301-347`: Enforces active pairing (`driver_vehicle_assignments`), on-shift driver (`drivers.on_shift = true`), and passing pre-trip DVIR today (`vehicle_inspections.is_safe_to_operate = true`).
- Migration 075:
  - `database/migrations/075_rescue_telemetry_and_diagnostics.sql`: Extends `fleet_rescue_incidents` with telemetry columns and creates `manifest_load_lines`, `manifest_exceptions`, and `gs1_ship_units`.

### Observation 1.7: Untracked Compiled Binaries
- The following untracked compiled binary files were left behind in the repository:
  - `backend/server` (27 MB executable)
  - `backend/smokecheck` (27 MB executable)

---

## 2. Logic Chain

1. **Premise**: In accordance with V.O.I.D. Universal Engineering Doctrine (GEMINI.md / AGENTS.md §3), all modifications must pass regression testing across affected targets and direct dependents. Live code is the sole Source of Truth; breaking downstream tests invalidates complete monorepo health.
2. **From Observation 1.3**: Worker M2 legitimately added 7 required methods to `warehouse.Repository` to support quarantine bin checking and blind receiving variance reconciliation in `internal/warehouse/service.go`.
3. **From Observation 1.3 & 1.2**: In Go, interfaces are satisfied implicitly. When an exported interface adds methods, any concrete type passed as that interface must implement all methods. `testWarehouseMockRepository` in `internal/api/warehouse_mock_test.go` is passed as `warehouse.Repository` to `warehouse.NewService(warehouseMock, nil)` at line 88 of `internal/api/retailer_e2e_test.go`.
4. **From Observation 1.2**: Because Worker M2 did not add mock implementations for these 7 methods to `testWarehouseMockRepository`, compilation of `internal/api.test` fails, breaking `go test ./...`.
5. **Conclusion**: Even though the packages directly modified by Worker M2 and Worker M3 pass their localized test command, the interface modification caused contract drift in consuming test code, breaking the monorepo test build. Per reviewer instructions ("do NOT fix them yourself"), this failure must be surfaced as a finding and changes requested.

---

## 3. Findings

### [Critical] Finding 1: Monorepo Compile Failure — `testWarehouseMockRepository` Missing 7 Methods of `warehouse.Repository`

- **What**: Interface contract drift causing build failure of `internal/api` tests during `go test ./...`.
- **Where**: `backend/internal/api/warehouse_mock_test.go:16-303` (instantiated and passed at `backend/internal/api/retailer_e2e_test.go:88`).
- **Why**: Worker M2 added 7 methods to `warehouse.Repository` in `backend/internal/warehouse/repository.go:40-48`:
  1. `EnsureQuarantineBin(ctx context.Context, warehouseID string) error`
  2. `IsBinATPExcluded(ctx context.Context, warehouseID, locationCode string) (bool, error)`
  3. `SaveBlindScan(ctx context.Context, scan BlindPalletScan) error`
  4. `ListBlindScans(ctx context.Context, poID string) ([]BlindPalletScan, error)`
  5. `SaveShortageClaim(ctx context.Context, claim ShortageClaim) error`
  6. `ListShortageClaims(ctx context.Context, poID string) ([]ShortageClaim, error)`
  7. `GetPOExpectedQuantities(ctx context.Context, poID string) (map[string]POItemExpectation, error)`
  `testWarehouseMockRepository` does not implement any of these 7 methods, causing Go compilation failure:
  `internal/api/retailer_e2e_test.go:88:39: cannot use warehouseMock (variable of type *testWarehouseMockRepository) as warehouse.Repository value in argument to warehouse.NewService: *testWarehouseMockRepository does not implement warehouse.Repository (missing method EnsureQuarantineBin)`.
- **Suggestion**: Add the 7 missing methods to `testWarehouseMockRepository` in `backend/internal/api/warehouse_mock_test.go`:
  ```go
  func (m *testWarehouseMockRepository) EnsureQuarantineBin(ctx context.Context, warehouseID string) error {
      return nil
  }
  func (m *testWarehouseMockRepository) IsBinATPExcluded(ctx context.Context, warehouseID, locationCode string) (bool, error) {
      if strings.EqualFold(strings.TrimSpace(locationCode), warehouse.CanonicalQuarantineBin) {
          return true, nil
      }
      return false, nil
  }
  func (m *testWarehouseMockRepository) SaveBlindScan(ctx context.Context, scan warehouse.BlindPalletScan) error {
      return nil
  }
  func (m *testWarehouseMockRepository) ListBlindScans(ctx context.Context, poID string) ([]warehouse.BlindPalletScan, error) {
      return nil, nil
  }
  func (m *testWarehouseMockRepository) SaveShortageClaim(ctx context.Context, claim warehouse.ShortageClaim) error {
      return nil
  }
  func (m *testWarehouseMockRepository) ListShortageClaims(ctx context.Context, poID string) ([]warehouse.ShortageClaim, error) {
      return nil, nil
  }
  func (m *testWarehouseMockRepository) GetPOExpectedQuantities(ctx context.Context, poID string) (map[string]warehouse.POItemExpectation, error) {
      return nil, nil
  }
  ```

### [Minor] Finding 2: Untracked Compiled Binaries in Monorepo

- **What**: Compiled binary files `backend/server` and `backend/smokecheck` were left untracked in `pegasus.x/backend/`.
- **Where**: `backend/server`, `backend/smokecheck`.
- **Why**: Binaries pollute git status and violate repository cleanliness standards.
- **Suggestion**: Delete `backend/server` and `backend/smokecheck` (or add them to `.gitignore`).

---

## 4. Adversarial Challenges & Stress Testing (Critic Role)

### Challenge 1: Cantilever Load Effect on Steer Axle (PASS)
- **Assumption Challenged**: Static moment formula must accurately model rear cantilever overhang ($x_i > L$, such as heavy tail-lifts or pallets placed behind the rear axle) by deducting load from the front steer axle.
- **Stress Test**: In `CalculateAxleFeasibility`, front load share is computed as $w_i \cdot (L - x_i) / L$. For $x_i > L$, $(L - x_i) < 0$, making `frontCargo` negative and directly reducing $W_{steer}$. This correctly models the front-wheel lifting moment.
- **Result**: PASS.

### Challenge 2: RFC 5652 CMS SignedData Tamper Resistance (PASS)
- **Assumption Challenged**: Manipulating payload bytes inside a DER CMS container must fail signature verification closed.
- **Stress Test**: Corrupted the encapsulated payload bytes via bitwise XOR in `eimzo_cms_test.go:126-136`. `VerifySignedDataCMS` caught the digest mismatch and failed closed with `ErrInvalidSignature`.
- **Result**: PASS.

### Challenge 3: Financial Precision & Tiyin Invariants (PASS)
- **Assumption Challenged**: Catch weight adjustments and shortage claims must strictly use 64-bit integer tiyins without floating-point currency representation.
- **Stress Test**: Inspected all calculations in `supplier.CalculateCatchWeightAdjustment`, `order.RecordOutboundCatchWeight`, and `warehouse.ReconcileBlindReceiving`. Calculations use `math.Round` converted immediately to `int64` tiyins. Order totals in PostgreSQL are updated via atomic arithmetic `gross_total_minor = gross_total_minor + $1`.
- **Result**: PASS.

### Challenge 4: Zero Order Cancellation on Fleet Rescue (PASS)
- **Assumption Challenged**: Hot-swapping stops from a broken truck to a rescuer vehicle must reassign orders without cancelling customer orders.
- **Stress Test**: Inspected `ExecuteRescue` in `dispatch/service.go`. The broken manifest is cancelled, but customer orders are updated to `driver_id = rescueDriverID`, `vehicle_id = rescueVehicleID`, `status = LOADED`, `dispatch_status = DISPATCHED`. Orders remain active, stops are recorded in `manifest_stop_transfers`, and the rescuer's manifest is updated.
- **Result**: PASS.

---

## 5. Verified Claims

| Milestone | Claim | Verification Method | Status |
| :--- | :--- | :--- | :--- |
| **M2** | Catch weight adjustment in 64-bit integer tiyins | Inspected `supplier/models.go:326`, `order/catch_weight.go:58`; unit tests passed | **VERIFIED** |
| **M2** | Outbox event `order.catch_weight_adjusted` paired in `pgx.Tx` | Inspected `order/catch_weight.go:214-227` | **VERIFIED** |
| **M2** | RFC 5652 CMS SignedData container DER ASN.1 | Inspected `soliq/eimzo.go:265-594`; tested via `eimzo_cms_test.go` | **VERIFIED** |
| **M2** | Warehouse auto-approval threshold (> 600,000 UZS) | Inspected `order/service.go:245-288`, `order_vetting_and_catch_weight_test.go` | **VERIFIED** |
| **M2** | Quarantine segregation `WH-QUARANTINE-01` & `is_atp_excluded` | Inspected `qm/quarantine.go:13`, `qm/repository.go`, `warehouse/service.go:467` | **VERIFIED** |
| **M2** | Blind receiving variance & shortage claims in tiyins | Inspected `warehouse/service.go:483-616`, `blind_receiving_and_quarantine_test.go` | **VERIFIED** |
| **M2** | Migration `076_supplier_catch_weight_and_order_vetting.sql` | Inspected migration SQL file, valid syntax | **VERIFIED** |
| **M3** | Payload repository zero mock purge | Inspected `payload/repository.go`; mock isolated to `mock_repository_test.go` | **VERIFIED** |
| **M3** | 3L-CVRP longitudinal static moment formula | Inspected `payload/service.go:293-425`; tested via `payload_test.go` | **VERIFIED** |
| **M3** | 11,500 kg single axle limit & $\ge 20\%$ steer ratio | Inspected `payload/models.go:36-39`, `payload/service.go:372-401` | **VERIFIED** |
| **M3** | Supervisor override PINFL, reason, bolt seal regex, SHA-256 | Inspected `payload/service.go:555-660`; tested via `TestPayload_AxleOverride` | **VERIFIED** |
| **M3** | Fleet rescue zero mock purge | Inspected `dispatch/fleet_rescue_service.go`; mock isolated to `fleet_rescue_mock_test.go` | **VERIFIED** |
| **M3** | Mid-shift rescue hot-swap via Redis Streams `XADD` | Inspected `dispatch/service.go:575-765`; `manifest_stop_transfers` recorded | **VERIFIED** |
| **M3** | Pre-trip DVIR gating before dispatch | Inspected `dispatch/service.go:301-347` (`CommitDispatch`) | **VERIFIED** |
| **M3** | Migration `075_rescue_telemetry_and_diagnostics.sql` | Inspected migration SQL file, valid syntax | **VERIFIED** |
| **M2/M3** | Zero Spanner & Zero Kafka references in `pegasus.x/backend` | Automated AST/grep across all Go source files | **VERIFIED** |

---

## 6. Caveats

- **Scale Hardware Drivers**: Integration with dock scale hardware is abstracted at the API boundary; certified scale weights and scale certifier IDs are submitted via authenticated API requests.
- **Sidecar CVRP**: In environments where the Python OR-Tools sidecar is offline, the system safely falls back to the native Go binpacking engine.

---

## 7. Conclusion

The domain implementations for both Milestone 2 and Milestone 3 are architecturally exceptional, fully aligned with the V.O.I.D. doctrine, and free of mocks in production code. However, because Worker M2's changes to `warehouse.Repository` broke the test build of `internal/api` (`testWarehouseMockRepository`), monorepo-wide test execution (`go test ./...`) fails.

**Verdict**: **REQUEST_CHANGES**

Once Worker M2 updates `backend/internal/api/warehouse_mock_test.go` with the 7 stub methods and deletes the untracked binaries `backend/server` and `backend/smokecheck`, the solution will be 100% clean for unconditional approval.

---

## 8. Verification Method

To independently verify this review and reproduce the findings:

1. **Reproduce the Package Test Success**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go test -count=1 -v -race ./internal/supplier/... ./internal/warehouse/... ./internal/order/... ./internal/qm/... ./internal/soliq/... ./internal/payload/... ./internal/dispatch/... ./internal/fleet/...
   ```
   *(Expected: PASS across all 8 packages)*

2. **Reproduce the Monorepo Build Failure**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go test -v ./internal/api/...
   ```
   *(Expected: Build error: `*testWarehouseMockRepository does not implement warehouse.Repository (missing method EnsureQuarantineBin)`)*

3. **Verify Zero Spanner & Zero Kafka**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   ! grep -rn "cloud.google.com/go/spanner" .
   ! grep -rn "Shopify/sarama" .
   ! grep -rn "segmentio/kafka-go" .
   ```
