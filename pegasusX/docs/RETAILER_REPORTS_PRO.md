# Retailer Reports Pro (Retail OS Phase 6)

Aggregates over POS sales, store stock, and shift variance. No new fact tables — reads existing ledgers.

## Dependencies

None hard (CORE). Soft empty modules when POS/stock off.

First report GET auto-enables **REPORTS_PRO**. Permission: `reports.view`.

## APIs

| Method | Path |
|--------|------|
| GET | `/v1/retailer/reports/summary` |
| GET | `/v1/retailer/reports/sales?group_by=sku\|hour\|cashier` |
| GET | `/v1/retailer/reports/inventory` |
| GET | `/v1/retailer/reports/shifts` |
| GET | `/v1/retailer/reports/export?report=sales\|inventory\|shifts` (CSV) |

Default window: last 7 days (`from`/`to` RFC3339). Money int64 minor units.

## Clients

Desktop `/reports` + CSV · Android/iOS Profile → **Reports Pro**
