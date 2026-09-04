# Search Decision (W1 — 2026-08-12)

**Decision:** keep Spanner prefix / `LIKE` search for catalog and POS. Do **not** introduce Elasticsearch, OpenSearch, Typesense, or Meilisearch in this prod bar.

## Why

- Current surfaces (`GET /v1/catalog/suppliers/search`, POS catalog search, replenishment SKU filters) are point/list + `LOWER(Name) LIKE` — adequate for single-tenant supplier catalogs and retailer local SKUs.
- A search engine adds another operational plane (index lag, tenancy stamps, rebuild jobs) without measured SLO pain.
- Coverage rule and Class A loops are higher leverage than search infra right now.

## When to revisit

Open a search engine initiative only when **any** of:

1. Catalog search p99 > 500ms under realistic SSMR/prod load for ≥7 days, or
2. Supplier catalog size routinely exceeds ~50k active SKUs per tenant with name/barcode search as a primary workflow, or
3. Cross-supplier global product match needs full-text / fuzzy beyond `GlobalProducts` match queue.

## Interim rules

- Prefer prefix / equality filters over leading-wildcard `LIKE '%…%'`.
- Every new search endpoint must be tenant-scoped (claims / TenantContext), never global unscoped `LIKE`.
- Document any new SQL search path in [`DATA_FLOW_AS_IMPLEMENTED.md`](./DATA_FLOW_AS_IMPLEMENTED.md).

Gap anchor: P2-13 (accepted as deferred, not open defect).
