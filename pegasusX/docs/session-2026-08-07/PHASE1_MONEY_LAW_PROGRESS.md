# Enterprise Phase 1 — Money and law

**Date:** 2026-08-11  
**Plan source:** `docs/session-2026-08-07/ENTERPRISE_GRADE_EXECUTION_PLAN.md` §Phase 1  
**Gate:** `make phase1-gate` → **`phase1-gate-ok`** (2026-08-11, Spanner emulator `localhost:9010`)  
**Status:** Wired (backend / simulator); owner credentials remain residuals

## Proof

| Step | Result |
|------|--------|
| [1/7] Phase-0 money-path regression | `money-path-gate-ok` |
| [2/7] AR shop-closed activation (`TestMoneyPathGate_ShopClosed*`) | PASS |
| [3/7] Refunds (`TestRefund_*`) | PASS |
| [4/7] Payouts (`./payout/`) | PASS |
| [5/7] Billing fee schedule + monthly AR invoice | PASS |
| [6/7] Soliq recorded-contract (`TestMySoliqContract` / `TestSoliqSigner`) | PASS |
| [7/7] Global Pay simulator refund + capture | PASS |

## Wired (in-tree; proved by gate)

| Area | Status | Evidence |
|------|--------|----------|
| AR fail-closed + shop-closed invoice | Wired | `ar.ErrInvoicesDisabled`; money-path shop-closed tests |
| Off-app SMS/email dunning | Wired | `ar/dunning_channels.go` — Twilio / PlayMobile / SendGrid via `DUNNING_SMS_PROVIDER` / `DUNNING_EMAIL_PROVIDER` |
| Refunds (full/partial card path) | Wired | `order/refunds.go`, `payment.RefundCardPayment`, `TestRefund_*` |
| Payouts (bank-file export) | Wired | `payout/`, gate `./payout/` |
| Billing fee schedule + monthly AR | Wired | `internal/services/billing/` |
| Soliq EDS signer + MY_SOLIQ contract | Wired | `fiscal/signer*.go`, `MySoliqProvider.SetSigner`, order Soliq contract tests |
| GP simulator refund/capture | Wired | `payment` simulator happy-path tests |

## Owner residuals (not blocked in-repo)

- Soliq sandbox / prod credentials + flip `FISCAL_PROVIDER=MY_SOLIQ` for legal OFD SUCCESS
- Global Pay merchant password (live capture/refund against real GP)
- Twilio / SendGrid / PlayMobile production keys (enable off-app dunning in staging/prod)
- Firebase SMS / APNs / SHA-1 ops for OTP/push
- Live staging AR soak; unskip `PX_E2E_COLLECTIONS_DUNNING_OK` only with env flags + stack
- WhatsApp dunning transport (still absent; SMS + email covered)

## Explicitly out of this phase

- Optimizer cloud deploy / shadow place flip
- Client i18n/a11y 10/10
- Marketplace §8.10 Phase 4–5

## Next deep-dive candidates (pick one)

1. ~~**Enterprise Phase 2–5**~~ → **Wired** (see session progress docs)
2. **Analytics column tenancy** — **LOCKED next** (fork pick default 2026-08-11)
3. **Enterprise Phase 6** — marketplace / cert (decision-gated; deferred)
4. **Client 10/10 residuals** — deferred
