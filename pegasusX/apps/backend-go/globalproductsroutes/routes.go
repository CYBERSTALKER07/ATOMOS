package globalproductsroutes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/globalproducts"
)

// Deps for route registration.
type Deps struct {
	Service             *globalproducts.Service
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
	AllowAuthBypass     bool
	// StepUp optional MFA middleware for PLATFORM_ADMIN on match-queue mutators (B5 M-P1-11).
	StepUp func(http.Handler) http.Handler
}

// RegisterRoutes mounts GlobalProducts endpoints when the service is present.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}
	auth.ProtectMutations(r, auth.MutationGuardConfig{
		FirebaseEnabled:  d.FirebaseAuthEnabled,
		FirebaseVerifier: d.FirebaseVerifier,
		AllowBypass:      d.AllowAuthBypass,
	}, func(gr chi.Router) {
		gr.Get("/v1/global-products/{id}", d.Service.HandleGetGlobal)
		gr.Get("/v1/global-products/{id}/offers", d.Service.HandleListOffers)
		gr.With(auth.RequireRole(auth.RoleAdmin)).
			Post("/v1/supplier/products/{productId}/link-global", d.Service.HandleLinkProduct)
		match := gr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin))
		if d.StepUp != nil {
			match = match.With(d.StepUp)
		}
		match.Get("/v1/admin/product-match-queue", d.Service.HandleListMatchQueue)
		match.Post("/v1/admin/product-match-queue/{id}/resolve", d.Service.HandleResolveMatch)
	})
}
