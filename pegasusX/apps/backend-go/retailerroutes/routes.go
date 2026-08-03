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
		HandleRetailerRespondShopClosed(http.ResponseWriter, *http.Request)
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
	// C1.2 multi-org: intermediate token allowed (SessionAuth attaches claims; no RequireRole).
	r.Get("/v1/auth/retailer/memberships", d.Service.HandleListMemberships)
	r.Post("/v1/auth/retailer/select-org", d.Service.HandleSelectOrg)

	mountProtected := func(rr chi.Router) {
		// C1.2 switch-org requires full JWT (RequireRole rejects PendingOrgSelect).
		rr.Post("/v1/auth/retailer/switch-org", d.Service.HandleSwitchOrg)
		// Retail OS Phase 0 identity + capability packs
		rr.Get("/v1/retailer/me", d.Service.HandleMe)
		rr.Get("/v1/retailer/capabilities", d.Service.HandleCapabilitiesList)
		rr.Post("/v1/retailer/capabilities/{packID}/enable", d.Service.HandleCapabilityEnable)
		rr.Post("/v1/retailer/capabilities/{packID}/disable", d.Service.HandleCapabilityDisable)

		// Retail OS Phase 7 — honest ops pulse (no demo supplier)
		rr.Get("/v1/retailer/control-tower/pulse", d.Service.HandleControlTowerPulse)

		// Retail OS Phase 1 team (org members)
		rr.Get("/v1/retailer/org/members", d.Service.HandleOrgMembers)
		rr.Post("/v1/retailer/org/members", d.Service.HandleOrgMembers)
		rr.Patch("/v1/retailer/org/members/{userID}", d.Service.HandleOrgMemberByID)
		rr.Put("/v1/retailer/org/members/{userID}", d.Service.HandleOrgMemberByID)
		rr.Delete("/v1/retailer/org/members/{userID}", d.Service.HandleOrgMemberByID)
		rr.Put("/v1/retailer/org/members/{userID}/locations", d.Service.HandleMemberLocations)
		rr.Post("/v1/retailer/org/members/{userID}/locations", d.Service.HandleMemberLocations)

		// Retail OS Phase 2 locations
		rr.Get("/v1/retailer/locations", d.Service.HandleLocations)
		rr.Post("/v1/retailer/locations", d.Service.HandleLocations)
		rr.Patch("/v1/retailer/locations/{locationID}", d.Service.HandleLocationByID)
		rr.Put("/v1/retailer/locations/{locationID}", d.Service.HandleLocationByID)
		rr.Post("/v1/retailer/locations/{locationID}/set-primary", d.Service.HandleLocationSetPrimary)
		rr.Post("/v1/auth/retailer/switch-location", d.Service.HandleSwitchLocation)

		// Retail OS Phase 3 store stock
		rr.Get("/v1/retailer/stock", d.Service.HandleStockList)
		rr.Get("/v1/retailer/stock/movements", d.Service.HandleStockMovements)
		rr.Get("/v1/retailer/stock/{sku}", d.Service.HandleStockSKU)
		rr.Post("/v1/retailer/stock/receive-sessions", d.Service.HandleStockReceiveSession)
		rr.Post("/v1/retailer/stock/receive-sessions/{sessionID}/confirm", d.Service.HandleStockReceiveConfirm)
		rr.Post("/v1/retailer/stock/transfer", d.Service.HandleStockTransfer)
		rr.Post("/v1/retailer/stock/adjust", d.Service.HandleStockAdjust)
		rr.Post("/v1/retailer/stock/counts", d.Service.HandleStockCount)
		// Wave C3.3 offline count version protocol
		rr.Get("/v1/retailer/stock/counts/version", d.Service.HandleStockCountVersion)
		rr.Post("/v1/retailer/stock/counts/commit", d.Service.HandleStockCountCommit)

		// Retail OS Phase 4 POS
		rr.Get("/v1/retailer/registers", d.Service.HandleRegisters)
		rr.Post("/v1/retailer/registers", d.Service.HandleRegisters)
		rr.Post("/v1/retailer/pos/sessions/open", d.Service.HandlePosSessionOpen)
		rr.Post("/v1/retailer/pos/sessions/{sessionID}/close", d.Service.HandlePosSessionClose)
		rr.Get("/v1/retailer/pos/sessions/{sessionID}", d.Service.HandlePosSessionGet)
		rr.Post("/v1/retailer/pos/sales", d.Service.HandlePosSale)
		rr.Post("/v1/retailer/pos/sales/{saleID}/void", d.Service.HandlePosSaleVoid)
		rr.Post("/v1/retailer/pos/sales/{saleID}/refund", d.Service.HandlePosSaleRefund)
		rr.Get("/v1/retailer/pos/catalog", d.Service.HandlePOSCatalogSearch)

		// Wave C3.1 parked carts (holds) — flag POS_HOLDS_ENABLED
		rr.Get("/v1/retailer/pos/holds", d.Service.HandlePosHolds)
		rr.Post("/v1/retailer/pos/holds", d.Service.HandlePosHolds)
		rr.Post("/v1/retailer/pos/holds/{holdID}/resume", d.Service.HandlePosHoldResume)
		rr.Post("/v1/retailer/pos/holds/{holdID}/void", d.Service.HandlePosHoldVoid)

		// Local/manual POS catalog (non-Pegasus SKUs)
		rr.Get("/v1/retailer/local-skus", d.Service.HandleLocalSKUs)
		rr.Post("/v1/retailer/local-skus", d.Service.HandleLocalSKUs)
		rr.Patch("/v1/retailer/local-skus/{localSkuID}", d.Service.HandleLocalSKUByID)

		// Retail OS Phase 5 shifts & time clock
		rr.Post("/v1/retailer/time/clock-in", d.Service.HandleClockIn)
		rr.Post("/v1/retailer/time/clock-out", d.Service.HandleClockOut)
		rr.Get("/v1/retailer/time/entries", d.Service.HandleTimeEntries)
		rr.Get("/v1/retailer/shifts", d.Service.HandleShifts)
		rr.Post("/v1/retailer/shifts", d.Service.HandleShifts)
		rr.Post("/v1/retailer/shifts/{shiftID}/close", d.Service.HandleShiftClose)

		// Retail OS Phase 6 sections + reports + assist
		// unassigned-skus before {sectionID} so chi does not treat it as an id
		rr.Get("/v1/retailer/sections/unassigned-skus", d.Service.HandleUnassignedSkus)
		rr.Get("/v1/retailer/sections", d.Service.HandleSections)
		rr.Post("/v1/retailer/sections", d.Service.HandleSections)
		rr.Get("/v1/retailer/sections/{sectionID}", d.Service.HandleSectionByID)
		rr.Patch("/v1/retailer/sections/{sectionID}", d.Service.HandleSectionByID)
		rr.Put("/v1/retailer/sections/{sectionID}", d.Service.HandleSectionByID)
		rr.Delete("/v1/retailer/sections/{sectionID}", d.Service.HandleSectionByID)
		rr.Get("/v1/retailer/sections/{sectionID}/skus", d.Service.HandleSectionSkus)
		rr.Put("/v1/retailer/sections/{sectionID}/skus", d.Service.HandleSectionSkus)
		rr.Get("/v1/retailer/sections/{sectionID}/staff", d.Service.HandleSectionStaff)
		rr.Put("/v1/retailer/sections/{sectionID}/staff", d.Service.HandleSectionStaff)
		rr.Get("/v1/retailer/me/sections", d.Service.HandleMySections)

		rr.Get("/v1/retailer/reports/summary", d.Service.HandleReportsSummary)
		rr.Get("/v1/retailer/reports/sales", d.Service.HandleReportsSales)
		rr.Get("/v1/retailer/reports/inventory", d.Service.HandleReportsInventory)
		rr.Get("/v1/retailer/reports/shifts", d.Service.HandleReportsShifts)
		rr.Get("/v1/retailer/reports/export", d.Service.HandleReportsExport)

		// Wave C2.2 franchise HQ (reads; flag HQ_ANALYTICS_ENABLED)
		rr.Get("/v1/retailer/hq/summary", d.Service.HandleHqSummary)
		rr.Get("/v1/retailer/hq/sales-by-location", d.Service.HandleHqSalesByLocation)
		rr.Get("/v1/retailer/hq/sales-by-sku", d.Service.HandleHqSalesBySku)
		rr.Get("/v1/retailer/hq/shrinkage", d.Service.HandleHqShrinkage)
		rr.Get("/v1/retailer/hq/export", d.Service.HandleHqExport)

		// L3 sell-through flywheel insights
		rr.Get("/v1/retailer/insights/sell-through", d.Service.HandleSellThroughInsights)
		rr.Get("/v1/retailer/reorder-suggestions", d.Service.HandleRetailerReorderSuggestions)

		rr.Get("/v1/retailer/assist/tickets", d.Service.HandleAssistTickets)
		rr.Post("/v1/retailer/assist/tickets", d.Service.HandleAssistTickets)
		rr.Post("/v1/retailer/assist/tickets/{ticketID}/claim", d.Service.HandleAssistClaim)
		rr.Post("/v1/retailer/assist/tickets/{ticketID}/complete", d.Service.HandleAssistComplete)
		rr.Post("/v1/retailer/assist/tickets/{ticketID}/cancel", d.Service.HandleAssistCancel)

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
		rr.Post("/v1/retailer/family-members/migrate-to-team", d.Service.HandleFamilyMigrateToTeam)

		// Auto-order execution (Retail OS close-out)
		rr.Post("/v1/retailer/settings/auto-order/run", d.Service.HandleAutoOrderRun)
		rr.Get("/v1/retailer/settings/auto-order/runs", d.Service.HandleAutoOrderRuns)

		if d.OrderService != nil {
			rr.Post("/v1/retailer/shop-closed-response", d.OrderService.HandleShopClosedResponse)
			rr.Post("/v1/retailer/orders/{orderID}/shop-closed/respond", d.OrderService.HandleRetailerRespondShopClosed)
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
