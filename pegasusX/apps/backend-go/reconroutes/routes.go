// Package reconroutes mounts the ATOMOS M0.3 four-way reconciliation HTTP endpoints onto chi.
package reconroutes

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/recon"
)

// Deps holds runtime dependencies required by the reconciliation route group.
type Deps struct {
	Service         *recon.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts the M0.3 four-way reconciliation endpoints onto r.
func RegisterRoutes(r chi.Router, d Deps) {
	if r == nil || d.Service == nil {
		return
	}

	roleGuard := auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin)
	readPerm := auth.RequireAnySupplierPerm(auth.PermReconManage, auth.PermAuditRead)
	writePerm := auth.RequireSupplierPerm(auth.PermReconManage)
	mount := func(gr chi.Router) {
		gr.With(roleGuard, readPerm).Get("/v1/finance/recon/runs", d.Service.HandleListRuns)
		gr.With(roleGuard, writePerm).Post("/v1/finance/recon/run", d.Service.HandlePostRun)
		gr.With(roleGuard, readPerm).Get("/v1/finance/recon/breaks", d.Service.HandleListBreaks)
		gr.With(roleGuard, writePerm).Patch("/v1/finance/recon/breaks/{id}", d.Service.HandlePatchBreak)
	}

	r.Group(func(gr chi.Router) {
		if strings.TrimSpace(d.JWTSecret) != "" {
			gr.Use(auth.CookieAuth(d.JWTSecret))
		}
		gr.Use(auth.AttachTenantFromClaims)
		auth.ProtectMutations(gr, auth.MutationGuardConfig{
			AllowBypass: d.AllowAuthBypass,
		}, mount)
	})
}

// Verify chi.Router route helper compiles cleanly across adapters.
var _ http.HandlerFunc
