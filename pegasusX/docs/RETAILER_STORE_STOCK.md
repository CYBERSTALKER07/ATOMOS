# Retailer store stock (Retail OS Phase 3)

**Separate from warehouse ATP.** Supplier `InventoryLevels` remain logistics SoT.  
Store ledger tracks what is **in the retailer’s building**.

## Bins

| Bin | Use |
|-----|-----|
| BACKROOM | Default receive target |
| FLOOR | Sellable / shelf |
| QUARANTINE | Damaged / hold |

## APIs

| Method | Path | Notes |
|--------|------|--------|
| GET | `/v1/retailer/stock?location_id=` | Balances for location |
| GET | `/v1/retailer/stock/{sku}` | Per-SKU balances + recent movements |
| GET | `/v1/retailer/stock/movements` | Ledger query |
| POST | `/v1/retailer/stock/receive-sessions` | From Pegasus `order_id`; default confirm into BACKROOM |
| POST | `/v1/retailer/stock/receive-sessions/{id}/confirm` | Confirm draft |
| POST | `/v1/retailer/stock/transfer` | Bin or location transfer |
| POST | `/v1/retailer/stock/adjust` | `qty_delta` (+/-) |
| POST | `/v1/retailer/stock/counts` | Cycle count; `commit:true` posts variance |

First receive auto-enables **STORE_STOCK** capability pack.

## Math

```
available = on_hand - reserved
receive  += accepted_qty on target bin
transfer atomic −from +to
adjust   on_hand += qty_delta (reject if result < 0)
count    variance = counted - system; apply as adjust
```

## Reorder AI

`replenishment.retailerStockEstimate` prefers `SUM(OnHand)` from `RetailerStockBalances` when ledger rows exist; otherwise legacy last completed line qty.

## Schema

Migration: `schema/migrations/20260802_retail_os_phase3_store_stock.ddl`

## Clients

Desktop `/stock`, Android/iOS Profile → **Store stock**.
