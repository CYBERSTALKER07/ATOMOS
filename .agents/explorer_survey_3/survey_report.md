# Architectural Survey Report: Roles 5-7 Backend Packages, Retailer Scope, & Test Baseline

**Surveyor**: Explorer Survey 3  
**Target Codebase**: `pegasus.x` (Sovereign Lean Single-Tenant National Operating Core: PostgreSQL 16 + Redis 7 Streams)  
**Date**: 2026-09-23  
**Status**: COMPLETE  

---

## 1. Executive Summary & Monorepo Reality Check

This survey provides a comprehensive line-by-line architectural investigation into the backend packages and client surfaces of `pegasus.x`, focusing specifically on:
- **Role 5: Driver** (Field delivery, shift readiness, DVIR, doorstep custody handover, dual-tender settlement, ePoD).
- **Role 6: Retailer** (B2B wholesale procurement terminal ONLY; verification of zero store POS, shelf counting, or cashier shifts).
- **Role 7: Finance & Auditor** (Soliq 12% VAT, Soliq OFD fiscalization, double-entry general ledger balance, driver cash-in-transit CIT thresholds, depot vault drops, and bank deposit reconciliation).
- **Backend Test Baseline**: Statement coverage, existing test targets, passing tests, and stub/mock masks.
- **REST API Route Registry**: Complete inventory of all endpoints mounted in `internal/api/router.go` for Roles 5–7.

### Core Discoveries & Urgent Flags
1. **Critical Architectural Scope Violation in Role 6 (Retailer)**:
   - The root doctrine across `AGENTS.md`, `GEMINI.md`, and the approved specification (`prompt_draft.md`) strictly commands:
     > *"Retailer clients (Desktop Tauri, Android Compose, iOS SwiftUI, Telegram MiniApp) are strictly procurement and wholesale ordering surfaces. Zero in-store POS, shelf cycle-counting, cashier shifts, or store product management."*
   - **Codebase Reality**: PostgreSQL Migration `055_retailer_pos_shifts_and_auto_order.sql`, package `internal/retailer/`, handler file `handlers_retailer.go`, test suite `retailer_pos_auto_order_e2e_test.go`, and client surfaces across Desktop (`app/(portal)/pos/page.tsx` - 1,087 lines), Android (`PosTerminalScreen.kt` - 46,494 bytes, `StockScreen.kt` - 20,520 bytes), and iOS (`PosTerminalView.swift`, `StockView.swift`) have implemented a full in-store retail POS, till registers, cashier shifts, cash drops, parked carts, and store inventory shelf-counting system!
   - This represents a major architectural pollution of the Retailer's wholesale procurement boundary that must be cleanly purged or decoupled from the B2B wholesale procurement core.
2. **Two Parallel & Inconsistent Delivery Pipelines for Role 5 (Driver)**:
   - Package `internal/epod/` implements real PostgreSQL transactions (`SubmitEPODTx`, `MarkStopArrivedTx`, `ProcessOfflineDeliveryTx`) against `manifest_stops` and `epod_records`.
   - In contrast, package `internal/fleet/repository.go` and `handlers_fleet_driver.go` implement duplicate delivery routes (`/v1/delivery/arrive`, `/v1/order/deliver`, `/v1/delivery/scan-qr`, `/v1/delivery/partial-offload`, `/v1/order/validate-qr`) that bypass `epod` and return **in-memory mock stubs** (e.g. hardcoded 5,400,000 / 600,000 UZS values, dummy QR validations, and fake credit note strings).
3. **Test Suite Baseline & Mock Masking**:
   - `go test ./...` in `pegasus.x/backend` passes 100% cleanly (0 failures across all 83 packages).
   - However, numerous tests (e.g. `driver_mobile_e2e_test.go`, `retailer_pos_auto_order_e2e_test.go`) pass only because they assert against in-memory repository fallbacks and mock handlers rather than verified PostgreSQL 16 persistence.
4. **Finance & Audit Status (Role 7)**:
   - Statutory 12% Soliq VAT is robustly implemented with 64-bit integer minor unit math in `internal/fiscal/calculator.go`.
   - Soliq OFD E-Factura PKCS#7 signing is implemented in `internal/soliq/service.go`.
   - Double-entry bookkeeping invariant ($\sum \text{Debits} == \sum \text{Credits}$) is strictly verified in `internal/payment/handover.go` and `internal/payment/globalpay_reconciler.go`.
   - **Gaps**: Driver Cash-In-Transit (CIT) threshold alerts (e.g. 100M UZS), mid-shift depot vault drop workflows, and commercial bank deposit reconciliation are completely missing.

---

## 2. Role 5: Driver Backend Packages & Feature Deep Dive

### 2.1 Pre-Trip DVIR Inspection Checklist
- **Location**:
  - Models: `internal/fleet/models.go:723` (`DriverDVIRRequest`, `DriverDVIRResponse`).
  - Service: `internal/fleet/service.go:1479` (`SubmitDriverDVIR`), `1575` (`GetDriverDVIR`).
  - Repository: `internal/fleet/repository.go:1329-1440` (`RecordInspection`), `2653` (`UpdateDriverDVIRPassed`).
  - Routes: `POST /v1/driver/onboarding/dvir`, `GET /v1/driver/onboarding/dvir`, `POST /v1/fleet/dvir`, `GET /v1/fleet/dvir`.
- **Implementation Reality**:
  - `DriverShiftOnboardingGate` in `internal/api/router.go:2070-2165` intercepts requests to operational dispatch routes (`/v1/fleet/*`, `/v1/dispatch/*`, `/v1/driver/active-route`) with HTTP 428 Precondition Required until driver completes step 3 DVIR.
  - Rejection of failed brakes/tires/lights is enforced with HTTP 422 Unprocessable Entity.
  - Tests verify this gate in `driver_onboarding_e2e_test.go:274-330` and `cross_role_golden_path_e2e_test.go:742-775`.
- **Gaps**:
  - `RefrigerationTempC` in `DriverDVIRRequest` is an optional float pointer (`*float64`), but for cold chain manifests, prompt_draft.md requires an enforced hard-stop if compartment temperature exceeds the statutory range (+2°C to +6°C for chilled, -18°C for frozen).

### 2.2 100-Meter Proximity Trigger
- **Location**:
  - `internal/epod/service.go:80-95` (`MarkStopArrived` emits Redis event `delivery.driver.arrived` with `handshake.proximity_meters = 100`).
  - `internal/fleet/service.go:1015-1034` (`DeliveryArrive` publishes Redis event `events:delivery` with `type = "ORDER_ARRIVED"`).
  - Routes: `POST /v1/delivery/arrive`, `POST /v1/warehouse/manifests/{id}/stops/{orderID}/arrive`.
- **Gaps**:
  - `fleet/repository.go:2447` has a stub `DeliveryArrive` that simply returns `Status: "ARRIVED"` without calculating distance!
  - Neither handler verifies that Haversine distance between driver GPS coordinates `(lat, lng)` and the retailer store doorstep is $\le 100\text{ meters}$.
  - Urban canyon drift bypass is implemented in `internal/doorstep/edge_cases.go:74` (`UnlockUrbanCanyonDrift`), but it is not cleanly wired into the driver mobile workflow.

### 2.3 Dynamic OTP/QR Token Handshake
- **Location**:
  - Route: `POST /v1/order/validate-qr` (`handlers_fleet_driver.go:565`).
  - Service: `fleet/service.go:1045` (`ScanDeliveryQR`), `fleet/repository.go:2258` (`ValidateQR`).
- **Critical Defect / Theatre Found**:
  - In `internal/fleet/repository.go:2258-2273`:
    ```go
    func (r *Repository) ValidateQR(ctx context.Context, req ValidateQRRequest) (*ValidateQRResponse, error) {
        if req.QRToken == "" {
            return &ValidateQRResponse{Valid: false, ...}, nil
        }
        return &ValidateQRResponse{
            Valid: true,
            OrderID: req.OrderID,
            StoreName: "Korzinka Chilonzor Branch",
            Message: "Doorstep OTP QR verified. Custody handover unlocked.",
        }, nil
    }
    ```
  - Any non-empty string is treated as valid! It returns a hardcoded string `"Korzinka Chilonzor Branch"`. Zero cryptographic TOTP, zero nonce expiration, zero validation against the retailer's active session.

### 2.4 Itemized Offload Screen & Damaged Carton Rejection
- **Location**:
  - Models: `internal/fleet/models.go:1029-1048` (`PartialOffloadItem`, `PartialOffloadRequest`, `PartialOffloadResponse`).
  - Routes: `POST /v1/driver/orders/{orderId}/partial-offload`, `POST /v1/delivery/partial-offload`.
  - Repository: `internal/fleet/repository.go:2289-2312`.
- **Critical Defect / Stub Found**:
  - In `internal/fleet/repository.go:2299-2302`:
    ```go
    if deliveredMinor == 0 && rejectedMinor == 0 {
        deliveredMinor = 540000000 // 5,400,000 UZS
        rejectedMinor = 60000000   // 600,000 UZS (e.g. 2 damaged cartons)
    }
    return &PartialOffloadResponse{
        OrderID: req.OrderID,
        DeliveredMinor: deliveredMinor,
        CreditNoteIssued: rejectedMinor > 0,
        CreditNoteID: "cn_part_" + uuid.New().String()[:8],
        CreditNoteAmountMinor: rejectedMinor,
        Status: "PARTIALLY_DELIVERED",
    }, nil
    ```
  - It does NOT update PostgreSQL `orders` or `manifest_stops`, does NOT create a credit note in `credit_notes`, and does NOT emit an outbox event.
  - In `internal/epod`, adjustments are passed to `umpEngine.ProcessAdjustment`, but this is disconnected from the `/v1/delivery/partial-offload` endpoint.

### 2.5 Native Camera Lockout (Gallery Upload Blocked)
- **Codebase Reality**:
  - `GET /v1/media/upload-ticket` (`handlers_fleet_driver.go:1178`) generates an upload URL (`/v1/media/upload/asset_xxxx.jpg`).
  - **Zero gallery blocking logic exists**. Neither backend nor mobile models enforce capture metadata (`CAPTURE_SOURCE=CAMERA_DIRECT`), camera EXIF validation, or anti-tamper device attestation to prevent drivers from picking recycled photos from the phone gallery.

### 2.6 Real-Time Bilateral Tiyin Recalculation
- **Codebase Reality**:
  - Tiyin math is used in `fleet.PartialOffloadItem.UnitPrice`, but recalculation results are only returned in the HTTP response to the driver.
  - There is NO real-time push (via Redis Streams `events:doorstep:recalculated` or WebSocket) to update the retailer's terminal screen bilaterally before tender settlement.

### 2.7 Dual-Tender Doorstep Settlement & Unrestricted Cash Collection
- **Location**:
  - Routes: `POST /v1/delivery/split-payment`, `POST /v1/payments/handover`, `POST /v1/payment/globalpay/charge`.
  - Logic: `internal/payment/handover.go:94-255` (`ProcessStorefrontHandover`).
- **Status**:
  - `ProcessStorefrontHandover` accurately supports split cash + corporate card tenders and calculates double-entry ledger postings.
  - Unrestricted cash is supported (`MaxB2BCashLimitMinor = math.MaxInt64` in `internal/fiscal/calculator.go:19`).
  - **Defect in Fleet Handler**: `handleDriverSplitPayment` routes to `fleetSvc.SplitPayment` -> `fleet/repository.go:2325`, which is a stub returning dummy JSON without persisting payment legs into PostgreSQL!

### 2.8 Soliq OFD Fiscal QR Receipt
- **Location**:
  - `internal/soliq/service.go:65-220` (`GenerateAndSignFactura`).
  - Persists to `signed_facturas` and generates `qr_verification_url` (`https://my.soliq.uz/invoice/qr?...`).
  - Route: `POST /v1/order/{orderId}/fiscal/retry` calls `fleetSvc.RetryFiscal`, which in `fleet/repository.go:2484` returns a fake URL `"https://ofd.soliq.uz/receipt/" + req.OrderID` rather than invoking `soliqSvc`.

### 2.9 Electronic Proof of Delivery (ePoD)
- **Location**:
  - Models: `internal/epod/models.go:81-98` (`EPODRecord`).
  - Service: `internal/epod/service.go:106-182` (`SubmitEPOD`).
  - Route: `POST /v1/warehouse/manifests/{id}/stops/{orderID}/confirm-epod`.
- **Status**:
  - `internal/epod` has full SVG signature storage, geofence verification, cash collected tracking, and database commits (`SubmitEPODTx`).
  - However, `POST /v1/order/deliver` in `handlers_fleet_driver.go:928` calls `fleetSvc.OrderDeliver` which is an empty stub returning `Success: true`. The two pipelines must be unified.

---

## 3. Role 6: Retailer Scope & Complete Architecture Audit

### 3.1 Strict Procurement Terminal Scope Mandate vs Codebase Reality

| Feature / Domain | Architectural Doctrine (AGENTS.md / Specs) | Current Codebase Implementation | Discrepancy Status |
| :--- | :--- | :--- | :--- |
| **Store POS / Tills** | **STRICTLY ZERO**. Procurement terminal only. | `retailer_registers`, `retailer_pos_sales`, `retailer_pos_holds` in Migration 055 & `internal/retailer`. `app/(portal)/pos/page.tsx` (1,087 lines) in Desktop. `PosTerminalScreen.kt` (46 KB) in Android. `PosTerminalView.swift` in iOS. | **CRITICAL VIOLATION** |
| **Cashier Shifts & Drops** | **STRICTLY ZERO**. No cashier shifts or store drawer tracking. | `retailer_shifts`, `retailer_cash_drops` in Migration 055. Handlers in `handlers_retailer.go:1230-1236`. Tested in `retailer_pos_auto_order_e2e_test.go:69-119`. | **CRITICAL VIOLATION** |
| **Store Stock & Shelf Counts** | **STRICTLY ZERO**. No in-store retail stock management or cycle counts. | `retailer_store_stock`, `retailer_store_sections`, `retailer_shelf_alerts`, stock count sessions in Migration 055. `StockScreen.kt` (20 KB) in Android. `StockView.swift` in iOS. | **CRITICAL VIOLATION** |
| **Wholesale Catalog Discovery** | Fully supported (Tiered MOQ breaks, MXIK, tiyins). | Implemented via `/v1/supplier/products`, `/v1/retailer/catalog/local-skus`, and wholesale cart sync. | **Compliant** |
| **Live Inbound Truck Map Tracking** | Required live map tracking with MapLibre + Carto. | `TrackingMap.tsx` in `retailer-desktop` has MapLibre, but `tracking/page.tsx` does NOT use it (only text list). Android `DeliveryTrackingScreen.kt` uses static fake mock data. | **Deficient / Unwired** |
| **100m Proximity Handshake Pop-up** | Required dynamic pop-up with rotating QR/OTP token upon 100m geofence trigger. | **Missing across all 4 retailer clients** (Desktop, Android, iOS, MiniApp). No dynamic QR generation endpoint on backend. | **CRITICAL GAP** |
| **Doorstep Inspection & Offload** | Itemized carton review and damaged item rejection. | Android `InboundDockScreen.kt` has pallet count and claim dialog, but falls back to fake mock claims. No bilateral offload review screen. | **Partial / Stubbed** |
| **Soliq Fiscal Receipt Download** | Immediate view and download of Soliq OFD fiscal QR receipt and signed ePoD. | Missing dedicated retailer download endpoint for receipt and certificate. | **Missing** |

### 3.2 Evidence of Architectural Violation in Migration 055 & Code
- File: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/database/migrations/055_retailer_pos_shifts_and_auto_order.sql`:
  - Lines 8–18: `retailer_registers` (Tills / Hardware POS terminals).
  - Lines 21–38: `retailer_shifts` (Cash register shifts: `opening_float_tiyins`, `closing_cash_tiyins`, `variance_tiyins`).
  - Lines 40–50: `retailer_cash_drops` (Cash drops & paid-in).
  - Lines 52–71: `retailer_pos_sales` (`receipt_number`, `lines JSONB`, `tenders JSONB`).
  - Lines 73–85: `retailer_pos_holds` (Parked cart holds).
  - Lines 88–101: `retailer_store_stock` (`on_hand_qty`, `safety_stock_qty`, `reorder_point`).
  - Lines 120–143: `retailer_store_sections` and `retailer_shelf_alerts`.
- File: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/router.go`:
  - Lines 1226–1304: 35 endpoints exposing registers, shifts, POS sales, cart holds, stock adjustments, store sections, and shelf alerts.
- File: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/retailer_pos_auto_order_e2e_test.go`:
  - 399 lines testing POS sales, register creation, shift open/close, cash drops, parked carts, and voiding sales.
- File: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/retailer-desktop/app/(portal)/pos/page.tsx`:
  - 1,087 lines of frontend code implementing a point-of-sale checkout cash register.
- File: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/PosTerminalScreen.kt`:
  - 46,494 bytes implementing an Android POS cash register.

### 3.3 Root Cause & Remediation Strategy
Migration 055 was introduced during an earlier exploration phase without enforcing the strict boundary that PegasusX Retailer is a **B2B wholesale buyer terminal** that places orders to FMCG distributors, NOT a grocery store cashier till.  
**Remediation for Implementation Phase**:
1. Quarantine or decommission POS, cashier shifts, and shelf counting from the active wholesale API routes.
2. Replace POS pages on client apps with the required **B2B Wholesale Procurement Terminal**:
   - Order placement & catalog discovery with MOQ volume tiers.
   - Live Inbound Truck Telemetry Map (MapLibre + Carto Positron/Dark Matter).
   - 100m Proximity Handshake Pop-up Modal with Dynamic Rotating QR/OTP.
   - Doorstep Delivery Review & Tender Choice.
   - Soliq Fiscal Receipt Viewer & PDF Download.

---

## 4. Role 7: Finance & Auditor Deep Dive

### 4.1 Statutory Soliq 12% VAT Calculation
- **Location**: `internal/fiscal/calculator.go:10-150`.
- **Implementation**:
  - `DefaultVatRateBps = 1200` (12.00%).
  - Invariant: `grossMinor == netMinor + vatMinor` strictly enforced.
  - Banker's half-up rounding implemented using integer math: `(netMinor * vatRateBps + 5000) / 10000`.
  - Tested in `fiscal/calculator_test.go`: 100% passing.

### 4.2 Soliq OFD Fiscalization & E-Factura Signing
- **Location**:
  - `internal/soliq/service.go`: `GenerateAndSignFactura`, `GenerateAndSignCorrectiveFactura`.
  - `internal/soliq/eimzo.go`: E-IMZO software signer using RSA-SHA256 PKCS#7 envelope.
  - `internal/soliq/efactura.go`: Standardized EFactura document structures matching Uzbekistan Soliq EHF standards.
  - Database: `signed_facturas` table in migration `067_soliq_facturas.sql`.
- **Status**: Production-ready and fully tested in `soliq_e2e_test.go`.

### 4.3 Double-Entry General Ledger Balance ($\sum \text{Debits} == \sum \text{Credits}$)
- **Location**:
  - PostgreSQL Schema: Migration `005_split_payments_debts_and_ledger.sql:50-67`:
    - `ledger_journal_entries`: `entry_id`, `transaction_ref`, `entry_type`, `memo`, `occurred_at`.
    - `ledger_postings`: `posting_id`, `entry_id`, `account_code`, `account_type`, `direction` ('DEBIT'/'CREDIT'), `amount_minor` (BIGINT), `currency`, `party_id`.
  - Logic: `internal/payment/handover.go:228-241`:
    ```go
    if sumDebits != sumCredits {
        return nil, fmt.Errorf("%w: debits %d != credits %d", ErrUnbalancedJournalEntry, sumDebits, sumCredits)
    }
    ```
  - Accounts used: `CASH:DRIVER:{id}` (Asset), `PSP:GATEWAY:GLOBAL_PAY` (Asset), `ESCROW:ORDER:{id}` (Liability), `WALLET:RETAILER:{id}` (Liability), `AR:RETAILER:{id}` (Asset).
- **Status**: Double-entry invariant is mathematically sound and tested.
- **Gap**: Handover postings are calculated in Go memory, but not consistently persisted to `ledger_postings` table in every delivery route.

### 4.4 Driver Cash-in-Transit (CIT) Drawer Thresholds
- **Requirement**: Real-time tracking of cash in driver safe; trigger alert and mid-shift vault drop recommendation when cash exceeds insurance threshold (e.g. 100M UZS).
- **Current Status**: **COMPLETELY MISSING**.
  - `internal/fleet` and `internal/cashrecon` have zero threshold checks on active route cash accumulations.

### 4.5 Mid-Shift Depot Vault Drops
- **Requirement**: Workflow allowing a driver on route with high cash volume to drop physical cash into a regional depot smart safe or armored courier vault, resetting their CIT risk without completing their shift.
- **Current Status**: **MISSING**.
  - Only end-of-shift cash turn-in (`/v1/fleet/driver/cash-bag/turn-in` or ADM drop) exists.

### 4.6 End-of-Shift Bank Deposit Reconciliation
- **Location**: `internal/cashrecon/service.go:393-450` (`RecordCashDeposit`), `handlers_cashrecon.go:125`.
- **Status**:
  - Driver cash deposit into depot smart safe or cashier is recorded.
  - Automatically compares `actualCashMinor` against expected cash from `manifest_stops`, flags discrepancy (`DISCREPANCY_FLAGGED`), blocks driver from next shift, and allows supervisor override (`SupervisorOverride`).
- **Gap**: Lacks reconciliation between depot smart safe collections and commercial bank cash deposits (bank deposit slips, armored CIT bag handovers, bank statement reconciliation).

---

## 5. Test Baseline & Coverage Analysis

### 5.1 Test Execution Results (`go test -cover ./...`)
- **Total Packages Scanned**: 85 packages (83 internal + 2 cmd).
- **Compilation Status**: Clean, 0 syntax or type errors.
- **Test Execution Status**: **100% PASS** (0 failing tests).
- **Total Test Files**: 112 test files across `pegasus.x/backend`.

### 5.2 Package Statement Coverage Breakdown

| Package | Statement Coverage | Test Files | Primary Focus / Notes |
| :--- | :--- | :--- | :--- |
| `internal/marketpack` | 94.9% | 2 | Market pack configuration & policies |
| `internal/crm` | 92.6% | 1 | RFM classification & customer profiles |
| `internal/commission` | 91.4% | 1 | Sales rep commission clawbacks |
| `internal/offline` | 89.3% | 1 | Subterranean offline sync engine |
| `internal/planning` | 87.4% | 3 | S&OP planning & replenishment |
| `internal/adm` | 86.0% | 1 | ADM smart safe IoT hardware drops |
| `internal/aiorder` | 86.0% | 2 | Autonomous voice & AI order intake |
| `internal/speech` | 84.6% | 1 | Uzbek/Russian speech-to-text processing |
| `internal/legal` | 84.1% | 1 | Legal contracts & terms compliance |
| `internal/observability` | 84.1% | 1 | Prometheus metrics & structured logs |
| `internal/ewm` | 83.4% | 1 | Extended warehouse management |
| `internal/fiscal` | 82.1% | 2 | 12% VAT calculator & FX indexation |
| `internal/multisupplier` | 81.7% | 1 | Multi-supplier parent/child split orders |
| `internal/allocation` | 81.6% | 1 | Inventory allocation algorithms |
| `internal/secrets` | 81.8% | 1 | HashiCorp Vault secrets provider |
| `internal/spatial` | 81.2% | 2 | Uber H3 spatial indexing & surge |
| `internal/compliance` | 79.8% | 2 | Asl Belgisi & GS1 compliance |
| `internal/config` | 79.6% | 1 | Environment & Vault config validation |
| `internal/opex` | 77.8% | 1 | Operating expense allocations |
| `internal/returns` | 77.8% | 1 | Reverse logistics & RMA workflows |
| `internal/doorstep` | 77.5% | 1 | Urban canyon bypass & shop closed logic |
| `internal/payroll` | 77.3% | 1 | Driver/payloader payroll calculations |
| `internal/softpos` | 77.3% | 1 | SoftPOS EMV card processing |
| `internal/regional` | 76.8% | 1 | Regional warehouse zones |
| `internal/hrm` | 73.7% | 1 | Staff shift clock-in/out |
| `internal/telemetry` | 70.3% | 1 | Driver GPS Redis ingestion |
| `internal/notifications` | 68.9% | 1 | Multi-channel notifications |
| `internal/ar` | 68.7% | 1 | Accounts receivable & dunning ladder |
| `internal/onboarding` | 67.8% | 1 | Multi-role onboarding workflows |
| `internal/controltower` | 67.4% | 1 | Playbooks & incident runs |
| `internal/promotion` | 67.3% | 1 | Trade promotions & volume tiers |
| `internal/copa` | 65.6% | 1 | Profitability analysis |
| `internal/gs1core` | 65.5% | 1 | GS1 EAN-13 & SSCC barcodes |
| `internal/loyalty` | 64.8% | 1 | Retailer cashback loyalty |
| `internal/soliq` | 64.1% | 4 | E-Factura generation & E-IMZO signing |
| `internal/forecasting` | 63.8% | 1 | Croston-SBA demand forecasting |
| `internal/payment` | 63.2% | 4 | Handover settlement & GlobalPay |
| `internal/commitments` | 63.1% | 1 | Preorders & stock commitments |
| `internal/fxrates` | 62.0% | 1 | CBU exchange rate fetcher |
| `internal/scheduling` | 61.7% | 1 | Delivery corridor schedules |
| `internal/qm` | 61.7% | 1 | Quality quarantine segregation |
| `internal/floorexception` | 57.9% | 1 | Warehouse floor exceptions |
| `internal/payload` | 57.2% | 1 | 3L-CVRP axle statics & override |
| `internal/crossdock` | 56.3% | 1 | Cross-docking wave generation |
| `internal/fscm` | 56.1% | 1 | Financial supply chain management |
| `internal/auth` | 55.9% | 2 | JWT auth & Telegram verification |
| `internal/seasonalcore` | 54.7% | 1 | Ramadan & seasonal demand multipliers |
| `internal/empties` | 54.5% | 1 | Returnable transport packaging (RTI) |
| `internal/bins` | 52.2% | 1 | WMS bin slotting |
| `internal/dispatch` | 51.2% | 2 | VRP dispatch & bin packing |
| `internal/epod` | 50.1% | 1 | Electronic proof of delivery |
| `internal/pickwave` | 49.6% | 1 | Pick wave clustering |
| `internal/payout` | 48.0% | 1 | Supplier disbursement batches |
| `internal/coverage` | 47.9% | 1 | Geographic delivery tariffs |
| `internal/api` | **47.6%** | **32** | HTTP REST router & E2E integration suites |
| `internal/onec` | 44.9% | 1 | 1C Enterprise ERP sync |
| `internal/transfer` | 42.9% | 1 | Inter-warehouse transfers |
| `internal/wms` | 39.4% | 1 | Warehouse storage & optical gate |
| `internal/consignment` | 39.2% | 1 | Consignment stock tracking |
| `internal/fleet` | **37.8%** | 3 | Fleet management & driver operations |
| `internal/matching` | 36.4% | 1 | Payment leg matching |
| `internal/wmsops` | 36.0% | 1 | Warehouse operations board |
| `internal/rebate` | 36.0% | 1 | Supplier volume rebates |
| `internal/cyclecount` | 35.9% | 1 | Cycle counting & adjustments |
| `internal/cashrecon` | **34.5%** | 1 | Cash reconciliation & smart safe |
| `internal/warehouse` | 32.3% | 1 | Warehouse topology & auto-approval |
| `internal/credit` | 31.3% | 1 | Trade credit quota & PIN delegation |
| `internal/supplier` | 24.1% | 2 | Supplier portal & onboarding |
| `internal/ws` | 23.4% | 1 | Real-time WebSocket hub |
| `internal/retailer` | **22.7%** | 1 | Retailer domain & auto-order |
| `internal/order` | 21.0% | 1 | Order processing & reservation |
| `internal/dock` | 13.1% | 1 | Dock bay scheduling |
| `internal/inbound` | 10.3% | 1 | Dock receiving & ASNs |
| `internal/geolocation` | 8.3% | 1 | Geocoding & reverse geocoding |
| `internal/claims` | 6.1% | 1 | Damage claims |
| `internal/manifest` | 4.9% | 1 | Delivery manifests |
| `internal/creditnote` | **0.0%** | 0 | Corrective credit notes (untested) |
| `internal/db` | **0.0%** | 0 | Database pool wrapper |
| `internal/inventory` | **0.0%** | 0 | Inventory services (untested) |
| `internal/models` | **0.0%** | 0 | Domain struct definitions |
| `internal/outbox` | **0.0%** | 0 | Outbox relay worker |
| `internal/redis` | **0.0%** | 0 | Redis client wrapper |
| `internal/ump` | **0.0%** | 0 | Unit movement protocol |
| `cmd/server` | **0.0%** | 0 | Server entry point |
| `cmd/smokecheck` | **0.0%** | 0 | Smokecheck CLI suite |

---

## 6. REST API Endpoint Catalog for Roles 5–7

### 6.1 Role 5: Driver Endpoints

| Method | Route | Handler Function | Purpose | Database Persistence Status |
| :--- | :--- | :--- | :--- | :--- |
| `POST` | `/v1/auth/driver/login` | `handleDriverLogin` | Driver phone + 4-digit PIN authentication | Dynamic PG16 lookup |
| `GET` | `/v1/driver/onboarding/status` | `handleGetDriverOnboardingStatus` | Driver shift onboarding progress (4 steps) | Dynamic PG16 lookup |
| `POST` | `/v1/driver/onboarding/compliance` | `handleDriverOnboardingCompliance` | Driver medical & license verification | Dynamic PG16 commit |
| `POST` | `/v1/driver/onboarding/vehicle-pairing`| `handleDriverOnboardingVehiclePairing` | Dynamic shift vehicle pairing | Dynamic PG16 commit |
| `POST` | `/v1/driver/onboarding/dvir` | `handleDriverOnboardingDVIR` | Pre-trip digital DVIR safety inspection | Dynamic PG16 commit |
| `GET` | `/v1/driver/onboarding/dvir` | `handleGetDriverOnboardingDVIR` | Fetch latest pre-trip DVIR inspection | Dynamic PG16 query |
| `POST` | `/v1/driver/onboarding/complete` | `handleCompleteDriverOnboarding` | Finalizes shift readiness, unlocks dispatch gate | Dynamic PG16 commit |
| `GET` | `/v1/driver/profile` | `handleDriverProfile` | Driver license, vehicle specs, PINFL | Dynamic PG16 query |
| `GET` | `/v1/driver/earnings` | `handleDriverEarnings` | Driver daily and monthly tiyin earnings | Dynamic PG16 query |
| `GET` | `/v1/driver/availability` | `handleGetDriverAvailability` | Driver online/on-shift availability | Dynamic PG16 query |
| `POST` | `/v1/driver/availability` | `handleUpdateDriverAvailability`| Toggle on-shift / off-shift presence | Dynamic PG16 commit |
| `GET` | `/v1/driver/active-route` | `handleGetDriverActiveRoute` | Active route stops, order itinerary, ETAs | Dynamic PG16 query |
| `POST` | `/v1/driver/location` | `handleDriverLocation` | Live driver GPS coordinates push | Ingested to Redis Geo |
| `GET` | `/v1/driver/pending-collections` | `handleDriverPendingCollections` | Pending COD cash collections | Dynamic PG16 query |
| `GET` | `/v1/driver/open-fiscal` | `handleDriverOpenFiscal` | Checks unfinalized fiscal receipts on shift | Dynamic PG16 query |
| `GET` | `/v1/driver/manifest-gate` | `handleDriverManifestGate` | Verifies vehicle sealed & DVIR before yard exit | Dynamic PG16 query |
| `GET` | `/v1/driver/manifest` | `handleDriverManifest` | Detailed manifest cargo & stop sequence | Dynamic PG16 query |
| `POST` | `/v1/driver/ops/rescue/request` | `handleDriverRescueRequest` | Breakdown SOS distress call | Dynamic PG16 commit |
| `POST` | `/v1/driver/ops/rescue/respond` | `handleDriverRescueRespond` | Peer driver accepts roadside rescue | Dynamic PG16 commit |
| `POST` | `/v1/driver/orders/{id}/shop-closed` | `handleDriverReportShopClosed` | Shuttered storefront 30-min grace countdown | Dynamic PG16 / Doorstep |
| `POST` | `/v1/driver/orders/{id}/partial-offload`| `handleDriverPartialOffload` | Damaged carton rejection & credit note | **MOCK STUB** in `fleet/repo` |
| `POST` | `/v1/driver/orders/{id}/credit-leave` | `handleDriverCreditLeave` | Leaves delivery on trade credit (AR invoice) | **MOCK STUB** in `fleet/repo` |
| `GET` | `/v1/fleet/route/{id}/geometry` | `handleFleetRouteGeometry` | Turn-by-turn navigation steps & polyline | **MOCK STUB** in `fleet/repo` |
| `POST` | `/v1/fleet/route/reorder` | `handleFleetRouteReorder` | Dynamic in-flight stop sequence reordering | Dynamic PG16 commit |
| `POST` | `/v1/fleet/driver/depart` | `handleDriverDepart` | Records gate departure odometer & seal serial | Dynamic PG16 commit |
| `GET` | `/v1/fleet/driver/cash-bag/summary` | `handleDriverCashBagSummary` | End-of-shift COD cash turn-in summary | In-memory map fallback |
| `POST` | `/v1/fleet/driver/cash-bag/turn-in` | `handleDriverCashBagTurnIn` | Declares physical cash collected in drawer | Dynamic PG16 / In-memory |
| `POST` | `/v1/fleet/driver/return-complete` | `handleDriverReturnComplete` | Closes shift, returns vehicle to yard standby | Dynamic PG16 commit |
| `POST` | `/v1/delivery/arrive` | `handleDeliveryArrive` | Doorstep 100m geofence arrival event | **MOCK STUB** in `fleet/repo` |
| `POST` | `/v1/order/validate-qr` | `handleOrderValidateQR` | Retailer dynamic OTP/QR token verification | **MOCK STUB** in `fleet/repo` |
| `POST` | `/v1/delivery/scan-qr` | `handleScanDeliveryQR` | Doorstep presence verified, moves to payment | **MOCK STUB** in `fleet/repo` |
| `POST` | `/v1/order/deliver` | `handleOrderDeliver` | Marks delivery completed | **MOCK STUB** in `fleet/repo` |
| `POST` | `/v1/warehouse/manifests/{id}/stops/{orderID}/confirm-epod` | `handleConfirmEPOD` | Electronic Proof of Delivery submission | Real PG16 (`SubmitEPODTx`) |
| `POST` | `/v1/delivery/split-payment` | `handleDriverSplitPayment` | Multi-tender (Cash + Card) doorstep settlement | **MOCK STUB** in `fleet/repo` |
| `POST` | `/v1/delivery/proximity-unlock` | `handleDriverProximityUnlock` | Urban canyon GPS drift photo bypass | `doorstep.UnlockUrbanCanyonDrift` |
| `POST` | `/v1/delivery/exception-report` | `handleDeliveryExceptionReport` | OS&D carton damage report with photo URL | **MOCK STUB** in `fleet/repo` |
| `POST` | `/v1/sync/batch` | `handleSyncBatch` | Subterranean offline action queue flush | **MOCK STUB** in `fleet/repo` |
| `POST` | `/v1/epod/offline-sync` | `handleOfflineEPODSync` | Idempotent HMAC batch sync of offline ePODs | Real PG16 (`ProcessOfflineDeliveryTx`)|
| `GET` | `/v1/media/upload-ticket` | `handleGetMediaUploadTicket` | Presigned URL for exception photo upload | In-memory ID generator |

### 6.2 Role 6: Retailer Endpoints

| Method | Route | Handler Function | Current Functionality | Compliance Status |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/v1/retailer/me` | `handleGetRetailerMe` | Retailer identity, legal name, INN/STIR | Compliant (Wholesale) |
| `GET` | `/v1/retailer/profile` | `handleGetRetailerProfile` | Retailer profile, address, coordinates | Compliant (Wholesale) |
| `PUT` | `/v1/retailer/profile` | `handleUpdateRetailerProfile`| Update delivery storefront address & GPS | Compliant (Wholesale) |
| `GET` | `/v1/retailer/cart` | `handleGetRetailerCart` | Current wholesale cart items & MOQ tiers | Compliant (Wholesale) |
| `POST` | `/v1/retailer/cart/sync` | `handleSyncRetailerCart` | Syncs wholesale procurement cart | Compliant (Wholesale) |
| `DELETE`| `/v1/retailer/cart` | `handleClearRetailerCart` | Clears procurement cart | Compliant (Wholesale) |
| `POST` | `/v1/retailer/checkout/quote` | `handleCheckoutQuote` | Calculates order total, VAT, & MOQ discounts | Compliant (Wholesale) |
| `GET` | `/v1/retailer/orders/active-fulfillment`| `handleListActiveFulfillment`| Active inbound orders from warehouse | Compliant (Wholesale) |
| `GET` | `/v1/retailer/orders/{id}/claim-eligibility`| `handleGetOrderClaimEligibility`| Checks 48h concealed damage window | Compliant (Wholesale) |
| `GET` | `/v1/retailer/suppliers` | `handleListRetailerSuppliers`| Authorized FMCG brand distributors | Compliant (Wholesale) |
| `GET` | `/v1/retailer/registers` | `handleListRetailerRegisters`| List in-store POS cash registers | **ARCHITECTURAL VIOLATION** |
| `POST` | `/v1/retailer/registers` | `handleCreateRetailerRegister`| Create in-store POS till | **ARCHITECTURAL VIOLATION** |
| `GET` | `/v1/retailer/shifts/current`| `handleGetCurrentShift` | Get active cashier shift on register | **ARCHITECTURAL VIOLATION** |
| `POST` | `/v1/retailer/shifts/open` | `handleOpenShift` | Open cashier till shift with float | **ARCHITECTURAL VIOLATION** |
| `POST` | `/v1/retailer/shifts/close` | `handleCloseShift` | Close cashier shift & calculate variance | **ARCHITECTURAL VIOLATION** |
| `POST` | `/v1/retailer/shifts/cash-drop` | `handleRecordCashDrop` | Record mid-day cashier cash drop | **ARCHITECTURAL VIOLATION** |
| `POST` | `/v1/retailer/pos/sales` | `handleProcessPOSSale` | Retail checkout sale & receipt printing | **ARCHITECTURAL VIOLATION** |
| `POST` | `/v1/retailer/pos/sales/{id}/void`| `handleVoidPOSSale` | Void retail checkout transaction | **ARCHITECTURAL VIOLATION** |
| `GET` | `/v1/retailer/pos/holds` | `handleListPOSHolds` | Parked customer retail carts | **ARCHITECTURAL VIOLATION** |
| `POST` | `/v1/retailer/pos/holds` | `handleCreatePOSHold` | Hold parked retail cart | **ARCHITECTURAL VIOLATION** |
| `GET` | `/v1/retailer/stock` | `handleListStoreStock` | Retail store shelf on-hand inventory | **ARCHITECTURAL VIOLATION** |
| `POST` | `/v1/retailer/stock/counts`| `handleCreateStockCount` | In-store physical cycle counting | **ARCHITECTURAL VIOLATION** |
| `GET` | `/v1/retailer/sections` | `handleListStoreSections` | Retail store aisles and shelf planograms | **ARCHITECTURAL VIOLATION** |
| `POST` | `/v1/retailer/shelves/check-alerts`| `handleCheckShelfAlerts` | Out-of-stock shelf sensor alerts | **ARCHITECTURAL VIOLATION** |

### 6.3 Role 7: Finance & Auditor Endpoints

| Method | Route | Handler Function | Purpose | Database Persistence Status |
| :--- | :--- | :--- | :--- | :--- |
| `POST` | `/v1/soliq/facturas/generate` | `handleGenerateFactura` | Creates & E-IMZO signs Soliq E-Factura | Dynamic PG16 (`signed_facturas`) |
| `POST` | `/v1/soliq/facturas/corrective` | `handleGenerateCorrectiveFactura` | Tax Code Art. 257 Tuzatuvchi Faktura | Dynamic PG16 (`signed_facturas`) |
| `GET` | `/v1/soliq/facturas/{orderID}` | `handleGetFacturaByOrder` | Fetches signed fiscal invoice by order ID | Dynamic PG16 query |
| `GET` | `/v1/soliq/facturas/by-id/{id}`| `handleGetFacturaByID` | Fetches signed fiscal invoice by factura ID | Dynamic PG16 query |
| `POST` | `/v1/cash/payment-legs` | `handleRecordPaymentLeg` | Records cash/card delivery payment leg | Dynamic PG16 (`order_payment_legs`) |
| `GET` | `/v1/cash/drivers/{id}/shift-summary`| `handleGetDriverShiftSummary`| Sums expected COD cash collected on shift | Dynamic PG16 query |
| `POST` | `/v1/cash/reconcile` | `handleReconcileShift` | Balances driver cash against ledger | Dynamic PG16 (`cash_reconciliations`)|
| `GET` | `/v1/cash/drivers/{id}/eligibility` | `handleCheckDriverShiftEligibility`| Checks driver for cash discrepancies | Dynamic PG16 query |
| `POST` | `/v1/cash/deposits` | `handleRecordCashDeposit` | Records cash turn-in at smart safe | Dynamic PG16 (`driver_cash_deposits`)|
| `GET` | `/v1/cash/deposits` | `handleListCashDeposits` | Audits driver cash deposits | Dynamic PG16 query |
| `POST` | `/v1/cash/deposits/{id}/override` | `handleSupervisorOverride` | Supervisor override of counterfeit/loss | Dynamic PG16 commit |
| `POST` | `/v1/payments/handover` | `handleProcessHandover` | Full storefront handover & GL postings | Dynamic PG16 commit |
| `POST` | `/v1/payments/qr-token` | `handleCreateGlobalPayQRToken`| Dynamic corporate card checkout token | Dynamic PG16 / Memory |
| `GET` | `/v1/payments/credit-account` | `handleGetRetailerCreditAccount`| Retailer credit limit, debt, & block flag | Dynamic PG16 query |
| `POST` | `/v1/ar/dunning/evaluate` | `handleEvaluateDunning` | Evaluates overdue debts against aging ladder | Dynamic PG16 commit |
| `GET` | `/v1/ar/aging-ledger` | `handleGetAgingLedger` | AR aging buckets (Current, 1-15, 16-30, 31+) | Dynamic PG16 query |
| `POST` | `/v1/adm/drops/reconcile` | `handleReconcileADMCashDrop` | Hardware ADM smart safe banknote audit | Dynamic PG16 / ADM |

---

## 7. Concrete Actionable Recommendations for Implementation Workers

1. **Unify Role 5 Delivery Pipeline into `internal/epod`**:
   - Eliminate stubbed methods in `internal/fleet/repository.go` (`DeliveryArrive`, `ScanDeliveryQR`, `OrderDeliver`, `PartialOffload`, `ValidateQR`, `SplitPayment`).
   - Route all driver delivery endpoints directly to `internal/epod/service.go` and `internal/payment/handover.go` with atomic PostgreSQL transactions (`SubmitEPODTx`, `MarkStopArrivedTx`, `ProcessOfflineDeliveryTx`, and `ProcessStorefrontHandover`).
2. **Implement Dynamic Retailer OTP / Rotating QR Handshake**:
   - Create `internal/retailer/handshake.go` with dynamic 6-digit TOTP (30s window) or Redis-backed ephemeral nonce (`SETEX handshake:order:{id} 60 token`).
   - Expose `GET /v1/retailer/orders/{id}/handshake-token` for Retailer apps.
   - Update `POST /v1/order/validate-qr` to validate this rotating token against Redis.
3. **Purge Retailer POS & Cashier Shifts (Restore Pure Procurement Scope)**:
   - Quarantine or remove in-store POS handlers from `/v1/retailer/registers`, `/v1/retailer/shifts/*`, `/v1/retailer/pos/*`, and `/v1/retailer/stock/*`.
   - Remove `app/(portal)/pos/page.tsx` from `retailer-desktop`.
   - Wire `TrackingMap.tsx` with MapLibre + Carto directly into `retailer-desktop/app/(portal)/tracking/page.tsx`.
   - Remove `PosTerminalScreen.kt` from `retailer-app-android` and replace mock data in `DeliveryTrackingScreen.kt` with live API calls.
4. **Implement Driver CIT Limit & Mid-Shift Vault Drop**:
   - In `internal/cashrecon`, add `CITLimitMinor` (default 100,000,000 UZS in tiyins = 10,000,000,000 tiyins).
   - In `handlers_fleet_driver.go`, add real-time drawer check on payment collection. If cash exceeds threshold, emit alert and provide route waypoint to nearest depot smart safe.
   - Implement `POST /v1/fleet/driver/mid-shift-vault-drop` allowing cash deposits that decrement the driver's active cash drawer without closing their delivery shift.
5. **Persist Double-Entry Ledger Postings**:
   - Ensure `ProcessStorefrontHandover` in `internal/payment/handover.go` and `CreateGlobalPayCaptureJournalEntry` atomically insert into `ledger_journal_entries` and `ledger_postings` tables during the payment transaction.
