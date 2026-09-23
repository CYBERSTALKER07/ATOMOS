# Handoff Report — Explorer Survey 3: Roles 5–7 Backend Packages, Retailer Scope, & Test Baseline

**Agent**: Explorer Survey 3  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/explorer_survey_3`  
**Target Codebase**: `pegasus.x` (PostgreSQL 16 + Redis 7 Streams)  
**Date**: 2026-09-23  
**Handoff Type**: Hard (Investigation & Survey Complete)  

---

## 1. Observation

### 1.1 Role 6 (Retailer) Scope Violation
- **Doctrine & Specification Mandate**:
  - `AGENTS.md` and `prompt_draft.md` Section 3.6 dictate:  
    > *"Retailer clients (Desktop Tauri, Android Compose, iOS SwiftUI, Telegram MiniApp) are strictly procurement and wholesale ordering surfaces. Zero in-store POS, shelf cycle-counting, cashier shifts, or store product management."*
- **Observed Codebase Reality**:
  - `pegasus.x/database/migrations/055_retailer_pos_shifts_and_auto_order.sql:8-143` creates tables: `retailer_registers` (hardware tills), `retailer_shifts` (`opening_float_tiyins`, `closing_cash_tiyins`), `retailer_cash_drops`, `retailer_pos_sales` (`receipt_number`, `lines JSONB`, `tenders JSONB`), `retailer_pos_holds` (parked carts), `retailer_store_stock`, `retailer_store_sections`, and `retailer_shelf_alerts`.
  - `pegasus.x/backend/internal/api/router.go:1226-1304` mounts 35 REST routes dedicated to retail POS: `/v1/retailer/registers`, `/v1/retailer/shifts/*`, `/v1/retailer/pos/*`, `/v1/retailer/stock/*`, and `/v1/retailer/shelves/*`.
  - `pegasus.x/backend/internal/api/handlers_retailer.go:1230-1450` contains full cashier shift and retail sale handlers.
  - `pegasus.x/backend/internal/api/retailer_pos_auto_order_e2e_test.go:1-399` contains 399 lines testing grocery checkout, cashier float, cash drops, parked carts, and till reconciliation.
  - `pegasus.x/apps/retailer-desktop/app/(portal)/pos/page.tsx:1-1087` (1,087 lines of cash register UI).
  - `pegasus.x/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/PosTerminalScreen.kt` (46,494 bytes) and `StockScreen.kt` (20,520 bytes).
  - `pegasus.x/apps/retailer-app-ios/Sources/RetailerApp/Views/PosTerminalView.swift` and `StockView.swift`.

### 1.2 Role 5 (Driver) Pipeline Split & In-Memory Mock Stubs
- **Two Parallel Pipelines**:
  - Pipeline A (`internal/epod/`): True PostgreSQL transactional persistence (`SubmitEPODTx`, `MarkStopArrivedTx`, `ProcessOfflineDeliveryTx`) against `manifest_stops` and `epod_records`.
  - Pipeline B (`internal/fleet/repository.go` & `handlers_fleet_driver.go`): Exposes `/v1/delivery/arrive`, `/v1/order/deliver`, `/v1/delivery/scan-qr`, `/v1/delivery/partial-offload`, and `/v1/order/validate-qr`.
- **Verbatim Mock Stubs in Production Code**:
  - `internal/fleet/repository.go:2258-2273` (`ValidateQR`):
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
  - `internal/fleet/repository.go:2299-2302` (`PartialOffload`):
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
  - `internal/fleet/repository.go:2484` (`RetryFiscal`): returns `"https://ofd.soliq.uz/receipt/" + req.OrderID`.
  - `internal/api/handlers_fleet_driver.go:1234-1258`: `handleGetOrderItems` and `handleGetActiveFleetMissions` return hardcoded JSON payloads with fake mock products ("Coca-Cola 1.5L", "Nestle Sut 1L").
  - Native camera lockout: `handlers_fleet_driver.go:1178` (`handleGetMediaUploadTicket`) returns an upload URL without checking camera hardware capture metadata or preventing gallery image uploads.

### 1.3 Role 7 (Finance & Auditor) Status
- `internal/fiscal/calculator.go:19, 74-120`: Statutory 12% Soliq VAT (`DefaultVatRateBps = 1200`) strictly implemented with 64-bit integer half-up rounding `(netMinor * vatRateBps + 5000) / 10000`. Passing unit tests in `calculator_test.go`.
- `internal/soliq/service.go:65-220`: Soliq OFD E-Factura PKCS#7 envelope signing and persistence to `signed_facturas` table. Passing tests in `soliq_e2e_test.go`.
- `internal/payment/handover.go:228-241`: Double-entry general ledger balance invariant ($\sum \text{Debits} == \sum \text{Credits}$) strictly enforced (`ErrUnbalancedJournalEntry`). Passing tests in `handover_test.go`.
- **Gaps**:
  - Driver Cash-In-Transit (CIT) drawer threshold alerts (> 100M UZS) are completely absent.
  - Mid-shift depot smart safe vault drop workflow is absent (only end-of-shift cash turn-in exists).
  - Commercial bank deposit reconciliation (`internal/cashrecon/service.go:393-450`) reconciles driver turn-in against expected cash from `manifest_stops`, but lacks reconciliation with commercial bank deposit slips or bank statements.

### 1.4 Test Baseline & Statement Coverage Execution
- Executed `go test -cover ./...` in `pegasus.x/backend`:
  - **100% PASS**: 85 packages (83 internal, 2 cmd), 112 test files, 0 failures.
  - High coverage packages: `internal/marketpack` (94.9%), `internal/fiscal` (82.1%), `internal/adm` (86.0%), `internal/soliq` (64.1%), `internal/payment` (63.2%).
  - Medium/Low coverage: `internal/api` (47.6%), `internal/fleet` (37.8%), `internal/cashrecon` (34.5%), `internal/retailer` (22.7%).
  - 0% coverage: `internal/creditnote`, `internal/db`, `internal/inventory`, `internal/models`, `internal/outbox`, `internal/redis`, `internal/ump`.
  - Several tests pass only because they assert against in-memory repository fallbacks and mock responses.

---

## 2. Logic Chain

1. **Premise 1 (Wholesale vs POS Scope)**: Observation 1.1 reveals extensive grocery POS and cashier shift code in Migration 055, `internal/retailer`, `handlers_retailer.go`, and client applications.  
   *Reasoning*: The canonical architecture doctrine (AGENTS.md / GEMINI.md / prompt_draft.md) specifies Retailer as a **B2B wholesale procurement terminal ONLY**. Building and maintaining store POS systems distracts from the core B2B ordering and delivery lifecycle.  
   *Deduction*: Migration 055 and all retail POS routes/screens must be quarantined or decommissioned, and client surfaces redirected to wholesale ordering, inbound delivery tracking, proximity handshake pop-ups, and fiscal receipt downloads.

2. **Premise 2 (Delivery Pipeline Duality & Zero Mock Policy)**: Observation 1.2 demonstrates that `internal/epod` has genuine PostgreSQL transactions, while `internal/fleet` exposes duplicate delivery endpoints populated with hardcoded strings (`"Korzinka Chilonzor Branch"`, `540000000` tiyins, fake credit note IDs).  
   *Reasoning*: AGENTS.md Section 3 explicitly bans mock data and in-memory repository fallbacks in production packages.  
   *Deduction*: The delivery endpoints in `internal/fleet` must be merged into `internal/epod` and `internal/payment/handover`, binding directly to `manifest_stops` and real PostgreSQL updates.

3. **Premise 3 (Cash-In-Transit & Security Controls)**: Observation 1.3 reveals that drivers can collect unbounded cash without real-time CIT threshold monitoring, mid-shift vault drops, or camera gallery lockouts.  
   *Reasoning*: In high-volume B2B wholesale distribution in Uzbekistan, cash collections can easily exceed 100,000,000 UZS on a single route, creating robbery and loss exposure. Furthermore, accepting gallery uploads allows fraudulent damage claims.  
   *Deduction*: Implement real-time CIT threshold checks with depot vault drop waypoints, and require native camera capture source verification (`CAPTURE_SOURCE=CAMERA_DIRECT`).

4. **Premise 4 (Test Reliability vs Mock Masks)**: Observation 1.4 confirms that `go test ./...` passes 100%, but several E2E tests assert against mock stubs in `fleet/repository.go` rather than PostgreSQL.  
   *Reasoning*: Green tests asserting against stubs mask underlying persistence defects.  
   *Deduction*: As delivery endpoints are wired to real PostgreSQL transactions, integration tests must be updated to verify database rows and outbox events.

---

## 3. Caveats

1. **Read-Only Scope**: This investigation was strictly analytical; no source code or database migrations were modified.
2. **Client Surface Audits**: Frontend and mobile applications (`retailer-desktop`, `retailer-app-android`, `retailer-app-ios`) were inspected via static code analysis. UI components were not run inside live Android/iOS emulators or browser runtimes during this turn.
3. **Database Migration State**: Analysis of migration SQL files reflects the repository files on disk. The exact state of a deployed database depends on which migration scripts have been executed.
4. **Third-Party APIs**: External Soliq OFD and GlobalPay APIs are integrated via interface contracts; live external sandboxes were not invoked during static analysis.

---

## 4. Conclusion

The backend architecture of `pegasus.x` possesses robust mathematical and legal foundations (12% Soliq VAT integer arithmetic, E-IMZO PKCS#7 signing, double-entry ledger invariant, and transactional ePOD in `internal/epod`).

However, four major defects must be addressed during the upcoming implementation phase:
1. **Purge/Quarantine In-Store POS from Retailer Scope**: Strip out Migration 055, 35 REST routes in `router.go`, and cashier UI screens from desktop and mobile apps to re-align Retailer strictly as a B2B wholesale procurement terminal.
2. **Eliminate Fleet Delivery Mock Stubs**: Route all driver delivery actions (`/v1/delivery/arrive`, `/v1/order/validate-qr`, `/v1/delivery/partial-offload`, `/v1/delivery/split-payment`, `/v1/order/deliver`) through `internal/epod` and `internal/payment/handover` with verified PostgreSQL transactions.
3. **Dynamic Proximity Handshake**: Implement 6-digit TOTP / rotating QR token generation on Retailer endpoints (`/v1/retailer/orders/{id}/handshake-token`) with cryptographic validation on driver scan.
4. **Financial Security Hardening**: Add Driver Cash-In-Transit threshold alerts (> 100M UZS), mid-shift depot smart safe vault drops, and commercial bank deposit reconciliation.

---

## 5. Verification Method

To independently verify all observations and conclusions in this report, execute the following commands in the workspace:

### 1. Verify 100% Test Passing Baseline
```bash
cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
go test -v -race ./internal/fiscal/... ./internal/soliq/... ./internal/payment/...
```
*Expected*: All tests pass with zero race conditions.

### 2. Verify Retailer POS Scope Violation
```bash
# Check Migration 055 DDL
grep -E "retailer_registers|retailer_shifts|retailer_pos_sales" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/database/migrations/055_retailer_pos_shifts_and_auto_order.sql

# Check Desktop POS page
head -n 25 /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/retailer-desktop/app/\(portal\)/pos/page.tsx

# Check Android POS Screen
head -n 25 /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/PosTerminalScreen.kt
```
*Expected*: Confirms full in-store cash register / POS implementation across schema, backend, and clients.

### 3. Verify Mock Stubs in Fleet Repository
```bash
# Check hardcoded store name in QR validation
grep -n -C 5 "Korzinka Chilonzor Branch" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/fleet/repository.go

# Check hardcoded partial offload tiyins & fake credit note ID
grep -n -C 5 "cn_part_" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/fleet/repository.go
```
*Expected*: Lines 2268 and 2301 show hardcoded mock values.

### 4. Verify Double-Entry Balance & Soliq VAT
```bash
# Check double-entry ledger balance check
grep -n -C 5 "ErrUnbalancedJournalEntry" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/payment/handover.go

# Check 12% VAT half-up calculation
grep -n -C 5 "DefaultVatRateBps" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/fiscal/calculator.go
```
*Expected*: Shows strict integer math and balance validation.

### 5. Invalidation Conditions
- If the project sponsor explicitly specifies that `pegasus.x` Retailer client is intended to function as an all-in-one store management ERP (including grocery cash register tills), the Retailer scope conclusion is invalidated and the doctrine in `AGENTS.md` must be updated.
- If mock stubs in `fleet/repository.go` are replaced with calls to `internal/epod` and tests are updated, the "Dual Pipeline" finding is resolved.
