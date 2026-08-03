# Retailer Shifts & Time (Retail OS Phase 5)

Labor clock + cash shift reconciliation. Soft dependency of **POS**; hard dependency **TEAM**.

All money is **int64 minor units**.

## Dependencies

| Pack | Relation |
|------|----------|
| TEAM | hard (staff identity) |
| POS | soft (link session for cash sales on close) |

First clock-in or shift open **auto-enables SHIFTS** with defaults:

```json
{
  "require_clock_in": true,
  "require_shift_to_open_register": true,
  "max_shift_hours": 12,
  "variance_alert_minor": 10000
}
```

When SHIFTS is **off**, POS open is unrestricted (Phase 4 behavior).

## Flow

1. **Clock in** (location defaults to JWT `active_location_id` / primary)
2. Optional **open shift** (opening float + optional register)
3. **Open POS session** — blocked with `clock_in_required` if pack config requires it
4. Sell as usual (Phase 4)
5. **Close POS** — variance alert if `|variance| >= variance_alert_minor`
6. **Close shift** — expected cash = float + cash tenders on linked POS session; owners notified on large variance
7. **Clock out**

Max shift hours: open time entry older than `max_shift_hours` is auto-closed; next POS open returns `clock_in_expired_reclock_required`.

## APIs

| Method | Path |
|--------|------|
| POST | `/v1/retailer/time/clock-in` |
| POST | `/v1/retailer/time/clock-out` |
| GET | `/v1/retailer/time/entries` |
| GET/POST | `/v1/retailer/shifts` |
| POST | `/v1/retailer/shifts/{shiftID}/close` |

Permissions: `shift.open` (clock + open/list/close as opener), `shift.close` (manager close).

## Cash recon (shift close)

```
expected = opening_float_minor + sum(CASH tenders on linked POS session COMPLETED sales)
variance = closing_cash_minor - expected
```

POS open links the open shift for that register (`linked_pos_session_id`).

## Events

- `RETAILER_CLOCK_IN` / `RETAILER_CLOCK_OUT`
- `RETAILER_SHIFT_OPENED` / `RETAILER_SHIFT_CLOSED`
- `RETAILER_SHIFT_CASH_VARIANCE` (inbox + outbox; also used for large POS session variance)

## Schema

`schema/migrations/20260802_retail_os_phase5_shifts.ddl`  
Tables: `RetailerTimeEntries`, `RetailerShifts` (also in `schema/spanner.ddl`).

## Clients

- Desktop `/shifts`
- Android / iOS Profile → **Shifts**
