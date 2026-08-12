// Package supplierroutes mounts the supplier-portal URL surface onto the chi
// router. Handlers live in the supplier package; this file is thin by design.
package supplierroutes

import (
	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/payload"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"github.com/pegasusx/pegasusx/apps/backend-go/compliance"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
	"github.com/pegasusx/pegasusx/apps/backend-go/segment"
	"github.com/pegasusx/pegasusx/apps/backend-go/twin"
)

// Deps is the narrow dependency contract for this routes package.
type Deps struct {
	Service           *supplier.Service
	OrderService      *order.Service
	PayloadService    *payload.Service
	NotificationInbox *notifications.InboxHandlers
	ComplianceHandler *compliance.Handler
	ExceptionResolve  supplier.ExceptionResolveDeps
	JWTSecret         string
	Spanner           *spanner.Client
	SupplierHub       *ws.Hub
	WarehouseHub      *ws.Hub
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
//	GET  /v1/supplier/returns
//	POST /v1/supplier/returns/resolve
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
		if d.OrderService != nil {
			gr.Get("/v1/supplier/return-policy", d.OrderService.HandleSupplierReturnPolicy)
			gr.Put("/v1/supplier/return-policy", d.OrderService.HandleSupplierReturnPolicy)
		}
		gr.Put("/v1/supplier/profile", d.Service.HandleProfile)
		// Enterprise alias: api-client getSupplierSettings → same projection as profile.
		gr.Get("/v1/supplier/settings", d.Service.HandleProfile)
		gr.Get("/v1/supplier/pricing/rules", d.Service.HandlePricingRules)
		gr.Patch("/v1/supplier/pricing/rules", d.Service.HandlePricingRules)
		gr.Get("/v1/supplier/pricing/retailer-overrides", d.Service.HandleRetailerPricingOverrides)
		gr.Post("/v1/supplier/pricing/retailer-overrides", d.Service.HandleRetailerPricingOverrides)
		gr.Post("/v1/supplier/pricing/retailer-overrides/preview", d.Service.HandleRetailerPricingOverridePreview)
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
		gr.Get("/v1/supplier/dispatch/tracking", d.Service.HandleDispatchTracking)
		gr.Get("/v1/supplier/activity", d.Service.HandleActivity)
		gr.Get("/v1/supplier/supply-lanes", d.Service.HandleSupplyLanes)
		gr.Get("/v1/supplier/exceptions", d.Service.HandleExceptions)
		gr.Post("/v1/supplier/exceptions/{kind}/{id}/resolve", supplier.HandleResolveException(d.ExceptionResolve))
		gr.Get("/v1/supplier/ops/exception-map", d.Service.HandleExceptionMap)
		gr.Get("/v1/supplier/manifest-exceptions", d.Service.HandleManifestExceptions)
		gr.Get("/v1/supplier/earnings", d.Service.HandleEarnings)
		gr.Get("/v1/supplier/inventory", d.Service.HandleInventory)
		gr.Patch("/v1/supplier/inventory", d.Service.HandleInventory)
		gr.Patch("/v1/supplier/inventory/policy", d.Service.HandleInventoryPolicy)
		gr.Post("/v1/supplier/inventory/import", d.Service.HandleInventoryImport)
		supplier.RegisterImportRoutes(gr, supplier.ImportRoutesDeps{
			Spanner:      d.Spanner,
			Service:      d.Service,
			SupplierHub:  d.SupplierHub,
			WarehouseHub: d.WarehouseHub,
		})
		gr.Get("/v1/supplier/inventory/audit", d.Service.HandleInventoryAudit)
		gr.Get("/v1/supplier/analytics/velocity", d.Service.HandleAnalyticsVelocity)
		gr.Get("/v1/supplier/analytics/revenue", d.Service.HandleAnalyticsRevenue)
		gr.Get("/v1/supplier/analytics/demand/today", d.Service.HandleAnalyticsDemandToday)
		gr.Get("/v1/supplier/analytics/demand/history", d.Service.HandleAnalyticsDemandHistory)
		gr.Get("/v1/supplier/analytics/demand/accuracy", d.Service.HandleAnalyticsDemandAccuracy)
		// B4.4 STORE_POS flywheel DEMAND_SIGNAL feed (distinct from planning DemandSignals).
		gr.Get("/v1/supplier/analytics/demand/flywheel", d.Service.HandleAnalyticsDemandFlywheel)
		gr.Get("/v1/supplier/orders", d.Service.HandleOrders)
		gr.Post("/v1/supplier/orders/vet", d.Service.HandleVetOrder)
		gr.Get("/v1/supplier/ai/recommendations", d.Service.HandleAIRecommendations)
		gr.Post("/v1/supplier/ai/recommendations", d.Service.HandleAIRecommendations)

		if d.OrderService != nil {
			gr.Get("/v1/supplier/shop-closed/active", d.OrderService.HandleListActiveShopClosedAttempts)
			gr.Post("/v1/supplier/shop-closed/resolve", d.OrderService.HandleResolveShopClosed)
			gr.Post("/v1/supplier/orders/payment-bypass", d.OrderService.HandleIssuePaymentBypass)
			// Quantity negotiation product-disabled → empty pending list / 410 resolve.
			gr.Get("/v1/supplier/negotiations/pending", d.OrderService.HandleListPendingNegotiations)
			gr.Post("/v1/supplier/negotiate/resolve", d.OrderService.HandleResolveNegotiation)
			gr.Post("/v1/supplier/route/approve-early-complete", d.OrderService.HandleApproveEarlyComplete)
			gr.Get("/v1/compliance/fiscal-open", d.OrderService.HandleComplianceFiscalOpen)
			gr.Get("/v1/compliance/force-completes", d.OrderService.HandleComplianceForceCompletes)
			gr.Get("/v1/compliance/claim-mismatches", d.OrderService.HandleComplianceClaimMismatches)
			gr.Get("/v1/compliance/credit-freezes", d.OrderService.HandleComplianceCreditFreezes)
		}

		if d.ComplianceHandler != nil {
			gr.Get("/v1/compliance/dashboard", d.ComplianceHandler.GetDashboard)
			gr.Get("/v1/compliance/export", d.ComplianceHandler.ExportCSV)
		}
		
		if d.PayloadService != nil {
			gr.Post("/v1/supplier/reassign-order", d.PayloadService.HandleApplyReassign)
			gr.Post("/v1/supplier/recommend-reassign", d.PayloadService.HandleRecommendReassign)
		}

		gr.Get("/v1/supplier/empathy/adoption", d.Service.HandleEmpathyAdoption)
		gr.Post("/v1/supplier/broadcast", d.Service.HandleBroadcast)
		gr.Post("/v1/supplier/replenishment/trigger", d.Service.HandleReplenishmentTrigger)
		gr.Get("/v1/supplier/replenishment/policies", d.Service.HandleReplenishmentPolicies)
		gr.Patch("/v1/supplier/replenishment/policies", d.Service.HandleReplenishmentPolicies)
		gr.Get("/v1/supplier/replenishment/traceability", d.Service.HandleReplenishmentTraceability)

		if d.Spanner != nil && d.OrderService != nil {
			suggestionsAPI := replenishment.NewSuggestionsAPI(d.Spanner, d.OrderService, d.Service.ScopedSupplierID)
			gr.Get("/v1/replenishment/suggestions", suggestionsAPI.HandleList)
			gr.Post("/v1/replenishment/suggestions/dismiss", suggestionsAPI.HandleDismiss)
			gr.Post("/v1/replenishment/suggestions/create-draft", suggestionsAPI.HandleCreateDraft)
			gr.Post("/v1/replenishment/suggestions/create-drafts", suggestionsAPI.HandleBulkCreateDrafts)
		}

		if d.Spanner != nil {
			twinHandler := twin.NewSupplierHTTPHandler(twin.NewSpannerRepository(d.Spanner), d.Service.ScopedSupplierID)
			gr.Get("/v1/twin/routes/active", twinHandler.ListActiveRoutes)
			gr.Get("/v1/twin/routes/{routeID}", twinHandler.GetRoute)
			gr.Get("/v1/twin/routes/{routeID}/inventory", twinHandler.GetRouteInventory)

			segmentHandlers := &segment.Handlers{
				Service:    segment.NewService(segment.NewSpannerRepository(d.Spanner)),
				SupplierID: d.Service.ScopedSupplierID,
			}
			gr.Post("/v1/supplier/segmentation/bootstrap", segmentHandlers.HandleBootstrap)
			gr.Get("/v1/supplier/segmentation/retailers", segmentHandlers.HandleRetailerSegments)
			gr.Patch("/v1/supplier/segmentation/retailers/{retailerID}", segmentHandlers.HandleRetailerSegmentByID)
			gr.Get("/v1/supplier/segmentation/sku-classes", segmentHandlers.HandleSkuClasses)
			gr.Patch("/v1/supplier/segmentation/sku-classes/{sku}", segmentHandlers.HandleSkuClassBySKU)
		}

		gr.Get("/v1/supplier/meio/network-summary", d.Service.HandleMEIONetworkSummary)
		gr.Get("/v1/supplier/control-tower/zone-overrides", d.Service.HandleControlTowerZoneOverrides)
		gr.Post("/v1/supplier/control-tower/zone-overrides", d.Service.HandleControlTowerZoneOverrides)
		gr.Post("/v1/supplier/planning/scenarios/run", d.Service.HandlePlanningScenarioRun)
		gr.Get("/v1/supplier/planning/scenarios", d.Service.HandlePlanningScenarioList)
		gr.Post("/v1/supplier/planning/scenarios/compare", d.Service.HandlePlanningScenarioCompare)
		gr.Post("/v1/supplier/planning/scenarios/{scenarioID}/clone", d.Service.HandlePlanningScenarioClone)
		gr.Post("/v1/supplier/planning/scenarios/{scenarioID}/publish", d.Service.HandlePlanningScenarioPublish)
		gr.Get("/v1/supplier/planning/s-and-op", d.Service.HandlePlanningSAndOP)
		gr.Get("/v1/supplier/knowledge-graph", d.Service.HandleKnowledgeGraph)
		gr.Post("/v1/supplier/planning/agent/invoke", d.Service.HandleGovernedAgentHook)
		gr.Get("/v1/supplier/planning/seasonal-overrides", d.Service.HandlePlanningSeasonalOverrides)
		gr.Post("/v1/supplier/planning/seasonal-overrides", d.Service.HandlePlanningSeasonalOverrides)
		gr.Post("/v1/supplier/planning/seasonal-estimate", d.Service.HandlePlanningSeasonalEstimate)
		gr.Post("/v1/supplier/planning/signals/ingest", d.Service.HandlePlanningSignalIngest)
		gr.Get("/v1/supplier/planning/signals/status", d.Service.HandlePlanningSignalStatus)
		gr.Post("/v1/supplier/planning/promotions/simulate", d.Service.HandlePlanningPromoSimulate)
		gr.Get("/v1/supplier/planning/promotions/{promotionID}/performance", d.Service.HandlePlanningPromoPerformance)
		gr.Get("/v1/supplier/planning/sparsity/{retailerID}", d.Service.HandlePlanningSparsityCheck)
		gr.Get("/v1/supplier/fleet/orders", d.Service.HandleSupplierFleetOrders)
		gr.Get("/v1/supplier/fleet/live-map", d.Service.HandleSupplierFleetLiveMap)
		gr.Get("/v1/supplier/returns", d.Service.HandleReturns)
		gr.Post("/v1/supplier/returns/resolve", d.Service.HandleResolveReturn)
		if d.NotificationInbox != nil {
			gr.Get("/v1/user/notifications", d.NotificationInbox.HandleList)
			gr.Post("/v1/user/notifications/read", d.NotificationInbox.HandleMarkRead)
		}
	})
}
