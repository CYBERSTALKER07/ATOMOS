# Retailer Floor Assist (Retail OS Phase 6)

Section help queue for large stores.

## Dependencies

- Hard: **SECTIONS** + **TEAM**
- Soft: SHIFTS (prefer on-duty later)

First ticket auto-enables TEAM / STORE_STOCK / SECTIONS / CUSTOMER_ASSIST as needed.

## Lifecycle

`OPEN` → `CLAIMED` → `DONE` (or `CANCELLED`)

SLA from pack config `sla_minutes` (default 15). Notifies section staff + owners via NotificationWriter when wired.

## APIs

| Method | Path |
|--------|------|
| GET/POST | `/v1/retailer/assist/tickets` |
| POST | `/v1/retailer/assist/tickets/{id}/claim` |
| POST | `/v1/retailer/assist/tickets/{id}/complete` |
| POST | `/v1/retailer/assist/tickets/{id}/cancel` |

Create: `stock.view`. Claim/complete: `assist.respond`.

## Clients

Desktop `/assist` · Android/iOS Profile → **Floor assist**
