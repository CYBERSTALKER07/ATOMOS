# Privacy & Multi-Tenant Isolation

> **Runtime status (2026-08-12):** Gate 5 Phase 1–3 backends wired (`TenantContext` / ParentOrders / GlobalProducts). Seed supplier remains bootstrap fallback. Isolation rules below are enforced intent + residual seed/UI gaps — not “uncoded.”


> **PLANNING BASELINE** — not living runtime status. Prefer [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md) and code for what is shipped.


**Runtime status (2026-08-07):** Schema is multi-tenant-shaped; request plane is still single-supplier (seed injected at bootstrap). **Gate 5 Phase 1 program ADR:** [MULTI_TENANCY_GATE5_PHASE1.md](../../MULTI_TENANCY_GATE5_PHASE1.md) (Accepted plan; not yet coded). Isolation rules below are the **target** once Phase 1 lands.

## Rules

| Viewer | Scope |
|--------|--------|
| Supplier | Own supplier_id only |
| Warehouse/Factory staff | Home node + supplier |
| Retailer | Own retailer_id |
| Driver | Assigned manifests/orders |
| Payload | Warehouse-scoped loads |
| Platform admin | Cross-tenant with explicit audit |

## Control tower

Role-scoped views of inventory, exceptions, fiscal, credit exposure. No cross-supplier leakage in JWT claims or list queries (IDOR tests mandatory).

## Phase 1 enforcement (target)

- Tenant key = `SupplierId` from JWT `supplier_id` or partner `Principal.TenantID` via shared `TenantContext`.
- Fail-closed middleware when tenant context is absent (`TENANT_CONTEXT_ENFORCED`).
- Coding order: freeze multi-supplier register → spine → vertical slices → remove bootstrap seed. See ADR for outbox partition and per-tenant rate limits.
