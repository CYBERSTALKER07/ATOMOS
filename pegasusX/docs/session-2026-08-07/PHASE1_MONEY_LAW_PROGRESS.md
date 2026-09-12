# Enterprise Phase 1 — Money and law

> **HISTORICAL / FROZEN — session progress note; do not treat as current gap SoT.**
> Living residuals: [`../PROD_READINESS_SEQUENCE.md`](../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](./ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md).


**Date:** 2026-08-11 (clarified 2026-08-12)  
**Plan source:** `docs/session-2026-08-07/ENTERPRISE_GRADE_EXECUTION_PLAN.md` §Phase 1  
**Gate:** `make phase1-gate` → **`phase1-gate-ok`** (2026-08-11, Spanner emulator `localhost:9010`)  
**Status:** **Code-wired + simulator/CI proved.** Live partner rails and production keys remain owner residuals — do **not** read “Wired” as “live in staging/prod.”

## Proof

| Step | Result |
|------|--------|
| [1/7] Phase-0 money-path regression | `money-path-gate-ok` |
| [2/7] AR shop-closed activation (`TestMoneyPathGate_ShopClosed*`) | PASS |
| [3/7] Refunds (`TestRefund_*`) | PASS |
| [4/7] Payouts (`./payout/`) | PASS |
| [5/7] Billing fee schedule + monthly AR invoice | PASS |
| [6/7] Soliq recorded-contract (`TestMySoliqContract` / `TestSoliqSigner`) | PASS (httptest + DevHMAC — **not** legal OFD) |
| [7/7] Global Pay simulator refund + capture | PASS (in-repo simulator — **not** live merchant) |

## Wired (in-tree; proved by gate / unit tests)

| Area | Status | Evidence | Live meaning |
|------|--------|----------|--------------|
| AR fail-closed + shop-closed invoice | Code-wired | `ar.ErrInvoicesDisabled`; money-path shop-closed tests | Needs `AR_INVOICES_ENABLED` in env |
| Off-app SMS/email/WhatsApp dunning | Code-wired | `ar/dunning_channels.go` — Twilio / PlayMobile / SendGrid / Twilio WhatsApp via `DUNNING_SMS_PROVIDER` / `DUNNING_EMAIL_PROVIDER` / `DUNNING_WHATSAPP_PROVIDER` | **Default = no-op** until providers + keys set. Retailer contacts are **phone-only** (email channel applies to supplier staff). WhatsApp needs Content SID + approved template |
| Refunds (full/partial card path) | Code-wired | `order/refunds.go`, `payment.RefundCardPayment`, `TestRefund_*` | Live GP needs merchant password |
| Payouts (bank-file export) | Code-wired | `payout/`, gate `./payout/`; fail-closed if `live=true` without real rail | Bank-file is permanent prod bar ([`PAYOUT_RAIL_DECISION.md`](../PAYOUT_RAIL_DECISION.md)) |
| Billing fee schedule + monthly AR | Code-wired | `internal/services/billing/` | Ops schedule enablement |
| Soliq EDS signer + MY_SOLIQ contract | Code-wired (contract) | `fiscal/signer*.go`, `MySoliqProvider.SetSigner`, order Soliq contract tests | Default `FISCAL_PROVIDER` is not MY_SOLIQ; legal OFD needs E-IMZO PKCS#12 + sandbox/prod flip — [`FISCAL_EDS_PROOF.md`](../FISCAL_EDS_PROOF.md) |
| GP simulator refund/capture | Simulator-wired | `payment` simulator happy-path tests | Live merchant residual — [`GLOBAL_PAY_REFUND_PROOF.md`](../GLOBAL_PAY_REFUND_PROOF.md) |

### Reading guide

- **Code-wired** = transports/handlers/fail-closed construction exist and pass in-repo gates.
- **Simulator/CI proved** = httptest or in-repo partner simulator; not a live SUCCESS against Soliq/GP.
- **Owner residual** = credentials, templates, or flips that only ops can supply.

## Owner residuals (not blocked in-repo)

- Soliq sandbox / prod credentials + flip `FISCAL_PROVIDER=MY_SOLIQ` for legal OFD SUCCESS
- Global Pay merchant password (live capture/refund against real GP)
- Twilio / SendGrid / PlayMobile / Twilio WhatsApp production keys + approved WhatsApp Content template (enable off-app dunning in staging/prod)
- Firebase SMS / APNs / SHA-1 ops for OTP/push
- Live staging AR soak; unskip `PX_E2E_COLLECTIONS_DUNNING_OK` only with env flags + stack

## Explicitly out of this phase

- Optimizer cloud deploy / shadow place flip
- Client i18n/a11y 10/10
- Marketplace §8.10 Phase 4–5

## Next deep-dive candidates (pick one)

1. ~~**Enterprise Phase 2–5**~~ → **Wired** (see session progress docs)
2. **Analytics column tenancy** — **LOCKED next** (fork pick default 2026-08-11)
3. **Enterprise Phase 6** — marketplace / cert (decision-gated; deferred)
4. **Client 10/10 residuals** — deferred
