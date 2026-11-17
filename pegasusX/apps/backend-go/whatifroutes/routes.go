// Package whatifroutes mounts ATOMOS §7.9 Full What-If Simulator E2E HTTP
// endpoints onto chi with active-auditor enforcement, AUDITOR/AGENT mutation
// blocks, and RBAC guards.
package whatifroutes

import (
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/whatif"
)

// Deps holds runtime dependencies for mounting What-If Simulator routes.
type Deps struct {
	Service         *whatif.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts /v1/whatif/* endpoints onto r.
func RegisterRoutes(r chi.Router, d Deps) {
	if r == nil || d.Service == nil {
		return
	}

	roleGuard := auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin)
	readPerm := auth.RequireAnySupplierPerm(
		auth.PermAuditRead,
		auth.PermReconManage,
		auth.PermAgentProposalApprove,
		auth.PermStockAdjustApprove,
		auth.PermConfigManage,
	)
	writePerm := auth.RequireAnySupplierPerm(
		auth.PermReconManage,
		auth.PermAgentProposalApprove,
		auth.PermStockAdjustApprove,
		auth.PermConfigManage,
	)
	noAuditorNoAgent := auth.RejectAuditorOrAgentMutation(true)

	mount := func(gr chi.Router) {
		gr.With(roleGuard, noAuditorNoAgent, writePerm).Post("/v1/whatif/simulate", d.Service.HandleSimulate)
		gr.With(roleGuard, readPerm).Get("/v1/whatif/simulations", d.Service.HandleListSimulations)
		gr.With(roleGuard, readPerm).Get("/v1/whatif/simulations/{simID}", d.Service.HandleGetSimulation)
		gr.With(roleGuard, noAuditorNoAgent, writePerm).Post("/v1/whatif/simulations/{simID}/commit-proposals", d.Service.HandleCommitProposals)
	}

	r.Group(func(gr chi.Router) {
		if strings.TrimSpace(d.JWTSecret) != "" {
			gr.Use(auth.CookieAuth(d.JWTSecret))
		}
		gr.Use(auth.RequireAuditorActive)
		gr.Use(auth.AttachTenantFromClaims)
		auth.ProtectMutations(gr, auth.MutationGuardConfig{
			AllowBypass: d.AllowAuthBypass,
		}, mount)
	})
}
