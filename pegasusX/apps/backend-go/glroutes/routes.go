// Package glroutes mounts the ATOMOS General Ledger HTTP endpoints onto chi.
package glroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
)

// Deps is the narrow dependency contract for mounting GL routes.
type Deps struct {
	Service         *gl.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts the M0.2 GL query, manual posting, reversal, and backfill routes.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	readPerm := auth.RequireAnySupplierPerm(
		auth.PermJournalPost,
		auth.PermJournalReverse,
		auth.PermJournalApprove,
		auth.PermReconManage,
		auth.PermAuditRead,
	)
	noAuditorNoAgent := auth.RejectAuditorOrAgentMutation(true)
	mountGL := func(gr chi.Router) {
		gr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), readPerm).
			Get("/v1/finance/gl/trial-balance", d.Service.HandleTrialBalance)
		gr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), readPerm).
			Get("/v1/finance/trial-balance", d.Service.HandleTrialBalance)
		gr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), readPerm).
			Get("/v1/finance/gl", d.Service.HandleListJournals)
		gr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), readPerm).
			Get("/v1/finance/gl/accounts/{code}/ledger", d.Service.HandleAccountLedger)
		gr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), readPerm).
			Get("/v1/finance/journal/{entryId}", d.Service.HandleGetJournalEntry)
		gr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), noAuditorNoAgent, auth.RequireSupplierPerm(auth.PermJournalPost)).
			Post("/v1/finance/journal/manual", d.Service.HandleManualJournal)
		gr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), noAuditorNoAgent, auth.RequireSupplierPerm(auth.PermJournalReverse)).
			Post("/v1/finance/journal/{id}/reverse", d.Service.HandleReverseJournal)
		gr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), noAuditorNoAgent, auth.RequireSupplierPerm(auth.PermJournalPost)).
			Post("/v1/finance/gl/backfill", d.Service.HandleBackfill)
	}

	if d.AllowAuthBypass {
		r.Group(mountGL)
		return
	}

	r.Group(func(gr chi.Router) {
		if d.JWTSecret != "" {
			gr.Use(auth.CookieAuth(d.JWTSecret))
		}
		gr.Use(auth.RequireAuditorActive)
		gr.Use(auth.AttachTenantFromClaims)
		auth.ProtectMutations(gr, auth.MutationGuardConfig{
			AllowBypass: false,
		}, mountGL)
	})
}
