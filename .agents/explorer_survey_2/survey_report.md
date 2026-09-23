# Roles 1–4 Backend Packages & Architecture Survey Report
**Target System**: `pegasus.x` (Sovereign Lean Single-Tenant National Operating Core: PostgreSQL 16 + Redis 7 Streams)  
**Surveyed Packages**: `internal/supplier`, `internal/warehouse`, `internal/payload`, `internal/dispatch`, `internal/fleet`, plus supporting packages (`internal/soliq`, `internal/bins`, `internal/wms`, `internal/inbound`, `internal/crossdock`, `internal/telemetry`, `internal/order`, `internal/api/router.go`)  
**Date**: 2026-09-23  
**Status**: COMPLETE  

---

## 1. Executive Summary & Monorepo Topology Audit

The `pegasus.x` codebase represents the sovereign single-tenant national operating core designed for the Republic of Uzbekistan. In accordance with the root architectural doctrine (`AGENTS.md` and `GEMINI.md`):
- **Database Engine**: Strictly PostgreSQL 16 (`pgx/v5` connection pool) with TimescaleDB and PostGIS capabilities.
- **Messaging & Streaming**: Strictly Redis 7 Streams (`XADD`/`XREADGROUP`) and Transactional Outbox pattern (`outbox_events` table with `FOR UPDATE SKIP LOCKED` relay worker).
- **Financial Arithmetic**: Strictly 64-bit integer minor units (`tiyins` for UZS) with zero floating-point arithmetic for currency.
- **Architectural Boundary**: Zero Spanner SDKs, Spanner DDL, or Apache Kafka drivers inside `pegasus.x`.
- **Truth Protocol & Zero Mock Data Policy**: Live code is the sole status source of truth. All domain entities must dynamically persist in PostgreSQL 16. In-memory stub repositories or static mock seeds in production packages are critical defects that must be surfaced and purged.

This survey provides a line-by-line, evidence-backed inspection of the backend packages supporting **Role 1 (Supplier)**, **Role 2 (Warehouse Admin)**, **Role 3 (Payloader/Picker)**, and **Role 4 (Dispatcher)**, identifying existing implementations, missing features, mock data theatre, and all mounted REST endpoints.

---

## 2. Role-by-Role Deep Dive Analysis

```
+---------------------------------------------------------------------------------------------------+
|                                 PEGASUS.X SUPPLY CHAIN TOPOLOGY                                  |
|                                                                                                   |
|  [ 1. SUPPLIER ] --------> [ 2. WAREHOUSE ADMIN ] -------> [ 3. PAYLOADER ]                      |
|  - Catalog & MXIK          - ASN Blind Receiving           - 3L-CVRP Axle Statics                 |
|  - Tiered MOQ Pricing      - Auto-Vetting Queue            - Steer Traction Check                 |
|  - Batch/Lot FEFO          - Dock Wave Scheduling          - Supervisor Override & Bolt Seal      |
|                                                                    |                              |
|                                                                    v                              |
|                                [ 4. DISPATCHER ] <-----------------+                              |
|                                - Dynamic VRP Fleet Routing                                        |
|                                - Breakdown Hot-Swap Rescue                                        |
|                                - Live Telemetry & Geofence                                        |
+---------------------------------------------------------------------------------------------------+
```

---

### Role 1: Supplier (Manufacturer & Brand Principal)

#### 1. Package Structure & Organization
- **Primary Package**: `backend/internal/supplier/`
  - `models.go` (352 lines): Defines `SupplierProfile`, `SupplierRecord`, `Product`, `PricingRule`, `RetailerPricingOverride`, `PricingPreviewRequest`, `PricingPreviewResponse`, `OrderVetLog`, `ServicePolicy`, `ServicePromiseBreach`, `PaymentGatewayConfig`, `Warehouse`, `WarehouseTruck`, `WarehousePayloader`.
  - `repository.go` (1,915 lines): Defines `Repository` interface and implements `PostgresRepository` targeting PostgreSQL 16 tables (`suppliers`, `supplier_profiles`, `products`, `supplier_pricing_rules`, `retailer_pricing_overrides`, `order_vet_logs`, `supplier_service_policies`, `supplier_promise_breaches`, `supplier_payment_gateways`, `warehouses`, `warehouse_trucks`, `warehouse_payloaders`).
  - `service.go` (1,707 lines): Implements supplier registration, authentication, STIR validation, product catalog management, tiered pricing preview, order vetting, KYC document review, and outbox event broadcasting.
- **Product Package Observation**: There is **no dedicated `internal/product/` package** in `pegasus.x`. Product catalog management is divided across:
  1. `internal/supplier/`: Dedicated `products` table models (`Product`) and CRUD operations for supplier catalog onboarding (`POST /v1/supplier/onboarding/products`).
  2. `internal/inventory/`: SKU models (`models.SKU`), stock balances, and catalog lookups used by warehouse and ordering services (`POST /v1/catalog/products`).
  3. `internal/api/handlers_catalog.go`: Exposes `/v1/catalog/products` and `/v1/catalog/categories` routing to `inventory.Service`.

#### 2. MXIK 17-Digit Code Handling
- **Implementation**:
  - `internal/supplier/service.go:40`: `mxikRegex = regexp.MustCompile(`^[0-9]{17}$`)`.
  - `internal/supplier/service.go:252-254`: Product creation checks `!mxikRegex.MatchString(prod.MxikCode)` and returns `ErrInvalidMXIK` ("mxik_code must be a 17-digit statutory code").
  - `internal/soliq/efactura.go:18-28`: `ValidateMXIK(code string) error` verifies exact length 17 and numeric characters `[0-9]`, returning `ErrInvalidMXIKCode`.
  - `internal/api/handlers_supplier.go:1194-1197`: Returns HTTP 400 Bad Request if MXIK is invalid.
- **Status**: **FULLY WIRED & ENFORCED**.

#### 3. EAN-13 Barcode Mod-10 Check Digit Validation
- **Implementation**:
  - `internal/supplier/service.go:61-85`: `ValidateEAN13(barcode string) bool` performs:
    1. Length validation (`len(barcode) != 13`).
    2. Character check (all numeric digits).
    3. Standard GS1 Modulo-10 check digit math:
       $$\text{Sum} = \sum_{i=0}^{11} d_i \times (\text{if } i \pmod 2 == 0 \text{ then } 1 \text{ else } 3)$$
       $$\text{CheckDigit} = (10 - (\text{Sum} \pmod{10})) \pmod{10}$$
    4. Compares check digit against `barcode[12]`.
  - `internal/supplier/service.go:249-251`: Enforced on `CreateProduct`.
- **Status**: **FULLY WIRED & VERIFIED**.

#### 4. Tiered Volume Discounts & Pricing Rules
- **Implementation**:
  - `internal/supplier/models.go:85-99`: `PricingRule` contains:
    - `VolumeTier1Qty` (int), `VolumeTier1DiscountPct` (float64)
    - `VolumeTier2Qty` (int), `VolumeTier2DiscountPct` (float64)
    - `MinOrderAmountMinor` (int64)
  - `internal/supplier/service.go:1013-1019`: `PreviewPricing` dynamically calculates volume tier discounts:
    ```go
    if req.OrderQuantity >= pricingRule.VolumeTier2Qty && pricingRule.VolumeTier2DiscountPct > 0 {
        volumeDiscountPct = pricingRule.VolumeTier2DiscountPct
        appliedRules = append(appliedRules, fmt.Sprintf("VOLUME_TIER_2 (>=%d units: %.1f%%)", pricingRule.VolumeTier2Qty, volumeDiscountPct))
    } else if req.OrderQuantity >= pricingRule.VolumeTier1Qty && pricingRule.VolumeTier1DiscountPct > 0 {
        volumeDiscountPct = pricingRule.VolumeTier1DiscountPct
        appliedRules = append(appliedRules, fmt.Sprintf("VOLUME_TIER_1 (>=%d units: %.1f%%)", pricingRule.VolumeTier1Qty, volumeDiscountPct))
    }
    ```
  - `internal/promotion/evaluator.go:185`: Also supports volume tier promos (`PromoTypeVolumeTier`).
- **Status**: **FULLY WIRED**.

#### 5. FEFO Lot Tracking
- **Implementation**:
  - Modeled in `internal/bins/models.go:84-110` (`StockLot` with `ExpiryDate`, `QuantityOnHand`, `QuantityReserved`, `Status = AVAILABLE | QUARANTINE | EXPIRED`).
  - Implemented in `internal/wms/lots.go:176-248` (`AllocateLotsFEFO` with `SELECT ... FOR UPDATE` ordered by `expiry_date ASC, received_at ASC` and filtered by `expiry_date >= cutoffDate`).
  - Inbound receiving generates FEFO stock lot records in `internal/inbound/repository.go:591`.
- **Status**: **FULLY WIRED**.

#### 6. Catch Weight / Variable Weight Tolerances
- **Audit Finding**:
  - Grep across the entire `pegasus.x` codebase for `catch weight` yielded **zero matches**.
  - Nominal order placement vs outbound scale true certified catch-weight recording and automatic integer tiyin balance recalculation is **MISSING**.
- **Status**: **NOT IMPLEMENTED / ARCHITECTURAL GAP**.

#### 7. E-Factura PKCS#7 Signing
- **Implementation**:
  - `internal/soliq/efactura.go`: `EFactura`, `GenerateEFactura`, `GenerateCorrectiveFactura` (Tax Code Art. 257 Tuzatuvchi invoices).
  - `internal/soliq/eimzo.go`: `VerifyEImzoSignature`, `SignFacturaWithKey`, `ComputeFacturaDigest`, `ExtractSignerINN`.
  - Deterministic SHA-256 digest over canonical JSON (`CanonicalFacturaPayload`), signed with RSA PKCS#1 v1.5 (`rsa.SignPKCS1v15` / `rsa.VerifyPKCS1v15`), validating 9-digit INN on X.509 certificate.
  - **Caveat**: The implementation uses RSA PKCS#1 v1.5 over canonical JSON rather than a full RFC 5652 CMS/PKCS#7 enveloped/signed data structure.
- **Status**: **FUNCTIONAL (DIGEST + RSA SIGNATURE), FULL RFC 5652 CMS ENVELOPE MISSING**.

#### 8. Mock Data / In-Memory Repository Audit
- `internal/supplier/repository.go`: **100% pure PostgreSQL 16**. Zero memory fallback maps in production code. All operations execute against `supplier_profiles`, `products`, `supplier_payment_gateways`, `warehouses`, etc.
- Test stub: `mock_repository_test.go` exists solely for unit testing in `_test.go`.
- **Cross-Domain Violation**: `internal/order/service.go:37,148-240` maintains an in-memory orders fallback `inMemoryOrders map[string]*models.Order` when `s.pool == nil`.
- **Status**: `internal/supplier` passes Zero Mock Data Policy; `internal/order` violates it.

---

### Role 2: Warehouse Admin (Distribution Center & Cross-Dock Hub)

#### 1. Package Structure & Organization
- **Primary Package**: `backend/internal/warehouse/`
  - `models.go` (150 lines): `WarehouseRecord`, `WarehouseApprovalSettings`, `WarehouseRegisterRequest`, `DockBayConfig`, `WarehouseBinConfig`, `InitialStockConfig`, `WarehouseOnboardingState`, `WarehouseAuditEvent`.
  - `repository.go` (550 lines): `Repository` interface and `pgRepository` (Postgres 16 via `pgxpool`).
  - `service.go` (465 lines): Onboarding wizard (`ConfigureDockBays`, `ConfigureBins`, `ConfigureInitialStock`, `CompleteOnboarding`), plus approval settings (`GetApprovalSettings`, `UpdateApprovalSettings`).
- **Supporting Packages**:
  - `internal/crossdock/`: Direct transshipment and cross-docking buffer lanes.
  - `internal/inbound/`: Replenishment purchase orders, dock goods receipts.
  - `internal/pickwave/`: Pick wave generation and batch pick confirmation.
  - `internal/wms/`: Lot homogeneity and FEFO allocation.
  - `internal/qm/`: Quality management quarantine lots and disposition.

#### 2. Configurable Auto-Approval Threshold
- **Implementation**:
  - `internal/warehouse/models.go:42-45,50-56`:
    - `AutoOrderApprovalMode`: `'ALWAYS_AUTO'`, `'THRESHOLD_BASED'`, `'ALWAYS_MANUAL_VETTING'`
    - `UMPAutoApplyThresholdTiyin`: int64 (default `60000000` tiyins = 600,000 UZS)
    - `AutoApprovalEnabled`: bool
    - `MaxDiscrepancyTolerancePct`: float64 (default 5.0%)
  - Migration `072_warehouse_auto_approval_and_vetting_settings.sql:5-18`: Added columns and check constraint `chk_warehouses_auto_approval_mode` to `warehouses` table.
  - Endpoints wired:
    - `GET /v1/warehouse/approval-settings` / `PUT /v1/warehouse/approval-settings`
    - `GET /v1/warehouse/{id}/approval-settings` / `PUT /v1/warehouse/{id}/approval-settings`
- **Audit Finding / Gap**:
  - While warehouse approval settings are stored and updated in PostgreSQL, **`order/service.go` does not inspect `UMPAutoApplyThresholdTiyin` when creating an order**. When an order is created, it unconditionally transitions to `PENDING` (or `DRAFT` if auto-order). The automatic vetting gate (auto-approving if order $\le$ threshold and account is creditworthy, or routing to a warehouse manual vetting queue if $>$ threshold) is **NOT WIRED into the order lifecycle**.
- **Status**: **SETTINGS STORED IN DB; AUTOMATIC ORDER VETTING PIPELINE UNWIRED**.

#### 3. Cross-Docking Peak Waves
- **Implementation**:
  - `internal/crossdock/models.go`: `CrossDockOrder` (`ALLOCATED`, `STAGED_AT_CROSSDOCK`, `LOADED_DIRECT`, `REVERTED_TO_PUTAWAY`, `CANCELLED`), `CrossDockScan`.
  - `internal/crossdock/engine.go`: Direct inbound PO-to-outbound manifest matching.
  - `internal/crossdock/service.go`: Transshipment allocation and pallet laser scan verification.
  - Endpoints under `/v1/warehouse/crossdock/*` in `router.go:969-979`.
- **Status**: **FULLY WIRED & FUNCTIONAL**.

#### 4. Blind Receiving & Barcode Variance Reconciliation
- **Implementation**:
  - `internal/inbound/models.go`: `ReceiveShipmentDTO` accepts `ReceivedQty`, `DamagedQty`, `LotNumber`, `ExpiryDate`.
  - `internal/inbound/service.go:185-225`: `RecordGoodsReceipt` updates stock balances and emits `inbound.goods_received`.
  - `internal/cyclecount/models.go:9-28`: Describes blind counting workflows.
- **Audit Finding / Gap**:
  - True "blind receiving" (where expected PO quantities are masked from dock operators, and any recorded discrepancy automatically generates an audited carrier shortage claim or adjustment note) is documented in architectural comments but **not enforced programmatically in `inbound/service.go`**. The handler allows passing arbitrary quantities without automated shortage claim reconciliation.
- **Status**: **PARTIAL / MANUAL GOODS RECEIPT ONLY**.

#### 5. Quarantine & Damaged Goods Segregation (`WH-QUARANTINE-01`)
- **Implementation**:
  - `internal/qm/service.go` & `internal/claims/repository.go:638-647`: Inserts damaged returns into `qm_quarantine_lots` with `is_atp_excluded = true`.
  - `internal/warehouse/models.go:103`: `WarehouseBinConfig.BinType` includes `'QUARANTINE'`.
  - `internal/warehouse/models.go:123`: `InitialStockConfig.Status` includes `'QUARANTINE'`.
  - `internal/api/handlers_empties_qm.go:278-390`: Handlers for `POST /v1/qm/quarantine`, `GET /v1/qm/warehouses/{warehouseID}/quarantine`, and `POST /v1/qm/quarantine/{lotID}/disposition`.
- **Audit Finding / Gap**:
  - The specific canonical quarantine bin code **`WH-QUARANTINE-01`** is **not hardcoded, seeded, or automatically assigned** when damaged goods are offloaded.
- **Status**: **QUARANTINE DOMAIN WIRED; STANDARD `WH-QUARANTINE-01` LOCATION CODE UNSEEDED**.

#### 6. FEFO Pick Allocation
- **Implementation**:
  - `internal/wms/lots.go:176-248`: `AllocateLotsFEFO` executes row-level locked `FOR UPDATE` query:
    ```sql
    SELECT lot_id, on_hand_qty, reserved_qty, expiry_date
    FROM stock_lots
    WHERE warehouse_id = $1 AND sku_id = $2 AND status = 'AVAILABLE' AND expiry_date >= $3
    ORDER BY expiry_date ASC, received_at ASC
    FOR UPDATE
    ```
  - `internal/wms/homogeneity.go:67-95`: Single-lot and multi-lot FEFO accumulation with lot homogeneity score calculation.
- **Status**: **FULLY WIRED & PRODUCTION-GRADE**.

---

### Role 3: Payloader & Picker (Dock Staging, 3L-CVRP & Axle Physics)

#### 1. Package Structure & Organization
- **Primary Package**: `backend/internal/payload/`
  - `models.go` (573 lines): `ManifestRow`, `PalletLoadPosition`, `AxleFeasibilityReq`, `AxleFeasibilityResp`, `SealManifestReq`, `SealManifestResp`, `LoadLine`, `ShipUnit`, `PayloaderEntity`.
  - `service.go` (1,421 lines): `CalculateAxleFeasibility`, `CheckAxleFeasibility`, `SealManifest`, `ScanLoadItem`, `ApproveVariance`, `EnsureShipUnits`, `RegisterPayloader`.
  - `repository.go` (1,655 lines): Defines `Repository` interface and `pgRepository`.

#### 2. 3L-CVRP Longitudinal Static Moment Axle Calculation
- **Implementation**:
  - Located in `internal/payload/service.go:293-428` (`CalculateAxleFeasibility`):
    $$W_{\text{steer}} = W_{\text{curb,steer}} + \sum_{i=1}^n \frac{w_i \cdot (L - x_i)}{L}$$
    $$W_{\text{drive}} = W_{\text{curb,drive}} + \sum_{i=1}^n \frac{w_i \cdot x_i}{L}$$
    $$W_{\text{gross}} = W_{\text{steer}} + W_{\text{drive}}$$
    $$\text{SteerRatio} = \frac{W_{\text{steer}}}{W_{\text{gross}}} \times 100\%$$
  - Parameters:
    - Wheelbase $L$: default `4200.0` mm.
    - Curb Front: default `2500.0` kg; Curb Rear: default `2300.0` kg.
    - Mechanical bounds check: pallet position $x \in [-500, L + 1500]$ mm (allows rear cantilever tail-lift overhangs).
- **Status**: **FULLY WIRED & MATHEMATICALLY EXACT**.

#### 3. Statutory 11,500 kg Single Axle Limit & $\ge 20\%$ Steer Tractive Authority
- **Implementation**:
  - `internal/payload/models.go:35-39`:
    - `MaxAllowedSingleAxleKg = 11500.0`
    - `MinSteerAxleShareRatio = 0.20`
  - `internal/payload/service.go:372-389`: Evaluates front axle overload ($W_{\text{steer}} > 11,500$ kg) and rear drive axle overload ($W_{\text{drive}} > 11,500$ kg).
  - `internal/payload/service.go:398-401`: Evaluates steer axle underload ($\text{SteerRatio} < 20.0\%$), flagging dangerous traction loss risk for mountain passes (Kamchik Pass).
  - `internal/payload/service.go:559-578`: Enforced in `SealManifest` as Physical Gate 1. Returns `ErrAxleOverloadViolation` if violated and `ForceAxleOverride` is false.
- **Status**: **FULLY WIRED & RIGOROUSLY ENFORCED**.

#### 4. Supervisor Axle & Capacity Manual Override
- **Implementation**:
  - `SealManifestReq` accepts:
    - `ForceAxleOverride`: bool
    - `AxleOverrideReason`: string (e.g. `ROAD_EMERGENCY_SPLIT`, `SPECIAL_PERMIT_ESCORT`)
    - `OverrideSupervisorID`: string
  - `internal/payload/service.go:580-589`: If `(isAxleOverloaded || isSteerTractionLost) && req.ForceAxleOverride`, logs `PAYLOAD_AXLE_OVERRIDE_TRIGGERED` and allows sealing to proceed.
  - Persisted in PostgreSQL via migration `072_warehouse_auto_approval_and_vetting_settings.sql`: `is_axle_overridden`, `axle_override_reason`, `axle_override_by`.
- **Audit Finding / Defect**:
  - While Payloader registration enforces 14-digit PINFL format (`pinflRegex = regexp.MustCompile(`^\d{14}$`)`), **`SealManifest` does not strictly validate that `OverrideSupervisorID` is a valid 14-digit PINFL** or verify supervisor credentials before applying the override.
- **Status**: **LOGICAL OVERRIDE WORKING; PINFL FORMAT VALIDATION MISSING ON OVERRIDE**.

#### 5. Bolt Seal Serial Number (`SEAL-UZ-XXXXXX`)
- **Implementation**:
  - `SealManifestReq.BoltSealSerial` is strictly mandatory (`strings.TrimSpace(req.BoltSealSerial) == "" -> error`).
  - Hashed into cryptographic SHA-256 `digital_seal_hash`:
    `fmt.Sprintf("%s:%s:%s:%.1f:%.1f:%s:OVERRIDE:%t:%s", manifestID, truckPlate, boltSealSerial, ...)`
  - Mints GS1 SSCC-18 ship units upon sealing (`EnsureShipUnits`).
- **Audit Finding / Defect**:
  - The code checks `strings.TrimSpace(req.BoltSealSerial) == ""` but **does not enforce the canonical regex pattern `^SEAL-UZ-[A-Z0-9]{6}$`**. Arbitrary strings are currently accepted.
- **Status**: **MANDATORY SEAL FIELD WIRED; REGEX `SEAL-UZ-XXXXXX` FORMAT NOT STRICTLY VALIDATED**.

#### 6. CRITICAL DEFECT: Mock Data / In-Memory Fallback in `internal/payload`
- **Location**: `internal/payload/repository.go:83-220, 367-375`
- **Evidence**:
  ```go
  type pgRepository struct {
      pool *db.Pool
      manifests map[string]*ManifestRow
      loadLedgers map[string][]LoadLine
      exceptions []ManifestException
      stagedOrders map[string]*PayloaderOrderSummary
      ...
  }
  func (r *pgRepository) seedInitialDockData() { ... } // Seeds mnf_tashkent_bay4_01, line_01, etc.
  ```
- **Analysis**:
  When `r.pool == nil` or when a query fails, `GetManifest`, `ListManifests`, `GetLoadLedger`, etc., silently fall back to `r.manifests` or `r.loadLedgers`. This violates the **Zero Mock Data Policy** in `AGENTS.md` and `GEMINI.md`. All dock data must be queried directly from PostgreSQL 16 tables (`manifests`, `manifest_stops`, `manifest_exceptions`, `gs1_ship_units`).
- **Severity**: **CRITICAL VIOLATION OF ZERO MOCK DATA POLICY**.

---

### Role 4: Dispatcher (Fleet Routing & Control Tower)

#### 1. Package Structure & Organization
- **Primary Packages**:
  - `backend/internal/dispatch/`:
    - `service.go` (863 lines): `PreviewDispatch`, `CommitDispatch`, `AcquireWarehouseLock`, `ReleaseWarehouseLock`, `ExecuteRescue`, `callSidecarCVRP`.
    - `binpack.go` (215 lines): Vehicle load packing heuristics (`PackVehicles`, `PackOrdersIntoVehicles`).
    - `localsearch.go` (105 lines): Native 2-Opt TSP route improvement (`OptimizeRoute2Opt`).
    - `rescue.go` (103 lines): `RankRescueCandidates`, `BuildRescueRoute`.
    - `fleet_rescue_models.go` (118 lines): `FleetRescueIncident`, `ReportBreakdownRequest`, `DispatchRescueIncidentRequest`, `FleetRescueKPI`.
    - `fleet_rescue_service.go` (466 lines): Breakdown reporting, incident proposal, and dispatch.
  - `backend/internal/fleet/`:
    - `models.go` (1,500 lines): Vehicles, Drivers, `DriverVehicleAssignment`, `VehicleInspection` (DVIR), GAWR ratings.
    - `repository.go` (2,500 lines): PostgreSQL 16 persistence for fleet and driver management.
    - `service.go` (1,800 lines): Bijective shift pairing, hot-swapping, DVIR inspection audits.

#### 2. Multi-Vehicle VRP Routing & Dispatch Planning
- **Implementation**:
  - `internal/dispatch/service.go:84-330` (`PreviewDispatch`):
    1. Acquires warehouse lock to eliminate concurrent dispatch races (`AcquireWarehouseLock`).
    2. Fetches warehouse coordinates and eligible unassigned orders (`status IN ('CONFIRMED', 'PACKED') AND dispatch_status = 'UNASSIGNED'`).
    3. Fetches on-shift drivers with strict enterprise logistics invariants (`driverQuery`, lines 224-261):
       - Active shift assignment (`dva.released_at IS NULL`).
       - Pre-trip DVIR inspection passed today (`vi.is_safe_to_operate = true`).
       - Vehicle operational status (`YARD_STANDBY`, `LOADING_AT_DOCK`, `ACTIVE_ON_ROAD`).
    4. Optimization: Calls Python OR-Tools sidecar if configured; falls back to native Go engine (`PackVehicles` + 2-Opt local search).
    5. Computes deterministic SHA-256 fingerprint (`ComputePlanFingerprint`).
  - `internal/dispatch/service.go:332-480` (`CommitDispatch`):
    - Atomically creates manifests, populates stops, updates order `dispatch_status = 'DISPATCHED'`, and emits outbox events inside a single `pgx.Tx`.
- **Status**: **PRODUCTION-GRADE & FULLY WIRED**.

#### 3. Live GPS Telemetry & Geofencing
- **Implementation**:
  - `internal/telemetry/ingestion.go`: Ingests GPS pings (`TelemetryPoint`), updates Redis geospatial index (`redis.UpdateDriverTelemetry`), maintains presence TTL (20s), and throttles map broadcasts (2.5s per vehicle).
  - `internal/telemetry/geofence.go`: `CalculateHaversineDistance` and `IsWithinGeofence` (default 150m radius). Supports `VerifyArrival` with urban canyon photo bypass.
- **Audit Finding / Gap**:
  - Proximity trigger in `geofence.go` is parameterized for 150m, whereas the prompt specification specifies a **100m proximity trigger** to pop up the dynamic handshake QR code on the Retailer terminal.
- **Status**: **TELEMETRY & HAVERSINE WIRED; 100M HANDSHAKE TRIGGER NEEDS STRICT PARAMETERIZATION**.

#### 4. Mid-Shift Road Breakdown Emergency Reporting
- **Implementation**:
  - `internal/dispatch/fleet_rescue_service.go:159-229` (`ReportBreakdown`):
    - Accepts `ReportBreakdownRequest`: stranded driver, vehicle, GPS coordinates, failure reason, remaining stops, cargo weight/volume.
    - Generates incident code (`RSC-2026-XXX`), sets status `ACTIVE_ALERT`.
    - Persists in PostgreSQL `fleet_rescue_incidents` table.
- **Status**: **FULLY WIRED & PERSISTED IN PG16**.

#### 5. Dynamic Rescue Hot-Swapping via Redis Streams & Atomicity
- **Implementation**:
  - `internal/dispatch/rescue.go:9-57` (`RankRescueCandidates`):
    - Filters active on-shift drivers (`d.DriverID != brokenDriverID && d.OnShift`).
    - Calculates available capacity with tetris packing buffer:
      $$\text{AvailCap} = (\text{MaxVolumeVU} \times 0.95) - \text{CurrentLoadVU}$$
    - Requires $\text{AvailCap} \ge \text{RequiredVolumeVU}$.
    - Calculates Haversine proximity distance to disabled vehicle.
    - Ranks candidates by available capacity descending, then proximity ascending.
  - `internal/dispatch/service.go:603-737` (`ExecuteRescue`):
    - Atomically executes within `pgx.Tx`:
      1. Cancels broken driver's active manifest (`status = 'CANCELLED'`).
      2. Marks broken driver `OFFLINE` and releases broken vehicle to `MAINTENANCE` with reason `BREAKDOWN_MID_SHIFT`.
      3. Resolves rescue driver's vehicle.
      4. Creates new rescue manifest for rescue driver.
      5. Reassigns undelivered stops to rescue manifest and updates orders (`driver_id = rescueDriverID, status = 'LOADED'`). **Zero retailer orders are canceled!**
      6. Emits atomic transactional outbox event: `FLEET_BREAKDOWN_RESCUED`.
    - After transaction commits: publishes real-time rescue route update to Redis.
- **Audit Finding / Gap**:
  - In `dispatch/service.go:709-734`, the post-commit alert is published via Redis Pub/Sub (`redis.Publish`) to `alerts:fleet:breakdown_rescue` and `events:FLEET`, rather than calling `redis.XAddFleetEvent` directly. However, the outbox record `FLEET_BREAKDOWN_RESCUED` is emitted in the transaction and will be published to Redis 7 Streams by `outbox/relay.go`.
- **Status**: **ATOMIC HOT-SWAP WORKING WITHOUT ORDER CANCELLATION**.

#### 6. CRITICAL DEFECT: Mock Data / In-Memory Fallback in `internal/dispatch`
- **Location**: `internal/dispatch/fleet_rescue_service.go:18-105, 109-123`
- **Evidence**:
  ```go
  var rescueIncidentsMu sync.RWMutex
  var rescueIncidents map[string]*FleetRescueIncident
  func init() { initRescueIncidents() } // Seeds RSC-2026-081, Javokhir Sobirov, etc.
  ```
- **Analysis**:
  When `s.pool == nil`, `ListRescueIncidents`, `GetRescueIncident`, `ProposeRescueIncident`, `ResolveRescueIncident`, and `GetRescueKPI` fall back to static mock seeds in `rescueIncidents`. Additionally, `PreviewDispatch` (lines 89-154) returns hardcoded routes (`drv_farrukh_01`, `01 772 AAA`) if `s.pool == nil`.
- **Severity**: **CRITICAL VIOLATION OF ZERO MOCK DATA POLICY**.

---

## 3. Comprehensive REST Endpoints Matrix for Roles 1–4

The following table documents all endpoints mounted in `pegasus.x/backend/internal/api/router.go` for Roles 1 through 4:

| Role | HTTP Method | Route Path | Handler Function | Description & Source Location |
| :--- | :--- | :--- | :--- | :--- |
| **Role 1 (Supplier)** | `POST` | `/v1/auth/supplier/register` | `handleSupplierRegister` | STIR legal deduplication & sign-up (`handlers_supplier.go:210`) |
| **Role 1 (Supplier)** | `POST` | `/v1/auth/supplier/login` | `handleSupplierLogin` | Password authentication & JWT issuance (`handlers_supplier.go:250`) |
| **Role 1 (Supplier)** | `GET` | `/v1/supplier/onboarding/status` | `handleSupplierOnboardingStatus` | Onboarding wizard progress tracker (`handlers_supplier.go:1115`) |
| **Role 1 (Supplier)** | `POST` | `/v1/supplier/onboarding/products` | `handleSupplierOnboardingCreateProduct` | Product creation with MXIK, EAN-13 Mod-10, tiyins (`handlers_supplier.go:1130`) |
| **Role 1 (Supplier)** | `GET` | `/v1/supplier/onboarding/products` | `handleSupplierOnboardingListProducts` | List supplier catalog products (`handlers_supplier.go:1209`) |
| **Role 1 (Supplier)** | `PUT` | `/v1/supplier/onboarding/products/{id}` | `handleSupplierOnboardingUpdateProduct` | Update product attributes and pricing (`handlers_supplier.go:1219`) |
| **Role 1 (Supplier)** | `DELETE` | `/v1/supplier/onboarding/products/{id}` | `handleSupplierOnboardingDeleteProduct` | Remove product from catalog (`handlers_supplier.go:1270`) |
| **Role 1 (Supplier)** | `POST` | `/v1/supplier/onboarding/payment` | `handleSupplierOnboardingConfigurePayment` | Configures Cash & Global Pay corporate card (`handlers_supplier.go:1290`) |
| **Role 1 (Supplier)** | `GET` | `/v1/supplier/onboarding/payment` | `handleSupplierOnboardingGetPayment` | Retrieve active payment gateway configs (`handlers_supplier.go:1320`) |
| **Role 1 (Supplier)** | `POST` | `/v1/supplier/onboarding/complete` | `handleSupplierOnboardingComplete` | Finalize onboarding; transition to COMPLETED (`handlers_supplier.go:1335`) |
| **Role 1 (Supplier)** | `GET` | `/v1/supplier/warehouses` | `handleSupplierListWarehouses` | List supplier fulfillment hubs (`handlers_supplier.go:1360`) |
| **Role 1 (Supplier)** | `POST` | `/v1/supplier/warehouses` | `handleSupplierCreateWarehouse` | Register new warehouse with GPS coordinates (`handlers_supplier.go:1380`) |
| **Role 1 (Supplier)** | `DELETE` | `/v1/supplier/warehouses/{id}` | `handleSupplierDeleteWarehouse` | Blocks deletion if active stock > 0 (`handlers_supplier.go:1420`) |
| **Role 1 (Supplier)** | `POST` | `/v1/supplier/pricing/preview` | `handlePreviewRetailerPricingOverride` | Volume tier discount & margin preview (`handlers_supplier.go:760`) |
| **Role 1 (Supplier)** | `POST` | `/v1/supplier/orders/vet` | `handleVetSupplierOrder` | Manual order approval/rejection logging (`handlers_supplier.go:785`) |
| **Role 1 (Supplier)** | `GET` | `/v1/catalog/products` | `handleCatalogListProducts` | Global catalog browse with floor prices (`handlers_catalog.go:40`) |
| **Role 1 (Supplier)** | `POST` | `/v1/catalog/products` | `handleCatalogCreateProduct` | Register SKU with units_per_case and VAT (`handlers_catalog.go:133`) |
| **Role 2 (Warehouse)** | `POST` | `/v1/auth/warehouse/register` | `handleWarehouseRegister` | Warehouse registration with Uzbekistan GPS check (`handlers_warehouse_auth.go:30`) |
| **Role 2 (Warehouse)** | `POST` | `/v1/auth/warehouse/login` | `handleWarehouseLogin` | STIR/phone + password authentication (`handlers_warehouse_auth.go:70`) |
| **Role 2 (Warehouse)** | `GET` | `/v1/warehouse/onboarding/status` | `handleGetWarehouseOnboardingStatus` | Onboarding state, bays, bins, stock counts (`handlers_warehouse_auth.go:110`) |
| **Role 2 (Warehouse)** | `POST` | `/v1/warehouse/onboarding/bays` | `handleConfigureDockBays` | Step 1: Configure loading dock bays (`handlers_warehouse_auth.go:130`) |
| **Role 2 (Warehouse)** | `POST` | `/v1/warehouse/onboarding/bins` | `handleConfigureWarehouseBins` | Step 2: Configure 3D bin locations (`handlers_warehouse_auth.go:170`) |
| **Role 2 (Warehouse)** | `POST` | `/v1/warehouse/onboarding/initial-stock` | `handleConfigureInitialStock` | Step 3: Seed initial FEFO stock lots (`handlers_warehouse_auth.go:210`) |
| **Role 2 (Warehouse)** | `POST` | `/v1/warehouse/onboarding/complete` | `handleCompleteWarehouseOnboarding` | Finalize commissioning verification (`handlers_warehouse_auth.go:250`) |
| **Role 2 (Warehouse)** | `GET` | `/v1/warehouse/approval-settings` | `handleGetWarehouseApprovalSettings` | Fetch auto-approval mode and UMP threshold (`handlers_warehouse_portal.go:591`) |
| **Role 2 (Warehouse)** | `PUT` | `/v1/warehouse/approval-settings` | `handleUpdateWarehouseApprovalSettings` | Update auto-approval mode & tiyin threshold (`handlers_warehouse_portal.go:592`) |
| **Role 2 (Warehouse)** | `GET` | `/v1/warehouse/metrics` | `handleGetWarehouseMetrics` | Live KPI banner metrics (`handlers_warehouse_portal.go:65`) |
| **Role 2 (Warehouse)** | `GET` | `/v1/warehouse/orders` | `handleGetWarehouseOrders` | Live warehouse order fulfillment list (`handlers_warehouse_portal.go:130`) |
| **Role 2 (Warehouse)** | `GET` | `/v1/warehouse/inventory` | `handleGetWarehouseInventory` | On-hand inventory with FEFO lot link (`handlers_warehouse_portal.go:186`) |
| **Role 2 (Warehouse)** | `PUT` | `/v1/warehouse/stock/adjust` | `handleAdjustWarehouseStock` | Audited stock adjustment with reason (`handlers_warehouse_portal.go:266`) |
| **Role 2 (Warehouse)** | `Route` | `/v1/warehouse/crossdock/*` | `handlers_crossdock.go` | Direct transshipment and buffer lane scans (`router.go:969`) |
| **Role 2 (Warehouse)** | `Route` | `/v1/warehouse/pick-waves/*` | `handlers_pickwave.go` | Batch wave creation, assignment, sealing (`router.go:999`) |
| **Role 2 (Warehouse)** | `Route` | `/v1/qm/quarantine/*` | `handlers_empties_qm.go` | Damaged stock ingestion and disposition (`router.go:1092`) |
| **Role 3 (Payloader)** | `POST` | `/v1/auth/payloader/register` | `handlePayloaderRegister` | Payloader sign-up with 14-digit PINFL check (`handlers_payloader_auth.go:30`) |
| **Role 3 (Payloader)** | `POST` | `/v1/auth/payloader/login` | `handlePayloaderLogin` | Phone + 4-digit PIN login (`handlers_payloader_auth.go:70`) |
| **Role 3 (Payloader)** | `GET` | `/v1/payloader/onboarding/status` | `handleGetPayloaderOnboardingStatus` | Terminal & calibration onboarding state (`handlers_payloader_auth.go:110`) |
| **Role 3 (Payloader)** | `POST` | `/v1/payloader/onboarding/bay-bind` | `handlePayloaderOnboardingBayBind` | Step 1: Bind payloader to dock bay (`handlers_payloader_auth.go:130`) |
| **Role 3 (Payloader)** | `POST` | `/v1/payloader/onboarding/axle-calibration`| `handlePayloaderOnboardingAxleCalibration` | Step 2: Calibrate scale and statics (`handlers_payloader_auth.go:160`) |
| **Role 3 (Payloader)** | `POST` | `/v1/payloader/onboarding/scanner-verify` | `handlePayloaderOnboardingScannerVerify` | Step 3: Optical barcode hardware verification (`handlers_payloader_auth.go:190`) |
| **Role 3 (Payloader)** | `POST` | `/v1/payloader/onboarding/complete` | `handleCompletePayloaderOnboarding` | Commission payloader for dock operations (`handlers_payloader_auth.go:220`) |
| **Role 3 (Payloader)** | `POST` | `/v1/payload/axle-feasibility` | `handlePayloaderAxleFeasibility` | 3L-CVRP static moment balance calculation (`handlers_payload.go:450`) |
| **Role 3 (Payloader)** | `POST` | `/v1/payloader/manifests/{id}/start-loading` | `handlePayloaderStartLoading` | Begin loading sequence at dock bay (`handlers_payload.go:80`) |
| **Role 3 (Payloader)** | `GET` | `/v1/payloader/manifests/{id}/load-ledger` | `handlePayloaderLoadLedger` | Get load verification ledger (`handlers_payload.go:110`) |
| **Role 3 (Payloader)** | `POST` | `/v1/payloader/manifests/{id}/scan` | `handlePayloaderScanItem` | Scan carton barcode at dock door (`handlers_payload.go:140`) |
| **Role 3 (Payloader)** | `POST` | `/v1/payloader/manifests/{id}/seal` | `handlePayloaderSealManifest` | Seal manifest with axle weights & bolt seal (`handlers_payload.go:280`) |
| **Role 3 (Payloader)** | `POST` | `/v1/payloader/manifest-exception` | `handlePayloaderManifestException` | Report overflow / damaged pallet at dock (`handlers_payload.go:210`) |
| **Role 3 (Payloader)** | `POST` | `/v1/payloader/reassign-order` | `handlePayloaderApplyReassign` | Reassign overflowing order to backup truck (`handlers_payload.go:250`) |
| **Role 4 (Dispatcher)** | `POST` | `/v1/dispatch/lock` | `handleLockWarehouseDispatch` | Acquire exclusive warehouse planning lock (`handlers_dispatch.go:30`) |
| **Role 4 (Dispatcher)** | `POST` | `/v1/dispatch/unlock` | `handleUnlockWarehouseDispatch` | Release warehouse planning lock (`handlers_dispatch.go:60`) |
| **Role 4 (Dispatcher)** | `POST` | `/v1/dispatch/preview` | `handlePreviewDispatch` | Multi-vehicle CVRP preview with 2-Opt (`handlers_dispatch.go:80`) |
| **Role 4 (Dispatcher)** | `POST` | `/v1/dispatch/commit` | `handleCommitDispatch` | Atomic manifest generation & stop sequencing (`handlers_dispatch.go:130`) |
| **Role 4 (Dispatcher)** | `POST` | `/v1/warehouse/dispatch/rescues` | `handleReportBreakdown` | Report mid-shift breakdown incident (`handlers_rescue.go:62`) |
| **Role 4 (Dispatcher)** | `GET` | `/v1/warehouse/dispatch/rescues` | `handleListRescueIncidents` | List active breakdown and rescue tickets (`handlers_rescue.go:36`) |
| **Role 4 (Dispatcher)** | `GET` | `/v1/warehouse/dispatch/rescues/{id}/candidates` | `handleGetRescueCandidates` | Rank on-shift trucks by spare capacity & proximity (`handlers_rescue.go:114`) |
| **Role 4 (Dispatcher)** | `POST` | `/v1/warehouse/dispatch/rescues/{id}/propose` | `handleProposeRescueIncident` | Propose rescuer truck and ETA (`handlers_rescue.go:169`) |
| **Role 4 (Dispatcher)** | `POST` | `/v1/warehouse/dispatch/rescues/{id}/dispatch` | `handleDispatchRescueIncident` | Dispatch rescue truck & hot-swap stops (`handlers_rescue.go:193`) |
| **Role 4 (Dispatcher)** | `POST` | `/v1/warehouse/dispatch/rescues/{id}/resolve` | `handleResolveRescueIncident` | Confirm transloading; mark ticket RESOLVED (`handlers_rescue.go:223`) |
| **Role 4 (Dispatcher)** | `POST` | `/v1/fleet/location` | `handleDriverLocation` | GPS telemetry ping ingestion (`handlers_fleet.go:120`) |
| **Role 4 (Dispatcher)** | `POST` | `/v1/warehouse/fleet/assign` | `handleAssignWarehouseDriverVehicle` | Shift driver-vehicle daily pairing (`handlers_warehouse_portal.go:589`) |
| **Role 4 (Dispatcher)** | `POST` | `/v1/warehouse/fleet/swap-vehicle` | `handleWarehouseSwapVehicle` | Vehicle hot-swap mid-shift (`handlers_warehouse_portal.go:731`) |
| **Role 4 (Dispatcher)** | `POST` | `/v1/warehouse/fleet/swap-driver` | `handleWarehouseSwapDriver` | Driver swap mid-shift (`handlers_warehouse_portal.go:686`) |
| **Role 4 (Dispatcher)** | `POST` | `/v1/fleet/dvir` | `handleSubmitDVIRInspection` | Pre-trip DVIR inspection submission (`handlers_fleet.go:350`) |

---

## 4. Key Gaps, Architectural Inconsistencies & Critical Defects

### 1. Zero Mock Data Policy Violations
- **`internal/payload/repository.go:83-220`**: Contains a massive in-memory repository fallback `r.manifests`, `r.loadLedgers`, `r.stagedOrders` seeded with static fake records (`mnf_tashkent_bay4_01`, `line_01`, `line_02`, `line_03`, `mnf_tashkent_bay6_02`, `ord_smoke_06`). When `r.pool == nil` or on query error, operations fall back to this in-memory cache.
- **`internal/dispatch/fleet_rescue_service.go:18-105`**: Contains in-memory fallback `rescueIncidents map[string]*FleetRescueIncident` seeded with fake incidents `RSC-2026-081`, `RSC-2026-080`, `RSC-2026-082`, `Javokhir Sobirov`, `01 341 BBA`.
- **`internal/dispatch/service.go:89-154`**: In `PreviewDispatch`, if `s.pool == nil`, it returns static hardcoded routes (`drv_farrukh_01`, `01 772 AAA`, `ord_korzinka_samarkand_darvoza`).
- **`internal/order/service.go:37,148-240`**: Maintains an in-memory order map `inMemoryOrders map[string]*models.Order` when `s.pool == nil`.
- **Remediation**: Purge all in-memory repository fallbacks and static seeds from production code. If `s.pool == nil`, return `ErrDatabaseNotConnected`.

### 2. Missing Catch Weight Architecture (Role 1)
- No domain model or service method currently supports variable weight / catch weight goods (e.g. bulk cheese, meat, produce) where order placement reserves nominal weight, and outbound dock scale weighing calculates actual certified weight and adjusts integer tiyin totals.
- **Remediation**: Add `IsCatchWeight bool`, `NominalWeightKg float64`, `ActualWeightKg float64`, and `CatchWeightTolerancePct float64` to `products`, `order_items`, and `load_lines`. Recalculate invoice minor units upon dock scale scanning.

### 3. Unwired Order Auto-Approval Pipeline (Role 2)
- While `warehouses` table stores `auto_order_approval_mode` and `ump_auto_apply_threshold_tiyin` (default 600,000 UZS = 60M tiyins), `order/service.go` never queries these settings. New orders go straight to `PENDING` without automatic vetting against the threshold.
- **Remediation**: In `order.Service.CreateOrder`, query the target warehouse's `auto_order_approval_mode` and `ump_auto_apply_threshold_tiyin`. If total $\le$ threshold and retailer credit is sound, automatically transition order to `CONFIRMED`; otherwise, route to `PENDING_VETTING` queue.

### 4. Unformalized Quarantine Location `WH-QUARANTINE-01` (Role 2)
- Although `qm_quarantine_lots` exists, the canonical warehouse location code `WH-QUARANTINE-01` is not seeded or automatically assigned to offloaded damaged goods.
- **Remediation**: Ensure migration and warehouse initialization scripts create a default bin `WH-QUARANTINE-01` (`zone_code = 'QM', bin_type = 'QUARANTINE'`) and route damaged returns to this bin.

### 5. Supervisor PINFL & Bolt Seal Regex Validation Gaps (Role 3)
- `internal/payload/service.go` allows manual axle and capacity override, but does not validate that `OverrideSupervisorID` matches a valid 14-digit PINFL.
- `SealManifest` checks that `BoltSealSerial` is non-empty, but does not enforce the national format `^SEAL-UZ-[A-Z0-9]{6}$`.
- **Remediation**: Add regex validation `pinflRegex.MatchString(req.OverrideSupervisorID)` and `boltSealRegex.MatchString(req.BoltSealSerial)`.

### 6. Redis Streams Direct Publication Gap (Role 4)
- In `internal/dispatch/service.go:709-734`, rescue alerts are published via Redis Pub/Sub (`redis.Publish`) rather than directly appending to Redis 7 Streams via `redis.XAddFleetEvent`.
- **Remediation**: Call `s.redis.XAddFleetEvent(ctx, "fleet.breakdown.rescued", rescuePayload)` in addition to the outbox relay.

---

## 5. Verification Commands & Test Results

The backend targets were compiled and tested via automated test suites:

```bash
cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
go test -v ./internal/supplier ./internal/warehouse ./internal/payload ./internal/dispatch ./internal/fleet
```

**Results**:
- `github.com/pegasus-x/core/internal/supplier`: **PASS**
- `github.com/pegasus-x/core/internal/warehouse`: **PASS**
- `github.com/pegasus-x/core/internal/payload`: **PASS**
- `github.com/pegasus-x/core/internal/dispatch`: **PASS**
- `github.com/pegasus-x/core/internal/fleet`: **PASS**

---

## 6. Summary Conclusion

The backend architecture in `pegasus.x` for Roles 1–4 has strong foundational domain modeling and mathematical implementation (3L-CVRP axle statics, GS1 Mod-10 check, 2-Opt VRP routing, atomic rescue hot-swapping without order cancellation). However, significant gaps remain to reach full production grade:
1. Purging in-memory fallback maps and fake seeds from `internal/payload/repository.go` and `internal/dispatch/fleet_rescue_service.go`.
2. Implementing the Catch Weight domain workflow for Role 1.
3. Wiring the warehouse auto-approval threshold into `order.Service.CreateOrder` for Role 2.
4. Enforcing 14-digit PINFL and `SEAL-UZ-XXXXXX` regex checks for Role 3.
5. Standardizing Redis 7 Stream `XADD` publication across all fleet rescue events for Role 4.
