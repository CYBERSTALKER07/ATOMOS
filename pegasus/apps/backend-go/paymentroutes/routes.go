// Package paymentroutes owns the authenticated /v1/checkout/* and
// /v1/payment/* surface. Gateway-facing webhooks (/v1/webhooks/*) live in
// backend-go/webhookroutes — this package only hosts the surfaces a
// principal calls directly with a JWT.
package paymentroutes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	"backend-go/payment"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Checkout is the narrow interface /v1/checkout/* needs — supplied by
// order.OrderService in main.
type Checkout interface {
	HandleB2BCheckout(w http.ResponseWriter, r *http.Request)
	HandleUnifiedCheckout(w http.ResponseWriter, r *http.Request)
}

// Deps bundles the collaborators required to register payment routes.
type Deps struct {
	Spanner       *spanner.Client
	Checkout      Checkout
	Chargeback    *payment.ChargebackService
	Log           Middleware
	PriorityGuard Middleware
	Idempotency   Middleware
}

// RegisterRoutes mounts:
//
//	POST /v1/checkout/b2b              — retailer procurement checkout
//	POST /v1/checkout/unified          — cart fan-out across suppliers
//	POST /v1/payment/chargeback        — record provider-initiated chargeback
//	POST /v1/payment/chargeback/reversal — reverse a settled payment session
//	POST /v1/payment/global_pay/initiate    — DEPRECATED direct GlobalPay initiation
func RegisterRoutes(r chi.Router, d Deps) {
	log := d.Log
	guard := d.PriorityGuard
	idem := d.Idempotency
	retailer := []string{"RETAILER"}
	adminSupplier := []string{"ADMIN", "SUPPLIER"}

	r.HandleFunc("/v1/checkout/b2b",
		guard(auth.RequireRole(retailer, log(idem(d.Checkout.HandleB2BCheckout)))))
	r.HandleFunc("/v1/checkout/unified",
		guard(auth.RequireRole(retailer, log(idem(d.Checkout.HandleUnifiedCheckout)))))

	r.HandleFunc("/v1/payment/chargeback",
		auth.RequireRole(adminSupplier, log(handleChargeback(d.Chargeback))))
	r.HandleFunc("/v1/payment/chargeback/reversal",
		auth.RequireRole(adminSupplier, log(handleReversal(d.Chargeback))))
	r.HandleFunc("/v1/payment/global_pay/initiate",
		auth.RequireRole(retailer, log(handleGlobalPayInitiate(d.Spanner))))
}

// handleChargeback — POST /v1/payment/chargeback. Behaviour preserved
// verbatim from the inline closure it replaced.
func handleChargeback(cs *payment.ChargebackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			OrderID    string `json:"order_id"`
			RetailerID string `json:"retailer_id"`
			Gateway    string `json:"gateway"`
			AmountUZS  int64  `json:"amount_uzs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		if req.OrderID == "" || req.RetailerID == "" || req.Gateway == "" || req.AmountUZS <= 0 {
			http.Error(w, `{"error":"order_id, retailer_id, gateway, amount_uzs are required"}`, http.StatusBadRequest)
			return
		}
		if err := cs.HandleChargeback(r.Context(), req.OrderID, req.RetailerID, req.Gateway, req.AmountUZS); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "chargeback_recorded"})
	}
}

// handleReversal — POST /v1/payment/chargeback/reversal.
func handleReversal(cs *payment.ChargebackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			SessionID string `json:"session_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		if req.SessionID == "" {
			http.Error(w, `{"error":"session_id is required"}`, http.StatusBadRequest)
			return
		}
		if err := cs.HandleReversal(r.Context(), req.SessionID); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "reversal_recorded"})
	}
}

// handleGlobalPayInitiate — DEPRECATED POST /v1/payment/global_pay/initiate. Clients
// should migrate to POST /v1/order/card-checkout. Retained for backward
// compatibility with older iOS/Android builds.
func handleGlobalPayInitiate(sc *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		log.Printf("[DEPRECATED] /v1/payment/global_pay/initiate called — clients should migrate to /v1/order/card-checkout")

		var req struct {
			OrderID   string `json:"order_id"`
			InvoiceID string `json:"invoice_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, `{"error":"order_id is required"}`, http.StatusBadRequest)
			return
		}

		row, err := sc.Single().ReadRow(r.Context(), "Orders", spanner.Key{req.OrderID}, []string{"Amount", "State"})
		if err != nil {
			log.Printf("[PAYMENT INITIATE] Order %s not found: %v", req.OrderID, err)
			http.Error(w, `{"error":"order not found"}`, http.StatusNotFound)
			return
		}
		var amount spanner.NullInt64
		var state string
		if err := row.Columns(&amount, &state); err != nil {
			http.Error(w, `{"error":"failed to read order"}`, http.StatusInternalServerError)
			return
		}

		gw, err := payment.NewGatewayClient("GLOBAL_PAY")
		if err != nil {
			log.Printf("[PAYMENT INITIATE] GlobalPay client init failed: %v", err)
			http.Error(w, `{"error":"payment gateway unavailable"}`, http.StatusServiceUnavailable)
			return
		}

		if err := gw.Charge(req.OrderID, amount.Int64); err != nil {
			log.Printf("[PAYMENT INITIATE] GlobalPay charge failed for %s: %v", req.OrderID, err)
			http.Error(w, fmt.Sprintf(`{"error":"charge failed: %s"}`, err.Error()), http.StatusBadGateway)
			return
		}

		log.Printf("[PAYMENT INITIATE] GlobalPay charge initiated for order %s: %d", req.OrderID, amount.Int64)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":   "INITIATED",
			"order_id": req.OrderID,
		})
	}
}
