# Analytics column tenancy — progress (2026-08-11)

**Gate:** `make analytics-tenancy-gate` → **`analytics-tenancy-gate-ok`**

## Wave 1 — RoutePerformanceAnalytics

| Area | Detail |
|------|--------|
| DDL | `SupplierId` + `Idx_RoutePerformanceAnalytics_BySupplierComputed` |
| Migration | `20260819_route_performance_supplier_id.ddl` |
| Writer | Nightly worker stamps from `SupplierTruckManifests` (Orders fallback) |
| Reader | Tenant filter via `PreferTenantSupplierID`; empty → fail-closed |

## Wave 2 — Demand analytics + shadow index

| Area | Detail |
|------|--------|
| DDL | `DemandSignals.SupplierId`, `DemandAdjustments.SupplierId` + indexes |
| Shadow | `Idx_RetailerAutoOrderShadow_BySupplierBucket` (column already existed) |
| Migration | `20260819_demand_analytics_supplier_id.ddl` |
| Writers | Weather → `_platform`; CreateSignal → PreferTenant/`_platform`; sensing stamps from `Orders.SupplierId`; sell-through looks up `Products.SupplierId` |
| Readers | `ListSignals` / `GetAdjustments` / `GetSignal` tenant-scoped (global `_platform` + NULL still visible to tenants) |
| Bootstrap | `demand.Service.SetSupplierID(PreferTenantSupplierID)` |

## Already had SupplierId (unchanged)

- `DemandForecastBaseline`, `ForecastAccuracyDaily`
- `RetailerAutoOrderShadowProposals.SupplierId` (index added)

## Residual

- Retailer-scoped HQ tables (`RetailerHqSalesDaily`, `RetailerHqStockSnapshot`, `RetailerSellThroughDaily`) — retailer PK; not supplier-partitioned by design
- Owner: apply both `20260819_*` migrations on live; recompute routes / re-run sensing for stamps
- Optional: NOT NULL tighten after backfill

## Next

Enterprise Phase 6 (decision-gated) or client 10/10 residuals.
