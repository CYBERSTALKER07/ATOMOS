// Package anomalyroutes mounts ATOMOS §7.7 Anomaly & Fraud Detector Catalogue
// HTTP endpoints onto chi with active-auditor enforcement, AUDITOR/AGENT
// mutation blocks, and RBAC guards.
package anomalyroutes

import (
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/anomalies"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Deps holds runtime dependencies for mounting Anomaly & Fraud Catalogue routes.
type Deps struct {
	Service         *anomalies.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts /v1/anomalies/* endpoints onto r.
func RegisterRoutes(r chi.Router, d Deps) {
	if r == nil || d.Service == nil {
		return
	}

	roleGuard := auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin)
	readPerm := auth.RequireAnySupplierPerm(
		auth.PermAuditRead,
		auth.PermReconManage,
		auth.PermJournalApprove,
		auth.PermStockAdjustApprove,
		auth.PermConfigManage,
	)
	writePerm := auth.RequireAnySupplierPerm(
		auth.PermReconManage,
		auth.PermJournalApprove,
		auth.PermStockAdjustApprove,
		auth.PermAgentProposalApprove,
		auth.PermConfigManage,
	)
	noAuditorNoAgent := auth.RejectAuditorOrAgentMutation(true)

	mount := func(gr chi.Router) {
		gr.With(roleGuard, readPerm).Get("/v1/anomalies/catalogue", d.Service.HandleCatalogue)
		gr.With(roleGuard, noAuditorNoAgent, writePerm).Post("/v1/anomalies/scan", d.Service.HandleScan)
		gr.With(roleGuard, readPerm).Get("/v1/anomalies/cases", d.Service.HandleListCases)
		gr.With(roleGuard, noAuditorNoAgent, writePerm).Post("/v1/anomalies/cases/{caseID}/resolve", d.Service.HandleResolveCase)
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
