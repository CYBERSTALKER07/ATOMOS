// Package warehouseroutes mounts warehouse-role endpoints.
package warehouseroutes

import (
	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/driver"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/payload"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
)

// Deps is the narrow dependency contract for warehouse routes.
type Deps struct {
	Service        *warehouse.Service
	DriverService  *driver.Service
	OrderService   *order.Service
	PayloadService *payload.Service
	WMSHandler     *stocklots.Handler
	JWTSecret      string
	JWTIssuer      string
	Spanner        *spanner.Client
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

	mountReplenishmentInsights := func(rr chi.Router) {
		rr.Get("/v1/warehouse/replenishment/insights", d.Service.HandleReplenishmentInsights)
		rr.Post("/v1/warehouse/replenishment/insights/{id}/{action}", d.Service.HandleReplenishmentInsightAction)
	}

	mountProtected := func(rr chi.Router) {
		// Ecosystem CRUD
		rr.Post("/v1/warehouses", d.Service.HandleCreateWarehouse)
		rr.Get("/v1/warehouses/{warehouseId}", d.Service.HandleGetWarehouse)
		rr.Put("/v1/warehouses/{warehouseId}", d.Service.HandleUpdateWarehouse)
		rr.Get("/v1/warehouses", d.Service.HandleListWarehouses)
		rr.Post("/v1/warehouses/publish-perimeter", d.Service.HandlePublishPerimeter)

		rr.Post("/v1/warehouse/transfers/emergency", d.Service.HandleEmergencyTransfer)
		rr.Post("/v1/warehouse/transfers/force-receive", d.Service.HandleForceReceive)
		rr.Post("/v1/warehouse/transfers/{id}/receive", d.Service.HandleReceiveTransfer)

		rr.Get("/v1/warehouse/ops/dashboard", d.Service.HandleDashboard)
		rr.HandleFunc("/v1/warehouse/ops/inventory", d.Service.HandleInventory)
		rr.Patch("/v1/warehouse/ops/inventory/{productID}/policy", d.Service.HandleInventoryPolicy)
		if d.WMSHandler != nil {
			rr.Get("/v1/warehouse/ops/bins", d.WMSHandler.HandleBins)
			rr.Post("/v1/warehouse/ops/bins", d.WMSHandler.HandleBins)
			rr.Get("/v1/warehouse/ops/bins/{locationID}", d.WMSHandler.HandleBinByID)
			rr.Patch("/v1/warehouse/ops/bins/{locationID}", d.WMSHandler.HandleBinByID)
			rr.Get("/v1/warehouse/ops/lots", d.WMSHandler.HandleLots)
			rr.Post("/v1/warehouse/ops/lots/putaway", d.WMSHandler.HandlePutaway)
			rr.Get("/v1/warehouse/ops/lots/{lotID}", d.WMSHandler.HandleLotByID)
			rr.Get("/v1/warehouse/ops/lots/{lotID}/trace", d.WMSHandler.HandleTraceLot)
			rr.Post("/v1/warehouse/ops/lots/{lotID}/quarantine", d.WMSHandler.HandleQuarantineLot)
			rr.Post("/v1/warehouse/ops/lots/{lotID}/release", d.WMSHandler.HandleReleaseLot)
			rr.Get("/v1/warehouse/ops/recalls", d.WMSHandler.HandleRecalls)
			rr.Post("/v1/warehouse/ops/recalls", d.WMSHandler.HandleRecalls)
			rr.Get("/v1/warehouse/ops/recalls/{campaignID}", d.WMSHandler.HandleRecallByID)
			rr.Get("/v1/warehouse/ops/pick-waves", d.WMSHandler.HandlePickWaves)
			rr.Post("/v1/warehouse/ops/pick-waves", d.WMSHandler.HandlePickWaves)
			rr.Get("/v1/warehouse/ops/pick-waves/{waveID}", d.WMSHandler.HandlePickWaveByID)
			rr.Post("/v1/warehouse/ops/pick-waves/{waveID}/tasks/{taskID}/confirm", d.WMSHandler.HandleConfirmPickTask)
			rr.Get("/v1/warehouse/ops/cycle-counts", d.WMSHandler.HandleCycleCounts)
			rr.Post("/v1/warehouse/ops/cycle-counts", d.WMSHandler.HandleCycleCounts)
			rr.Get("/v1/warehouse/ops/cycle-counts/{countID}", d.WMSHandler.HandleCycleCountByID)
			rr.Post("/v1/warehouse/ops/cycle-counts/{countID}/submit", d.WMSHandler.HandleSubmitCycleCount)
			rr.Get("/v1/warehouse/ops/inventory-adjustments", d.WMSHandler.HandleInventoryAdjustments)
			rr.Get("/v1/warehouse/ops/inventory-accuracy", d.WMSHandler.HandleInventoryAccuracy)
			rr.Get("/v1/warehouse/ops/inventory-reconcile", d.WMSHandler.HandleReconcileInventoryV2)
			rr.Post("/v1/warehouse/ops/cycle-counts/enqueue-abc", d.WMSHandler.HandleEnqueueABCCounts)
			rr.Get("/v1/warehouse/ops/temperature-readings", d.WMSHandler.HandleTemperatureReadings)
			rr.Post("/v1/warehouse/ops/temperature-readings", d.WMSHandler.HandleTemperatureReadings)
			rr.Group(func(admin chi.Router) {
				admin.Use(auth.RequireRole(auth.RoleWarehouseAdmin, auth.RoleAdmin))
				admin.Post("/v1/warehouse/ops/pick-waves/{waveID}/waive-shorts", d.WMSHandler.HandleWaivePickShorts)
				admin.Post("/v1/warehouse/ops/inventory-adjustments/{adjustmentID}/approve", d.WMSHandler.HandleApproveInventoryAdjustment)
			})
		}
		if d.DriverService != nil {
			rr.Get("/v1/warehouse/ops/cash-reconciliations", d.DriverService.HandleListCashReconciliations)
			rr.Post("/v1/warehouse/ops/cash-reconciliations/{id}/accept", d.DriverService.HandleCashReconciliationAccept)
			rr.Post("/v1/warehouse/ops/cash-reconciliations/{id}/dispute", d.DriverService.HandleCashReconciliationDispute)
		}
		rr.Get("/v1/warehouse/ops/settings", d.Service.HandleOpsSettings)
		rr.Patch("/v1/warehouse/ops/settings", d.Service.HandleOpsSettings)
		rr.Get("/v1/warehouse/ops/location", d.Service.HandleOpsLocation)
		rr.Patch("/v1/warehouse/ops/location", d.Service.HandleOpsLocation)
		rr.Get("/v1/warehouse/ops/coverage", d.Service.HandleOpsCoverage)
		rr.Get("/v1/warehouse/ops/supply-factory", d.Service.HandleOpsSupplyFactory)
		rr.Get("/v1/warehouse/ops/orders", d.Service.HandleOrders)
		rr.Get("/v1/warehouse/ops/orders/*", d.Service.HandleOrders)
		if d.OrderService != nil {
			rr.Group(func(admin chi.Router) {
				admin.Use(auth.RequireRole(auth.RoleWarehouseAdmin, auth.RoleAdmin))
				admin.Get("/v1/warehouse/return-policy", d.OrderService.HandleWarehouseReturnPolicy)
				admin.Put("/v1/warehouse/return-policy", d.OrderService.HandleWarehouseReturnPolicy)
			})
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
		rr.Get("/v1/warehouse/dispatch/tracking", d.Service.HandleDispatchTracking)

		// Payload parity for reassignment without scanning
		if d.PayloadService != nil {
			rr.Post("/v1/warehouse/reassign-order", d.PayloadService.HandleApplyReassign)
			rr.Post("/v1/warehouse/recommend-reassign", d.PayloadService.HandleRecommendReassign)
		}

		// Rescue routes
		rr.Post("/v1/warehouse/ops/dispatch/rescue/preview", d.Service.HandleOpsDispatchRescuePreview)
		rr.Post("/v1/warehouse/ops/dispatch/rescue/propose", d.Service.HandleOpsDispatchRescuePropose)
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
		if d.PayloadService != nil {
			rr.Get("/v1/warehouse/manifests/{manifestID}/ship-units", d.PayloadService.HandleListShipUnits)
			rr.Post("/v1/warehouse/manifests/{manifestID}/labels", d.PayloadService.HandleManifestLabels)
		}
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
		rr.Get("/v1/warehouse/supply-requests/{id}/qc", d.Service.HandleSupplyRequestQC)
		rr.Post("/v1/warehouse/supply-requests/{id}/qc", d.Service.HandleSupplyRequestQC)
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
		gr.Group(func(ws chi.Router) {
			ws.Use(auth.RequireRole(allowed...))
			ws.Get("/v1/warehouse/ws-session", auth.WSSessionHandler(d.JWTSecret, d.JWTIssuer, 0))
		})
		gr.Use(auth.RequireRole(allowed...))
		gr.Use(auth.RequireWarehouseOpsScope)
		mountProtected(gr)
	}

	r.Group(register)
}
