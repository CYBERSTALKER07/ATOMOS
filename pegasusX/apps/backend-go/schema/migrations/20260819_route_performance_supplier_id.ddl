-- Analytics column tenancy: RoutePerformanceAnalytics.SupplierId
-- Writer stamps from SupplierTruckManifests.SupplierId (fallback Orders).
-- List handler filters by PreferTenantSupplierID.

ALTER TABLE RoutePerformanceAnalytics ADD COLUMN SupplierId STRING(36);

CREATE NULL_FILTERED INDEX Idx_RoutePerformanceAnalytics_BySupplierComputed
  ON RoutePerformanceAnalytics(SupplierId, ComputedAt DESC);
