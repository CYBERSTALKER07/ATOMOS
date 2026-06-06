// Package orderroutes mounts the order URL surface onto chi. Handlers live in
// the order package; this file is thin by design.
package orderroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

// Deps is the narrow dependency contract for this routes package.
type Deps struct {
	Service             *order.Service
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
	AllowAuthBypass     bool
}

// RegisterRoutes mounts the order endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	mount := func(gr chi.Router) {
		gr.With(auth.RequireRole(auth.RoleRetailer)).Post("/v1/order/create", d.Service.HandleCreate)
		gr.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer)).Patch("/v1/order/{orderID}/status", d.Service.HandleUpdateStatus)
		gr.With(auth.RequireRole(auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleFactoryAdmin)).Post("/v1/orders/{orderID}/assign", d.Service.HandleAssignOrder)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/delivery/arrive", d.Service.HandleMarkArrived)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/order/deliver", d.Service.HandleSubmitDelivery)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/order/confirm-offload", d.Service.HandleConfirmOffload)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/order/complete", d.Service.HandleCompleteOrder)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/order/collect-cash", d.Service.HandleCollectCash)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/shop-closed/resolve", d.Service.HandleResolveShopClosed)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/fleet/route/request-early-complete", d.Service.HandleRequestEarlyComplete)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/delivery/confirm-payment-bypass", d.Service.HandleConfirmPaymentBypass)
	}

	auth.ProtectMutations(r, auth.MutationGuardConfig{
		FirebaseEnabled:  d.FirebaseAuthEnabled,
		FirebaseVerifier: d.FirebaseVerifier,
		AllowBypass:      d.AllowAuthBypass,
	}, mount)
}
