package globalproductsroutes

import (
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
		gr.With(auth.RequireRole(auth.RoleAdmin)).
			Get("/v1/admin/product-match-queue", d.Service.HandleListMatchQueue)
		gr.With(auth.RequireRole(auth.RoleAdmin)).
			Post("/v1/admin/product-match-queue/{id}/resolve", d.Service.HandleResolveMatch)
	})
}
