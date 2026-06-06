# Dispute Classification Vocabulary (B04)

## Purpose
Define canonical payment and dispute terms for support, finance, and engineering handoffs in pegasusX.

## Canonical terms

| Term | Definition | Primary signal | First owner |
|---|---|---|---|
| `PAYMENT_REQUIRED` | payment is needed before fulfillment can proceed | payment state/event indicates required payment | finance support |
| `PAYMENT_CLEARED` | payment condition for fulfillment has been satisfied | payment state/event indicates cleared status | finance support |
| `SETTLEMENT_REQUIRED` | delivery/payment handoff needs explicit settlement action | settlement-required event/state | treasury operations |
| `DELIVERY_DISPUTED` | delivery/payment outcome is contested and requires review | disputed event/state in delivery-payment flow | finance support |
| `chargeback_recorded` | chargeback mutation has been accepted and persisted | `POST /v1/payment/chargeback` success | finance support |
| `reversal_recorded` | chargeback reversal mutation has been accepted and persisted | `POST /v1/payment/chargeback/reversal` success | finance support |
| `settlement_mismatch` | net mismatch detected between signed grouped settlement contributions | `GET /v1/payment/reconciliation/mismatches` returns non-zero net row | finance support + treasury |
| `webhook_replay` | repeated gateway callback for already-processed transaction | idempotency replay path hit on webhook key | backend payment platform |
| `idempotency_conflict` | same idempotency key reused with different payload | `409 idempotency_key_payload_mismatch` | finance support |
| `policy_violation` | requested payment path violates configured gateway/policy constraints | `422 payment_gateway_policy_violation` | finance support |
| `gateway_execution_failure` | provider execution failed after bounded retries | `502 payment_gateway_execution_failed` | backend payment platform |

## Classification rules
1. Use one primary term per ticket for routing.
2. Add one optional secondary term when both settlement and replay signals appear.
3. If no canonical term applies, classify as `unmapped_payment_exception` and escalate with evidence.

## Term usage guardrails
1. Do not label an issue as `settlement_mismatch` without mismatch endpoint evidence or ledger-confirmed sign imbalance.
2. Do not label an issue as `webhook_replay` when the signature check failed (`401 invalid_signature`); that is an unauthenticated callback.
3. Do not mark `PAYMENT_CLEARED` until settlement/ledger reads show coherent supplier-scoped evidence.

## Evidence minimum for dispute-classified tickets
1. Supplier scope and identifiers (supplier_id, order_id or session_id).
2. Endpoint path, status code, and error code.
3. Gateway reference (`transaction_id`, Adyen `pspReference`, Stripe event id).
4. UTC timestamp and recurrence pattern.
