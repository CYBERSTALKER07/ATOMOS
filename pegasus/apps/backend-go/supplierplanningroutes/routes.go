// Package supplierplanningroutes owns supplier planning and network route
// composition. Handler implementations remain in backend-go/supplier,
// backend-go/factory, and backend-go/proximity.
package supplierplanningroutes

import (
	"context"
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	cachepkg "backend-go/cache"
	"backend-go/factory"
	"backend-go/proximity"
	"backend-go/supplier"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to mount supplier planning routes.
type Deps struct {
	Spanner          *spanner.Client
	Cache            *cachepkg.Cache
	NetworkOptimizer *factory.NetworkOptimizerService
	SupplyLanes      *factory.SupplyLanesService
	KillSwitch       *factory.KillSwitchService
	PullMatrix       *factory.PullMatrixService
	PredictivePush   *factory.PredictivePushService
	IsDispatchLocked func(ctx context.Context, supplierID, warehouseID, factoryID string) bool
	Log              Middleware
	Idempotency      Middleware
}

// RegisterRoutes mounts the supplier planning and network surface:
//
//	GET/POST /v1/supplier/delivery-zones               — delivery zone list/create
//	PUT/DELETE /v1/supplier/delivery-zones/{id}        — delivery zone mutate
//	GET/POST /v1/supplier/factories                    — supplier factory list/create
//	GET/PATCH/DELETE /v1/supplier/factories/{id}       — supplier factory detail
//	GET /v1/supplier/factories/recommend-warehouses    — warehouse recommendation
//	GET /v1/supplier/factories/optimal-assignments     — factory assignment matrix
//	GET /v1/supplier/geocode/reverse                   — reverse geocoding helper
//	GET /v1/supplier/retailers/locations               — retailer map layer
//	GET/POST /v1/supplier/supply-lanes                 — supply lane list/create
//	GET/PATCH/DELETE /v1/supplier/supply-lanes/{id}    — supply lane detail
//	GET/PUT /v1/supplier/network-mode                  — network optimization mode
//	GET /v1/supplier/network-analytics                 — network analytics
//	POST /v1/supplier/replenishment/kill-switch        — automation halt
//	GET /v1/supplier/replenishment/audit               — replenishment audit
//	POST /v1/supplier/replenishment/pull-matrix        — manual pull matrix trigger
//	POST /v1/supplier/replenishment/predictive-push    — manual predictive push trigger
//	GET /v1/supplier/warehouses/territory-preview      — territory preview
//	POST /v1/supplier/warehouses/apply-territory       — territory reassignment
func RegisterRoutes(r chi.Router, d Deps) {
	log := d.Log
	idem := d.Idempotency
	supplierRole := []string{"SUPPLIER", "ADMIN"}

	r.HandleFunc("/v1/supplier/delivery-zones/",
		auth.RequireRole(supplierRole, log(supplier.HandleDeliveryZoneAction(d.Spanner))))
	r.HandleFunc("/v1/supplier/delivery-zones",
		auth.RequireRole(supplierRole, log(supplier.HandleDeliveryZones(d.Spanner))))
	r.HandleFunc("/v1/supplier/factories",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(factory.HandleSupplierFactories(d.Spanner, d.Cache)))))
	r.HandleFunc("/v1/supplier/factories/",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(factory.HandleSupplierFactoryDetail(d.Spanner, d.Cache)))))
	r.HandleFunc("/v1/supplier/factories/recommend-warehouses",
		auth.RequireRole(supplierRole, log(factory.HandleRecommendWarehouses(d.Spanner))))
	r.HandleFunc("/v1/supplier/factories/optimal-assignments",
		auth.RequireRole(supplierRole, log(factory.HandleOptimalAssignments(d.Spanner))))
	r.HandleFunc("/v1/supplier/geocode/reverse",
		auth.RequireRole(supplierRole, log(proximity.HandleReverseGeocode())))
	r.HandleFunc("/v1/supplier/retailers/locations",
		auth.RequireRole(supplierRole, log(supplier.HandleRetailerLocations(d.Spanner))))
	r.HandleFunc("/v1/supplier/supply-lanes",
		auth.RequireRole(supplierRole, log(idem(d.SupplyLanes.HandleSupplyLanes))))
	r.HandleFunc("/v1/supplier/supply-lanes/",
		auth.RequireRole(supplierRole, log(d.SupplyLanes.HandleSupplyLaneAction)))
	r.HandleFunc("/v1/supplier/network-mode",
		auth.RequireRole(supplierRole, log(networkModeHandler(d.NetworkOptimizer))))
	r.HandleFunc("/v1/supplier/network-analytics",
		auth.RequireRole(supplierRole, log(d.NetworkOptimizer.HandleNetworkAnalytics)))
	r.HandleFunc("/v1/supplier/replenishment/kill-switch",
		auth.RequireRole(supplierRole, log(d.KillSwitch.HandleKillSwitch)))
	r.HandleFunc("/v1/supplier/replenishment/audit",
		auth.RequireRole(supplierRole, log(d.KillSwitch.HandleListKillSwitchAudit)))
	r.HandleFunc("/v1/supplier/replenishment/pull-matrix",
		auth.RequireRole(supplierRole, log(d.PullMatrix.HandleManualPullMatrix)))
	r.HandleFunc("/v1/supplier/replenishment/predictive-push",
		auth.RequireRole(supplierRole, log(d.PredictivePush.HandleManualPredictivePush)))
	r.HandleFunc("/v1/supplier/warehouses/territory-preview",
		auth.RequireRole(supplierRole, log(proximity.HandlePreviewTerritories(d.Spanner))))
	r.HandleFunc("/v1/supplier/warehouses/apply-territory",
		auth.RequireRole(supplierRole, log(proximity.HandleApplyTerritory(d.Spanner, d.IsDispatchLocked))))
}

func networkModeHandler(service *factory.NetworkOptimizerService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			service.HandleGetNetworkMode(w, r)
		case http.MethodPut:
			service.HandleSetNetworkMode(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}
