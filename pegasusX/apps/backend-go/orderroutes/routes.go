// Package orderroutes mounts the order URL surface onto chi. Handlers live in
// the order package; this file is thin by design.
package orderroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

// Deps is the narrow dependency contract for this routes package.
type Deps struct {
	Service             *order.Service
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
}

// RegisterRoutes mounts the order endpoints. Auth middleware (RETAILER role)
// is composed at bootstrap and wraps the handler outside this function.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
		r.Group(func(gr chi.Router) {
			gr.Use(auth.FirebaseAuth(d.FirebaseVerifier))
			gr.With(auth.RequireRole(auth.RoleRetailer)).Post("/v1/order/create", d.Service.HandleCreate)
			gr.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer)).Patch("/v1/order/{orderID}/status", d.Service.HandleUpdateStatus)
		})
		return
	}

	// Backward-compatible fallback for early local scaffold runs. The handler
	// still returns 401 when no claims are present.
	r.Post("/v1/order/create", d.Service.HandleCreate)
	r.Patch("/v1/order/{orderID}/status", d.Service.HandleUpdateStatus)
}
