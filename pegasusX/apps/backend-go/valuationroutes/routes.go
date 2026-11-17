// Package valuationroutes mounts ATOMOS M1.3 inventory valuation & cost layer HTTP endpoints onto chi.
package valuationroutes

import (
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/valuation"
)

// Deps holds runtime dependencies for mounting inventory valuation endpoints.
type Deps struct {
	Service         *valuation.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts the M1.3 inventory valuation, movement, and cost-layer receipt routes.
func RegisterRoutes(r chi.Router, d Deps) {
	if r == nil || d.Service == nil {
		return
	}

	roleGuard := auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin, auth.RoleWarehouse)
	readPerm := auth.RequireAnySupplierPerm(
		auth.PermReconManage,
		auth.PermAuditRead,
		auth.PermStockAdjustApprove,
		auth.PermSupplierStockCount,
		auth.PermJournalPost,
	)
	writePerm := auth.RequireAnySupplierPerm(
		auth.PermJournalPost,
		auth.PermStockAdjustApprove,
		auth.PermSupplierStockCount,
	)

	mount := func(gr chi.Router) {
		gr.With(roleGuard, readPerm).Get("/v1/inventory/valuation", d.Service.HandleGetValuation)
		gr.With(roleGuard, readPerm).Get("/v1/inventory/movements", d.Service.HandleListMovements)
		gr.With(roleGuard, writePerm).Post("/v1/inventory/cost-layers/receipt", d.Service.HandleRecordReceipt)
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
