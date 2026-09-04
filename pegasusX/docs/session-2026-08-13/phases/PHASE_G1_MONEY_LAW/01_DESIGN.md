# 01 — Design — G1.A (implemented)

## Goal

Cash payment leg + AR invoice pay-down in the **same Spanner RW commit** as CollectCash → FISCALIZING.

## Implementation

| Piece | Location |
|-------|----------|
| `applyPaymentInTxn` | `ar/service.go` — ledger PAYMENT + balance + outbox; idempotency by `IdempotencyKey` query |
| `getInvoiceByOrderInTxn` | `ar/service.go` |
| `RecordPaymentForOrderInTxn` | `ar/service.go` — Spanner path uses caller txn; memory falls back to sequential |
| CollectCash InTxn | `order/service.go` — after `RecordPaymentLeg`; fail-closed if AR fails; ar nil + DELIVERED_ON_CREDIT + invoices on → error |
| Post-commit fail-open block | **removed** |

## Out of scope (still open)

G1-A2 ClearBalance, G1.B fiscal, G1.C theatre, G1.D payout/FCM.
