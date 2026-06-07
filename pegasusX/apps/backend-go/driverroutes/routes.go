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
	OrderService interface {
		HandleReportShopClosed(http.ResponseWriter, *http.Request)
		HandleConfirmPaymentBypass(http.ResponseWriter, *http.Request)
		HandleRequestEarlyComplete(http.ResponseWriter, *http.Request)
		HandleProposeNegotiation(http.ResponseWriter, *http.Request)
		HandleFleetRouteReorder(http.ResponseWriter, *http.Request)
		HandleBypassOffload(http.ResponseWriter, *http.Request)
		HandleCreditDelivery(http.ResponseWriter, *http.Request)
		HandleMissingItems(http.ResponseWriter, *http.Request)
		HandleSplitPayment(http.ResponseWriter, *http.Request)
	}
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
	AllowAuthBypass     bool
}

// RegisterRoutes mounts driver role-row endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	r.Post("/v1/auth/driver/login", d.Service.HandleDriverLogin)

	mountProtected := func(rr chi.Router) {
		rr.Get("/v1/driver/profile", d.Service.HandleProfile)
		rr.Get("/v1/driver/history", d.Service.HandleHistory)
		rr.Get("/v1/driver/earnings", d.Service.HandleEarnings)
		rr.Get("/v1/driver/availability", d.Service.HandleAvailability)
		rr.Patch("/v1/driver/availability", d.Service.HandleAvailability)
		rr.Post("/v1/driver/availability", d.Service.HandleAvailability)
		rr.Get("/v1/driver/pending-collections", d.Service.HandlePendingCollections)
		rr.Get("/v1/driver/manifest-gate", d.Service.HandleManifestGate)
		rr.Get("/v1/driver/manifest", d.Service.HandleManifest)
		rr.Get("/v1/fleet/manifest", d.Service.HandleManifest)

		rr.Get("/v1/fleet/orders", d.Service.HandleFleetOrders)
		rr.Post("/v1/fleet/driver/depart", d.Service.HandleDriverDepart)
		rr.Post("/v1/fleet/driver/return-complete", d.Service.HandleDriverReturnComplete)
		if d.OrderService != nil {
			rr.Post("/v1/fleet/route/reorder", d.OrderService.HandleFleetRouteReorder)
		} else {
			rr.Post("/v1/fleet/route/reorder", d.Service.HandleFleetRouteReorder)
		}
		if d.OrderService != nil {
			rr.Post("/v1/fleet/route/request-early-complete", d.OrderService.HandleRequestEarlyComplete)
		} else {
			rr.Post("/v1/fleet/route/request-early-complete", d.Service.HandleFleetEarlyComplete)
		}

		rr.Get("/v1/orders/{orderID}", d.Service.HandleOrderGet)
		rr.Patch("/v1/orders/{orderID}/state", d.Service.HandleOrderStatePatch)
		rr.Post("/v1/order/validate-qr", d.Service.HandleOrderValidateQR)
		rr.Post("/v1/order/amend", d.Service.HandleOrderAmend)
		if d.OrderService != nil {
			rr.Post("/v1/delivery/shop-closed", d.OrderService.HandleReportShopClosed)
		} else {
			rr.Post("/v1/delivery/shop-closed", d.Service.HandleDeliveryShopClosed)
		}
		if d.OrderService != nil {
			rr.Post("/v1/delivery/bypass-offload", d.OrderService.HandleBypassOffload)
		} else {
			rr.Post("/v1/delivery/bypass-offload", d.Service.HandleDeliveryBypass)
		}
		if d.OrderService != nil {
			rr.Post("/v1/delivery/confirm-payment-bypass", d.OrderService.HandleConfirmPaymentBypass)
		} else {
			rr.Post("/v1/delivery/confirm-payment-bypass", d.Service.HandleDeliveryBypass)
		}

		rr.Post("/v1/ws/ack", d.Service.HandleWSAck)

		rr.Get("/v1/user/notifications", d.Service.HandleUserNotifications)
		rr.Post("/v1/user/notifications/read", d.Service.HandleMarkNotificationsRead)
		if d.OrderService != nil {
			rr.Post("/v1/delivery/negotiate", d.OrderService.HandleProposeNegotiation)
		} else {
			rr.Post("/v1/delivery/negotiate", d.Service.HandleDeliveryCompatOK)
		}
		if d.OrderService != nil {
			rr.Post("/v1/delivery/credit-delivery", d.OrderService.HandleCreditDelivery)
			rr.Post("/v1/delivery/missing-items", d.OrderService.HandleMissingItems)
			rr.Post("/v1/delivery/split-payment", d.OrderService.HandleSplitPayment)
		} else {
			rr.Post("/v1/delivery/credit-delivery", d.Service.HandleDeliveryCompatOK)
			rr.Post("/v1/delivery/missing-items", d.Service.HandleDeliveryCompatOK)
			rr.Post("/v1/delivery/split-payment", d.Service.HandleDeliveryCompatOK)
		}
	}

	auth.ProtectMutations(r, auth.MutationGuardConfig{
		FirebaseEnabled:  d.FirebaseAuthEnabled,
		FirebaseVerifier: d.FirebaseVerifier,
		AllowBypass:      d.AllowAuthBypass,
	}, func(gr chi.Router) {
		gr.Use(auth.RequireRole(auth.RoleDriver))
		mountProtected(gr)
	})
}
