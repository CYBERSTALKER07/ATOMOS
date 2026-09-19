# Handoff Report: Track 4 Codebase Audit (Retailer, Warehouse & Stock Fulfillment Domain)

**Agent**: `audit_track4_retailer_warehouse`  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/audit_track4_retailer_warehouse`  
**Date**: 2026-08-30  
**Handoff Type**: Hard (Task Complete)

---

## 1. Observation

Direct line-by-line inspection of `pegasusX/apps/backend-go` across `order/`, `stocklots/`, `warehouse/`, `inventory/`, `retailer/`, `returns/`, `credit/`, and `schema/spanner.ddl` revealed the following exact observations:

1. **Perishable Putaway Crash in Restock / Returns / Credit Notes**:
   - `pegasusX/apps/backend-go/stocklots/credit_putaway.go:54-61`: `CreditViaDefaultPutawayInTxn` constructs `PutawayRequest` with `ExpiryDate: nil`.
   - `pegasusX/apps/backend-go/stocklots/lots.go:76-78`: `PutawayInTxn` checks `if shelfMeta.Perishable && req.ExpiryDate == nil`, returning error `putaway: expiry_date required for perishable product`.
   - Direct Callers: `returns/inbound.go:570`, `supplier/returns.go:283`, `creditnote/repository_spanner.go:587`, `factory/supply_spanner.go:361`.

2. **Spanner DDL NOT NULL Constraint Violation in Inbound Receiving**:
   - `pegasusX/apps/backend-go/warehouse/receive_items.go:159-178`: Inserts a `WarehouseSupplyRequests` row without `CoverageStartDate`, `CoverageDays`, `ProjectedUnits`, `CommittedUnits`, `PendingConfirmationUnits`, and inserts `WarehouseSupplyRequestItems` without `CreatedAt`.
   - `pegasusX/apps/backend-go/schema/spanner.ddl:509-545`: All columns above are defined as `NOT NULL` without default clauses.

3. **Shop-Closed Cancellation Crash Under WMS Lot Tracking**:
   - `pegasusX/apps/backend-go/order/shop_closed.go:522, 725`: Calls `ReleaseReservationsFromOrderFields(...)`.
   - `pegasusX/apps/backend-go/order/inventory_release.go:26-30, 79`: Passes `orderID = ""` to `ReleaseReservationsForOrderInTxn`, which returns `fmt.Errorf("lot inventory release requires order_id when WMS_LOTS_ENABLED")`.

4. **Dual Inventory Models & Double-Deduction Math in Legacy Repository**:
   - `pegasusX/apps/backend-go/inventory/repository.go:29-34, 231-232`: `ReserveForOrder` decrements `QuantityOnHand` AND increments `QuantityReserved`, while `Available()` calculates `QuantityOnHand - QuantityReserved` (double subtraction).
   - `pegasusX/apps/backend-go/warehouse/demand_products.go:352`: Queries `InventoryLevels` table while order reservations and WMS write to `SupplierInventoryV2` and `StockLots`.

5. **Quarantined Bin Inventory Remains Sellable**:
   - `pegasusX/apps/backend-go/stocklots/locations.go:114-184`: `PatchBinInTxn` modifies `LocationType` to `QUARANTINE` or `IsActive` to `false`, but does not update the `Status` of child `StockLots` or invoke `RollupInventoryV2InTxn`.

6. **Unbatched Multi-Transaction Failure & Idempotency Hazard in Retailer Stock Counting**:
   - `pegasusX/apps/backend-go/retailer/store_stock.go:671-686` and `pegasusX/apps/backend-go/retailer/stock_count_commit.go:298-315`: Loops over counted SKU lines and calls `s.applyAdjust(...)`, each opening a standalone Spanner `ReadWriteTransaction`. Partial failure leaves partially applied adjustments and retries apply duplicate adjustments.

7. **Stale Reads on User-Facing Cart APIs**:
   - `pegasusX/apps/backend-go/retailer/repository_cart.go:53, 83`: Uses `spanner.ExactStaleness(5 * time.Second)` for `ListByRetailer` and `ListByRetailerAll`.

8. **Credit Limit Check-to-Delivery Race Condition**:
   - `pegasusX/apps/backend-go/order/service.go:1492-1498`: Credit reservation at create is lazy, optional, and executed outside the order creation transaction.

9. **Inbound Return Confirmation Disconnected from Financial Credit Notes**:
   - `pegasusX/apps/backend-go/returns/inbound.go:581-606`: Updates physical status and emits `EventReturnReceivedAtWarehouse`, but never triggers credit note generation or wallet settlement.

---

## 2. Logic Chain

1. **Perishable Returns**: Because customer returns, supplier return tickets, credit notes, and factory supply completion all funnel through `CreditViaDefaultPutawayInTxn` without an `ExpiryDate`, and because perishable products mandate an expiry date, all perishable restock transactions abort immediately on execution.
2. **Short Inbound Receiving**: Because `receive_items.go` omits 6 required `NOT NULL` columns during backorder generation, any shipment arriving with `received < shipped` causes Spanner transaction rejection, blocking inbound dock receiving.
3. **Shop-Closed Cancellation**: Because `order/shop_closed.go` omits `orderID` when calling inventory release, all cancellations on shop-closed orders fail when `WMS_LOTS_ENABLED` is active.
4. **Inventory Duality**: `inventory/repository.go` and `warehouse/demand_products.go` operate on a legacy `InventoryLevels` table with broken math, while `order/` and `stocklots/` operate on `SupplierInventoryV2` and `StockLots`. This duality causes replenishment alerts and warehouse demand projections to diverge from physical warehouse balances.
5. **Stock Counting**: In-store stock count commits do not execute in an atomic transaction; instead, each SKU executes its own `ReadWriteTransaction`, meaning network interruptions or line errors result in corrupted partial inventory adjustments that double upon retry.

---

## 3. Caveats

- **External Route Optimizer**: Audited the backend Go integration points with the optimizer (`warehouse/auto_dispatch.go`, `warehouse/dispatch_execute.go`, `warehouse/pick_waves.go`). The Python/C++ optimizer engine itself was outside Track 4 scope.
- **Client Quicktype Bindings**: Checked Go contracts and payload schemas; native Swift/Kotlin quicktype stub generation was spot-checked against `contracts/events.schema.json`.
- **No Direct Code Modifications**: In strict adherence to Explorer read-only guidelines, no application source files were modified. All proposed changes are documented in `findings.md`.

---

## 4. Conclusion

Track 4 possesses a solid architectural foundation with robust Spanner outbox patterns and FEFO picking mechanics. However, **3 Critical Defects** and **6 High-Severity Inconsistencies** must be resolved before production deployment:
1. Fix `CreditViaDefaultPutawayInTxn` for perishable items.
2. Fix missing `NOT NULL` columns in `receive_items.go`.
3. Pass `req.OrderID` to `ReleaseReservationsFromOrderFieldsWithID` in `order/shop_closed.go`.
4. Eliminate legacy `InventoryLevels` duality and fix `inventory/repository.go` double-deduction math.
5. Invalidate and re-rollup `StockLots` when a warehouse bin is quarantined or deactivated.
6. Wrap retailer stock count commits in a single atomic Spanner transaction.
7. Remove stale reads from cart list endpoints.
8. Enforce atomic credit limit reservations at order creation.
9. Link return confirmation to automated credit note generation.

---

## 5. Verification Method

To independently verify these findings:
1. **Perishable Putaway**: Inspect `pegasusX/apps/backend-go/stocklots/credit_putaway.go:54-61` and run:
   ```bash
   cd pegasusX/apps/backend-go && go test -v ./stocklots -run TestCreditViaDefaultPutaway
   ```
2. **Inbound Receiving Backorder**: Inspect `pegasusX/apps/backend-go/warehouse/receive_items.go:159-178` against `schema/spanner.ddl:509-545`.
3. **Shop Closed Lot Release**: Inspect `pegasusX/apps/backend-go/order/shop_closed.go:522` and run:
   ```bash
   cd pegasusX/apps/backend-go && go test -v ./order -run TestShopClosed
   ```
4. **Cart Stale Reads**: Inspect `pegasusX/apps/backend-go/retailer/repository_cart.go:53`.
5. **Full Test Suite**:
   ```bash
   cd pegasusX/apps/backend-go && go test ./order/... ./stocklots/... ./warehouse/... ./retailer/... ./returns/... ./inventory/... ./credit/...
   ```
