package creditnoteroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditnote"
)

type Deps struct {
	Handlers        *creditnote.Handlers
	AllowAuthBypass bool
}

func RegisterRoutes(r chi.Router, d Deps) {
	if d.Handlers == nil {
		return
	}
	h := d.Handlers
	auth.ProtectMutations(r, auth.MutationGuardConfig{
		AllowBypass: d.AllowAuthBypass,
	}, func(gr chi.Router) {
		gr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/credit-notes", h.HandleList)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/credit-notes", h.HandleCreateManual)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/credit-notes/{id}/issue", h.HandleIssue)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/credit-notes/order-lines", h.HandleOrderLines)
		// B7 WH-P0-3: WAREHOUSE + WAREHOUSE_ADMIN; handlers pin home-node (ops scope preferred).
		gr.With(auth.RequireRole(auth.RoleWarehouse, auth.RoleWarehouseAdmin), auth.RequireWarehouseOpsScope).
			Get("/v1/warehouse/reverse-logistics", h.HandleListReverseTasks)
		gr.With(auth.RequireRole(auth.RoleWarehouse, auth.RoleWarehouseAdmin), auth.RequireWarehouseOpsScope).
			Post("/v1/warehouse/reverse-logistics/{taskId}/receive", h.HandleReceiveReverse)
	})
}
