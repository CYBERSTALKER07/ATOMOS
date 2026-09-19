# PegasusX Codebase Infrastructure & Service Inventory

**Document Version:** 1.0.0  
**Classification:** Authoritative Technical Inventory & System Catalog  
**Generated Date:** 2026-08-27  
**Workspace Root:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX`  
**Governing Standard:** `CODEBASE_GAP_REPORT.md`, `DOC_SUMMARY.md`, `PLATFORM_AUDIT.md`, `PROJECT.md`

---

## Executive Summary

**PegasusX** is a distributed FMCG (Fast-Moving Consumer Goods) supply chain operating system and B2B wholesale commerce platform engineered for high-throughput multi-tier distribution, warehouse management (WMS), route dispatch optimization, and retail Point-of-Sale (POS) store operations.

This inventory provides an authoritative, exhaustive catalog of all deployable binaries, daemons, background workers, command-line utilities, relational database schemas, in-memory caches, object storage vaults, Kafka event streams, frontend portals, mobile applications, third-party integrations, and cloud networking topologies across the monorepo.

```
+---------------------------------------------------------------------------------------------------------+
|                                    PEGASUSX CORE ARCHITECTURE OVERVIEW                                  |
+---------------------------------------------------------------------------------------------------------+
| [ Native Mobile Apps ]        [ Web Portals & POS ]        [ External B2B / Tax / Gateways ]           |
| (6 iOS & 6 Android apps)     (4 Next.js, 1 Tauri, 1 Expo)  (Soliq OFD, Global Pay, PlayMobile, FCM)     |
|          │                               │                                     │                        |
|          └───────────────────────┬───────┴─────────────────────────────────────┘                        |
|                                  v                                                                      |
|                  [ Cloud Armor WAF & GCE L7 Load Balancer ]                                             |
|                                  │                                                                      |
|                                  v                                                                      |
|           [ GKE Autopilot: backend-go API Gateway & WebSocket Mesh ]                                    |
|             (HTTP :8080 | 8 Role-Specific WebSocket Hubs | Chi Router)                                  |
|                 │                                 │                            │                        |
|                 ├───> [ Cloud Spanner ]           ├───> [ Memorystore Redis ]  ├───> [ optimizer-core ] |
|                 │     (136 Tables, 13 Interleaves,│     (Redis 7.0, Pub/Sub    │     (Python OR-Tools   |
|                 │      193 Secondary Indexes)     │      Mesh, Idempotency)    │      CVRPTW Solver)    |
|                 │                                 │                            │                        |
|                 v                                 v                            v                        |
|           [ Outbox Relay ] ──────────> [ Managed Apache Kafka ] ───> [ GKE ai-worker / Workers ]        |
|           (250ms Poll, Dual-Write)     (10 Canonical Topics)         (12 Consumer Groups, AI Forecast)  |
+---------------------------------------------------------------------------------------------------------+
```

---

## 1. Deployable Services & Microservices

The PegasusX backend is structured under a unified Go workspace (`go.work`, Go 1.25.0), supporting modular microservices, sidecars, and specialized CLI daemons.

### 1.1 Core Backend Services Topology

```
+---------------------------------------------------------------------------------------------------------------------------+
|                                              DEPLOYABLE SERVICES TOPOLOGY                                                 |
+----------------------+--------------------------+-------------+--------------------+--------------------------------------+
| Service Identifier   | Directory Path           | Language    | Ports & Protocols  | Runtime Role & Description           |
+----------------------+--------------------------+-------------+--------------------+--------------------------------------+
| `pegasusx-backend`   | `apps/backend-go`        | Go 1.25.0   | HTTP :8080 (REST)  | Core API Gateway, State Machine,     |
| (API Mode)           |                          | (CGO/glibc) | WSS :8080 (WS Hub) | Session Auth, Real-time Hubs         |
| `pegasusx-backend`   | `apps/backend-go`        | Go 1.25.0   | HTTP :8081 (Health)| Outbox Relay, Kafka Consumers,       |
| (Worker Mode)        |                          | (CGO/glibc) | Internal Daemons   | Schedulers, Dunning, Auto-Dispatch   |
| `ai-worker`          | `apps/ai-worker`         | Go 1.25.0   | HTTP :8081 (Health)| AI Demand Forecasting, Push Cron,    |
|                      |                          | (CGO/glibc) | Kafka Consumers    | Bulk GCS Spreadsheet Ingestion       |
| `handoff-service`    | `apps/handoff-service`   | Go 1.25.0   | HTTP :8082 (REST)  | Stateless Delivery Token & QR Code   |
|                      |                          | (Pure Go)   |                    | Cryptographic HMAC Validation Engine |
| `optimizer-core`     | `services/optimizer-core`| Python 3.12 | HTTP :8082 (REST)  | Operations Research (OR-Tools v9.15) |
| (Python Solver)      | `/server`                | + OpenMP    |                    | Multi-Depot CVRPTW & CP-SAT Solver   |
| `optimizer-core-rust`| `services/optimizer-core`| Rust 1.85   | gRPC :50055        | Low-latency Heuristic Route Solver   |
| (Rust Sidecar)       | `/server-rust`           |             |                    | & Bin-Packing Validator              |
+----------------------+--------------------------+-------------+--------------------+--------------------------------------+
```

---

### 1.2 Detailed Service Profiles

#### 1.2.1 `apps/backend-go` (API & Worker Engine)
- **Directory Path:** `apps/backend-go/`
- **Entrypoint:** `apps/backend-go/main.go`
- **Runtime Environment:** Go 1.25.0 on Debian Bookworm (`debian:bookworm-slim`).
- **CGO / System Dependencies:** Requires Debian `glibc` and CGO enabled due to `github.com/uber/h3-go/v4` C bindings. **Alpine Linux / musl libc is strictly incompatible.**
- **Operational Execution Modes (`PEGASUSX_RUN_MODE`):**
  1. `all` (Default): Runs both the Chi HTTP REST/WebSocket router (:8080) and all in-process background worker daemons.
  2. `api`: Dedicated API Gateway. Disables heavy background workers; runs only the HTTP REST API, WebSocket hubs, and Redis pub/sub relays.
  3. `worker`: Dedicated Worker Pod. Disables public HTTP API endpoints; starts internal health/metrics endpoint (:8081) and executes transactional outbox relays, Kafka event consumers, replenishment schedulers, AR credit aging workers, and batch reconcilers.
- **Port Allocations:**
  - `HTTP_PORT`: `8080` (API Gateway & WebSocket Mesh)
  - `WORKER_HTTP_PORT`: `8081` (Health probes `/healthz`, `/ready`, Prometheus `/metrics`)
- **Transport Protocols:** HTTP/1.1, HTTP/2 (TLS via Ingress), WebSocket (WSS), gRPC (to Cloud Spanner).
- **Core Middlewares:**
  - `bootstrap.TraceMiddleware`: OpenTelemetry trace propagation and W3C `traceparent` parsing.
  - `telemetry.HTTPMetricsMiddleware`: Request durations, status codes, and latency histograms for Prometheus.
  - `auth.SessionAuth`: HS256 JWT validation, extracting user ID, tenant ID, home node ID, and RBAC roles.
  - `partner.AuthMiddlewareOpts`: Partner API key (`pxk_*`) and OAuth Bearer token validation.
  - `auth.RequireTenant`: Enforces tenant context isolation across multi-tenant data routes.
  - `reliability.Middleware`: Bounded concurrency shedding, inflight request throttling, and load shedding.
  - `idempotency.Middleware`: Deduplicates state-mutating requests via SHA-256 body hashes guarded by Redis `SET key token NX EX 120`.
- **Resource Sizing & Profile:**
  - *API Pod:* 500m CPU (Request) / 1000m CPU (Limit), 512Mi (Request) / 1024Mi (Limit). Sized for high concurrency I/O.
  - *Worker Pod:* 1000m CPU (Request) / 2000m CPU (Limit), 1024Mi (Request) / 2048Mi (Limit). Sized for batch Spanner writes and Kafka consumer streams.
- **Scaling Characteristics:**
  - *API Tier:* Autoscales on CPU utilization (Target: 70%) and HTTP Request Rate via Horizontal Pod Autoscaler (HPA: Min 3, Max 10).
  - *Worker Tier:* Autoscales via KEDA based on Kafka Consumer Group Lag (`void_kafka_consumer_lag_seconds`).

---

#### 1.2.2 `apps/ai-worker` (Demand Forecasting & Bulk Import Service)
- **Directory Path:** `apps/ai-worker/`
- **Entrypoint:** `apps/ai-worker/main.go`
- **Runtime Environment:** Go 1.25.0 on Debian Bookworm.
- **CGO Requirements:** CGO enabled.
- **Port Allocation:** `8081` (`AI_WORKER_HTTP_PORT` exposing `/healthz`, `/ready`, `/metrics`, `/v1/optimizer/solve`).
- **Transport Protocols:** HTTP/1.1, Kafka Protocol (SASL_SSL).
- **Execution Modes (`AI_WORKER_MODE`):**
  1. `daemon` (Default): Continuously consumes Kafka topics (`pegasusx-main`, `pegasusx-freeze-locks`, `pegasusx-inventory-import`), executes demand forecasting synthesis, and streams bulk spreadsheet rows.
  2. `predictive-push-cron`: Batch cron execution mode generating predictive replenishment notifications into Spanner `AIPredictions`.
- **Core Subsystems:**
  - `synthesis/`: Real-time order demand forecasting (Croston intermittency & Holt-Winters exponential smoothing).
  - `predictivepush/`: Scheduled push alert generator for retailer inventory stockout prevention.
  - `planningingest/`: Capacity intake for factory production and supplier allocation lines.
  - `import_worker.go`: Background consumer for `pegasusx-inventory-import`, streaming large supplier CSV/XLSX spreadsheets from GCS directly into Spanner staging tables.
  - `optimizer/`: Embedded Clarke-Wright savings and Two-Opt heuristic solver.
- **Resource Sizing & Profile:** 500m CPU / 1024Mi RAM (2 Replicas).

---

#### 1.2.3 `apps/handoff-service` (Stateless Token Validation Service)
- **Directory Path:** `apps/handoff-service/`
- **Entrypoint:** `apps/handoff-service/main.go`
- **Runtime Environment:** Pure Go 1.25.0 binary running on `gcr.io/distroless/base-debian12`.
- **CGO Requirements:** None (CGO Disabled, static binary).
- **Port Allocation:** `8082` (`HTTP_PORT`).
- **Transport Protocol:** HTTP/1.1 REST (`POST /v1/handoff/validate`).
- **Security & Authentication:** Protected via `X-Internal-Api-Key` middleware.
- **Functionality:** Provides an isolated, stateless compute perimeter for validating delivery QR codes and cryptographic HMAC driver-retailer handshakes (`packages/handoff`), isolating cryptographic token validation from main API memory.
- **Resource Sizing & Profile:** 100m CPU / 128Mi RAM (Min 2 Replicas).

---

#### 1.2.4 `services/optimizer-core` (Python OR-Tools VRP Solver)
- **Directory Path:** `services/optimizer-core/`
- **Entrypoint:** `services/optimizer-core/server/http_main.py`
- **Runtime Environment:** Python 3.12 (`python:3.12-slim-bookworm`).
- **Core Dependencies:** `ortools==9.15.6755`, `numpy`, `protobuf`.
- **OS Libraries:** `libgomp1` (GNU OpenMP multi-threading library for C++ OR-Tools parallel search).
- **Port Allocation:** `8082` (`OPTIMIZER_HTTP_PORT`).
- **Transport Protocol:** HTTP/1.1 REST (`POST /v1/optimizer/solve`).
- **Constraint Formulation Solved:**
  - Multi-Depot Capacitated Vehicle Routing Problem with Time Windows (CVRPTW).
  - Vehicle capacity constraints scaled by factor `10_000` with `tetris_buffer` packing ratio (0.95).
  - Cold-chain strict separation (`Stop.requires_cold_chain` restricted to `Vehicle.has_refrigeration`).
  - Hazardous material isolation (`Stop.is_hazardous` restricted to `Vehicle.hazmat_certified`).
  - Maximum stop limits per route and driver shift duration limits.
  - Guided Local Search (`LocalSearchMetaheuristic.GUIDED_LOCAL_SEARCH`) with Path Cheapest Arc initial solution.
- **Compute Sizing:** Pure CPU compute (no GPU required). Multi-threaded C++ solver using OpenMP.
  - *Requests:* 1000m CPU, 512Mi Memory.
  - *Limits:* 2000m CPU, 1024Mi Memory.
  - *Sizing:* 2 to 4 Replicas on GKE.

---

#### 1.2.5 `services/optimizer-core/server-rust` (Rust Heuristic Solver Sidecar)
- **Directory Path:** `services/optimizer-core/server-rust/`
- **Entrypoint:** `services/optimizer-core/server-rust/src/main.rs`
- **Runtime Environment:** Rust 1.85 compiled binary on `debian:bookworm-slim`.
- **Port Allocation:** `50055` (`GRPC_PORT`).
- **Transport Protocol:** gRPC (`OptimizerCore` service defined in `optimizer_core.proto`).
- **Functionality:** High-speed heuristic solver for dense 2D/3D bin-packing verification and sub-50ms heuristic dispatch calculations. Strictly emits solver status `HEURISTIC` (5) and never misreports `OPTIMAL`.
- **Resource Sizing & Profile:** 200m CPU / 256Mi RAM.

---

### 1.3 Complete Catalog of 18 CLI Utilities & Daemons (`apps/backend-go/cmd/*`)

The monorepo contains 18 dedicated Go command-line tools and batch daemons under `apps/backend-go/cmd/`:

```
+--------------------------------------------------------------------------------------------------------------------+
|                                      CATALOG OF ALL 18 CLI UTILITIES & DAEMONS                                     |
+----+----------------------------+------------------------------------+---------------------+-----------------------+
| #  | Binary Name                | Source Directory                   | Primary Execution   | Operational Role      |
+----+----------------------------+------------------------------------+---------------------+-----------------------+
| 1  | `pegasusx-setup`           | `apps/backend-go/cmd/setup`        | One-off Init Job    | Initializes Spanner   |
|    |                            |                                    | / CI/CD pipeline    | DDL, base schema,     |
|    |                            |                                    |                     | and seed admin users  |
| 2  | `apply-migration`          | `apps/backend-go/cmd/apply-migrati`| Pre-deploy K8s Job  | Applies incremental   |
|    |                            | `on`                               | (`migrate-job.yaml`)| Spanner DDL migrations|
| 3  | `schema-drift`             | `apps/backend-go/cmd/schema-drift` | CI/CD Pre-flight    | Verifies live Spanner |
|    |                            |                                    | Verification Gate   | schema against DDL    |
| 4  | `planning-forecast`        | `apps/backend-go/cmd/planning-fore`| K8s CronJob         | Generates 90-day base |
|    |                            | `cast`                             | (`0 2 * * *` Daily) | demand forecasts      |
| 5  | `planning-training-export` | `apps/backend-go/cmd/planning-trai`| K8s CronJob         | Exports completed     |
|    |                            | `ning-export`                      | (`30 2 * * *` Daily)| orders to GCS JSONL   |
| 6  | `planning-accuracy`        | `apps/backend-go/cmd/planning-accu`| K8s CronJob         | Computes WAPE, bias,  |
|    |                            | `racy`                             | (`0 3 * * *` Daily) | and tracking signal   |
| 7  | `safety-stock-replay`      | `apps/backend-go/cmd/safety-stock-`| Admin CLI / On-     | Simulates multi-      |
|    |                            | `replay`                           | demand simulation   | echelon fill rates    |
| 8  | `replay-dlq`               | `apps/backend-go/cmd/replay-dlq`   | SRE Ops Tool        | Replays dead-letter   |
|    |                            |                                    |                     | messages from DLQ     |
| 9  | `backfill-order-timeline`  | `apps/backend-go/cmd/backfill-orde`| Maintenance CLI     | Backfills historical  |
|    |                            | `r-timeline`                       |                     | order transitions     |
| 10 | `backfill-outbox-supplier-`| `apps/backend-go/cmd/backfill-outb`| Maintenance CLI     | Populates missing     |
|    | `id`                       | `ox-supplier-id`                   |                     | outbox tenant IDs     |
| 11 | `backfill-route-geometry`  | `apps/backend-go/cmd/backfill-rout`| Maintenance CLI     | Computes polyline     |
|    |                            | `e-geometry`                       |                     | geometries for routes |
| 12 | `ecosystem-simulator`      | `apps/backend-go/cmd/ecosystem-sim`| Staging / Load Test | High-fidelity synthetic|
|    |                            | `ulator`                           | Synthetic Generator | transaction generator |
| 13 | `ssmr-smokecheck`          | `apps/backend-go/cmd/ssmr-smokeche`| Post-deploy Smoke   | E2E integration test  |
|    |                            | `ck`                               | Test Gate           | suite on live cluster |
| 14 | `mint-dev-jwt`             | `apps/backend-go/cmd/mint-dev-jwt` | Developer Tool      | Mints signed HS256    |
|    |                            |                                    |                     | JWTs for test roles   |
| 15 | `gen-contracts`            | `apps/backend-go/cmd/gen-contracts`| Build-time Generator| Generates Go/TS types  |
|    |                            |                                    |                     | from OpenAPI schemas  |
| 16 | `seed-demo-scope`          | `apps/backend-go/cmd/seed-demo-sco`| Demo Provisioner    | Seeds localized demo  |
|    |                            | `pe`                               |                     | catalogs and stores   |
| 17 | `seed-supplier-prodsim`    | `apps/backend-go/cmd/seed-supplier-`| Benchmark Seeder   | Populates production- |
|    |                            | `prodsim`                          |                     | scale supplier catalogs|
| 18 | `seed-warehouse-prodsim`   | `apps/backend-go/cmd/seed-warehouse`| Benchmark Seeder   | Seeds realistic WMS   |
|    |                            | `-prodsim`                         |                     | bins, lots, and stock |
+----+----------------------------+------------------------------------+---------------------+-----------------------+
```

---

## 2. Database & Storage Inventory

### 2.1 Google Cloud Spanner (Primary Authoritative Relational Database)
Cloud Spanner serves as the primary system of record for all multi-tenant transactional, financial, and logistics operations.

- **Instance Configuration:** `google_spanner_instance.ledger`
- **Instance Sizing:** Regional or Multi-Regional (e.g. `regional-asia-south1`, `regional-europe-west1`), 100 Processing Units (PU) baseline with automated autoscaling up to 1000 PU (1 Node).
- **Database Name:** `main` (or `pegasusx-ssmr-db` in SSMR staging).
- **Configuration Flags:**
  - `enable_drop_protection = true` (Prevents accidental deletion of production data).
  - `version_retention_period = "7d"` (Point-In-Time Recovery / PITR window).
- **Automated Backup Schedule:** Daily full backups automated via `google_spanner_backup_schedule.daily_full` (`spanner_backup.tf`), retention 30 days.

#### 2.1.1 Schema Metrics & Bounded Contexts
- **Total Relational Tables:** **136 tables** defined in `apps/backend-go/schema/spanner.ddl`.
- **Interleaved Parent-Child Hierarchies:** **13 tables** using physical split co-location.
- **Secondary Indexes:** **193 indexes** (including unique constraints and storing indexes).
- **Primary Key Multi-Tenancy Strategy:** Root tables utilize composite primary keys prefixed with tenant identifiers (e.g., `SupplierId`, `RetailerId`, `WarehouseId`) or UUIDv4 identifiers to ensure balanced distribution across Spanner splits.

```
Cloud Spanner Bounded Contexts (136 Tables):
├── 1. Identity & Multi-Tenancy (Suppliers, Retailers, Drivers, Warehouses, Factories, Users, Roles)
├── 2. Commerce & Order Engine (Orders, CartItems, Allocations, StateTransitions, ShopClosedLog)
├── 3. Warehouse Execution (WMS) (InventoryLevels, StockLots, Bins, WarehouseSupplyRequests)
├── 4. Logistics, Dispatch & Fleet (SupplierTruckManifests, ManifestOrders, Routes, ReverseLogistics)
├── 5. Trade Credit & Accounts Receivable (ArInvoices, ArLedgerEntries, RetailerCreditProfiles, Dunning)
├── 6. Payments & Settlement (PaymentSessions, PaymentAttempts, PaymentLedgerEntries, WebhookInbox)
├── 7. Pricing, Catalog & Promotions (PriceLists, PriceListItems, Promotions, CustomerSegments)
├── 8. Fiscalization & Soliq OFD (OrderFiscalReceipts, TaxRegimeVersions, OrderLineFiscalSnapshots)
├── 9. Retail OS & In-Store POS (RetailerRegisters, PosSessions, StockBalances, Shifts, TimeEntries)
├── 10. Planning & AI Demand Sensing (DemandForecastBaseline, DemandSignals, ReplenishmentInsights)
├── 11. Evidence Vault & Claims (EvidenceDossiers, EvidenceItems, Claims, ClaimEvidences, Returns)
└── 12. Platform Infrastructure (OutboxEvents, AuditLog, DeviceTokens, ClientVersionPolicies)
```

#### 2.1.2 The 13 Interleaved Hierarchies
Physical table interleaving co-locates child rows directly with parent rows on the same storage split, providing zero-latency single-split joins, atomic cascading deletions (`ON DELETE CASCADE`), and localized transaction isolation:

```
+-------------------------------------------------------------------------------------------------------------------+
|                                      CLOUD SPANNER INTERLEAVED HIERARCHIES                                        |
+----+--------------------------------+----------------------------+------------------------------------------------+
| #  | Child Table                    | Parent Table               | Primary Key Signature                          |
+----+--------------------------------+----------------------------+------------------------------------------------+
| 1  | `ClaimEvidences`               | `Claims`                   | `(ClaimId, EvidenceId)`                        |
| 2  | `WarehouseSupplyRequestItems`  | `WarehouseSupplyRequests`  | `(RequestId, ItemId)`                          |
| 3  | `ManifestReplanLog`            | `SupplierTruckManifests`   | `(ManifestId, ReplanId)`                       |
| 4  | `ManifestOrders`               | `SupplierTruckManifests`   | `(ManifestId, OrderId)`                        |
| 5  | `RegionalConfigs`              | `Regions`                  | `(RegionId, ConfigKey)`                        |
| 6  | `SupplierImportStagedRows`     | `SupplierImportSessions`   | `(supplier_id, session_id, row_index)`         |
| 7  | `SupplierImportMapping`        | `SupplierImportSessions`   | `(supplier_id, session_id)`                    |
| 8  | `OrderShopClosedLog`           | `Orders`                   | `(OrderId, EventId)`                           |
| 9  | `OrderLineFiscalSnapshots`     | `Orders`                   | `(OrderId, LineSku)`                           |
| 10 | `OrderPaymentLegs`             | `Orders`                   | `(OrderId, LegId)`                             |
| 11 | `CreditNoteLines`              | `CreditNotes`              | `(CreditNoteId, LineId)`                       |
| 12 | `PriceListItems`               | `PriceLists`               | `(PriceListId, Sku)`                           |
| 13 | `OrderLineAllocations`         | `Orders`                   | `(OrderId, OrderLineId, WarehouseId)`          |
+----+--------------------------------+----------------------------+------------------------------------------------+
```

#### 2.1.3 Critical Secondary Indexes
- `Idx_OutboxEvents_Unpublished`: `OutboxEvents(PublishedAt, CreatedAt) STORING (Topic, EventType, TraceId, SupplierId)` — Ensures zero-overhead index-only scans during continuous outbox polling.
- `Idx_Orders_BySupplierStatus`: `Orders(SupplierId, Status, CreatedAt DESC)` — Drives supplier portal order queues.
- `Idx_Orders_ByRetailerStatus`: `Orders(RetailerId, Status, CreatedAt DESC)` — Powers retailer purchase history.
- `Idx_ManifestOrders_ByDriver`: `ManifestOrders(DriverId, SequenceNumber)` — Turn-by-turn driver delivery progression.
- `Idx_PaymentSessions_ByGatewayRef`: `PaymentSessions(GatewayReference, Status)` — Webhook idempotency resolution.
- `Idx_ArInvoices_BySupplierDue`: `ArInvoices(SupplierId, Status, DueAt)` — High-speed AR dunning sweepers.

---

### 2.2 Google Cloud Memorystore for Redis 7.0 (In-Memory Tier)
- **Deployment Mode:** Google Cloud Memorystore for Redis (Version 7.0, `STANDARD_HA` with cross-zone read replica).
- **VPC Configuration:** Private Service Access (PSA) peering (`10.42.205.148:6378`).
- **Security:** In-transit TLS encryption (`transit_encryption_mode = "SERVER_AUTHENTICATION"`), mandatory Redis AUTH password injected via Secret Manager.
- **Eviction Policy:** `allkeys-lru` (Least Recently Used).
- **Capacity Sizing:** 2 GB baseline memory for SSMR, 5 GB for Production.
- **Fail-Closed Policy (`cache/circuit_breaker.go`):** In production (`REQUIRE_INFRA_ADAPTERS=true`), cache failures fail closed to prevent silent state leakage.

#### 2.2.1 Real-Time WebSocket Mesh (8 Role-Specific Hubs)
Mounted at `/v1/ws` with cross-pod Redis Pub/Sub synchronization (`apps/backend-go/ws/`):

```
+--------------------------------------------------------------------------------------------------------------------+
|                                           REDIS WEBSOCKET HUBS MATRIX                                              |
+----+-------------------+-----------------------+-----------------------------+-------------------------------------+
| #  | Hub Name          | Mount Path            | Redis Pub/Sub Channel       | Functional Role & Event Streaming   |
+----+-------------------+-----------------------+-----------------------------+-------------------------------------+
| 1  | `RetailerHub`     | `/v1/ws` (Role auth)  | `pubsub:ws:retailer`        | Order status, promotions, cart holds|
| 2  | `SupplierHub`     | `/v1/ws` (Role auth)  | `pubsub:ws:supplier`        | Inbound orders, manifest progress   |
| 3  | `DriverHub`       | `/v1/ws` (Role auth)  | `pubsub:ws:driver`          | Route stops, door geofence unlock   |
| 4  | `PayloadHub`      | `/v1/ws` (Role auth)  | `pubsub:ws:payload`         | Dock pallet scans, truck seal alert |
| 5  | `WarehouseHub`    | `/v1/ws` (Role auth)  | `pubsub:ws:warehouse`       | Wave picking allocations, bin moves |
| 6  | `FactoryHub`      | `/v1/ws` (Role auth)  | `pubsub:ws:factory`         | Production runs, transfer dispatches|
| 7  | `TelemetryHub`    | `/v1/ws` (Role auth)  | `pubsub:ws:telemetry`       | 1Hz driver Kalman-filtered GPS stream|
| 8  | `PlatformAdminHub`| `/v1/ws` (Role auth)  | `pubsub:ws:platformAdmin`   | Global audit events, SLO monitors   |
+----+-------------------+-----------------------+-----------------------------+-------------------------------------+
```

#### 2.2.2 Redis In-Memory Key Patterns
- **Idempotency Locks:** `idem:<sha256_hash>` (TTL: 120 seconds, `SETNX`).
- **Kafka Deduplication:** `kafka:dedup:<consumer_group>:<event_id>` (TTL: 7 days).
- **Driver GPS Coordinates:** `telemetry:driver:<driver_id>` (TTL: 60 seconds).
- **Cache Invalidation Channel:** `pubsub:cache_invalidate` (Entity cache eviction broadcasts across pods).
- **Worker Liveness Heartbeat:** `pegasusx:worker:heartbeat` (TTL: 30 seconds).

---

### 2.3 Google Cloud Storage (GCS Bucket Inventory)

PegasusX requires 4 dedicated Google Cloud Storage buckets configured with uniform bucket-level access:

```
+--------------------------------------------------------------------------------------------------------------------+
|                                      GOOGLE CLOUD STORAGE (GCS) INVENTORY                                          |
+----+--------------------------------+-----------------+---------------------------+--------------------------------+
| #  | Bucket Identifier              | Storage Class   | Access / Permissions      | Lifecycle & Object Schema      |
+----+--------------------------------+-----------------+---------------------------+--------------------------------+
| 1  | `pegasusx-prod-media`          | Standard        | Signed V4 PUT (15m expiry)| `catalog/{supplier_id}/*.jpg`  |
|    | (`GCS_BUCKET_NAME`)            | (Multi-Region)  | Public / Signed GET read  | `evidence/driver/*.jpg` (POD)  |
|    |                                |                 |                           | `evidence/claim/*.jpg` (Claims)|
|    |                                |                 |                           | `evidence/credit/*.pdf` (AR)   |
| 2  | `pegasusx-prod-app-updates`    | Standard        | Public Read via Cloud CDN | `android/{role}/*.apk` (OTA)   |
|    |                                | (Multi-Region)  | (`allUsers:objectViewer`) | `ios/{role}/manifest.plist`    |
|    |                                |                 |                           | `{portal}-desktop/*/updater.js`|
| 3  | `pegasusx-prod-imports-exports`| Standard        | Private IAM               | `imports/inventory/*.csv|.xlsx`|
|    |                                | (Regional)      | Service Account only      | `exports/compliance/*.csv|.json`|
|    |                                |                 |                           | Auto-delete after 30 days      |
| 4  | `pegasus-503013-terraform-stat`| Standard        | Strictly Private IAM      | `default.tfstate`              |
|    | `e`                            | (Regional)      | DevOps / CI/CD only       | Object Versioning Enabled      |
+----+--------------------------------+-----------------+---------------------------+--------------------------------+
```

#### Cryptographic Evidence Dossier Vault (`apps/backend-go/storage/evidence_vault.go`)
- All photo Proof-of-Delivery (POD), damaged item returns, and credit promissory agreements are uploaded to `pegasusx-prod-media`.
- Metadata is recorded in Spanner `EvidenceDossiers` and `EvidenceItems` with SHA-256 binary content hashes, GPS capture coordinates (`Latitude`, `Longitude`), and ISO timestamps.
- Sealing a dossier executes `SealDossier()`, computing a deterministic aggregate SHA-256 checksum over all item hashes, rendering the audit dossier cryptographically immutable.

---

## 3. Message Brokers, Event Queues & Streaming

### 3.1 Apache Kafka / Google Managed Service for Apache Kafka
PegasusX employs Apache Kafka as its durable, distributed event backbone for transactional domain events, driver telemetry, and AI forecasting pipelines.

- **Cluster Architecture:** 3-Node Managed Kafka Cluster deployed across 3 Availability Zones.
- **Transport Security:** `SASL_SSL` (Port 9092) with Google Cloud IAM OAuth tokens (`KAFKA_AUTH_MODE=GCP_MANAGED_OAUTH`).
- **Cluster Sizing:** 3 vCPUs, 16 GiB RAM per broker, 1000 GiB SSD persistent storage.

#### 3.1.1 The 10 Canonical Kafka Topics Registry

```
+-----------------------------------------------------------------------------------------------------------------------+
|                                           CANONICAL KAFKA TOPIC REGISTRY                                              |
+----+----------------------------+------------+---------------+----------------------------------+---------------------+
| #  | Topic Name                 | Partitions | Replication F.| Primary Event Payloads           | Partition Key       |
+----+----------------------------+------------+---------------+----------------------------------+---------------------+
| 1  | `pegasusx-main`            | 12         | 3 (min ISR 2) | Core transactional events,       | `supplier_id` or    |
|    |                            |            |               | Orders, Suppliers, Retailers     | `order_id`          |
| 2  | `pegasusx-orders`          | 12         | 3 (min ISR 2) | Order FSM transitions, payments, | `order_id`          |
|    |                            |            |               | ADR-009 fiscal state changes     |                     |
| 3  | `pegasusx-dispatch`        | 6          | 3 (min ISR 2) | Manifest creation, sealing,      | `warehouse_id` or   |
|    |                            |            |               | wave picking, route dispatches   | `manifest_id`       |
| 4  | `pegasusx-realtime`        | 12         | 3 (min ISR 2) | High-frequency driver GPS,       | `driver_id`         |
|    |                            |            |               | doorstep approach geofences      |                     |
| 5  | `pegasusx-demand`          | 6          | 3 (min ISR 2) | POS sell-through flywheel signals| `supplier_id:sku`   |
|    |                            |            |               | for inventory intelligence       |                     |
| 6  | `logistics.exceptions.v1`  | 6          | 3 (min ISR 2) | OS&D damage claims, missing      | `claim_id` or       |
|    |                            |            |               | items, return dock handshakes    | `order_id`          |
| 7  | `logistics.telemetry.v1`   | 6          | 3 (min ISR 2) | Cold-chain temperature breaches, | `vehicle_id` or     |
|    |                            |            |               | electronic seal tampering        | `sensor_id`         |
| 8  | `pegasusx-freeze-locks`    | 6          | 3 (min ISR 2) | Dispatcher catalog & route locks | `supplier_id`       |
| 9  | `pegasusx-inventory-import`| 6          | 3 (min ISR 2) | Supplier bulk catalog spreadsheet| `import_session_id` |
|    |                            |            |               | ingestion triggers               |                     |
| 10 | `pegasusx-main-dlq`        | 6          | 3 (min ISR 2) | Dead-letter queue for failed/    | Original message key|
|    |                            |            |               | poison events (14-day retention) |                     |
+----+----------------------------+------------+---------------+----------------------------------+---------------------+
```

---

### 3.2 Transactional Outbox Relay Pattern (`apps/backend-go/outbox/`)
- **Dual-Write Architecture:** Every mutating business action writes domain state and an `OutboxEvents` row in the **exact same atomic Spanner transaction closure** (`spanner.BufferWrite`).
- **Continuous Polling Loop (`outbox/relay.go`):** The `OutboxRelay` background daemon queries unpublished outbox rows every **250ms** (`BatchSize=100`, max 20 delivery attempts per event) using index-only scans on `Idx_OutboxEvents_Unpublished`.
- **Dual-Write Domain Routing (`events/topic_routing.go`):** When `KAFKA_TOPIC_DUAL_WRITE=true`, the relay publishes events simultaneously to `pegasusx-main` and specific domain topics (`pegasusx-orders`, `pegasusx-dispatch`, `pegasusx-realtime`).
- **Consumer Deduplication (`kafka/redis_event_dedup.go`):** Consuming workers guard against duplicate processing via Redis `SET kafka:dedup:<group>:<event_id> 1 NX EX 604800` (7-day TTL).

---

### 3.3 Active Kafka Consumer Groups Catalog (12 Groups)

```
+----------------------------------------------------------------------------------------------------------------------+
|                                            KAFKA CONSUMER GROUPS CATALOG                                             |
+----+---------------------------------------+---------------+-------------------------+-------------------------------+
| #  | Consumer Group ID                     | Host Service  | Subscribed Topics       | Functional Responsibility     |
+----+---------------------------------------+---------------+-------------------------+-------------------------------+
| 1  | `void-notification-dispatcher`        | `backend-go`  | `pegasusx-main`,        | Fans out FCM push alerts, APNs|
|    |                                       | (Worker)      | `pegasusx-orders`,      | notices, and in-app inboxes   |
|    |                                       |               | `pegasusx-dispatch`     |                               |
| 2  | `void-order-mutator`                  | `backend-go`  | `pegasusx-orders` (or   | Maintains order read models   |
|    |                                       | (Worker)      | `pegasusx-main`)        | and inventory counters        |
| 3  | `void-warehouse-mutator`              | `backend-go`  | `pegasusx-dispatch` (or | Allocates dock bays and wave  |
|    |                                       | (Worker)      | `pegasusx-main`)        | pick task releases            |
| 4  | `void-returns-reverse`                | `backend-go`  | `logistics.exceptions.v1| Dispatches reverse logistics  |
|    |                                       | (Worker)      | `                       | and intake quarantine tasks   |
| 5  | `void-claims-bridge`                  | `backend-go`  | `logistics.exceptions.v1| Locks defective stock into    |
|    |                                       | (Worker)      | `                       | Spanner quarantine locations  |
| 6  | `void-billing-tier`                   | `backend-go`  | `pegasusx-main`         | Meters API calls and computes |
|    |                                       | (Worker)      |                         | supplier SaaS fee schedules   |
| 7  | `void-partner-webhooks`               | `backend-go`  | `pegasusx-main`         | Dispatches outbound partner   |
|    |                                       | (Worker)      |                         | webhooks with exponential retry|
| 8  | `void-digital-twin`                   | `backend-go`  | `pegasusx-main`,        | Ingests telemetry into spatial|
|    |                                       | (Worker)      | `pegasusx-realtime`     | digital twin graph models     |
| 9  | `pegasusx-ai-worker`                  | `ai-worker`   | `pegasusx-main`         | Ingests orders for demand     |
|    |                                       |               |                         | forecasting synthesis         |
| 10 | `pegasusx-ai-worker-freeze`           | `ai-worker`   | `pegasusx-freeze-locks` | Tracks manual route locks     |
| 11 | `pegasusx-ai-worker-planning-ingest`  | `ai-worker`   | `pegasusx-main`         | Ingests factory production and|
|    |                                       |               |                         | supplier capacity matrices    |
| 12 | `pegasusx-ai-worker-inventory-import` | `ai-worker`   | `pegasusx-inventory-impo| Streams bulk catalog CSV/XLSX |
|    |                                       |               | `rt`                    | spreadsheets from GCS to DB   |
+----+---------------------------------------+---------------+-------------------------+-------------------------------+
```

---

## 4. Client Applications & Frontend Surfaces

PegasusX features 18 distinct client application surfaces covering web portals, desktop POS terminals, dock kiosks, and native iOS / Android mobile applications:

```
+----------------------------------------------------------------------------------------------------------------------+
|                                        CLIENT APPLICATIONS & FRONTENDS MATRIX                                        |
+----+--------------------------+-----------------------+-------------------+--------------------+---------------------+
| #  | Application Name         | Directory Path        | Framework / Stack | Target Platform    | Port / Build Output |
+----+--------------------------+-----------------------+-------------------+--------------------+---------------------+
| 1  | `supplier-portal`        | `apps/supplier-portal`| Next.js 15 App    | Web / Desktop      | Port :3000          |
|    |                          |                       | Router, React 19  | (Tauri v2 shell)   | Nginx / Standalone  |
| 2  | `warehouse-portal`       | `apps/warehouse-portal`Next.js 15 App    | Web / Desktop      | Port :3002          |
|    |                          |                       | Router, React 19  | (Tauri v2 shell)   | Nginx / Standalone  |
| 3  | `factory-portal`         | `apps/factory-portal` | Next.js 15 App    | Web / Desktop      | Port :3003          |
|    |                          |                       | Router, React 19  | (Tauri v2 shell)   | Nginx / Standalone  |
| 4  | `marketing-site`         | `apps/marketing-site` | Next.js 15 App    | Public Web         | Port :3004          |
|    |                          |                       | Router, React 19  | (Three.js 3D)      | Static / Cloud CDN  |
| 5  | `retailer-app-desktop`   | `apps/retailer-app-des`Next.js + Tauri v2 | Desktop POS        | Port :3001          |
|    |                          | `ktop`                | (Rust) + SQLite   | (macOS & Windows)  | Native App Bundle   |
| 6  | `payload-terminal`       | `apps/payload-terminal`Expo 55 / React    | Kiosk Tablet       | Web / Android Kiosk |
|    |                          |                       | Native 0.83       | (Dock Loading)     | Standalone Build    |
| 7  | `driver-app-ios`         | `apps/driver-app-ios` | Swift 6 / SwiftUI | iOS 17+ Native     | Xcode App Bundle    |
| 8  | `retailer-app-ios`       | `apps/retailer-app-ios`Swift 6 / SwiftUI | iOS 17+ Native     | Xcode App Bundle    |
| 9  | `supplier-app-ios`       | `apps/supplier-app-ios`Swift 6 / SwiftUI | iOS 17+ Native     | Xcode App Bundle    |
| 10 | `warehouse-app-ios`      | `apps/warehouse-app-io`Swift 6 / SwiftUI | iOS 17+ Native     | Xcode App Bundle    |
| 11 | `factory-app-ios`        | `apps/factory-app-ios`| Swift 6 / SwiftUI | iOS 17+ Native     | Xcode App Bundle    |
| 12 | `payload-app-ios`        | `apps/payload-app-ios`| Swift 6 / SwiftUI | iOS 17+ Native     | Xcode App Bundle    |
| 13 | `driver-app-android`     | `apps/driver-app-andro`Kotlin 2.0 /       | Android 14+ Native | Gradle APK / AAB    |
|    |                          | `id`                  | Jetpack Compose   |                    |                     |
| 14 | `retailer-app-android`   | `apps/retailer-app-and`Kotlin 2.0 /       | Android 14+ Native | Gradle APK / AAB    |
|    |                          | `roid`                | Jetpack Compose   |                    |                     |
| 15 | `supplier-app-android`   | `apps/supplier-app-and`Kotlin 2.0 /       | Android 14+ Native | Gradle APK / AAB    |
|    |                          | `roid`                | Jetpack Compose   |                    |                     |
| 16 | `warehouse-app-android`  | `apps/warehouse-app-an`Kotlin 2.0 /       | Android 14+ Native | Gradle APK / AAB    |
|    |                          | `droid`               | Jetpack Compose   |                    |                     |
| 17 | `factory-app-android`    | `apps/factory-app-andr`Kotlin 2.0 /       | Android 14+ Native | Gradle APK / AAB    |
|    |                          | `oid`                 | Jetpack Compose   |                    |                     |
| 18 | `payload-app-android`    | `apps/payload-app-andr`Kotlin 2.0 /       | Android 14+ Native | Gradle APK / AAB    |
|    |                          | `oid`                 | Jetpack Compose   |                    |                     |
+----+--------------------------+-----------------------+-------------------+--------------------+---------------------+
```

### 4.1 Client Features & Native Technologies
- **Web Portals:** Built with Tailwind CSS v4, `@heroui/react`, and `packages/ui-kit`. Integrate MapLibre GL for live vehicle tracking, Recharts for financial analytics, and persistent WebSocket connections.
- **Desktop POS (`retailer-app-desktop`):** Tauri v2 Rust wrapper providing local SQLite offline transaction persistence (`packages/desktop-cache`), ESC/POS thermal receipt printing, and hardware barcode scanner integration.
- **iOS Mobile Apps:** Modern Swift Concurrency (`async`/`await`), `@Observable` macros, Apple Keychain secure storage, Apple Native MapKit / MapLibre, and background push notifications.
- **Android Mobile Apps:** Jetpack Compose, Material 3, Hilt dependency injection, Room DB offline persistence, Kotlin Coroutines/Flow, and ML Kit Barcode Scanning.

---

## 5. External & Third-Party API Integrations

```
+-----------------------------------------------------------------------------------------------------------------------+
|                                           THIRD-PARTY INTEGRATIONS MATRIX                                             |
+---------------------+-------------------------------+-------------------------+---------------------------------------+
| Provider / Standard | Integration Purpose           | Communication Protocol  | Credentials & GSM Secret References   |
+---------------------+-------------------------------+-------------------------+---------------------------------------+
| **Soliq OFD (EHF)** | Electronic Tax Invoicing,     | HTTPS REST              | `fiscal-my-soliq-api-key`,            |
| (Uzbekistan Tax)    | PKCS#12 Digital EDS Signing   | (8s Timeout)            | `fiscal-my-soliq-tin`,                |
|                     |                               |                         | `fiscal-my-soliq-pkcs12-password`     |
| **Global Pay**      | Primary Card Acquiring        | HTTPS REST &            | `global-pay-service-id`,              |
| (Uzcard / Humo)     | (Uzcard / Humo) & Settlements | Webhook HMAC-SHA256     | `global-pay-username`,                |
|                     |                               |                         | `global-pay-password`,                |
|                     |                               |                         | `global-pay-webhook-secret`           |
| **Payme**           | Card Acquiring & Wallets      | JSON-RPC 2.0 Merchant   | `payme-webhook-secret` (HTTP Basic)   |
| **Click**           | Wallet & USSD Payments        | REST form-urlencoded    | `click-webhook-secret` (Sign hash)    |
| **Adyen**           | International Multi-Acquiring | Webhook HMAC-SHA256     | `adyen-webhook-secret`                |
| **Stripe**          | International Credit Cards    | Webhook HMAC-SHA256     | `stripe-webhook-secret`               |
| **PlayMobile SMS**  | Uzbekistan Carrier-Grade SMS  | HTTP Basic Auth REST    | `playmobile-login`,                   |
|                     | Dunning & Doorstep PIN Alerts | `broker-api/send`       | `playmobile-password`                 |
| **Twilio**          | International SMS & WhatsApp  | REST / WhatsApp Content | `twilio-account-sid`,                 |
|                     | Payment Notices               | API                     | `twilio-auth-token`                   |
| **SendGrid v3**     | Financial Invoices & Dunning  | HTTPS REST              | `sendgrid-api-key`,                   |
|                     | Email Statements              | `v3/mail/send`          | `sendgrid-from-email`                 |
| **Google Maps**     | Routes v2 `ComputeRoutes`,    | HTTPS REST              | `google-maps-api-key`                 |
|                     | Places Autocomplete, Geocoding|                         |                                       |
| **OSRM (Self-Host)**| High-Speed Distance Matrices  | HTTP REST               | Internal Service Endpoint             |
|                     | `/table/v1/driving/`          |                         | `http://osrm:5000`                    |
| **Firebase Auth**   | Retailer & Driver Phone OTP   | Google Identity SDK     | `firebase-credentials`,               |
|                     | Authentication                | OIDC JWT verification   | `FIREBASE_PROJECT_ID`                 |
| **Firebase FCM**    | Mobile Push Notifications &   | FCM HTTP v1 API +       | `firebase-credentials` (ServiceAcc)   |
|                     | APNs Apple Bridge             | APNs Priority 10        |                                       |
+---------------------+-------------------------------+-------------------------+---------------------------------------+
```

---

## 6. Network & Security Topology

```
+---------------------------------------------------------------------------------------------------------+
|                                    GCP NETWORK & SECURITY ARCHITECTURE                                  |
+---------------------------------------------------------------------------------------------------------+
|                                         [ Public Internet ]                                             |
|                                                  │                                                      |
|                                                  v                                                      |
|                                      [ Cloud Armor WAF Policy ]                                         |
|                                  (Rate limiting, GeoIP, XSS / SQLi)                                     |
|                                                  │                                                      |
|                                                  v                                                      |
|                                    [ GCE L7 HTTPS Load Balancer ]                                       |
|                                (Static External IP: 136.69.43.141)                                      |
|                                                  │                                                      |
|          ┌───────────────────────────────────────┴────────────────────────────────────────┐             |
|          │ /v1/ws (Session Affinity 3600s)       │ /v1, /partner (Timeout 120s)           │ /* (Static) |
|          v                                       v                                        v             |
|   [ backend-go-ws ]                       [ backend-go API ]                      [ Cloud CDN / GCS ]   |
|   (GKE Autopilot)                         (GKE Autopilot)                         (Web Portals & OTA)   |
|          │                                       │                                                      |
|          └───────────────────────┬───────────────┘                                                      |
|                                  │                                                                      |
|                                  v                                                                      |
|            [ VPC: pegasusx-vpc (Regional Subnet 10.10.0.0/20) ]                                         |
|            ├── Secondary Pod CIDR:     10.20.0.0/16 (VPC-Native)                                        |
|            └── Secondary Service CIDR: 10.30.0.0/20                                                     |
|                                  │                                                                      |
|         ┌────────────────────────┼────────────────────────┬─────────────────────────┐                   |
|         v                        v                        v                         v                   |
|  [ Cloud Spanner ]      [ Memorystore Redis ]    [ Managed Kafka ]          [ Cloud NAT Gateway ]       |
|  (Encrypted gRPC)       (PSA: 10.42.205.148/29   (SASL_SSL 9092             (Static Egress EIPs for     |
|                          AUTH + TLS 6378)         3 AZ Brokers)              Soliq & Bank Allowlisting) |
+---------------------------------------------------------------------------------------------------------+
```

### 6.1 VPC & Subnet Allocations
- **VPC Network:** `pegasusx-vpc` (Custom Subnet Mode).
- **Primary Node Subnet:** `10.10.0.0/20` (Subnet: `pegasusx-vpc-subnet`).
- **Secondary Pod Range:** `10.20.0.0/16` (Supports up to 65,536 Pod IPs via VPC-Native IP aliasing).
- **Secondary Service Range:** `10.30.0.0/20` (Supports up to 4,096 ClusterIP services).
- **Private Service Access (PSA):** Peered with `10.42.205.148/29` for Cloud Memorystore Redis 7.0 and Managed Kafka.

### 6.2 GKE Workload Identity Federation
- **Workload Pool:** `${var.project_id}.svc.id.goog`.
- **Kubernetes Service Account:** `backend-go` in namespace `pegasusx`.
- **Google Service Account (GSA):** `pegasusx-backend@${var.project_id}.iam.gserviceaccount.com`.
- **IAM Binding:** `roles/iam.workloadIdentityUser` bound to `serviceAccount:${var.project_id}.svc.id.goog[pegasusx/backend-go]`.

### 6.3 Static Outbound Egress via Cloud NAT
- **Requirement:** Banking rails (Global Pay, Payme, Click) and regulatory tax gateways (Soliq OFD) require static, allowlisted egress IP addresses.
- **Topology:** Cloud Router in `asia-south1` / `europe-west3` paired with Cloud NAT using dedicated static external IP allocations (`google_compute_address.egress_ips`). All outbound traffic from GKE nodes uses these deterministic IP addresses.

### 6.4 Edge Ingress, Cloud Armor & SSL Termination
- **Load Balancer:** Google Cloud External L7 HTTPS Load Balancer with Google-managed SSL Certificate (`ManagedCertificate`).
- **Cloud Armor Security Policy (`google_compute_security_policy`):**
  - Rate limiting: Max 500 requests per 10 seconds per IP with banned thresholds.
  - OWASP Top 10 ModSecurity rule sets for SQL injection and XSS filtering.
  - Geo-fencing: Restricts administrative routes to specified regional CIDRs.
- **Ingress Route Paths:** Explicitly exposes `/v1`, `/v1/ws`, `/partner`, `/healthz`, and `/ready`. Internal `/metrics` and `/debug` endpoints are blocked from public ingress.

---

## 7. Master Environment Variables & Secret Inventory

```
+-----------------------------------------------------------------------------------------------------------------------+
|                                    MASTER ENVIRONMENT & SECRET MAPPING DICTIONARY                                     |
+------------------------------------+------------------+---------------------+-----------------------------------------+
| Environment Variable               | Type             | Source / GSM Key    | Functional Scope & Behavior             |
+------------------------------------+------------------+---------------------+-----------------------------------------+
| `HTTP_PORT`                        | Port (8080)      | ConfigMap           | Chi HTTP REST API & WebSocket listener  |
| `WORKER_HTTP_PORT`                 | Port (8081)      | ConfigMap           | Worker health, readiness, and metrics   |
| `PEGASUSX_RUN_MODE`                | Enum             | ConfigMap           | `api`, `worker`, or `all`               |
| `PEGASUSX_ENV`                     | Enum             | ConfigMap           | `production`, `ssmr`, or `dev`          |
| `REQUIRE_INFRA_ADAPTERS`           | Boolean          | ConfigMap (`true`)  | Fails boot if Spanner/Redis/Kafka down  |
| `ALLOW_MEMORY_FALLBACK`            | Boolean          | ConfigMap (`false`) | Disallows memory mock stubs in prod     |
| `TENANT_CONTEXT_ENFORCED`          | Boolean          | ConfigMap (`true`)  | Enforces tenant isolation on all routes |
| `SPANNER_PROJECT`                  | GCP Project ID   | ConfigMap           | Host GCP project for Cloud Spanner      |
| `SPANNER_INSTANCE`                 | String           | ConfigMap           | Cloud Spanner instance identifier       |
| `SPANNER_DATABASE`                 | String           | ConfigMap           | Cloud Spanner database name (`main`)    |
| `REDIS_ADDR`                       | Host:Port        | GSM / ESO           | Memorystore Redis host and port         |
| `REDIS_PASSWORD`                   | Secret           | `redis-auth`        | Redis AUTH password                     |
| `REDIS_TLS_ENABLED`                | Boolean          | ConfigMap (`true`)  | In-transit TLS encryption flag          |
| `KAFKA_BROKERS`                    | CSV String       | GSM / ESO           | Managed Kafka bootstrap broker list     |
| `KAFKA_AUTH_MODE`                  | Enum             | ConfigMap           | `GCP_MANAGED_OAUTH` (SASL_SSL)          |
| `KAFKA_TOPIC_MAIN`                 | String           | ConfigMap           | Primary outbox topic (`pegasusx-main`)  |
| `KAFKA_TOPIC_MAIN_DLQ`             | String           | ConfigMap           | Dead-letter topic (`pegasusx-main-dlq`) |
| `JWT_SECRET`                       | Secret           | `jwt-secret`        | HS256 JWT signature secret key          |
| `INTERNAL_API_KEY`                 | Secret           | `internal-api-key`  | Inter-service handshake authorization   |
| `PLATFORM_ADMIN_MFA_REQUIRED`      | Boolean          | ConfigMap (`true`)  | Mandatory TOTP MFA for platform admins  |
| `FIREBASE_PROJECT_ID`              | String           | ConfigMap           | Firebase project identifier              |
| `GLOBAL_PAY_SERVICE_ID`            | Secret           | `global-pay-svc-id` | Global Pay merchant service ID          |
| `GLOBAL_PAY_USERNAME`              | Secret           | `global-pay-user`   | Global Pay API username                 |
| `GLOBAL_PAY_PASSWORD`              | Secret           | `global-pay-pass`   | Global Pay API password                 |
| `GLOBAL_PAY_WEBHOOK_SECRET`        | Secret           | `global-pay-wh-sec` | Global Pay webhook HMAC signature key   |
| `PAYME_WEBHOOK_SECRET`             | Secret           | `payme-wh-secret`   | Payme JSON-RPC webhook password         |
| `CLICK_WEBHOOK_SECRET`             | Secret           | `click-wh-secret`   | Click payment webhook secret key        |
| `ADYEN_WEBHOOK_SECRET`             | Secret           | `adyen-wh-secret`   | Adyen webhook signature key             |
| `STRIPE_WEBHOOK_SECRET`            | Secret           | `stripe-wh-secret`  | Stripe webhook signature key            |
| `FISCAL_PROVIDER`                  | Enum             | ConfigMap           | `MY_SOLIQ` (Soliq OFD), `PEGASUS`       |
| `FISCAL_MY_SOLIQ_BASE_URL`         | URL              | ConfigMap           | Soliq OFD API endpoint URL              |
| `FISCAL_MY_SOLIQ_API_KEY`          | Secret           | `soliq-api-key`     | Soliq OFD Bearer authorization token    |
| `FISCAL_MY_SOLIQ_TIN`              | Secret           | `soliq-tin`         | Supplier STIR / Tax ID                  |
| `FISCAL_MY_SOLIQ_PKCS12_FILE`      | File Path        | Secret Volume Mount | `/etc/certs/eds.p12` container           |
| `FISCAL_MY_SOLIQ_PKCS12_PASSWORD`  | Secret           | `soliq-pkcs12-pass` | E-IMZO certificate password             |
| `DUNNING_SMS_PROVIDER`             | Enum             | ConfigMap           | `playmobile` or `twilio`                |
| `PLAYMOBILE_LOGIN`                 | Secret           | `playmobile-login`  | PlayMobile SMS gateway login            |
| `PLAYMOBILE_PASSWORD`              | Secret           | `playmobile-pass`   | PlayMobile SMS gateway password         |
| `DUNNING_EMAIL_PROVIDER`           | Enum             | ConfigMap           | `sendgrid`                              |
| `SENDGRID_API_KEY`                 | Secret           | `sendgrid-api-key`  | SendGrid mail dispatch API token        |
| `TWILIO_ACCOUNT_SID`               | Secret           | `twilio-sid`        | Twilio account SID                      |
| `TWILIO_AUTH_TOKEN`                | Secret           | `twilio-auth-token` | Twilio authentication token             |
| `OPTIMIZER_BASE_URL`               | URL              | ConfigMap           | `http://optimizer-core:8082`            |
| `ROUTING_OSRM_URL`                 | URL              | ConfigMap           | `http://osrm:5000`                      |
| `GOOGLE_MAPS_API_KEY`              | Secret           | `google-maps-key`   | Google Routes / Places server API key   |
| `GCS_BUCKET_NAME`                  | String           | ConfigMap           | `pegasusx-prod-media`                   |
| `UPDATES_BASE_URL`                 | URL              | ConfigMap           | `https://updates.pegasusx.example.com`  |
+------------------------------------+------------------+---------------------+-----------------------------------------+
```

---

*Authoritative Codebase Infrastructure Inventory concluded. Authored by Worker M1 for the PegasusX Infrastructure Project.*
