# Wave B1 — Money integrity implementation
> **POINT-IN-TIME SNAPSHOT (2026-08-12) — do not treat as current status.**
> Re-verify any claim against live code before acting. Multiple ecosystem hardening phases have shipped since this audit.


**Date:** 2026-08-12  
**Master:** [`BACKEND_PARITY_MASTER.md`](./BACKEND_PARITY_MASTER.md)

## Changes

### M-P0-1 Stable payment leg keys
- `CollectCash` leg key: `cash-{orderID}` (was `cash-{orderID}-{newID}`)
- Credit-delivery path: `credit-leave-{orderID}` (was `credit-leave-{orderID}-{newID}`)
- Credit-leave path already stable

### M-P0-2 Fail-closed silent driver mutators
- `PATCH /v1/orders/{id}/state` → **501** with real edge route list
- Stub `collect-cash` / `arrive` without OrderService → **503**
- `POST /v1/delivery/update-order-during-delivery` → **501 not_implemented** (no durable write)

### M-P0-5 Cash checkout honesty
- New `order.SelectCashAtDelivery`: ARRIVED|AWAITING_PAYMENT → PENDING_CASH_COLLECTION + outbox
- `payment.HandleOrderCashCheckout` calls selector (wired in bootstrap)
- Retailer mobile_compat cash stub → 409 `use_confirm_cash`

### M-P0-14 Payout tenant
- `HandleGenerate` ignores body `supplier_id` for auth; uses `PreferTenantSupplierID` / claims

### M-P0-13 Claim under review emit
- New event `CLAIM_UNDER_REVIEW`
- Approve intermediate transition emits dual-topic outbox

### M-P1-1 Dispatcher coverage
- AR_INVOICE_* and PAYOUT_BATCH_* cases + formatters + inbox

## Verification

```bash
cd apps/backend-go
go test ./order/ ./payment/ ./payout/ ./claims/ ./kafka/ ./notifications/ ./driver/ ./bootstrap/ -count=1
```

All green as of 2026-08-12.
