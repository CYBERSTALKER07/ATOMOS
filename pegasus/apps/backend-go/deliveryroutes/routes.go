// Package deliveryroutes owns the /v1/delivery/* surface — nine driver-facing
// endpoints that run at the retailer handoff: arrival marker, QR/offload
// handshake variants, shop-closed protocol, negotiation, credit-delivery,
// missing-items claim, split-payment, and the SMS fallback confirmation.
//
// Handler bodies live in backend-go/order — this package is a thin composer
// that mounts them under DRIVER-role guards with the caller-supplied
// observability middleware. The sole inline closure (/v1/delivery/arrive) is
// owned here because it orchestrates three collaborators (OrderService,
// FleetHub, Cache) rather than a single domain service.
//
// V.O.I.D. Wave B adoption notes:
//   - slog + TraceID: the arrive handler emits structured logs keyed by
//     trace_id (X-Request-Id / X-Trace-Id header, empty when absent).
//   - outbox.Emit: none of the nine routes contain an inline Kafka producer.
//     Kafka emission for state changes lives inside OrderService methods and
//     is tracked for migration under Known Gap §7 (progressive adoption).
//   - cache.Invalidate: arrive intentionally does NOT invalidate the delivery
//     token — RefreshDeliveryTokenTTL keeps it live through the handoff
//     window. For the delegated handlers, invalidation belongs at the
//     Spanner-commit site inside the order package (Known Gap §7).
package deliveryroutes

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/order"
	"backend-go/telemetry"
)

// Middleware is the handler-wrap contract supplied by the caller (typically
// main.loggingMiddleware).
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to mount /v1/delivery/*.
// Pointer fields to the shared *Deps structs match the order package's
// existing handler signatures and avoid copying the function-valued
// dispatchers on every request.
type Deps struct {
	Order             *order.OrderService
	Cache             *cache.Cache
	FleetHub          *telemetry.Hub
	ShopClosedDeps    *order.ShopClosedDeps
	EarlyCompleteDeps *order.EarlyCompleteDeps
	NegotiationDeps   *order.NegotiationDeps
	Log               Middleware
}

// RegisterRoutes mounts the delivery surface:
//
//	POST /v1/delivery/arrive                  — mark arrival at shop
//	POST /v1/delivery/confirm-payment-bypass  — driver confirms with bypass token
//	POST /v1/delivery/sms-complete            — SMS-gateway dead-phone fallback
//	POST /v1/delivery/shop-closed             — driver reports shop closed
//	POST /v1/delivery/bypass-offload          — driver uses bypass token for offload
//	POST /v1/delivery/negotiate               — driver proposes quantity negotiation
//	POST /v1/delivery/credit-delivery         — delivered on credit
//	POST /v1/delivery/missing-items           — missing items after seal (driver or payloader)
//	POST /v1/delivery/split-payment           — partial cash + credit
func RegisterRoutes(r chi.Router, d Deps) {
	svc := d.Order
	log := d.Log
	driver := []string{"DRIVER"}
	driverOrPayloader := []string{"DRIVER", "PAYLOADER"}

	// Arrival orchestration lives inline — it spans three collaborators.
	r.HandleFunc("/v1/delivery/arrive",
		auth.RequireRole(driver, log(handleArrive(d))))

	// Edge 5: Driver confirms with bypass token
	r.HandleFunc("/v1/delivery/confirm-payment-bypass",
		auth.RequireRole(driver, log(order.HandleConfirmPaymentBypass(svc))))

	// Edge 23: SMS gateway webhook for dead-phone delivery confirm (unauthenticated).
	r.HandleFunc("/v1/delivery/sms-complete",
		log(order.HandleSMSComplete(svc)))

	// P0: Driver reports shop closed
	r.HandleFunc("/v1/delivery/shop-closed",
		auth.RequireRole(driver, log(svc.HandleReportShopClosed(d.ShopClosedDeps))))

	// P0: Driver uses bypass token for offload
	r.HandleFunc("/v1/delivery/bypass-offload",
		auth.RequireRole(driver, log(svc.HandleBypassOffload(d.ShopClosedDeps))))

	// Edge 28: Driver proposes quantity negotiation
	r.HandleFunc("/v1/delivery/negotiate",
		auth.RequireRole(driver, log(order.HandleProposeNegotiation(svc, d.NegotiationDeps))))

	// Edge 32: Driver marks order as delivered on credit
	r.HandleFunc("/v1/delivery/credit-delivery",
		auth.RequireRole(driver, log(order.HandleCreditDelivery(svc, d.EarlyCompleteDeps))))

	// Edge 33: Driver reports missing items after seal
	r.HandleFunc("/v1/delivery/missing-items",
		auth.RequireRole(driverOrPayloader, log(order.HandleMissingItems(svc, d.EarlyCompleteDeps))))

	// Edge 35: Driver creates split payment
	r.HandleFunc("/v1/delivery/split-payment",
		auth.RequireRole(driver, log(order.HandleSplitPayment(svc))))
}

// handleArrive moves IN_TRANSIT → ARRIVED, extends the delivery-token TTL so
// the handoff window stays warm, and pushes ORDER_STATE_CHANGED to the
// supplier admin portal over the FleetHub WebSocket relay.
func handleArrive(d Deps) http.HandlerFunc {
	svc := d.Order
	hub := d.FleetHub
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		tid := traceID(r)
		var req struct {
			OrderID string `json:"order_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, "Invalid JSON body or missing order_id", http.StatusBadRequest)
			return
		}
		supplierID, err := svc.MarkArrived(r.Context(), req.OrderID)
		if err != nil {
			slog.ErrorContext(r.Context(), "delivery.arrive: MarkArrived failed",
				"trace_id", tid, "order_id", req.OrderID, "error", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		go svc.RefreshDeliveryTokenTTL(context.Background(), req.OrderID)
		if hub != nil && supplierID != "" {
			go hub.BroadcastOrderStateChange(supplierID, req.OrderID, "ARRIVED", "")
		}
		slog.InfoContext(r.Context(), "delivery.arrive",
			"trace_id", tid, "order_id", req.OrderID, "supplier_id", supplierID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"status":"ARRIVED","order_id":%q}`, req.OrderID)
	}
}

// traceID extracts the correlation token stashed by TraceMiddleware.
func traceID(r *http.Request) string {
	return telemetry.TraceIDFromContext(r.Context())
}
