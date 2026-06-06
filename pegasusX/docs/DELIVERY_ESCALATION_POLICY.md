# Delivery escalation policy

Defines when a delivery issue moves beyond first-line driver support and into engineering, treasury, or supplier ownership. Pairs with `DRIVER_SUPPORT_PLAYBOOK.md` (first-line triage) and `LIVE_TRACKING_EXPECTATIONS.md` (visibility contract).

## Escalation triggers

Escalate to the named owner when any of the following holds:

| Trigger | Owner | Reason |
| --- | --- | --- |
| Geofence rejection persists after driver re-arrival inside expected radius | Engineering | Retailer location drift or backend geofence misconfiguration |
| Telemetry stall persists after device network is verified healthy | Engineering | Backend telemetry pipeline or hub fanout issue |
| Cash collected on driver side but order is not reaching `COMPLETED` | Treasury and engineering | Payment handshake or ledger reconciliation issue |
| Manifest reassignment dispute between driver and dispatcher | Supplier dispatcher | Operational decision, not a technical fault |
| Repeated reconnect loops on a healthy device | Engineering | WebSocket hub or auth token issue |
| Driver reports order data inconsistent with retailer expectation | Supplier support | Order capture or pricing rule issue |
| Suspected role-spoofing or credential leakage | Security on-call | Auth boundary review required |

## Escalation method

1. Log the issue with: driver ID, supplier ID, manifest ID (if applicable), order ID (if applicable), trace ID (if observable), and a one-paragraph summary of the symptom and what first-line support already tried.
2. Open the escalation in the owner's queue; do not route through ad hoc messaging.
3. Tag with severity:
   - `S1` if the delivery is blocked and the retailer is waiting.
   - `S2` if the delivery is delayed but expected to recover within the SLA window.
   - `S3` if the issue is observed but does not block the delivery.

## Engineering response expectations

- `S1`: acknowledge within minutes, mitigate within the SLA hour.
- `S2`: acknowledge same business day, mitigate within the SLA day.
- `S3`: triaged in the next planning window, no immediate mitigation required.

## Treasury response expectations

- Any cash, settlement, or ledger reconciliation escalation should be picked up the same business day. The corresponding payment dispute classification (see `DISPUTE_CLASSIFICATION_VOCABULARY.md`) should be applied before further action.

## Closure

An escalation is closed only when:

1. The root cause is recorded.
2. The mitigation is verified.
3. Any required follow-up is tracked (for example, a code fix, a config change, or a retailer location correction).
4. The driver and any other affected role are informed of the outcome.

## Cross-references

- `DRIVER_SUPPORT_PLAYBOOK.md` for first-line driver-side triage.
- `LIVE_TRACKING_EXPECTATIONS.md` for the operator and retailer visibility contract.
- `PAYMENT_EXCEPTION_SOP.md` and `DISPUTE_CLASSIFICATION_VOCABULARY.md` for payment-affected escalations.
