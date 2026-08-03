# Edge-Case Matrix Template

Every feature PR fills this table (copy into PR description).

| Category | Cases to cover | Pass criteria |
|----------|----------------|---------------|
| Null / empty | Missing optional fields, empty lists | 4xx with stable codes |
| Cancel / terminal | Cancel mid-feature, claim after cancel | Inventory released; no double money |
| Offline | Driver offline actions | Hash reconcile; no double apply |
| Concurrent | Two actors same order | CAS / 409 / idempotent replay |
| Overflow | Capacity, credit, claim qty | Reject or cap; never wrap int64 |
| Fiscal fail | Provider error | FISCAL_FAILED; retry/force audited |
| Claim window | After 48h | Reject file |
| IDOR | Other supplier/retailer | 403/empty |
| Idempotency | Replay same key | Same response body/status |
| Partition | Redis/Kafka lag | Spanner freezes still enforce |

## Feature-specific rows

Add rows per feature (shop-closed, MEIO publish, cart session, EHF submit, …).
