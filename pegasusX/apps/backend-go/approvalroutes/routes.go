// Package approvalroutes mounts the ATOMOS M1.2 CEL Approval Engine and Telegram
// approval webhook HTTP endpoints onto chi.
package approvalroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Deps is the narrow dependency contract for mounting approval routes.
type Deps struct {
	Service         *approvals.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts M1.2 approval policy, inbox, decision, delegation, and
// Telegram callback webhook endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	// Telegram approval webhook uses X-Telegram-Bot-Api-Secret-Token / X-Bot-Webhook-Secret / HMAC verification.
	r.Post("/v1/webhooks/telegram/approvals", d.Service.HandleTelegramWebhook)

	readPerm := auth.RequireAnySupplierPerm(
		auth.PermConfigManage,
		auth.PermJournalApprove,
		auth.PermDraftApprove,
		auth.PermCreditNoteApprove,
		auth.PermPeriodReopen,
		auth.PermPayoutRelease,
		auth.PermStockAdjustApprove,
		auth.PermWriteoffApprove,
		auth.PermAgentProposalApprove,
		auth.PermOrderOverLimitApprove,
		auth.PermAuditRead,
	)

	mountApprovals := func(ar chi.Router) {
		ar.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), readPerm).
			Get("/v1/approvals/policies", d.Service.HandleListPolicies)
		ar.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), auth.RequireSupplierPerm(auth.PermConfigManage)).
			Post("/v1/approvals/policies", d.Service.HandleCreatePolicy)
		ar.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), auth.RequireSupplierPerm(auth.PermConfigManage)).
			Put("/v1/approvals/policies/{id}", d.Service.HandleUpdatePolicy)
		ar.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin), auth.RequireSupplierPerm(auth.PermConfigManage)).
			Patch("/v1/approvals/policies/{id}", d.Service.HandleUpdatePolicy)

		ar.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin)).
			Post("/v1/approvals/requests", d.Service.HandleSubmitRequest)
		ar.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin), readPerm).
			Get("/v1/approvals/inbox", d.Service.HandleInbox)
		ar.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin), readPerm).
			Get("/v1/approvals/{id}", d.Service.HandleGetRequest)
		ar.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin)).
			Post("/v1/approvals/{id}/approve", d.Service.HandleApprove)
		ar.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin)).
			Post("/v1/approvals/{id}/reject", d.Service.HandleReject)

		ar.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin)).
			Get("/v1/approvals/delegations", d.Service.HandleListDelegations)
		ar.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin)).
			Post("/v1/approvals/delegations", d.Service.HandleCreateDelegation)
		ar.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin)).
			Post("/v1/approvals/delegations/{id}/revoke", d.Service.HandleRevokeDelegation)
	}

	if d.AllowAuthBypass {
		r.Group(mountApprovals)
		return
	}

	r.Group(func(ar chi.Router) {
		if d.JWTSecret != "" {
			ar.Use(auth.CookieAuth(d.JWTSecret))
		}
		ar.Use(auth.AttachTenantFromClaims)
		auth.ProtectMutations(ar, auth.MutationGuardConfig{
			AllowBypass: false,
		}, mountApprovals)
	})
}
