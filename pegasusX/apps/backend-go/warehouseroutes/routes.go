// Package warehouseroutes mounts warehouse-role endpoints.
package warehouseroutes

import (
	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
)

// Deps is the narrow dependency contract for warehouse routes.
type Deps struct {
	Service             *warehouse.Service
	OrderService        *order.Service
	JWTSecret           string
	Spanner             *spanner.Client
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
}

// RegisterRoutes mounts warehouse role-row operational endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	r.Post("/v1/auth/warehouse/login", d.Service.HandleWarehouseLogin)
	r.Post("/v1/auth/warehouse/register", d.Service.HandleWarehouseRegister)
	r.Post("/v1/auth/warehouse/refresh", d.Service.HandleWarehouseRefresh)
	r.Post("/v1/warehouse/setup", d.Service.HandleWarehouseSetup)

	// Ecosystem CRUD
	r.Post("/v1/warehouses", d.Service.HandleCreateWarehouse)
	r.Get("/v1/warehouses/{warehouseId}", d.Service.HandleGetWarehouse)
	r.Put("/v1/warehouses/{warehouseId}", d.Service.HandleUpdateWarehouse)
	r.Get("/v1/warehouses", d.Service.HandleListWarehouses)

	mountReplenishmentInsights := func(rr chi.Router) {
		rr.Get("/v1/warehouse/replenishment/insights", d.Service.HandleReplenishmentInsights)
		rr.Post("/v1/warehouse/replenishment/insights/{id}/{action}", d.Service.HandleReplenishmentInsightAction)
	}

	mountProtected := func(rr chi.Router) {
		rr.Post("/v1/warehouse/transfers/emergency", d.Service.HandleEmergencyTransfer)
		rr.Post("/v1/warehouse/transfers/force-receive", d.Service.HandleForceReceive)
		rr.Post("/v1/warehouse/transfers/{id}/receive", d.Service.HandleReceiveTransfer)

		rr.Get("/v1/warehouse/ops/dashboard", d.Service.HandleDashboard)
		rr.HandleFunc("/v1/warehouse/ops/inventory", d.Service.HandleInventory)
		rr.Patch("/v1/warehouse/ops/inventory/{productID}/policy", d.Service.HandleInventoryPolicy)
		rr.Get("/v1/warehouse/ops/settings", d.Service.HandleOpsSettings)
		rr.Patch("/v1/warehouse/ops/settings", d.Service.HandleOpsSettings)
		rr.Get("/v1/warehouse/ops/location", d.Service.HandleOpsLocation)
		rr.Patch("/v1/warehouse/ops/location", d.Service.HandleOpsLocation)
		rr.Get("/v1/warehouse/ops/orders", d.Service.HandleOrders)
		rr.Get("/v1/warehouse/ops/orders/*", d.Service.HandleOrders)
		if d.OrderService != nil {
			rr.Post("/v1/warehouse/ops/orders/{id}/delay", d.OrderService.HandleWarehouseMarkDelayed)
			rr.Post("/v1/warehouse/ops/orders/{id}/reject", d.OrderService.HandleWarehouseRejectOrder)
			rr.Post("/v1/warehouse/ops/orders/{id}/overflow", d.OrderService.HandleWarehousePayloadOverflow)
			rr.Post("/v1/warehouse/ops/orders/{id}/propose-delivery", d.OrderService.HandleWarehouseProposeDelivery)
			rr.Post("/v1/warehouse/ops/preorders/{id}/propose-delivery", d.OrderService.HandleWarehouseProposeDelivery)
			rr.Get("/v1/warehouse/ops/preorders", d.OrderService.HandleWarehouseListPreorders)
			rr.Post("/v1/warehouse/ops/preorders/{id}/edit", d.OrderService.HandleWarehouseEditPreorder)
			rr.Post("/v1/warehouse/ops/preorders/{id}/reject", d.OrderService.HandleWarehouseRejectPreorder)
		}
		rr.Get("/v1/warehouse/ops/stock-commitments", d.Service.HandleStockCommitments)
		rr.Get("/v1/warehouse/ops/stock-commitments/{skuId}", d.Service.HandleStockCommitmentBySKU)
		rr.Get("/v1/warehouse/ops/dispatch/preview", d.Service.HandleDispatchPreview)
		rr.Post("/v1/warehouse/ops/dispatch/preview", d.Service.HandleDispatchPreview)
		rr.Post("/v1/warehouse/ops/dispatch/execute", d.Service.HandleDispatchExecute)
		rr.Get("/v1/warehouse/ops/dispatch/runs", d.Service.HandleDispatchRuns)
		rr.Get("/v1/warehouse/ops/dispatch/runs/{runID}", d.Service.HandleDispatchRunDetail)
		rr.Get("/v1/warehouse/ops/board", d.Service.HandleOpsBoard)
		rr.Get("/v1/warehouse/ops/exceptions", d.Service.HandleOpsExceptions)
		rr.Get("/v1/warehouse/ops/broadcast/templates", d.Service.HandleWarehouseBroadcastTemplates)
		rr.Post("/v1/warehouse/ops/broadcast/templates", d.Service.HandleWarehouseBroadcastTemplates)
		rr.Delete("/v1/warehouse/ops/broadcast/templates/{id}", d.Service.HandleWarehouseBroadcastTemplateDelete)
		rr.Post("/v1/warehouse/ops/broadcast", d.Service.HandleWarehouseBroadcast)
		rr.Post("/v1/warehouse/ops/pricing/retailer-overrides/preview", d.Service.HandleWarehouseRetailerPricingPreview)
		rr.Get("/v1/warehouse/ops/dispatch/settings", d.Service.HandleDispatchSettings)
		rr.Patch("/v1/warehouse/ops/dispatch/settings", d.Service.HandleDispatchSettings)
		rr.Get("/v1/warehouse/ops/drivers", d.Service.HandleOpsDrivers)
		rr.Post("/v1/warehouse/ops/drivers", d.Service.HandleOpsDrivers)
		rr.Patch("/v1/warehouse/ops/drivers/*", d.Service.HandleOpsDrivers)
		rr.Get("/v1/warehouse/ops/vehicles", d.Service.HandleOpsVehicles)
		rr.Post("/v1/warehouse/ops/vehicles", d.Service.HandleOpsVehicles)
		rr.Patch("/v1/warehouse/ops/vehicles/*", d.Service.HandleOpsVehicles)
		rr.Get("/v1/warehouse/ops/staff", d.Service.HandleOpsStaff)
		rr.Post("/v1/warehouse/ops/staff", d.Service.HandleOpsStaff)
		rr.Get("/v1/warehouse/ops/products", d.Service.HandleOpsProducts)
		rr.Get("/v1/warehouse/ops/manifests", d.Service.HandleOpsManifests)
		rr.Get("/v1/warehouse/ops/fleet/live-map", d.Service.HandleWarehouseFleetLiveMap)
		rr.Get("/v1/warehouse/ops/analytics", d.Service.HandleOpsAnalytics)
		rr.Get("/v1/warehouse/ops/crm", d.Service.HandleOpsCRM)
		rr.Get("/v1/warehouse/ops/returns", d.Service.HandleOpsReturns)
		rr.Get("/v1/warehouse/ops/treasury", d.Service.HandleOpsTreasury)
		rr.Get("/v1/warehouse/ops/financials", d.Service.HandleOpsFinancials)
		rr.Get("/v1/warehouse/ops/payment-config", d.Service.HandleOpsPaymentConfig)
		rr.Post("/v1/warehouse/ops/payment-config", d.Service.HandleOpsPaymentConfig)
		rr.Get("/v1/warehouse/demand/forecast", d.Service.HandleDemandForecast)
		rr.Get("/v1/warehouse/supply-requests", d.Service.HandleSupplyRequests)
		rr.Post("/v1/warehouse/supply-requests", d.Service.HandleSupplyRequests)
		rr.Get("/v1/warehouse/supply-requests/*", d.Service.HandleSupplyRequestByID)
		rr.Patch("/v1/warehouse/supply-requests/*", d.Service.HandleSupplyRequestByID)
		rr.Get("/v1/warehouse/dispatch-locks", d.Service.HandleDispatchLocks)
		rr.Post("/v1/warehouse/dispatch-lock", d.Service.HandleDispatchLock)
		rr.Delete("/v1/warehouse/dispatch-lock", d.Service.HandleDispatchLock)
	}

	allowed := []auth.Role{auth.RoleWarehouse, auth.RoleWarehouseAdmin, auth.RoleAdmin}
	insightsRoles := []auth.Role{
		auth.RoleWarehouse,
		auth.RoleWarehouseAdmin,
		auth.RoleFactory,
		auth.RoleFactoryAdmin,
		auth.RoleAdmin,
	}
	register := func(gr chi.Router) {
		gr.Group(func(insights chi.Router) {
			insights.Use(auth.RequireRole(insightsRoles...))
			insights.Use(auth.RequireReplenishmentInsightsScope(d.Spanner))
			mountReplenishmentInsights(insights)
		})
		gr.Use(auth.RequireRole(allowed...))
		gr.Use(auth.RequireWarehouseOpsScope)
		mountProtected(gr)
	}

	if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
		r.Group(func(gr chi.Router) {
			gr.Use(auth.FirebaseAuth(d.FirebaseVerifier))
			register(gr)
		})
		return
	}

	r.Group(register)
}
