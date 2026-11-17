// Package proposalroutes mounts ATOMOS M1.7 Governed Proposals Inbox and L1 Agent endpoints onto chi.
package proposalroutes

import (
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposals"
)

// Deps holds runtime dependencies for mounting M1.7 proposals routes.
type Deps struct {
	Service         *proposals.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts /v1/proposals/* endpoints onto r with tenant and RBAC guards.
func RegisterRoutes(r chi.Router, d Deps) {
	if r == nil || d.Service == nil {
		return
	}

	roleGuard := auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin)
	readPerm := auth.RequireAnySupplierPerm(
		auth.PermAuditRead,
		auth.PermAgentProposalApprove,
		auth.PermConfigManage,
		auth.PermReconManage,
		auth.PermStockAdjustApprove,
	)
	approvePerm := auth.RequireSupplierPerm(auth.PermAgentProposalApprove)

	mount := func(gr chi.Router) {
		gr.With(roleGuard, readPerm).Get("/v1/proposals", d.Service.HandleList)
		gr.With(roleGuard).Post("/v1/proposals", d.Service.HandleCreate)
		gr.With(roleGuard).Post("/v1/proposals/agents/{agentId}/run", d.Service.HandleRunAgent)
		gr.With(roleGuard, readPerm).Get("/v1/proposals/l2-evaluate", d.Service.HandleEvaluateL2)
		gr.With(roleGuard, approvePerm, auth.RejectAuditorOrAgentMutation(true)).Post("/v1/proposals/l2-evaluate", d.Service.HandleEvaluateL2)
		gr.With(roleGuard, readPerm).Get("/v1/proposals/{id}", d.Service.HandleGet)
		gr.With(roleGuard, approvePerm).Post("/v1/proposals/{id}/approve", d.Service.HandleApprove)
		gr.With(roleGuard, approvePerm).Post("/v1/proposals/{id}/reject", d.Service.HandleReject)
		gr.With(roleGuard, approvePerm).Post("/v1/proposals/{id}/execute", d.Service.HandleExecute)
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
