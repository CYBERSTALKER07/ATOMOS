// Package supplierroutes mounts the supplier-portal URL surface onto the chi
// router. Handlers live in the supplier package; this file is thin by design.
package supplierroutes

import (
	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
)

// Deps is the narrow dependency contract for this routes package.
type Deps struct {
	Service          *supplier.Service
	OrderService     *order.Service
	NotificationInbox *notifications.InboxHandlers
	JWTSecret        string
	Spanner          *spanner.Client
}

// RegisterRoutes mounts:
//
//	POST /v1/auth/supplier/register    (public)
//	POST /v1/auth/supplier/login       (public)
//	POST /v1/supplier/configure        (requires session cookie, ADMIN role)
//	POST /v1/supplier/billing/setup    (requires session cookie, ADMIN role)
//	GET/PUT /v1/supplier/profile       (requires session cookie, ADMIN role)
//	GET/PUT /v1/supplier/topology      (requires session cookie, ADMIN role)
//	GET/POST /v1/supplier/org/members  (requires session cookie, ADMIN role)
//	GET/POST /v1/supplier/fleet/drivers  (requires session cookie, ADMIN role)
//	GET/POST /v1/supplier/fleet/vehicles (requires session cookie, ADMIN role)
//	GET /v1/supplier/ws-session         (requires session cookie, ADMIN role)
//	GET /v1/supplier/dashboard         (requires session cookie, ADMIN role)
//	GET /v1/supplier/earnings          (requires session cookie, ADMIN role)
//	GET/PATCH /v1/supplier/inventory   (requires session cookie, ADMIN role)
//	GET /v1/supplier/inventory/audit   (requires session cookie, ADMIN role)
//	GET /v1/supplier/orders            (requires session cookie, ADMIN role)
//	POST /v1/supplier/orders/vet       (requires session cookie, ADMIN role)
//	GET/POST /v1/supplier/ai/recommendations (requires session cookie, ADMIN role)
//	GET  /v1/supplier/shop-closed/active
//	POST /v1/supplier/orders/payment-bypass
//	GET  /v1/supplier/negotiations/pending
//	POST /v1/supplier/negotiate/resolve
//	POST /v1/supplier/route/approve-early-complete
//	GET  /v1/supplier/empathy/adoption
//	POST /v1/supplier/broadcast
//	POST /v1/supplier/replenishment/trigger
//	GET  /v1/supplier/fleet/orders
//	GET  /v1/supplier/fleet/live-map
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}
	r.Post("/v1/auth/supplier/register", d.Service.HandleRegister)
	r.Post("/v1/auth/supplier/login", d.Service.HandleLogin)
	r.Post("/v1/auth/supplier/refresh", d.Service.HandleSupplierRefresh)

	warehouseScope := auth.RequireWarehouseScope
	if d.Spanner != nil {
		warehouseScope = auth.RequireWarehouseScopeWithClient(d.Spanner)
	}

	r.Group(func(gr chi.Router) {
		gr.Use(auth.CookieAuth(d.JWTSecret))
		gr.Use(auth.RequireRole(auth.RoleAdmin))
		gr.Post("/v1/supplier/configure", d.Service.HandleConfigure)
		gr.Post("/v1/supplier/business/setup", d.Service.HandleSupplierBusinessSetup)
		gr.Post("/v1/supplier/billing/setup", d.Service.HandleConfigureBilling)
		gr.Get("/v1/supplier/profile", d.Service.HandleProfile)
		gr.Put("/v1/supplier/profile", d.Service.HandleProfile)
		gr.Get("/v1/supplier/pricing/rules", d.Service.HandlePricingRules)
		gr.Patch("/v1/supplier/pricing/rules", d.Service.HandlePricingRules)
		gr.Get("/v1/supplier/pricing/retailer-overrides", d.Service.HandleRetailerPricingOverrides)
		gr.Post("/v1/supplier/pricing/retailer-overrides", d.Service.HandleRetailerPricingOverrides)
		gr.Delete("/v1/supplier/pricing/retailer-overrides/{overrideID}", d.Service.HandleRetailerPricingOverrideDelete)
		gr.Get("/v1/supplier/topology", d.Service.HandleTopology)
		gr.Put("/v1/supplier/topology", d.Service.HandleTopology)
		gr.Get("/v1/supplier/org/members", d.Service.HandleOrgMembers)
		gr.Post("/v1/supplier/org/members", d.Service.HandleOrgMembers)
		gr.Patch("/v1/supplier/org/members/{userID}", d.Service.HandleOrgMemberByID)
		gr.Put("/v1/supplier/org/members/{userID}", d.Service.HandleOrgMemberByID)
		gr.Delete("/v1/supplier/org/members/{userID}", d.Service.HandleOrgMemberByID)
		gr.Get("/v1/supplier/fleet/drivers", d.Service.HandleFleetDrivers)
		gr.Post("/v1/supplier/fleet/drivers", d.Service.HandleFleetDrivers)
		gr.Get("/v1/supplier/fleet/vehicles", d.Service.HandleFleetVehicles)
		gr.Post("/v1/supplier/fleet/vehicles", d.Service.HandleFleetVehicles)
		gr.Get("/v1/supplier/ws-session", d.Service.HandleWebSocketSession)
		gr.Get("/v1/supplier/dashboard", d.Service.HandleDashboard)
		gr.Get("/v1/supplier/manifests", d.Service.HandleManifests)
		gr.With(warehouseScope).Get("/v1/supplier/dispatch/preview", d.Service.HandleDispatchPreview)
		gr.With(warehouseScope).Post("/v1/supplier/dispatch/preview", d.Service.HandleDispatchPreview)
		gr.With(warehouseScope).Post("/v1/supplier/dispatch/execute", d.Service.HandleDispatchExecute)
		gr.Get("/v1/supplier/activity", d.Service.HandleActivity)
		gr.Get("/v1/supplier/supply-lanes", d.Service.HandleSupplyLanes)
		gr.Get("/v1/supplier/exceptions", d.Service.HandleExceptions)
		gr.Get("/v1/supplier/manifest-exceptions", d.Service.HandleManifestExceptions)
		gr.Get("/v1/supplier/earnings", d.Service.HandleEarnings)
		gr.Get("/v1/supplier/inventory", d.Service.HandleInventory)
		gr.Patch("/v1/supplier/inventory", d.Service.HandleInventory)
		gr.Post("/v1/supplier/inventory/import", d.Service.HandleInventoryImport)
		gr.Get("/v1/supplier/inventory/audit", d.Service.HandleInventoryAudit)
		gr.Get("/v1/supplier/analytics/velocity", d.Service.HandleAnalyticsVelocity)
		gr.Get("/v1/supplier/analytics/revenue", d.Service.HandleAnalyticsRevenue)
		gr.Get("/v1/supplier/analytics/demand/today", d.Service.HandleAnalyticsDemandToday)
		gr.Get("/v1/supplier/analytics/demand/history", d.Service.HandleAnalyticsDemandHistory)
		gr.Get("/v1/supplier/orders", d.Service.HandleOrders)
		gr.Post("/v1/supplier/orders/vet", d.Service.HandleVetOrder)
		gr.Get("/v1/supplier/ai/recommendations", d.Service.HandleAIRecommendations)
		gr.Post("/v1/supplier/ai/recommendations", d.Service.HandleAIRecommendations)

		if d.OrderService != nil {
			gr.Get("/v1/supplier/shop-closed/active", d.OrderService.HandleListActiveShopClosedAttempts)
			gr.Post("/v1/supplier/shop-closed/resolve", d.OrderService.HandleResolveShopClosed)
			gr.Post("/v1/supplier/orders/payment-bypass", d.OrderService.HandleIssuePaymentBypass)
			// Quantity negotiation disabled — handlers return empty list or 410.
			gr.Get("/v1/supplier/negotiations/pending", d.OrderService.HandleListPendingNegotiations)
			gr.Post("/v1/supplier/negotiate/resolve", d.OrderService.HandleResolveNegotiation)
			gr.Post("/v1/supplier/route/approve-early-complete", d.OrderService.HandleApproveEarlyComplete)
		}
		gr.Get("/v1/supplier/empathy/adoption", d.Service.HandleEmpathyAdoption)
		gr.Post("/v1/supplier/broadcast", d.Service.HandleBroadcast)
		gr.Post("/v1/supplier/replenishment/trigger", d.Service.HandleReplenishmentTrigger)
		gr.Get("/v1/supplier/fleet/orders", d.Service.HandleSupplierFleetOrders)
		gr.Get("/v1/supplier/fleet/live-map", d.Service.HandleSupplierFleetLiveMap)
		if d.NotificationInbox != nil {
			gr.Get("/v1/user/notifications", d.NotificationInbox.HandleList)
			gr.Post("/v1/user/notifications/read", d.NotificationInbox.HandleMarkRead)
		}
	})
}
