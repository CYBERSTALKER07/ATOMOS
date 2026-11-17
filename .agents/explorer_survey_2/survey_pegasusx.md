# Comprehensive Architectural & Code Survey: PegasusX Ecosystem

> **Target Repository**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX`  
> **Investigation Date**: 2026-09-26  
> **Investigation Mode**: Read-Only Exhaustive Code-Grounded Exploration  
> **Primary Ecosystem Archetype**: Single-Supplier Multi-Retailer (SSMR) FMCG Logistics & Enterprise Supply Chain Stack

---

## 1. Executive Summary & Ecosystem Mission

The **PegasusX** codebase represents an enterprise single-supplier logistics, distribution, and warehouse management platform designed for wholesale Fast-Moving Consumer Goods (FMCG) distribution in Central Asia (specifically Uzbekistan / Tashkent, expanding into multi-region cells).

Unlike multi-vendor marketplace architectures, PegasusX operates on the **Single Supplier Multi Retailer (SSMR)** doctrine (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/package.json:5`):
- One authoritative supplier organization distributes goods across a network of hundreds or thousands of retail grocery and convenience stores ("Retailers").
- The system manages the entire operational lifecycle: Supplier catalog, dynamic pricing rules, B2B orders, warehouse inventory with cold-chain and lot tracking, multi-stop delivery route optimization, physical truck loading and seal verification, driver turn-by-turn dispatch, electronic proof of delivery (EPOD), cash collections and credit leave-behind with Accounts Receivable (AR) dunning, and mandatory fiscal registration with the State Tax Committee of Uzbekistan (**Soliq** EHF electronic invoices).
- The platform strictly enforces **zero-theatre architecture**: real Spanner ReadWrite transactions, transactional outbox relays to Kafka, post-commit Redis cache invalidations, cross-pod WebSocket fanouts with source suppression, and automated CI gates preventing stubbed screens ("TODO: Inject"), fake images, mock data, and unverified route calls.

---

## 2. Monorepo Architecture & Dependency Manifests

PegasusX is organized as a multi-language polyglot monorepo containing Go microservices, TypeScript/React web portals and Tauri desktop apps, native Android (Kotlin) applications, native iOS (Swift) applications, and Python optimization solvers.

```
file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/
├── apps/                          # 22 Applications (Backend, AI Worker, Web, Desktop, Mobile, Sidecars)
│   ├── backend-go/                # Core Go HTTP API & background worker engine
│   ├── ai-worker/                 # Go event processor, planning ingest & predictive push
│   ├── handoff-service/           # Stateless delivery QR token validation service
│   ├── dispatch-optimizer-py/     # Python OR-Tools TSP / VRP pick path sidecar
│   ├── admin-portal/              # Next.js 15 platform governance console
│   ├── supplier-portal/           # Next.js 15 + Tauri 2 supplier control tower
│   ├── warehouse-portal/          # Next.js 15 + Tauri 2 warehouse operations portal
│   ├── factory-portal/            # Next.js 15 + Tauri 2 manufacturing & transfer portal
│   ├── retailer-app-desktop/      # Next.js 15 + Tauri 2 offline-capable POS & ordering
│   ├── payload-terminal/          # Expo 55 / React Native dock loading terminal
│   ├── driver-app-android/        # Kotlin + Jetpack Compose driver execution app
│   ├── driver-app-ios/            # Swift + SwiftUI native driver execution app
│   ├── retailer-app-android/      # Kotlin + Jetpack Compose retailer store app
│   ├── retailer-app-ios/          # Swift + SwiftUI native retailer store app
│   ├── supplier-app-android/      # Kotlin + Jetpack Compose mobile supplier app
│   ├── supplier-app-ios/          # Swift + SwiftUI native mobile supplier app
│   ├── warehouse-app-android/     # Kotlin + Jetpack Compose warehouse scanner app
│   ├── warehouse-app-ios/         # Swift + SwiftUI native warehouse scanner app
│   ├── factory-app-android/       # Kotlin + Jetpack Compose factory floor app
│   ├── factory-app-ios/           # Swift + SwiftUI native factory floor app
│   ├── payload-app-android/       # Kotlin + Jetpack Compose secure loading app
│   └── payload-app-ios/           # Swift + SwiftUI native secure loading app
├── contracts/                     # Formal schemas, OpenAPI specs, marker definitions
├── design-system/                 # Design tokens and shared visual components
├── infra/                         # Docker Compose, Kubernetes, Terraform, Redis config
├── packages/                      # 24 Shared libraries (TS, Go, Android Gradle, iOS SPM)
├── scripts/                       # 105 Automation, migration, QA, and validation scripts
├── sdk/                           # Partner integration SDKs in Go and TypeScript
├── services/                      # Standalone services (optimizer-core in C++ & Rust)
└── context/                       # Technology inventory, architecture graph, Cypher seeds
```

### Monorepo Workspaces & Toolchains

1. **Go Workspace (`go.work`)**  
   - Path: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/go.work:1-12`
   - Go Version: `1.25.0`
   - Active Modules:
     - `./apps/ai-worker`
     - `./apps/backend-go`
     - `./apps/handoff-service`
     - `./packages/config`
     - `./packages/handoff`
     - `./packages/optimizer-contract`
     - `./sdk/partner/go`

2. **PNPM Workspace (`pnpm-workspace.yaml` & root `package.json`)**  
   - Path: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/pnpm-workspace.yaml:1-23`
   - Path: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/package.json:1-47`
   - Package Manager: `pnpm@9.0.0`, Node `>=20`
   - Monorepo Portals & Packages:
     - Portals: `apps/supplier-portal`, `apps/admin-portal`, `apps/retailer-app-desktop`, `apps/warehouse-portal`, `apps/factory-portal`, `apps/payload-terminal`
     - Shared Packages: `packages/types`, `packages/api-core`, `packages/api-react`, `packages/validation`, `packages/i18n`, `packages/motion-tokens`, `packages/ui-kit`, `packages/pulse-ui`, `packages/explain-ui`, `packages/ws-refresh-contract`, `packages/desktop-bridge`, `packages/desktop-cache`, `packages/ui-maps`, `packages/ui-charts`

3. **Android Gradle Workspaces**  
   - Every one of the 6 Android apps (`driver-app-android`, `retailer-app-android`, `supplier-app-android`, `warehouse-app-android`, `factory-app-android`, `payload-app-android`) ships its own standalone Gradle wrapper (`gradlew`), Gradle properties, and Kotlin 1.9+/2.0+ DSL build files (`build.gradle.kts`, `settings.gradle.kts`).
   - Shared Android Gradle libraries: `packages/mobile-android-kit`, `packages/mobile-android-design`, `packages/mobile-android-barcode-scanner`.

4. **iOS Swift Package Manager & XcodeGen**  
   - iOS native apps use XcodeGen (`project.yml`) or native Xcode projects:
     - Driver: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/driver-app-ios/driverappios/driverappios.xcodeproj`
     - Retailer: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/retailer-app-ios/retailerapp/retailerapp.xcodeproj`
     - Supplier: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/supplier-app-ios/project.yml`
     - Warehouse: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/warehouse-app-ios/project.yml`
     - Factory: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/factory-app-ios/project.yml`
     - Payload: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/payload-app-ios/project.yml`
   - Shared iOS SPM Packages: `packages/mobile-ios-core` (`Package.swift`), `packages/mobile-ios-kit`, `packages/mobile-ios-design`, `packages/mobile-ios-barcode`.

5. **Rust & C++ Solvers**  
   - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/services/optimizer-core/server-rust/Cargo.toml`
   - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/services/optimizer-core/Dockerfile`

---

## 3. Backend Services & System Modules

### 3.1. `apps/backend-go` (Primary Operational Core)

- **Entry Point**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/main.go:1-493`
- **Architecture**: Domain-Driven Design (DDD) with clear separation between:
  - Transport Layer: Chi routers (`*routes` packages)
  - Domain Services: Business rules, invariants, validation (`order`, `payment`, `supplier`, etc.)
  - Persistence Layer: Google Cloud Spanner repositories wrapping `spanner.ReadWriteTransaction`
  - Event Outbox: Transactional outbox storage and background publisher
  - Realtime Layer: Goroutine-backed WebSocket hubs with Redis Pub/Sub cross-pod broadcasting

#### Dual Run-Mode Architecture (`PEGASUSX_RUN_MODE`)
Configured in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/bootstrap/run_mode.go` and executed in `main.go:106-127`:
1. `api`: Exclusively serves HTTP routes, WebSocket hubs, and proxy endpoints. Disables heavy background cron loops. Includes safety fallback: if no worker heartbeat is detected in Redis (`bootstrap.WorkerLive`), starts a local notification consumer (`main.go:119, 245-295`).
2. `worker`: Runs background event processing, Kafka consumers, outbox relays, reconcilers, and scheduled sweeps. Exposes health monitoring server on port 8081 without mounting public business routes.
3. `all`: Combined mode used for local development and Docker Compose SSMR sandbox.

#### Background Workers Tier (`apps/backend-go/runtime_workers.go`)
Defined in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/runtime_workers.go:19-229`:
1. **Outbox Relay**: `app.OutboxRelay.Start(ctx)` — Drains Spanner `OutboxEvents` table every 250ms and publishes to Kafka topics with exponential backoff and jitter (`line 24`).
2. **Outbox Supplier Backfill**: `outbox.StartSupplierIDBackfill(ctx, app.Spanner)` (`line 28`).
3. **Cache Invalidation Subscriber**: Subscribes to Redis Pub/Sub channel `cache:invalidate` for multi-pod cache coherence (`line 34`).
4. **Kafka Notification Consumer**: Consumes from Kafka for FCM push notifications and user inbox persistence (`line 38`).
5. **Domain Event Consumers**: `OrderEventConsumer`, `WarehouseEventConsumer`, `ClaimsEventConsumer`, `ReturnsEventConsumer` (`lines 41-56`).
6. **Warehouse Auto-Dispatch & Warmer**: `warehouse.StartAutoDispatchWorker` and `warehouse.StartDispatchPlanWarmer` (`lines 58-61`).
7. **Webhook Inbox Reconciler**: Replays and settles pending payment webhooks (`line 64`).
8. **Stuck Payment Reconciler**: `WebhookReconciler.ReconcileStuckSessions` — runs every 5 minutes with initial 30s jitter, querying payment gateways for sessions stuck >15 min (`lines 71-95`).
9. **Replenishment Engine Cron**: Automated inventory reordering (`line 97`).
10. **Factory Planning Cron**: Automated production scheduling (`line 101`).
11. **Labor Capacity Workers**: Driver scoring worker (daily) & capacity snapshot worker (hourly) (`lines 105-107`).
12. **Route Analytics Worker**: Nightly route performance aggregation (`line 110`).
13. **Order Saga Recovery Worker**: Sweeps pending order sagas every 15s (`line 114`).
14. **Cash Reconciliation Escalation Worker**: Nightly variance escalation (`line 122`).
15. **Reorder Suggestion Batch Worker**: Runs every 12 hours (`line 126`).
16. **Weather Ingestion Worker**: Fetches 14-day weather forecasts every 6h for demand sensing (`lines 129-148`).
17. **Demand Density Worker**: Aggregates spatial demand every 6 hours (`line 151`).
18. **Control Tower Playbook Worker**: Executes automated FMCG mitigation playbooks (`line 156`).
19. **Partner Integration Workers**: `PartnerEventConsumer`, `TwinEventConsumer` (digital twin), `PartnerWebhookDelivery`, `PartnerExportWorker`, `PartnerEdiInbound`, `PartnerEdiOutbound` (`lines 159-186`).
20. **Accounts Receivable (AR) Dunning Worker**: Hourly sweep sending progressive collection notifications (`line 188`).
21. **Buyer Acceptance Poller**: Polls Uzbekistan Soliq EHF buyer clearance (`line 193`).
22. **Auto-Confirm Preorders Sweeper**: Automatically confirms preorders whose lock deadline has arrived (`lines 197-213`).
23. **Retail OS Sweepers**: POS holds sweeper (expires holds after 24h), Assist SLA breach worker (checks response time every minute), Auto-order worker (`lines 215-223`).
24. **Factory SLA Breach Worker**: Every 5 minutes checks open supply requests and transit delays (`line 226`).

---

### 3.2. `apps/ai-worker` (Predictive & Ingest Processing)

- **Entry Point**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/ai-worker/main.go:1-438`
- **Core Functions**:
  1. **Kafka Event Processing**: Consumes `pegasusx-main` topic with group `pegasusx-ai-worker` (`line 242`).
  2. **AI Synthesis Engine** (`synthesis/`): Listens for `ORDER_CREATED`, `ORDER_COMPLETED`, `ORDER_STATUS_CHANGED`, `ORDER_DELIVERED`. Computes explainable replenishment recommendations, writes `AIPredictions` rows into Spanner, and emits `AI_RECOMMENDATION_CREATED` (`lines 412-427`).
  3. **Circuit Breaker**: `CircuitBreaker` pauses Kafka fetches for 15s if 5 consecutive errors occur (`lines 52-95, 314-353`).
  4. **Dynamic Freeze Registry**: Consumes `pegasusx-freeze-locks` topic to pause automated dispatch and AI rebalancing when manual dispatcher locks are placed on specific warehouses or orders (`lines 251-263, 362-379`).
  5. **Bulk Inventory Import Runtime**: Processes bulk CSV/Excel supplier catalog uploads from Google Cloud Storage or local file root, updating Spanner inventory in batches (`lines 286-296`).
  6. **Planning Ingest Runtime** (`planningingest/`): Consumes demand signals for baseline forecasting (`lines 298-304`).
  7. **Predictive Push Cron**: Dedicated cron entrypoint (`AI_WORKER_MODE=predictive-push-cron`) for retail reorder push suggestions (`line 227-233`).
  8. **Embedded Routing Solver Endpoint**: Mounts `contract.SolvePath` on port 8081 for internal routing optimization (`line 186`).
  9. **Prometheus Monitoring**: Exposes `/healthz`, `/ready`, and `/metrics` emitting `void_ai_worker_up`, `void_ai_worker_ready`, and per-partition `void_kafka_consumer_lag_seconds` (`lines 147-195`).

---

### 3.3. `apps/handoff-service` (Stateless Token Validation)

- **Entry Point**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/handoff-service/main.go:1-98`
- **Purpose**: Provides isolated, high-performance validation of delivery QR handoff tokens for drivers and retailers.
- **Endpoints**:
  - `POST /internal/v1/handoff/validate`: Validates presented driver/retailer QR token against stored token using HMAC/SHA-256 validation logic from `packages/handoff` (`line 43`).
  - `POST /internal/v1/handoff/public-token`: Generates current dynamic delivery token (`line 55`).

---

### 3.4. `services/optimizer-core` & `apps/dispatch-optimizer-py`

- **Protobuf Schema**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/services/optimizer-core/proto/optimizer_core.proto:1-123`
- **Solver Capabilities**:
  1. **Vehicle Routing Problem (VRP)**: Solves multi-vehicle, capacity-constrained routing with delivery time windows and distance matrix. Uses Google OR-Tools.
  2. **Constraint Programming SAT (CP-SAT)**: Solves factory dock scheduling and manifest capacity assignment.
  3. **Strict Honesty Enum**: Solver status explicitly differentiates `OPTIMAL`, `FEASIBLE`, `INFEASIBLE`, `MODEL_INVALID`, and `HEURISTIC` (`line 8-16`). Greedy/nearest-neighbor algorithms are strictly forbidden from reporting `OPTIMAL`.
- **Implementations**:
  - Python HTTP Sidecar (`apps/dispatch-optimizer-py/main.py:1-60`): Exposes `/optimize/pick-path` utilizing OR-Tools TSP solver with Euclidean distance matrix.
  - Rust Core (`services/optimizer-core/server-rust`): High-throughput VRP solver.
  - Python Contract Solver (`services/optimizer-core/server/contract_solver.py`): Reference implementation with certification harness.

---

## 4. Route Authority & API Catalog

The backend Chi router exposes over 411 endpoints divided into 10 strict Route Authorities (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/context/technology-inventory.json:66-77`):

| Route Authority | Mount Package | Primary Endpoints | Access Control & Protection |
|---|---|---|---|
| **Supplier Authority** | `apps/backend-go/supplierroutes/` | `/v1/auth/supplier/*`, `/v1/supplier/configure`, `/v1/supplier/profile`, `/v1/supplier/topology`, `/v1/supplier/org/*`, `/v1/supplier/fleet/*`, `/v1/supplier/pricing/*`, `/v1/supplier/inventory/*`, `/v1/supplier/orders/*`, `/v1/supplier/ai/recommendations` | JWT role `SUPPLIER` or `ADMIN`, tenant context enforced, scoped WebSocket token (`/v1/supplier/ws-session`) |
| **Retailer Authority** | `apps/backend-go/retailerroutes/` | `/v1/auth/retailer/*`, `/v1/retailer/profile`, `/v1/retailer/suppliers`, `/v1/retailer/cart/sync`, `/v1/retailers/{retailerID}/orders`, `/v1/retailer/tracking`, `/v1/retailer/pending-payments`, `/v1/retailer/active-fulfillment`, `/v1/retailer/orders/{confirm,reject}-ai`, `/v1/orders/{edit,confirm}-preorder` | JWT role `RETAILER`, optional Firebase ID token, claims validation |
| **Driver Authority** | `apps/backend-go/driverroutes/` & `deliveryroutes/` | `/v1/driver/profile`, `/v1/driver/history`, `/v1/driver/earnings`, `/v1/driver/availability`, `/v1/driver/pending-collections`, `/v1/driver/manifest-gate`, `/v1/driver/manifest`, `/v1/delivery/arrive` | JWT role `DRIVER`, claims-derived driver ID |
| **Order Authority** | `apps/backend-go/orderroutes/` | `/v1/order/create`, `/v1/order/{orderID}/status`, `/v1/orders/{orderID}/assign`, `/v1/order/deliver`, `/v1/order/confirm-offload`, `/v1/order/complete`, `/v1/order/collect-cash` | Role-gated mutations, ReadWriteTransaction with atomic outbox emission |
| **Warehouse Authority** | `apps/backend-go/warehouseroutes/` | `/v1/warehouse/ops/dashboard`, `/v1/warehouse/ops/inventory`, `/v1/warehouse/ops/orders`, `/v1/warehouse/ops/dispatch/preview`, `/v1/warehouse/demand/forecast`, `/v1/warehouse/supply-requests`, `/v1/warehouse/dispatch-lock` | JWT role `WAREHOUSE_ADMIN`, node ID scoping |
| **Factory Authority** | `apps/backend-go/factoryroutes/` | `/v1/factory/dashboard`, `/v1/factory/transfers/*`, `/v1/factory/manifests/*`, `/v1/factory/manifests/{id}/{start-loading,seal,dispatch,complete}`, `/v1/factory/manifests/rebalance` | JWT role `FACTORY_ADMIN`, node ID scoping |
| **Payload Authority** | `apps/backend-go/payloaderoutes/` | `/v1/payloader/trucks`, `/v1/payloader/manifests/*`, `/v1/payloader/manifests/{id}/inject-order`, `/v1/payloader/reassign-order`, `/v1/payload/seal`, `/v1/payloader/manifest-exceptions` | JWT role `PAYLOAD`, strict seal verification |
| **Payment Authority** | `apps/backend-go/paymentroutes/` | `/v1/checkout/b2b`, `/v1/checkout/unified`, `/v1/payment/chargeback`, `/v1/payment/chargeback/reversal`, `/v1/payment/ledger`, `/v1/payment/settlement/authority`, `/v1/payment/reconciliation/mismatches`, `/v1/payment/global_pay/initiate` | Idempotency middleware, client idempotency keys, provider-specific routers |
| **Webhook Authority** | `apps/backend-go/webhookroutes/` | `/v1/webhooks/global-pay`, `/v1/webhooks/adyen`, `/v1/webhooks/stripe` | Signature-first verification, raw body verification, no JWT required |
| **Telemetry Authority** | `apps/backend-go/telemetryroutes/` | `POST /v1/telemetry/location`, `GET /v1/fleet/route/{routeID}/geometry` | JWT role `DRIVER`, claims-derived identity, Redis point cache + throttled Kafka bus emit |

### OpenAPI Contracts
1. **Partner OpenAPI 3.0.3 (`contracts/partner.openapi.yaml:1-60`)**:
   - Machine-to-machine B2B integration surface (OAuth2 `client_credentials` + `PartnerApiKeys` prefix `pxk_*`).
   - Covers: Orders, Catalog Products, Prices, Stock, POS demand feed, Webhook secret rotation, EDI (CONTRL/APERAK/ORDRSP/INVOIC), SFTP host-key pinning, AS2 transport.
2. **Human JWT Core OpenAPI 3.0.3 (`contracts/jwt-core.openapi.yaml:1-60`)**:
   - Codegen-ready specification for the ~45 highest-traffic human user paths across all role portals.

---

## 5. Data Architecture & Storage Models

### Primary Database: Google Cloud Spanner
The authoritative ledger is defined in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/schema/spanner.ddl` (159KB, 220+ tables) accompanied by 125 incremental migration files (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/schema/migrations/`).

#### Core Data Entities

1. **Topology & Organizational Master Data**
   - `Suppliers` (`line 11`): `SupplierId`, `Name`, `Status`, `DefaultCurrency`, `CreatedAt`, `UpdatedAt`.
   - `SupplierProfiles` (`line 37`): Business registration, tax TIN, Soliq configuration, bank routing details.
   - `SupplierUsers` (`line 557`): Staff roster with role enum (`ADMIN`, `OPERATOR`, `FINANCE`, `VIEWER`).
   - `Retailers` (`line 145`): Storefront entity with H3 spatial index, credit limit, payment terms, Soliq TIN.
   - `Warehouses` (`line 437`): Distribution nodes with lat/lng, storage capacity, cold-chain capabilities.
   - `Factories` (`line 496`): Manufacturing and regional inbound hubs with loading dock capacities.
   - `WarehouseCoverageCells` (`line 476`): H3 resolution-9 delivery zones mapped to specific warehouses.

2. **Order & Parent Order Lifecycle**
   - `Orders` (`line 169`): Core transaction record. Stores:
     - `OrderId`, `SupplierId`, `RetailerId`, `WarehouseId`, `DriverId`, `VehicleId`, `RouteId`, `ManifestId`
     - `DeliveryToken` (dynamic HMAC token for proof of delivery)
     - `Status` (`PENDING`, `LOADED`, `IN_TRANSIT`, `ARRIVED`, `SHOP_CLOSED_PENDING`, `AWAITING_PAYMENT`, `PENDING_CASH_COLLECTION`, `DELIVERED_ON_CREDIT`, `FISCALIZING`, `FISCAL_FAILED`, `COMPLETED`, `CANCELLED`, `DELAYED`, etc.)
     - `OrderSource` (`MANUAL`, `MANUAL_PREORDER`, `AI_PREORDER`, `BACKORDER`)
     - `LineItemsJson` (JSON-encoded array of items, quantities, and pricing)
     - `TotalMinor`, `OriginalTotalMinor`, `Currency`, `H3Cell`, `Lat`, `Lng`
     - Fiscal tracking: `FiscalStatus`, `LatestFiscalReceiptId`, `FiscalizedAt`
     - Shop-closed handling: `ShopClosedAt`, `ShopClosedReason`, `ShopClosedGraceEndsAt`, `PartialDelivery`, `ProximityUnlockedAt`
     - Soliq EHF buyer clearance: `BuyerAcceptanceStatus`, `BuyerAcceptanceDeadline`
   - `ParentOrders` (`line 222`): Multi-supplier shopping cart rollup aggregate enabling retailers to checkout across multiple suppliers in one session.
   - `OrderDeliveryProofs` (`line 331`): Immutable EPOD evidence capturing QR verification hash, driver lat/lng, customer e-signature, and photos.

3. **Financial, Billing & AR Ledgers**
   - `PaymentSessions` (`line 576`): Checkout sessions with payment provider state machine.
   - `PaymentAttempts` (`line 595`): Detailed provider transaction attempts with gateway request IDs.
   - `PaymentLedgerEntries` (`line 660`): **Immutable double-entry transaction ledger**. Tracks every debit/credit (`PAYMENT_CAPTURED`, `CHARGEBACK_RECORDED`, `CHARGEBACK_REVERSED`, `SETTLEMENT_PROCESSED`).
   - `ARInvoices` (migration `20260807`): Accounts receivable open items with dynamic aging buckets (`CURRENT`, `1_30`, `31_60`, `61_90`, `90_PLUS`) and automated dunning progression.
   - `PayoutBatches` (migration `20260807`): Supplier payout disbursements with banking rail references.
   - `OrderFiscalReceipts` (migration `20260720`): ADR-009 fiscal receipts linking order payments to OFD fiscal memory and Soliq invoice IDs.

4. **Fleet, Manifests & Transport**
   - `Drivers` (`line 394`): Driver profile, license, availability state, current vehicle assignment.
   - `Vehicles` (`line 418`): Truck fleet with volume capacity (`CapacityVU`), weight limit, refrigeration type.
   - `SupplierTruckManifests` (`line 901`): Transport manifests representing a truck dispatch run. Stores `EncodedRoutePolyline` (OSRM route geometry), seal number, loading sequence, departure time.
   - `ManifestOrders` (`line 959`): Many-to-many relationship mapping orders to manifests with offload sequence.

5. **WMS & Inventory Control**
   - `Products` (`line 758`): Master SKU catalog with dimensions, unit volume (`VolumeVU`), barcode, excise tags.
   - `InventoryLevels` (`line 793`): Warehouse SKU balance with reserved, on-hand, and in-transit quantities.
   - `WMSStockLots` (migration `20260806`): Lot-level traceability with expiration dates and cold-chain temperature thresholds.
   - `WMSPickWaves` (migration `20260806`): Warehouse batch picking waves.

6. **Transactional Outbox & Governance**
   - `OutboxEvents` (`line 685`): Transactional outbox table (`EventId`, `AggregateType`, `AggregateId`, `EventType`, `PayloadJson`, `CreatedAt`, `PublishedAt`, `PublishTries`).
   - `OutboxDeadLetters` (`line 704`): Dead-letter store for un-publishable events after attempt budget exhaustion.
   - `PlatformAdminUsers` & `FeatureFlags` (migration `20260811`, `20260813`): Governance with dual-control review and mandatory MFA step-up.

---

## 6. Real-time Event Architecture & Messaging

### 6.1. Kafka Topics
Configured in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/events/events.go:10-30` and `infra/docker-compose.ssmr.yml:73-87`:
- `pegasusx-main`: Primary multi-event topic for all domain state transitions.
- `pegasusx-main-dlq`: Dead-letter queue for failed event consumers.
- `pegasusx-freeze-locks`: Realtime locks broadcast to AI workers to pause automated dispatch.
- `pegasusx-inventory-import`: Bulk CSV/Excel catalog ingestion pipeline.
- `pegasusx-demand`: Downstream POS sell-through demand signal feed (`STORE_POS` flywheel).
- `planning.signal.ingest.v1`, `planning.forecast.request.v1`, `planning.forecast.result.v1`: Machine learning planning pipeline.

### 6.2. Domain Event Catalog
Over 100 domain events categorized in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/events/events.go:33-360` and mirrored in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/contracts/events.schema.json` (141KB):
- **Order Lifecycle**: `ORDER_CREATED`, `ORDER_STATUS_CHANGED`, `ORDER_ASSIGNED`, `ORDER_REASSIGNED`, `ORDER_ALLOCATED`, `ORDER_FINALIZED`, `ORDER_FORCE_COMPLETED`, `ORDER_AMENDED`, `PARENT_ORDER_CREATED`.
- **Manifest & Logistics**: `MANIFEST_DRAFT_CREATED`, `MANIFEST_LOADING_STARTED`, `MANIFEST_SEALED`, `MANIFEST_DISPATCHED`, `MANIFEST_COMPLETED`, `MANIFEST_ORDER_INJECTED`, `MANIFEST_ORDER_EXCEPTION`, `MANIFEST_REBALANCED`, `SPLIT_SHIPMENT_CREATED`.
- **Last-Mile Exceptions**: `SHOP_CLOSED`, `SHOP_CLOSED_RESPONSE`, `SHOP_CLOSED_ESCALATED`, `SHOP_CLOSED_RESOLVED`, `PROXIMITY_UNLOCKED`, `PARTIAL_OFFLOAD`, `CREDIT_LEAVE`, `CREDIT_DELIVERY_MARKED`.
- **Finance & Fiscal**: `PAYMENT_REQUIRED`, `PAYMENT_CLEARED`, `PAYMENT_FAILED`, `SETTLEMENT_REQUIRED`, `REFUND_REQUESTED`, `REFUND_SUCCEEDED`, `FISCAL_RECEIPT_REQUESTED`, `FISCAL_RECEIPT_SUCCEEDED`, `FISCAL_RECEIPT_FAILED`, `BUYER_ACCEPTANCE_PENDING`, `BUYER_ACCEPTANCE_ACCEPTED`.
- **AI & Planning**: `AI_RECOMMENDATION_CREATED`, `AI_RECOMMENDATION_DECIDED`, `PRE_ORDER_NOTIFIED`, `PRE_ORDER_CONFIRMED`, `REPLENISHMENT_AUTO_APPROVED`, `PLANNING_FORECAST_UPDATED`.

### 6.3. WebSocket Hubs & Realtime Fanout
Implemented in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/ws/hub.go:1-60` and `main.go:417-433`:
- **8 Dedicated Hubs**:
  1. `RetailerHub` (room: `retailer:{retailer_id}`)
  2. `SupplierHub` (room: `supplier:{supplier_id}`)
  3. `DriverHub` (room: `driver:{driver_id}`)
  4. `PayloadHub` (room: `payload:{manifest_id}`)
  5. `WarehouseHub` (room: `warehouse:{warehouse_id}`)
  6. `FactoryHub` (room: `factory:{factory_id}`)
  7. `TelemetryHub` (rooms: `telemetry:driver:{driver_id}`, `telemetry:supplier:{supplier_id}`)
  8. `PlatformAdminHub` (room: `platform:admin`)
- **Cross-Pod Synchronization**: Uses Redis Pub/Sub channel `ws:<hub>:fanout` with JSON envelope `{source, room, payload}`.
- **Source Suppression**: Senders tag their pod instance ID so peer pods fan out locally while the originating pod ignores its own echo.
- **Reconnect Buffer**: 256-event in-memory ring buffer per hub room, enabling reconnection replay with monotonic sequence numbers.

---

## 7. Client & Surface Architecture

PegasusX provides 100% surface parity across the **6 Core Role-Rows** through responsive Next.js/Tauri web/desktop portals and native mobile apps:

```
+--------------------+--------------------------------+----------------------------+-----------------------------+
| Role Row           | Web / Desktop Application      | Native Android App         | Native iOS App              |
+--------------------+--------------------------------+----------------------------+-----------------------------+
| 1. SUPPLIER        | apps/supplier-portal           | apps/supplier-app-android  | apps/supplier-app-ios       |
| 2. RETAILER        | apps/retailer-app-desktop      | apps/retailer-app-android  | apps/retailer-app-ios       |
| 3. DRIVER          | (Uses Driver Mobile)           | apps/driver-app-android    | apps/driver-app-ios         |
| 4. WAREHOUSE       | apps/warehouse-portal          | apps/warehouse-app-android | apps/warehouse-app-ios      |
| 5. FACTORY         | apps/factory-portal            | apps/factory-app-android   | apps/factory-app-ios        |
| 6. PAYLOAD         | apps/payload-terminal (Expo)   | apps/payload-app-android   | apps/payload-app-ios        |
| 7. PLATFORM_ADMIN  | apps/admin-portal              | -                          | -                           |
+--------------------+--------------------------------+----------------------------+-----------------------------+
```

### Detailed Surface Breakdown

1. **Supplier Portal (`apps/supplier-portal`)**
   - Stack: Next.js 15, React 19, HeroUI, Tailwind CSS v4, Tauri 2 desktop packaging.
   - Key Modules:
     - Control Tower & Executive Analytics (`app/page.tsx`, `app/analytics/`)
     - Order Management & Vetting (`app/orders/`, `app/orders/vet/`)
     - Inventory Catalog & Stock Auditing (`app/inventory/`, `app/inventory/audit/`)
     - AI Recommendations & Human Decision Review (`app/ai/recommendations/`)
     - Org & Fleet Management (`app/org-fleet/`)
     - Live Fleet Map: Real-time driver telemetry using MapLibre GL (`@pegasusx/ui-maps`)
     - Finance, Billing & Treasury Operations (`app/payments/`, `app/earnings/`)
   - Direct-Backend Proxy: `app/api/[...path]/route.ts` proxies requests to `PUBLIC_BASE_URL` while preserving cookies and idempotency headers.

2. **Retailer Desktop & POS (`apps/retailer-app-desktop`)**
   - Stack: Next.js 15, React 19, HeroUI, Tauri 2 with SQLite plugin (`@tauri-apps/plugin-sql`).
   - Offline Capability: Uses `packages/desktop-cache` to store pending checkout orders, pending POS sales, and catalog caches locally during network interruptions.
   - Core Features: Catalog browsing, multi-supplier cart, preorder scheduling, live delivery order tracking with driver breadcrumb polylines, dispute evidence dossier viewer, and QR delivery token presentation.

3. **Warehouse Portal (`apps/warehouse-portal`)**
   - Stack: Next.js 15, React 19, HeroUI, Tauri 2.
   - Core Features: Inbound stock receiving, demand forecast inspection, pick-and-pack management, dispatch lock administration (`/v1/warehouse/dispatch-locks`), supply request creation to factories, and live fleet map.

4. **Factory Portal (`apps/factory-portal`)**
   - Stack: Next.js 15, React 19, HeroUI, Tauri 2.
   - Core Features: Production order tracking, inter-warehouse transfers, manifest creation and truck loading, seal verification, and supply QC inspection.

5. **Payload Terminal (`apps/payload-terminal`)**
   - Stack: Expo SDK 55, React Native 0.83, React 19, NativeWind v4.
   - Target: Touchscreen terminal mounted on warehouse loading docks or rugged handhelds.
   - Core Features: Camera barcode/QR scanning (`expo-camera`), truck dock check-in, order injection into manifests, tamper-evident seal recording, and manifest rebalance exception handling.

6. **Platform Admin Portal (`apps/admin-portal`)**
   - Stack: Next.js 15, React 19, HeroUI, Tailwind CSS v4.
   - Target: Platform governance, tenant lifecycle, KYC/KYB approval, feature-flag toggling with dual-control review and mandatory MFA step-up.

7. **Mobile Apps (Android & iOS)**
   - **Driver Apps** (`driver-app-android` & `driver-app-ios`): Turn-by-turn route geometry overlay from OSRM, background GPS telemetry emission (`POST /v1/telemetry/location`), QR code scanner for handoff verification, cash collection calculator with Tiyin integer precision and shortfall logging, offline proof-of-delivery queue.
   - **Retailer Apps** (`retailer-app-android` & `retailer-app-ios`): Mobile ordering, preorder confirmation, push notifications via FCM, and delivery tracking.
   - **Warehouse & Factory Apps**: Barcode scanning for lot numbers, cycle counts, pick confirmation, and pallet moves.

---

## 8. Shared Packages & Core Libraries

Located in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/packages/`:

### 8.1. TypeScript Shared Packages
1. **`@pegasusx/types`**: Canonical TypeScript type definitions re-exported from domain-specific modules (`src/supplier.ts`, `src/events.ts`, `src/envelope.ts`, `src/warehouse.ts`, `src/compliance.ts`, `src/claims.ts`, `src/auto-order.ts`, `src/admin.ts`).
2. **`@pegasusx/api-core`** (128KB client): Complete typed HTTP client wrapping `fetch` with automatic Bearer token injection, retry with jitter, structured `ProblemDetails` error parsing, and Idempotency-Key header injection.
3. **`@pegasusx/api-react`**: React hooks (`usePolling`, `useMarketPack`) for declarative data fetching and polling fallback.
4. **`@pegasusx/desktop-bridge`**: Tauri desktop native integration (file exports to CSV/Excel, native printing, deep linking, auto-updater integration).
5. **`@pegasusx/desktop-cache`**: SQLite-backed offline caching library for Tauri applications (`pending-pos-sales.ts`, `pending-checkout.ts`, `kv.ts`).
6. **`@pegasusx/ws-refresh-contract`**: Shared WebSocket and SSE event parsing library defining canonical event sets (`ORDER_STATUS_REFRESH_EVENTS`, `DISPATCH_REFRESH_EVENTS`, `PREORDER_REFRESH_EVENTS`).
7. **`@pegasusx/validation`**: Zod validation schemas for forms, checkout payloads, and API requests.
8. **`@pegasusx/i18n`**: Multi-language localization dictionary (Uzbek, Russian, English).
9. **`@pegasusx/ui-kit`**: Shared UI component library styled with Tailwind CSS v4.
10. **`@pegasusx/ui-maps`**: MapLibre GL wrapper for delivery routes, driver breadcrumb trails, and warehouse coverage cells.
11. **`@pegasusx/ui-charts`**: Recharts wrappers for revenue, order velocity, and inventory depletion graphs.
12. **`@pegasusx/pulse-ui` & `@pegasusx/explain-ui`**: Status banners, handoff verification cards, and AI prediction explanation chips.
13. **`@pegasusx/motion-tokens`**: Framer Motion animation configurations and easing curves.

### 8.2. Go Shared Packages
1. **`packages/config`**: Shared configuration helpers and runtime contract version.
2. **`packages/handoff`**: Delivery QR token generation and HMAC validation engine.
3. **`packages/optimizer-contract`**: Protobuf-backed Go types and HTTP client for `services/optimizer-core`.

### 8.3. Mobile Native Shared Packages
1. **Android Kotlin**: `mobile-android-kit`, `mobile-android-design`, `mobile-android-barcode-scanner`.
2. **iOS SPM**: `mobile-ios-core` (exposes `PegasusNetworking`, `PegasusUIKit`, `PegasusLiveActivities`), `mobile-ios-kit`, `mobile-ios-design`, `mobile-ios-barcode`.

---

## 9. Infrastructure, Orchestration & DevOps

### 9.1. Local Development & Sandbox (`docker-compose.ssmr.yml`)
Path: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/infra/docker-compose.ssmr.yml:1-223`
Provides a completely isolated, self-bootstrapping local replica of the entire backend:
- `spanner-emulator` (ports 9110:9010, 9120:9020)
- `redis:7-alpine` (port 6389:6379)
- `cp-zookeeper:7.5.0` (port 22181:2181)
- `cp-kafka:7.5.0` (port 9094:9094)
- `kafka-init`: Auto-provisions all 14 canonical Kafka topics with 3 partitions each
- `kafka-ui` (port 8083:8080)
- `backend-setup`: Applies Spanner DDL migrations sequentially
- `backend-go` (host port 8180 -> container port 8080)
- `ai-worker` (host port 8181 -> container port 8081)
- `optimizer-core` (host port 8182 -> container port 8082)
- Persistent Go module and build cache volumes (`pegasusx-ssmr-go-mod`, `pegasusx-ssmr-go-build`).

### 9.2. Kubernetes Production Orchestration (`infra/k8s/`)
Base manifests in `infra/k8s/base/` with 7 environment overlays (`prod`, `staging`, `pilot`, `dev`, `sandbox`, `ssmr`, `cells`):
- Deployment & Services: `backend-go` (HTTP + dedicated WebSocket service `service-ws.yaml`), `backend-go-worker`, `ai-worker`, `optimizer-core`.
- Ingress: Cloud Armor protected HTTPS Ingress with Google-managed SSL certificates and GCP BackendConfig.
- Autoscaling: HorizontalPodAutoscaler (`hpa.yaml`) and PodDisruptionBudgets (`pdb.yaml`).
- Automated CronJobs:
  - `billing_monthly_cronjob.yaml`: Monthly billing invoice generation
  - `planning_accuracy_cronjob.yaml`: Daily MAPE forecast accuracy evaluation
  - `planning_forecast_cronjob.yaml`: Demand forecasting recalculation
  - `planning_training_export_cronjob.yaml`: Training data export for ML pipelines
  - `predictive_push_cronjob.yaml`: Retailer predictive restocking notifications

### 9.3. Google Cloud Platform Terraform (`infra/terraform/`)
Modularized into 6 rollout phases (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/infra/terraform/main.tf:1-109`):
1. **Phase 1: Networking**: VPC, subnets, Cloud NAT with 2 static external egress IPs, Private Service Access (PSA) peering, firewall rules.
2. **Phase 2: Database**: Google Cloud Spanner regional instance with automated backup schedule + Memorystore Redis 7.0 HA.
3. **Phase 2: Messaging**: Google Managed Service for Apache Kafka with 10 canonical topics.
4. **Phase 2: Storage & Security**: 4 GCS Buckets (Media, Updates, Imports, Terraform State), Cloud Armor WAF security policy, Secret Manager secrets, IAM Workload Identity bindings.
5. **Phase 3: Compute**: GKE Autopilot / Standard Regional multi-zone Kubernetes cluster.
6. **Phase 4: Monitoring**: 12 Cloud Monitoring alert policies (outbox lag, fiscal success ratio, capture failure, worker crash), Slack/Email notification channels, Uptime probes.

### 9.4. Multi-Region Cell Architecture (GS-C)
Defined in `infra/terraform/cells/` and enforced by `scripts/assert_cell_backend.sh`:
- Cell UZ (`me-central1` / Tashkent): Primary SSMR operating cell.
- Cell EU (`europe-west1`): Isolated secondary cell.
- **Strict Isolation Guarantee**: EU cell state is completely isolated; Terraform state prefix `pegasusx/cell-eu` is strictly prevented from opening `pegasusx/ssmr` state. Workload identity namespaces are dynamically derived per cell.

### 9.5. Continuous Integration (`.github/workflows/ci.yml`)
The GitHub Actions workflow enforces 12 parallel jobs:
1. `backend-unit`: Go 1.25 unit tests
2. `backend`: Parity contracts (`make parity-contract`, `make parity-contract-full`), gap hunter gate, K8s manifest validation, image placeholder checks, TODO:Inject placeholder check, no-mock checks
3. `cell-isolation`: Proof of cell backend isolation + no unattended Terraform apply
4. `backend-lint`: `golangci-lint` v2.12.2 and `govulncheck`
5. `secrets`: `gitleaks` scanning with `.gitleaks.toml`
6. `backend-spanner`: Spanner emulator integration tests, schema drift gate, Phase-0 money-path gate, Phase-1 gate
7. `enterprise-gates`: Phase 2 through Phase 5c gates + analytics tenancy gate
8. `ai-worker`: Compilation and `go vet`
9. `android-apps`: Matrix compilation of all 6 Android apps via Gradle
10. `ios-apps`: Matrix build of all 6 iOS apps via XcodeGen and `xcodebuild`
11. `supplier-portal`: Vitest and TypeScript checks for all Tauri desktop clients (`retailer`, `supplier`, `warehouse`, `factory`) + updater validation
12. `admin-portal`: Typecheck and Next.js production build

---

## 10. Integrity, Quality & Honesty Enforcement

PegasusX has instituted a zero-tolerance policy against software "theatre" (fake mocks, dummy endpoints, orphaned UI screens, and ungrounded claims).

### The 5 Foundational Cross-Boundary Seams
Extracted by `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/extract_codegraph_seams.py`:
1. **Client API Call -> Backend HTTP Route**: Every client call (`ApiClientMethod`) maps to a real registered Chi route (`RouteEndpoint`) which maps to a real service method (`ServiceMethod`).
2. **Domain Service/Repo -> Spanner Table**: Every mutation runs inside a Spanner ReadWriteTransaction modifying real DDL tables.
3. **Transactional Outbox -> Domain Event**: State mutations atomically buffer `OutboxEvents` in the same transaction.
4. **Kafka Ingestion -> Downstream Consumer**: Outbox relay publishes to Kafka; consumer groups parse typed event envelopes.
5. **WebSocket Realtime Hub -> Role Rooms**: Notification dispatchers fan out typed events to scoped Redis rooms and connected client WebSockets.

### Active Automated Honesty Gates
1. **No "TODO: Inject" Placeholders** (`scripts/ci_fail_todo_inject.sh`): Fails CI if any client or backend file contains `TODO: Inject` — the signature of an orphaned screen or ViewModel that does nothing.
2. **No Placeholder Images in Production Overlays** (`scripts/ci_fail_placeholder_images.sh`): Refuses Kustomize manifests containing `IMAGE_PLACEHOLDER`, `:latest`, or `:local` tags. Ensures `optimizer-core` is never remapped to `backend-go`.
3. **No Mock Strings in Retailer Clients** (`scripts/ci_no_mock_control_tower.sh`): Fails if `Mock Data`, `hardcoded BarMark`, or `fakeH3Pulse` appear in retailer client code.
4. **Money Path Gate** (`scripts/money_path_gate.sh`): Proves against Spanner emulator that capture failures never write `CAPTURED`, duplicate idempotency keys never double-record, and shop-closed credit debt is always recorded.
5. **Schema Drift Gate** (`scripts/ci_schema_drift_gate.sh`): Asserts all production Spanner tables and columns match the application models.
6. **Full Route Parity Check** (`scripts/parity/role_row_contract_check_full.sh`): Parses all `/v1/...` strings in client applications and verifies every single one is mounted in `apps/backend-go`.

---

## 11. Recommendations for `pegasusX/agents.md`

Based on the ground truth of the PegasusX codebase, `pegasusX/agents.md` must be constructed according to the following directives:

### 11.1. Purpose & Identity
- Define PegasusX clearly: "PegasusX is the enterprise Single-Supplier Multi-Retailer (SSMR) FMCG logistics, distribution, and warehouse management stack."
- State its operational geography: Primary deployment in Central Asia (Uzbekistan / Tashkent region with Soliq fiscal integration), supporting cell-based multi-region expansion.

### 11.2. Architectural Conventions
1. **Atomic Outbox Rule**: Never perform a database mutation without writing the corresponding outbox event in the SAME `spanner.ReadWriteTransaction`. Direct Kafka publishing from HTTP request handlers is strictly forbidden.
2. **Post-Commit Invalidation Rule**: Redis cache invalidations and WebSocket broadcasts must ONLY occur AFTER the database transaction successfully commits.
3. **Fail-Open Realtime Fanout**: WebSocket broadcasts over Redis Pub/Sub must be fail-open. A Redis relay failure must never fail an HTTP mutation or crash a pod.
4. **Source Suppression**: Every WebSocket envelope must include its originating pod/client ID so subscribers avoid self-echo reflection.
5. **Integer Currency Rule**: All monetary values are integer minor units (e.g., Uzbek Tiyin, 1 UZS = 100 Tiyin). Floating-point currency calculations are strictly prohibited.
6. **Stateless Handoffs**: Driver delivery handoffs must use cryptographically secure HMAC QR tokens verified via `packages/handoff` or `handoff-service`.

### 11.3. Honesty & Anti-Theatre Rules
1. **No Fake Stubs / Mock Data**: Do not leave `TODO: Inject`, dummy data arrays, or unbacked endpoints in client or backend code.
2. **No Image Placeholders**: Production manifests must use immutable digest-pinned images.
3. **Solver Honesty**: Solvers must return `HEURISTIC` if using greedy approximations; never claim `OPTIMAL` unless proven by an exact solver (OR-Tools branch-and-bound/CP-SAT).
4. **Full Parity Contract**: If you add a frontend API method or route call, you MUST register and implement the corresponding backend route in `apps/backend-go` in the same batch.
5. **Schema Synchronization**: If an event type is added or modified, update all three sources of truth simultaneously:
   - `apps/backend-go/events/events.go`
   - `contracts/events.schema.json`
   - `packages/types/src/events.ts`

### 11.4. Development & Testing Workflows
- **Local Testing Sequence**:
  1. Start local stack: `make sandbox-infra-up` (or `make ssmr-infra-up`)
  2. Run unit tests: `make qa-gate`
  3. Run smoke and lifecycle gates: `make test-sandbox-infra`, `make test-sandbox-lifecycle`, `make test-sandbox-fiscal`
  4. Verify route parity: `make parity-contract-full`
  5. Run enterprise gates: `make ci-enterprise-gates`
- **Pre-deployment Verification**:
  - `make wire-ready`
  - `make p0-preflight`
  - `make validate-launch-readiness`

---

## 12. Complete Code-Grounded Evidence Index

| Component / Architectural Claim | Verified File Path | Key Line References |
|---|---|---|
| Monorepo Definition | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/package.json` | Lines 1–47 |
| Go Workspace Configuration | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/go.work` | Lines 1–12 |
| PNPM Workspace Manifest | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/pnpm-workspace.yaml` | Lines 1–23 |
| Makefile Operational Targets | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/Makefile` | Lines 1–413 |
| Backend Entry Point & Route Mounts | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/main.go` | Lines 77–493 |
| Background Workers Tier | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/runtime_workers.go` | Lines 19–229 |
| Spanner Schema Definition (220+ tables) | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/schema/spanner.ddl` | Lines 11, 169, 576, 685 |
| Spanner Migration History (125 files) | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/schema/migrations/` | Migration files 20250611–20260904 |
| Canonical Kafka Topics & Events | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/events/events.go` | Lines 10–30, 33–360 |
| AI Worker Entry Point & Processing | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/ai-worker/main.go` | Lines 147–195, 412–427 |
| QR Handoff Verification Service | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/handoff-service/main.go` | Lines 29–75 |
| Routing Optimization Protobuf Contract | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/services/optimizer-core/proto/optimizer_core.proto` | Lines 8–16, 22–65 |
| Python TSP Optimization Sidecar | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/dispatch-optimizer-py/main.py` | Lines 1–60 |
| Order Domain Aggregate & Service | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/order/service.go` | Lines 50–71, 150–210 |
| Order Spanner Persistence & Outbox | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/order/repository_spanner.go` | Lines 48–60, 120–180 |
| Payment Service & Ledgers | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/payment/service.go` | Lines 30–78, 120–210 |
| Transactional Outbox Relay | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/outbox/relay.go` | Lines 14–60 |
| WebSocket Hubs & Cross-Pod Relay | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/ws/hub.go` | Lines 1–60 |
| Accounts Receivable & Dunning | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/ar/service.go` | Lines 21–60 |
| Soliq Uzbekistan EHF Integration | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/soliq/client.go` | Lines 13–60 |
| Partner B2B OpenAPI 3.0.3 Contract | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/contracts/partner.openapi.yaml` | Lines 1–60 |
| Human JWT Core OpenAPI 3.0.3 Contract | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/contracts/jwt-core.openapi.yaml` | Lines 1–60 |
| Local SSMR Docker Compose | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/infra/docker-compose.ssmr.yml` | Lines 1–223 |
| CodeGraph Memgraph Platform | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/infra/docker-compose.codegraph.yml` | Lines 1–26 |
| Kubernetes Production Kustomize | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/infra/k8s/base/kustomization.yaml` | Lines 1–26 |
| GCP Terraform Modularized Stack | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/infra/terraform/main.tf` | Lines 1–109 |
| Multi-Region Cell Guard | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/assert_cell_backend.sh` | Lines 1–60 |
| Money Path Honesty Gate | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/money_path_gate.sh` | Lines 1–56 |
| Anti-Placeholder UI Gate | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/ci_fail_todo_inject.sh` | Lines 1–12 |
| Anti-Placeholder Image Gate | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/ci_fail_placeholder_images.sh` | Lines 1–44 |
| Anti-Mock String Gate | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/ci_no_mock_control_tower.sh` | Lines 1–15 |
| Full Client-Backend Parity Script | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/parity/role_row_contract_check_full.sh` | Lines 1–60 |
| 5 Cross-Boundary Seams Extractor | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/extract_codegraph_seams.py` | Lines 1–70 |
| GitHub Actions CI Pipeline | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/.github/workflows/ci.yml` | Lines 1–304 |
| Technology Inventory Ground Truth | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/context/technology-inventory.json` | Lines 1–431 |
| Architecture Graph Ledger | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/context/architecture-graph.json` | Lines 1–564 |
