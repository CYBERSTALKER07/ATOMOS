# Compliance & Fiscal Audit Dashboard

## Queryable / exportable sets

| Bucket | Examples |
|--------|----------|
| Open fiscal | Orders in FISCALIZING / FISCAL_FAILED |
| Missing EHF | COMPLETED without Soliq clearance (when enabled) |
| Claim mismatches | Claim total vs order residual |
| Credit freezes | Active FROZEN profiles |
| Force-completes | ORDER_FORCE_COMPLETED with actor |
| Concurrency conflicts | High retry rates / 409 storms |
| Inventory integrity | Reserved vs open reservations |

## Consumers

- Internal compliance  
- Soliq audit export (CSV/JSON)  
- Supplier finance desk  

## Phase 1 DoD

Dashboard + export for open fiscal, force-completes, claim mismatches, credit freezes.
