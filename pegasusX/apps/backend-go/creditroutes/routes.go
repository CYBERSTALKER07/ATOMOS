// Package creditroutes mounts retailer credit profile endpoints.
package creditroutes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
)

// Deps is the narrow dependency contract for credit routes.
type Deps struct {
	Service *credit.Service
}

// RegisterRoutes mounts credit profile endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	mount := func(gr chi.Router) {
		gr.With(auth.RequireRole(auth.RoleRetailer, auth.RoleAdmin)).Get("/v1/retailer/credit-profile", d.Service.HandleGetRetailerProfile)
		// Collections desk — supplier-scoped list (limit / freeze / freeze via PATCH).
		gr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/credit-profiles", d.Service.HandleListSupplierProfiles)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Patch("/v1/supplier/retailer-credit-profile", d.Service.HandleUpsertSupplierProfile)
	}

	auth.ProtectMutations(r, auth.MutationGuardConfig{}, mount)
}

// Middleware is a local alias for route middleware stacking.
type Middleware func(http.HandlerFunc) http.HandlerFunc
