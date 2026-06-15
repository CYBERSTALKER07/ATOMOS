# Migration Runbook — Warehouse Stock Policy & Supply Request Items

**Change ID:** `20250616-warehouse-stock-policy-supply-items`  
**Owner:** Platform / Warehouse ops  
**Blast radius:** Spanner `Warehouses`, `SupplierInventoryV2`, `WarehouseSupplyRequests`, new `WarehouseSupplyRequestItems`; warehouse/factory supply APIs; retailer catalog stock enrichment; checkout backorders  
**Downtime:** None (additive DDL + backward-compatible API)

---

## 1. What this migration enables

| Capability | Depends on |
|---|---|
| Per-warehouse default OOS policy + operating schedule (display-only) | `Warehouses.DefaultOutOfStockPolicy`, `Warehouses.OperatingSchedule` |
| Per-SKU OOS policy + reorder threshold at warehouse | `SupplierInventoryV2.OutOfStockPolicy`, `SupplierInventoryV2.ReorderThreshold` |
| Supply requests with SKU line items, priority, notes, region | `WarehouseSupplyRequestItems` + enriched `WarehouseSupplyRequests` columns |
| Retailer catalog stock badges + backorder checkout | `SupplierInventoryV2` reads + policy resolution |
| Supplier topology seeds starter inventory + warehouse policy | Topology replace writes `SupplierInventoryV2` |

**Canonical DDL (live instances that predate the columns):**

`apps/backend-go/schema/migrations/20250616_warehouse_stock_policy_supply_items.ddl`

Fresh installs that apply full `schema/spanner.ddl` already include the columns and child table — skip §3 if `INFORMATION_SCHEMA` shows them.

---

## 2. Preconditions

- [ ] Backend build containing `warehouse/stock_policy.go`, `warehouse/ops_settings.go`, `warehouse/ops_inventory_policy.go`, `catalog/stock.go`, `order/inventory_plan.go`, and supply-request item persistence is staged.
- [ ] Spanner admin IAM: `spanner.databases.updateDdl` on target database.
- [ ] Maintenance window **not required** — additive `ALTER TABLE` + new interleaved table only.
- [ ] Warehouse portal / factory portal / retailer clients updated to consume enriched supply-request and stock fields.

---

## 3. Spanner DDL — production / staging

```bash
cd pegasusX/apps/backend-go
gcloud spanner databases ddl update DATABASE_ID \
  --instance=INSTANCE_ID \
  --ddl-file=schema/migrations/20250616_warehouse_stock_policy_supply_items.ddl
```

Verify:

```sql
SELECT TABLE_NAME, COLUMN_NAME
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_NAME IN ('Warehouses', 'SupplierInventoryV2', 'WarehouseSupplyRequests')
  AND COLUMN_NAME IN (
    'DefaultOutOfStockPolicy', 'OperatingSchedule',
    'OutOfStockPolicy', 'ReorderThreshold',
    'Priority', 'Notes', 'RegionId', 'RequestedDeliveryDate', 'TotalVolumeVU'
  )
ORDER BY TABLE_NAME, COLUMN_NAME;

SELECT TABLE_NAME
FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_NAME = 'WarehouseSupplyRequestItems';
```

---

## 4. Post-deploy smoke

1. `GET /v1/warehouse/ops/settings?warehouse_id={id}` — returns `default_out_of_stock_policy`, `ops_always_available`.
2. `PATCH /v1/warehouse/ops/inventory/{product_id}/policy` — per-SKU policy update.
3. `POST /v1/warehouse/supply-requests` with JSON `items[]` — create succeeds.
4. `GET /v1/warehouse/supply-requests` and `GET /v1/warehouse/supply-requests/{id}` — list/detail include `items`, `priority`, `notes`, `item_count`.
5. `GET /v1/factory/supply-requests` — factory list includes same line items.
6. Retailer catalog — `available_stock`, `is_out_of_stock`, `accepts_backorder` populated.

Local SSMR smoke (fresh emulator applies full DDL via `cmd/setup`):

```bash
./scripts/smoke_ssmr.sh
```

Expect markers: `PX_E2E_WAREHOUSE_STOCK_POLICY_OK`, `PX_E2E_WAREHOUSE_SUPPLY_REQUEST_ITEMS_OK`, `PX_E2E_FACTORY_SUPPLY_REQUEST_OK`, `PX_E2E_RETAILER_CATALOG_STOCK_OK`.

---

## 5. Rollback

- **API:** Revert backend deployment; old clients ignore new JSON fields.
- **DDL:** Do not drop columns or `WarehouseSupplyRequestItems` in production without a coordinated data migration — Spanner `DROP COLUMN` / `DROP TABLE` is irreversible for row data.

---

## 6. Notes

- Legacy forecast-based supply create (`POST ?start_date=&days=`) remains supported; line-item create requires JSON body with `items`.
- `OperatingSchedule` is informational only — dispatch and supply fulfillment are not blocked by hours (`ops_always_available: true` in settings API).
