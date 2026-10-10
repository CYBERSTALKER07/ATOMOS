# Cloud Spanner Reliability & High-Throughput Index Tuning Plan

## TL;DR
Optimize Cloud Spanner indexing architecture, eliminate redundant index mutations and full-table scans, eliminate base-table back-joins on high-frequency transaction paths via `STORING` covering clauses, and offload read-heavy dashboard and list endpoints from strong reads to bounded staleness (`spanner.MaxStaleness(15*time.Second)`). Delivers an incremental migration `20261009_cloud_spanner_index_tuning.ddl`, updates canonical `schema/spanner.ddl`, and ensures 100% parity across `cmd/schema-drift -offline`, `scripts/validate_spanner_stale_reads.sh`, and backend unit test suites.

## Objective
Accelerate read/write throughput and resilience in Cloud Spanner by:
1. Eliminating write amplification from the redundant `Idx_OrderStatusTransitions_ByOrder` index (which duplicated the table primary key prefix `(OrderId, CreatedAt DESC)`).
2. Introducing `Idx_OrderStatusTransitions_ByCreatedAt` covering index `(CreatedAt DESC, OrderId) STORING (NewStatus, Reason, EventKind, PreviousStatus, ActorRole, ActorId)` to eliminate full-table scans in the pulse activity stream and status event timeline.
3. Adding `STORING` covering clauses to `Idx_OutboxEvents_Unpublished`, `Idx_StockLots_BySupplierWarehouseProduct`, `Idx_StockLots_ByWarehouseProductExpiry`, and `Idx_RetailerStockBalances_ByRetailerSku` to prevent base-table back-joins and eliminate redundant shared read lock contention in read-write transactions.
4. Converting `Idx_OrderFiscalReceipts_ByReceiptId` into a `NULL_FILTERED` covering index `STORING (Status, Provider, AmountMinor, Currency)` to prune non-receipted rows from index splits.
5. Converting read-only list/dashboard queries (`pulse/service.go`, `order/repository_spanner.go:ListRetailerOrders`, `ListWarehouseOrdersByDeliveryWindow`, `ListOrdersByStatus`, `order/status_timeline.go:ListOrderTimeline`) to bounded staleness (`spanner.MaxStaleness(15*time.Second)`), offloading queries to read replicas and shielding the Paxos leader.

## Non-goals
- Dropping or renaming existing table columns (preserves zero-downtime rolling upgrades).
- Altering business domain entities, proto contracts, or Kafka topic event payloads.
- Applying bounded staleness to mutating workflows, transaction closures, or financial balance calculations that require linearizable reads.

---

## TODOs

### Wave 1: Spanner DDL Migrations & Canonical Schema Synchronization
- [x] **Task 1: Create Incremental DDL Migration**
  - Create `apps/backend-go/schema/migrations/20261009_cloud_spanner_index_tuning.ddl`.
  - Drop redundant `Idx_OrderStatusTransitions_ByOrder`.
  - Create covering index `Idx_OrderStatusTransitions_ByCreatedAt` with `STORING`.
  - Recreate `Idx_OutboxEvents_Unpublished` with `STORING (ClaimedUntil, AggregateType, AggregateId, TopicName, SupplierId)`.
  - Recreate `Idx_StockLots_BySupplierWarehouseProduct` with `STORING (QuantityOnHand, QuantityReserved, ExpiryDate, ReceivedAt)`.
  - Recreate `Idx_StockLots_ByWarehouseProductExpiry` with `STORING (QuantityOnHand, QuantityReserved, Status, LocationId)`.
  - Recreate `Idx_RetailerStockBalances_ByRetailerSku` with `STORING (OnHand, Reserved)`.
  - Recreate `Idx_OrderFiscalReceipts_ByReceiptId` with `NULL_FILTERED` and `STORING (Status, Provider, AmountMinor, Currency)`.
- [x] **Task 2: Canonical DDL Synchronization (`schema/spanner.ddl`)**
  - Synchronize `apps/backend-go/schema/spanner.ddl` to reflect the updated index topology for greenfield provisioning and emulator parity.
  - Run `go run ./cmd/schema-drift -offline` to guarantee zero schema drift.

### Wave 2: Query Refactoring & Bounded Staleness Offloading
- [x] **Task 3: Pulse Service Covering Index Routing & Bounded Staleness**
  - In `apps/backend-go/pulse/service.go`, route queries on `OrderStatusTransitions` to use `Idx_OrderStatusTransitions_ByCreatedAt`.
  - Convert `listRecentTransitions` to `WithTimestampBound(spanner.MaxStaleness(15*time.Second))`.
- [x] **Task 4: Order Service & Status Timeline Bounded Staleness**
  - In `apps/backend-go/order/status_timeline.go`, convert `ListOrderTimeline` to bounded staleness.
  - In `apps/backend-go/order/repository_spanner.go`, convert `ListRetailerOrders`, `ListWarehouseOrdersByDeliveryWindow`, and `ListOrdersByStatus` to bounded staleness (`spanner.MaxStaleness(15*time.Second)`).
- [x] **Task 5: Refresh Spanner Stale Read Allowlist**
  - Run `bash scripts/validate_spanner_stale_reads.sh --update` to refresh `scripts/spanner_stale_read_allowlist.txt`.
  - Verify `bash scripts/validate_spanner_stale_reads.sh` exits 0.

### Wave 3: Verification & Integration Testing
- [x] **Task 6: Verification Gates**
  - Run `go run ./cmd/schema-drift -offline`.
  - Run `go test -v -race ./order/... ./outbox/... ./stocklots/... ./pulse/... ./schemadrift/...`.
  - Run `make validate-all-k8s` to ensure Kubernetes manifests and gates remain healthy.
