#!/usr/bin/env bash
# Analytics column tenancy gate: route + demand analytics SupplierId surfaces.
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "[1/5] RoutePerformanceAnalytics migration + ddl ..."
test -f apps/backend-go/schema/migrations/20260819_route_performance_supplier_id.ddl
grep -q 'ADD COLUMN SupplierId' apps/backend-go/schema/migrations/20260819_route_performance_supplier_id.ddl
grep -q 'Idx_RoutePerformanceAnalytics_BySupplierComputed' apps/backend-go/schema/migrations/20260819_route_performance_supplier_id.ddl
grep -A12 'CREATE TABLE RoutePerformanceAnalytics' apps/backend-go/schema/spanner.ddl | grep -q 'SupplierId'
grep -q 'Idx_RoutePerformanceAnalytics_BySupplierComputed' apps/backend-go/schema/spanner.ddl

echo "[2/5] DemandSignals / DemandAdjustments / shadow supplier index ..."
test -f apps/backend-go/schema/migrations/20260819_demand_analytics_supplier_id.ddl
grep -q 'ALTER TABLE DemandSignals ADD COLUMN SupplierId' apps/backend-go/schema/migrations/20260819_demand_analytics_supplier_id.ddl
grep -q 'ALTER TABLE DemandAdjustments ADD COLUMN SupplierId' apps/backend-go/schema/migrations/20260819_demand_analytics_supplier_id.ddl
grep -q 'Idx_DemandSignals_BySupplierCreated' apps/backend-go/schema/migrations/20260819_demand_analytics_supplier_id.ddl
grep -q 'Idx_DemandAdjustments_BySupplierDate' apps/backend-go/schema/migrations/20260819_demand_analytics_supplier_id.ddl
grep -q 'Idx_RetailerAutoOrderShadow_BySupplierBucket' apps/backend-go/schema/migrations/20260819_demand_analytics_supplier_id.ddl
grep -q 'Idx_DemandSignals_BySupplierCreated' apps/backend-go/schema/spanner.ddl
grep -q 'Idx_DemandAdjustments_BySupplierDate' apps/backend-go/schema/spanner.ddl
grep -q 'Idx_RetailerAutoOrderShadow_BySupplierBucket' apps/backend-go/schema/spanner.ddl

echo "[3/5] Writers stamp SupplierId ..."
grep -q 'SELECT SupplierId, DriverId, StopCount' apps/backend-go/analytics/route_analytics_worker.go
grep -q 'perf.SupplierId' apps/backend-go/analytics/route_analytics_worker.go
grep -q 'PlatformSupplierID' apps/backend-go/demand/models.go
grep -q 'SupplierId": ResolveSupplierID' apps/backend-go/demand/worker_weather.go
grep -q '"SupplierId":     adj.SupplierId' apps/backend-go/demand/worker_sensing.go

echo "[4/5] Handlers / PreferTenant wiring ..."
grep -q 'routePerfListStmt' apps/backend-go/analytics/handlers.go
grep -q 'WHERE SupplierId = @supplierId' apps/backend-go/analytics/handlers.go
grep -q 'SetSupplierID' apps/backend-go/demand/service.go
grep -q 'PreferTenantSupplierID' apps/backend-go/bootstrap/bootstrap.go

echo "[5/5] Unit tests ..."
(cd apps/backend-go && go test ./analytics/ ./demand/ -count=1)

echo "analytics-tenancy-gate-ok"
