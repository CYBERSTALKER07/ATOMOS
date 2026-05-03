// Package proximityroutes owns the supplier geo-spatial planning surface —
// the five /v1/supplier/* endpoints for serving-warehouse resolution, coverage
// reporting, zone preview, coverage validation, and live warehouse load stats.
//
// Handler bodies live in backend-go/proximity. This package is a thin composer
// that mounts them behind the SUPPLIER / ADMIN role guard and the caller-
// supplied logging middleware.
package proximityroutes

import (
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	"backend-go/proximity"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to mount the supplier geo routes.
type Deps struct {
	Spanner    *spanner.Client
	ReadRouter proximity.ReadRouter
	Log        Middleware
}

// RegisterRoutes mounts the supplier geo-spatial planning surface:
//
//	GET  /v1/supplier/serving-warehouse           — exclusive warehouse resolver
//	GET  /v1/supplier/geo-report                  — coverage health report
//	GET  /v1/supplier/zone-preview                — real-time coverage preview
//	POST /v1/supplier/warehouses/validate-coverage — H3 coverage + conflict check
//	GET  /v1/supplier/warehouse-loads             — live warehouse queue depth
func RegisterRoutes(r chi.Router, d Deps) {
	log := d.Log
	supplier := []string{"SUPPLIER", "ADMIN"}

	r.HandleFunc("/v1/supplier/serving-warehouse",
		auth.RequireRole(supplier, log(proximity.HandleGetServingWarehouse(d.Spanner, d.ReadRouter))))
	r.HandleFunc("/v1/supplier/geo-report",
		auth.RequireRole(supplier, log(proximity.HandleGeoReport(d.Spanner))))
	r.HandleFunc("/v1/supplier/zone-preview",
		auth.RequireRole(supplier, log(proximity.HandleZonePreview(d.Spanner))))
	r.HandleFunc("/v1/supplier/warehouses/validate-coverage",
		auth.RequireRole(supplier, log(proximity.HandleValidateCoverage(d.Spanner))))
	r.HandleFunc("/v1/supplier/warehouse-loads",
		auth.RequireRole(supplier, log(proximity.HandleWarehouseLoads(d.Spanner))))
}
