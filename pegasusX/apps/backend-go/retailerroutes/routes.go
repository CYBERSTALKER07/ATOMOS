// Package retailerroutes mounts the retailer-facing URL surface onto the chi
// router. Handlers live in the retailer package; this file is thin by design.
package retailerroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
)

// Deps is the narrow dependency contract for this routes package.
type Deps struct {
	Service             *retailer.Service
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
}

// RegisterRoutes mounts the retailer role-row surface.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	r.Post("/v1/auth/retailer/register", d.Service.HandleRegister)

	mountProtected := func(rr chi.Router) {
		rr.Get("/v1/retailer/profile", d.Service.HandleProfile)
		rr.Put("/v1/retailer/profile", d.Service.HandleProfile)

		rr.Get("/v1/retailer/suppliers", d.Service.HandleSuppliers)
		rr.Post("/v1/retailer/suppliers/{supplierID}/{action}", d.Service.HandleSuppliers)

		rr.Get("/v1/retailer/cart/sync", d.Service.HandleCartSync)
		rr.Post("/v1/retailer/cart/sync", d.Service.HandleCartSync)

		rr.Get("/v1/retailers/{retailerID}/orders", d.Service.HandleOrders)

		rr.Post("/v1/orders/request-cancel", d.Service.HandleRequestCancel)
		rr.Post("/v1/order/cancel", d.Service.HandleCancelOrder)

		rr.Get("/v1/retailer/analytics/expenses", d.Service.HandleExpensesAnalytics)
		rr.Get("/v1/retailer/analytics/detailed", d.Service.HandleDetailedAnalytics)

		rr.Get("/v1/retailer/family-members", d.Service.HandleFamilyMembers)
		rr.Post("/v1/retailer/family-members", d.Service.HandleFamilyMembers)
		rr.Delete("/v1/retailer/family-members/{memberID}", d.Service.HandleDeleteFamilyMember)

		rr.Post("/v1/retailer/shop-closed-response", d.Service.HandleShopClosedResponse)
		rr.Post("/v1/retailer/orders/confirm-ai", d.Service.HandleConfirmAIOrder)
		rr.Post("/v1/retailer/orders/reject-ai", d.Service.HandleRejectAIOrder)
		rr.Post("/v1/orders/edit-preorder", d.Service.HandleEditPreorder)
		rr.Post("/v1/orders/confirm-preorder", d.Service.HandleConfirmPreorder)

		rr.Get("/v1/retailer/pending-payments", d.Service.HandlePendingPayments)
		rr.Get("/v1/retailer/active-fulfillment", d.Service.HandleActiveFulfillment)
		rr.Get("/v1/retailer/tracking", d.Service.HandleTracking)
	}

	if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
		r.Group(func(gr chi.Router) {
			gr.Use(auth.FirebaseAuth(d.FirebaseVerifier))
			gr.Use(auth.RequireRole(auth.RoleRetailer))
			mountProtected(gr)
		})
		return
	}

	// Local scaffold fallback when Firebase is disabled.
	mountProtected(r)
}
