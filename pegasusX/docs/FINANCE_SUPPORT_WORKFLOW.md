# Finance Support Workflow (B04)

## Purpose
Standardize finance-support handling for payment, settlement, and reconciliation incidents in pegasusX.

## Workflow stages
1. Intake and classification.
2. Source-of-truth verification.
3. Exception path execution.
4. Resolution confirmation.
5. Closure and escalation handoff.

## Stage 1: Intake and classification
1. Capture identifiers: supplier_id, order_id, session_id, gateway, currency, event timestamp.
2. Capture user symptom in one sentence.
3. Assign classification using `DISPUTE_CLASSIFICATION_VOCABULARY.md`.

## Stage 2: Source-of-truth verification
Run reads in this order:
1. `GET /v1/payment/settlement/authority` for grouped authority view.
2. `GET /v1/payment/reconciliation/mismatches` for net mismatch and thresholded groups.
3. `GET /v1/payment/ledger` for immutable row-level fallback evidence.

Rules:
1. Settlement authority is preferred read model.
2. Ledger is immutable fallback for group-level disagreement.
3. If mismatch endpoint is unavailable, continue with settlement + ledger and escalate query failure.

## Stage 3: Exception path execution
1. For chargeback disputes:
   - `POST /v1/payment/chargeback` with stable idempotency key.
2. For chargeback reversal decisions:
   - `POST /v1/payment/chargeback/reversal` with stable idempotency key.
3. For webhook replay incidents:
   - do not manually replay provider callbacks from support tools.
   - escalate with gateway identifiers and signature evidence.

## Stage 4: Resolution confirmation
1. Verify business outcome reflects exactly one durable state transition.
2. Re-read settlement authority and mismatch views to confirm net movement is expected.
3. Confirm mismatch trend is stable or improving for the affected supplier scope.

## Stage 5: Closure and handoff
1. Record final classification and resolution action.
2. Attach evidence package (request/response IDs, timestamps, gateway refs).
3. Mark ticket status:
   - resolved in support
   - escalated to backend payment platform
   - escalated to treasury policy owner

## Priority model
| Priority | Trigger | Target owner |
|---|---|---|
| P1 | duplicate durable outcomes, replay corruption, signature anomalies at scale | backend payment platform + on-call |
| P2 | persistent settlement mismatch beyond threshold | finance support + treasury operations |
| P3 | single-ticket policy misuse or payload validation issue | finance support |

## Communication standards
1. Use source-of-truth language: pending, cleared, settlement-required, disputed, reversed.
2. Avoid speculative root causes before source verification is complete.
3. Include endpoint and error code in every support-to-engineering handoff.
