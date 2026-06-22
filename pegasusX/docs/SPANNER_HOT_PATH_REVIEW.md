# Spanner Hot-Path Review — Pilot

Review date: 2026-06-22. Re-run when retailer count &gt; 150 or dispatch preview &gt; 500 orders.

## Summary

| Path | Route / caller | Index | Stale read | Risk |
|------|----------------|-------|------------|------|
| Retailer order list | `order.ListRetailerOrders` | `Idx_Orders_ByRetailerCreated` | No (fresh) | Low |
| Retailer tracking | `retailer.ListTrackingOrders` | `Idx_Orders_ByRetailerCreated` | No | Low |
| Supplier order list | `supplier.ListOrders` | `Idx_Orders_BySupplierUpdated` | 15s | Low |
| Dispatch preview | `dispatch.FetchDispatchable` | `Idx_Orders_BySupplierStatusUpdated` | 15s | Medium |
| Warehouse ops orders | `warehouse/ops_portal` | `Idx_Orders_ByWarehouseCreated` | 15s | Low |
| Inventory list | `warehouse.GetInventoryList` | PK `SupplierInventoryV2` | 15s | Medium at scale |
| Driver active orders | `bootstrap.driverOrderListQuery` | None (driver filter) | 15s | Medium — add index if &gt; 20 stops |

## 1. Order list (retailer)

```sql
-- order/repository_spanner.go ListRetailerOrders
FROM Orders@{FORCE_INDEX=Idx_Orders_ByRetailerCreated}
WHERE RetailerId = @retailer_id
ORDER BY CreatedAt DESC LIMIT @limit
```

**Verdict:** Correct index. Default limit 25. Keep limit ≤ 50 on mobile.

## 2. Dispatch preview

```sql
-- dispatch/repository.go FetchDispatchable
FROM Orders@{FORCE_INDEX=Idx_Orders_BySupplierStatusUpdated} o
JOIN Retailers r ON o.RetailerId = r.RetailerId
WHERE o.SupplierId = @supplierId
  AND o.Status = 'PENDING'
  AND ... eligibility filters ...
ORDER BY o.UpdatedAt DESC LIMIT @limit
```

**Verdict:** Added `Idx_Orders_BySupplierStatusUpdated` (migration `20250622_pilot_hot_path_indexes.ddl`). Stale read 15s acceptable for preview UI.

**Watch:** `enrichDispatchableVolumes` does a second query per batch — monitor if line-item volume lookup becomes hot.

## 3. Inventory list

```sql
-- warehouse/repository_spanner.go GetInventoryList
FROM SupplierInventoryV2 si
INNER JOIN Warehouses w ON ...
WHERE si.WarehouseId = @wid
ORDER BY si.ProductId
LIMIT @limit OFFSET @offset  -- when limit > 0
```

**Verdict:** Full warehouse scan when `limit=0` (default). OK for &lt; 500 SKUs. Use `?limit=500&offset=N` for pagination at scale.

**Mitigation applied:** 15s stale read on read path; `?fresh=1` for e2e/strong reads only.

## 3b. Notification inbox (2026-06 audit)

```sql
-- notifications/repository.go ListForRecipient / UnreadCount
FROM Notifications WHERE RecipientId = @rid ...
```

**Verdict:** **Fixed** — `ExactStaleness(15s)` on list + unread count (cost governance audit).

## 4. Supplier portal order desk

```sql
-- supplier/repository_spanner.go ListOrders
FROM Orders@{FORCE_INDEX=Idx_Orders_BySupplierUpdated}
WHERE SupplierId = @supplierId
ORDER BY UpdatedAt DESC
```

**Verdict:** Previously sorted by `UpdatedAt` without matching index — **fixed** with `Idx_Orders_BySupplierUpdated`.

## 5. Query discipline rules (pilot)

1. **Dashboards and previews:** prefer `ExactStaleness(15s)` — already on dispatch, supplier list, inventory, driver list.
2. **Mutations and payment:** always strong read inside RW transaction (unchanged).
3. **Never** `SELECT *` on `Orders` — `orderSelectColumns` projection only.
4. **Cap limits:** dispatch preview default 300, max 5000 — do not raise without load cert.
5. **Monitor** Spanner query insights weekly in GCP console; export top 10 by CPU.

## 6. Stale-read audit (2026-06 cost governance)

CI gate: `bash scripts/validate_spanner_stale_reads.sh` — flags new `.Single().Query` without `WithTimestampBound`. Baseline allowlist: `scripts/spanner_stale_read_allowlist.txt`.

| Classification | Examples | Action |
|----------------|----------|--------|
| `stale_ok` | dashboards, lists, previews, inbox | `ExactStaleness(15s)` or `MaxStaleness(15s)` |
| `must_be_fresh` | checkout, payment, dispatch execute, inventory `?fresh=1` | strong read inside RW txn or `fresh=1` bypass |

**Fixed this audit:** notification inbox list + unread count.

**Backlog (allowlisted):** retailer catalog reads, replenishment insights, factory supply lists — add stale bound when touched.

## 7. When to add capacity

| Signal | Action |
|--------|--------|
| CPU &gt; 65% sustained | Raise PU in steps of 100; do not skip query review |
| Hot row on warehouse dispatch | Serialize dispatch window; avoid parallel manifest commits |
| Retailer list p99 &gt; 500ms | Add covering index on `(RetailerId, Status, CreatedAt DESC)` |

## Verification

```bash
cd pegasusX/apps/backend-go && go test ./order/... ./dispatch/... ./supplier/... ./warehouse/... -short -count=1
```
