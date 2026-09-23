# Handoff Report — Roles 1–4 Backend Packages & Architecture Survey

**Date**: 2026-09-23T01:38:00+05:00  
**From**: Explorer Survey 2 (`.agents/explorer_survey_2`)  
**To**: Orchestrator (`parent`, id: `ad1f9c1c-f299-449f-994a-6471265e9382`)  
**Type**: Hard Handoff (Investigation & Architectural Survey Complete)  
**Target Codebase**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x` (PostgreSQL 16 + Redis 7 Streams)  
**Full Companion Survey**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/explorer_survey_2/survey_report.md` (42 KB detailed analysis)

---

## 1. Observation

Direct observations from AST inspection, raw code reading, pattern greps, and test execution in `pegasus.x/backend`:

### A. Role 1 (Supplier & Product/Catalog)
- **Files & Packages**:
  - `internal/supplier/models.go`, `repository.go`, `service.go`
  - `internal/soliq/efactura.go`, `eimzo.go`
  - `internal/inventory/models.go`
  - `internal/promotion/evaluator.go`, `internal/bins/models.go`, `internal/wms/lots.go`
- **MXIK 17-digit code**:
  - `internal/supplier/service.go:40`: `var mxikRegex = regexp.MustCompile(`^[0-9]{17}$`)`
  - `internal/supplier/service.go:252-254`:
    ```go
    if !mxikRegex.MatchString(req.MxikCode) {
        return nil, fmt.Errorf("invalid MXIK code: must be exactly 17 digits")
    }
    ```
  - `internal/soliq/efactura.go:18-28`: Validated again on invoice items prior to Soliq transmission.
- **EAN-13 Barcode Mod-10 Check**:
  - `internal/supplier/service.go:61-85`: `ValidateEAN13(barcode string) error`:
    Computes weighted sum $\sum_{i=0}^{11} d_i \times (1 \text{ or } 3)$, checks $(10 - (\text{sum} \pmod{10})) \pmod{10} == d_{12}$.
- **Tiered Volume Discounts**:
  - `internal/supplier/models.go:85-99`: `PricingRule` with `MinQuantity int`, `DiscountPercent float64`, `TierLabel string`.
  - `internal/supplier/service.go:1013-1019`: Evaluated in `PreviewPricing`.
  - `internal/promotion/evaluator.go:185`: Evaluated during cart checkout.
- **FEFO Lot Tracking**:
  - `internal/bins/models.go:15-20`: `ExpiryDate time.Time` in bin allocation.
  - `internal/wms/lots.go:176`: `AllocateLotsFEFO` orders lots by `expiration_date ASC`.
- **Catch Weight Tolerances**:
  - Search across entire `backend/` for `catch_weight`, `catchWeight`, `weight_tolerance`: **0 matches found**. Catch weight tolerance logic is completely missing from product and inventory schemas.
- **E-Factura PKCS#7 / E-IMZO**:
  - `internal/soliq/eimzo.go:17-48`: Implements `SignDataPKCS7(certPEM, keyPEM, rawPayload)` using `crypto/rsa.SignPKCS1v15` with `crypto.SHA256` after checking 9-digit INN in subject common name/serial.
  - **Limitation**: Generates raw RSA PKCS#1 v1.5 signature, not full ASN.1 RFC 5652 CMS/PKCS#7 EnvelopedData/SignedData container required by production Soliq EHF gate.
- **Mock Data Audit**:
  - `internal/supplier/repository.go`: 100% pure PostgreSQL 16 via `pgxpool.Pool`.
  - `internal/order/service.go:37, 148-240`: Contains in-memory fallback `inMemoryOrders map[string]*models.Order` when `s.pool == nil`.

---

### B. Role 2 (Warehouse Admin & Intake Queue)
- **Files & Packages**:
  - `internal/warehouse/models.go`, `repository.go`, `service.go`
  - `internal/crossdock/models.go`, `repository.go`, `service.go`, `engine.go`
  - `internal/inbound/models.go`, `service.go`
  - `internal/qm/quarantine.go`
  - `database/migrations/072_warehouse_settings_and_axle_override.sql`
- **Configurable Auto-Approval Threshold (> 600,000 UZS / 60M Tiyins)**:
  - Database schema (`072_warehouse_settings_and_axle_override.sql:4-13`):
    `auto_order_approval_mode VARCHAR(32) DEFAULT 'THRESHOLD_BASED'`,
    `ump_auto_apply_threshold_tiyin BIGINT DEFAULT 60000000`,
    `max_discrepancy_tolerance_pct NUMERIC(5,2) DEFAULT 5.00`.
  - `internal/warehouse/service.go:177-220`: `GetApprovalSettings` and `UpdateApprovalSettings`.
  - `internal/api/router.go:281-282`: Endpoints `GET/PUT /v1/warehouse/approval-settings`.
  - **Gap in Order Vetting**: `internal/order/service.go:102-140`: Does **NOT** query or evaluate `ump_auto_apply_threshold_tiyin`. Incoming orders unconditionally transition to `PENDING` (manual) or `DRAFT` (auto) without checking threshold.
- **Cross-Docking Peak Waves**:
  - `internal/crossdock/engine.go:1-120`: Operational cross-docking engine supporting dock staging scans, wave grouping, and pallet dispatch.
- **Blind Receiving Variance Reconciliation**:
  - `internal/inbound/service.go:88-142`: Allows recording `ReceivedQuantity` and `DamagedQuantity`. Discrepancies generate notes, but expected quantity masking for receiver and automated carrier shortage claims are documented in comments only.
- **Quarantine Segregation (`WH-QUARANTINE-01`)**:
  - `internal/qm/quarantine.go`: Implements `QuarantineLot` with `is_atp_excluded = true` preventing allocation into available-to-promise inventory.
  - The canonical bin string `"WH-QUARANTINE-01"` is not statically defined or seeded.

---

### C. Role 3 (Payloader / Picker & 3L-CVRP Longitudinal Statics)
- **Files & Packages**:
  - `internal/payload/models.go`, `repository.go`, `service.go`
  - `database/migrations/072_warehouse_settings_and_axle_override.sql:16-24`
- **Longitudinal Statics Mathematical Formula**:
  - `internal/payload/service.go:343-412` (`CalculateAxleFeasibility`):
    $$W_{\text{steer}} = W_{\text{curb,steer}} + \sum_{i=1}^n \frac{w_i (L - x_i)}{L}$$
    $$W_{\text{drive}} = W_{\text{curb,drive}} + \sum_{i=1}^n \frac{w_i x_i}{L}$$
    Where $L$ is wheelbase, $x_i$ is distance from front steer axle to center of bay $i$, and $w_i$ is cargo weight in bay $i$.
- **Statutory Limits**:
  - `internal/payload/service.go:41`: `MaxAllowedSingleAxleKg = 11500.0` (11,500 kg statutory maximum).
  - `internal/payload/service.go:42`: `MinSteerAxleShareRatio = 0.20` ($\ge 20\%$ steer axle tractive authority).
  - Verified and blocked during manifest sealing in `payload/service.go:559-578`.
- **Supervisor Override Protocol**:
  - Implemented in `payload/service.go:554-558` and persisted in DB (`072_warehouse_settings_and_axle_override.sql:18-20`): `is_axle_overridden BOOLEAN`, `axle_override_reason TEXT`, `axle_override_by UUID`.
- **Bolt Seal Verification**:
  - `payload/service.go:580-594`: Mandatory bolt seal serial, hashed via SHA-256 into `digital_seal_hash`.
- **CRITICAL DEFECT — Zero Mock Data Policy Violation**:
  - `internal/payload/repository.go:83-220, 367-375`: Contains a massive fake mock fallback `pgRepository` with static fake UUIDs (`mnf_tashkent_bay4_01`, etc.) when `r.pool == nil` or on query error!
  - `SealManifest` lacks regex validation for canonical format `^SEAL-UZ-[0-9A-Z]{6}$`.
  - `OverrideSupervisorID` is not validated as a 14-digit PINFL.

---

### D. Role 4 (Dispatcher & Fleet Management)
- **Files & Packages**:
  - `internal/dispatch/service.go`, `fleet_rescue_service.go`, `rescue.go`, `binpack.go`, `localsearch.go`
  - `internal/fleet/models.go`, `repository.go`, `service.go`
  - `internal/telemetry/ingestion.go`, `geofence.go`
- **Multi-Vehicle VRP Routing**:
  - `internal/dispatch/service.go:210-380` (`PreviewDispatch`):
    - Validates active driver shift (`driver_shifts` where `status = 'ACTIVE'`).
    - Validates dynamic vehicle pairing (`vehicle_id` assigned to shift).
    - Validates pre-trip DVIR inspection passed on current calendar date (`dvir_inspections.inspection_type = 'PRE_TRIP'`, `status = 'PASSED'`).
    - Bin packs orders into vehicle payload capacity and optimizes route sequences via 2-Opt local search.
- **Live GPS Telemetry**:
  - `internal/telemetry/ingestion.go:48-96`: Ingests driver coordinates, writes to Redis Geo (`GEOADD driver_positions`) with 20s TTL, and throttles pubsub broadcasts to 2.5s.
- **Mid-Shift Breakdown Reporting & Dynamic Rescue Hot-Swap**:
  - `internal/dispatch/fleet_rescue_service.go:137-210`: `ReportBreakdown` logs breakdown with odometer, GPS, and diagnostic reason code, updating incident to `ACTIVE_ALERT`.
  - `internal/dispatch/fleet_rescue_service.go:215-320`: `RankRescueCandidates` evaluates on-shift vehicles for spare volume ($V_{\text{capacity}} \times 0.95 - V_{\text{loaded}} \ge V_{\text{needed}}$) and sorts by Haversine distance.
  - `internal/dispatch/fleet_rescue_service.go:325-450`: `ExecuteRescue`:
    - Executes atomic transaction: marks broken vehicle as `MAINTENANCE`, cancels old manifest, creates rescue manifest, reassigns stops and orders without cancellation, and records outbox event `FLEET_BREAKDOWN_RESCUED`.
- **CRITICAL DEFECTS**:
  - `internal/dispatch/fleet_rescue_service.go:18-105`: Hardcoded mock seeds (`RSC-2026-081`, `Javokhir Sobirov`, `01 341 BBA`, etc.) in in-memory map `rescueIncidents`. **Direct violation of Zero Mock Data Policy**.
  - `internal/dispatch/service.go:310-340`: `PreviewDispatch` falls back to hardcoded mock routes when `s.pool == nil`.
  - Direct alert in `ExecuteRescue` calls `redis.Publish` instead of Redis Streams (`XADD`).

---

### E. Automated Test Suite Execution
- Command executed:
  `go test -v -race ./internal/supplier ./internal/warehouse ./internal/payload ./internal/dispatch ./internal/fleet`
- Result: **All tests pass cleanly** (zero race conditions, zero compile failures).

---

## 2. Logic Chain

1. **Premise 1 (Doctrine Requirement)**: Per `AGENTS.md` and `GEMINI.md`, `pegasus.x` is single-tenant PostgreSQL 16 + Redis 7 Streams, zero mock data policy, zero floating point for currency (`tiyins`), with full operational lifecycle for Roles 1–4.
2. **Premise 2 (Role 1 State)**: Supplier, catalog, MXIK (17 digits), EAN-13 (Mod-10), and tiered pricing are implemented with pure PG16. However, catch weight tolerances are completely absent, and E-Factura signature is RSA PKCS#1 v1.5 rather than full RFC 5652 CMS envelope.
3. **Premise 3 (Role 2 State)**: Warehouse approval settings (threshold 60M tiyins) and cross-docking are persisted in PG16 migration 072, but the intake flow in `order/service.go` is disconnected from these settings.
4. **Premise 4 (Role 3 State)**: 3L-CVRP static axle moment math ($W_{\text{steer}}, W_{\text{drive}}$), 11,500 kg limit, $\ge 20\%$ steer tractive authority, supervisor override, and digital seal hash are mathematically verified. However, `payload/repository.go` contains a fake in-memory repository fallback.
5. **Premise 5 (Role 4 State)**: Shift check, vehicle pairing, pre-trip DVIR gating, and dynamic rescue hot-swapping are fully coded. However, `dispatch/fleet_rescue_service.go` contains hardcoded mock seeds.
6. **Conclusion**: The architectural foundation in `pegasus.x/backend/internal/` is structurally sound and mathematically rigorous for all four roles, but contains **two critical Zero Mock Data Policy violations** (in `payload/repository.go` and `dispatch/fleet_rescue_service.go`) and **four logic wiring gaps** (missing catch weight, non-CMS E-Factura, disconnected order vetting threshold, and bolt seal/supervisor ID regex validation) that must be remedied before production deployment.

---

## 3. Caveats

1. **Read-Only Scope**: This survey was strictly read-only. No fixes or modifications were committed to production code during this run.
2. **No Live Database Instance**: Tests were run using package unit suites; end-to-end integration tests requiring an active PostgreSQL 16 + TimescaleDB + PostGIS container or Redis 7 cluster were not spun up.
3. **Frontend Parity**: This survey focused exclusively on `pegasus.x/backend/internal/` packages; client UI components (Tauri v2 / Next.js 15) were not audited in this survey step (delegated to separate survey).

---

## 4. Conclusion

The backend architecture of `pegasus.x` demonstrates high adherence to Uzbekistan statutory standards (MXIK 17-digit validation, EAN-13 Mod-10, Soliq INN verification, 64-bit integer minor unit arithmetic in `tiyins`) and heavy engineering rigor (3L-CVRP axle load distribution statics, DVIR gating, dynamic fleet rescue hot-swapping).

### Mandatory Remediations Required:
1. **Purge In-Memory Mock Fallbacks**:
   - Delete fake seed records and memory fallback in `internal/payload/repository.go:83-220`. Return database error when pool is unavailable.
   - Delete fake incident seeds and memory fallback in `internal/dispatch/fleet_rescue_service.go:18-105`. Ensure all rescue queries hit `pgxpool.Pool`.
   - Remove `inMemoryOrders` fallback in `internal/order/service.go:37,148-240`.
2. **Wire Order Intake Vetting Threshold**:
   - In `internal/order/service.go:CreateOrder`, inject `warehouse.Service` or repository query to check `ump_auto_apply_threshold_tiyin` and `auto_order_approval_mode`.
   - If order total exceeds threshold or mode is `THRESHOLD_BASED`/`ALWAYS_MANUAL_VETTING`, route to vetting queue with status `REQUIRES_APPROVAL`.
3. **Implement Catch Weight Tolerances**:
   - Add `catch_weight_tolerance_pct` (e.g. 5.0%) to product/SKU models.
   - Validate received weight vs ordered weight during warehouse receiving and payload loading.
4. **Enforce Formatting Constraints**:
   - Enforce regex `^SEAL-UZ-[0-9A-Z]{6}$` for bolt seals in `payload/service.go:SealManifest`.
   - Enforce 14-digit PINFL regex `^[0-9]{14}$` for `OverrideSupervisorID`.
5. **Upgrade E-Factura Signing to CMS**:
   - Implement RFC 5652 CMS SignedData envelope in `internal/soliq/eimzo.go` for full Soliq EHF compliance.

---

## 5. Verification Method

To independently verify the observations, logic, and conclusions in this report:

### A. Run Automated Unit & Race Detection Tests
```bash
cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
go test -v -race ./internal/supplier
go test -v -race ./internal/warehouse
go test -v -race ./internal/payload
go test -v -race ./internal/dispatch
go test -v -race ./internal/fleet
```

### B. Verify Zero Mock Data Violations via Grep
```bash
# Check payload in-memory mock repository fallback
grep -n -C 5 "mnf_tashkent_bay4_01" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/payload/repository.go

# Check dispatch rescue in-memory mock incident seeds
grep -n -C 5 "RSC-2026-081" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/dispatch/fleet_rescue_service.go

# Check order service in-memory fallback
grep -n -C 5 "inMemoryOrders" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/order/service.go
```

### C. Verify Statutory & Mathematical Logic
```bash
# Verify MXIK 17-digit regex
grep -n -C 3 "mxikRegex" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/supplier/service.go

# Verify EAN-13 Mod-10 algorithm
grep -n -A 25 "func ValidateEAN13" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/supplier/service.go

# Verify 3L-CVRP axle statics math
grep -n -A 35 "func (s *Service) CalculateAxleFeasibility" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/payload/service.go

# Verify 11,500 kg axle limit and 20% steer ratio constants
grep -n -C 3 "MaxAllowedSingleAxleKg" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/payload/service.go
```

### D. Invalidation Conditions
This handoff report is invalidated if:
1. `pegasus.x/backend` adopts Google Cloud Spanner or Apache Kafka (violating the architectural boundary doctrine).
2. The mock data fallbacks in `payload/repository.go` or `dispatch/fleet_rescue_service.go` are verified as intentionally deployed in production without PostgreSQL tables.
3. Migration 072 is rolled back or modified without corresponding code adjustments.
