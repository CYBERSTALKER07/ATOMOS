// Package supplierinsightsroutes owns the supplier portal read-side route
// composition for country overrides, analytics, financials, and CRM. Handler
// bodies live in backend-go/countrycfg, backend-go/analytics, and
// backend-go/supplier.
package supplierinsightsroutes

import (
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/analytics"
	"backend-go/auth"
	"backend-go/countrycfg"
	"backend-go/proximity"
	"backend-go/supplier"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to mount supplier insight routes.
type Deps struct {
	Spanner     *spanner.Client
	ReadRouter  proximity.ReadRouter
	CountryCfg  *countrycfg.Service
	Log         Middleware
	Idempotency Middleware
}

// RegisterRoutes mounts the supplier insights surface:
//
//	GET/PUT /v1/supplier/country-overrides                 — supplier country override list/upsert
//	GET/DELETE /v1/supplier/country-overrides/{code}       — single override detail/revert
//	GET /v1/supplier/analytics/velocity                    — sales velocity
//	GET /v1/supplier/analytics/demand/today                — demand snapshot
//	GET /v1/supplier/analytics/demand/history              — demand history
//	GET /v1/supplier/analytics/{transit-heatmap,throughput,load-distribution,node-efficiency,sla-health,revenue,top-retailers}
//	GET /v1/supplier/financials                            — supplier financials
//	GET /v1/supplier/crm/retailers                         — CRM retailer list
//	GET /v1/supplier/crm/retailers/{id}                    — CRM retailer detail
func RegisterRoutes(r chi.Router, d Deps) {
	log := d.Log
	idem := d.Idempotency
	supplierRole := []string{"SUPPLIER", "ADMIN"}

	r.HandleFunc("/v1/supplier/country-overrides",
		auth.RequireRole(supplierRole, log(idem(countrycfg.HandleSupplierCountryOverrides(d.CountryCfg)))))
	r.HandleFunc("/v1/supplier/country-overrides/*",
		auth.RequireRole(supplierRole, log(idem(countrycfg.HandleSupplierCountryOverrideByCode(d.CountryCfg)))))
	r.HandleFunc("/v1/supplier/analytics/velocity",
		auth.RequireRole(supplierRole, log(analytics.HandleGetVelocity(d.Spanner, d.ReadRouter))))
	r.HandleFunc("/v1/supplier/analytics/demand/today",
		auth.RequireRole(supplierRole, log(analytics.HandleDemandToday(d.Spanner, d.ReadRouter))))
	r.HandleFunc("/v1/supplier/analytics/demand/history",
		auth.RequireRole(supplierRole, log(analytics.HandleDemandHistory(d.Spanner, d.ReadRouter))))
	r.HandleFunc("/v1/supplier/analytics/transit-heatmap",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(analytics.HandleTransitHeatmap(d.Spanner, d.ReadRouter)))))
	r.HandleFunc("/v1/supplier/analytics/throughput",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(analytics.HandleThroughput(d.Spanner, d.ReadRouter)))))
	r.HandleFunc("/v1/supplier/analytics/load-distribution",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(analytics.HandleLoadDistribution(d.Spanner, d.ReadRouter)))))
	r.HandleFunc("/v1/supplier/analytics/node-efficiency",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(analytics.HandleNodeEfficiency(d.Spanner, d.ReadRouter)))))
	r.HandleFunc("/v1/supplier/analytics/sla-health",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(analytics.HandleSLAHealth(d.Spanner, d.ReadRouter)))))
	r.HandleFunc("/v1/supplier/analytics/revenue",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(analytics.HandleRevenue(d.Spanner, d.ReadRouter)))))
	r.HandleFunc("/v1/supplier/analytics/top-retailers",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(analytics.HandleTopRetailers(d.Spanner, d.ReadRouter)))))
	r.HandleFunc("/v1/supplier/financials",
		auth.RequireRole(supplierRole, log(analytics.HandleSupplierFinancials(d.Spanner, d.ReadRouter))))
	r.HandleFunc("/v1/supplier/crm/retailers",
		auth.RequireRole(supplierRole, log(supplier.HandleCRMRetailers(d.Spanner))))
	r.HandleFunc("/v1/supplier/crm/retailers/*",
		auth.RequireRole(supplierRole, log(supplier.HandleCRMRetailerDetail(d.Spanner))))
}
