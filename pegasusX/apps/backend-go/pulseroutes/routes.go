// Package pulseroutes mounts role-scoped network pulse endpoints.
package pulseroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/pulse"
)

// Deps is the narrow dependency contract for pulse routes.
type Deps struct {
	Handlers        *pulse.Handlers
	AllowAuthBypass bool
}

// RegisterRoutes mounts GET /v1/*/pulse behind the standard auth guard.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Handlers == nil {
		return
	}
	h := d.Handlers

	mount := func(gr chi.Router) {
		gr.Group(func(rr chi.Router) {
			rr.Use(auth.RequireRole(auth.RoleRetailer))
			rr.Get("/v1/retailer/pulse", h.HandleRetailerPulse)
		})
		gr.Group(func(rr chi.Router) {
			rr.Use(auth.RequireRole(auth.RoleAdmin))
			rr.Get("/v1/supplier/pulse", h.HandleSupplierPulse)
		})
		gr.Group(func(rr chi.Router) {
			rr.Use(auth.RequireRole(auth.RoleWarehouse, auth.RoleWarehouseAdmin))
			rr.Get("/v1/warehouse/ops/pulse", h.HandleWarehousePulse)
		})
		gr.Group(func(rr chi.Router) {
			rr.Use(auth.RequireRole(auth.RoleDriver, auth.RoleFactoryDriver))
			rr.Get("/v1/driver/pulse", h.HandleDriverPulse)
		})
		gr.Group(func(rr chi.Router) {
			rr.Use(auth.RequireRole(auth.RolePayload))
			rr.Get("/v1/payloader/pulse", h.HandlePayloaderPulse)
		})
		gr.Group(func(rr chi.Router) {
			rr.Use(auth.RequireRole(auth.RoleFactoryAdmin, auth.RoleFactory))
			rr.Get("/v1/factory/pulse", h.HandleFactoryPulse)
		})
	}

	auth.ProtectMutations(r, auth.MutationGuardConfig{
		AllowBypass: d.AllowAuthBypass,
	}, mount)
}
