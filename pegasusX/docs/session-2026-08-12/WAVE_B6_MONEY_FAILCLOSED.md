# Wave B6 — Money fail-closed
> **POINT-IN-TIME SNAPSHOT (2026-08-12) — do not treat as current status.**
> Re-verify any claim against live code before acting. Multiple ecosystem hardening phases have shipped since this audit.


**Date:** 2026-08-13  
**Master:** [`BACKEND_PARITY_MASTER.md`](./BACKEND_PARITY_MASTER.md)

## Changes

### SPINE-P0-4 AR open at credit leave
- `OpenFromCreditLeaveInTxn` opens AR invoice + ledger OPEN + outbox **on the same Spanner RW txn** as order credit leave
- Wired into `HandleCreditLeave` and `HandleCreditDelivery` InTxn paths
- Disabled AR + positive amount → `ErrInvoicesDisabled` (fail-closed)
- Shop-closed worker: post-commit open now **returns error** so sweeper retries (idempotent)

### S-P0-4 / claim approve
- HTTP **Idempotency-Key** on approve/reject handlers (domain chargeback id already stable)
- UNDER_REVIEW + RESOLVED dual-topic outbox retained from B1

### M-P1-6 AR aging outbox
- Event `AR_INVOICE_AGING_UPDATED`
- `RecomputeAging` writes bucket change + outbox per changed invoice (skips no-ops)
- Dispatcher case → `handleARInvoiceEvent` (supplier + retailer rooms)

### Credit-leave event parity
- `HandleCreditLeave` emits `ORDER_STATUS_CHANGED` **and** `CREDIT_LEAVE` (match delivery path)

### Refund / buyer-acceptance bus
- Dispatcher cases for `REFUND_*` and `BUYER_ACCEPTANCE_*` → supplier + retailer hubs

## Residual
- Cash collect AR pay-down still fail-open after CAPTURE (payment already durable; log + retry ops)
- Claim gateway settlement remains outside Spanner (external money); domain idempotency + HTTP idem key mitigate double settle

## Verification

```bash
cd apps/backend-go
go test ./ar/ ./order/ ./claims/ ./kafka/ -count=1
go build -o /tmp/pegasusx-backend .
```
