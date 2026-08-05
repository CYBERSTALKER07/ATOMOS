package payloaderoutes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/payload"
)

// Deps is the narrow dependency contract for payload routes.
type Deps struct {
	Service      *payload.Service
	OrderService interface {
		HandleExceptionReport(http.ResponseWriter, *http.Request)
	}
	JWTSecret           string
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
}

// RegisterRoutes mounts payload role-row operational endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	r.Post("/v1/auth/payloader/login", d.Service.HandlePayloaderLogin)
	r.Post("/v1/auth/payloader/refresh", d.Service.HandlePayloaderRefresh)

	mountProtected := func(rr chi.Router) {
		rr.Get("/v1/payloader/trucks", d.Service.HandleTrucks)
		rr.Get("/v1/payloader/orders", d.Service.HandleOrders)
		rr.Get("/v1/payloader/manifests", d.Service.HandleManifestsList)
		rr.Post("/v1/payloader/manifests/seal-all", d.Service.HandleSealAll)
		rr.Post("/v1/payloader/manifests/seal-completed", d.Service.HandleSealCompletedManifests)
		rr.Get("/v1/payloader/manifests/{manifestID}", d.Service.HandleManifestDetail)
		rr.Post("/v1/payloader/manifests/{manifestID}/start-loading", d.Service.HandleStartLoading)
		rr.Post("/v1/payloader/manifests/{manifestID}/inject-order", d.Service.HandleInjectOrder)
		rr.Post("/v1/payloader/manifests/{manifestID}/seal", d.Service.HandleSealManifest)
		rr.Get("/v1/payloader/manifests/{manifestID}/ship-units", d.Service.HandleListShipUnits)
		rr.Post("/v1/payloader/manifests/{manifestID}/labels", d.Service.HandleManifestLabels)
		rr.Post("/v1/payload/manifest-exception", d.Service.HandleManifestException)
		rr.Get("/v1/payloader/manifest-exceptions", d.Service.HandleManifestExceptions)
		rr.Post("/v1/payloader/recommend-reassign", d.Service.HandleRecommendReassign)
		rr.Post("/v1/payloader/reassign-order", d.Service.HandleApplyReassign)
		rr.Post("/v1/payload/seal", d.Service.HandleSeal)
		rr.Get("/v1/payload/capacity/{vehicleID}", d.Service.HandleVehicleCapacity)
		rr.Get("/v1/payload/manifests/{manifestID}/ship-units", d.Service.HandleListShipUnits)
		rr.Post("/v1/payload/manifests/{manifestID}/labels", d.Service.HandleManifestLabels)
		rr.Post("/v1/fleet/reassign", d.Service.HandleFleetReassign)

		rr.Get("/v1/supplier/manifests", d.Service.HandleManifestsList)
		rr.Get("/v1/supplier/manifests/{id}", d.Service.HandleManifestDetail)
		rr.Post("/v1/supplier/manifests/seal-all", d.Service.HandleSealAll)
		rr.Post("/v1/supplier/manifests/{id}/start-loading", d.Service.HandleStartLoading)
		rr.Post("/v1/supplier/manifests/{id}/inject-order", d.Service.HandleInjectOrder)
		rr.Post("/v1/supplier/manifests/{id}/seal", d.Service.HandleSealManifest)
		rr.Get("/v1/supplier/manifests/{id}/ship-units", d.Service.HandleListShipUnits)
		rr.Post("/v1/supplier/manifests/{id}/labels", d.Service.HandleManifestLabels)

		rr.Get("/v1/user/notifications", d.Service.HandleUserNotifications)
		rr.Post("/v1/user/notifications/read", d.Service.HandleMarkNotificationsRead)

		if d.OrderService != nil {
			rr.Post("/v1/delivery/exception-report", d.OrderService.HandleExceptionReport)
		}
	}

	allowed := []auth.Role{auth.RolePayload, auth.RoleAdmin}
	if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
		r.Group(func(gr chi.Router) {
			gr.Use(auth.FirebaseAuth(d.FirebaseVerifier))
			gr.Use(auth.RequireRole(allowed...))
			mountProtected(gr)
		})
		return
	}

	r.Group(func(gr chi.Router) {
		gr.Use(auth.RequireRole(allowed...))
		mountProtected(gr)
	})
}
