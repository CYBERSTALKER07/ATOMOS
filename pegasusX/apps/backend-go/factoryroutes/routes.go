package factoryroutes

import (
	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/factory"
)

// Deps is the narrow dependency contract for factory routes.
type Deps struct {
	Service             *factory.Service
	JWTSecret           string
	JWTIssuer           string
	Spanner             *spanner.Client
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
}

// RegisterRoutes mounts factory role-row operational endpoints.
//
// Loading-bay routes (list/detail/start-loading/seal) are also open to RolePayload
// so the payload terminal can close the factory → payload Class A loop (P1-18).
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	r.Post("/v1/auth/factory/login", d.Service.HandleFactoryLogin)
	r.Post("/v1/auth/factory/register", d.Service.HandleFactoryRegister)
	r.Post("/v1/auth/factory/refresh", d.Service.HandleFactoryRefresh)
	r.Post("/v1/factory/setup", d.Service.HandleFactorySetup)

	mountLoadingBay := func(rr chi.Router) {
		rr.Get("/v1/factory/manifests", d.Service.HandleManifests)
		rr.Get("/v1/factory/manifests/{manifestID}", d.Service.HandleManifestDetail)
		rr.Post("/v1/factory/manifests/{manifestID}/start-loading", d.Service.HandleManifestStartLoading)
		rr.Post("/v1/factory/manifests/{manifestID}/seal", d.Service.HandleManifestSeal)
		rr.Get("/v1/factory/manifest-exceptions", d.Service.HandleManifestExceptions)
	}

	mountOps := func(rr chi.Router) {
		rr.Post("/v1/factories", d.Service.HandleCreateFactory)
		rr.Get("/v1/factories/{factoryId}", d.Service.HandleGetFactory)
		rr.Put("/v1/factories/{factoryId}", d.Service.HandleUpdateFactory)
		rr.Get("/v1/factories", d.Service.HandleListFactories)

		rr.Get("/v1/factory/analytics/overview", d.Service.HandleAnalyticsOverview)
		rr.Get("/v1/factory/dashboard", d.Service.HandleDashboard)
		rr.Get("/v1/factory/profile", d.Service.HandleProfile)
		rr.Get("/v1/factory/ops/location", d.Service.HandleOpsLocation)
		rr.Patch("/v1/factory/ops/location", d.Service.HandleOpsLocation)
		rr.Get("/v1/factory/transfers", d.Service.HandleTransfers)
		rr.Post("/v1/factory/transfers/create", d.Service.HandleTransfers)
		rr.Get("/v1/factory/transfers/{transferID}", d.Service.HandleTransferByID)
		rr.Post("/v1/factory/transfers/{transferID}/transition", d.Service.HandleTransferTransition)
		rr.Get("/v1/factory/fleet", d.Service.HandleFleet)
		rr.Get("/v1/factory/fleet/live-map", d.Service.HandleFactoryFleetLiveMap)
		rr.Post("/v1/factory/manifests/{manifestID}/dispatch", d.Service.HandleManifestDispatch)
		rr.Post("/v1/factory/manifests/{manifestID}/complete", d.Service.HandleManifestComplete)
		rr.Post("/v1/factory/manifests/rebalance", d.Service.HandleManifestRebalance)
		rr.Post("/v1/factory/manifests/cancel-transfer", d.Service.HandleManifestCancelTransfer)
		rr.Post("/v1/factory/manifests/cancel", d.Service.HandleManifestCancel)
		rr.Post("/v1/factory/manifest-exceptions/{exceptionID}/resolve", d.Service.HandleResolveManifestException)
		rr.Get("/v1/factory/fleet/drivers", d.Service.HandleFleetDrivers)
		rr.Get("/v1/factory/fleet/vehicles", d.Service.HandleFleetVehicles)
		rr.Get("/v1/factory/staff", d.Service.HandleStaff)
		rr.Post("/v1/factory/staff", d.Service.HandleStaff)
		rr.Get("/v1/factory/staff/{staffID}", d.Service.HandleStaffDetail)
		rr.Post("/v1/factory/dispatch", d.Service.HandleDispatch)
		rr.Get("/v1/factory/supply-requests", d.Service.HandleSupplyRequests)
		rr.Get("/v1/factory/sla-board", d.Service.HandleSLABoard)
		rr.Get("/v1/factory/supply-requests/{id}/fulfill-options", d.Service.HandleSupplyRequestFulfillOptions)
		rr.Post("/v1/factory/supply-requests/{requestID}/accept", d.Service.HandleAcceptSupplyRequest)
		rr.Patch("/v1/factory/supply-requests/{id}", d.Service.HandleSupplyRequestTransition)
	}

	factoryRoles := []auth.Role{auth.RoleFactory, auth.RoleFactoryAdmin, auth.RoleAdmin}
	loadingBayRoles := []auth.Role{auth.RoleFactory, auth.RoleFactoryAdmin, auth.RoleAdmin, auth.RolePayload}

	register := func(gr chi.Router) {
		gr.Group(func(ws chi.Router) {
			ws.Use(auth.RequireRole(factoryRoles...))
			ws.Get("/v1/factory/ws-session", auth.WSSessionHandler(d.JWTSecret, d.JWTIssuer, 0))
		})
		gr.Group(func(bay chi.Router) {
			bay.Use(auth.RequireRole(loadingBayRoles...))
			bay.Use(auth.RequireFactoryScope)
			mountLoadingBay(bay)
		})
		gr.Group(func(ops chi.Router) {
			ops.Use(auth.RequireRole(factoryRoles...))
			ops.Use(auth.RequireFactoryScope)
			mountOps(ops)
			// Factory roles also need loading-bay under the same scope group when
			// payload routes are registered separately — mountLoadingBay is already
			// on loadingBayRoles which includes factory roles, so no duplicate needed.
		})
	}

	if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
		r.Group(func(gr chi.Router) {
			gr.Use(auth.FirebaseAuth(d.FirebaseVerifier))
			register(gr)
		})
		return
	}

	r.Group(register)
}
