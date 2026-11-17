// Package auditroutes mounts the ATOMOS M0.4 immutable audit query, chain
// verification, WORM anchor, and legal-hold endpoints onto chi.
package auditroutes

import (
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Deps is the narrow dependency contract for mounting audit routes.
type Deps struct {
	Service         *audit.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts the M0.4 audit query, verification, anchor, and legal-hold routes.
func RegisterRoutes(r chi.Router, d Deps) {
	if r == nil || d.Service == nil {
		return
	}

	roleGuard := auth.RequireRole(
		auth.RoleAdmin,
		auth.RolePlatformAdmin,
		auth.RoleWarehouseAdmin,
		auth.RoleWarehouse,
		auth.RoleFactoryAdmin,
		auth.RoleFactory,
		auth.RolePayload,
	)
	readGuard := auth.RequireAnySupplierPerm(auth.PermAuditRead, auth.PermAuditManage)
	manageGuard := auth.RequireSupplierPerm(auth.PermAuditManage)

	mount := func(gr chi.Router) {
		gr.With(roleGuard, readGuard).Get("/v1/audit/events", d.Service.HandleListEvents)
		gr.With(roleGuard, readGuard).Get("/v1/audit/history/{objectType}/{objectId}", d.Service.HandleObjectHistory)
		gr.With(roleGuard, readGuard).Get("/v1/audit/verify", d.Service.HandleVerifyDayChain)
		gr.With(roleGuard, readGuard).Get("/v1/audit/anchors", d.Service.HandleListAnchors)
		gr.With(roleGuard, manageGuard).Post("/v1/audit/anchors", d.Service.HandleWriteAnchor)
		gr.With(roleGuard, manageGuard).Post("/v1/audit/legal-hold", d.Service.HandleCreateLegalHold)
		gr.With(roleGuard, readGuard).Get("/v1/audit/legal-holds", d.Service.HandleListLegalHolds)
	}

	if d.AllowAuthBypass {
		r.Group(mount)
		return
	}

	r.Group(func(gr chi.Router) {
		if strings.TrimSpace(d.JWTSecret) != "" {
			gr.Use(auth.CookieAuth(d.JWTSecret))
		}
		gr.Use(auth.AttachTenantFromClaims)
		auth.ProtectMutations(gr, auth.MutationGuardConfig{
			AllowBypass: false,
		}, mount)
	})
}
