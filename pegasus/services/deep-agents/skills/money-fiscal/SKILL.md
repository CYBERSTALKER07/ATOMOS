---
name: money-fiscal
description: Order-to-cash, AR, payout, fiscal OFD, buyer EHF acceptance, payment webhooks — money integrity and legality gates.
---

# Money & fiscal

## Hard rules

- Money: int64 minor units; double-entry where ledger applies
- Capture/refund idempotency keys required
- ADR-009: `FISCALIZING` → OFD SUCCESS → `COMPLETED` / `ORDER_FINALIZED` (or `FISCAL_FAILED` / force)
- Payout live rail fail-closed without a live rail (`payout.ErrNoLiveRail`, P0-2)

## Wired (verify before re-opening gaps)

| Area | Evidence |
|------|----------|
| AR pay-down on cash collect | `ar.RecordPaymentForOrder` from `order.CollectCash` (P0-1) |
| AR outbox | `OpenInvoice` / `ApplyPayment` / `UpdateDunning` via `outbox.SpannerTxnBuffer` (P0-5) |
| Payout outbox | `Insert` / `UpdateStatusRef` → `PAYOUT_BATCH_*` (P0-4) |
| Webhook reconciler | `App.WebhookReconciler` + worker ticker (P1-8); `PAYMENT_RECONCILE_DISABLED=1` to stop |
| Buyer EHF track | MySoliq SUCCESS stamps `BuyerAcceptanceStatus=PENDING` + deadline; poller emits `BUYER_ACCEPTANCE_*`; auto credit-note on REJECT **default ON** (`CREDIT_NOTE_AUTO_FROM_BUYER_REJECT=false` to opt out) (P1-6) |

## Still Class D / open

- Legal Soliq OFD without EDS/E-IMZO proof (P1-7)
- Live bank payout rail integration (beyond fail-closed file rail)
- Multi-currency AR (P2)

## Security

- Detail IDOR review on payment/driver/warehouse get-by-id
- Dual-control on money feature flags (admin portal approve path)

## Refs

`order/fiscal.go`, `order/buyer_acceptance_poller.go`, `payment/reconciliation.go`, `ar/`, `payout/`, gap register money rows
