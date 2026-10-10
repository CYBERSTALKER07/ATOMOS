# Pegasus Architectural Specification & System Blueprint

> **Ecosystem**: Pegasus Multi-Supplier Logistics Platform  
> **Source Path**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/ARCHITECTURE.md`  
> **Referenced Codebase**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus`

---

## 1. System Overview & Macro Dataflow

### 1.1 What It Is
The **Pegasus** platform is an enterprise-grade, hyperscale multi-tenant B2B supply chain, logistics execution engine, and predictive commerce operating system. Built to serve hundreds of thousands of retail shops, suppliers, regional warehouses, and manufacturing factories across Central Asia and international markets, the ecosystem coordinates the entire end-to-end commerce lifecycle:
1. **Predictive Commerce & Empathy Engine**: Machine-learning driven SKU-level purchase pattern forecasting, scheduled pre-order generation, and automated replenishment loops.
2. **Multi-Role Desktop & Mobile Operations**: Native and desktop portals tailored for 5 core personas: Suppliers, Warehouse Operators, Factory Managers, Retailers, and Fleet Drivers.
3. **Hyperscale Transactional Core**: A Go 1.25 core backed by Google Cloud Spanner (94 tables, multi-region `nam-eur-asia3` topology, H3 geospatial hexagonal indexing), Apache Kafka (KRaft mode, 8 event topics with 128 partitions), Redis Memorystore, and transactional outbox relays.
4. **Operations Research & Dispatch Optimization**: Vehicle Routing Problem (VRP) and Constraint Programming (CP-SAT) solvers running as high-throughput Rust sidecars (`optimizer-core`) and Python AI worker microservices.
5. **Multi-Gateway Payment & Double-Entry Treasury**: Immutable fee snapshots, degressive regional fee scheduling, multi-provider execution (Adyen, Global Pay, Airwallex, Payme, Click), and strict double-entry ledgering with anomaly detection.

### 1.2 How It Works
The architecture orchestrates information flow across three synchronized operational planes:
- **Transactional Control Plane (`apps/backend-go`)**: Evaluates authentication, enforces role-based access control, validates spatial proximity, runs state-machine transitions on orders and manifests, writes to Cloud Spanner, and stages outbox events.
- **Event & Telemetry Streaming Plane (`kafka`, `ws`)**: Distributed message brokers stream state events across 8 dedicated Kafka topics. 6 distinct WebSocket hubs broadcast push frames to connected web portals and mobile clients in real-time.
- **Optimization & Planning Plane (`apps/ai-worker`, `services/optimizer-core`)**: Consumes demand events, executes Clarke-Wright route generation, solves integer-scaled CP-SAT capacity constraints, and generates factory replenishment transfer orders.

```
       ┌────────────────────────────────────────────────────────┐
       │             Client Fleet (Web, Desktop, Mobile)        │
       │   Admin Portal | Factory Portal | Warehouse Portal     │
       │   Retailer Desktop/Mobile | Driver App | Scanner Bay   │
       └──────────────────────────┬─────────────────────────────┘
                                  │ HTTPS REST / WSS Events
                                  ▼
       ┌────────────────────────────────────────────────────────┐
       │             apps/backend-go (Chi Mux)                  │
       │  • Composition Root: bootstrap.NewApp()                │
       │  • 20+ Domain Subrouters (Auth, Order, Fleet, etc.)    │
       │  • Single-Flight Redis Cache Coalescing                │
       │  • Maglev Multi-Region Spanner Read Router             │
       └──────────────┬───────────────────────────┬─────────────┘
                      │ ReadWriteTransaction      │ Read-Replica
                      ▼                           ▼
       ┌──────────────────────────────┐    ┌────────────────────┐
       │   Cloud Spanner Primary DB   │    │ Regional Replicas  │
       │   (94 Canonical Tables)      │◄───┤ (Asia, EU, US)     │
       │   • Orders, Ledger, Manifests│    └────────────────────┘
       │   • OutboxEvents (Staged)    │
       └──────────────┬───────────────┘
                      │ Polled by outbox.Relay
                      ▼
       ┌────────────────────────────────────────────────────────┐
       │               Apache Kafka (KRaft Mode)                │
       │   8 Topics: logistics-events, demand-forecast, etc.    │
       └──────┬───────────────────────┬───────────────────┬─────┘
              │                       │                   │
              ▼                       ▼                   ▼
    ┌──────────────────┐    ┌───────────────────┐  ┌────────────┐
    │  apps/ai-worker  │    │  optimizer-core   │  │ WebSockets │
    │  • Empathy Engine│    │  • Rust Sidecar   │  │ • 6 Hubs   │
    │  • Clarke-Wright │    │  • CP-SAT / VRP   │  │ • Realtime │
    │  • Bulk Importer │    │  • Go Adapter     │  │   Push     │
    └──────────────────┘    └───────────────────┘  └────────────┘
```

### 1.3 Why It Is There
Traditional supply chains in emerging markets suffer from severe information asymmetry: retail shops order via paper or uncoordinated phone calls, leading to stockouts, inventory obsolescence, and inefficient multi-stop delivery routes. Pegasus establishes a unified digital spine that synchronizes demand forecasting, inventory visibility, dispatch routing, and automated financial settlements into an immutable, mathematically verified execution loop.

---

## 2. Monorepo Topology & Code Organization

### 2.1 What It Is
The Pegasus repository (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus`) is organized as a high-density polyglot monorepo containing Go microservices, TypeScript/React web portals, Tauri desktop wrappers, Rust optimization sidecars, Python deep agent tooling, Kotlin/Jetpack Compose Android applications, and Swift/SwiftUI iOS applications.

### 2.2 How It Works
The repository is partitioned into 5 key architectural directories:
- **`apps/` (18 Subsystems)**:
  - `apps/backend-go` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go`): Core enterprise API backend (Go 1.25, Chi, Spanner, Kafka).
  - `apps/ai-worker` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/ai-worker`): Predictive preorder forecasting and Clarke-Wright heuristics.
  - `apps/admin-portal` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/admin-portal`): Next.js 15, React 19, Tauri 2, HeroUI 3 desktop & web app for suppliers and global operators.
  - `apps/factory-portal` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/factory-portal`): Next.js 15 / Tauri 2 portal for factory floor scheduling.
  - `apps/warehouse-portal` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-portal`): Next.js 15 / Tauri 2 portal for warehouse intake and dispatch locks.
  - `apps/retailer-app-desktop` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-desktop`): Next.js 15 / Tauri 2 point-of-sale wholesale ordering terminal.
  - `apps/payload-terminal` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/payload-terminal`): Expo 55 / React Native 0.83 dock barcode terminal.
  - Android Fleet (Kotlin 2.x, Jetpack Compose, Room): `driver-app-android` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/driver-app-android`), `retailer-app-android` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-android`), `factory-app-android` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/factory-app-android`), `warehouse-app-android` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-app-android`), `payload-app-android` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/payload-app-android`).
  - iOS Fleet (Swift 6, SwiftUI, SwiftData): `driverappios` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/driverappios`), `retailer-app-ios` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-ios`), `factory-app-ios` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/factory-app-ios`), `warehouse-app-ios` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-app-ios`), `payload-app-ios` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/payload-app-ios`).
  - `apps/synthetic-tester` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/synthetic-tester`): High-throughput load generator.
- **`services/` (2 Autonomous Engines)**:
  - `services/optimizer-core` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/optimizer-core`): Rust gRPC solver daemon (`server-rust`) and Go Kafka adapter (`adapters/go`).
  - `services/deep-agents` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/deep-agents`): LangGraph autonomous multi-agent reasoning fleet.
- **`packages/` (8 Shared Packages)**:
  - `packages/ai-bridge` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/ai-bridge`): Gemini AI bridge.
  - `packages/api-client` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/api-client`): Shared TypeScript HTTP client.
  - `packages/config` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/config`): Fail-closed Go configuration validator.
  - `packages/i18n` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/i18n`): Multilingual localization engine.
  - `packages/optimizer-contract` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/optimizer-contract`): Solver request/response models.
  - `packages/types` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/types`): Shared TypeScript interfaces and WebSocket protocols.
  - `packages/ui-kit` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/ui-kit`): Material 3 design tokens.
  - `packages/validation` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/validation`): Shared schema validation.
- **`infra/` (Cloud & Provisioning)**:
  - `infra/terraform` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/infra/terraform`): Multi-region Google Cloud IaC.
  - `infra/k8s` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/infra/k8s`): Kubernetes deployments, KEDA scalers, and Prometheus alerting rules.
  - `infra/spanner` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/infra/spanner`): Spanner schema definitions.
- **`scripts/` (Automated Governance)**:
  - Enterprise gates for contract drift, boundary enforcement, design tokens, and security audits (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/scripts`).

### 2.3 Why It Is There
A single monorepo guarantees that cross-cutting domain changes (such as modifying order state machine states, introducing new WebSocket frames, or changing Spanner DDL definitions) can be verified atomically across backend services, solvers, shared contracts, and client portals simultaneously without multi-repository version fragmentation.

---

## 3. Cloud Spanner Schema Architecture (94 Tables)

### 3.1 What It Is
The core datastore is Google Cloud Spanner, defined in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/schema/spanner.ddl`. Spanner is chosen for its external consistency (TrueTime API), synchronous multi-region cross-datacenter replication, and horizontally scalable ACID transactions. The schema contains exactly 94 canonical tables.

### 3.2 How It Works: Complete Functional Catalog

The 94 tables are grouped into 10 cohesive relational subsystems:

#### 1. Core Logistics, Orders & Fleet Execution (10 Tables)
- `Retailers` (`spanner.ddl:15`): Merchant master record, store coordinates, H3 index, operating hours, credit limits.
- `Drivers` (`spanner.ddl:50`): Delivery personnel profile, vehicle binding, license status, active shift.
- `Vehicles` (`spanner.ddl:74`): Commercial fleet trucks, volume capacity (VU), weight limits, refrigeration capabilities.
- `Orders` (`spanner.ddl:103`): Central order aggregate with CHECK constraint on states (`PENDING`, `LOADED`, `IN_TRANSIT`, `ARRIVING`, `ARRIVED`, `COMPLETED`, `CANCELLED`, `QUARANTINE`).
- `OrderLineItems` (`spanner.ddl:275`): SKU-level order details, requested quantity, offloaded quantity, unit price.
- `DeliverySessions` (`spanner.ddl:897`): Real-time driver-to-shop offload handshake, geofence validations, QR token match.
- `DeliverySessionAdjustments` (`spanner.ddl:945`): Downward doorstep price and quantity adjustments.
- `ScheduledJobs` (`spanner.ddl:1025`): Temporal background task scheduling metadata.
- `DriverTelemetry` (`spanner.ddl:2256`): Ingested GPS points, bearing, speed, and battery percentage.
- `GlobalPins` (`spanner.ddl:2296`): Spatial map points of interest and geographic pin markers.

#### 2. Financial, Treasury & Double-Entry Ledger (10 Tables)
- `MasterInvoices` (`spanner.ddl:140`): Immutable fiscal invoice headers, total gross, net payout, cash custody status.
- `SupplierPayoutPolicies` (`spanner.ddl:171`): Tenant-specific payout distribution rules (`HQ_SUPPLIER` vs `WAREHOUSE_LOCAL`).
- `InvoiceSettlementSlices` (`spanner.ddl:188`): Exact fee breakdown snapshots per invoice, platform fee basis points, net warehouse slices.
- `LedgerEntries` (`spanner.ddl:290`): Immutable double-entry debit/credit ledger rows with strict `(OrderId, EntryType)` uniqueness index.
- `LedgerAnomalies` (`spanner.ddl:313`): Reconciliation discrepancies detected by the hourly audit cron.
- `Refunds` (`spanner.ddl:1659`): Reverse payment tracking for downward amendments, cancellations, and stale order returns.
- `SupplierGlobalPayntConfigs` (`spanner.ddl:753`): Merchant gateway configuration for Global Pay.
- `GlobalPayntSessions` (`spanner.ddl:774`): Checkout session states for electronic payment intents.
- `GlobalPayntAttempts` (`spanner.ddl:814`): Detailed gateway attempt logs, raw response codes, and network latency.
- `RetailerCardTokens` (`spanner.ddl:1047`): PCI-compliant tokenized card references for recurring 1-click B2B wholesale reorders.

#### 3. Product Catalog & Dynamic Pricing (6 Tables)
- `Products` (`spanner.ddl:225`): Universal master product catalog and barcode definitions.
- `SupplierProducts` (`spanner.ddl:235`): Supplier-specific SKU definitions, packaging step-sizes, and tiered base pricing.
- `Categories` (`spanner.ddl:429`): Hierarchical catalog taxonomy.
- `PlatformCategories` (`spanner.ddl:487`): Standard platform category classification.
- `RetailerPricingOverrides` (`spanner.ddl:1901`): Dynamic B2B custom price overrides per retailer-supplier pair.
- `PricingAuditLog` (`spanner.ddl:2188`): Historical audit trail of all manual and algorithmic price updates.

#### 4. Inventory, Warehouse Staging & Bulk Ingestion (10 Tables)
- `SupplierInventory` (`spanner.ddl:510`): Legacy supplier inventory table.
- `SupplierInventoryV2` (`spanner.ddl:523`): High-concurrency inventory ledger keyed by `(SupplierId, WarehouseId, SkuId)`.
- `InventoryAuditLog` (`spanner.ddl:543`): Immutable stock movement log tracking intake, reservations, and dispatch offloads.
- `InventoryImportSessions` (`spanner.ddl:560`): Bulk file upload session tracker.
- `InventoryImportRows` (`spanner.ddl:592`): Raw uploaded rows during catalog spreadsheet ingestion.
- `SupplierImportSessions` (`spanner.ddl:634`): Supplier import batch manager.
- `SupplierImportStagedRows` (`spanner.ddl:656`): AI-mapped staged rows awaiting validation.
- `SupplierImportMapping` (`spanner.ddl:672`): Gemini-generated zero-shot header column mappings.
- `SupplierImportAnalyticsFacts` (`spanner.ddl:684`): Summary metrics extracted from bulk catalog imports.
- `SupplierReturns` (`spanner.ddl:708`): Physical return manifest for damaged or rejected goods.

#### 5. Retailer Settings, AI Predictions & Cart (11 Tables)
- `RetailerGlobalSettings` (`spanner.ddl:361`): Merchant-level automated ordering toggles and preferred delivery windows.
- `RetailerSupplierSettings` (`spanner.ddl:368`): Per-supplier auto-order preferences.
- `RetailerProductSettings` (`spanner.ddl:376`): SKU-level minimum and maximum inventory limits.
- `RetailerVariantSettings` (`spanner.ddl:386`): Variant packaging rules.
- `AIPredictions` (`spanner.ddl:399`): Predicted reorder headers generated by the Empathy Engine.
- `AIPredictionItems` (`spanner.ddl:413`): SKU recommendations interleaved within `AIPredictions`.
- `CorrectionWeights` (`spanner.ddl:1645`): RLHF human feedback weights adjusting algorithm predictions after retailer edits.
- `RetailerCarts` (`spanner.ddl:1010`): Persistent cross-device draft shopping carts.
- `RetailerFamilyMembers` (`spanner.ddl:1615`): Authorized store clerks eligible to receive deliveries and sign offloads.
- `RetailerLoyaltyTiers` (`spanner.ddl:2351`): Retailer turnover loyalty brackets.
- `RetailerRatings` (`spanner.ddl:2362`): Driver and service quality feedback scores.

#### 6. Multi-Facility Supply Chain & Manufacturing (15 Tables)
- `Suppliers` (`spanner.ddl:450`): Top-level wholesale supplier entity.
- `Admins` (`spanner.ddl:495`): System and tenant administrator profiles.
- `SupplierUsers` (`spanner.ddl:1123`): Staff memberships and role permissions.
- `Warehouses` (`spanner.ddl:1097`): Regional distribution centers with H3 coverage polygon arrays.
- `WarehouseStaff` (`spanner.ddl:725`): Warehouse operator credentials and bay assignments.
- `Factories` (`spanner.ddl:1173`): Manufacturing plants supplying regional warehouses.
- `FactoryStaff` (`spanner.ddl:1193`): Factory shift supervisors and loading bay personnel.
- `InternalTransferOrders` (`spanner.ddl:1214`): Inter-facility stock transfers (`DRAFT` → `DISPATCHED` → `RECEIVED`).
- `InternalTransferItems` (`spanner.ddl:1236`): Line items for stock moving between factory and warehouse.
- `FactoryTruckManifests` (`spanner.ddl:1248`): Bulk transfer line-haul truck routes.
- `ReplenishmentInsights` (`spanner.ddl:1269`): Algorithmic stockout warnings and production order suggestions.
- `SupplyRequests` (`spanner.ddl:1712`): Warehouse stock replenishment requests sent to factories.
- `FactorySupplyRequestQC` (`spanner.ddl:1738`): Quality control inspection logs on outgoing factory pallets.
- `SupplyRequestItems` (`spanner.ddl:1749`): SKU requirements within warehouse supply requests.
- `DispatchLocks` (`spanner.ddl:1762`): Mutual exclusion locks halting dispatches during warehouse cycle counts.

#### 7. Replenishment Graph & Network Optimization (8 Tables)
- `SupplyLanes` (`spanner.ddl:1986`): Directed manufacturing lanes connecting specific factories to warehouses.
- `ReplenishmentLocks` (`spanner.ddl:2012`): Algorithmic locks preventing duplicate replenishment triggers.
- `FactorySLAEvents` (`spanner.ddl:2025`): Transit delay and factory loading bottleneck audit log.
- `NetworkOptimizationMode` (`spanner.ddl:2045`): Global routing policy switch (`COST_MINIMIZATION` vs `SLA_MAXIMIZATION`).
- `PullMatrixRuns` (`spanner.ddl:2053`): Execution logs of 4-hour pull matrix inventory calculations.
- `SupplierOverrides` (`spanner.ddl:2080`): Production lane manual overrides.
- `SupplierRetailerClients` (`spanner.ddl:1928`): Approved supplier-retailer business relations.
- `DeliveryZones` (`spanner.ddl:1958`): Geographic service polygons defined as H3 cell collections.

#### 8. Regional Enterprise & Billing Metering (8 Tables)
- `CountryConfigs` (`spanner.ddl:1364`): Sovereign operational rules (currency, tax rules, date formats).
- `SupplierCountryOverrides` (`spanner.ddl:1392`): Country-specific supplier configurations.
- `Regions` (`spanner.ddl:1418`): Administrative geographic divisions.
- `RegionalConfigs` (`spanner.ddl:1435`): Degressive fee tier schedules and default platform fee basis points.
- `SystemConfig` (`spanner.ddl:838`): Dynamic feature flags and circuit breaker thresholds.
- `BillingMeterEvents` (`spanner.ddl:847`): Metered platform usage events (orders processed, solver seconds).
- `BillingSupplierMeters` (`spanner.ddl:865`): Aggregated monthly billing counters per supplier.
- `BillingGlobalMeters` (`spanner.ddl:881`): Platform-wide consumption totals.

#### 9. Auditing, Edge Protocols & Security (7 Tables)
- `OrderEvents` (`spanner.ddl:1469`): Immutable event stream of order state transitions with actor metadata.
- `ShopClosedAttempts` (`spanner.ddl:1495`): Driver arrival records when retail shop is shuttered.
- `DeviceFingerprints` (`spanner.ddl:1522`): Hardware and client device IDs for Edge 24 single-device enforcement.
- `NegotiationProposals` (`spanner.ddl:1595`): Doorstep bargaining proposals submitted by drivers.
- `AuditLog` (`spanner.ddl:995`): General security audit trail for administrative mutations.
- `OrderActivityEvents` (`spanner.ddl:2334`): High-resolution order interaction telemetry.
- `DispatchAudit` (`spanner.ddl:2107`): Historical record of fleet dispatch decisions and vehicle assignments.

#### 10. Transactional Outbox, Jobs & Communication (9 Tables)
- `OutboxEvents` (`spanner.ddl:2151`): Staged Kafka events written atomically within business transactions.
- `OutboxDLQ` (`spanner.ddl:2171`): Dead letter queue for events that permanently fail Kafka publication.
- `OptimizationJobs` (`spanner.ddl:2215`): Asynchronous VRP and CP-SAT solver job queue ledger.
- `SupplierTruckManifests` (`spanner.ddl:1792`): Outbound delivery manifests grouping multiple customer orders.
- `ManifestOrders` (`spanner.ddl:1823`): Association table linking specific orders to truck manifests.
- `ManifestExceptions` (`spanner.ddl:1840`): Missing or damaged item exceptions recorded at loading bay.
- `DeviceTokens` (`spanner.ddl:966`): FCM and APNS push notification device tokens.
- `Notifications` (`spanner.ddl:979`): In-app notification inbox records.
- `RetailerSuppliers` (`spanner.ddl:441`): Many-to-many relationship mapping retail stores to authorized suppliers.

### 3.3 Why It Is There
Spanner's synchronous Paxos-based replication ensures zero data loss (RPO = 0) and instantaneous failover (RTO < 5s) across regions. All monetary columns are explicitly defined as `INT64` (e.g. `Amount INT64 NOT NULL`), guaranteeing mathematical integrity and eliminating floating-point IEEE-754 calculation drifts.

---

## 4. Transactional Outbox Pattern (V.O.I.D. Phase VII)

### 4.1 What It Is
The Transactional Outbox pattern guarantees that state changes in Cloud Spanner and domain event publications to Apache Kafka occur with atomic consistency. It is implemented in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/outbox/relay.go`.

### 4.2 How It Works
1. **Atomic Write**: When an HTTP handler mutates database state (e.g. creating an order or updating a delivery status), it writes the entity mutations AND an `OutboxEvents` row inside the exact same Spanner `ReadWriteTransaction`:
   ```sql
   INSERT INTO OutboxEvents (EventId, AggregateId, EventType, Topic, Payload, Status, CreatedAt)
   VALUES (@eventId, @aggregateId, @eventType, @topic, @payload, 'PENDING', PENDING_COMMIT_TIMESTAMP());
   ```
2. **Sharded Background Relay**: The `outbox.Relay` daemon (`outbox/relay.go:58`) polls `OutboxEvents` where `Status = 'PENDING'` at regular intervals (default 2s, batch size 100).
3. **Partition-Preserving Sharding**: Incoming events are distributed across parallel publisher worker goroutines using `FNV32(AggregateID) % numShards` (`outbox/relay.go:27`). Because the same `AggregateID` (e.g. `OrderId` or `SupplierId`) is always assigned to the same worker shard, strict per-entity sequential ordering in Kafka is guaranteed.
4. **Batch Mark-Published**: After workers successfully publish messages to Kafka via `goKafka.Writer`, all completed `EventId`s from the tick are updated in Spanner within a single `Apply` RPC mutation (`outbox/relay.go:33`), setting `Status = 'PUBLISHED'` and recording `PublishedAt`.
5. **Dead-Letter Routing & Real-Time Alerting**: On permanent failure, rows transition to `OutboxDLQ` (`spanner.ddl:2171`), and the `FailureCallback` (`outbox/relay.go:38`) immediately broadcasts an `OUTBOX_FAILED` frame across `WarehouseHub` and `FactoryHub` (`bootstrap/app.go:679`).

### 4.3 Why It Is There
Directly calling Kafka producers within HTTP request handlers introduces the classic dual-write vulnerability: if the Kafka publish succeeds but the database commit fails, phantom events are broadcast; conversely, if the database commits but the network to Kafka drops, downstream workers never learn of the state change. The Transactional Outbox eliminates both failure modes completely without distributed 2PC locking overhead.

---

## 5. Single-Flight Redis Cache Coalescing & Backpressure

### 5.1 What It Is
To protect Cloud Spanner from high-frequency read spikes, Pegasus implements a multi-tiered caching architecture combining Redis Memorystore, an in-memory L1 cache, and Go `singleflight` request deduplication (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/cache/cache.go` and `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/bootstrap/app.go:74`).

### 5.2 How It Works
1. **Cache Flight Deduplication**: High-traffic read operations—such as retrieving supplier profiles (`supplier/registration.go:482`), retailer profiles (`supplier/discovery.go:426`), driver credentials (`supplier/fleet.go:839`), or factory profiles (`factory/crud.go:101`)—wrap their cache-miss queries in `singleflight.Group.Do`:
   ```go
   val, err, _ := flight.Do(cacheKey, func() (interface{}, error) {
       // Check Redis L2
       // On miss: Execute Spanner Read
       // Populate Redis L2
       return data, nil
   })
   ```
   If 1,000 retail apps simultaneously request the catalog for the same supplier, only 1 query hits Cloud Spanner; the remaining 999 callers await the single in-flight resolution and share the identical result in-memory.
2. **L1 In-Process Tier with Invalidation Hooks**: Local in-memory caches handle ultra-fast microsecond lookups. Invalidation signals published to Redis trigger `l1Evict` (`cache/cache.go:26`) across all backend pods in the fleet.
3. **Adaptive Backpressure Engine**: `cache.BackpressureEngine` (`cache/backpressure.go`) continuously monitors CPU, memory, and Spanner query latency. Under severe database load, non-essential background traffic and lower-priority reporting requests are gracefully shed via `PriorityGuard` middleware (`bootstrap/app.go:158`).

### 5.3 Why It Is There
Spanner query nodes are optimized for transactional throughput and consistency, not serving millions of repetitive static profile queries. Coalescing concurrent cache misses eliminates the "thundering herd" (dogpiling) problem during peak morning order-placement windows.

---

## 6. Multi-Region Spanner Read Routing (Maglev Architecture)

### 6.1 What It Is
When running multi-continental deployments (`enable_multiregion=true`), Pegasus routes read-only database queries to the nearest geographic replica based on the entity's **Uber H3** hexagonal spatial index. The routing engine is implemented in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/bootstrap/spannerrouter/router.go`.

### 6.2 How It Works
1. **Zero-Allocation Static Table**: At package `init()` (`spannerrouter/router.go:42`), a static mapping table `regionCells` is constructed from sample urban points across Central Asia (`asia`), Europe (`eu`), and North/South America (`us`). Each sample coordinate is converted to an H3 resolution-2 cell (~90,000 km² macro-cell).
2. **O(1) Spatial Region Resolution**: When a read query is issued for an entity with an H3 resolution-7 index (e.g. an order or warehouse location), `cellToRegion` (`spannerrouter/router.go:193`) masks the cell to its resolution-2 parent via `c.Parent(2)` (a 50-nanosecond bit-shift operation) and looks up the region in `regionCells`.
3. **Replica Selection**:
   - `Router.For(h3Cell)` (`spannerrouter/router.go:166`): Returns the regional read-replica client for `asia`, `eu`, or `us`. Unknown cells safely fall back to the primary client.
   - `Router.Primary()` (`spannerrouter/router.go:187`): Always returns the primary write client.
4. **Strict Isolation of Writes**: All write mutations (`client.ReadWriteTransaction`), multi-region dashboard aggregations, and the transactional outbox relay MUST strictly use `Primary()`.

```
                    H3 Res-7 Location Index
                              │
                              ▼
                   Bitmask Parent to Res-2 (~50 ns)
                              │
                              ▼
                regionCells[Parent] Lookup Table (O(1))
                              │
         ┌────────────────────┼────────────────────┐
         │                    │                    │
    "asia" (Uzbekistan)   "eu" (Europe)       "us" (Americas)
         │                    │                    │
         ▼                    ▼                    ▼
   Spanner Replica      Spanner Replica      Spanner Replica
   (asia-south1)        (europe-west1)       (us-central1)
   ~10 ms Latency       ~12 ms Latency       ~15 ms Latency
```

### 6.3 Why It Is There
A retailer querying order tracking from Tashkent should not suffer a 160ms WAN roundtrip to a US Spanner leader pod when an Asia replica is 10ms away. The Maglev pre-computed spatial table provides instant sub-millisecond routing without requiring Redis lookups or database hops to determine geographic locality.
