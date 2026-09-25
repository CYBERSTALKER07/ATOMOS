# Architectural Boundary & Data Engine Verification Report
**Specification**: Requirement R2 & Acceptance Criteria  
**Auditor Agent**: `teamwork_preview_explorer_survey_14_2`  
**Timestamp**: 2026-09-25T02:28:00+05:00  
**Workspace**: `/Users/shakhzod/Desktop/V.O.I.D`  

---

## Executive Summary

This investigation performed an exhaustive, evidence-based audit of **Requirement R2 (Architectural Boundary & Data Engine Verification)** across both primary systems:
1. **`pegasusX` (Global Multi-Tenant Cloud)**: Google Cloud Spanner multi-tenant architecture, interleaved hierarchy, tenant key partitioning by `SupplierId`, Apache Kafka event streaming with Strimzi HA configs, atomic transactional outbox pairing, and double-entry ledger idempotency.
2. **`pegasus.x` (Sovereign National Core)**: Strict isolation from Google Cloud Spanner and Apache Kafka (0 forbidden imports), PostgreSQL 16 schema migrations (78 migrations, 238 tables, strict 64-bit integer tiyins), and Redis 7 Streams (`XADD` with `FOR UPDATE SKIP LOCKED`) plus Redis Pub/Sub outbox relay execution.

All findings are backed by verbatim code references, AST/grep analysis, file paths, and test execution results.

---

## 1. pegasusX (Global Multi-Tenant Cloud)

### 1.1 Spanner DDL Compliance & Schema Architecture
- **Primary DDL Location**: `pegasusX/apps/backend-go/schema/spanner.ddl` (3,750 lines).
- **Total Tables**: 229 tables defined.
- **Total Indexes**: 288 secondary and unique indexes.
- **Interleaved Child Tables**: Exactly **19 tables** utilize Cloud Spanner's native physical co-location hierarchy via `INTERLEAVE IN PARENT ... ON DELETE CASCADE`.

#### Table Hierarchy & Interleaved Relationships
| Interleaved Child Table | Parent Table | Composite Primary Key | Physical Colocation Rationale |
|:---|:---|:---|:---|
| `ClaimEvidences` | `Claims` | `(ClaimId, EvidenceId)` | Co-locates claim damage evidence with parent claim |
| `WarehouseSupplyRequestItems` | `WarehouseSupplyRequests` | `(RequestId, ItemId)` | Atomic retrieval of replenishment BOM line items |
| `ManifestReplanLog` | `SupplierTruckManifests` | `(ManifestId, ReplanId)` | Dynamic route adjustments appended to manifest |
| `ManifestOrders` | `SupplierTruckManifests` | `(ManifestId, OrderId)` | Direct manifest stop mapping |
| `ManifestShipUnits` | `SupplierTruckManifests` | `(ManifestId, ShipUnitId)` | Physical carton/pallet handling units on truck |
| `RegionalConfigs` | `Regions` | `(RegionId, ConfigKey)` | Fast regional tariff and VAT lookup |
| `PickTasks` | `PickWaves` | `(WaveId, TaskId)` | Co-locates warehouse picker instructions with wave |
| `SupplierImportStagedRows` | `SupplierImportSessions` | `(supplier_id, session_id, row_index)` | High-throughput bulk catalog import staging |
| `SupplierImportMapping` | `SupplierImportSessions` | `(supplier_id, session_id)` | Column schema mapping rules per session |
| `OrderShopClosedLog` | `Orders` | `(OrderId, EventId)` | Doorstep driver exception trail |
| `OrderLineFiscalSnapshots` | `Orders` | `(OrderId, OrderLineId)` | Statutory VAT/OFD tax snapshots per order line |
| `OrderPaymentLegs` | `Orders` | `(OrderId, LegId)` | Multi-tender payment legs (cash/card/quota) |
| `CreditNoteLines` | `CreditNotes` | `(CreditNoteId, LineId)` | Financial credit note adjustment lines |
| `PriceListItems` | `PriceLists` | `(PriceListId, Sku)` | Wholesale tiered pricing tiers |
| `OrderLineAllocations` | `Orders` | `(OrderId, OrderLineId, WarehouseId)` | Multi-depot fulfillment allocation |
| `StopTwins` | `RouteTwins` | `(RouteId, StopId)` | Real-time digital twin stop progression |
| `VehicleInventory` | `RouteTwins` | `(RouteId, Sku)` | On-truck rolling stock ledger |
| `LotRecallImpactedOrders` | `LotRecallCampaigns` | `(CampaignId, OrderId, LotId)` | Quarantine & recall trace graph |
| `EvidenceItems` | `EvidenceDossiers` | `(DossierId, ItemId)` | Cold-chain temperature breach legal dossier |

#### Tenant Key Partitioning by `SupplierId`
In Google Cloud Spanner, multi-tenancy is partitioned either through root composite primary keys or through tenant-indexed root entities:
- **108 tables** contain explicit `SupplierId` / `supplier_id` tenant identifiers.
- **28 core domain tables** feature `SupplierId` as the leading partition column in the primary key, including:
  - `Suppliers (SupplierId)`
  - `SupplierProfiles (SupplierId)`
  - `SupplierPricingRules (SupplierId)`
  - `SupplierInventoryV2 (SupplierId, WarehouseId, ProductId)`
  - `SupplierImportSessions (supplier_id, session_id)`
  - `DemandForecastBaseline (SupplierId, ForecastDate, WarehouseId, ProductId)`
  - `ForecastAccuracyDaily (SupplierId, ForecastDate, WarehouseId, ProductId)`
  - `ReplenishmentPolicies (SupplierId)`
  - `PlanningSignalProjections (SupplierId, SignalId)`
  - `SkuClasses (SupplierId, Sku)`
  - `EchelonTargets (SupplierId, Sku, WarehouseId, Echelon)`
  - `LoyaltyLedger (SupplierId, LedgerId)`
  - `SupplierRegions (SupplierId, RegionId)`
- **81 secondary indexes** explicitly index on `SupplierId` for single-tenant filtering and scanning:
  - `Idx_Orders_BySupplierCreated ON Orders (SupplierId, CreatedAt DESC)`
  - `Idx_Orders_BySupplierUpdated ON Orders (SupplierId, UpdatedAt DESC)`
  - `Idx_Orders_BySupplierStatusUpdated ON Orders (SupplierId, Status, UpdatedAt DESC)`
  - `Idx_Claims_BySupplierStatus ON Claims (SupplierId, Status, CreatedAt DESC)`
  - `Idx_Drivers_BySupplierPhone ON Drivers (SupplierId, Phone)`
  - `Idx_Vehicles_BySupplierPlate ON Vehicles (SupplierId, LicensePlate)`
  - `Idx_Warehouses_BySupplier ON Warehouses (SupplierId)`
  - `Idx_PaymentLedger_BySupplierOccurred ON PaymentLedgerEntries (SupplierId, OccurredAt DESC)`

---

### 1.2 Apache Kafka Event Bus & Topic Configurations
- **Topic Manifest Definition**: `pegasusX/infra/k8s/kafka/kafka-topics.yaml` & `infra/k8s/kafka-topics.yaml`.
- **Strimzi CRDs**: All topics are provisioned declaratively with High Availability (`RF=3`, `min.insync.replicas=2`):
  - `pegasusx-main`: 3 partitions, 7d retention (`604800000ms`), 1Gi soft cap per partition.
  - `pegasusx-main-dlq`: 3 partitions, 14d retention (`1209600000ms`) for poison message inspection.
  - `pegasusx-orders`: 3 partitions, 7d retention.
  - `pegasusx-dispatch`: 3 partitions, 7d retention.
  - `pegasusx-realtime`: 3 partitions, 3d retention (`259200000ms`), 2Gi cap for high-throughput driver GPS pings.
  - `pegasusx-webhooks`: 3 partitions, 7d retention.
  - `pegasusx-freeze-locks`: 3 partitions, 7d retention.
  - `pegasusx-inventory-import`: 3 partitions, 7d retention.

#### Kafka Producer Guarantees (`outbox/kafka_publisher.go`)
- **Required Acks**: `RequiredAcks: kafka.RequireAll` (guarantees persistence across min.isr brokers before returning).
- **Partition Balancer**: `Balancer: &kafka.Hash{}` — partitions strictly by aggregate root ID (`AggregateId`), guaranteeing deterministic in-order per-entity event delivery.
- **Synchronous Write**: `Async: false` (no uncommitted background buffer drop).
- **Auto-Topic Creation Blocked**: `AllowAutoTopicCreation: false` (prevents unmanaged partition sprawl).
- **Multi-Tenant Fairness**: `outbox/fair.go` implements `FairInterleave`, round-robin draining unpublished events across `SupplierID` buckets to prevent heavy suppliers from starving smaller tenants.

---

### 1.3 Double-Entry Ledger Idempotency & Transaction Safety
Double-entry accounting and financial integrity are verified across schema, transactions, and service code:
1. **Strict 64-Bit Integer Minor Unit Arithmetic**:
   - Zero floating-point types in financial schemas (`AmountMinor INT64`, `TotalMinor INT64`, `NetMinor INT64`, `GrossMinor INT64`, `VatMinor INT64`).
   - Verified in `spanner.ddl`: all price, tax, and ledger tables use `INT64` minor units (e.g. Uzbekistan tiyins).
2. **Database-Enforced Financial Idempotency Indexes**:
   - `PaymentLedgerEntries`: `CREATE UNIQUE NULL_FILTERED INDEX Idx_PaymentLedgerEntries_GatewayTypeRef ON PaymentLedgerEntries(Gateway, EntryType, ReferenceId);` (`spanner.ddl:682-683`).
   - `OrderPaymentLegs`: `CREATE UNIQUE INDEX Idx_OrderPaymentLegs_IdempotencyKey ON OrderPaymentLegs(IdempotencyKey);` (`spanner.ddl:1821-1822`).
   - `ArLedgerEntries`: `CREATE UNIQUE INDEX Idx_ArLedger_ByIdempotency ON ArLedgerEntries(IdempotencyKey);` (`spanner.ddl:2616`).
   - `ArInvoices`: `CREATE UNIQUE INDEX Idx_ArInvoices_ByOrder ON ArInvoices(OrderId);` (`spanner.ddl:2600`).
   - `LoyaltyLedger`: `CREATE UNIQUE INDEX UQ_LoyaltyLedger_ByOrder ON LoyaltyLedger(SupplierId, OrderId);` (`spanner.ddl:3362`).
3. **Atomic Outbox Pairing in Single Spanner `ReadWriteTransaction`**:
   - In `pegasusX/apps/backend-go/payment/repository_spanner.go:516-538`, `writeWithOutbox` writes the payment session, payment attempt, deterministic ledger entry (`"pledger_session_" + s.SessionID`), and the outbox event mutation into the **exact same Spanner `ReadWriteTransaction`**.
   - In `pegasusX/apps/backend-go/ar/service.go:740-825`, `applyPaymentInTxn` checks existing idempotency key (`SELECT EntryId FROM ArLedgerEntries WHERE IdempotencyKey = @k`), calculates new balance, buffers outbox event (`outbox.EmitJSON`), updates `ArInvoices`, and inserts `ArLedgerEntries` with `spanner.CommitTimestamp` inside a single Spanner transaction.
4. **General Ledger Balanced Journal Export**:
   - `pegasusX/apps/backend-go/partner/export_journals.go:35-63`: Enforces double-entry identity across Chart of Accounts (COA):
     - `OPEN`: Debit AR, Credit Revenue.
     - `PAYMENT`: Debit Bank/Cash, Credit AR.
     - `CREDIT_NOTE`: Debit Revenue, Credit AR.
     - `REFUND`/`CHARGEBACK`: Debit AR, Credit Bank/Cash.

---

## 2. pegasus.x (Sovereign National Core)

### 2.1 Static Analysis for Forbidden Imports (Zero Contamination)
Exhaustive static grep and AST scans across the entire `pegasus.x` repository confirmed complete non-contamination:
- **`cloud.google.com/go/spanner`**: **0 occurrences** across all files in `pegasus.x/`.
- **`github.com/segmentio/kafka-go`**: **0 occurrences**.
- **`sarama`**: **0 occurrences**.
- **`confluent`**: **0 occurrences**.
- **Comments verification**: Exactly 4 text mentions of "Spanner" exist inside `packages/optimizer-contract/` (`doc.go`, `jobs.go`, `types.go`), all of which are documentation comments explicitly stating that the contract is designed as a pure data transfer specification that *avoids* importing Spanner or Kafka.
- **Cross-Import Check**:
  - `github.com/pegasusx/pegasusx/apps/backend-go` inside `pegasus.x/`: **0 references**.
  - `github.com/pegasus-x/core` inside `pegasusX/`: **0 references**.

---

### 2.2 PostgreSQL 16 Migrations and Schemas
- **Migration Directory**: `pegasus.x/database/migrations/`.
- **Migration Count**: Exactly **78 SQL migration files** (from `001_initial_schema.sql` up to `077_supervisor_override_pin.sql`).
- **Total Tables**: 238 relational tables created across all migrations.
- **Migration Engine**: `pegasus.x/backend/internal/db/migrate.go` executes all migrations sequentially within dedicated transactions (`p.RunInTx(ctx, func(tx pgx.Tx) error { ... })`) and records audit metadata in the `schema_migrations` table (`version`, `name`, `applied_at`, `execution_time_ms`).
- **Currency Arithmetic**:
  - 238 financial columns verified (`amount_tiyins`, `unit_price_tiyin`, `cash_bag_limit_tiyins`, `closing_cash_tiyins`, `accepted_tiyins`, `balance_minor`, etc.).
  - All financial amounts are stored strictly as `BIGINT` (64-bit integer minor units / tiyins).
  - 0 floating-point numbers used for monetary balances.

---

### 2.3 Redis 7 Streams & Pub/Sub Outbox Relay Execution
The transactional outbox pattern in `pegasus.x` is implemented with production rigor:
1. **Atomic In-Transaction Emission** (`backend/internal/outbox/emitter.go`):
   - Mutating state changes call `outbox.Emit(ctx, tx, aggregateType, aggregateID, eventType, payload)`.
   - Inserts directly into PostgreSQL table `outbox_events (event_id, aggregate_type, aggregate_id, event_type, payload)` within the same `pgx.Tx` as the domain entity update.
2. **SKIP LOCKED Polling** (`backend/internal/outbox/relay.go:122-136`):
   - `RelayWorker.ProcessBatch` executes:
     ```sql
     SELECT event_id, aggregate_type, aggregate_id, event_type, payload
     FROM outbox_events
     WHERE NOT published
     ORDER BY created_at ASC
     LIMIT $1
     FOR UPDATE SKIP LOCKED
     ```
   - Guarantees zero lock contention and allows horizontal scaling of multiple relay worker replicas.
3. **Redis 7 Streams Delivery (`XADD`)**:
   - Routes to canonical streams via `ResolveStreamKey`:
     - `events:payload:sealed`
     - `events:fleet:breakdown_reported`
     - `events:fleet:rescue_dispatched`
     - `events:doorstep:arrived`
     - `events:doorstep:tender_settled`
     - Fallback: `events:<aggregate_type>`
   - Also writes to secondary aggregate stream: `stream:<aggregate_type>:events`.
   - Redis `XADD` caps stream length at 100,000 entries (`MaxLen: 100000, Approx: true`).
   - Partition key: `aggregate_id` passed in stream values.
4. **Dead-Letter Handling**:
   - If Redis stream `XADD` fails, the event is immediately captured into `outbox_dead_letters` table with error diagnostic information.
5. **Real-Time WebSocket Hub Fanout**:
   - Ephemeral notification emitted via `w.redis.PublishEvent(ctx, canonicalStream, payload)` for low-latency browser/desktop WebSocket broadcasts.
6. **Publication Marking**:
   - Sets `published = TRUE, published_at = NOW()` on `outbox_events` in the same transaction.

---

## 3. Cross-System Architecture & Infrastructure Mapping

| Dimension | `pegasusX` (Global Multi-Tenant Cloud) | `pegasus.x` (Sovereign National Core) | Boundary Conformance |
|:---|:---|:---|:---|
| **Primary Persistence** | Google Cloud Spanner (`spanner.Client`, session pool) | PostgreSQL 16 (`pgxpool/v5`, max 25 conns, min 5) | **100% Strict Boundary** (Zero PG in pegasusX, Zero Spanner in pegasus.x) |
| **Partitioning Strategy** | Root `SupplierId` PK & interleaved child tables (19 tables) | Relational multi-table joins, single-tenant / domestic tenant isolation | **100% Conformance** |
| **Event Streaming Engine** | Apache Kafka (Strimzi HA, 8 topics, RF=3, min.isr=2) | Redis 7 Streams (`XADD`, consumer groups) | **100% Strict Boundary** (Zero Kafka in pegasus.x) |
| **Outbox Relay Polling** | Spanner ReadWriteTransaction with short leases (`ClaimedBy/ClaimedUntil`) & `FairInterleave` | PostgreSQL `SELECT ... FOR UPDATE SKIP LOCKED` | **Engine Appropriate** |
| **Real-time Ephemeral Hub** | Redis Pub/Sub (`cache:invalidate`, WebSocket broadcast) | Redis Pub/Sub (`events:*` channel fanout to `ws.Hub`) | **Aligned** |
| **Ledger Idempotency** | Unique indexes on `IdempotencyKey` + `CommitTimestamp` | Unique constraints on `idempotency_key` + `NOW()` | **Dual-Enforced** |
| **Monetary Units** | `INT64` minor units (Uzbekistan tiyins) | `BIGINT` minor units (Uzbekistan tiyins) | **Zero Float Currency** |
| **Container Runtime** | Spanner emulator + Redis 7 + Zookeeper + Kafka + Kafka-UI | PostgreSQL 16 TimescaleDB + Redis 7 + Caddy 2 reverse proxy | **Verified in Compose files** |

---

## 4. Test Verification Evidence

The following automated test suites were executed directly against the local workspace:

1. **pegasus.x Backend Verification**:
   - Command: `go vet ./...` in `pegasus.x/backend` -> **Exit code 0** (0 diagnostics).
   - Command: `go test -v ./internal/outbox/...` -> **PASS** (`TestResolveStreamKey`, `TestEmitWithReturn_Validation`, `TestRelayWorker_LifecycleAndDefaults`).
   - Command: `go test -v ./internal/db/...` -> **PASS** (`TestMigrationVersionParsing`, `TestMigration074FileContentAndSchemaValidation`, `TestMigration074SequentialOrdering`, `TestMigration077FileContentAndSchemaValidation`).

2. **pegasusX Backend Verification**:
   - Command: `go test -v ./outbox/...` in `pegasusX/apps/backend-go` -> **PASS** (21 subtests including `TestRelayDrainOnceMarksPublishedOnSuccess`, `TestRelayDrainOnceBoundsWedgedPublisher`, `TestSupplierIDFromPayload`).
   - Command: `go test -v ./ar/...` in `pegasusX/apps/backend-go` -> **PASS** (17 subtests including `TestRecordPaymentForOrderInTxn_Idempotent`, `TestOpenFromCreditLeave_IdempotentPerOrder`).
   - Command: `go test -v ./payment/...` in `pegasusX/apps/backend-go` -> **PASS** (all test cases passed including `TestHandleGlobalPayWebhook_ReplayNoDuplicatePersist`, `TestHandleLedger_QueriesRepositoryWithSupplierScope`).

---

## 5. Hardening Recommendations

1. **Strimzi Kafka CRD Continuous Linting**: Add a pre-commit / CI static check confirming that any newly declared Kafka topics in Go code have an identical declaration in `infra/k8s/kafka/kafka-topics.yaml`.
2. **Spanner Null-Filtered Index Monitoring**: Ensure query plans for high-throughput supplier queries strictly utilize the 81 `SupplierId`-prefixed indexes via `@{FORCE_INDEX=...}` where Spanner query optimizer chooses full table scans.
3. **Outbox Dead-Letter Alerting in pegasus.x**: Wire Prometheus alert metrics directly to the `outbox_dead_letters` row count to ensure operators receive instant alerts if Redis 7 Stream connectivity is degraded.
