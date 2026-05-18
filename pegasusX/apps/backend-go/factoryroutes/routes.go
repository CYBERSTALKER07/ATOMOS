package factoryroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/factory"
)

// Deps is the narrow dependency contract for factory routes.
type Deps struct {
	Service             *factory.Service
	JWTSecret           string
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
}

// RegisterRoutes mounts factory role-row operational endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	mountProtected := func(rr chi.Router) {
		rr.Get("/v1/factory/analytics/overview", d.Service.HandleAnalyticsOverview)
		rr.Get("/v1/factory/dashboard", d.Service.HandleDashboard)
		rr.Get("/v1/factory/profile", d.Service.HandleProfile)
		rr.Get("/v1/factory/transfers", d.Service.HandleTransfers)
		rr.Post("/v1/factory/transfers/create", d.Service.HandleTransfers)
		rr.Get("/v1/factory/manifests", d.Service.HandleManifests)
		rr.Get("/v1/factory/manifests/{manifestID}", d.Service.HandleManifestDetail)
		rr.Post("/v1/factory/manifests/{manifestID}/start-loading", d.Service.HandleManifestStartLoading)
		rr.Post("/v1/factory/manifests/{manifestID}/seal", d.Service.HandleManifestSeal)
		rr.Post("/v1/factory/manifests/{manifestID}/dispatch", d.Service.HandleManifestDispatch)
		rr.Post("/v1/factory/manifests/{manifestID}/complete", d.Service.HandleManifestComplete)
		rr.Post("/v1/factory/manifests/rebalance", d.Service.HandleManifestRebalance)
		rr.Post("/v1/factory/manifests/cancel-transfer", d.Service.HandleManifestCancelTransfer)
		rr.Post("/v1/factory/manifests/cancel", d.Service.HandleManifestCancel)
		rr.Get("/v1/factory/manifest-exceptions", d.Service.HandleManifestExceptions)
		rr.Get("/v1/factory/fleet/drivers", d.Service.HandleFleetDrivers)
		rr.Get("/v1/factory/fleet/vehicles", d.Service.HandleFleetVehicles)
		rr.Get("/v1/factory/staff", d.Service.HandleStaff)
		rr.Post("/v1/factory/dispatch", d.Service.HandleDispatch)
		rr.Get("/v1/factory/supply-requests", d.Service.HandleSupplyRequests)
	}

	if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
		r.Group(func(gr chi.Router) {
			gr.Use(auth.FirebaseAuth(d.FirebaseVerifier))
			gr.Use(auth.RequireRole(auth.RoleFactoryAdmin, auth.RoleAdmin))
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
