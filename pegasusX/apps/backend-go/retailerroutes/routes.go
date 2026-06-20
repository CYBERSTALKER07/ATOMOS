// Package retailerroutes mounts the retailer-facing URL surface onto the chi
// router. Handlers live in the retailer package; this file is thin by design.
package retailerroutes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
	"github.com/pegasusx/pegasusx/apps/backend-go/promotion"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
)

// Deps is the narrow dependency contract for this routes package.
type Deps struct {
	Service          *retailer.Service
	PaymentService   *payment.Service
	PromotionService *promotion.Service
	OrderService interface {
		HandleShopClosedResponse(http.ResponseWriter, *http.Request)
		HandleRetailerCancel(http.ResponseWriter, *http.Request)
		HandleRetailerRequestCancel(http.ResponseWriter, *http.Request)
	}
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
	AllowAuthBypass     bool
}

// RegisterRoutes mounts the retailer role-row surface.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	r.Post("/v1/auth/retailer/login", d.Service.HandleRetailerLogin)
	r.Post("/v1/auth/retailer/refresh", d.Service.HandleRetailerRefresh)
	r.Post("/v1/auth/retailer/register", d.Service.HandleMobileRegister)

	mountProtected := func(rr chi.Router) {
		rr.Post("/v1/retailer/setup", d.Service.HandleRetailerSetup)
		rr.Get("/v1/retailer/profile", d.Service.HandleProfile)
		rr.Put("/v1/retailer/profile", d.Service.HandleProfile)

		rr.Get("/v1/retailer/suppliers", d.Service.HandleSuppliers)
		rr.Post("/v1/retailer/suppliers/{supplierID}/{action}", d.Service.HandleSuppliers)
		rr.Post("/v1/retailer/suppliers/{supplierID}/add", d.Service.HandleSupplierAdd)
		rr.Post("/v1/retailer/suppliers/{supplierID}/remove", d.Service.HandleSupplierRemove)
		rr.Get("/v1/retailer/pricing/rules", d.Service.HandlePricingRule)

		rr.Get("/v1/retailer/cart/sync", d.Service.HandleCartSync)
		rr.Post("/v1/retailer/cart/sync", d.Service.HandleCartSync)
		if d.PromotionService != nil {
			rr.Post("/v1/retailer/checkout/quote", d.PromotionService.HandleCheckoutQuote)
			rr.Post("/v1/retailer/promotions/watch", d.PromotionService.HandleWatchSupplierPromotions)
		}

		rr.Get("/v1/retailers/{retailerID}/orders", d.Service.HandleOrders)
		rr.Get("/v1/orders", d.Service.HandleOrdersAlias)

		if d.OrderService != nil {
			rr.Post("/v1/orders/request-cancel", d.OrderService.HandleRetailerRequestCancel)
		} else {
			rr.Post("/v1/orders/request-cancel", d.Service.HandleRequestCancel)
		}
		if d.OrderService != nil {
			rr.Post("/v1/order/cancel", d.OrderService.HandleRetailerCancel)
		} else {
			rr.Post("/v1/order/cancel", d.Service.HandleCancelOrder)
		}
		// POST /v1/order/create is served by orderroutes (order.Service.HandleCreate)
		// with Spanner persistence + outbox. Do NOT register a stub here.

		rr.Get("/v1/retailer/analytics/expenses", d.Service.HandleExpensesAnalytics)
		rr.Get("/v1/retailer/analytics/detailed", d.Service.HandleDetailedAnalytics)

		rr.Get("/v1/retailer/family-members", d.Service.HandleFamilyMembers)
		rr.Post("/v1/retailer/family-members", d.Service.HandleFamilyMembers)
		rr.Delete("/v1/retailer/family-members/{memberID}", d.Service.HandleDeleteFamilyMember)

		if d.OrderService != nil {
			rr.Post("/v1/retailer/shop-closed-response", d.OrderService.HandleShopClosedResponse)
		} else {
			rr.Post("/v1/retailer/shop-closed-response", d.Service.HandleShopClosedResponse)
		}
		rr.Get("/v1/retailer/ai/predictions", d.Service.HandleAIPredictions)
		rr.Get("/v1/ai/predictions", d.Service.HandleAIPredictionsAlias)
		rr.Post("/v1/ai/preorder", d.Service.HandleAIPreorder)
		rr.Patch("/v1/ai/predictions/correct", d.Service.HandleCorrectPrediction)
		rr.Post("/v1/retailer/orders/confirm-ai", d.Service.HandleConfirmAIOrder)
		rr.Post("/v1/retailer/orders/reject-ai", d.Service.HandleRejectAIOrder)
		rr.Post("/v1/orders/edit-preorder", d.Service.HandleEditPreorder)
		rr.Post("/v1/orders/confirm-preorder", d.Service.HandleConfirmPreorder)
		rr.Post("/v1/orders/accept-delivery-proposal", d.Service.HandleAcceptDeliveryProposal)
		rr.Post("/v1/orders/reject-delivery-proposal", d.Service.HandleRejectDeliveryProposal)
		rr.Post("/v1/orders/reject-preorder", d.Service.HandleRejectPreorder)

		rr.Get("/v1/retailer/pending-payments", d.Service.HandlePendingPayments)
		rr.Get("/v1/retailer/active-fulfillment", d.Service.HandleActiveFulfillment)
		rr.Get("/v1/retailer/tracking", d.Service.HandleTracking)

		// GET /v1/catalog/* served by catalogroutes (catalog.Service) with Spanner
		// persistence. Do NOT register demo stubs here.

		// POST /v1/checkout/unified is served by paymentroutes (payment.Service.HandleUnifiedCheckout)
		// with Spanner persistence + outbox. Do NOT register a stub here.
		if d.PaymentService != nil {
			rr.Post("/v1/order/cash-checkout", d.PaymentService.HandleOrderCashCheckout)
			rr.Post("/v1/order/card-checkout", d.PaymentService.HandleOrderCardCheckout)
		} else {
			rr.Post("/v1/order/cash-checkout", d.Service.HandleCashCheckout)
			rr.Post("/v1/order/card-checkout", d.Service.HandleCardCheckout)
		}

		rr.Get("/v1/retailer/settings/auto-order", d.Service.HandleAutoOrderSettings)
		rr.Patch("/v1/retailer/settings/auto-order/global", d.Service.HandleAutoOrderPatch)
		rr.Patch("/v1/retailer/settings/auto-order/supplier/{supplierID}", d.Service.HandleAutoOrderPatch)
		rr.Patch("/v1/retailer/settings/auto-order/category/{categoryID}", d.Service.HandleAutoOrderPatch)
		rr.Patch("/v1/retailer/settings/auto-order/product/{productID}", d.Service.HandleAutoOrderPatch)
		rr.Patch("/v1/retailer/settings/auto-order/variant/{variantID}", d.Service.HandleAutoOrderPatch)

		rr.Get("/v1/retailer/cards", d.Service.HandleRetailerCards)
		rr.Post("/v1/retailer/card/initiate", d.Service.HandleRetailerCardMutation)
		rr.Post("/v1/retailer/card/confirm", d.Service.HandleRetailerCardMutation)
		rr.Post("/v1/retailer/card/deactivate", d.Service.HandleRetailerCardMutation)
		rr.Post("/v1/retailer/card/default", d.Service.HandleRetailerCardMutation)

		rr.Get("/v1/user/notifications", d.Service.HandleUserNotifications)
		rr.Post("/v1/user/notifications/read", d.Service.HandleMarkNotificationsRead)
	}

	auth.ProtectMutations(r, auth.MutationGuardConfig{
		FirebaseEnabled:  d.FirebaseAuthEnabled,
		FirebaseVerifier: d.FirebaseVerifier,
		AllowBypass:      d.AllowAuthBypass,
	}, func(gr chi.Router) {
		gr.Use(auth.RequireRole(auth.RoleRetailer))
		mountProtected(gr)
	})
}
