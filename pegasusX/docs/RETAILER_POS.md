# Retailer POS (Retail OS Phase 4 + Offline)

Walk-in sales against **store stock** (not warehouse ATP). All money is **int64 minor units**.

## Dependencies

- Hard: **STORE_STOCK** (sale decrements on-hand)
- Soft: TEAM, SHIFTS (when SHIFTS on + `require_shift_to_open_register`, POS open requires clock-in)

Creating a register auto-enables STORE_STOCK + POS packs.

## Flow

1. Create **register** at a location  
2. **Open session** online (opening float minor)  
3. **Sale**: lines + tenders; stock from FLOOR then BACKROOM  
4. **Offline cash**: queue locally with `client_sale_id`; sync on reconnect  
5. **Void** restocks (synced sales only); manager/owner for unrestricted void  
6. **Close session** with counted cash → variance (blocked while offline queue pending)

## APIs

| Method | Path |
|--------|------|
| GET/POST | `/v1/retailer/registers` |
| POST | `/v1/retailer/pos/sessions/open` |
| GET | `/v1/retailer/pos/sessions/{id}` |
| POST | `/v1/retailer/pos/sessions/{id}/close` |
| POST | `/v1/retailer/pos/sales` |
| POST | `/v1/retailer/pos/sales/{id}/void` |
| POST | `/v1/retailer/pos/sales/{id}/refund` (alias void) |

## Sale rules

- `sum(line qty * unit_price_minor) == sum(tenders)`  
- Default tender = full CASH if omitted  
- Idempotency-Key required on mutations (`pos-sale:{client_sale_id}` recommended)  

### Offline fields (optional body)

```json
{
  "client_sale_id": "uuid",
  "client_created_at": "RFC3339",
  "origin": "online | offline"
}
```

| Rule | Behavior |
|------|----------|
| `origin=offline` + non-CASH tender | **422** `offline_card_forbidden` |
| Session closed at sync | **409** `session_closed_for_offline_sync` |
| Duplicate `client_sale_id` | Returns existing sale (no double stock) |
| Session open | Must be opened **online** first; offline open not supported |

## Cash recon (session close)

```
expected = opening_float + cash_tenders(COMPLETED sales on server)
variance = closing_cash - expected
```

Offline sales affect expected **only after sync**. Clients block close while pending/failed queue items exist for the session.

## Schema

- `schema/migrations/20260802_retail_os_phase4_pos.ddl`
- `schema/migrations/20260802_retail_os_pos_offline.ddl` — `ClientSaleId`, `Origin`, `ClientCreatedAt` + unique null-filtered index

## Clients

| Surface | Offline |
|---------|---------|
| Desktop `/pos` | Park cart + pending queue + flush + close block |
| Android POS | Room `pending_pos_sales` + flush |
| iOS POS | `PendingPosStore` UserDefaults + NWPathMonitor flush |

## Tests

```bash
cd apps/backend-go && go test ./retailer/ -run 'POS' -count=1
```
