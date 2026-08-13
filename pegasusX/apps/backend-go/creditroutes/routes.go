// Package creditroutes mounts retailer credit profile and policy endpoints.
package creditroutes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
)

// Deps is the narrow dependency contract for credit routes.
type Deps struct {
	Service       *credit.Service
	PolicyService *credit.PolicyService
	ARService     *ar.Service
	DunningWorker *ar.DunningWorker
	// StepUp optional MFA middleware for PLATFORM_ADMIN on dunning run-once (B5 M-P1-11).
	StepUp func(http.Handler) http.Handler
}

// RegisterRoutes mounts credit profile + policy + AR endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil && d.PolicyService == nil {
		return
	}

	mount := func(gr chi.Router) {
		if d.Service != nil {
			gr.With(auth.RequireRole(auth.RoleRetailer, auth.RoleAdmin)).Get("/v1/retailer/credit-profile", d.Service.HandleGetRetailerProfile)
			gr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/credit-profiles", d.Service.HandleListSupplierProfiles)
			gr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/credit-scores", d.Service.HandleGetScores)
			gr.With(auth.RequireRole(auth.RoleAdmin)).Patch("/v1/supplier/retailer-credit-profile", d.Service.HandleUpsertSupplierProfile)
		}
		if d.PolicyService != nil {
			ps := d.PolicyService
			finance := []auth.Role{auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleWarehouse}
			gr.With(auth.RequireRole(finance...)).Get("/v1/supplier/credit-program", ps.HandleGetCreditProgram)
			gr.With(auth.RequireRole(finance...)).Post("/v1/supplier/credit-program", ps.HandleEnableCreditProgram)
			gr.With(auth.RequireRole(finance...)).Get("/v1/supplier/credit-program/defaults", ps.HandleCreditProgramDefaults)
			gr.With(auth.RequireRole(finance...)).Patch("/v1/supplier/credit-program/defaults", ps.HandleCreditProgramDefaults)
			gr.With(auth.RequireRole(finance...)).Get("/v1/supplier/credit-relationships", ps.HandleListCreditRelationships)
			gr.With(auth.RequireRole(finance...)).Post("/v1/supplier/credit-relationships/{retailerId}/enable", ps.HandleEnableCreditRelationship)
			gr.With(auth.RequireRole(finance...)).Patch("/v1/supplier/credit-relationships/{retailerId}/terms", ps.HandlePatchCreditRelationshipTerms)
			gr.With(auth.RequireRole(finance...)).Post("/v1/supplier/credit-relationships/{retailerId}/hold", ps.HandleHoldCreditRelationship)
			gr.With(auth.RequireRole(finance...)).Post("/v1/supplier/credit-relationships/{retailerId}/unhold", ps.HandleUnholdCreditRelationship)
			gr.With(auth.RequireRole(finance...)).Post("/v1/supplier/credit-relationships/{retailerId}/disable", ps.HandleSelfServeDisableCreditRelationship)
			gr.With(auth.RequireRole(auth.RoleRetailer, auth.RoleAdmin)).Get("/v1/retailer/credit-relationships", ps.HandleListRetailerCreditRelationships)
			gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/admin/credit-relationships/{supplierId}/{retailerId}/disable", ps.HandleAdminDisableRelationship)
			gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/admin/credit-program/{supplierId}/disable", ps.HandleAdminDisableProgram)
		}
		if d.ARService != nil {
			gr.With(auth.RequireRole(auth.RoleRetailer, auth.RoleAdmin)).Get("/v1/retailer/ar/invoices", d.ARService.HandleListRetailerInvoices)
			gr.With(auth.RequireRole(auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleWarehouse)).Get("/v1/supplier/ar/invoices", d.ARService.HandleListSupplierInvoices)
		}
		if d.DunningWorker != nil {
			dunning := gr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin))
			if d.StepUp != nil {
				dunning = dunning.With(d.StepUp)
			}
			dunning.Post("/v1/admin/ar/dunning/run-once", d.DunningWorker.HandleRunDunningOnce)
			dunning.Get("/v1/admin/ar/dunning/status", d.DunningWorker.HandleDunningStatus)
		}
	}

	auth.ProtectMutations(r, auth.MutationGuardConfig{}, mount)
}

// Middleware is a local alias for route middleware stacking.
type Middleware func(http.HandlerFunc) http.HandlerFunc
