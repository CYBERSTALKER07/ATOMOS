# Phase 5 progress — Runtime multi-tenancy Phase 1 + Outbox soak (2026-08-11)

> **HISTORICAL / FROZEN — session progress note; do not treat as current gap SoT.**
> Living residuals: [`../PROD_READINESS_SEQUENCE.md`](../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](./ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md).


**ADR:** `docs/MULTI_TENANCY_GATE5_PHASE1.md`  
**Gate:** `make phase5-gate` → **`phase5-gate-ok`** (re-proved 2026-08-11, Spanner emulator `localhost:9010`; full regression)  
**Audit:** `PLATFORM_AUDIT.md` §8.10 → Phase 1 **Wired (Done)**; Outbox `SupplierId` NOT NULL soak closed  
**Live proof (prior):** `go run ./cmd/ssmr-smokecheck tenant` → all four `PX_E2E_TENANT_*` / `PX_E2E_OUTBOX_TENANT_*` **OK**

## Proof (2026-08-11)

| Step | Result |
|------|--------|
| [1/7] Phase-4 regression | `phase4-gate-ok` |
| [2/7] Registration freeze | PASS |
| [3/7] TenantContext spine | PASS |
| [4/7] Order tenant / IDOR | PASS |
| [5/7] Outbox + NOT NULL migration asserts | PASS |
| [6/7] Vertical scope + rate limits | PASS |
| [7/7] Env + SSMR markers | OK |

## Shipped

| Week | Item | Detail |
|------|------|--------|
| 0–1 | Freeze + TenantContext spine | Registration freeze; `PreferTenantSupplierID` / `RequireTenant` |
| 2–5 | Order/payment/warehouse | HTTP IDOR loads; checkout tenant stamp; dispatch/fleet scope |
| 6–8 | Driver/payload/factory/portal | Package `resolveSupplierScope`; portal `ScopedSupplierID`; credit notes |
| Workers | Cash recon escalation | JOIN `Drivers.SupplierId`; attach worker `TenantContext` |
| 11 | Fail-closed seed | Enforced + authenticated → no seed; production requires enforcement |
| 11 | Order/payment/domain seed demotion | `seedSupplierID` + request `resolveSupplierScope` |
| 9+ | Outbox stamp + backfill | Write paths stamp `SupplierId`; worker + `cmd/backfill-outbox-supplier-id` |
| 12 | SSMR tenant runners | Early in e2e + focused `tenant` check; demo scope marks seed `IsRegistered` |
| Soak | Outbox `SupplierId` NOT NULL | `ResolveSupplierID` + `_platform` sentinel; `20260819_outbox_supplier_id_not_null.ddl`; `spanner.ddl` NOT NULL |

## Fix landed this run (soak)

- Writers always stamp `SupplierId` via `ResolveSupplierID` (`PlatformSupplierID=_platform` when no tenant).
- Backfill stamps `_platform` for unresolvable rows (no longer skips empty payload).
- Migration + `spanner.ddl` tighten column to NOT NULL; `phase5_gate.sh` asserts files + symbols.

## Residual

- ~~Analytics platform tables still lack column-level `SupplierId`~~ → **RoutePerformanceAnalytics + DemandSignals/DemandAdjustments Wired** — see [`ANALYTICS_COLUMN_TENANCY_PROGRESS.md`](./ANALYTICS_COLUMN_TENANCY_PROGRESS.md) (`analytics-tenancy-gate-ok`); retailer HQ rollups remain retailer-keyed by design
- Full ecosystem `smoke_ssmr.sh` can still fail later on unrelated payloader timeouts (tenant markers via `tenant` check)
- Owner: apply `20260819_outbox_supplier_id_not_null.ddl` + `20260819_route_performance_supplier_id.ddl` + `20260819_demand_analytics_supplier_id.ddl` on live after backfill

## Next deep-dive

**Enterprise Phase 6** (marketplace/cert — decision-gated) or further analytics tables / client 10/10 residuals.
