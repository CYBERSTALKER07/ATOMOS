// Package connectorroutes mounts ATOMOS M1.9 Enterprise Connector Spine endpoints onto chi.
package connectorroutes

import (
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/connectors"
)

// Deps holds runtime dependencies for mounting M1.9 connector routes.
type Deps struct {
	Service         *connectors.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts /v1/connectors/* endpoints onto r with tenant and RBAC guards.
func RegisterRoutes(r chi.Router, d Deps) {
	if r == nil || d.Service == nil {
		return
	}

	roleGuard := auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin)
	readPerm := auth.RequireAnySupplierPerm(
		auth.PermAuditRead,
		auth.PermConfigManage,
		auth.PermReconManage,
		auth.PermStockAdjustApprove,
	)
	configPerm := auth.RequireSupplierPerm(auth.PermConfigManage)
	syncPerm := auth.RequireAnySupplierPerm(
		auth.PermConfigManage,
		auth.PermReconManage,
		auth.PermStockAdjustApprove,
	)
	pushPerm := auth.RequireAnySupplierPerm(
		auth.PermConfigManage,
		auth.PermJournalPost,
	)
	noAuditorNoAgent := auth.RejectAuditorOrAgentMutation(true)

	mount := func(gr chi.Router) {
		gr.With(roleGuard, readPerm).Get("/v1/connectors", d.Service.HandleListConnectors)
		gr.With(roleGuard, noAuditorNoAgent, configPerm).Put("/v1/connectors/credentials", d.Service.HandleUpsertCredential)
		gr.With(roleGuard, noAuditorNoAgent, syncPerm).Post("/v1/connectors/{provider}/health", d.Service.HandleHealth)
		gr.With(roleGuard, noAuditorNoAgent, syncPerm).Post("/v1/connectors/{provider}/pull", d.Service.HandlePull)
		gr.With(roleGuard, noAuditorNoAgent, pushPerm).Post("/v1/connectors/{provider}/push", d.Service.HandlePush)
		gr.With(roleGuard, noAuditorNoAgent, syncPerm).Post("/v1/connectors/{provider}/shadow-compare", d.Service.HandleShadowCompare)
		gr.With(roleGuard, readPerm).Get("/v1/connectors/runs", d.Service.HandleListRuns)
		gr.With(roleGuard, readPerm).Get("/v1/connectors/runs/{runID}", d.Service.HandleGetRun)
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
