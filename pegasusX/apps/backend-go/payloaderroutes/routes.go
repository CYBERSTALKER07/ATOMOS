package payloaderroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/payload"
)

// Deps is the narrow dependency contract for payload routes.
type Deps struct {
	Service             *payload.Service
	JWTSecret           string
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
}

// RegisterRoutes mounts payload role-row operational endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	mountProtected := func(rr chi.Router) {
		rr.Get("/v1/payloader/trucks", d.Service.HandleTrucks)
		rr.Get("/v1/payloader/orders", d.Service.HandleOrders)
		rr.Get("/v1/payloader/manifests", d.Service.HandleManifests)
		rr.Get("/v1/payloader/manifests/{manifestID}", d.Service.HandleManifestDetail)
		rr.Post("/v1/payloader/manifests/{manifestID}/start-loading", d.Service.HandleStartLoading)
		rr.Post("/v1/payloader/manifests/{manifestID}/inject-order", d.Service.HandleInjectOrder)
		rr.Post("/v1/payloader/manifests/{manifestID}/seal", d.Service.HandleSealManifest)
		rr.Post("/v1/payload/manifest-exception", d.Service.HandleManifestException)
		rr.Get("/v1/payloader/manifest-exceptions", d.Service.HandleManifestExceptions)
		rr.Post("/v1/payloader/recommend-reassign", d.Service.HandleRecommendReassign)
		rr.Post("/v1/payloader/reassign-order", d.Service.HandleApplyReassign)
		rr.Post("/v1/payload/seal", d.Service.HandleSeal)
	}

	if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
		r.Group(func(gr chi.Router) {
			gr.Use(auth.FirebaseAuth(d.FirebaseVerifier))
			gr.Use(auth.RequireRole(auth.RolePayload, auth.RoleAdmin))
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
