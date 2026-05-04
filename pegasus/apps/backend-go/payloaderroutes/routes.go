// Package payloaderroutes owns the /v1/payloader/* surface that serves the
// Warehouse-staff Payloader app. Handlers live in backend-go/supplier and
// backend-go/fleet.
package payloaderroutes

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	"backend-go/fleet"
	"backend-go/idempotency"
	"backend-go/order"
	"backend-go/proximity"
	"backend-go/supplier"
	"backend-go/ws"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to register /v1/payloader routes.
type Deps struct {
	Spanner      *spanner.Client
	ReadRouter   proximity.ReadRouter
	Order        *order.OrderService
	RetailerHub  *ws.RetailerHub
	PayloaderHub *ws.PayloaderHub
	Log          Middleware
}

// RegisterRoutes mounts the payloader-facing surface:
//
//	GET  /v1/payloader/trucks             — vehicles for the payloader's supplier
//	GET  /v1/payloader/orders             — orders scoped to the payloader's vehicles
//	POST /v1/payloader/recommend-reassign — GPS-based truck recommendations
//	POST /v1/payload/seal                 — payload seal + DISPATCHED transition
//	GET  /v1/ws/payloader                 — payloader realtime websocket
func RegisterRoutes(r chi.Router, d Deps) {
	s := d.Spanner
	log := d.Log
	payloader := []string{"PAYLOADER"}
	payloaderSupplyAdmin := []string{"PAYLOADER", "ADMIN", "SUPPLIER"}

	r.HandleFunc("/v1/payloader/trucks",
		auth.RequireRole(payloader, log(supplier.HandlePayloaderTrucks(s))))
	r.HandleFunc("/v1/payloader/orders",
		auth.RequireRole(payloader, log(supplier.HandlePayloaderOrders(s))))
	r.HandleFunc("/v1/payloader/recommend-reassign",
		auth.RequireRole(payloaderSupplyAdmin, log(idempotency.Guard(fleet.HandleRecommendReassign(s, d.ReadRouter)))))
	r.HandleFunc("/v1/payload/seal",
		auth.RequireRole(payloaderSupplyAdmin, log(idempotency.Guard(handlePayloadSeal(d.Order, d.RetailerHub)))))
	if d.PayloaderHub != nil {
		r.HandleFunc("/v1/ws/payloader",
			auth.RequireRole(payloaderSupplyAdmin, d.PayloaderHub.HandleConnection))
	}
}

func handlePayloadSeal(orderSvc *order.OrderService, retailerHub *ws.RetailerHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if orderSvc == nil {
			http.Error(w, "Payload seal service unavailable", http.StatusServiceUnavailable)
			return
		}

		var req order.PayloadSealRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		retailerID, err := orderSvc.SealPayload(r.Context(), req)
		if err != nil {
			if strings.Contains(err.Error(), "bad request") {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if strings.Contains(err.Error(), "conflict") {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, "Internal Server Error during Payload Seal", http.StatusInternalServerError)
			return
		}

		if retailerHub != nil && retailerID != "" {
			go retailerHub.PushToRetailer(retailerID, map[string]interface{}{
				"type":      ws.EventOrderStatusChanged,
				"order_id":  req.OrderID,
				"state":     "DISPATCHED",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		dispatchCode := order.GenerateSecureToken()
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":        "PAYLOAD_SEALED_AND_DISPATCHED",
			"dispatch_code": dispatchCode,
			"order_id":      req.OrderID,
		})
	}
}
