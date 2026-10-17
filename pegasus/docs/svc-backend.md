# Pegasus Backend Services & Operations Research Blueprint

> **Ecosystem**: Pegasus Backend & Algorithmic Core  
> **Source Path**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/BACKEND_SERVICES.md`  
> **Referenced Codebases**:
> - `apps/backend-go`: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go`
> - `apps/ai-worker`: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/ai-worker`
> - `services/optimizer-core`: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/optimizer-core`
> - `services/deep-agents`: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/deep-agents`

---

## 1. Composition Root & Lifecycle Orchestration

### 1.1 What It Is
The transactional core of Pegasus is built in Go 1.25 using a strict **Composition Root** pattern located at `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/bootstrap/app.go` (`NewApp()` in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/bootstrap/new.go`).

### 1.2 How It Works
Rather than scattering client creation, global singletons, or uncoordinated connections across handlers:
- `bootstrap.NewApp(ctx, cfg)` initializes all database connection pools (Google Cloud Spanner), caching layers (Redis Memorystore handle), message queues (Apache Kafka writers/readers), Firebase Auth clients, GCS clients, and in-memory WebSocket hubs in a strictly verified dependency order.
- The resulting `*bootstrap.App` struct aggregates all initialized clients, services, and middleware.
- `apps/backend-go/main.go` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/main.go`) acts solely as the runtime orchestrator:
  1. Loads fail-closed environment variables via `packages/config` (`main.go:142`).
  2. Initializes authentication secrets via `auth.Init` (`main.go:146`).
  3. Constructs the application graph via `bootstrap.NewApp` (`main.go:151`).
  4. Instantiates a single `chi.NewRouter()` and applies global telemetry (`TraceMiddleware`) and CORS (`main.go:115, 195`).
  5. Mounts domain subrouters by passing typed `Deps` structures containing only the specific services each router requires.
  6. Starts background temporal crons and Kafka event consumers (`main.go:636-737`).
  7. Arms SIGTERM/SIGINT listeners for a 30-second graceful teardown, ensuring OpenTelemetry span flushing (`TracerShutdown`) before exit (`main.go:931-965`).

### 1.3 Why It Is There
Decoupling handler routing from dependency instantiation prevents cyclic dependencies in Go, allows unit test suites to substitute mock clients effortlessly, and ensures that critical teardown steps—such as flushing pending OpenTelemetry traces and draining in-flight Kafka writers—execute deterministically on pod shutdown.

---

## 2. Chi Router & Specialized Domain Subrouters

### 2.1 What It Is
The HTTP routing surface in `apps/backend-go` is built on `github.com/go-chi/chi/v5` and modularized into over 20 specialized domain subrouters. Each subrouter encapsulates its own URL namespaces, parameter parsing, authorization checks, and service invocations.

### 2.2 Functional Subrouter Directory

#### 1. Authentication & Security Subrouter
- **Source**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/authroutes/routes.go`
- **What It Does**: Handles credential authentication, token refreshes, phone OTP verifications, and multi-persona onboarding for all 5 roles.
- **Key Endpoints**: `POST /v1/auth/login`, `POST /v1/auth/refresh`, `POST /v1/auth/supplier/register`, `POST /v1/auth/retailer/login`, `POST /v1/auth/driver/login`, `POST /v1/auth/warehouse/login`, `POST /v1/auth/factory/login`, `POST /v1/auth/payloader/login`, `POST /debug/mint-token`.
- **Why It Is There**: Consolidates authentication while enforcing rate limits (`cache.AuthRateLimit()`) and single-device login bindings (`DeviceFingerprints`).

#### 2. Order Core & Retailer Subrouters
- **Sources**:
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/orderroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/retailerroutes/routes.go`
- **What It Does**: Manages the lifecycle of customer orders, cart synchronization, checkout execution, preorder approvals, and doorstep delivery handshakes.
- **Key Endpoints**: `POST /v1/order/create`, `GET /v1/orders/{id}`, `POST /v1/order/deliver`, `POST /v1/order/validate-qr`, `POST /v1/order/confirm-offload`, `POST /v1/order/collect-cash`, `POST /v1/retailer/cart/sync`, `POST /v1/orders/request-cancel`.
- **Why It Is There**: Enforces the deterministic order state machine (`PENDING` → `LOADED` → `IN_TRANSIT` → `ARRIVING` → `ARRIVED` → `COMPLETED`), ensuring orders cannot transition illegally.

#### 3. Delivery Edge-Case Subrouter
- **Source**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/deliveryroutes/routes.go`
- **What It Does**: Handles exceptional operational conditions encountered by drivers at physical retail storefronts.
- **Key Endpoints**:
  - `POST /v1/delivery/shop-closed`: Initiates Shop-Closed Protocol when store is dark (`main.go:374`).
  - `POST /v1/delivery/bypass-offload`: Admin-issued 6-digit OTP bypass when camera/QR scanner fails.
  - `POST /v1/delivery/negotiate`: Doorstep quantity down-adjustment and price bargaining.
  - `POST /v1/delivery/credit-delivery`: Offloading without cash for pre-approved credit accounts.
  - `POST /v1/delivery/missing-items`: Loading discrepancy logging in `ManifestExceptions`.
  - `POST /v1/delivery/split-payment`: Combined electronic gateway and physical cash settlement.
- **Why It Is There**: Real-world emerging market delivery encounters numerous edge cases; hardcoding fail-closed rejection would halt fleet productivity.

#### 4. Supplier Operations & Catalog Subrouters
- **Sources**:
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/suppliercatalogroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/suppliercoreroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/supplierlogisticsroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/supplierplanningroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/supplieroperationsroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/supplierinsightsroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/supplierroutes/routes.go`
- **What It Does**: Full merchant admin suite: tiered SKU pricing, dynamic B2B price overrides, driver fleet assignments, picking manifests, demand tournament forecasts, and delivery territory polygon creation.
- **Key Endpoints**: `POST /v1/supplier/products`, `POST /v1/supplier/pricing/override`, `POST /v1/supplier/manifest/generate`, `POST /v1/supplier/analytics/forecast/tournament`, `POST /v1/supplier/zones`.
- **Why It Is There**: Gives suppliers complete autonomy over wholesale catalog rules, logistics parameters, and predictive restocking.

#### 5. Factory & Warehouse Supply Chain Subrouters
- **Sources**:
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/factoryroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/warehouseroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/payloaderroutes/routes.go`
- **What It Does**: Coordinates internal supply movements from manufacturing plants to regional fulfillment hubs: internal transfer creation, quality control signoffs, dock bay scanning, digital seal validation, and emergency transfers.
- **Key Endpoints**: `POST /v1/factory/transfers/create`, `POST /v1/warehouse/supply-requests`, `POST /v1/warehouse/transfers/force-receive`, `POST /v1/payload/seal`, `POST /v1/warehouse/dispatch-lock`.
- **Why It Is There**: Decouples external customer deliveries from internal inter-facility replenishments while preserving traceability.

#### 6. Treasury, Payments & Webhooks Subrouters
- **Sources**:
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/treasury/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/paymentroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/webhookroutes/routes.go`
- **What It Does**: Executes multi-gateway transactions (Adyen, Global Pay, Airwallex, Payme, Click), registers card tokens, processes webhooks, snapshots master invoices, and compiles settlement slices.
- **Key Endpoints**: `POST /v1/checkout/unified`, `POST /v1/payment/chargeback`, `POST /v1/webhooks/adyen`, `POST /v1/webhooks/global_pay`, `GET /v1/treasury/settlement-slices`.
- **Why It Is There**: Shields downstream services from raw gateway protocol nuances and guarantees financial immutability.

#### 7. Proximity, Telemetry & Synchronization Subrouters
- **Sources**:
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/proximityroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/telemetryroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/sync/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/driverroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/fleetroutes/routes.go`
- **What It Does**: Live GPS ingestion, warehouse H3 coverage queries, offline mobile sync (Desert Protocol), driver earnings, and route reordering.
- **Key Endpoints**: `POST /v1/driver/telemetry`, `POST /v1/sync/batch`, `GET /v1/proximity/warehouses/validate-coverage`.
- **Why It Is There**: Allows vehicles to traverse connectivity dead-zones ("deserts") and sync queued deliveries reliably upon reconnecting.

#### 8. Admin, Infrastructure & Simulation Subrouters
- **Sources**:
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/adminroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/infraroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/simroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/airoutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/userroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/entityresolutionroutes/routes.go`
- **What It Does**: Platform health checks (`/v1/health`), Prometheus metrics (`/metrics`), DLQ replays, AI prediction feedback, notification inbox management, and administrative load simulation.
- **Why It Is There**: Provides complete operational visibility and platform lifecycle administration.

---

## 3. The 13 Background Cron Engines (`cron.go`)

### 3.1 What It Is
Pegasus runs 13 specialized temporal background workers declared in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/cron.go`. These workers execute periodic sweeps to guarantee eventual consistency, clean orphaned records, enforce service agreements, and drive automated replenishment.

### 3.2 Complete Technical Breakdown of All 13 Crons

```
                    PEGASUS 13 TEMPORAL CRON ENGINES
 ┌──────────────────────────────────────┬──────────────────────────────────────┐
 │ 1. StartAwakener (Hourly)            │ 8. StartAutoConfirmSweeper (Periodic)│
 │    Converts predictions to preorders │    Auto-approves unread preorders    │
 ├──────────────────────────────────────┼──────────────────────────────────────┤
 │ 2. StartScheduledOrderPromoter (1h)  │ 9. StartNotificationExpirer (Daily)  │
 │    Promotes SCHEDULED -> PENDING     │    Soft-deletes stale notifications  │
 ├──────────────────────────────────────┼──────────────────────────────────────┤
 │ 3. StartGlobalPaySweeper (Periodic)  │ 10. StartPullMatrixAggregator (4h)   │
 │    Reconciles expired payment sessions│    Calculates inventory burn rates  │
 ├──────────────────────────────────────┼──────────────────────────────────────┤
 │ 4. StartPaymentSessionExpirer (15m)  │ 11. StartFactorySLAMonitor (30m)     │
 │    Expires abandoned carts/checkouts │    Audits delayed factory transfers  │
 ├──────────────────────────────────────┼──────────────────────────────────────┤
 │ 5. StartStaleOrderAuditor (15m)      │ 12. StartCurrentLoadReset (24h)      │
 │    Detects stuck transit (>12h)      │    Resets vehicle daily VU counters  │
 ├──────────────────────────────────────┼──────────────────────────────────────┤
 │ 6. StartOrphanedPredictionCleaner (1d│ 13. StartCoverageAuditor (Daily)     │
 │    Purges stale prediction items     │    Finds unserved retailer H3 cells  │
 ├──────────────────────────────────────┴──────────────────────────────────────┤
 │ 7. StartPreOrderConfirmationSweeper (Hourly)                                │
 │    Enforces T-4 confirmation lock policy and sends push/Telegram warnings   │
 └─────────────────────────────────────────────────────────────────────────────┘
```

1. **`StartAwakener` (`cron.go:32-145`)**:
   - *Frequency*: Hourly heartbeat.
   - *Logic*: Queries `AIPredictions` where `TriggerDate <= CURRENT_TIMESTAMP()` and `AutoOrderApproved = true`. Converts predicted line items into valid `Orders` in `PENDING` state, dispatches notifications via FCM and Telegram, and emits Kafka event `PRE_ORDER_NOTIFIED`.
   - *Why*: Automates inventory replenishment before the merchant realizes they are running low.

2. **`StartScheduledOrderPromoter` (`cron.go:146-229`)**:
   - *Frequency*: Every 60 minutes.
   - *Logic*: Scans `Orders` with `State = 'SCHEDULED'`. When `DeliveryDate` falls within 24 hours of fulfillment, it promotes the order to `PENDING` and triggers warehouse picking queues.
   - *Why*: Allows advance scheduling without prematurely reserving warehouse dock space or driver capacity.

3. **`StartGlobalPaySweeper` (`cron.go:230-274`)**:
   - *Frequency*: Periodic cycle.
   - *Logic*: Iterates over open `GlobalPayntSessions` older than 30 minutes, executes gateway status inquiries, and updates Spanner to `SETTLED` or `FAILED`.
   - *Why*: Recovers payment states lost due to dropped mobile network connections during 3D-Secure redirects.

4. **`StartPaymentSessionExpirer` (`cron.go:275-349`)**:
   - *Frequency*: Every 15 minutes.
   - *Logic*: Flags checkout sessions past their TTL as `EXPIRED` and pushes real-time cart expiration notices to `RetailerHub`.
   - *Why*: Releases inventory holds tied to abandoned checkout sessions.

5. **`StartStaleOrderAuditor` (`cron.go:350-563`)**:
   - *Frequency*: Every 15 minutes.
   - *Logic*: Identifies orders in `IN_TRANSIT` or `ARRIVING` for >12 hours without a driver arrival handshake. Marks them `STALE_AUDIT`, logs security alerts, and triggers automated refund compensations via `RefundService`.
   - *Why*: Catches lost trucks, stolen shipments, or abandoned deliveries automatically.

6. **`StartOrphanedPredictionCleaner` (`cron.go:564-701`)**:
   - *Frequency*: Daily at 03:00 UTC.
   - *Logic*: Performs an outer-join deletion on `AIPredictionItems` whose parent `AIPredictions` were deleted or rejected >30 days ago.
   - *Why*: Prevents unbounded storage bloat in Spanner prediction tables.

7. **`StartPreOrderConfirmationSweeper` (`cron.go:702-1086`)**:
   - *Frequency*: Every 15 minutes.
   - *Logic*: Enforces the strict T-4 confirmation lock policy. Preorders that remain unacknowledged 4 hours prior to dispatch cutoff are broadcast via FCM/Telegram warnings.
   - *Why*: Ensures truck packing manifests are not scrambled by last-minute order cancellations.

8. **`StartAutoConfirmSweeper` (`cron.go:1087-1168`)**:
   - *Frequency*: Periodic.
   - *Logic*: Consumes `AutoConfirmAt` timestamps set by the Awakener. If the retailer does not actively reject the draft within the grace period, the preorder automatically promotes to confirmed status.
   - *Why*: Reduces manual friction for busy store owners who prefer automated stock top-ups.

9. **`StartNotificationExpirer` (`cron.go:1169-1228`)**:
   - *Frequency*: Daily.
   - *Logic*: Soft-deletes read notifications older than 14 days and unread notifications older than 30 days from `Notifications`.
   - *Why*: Keeps user inbox API responses compact and fast.

10. **`StartPullMatrixAggregator` (`cron.go:1229-1259`)**:
    - *Frequency*: Every 4 hours.
    - *Logic*: Evaluates regional warehouse inventory burn velocity against incoming preorder pipelines and lead times. Invokes `PullMatrixService` to generate automated factory `InternalTransferOrders`.
    - *Why*: Eliminates regional distribution center stockouts by pulling manufacturing stock dynamically.

11. **`StartFactorySLAMonitor` (`cron.go:1260-1280`)**:
    - *Frequency*: Every 30 minutes.
    - *Logic*: Checks `InternalTransferOrders` in `APPROVED` or `LOADING` status against configured `SupplyLanes` transit time SLAs. Flags delayed movements in `FactorySLAEvents`.
    - *Why*: Prevents inter-city transfer bottlenecks from paralyzing local warehouse dispatches.

12. **`StartCurrentLoadReset` (`cron.go:1281-1324`)**:
    - *Frequency*: Daily at midnight local time (UTC+5).
    - *Logic*: Resets active daily volume unit (`CurrentLoadVU`) counters on `Vehicles` and `Drivers`.
    - *Why*: Prepares fleet capacity counters for the subsequent day's routing optimization runs.

13. **`StartCoverageAuditor` (`cron.go:1325-1337`)**:
    - *Frequency*: Daily.
    - *Logic*: Scans the `Retailers` table for newly onboarded shops whose H3 resolution-7 cell does not fall inside any active warehouse coverage polygon (`Warehouses.H3Indexes`).
    - *Why*: Alerts supplier sales managers to unserved retail territories requiring warehouse boundary expansion.

---

## 4. Real-time WebSocket Hub Ecosystem

### 4.1 What It Is
Pegasus maintains 6 dedicated, isolated WebSocket hubs implemented in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/ws/` and instantiated in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/bootstrap/app.go:115`.

### 4.2 Hub Architecture & Event Responsibilities

```
                                WebSocket Hub Fleet
  ┌───────────────┬───────────────┬───────────────┬───────────────┬───────────────┬───────────────┐
  │  RetailerHub  │   DriverHub   │ PayloaderHub  │  SupplierHub  │ WarehouseHub  │  FactoryHub   │
  │ /ws/retailer  │/v1/ws/driver  │/v1/ws/payloader│ /ws/supplier │ /ws/warehouse │ (IPC / Mesh)  │
  └───────┬───────┴───────┬───────┴───────┬───────┴───────┬───────┴───────┬───────┴───────┬───────┘
          │               │               │               │               │               │
          ▼               ▼               ▼               ▼               ▼               ▼
    • APPROACHING   • New Routes    • Dock Seals    • Live Fleet    • Dispatch Lock • Inter-transfer
    • Cart Updates  • Cancellations • Manifest QA   • Solver Alerts • DLQ Errors    • Dock Manifests
    • Preorders     • Reroutes      • Offload ACKs  • KPI Streams   • Inbound Alert • QC Status
```

1. **`RetailerHub` (`/ws/retailer`)**:
   - Pushes `DRIVER_APPROACHING` (triggered by Kafka `StartApproachConsumer` when a truck enters the 1.5 km geofence).
   - Syncs cart modifications across multiple devices belonging to the same shopkeeper.
   - Pushes AI preorder notifications and confirmation timers.
2. **`DriverHub` (`/v1/ws/driver`)**:
   - Dispatches newly solved route itineraries and stop orders directly to the driver's phone.
   - Pushes emergency stop cancellations and doorstep negotiation resolutions.
   - Validates live connectivity for dead-reckoning tracking.
3. **`PayloaderHub` (`/v1/ws/payloader`)**:
   - Dedicated connection for warehouse and factory loading dock scanners.
   - Broadcasts real-time seal locks (`SEAL_APPLIED`, `SEAL_BROKEN`) and offload discrepancies.
4. **`SupplierHub` (`/ws/supplier`)**:
   - Streams live vehicle GPS telemetry to the supplier's desktop map.
   - Emits `OPTIMIZATION_SOLVED` events when large-scale VRP batch runs complete.
5. **`WarehouseHub` (`/ws/warehouse`)**:
   - Broadcasts real-time dispatch lock states (`DISPATCH_LOCKED`, `DISPATCH_UNLOCKED`) when inventory counts occur.
   - Immediately alerts warehouse staff on transactional outbox failures (`BroadcastOutboxFailure`).
6. **`FactoryHub`**:
   - Connects manufacturing plant dispatch desks, streaming inter-facility transfer orders and loading bay manifests.

### 4.3 Command Handshake Protocol (`CommandRegistry`)
Implemented in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/ws/command_registry.go`. When a desktop operator issues a critical command to a mobile device (e.g., immediate driver reroute or dock bay reassignment):
1. A unique `CommandId` is generated and stored in Redis with a 30-second TTL.
2. The command is transmitted over the corresponding WebSocket connection.
3. The client MUST respond with an explicit ACK frame containing `CommandId`.
4. If the ACK is not received within the timeout, the command escalates to push notifications and alerts the operator of unconfirmed execution.

---

## 5. Operations Research & AI Subsystems

### 5.1 AI Worker (`apps/ai-worker`)
Located at `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/ai-worker`. It handles demand forecasting, document imports, and heuristic dispatch routing:

- **Predictive Empathy Engine (`ai-worker/main.go:45-180`)**:
  - Analyzes historical order cadence per SKU for each retail shop.
  - Computes running medians and standard deviations of consumption.
  - Applies supplier packaging rules: Minimum Order Quantity (`MOQ`) and carton `StepSize`.
  - Incorporates human-in-the-loop feedback from `CorrectionWeights` (`spanner.ddl:1645`): if a merchant consistently reduces predicted tea cases from 10 to 8, the algorithm dynamically applies a dampening coefficient to future runs.
- **Bulk Spreadsheet Importer (`ai-worker/import_worker.go`)**:
  - Consumes `INVENTORY_IMPORT_UPLOADED` from `inventory.import.events`.
  - Streams massive Excel (`.xlsx`) or CSV files directly from Google Cloud Storage using `excelize/v2`.
  - Calls Google Gemini via `packages/ai-bridge` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/ai-bridge`) to perform zero-shot schema mapping across arbitrary multilingual column headers (e.g. mapping "Количество", "Qty", or "Miqdor" to standard SKU units).
  - Inserts validated rows into `SupplierImportStagedRows`.
- **Clarke-Wright Savings Heuristic (`ai-worker/optimizer/clarke_wright.go`)**:
  - Fast Go-native dispatch solver.
  - Computes pairwise distance savings: `S(i, j) = d(depot, i) + d(depot, j) - d(i, j)`.
  - Boosts savings pairs using recovery priority weights (`stops[i].Priority`).
  - Greedily packs vehicles from smallest to largest (Tetris discipline).
  - Refines final vehicle tours using 2-opt edge exchange local search (`ai-worker/optimizer/two_opt.go`).

### 5.2 Production Optimizer Core (`services/optimizer-core`)
Located at `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/optimizer-core`. It provides mathematical optimization for large fleet deployments and multi-plant manufacturing:

- **Protobuf Interface (`proto/optimizer_core.proto`)**:
  - Declares `OptimizerCoreService` with RPCs `CalculateRoute` (VRP) and `ResolveConstraint` (CP-SAT).
- **Rust Sidecar (`services/optimizer-core/server-rust`)**:
  - High-performance Tonic/Prost gRPC daemon listening on `:50055`.
  - Solves Capacitated Vehicle Routing Problems with Time Windows (CVRPTW) in `src/solver/vrp.rs`.
  - Solves multi-facility manufacturing constraint satisfaction problems in `src/solver/cpsat.rs`.
  - **Integer Scaling**: In `src/scaling.rs`, enforces `SCALE_FACTOR = 10000.0`. All distances, vehicle capacities, and time windows are converted from floating-point values into 64-bit signed integers before entering the solver matrix, completely eliminating floating-point rounding imprecision.
- **Go Adapter Worker (`services/optimizer-core/adapters/go/cmd/optimizer-worker/main.go`)**:
  - Subscribes to `pegasus-optimizer-jobs` Kafka topic.
  - Deserializes job requests, converts them into gRPC calls to the Rust sidecar, and implements exponential jitter retries.
  - Writes solved routes transactionally into `OptimizationJobs` and stages outbound events into `OutboxEvents` in Cloud Spanner.

### 5.3 Deep Agents (`services/deep-agents`)
Located at `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/deep-agents`. Built using Python 3.11, LangChain, and LangGraph (`pyproject.toml:182`):
- Provides autonomous multi-agent reasoning fleets (`fleet.py`, `factory.py`, `subagents.py`).
- Powers automated codebase architectural compliance audits (`void-ecosystem-audit` in `ecosystem_audit.py`).
- Analyzes logistics performance anomalies and suggests supply lane adjustments to executive administrators.
