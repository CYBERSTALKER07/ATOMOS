# State Machines

> **PLANNING BASELINE** — not living runtime status. Prefer [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md) and code for what is shipped.


## Order (canonical — already implemented)

See `order/state_machine.go` and `ECOSYSTEM_FEATURES_BY_ROLE.md`.

Critical: **no ARRIVED → COMPLETED**; fiscal gate required.

## Fiscal receipt attempt

```
PENDING → SUCCESS | FAILED
FAILED → (retry) PENDING → …
```

Stored in `OrderFiscalReceipts`; worker applies via `ApplyFiscalWorkerResult`.

## Shop-closed (target)

```
ARRIVED → SHOP_CLOSED_PENDING → RESOLVED_*
```

Must not start claim window until COMPLETED.

## Claim

```
OPEN → UNDER_REVIEW → APPROVED | REJECTED | RESOLVED
```

Optional auto-approve OPEN → RESOLVED under threshold.

## Returns physical

```
OPEN → ARRIVED → RECEIVING → RESTOCK | WRITE_OFF | QUARANTINE | …
```

## WMS task (target)

```
CREATED → ASSIGNED → IN_PROGRESS → DONE | CANCELLED
```

Capacity reserved while ASSIGNED/IN_PROGRESS.

## Cart session (target)

```
OPEN → CHECKED_OUT → SETTLING → CLOSED | PARTIAL_FAILED
```
