---
name: data-flow-coverage
description: Enforce Spanner→outbox→Kafka→WS coverage rule and Class A/B/C/D classification for any feature.
---

# Data-flow coverage

## Coverage rule

Every Spanner state mutation must:

1. Emit an outbox event in the **same RW transaction** (`outbox.SpannerTxnBuffer` / `EmitJSON`).
2. Have a **declared consumer** (NotificationDispatcher fanout, domain mutator, webhook, or explicit “no fanout” ADR).
3. Reach **clients** that need to react (WS hub envelope + inbox/push as appropriate).

## Classification

| Class | Meaning | Action |
|-------|---------|--------|
| A | Full E2E wired | Ship |
| B | Backend island (API/schema, weak client/consumer) | Wire consumer + clients |
| C | UI island (screen without real hop) | Ban for prod; wire backend |
| D | Flag / cert / external key blocked | Document gate; do not claim live |

## Audit steps

1. Find mutation entrypoint (`service.go`, routes).
2. Search for `outbox.Emit` / `EmitJSON` in same txn path.
3. Map event type → `kafka/` consumer or dispatcher switch.
4. Map WS hub + `packages/types` envelope.
5. Check role apps consume the event or poll with documented reason.
6. Cite gap register if already tracked (`docs/session-2026-08-07/ECOSYSTEM_GAP_REGISTER_*.md`).

## Kernel references

- `docs/DATA_FLOW_AS_IMPLEMENTED.md`
- `docs/session-2026-08-07/MASTER_ALIGNMENT_DATAFLOW_*.md`
- `apps/backend-go/outbox/`, `kafka/`, `events/`, `ws/`
