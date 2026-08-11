# Fork pick — post Enterprise Phase 5 (2026-08-11)

**Decision:** Analytics column tenancy  
**Basis:** Default from [`Clarify Next Fork`](../../.cursor/plans/) plan when implement/go is requested without an explicit 1/2/3 pick.

| Option | Choice |
|--------|--------|
| 1. Enterprise Phase 6 (marketplace/cert) | Deferred (decision-gated; procurement-heavy) |
| 2. Analytics column tenancy | **Selected** |
| 3. Client 10/10 residuals | Deferred |

**First implementable surface:** `RoutePerformanceAnalytics` + `analytics.Handlers.SupplierID` filter (see ADR residual in `docs/MULTI_TENANCY_GATE5_PHASE1.md`).

**Next action:** completed — see [`ANALYTICS_COLUMN_TENANCY_PROGRESS.md`](./ANALYTICS_COLUMN_TENANCY_PROGRESS.md).
