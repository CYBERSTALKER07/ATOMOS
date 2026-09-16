# Requirement R1: Enterprise Distributed Systems & Infrastructure Audit Report

**Author:** `teamwork_preview_explorer_infra_1` (Infrastructure & Distributed Systems Specialist)  
**Date:** 2026-09-16  
**Target Codebases:** `pegasus`, `pegasusX`, and `pegasus.x` under `/Users/shakhzod/Desktop/V.O.I.D`  
**Classification:** Compiler-Grade Architectural Audit (R1)  

---

## 1. Observation

Direct, verified observations from code, schemas, configurations, and test suites across all three codebases:

### 1.1 Persistence & Sharding

1. **Google Cloud Spanner in `pegasusX`**:
   - **Schema Scale & Tables**: `pegasusX/apps/backend-go/schema/spanner.ddl` contains **3,749 lines of DDL** defining **229 tables**.
   - **Root Tenant Partitioning**: Over 108 tables explicitly carry `SupplierId STRING(36)` or `SupplierId STRING(64)` as the root partitioning column (e.g. `Suppliers` line 11, `Orders` line 171, `Products` line 760, `InventoryLevels` line 795, `SupplierTruckManifests` line 903, `OutboxEvents` line 696).
   - **Table Interleaving Hierarchy**: Exactly **19 tables** are physically interleaved into parent splits via `INTERLEAVE IN PARENT ... ON DELETE CASCADE`:
     1. `ClaimEvidences` (`spanner.ddl:328-329`) interleaved in `Claims` (`PRIMARY KEY (ClaimId, EvidenceId)`)
     2. `WarehouseSupplyRequestItems` (`spanner.ddl:551-552`) interleaved in `WarehouseSupplyRequests` (`PRIMARY KEY (RequestId, ItemId)`)
     3. `ManifestReplanLog` (`spanner.ddl:939-940`) interleaved in `SupplierTruckManifests` (`PRIMARY KEY (ManifestId, ReplanId)`)
     4. `ManifestOrders` (`spanner.ddl:968-969`) interleaved in `SupplierTruckManifests` (`PRIMARY KEY (ManifestId, OrderId)`)
     5. `ManifestShipUnits` (`spanner.ddl:980-981`) interleaved in `SupplierTruckManifests` (`PRIMARY KEY (ManifestId, ShipUnitId)`)
     6. `RegionalComplianceRules` (`spanner.ddl:1153-1154`) interleaved in `Regions` (`PRIMARY KEY (RegionId, ConfigKey)`)
     7. `PickTasks` (`spanner.ddl:1308-1309`) interleaved in `PickWaves` (`PRIMARY KEY (WaveId, TaskId)`)
     8. `SupplierImportStagedRows` (`spanner.ddl:1404-1405`) interleaved in `SupplierImportSessions` (`PRIMARY KEY (supplier_id, session_id, row_index)`)
     9. `SupplierImportPreflights` (`spanner.ddl:1418-1419`) interleaved in `SupplierImportSessions` (`PRIMARY KEY (supplier_id, session_id)`)
     10. `OrderShopClosedLog` (`spanner.ddl:1717-1718`) interleaved in `Orders` (`PRIMARY KEY (OrderId, EventId)`)
     11. `OrderLineFiscalSnapshots` (`spanner.ddl:1753-1754`) interleaved in `Orders` (`PRIMARY KEY (OrderId, OrderLineId)`)
     12. `OrderPaymentLegs` (`spanner.ddl:1817-1818`) interleaved in `Orders` (`PRIMARY KEY (OrderId, LegId)`)
     13. `CreditNoteLines` (`spanner.ddl:1868-1869`) interleaved in `CreditNotes` (`PRIMARY KEY (CreditNoteId, LineId)`)
     14. `PriceListItems` (`spanner.ddl:1969-1970`) interleaved in `PriceLists` (`PRIMARY KEY (PriceListId, Sku)`)
     15. `OrderLineAllocations` (`spanner.ddl:2050-2051`) interleaved in `Orders` (`PRIMARY KEY (OrderId, OrderLineId, WarehouseId)`)
     16. `RouteTwinWaypoints` (`spanner.ddl:3036-3037`) interleaved in `RouteTwins` (`PRIMARY KEY (RouteId, StopId)`)
     17. `VehicleInventory` (`spanner.ddl:3044-3045`) interleaved in `RouteTwins` (`PRIMARY KEY (RouteId, Sku)`)
     18. `LotRecallImpactedOrders` (`spanner.ddl:3467-3468`) interleaved in `LotRecallCampaigns` (`PRIMARY KEY (CampaignId, OrderId, LotId)`)
     19. `EvidenceDossierItems` (`spanner.ddl:3609-3610`) interleaved in `EvidenceDossiers` (`PRIMARY KEY (DossierId, ItemId)`)
   - **Hotspot Avoidance**: Zero tables use timestamps (`CreatedAt`, `Date`) as leading primary key columns. Monotonically increasing timestamps are stored with `OPTIONS (allow_commit_timestamp=true)` only as trailing non-key attributes or secondary indexes (`Idx_Orders_BySupplierCreated ON Orders(SupplierId, CreatedAt DESC)` at `spanner.ddl:212`).
   - **Line Items Strategy**: In `pegasusX`, `Orders` stores aggregate order lines inside `LineItemsJson BYTES(MAX) NOT NULL` (`spanner.ddl:183`), while fiscal and allocation splits are maintained in interleaved child tables (`OrderLineFiscalSnapshots` and `OrderLineAllocations`).

2. **PostgreSQL 16 in `pegasus.x`**:
   - **Migration Sequence**: Exactly **69 migration files** exist in `pegasus.x/database/migrations/*.sql` (`001_initial_schema.sql` through `068_trade_credit_quota_system.sql`). Note: `004` appears twice (`004_dispatch_capacities_and_locks.sql` and `004_enterprise_fiscal_dispatch_and_compliance.sql`), yielding 69 files.
   - **Relational Integrity**: Foreign key constraints with cascading deletes are enforced (e.g. `order_items` -> `orders` in `001_initial_schema.sql:99`, `manifest_stops` -> `manifests` in `001_initial_schema.sql:121`).
   - **TimescaleDB & PostGIS Reality Check**:
     - Container image: `timescale/timescaledb-ha:pg16` (`pegasus.x/docker-compose.yml:5`, `pegasus.x/docker/docker-compose.yml:8`).
     - Hypertables: Created in `013_fleet_integrity_coldchain_and_blindspots.sql:24-25` (`SELECT create_hypertable('driver_telemetry_stream', 'recorded_at', if_not_exists => TRUE);`) and line 46 (`SELECT create_hypertable('coldchain_telemetry_stream', 'recorded_at', if_not_exists => TRUE);`).
     - PostGIS: Extension enabled in `013_fleet_integrity_coldchain_and_blindspots.sql:3` and `020_spatial_demurrage_homogeneity_and_fiscal_fx.sql:4`. Spatial geometry column `geofence_polygon GEOMETRY(Polygon, 4326)` with GIST index in `020...sql:25-29`.
     - Runtime Reality: Core operational tables (`orders`, `warehouses`, `manifest_stops`) store coordinates as `DOUBLE PRECISION` or `NUMERIC(10, 6)`. Real-time spatial proximity filtering in Go backend uses in-memory Haversine distance and Redis GEO commands (`GEOADD`, `GEODIST`, `GEOSEARCH`) rather than heavy PostGIS queries.

3. **Legacy `pegasus` Schema**:
   - `pegasus/apps/backend-go/schema/spanner.ddl` contains **2,373 lines** and **94 tables**.
   - Architectural Note on `OrderItems`: Lines 264–268 document:
     `ARCHITECTURAL DECISION (Phase 4): Previous design: OrderItems INTERLEAVE IN PARENT Orders. Problem: cross-order SKU aggregations caused full Orders table scans. Solution: Standalone table with distributed PK (LineItemId = UUID).`

---

### 1.2 Messaging & Streaming

1. **Apache Kafka in `pegasusX`**:
   - **Topic Routing**: Defined in `pegasusX/apps/backend-go/events/topic_routing.go:10-24`:
     - `TopicOrders`: `pegasusx-orders`
     - `TopicDispatch`: `pegasusx-dispatch`
     - `TopicRealtime`: `pegasusx-realtime`
     - `TopicExceptions`: `logistics.exceptions.v1`
     - `TopicTelemetryLogistics`: `logistics.telemetry.v1`
     - Supports dual-write transition via `KAFKA_TOPIC_DUAL_WRITE=true` (`topic_routing.go:28`).
   - **Producer Configuration**: `pegasusX/apps/backend-go/outbox/kafka_publisher.go:81-92`:
     - `RequiredAcks: kafka.RequireAll` (line 83) — broker ISR acknowledgement.
     - `Balancer: &kafka.Hash{}` (line 88) — hashes the aggregate root ID key.
     - `Async: false` (line 89) — synchronous socket writes.
     - `AllowAutoTopicCreation: false` (line 90) — enforces strict topic governance via Kubernetes Strimzi CRDs.
     - `MaxAttempts: 3`, `WriteTimeout: 10s`, `BatchTimeout: 250ms`.
   - **Consumer Workerpool**: `pegasusX/apps/backend-go/kafka/workerpool/workerpool.go`:
     - Partition routing: `idx := int(uint(m.Partition)) % p.workers` (line 160) distributes messages to dedicated per-partition worker channels, guaranteeing strict in-order processing.
     - Poison pill handling & monotonic offset preservation: Lines 206–211 halt workerpool (`cancel()`) upon `ErrSkipCommit`, preventing subsequent higher offset commits from silently dropping failed events.

2. **Redis 7 Streams & Pub/Sub in `pegasus.x`**:
   - **Durable Streams**: `pegasus.x/backend/internal/outbox/relay.go:99-111` writes to `stream:<aggregate_type>:events` via `XAdd` with `MaxLen: 100000, Approx: true`.
   - **Geospatial & Fleet Telemetry**: `pegasus.x/backend/internal/redis/client.go:35-55` updates driver locations via `GeoAdd(ctx, "drivers:active", ...)` and refreshes presence key `driver:presence:<driver_id>`.
   - **Fleet Events Stream**: `pegasus.x/backend/internal/redis/client.go:64-87`: `XAddFleetEvent` publishes to stream `events:fleet` with `MaxLen: 100000` and fans out to Pub/Sub channel `events:fleet`.
   - **Ephemeral Fan-out**: `outbox/relay.go:122-124`: `_ = w.redis.PublishEvent(ctx, fmt.Sprintf("events:%s", it.aggregateType), string(it.payload))` triggers live fanout to connected WebSocket clients.
   - **Monotonic Sequencer & Ring Buffer**: `pegasus.x/backend/internal/ws/hub.go:110-131`:
     - Generates atomic 64-bit sequence numbers: `seq := atomic.AddInt64(&h.seq, 1)` (line 111).
     - Maintains in-memory ring buffer of **2,000 events** (`maxHistory: 2000`, line 62).
     - `GetEventsSince` (lines 135-160) enables fast sub-millisecond replay for mobile clients reconnecting across cellular handovers, setting `fullResync: true` if sequence is expired.

---

### 1.3 Connection Pooling

1. **Spanner gRPC Session Pool in `pegasusX`**:
   - Initialization: `pegasusX/apps/backend-go/bootstrap/runtime_adapters.go:43-45`:
     ```go
     newSpannerRuntimeClient = func(ctx context.Context, database string) (*spanner.Client, error) {
         return spanner.NewClient(ctx, database)
     }
     ```
   - Operates with Google Cloud Spanner Go SDK defaults:
     - `MinOpened: 100` sessions
     - `MaxOpened: 400` sessions
     - `WriteSessions: 0.2` (20% pre-warmed write sessions)
     - `HealthCheckWorkers: 10`
     - Multiplexed across 4 HTTP/2 gRPC sub-channels per Spanner client.

2. **PostgreSQL pgxpool in `pegasus.x`**:
   - Configuration in `pegasus.x/backend/internal/db/postgres.go:18-43`:
     ```go
     cfg.MaxConns = 25              // Line 25
     cfg.MinConns = 5               // Line 26
     cfg.MaxConnLifetime = 1 * time.Hour   // Line 27
     cfg.MaxConnIdleTime = 15 * time.Minute // Line 28
     ```
   - Connectivity verification: `pool.Ping(pingCtx)` with 5-second timeout (lines 36-40).
   - Transaction boundary management: `RunInTx` enforces `IsoLevel: pgx.ReadCommitted` (line 47) and handles panics safely:
     ```go
     defer func() {
         if p := recover(); p != nil {
             _ = tx.Rollback(ctx)
             panic(p)
         }
     }()
     ```

---

### 1.4 Load Balancing & Routing

1. **Maglev Consistent Hashing Prototype in `pegasus` (Legacy Reference)**:
   - File: `pegasus/apps/backend-go/bootstrap/spannerrouter/router.go:1-213`.
   - Maglev Principle: Uses pre-computed lookup table built once at `init()`:
     ```go
     // Lines 193-212
     func cellToRegion(cell string) string {
         if len(cell) != 15 { return "" }
         c := h3.CellFromString(cell)
         if c == 0 { return "" }
         parent, err := c.Parent(2)
         if err != nil { return "" }
         return regionCells[parent]
     }
     ```
   - Maps H3 Resolution 7 cells to Resolution 2 parent cells (~90,000 km² macro-regions), routing read queries to regional replicas (`"asia"`, `"eu"`, `"us"`) in ~50ns without runtime Spanner or Redis lookups.

2. **Global Cell Architecture in `pegasusX`**:
   - `pegasusX/apps/backend-go/auth/cell_directory.go:39-46`: `ListCells()` catalogues `cell-uz` (shipped live cell at `api.pegasusx.app`), `cell-eu`, `cell-us`, and `cell-kz` (planned).
   - Ingress Enforcement: `pegasusX/apps/backend-go/auth/cell_isolation.go:24-40`: `rejectForeignCell` checks JWT `home_cell` against environment variable `HOME_CELL`. Unauthenticated or missing `home_cell` claims are rejected with `ErrWrongCell (missing home_cell claim)` (remediation of VULN-01).

3. **Sovereign Caddy 2 Reverse Proxy in `pegasus.x`**:
   - File: `pegasus.x/docker/Caddyfile:1-60` and `pegasus.x/docs/PRODUCTION_INFRASTRUCTURE_AND_HOSTING_BLUEPRINT.md:215-265`.
   - Direct TAS-IX Domestic Peering: Hosted at Servercore Tashkent Tier III datacenter, bypassing international transit.
   - Gateway Routing:
     - `api.pegasusx.uz`: reverse proxies to `backend:8080` (`flush_interval -1` for real-time WebSocket streaming).
     - `supplier.pegasusx.uz`: proxies to `supplier-portal:3000`.
     - `retailer.pegasusx.uz`: proxies to `retailer-portal:3000`.
     - `warehouse.pegasusx.uz`: proxies to `warehouse-portal:3000`.
     - `tma.pegasusx.uz`: proxies to `retailer-telegram-miniapp:3000`.
     - `storage.pegasusx.uz`: proxies to `minio:9000`.
   - Acceleration: Native HTTP/3 QUIC and `encode zstd gzip`.

---

### 1.5 Transactional Outbox & CDC

1. **Atomic Outbox Pairing**:
   - `pegasusX`: `apps/backend-go/outbox/spanner_txn_buffer.go:14-40`:
     - `SpannerTxnBuffer` wraps `*spanner.ReadWriteTransaction`.
     - Domain services buffer mutations, and `Flush(ctx)` inserts `OutboxEvents` in the exact same Spanner transaction commit.
   - `pegasus.x`: `backend/internal/outbox/emitter.go:12-28`:
     - `outbox.Emit` takes `pgx.Tx` directly, executing `INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload) VALUES ($1, $2, $3, $4)`. Commits atomically with the relational transaction.

2. **Concurrency & Locking**:
   - `pegasusX`: `apps/backend-go/outbox/spanner_store.go:88-195`:
     - Distributed lease locking inside a `ReadWriteTransaction`:
       ```sql
       SELECT EventId, AggregateType, AggregateId, TopicName, Payload, CreatedAt, PublishedAt, SupplierId
       FROM OutboxEvents@{FORCE_INDEX=Idx_OutboxEvents_Unpublished}
       WHERE PublishedAt IS NULL
         AND (ClaimedUntil IS NULL OR ClaimedUntil < @now)
       ORDER BY CreatedAt
       LIMIT @limit
       ```
     - Acquired rows are locked with `ClaimedBy = claimant` and `ClaimedUntil = leaseUntil` (2-minute lease).
   - `pegasus.x`: `backend/internal/outbox/relay.go:59-67`:
     - PostgreSQL row-level locking:
       ```sql
       SELECT event_id, aggregate_type, aggregate_id, event_type, payload
       FROM outbox_events
       WHERE NOT published
       ORDER BY created_at ASC
       LIMIT $1
       FOR UPDATE SKIP LOCKED
       ```
     - Non-blocking concurrent processing across multiple relay workers.

3. **Fair Multi-Tenant Interleaving**:
   - `pegasusX`: `apps/backend-go/outbox/fair.go:8-52`:
     - `FairInterleave(events []Event, limit int) []Event`:
     - Buckets events by `SupplierId`, sorts tenant keys stably, and performs round-robin interleaving across tenant buckets. High-throughput suppliers cannot saturate the outbox and starve smaller suppliers.
   - `pegasus.x`: Single-tenant sovereign deployment processes events strictly FIFO by `created_at ASC`.

4. **Poison Pill & Dead-Letter Queue (DLQ) Isolation**:
   - `pegasusX`: `apps/backend-go/outbox/spanner_store.go:255-340`:
     - `RecordPublishFailures` increments `PublishAttempts`.
     - Upon reaching `maxAttempts` (default 20), atomically inserts the event into `OutboxDeadLetters` (`DeadLetteredAt = spanner.CommitTimestamp`) and deletes it from `OutboxEvents`.
   - `pegasus.x`: `backend/internal/outbox/relay.go:113-118`:
     - When Redis `XAdd` fails, immediately inserts into `outbox_dead_letters (event_id, aggregate_type, aggregate_id, event_type, payload, error_message)` and continues the batch.

---

### 1.6 Background Schedulers & Workers

1. **In `pegasusX`**:
   - Worker pool wiring: `bootstrap/workers.go:42-255`.
   - Execution lifecycle: `runtime_workers.go:19-230`.
   - **8 Kafka Consumer Groups**:
     1. `void-notification-dispatcher` (fans in `TopicMain`, `TopicOrders`, `TopicDispatch`, `TopicRealtime`, `TopicExceptions`, `TopicTelemetryLogistics`)
     2. `void-order-mutator` (`pegasusx-orders`)
     3. `void-warehouse-mutator` (`pegasusx-dispatch`)
     4. `void-returns-reverse` (`logistics.exceptions.v1`)
     5. `void-claims-bridge` (`TopicMain`)
     6. `void-billing-tier` (`pegasusx-orders`)
     7. `void-partner-webhooks` (`pegasusx-orders`, `TopicExceptions`)
     8. `void-digital-twin` (`TopicMain`, `TopicOrders`, `TopicDispatch`, `TopicRealtime`, `route.eta.updated`)
   - **15+ Background Loops & Schedulers**:
     - `OutboxRelay.Start(ctx)`
     - `warehouse.StartAutoDispatchWorker`
     - `warehouse.StartDispatchPlanWarmer`
     - `WebhookInbox.StartReconciler`
     - `WebhookReconciler.ReconcileStuckSessions` (every 5m)
     - `ReplenishmentEngine.StartCron(ctx)`
     - `FactoryPlanning.StartPlanningCron(ctx)`
     - `LaborCapacityService.RunDriverScoreWorker` (every 24h)
     - `LaborCapacityService.RunCapacitySnapshotWorker` (every 1h)
     - `RouteAnalyticsWorker.RunNightlyWorker` (every 24h)
     - `order.StartSagaRecoveryWorker` (every 15s)
     - `CashReconEscalation.RunNightlyWorker` (every 24h)
     - `ReorderSuggestionWorker.RunBatchWorker` (every 12h)
     - `DemandService.RunDensityWorker` (every 6h)
     - `ControlTowerWorker.Run(ctx)`
     - `ARDunningWorker.Start(ctx, time.Hour)`
     - `OrderService.AutoConfirmDueOrders` (every 1m)
     - `RetailerService.RunPosHoldsSweeper` (every 15m)
     - `FactoryService.RunFactorySLABreachWorker` (every 5m)
   - Worker tier heartbeat: `StartWorkerHeartbeat` in `bootstrap/workers.go:22` publishes heartbeat to Redis so api-only pods avoid double-running consumers.

2. **In `pegasus.x`**:
   - `backend/cmd/server/main.go:83-91`:
     - Spawns `relayWorker := outbox.NewRelayWorker(pool, rdb, 500*time.Millisecond, 50)`
     - Spawns `wsHub.Run(ctx)`
   - `backend/internal/telemetry/ingestion.go:50-86`:
     - Spawns `pruneWorker` (5-minute ticker) evicting drivers without pings for >15 minutes.
   - **Architectural Gap Surfaced**:
     - `DebtRecoveryWorker` (`internal/credit/debt_recovery.go:20-276`) is fully implemented with `MarkOverdueDebts`, `AttemptAutoCharges`, and `ProcessDunning`, but is **not instantiated or spawned in `main.go`**.

---

### 1.7 Architectural Boundary Verification

Automated scan verifying the strict two-system boundary:
- `grep -rnE "cloud\.google\.com/go/spanner" pegasus.x/` -> **0 matches** (100% clean)
- `grep -rnE "segmentio/kafka-go|confluent-kafka" pegasus.x/` -> **0 matches** (100% clean)
- `grep -rnE "github\.com/jackc/pgx" pegasusX/apps/backend-go/` -> **0 matches** (100% clean)

---

## 2. Logic Chain

1. **Persistence & Sharding Invariance**:
   - *Observation*: `pegasusX` uses Spanner composite keys rooted in `SupplierId` with 19 interleaved child tables, whereas `pegasus.x` uses PostgreSQL 16 with 69 migrations and relational foreign keys.
   - *Deduction*: `pegasusX` achieves horizontal scaling and eliminates two-phase commit (2PC) coordination by co-locating parent and child rows in the same physical storage split via `INTERLEAVE IN PARENT`. `pegasus.x`, being single-tenant, relies on PostgreSQL ACID transactions, avoiding distributed commit latencies entirely.
   - *Deduction regarding TimescaleDB/PostGIS*: Although TimescaleDB hypertables exist for telemetry streams (`013...sql:25`), operational geospatial lookups in `pegasus.x` are offloaded to Redis GEO (`drivers:active`) to prevent high-frequency write contention on PostgreSQL.

2. **Messaging & Streaming Guarantees**:
   - *Observation*: `pegasusX` configures Kafka producer with `RequiredAcks = kafka.RequireAll`, synchronous writes, and hash balancing by aggregate root ID (`kafka_publisher.go:81-92`). `pegasus.x` combines PostgreSQL outbox with Redis 7 Streams (`XAdd` capped at 100,000) and Redis Pub/Sub channels (`relay.go:99-124`).
   - *Deduction*: `pegasusX` guarantees enterprise at-least-once message delivery with strict partition ordering. `pegasus.x` achieves sub-millisecond local messaging with bounded in-memory streams suitable for single-node national deployments, backstopped by transactional PostgreSQL durability.

3. **Connection Pooling & Resiliency**:
   - *Observation*: `pegasus.x` caps `pgxpool` at `MaxConns = 25`, `MinConns = 5` (`postgres.go:25-26`). `pegasusX` relies on Spanner client default session pooling (100–400 sessions) across 4 gRPC sub-channels.
   - *Deduction*: `pegasus.x` is tuned for a cost-effective $135/month Servercore Tashkent VPS (preventing PostgreSQL process exhaustion), whereas `pegasusX` is designed for auto-scaling multi-pod Kubernetes clusters communicating with managed Cloud Spanner.

4. **Routing & Latency Topology**:
   - *Observation*: Legacy `pegasus` prototyped Maglev consistent hashing using H3 Res-7 -> Res-2 bitmask (`spannerrouter/router.go:204-212`). `pegasusX` evolved this into Global Cell Architecture (`cell-uz` live, `cell-eu/us/kz` planned) with JWT `home_cell` claims enforcement (`cell_isolation.go:24-40`). `pegasus.x` operates via Caddy 2 reverse proxy with TAS-IX domestic peering (`Caddyfile:1-60`).
   - *Deduction*: `pegasusX` ensures regional regulatory compliance (GDPR, EU data localization) by isolating requests per cell. `pegasus.x` fulfills Uzbekistan Law No. ZRU-547 by ensuring all traffic stays within domestic TAS-IX exchange points with zero international transit hops.

5. **Transactional Outbox & Fair Scheduling**:
   - *Observation*: `pegasusX` uses `SpannerTxnBuffer` + distributed lease locking (`ClaimedUntil`) + `FairInterleave` round-robin tenant bucketing (`fair.go:8-52`). `pegasus.x` uses `pgx.Tx` + `SELECT ... FOR UPDATE SKIP LOCKED` (`relay.go:66`).
   - *Deduction*: In multi-tenant `pegasusX`, high-volume suppliers generating thousands of orders could starve lower-volume tenants without `FairInterleave`. In single-tenant `pegasus.x`, `FOR UPDATE SKIP LOCKED` provides optimal lock-free concurrency for multiple local worker goroutines without needing tenant interleaving logic.

6. **Worker Lifecycle Completeness**:
   - *Observation*: `pegasusX` starts 8 Kafka consumers and 15+ background worker loops in `runtime_workers.go:19-230`. `pegasus.x` starts `RelayWorker`, `wsHub`, and `pruneWorker`, but leaves `DebtRecoveryWorker` (`internal/credit/debt_recovery.go:20-276`) un-wired in `cmd/server/main.go`.
   - *Deduction*: Automated AR overdue marking and card-on-file auto-recovery is dormant in `pegasus.x` production deployments until explicitly registered in `main.go`.

---

## 3. Caveats

1. **Simulated Multi-Cell in `pegasusX`**:
   - While `cell-uz` is fully implemented and tested via `ssmr-smokecheck`, `cell-eu`, `cell-us`, and `cell-kz` remain defined as `status: "planned"` in `auth/cell_directory.go:42-44`. No live Google Cloud Spanner multi-region clusters are provisioned outside the UZ region.
2. **Kafka Producer Idempotence**:
   - `segmentio/kafka-go` Writer does not support Kafka broker-level `enable.idempotence=true` (as documented in `kafka_publisher.go:61-64`). Deduplication relies on the `OutboxEvents` `PublishedAt` state machine plus consumer-side deduplication middleware (`kafka.WithEventDedup`).
3. **TimescaleDB Continuous Aggregates**:
   - While hypertables are created in migration 013, continuous aggregate views (e.g. 5-minute rolling vehicle speed or temperature rollups) are not defined in the migrations; raw chunks are queried directly.

---

## 4. Conclusion

Requirement R1 (Enterprise Distributed Systems & Infrastructure Audit) is **completely analyzed and verified**:

1. **Architectural Boundary Enforcement**:
   - **Zero cross-contamination confirmed**: `pegasus.x` contains 0 Spanner and 0 Kafka imports; `pegasusX` contains 0 single-tenant PostgreSQL imports.
2. **Persistence Rigor**:
   - `pegasusX` Spanner schema contains 229 tables with 19 interleaved parent-child splits, 108+ `SupplierId` root-partitioned tables, and zero timestamp hotspot keys.
   - `pegasus.x` contains 69 migrations on PostgreSQL 16 with TimescaleDB hypertables, PostGIS polygon clustering, and cost-effective connection pooling (`MaxConns: 25`).
3. **Event Streaming Rigor**:
   - `pegasusX` Kafka pipeline enforces `RequiredAcks = kafka.RequireAll`, partition-keyed worker routing, and safe termination on `ErrSkipCommit`.
   - `pegasus.x` Redis 7 Streams pipeline uses capped 100k streams (`XAdd`), atomic 64-bit monotonic WebSocket envelopes, and a 2,000-event memory ring buffer.
4. **Outbox Pattern**:
   - Both systems enforce strict atomic outbox pairing (`SpannerTxnBuffer.Flush` in `pegasusX` and `outbox.Emit` with `pgx.Tx` in `pegasus.x`). Multi-tenant fairness is guaranteed in `pegasusX` via `FairInterleave`.
5. **Actionable Remediation**:
   - In `pegasus.x`, wire `DebtRecoveryWorker` into `backend/cmd/server/main.go` alongside `RelayWorker` to activate autonomous overdue debt processing and card-on-file dunning.

---

## 5. Verification Method

To independently reproduce and verify all findings:

### 5.1 Test Suite Verification
```bash
# 1. Verify pegasus.x backend builds and passes tests
cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
go test ./internal/db ./internal/outbox ./internal/ws

# 2. Verify pegasusX backend outbox and workerpool tests pass
cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go
go test ./outbox ./kafka/workerpool
```

### 5.2 Schema & Boundary Inspection
```bash
# Verify Spanner table count and interleaving
grep -E "^CREATE TABLE [A-Za-z0-9_]+" pegasusX/apps/backend-go/schema/spanner.ddl | wc -l # 229
grep -n "INTERLEAVE IN PARENT" pegasusX/apps/backend-go/schema/spanner.ddl | wc -l      # 19

# Verify PostgreSQL 69 migrations in pegasus.x
ls pegasus.x/database/migrations/*.sql | wc -l                                          # 69

# Verify Two-System Boundary isolation
grep -rnE "cloud\.google\.com/go/spanner|segmentio/kafka-go" pegasus.x/                  # Must return 0
grep -rnE "github\.com/jackc/pgx" pegasusX/apps/backend-go/                             # Must return 0
```

### 5.3 Code Citation Spot Checks
- `pegasusX/apps/backend-go/outbox/spanner_txn_buffer.go:31-40`: Verify `SpannerTxnBuffer.Flush`
- `pegasusX/apps/backend-go/outbox/spanner_store.go:174`: Verify `FairInterleave(candidates, limit)`
- `pegasusX/apps/backend-go/outbox/kafka_publisher.go:81-92`: Verify `RequiredAcks: kafka.RequireAll`
- `pegasusX/apps/backend-go/kafka/workerpool/workerpool.go:160`: Verify partition worker routing
- `pegasus.x/backend/internal/db/postgres.go:24-29`: Verify `MaxConns = 25`, `MinConns = 5`
- `pegasus.x/backend/internal/outbox/relay.go:66`: Verify `FOR UPDATE SKIP LOCKED`
- `pegasus.x/backend/internal/ws/hub.go:111`: Verify `atomic.AddInt64(&h.seq, 1)` and 2000-event ring buffer
- `pegasus.x/backend/internal/credit/debt_recovery.go:20`: Verify `DebtRecoveryWorker` implementation
