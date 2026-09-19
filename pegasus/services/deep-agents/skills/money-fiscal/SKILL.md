---
name: money-fiscal
description: Order-to-cash, AR, payout, fiscal OFD, buyer EHF, PSP webhooks, gov fiscal legality.
---

# Money & fiscal

Also load skill **`regulatory-gov`** for Soliq/GS1/AS2 checklists.

## Hard rules

- Money: int64 minor units; double-entry where ledger applies
- Capture/refund idempotency keys required
- ADR-009: `FISCALIZING` → OFD SUCCESS → `COMPLETED` / `ORDER_FINALIZED` (or `FISCAL_FAILED` / force)
- Payout live rail fail-closed without a live rail (`payout.ErrNoLiveRail`, P0-2)
- Cash collect on credit leave must pay down AR (P0-1)

## Wired (verify before re-opening gaps)

| Area | Evidence |
|------|----------|
| AR pay-down on cash collect | `ar.RecordPaymentForOrder` from `order.CollectCash` (P0-1) |
| AR outbox | `OpenInvoice` / `ApplyPayment` / `UpdateDunning` via `outbox.SpannerTxnBuffer` (P0-5) |
| Payout outbox | `Insert` / `UpdateStatusRef` → `PAYOUT_BATCH_*` (P0-4) |
| Webhook reconciler | `App.WebhookReconciler` + worker ticker (P1-8); `PAYMENT_RECONCILE_DISABLED=1` to stop |
| Buyer EHF track | MySoliq SUCCESS stamps PENDING + deadline; poller `BUYER_ACCEPTANCE_*`; credit-note on REJECT default ON (P1-6) |

## Still Class D / open

- Legal Soliq OFD without EDS/E-IMZO proof (P1-7) — **gov**
- Live bank payout rail (beyond fail-closed file rail)
- Multi-currency AR (P2); Currency hardcoded UZS
- Global Pay refund `RF` live verification (P2-10)

## Role touchpoints

| Role | Money-critical actions |
|------|------------------------|
| Driver | collect-cash, credit-leave, fiscal retry, cash recon |
| Retailer | confirm-cash, credit profile, AR invoices (parity P1-17) |
| Supplier | treasury, payout batches, claim chargebacks, credit program |
| Warehouse admin | force-complete (fiscal stuck paths) |
| Platform admin | dual-control money flags |

## Security

- Detail IDOR review on payment/driver/warehouse get-by-id
- Dual-control on money feature flags (admin portal approve path)

## Refs

`order/fiscal.go`, `order/buyer_acceptance_poller.go`, `payment/reconciliation.go`,
`ar/`, `payout/`, gap register money rows, skill `regulatory-gov`


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
