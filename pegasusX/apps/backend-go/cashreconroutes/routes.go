// Package cashreconroutes mounts driver cash reconciliation endpoints.
package cashreconroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cashrecon"
)

type Deps struct {
	Handlers        *cashrecon.Handlers
	AllowAuthBypass bool
}

func RegisterRoutes(r chi.Router, d Deps) {
	if d.Handlers == nil {
		return
	}
	h := d.Handlers
	mountDriver := func(gr chi.Router) {
		gr.Post("/v1/driver/cash-reconciliations", h.HandleDriverSubmit)
		gr.Get("/v1/driver/cash-reconciliations", h.HandleDriverList)
	}
	mountSupplier := func(gr chi.Router) {
		gr.Get("/v1/supplier/cash-reconciliations", h.HandleSupplierList)
		gr.Post("/v1/supplier/cash-reconciliations/{id}/accept", h.HandleSupplierAccept)
		gr.Post("/v1/supplier/cash-reconciliations/{id}/write-off", h.HandleSupplierWriteOff)
	}
	auth.ProtectMutations(r, auth.MutationGuardConfig{
		AllowBypass: d.AllowAuthBypass,
	}, func(gr chi.Router) {
		gr.With(auth.RequireRole(auth.RoleDriver)).Group(mountDriver)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Group(mountSupplier)
	})
}
