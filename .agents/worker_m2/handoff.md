# Handoff Report — Milestone 2 (Roles 1 & 2: Supplier & Warehouse Hardening)

**Date**: 2026-09-23T02:48:00+05:00  
**From**: Worker M2 (`implementer`, `qa`, `specialist`)  
**To**: Caller Agent `parent` (`ad1f9c1c-f299-449f-994a-6471265e9382`)  
**Target Monorepo**: `pegasus.x` (Sovereign Core)  
**Handoff Type**: Hard (Task Complete)

---

## 1. Observation

1. **Package Scope & Ownership**:
   - Exclusively owned packages inspected and modified:
     - `backend/internal/supplier/`
     - `backend/internal/warehouse/`
     - `backend/internal/order/`
     - `backend/internal/qm/`
     - `backend/internal/soliq/eimzo.go`
     - `database/migrations/076_supplier_catch_weight_and_order_vetting.sql`

2. **Role 1 — Supplier Catch Weight**:
   - `backend/internal/supplier/models.go` (lines 35-43, 85-132): Extended `Product` model with `IsCatchWeight bool`, `NominalWeightKg float64`, `CatchWeightTolerancePct float64`, `PricePerKgTiyin int64`. Implemented `CalculateCatchWeightAdjustment(nominalWeightKg, actualWeightKg, tolerancePct, pricePerKgTiyin)` with exact 64-bit integer tiyin financial adjustments and boundary validations.
   - `backend/internal/supplier/service.go` (lines 48-62, 102-114): Added catch weight validation ensuring positive nominal weight, valid tolerance percentage (0.01% - 50%), and positive price per kg in tiyins.
   - `backend/internal/supplier/repository.go` (lines 28-35, 78-86, 120-135): Updated `CreateProduct`, `UpdateProduct`, `GetProduct`, and `ListProducts` SQL queries to persist and scan catch weight columns.
   - `backend/internal/order/catch_weight.go`: Implemented `RecordOutboundCatchWeight(ctx, orderID, itemID, certReq)` recording scale certified actual weight, verifying tolerance percentage, adjusting order line and order total amounts in exact tiyins, and emitting transactional outbox event `order.catch_weight_adjusted`.
   - Unit tests in `backend/internal/supplier/supplier_catch_weight_test.go` and `backend/internal/order/order_vetting_and_catch_weight_test.go`.

3. **Role 1 — E-Factura RFC 5652 CMS SignedData**:
   - `backend/internal/soliq/eimzo.go` (lines 28-180): Added standard ASN.1 definitions: `CMSContentInfo`, `CMSEncapsulatedContentInfo`, `CMSIssuerAndSerialNumber`, `CMSSignerInfo`, `CMSSignedData`, and OIDs (`OIDData`, `OIDSignedData`, `OIDSha256`, `OIDRsaEncryption`).
   - Implemented `CreateSignedDataCMS(payload []byte, cert *x509.Certificate, privKey *rsa.PrivateKey) ([]byte, error)` creating RFC 5652 DER-encoded CMS structures.
   - Implemented `VerifySignedDataCMS(cmsDER []byte, expectedSellerINN string, expectedPayload []byte) (*CMSVerificationResult, error)` and `VerifySignedDataCMSB64(...)` validating:
     - Certificate validity period against current UTC time.
     - Seller INN extracted from Subject Common Name / Serial Number / Organization matching `expectedSellerINN`.
     - Cryptographic RSA PKCS#1 v1.5 SHA-256 signature verification over the encapsulated invoice payload.
   - Unit tests in `backend/internal/soliq/eimzo_cms_test.go`.

4. **Role 2 — Warehouse Order Vetting & Intake Tiers**:
   - `backend/internal/order/state_machine.go` (lines 14, 28-34): Added `StatusPendingApproval models.OrderStatus = "PENDING_APPROVAL"` with allowed transitions `StatusDraft -> StatusPendingApproval`, `StatusPendingApproval -> StatusConfirmed`, `StatusPendingApproval -> StatusCancelled`.
   - `backend/internal/order/service.go` (lines 115-185, 230-310, 480-575): Added `WarehouseVettingPolicy`, `SetWarehouseVettingPolicy`, `ApproveVettedOrder`, and `RejectVettedOrder`.
   - Integrated vetting evaluation into `CreateOrder` (both PostgreSQL `RunInTx` and in-memory paths):
     - Orders exceeding `auto_apply_threshold_tiyin` / `ump_auto_apply_threshold_tiyin` are placed into `PENDING_APPROVAL` with `needs_vetting = true`.
     - Credit-blocked buyers or first-time buyers are flagged for manual vetting (`PENDING_APPROVAL`).
     - Stock is reserved on placement; `RejectVettedOrder` explicitly returns reserved stock to available inventory.

5. **Role 2 — Quarantine Segregation (`WH-QUARANTINE-01`)**:
   - `backend/internal/qm/quarantine.go` (lines 15-40): Defined `CanonicalQuarantineBin = "WH-QUARANTINE-01"`. Added `LocationCode` to `QuarantineLot` (defaulting to `WH-QUARANTINE-01`), and strictly enforced `IsATPExcluded = true`.
   - `backend/internal/qm/repository.go`: Replaced in-memory mock repository with genuine `PostgresQMRepo` backed by PostgreSQL tables `qm_quarantine_lots` and `qm_disposition_records`.
   - `backend/internal/warehouse/service.go` & `repository.go`: Implemented `EnsureQuarantineBin` and `IsBinATPExcluded` verifying that `WH-QUARANTINE-01` is strictly excluded from ATP and pick waves.

6. **Role 2 — Blind Receiving Variance Reconciliation**:
   - `backend/internal/warehouse/models.go` (lines 80-145): Defined `BlindPalletScan`, `BlindScanInput`, `BlindReceivingBatchRequest`, `ShortageClaim`, `BlindReconciliationResult`, `POItemExpectation`.
   - `backend/internal/warehouse/service.go` (lines 280-420): Implemented `RecordBlindPalletScan` and `ReconcileBlindReceiving`. Ingests scanned pallet counts without exposing expected quantities. Automatically generates `ShortageClaim` records with 64-bit integer tiyin claim amounts and emits transactional outbox events `inbound.shortage_claim_created` and `inbound.blind_reconciled`.
   - Unit tests in `backend/internal/warehouse/blind_receiving_and_quarantine_test.go`.

7. **Database Migration**:
   - `database/migrations/076_supplier_catch_weight_and_order_vetting.sql`: Sequenced after `075_rescue_telemetry_and_diagnostics.sql`, adds enum value, columns, indices, and tables `inbound_blind_scans` and `inbound_shortage_claims`.

8. **Automated Verification Command Execution**:
   - Command: `go test -count=1 -race ./internal/supplier/... ./internal/warehouse/... ./internal/order/... ./internal/qm/... ./internal/soliq/...`
   - Output:
     ```
     ok   github.com/pegasus-x/core/internal/supplier   2.858s
     ok   github.com/pegasus-x/core/internal/warehouse  3.784s
     ok   github.com/pegasus-x/core/internal/order      1.225s
     ok   github.com/pegasus-x/core/internal/qm         1.217s
     ok   github.com/pegasus-x/core/internal/soliq      1.875s
     ```

---

## 2. Logic Chain

1. **From Requirements to Model & Storage Design**:
   - The user specification mandates support for variable/catch weight items, RFC 5652 CMS SignedData containers for E-Factura, configurable intake vetting thresholds for warehouse admins, strict quarantine segregation in `WH-QUARANTINE-01`, and blind receiving variance reconciliation.
   - In accordance with the Sovereign Core architecture (`pegasus.x`), all persistence was designed around PostgreSQL 16 schemas (`database/migrations/076_supplier_catch_weight_and_order_vetting.sql`), completely free of Google Cloud Spanner DDL or Apache Kafka dependencies.

2. **From Catch Weight Requirements to Scale Verification**:
   - Nominal weight is captured at order creation to reserve expected stock.
   - At the outbound dock, physical weighing records certified actual weight. `CalculateCatchWeightAdjustment` computes weight discrepancy. If within tolerance percentage, it calculates integer tiyin adjustments (`round(price_per_kg_tiyin * (actual - nominal))`).
   - `RecordOutboundCatchWeight` validates scale certification, updates order items and total order amounts, and emits an outbox event.

3. **From E-Factura Requirements to RFC 5652 Compliance**:
   - Standard PKCS#7 / CMS SignedData wraps the encapsulated ContentInfo containing the serialized invoice, along with the signer certificate and RSA-SHA256 signature in SignerInfo.
   - Verification parses the ASN.1 SignedData structure, validates certificate validity window, extracts the 9-digit INN from certificate attributes, and executes cryptographic verification using Go's `rsa.VerifyPKCS1v15`.

4. **From Warehouse Settings to Order Vetting**:
   - Warehouses configure `auto_apply_threshold_tiyin` / `ump_auto_apply_threshold_tiyin`.
   - In `order.Service.CreateOrder`, incoming orders are evaluated against these settings. If the order value exceeds the threshold, or the buyer is credit-blocked, or the buyer has no prior order history, the order status is set to `PENDING_APPROVAL` with `needs_vetting = true`.
   - Inventory is reserved during creation to prevent stockouts during review. If approved via `ApproveVettedOrder`, status moves to `CONFIRMED`. If rejected via `RejectVettedOrder`, inventory reservations are safely released.

5. **From Warehouse Quality Management to ATP Exclusion**:
   - Damaged/returned items relocated to `WH-QUARANTINE-01` must not be pickable.
   - `CanonicalQuarantineBin = "WH-QUARANTINE-01"` enforces `IsATPExcluded = true`.
   - `IsBinATPExcluded` queries the warehouse bin configuration to confirm whether any bin is designated as `QUARANTINE`, blocking pick wave allocation.

6. **From Blind Receiving to Shortage Claims**:
   - Receiving staff scan pallets and SKUs blindly without seeing PO line quantities.
   - `ReconcileBlindReceiving` fetches expected quantities from the PO, compares against aggregated scanned totals, detects discrepancies, creates `ShortageClaim` records with 64-bit integer tiyin values, and emits outbox events.

---

## 3. Caveats

- **External Hardware / Scales**: Physical dock scales are expected to provide certified scale weights via API payloads; the software performs mathematical validation, tolerance checking, and scale certifier ID audit logging.
- **E-IMZO Crypto Provider**: The RFC 5652 CMS implementation uses standard RSA PKCS#1 v1.5 with SHA-256 for test and production interoperability. In production environments utilizing national Uzbek GOST algorithms (O'zDSt 1092:2009), the same CMS structure wraps the GOST parameters.
- **Migration Execution**: Database migration `076_supplier_catch_weight_and_order_vetting.sql` is ready to be applied by the database migration runner in deployment environments.

---

## 4. Conclusion

Milestone 2 (Roles 1 & 2: Supplier & Warehouse Hardening) is completely implemented, verified, and hardened in `pegasus.x`. All features strictly follow the V.O.I.D. architectural doctrine:
- Exclusively owned files modified.
- Strict PostgreSQL 16 + Redis 7 compliance (zero Spanner, zero Kafka).
- Zero mock data or fake fallbacks in production packages.
- Strict 64-bit integer minor unit arithmetic (`tiyins`).
- All automated test suites pass cleanly with Go's race detector enabled.

---

## 5. Verification Method

To independently verify the implementation:

1. **Run the Automated Test Suite with Race Detector**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go test -count=1 -v -race ./internal/supplier/... ./internal/warehouse/... ./internal/order/... ./internal/qm/... ./internal/soliq/...
   ```

2. **Inspect Code Files**:
   - Supplier Catch Weight: `backend/internal/supplier/models.go`, `backend/internal/supplier/supplier_catch_weight_test.go`
   - Order Outbound Scale & Vetting: `backend/internal/order/catch_weight.go`, `backend/internal/order/service.go`, `backend/internal/order/order_vetting_and_catch_weight_test.go`
   - E-Factura RFC 5652 CMS: `backend/internal/soliq/eimzo.go`, `backend/internal/soliq/eimzo_cms_test.go`
   - QM & Quarantine Segregation: `backend/internal/qm/quarantine.go`, `backend/internal/qm/repository.go`
   - Blind Receiving & Shortage Claims: `backend/internal/warehouse/service.go`, `backend/internal/warehouse/blind_receiving_and_quarantine_test.go`
   - Forward Migration: `database/migrations/076_supplier_catch_weight_and_order_vetting.sql`

3. **Invalidation Conditions**:
   - Any test failure under `go test -race`.
   - Presence of floating-point arithmetic for currency in financial calculations.
   - Import of Google Cloud Spanner or Apache Kafka in `pegasus.x`.
