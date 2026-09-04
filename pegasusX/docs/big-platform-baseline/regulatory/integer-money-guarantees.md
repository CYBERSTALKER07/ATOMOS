# Integer Money Everywhere + Zero-Leak Guarantees

> **PLANNING BASELINE** — not living runtime status. Prefer [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md) and code for what is shipped.


## Law

All money fields are **int64 minor units**. No float for:

- Prices, carts, checkout, ledger, chargebacks, claims  
- Planning costs, freight settlement, carbon cost proxies if monetized  
- Credit limits/balances, splits, shortfalls  

## Guarantees

| Guarantee | Mechanism |
|-----------|-----------|
| No silent float drift | Types + lint + tests |
| Cap chargeback ≤ paid | Session amount cap |
| Split remainder | Last leg absorbs remainder minor |
| Claim ≤ order residual | CapAmount + cumulative claims |
| Idempotent settle | Deterministic chargeback id per claim |

## Tests required for every money PR

- Overflow-safe adds  
- Negative amount reject  
- Currency consistency  
