# FINAL GOAL (Load on Every New Session)

**Date:** 2026-08-20 (Master State Synchronization)  
**Living Product Tree:** `pegasusX/` (`pegasus/` = legacy reference / port source only)

---

## 1. Canonical Program — Authoritative Destination Documents

| Document | Purpose & Architectural Role |
|---|---|
| [`pegasusX/docs/FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](../../pegasusX/docs/FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md) | **Master Ecosystem Blueprint**: Full architectural specification covering 29 route packages, Spanner multi-tenancy (`SupplierId`), Outbox event pipeline, and 6 role-row client apps. |
| [`pegasusX/docs/GLOBAL_SCALE_PROGRAM.md`](../../pegasusX/docs/GLOBAL_SCALE_PROGRAM.md) | **Global Scale Program**: Multi-supplier registration (`SupplierId`), home cells, versioned MarketPacks, and Class A execution (GS-A/T/M/C/I/R/P). |
| [`pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`](../../pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md) | **Local Ecosystem Program**: Local-first warehouse matching (`CoverageEngine`), H3 Res 7 clustering, `ServicePins`, `SupplierRegions`, and pack-owned PSP catalog (GS-L/K, W1–W26 resolved). |
| [`pegasusX/docs/GLOBAL_SCALE_CLIENT_UI.md`](../../pegasusX/docs/GLOBAL_SCALE_CLIENT_UI.md) | **Client Visualization Program**: Realtime command dashboards, `StatusStack` kit, and Plan & Brain tabs across all role surfaces (GS-U0–U9 shipped). |
| [`pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md`](../../pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md) | **Role Parity Matrix**: Verified implementation and test pass status across all 6 role rows (Supplier, Retailer, Driver, Warehouse, Factory, Payload) + Platform Admin. |

---

## 2. North Star Architecture

A **global, local-first, multi-supplier logistics operating system**.

Many companies, in many markets, **register** as isolated suppliers (`SupplierId`), land in a **home cell**, receive a **market pack that checkout / fiscal / proximity / PSP catalog actually use**, invite their roles, and run Class A operations:

$$\text{Order} \longrightarrow \text{Stock} \longrightarrow \text{Truck} \longrightarrow \text{Cash/Credit} \longrightarrow \text{Fiscal} \longrightarrow \text{Payout}$$

Retailers attach **multiple** suppliers; mixed carts split into per-supplier child orders (`ParentOrders`) within the same market pack country. Same codebase, cloned regional cells. Zero country-specific forks. Zero secondary tenant keys.

---

## 3. Core Product Laws & Invariants

1. **One Tenant Key**: `SupplierId STRING(36) NOT NULL`. Pack, cell, country, city, and region are attributes.
2. **Market Owns Money**: Currency, minor decimal places, PSP catalog, fiscal adapter, and payout rail derive strictly from shipped `MarketPack`.
3. **Same-Market Orders Only**: Retailer location, warehouse, factory, and supplier `MarketCode` countries must match. Mismatches return `422 cross_market_deferred`.
4. **Local-First Default**: Store resolves to closest covering warehouse; warehouse replenishment resolves to closest factory on an active `SupplyLane`.
5. **Supplier Override Wins**: `ServicePins` (location/retailer/region) and `WarehouseCoverageCells` take precedence over distance-based matching.
6. **Empty Geography Fails Closed**: Missing `CountryCode` on warehouse/factory/store returns `422 geography_incomplete`.
7. **Pack Filters PSP UI**: GET `/v1/*/payment-catalog` returns `LivePackGateways(pack)`. Foreign gateways are hidden.
8. **Unkeyed Rails Fail Honestly**: Missing keys return HTTP 501 `no_live_keys` via `catalogHonestyExecutor`. Never a fake 200 redirect.
9. **One Country = One Pack + Adapters**: New countries introduce a versioned `MarketPack` + 1–3 adapters without codebase forks.
10. **Class A Integrity**: Integer minor units, fiscal hard-gate, pay-at-delivery, dual manifests (`SupplierTruckManifests` vs `FactoryTruckManifests`), H3 Res 7, and transactional outbox event emissions.

---

## 4. Implementation Phasing & Status

```
GS-A  Auth & Session MarketPack      (A0–A2 SHIPPED)
GS-T  Self-Serve Tenant Register     (T1–T5 SHIPPED)
GS-M  MarketPack Reading             (M1–M7 SHIPPED; Flag stays false in local/SSMR until live Soliq)
GS-C  Regional Cell Scaffold         (C1–C5 SCAFFOLDED; apply deferred)
GS-I  Enterprise Identity (OIDC)     (SHIPPED; SupplierOIDC + endpoints)
GS-L  Local Matching & ServicePins   (L0–L4 SHIPPED; W1–W26 items closed)
GS-K  Pack PSP Catalog & Honesty     (K1–K3 SHIPPED; HTTP 501 placeholder executors)
GS-U  Client Visualization           (U0–U9 SHIPPED; command dashboards & StatusStack)
GS-P  Partner Dialects (EDI/1C/AS2)  (SHIPPED; B2B integration layer)
```

---

## 5. Explicit Non-Goals & Boundaries

- No open-world public ads/discovery marketplace in core trade loops.
- No cross-country checkout/orders/payouts in v1 (same-market only).
- No unified global Spanner across geographic borders (regional cloned cells).
- No unattended `terraform apply` of second cells in development sessions.
- No live charge execution on unkeyed payment gateways without verified credentials (returns HTTP 501 `no_live_keys`).
- Factory planning and auto-order `place` features default to flag-off.
- Deprecated/unwired endpoints return RFC 7807 HTTP 410 (`audit_unwired`, `feature_disabled`).

---

## 6. Verification & Quality Assurance Standard

Re-verify all status claims directly in source code with exact `file:line` citations and test suite runs.

- Working Memory: `.agents/memory/WORKSPACE.md`
- Documentation Source of Truth: `pegasusX/docs/DOCS_SOURCE_OF_TRUTH.md`
- Feature Classification: `pegasusX/docs/GLOBAL_SCALE_BACKEND_FEATURES.md`
- Class A Coverage Definition: `pegasusX/docs/PROD_ECOSYSTEM_GOAL.md`


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
