// Package driverroutes mounts driver-role endpoints.
package driverroutes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/driver"
)

// Deps is the narrow dependency contract for driver routes.
type Deps struct {
	Service      *driver.Service
	WarehouseSvc interface {
		HandleDriverListSupplyTransfers(http.ResponseWriter, *http.Request)
		HandleDriverArriveSupplyTransfer(http.ResponseWriter, *http.Request)
	}
	OrderService interface {
		HandleReportShopClosed(http.ResponseWriter, *http.Request)
		HandleProximityUnlock(http.ResponseWriter, *http.Request)
		HandlePartialOffload(http.ResponseWriter, *http.Request)
		HandleCreditLeave(http.ResponseWriter, *http.Request)
		HandleConfirmPaymentBypass(http.ResponseWriter, *http.Request)
		HandleRequestEarlyComplete(http.ResponseWriter, *http.Request)
		HandleProposeNegotiation(http.ResponseWriter, *http.Request)
		HandleFleetRouteReorder(http.ResponseWriter, *http.Request)
		HandleBypassOffload(http.ResponseWriter, *http.Request)
		HandleCreditDelivery(http.ResponseWriter, *http.Request)
		HandleMissingItems(http.ResponseWriter, *http.Request)
		HandleExceptionReport(http.ResponseWriter, *http.Request)
		HandleSplitPayment(http.ResponseWriter, *http.Request)
		HandleValidateQR(http.ResponseWriter, *http.Request)
		HandleAmendOrder(http.ResponseWriter, *http.Request)
		HandleReassignHandshake(http.ResponseWriter, *http.Request)
	}
	AllowAuthBypass bool
}

// RegisterRoutes mounts driver role-row endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	r.Post("/v1/auth/driver/login", d.Service.HandleDriverLogin)

	mountEcosystemCRUD := func(rr chi.Router) {
		rr.Post("/v1/drivers", d.Service.HandleCreateDriver)
		rr.Get("/v1/drivers/{driverId}", d.Service.HandleGetDriver)
		rr.Put("/v1/drivers/{driverId}", d.Service.HandleUpdateDriver)
		rr.Get("/v1/drivers", d.Service.HandleListDrivers)

		rr.Post("/v1/vehicles", d.Service.HandleCreateVehicle)
		rr.Get("/v1/vehicles/{vehicleId}", d.Service.HandleGetVehicle)
		rr.Put("/v1/vehicles/{vehicleId}", d.Service.HandleUpdateVehicle)
		rr.Get("/v1/vehicles", d.Service.HandleListVehicles)
	}

	mountProtected := func(rr chi.Router) {
		rr.Get("/v1/driver/profile", d.Service.HandleProfile)
		rr.Get("/v1/driver/history", d.Service.HandleHistory)
		rr.Get("/v1/driver/earnings", d.Service.HandleEarnings)
		rr.Get("/v1/driver/availability", d.Service.HandleAvailability)
		rr.Patch("/v1/driver/availability", d.Service.HandleAvailability)
		rr.Post("/v1/driver/availability", d.Service.HandleAvailability)
		rr.Post("/v1/driver/ops/rescue/request", d.Service.HandleRescueRequest)
		rr.Post("/v1/driver/ops/rescue/respond", d.Service.HandleRescueRespond)
		rr.Get("/v1/driver/pending-collections", d.Service.HandlePendingCollections)
		rr.Get("/v1/driver/open-fiscal", d.Service.HandleOpenFiscal)
		rr.Get("/v1/driver/manifest-gate", d.Service.HandleManifestGate)
		rr.Get("/v1/driver/manifest", d.Service.HandleManifest)
		rr.Get("/v1/fleet/manifest", d.Service.HandleManifest)

		rr.Get("/v1/fleet/orders", d.Service.HandleFleetOrders)
		rr.Get("/v1/fleet/route/{routeID}/geometry", d.Service.HandleRouteGeometry)
		rr.Post("/v1/fleet/driver/depart", d.Service.HandleDriverDepart)
		rr.Post("/v1/fleet/driver/return-complete", d.Service.HandleDriverReturnComplete)

		if d.WarehouseSvc != nil {
			rr.Get("/v1/driver/supply-transfers", d.WarehouseSvc.HandleDriverListSupplyTransfers)
			rr.Post("/v1/driver/supply-transfers/{id}/arrive", d.WarehouseSvc.HandleDriverArriveSupplyTransfer)
		}

		// Order & Delivery lifecycle proxies (if OrderService is wired)
		if d.OrderService != nil {
			rr.Post("/v1/fleet/route/reorder", d.OrderService.HandleFleetRouteReorder)
			rr.Post("/v1/fleet/route/request-early-complete", d.OrderService.HandleRequestEarlyComplete)
			rr.Get("/v1/orders/{orderID}", d.Service.HandleOrderGet)
			rr.Patch("/v1/orders/{orderID}/state", d.Service.HandleOrderStatePatch)
			rr.Post("/v1/order/validate-qr", d.OrderService.HandleValidateQR)
			rr.Post("/v1/order/amend", d.OrderService.HandleAmendOrder)
			rr.Post("/v1/fleet/orders/{orderID}/reassign-handshake", d.OrderService.HandleReassignHandshake)
			rr.Post("/v1/driver/orders/{orderId}/shop-closed", d.OrderService.HandleReportShopClosed)
			rr.Post("/v1/driver/orders/{orderId}/partial-offload", d.OrderService.HandlePartialOffload)
			rr.Post("/v1/driver/orders/{orderId}/credit-leave", d.OrderService.HandleCreditLeave)
			rr.Post("/v1/delivery/proximity-unlock", d.OrderService.HandleProximityUnlock)
			rr.Post("/v1/delivery/bypass-offload", d.OrderService.HandleBypassOffload)
			rr.Post("/v1/ws/ack", d.Service.HandleWSAck)
			// Quantity negotiation product-disabled → 410 feature_disabled.
			rr.Post("/v1/delivery/negotiate", d.OrderService.HandleProposeNegotiation)
			rr.Post("/v1/delivery/credit-delivery", d.OrderService.HandleCreditDelivery)
			rr.Post("/v1/delivery/missing-items", d.OrderService.HandleMissingItems)
			rr.Post("/v1/delivery/exception-report", d.OrderService.HandleExceptionReport)
			rr.Post("/v1/delivery/split-payment", d.OrderService.HandleSplitPayment)
			rr.Post("/v1/delivery/confirm-payment-bypass", d.OrderService.HandleConfirmPaymentBypass)
		} else {
			// In production, OrderService MUST be wired to support these edges.
			// Firing a panic here guarantees tests and staging don't silently degraded to stubs.
			panic("driverroutes: OrderService is nil. Must be wired for delivery mutations")
		}
	}

	auth.ProtectMutations(r, auth.MutationGuardConfig{
		AllowBypass: d.AllowAuthBypass,
	}, func(gr chi.Router) {
		gr.Use(auth.RequireRole(auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleFactoryAdmin))
		mountEcosystemCRUD(gr)
	})

	auth.ProtectMutations(r, auth.MutationGuardConfig{
		AllowBypass: d.AllowAuthBypass,
	}, func(gr chi.Router) {
		gr.Use(auth.RequireRole(auth.RoleDriver, auth.RoleFactoryDriver))
		mountProtected(gr)
	})
}
