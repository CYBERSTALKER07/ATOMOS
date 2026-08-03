# Claim Settlement + Insurance Trigger

## Existing settlement modes

`LEDGER_ONLY | STORE_CREDIT | GATEWAY_REFUND`  
Cash/COD: INTERNAL/CASH ledger clawback (no PSP).

## Extension

When `claim.amount_minor ≥ INSURANCE_THRESHOLD_MINOR` **and** photo evidence exists:

- Emit `INSURANCE_CLAIM_REQUIRED` (or partner webhook)  
- Do not block supplier ledger settle unless policy says so  

## Non-negotiable

`AggregateClaimLines` + order-line prices only — no retailer price fraud path.
