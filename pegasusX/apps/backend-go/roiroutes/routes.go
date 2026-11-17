// Package roiroutes mounts ATOMOS M1.8 ROI v1 HTTP endpoints onto chi.
package roiroutes

import (
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/roi"
)

// Deps holds runtime dependencies for mounting M1.8 ROI v1 endpoints.
type Deps struct {
	Service         *roi.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts /v1/roi/* routes onto r with tenant and RBAC guards.
func RegisterRoutes(r chi.Router, d Deps) {
	if r == nil || d.Service == nil {
		return
	}

	roleGuard := auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin)
	readPerm := auth.RequireAnySupplierPerm(
		auth.PermAuditRead,
		auth.PermReconManage,
		auth.PermConfigManage,
		auth.PermAgentProposalApprove,
		auth.PermJournalPost,
	)
	writePerm := auth.RequireAnySupplierPerm(
		auth.PermConfigManage,
		auth.PermReconManage,
		auth.PermAgentProposalApprove,
		auth.PermJournalPost,
	)

	mount := func(gr chi.Router) {
		gr.With(roleGuard, readPerm).Get("/v1/roi/baseline", d.Service.HandleGetBaseline)
		gr.With(roleGuard, writePerm).Post("/v1/roi/baseline", d.Service.HandleCaptureBaseline)
		gr.With(roleGuard, readPerm).Get("/v1/roi/summary", d.Service.HandleGetSummary)
		gr.With(roleGuard, readPerm).Get("/v1/roi/events", d.Service.HandleListEvents)
		gr.With(roleGuard, writePerm).Post("/v1/roi/events", d.Service.HandleRecordEvent)
		gr.With(roleGuard, readPerm).Get("/v1/roi/report", d.Service.HandleExportReport)
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
