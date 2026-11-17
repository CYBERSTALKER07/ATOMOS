# ADR: Gate 5 / §8.10 Phase 3 — GlobalProducts master

**Status:** Accepted — Wired (backend)  
**Date:** 2026-08-11  
**Deciders:** Platform engineering  
**Related:** [MULTI_TENANCY_GATE5_PHASE1.md](./MULTI_TENANCY_GATE5_PHASE1.md); [MULTI_TENANCY_GATE5_PHASE2.md](./MULTI_TENANCY_GATE5_PHASE2.md); [PLATFORM_AUDIT.md](../PLATFORM_AUDIT.md) §8.10 Phase 3; [PHASE5_PHASE3_PROGRESS.md](./session-2026-08-07/PHASE5_PHASE3_PROGRESS.md)  

---

## Context

Phase 1–2 delivered request-scoped tenancy and ParentOrders multi-supplier checkout. Catalog SKUs remain **per-supplier** (`Products.SupplierId` + nullable `Barcode` / `UnitsPerPack`). Cross-supplier comparison and marketplace commerce need a shared product identity keyed by GTIN.

Audit Phase 3: `GlobalProducts` + `SupplierProductOffers` + `ProductMatchQueue` + real UoM hierarchy; exact GTIN then fuzzy match with human review queue.

## Decision

1. Keep operational SKUs on `Products`. Global master is a **link layer**, not a catalog replacement.
2. Introduce `UnitsOfMeasure`, `GlobalProducts` (unique normalized GTIN), `SupplierProductOffers` `(SupplierId, ProductId) → GlobalProductId`, `ProductMatchQueue`.
3. Matching pipeline (when `GLOBAL_PRODUCTS_ENABLED`):
   - Normalize barcode via `gs1.NormalizeGTIN`; invalid → skip.
   - Exact GTIN hit → upsert offer `LINKED`.
   - Else fuzzy (normalized brand/name token + pack qty + UoM): single candidate auto-link; ambiguous → queue `FUZZY`.
   - No candidate → create `GlobalProducts` + offer.
4. Flag `GLOBAL_PRODUCTS_ENABLED` default **off**; **on** for SSMR (`.env.ssmr.example`). Production inert unless ops sets the flag.
5. APIs: master card, offers compare, supplier explicit link, admin/supplier match-queue resolve.
6. Marketplace commerce (fees/RFQ/escrow) and KYB console stay **out of scope** (Phases 4–5).

## Non-goals

| Out | Why |
|-----|-----|
| Checkout by `GlobalProductId` | Operational orders still use supplier SKUs |
| Marketplace fees / RFQ / escrow | §8.10 Phase 4 |
| Platform KYB admin console | §8.10 Phase 5 |
| Retailer multi-partner UI | Client follow-up |

## Flags

| Flag | SSMR | Notes |
|------|------|--------|
| `GLOBAL_PRODUCTS_ENABLED` | `true` | Match worker + catalog hooks + APIs |

## Success criteria

1. Exact GTIN links two supplier SKUs to one global (`PX_E2E_GLOBAL_PRODUCT_GTIN_LINK_OK`).
2. Ambiguous fuzzy match enqueues review (`PX_E2E_GLOBAL_PRODUCT_FUZZY_QUEUE_OK`).
3. Offers compare returns multi-supplier rows (`PX_E2E_GLOBAL_PRODUCT_OFFERS_COMPARE_OK`).
4. `phase5c_gate.sh` green.
5. Audit §8.10 Phase 3 marked Wired (backend).

## References

- [`apps/backend-go/globalproducts/`](../apps/backend-go/globalproducts/)
- [`apps/backend-go/gs1/checkdigit.go`](../apps/backend-go/gs1/checkdigit.go)
- [`apps/backend-go/schema/migrations/20260818_global_products.ddl`](../apps/backend-go/schema/migrations/20260818_global_products.ddl)
