// Package multientityroutes mounts ATOMOS §6.4 multi-entity, intercompany transfer,
// FX revaluation, and consolidation HTTP endpoints onto chi.
package multientityroutes

import (
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/multientity"
)

// Deps holds runtime dependencies for mounting §6.4 multientity routes.
type Deps struct {
	Service         *multientity.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts /v1/org/entities*, /v1/finance/intercompany-transfers,
// /v1/finance/fx-revaluation, and /v1/finance/consolidation onto r.
func RegisterRoutes(r chi.Router, d Deps) {
	if r == nil || d.Service == nil {
		return
	}

	roleGuard := auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin)
	readPerm := auth.RequireAnySupplierPerm(
		auth.PermAuditRead,
		auth.PermJournalPost,
		auth.PermJournalApprove,
		auth.PermReconManage,
		auth.PermConfigManage,
		auth.PermIAMManage,
		auth.PermPeriodClose,
	)
	entityWritePerm := auth.RequireAnySupplierPerm(
		auth.PermConfigManage,
		auth.PermIAMManage,
	)
	financeWritePerm := auth.RequireAnySupplierPerm(
		auth.PermJournalPost,
		auth.PermJournalApprove,
		auth.PermPeriodClose,
		auth.PermConfigManage,
	)
	noAuditorNoAgent := auth.RejectAuditorOrAgentMutation(true)

	mount := func(gr chi.Router) {
		gr.With(roleGuard, readPerm).Get("/v1/org/entities", d.Service.HandleListEntities)
		gr.With(roleGuard, noAuditorNoAgent, entityWritePerm).Post("/v1/org/entities", d.Service.HandleCreateEntity)
		gr.With(roleGuard, noAuditorNoAgent, entityWritePerm).Patch("/v1/org/entities/{entityID}", d.Service.HandlePatchEntity)

		gr.With(roleGuard, readPerm).Get("/v1/finance/intercompany-transfers", d.Service.HandleListTransfers)
		gr.With(roleGuard, noAuditorNoAgent, financeWritePerm).Post("/v1/finance/intercompany-transfers", d.Service.HandleCreateTransfer)

		gr.With(roleGuard, readPerm).Get("/v1/finance/fx-revaluation", d.Service.HandleListFxRevaluations)
		gr.With(roleGuard, noAuditorNoAgent, financeWritePerm).Post("/v1/finance/fx-revaluation", d.Service.HandleFxRevaluation)

		gr.With(roleGuard, readPerm).Get("/v1/finance/consolidation", d.Service.HandleConsolidation)
	}

	if d.AllowAuthBypass {
		r.Group(mount)
		return
	}

	r.Group(func(gr chi.Router) {
		if strings.TrimSpace(d.JWTSecret) != "" {
			gr.Use(auth.CookieAuth(d.JWTSecret))
		}
		gr.Use(auth.RequireAuditorActive)
		gr.Use(auth.AttachTenantFromClaims)
		auth.ProtectMutations(gr, auth.MutationGuardConfig{
			AllowBypass: false,
		}, mount)
	})
}
