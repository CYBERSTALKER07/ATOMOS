# Credit Engine (Compliance-Facing)

> **PLANNING BASELINE** — not living runtime status. Prefer [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md) and code for what is shipped.


## Existing

- Profiles: limit, balance, available, risk, delinquency, status  
- Utilization bps = balance * 10000 / limit  
- Freeze blocks credit-funded orders  

## Upgrade

- Real-time utilization + external signals → risk tier  
- Dynamic limit adjust with freeze:  
  - **Blocks new credit orders**  
  - **Allows cash/card**  
- Early-pay discounts; later invoice discounting marketplace  

## Regulatory / risk edges

| Case | Rule |
|------|------|
| Over-limit attempt | Reject credit path with clear code |
| Concurrent checkouts | CAS on profile version |
| Collections | Supplier list/freeze desk already planned |
