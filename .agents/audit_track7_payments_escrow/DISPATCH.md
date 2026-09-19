## 2026-08-30T00:18:54Z

You are a Codebase Explorer auditing Track 7 of the PegasusX Go backend: Payments, PSP, Escrow, Invoicing & Financial Integrity.

Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/audit_track7_payments_escrow
Original request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md
Target codebase: apps/backend-go (and pegasusX/apps/backend-go), specifically payment processing, PSP integrations (Stripe, local PSPs, Payme, Click, Uzum, etc.), escrow hold & release, driver payouts, supplier settlements, commission calculation, invoicing, tax, and double-entry ledger bookkeeping.

Your Mission:
Conduct a comprehensive, line-by-line code review of Payments, Escrow, Invoicing, and Financial services.
1. Inspect idempotency of payment webhooks and payment initialization, escrow lock and release state machines, partial refunds, chargebacks, currency conversion, fee splits, supplier payout calculation, and ledger reconciliation.
2. Audit Spanner transactions: are ledger entries immutable? Are debit and credit rows balanced in the exact same transaction? Is there risk of double-spend, floating point rounding errors, or unhandled gateway timeout states?
3. Check outbox event emissions for payment events, WebSocket notifications for receipt confirmation, and security of payment tokens and sensitive credentials.
4. Document every single finding with EXACT file path and line number(s) (`file:line`), explanation of the flaw, blast radius across the ecosystem, and recommendation.
5. Formulate deep architectural / edge-case open questions regarding unhandled scenarios or state inconsistencies.
6. Write your comprehensive findings into `/Users/shakhzod/Desktop/V.O.I.D/.agents/audit_track7_payments_escrow/findings.md` and send a completion message to the caller with a summary of findings.
