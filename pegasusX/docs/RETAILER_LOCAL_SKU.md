# Retailer Local / Manual POS SKUs (L6)

## Purpose

POS must sell non-Pegasus goods (bread, bags, local sourced) between wholesales without polluting supplier ATP or reorder suggestions.

## Model

- Table: `RetailerLocalCatalog`
- IDs use `local:` namespace (or server-assigned `local_sku_id`)
- Stock balances key by SKU string — same `RetailerStockBalances` ledger

## APIs

```
GET/POST  /v1/retailer/local-skus
PATCH     /v1/retailer/local-skus/{id}
```

## Clients

- Desktop: `/stock/local-skus`
- Android: Settings → Local SKUs
- iOS: Settings → Local SKUs

## Demand guard

Reorder suggestion batch **skips** SKUs with `local:` prefix — never emit supplier `ReorderSuggestions` for local goods.

## Barcode collision

If a barcode matches a Pegasus catalog SKU, **Pegasus wins** at POS search / scan.
