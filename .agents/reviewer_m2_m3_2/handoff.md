# Independent Review & Adversarial Audit Handoff Report — Milestones 2 & 3

**Date**: 2026-09-23T02:52:00+05:00  
**Reviewer**: Reviewer M2_M3_2 (`reviewer`, `critic`)  
**Target Monorepo**: `pegasus.x` (Sovereign Core)  
**Evaluated Work Products**: Worker M2 (`.agents/worker_m2/handoff.md`) and Worker M3 (`.agents/worker_m3/handoff.md`)  
**Verdict**: **APPROVE**

---

## 1. Observation

Direct code inspections, AST greps, and test execution directly conducted by Reviewer M2_M3_2:

### Milestone 2: Roles 1 & 2 (Supplier & Warehouse Hardening)
1. **Catch Weight / Variable Weight Logic**:
   - `backend/internal/supplier/models.go` (lines 298–373): `Product` has fields `IsCatchWeight bool`, `NominalWeightKg float64`, `CatchWeightTolerancePct float64`, `PricePerKgTiyin int64`. `CalculateCatchWeightAdjustment` computes actual vs nominal variance percentage, checks `isWithinTolerance := variancePct <= (tolerancePct + 1e-9)`, and computes 64-bit integer adjustments:
     ```go
     originalTotalTiyin := int64(math.Round(expectedTotalNominalWeight * float64(pricePerKgTiyin)))
     adjustedTotalTiyin := int64(math.Round(actualWeightKg * float64(pricePerKgTiyin)))
     adjustmentDeltaTiyin := adjustedTotalTiyin - originalTotalTiyin
     ```
   - `backend/internal/order/catch_weight.go` (lines 58–250): `RecordOutboundCatchWeight` verifies item catch weight configuration, locks order with `FOR UPDATE`, enforces `status != "DELIVERED"`, computes adjustment via `supplier.CalculateCatchWeightAdjustment`, updates order gross and effective totals in integer tiyins, and atomically emits `outbox.Emit(ctx, tx, "ORDER", req.OrderID, "order.catch_weight_adjusted", ...)`.
   - Verified tests: `supplier_catch_weight_test.go` and `order_vetting_and_catch_weight_test.go`.

2. **RFC 5652 CMS SignedData Container**:
   - `backend/internal/soliq/eimzo.go` (lines 268–550): Full ASN.1 standard implementation with standard OIDs:
     - `OIDData` (`1.2.840.113549.1.7.1`), `OIDSignedData` (`1.2.840.113549.1.7.2`), `OIDDigestSHA256` (`2.16.840.1.101.3.4.2.1`), `OIDRSAEncryption` (`1.2.840.113549.1.1.1`).
     - `CreateSignedDataCMS(factura *EFactura, certPEM string, privKey *rsa.PrivateKey) ([]byte, string, error)` parses X.509 cert, verifies validity window (`now.Before(cert.NotBefore) || now.After(cert.NotAfter)`), checks signer INN against seller INN, generates canonical JSON, signs with `rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, digest[:])`, and wraps in standard ASN.1 `CMSContentInfo` containing `CMSSignedData`.
     - `VerifySignedDataCMS(cmsDER []byte)` unpacks ASN.1 DER, checks certificate validity window, validates signer INN matches seller INN, extracts canonical payload, and executes cryptographic verification `rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, digest[:], si.Signature)`.
   - Verified negative gates in `eimzo_cms_test.go`: expired certificates, fraud INN mismatch, corrupt base64, and tampered payload bytes.

3. **Warehouse Auto-Approval Threshold & Vetting Queue**:
   - `backend/internal/order/service.go` (lines 30–37, 246–288, 448–488): `WarehouseVettingPolicy` defines `AutoOrderApprovalMode` (`ALWAYS_AUTO`, `THRESHOLD_BASED`, `ALWAYS_MANUAL_VETTING`), `AutoApplyThresholdTiyin` (default 60,000,000 tiyins = 600,000 UZS), and `AutoApprovalEnabled`.
   - In `CreateOrder` (both in-memory and PostgreSQL `RunInTx` paths), evaluates policy against incoming orders:
     - If first-time buyer (`priorOrderCount == 0`), credit blocked, or `effectiveTotalMinor > thresholdTiyin`, order is set to `StatusPendingApproval` with `needs_vetting = true`.
     - If creditworthy and under threshold, order transitions directly to `StatusConfirmed`.
     - Inventory is reserved for both to prevent stockouts during vetting.
   - `ApproveVettedOrder` (lines 1168–1263): Validates status transition from `PENDING_APPROVAL` to `CONFIRMED`, emits outbox event `order.vetting_approved`.
   - `RejectVettedOrder` (lines 1266–1340): Transitions to `CANCELLED`, releases reserved inventory back to available stock, emits outbox event.
   - Verified in `order_vetting_and_catch_weight_test.go`.

4. **Quarantine Segregation (`WH-QUARANTINE-01`) & ATP Exclusion**:
   - `backend/internal/qm/quarantine.go` (lines 13, 86, 94): Defines `CanonicalQuarantineBin = "WH-QUARANTINE-01"`. `CreateQuarantineLot` sets `LocationCode: CanonicalQuarantineBin` and `IsATPExcluded: true`.
   - `backend/internal/qm/repository.go` (lines 12–213): Implements `PostgresQMRepo` executing direct SQL on `qm_quarantine_lots` and `qm_disposition_records`. No dummy in-memory fallbacks in production.
   - `backend/internal/warehouse/service.go` (lines 467–481): `EnsureQuarantineBin` and `IsBinATPExcluded` verify that `WH-QUARANTINE-01` is strictly excluded from ATP pick allocation.

5. **Blind Receiving Variance Reconciliation**:
   - `backend/internal/warehouse/service.go` (lines 483–600): `RecordBlindPalletScan` and `ReconcileBlindReceiving` ingest field pallet scans into `inbound_blind_scans` without exposing PO line expectations to operators.
   - Reconciles total scanned vs PO expected quantities; any shortage creates `ShortageClaim` records in `inbound_shortage_claims` with `claimAmountMinor := int64(math.Round(shortage * float64(exp.UnitCostMinor)))` in 64-bit integer tiyins.
   - Emits outbox events `inbound.shortage_claim_created` and `inbound.blind_reconciled`.

6. **Migration 076**:
   - `database/migrations/076_supplier_catch_weight_and_order_vetting.sql`: Valid PostgreSQL 16 DDL adding `PENDING_APPROVAL` to `order_status` enum, catch weight columns to `products` and `order_items`, `location_code` to `qm_quarantine_lots`, and tables `inbound_blind_scans` and `inbound_shortage_claims`.

---

### Milestone 3: Roles 3 & 4 (Payloader & Dispatcher Hardening)
1. **Zero Mock Purge in `payload/repository.go`**:
   - `backend/internal/payload/repository.go` (lines 84–92): `pgRepository` retains only `pool *db.Pool`. All in-memory maps (`r.manifests`, `r.loadLedgers`, `r.exceptions`, etc.) and `seedInitialDockData()` have been eliminated from production code. Direct PostgreSQL 16 persistence across all 24 methods.
   - Test mock fixtures are isolated to `backend/internal/payload/mock_repository_test.go` (`_test.go` only).

2. **3L-CVRP Longitudinal Statics & Statutory Axle Equilibrium**:
   - `backend/internal/payload/service.go` (lines 293–428): `CalculateAxleFeasibility` computes:
     $$W_{steer} = W_{curb,front} + \sum_{i=1}^n w_i \frac{L - x_i}{L}, \quad W_{drive} = W_{curb,rear} + \sum_{i=1}^n w_i \frac{x_i}{L}$$
     $$\text{SteerRatio} = \frac{W_{steer}}{W_{steer} + W_{drive}} \times 100\%$$
   - Enforces Uzbekistan statutory single axle limit $11,500\text{ kg}$ (`MaxAllowedSingleAxleKg`) and $\ge 20.0\%$ steer tractive authority (`MinSteerAxleShareRatio`).
   - Correctly penalizes rear cantilever overhanging loads ($x_i > L$), reducing front axle share.

3. **Supervisor Override Validation & Cryptographic Sealing**:
   - `backend/internal/payload/service.go` (lines 554–660): `SealManifest` enforces:
     - Bolt seal regex: `^SEAL-UZ-[0-9A-Z]{6}$` (`ErrInvalidBoltSeal`).
     - Fails closed with `ErrAxleOverloadViolation` on axle overload or steer traction loss if `!req.ForceAxleOverride`.
     - When `req.ForceAxleOverride = true`, enforces 14-digit supervisor PINFL regex `^[0-9]{14}$` (`ErrInvalidSupervisorPINFL`) and non-empty override reason code (`ErrMissingOverrideReason`).
     - Computes SHA-256 digital seal hash binding `manifestID:truckPlate:boltSeal:frontAxleKg:rearAxleKg:steerRatio:override:supervisorPINFL:orders`.
     - Fails closed with `ErrManifestAlreadySealed` if manifest is already sealed, dispatched, or completed.

4. **Zero Mock Purge in `fleet_rescue_service.go` & Mid-Shift Hot-Swapping**:
   - `backend/internal/dispatch/fleet_rescue_service.go`: In-memory seeds (`RSC-2026-081`, etc.) purged into `fleet_rescue_mock_test.go`. Real PostgreSQL 16 persistence via `RescueStore` interface and `postgresRescueStore` querying `fleet_rescue_incidents`.
   - `backend/internal/dispatch/service.go` (`ExecuteRescue`, lines 575–768):
     - Cancels broken driver's manifest.
     - Sets broken driver `OFFLINE` and vehicle to `MAINTENANCE` with release reason `BREAKDOWN_MID_SHIFT`.
     - Creates new rescue manifest for rescue driver.
     - Transloads remaining stops into `manifest_stops`.
     - Records hot-swap records into `manifest_stop_transfers` table.
     - Updates orders to `DISPATCHED`/`LOADED` with rescue driver and vehicle without canceling orders.
     - Emits transactional outbox event `FLEET_BREAKDOWN_RESCUED`.
     - Publishes real-time rescue event `events:fleet:rescue_dispatched` to Redis Streams via `XADD`.

5. **Pre-Trip DVIR Gating Before Dispatch**:
   - `backend/internal/dispatch/service.go` (`CommitDispatch`, lines 301–347):
     - Checks active pairing in `driver_vehicle_assignments` (`released_at IS NULL`), returning `ErrVehicleNotPaired` if missing.
     - Checks driver on-shift status (`on_shift = true`), returning `ErrDriverNotOnShift`.
     - Checks `vehicle_inspections` for a `PRE_TRIP` inspection today (`created_at >= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tashkent')::date`), returning `ErrDVIRMissingToday` if missing and `ErrDVIRNotSafe` if `is_safe_to_operate = false`.

6. **Migration 075**:
   - `database/migrations/075_rescue_telemetry_and_diagnostics.sql`: Extends `fleet_rescue_incidents` with `odometer_km`, `diagnostic_code`, `cargo_snapshot`, `stranded_manifest_id`, and `rescue_manifest_id`. Creates tables `manifest_load_lines`, `manifest_exceptions`, and `gs1_ship_units`.

---

### Independent Test Execution & Zero Contamination Scan
- **Command executed**:
  ```bash
  cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
  go test -count=1 -race ./internal/supplier/... ./internal/warehouse/... ./internal/order/... ./internal/qm/... ./internal/soliq/... ./internal/payload/... ./internal/dispatch/... ./internal/fleet/...
  ```
- **Result**:
  ```
  ok  	github.com/pegasus-x/core/internal/supplier	3.043s
  ok  	github.com/pegasus-x/core/internal/warehouse	4.103s
  ok  	github.com/pegasus-x/core/internal/order	1.436s
  ok  	github.com/pegasus-x/core/internal/qm	1.427s
  ok  	github.com/pegasus-x/core/internal/soliq	1.774s
  ok  	github.com/pegasus-x/core/internal/payload	2.460s
  ok  	github.com/pegasus-x/core/internal/dispatch	1.381s
  ok  	github.com/pegasus-x/core/internal/fleet	4.025s
  ```
  **All 8 packages passed cleanly with Go's race detector enabled.**
- **Build verification**:
  `go build ./...`, `go build ./cmd/smokecheck`, `go build ./cmd/server` compiled cleanly with 0 errors.
- **AST Scan for Spanner and Kafka in `pegasus.x`**:
  - `go.mod`: 0 Spanner SDKs, 0 Kafka SDKs.
  - `backend/internal/`: 0 Spanner imports, 0 Kafka imports (only `migration_074_test.go` testing that `spanner` and `kafka` are prohibited words).

---

## 2. Logic Chain

1. **Integrity & Anti-Cheating Verification (Observation §1.1–1.6, §2.1–2.6)**:
   - Evaluated code for hardcoded test outputs or dummy facades. The calculations in `CalculateCatchWeightAdjustment` and `CalculateAxleFeasibility` use dynamic physical formulas and generic arithmetic. The RFC 5652 CMS implementation uses standard Go crypto and ASN.1 DER encoding. No integrity violations or shortcuts detected.
2. **Zero Mock Policy Adherence (Observation §2.1, §2.4)**:
   - In `payload/repository.go` and `dispatch/fleet_rescue_service.go`, all previous in-memory fallback maps and fake seeds have been removed from production types. Testing fixtures were moved exclusively into `_test.go` files (`mock_repository_test.go` and `fleet_rescue_mock_test.go`), adhering to the zero mock policy in production binaries.
3. **Financial Arithmetic Rigor (Observation §1.1, §1.3, §1.5)**:
   - In catch weight calculations, shortage claims, and order line adjustments, prices, costs, and deltas are stored and computed in 64-bit integer tiyins (`int64`). Floating point is restricted to physical dimensions (kg, percentages) and rounded explicitly to integer tiyins via `math.Round` before storage.
4. **Architectural Boundary Adherence (Observation §3)**:
   - Persistence is purely PostgreSQL 16 (`pgx/v5`). Real-time streaming uses Redis 7 Streams (`XADD`). Zero Spanner and zero Kafka libraries exist in `pegasus.x`.
5. **Operational Real-World Resilience (Observation §1.3, §2.2, §2.4, §2.5)**:
   - Edge cases are handled: pre-flight checks enforce that drivers are on shift, actively paired with vehicles, and vehicles have clean same-day DVIR records. Mechanical statics account for cantilever overhangs. Emergency rescues transfer stops dynamically without cancelling orders.

---

## 3. Caveats

1. **Scale Hardware Hardware Integration**: Weighing at the outbound warehouse dock and receiving bays is validated via certified API payloads with operator and scale certifier IDs; physical RS-232/Ethernet scale driver DAQ is outside the Go backend scope.
2. **National Cryptographic Algorithms**: The CMS SignedData engine implements RSA PKCS#1 v1.5 with SHA-256 for standard interoperability. When deployed with national Uzbek E-IMZO hardware tokens, the ASN.1 SignedData container wraps GOST O'zDSt 1092:2009 parameters through the E-IMZO browser/PKCS#11 bridge.
3. **Database Migration Execution**: Migrations `075_rescue_telemetry_and_diagnostics.sql` and `076_supplier_catch_weight_and_order_vetting.sql` are forward migrations ready to be executed against live PostgreSQL 16 instances.

---

## 4. Conclusion

The deliverables for Milestones 2 and 3 implemented by Worker M2 and Worker M3 fully comply with the approved specification, architectural constraints, and engineering guidelines:
- Zero mock data in production packages.
- Strict 64-bit integer minor unit arithmetic (`tiyins`).
- Strict two-system boundary (0 Spanner, 0 Kafka).
- 100% test pass rate with race detection across all 8 target packages.
- Verified physical mechanics, cryptographic signatures, and outbox event emissions.

**Verdict: APPROVE**

---

## 5. Verification Method

Any engineer or orchestrator can independently verify this audit:

1. **Execute Automated Race Test Suite**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go test -count=1 -v -race ./internal/supplier/... ./internal/warehouse/... ./internal/order/... ./internal/qm/... ./internal/soliq/... ./internal/payload/... ./internal/dispatch/... ./internal/fleet/...
   ```

2. **Verify Clean Binaries Compilation**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go build ./...
   go build ./cmd/smokecheck
   go build ./cmd/server
   ```

3. **Verify Zero Spanner / Kafka Dependencies**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   ! grep -rnI --exclude="*test.go" "spanner" ./internal
   ! grep -rnI --exclude="*test.go" "kafka" ./internal
   ```

4. **Inspect Source Locations**:
   - Supplier Catch Weight: `internal/supplier/models.go:298-373`, `internal/order/catch_weight.go:58-250`
   - E-Factura RFC 5652 CMS: `internal/soliq/eimzo.go:268-550`
   - Warehouse Vetting & Intake Tiers: `internal/order/service.go:30-37, 246-288, 448-488`
   - Quarantine Segregation: `internal/qm/quarantine.go:13, 86, 94`, `internal/warehouse/service.go:467-481`
   - Blind Receiving Reconciliation: `internal/warehouse/service.go:483-600`
   - Payload Direct PG16 & Axle Physics: `internal/payload/repository.go:84-92`, `internal/payload/service.go:293-428, 554-660`
   - Dispatch Rescue Hot-Swap & DVIR Gates: `internal/dispatch/fleet_rescue_service.go:31-140`, `internal/dispatch/service.go:301-347, 575-768`
   - Migrations: `database/migrations/075_rescue_telemetry_and_diagnostics.sql`, `076_supplier_catch_weight_and_order_vetting.sql`

5. **Invalidation Conditions**:
   - Any failure under `go test -race`.
   - Any import of `cloud.google.com/go/spanner` or Kafka client libraries in `pegasus.x/backend`.
   - Restoration of in-memory mock fallback maps in production files.
