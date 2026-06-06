# Pricing Authority Rules (B03)

## Purpose
Define who owns pricing decisions and how support should respond while pegasusX pricing APIs evolve.

## Current runtime authority
1. Pricing authority is backend-owned by supplier-side policy surfaces.
2. Retailer checkout/order entry (`POST /v1/order/create`) accepts line items and prices, but rule enforcement beyond request validation is still an active B03 technical track.
3. No dedicated `/v1/supplier/pricing/*` authority endpoints are mounted in pegasusX at this stage.

## Support-facing policy
1. Treat supplier pricing as canonical business authority.
2. Retailer-facing price disputes should be logged as policy disputes unless a clear backend contract defect is proven.
3. Do not ask clients to hardcode pricing behavior as a workaround.
4. If observed behavior conflicts with agreed supplier policy, escalate with evidence instead of advising manual frontend-only overrides.

## Operational handling categories

| Category | Definition | Initial owner | Escalation owner |
|---|---|---|---|
| Policy clarification | Retailer questions expected price/tier behavior | Retailer support | Supplier operations |
| Suspected backend pricing defect | Reproducible mismatch between requested and persisted/returned pricing state | Retailer support | Backend platform |
| Data-entry issue | Incorrect line item or quantity sent by client | Retailer support | Retailer operations |
| Scope mismatch | Retailer acting outside seeded supplier scope assumptions | Retailer support | Product + backend |

## Required evidence for pricing defect escalation
1. Retailer id and supplier id.
2. Full request payload (line items, quantities, unit prices, currency).
3. Response body and status.
4. Timestamp (UTC).
5. Expected policy behavior documented by supplier operations.

## Interim guardrails until dedicated pricing endpoints land
1. Keep all pricing-dispute outcomes documented in support tickets.
2. Link disputes to B03 anchor `PX2-A3` for implementation tracking.
3. Avoid undocumented side agreements that bypass backend authority.
