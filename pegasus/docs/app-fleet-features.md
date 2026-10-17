# Pegasus Features, Portals & Multi-Role Client Ecosystem

> **Ecosystem**: Pegasus Client Fleet, Commerce Flows & Operational Portals  
> **Source Path**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/FEATURES_AND_PORTALS.md`  
> **Referenced Codebases**:
> - Web/Desktop Portals: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/admin-portal`, `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/factory-portal`, `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-portal`, `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-desktop`
> - Terminal: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/payload-terminal`
> - Android Fleet: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/driver-app-android`, `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-android`, `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/factory-app-android`, `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-app-android`, `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/payload-app-android`
> - iOS Fleet: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/driverappios`, `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-ios`, `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/factory-app-ios`, `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-app-ios`, `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/payload-app-ios`

---

## 1. Multi-Role Client Fleet Overview

Pegasus delivers tailored native and desktop interfaces for 5 operational personas across 15 distinct applications:

```
                            PEGASUS CLIENT ECOSYSTEM
  ┌─────────────────────────────────────────────────────────────────────────────┐
  │                         4 Desktop & Web Portals                             │
  │     (Next.js 15.5 + React 19.1 + Tauri 2.10 + HeroUI 3.0 + Tailwind 4)      │
  ├─────────────────────┬─────────────────────┬─────────────────────────────────┤
  │ Admin Portal        │ Factory Portal      │ Warehouse Portal                │
  │ (Supplier & Global) │ (Production bay)    │ (Intake & locks)                │
  ├─────────────────────┴─────────────────────┴─────────────────────────────────┤
  │ Retailer App Desktop (POS terminal, wholesale cart & offline invoice PDF)   │
  └─────────────────────────────────────────────────────────────────────────────┘
  ┌─────────────────────────────────────────────────────────────────────────────┐
  │                         5 Native Android Applications                       │
  │              (Kotlin 2.x, Jetpack Compose, Material 3, Hilt, Room)          │
  ├──────────────┬──────────────┬──────────────┬──────────────┬─────────────────┤
  │ Driver App   │ Retailer App │ Factory App  │ Warehouse App│ Payload App     │
  └──────────────┴──────────────┴──────────────┴──────────────┴─────────────────┘
  ┌─────────────────────────────────────────────────────────────────────────────┐
  │                           5 Native iOS Applications                         │
  │                 (Swift 6, SwiftUI, SwiftData, CoreLocation, HIG)            │
  ├──────────────┬──────────────┬──────────────┬──────────────┬─────────────────┤
  │ Driver App   │ Retailer App │ Factory App  │ Warehouse App│ Payload App     │
  └──────────────┴──────────────┴──────────────┴──────────────┴─────────────────┘
  ┌─────────────────────────────────────────────────────────────────────────────┐
  │                         1 Rugged Terminal Application                       │
  │               (Expo 55, React Native 0.83, Barcode Scanner SDK)             │
  └─────────────────────────────────────────────────────────────────────────────┘
```

### 1.1 Desktop & Web Portals (Next.js 15, React 19, Tauri 2)
1. **Admin / Supplier Portal (`apps/admin-portal`)**:
   - *Manifest*: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/admin-portal/package.json`
   - *Core Capabilities*: Live MapLibre GL telemetry tracking showing real-time driver coordinates; 4-step supplier onboarding wizard (`supplier/registration.go`); dynamic B2B pricing override matrices; picking manifest compilation; billing and gateway management.
2. **Factory Portal (`apps/factory-portal`)**:
   - *Manifest*: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/factory-portal/package.json`
   - *Core Capabilities*: Production line assignment, outbound freight dispatch bays, inter-warehouse replenishment approvals, and factory truck loading schedule.
3. **Warehouse Portal (`apps/warehouse-portal`)**:
   - *Manifest*: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-portal/package.json`
   - *Core Capabilities*: Digital seal verification for inbound pallets, staging bay allocation, fleet queue dispatching, and live mutual-exclusion dispatch lock monitoring via `/ws/warehouse`.
4. **Retailer Desktop POS (`apps/retailer-app-desktop`)**:
   - *Manifest*: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-desktop/package.json`
   - *Core Capabilities*: Hardware barcode scanner support, keyboard-driven checkout, 1-click wholesale reordering, local invoice PDF printing, and offline catalog browsing.

### 1.2 Native Mobile Fleet (Android & iOS)
- **Driver Applications (`driver-app-android`, `driverappios`)**:
  - *Android*: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/driver-app-android`
  - *iOS*: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/driverappios`
  - *Capabilities*: Offline-first route execution (Desert Protocol), background GPS telemetry publishing, turn-by-turn stop sequences, camera QR scanner verification handshake, and cash collection ledger logging.
- **Retailer Applications (`retailer-app-android`, `retailer-app-ios`)**:
  - *Android*: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-android`
  - *iOS*: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-ios`
  - *Capabilities*: Empathy Engine preorder review (approve/edit/cancel), delivery tracking with `DRIVER_APPROACHING` alerts, authorized clerk management (`RetailerFamilyMembers`), and dispute submissions.
- **Factory Applications (`factory-app-android`, `factory-app-ios`)**:
  - *Capabilities*: Mobile loading dock verification, pallet barcode scanning, and manifest seals.
- **Warehouse Applications (`warehouse-app-android`, `warehouse-app-ios`)**:
  - *Capabilities*: Floor stock cycle counting, truck intake inspection, and exception recording.
- **Payload Scanners (`payload-app-android`, `payload-app-ios`, `payload-terminal`)**:
  - *Capabilities*: Industrial ruggedized scanning terminal for loading docks with tamper-evident digital seal locking.

---

## 2. Core Logistics Flow & Doorstep Handshake Protocols

### 2.1 What It Is
The end-to-end logistics execution flow governs an order from checkout through delivery confirmation. It is implemented in `apps/backend-go/order/service.go`, `orderroutes/routes.go`, and `deliveryroutes/routes.go`.

### 2.2 How It Works: The 7-Step Handshake Lifecycle

```
    [1. PREORDER / CHECKOUT]
           │
           ▼
    [2. WAREHOUSE PICK & PACK] ──► SupplierTruckManifests Created
           │
           ▼
    [3. TRUCK DISPATCH & SEAL] ──► Digital Seal Locked (/v1/payload/seal)
           │
           ▼
    [4. 1.5 KM GEOFENCE APPROACH] ──► DRIVER_APPROACHING Pushed to Retailer WS
           │
           ▼
    [5. DOORSTEP QR HANDSHAKE] ──► Driver & Retailer Scan (/v1/order/validate-qr)
           │
           ▼
    [6. PHYSICAL OFFLOAD & SIGN] ──► Verify SKU Counts (/v1/order/confirm-offload)
           │
           ▼
    [7. SETTLEMENT & RECONCILIATION] ──► Cash Collect or Gateway Charge (/v1/order/collect-cash)
```

1. **Order Creation & Verification**: The retail merchant places an order via mobile or desktop (`POST /v1/retailer/orders`). Backend verifies inventory in `SupplierInventoryV2`, confirms credit limits in `Retailers`, and writes an order in `PENDING` state.
2. **Picking & Packing**: Warehouse staff pull items according to picking tickets and pack line items into delivery trucks, grouping them into `SupplierTruckManifests` (`spanner.ddl:1792`).
3. **Dock Dispatch & Sealing**: Payloader staff scan pallets and apply a digital tamper-evident seal (`POST /v1/payload/seal`). The order state transitions to `LOADED`.
4. **Geofenced Approach (`DRIVER_APPROACHING`)**: As the driver approaches the retail shop and crosses the 1.5 km geofence radius, Kafka event `DRIVER_APPROACHING` is emitted. `StartApproachConsumer` (`main.go:645`) pushes a high-priority alert to the merchant's mobile device via `RetailerHub`, alerting them to prepare storefront clearance.
5. **Doorstep Cryptographic Handshake**:
   - When the truck arrives (`POST /v1/order/deliver`), a dynamic single-use QR token is presented.
   - The driver and retailer perform a dual-device QR scan validated by `POST /v1/order/validate-qr` (`orderroutes/routes.go`).
   - If the retailer's phone camera is broken, an admin can issue an emergency 6-digit cryptographic bypass token via `POST /v1/delivery/bypass-offload`.
6. **Physical Offload & Discrepancy Verification**: The driver unloads cases. If goods are damaged or missing, `POST /v1/delivery/missing-items` logs exceptions in `ManifestExceptions`. If quantities are negotiated downwards at the curb, `POST /v1/delivery/negotiate` adjusts the invoice.
7. **Settlement & Cash Custody**:
   - If electronic payment: Card token is charged via Global Pay / Adyen.
   - If cash-on-delivery: Driver receives physical currency, logs the amount via `POST /v1/order/collect-cash`, and sets `CustodyStatus = 'HELD_BY_DRIVER'` in `MasterInvoices`. The order reaches final state `COMPLETED`.

### 2.3 Exceptional Edge Protocols
- **Shop-Closed Protocol (`deliveryroutes/routes.go`, `main.go:374`)**: If the driver finds the store closed, `POST /v1/delivery/shop-closed` triggers push notifications, begins a 15-minute grace period timer, and allows the merchant to respond (`OPEN_NOW`, `5_MIN`, `CALL_ME`, or `CLOSED_TODAY`). If unresolved, the order transitions to `ARRIVED_SHOP_CLOSED` and the truck reroutes.
- **Desert Protocol (`sync/routes.go`)**: When drivers enter cellular blackouts in rural valleys, the mobile app caches offloads, digital signatures, and cash collections locally in Room/SwiftData. Upon regaining signal, `POST /v1/sync/batch` flushes queued actions idempotently to Spanner without duplicating ledger entries.

### 2.4 Why It Is There
High-value B2B delivery in emerging markets frequently suffers from informal disputes ("I never received case #4", "The shop was closed when I arrived", "The driver took the cash and didn't report it"). Cryptographic QR handshakes, automated geofenced alerts, and strict cash custody logging eliminate fraud and provide non-repudiable audit logs.

---

## 3. Multi-Facility Replenishment & Internal Supply Chain

### 3.1 What It Is
Pegasus separates external wholesale delivery (Warehouse → Retailer) from internal manufacturing replenishment (Factory → Warehouse). The internal supply network is coordinated by services in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/factory/`.

### 3.2 How It Works
- **Supply Lanes (`SupplyLanes`, `spanner.ddl:1986`)**: Direct transport channels connecting specific factories to fulfillment warehouses, configured with SLA duration thresholds and vehicle size restrictions.
- **Internal Transfer State Machine (`InternalTransferOrders`, `spanner.ddl:1214`)**:
  `DRAFT` → `APPROVED` → `LOADING` → `DISPATCHED` → `IN_TRANSIT` → `ARRIVED` → `RECEIVED`
- **Algorithmic Pull Matrix (`PullMatrixService`, `factory/pull_matrix.go`)**: Runs every 4 hours via `StartPullMatrixAggregator` (`cron.go:1229`). It evaluates current warehouse stock levels against 7-day sales velocity and forecasted preorder demand. If inventory drops below safe buffer thresholds, it automatically generates draft `InternalTransferOrders`.
- **Predictive Push Engine (`PredictivePushService`, `factory/predictive_push.go`)**: Evaluates factory production yields and automatically allocates bulk outputs across regional warehouses based on relative regional demand intensity.
- **Dispatch Mutual-Exclusion Locks (`DispatchLocks`, `spanner.ddl:1762`)**: When warehouse cycle counts, inventory audits, or replenishment intake operations are active, an operator arms a `DispatchLock`. While locked, `warehouseroutes` rejects driver dispatch requests, preventing trucks from departing with unverified stock.
- **Force-Receive Service (`ForceReceiveService`, `factory/force_receive.go`)**: Allows warehouse supervisors to accept incoming shipments with missing barcodes or damaged manifests, placing discrepancies into quarantine for audit.

### 3.3 Why It Is There
Without automated inter-facility replenishment, central factories overproduce low-demand items while regional fulfillment hubs suffer stockouts on high-velocity goods. The pull matrix ensures production directly tracks consumption.

---

## 4. Master Invoices, Degressive Fees & Settlement Slices

### 4.1 What It Is
All monetary transactions are tracked through immutable financial records defined in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/schema/spanner.ddl` and processed by `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/treasury/`.

### 4.2 How It Works

#### 1. Master Invoices (`MasterInvoices`, `spanner.ddl:140`)
When an order checkouts, a `MasterInvoices` row is created:
- Stores `Total INT64 NOT NULL` in minor currency units (e.g. tiyin).
- Records `FeePolicyVersion` and `FeeAmount`.
- Tracks `CustodyStatus` (`HELD_BY_DRIVER` or `DEPOSITED`).
- Records exact GPS latitude/longitude and distance to retailer at the moment physical cash was collected (`CollectionLat`, `CollectionLng`, `GeofenceDistanceM`).

#### 2. Immutable Settlement Slices (`InvoiceSettlementSlices`, `spanner.ddl:188`)
Each master invoice is split into settlement slices detailing exact fund allocation:
- Platform fee slice: `GrossAmount * FeeBasisPoints / 10000`.
- Supplier net payout slice: `NetPayoutAmount`.
- Regional warehouse fulfillment fee slice.
- Each slice records its `SelectedTierKey` and audit lineage (`RevisionOf`, `DeliverySessionId`). If doorstep negotiations reduce line items, a revision slice is appended, never overwriting historical rows.

#### 3. Degressive Regional Fee Scheduling (`order/fee_policy.go`)
Platform fees follow a degressive schedule configured per currency in `RegionalConfigs` (`spanner.ddl:1435`):
- **Base Tier**: Standard turnover (e.g. 250 basis points / 2.5%).
- **Growth Tier**: Above monthly volume threshold 1 (drops to 180 basis points / 1.8%).
- **Scale Tier**: Enterprise volume threshold 2 (drops to 120 basis points / 1.2%).
The fee tier is locked into `InvoiceSettlementSlices` at checkout time and remains immutable even if policies change later.

```
       [Order Checkout: 10,000,000 UZS (100,000,000 Tiyin)]
                                 │
                                 ▼
                     MasterInvoice (InvoiceId)
                                 │
       ┌─────────────────────────┴─────────────────────────┐
       │                                                   │
       ▼                                                   ▼
 Slice 1: Supplier Payout                            Slice 2: Platform Fee
 PayoutOwner: Supplier HQ                            PayoutOwner: Pegasus Platform
 Gross: 10,000,000 UZS                               FeeBasisPoints: 200 bps (2.0%)
 Net: 9,800,000 UZS                                  FeeAmount: 200,000 UZS
```

### 4.3 Why It Is There
Dynamic fee recalculations after fulfillment create trust deficits with suppliers. By locking fee basis points and settlement slices immutably at the instant of checkout, Pegasus guarantees 100% financial predictability and auditability.

---

## 5. Uber H3 Spatial Proximity & Hexagonal Clustering

### 5.1 What It Is
All spatial calculations in Pegasus rely on the **Uber H3** hexagonal hierarchical spatial index (`github.com/uber/h3-go/v4`), implemented in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/proximity/h3.go` and `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/proximity/engine.go`.

### 5.2 How It Works
- **Resolution-7 Spatial Cells**: Every retailer, warehouse, and factory is indexed by an H3 resolution-7 cell (~1.2 km² area, ~1.4 km hexagon edge length).
- **Warehouse Catchment Polygons (`Warehouses.H3Indexes`)**: A warehouse's delivery territory is modeled not as a complex geometric polygon shapefile, but as a compact JSON array of H3 resolution-7 string IDs.
- **Fast Ring Proximity via `GridDisk`**:
  To find all active warehouses serving a merchant at cell $C$, the system queries the set of warehouse coverage arrays containing $C$ in $O(1)$ time. Expanding geographic searches (e.g. finding nearby drivers) uses `h3.GridDisk(centerCell, radiusSteps)` to traverse concentric hexagonal rings without trigonometric floating-point math.
- **Panic-Safe Haversine Fallback**: If an H3 string is corrupted or invalid, `proximity/engine.go` catches errors gracefully and evaluates standard Great-Circle Haversine distance formulas.

```
                     Uber H3 Hexagonal Topology
                              ┌───┐
                          ┌───┤ 2 ├───┐
                          │ 2 ├───┤ 2 │
                      ┌───┼───┤ 1 ├───┼───┐
                      │ 2 │ 1 ├───┤ 1 │ 2 │
                      ├───┼───┤ C ├───┼───┤   C = Center Cell (Shop)
                      │ 2 │ 1 ├───┤ 1 │ 2 │   1 = GridDisk Ring 1 (~1.4 km)
                      └───┼───┤ 1 ├───┼───┘   2 = GridDisk Ring 2 (~2.8 km)
                          │ 2 ├───┤ 2 │
                          └───┤ 2 ├───┘
                              └───┘
```

### 5.3 Why It Is There
Relational databases running spatial polygon intersection queries (`ST_Contains`, `ST_Intersects`) experience significant CPU degradation under high concurrency. H3 converts geographic coordinates into 64-bit integer cell identifiers, turning spatial relationship tests into instantaneous hash lookups and index scans.
