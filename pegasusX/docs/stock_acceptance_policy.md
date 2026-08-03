# Stock acceptance policy (out-of-stock behaviour)

Warehouse and supplier operators control whether retailer checkout is **blocked** when available stock is below the requested quantity, or **accepted** with partial fulfillment and optional backorder lines.

This is **not** fair-share allocation (O9-1 constrained allocation). Allocation runs only after an order is accepted.

## Policy values

| Value | Boss intent | Behaviour at checkout |
|-------|-------------|------------------------|
| `REJECT` | Block when short | Line rejected if ATP &lt; requested qty |
| `ACCEPT_BACKORDER` | Accept when short | Fulfill up to ATP; remainder → backorder sibling order |
| `INHERIT` | (per-SKU only) | Use warehouse `DefaultOutOfStockPolicy` |

Default warehouse policy: **`REJECT`**.

## Schema

| Table | Column | Notes |
|-------|--------|--------|
| `Warehouses` | `DefaultOutOfStockPolicy` | Warehouse-wide default |
| `Warehouses` | `ShowStockCountsToRetailers` | Whether retailers see on-hand caps in catalog |
| `SupplierInventoryV2` | `OutOfStockPolicy` | Per warehouse + SKU override (`INHERIT` / `REJECT` / `ACCEPT_BACKORDER`) |

Migration: `apps/backend-go/schema/migrations/20250616_warehouse_stock_policy_supply_items.ddl`.

## Available-to-promise (ATP) today

```
ATP = QuantityOnHand - QuantityReserved
```

Implemented in `PlanInventoryCheckout` ([`apps/backend-go/order/inventory_plan.go`](apps/backend-go/order/inventory_plan.go)).

### Known gap: projected stock (deferred)

Boss target includes projected ATP:

```
on-hand − reserved − committed outbound + expected inbound (within horizon)
```

This is **not implemented**. Supply request inbound (`WarehouseSupplyRequestItems`) is not included in checkout guards. A separate formula sign-off and implementation pass is required before enabling projected stock checks.

## Enforcement call sites

1. **Checkout preview** — `checkout_preview.go` → `PlanInventoryCheckout`
2. **Order create** (including manual pre-orders) — `order/service.go` `Create` → `PlanInventoryCheckout`
3. **Unified checkout** — `unified_checkout.go` (inventory errors via `MarshalInventoryCheckoutError`)
4. **Pre-order edit / AI confirm** — `preorder_service.go` → `applyPreorderInventoryGuard`
5. **Stock reservation** — `ReserveLineItemsInTxn` reserves **fulfillable** qty only (`inventory_reservation.go`)
6. **Backorder sweeper** — re-plans when stock returns (`backorder_sweeper.go`)
7. **Allocation (O9-1)** — separate path after acceptance; does not replace checkout policy

On `REJECT`, API returns `InventoryCheckoutError` with codes `PARTIAL_OUT_OF_STOCK_REJECTED` or `ALL_ITEMS_OUT_OF_STOCK`.

## Portal editing

| Actor | Warehouse default | Per-SKU override |
|-------|-------------------|------------------|
| Warehouse staff | Settings → ops policy | Inventory list → stock policy column |
| Supplier admin | Topology editor | Supplier inventory → policy column |

### API routes

- Warehouse default: `PUT /v1/warehouse/ops/settings` (`default_out_of_stock_policy`)
- Per-SKU (warehouse): `PATCH /v1/warehouse/ops/inventory/{productID}/policy`
- Warehouse default (supplier): topology `PUT /v1/supplier/topology`
- Per-SKU (supplier): `PATCH /v1/supplier/inventory/policy`

## Retailer UX

- Catalog / cart caps use `accepts_backorder` and preview `default_out_of_stock_policy`
- `ACCEPT_BACKORDER` allows cart qty above visible ATP; checkout may split into fulfill + backorder

## Tests

- `apps/backend-go/order/inventory_plan_test.go` — policy resolution and plan split logic
- `apps/backend-go/order/warehouse_policy_test.go` — orderable quantity caps

## Related

- O9-1 constrained allocation: warehouse scoring after order acceptance (`CONSTRAINED_ALLOCATION_ENABLED`)
- Fair-share: intentionally not built; this policy is the business control point
