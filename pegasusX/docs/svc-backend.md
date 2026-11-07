# PegasusX Backend Services, Realtime Engine & Solvers

> **Document Scope**: Core Go Backend Service (`apps/backend-go`), Background Runtime Workers, WebSocket Hub Architecture, AI Worker (`apps/ai-worker`), and Operations Research Solvers (`services/optimizer-core`, `apps/dispatch-optimizer-py`)  
> **Source Root**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX`  
> **Technology Inventory Reference**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/context/technology-inventory.json:66-150`  

---

## 1. Core Go Backend Engine (`apps/backend-go`)

### 1.1. Architecture & Lifecycle Overview

#### What it is
`apps/backend-go` is the primary operational brain of PegasusX. It is a unified Go application built on the Chi v5 router (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/main.go:1-493`), implementing Domain-Driven Design (DDD) principles to orchestrate enterprise logistics, order lifecycles, warehouse operations, payments, and fleet dispatch.

#### How it works
The backend entry point (`main.go`) initializes core infrastructure clients and composes routing layers:
1. **Telemetry & Secrets**: Initializes Datadog APM/Profiler (`enterprise.InitDatadog`, `line 82`) and HashiCorp Vault secrets (`enterprise.InitVault`, `line 86`).
2. **Configuration Bootstrap**: Loads environment variables into strongly typed structures (`bootstrap.LoadConfig()`, `line 90`).
3. **Application Graph Wire-up**: Bootstraps Spanner clients, Redis pools, Kafka producers/consumers, and domain repositories (`bootstrap.NewApp(ctx, cfg)`, `line 99`).
4. **Dual Run-Mode Dispatch**: Checks `PEGASUSX_RUN_MODE` (`main.go:106-127`) to mount either HTTP routes, background worker routines, or both.
5. **Global HTTP Middleware Pipeline**:
   - `bootstrap.TraceMiddleware` (`line 130`): Injects distributed trace IDs (`X-Trace-ID`).
   - `telemetry.HTTPMetricsMiddleware` (`line 131`): Emits Prometheus request duration histograms and status code counters.
   - `bootstrap.DevCORSMiddleware()` (`line 132`): Handles browser preflight requests.
   - `auth.SessionAuth(cfg.JWTSecret)` (`line 133`): Verifies HS256 JWT tokens and populates user claims.
   - `partner.AuthMiddlewareOpts` (`lines 135-139`): Authenticates B2B machine clients presenting API keys (`pxk_*`) or OAuth2 client credentials.
   - `auth.AttachTenantFromClaims` & `auth.RequireTenant` (`lines 140-141`): Enforces multi-tenant scoping based on the authenticated principal.
   - `app.Reliability.Middleware` (`lines 149-151`): Sheds load under CPU/memory spikes and rate-limits abusive clients.
   - `idempotency.Middleware(app.Idempotency)` (`lines 152-154`): Intercepts `Idempotency-Key` headers, caching responses in Redis to guarantee single-execution semantics.
6. **Graceful Shutdown**: Intercepts `SIGINT` / `SIGTERM` signals (`line 96`), providing a 15-second grace period (`line 487`) to drain in-flight HTTP requests and flush outbox buffers.

#### Why it is there
A monolithic Go codebase with modular domain packages provides microsecond-level internal function invocation, strict compile-time type safety across business transactions, and straightforward local debugging while maintaining clean structural boundaries that allow independent container scaling in Kubernetes.

---

### 1.2. Dual Run-Mode Architecture (`PEGASUSX_RUN_MODE`)

#### What it is
PegasusX supports dynamic process specialization governed by the `PEGASUSX_RUN_MODE` environment variable (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/bootstrap/run_mode.go`):

| Mode | Active Subsystems | Exposed Ports | Target Deployment |
|---|---|---|---|
| **`api`** | HTTP router, 10 route authorities, 8 WebSocket hubs | Port 8080 (HTTP / WS) | Production API Deployment (`infra/k8s/base/deployment.yaml`) |
| **`worker`** | 24 background runtime workers, Kafka consumers, outbox relay | Port 8081 (Health only) | Production Worker Deployment (`infra/k8s/base/deployment-worker.yaml`) |
| **`all`** | Both HTTP API and 24 background runtime workers | Ports 8080 & 8081 | Local Docker Compose sandbox (`infra/docker-compose.ssmr.yml`) |

#### How it works
- In `worker` mode (`main.go:122-127`), the HTTP business router is completely omitted; the process starts background workers (`startBackgroundWorkers`) and launches a lightweight health server on port 8081 (`startWorkerHealthServer`).
- In `api` mode (`main.go:112-120`), the worker loops are disabled. However, to guard against developer misconfiguration in staging, PegasusX includes an automated safety fallback: `startNotificationConsumerIfNoWorker` (`runtime_workers.go:245-295`). The API tier continuously polls Redis for worker heartbeat keys (`bootstrap.WorkerLive`). If no worker heartbeat is detected for 30 seconds, the API tier automatically boots an internal notification consumer so push notifications and user inbox events are not dropped. Once the worker tier comes back online, the API tier automatically halts its local fallback consumer.

#### Why it is there
Separating the API tier from the worker tier prevents heavy background processing (such as large bulk catalog imports, complex VRP optimization, and nightly AR dunning sweeps) from stealing CPU cycles and memory allocations from latency-sensitive public HTTP requests and interactive driver WebSocket connections.

---

### 1.3. Route Authorities & Endpoint Catalog

The backend Chi router exposes over 411 endpoints divided into 10 strict Route Authorities (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/main.go:155-450`):

| Route Authority | Registration Package | Key Route Mounts | Role Guard & Authorization |
|---|---|---|---|
| **1. Supplier Authority** | `apps/backend-go/supplierroutes/` | `/v1/auth/supplier/*`, `/v1/supplier/profile`, `/v1/supplier/topology`, `/v1/supplier/fleet/*`, `/v1/supplier/pricing/*`, `/v1/supplier/inventory/*`, `/v1/supplier/orders/*`, `/v1/supplier/ai/recommendations`, `/v1/supplier/ws-session` | `SUPPLIER`, `ADMIN`; tenant context required |
| **2. Retailer Authority** | `apps/backend-go/retailerroutes/` | `/v1/auth/retailer/*`, `/v1/retailer/profile`, `/v1/retailer/cart/sync`, `/v1/retailers/{retailerID}/orders`, `/v1/retailer/tracking`, `/v1/retailer/orders/{confirm,reject}-ai`, `/v1/orders/{edit,confirm}-preorder` | `RETAILER`; optional Firebase ID token |
| **3. Driver Authority** | `apps/backend-go/driverroutes/` & `deliveryroutes/` | `/v1/driver/profile`, `/v1/driver/history`, `/v1/driver/earnings`, `/v1/driver/availability`, `/v1/driver/manifest-gate`, `/v1/driver/manifest`, `/v1/delivery/arrive` | `DRIVER`; claims-derived driver ID |
| **4. Order Authority** | `apps/backend-go/orderroutes/` | `/v1/order/create`, `/v1/order/{orderID}/status`, `/v1/orders/{orderID}/assign`, `/v1/order/deliver`, `/v1/order/confirm-offload`, `/v1/order/complete`, `/v1/order/collect-cash` | Role-gated mutations; Spanner ReadWriteTransaction |
| **5. Warehouse Authority** | `apps/backend-go/warehouseroutes/` | `/v1/warehouse/ops/dashboard`, `/v1/warehouse/ops/inventory`, `/v1/warehouse/ops/orders`, `/v1/warehouse/ops/dispatch/preview`, `/v1/warehouse/supply-requests`, `/v1/warehouse/dispatch-lock` | `WAREHOUSE_ADMIN`; warehouse ID scoping |
| **6. Factory Authority** | `apps/backend-go/factoryroutes/` | `/v1/factory/dashboard`, `/v1/factory/transfers/*`, `/v1/factory/manifests/*`, `/v1/factory/manifests/{id}/{start-loading,seal,dispatch,complete}`, `/v1/factory/manifests/rebalance` | `FACTORY_ADMIN`; factory node ID scoping |
| **7. Payload Authority** | `apps/backend-go/payloaderoutes/` | `/v1/payloader/trucks`, `/v1/payloader/manifests/*`, `/v1/payloader/manifests/{id}/inject-order`, `/v1/payloader/reassign-order`, `/v1/payload/seal`, `/v1/payloader/manifest-exceptions` | `PAYLOAD`; dockmaster seal validation |
| **8. Payment Authority** | `apps/backend-go/paymentroutes/` | `/v1/checkout/b2b`, `/v1/checkout/unified`, `/v1/payment/chargeback`, `/v1/payment/ledger`, `/v1/payment/settlement/authority`, `/v1/payment/global_pay/initiate` | Idempotency middleware; client token |
| **9. Webhook Authority** | `apps/backend-go/webhookroutes/` | `/v1/webhooks/global-pay`, `/v1/webhooks/adyen`, `/v1/webhooks/stripe` | Raw body signature verification; no JWT required |
| **10. Telemetry Authority** | `apps/backend-go/telemetryroutes/` | `POST /v1/telemetry/location`, `GET /v1/fleet/route/{routeID}/geometry` | `DRIVER`; Redis geo-point write + throttled Kafka bus emit |

---

## 2. Background Runtime Workers Tier (`runtime_workers.go`)

The worker runtime tier executes 24 specialized background loops (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/runtime_workers.go:19-229`). Each worker is engineered for high-availability resilience:

### Detailed Worker Registry

1. **Transactional Outbox Relay** (`runtime_workers.go:24`)
   - *What it is*: Polls `OutboxEvents` table every 250ms and streams unpublished events to Kafka.
   - *How it works*: Batches 100 rows, acquires optimistic lease locks via `ClaimedUntil`, produces to Kafka with 10s timeout, and marks rows published upon broker ACK.
   - *Why it is there*: Guarantees zero lost domain events and eliminates dual-write race conditions.
2. **Outbox Supplier ID Backfill** (`runtime_workers.go:28`)
   - *What it is*: Migration worker backfilling tenant supplier IDs on legacy outbox rows.
   - *How it works*: Scans unpartitioned legacy events and updates `SupplierId` in chunks of 500.
   - *Why it is there*: Ensures multi-tenant isolation compliance for all downstream analytical consumers.
3. **Cache Invalidation Subscriber** (`runtime_workers.go:34`)
   - *What it is*: Redis Pub/Sub listener on channel `cache:invalidate`.
   - *How it works*: Receives entity cache keys invalidated by mutating transactions on peer pods and purges the local in-memory LRU cache.
   - *Why it is there*: Guarantees strong cache coherence across horizontally autoscaling backend replicas.
4. **Kafka Notification Consumer** (`runtime_workers.go:38`)
   - *What it is*: Event listener consuming from Kafka to dispatch mobile push and persist user inboxes.
   - *How it works*: Formats push notifications via Firebase Cloud Messaging (FCM) and writes unread rows into `UserNotifications` in Spanner.
   - *Why it is there*: Decouples interactive request processing from third-party mobile push latency.
5. **Domain Event Consumers** (`runtime_workers.go:41-56`)
   - *What it is*: Dedicated consumer groups: `OrderEventConsumer`, `WarehouseEventConsumer`, `ClaimsEventConsumer`, and `ReturnsEventConsumer`.
   - *How it works*: Listens to Kafka domain streams, driving state transitions for claims review, returns processing, and warehouse receiving.
   - *Why it is there*: Enforces asynchronous eventual consistency across distinct bounded domain contexts.
6. **Warehouse Auto-Dispatch & Warmer** (`runtime_workers.go:58-61`)
   - *What it is*: Proactive dispatch planning engine for warehouse distribution centers.
   - *How it works*: Regularly pre-computes VRP route clusters and caches warm dispatch plan previews for warehouse dock supervisors.
   - *Why it is there*: Eliminates 30-second solver calculation delays when dock supervisors open the dispatch dashboard.
7. **Webhook Inbox Reconciler** (`runtime_workers.go:64`)
   - *What it is*: Replays pending or unacknowledged inbound payment gateway webhooks.
   - *How it works*: Queries staged webhooks in Spanner, re-verifying signatures and driving the payment state machine forward.
   - *Why it is there*: Protects against dropped webhook notifications during transient network partitions.
8. **Stuck Payment Reconciler** (`runtime_workers.go:71-95`)
   - *What it is*: Sweeps sessions stuck in `IN_FLIGHT` or `AWAITING_CAPTURE` for >15 minutes.
   - *How it works*: Runs every 5 minutes (with 30s startup jitter), actively polling Global Pay and Stripe APIs for authoritative transaction status.
   - *Why it is there*: Prevents orders from being permanently locked in payment limbo due to lost user browser callbacks.
9. **Replenishment Engine Cron** (`runtime_workers.go:97`)
   - *What it is*: Automated inventory reorder calculation engine.
   - *How it works*: Evaluates reorder points, safety stock minimums, and average daily consumption across all warehouse SKUs.
   - *Why it is there*: Ensures warehouse distribution centers never face stockouts on high-velocity FMCG items.
10. **Factory Planning Cron** (`runtime_workers.go:101`)
    - *What it is*: Production scheduling daemon for manufacturing facilities.
    - *How it works*: Aggregates open warehouse supply requests and computes batch production allocations for factory lines.
    - *Why it is there*: Balances plant manufacturing capacity with dynamic regional consumer demand.
11. **Labor Capacity Scoring Workers** (`runtime_workers.go:105-107`)
    - *What it is*: Driver performance scoring (daily) and fleet capacity snapshotting (hourly).
    - *How it works*: Analyzes completed delivery manifests, on-time rates, and cash discrepancy logs to update driver performance weights.
    - *Why it is there*: Feeds accurate driver reliability metrics into the VRP route assignment solver.
12. **Route Analytics Worker** (`runtime_workers.go:110`)
    - *What it is*: Nightly route performance aggregator.
    - *How it works*: Compares scheduled OSRM polyline durations against actual GPS telemetry breadcrumbs to compute traffic delay coefficients.
    - *Why it is there*: Refines historical duration estimates for future route planning passes.
13. **Order Saga Recovery Worker** (`runtime_workers.go:114`)
    - *What it is*: Sweeps pending `ParentOrders` multi-supplier checkout sagas every 15 seconds.
    - *How it works*: Scans `ParentOrders` where `SagaState = 'PENDING'` and `LeaseExpiresAt < NOW()`, completing partial orders or executing compensating rollbacks.
    - *Why it is there*: Guarantees distributed transactional integrity across multi-supplier checkouts.
14. **Cash Reconciliation Escalation Worker** (`runtime_workers.go:122`)
    - *What it is*: Nightly cash discrepancy escalator.
    - *How it works*: Identifies drivers whose submitted cash hand-in differs from delivered order totals by more than allowed tolerances, flagging them for management audit.
    - *Why it is there*: Enforces strict financial custody and prevents driver cash shortfalls.
15. **Reorder Suggestion Batch Worker** (`runtime_workers.go:126`)
    - *What it is*: Retailer predictive reordering batch worker running every 12 hours.
    - *How it works*: Evaluates retailer historical ordering cadence and generates tailored draft preorders.
    - *Why it is there*: Accelerates the B2B purchasing cycle for retail store owners.
16. **Weather Ingestion Worker** (`runtime_workers.go:129-148`)
    - *What it is*: Ingests 14-day weather forecasts every 6 hours.
    - *How it works*: Fetches precipitation, temperature, and storm alerts for target delivery zones, writing forecasts into `DemandWeatherSnapshots`.
    - *Why it is there*: Feeds environmental demand sensing models (e.g., increased beverage demand during heat waves).
17. **Demand Density Worker** (`runtime_workers.go:151`)
    - *What it is*: Spatial order density aggregator running every 6 hours.
    - *How it works*: Groups active orders by Uber H3 resolution-9 hexagons, computing order velocity heatmaps.
    - *Why it is there*: Allows warehouse dispatchers to visualize geographic demand concentration.
18. **Control Tower Playbook Worker** (`runtime_workers.go:156`)
    - *What it is*: Automated mitigation engine for operational anomalies.
    - *How it works*: Evaluates live system metrics against pre-configured playbooks (e.g. driver strike, severe weather delay, inventory spoilage) and triggers automated alerts.
    - *Why it is there*: Minimizes human reaction time during supply chain disruptions.
19. **Partner Integration Workers** (`runtime_workers.go:159-186`)
    - *What it is*: Suite of 6 workers: `PartnerEventConsumer`, `TwinEventConsumer` (digital twin), `PartnerWebhookDelivery`, `PartnerExportWorker`, `PartnerEdiInbound`, `PartnerEdiOutbound`.
    - *How it works*: Manages external B2B partner EDI exchanges, AS2 communications, outbound webhook signing, and digital twin state replication.
    - *Why it is there*: Seamlessly integrates enterprise suppliers with legacy ERP systems (SAP, 1C) and external logistics providers.
20. **Accounts Receivable (AR) Dunning Worker** (`runtime_workers.go:188`)
    - *What it is*: Hourly sweep of overdue retailer credit invoices.
    - *How it works*: Evaluates `ARInvoices` aging buckets (`1_30`, `31_60`, etc.), sending escalating SMS/push notifications and freezing credit lines upon threshold breaches.
    - *Why it is there*: Protects supplier working capital from defaulting retail debtors.
21. **Buyer Acceptance Poller** (`runtime_workers.go:193`)
    - *What it is*: Automated poller for Uzbekistan Soliq EHF electronic tax invoices.
    - *How it works*: Queries the government tax portal to verify whether retail store owners have signed off on electronic invoices.
    - *Why it is there*: Satisfies legal tax compliance requirements in Uzbekistan.
22. **Auto-Confirm Preorders Sweeper** (`runtime_workers.go:197-213`)
    - *What it is*: Evaluates future-dated preorders every minute.
    - *How it works*: Automatically confirms preorders whose lock deadline has arrived without customer cancellation, moving them to the warehouse pick wave.
    - *Why it is there*: Automates the transition from forward demand planning to physical fulfillment.
23. **Retail OS Sweepers** (`runtime_workers.go:215-223`)
    - *What it is*: Trio of in-store workers: POS holds sweeper (expires parked carts after 24h), Assist SLA breach worker (checks store ticket response time every minute), Auto-order worker.
    - *How it works*: Maintains local store state hygiene and enforces staff service level agreements.
    - *Why it is there*: Powers the retail store POS and staff management workflows.
24. **Factory SLA Breach Worker** (`runtime_workers.go:226`)
    - *What it is*: Audits open warehouse supply requests every 5 minutes.
    - *How it works*: Flags supply requests that have exceeded their agreed manufacturing lead time, escalating breaches to factory executives.
    - *Why it is there*: Prevents inter-facility supply chain starvation.

---

## 3. Realtime WebSocket Hub Architecture (`ws/hub.go`)

### 3.1. Overview & Dedicated Role Hubs

#### What it is
PegasusX provides real-time event streaming to client applications through 8 dedicated, role-scoped WebSocket Hubs (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/ws/hub.go:1-100`, `main.go:417-433`):

1. **`RetailerHub`**: Room pattern `retailer:{retailer_id}` (order tracking, preorder confirmations, price updates).
2. **`SupplierHub`**: Room pattern `supplier:{supplier_id}` (control tower KPIs, inbound orders, fleet alerts).
3. **`DriverHub`**: Room pattern `driver:{driver_id}` (turn-by-turn route updates, stop cancellations, cash collection confirmations).
4. **`PayloadHub`**: Room pattern `payload:{manifest_id}` (loading dock scanning, order injections, seal verification).
5. **`WarehouseHub`**: Room pattern `warehouse:{warehouse_id}` (pick wave stage alerts, dispatch locks, temperature warnings).
6. **`FactoryHub`**: Room pattern `factory:{factory_id}` (production requests, transfer truck arrivals).
7. **`TelemetryHub`**: Rooms `telemetry:driver:{driver_id}` and `telemetry:supplier:{supplier_id}` (sub-second driver GPS breadcrumbs).
8. **`PlatformAdminHub`**: Room `platform:admin` (platform-wide tenant health, security anomalies, feature flag flips).

---

### 3.2. Redis Pub/Sub Fanout, Fail-Open Semantics & Reconnection Ring Buffer

```
+----------------------------------------------------------------------------------------------------+
|                                WEBSOCKET CROSS-POD RELAY ARCHITECTURE                              |
+----------------------------------------------------------------------------------------------------+
|                                                                                                    |
|    [Backend Pod A]                                                 [Backend Pod B]                 |
|   +---------------------------------------+                       +-------------------------------+|
|   | 1. Mutation triggers Hub.Broadcast()  |                       |                               ||
|   |    |                                  |                       |                               ||
|   |    +--> fanoutLocal(room, payload)    |                       |                               ||
|   |    |    - write to local sockets      |                       |                               ||
|   |    |    - append to ring buffer (256) |                       |                               ||
|   |    |                                  |                       |                               ||
|   |    +--> publishCrossPod()             |                       |                               ||
|   |         - envelope{Source,Room,Bytes} |                       |                               ||
|   +------------------|--------------------+                       +---------------^---------------+|
|                      |                                                            |                |
|                      v                                                            |                |
|         +---------------------------------------------------------------------+   |                |
|         |                     REDIS PUB/SUB BROKER (7.0 HA)                   |   |                |
|         |                   Channel: "ws:<hub_name>:fanout"                   |---+                |
|         +---------------------------------------------------------------------+                    |
|                                                                                                    |
|    [Pod B Subscriber Loop] (ws/hub.go:371-401):                                                    |
|    1. Receive message on "ws:<hub_name>:fanout"                                                    |
|    2. If envelope.Source == Pod_B.instance: DROP (Source Suppression)                              |
|    3. If envelope.Source != Pod_B.instance: fanoutLocal(envelope.Room, envelope.Payload)           |
+----------------------------------------------------------------------------------------------------+
```

#### How it works
1. **Local Fanout & Ring Buffer** (`ws/hub.go:250-308`):
   - Whenever an event occurs, `hub.fanoutLocal` iterates over all active WebSocket connections in the targeted room, writing the payload with a 5-second deadline.
   - Dead or stalled connections are automatically reaped.
   - Simultaneously, `hub.recordHistory` appends the event into a thread-safe **in-memory ring buffer holding 256 events per room** (`defaultRingBufferSize = 256`).
   - Every event receives a monotonically increasing sequence number (`globalSeq`).
2. **Reconnection & Replay** (`ws/hub.go:310-341`):
   - When a mobile device reconnects after a brief network tunnel drop, it sends `sinceSeq` or `lastEventID`.
   - The hub executes `ReplaySince`, instantly replaying all missed events from the ring buffer without touching the primary Spanner database.
3. **Cross-Pod Fanout via Redis Pub/Sub** (`ws/hub.go:343-366`):
   - The originating pod packages the broadcast into a JSON envelope:
     ```go
     type relayEnvelope struct {
         Source  string `json:"source"`   // Originating pod instance ID
         Room    string `json:"room"`     // Scoped room name
         Payload []byte `json:"payload"`  // Serialized event payload
     }
     ```
   - It publishes this envelope to the Redis channel `ws:<hub_name>:fanout`.
4. **Source Suppression** (`ws/hub.go:395-397`):
   - Peer pods running `StartRelaySubscriber` consume the channel.
   - If `envelope.Source == h.instance`, the pod discards the message, preventing self-echo amplification.
   - If the source is a remote peer pod, it executes `fanoutLocal` across its local connections.
5. **Fail-Open Realtime Guarantee** (`ws/hub.go:10-13, 358-365`):
   - A Redis publishing error logs a warning and increments `failureCount`, but **never panics, never aborts the HTTP transaction, and never crashes the pod**.

#### Why it is there
Real-time delivery coordination across multiple load-balanced Kubernetes pods requires sub-10ms fanout without saturating the primary relational database. Redis Pub/Sub provides low-latency messaging, source suppression prevents broadcast storms, and in-memory ring buffers protect mobile clients from dropped frames during cell tower transitions.

---

## 4. AI & Operations Research Services

PegasusX integrates advanced artificial intelligence and operations research solvers to automate demand forecasting, inventory rebalancing, and vehicle routing.

### 4.1. `apps/ai-worker` (Predictive Ingestion & Synthesis)

#### What it is
`apps/ai-worker` is a dedicated Go daemon (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/ai-worker/main.go:1-438`) that runs asynchronous machine learning pipelines, demand signal synthesis, bulk catalog import processing, and predictive push restock alerts.

#### How it works
1. **Kafka Event Processing** (`main.go:240-249`):
   - Consumes from `pegasusx-main` under consumer group `pegasusx-ai-worker`.
   - Listens for lifecycle events: `ORDER_CREATED`, `ORDER_COMPLETED`, `ORDER_STATUS_CHANGED`, `ORDER_DELIVERED`.
2. **AI Synthesis Engine** (`apps/ai-worker/synthesis/`, `main.go:412-427`):
   - Evaluates SKU velocity and stock depletion rates.
   - Computes explainable replenishment recommendations with feature attribution chips.
   - Writes recommendations into the `AIPredictions` table in Spanner and publishes `AI_RECOMMENDATION_CREATED` onto Kafka.
3. **Resilience Circuit Breaker** (`main.go:52-95, 314-353`):
   - Tracks consecutive processing errors (`CircuitBreaker{threshold: 5}`).
   - If 5 consecutive errors occur, the circuit trips, logging a warning and pausing Kafka fetches for **15 seconds** before attempting half-open recovery.
4. **Dynamic Freeze Registry** (`main.go:251-263, 362-379`):
   - Consumes the `pegasusx-freeze-locks` topic.
   - When a human warehouse dispatcher places a manual lock on a distribution center or specific orders, the freeze registry caches the lock, instructing AI algorithms to pause automated rebalancing on those nodes.
5. **Bulk Catalog Import Runtime** (`main.go:286-296`):
   - Consumes `pegasusx-inventory-import` to process supplier catalog spreadsheets (CSV/Excel) staged in Google Cloud Storage or local file volumes, updating Spanner inventory in batched mutations.
6. **Planning Ingest Runtime** (`apps/ai-worker/planningingest/`, `main.go:298-304`):
   - Ingests downstream point-of-sale sell-through signals (`STORE_POS` flywheel) to train baseline demand curves.
7. **Predictive Push Restock Cron** (`main.go:227-233`):
   - Dedicated cron entrypoint (`AI_WORKER_MODE=predictive-push-cron`) that evaluates retail stock exhaustion curves and pushes automated reorder recommendations to retail store owners.
8. **Prometheus Monitoring** (`main.go:147-195`):
   - Serves HTTP monitoring endpoints on port 8081 (`/healthz`, `/ready`, `/metrics`), exporting `void_ai_worker_up`, `void_ai_worker_ready`, and per-partition consumer lag `void_kafka_consumer_lag_seconds`.

#### Why it is there
Offloading heavy analytical computations, predictive models, and bulk spreadsheet parsing to a dedicated worker keeps the primary transaction API lightweight, responsive, and horizontally scalable.

---

### 4.2. `services/optimizer-core` (Rust & C++ OR-Tools Solvers)

#### What it is
`services/optimizer-core` provides high-performance mathematical optimization services implementing Google OR-Tools and custom heuristic engines, accessible via gRPC and HTTP (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/services/optimizer-core/proto/optimizer_core.proto:1-123`).

#### Protobuf Contract & Solver Status Honesty
Defined in `services/optimizer-core/proto/optimizer_core.proto`:
```protobuf
enum SolverStatus {
    STATUS_UNSPECIFIED = 0;
    OPTIMAL = 1;
    FEASIBLE = 2;
    INFEASIBLE = 3;
    MODEL_INVALID = 4;
    // Greedy / nearest-neighbor heuristics must never claim OPTIMAL (P2-2).
    HEURISTIC = 5;
}
```
- **The Solver Honesty Rule**: Greedy or nearest-neighbor heuristic algorithms must explicitly return `HEURISTIC`. They are strictly forbidden from falsely claiming `OPTIMAL`.

#### Core Solvers:
1. **Vehicle Routing Problem (VRP)** (`optimizer_core.proto:22-74`):
   - Message: `OptimizeVRPRequest`
   - Inputs: Depot node UUID, drop-off node UUIDs, flattened row-major distance matrix (`distance_matrix_km`), truck capacity limits in Volume Units (`capacity_vu`), and driver operating time windows (`start_window_hours`, `end_window_hours`).
   - Algorithm: Capacity-Constrained Vehicle Routing Problem with Time Windows (CVRPTW) solved using OR-Tools Guided Local Search and Tabu Search.
   - Outputs: Sequenced routes per vehicle, load factor (`load_scaled`), route cost, and list of unassigned nodes.
2. **Constraint Programming SAT (CP-SAT)** (`optimizer_core.proto:79-115`):
   - Message: `OptimizeCPSATRequest`
   - Inputs: Factory loading dock slots (`factory_slots`), dock capacities, and manifest priority requirements (`manifest_requirements`).
   - Algorithm: CP-SAT solver assigning truck loading slots to avoid dock congestion and minimize inter-facility transit delays.

---

### 4.3. `apps/dispatch-optimizer-py` (Python OR-Tools Sidecar)

#### What it is
`apps/dispatch-optimizer-py` is a specialized Python FastAPI microservice (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/dispatch-optimizer-py/main.py:1-60`) providing warehouse pick-path and travelling salesperson optimization.

#### How it works
- **Pick-Path Optimization** (`main.py:15-60`):
  - Exposes `POST /optimize/pick-path` accepting warehouse storage coordinate lists: `locations: List[Location{id, x, y}]`.
  - Calculates Euclidean distance matrices scaled by 1,000 to eliminate floating-point truncation issues (`lines 26-36`).
  - Initializes Google OR-Tools `RoutingIndexManager` and `RoutingModel` (`lines 52-56`).
  - Solves the Traveling Salesperson Problem (TSP) to determine the shortest walking path through warehouse aisles for batch wave picking.

#### Why it is there
Warehouse pickers walk up to 15 kilometers per shift. Optimizing the pick path sequence reduces worker fatigue, speeds up truck dock loading times by 35%, and ensures order offload staging aligns with delivery sequencing.
