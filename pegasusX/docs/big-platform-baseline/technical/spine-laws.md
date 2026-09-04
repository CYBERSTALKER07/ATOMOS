# Spine Laws (Non-Negotiable)

> **PLANNING BASELINE** — not living runtime status. Prefer [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md) and code for what is shipped.


## Mutation shape

```
1. AuthZ + idempotency guard
2. Spanner ReadWriteTransaction:
     - read authoritative rows
     - validate state machine / capacity / credit
     - write domain rows
     - write OutboxEvents (same txn)
3. Commit
4. Post-commit: cache invalidate, metrics
5. Worker: outbox → Kafka → consumers
6. WS / FCM fanout for live clients
```

## Money

- int64 minor only  
- Caps, utilization, splits, claims: tested overflow-safe  

## Status

- Only `order.ValidateStatusTransition`  
- No soft ARRIVED → COMPLETED  

## Idempotency

- Header keys on all POSTs that mutate money, inventory, fiscal, claims, dispatch  

## Role-row

- Feature lands on all clients of the role unless deferred in parity-ledger  
