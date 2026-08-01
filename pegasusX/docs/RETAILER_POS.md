# Retailer POS (Retail OS Phase 4)

Walk-in sales against **store stock** (not warehouse ATP). All money is **int64 minor units**.

## Dependencies

- Hard: **STORE_STOCK** (sale decrements on-hand)
- Soft: TEAM, SHIFTS (Phase 5: when SHIFTS on + `require_shift_to_open_register`, POS open requires clock-in)

Creating a register auto-enables STORE_STOCK + POS packs.

## Flow

1. Create **register** at a location  
2. **Open session** (opening float minor)  
3. **Sale**: lines + tenders; stock from FLOOR then BACKROOM  
4. **Void** restocks; manager/owner for unrestricted void  
5. **Close session** with counted cash → variance  

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
- Online-required (no offline sale queue in v1)  
- Idempotency-Key required on mutations  

## Cash recon (session close)

```
expected = opening_float + cash_tenders(COMPLETED sales)
variance = closing_cash - expected
```

Voided sales do not add cash.

## Schema

`schema/migrations/20260802_retail_os_phase4_pos.ddl`

## Clients

- Desktop `/pos`  
- Android / iOS Profile → **POS**
