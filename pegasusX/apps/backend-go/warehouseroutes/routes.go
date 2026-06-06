// Package warehouseroutes mounts warehouse-role endpoints.
package warehouseroutes

import (
	"github.com/go-chi/chi/v5"
	"cloud.google.com/go/spanner"
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
	r.Post("/v1/auth/warehouse/refresh", d.Service.HandleWarehouseRefresh)

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
		rr.Get("/v1/warehouse/ops/orders", d.Service.HandleOrders)
		rr.Get("/v1/warehouse/ops/orders/*", d.Service.HandleOrders)
		if d.OrderService != nil {
			rr.Post("/v1/warehouse/ops/orders/{id}/delay", d.OrderService.HandleWarehouseMarkDelayed)
			rr.Post("/v1/warehouse/ops/orders/{id}/reject", d.OrderService.HandleWarehouseRejectOrder)
			rr.Post("/v1/warehouse/ops/orders/{id}/overflow", d.OrderService.HandleWarehousePayloadOverflow)
		}
		rr.Get("/v1/warehouse/ops/dispatch/preview", d.Service.HandleDispatchPreview)
		rr.Post("/v1/warehouse/ops/dispatch/preview", d.Service.HandleDispatchPreview)
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
		rr.Get("/v1/warehouse/ops/analytics", d.Service.HandleOpsAnalytics)
		rr.Get("/v1/warehouse/ops/crm", d.Service.HandleOpsCRM)
		rr.Get("/v1/warehouse/ops/returns", d.Service.HandleOpsReturns)
		rr.Get("/v1/warehouse/ops/treasury", d.Service.HandleOpsTreasury)
		rr.Get("/v1/warehouse/ops/financials", d.Service.HandleOpsFinancials)
		rr.Get("/v1/warehouse/ops/payment-config", d.Service.HandleOpsPaymentConfig)
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
