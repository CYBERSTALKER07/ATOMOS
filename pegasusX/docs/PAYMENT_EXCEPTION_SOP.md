# Payment Exception SOP (B04)

## Purpose
Provide a repeatable process for payment exceptions in pegasusX so support and finance can resolve incidents without ad hoc database interpretation.

## Scope
- Checkout write paths: `POST /v1/checkout/b2b`, `POST /v1/checkout/unified`
- Payment mutation paths: `POST /v1/payment/chargeback`, `POST /v1/payment/chargeback/reversal`
- Finance read authority: `GET /v1/payment/ledger`, `GET /v1/payment/settlement/authority`, `GET /v1/payment/reconciliation/mismatches`
- Gateway callbacks: `POST /v1/webhooks/global-pay`, `POST /v1/webhooks/adyen`, `POST /v1/webhooks/stripe`

## Ownership
1. First response: finance support operations.
2. Escalation: backend payment platform team for persistence, replay/idempotency, or signature verification defects.
3. Advisory stakeholder: treasury operations for settlement and mismatch decisions.

## Exception triage checklist
1. Capture incident metadata: supplier_id, order_id, session_id, gateway, timestamp (UTC), and reported symptom.
2. Verify supplier scope first:
   - `GET /v1/payment/ledger`
   - `GET /v1/payment/settlement/authority`
   - `GET /v1/payment/reconciliation/mismatches`
3. Confirm state-transition evidence:
   - payment state events (`PAYMENT_REQUIRED`, `PAYMENT_CLEARED`, `SETTLEMENT_REQUIRED`, `DELIVERY_DISPUTED`)
   - fiscal hard-gate events (`FISCAL_RECEIPT_*`, `ORDER_FORCE_COMPLETED`, `CASH_SHORTFALL` / `CASH_OVERAGE`) and order status `FISCALIZING` / `FISCAL_FAILED` / `COMPLETED`
   - immutable ledger row presence for the affected session/order; fiscal attempts in `OrderFiscalReceipts`.
4. Check replay/idempotency posture:
   - duplicate webhook should replay one durable outcome.
   - same idempotency key with different payload should produce `409 idempotency_key_payload_mismatch`.
5. Decide path:
   - policy violation -> support correction + retry.
   - gateway execution failure -> upstream retry/escalation.
   - settlement mismatch -> finance workflow + dispute vocabulary classification.

## Failure handling matrix

| Surface | Status + code | Meaning | Support action |
|---|---|---|---|
| Checkout (`/v1/checkout/*`) | `422 payment_gateway_policy_violation` | request conflicts with policy/capability constraints | Correct gateway/policy input and retry |
| Checkout (`/v1/checkout/*`) | `502 payment_gateway_execution_failed` | upstream execution failed after bounded retries | Capture provider reference, retry per workflow, escalate if repeated |
| Checkout or webhook | `409 idempotency_key_payload_mismatch` | same key reused with a different payload | Use a new idempotency key; do not mutate prior payload |
| Chargeback | `500 chargeback_record_failed` | durable mutation write failed | Escalate with evidence package |
| Chargeback reversal | `400 reversal_record_failed` | reversal persistence rejected | Revalidate session_id and payload, escalate if deterministic |
| Settlement authority | `500 settlement_summary_failed` | grouped settlement summary query failed | use ledger fallback and escalate |
| Reconciliation mismatches | `500 reconciliation_summary_failed` | mismatch aggregation source query failed | use settlement/ledger fallback and escalate |
| Webhooks | `401 invalid_signature` | signature verification failed | treat as unauthenticated callback, no mutation retry from support |
| Webhooks | `500 webhook_process_failed` | callback accepted but persistence/outbox path failed | escalate immediately with transaction_id and gateway |

## Immediate escalation triggers
1. A single payment intent yields more than one conflicting durable business outcome.
2. Replayed webhooks create duplicate settlement mutations.
3. `GET /v1/payment/ledger` and `GET /v1/payment/settlement/authority` disagree without a transient fallback condition.
4. Reconciliation mismatch net values increase across repeated reads with no new payment activity.

## Evidence package
1. Supplier_id, retailer_id (if available), order_id, session_id.
2. Endpoint + method and request payload hash/idempotency key.
3. HTTP status + error code/body.
4. Gateway transaction identifiers (`transaction_id`, `pspReference`, or Stripe event id).
5. UTC timestamp and whether the issue is deterministic or intermittent.
