# Gate 5 / §8.10 Phase 3 — GlobalProducts Progress

> **HISTORICAL / FROZEN — session progress note; do not treat as current gap SoT.**
> Living residuals: [`../PROD_READINESS_SEQUENCE.md`](../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](./ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md).


**Date:** 2026-08-11  
**ADR:** [`docs/MULTI_TENANCY_GATE5_PHASE3.md`](../../MULTI_TENANCY_GATE5_PHASE3.md)  
**Gate:** `bash scripts/phase5c_gate.sh` → `phase5c-gate-ok`  
**Focused smoke:** `go run ./cmd/ssmr-smokecheck global-products` (requires Spanner + `GLOBAL_PRODUCTS_ENABLED=true`)

## Shipped (backend slice)

| Area | Status |
|------|--------|
| ADR + `GLOBAL_PRODUCTS_ENABLED` (SSMR on / prod default off) | Done |
| DDL: UnitsOfMeasure, GlobalProducts, SupplierProductOffers, ProductMatchQueue | Done |
| `globalproducts` package: match (exact GTIN → fuzzy → create), offers, queue resolve | Done |
| Catalog hook on product create/update | Done |
| Routes: GET global + offers, supplier link-global, admin match-queue | Done |
| Match worker (queue depth logging) | Done |
| Unit tests + markers + `ssmr-smokecheck global-products` | Done |
| `make phase5c-gate` / `make apply-global-products-ddl` | Done |

## Markers

| Marker | Meaning |
|--------|---------|
| `PX_E2E_GLOBAL_PRODUCT_GTIN_LINK_OK` / `_SKIPPED` | Two SKUs same GTIN → one GlobalProduct |
| `PX_E2E_GLOBAL_PRODUCT_FUZZY_QUEUE_OK` / `_SKIPPED` | Ambiguous fuzzy → ProductMatchQueue |
| `PX_E2E_GLOBAL_PRODUCT_OFFERS_COMPARE_OK` / `_SKIPPED` | Offers list ≥2 suppliers |

## Residuals

- Marketplace commerce (fees / RFQ / escrow) — §8.10 Phase 4
- Platform KYB admin console — §8.10 Phase 5
- Retailer UI multi-partner browse by GlobalProductId
- Checkout still uses supplier SKUs (by design)

## Next forks

1. **§8.10 Phase 4** — marketplace commerce (decision-gated on economics)
2. **§8.10 Phase 5** — tenant ops / KYB console
3. Live prove `ssmr-smokecheck global-products` on SSMR after `make apply-global-products-ddl`
