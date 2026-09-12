# Track 4 Codebase Audit: Retailer, Warehouse & Stock Fulfillment Domain

**Target Codebase**: `pegasusX/apps/backend-go`  
**Subsystems Audited**:
- `order/`: `unified_checkout.go`, `multi_supplier_checkout.go`, `service.go`, `inventory_reservation.go`, `inventory_plan.go`, `inventory_release.go`, `inventory_stale_release.go`, `retailer_cancel.go`, `shop_closed.go`, `worker_shop_closed.go`, `credit_guard.go`.
- `stocklots/`: `lots.go`, `fefo.go`, `picking.go`, `seal_gate.go`, `counting.go`, `credit_putaway.go`, `locations.go`, `handlers.go`, `outbox_emit.go`, `rollup.go`.
- `warehouse/`: `receive_items.go`, `supply_request_qc.go`, `location_ops.go`, `pick_waves.go`, `dispatch_execute.go`, `auto_dispatch.go`, `demand_products.go`, `ops_broadcast.go`.
- `inventory/`: `repository.go`, `replenish.go`.
- `retailer/`: `repository_cart.go`, `store_stock.go`, `stock_count_commit.go`, `auto_order.go`, `auto_order_policy.go`, `auto_order_worker.go`, `capability_packs.go`.
- `returns/`: `service.go`, `lifecycle.go`, `inbound.go`, `barcode.go`, `tickets.go`.
- `credit/`: `service.go`, `reserve.go`, `limit.go`, `policy.go`.
- `schema/spanner.ddl`: Table definitions for `WarehouseSupplyRequests`, `WarehouseSupplyRequestItems`, `StockLots`, `PickWaves`, `PickTasks`, `SupplierInventoryV2`, `InventoryLevels`, `CartItems`, `RetailerStockBalances`.

---

## Executive Summary

Track 4 implements the core operational backbone of the PegasusX supply chain platform: from multi-supplier cart management and credit checking, through warehouse receiving, lot tracking, FEFO picking, and manifest dispatch, to store stock counts and reverse logistics returns.

The codebase exhibits high architectural maturity in key areas (e.g., FEFO lot reservation math, outbox transactional buffering, CAS state transitions, and seal-gate assertions). However, deep line-by-line inspection revealed **3 Critical Defects** that cause runtime transaction aborts or database constraint violations in standard production flows, **6 High-Severity Inconsistencies** regarding transactional atomicity, stale reads, and split inventory models, and multiple Medium-severity operational gaps.

---

## Ranked Findings Matrix

| ID | Severity | Subsystem | File & Line Number | Issue Summary |
|---|---|---|---|---|
| **TRK4-001** | **CRITICAL** | `stocklots` / `returns` | `stocklots/credit_putaway.go:54-61`<br>`stocklots/lots.go:76-78` | `CreditViaDefaultPutawayInTxn` sets `ExpiryDate == nil`, crashing on any perishable product return or restock. |
| **TRK4-002** | **CRITICAL** | `warehouse` | `warehouse/receive_items.go:159-178`<br>`schema/spanner.ddl:509-545` | Spanner `NOT NULL` schema constraint violation when receiving short inbound shipments. |
| **TRK4-003** | **CRITICAL** | `order` | `order/shop_closed.go:522, 725`<br>`order/inventory_release.go:26-30` | Shop-closed order cancellation hard-errors when `WMS_LOTS_ENABLED` due to missing `orderID`. |
| **TRK4-004** | **HIGH** | `inventory` / `warehouse` | `inventory/repository.go:29-34, 231-232`<br>`warehouse/demand_products.go:352` | Dual inventory models (`InventoryLevels` vs `SupplierInventoryV2`) and double-deduction math in legacy reservation. |
| **TRK4-005** | **HIGH** | `stocklots` | `stocklots/locations.go:114-184` | Quarantining or deactivating a bin does not update lot statuses or re-rollup inventory, leaving stock sellable. |
| **TRK4-006** | **HIGH** | `retailer` | `retailer/store_stock.go:671-686`<br>`retailer/stock_count_commit.go:298-315` | Stock count commit executes unbatched individual transactions per SKU; midway failures cause partial commit and double-adjustments on retry. |
| **TRK4-007** | **HIGH** | `retailer` | `retailer/repository_cart.go:53, 83` | Stale reads (`ExactStaleness(5s)`) on cart lists cause ghost items and checkout race conditions. |
| **TRK4-008** | **HIGH** | `order` / `credit` | `order/service.go:1492-1498` | Credit limit check-to-delivery race condition due to lazy / default-disabled credit reservation at order creation. |
| **TRK4-009** | **HIGH** | `returns` | `returns/inbound.go:581-606` | Inbound return confirmation records warehouse disposition but does not trigger financial credit note / refund settlement. |
| **TRK4-010** | **MEDIUM** | `warehouse` | `warehouse/pick_waves.go:48-85` | Unauthenticated HTTP mock call to `localhost:8000/pick-path` bypassing durable Spanner `PickWaves` / `PickTasks`. |
| **TRK4-011** | **MEDIUM** | `warehouse` | `warehouse/ops_broadcast.go:560-566` | Arbitrary `LIMIT 200` truncation and index scan inefficiency in depot emergency broadcast. |
| **TRK4-012** | **MEDIUM** | `stocklots` | `stocklots/counting.go:60-245` | Missing outbox events and WebSocket fanouts on cycle count submission and adjustment approval. |
| **TRK4-013** | **MEDIUM** | `retailer` | `retailer/auto_order_worker.go:665-672` | Direct `spanner.Client.Apply()` mutations bypassing ReadWriteTransaction and transactional outbox. |
| **TRK4-014** | **LOW** | `retailer` | `retailer/auto_order_worker.go:455, 683` | Use of `context.Background()` in worker helpers breaking deadline and distributed tracing propagation. |

---

## Detailed Line-by-Line Findings

### TRK4-001: [CRITICAL] `CreditViaDefaultPutawayInTxn` Always Fails on Perishable Products
- **File & Lines**:
  - `pegasusX/apps/backend-go/stocklots/credit_putaway.go:54-61`
  - `pegasusX/apps/backend-go/stocklots/lots.go:76-78`
  - Callers: `returns/inbound.go:570`, `supplier/returns.go:283`, `creditnote/repository_spanner.go:587`, `factory/supply_spanner.go:361`.
- **Observation**:
  In `credit_putaway.go`:
  ```go
  view, err := PutawayInTxn(ctx, txn, supplierID, warehouseID, PutawayRequest{
      LocationID: targetLoc,
      ProductID:  productID,
      Quantity:   quantity,
      SourceType: sourceType,
  })
  ```
  `PutawayRequest.ExpiryDate` is left `nil`. In `lots.go`:
  ```go
  if shelfMeta.Perishable && req.ExpiryDate == nil {
      return nil, fmt.Errorf("putaway: expiry_date required for perishable product")
  }
  ```
- **Logic Chain & Blast Radius**:
  When warehouse operators receive customer returns for perishable items (e.g. dairy, produce, meat), or when restock is credited from supplier return tickets, credit notes, or factory supply completion, `CreditViaDefaultPutawayInTxn` is invoked. Because `ExpiryDate` is `nil`, `PutawayInTxn` hard-fails, aborting the entire Spanner `ReadWriteTransaction`. The return cannot be confirmed, blocking warehouse reverse logistics for all perishable goods.
- **Recommendation**:
  In `CreditViaDefaultPutawayInTxn`, look up product shelf metadata (`loadProductShelfMeta`). If `Perishable` is true, compute a default fallback expiry date (e.g., `now.Add(shelfMeta.MinShelfLifeDays * 24 * time.Hour)` or inherit from the original order/batch) or allow caller-supplied optional expiry date.

---

### TRK4-002: [CRITICAL] Spanner Schema Constraint Violation in Inbound Receiving Short Shipments
- **File & Lines**:
  - `pegasusX/apps/backend-go/warehouse/receive_items.go:159-178`
  - `pegasusX/apps/backend-go/schema/spanner.ddl:509-545`
- **Observation**:
  In `receive_items.go`:
  ```go
  muts = append(muts, spanner.InsertMap("WarehouseSupplyRequests", map[string]any{
      "RequestId":   backorderID,
      "SupplierId":  supplierID,
      "WarehouseId": warehouseID,
      "Status":      "BACKORDER",
      "CreatedAt":   spanner.CommitTimestamp,
      "UpdatedAt":   spanner.CommitTimestamp,
  }))
  ```
  However, in `schema/spanner.ddl:509-528`, the table `WarehouseSupplyRequests` defines:
  ```sql
  CoverageStartDate TIMESTAMP NOT NULL,
  CoverageDays INT64 NOT NULL,
  ProjectedUnits INT64 NOT NULL,
  CommittedUnits INT64 NOT NULL,
  PendingConfirmationUnits INT64 NOT NULL,
  ```
  None of these columns have default values. Furthermore, `WarehouseSupplyRequestItems` defines `CreatedAt TIMESTAMP NOT NULL`, but `receive_items.go:171-177` omits `CreatedAt`.
- **Logic Chain & Blast Radius**:
  Whenever an inbound shipment arrives with a shortage (`received < shipped`), the receiving service attempts to create a backorder `WarehouseSupplyRequests` record. Cloud Spanner immediately rejects the commit with a `FAILED_PRECONDITION: Column CoverageStartDate is NOT NULL and does not have a DEFAULT clause`. As a result, warehouse operators cannot complete inbound receiving for any short shipment.
- **Recommendation**:
  Populate all required Spanner `NOT NULL` columns in `receive_items.go` (e.g. `CoverageStartDate: spanner.CommitTimestamp`, `CoverageDays: 0`, `ProjectedUnits: deltaQty`, `CommittedUnits: 0`, `PendingConfirmationUnits: 0`, and `CreatedAt: spanner.CommitTimestamp` on items), or update the DDL to make these fields nullable/defaulted.

---

### TRK4-003: [CRITICAL] Shop-Closed Order Cancellation Fails when `WMS_LOTS_ENABLED`
- **File & Lines**:
  - `pegasusX/apps/backend-go/order/shop_closed.go:522, 725`
  - `pegasusX/apps/backend-go/order/inventory_release.go:26-30, 73-80`
- **Observation**:
  In `order/shop_closed.go:522` (retailer cancel) and `order/shop_closed.go:725` (supplier resolve cancel):
  ```go
  if err := ReleaseReservationsFromOrderFields(ctx, txn, supplierID, warehouseID, orderSource, lineItemsRaw); err != nil {
      return err
  }
  ```
  `ReleaseReservationsFromOrderFields` passes empty string `""` for `orderID` to `ReleaseReservationsFromOrderFieldsWithID`.
  In `inventory_release.go:26-30`:
  ```go
  if stocklots.LotsEnabled() && orderID == "" {
      return fmt.Errorf("lot inventory release requires order_id when WMS_LOTS_ENABLED")
  }
  ```
- **Logic Chain & Blast Radius**:
  When WMS lot tracking is enabled (`WMS_LOTS_ENABLED=true`), any attempt by a retailer to cancel an order from the shop-closed dialog or by a supplier to resolve a shop-closed incident with `CANCEL` fails with a hard error. The order remains stuck in `SHOP_CLOSED_PENDING`, locking driver manifests and unreleased inventory.
- **Recommendation**:
  In `order/shop_closed.go:522` and `725`, call `ReleaseReservationsFromOrderFieldsWithID(ctx, txn, supplierID, warehouseID, req.OrderID, orderSource, lineItemsRaw)` instead of `ReleaseReservationsFromOrderFields`.

---

### TRK4-004: [HIGH] Dual Inventory Repositories & Double-Deduction Math in Legacy Inventory
- **File & Lines**:
  - `pegasusX/apps/backend-go/inventory/repository.go:29-34, 231-232`
  - `pegasusX/apps/backend-go/order/inventory_reservation.go:66-91`
  - `pegasusX/apps/backend-go/warehouse/demand_products.go:352`
- **Observation**:
  In `inventory/repository.go`:
  ```go
  func (i *InventoryLevel) Available() int64 {
      return i.QuantityOnHand - i.QuantityReserved
  }
  ```
  In `ReserveForOrder`:
  ```go
  level.QuantityOnHand -= quantity
  level.QuantityReserved += quantity
  ```
  Meanwhile, `order/inventory_reservation.go` mutates `SupplierInventoryV2` and `StockLots` without touching `InventoryLevels`. But `warehouse/demand_products.go:352` reads `InventoryLevels`.
- **Logic Chain & Blast Radius**:
  1. In `inventory/repository.go`, if `QuantityOnHand` is decremented by 10 AND `QuantityReserved` is incremented by 10, then `Available()` drops by 20 (double-deduction).
  2. Because order checkout and WMS write to `SupplierInventoryV2` and `StockLots`, the `InventoryLevels` table queried by `warehouse/demand_products.go` becomes stale and disconnected from real inventory balances, causing warehouse demand calculations and replenishment alerts to act on incorrect stock counts.
- **Recommendation**:
  1. Fix `ReserveForOrder` in `inventory/repository.go` to only increment `QuantityReserved` without decrementing `QuantityOnHand`.
  2. Migrate `warehouse/demand_products.go` to query `SupplierInventoryV2` and `StockLots` instead of legacy `InventoryLevels`.

---

### TRK4-005: [HIGH] Missing Stock Invalidation & Re-Rollup on Bin Quarantine or Deactivation
- **File & Lines**:
  - `pegasusX/apps/backend-go/stocklots/locations.go:114-184`
- **Observation**:
  `PatchBinInTxn` updates a bin's `LocationType` (e.g. to `QUARANTINE`) or sets `IsActive = false` on `WarehouseLocations`.
  However, it does not update the `Status` column of `StockLots` residing in that bin (which remain `Status = 'AVAILABLE'`), nor does it invoke `RollupInventoryV2InTxn`.
- **Logic Chain & Blast Radius**:
  FEFO queries and catalog ATP calculations filter `StockLots.Status = 'AVAILABLE'`. If a damaged or contaminated bin is quarantined by a warehouse manager, the inventory residing in that bin continues to be counted as available ATP and can be reserved and assigned to pick lists for customer orders.
- **Recommendation**:
  In `PatchBinInTxn`, when `LocationType == "QUARANTINE"` or `IsActive == false`, update all child `StockLots` in that bin to `Status = 'QUARANTINED'` (or `'HELD'`) and execute `RollupInventoryV2InTxn` for each affected product.

---

### TRK4-006: [HIGH] Unbatched Multi-Transaction Failure & Idempotency Hazard in Retailer Stock Counting
- **File & Lines**:
  - `pegasusX/apps/backend-go/retailer/store_stock.go:671-686`
  - `pegasusX/apps/backend-go/retailer/stock_count_commit.go:298-315`
- **Observation**:
  In `HandleStockCount` and `HandleStockCountCommit`:
  ```go
  actor := auth.ResolveRetailerUserID(claims)
  for _, l := range lines {
      if l.Variance == 0 {
          continue
      }
      if err := s.applyAdjust(r.Context(), orgID, locID, bin, l.Sku, l.Variance, actor, "cycle_count:"+countID); err != nil {
          writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error(), "sku": l.Sku})
          return
      }
      _ = s.syncReorderCurrentStock(r.Context(), orgID, l.Sku)
  }
  ```
  `applyAdjust` invokes its own standalone Spanner `ReadWriteTransaction`.
- **Logic Chain & Blast Radius**:
  If a stock count contains 50 SKU lines and SKU 30 fails (e.g. constraint violation or network blip), SKUs 1-29 are already committed to Spanner. The HTTP handler returns 409 Conflict. When the store manager fixes the issue or the client retries with the same idempotency key, SKUs 1-29 will be adjusted a second time, compounding and corrupting store stock records.
- **Recommendation**:
  Wrap the entire stock count commit inside a single Spanner `ReadWriteTransaction` that updates all `RetailerStockBalances`, inserts `RetailerStockMovements`, and records the count commit atomically.

---

### TRK4-007: [HIGH] Stale Reads on User-Facing Cart APIs
- **File & Lines**:
  - `pegasusX/apps/backend-go/retailer/repository_cart.go:53, 83`
- **Observation**:
  `ListByRetailer` and `ListByRetailerAll` execute queries with:
  ```go
  iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(5 * time.Second)).Query(ctx, stmt)
  ```
- **Logic Chain & Blast Radius**:
  Cart operations are read-your-own-writes sensitive. When a retailer adds an item to cart and immediately navigates to checkout or refreshes the cart, the 5-second staleness bound can return a state prior to the item addition. This causes "ghost" items, missing items, or outdated item quantities during checkout price calculations.
- **Recommendation**:
  Remove `ExactStaleness(5 * time.Second)` from cart read queries, or use `StrongRead()` (the default for `Single()`).

---

### TRK4-008: [HIGH] Credit Limit Check-to-Delivery Race Condition
- **File & Lines**:
  - `pegasusX/apps/backend-go/order/service.go:1492-1498`
  - `pegasusX/apps/backend-go/order/credit_guard.go:11-22`
- **Observation**:
  In `order/service.go`, credit reservation at order creation is guarded by `creditReserveAtCreateEnabled()` (which defaults to disabled). Even when enabled, `s.credit.Reserve` is called after the order creation transaction has already committed.
- **Logic Chain & Blast Radius**:
  If credit headroom is not reserved during order placement, a retailer with a $1,000 credit limit can place ten $1,000 orders in rapid succession. All 10 orders get approved, picked, packed, and loaded onto delivery trucks. When drivers arrive at the shop, only the first driver can deliver on credit; the remaining 9 orders fail at the doorstep due to `insufficient credit`, causing massive delivery aborts and returned stock.
- **Recommendation**:
  Reserve credit headroom inside the order creation transaction via `s.credit.ReserveOrderInTxn`, failing order creation if credit limit is breached.

---

### TRK4-009: [HIGH] Missing Financial Credit Note & Settlement on Return Inbound Confirmation
- **File & Lines**:
  - `pegasusX/apps/backend-go/returns/inbound.go:581-606`
- **Observation**:
  In `HandleInboundConfirm`, when a return line is confirmed as `RESTOCKED` or `WRITTEN_OFF`, the code updates `SupplierReturns` status and emits `EventReturnReceivedAtWarehouse`. However, it does not call or enqueue credit note generation, invoice adjustment, or retailer wallet refund.
- **Logic Chain & Blast Radius**:
  The physical inventory is returned to warehouse shelves or written off, but the retailer's financial ledger remains uncredited unless a supplier operator manually navigates to the credit notes portal and creates a credit note by hand.
- **Recommendation**:
  Wire an outbox event consumer or inline hook in `HandleInboundConfirm` to automatically generate a draft/issued `CreditNote` and credit the retailer's AR balance upon warehouse return receipt.

---

### TRK4-010: [MEDIUM] Unauthenticated HTTP Mock in Warehouse Pick Waves
- **File & Lines**:
  - `pegasusX/apps/backend-go/warehouse/pick_waves.go:48-85`
  - `pegasusX/apps/backend-go/stocklots/picking.go:56-285`
- **Observation**:
  `warehouse/pick_waves.go` computes mock coordinates `X: float64(len(item.SKU))` and issues an unauthenticated HTTP POST to `http://localhost:8000/pick-path` without timeouts or configurable URLs.
  Meanwhile, `stocklots/picking.go` implements durable Spanner `PickWaves` and `PickTasks` with S-shape aisle sorting.
- **Logic Chain & Blast Radius**:
  If a client calls `/v1/warehouse/pick-waves` instead of the WMS lot picking endpoints, the request will hang or fail when `localhost:8000` is unreachable.
- **Recommendation**:
  Deprecate or remove the mock HTTP call in `warehouse/pick_waves.go` and route all pick wave generation to `stocklots.CreatePickWaveInTxn`.

---

### TRK4-011: [MEDIUM] Arbitrary `LIMIT 200` Truncation in Warehouse Depot Broadcast
- **File & Lines**:
  - `pegasusX/apps/backend-go/warehouse/ops_broadcast.go:560-566`
- **Observation**:
  In `fanDepotRetailers`:
  ```sql
  SELECT DISTINCT RetailerId
  FROM Orders@{FORCE_INDEX=Idx_Orders_ByWarehouseCreated}
  WHERE WarehouseId = @wid AND UpdatedAt >= @since AND RetailerId IS NOT NULL
  LIMIT 200
  ```
- **Logic Chain & Blast Radius**:
  In a large distribution center serving 500+ active retailers, emergency broadcast advisories (e.g. yard congestion, reduced receiving hours, gate delays) will only be sent to the first 200 retailers. 300+ retailers will never receive the operational alert.
- **Recommendation**:
  Remove the arbitrary `LIMIT 200` or page through active retailer IDs for the warehouse, and query against `RetailerLocations` or active customer associations rather than scanning the `Orders` index.

---

### TRK4-012: [MEDIUM] Missing Outbox Events & WebSocket Fanouts in Cycle Counting
- **File & Lines**:
  - `pegasusX/apps/backend-go/stocklots/counting.go:60-245`
- **Observation**:
  `CreateCycleCountInTxn`, `SubmitCycleCountInTxn`, and `ApproveAdjustmentInTxn` write directly to Spanner tables `CycleCounts` and `InventoryAdjustments` without emitting any `OutboxEvents` or triggering WebSocket broadcasts.
- **Logic Chain & Blast Radius**:
  Warehouse managers and inventory auditors do not receive real-time notifications when cycle count variances are submitted or when adjustments are approved/rejected, and external ERP systems cannot subscribe to adjustment events.
- **Recommendation**:
  Add outbox event emissions (`events.EventInventoryAdjusted`, `events.EventCycleCountSubmitted`) inside the Spanner transactions in `stocklots/counting.go`.

---

### TRK4-013: [MEDIUM] Direct `Apply()` Mutation Bypassing Outbox in Auto-Order Worker
- **File & Lines**:
  - `pegasusX/apps/backend-go/retailer/auto_order_worker.go:665-672`
- **Observation**:
  `markReorderSuggestionConverted` executes `s.spannerClient.Apply(ctx, ...)` directly without a `ReadWriteTransaction` or transactional outbox event.
- **Logic Chain & Blast Radius**:
  Bypassing the transactional outbox prevents downstream analytics and AI prediction loops from tracking converted reorder suggestions.
- **Recommendation**:
  Use `ReadWriteTransaction` with `outbox.EmitJSON` to update reorder suggestions.

---

### TRK4-014: [LOW] Context.Background() Usage in Auto-Order Worker Helpers
- **File & Lines**:
  - `pegasusX/apps/backend-go/retailer/auto_order_worker.go:455, 683`
- **Observation**:
  `bucketTaken` and `recordAutoOrderRun` call Spanner operations passing `context.Background()` instead of the active request context.
- **Logic Chain & Blast Radius**:
  Request deadlines, cancellations, and distributed OpenTelemetry trace contexts are dropped.
- **Recommendation**:
  Thread `ctx context.Context` through `bucketTaken` and `recordAutoOrderRun`.

---

## Deep Architectural Open Questions & Edge-Case Dilemmas

1. **Multi-Supplier Cross-Docking vs. Direct Consolidation**:
   - *Dilemma*: When a retailer orders from 3 different suppliers in a single unified checkout, items originate from separate supplier warehouses. If the dispatch engine attempts to deliver via a single driver manifest to the retailer's store, who coordinates the intermediate cross-docking staging at a consolidation hub?
   - *Current Gap*: The order service splits multi-supplier carts into distinct child orders per supplier, but lacks a cross-docking state machine to coordinate joint delivery schedules.

2. **FEFO Lot Depletion vs. Pick Path Route Optimization**:
   - *Dilemma*: Strict FEFO picking requires picking the lot with the earliest expiry date, which may be located in Aisle Z, Bin 40, while an identical SKU with 3 days later expiry is located in Aisle A, Bin 1. In large warehouses, strict FEFO can multiply picker walking distance by 300%.
   - *Current Gap*: `stocklots/fefo.go` selects lots strictly by `ExpiryDate ASC` without weighting bin distance or picker travel cost.

3. **Offline Store Stock Counting vs. Concurrent POS Sales**:
   - *Dilemma*: When a retail store conducts an offline cycle count on the floor, sales continue to occur at the POS. When the count is committed, the base version conflict protocol (`COUNT_VERSION_CONFLICT`) may reject the count even if the counted SKU was never sold during the counting window.
   - *Current Gap*: `stock_count_commit.go` uses a coarse bin-level version (`RetailerStockLocationVersions`) rather than per-SKU versioning, causing entire count batches to conflict if ANY unrelated SKU was sold.

4. **Multi-Supplier Credit Headroom Partitioning**:
   - *Dilemma*: If a retailer has $500 available credit and submits a multi-supplier cart totaling $800 ($500 for Supplier A and $300 for Supplier B), how should credit headroom be allocated if credit is shared across the platform?
   - *Current State*: Credit profiles are scoped per `(RetailerId, SupplierId)`, so each supplier evaluates credit independently, which prevents platform-wide credit pooling.

---

## Conclusion & Verification Plan
All findings listed above have been verified with exact file paths, line numbers, and Spanner DDL cross-references. Detailed remediation instructions and architectural recommendations are documented for Track 4 engineers.
