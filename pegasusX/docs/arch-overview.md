# PegasusX Architecture & System Blueprint

> **Document Scope**: Core System Architecture, Storage Topology, Messaging Infrastructure, Monorepo Organization, and Multi-Region Cell Isolation  
> **Source Root**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX`  
> **Authoritative Inventory**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/context/technology-inventory.json`  
> **Architecture Graph**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/context/architecture-graph.json`  

---

## 1. Executive Summary & The SSMR Doctrine

### 1.1. Single-Supplier Multi-Retailer (SSMR) vs Open Marketplace

#### What it is
PegasusX is an enterprise logistics, supply chain execution, and distribution operating system engineered specifically for wholesale Fast-Moving Consumer Goods (FMCG) distribution in Central Asia (principally Uzbekistan / Tashkent metropolitan area and regional provinces) with built-in multi-region cell expansion.

At its core, PegasusX enforces the **Single-Supplier Multi-Retailer (SSMR)** architectural model (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/package.json:5`), distinguishing it fundamentally from open multi-vendor retail platforms (such as Amazon, UberEats, or Allegro):

| Architectural Dimension | Open Marketplace Architecture | PegasusX SSMR Architecture |
|---|---|---|
| **Catalog Authority** | Fragmented across tens of thousands of competitive merchants | Authoritative master catalog owned by the single supplier organization |
| **Pricing Strategy** | Real-time competitive bidding and per-merchant price wars | Hierarchical contract pricing, tiered volume discounts, and customer-group terms |
| **Warehouse & Fulfillment** | Decentralized merchant warehouses or third-party fulfillment (3PL) | Hub-and-spoke warehouse distribution centers and integrated manufacturing factories |
| **Fleet & Transport** | Crowdsourced gig drivers with ad-hoc route bidding | Dedicated supplier-owned truck fleet with volume-constrained route optimization (VRP) |
| **Physical Dock Control** | External courier pickup at merchant premises | Dock gate manifests, order injection, truck loading sequence, and tamper-evident seals |
| **Fiscal & Legal Realm** | B2C consumer invoices, payment splitters, merchant payouts | Mandatory B2B Soliq EHF electronic tax invoicing and OFD fiscal memory under ADR-009 |
| **Credit & Settlement** | Instant card capture before dispatch | Mixed payment rails: upfront card, cash-on-delivery (Tiyin integer), and Accounts Receivable (AR) credit |

#### How it works
1. **Catalog & Contract Pricing**: The supplier defines the master catalog (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/schema/spanner.ddl:758-791`), establishes base prices, and attaches customer-specific promotional overrides and credit terms (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/pricing/service.go`).
2. **Retailer Ordering & S&OP**: Retailers browse the catalog via responsive web/desktop portals or native mobile apps. Orders can be generated manually, scheduled as future-dated preorders, or created automatically by the AI replenishment engine (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/replenishment/engine.go`). Multi-supplier rollup carts are coordinated through the `ParentOrders` saga (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/schema/spanner.ddl:222-235`).
3. **WMS Wave Picking & Palletizing**: Orders assigned to a distribution warehouse are aggregated into pick waves (`spanner.ddl:261-264`). Workers scan lot numbers (`WMSStockLots`) verifying expiration dates and cold-chain compliance before staging goods on loading docks (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/warehouseroutes/routes.go`).
4. **VRP Fleet Routing & Manifest Sealing**: Google OR-Tools solvers (`services/optimizer-core/proto/optimizer_core.proto:22-74`) compute capacity-constrained multi-drop vehicle routes. Loading dock operators inspect truck volume utilization, inject late priority orders, verify physical loading sequences, and apply tamper-evident security seals (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/payloaderoutes/routes.go`).
5. **Turn-by-Turn Delivery & EPOD**: Drivers follow polyline geometries provided by OSRM (`apps/backend-go/deliveryroutes/`). GPS telemetry is streamed every few seconds (`apps/backend-go/telemetryroutes/`). At the retailer shop, proof of delivery is established using HMAC QR handoff tokens verified by the stateless verification service (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/handoff-service/main.go:29-75`).
6. **Cash Settlement & Soliq EHF Fiscalization**: Payments are collected either digitally via Global Pay, in cash with integer Tiyin precision, or left behind on credit. Captured payments immediately trigger fiscalization against the Uzbekistan State Tax Committee (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/soliq/client.go:1-60`), locking order status into `COMPLETED`.

#### Why it is there
Wholesale FMCG supply chains in emerging markets operate under severe physical constraints: narrow delivery windows in congested urban cores, high cash-on-delivery volume, strict tax enforcement with legal liability for unregistered goods, and low-margin economics requiring full truck utilization. The SSMR architecture eliminates marketplace transaction friction and enforces end-to-end custody and fiscal compliance.

---

## 2. Monorepo Organization & Toolchains

PegasusX is organized as a unified polyglot monorepo coordinating Go microservices, TypeScript/React web portals, Tauri desktop wrappers, native Android Kotlin applications, native iOS Swift applications, Rust/C++ solvers, and Python optimization sidecars.

```
file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/
├── apps/                          # 22 Standalone Applications
│   ├── backend-go/                # Core HTTP API & 24 runtime workers (Go 1.25)
│   ├── ai-worker/                 # Kafka predictive synthesis & bulk ingestion (Go 1.25)
│   ├── handoff-service/           # High-throughput stateless QR verification (Go 1.25)
│   ├── dispatch-optimizer-py/     # Python OR-Tools pick path solver sidecar
│   ├── admin-portal/              # Platform governance & tenant management (Next.js 15)
│   ├── supplier-portal/           # Supplier executive control tower (Next.js 15 + Tauri 2)
│   ├── warehouse-portal/          # Distribution center operations (Next.js 15 + Tauri 2)
│   ├── factory-portal/            # Manufacturing dock & inter-warehouse transfer (Next.js 15 + Tauri 2)
│   ├── retailer-app-desktop/      # Storefront POS & offline ordering (Next.js 15 + Tauri 2)
│   ├── payload-terminal/          # Loading dock dockmaster terminal (Expo 55 / React Native)
│   ├── driver-app-android/        # Native Android driver dispatch (Kotlin + Jetpack Compose)
│   ├── driver-app-ios/            # Native iOS driver dispatch (Swift + SwiftUI)
│   ├── retailer-app-android/      # Native Android retailer store app (Kotlin + Compose)
│   ├── retailer-app-ios/          # Native iOS retailer store app (Swift + SwiftUI)
│   ├── supplier-app-android/      # Native Android mobile supplier console (Kotlin + Compose)
│   ├── supplier-app-ios/          # Native iOS mobile supplier console (Swift + SwiftUI)
│   ├── warehouse-app-android/     # Native Android warehouse scanner (Kotlin + Compose)
│   ├── warehouse-app-ios/         # Native iOS warehouse scanner (Swift + SwiftUI)
│   ├── factory-app-android/       # Native Android factory floor scanner (Kotlin + Compose)
│   ├── factory-app-ios/           # Native iOS factory floor scanner (Swift + SwiftUI)
│   ├── payload-app-android/       # Native Android loading terminal (Kotlin + Compose)
│   └── payload-app-ios/           # Native iOS loading terminal (Swift + SwiftUI)
├── contracts/                     # Formal schemas, OpenAPI specs, marker definitions
├── design-system/                 # Shared design tokens and visual primitives
├── infra/                         # Docker Compose, Kubernetes manifests, Terraform modules
├── packages/                      # 24 Shared polyglot libraries (TS, Go, Android Gradle, iOS SPM)
├── scripts/                       # 105 Verification, migration, QA, and validation scripts
├── sdk/                           # Partner B2B SDKs in Go and TypeScript
└── services/                      # Standalone solver services (Rust & C++ OR-Tools)
```

### 2.1. Toolchain Manifests

#### Go Workspace (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/go.work:1-12`)
- **Version**: Go `1.25.0`
- **Configured Modules**:
  - `./apps/ai-worker`
  - `./apps/backend-go`
  - `./apps/handoff-service`
  - `./packages/config`
  - `./packages/handoff`
  - `./packages/optimizer-contract`
  - `./sdk/partner/go`
- **Architecture**: Modules share dependencies locally without requiring external git tags or private repository mirrors during active development.

#### PNPM & Node Workspace (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/pnpm-workspace.yaml:1-23`, `package.json:1-47`)
- **Package Manager**: `pnpm@9.0.0`, Node `>=20`
- **Portals Included**: `apps/supplier-portal`, `apps/admin-portal`, `apps/retailer-app-desktop`, `apps/warehouse-portal`, `apps/factory-portal`, `apps/payload-terminal`.
- **Shared TS Packages**: `@pegasusx/types`, `@pegasusx/api-client`, `@pegasusx/validation`, `@pegasusx/i18n`, `@pegasusx/motion-tokens`, `@pegasusx/ui-kit`, `@pegasusx/pulse-ui`, `@pegasusx/explain-ui`, `@pegasusx/ws-refresh-contract`, `@pegasusx/desktop-bridge`, `@pegasusx/desktop-cache`.

#### Android Gradle Workspaces
- Each of the 6 Android apps (`driver-app-android`, `retailer-app-android`, etc.) includes a dedicated `gradlew` wrapper, `settings.gradle.kts`, and `build.gradle.kts` targeting Kotlin 1.9+/2.0+ and Jetpack Compose.
- Shared Gradle packages: `packages/mobile-android-kit`, `packages/mobile-android-design`, `packages/mobile-android-barcode-scanner`.

#### iOS Swift Package Manager & XcodeGen
- Native iOS applications utilize XcodeGen (`project.yml`) or native `.xcodeproj` bundles:
  - `apps/driver-app-ios/driverappios/driverappios.xcodeproj`
  - `apps/retailer-app-ios/retailerapp/retailerapp.xcodeproj`
  - XcodeGen roots: `apps/supplier-app-ios/project.yml`, `apps/warehouse-app-ios/project.yml`, `apps/factory-app-ios/project.yml`, `apps/payload-app-ios/project.yml`.
- Shared SPM packages: `packages/mobile-ios-core` (`Package.swift`), `packages/mobile-ios-kit`, `packages/mobile-ios-design`, `packages/mobile-ios-barcode`.

#### Rust & C++ Solvers
- `services/optimizer-core/server-rust/Cargo.toml` implements high-throughput VRP solver routines in Rust.
- `services/optimizer-core/Dockerfile` compiles C++ Google OR-Tools binaries for CP-SAT dock allocation.

---

## 3. Storage Layer: Google Cloud Spanner Schema Architecture

### 3.1. Overview & Consistency Model

#### What it is
The authoritative system ledger is hosted on **Google Cloud Spanner**, defined in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/schema/spanner.ddl` (159KB, 220+ tables) and evolved through 125 incremental migration files in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/schema/migrations/`.

#### How it works
- **Strict Serializability**: Spanner provides external consistency (TrueTime-backed linearizability) with multi-version concurrency control (MVCC). All state updates run in `spanner.ReadWriteTransaction` closures.
- **Commit Timestamps**: Every mutating table maintains `CreatedAt` and `UpdatedAt` timestamps configured with `OPTIONS (allow_commit_timestamp=true)` (`spanner.ddl:206-207`), providing an immutable monotonic timeline for change data capture.
- **Interleaved Tables & Indexes**: Tables requiring tight co-location (such as `ManifestOrders` or `OrderDeliveryProofs`) use secondary indexes optimized for tenant-scoped querying.

#### Why it is there
FMCG operations cannot tolerate read-after-write skew, ghost inventory allocations, or double-spending in financial ledgers. Spanner provides distributed, cross-region linearizable transactions without the operational hazards of manual database sharding.

---

### 3.2. Primary Data Graphs & Entity Relational Model

```
+----------------------------------------------------------------------------------------------------+
|                                    SPANNER MASTER DATA GRAPH                                       |
+----------------------------------------------------------------------------------------------------+
|                                                                                                    |
|  [Suppliers] (1) <---+---> (N) [SupplierProfiles]                                                 |
|        |             |                                                                             |
|        |             +---> (N) [Warehouses] (1) <-----+                                            |
|        |             |                                |                                            |
|        |             +---> (N) [Factories]            |                                            |
|        |             |                                |                                            |
|        |             +---> (N) [Vehicles]             |                                            |
|        |             |                                |                                            |
|        |             +---> (N) [Drivers]              |                                            |
|        |                                              |                                            |
|        v                                              v                                            |
|  [Products] (1) ---------> (N) [InventoryLevels] (N) <+                                            |
|        |                                                                                           |
|        v                                                                                           |
|  [ParentOrders] (1) -----> (N) [Orders] (1) --------> (1) [OrderDeliveryProofs]                   |
|                                   |                                                                |
|                                   +-----------------> (N) [PaymentLedgerEntries]                   |
|                                   |                                                                |
|                                   +-----------------> (1) [OrderFiscalReceipts]                    |
|                                   |                                                                |
|                                   +-----------------> (1) [OutboxEvents]                           |
+----------------------------------------------------------------------------------------------------+
```

#### 1. Organizational Master Data & Topology
- **`Suppliers`** (`spanner.ddl:11-35`): Authoritative supplier entity. Stores `SupplierId`, `Name`, `Status`, `DefaultCurrency`, and timestamps.
- **`SupplierProfiles`** (`spanner.ddl:37-65`): Legal business credentials, Soliq tax TIN (`TaxIdentificationNumber`), bank routing codes (`MFO`), and physical headquarters address.
- **`Retailers`** (`spanner.ddl:145-168`): Storefront entity with H3 spatial coordinate (`H3Cell`), credit limits, payment terms, and registered Soliq TIN.
- **`Warehouses`** (`spanner.ddl:437-474`): Regional distribution centers. Stores geographic coordinates (`Lat`, `Lng`), storage capacities (`CapacityVU`), temperature refrigeration classifications, and operating hours.
- **`Factories`** (`spanner.ddl:496-530`): Manufacturing centers with designated inbound loading dock capacities and palletizing bays.
- **`WarehouseCoverageCells`** (`spanner.ddl:476-494`): Spatial mapping associating Uber H3 resolution-9 hexagon cells with designated serving warehouses.

#### 2. Order Lifecycle & Parent Order Rollup
- **`Orders`** (`spanner.ddl:169-208`): The central operational transaction.
  - Primary Key: `OrderId` (UUIDv4).
  - Foreign Keys: `SupplierId`, `RetailerId`, `WarehouseId`, `DriverId`, `VehicleId`, `RouteId`, `ManifestId`, `ParentOrderId`.
  - State Enumeration: `Status` (`PENDING`, `LOADED`, `IN_TRANSIT`, `ARRIVED`, `SHOP_CLOSED_PENDING`, `AWAITING_PAYMENT`, `PENDING_CASH_COLLECTION`, `DELIVERED_ON_CREDIT`, `FISCALIZING`, `FISCAL_FAILED`, `COMPLETED`, `CANCELLED`, `DELAYED`, etc.).
  - Financials: `TotalMinor` (`INT64`), `OriginalTotalMinor` (`INT64`), `Currency` (`STRING(3)`).
  - Payload Items: `LineItemsJson` (`BYTES(MAX)`).
  - Proximity & Security: `DeliveryToken` (`STRING(36)`), `H3Cell`, `Lat`, `Lng`, `ProximityUnlockedAt`.
  - Soliq EHF Fiscal Status: `BuyerAcceptanceStatus`, `BuyerAcceptanceDeadline`.
- **`ParentOrders`** (`spanner.ddl:222-235`): Enables retailers to checkout a multi-supplier basket in a single atomic consumer transaction.
  - Fields: `ParentOrderId`, `RetailerId`, `Status`, `Currency`, `TotalMinor`, `ChildCount`, `SagaState` (`PENDING`, `COMMITTED`, `COMPENSATING`, `FAILED`), `ExpectedChildCount`, `CreatedChildOrderIds` (`ARRAY<STRING(64)>`), `LeaseExpiresAt`.
  - Recovery Index: `Idx_ParentOrders_SagaRecovery ON ParentOrders(SagaState, LeaseExpiresAt)` (`spanner.ddl:238`) powers the background saga reconciler.
- **`OrderDeliveryProofs`** (`spanner.ddl:331-360`): Immutable EPOD verification containing cryptographic QR token hash, driver GPS coordinates, retailer signature, and offload photos.

#### 3. Financial, Ledger & Tax Entities
- **`PaymentSessions`** (`spanner.ddl:576-594`): Checkout session states supporting multiple providers (Global Pay, Cash, Credit).
- **`PaymentAttempts`** (`spanner.ddl:595-630`): Granular payment attempt records tracking gateway request payloads and transaction IDs.
- **`PaymentLedgerEntries`** (`spanner.ddl:660-684`): **Immutable double-entry transaction ledger**. Enforces zero modification: entries are append-only.
  - Unique Index: `Idx_PaymentLedgerEntries_GatewayTypeRef ON PaymentLedgerEntries(Gateway, EntryType, ReferenceId)` (`spanner.ddl:682-683`) guarantees that duplicate webhook deliveries or retried API requests can never double-record a financial debit or credit.
- **`ARInvoices`** (`migrations/20260807_ar_dunning.ddl`): Accounts Receivable ledger tracking open credit terms with dynamic aging buckets (`CURRENT`, `1_30`, `31_60`, `61_90`, `90_PLUS`) and automated dunning progression.
- **`OrderFiscalReceipts`** (`migrations/20260720_fiscal_receipts.ddl`): ADR-009 fiscal records linking order payments to OFD fiscal memory and Soliq electronic tax invoice IDs.

#### 4. Fleet, Manifests & Transport
- **`Drivers`** (`spanner.ddl:394-416`): Driver profile, license details, availability state, and assigned vehicle ID.
- **`Vehicles`** (`spanner.ddl:418-435`): Truck fleet with volume capacity in Volume Units (`CapacityVU`), weight thresholds, and refrigeration attributes.
- **`SupplierTruckManifests`** (`spanner.ddl:901-957`): Vehicle run manifests tracking driver assignment, truck departure times, tamper-evident security seal numbers, and compressed route polylines (`EncodedRoutePolyline`).
- **`ManifestOrders`** (`spanner.ddl:959-975`): Mapping linking individual orders to truck manifests with stop offload sequencing.

#### 5. Inventory & Lot Traceability
- **`Products`** (`spanner.ddl:758-791`): Master SKU catalog with dimensions, unit volume (`VolumeVU`), barcode strings, excise marks, and packaging ratios.
- **`InventoryLevels`** (`spanner.ddl:793-810`): Warehouse balance tracking on-hand, allocated, and reserved stock.
- **`WMSStockLots`** (`migrations/20260806_wms_foundation.ddl`): Lot-level traceability tracking manufacture dates, expiry dates, batch codes, and cold-chain temperature thresholds.

#### 6. Transactional Outbox & Dead Letters
- **`OutboxEvents`** (`spanner.ddl:685-702`): Transactional staging table:
  - `EventId` (`STRING(36)`), `AggregateType` (`STRING(64)`), `AggregateId` (`STRING(64)`), `TopicName` (`STRING(128)`), `Payload` (`BYTES(MAX)`), `CreatedAt` (`TIMESTAMP`), `PublishedAt` (`TIMESTAMP`), `ClaimedBy` (`STRING(64)`), `ClaimedUntil` (`TIMESTAMP`), `PublishAttempts` (`INT64`), `SupplierId` (`STRING(64)`).
  - Index: `Idx_OutboxEvents_Unpublished ON OutboxEvents(PublishedAt, CreatedAt)` (`spanner.ddl:699`).
- **`OutboxDeadLetters`** (`spanner.ddl:704-718`): Quarantine store capturing events that exhaust their retry budget (20 attempts):
  - Stores `EventId`, `AggregateType`, `AggregateId`, `TopicName`, `Payload`, `DeadLetteredAt`, `Attempts`, and `LastError`.

---

## 4. Messaging Backbone: Apache Kafka Event Spine & Transactional Outbox

### 4.1. Kafka Topic Architecture

#### What it is
Apache Kafka serves as the distributed asynchronous event spine connecting backend domain services, the AI worker, the digital twin, and external partner systems (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/events/events.go:10-30`, `infra/docker-compose.ssmr.yml:73-87`).

#### How it works
The broker runs with `KAFKA_AUTO_CREATE_TOPICS_ENABLE: "false"`. All topics are pre-provisioned with 3 partitions and configured replication factors:

| Topic Name | Environment Variable | Partitions | Purpose |
|---|---|---|---|
| **`pegasusx-main`** | `KAFKA_TOPIC_MAIN` | 3 | Default multi-event topic carrying all domain state transitions |
| **`pegasusx-main-dlq`** | `KAFKA_TOPIC_MAIN-dlq` | 3 | Dead-letter queue for un-processable events |
| **`pegasusx-orders`** | `KAFKA_TOPIC_ORDERS` | 3 | Dedicated order creation and status stream |
| **`pegasusx-dispatch`** | `KAFKA_TOPIC_DISPATCH` | 3 | Dispatch planning, manifest assignments, and route updates |
| **`pegasusx-demand`** | `KAFKA_TOPIC_DEMAND` | 3 | Downstream POS sell-through demand feed (`DEMAND_SIGNAL`) |
| **`pegasusx-freeze-locks`** | `KAFKA_TOPIC_FREEZE_LOCKS` | 3 | Real-time dispatcher freeze locks broadcast to AI workers |
| **`pegasusx-inventory-import`** | `KAFKA_TOPIC_INVENTORY_IMPORT` | 3 | Bulk CSV/Excel catalog ingestion lifecycle stream |
| **`ssmr.events.spatial`** | `KAFKA_TOPIC_SPATIAL` | 3 | High-frequency driver GPS breadcrumbs and spatial telemetry |
| **`planning.signal.ingest.v1`** | - | 3 | Ingestion feed for machine learning demand planning |
| **`planning.forecast.request.v1`** | - | 3 | Asynchronous forecasting recalculation trigger |
| **`planning.forecast.result.v1`** | - | 3 | Forecast calculation results published to downstream consumers |

---

### 4.2. Transactional Outbox Relay Loop

```
+----------------------------------------------------------------------------------------------------+
|                                TRANSACTIONAL OUTBOX RELAY PATTERN                                  |
+----------------------------------------------------------------------------------------------------+
|                                                                                                    |
|  [HTTP Route Handler]                                                                              |
|         |                                                                                          |
|         v                                                                                          |
|  [Spanner ReadWriteTransaction]                                                                    |
|         |---> 1. Update Domain Entity (e.g. Orders, Manifests, Payments)                           |
|         +---> 2. Insert Staged Event into OutboxEvents table                                       |
|         v                                                                                          |
|  [Transaction Commit]                                                                              |
|         |                                                                                          |
|         v (Async Polling - 250ms tick)                                                             |
|  [Outbox Relay Worker] (apps/backend-go/outbox/relay.go)                                            |
|         |                                                                                          |
|         +---> Fetch 100 Unpublished Events (ClaimedUntil lock)                                     |
|         +---> Publish to Kafka (10s timeout, exponential backoff)                                  |
|         |        |                                                                                 |
|         |        +--[Success]--> Mark PublishedAt = NOW() in Spanner                               |
|         |        |                                                                                 |
|         |        +--[Fail < 20]--> Increment PublishAttempts, Backoff + Jitter                     |
|         |        |                                                                                 |
|         |        +--[Fail >= 20]--> Move to OutboxDeadLetters Sink                                 |
|         v                                                                                          |
|  [Redis Cache Invalidation & WebSocket Fanout] (Post-Commit Only)                                  |
+----------------------------------------------------------------------------------------------------+
```

#### How it works
1. **Atomic Insertion**: When an order is created or updated (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/order/repository_spanner.go:48-60`), the domain mutation and the serialized outbox event row are inserted within the same `spanner.ReadWriteTransaction`.
2. **Relay Daemon**: The outbox relay (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/outbox/relay.go:14-65`) runs as a dedicated goroutine on the worker tier:
   - Polling Frequency: `TickInterval = 250ms`.
   - Batch Processing: `BatchSize = 100`.
   - Broker Produce Timeout: `PublishTimeout = 10s` (prevents wedged brokers from stalling the drain loop).
   - In-Memory Backoff: Exponential backoff with random jitter from `BaseBackoff = 100ms` up to `MaxBackoff = 5s`.
   - Dead-Letter Handling: If an event fails across ticks and exhausts its budget (`MaxTotalAttempts = 20`), the relay writes the event into `OutboxDeadLetters` (`spanner.ddl:704-718`) and marks it removed from the live outbox.

#### Why it is there
Directly publishing to Kafka during an HTTP request handler creates the classic dual-write distributed transaction failure: if Kafka succeeds but the database transaction aborts, phantom events poison downstream consumers; if the database succeeds but Kafka times out, events are lost forever. The transactional outbox pattern guarantees **at-least-once delivery** with zero phantom events.

---

### 4.3. Categorized Domain Event Catalog

PegasusX defines over 100 domain events in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/events/events.go:33-360`, mirrored in `contracts/events.schema.json`:

1. **Order Lifecycle**:
   - `ORDER_CREATED`, `ORDER_STATUS_CHANGED`, `ORDER_VALIDATION_FAILED`, `ORDER_ASSIGNED`, `ORDER_REASSIGNED`, `ORDER_FINALIZED`, `ORDER_AMENDED`, `ORDER_ALLOCATED`, `PARENT_ORDER_CREATED`, `PARENT_ORDER_UPDATED`, `ORDER_FORCE_COMPLETED`.
2. **Manifests & Physical Logistics**:
   - `MANIFEST_DRAFT_CREATED`, `MANIFEST_LOADING_STARTED`, `MANIFEST_ORDER_INJECTED`, `MANIFEST_ORDER_EXCEPTION`, `MANIFEST_EXCEPTION_RESOLVED`, `MANIFEST_SEALED`, `MANIFEST_DISPATCHED`, `MANIFEST_COMPLETED`, `SPLIT_SHIPMENT_CREATED`, `ORDER_CAPACITY_OVERFLOW`.
3. **Last-Mile & Delivery Exceptions**:
   - `SHOP_CLOSED`, `SHOP_CLOSED_RESPONSE`, `SHOP_CLOSED_ESCALATED`, `SHOP_CLOSED_RESOLVED`, `SHOP_CLOSED_BYPASS_OFFLOAD`, `PROXIMITY_UNLOCKED`, `PARTIAL_OFFLOAD`, `CREDIT_LEAVE`, `CREDIT_DELIVERY_MARKED`.
4. **Finance, Credit & Soliq Fiscalization**:
   - `PAYMENT_REQUIRED`, `PAYMENT_CLEARED`, `PAYMENT_FAILED`, `SETTLEMENT_REQUIRED`, `REFUND_REQUESTED`, `REFUND_SUCCEEDED`, `FISCAL_RECEIPT_REQUESTED`, `FISCAL_RECEIPT_SUCCEEDED`, `FISCAL_RECEIPT_FAILED`, `BUYER_ACCEPTANCE_PENDING`, `BUYER_ACCEPTANCE_ACCEPTED`, `CASH_SHORTFALL`, `CASH_OVERAGE`, `AR_INVOICE_OPENED`, `AR_INVOICE_DUNNED`.
5. **AI Synthesis & S&OP Planning**:
   - `AI_RECOMMENDATION_CREATED`, `AI_RECOMMENDATION_DECIDED`, `PRE_ORDER_NOTIFIED`, `PRE_ORDER_CONFIRMED`, `REPLENISHMENT_AUTO_APPROVED`, `PLANNING_FORECAST_UPDATED`, `DEMAND_SIGNAL`.
6. **Warehouse & Manufacturing Transfers**:
   - `WAREHOUSE_TRANSFER_CREATED`, `WAREHOUSE_TRANSFER_RECEIVED`, `WMS_PUTAWAY`, `WMS_PICK_CONFIRMED`, `WMS_CYCLE_APPROVED`, `WMS_TEMPERATURE_BREACH`, `WAREHOUSE_SUPPLY_REQUEST_OPENED`, `FACTORY_SLA_BREACH`.

---

## 5. Multi-Region Cell Isolation Architecture (GS-C)

### 5.1. The Cell Doctrine

#### What it is
PegasusX employs a strict **Cell Isolation Architecture** (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/infra/terraform/cells/`, `infra/terraform/legacy/cell.tf`) designed to partition geographic and regulatory domains into independent, self-contained operational blast radiuses.

```
+----------------------------------------------------------------------------------------------------+
|                                MULTI-REGION CELL TOPOLOGY (GS-C)                                   |
+----------------------------------------------------------------------------------------------------+
|                                                                                                    |
|                                       [GLOBAL ROUTING PLANE]                                       |
|                                     (infra/terraform/global/)                                      |
|                               Anycast DNS & Home-Cell Session Router                               |
|                                                 |                                                  |
|                        +------------------------+------------------------+                         |
|                        |                                                 |                         |
|                        v                                                 v                         |
|       +---------------------------------+               +---------------------------------+        |
|       |        CELL UZ (Primary)        |               |        CELL EU (Secondary)      |        |
|       |    Region: me-central1 (Tashkent)|               |    Region: europe-west1         |        |
|       |    GCP Project: pegasus-503013  |               |    GCP Project: pegasusx-cell-eu|        |
|       |    State: pegasusx/ssmr         |               |    State: pegasusx/cell-eu      |        |
|       |    Soliq: MY_SOLIQ (Live EHF)   |               |    Fiscal: Commercial Receipt   |        |
|       |    Workload Identity: uz        |               |    Workload Identity: eu        |        |
|       |                                 |               |                                 |        |
|       |  [Spanner UZ]  [Redis]  [GKE]   |               |  [Spanner EU]  [Redis]  [GKE]   |        |
|       +---------------------------------+               +---------------------------------+        |
|                        ^                                                 ^                         |
|                        |=========== HARD ISOLATION BARRIER ==============|                         |
|                                   (scripts/assert_cell_backend.sh)                                 |
+----------------------------------------------------------------------------------------------------+
```

#### How it works
1. **Cell UZ (Primary Operating Cell)**:
   - Geography: Central Asia (`me-central1` / Tashkent).
   - GCP Project: `pegasus-503013`.
   - Terraform State Prefix: `pegasusx/ssmr` (`infra/terraform/cells/uz/backend.hcl:3`).
   - Fiscal Gateway: Uzbekistan State Tax Committee (`MY_SOLIQ`).
2. **Cell EU (Secondary Expansion Cell)**:
   - Geography: Europe (`europe-west1`).
   - GCP Project: `pegasusx-cell-eu`.
   - Terraform State Prefix: `pegasusx/cell-eu` (`infra/terraform/cells/eu/backend.hcl:5`).
   - Fiscal Gateway: Commercial Receipts (Soliq disabled: `FISCAL_ALLOW_COMMERCIAL_RECEIPTS=true`, `FISCAL_PROVIDER!=MY_SOLIQ` in `infra/k8s/overlays/cells/eu/kustomization.yaml:102-106`).
3. **Hard Isolation Barrier**:
   - Enforced by `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/assert_cell_backend.sh:1-156`.
   - Applying `europe-west1` is cryptographically and logically barred from opening `pegasusx/ssmr` state.
   - Project factory backend (`infra/terraform/cells/eu/project/backend.hcl`) is strictly isolated to `pegasusx/cell-eu-project`.
   - Workload Identity is dynamically bound to `local.k8s_namespace` rather than a hardcoded cluster string (`assert_cell_backend.sh:46-52`).
   - Foreign cells strictly forbid restoring Spanner backups originating from UZ (`cell.tf:122`, `scripts/cell_migrate.sh:127-129`).
4. **Global Plane (`infra/terraform/global/`)**:
   - Maintains anycast routing and edge certificates (`modules/global_dns/`, `modules/global_ar/`).
   - Users authenticate and receive a JWT containing their `home_cell` attribute, routing subsequent API traffic directly to their geographic cell ingress.

#### Why it is there
1. **Data Sovereignty & Legal Compliance**: Uzbekistan data protection laws mandate that citizen personal data and fiscal records reside on sovereign domestic infrastructure. Merging European and Uzbek operations into a single shared database would violate both GDPR and Uzbek data statutes.
2. **Blast Radius Containment**: An infrastructure outage, network partition, or database lock escalation in Cell EU cannot cascade into Cell UZ, ensuring 100% uptime for core physical distribution operations.
3. **Latency Optimization**: Keeping database writes, Redis caches, and WebSocket hubs in proximity to local warehouse docks reduces handheld scanner latency from hundreds of milliseconds to under 20ms.
