// Package p2proutes mounts ATOMOS §6.6 Procure-to-Pay, UZ process templates,
// 3-Way Match, and Process Mining HTTP endpoints onto chi.
package p2proutes

import (
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/p2p"
)

// Deps holds runtime dependencies for mounting P2P and Process Mining routes.
type Deps struct {
	Service         *p2p.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts /v1/processes/* and /v1/p2p/* endpoints onto r with
// active-auditor enforcement, AUDITOR/AGENT mutation blocks, and RBAC checks.
func RegisterRoutes(r chi.Router, d Deps) {
	if r == nil || d.Service == nil {
		return
	}

	roleGuard := auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin)
	readPerm := auth.RequireAnySupplierPerm(
		auth.PermAuditRead,
		auth.PermJournalPost,
		auth.PermJournalApprove,
		auth.PermReconManage,
		auth.PermStockAdjustApprove,
		auth.PermConfigManage,
	)
	writePerm := auth.RequireAnySupplierPerm(
		auth.PermJournalPost,
		auth.PermJournalApprove,
		auth.PermReconManage,
		auth.PermStockAdjustApprove,
		auth.PermConfigManage,
	)
	approvePerm := auth.RequireAnySupplierPerm(
		auth.PermJournalApprove,
		auth.PermStockAdjustApprove,
		auth.PermConfigManage,
	)
	noAuditorNoAgent := auth.RejectAuditorOrAgentMutation(true)

	mount := func(gr chi.Router) {
		// Process Templates, Instances & Process Mining
		gr.With(roleGuard, readPerm).Get("/v1/processes/templates", d.Service.HandleListTemplates)
		gr.With(roleGuard, noAuditorNoAgent, writePerm).Post("/v1/processes/templates", d.Service.HandleCreateTemplate)
		gr.With(roleGuard, readPerm).Get("/v1/processes/instances", d.Service.HandleListInstances)
		gr.With(roleGuard, noAuditorNoAgent, writePerm).Post("/v1/processes/instances", d.Service.HandleUpsertInstance)
		gr.With(roleGuard, readPerm).Get("/v1/processes/mining", d.Service.HandleProcessMining)

		// Procure-to-Pay (POs, Goods Receipts, Vendor Invoices, 3-Way Match)
		gr.With(roleGuard, readPerm).Get("/v1/p2p/purchase-orders", d.Service.HandleListPurchaseOrders)
		gr.With(roleGuard, noAuditorNoAgent, writePerm).Post("/v1/p2p/purchase-orders", d.Service.HandleCreatePurchaseOrder)
		gr.With(roleGuard, readPerm).Get("/v1/p2p/purchase-orders/{poID}", d.Service.HandleGetPurchaseOrder)
		gr.With(roleGuard, noAuditorNoAgent, approvePerm).Post("/v1/p2p/purchase-orders/{poID}/approve", d.Service.HandleApprovePurchaseOrder)

		gr.With(roleGuard, readPerm).Get("/v1/p2p/receipts", d.Service.HandleListReceipts)
		gr.With(roleGuard, noAuditorNoAgent, writePerm).Post("/v1/p2p/receipts", d.Service.HandlePostReceipt)

		gr.With(roleGuard, readPerm).Get("/v1/p2p/vendor-invoices", d.Service.HandleListVendorInvoices)
		gr.With(roleGuard, noAuditorNoAgent, writePerm).Post("/v1/p2p/vendor-invoices", d.Service.HandleCreateVendorInvoice)

		gr.With(roleGuard, noAuditorNoAgent, writePerm).Post("/v1/p2p/three-way-match", d.Service.HandleThreeWayMatch)
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
