// Package periodroutes mounts ATOMOS M1.1 Accounting Period Close & Snapshot
// HTTP endpoints onto chi.
package periodroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/periods"
)

// Deps is the narrow dependency contract for mounting M1.1 period routes.
type Deps struct {
	Service         *periods.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts /v1/finance/periods endpoints onto r.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	readPerm := auth.RequireAnySupplierPerm(
		auth.PermPeriodClose,
		auth.PermPeriodReopen,
		auth.PermJournalPost,
		auth.PermReconManage,
		auth.PermAuditRead,
	)

	mountPeriods := func(pr chi.Router) {
		pr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), readPerm).
			Get("/v1/finance/periods", d.Service.HandleListPeriods)
		pr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), auth.RequireSupplierPerm(auth.PermPeriodClose)).
			Post("/v1/finance/periods", d.Service.HandleOpenPeriod)
		pr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), auth.RequireSupplierPerm(auth.PermPeriodClose)).
			Post("/v1/finance/periods/{id}/soft-close", d.Service.HandleSoftClose)
		pr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), auth.RequireSupplierPerm(auth.PermPeriodClose)).
			Post("/v1/finance/periods/{id}/close", d.Service.HandleClose)
		pr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), auth.RequireSupplierPerm(auth.PermPeriodReopen)).
			Post("/v1/finance/periods/{id}/reopen", d.Service.HandleReopen)
		pr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), readPerm).
			Get("/v1/finance/periods/{id}/checklist", d.Service.HandleChecklist)
		pr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), readPerm).
			Get("/v1/finance/periods/{id}/snapshot", d.Service.HandleSnapshot)
	}

	if d.AllowAuthBypass {
		r.Group(mountPeriods)
		return
	}

	r.Group(func(pr chi.Router) {
		if d.JWTSecret != "" {
			pr.Use(auth.CookieAuth(d.JWTSecret))
		}
		pr.Use(auth.AttachTenantFromClaims)
		auth.ProtectMutations(pr, auth.MutationGuardConfig{
			AllowBypass: false,
		}, mountPeriods)
	})
}
