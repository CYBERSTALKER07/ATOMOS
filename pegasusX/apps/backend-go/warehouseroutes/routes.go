// Package warehouseroutes mounts warehouse-role endpoints.
package warehouseroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
)

// Deps is the narrow dependency contract for warehouse routes.
type Deps struct {
	Service             *warehouse.Service
	JWTSecret           string
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
}

// RegisterRoutes mounts warehouse role-row operational endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	mountProtected := func(rr chi.Router) {
		rr.Get("/v1/warehouse/ops/dashboard", d.Service.HandleDashboard)
		rr.Get("/v1/warehouse/ops/inventory", d.Service.HandleInventory)
		rr.Get("/v1/warehouse/ops/orders", d.Service.HandleOrders)
		rr.Post("/v1/warehouse/ops/dispatch/preview", d.Service.HandleDispatchPreview)
		rr.Get("/v1/warehouse/demand/forecast", d.Service.HandleDemandForecast)
		rr.Get("/v1/warehouse/supply-requests", d.Service.HandleSupplyRequests)
		rr.Post("/v1/warehouse/supply-requests", d.Service.HandleSupplyRequests)
		rr.Get("/v1/warehouse/dispatch-locks", d.Service.HandleDispatchLocks)
		rr.Post("/v1/warehouse/dispatch-lock", d.Service.HandleDispatchLock)
		rr.Delete("/v1/warehouse/dispatch-lock", d.Service.HandleDispatchLock)
	}

	if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
		r.Group(func(gr chi.Router) {
			gr.Use(auth.FirebaseAuth(d.FirebaseVerifier))
			gr.Use(auth.RequireRole(auth.RoleWarehouseAdmin, auth.RoleAdmin))
			mountProtected(gr)
		})
		return
	}

	// Local scaffold fallback uses supplier-portal cookie auth.
	r.Group(func(gr chi.Router) {
		gr.Use(auth.CookieAuth(d.JWTSecret))
		gr.Use(auth.RequireRole(auth.RoleAdmin))
		mountProtected(gr)
	})
}
