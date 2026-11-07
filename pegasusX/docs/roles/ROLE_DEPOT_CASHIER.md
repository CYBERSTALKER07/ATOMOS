# Role 5 Specification: Warehouse Cashier & Depot Vault Operations

> **Role Number**: Role 5 of 9  
> **Persona ID**: `ROLE_DEPOT_CASHIER` / `ROLE_WAREHOUSE_CASHIER`  
> **Domain Scope**: Driver Shift Return, Tamper-Evident Bag Ingestion, Zero-Trust Dual-Blind Denomination Counting Ritual, Variance Ratchet, RRN Payment Tender Reclassification, Perpetual Depot Vault Ledger, Armored CIT Handover & Multi-Tender Settlement Tracking  
> **Authoritative Backend Seams**:  
> - Foundry Cashier & Vault Core: [`apps/backend-go/foundry/cashier_vault.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundry/cashier_vault.go)  
> - Foundry Cashier Routes: [`apps/backend-go/foundryroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundryroutes)  
> - Cash Reconciliation Domain: [`apps/backend-go/cashrecon`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/cashrecon)  
> - Cash Recon HTTP Handlers: [`apps/backend-go/cashreconroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/cashreconroutes)  
> **Frontend & Native Surfaces**:  
> - Web Cashier Station: [`apps/warehouse-portal/app/cashier`](file:///Users/shakhzod/Desktop/pegasusx/apps/warehouse-portal/app/cashier)  
> - Dual-Blind Counting UI: [`apps/warehouse-portal/components/cashier`](file:///Users/shakhzod/Desktop/pegasusx/apps/warehouse-portal/components/cashier)  
> - iOS Warehouse Cashier: [`apps/warehouse-app-ios/WarehouseApp/Views/Cashier`](file:///Users/shakhzod/Desktop/pegasusx/apps/warehouse-app-ios/WarehouseApp/Views/Cashier)  
> - Android Warehouse Cashier: [`apps/warehouse-app-android/app/src/main/java/com/pegasusx/warehouse/ui/screens/cashier`](file:///Users/shakhzod/Desktop/pegasusx/apps/warehouse-app-android/app/src/main/java/com/pegasusx/warehouse/ui/screens/cashier)  
> **Primary Cloud Spanner Tables**: `CashierBlindCountSessions`, `DepotVaultLedgerEntries`, `MerchantDailyCashExposure`, `CashReconciliations`  

---

## 1. Executive Summary & Operational Charter

The **Warehouse Cashier (Role 5)** governs the physical-to-fiscal settlement perimeter at the regional distribution center. In fast-moving consumer goods (FMCG) distribution across Central Asia, substantial delivery volumes settle via cash-on-delivery (COD) or split-tender payments at the retailer's doorstep. The Warehouse Cashier verifies physical banknote custody, eliminates route fraud, reconciles electronic proof-of-delivery (ePOD) logs, manages secure drop-safe vaults, and transfers funds to authorized Cash-in-Transit (CIT / Inkassatsiya) armored couriers.

The Warehouse Cashier operates under four unbreakable architectural principles:
1. **Zero-Trust Dual-Blind Counting**: The cashier is strictly prevented from seeing the system-expected cash total (`ExpectedCashMinor`) during counting and recounting. This guarantees that cashier inputs reflect physical reality rather than fudged numbers to match backend expectations.
2. **One-Way Blind Ratchet**: If Count #1 deviates from the system expectation beyond the allowed tolerance buffer, the session transitions to `BLIND_RECOUNT_REQUIRED` without exposing the expected amount. A second failing count immediately quarantines the session (`DISCREPANT_QUARANTINE`) and locks the driver's schedule for investigation.
3. **Compensating-Only Ledger Mutations**: Cashier operations never execute destructive `UPDATE` or `DELETE` statements on payment records. Doorstep tender mismatches (e.g., driver recorded cash instead of Humo/Uzcard) require payment gateway Retrieval Reference Number (RRN) cryptographic proof to post balanced compensating reversal legs.
4. **Uncapped Multi-Tender Settlement Policy**: Payments and trade settlements operate without artificial credit caps or statutory payment ceilings. All collections are recorded transparently in the immutable Spanner ledger for audit and fiscal reporting.

---

## 2. Account Setup, Identity, Hardware & Provisioning

```
[Cash Desk Hardware Setup] -> [Staff Biometrics / MFA] -> [Opening Float Allocation] -> [Terminal Lock Binding]
```

### 2.1. Cashier Identity & Role Provisioning
1. **Staff Onboarding**: The Warehouse Manager provisions the cashier account through [`POST /v1/staff/invite`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/staffinvite). The cashier is assigned the system role `ROLE_DEPOT_CASHIER` and scoped strictly to their assigned Distribution Center (`WarehouseID`).
2. **Hardware MFA Enrollment**: The cashier enrolls a physical FIDO2 WebAuthn security key or TOTP authenticator ([`apps/backend-go/mfa`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/mfa)). Every session opening, shift sign-off, and CIT vault transfer requires cryptographic step-up verification.
3. **Session Enclave & Station Binding**: The cashier client binds to a dedicated cash counter workstation IP address and peripheral workstation identifier (`CashDeskStationID`).

### 2.2. Hardware Peripheral & Vault Terminal Configuration
1. **Barcode Scanner Pairing**: The cashier pairs an optical 2D barcode scanner (Zebra DS2208 or Honeywell Xenon) via USB HID or Bluetooth. The scanner is calibrated to read Code 128 tamper-evident bag barcodes (`BAG-...`) and driver shift QR badges.
2. **Currency Counter / Validator Machine Integration**: An optical banknote counting machine (Magner 150 / Kisan Newton) connects via RS-232 serial/USB interface to feed automated denomination counts (200k, 100k, 50k, 20k, 10k, 5k, 2k, 1k UZS) directly into the blind count modal.
3. **Drop-Safe Electronic Interlock**: The physical high-security drop-safe vault connects to the cashier terminal. Opening the intake hopper or CIT retrieval compartment triggers an authenticated audit event in Cloud Spanner.
4. **Opening Cash Float Verification**: At shift start, the cashier verifies the opening till float (`OpeningFloatMinor`) in Cloud Spanner via [`POST /v1/cashrecon/shift/open`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/cashreconroutes). The opening float is recorded in `DepotVaultLedgerEntries` with entry type `TILL_FLOAT_OPEN`.

---

## 3. Comprehensive Role User Flow & Operational Cadence

```mermaid
sequenceDiagram
    autonumber
    actor Driver as Carrier / Fleet Driver
    actor Cashier as Warehouse Cashier
    participant Station as Cash Desk Terminal (Web/Native)
    participant Foundry as Foundry Kinetic Engine (Go)
    participant Spanner as Cloud Spanner Multi-Cell
    actor Supervisor as Finance Supervisor
    participant CIT as Armored Courier (Inkassatsiya)

    %% Phase 1: Morning Cash Desk Initialization
    Note over Cashier,Spanner: Phase 1: Morning Till Float Setup & Station Binding
    Cashier->>Station: Login (MFA Step-Up + Station HW Fingerprint)
    Station->>Foundry: POST /v1/cashrecon/shift/open (OpeningFloatMinor)
    Foundry->>Spanner: Insert DepotVaultLedgerEntries (TILL_FLOAT_OPEN)
    Spanner-->>Station: Shift Initialized (Session ID: CASH-SESS-901)

    %% Phase 2: Driver Ingestion & Bag Handoff
    Note over Driver,Cashier: Phase 2: Driver Return & Tamper-Evident Bag Ingestion
    Driver->>Cashier: Hands over sealed serialized bag + manifests
    Cashier->>Station: Scan Driver Badge & BagSealBarcode (BAG-882104)
    Station->>Foundry: POST /v1/foundry/cashier/sessions (BagSealBarcode, DriverID, RouteID)
    Foundry->>Spanner: Query Confirmed CASH ePOD Legs -> Compute ExpectedCashMinor
    Spanner-->>Foundry: ExpectedCash: Masked from Response Payload
    Foundry-->>Station: Session Created (Status: AWAITING_BLIND_COUNT_1, ExpectedTotal: MASKED)

    %% Phase 3: Zero-Trust Blind Denomination Counting Ritual
    Note over Cashier,Foundry: Phase 3: Blind Denomination Counting Ritual & Ratchet
    Cashier->>Station: Inspect Bag Seal (Intact = True); Unseal & Enter Denomination Counts
    Station->>Foundry: POST /v1/foundry/cashier/sessions/{id}/blind-count (200k, 100k, 50k...)
    Foundry->>Foundry: Calculate Total Minor Units & Evaluate Variance vs ExpectedCashMinor
    alt Variance == 0 (Exact Match)
        Foundry->>Spanner: Set Status = BALANCED; Insert DepotVaultLedgerEntries (DRIVER_DROP_INFLOW)
        Foundry-->>Station: 200 OK: Signed Shift Clearance Certificate & Zero Driver Liability
    else Variance <= Tolerance Threshold
        Foundry->>Spanner: Set Status = TOLERANCE_VARIANCE; Evaluate Micro-Skimming Frequency
        Foundry-->>Station: 200 OK: Minor Variance Logged & Conditional Clearance Issued
    else Variance > Tolerance (First Attempt)
        Foundry->>Spanner: Set Status = BLIND_RECOUNT_REQUIRED (ExpectedCash Still Hidden)
        Foundry-->>Station: 422 Unprocessable: Mandatory Recount #2 Triggered
        Cashier->>Station: Execute Secondary Blind Recount
        Station->>Foundry: POST /v1/foundry/cashier/sessions/{id}/blind-count (Attempt #2)
        alt Recount Passes
            Foundry->>Spanner: Set Status = BALANCED; Insert DepotVaultLedgerEntries
            Foundry-->>Station: 200 OK: Resolved on Recount
        else Recount Fails Again
            Foundry->>Spanner: Set Status = DISCREPANT_QUARANTINE; Lock Driver Schedule
            Foundry-->>Supervisor: Immediate Push Alert: Route Audit & Camera Investigation Required
        end
    end

    %% Phase 4: Doorstep Tender Misclassification Resolution
    Note over Cashier,Spanner: Phase 4: RRN Payment Tender Reclassification (Zero-Destructive)
    Driver->>Cashier: Reports "Stop #4 Paid via Uzcard POS, Mistakenly Tagged as Cash"
    Cashier->>Station: Enter Gateway Retrieval Reference Number (RRN: 994821048821)
    Station->>Foundry: POST /v1/foundry/cashier/reclassify-tender (OrderID, RRN, AmountMinor)
    Foundry->>Foundry: Verify RRN Gateway Log + Geofence Timestamp Window (±15 min)
    Foundry->>Spanner: Insert CASH_REVERSAL + CARD_VERIFIED_RRN Compensating Legs
    Foundry->>Spanner: Recalculate Cashier Session Variance -> Rebalance to Zero
    Foundry-->>Station: Reclassification Confirmed; Session Status Updated to BALANCED

    %% Phase 5: Daily Vault Consolidation & CIT Armored Handover
    Note over Cashier,CIT: Phase 5: Vault Consolidation & CIT Armored Courier Handover
    CIT->>Cashier: Armored Guards Arrive (Badge Scan & OTP Handshake)
    Cashier->>Station: Select Balanced Sessions for Banking Transfer
    Station->>Foundry: POST /v1/foundry/cashier/vault/cit-handover (SessionIDs[], CourierBadgeID, CITSealBarcode)
    Foundry->>Spanner: Insert DepotVaultLedgerEntries (CIT_ARMORED_OUTFLOW) & Decrement Vault Balance
    Foundry-->>Station: Print Standardized Inkassatsiya Manifest & 3-Part Deposit Slip
    Cashier->>CIT: Handover Safe Bags + Countersigned Manifest
    Station->>Spanner: Seal Daily Cashier Shift Audit Trail
```

---

## 4. Exhaustive Feature Catalog

### 4.1. Serialized Tamper-Evident Bag Intake Scanner
- **Feature Name**: `BagIntakeScanner`
- **Purpose**: Prevents theft, bag swapping, and double-turn-in by binding physical high-security tamper-evident bags (`BAG-...`) to driver route shifts before the seal is opened.
- **UI Mechanics**: Dedicated camera/scanner input field with barcode format validation, instant audio feedback, and visual status badges (`SEAL_INTACT`, `SEAL_BROKEN`, `BAG_ALREADY_INGESTED`).
- **Backend API Seam**: [`POST /v1/foundry/cashier/sessions`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundryroutes)
- **Pre-Execution Guards**:
  - Rejects duplicate bag barcodes with `409 Conflict`.
  - Verifies that the route ID is in `COMPLETED_ROUTE` or `RETURNING_TO_DEPOT` status.
  - Automatically queries all confirmed cash legs from `OrderPaymentLegs` to compute `ExpectedCashMinor`, storing it in Spanner while stripping it from the HTTP response payload.

### 4.2. Zero-Trust Dual-Blind Denomination Counting Terminal
- **Feature Name**: `DualBlindCountingTerminal`
- **Purpose**: Eliminates confirmation bias and cashier collusion by forcing the physical counting and entry of all 8 national currency banknote denominations without knowing the target amount.
- **UI Mechanics**: Large-format touch keypad displaying all official Uzbekistan Som banknote denominations (`200,000`, `100,000`, `50,000`, `20,000`, `10,000`, `5,000`, `2,000`, `1,000` UZS). Displays running bill counts and total physical Som/Tiyin minor units. The "Expected Total" column displays a locked padlock icon and the label `HIDDEN UNTIL RATIFIED`.
- **Backend API Seam**: [`POST /v1/foundry/cashier/sessions/{id}/blind-count`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundryroutes)
- **Deterministic Tiyin Math**:
  $$\text{CountedMinor} = \sum_{d \in \text{Denominations}} (\text{Quantity}_d \times \text{FaceValue}_d \times 100)$$
- **Ratchet Logic**:
  - `Variance == 0`: Session advances to `BALANCED`. Driver shift liability cleared immediately.
  - `0 < |Variance| <= Tolerance`: Session tagged `TOLERANCE_VARIANCE`. Driver cleared conditionally; variance added to driver historical ledger.
  - `|Variance| > Tolerance` (Count 1): Session locked into `BLIND_RECOUNT_REQUIRED`. Target amount remains masked. Cashier must perform a fresh count.
  - `|Variance| > Tolerance` (Count 2): Session transitions to `DISCREPANT_QUARANTINE`. Driver profile flagged; supervisor notification emitted.

### 4.3. RRN Tender Reclassification & Discrepancy Resolution
- **Feature Name**: `RRNTenderReclassification`
- **Purpose**: Resolves common driver errors where customer doorstep card payments (via Humo or Uzcard mPOS terminals) were mistakenly logged as cash on the mobile ePOD tablet, without permitting arbitrary manual ledger adjustments.
- **UI Mechanics**: Modal window requiring the cashier to enter the customer Order ID, the payment rail (`UZCARD` or `HUMO`), the transaction amount, and the 12-digit payment gateway **Retrieval Reference Number (RRN)** printed on the bank POS receipt slip.
- **Backend API Seam**: [`POST /v1/foundry/cashier/reclassify-tender`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundryroutes)
- **Audit Rules**:
  - Checks that the RRN exists in payment gateway capture logs with status `SUCCESS`.
  - Verifies that the gateway capture timestamp is within $\pm 15\text{ minutes}$ of the driver's doorstep geofence unlock event.
  - Atomically appends compensating entries: `CASH_REVERSAL` (negative cash leg) and `CARD_VERIFIED_RRN` (positive card leg).
  - Recalculates the active cashier blind count session; if the variance is now resolved, marks the session `BALANCED`.

### 4.4. Micro-Skimming & Driver Velocity Behavioral Monitor
- **Feature Name**: `MicroSkimmingDetector`
- **Purpose**: Detects pattern-based theft where drivers intentionally under-report small amounts of cash within the acceptable tolerance buffer (e.g., keeping 5,000 UZS per route) across multiple shifts.
- **UI Mechanics**: Visual driver reliability card featuring a 30-day variance sparkline, cumulative tolerance write-off totals, and automated risk scoring (`LOW`, `MEDIUM`, `HIGH_RISK`).
- **Backend Implementation**: [`apps/backend-go/foundry/cashier_vault.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundry/cashier_vault.go#L110-L135).
- **Rule**: If a driver has negative variances on $\ge 80\%$ of their last 5 completed shifts, the engine flags the driver with `MARKING:MICRO_SKIMMING_SUSPECT` and reduces the automated tolerance buffer for that driver to zero.

### 4.5. Perpetual Depot Vault Ledger & Heavy Safe Monitor
- **Feature Name**: `DepotVaultLedger`
- **Purpose**: Maintains a continuous, immutable double-entry accounting ledger of all physical fiat currency sitting inside the distribution center's heavy drop-safe vault.
- **UI Mechanics**: Real-time vault balance KPI card, running entry grid with color-coded transaction types (`TILL_FLOAT_OPEN`, `DRIVER_DROP_INFLOW`, `DISPUTE_ADJUSTMENT`, `CIT_ARMORED_OUTFLOW`), and visual capacity gauge.
- **Backend API Seams**:
  - `GET /v1/foundry/cashier/vault/balance`
  - `GET /v1/foundry/cashier/vault/ledger`
- **Spanner Data Contract**: Every ledger mutation requires a unique `EntryID`, monotonic `SequenceNumber`, entry type, signed amount in Tiyin, resulting vault balance, and cashier identity.

### 4.6. Armored Courier (CIT / Inkassatsiya) Handover Console
- **Feature Name**: `CITHandoverConsole`
- **Purpose**: Governs the official custody transfer of accumulated vault cash from the warehouse distribution center to bank-contracted armored car services (Inkassatsiya).
- **UI Mechanics**: Multi-select interface displaying all verified `BALANCED` driver sessions. Computes gross pickup amount, verifies armored courier ID badge, scans heavy tamper-evident courier canvas bag seals, and generates official 3-part bank deposit manifests.
- **Backend API Seam**: [`POST /v1/foundry/cashier/vault/cit-handover`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundryroutes)
- **Pre-Execution Invariants**:
  - All included cashier sessions must be in `BALANCED` status (quarantined sessions cannot be handed over).
  - Generates an immutable `CIT_ARMORED_OUTFLOW` ledger entry decrementing the depot vault balance.
  - Generates a cryptographically signed handover certificate stamped with courier badge ID, timestamp, and bag seal barcode.

### 4.7. Multi-Tender Exposure & Ledger Audit Surface
- **Feature Name**: `MultiTenderExposureAuditor`
- **Purpose**: Provides real-time visibility into cumulative daily cash, card, and credit settlements across legal entities (`TIN/PINFL`) and suppliers for enterprise audit and fiscal compliance.
- **UI Mechanics**: Interactive data table displaying Legal Entity TIN, business trade name, cumulative cash collected today, and breakdown by tender type.
- **Backend API Seam**: [`POST /v1/foundry/guards/aml-check`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundryroutes)
- **Policy Contract**: Under the uncapped multi-tender policy (`UNLIMITED_PAYMENTS=1` / `AML_LIMITS_DISABLED=1`), the system records exposure transparently without blocking legitimate high-volume commercial transactions or cash-on-delivery orders.

---

## 5. Security Markings, Error Codes & Invariants

### 5.1. Standard Cashier Error Matrix
| Error Code | HTTP Status | Root Cause | Deterministic Remediation |
| :--- | :---: | :--- | :--- |
| `ERR_BAG_SEAL_ALREADY_LOGGED` | `409 Conflict` | Bag barcode has already been scanned in another session. | Inspect physical bag; confirm barcode with driver and supervisor. |
| `ERR_BLIND_RECOUNT_REQUIRED` | `422 Unprocessable` | Count #1 variance exceeded the tolerance threshold. | Re-tally physical currency and submit Count #2. Target amount remains hidden. |
| `ERR_SESSION_DISCREPANT_QUARANTINE` | `422 Unprocessable` | Count #2 failed to balance. Driver shift quarantined. | Route escalated to Finance Supervisor for camera and invoice audit. |
| `ERR_INVALID_RRN_PROOF` | `400 Bad Request` | Provided RRN does not exist in bank gateway or outside time window. | Verify physical customer receipt slip and terminal transaction history. |
| `ERR_UNBALANCED_CIT_HANDOVER` | `400 Bad Request` | Attempted to include an unbalanced or quarantined session in CIT transfer. | Deselect discrepant session; include only certified `BALANCED` drops. |

### 5.2. Role Invariants Checklist
- [x] **Expected Amount Blindness**: The backend never returns `ExpectedCashMinor` or `VarianceMinor` while a count is pending.
- [x] **Integer Minor Units**: All monetary calculations use `int64` Tiyin (1 UZS = 100 Tiyin). No floats.
- [x] **Compensating-Only Corrections**: Zero direct mutations on `OrderPaymentLegs`. Reclassifications require valid RRN proof.
- [x] **Uncapped Multi-Tender Policy**: No payment ceilings or artificial credit limits block order settlement.
