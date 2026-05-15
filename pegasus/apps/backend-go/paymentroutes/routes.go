// Package paymentroutes owns the authenticated /v1/checkout/* and
// /v1/payment/* surface. Gateway-facing webhooks (/v1/webhooks/*) live in
// backend-go/webhookroutes — this package only hosts the surfaces a
// principal calls directly with a JWT.
package paymentroutes

import (
	"encoding/json"
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
		guard(auth.RequireRole(adminSupplier, log(idem(handleChargeback(d.Chargeback))))))
	r.HandleFunc("/v1/payment/chargeback/reversal",
		guard(auth.RequireRole(adminSupplier, log(idem(handleReversal(d.Chargeback))))))
}

// handleChargeback — POST /v1/payment/chargeback. Behaviour preserved
// verbatim from the inline closure it replaced.
func handleChargeback(cs *payment.ChargebackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writePaymentRouteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "/v1/payment/chargeback", false, "")
			return
		}
		var req struct {
			OrderID    string `json:"order_id"`
			RetailerID string `json:"retailer_id"`
			Gateway    string `json:"gateway"`
			Amount     int64  `json:"amount"`
			Currency   string `json:"currency,omitempty"`
			AmountUZS  int64  `json:"amount_uzs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writePaymentRouteError(w, http.StatusBadRequest, "invalid_json_payload", "Invalid JSON payload", "/v1/payment/chargeback", false, "")
			return
		}
		amount := req.Amount
		if amount <= 0 {
			amount = req.AmountUZS
		}
		if req.OrderID == "" || req.RetailerID == "" || req.Gateway == "" || amount <= 0 {
			writePaymentRouteError(w, http.StatusBadRequest, "invalid_request", "order_id, retailer_id, gateway, and amount (or amount_uzs) are required", "/v1/payment/chargeback", false, "")
			return
		}
		if err := cs.HandleChargeback(r.Context(), req.OrderID, req.RetailerID, req.Gateway, amount, req.Currency); err != nil {
			writePaymentRouteError(w, http.StatusInternalServerError, "chargeback_record_failed", err.Error(), "/v1/payment/chargeback", false, "")
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
			writePaymentRouteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "/v1/payment/chargeback/reversal", false, "")
			return
		}
		var req struct {
			SessionID string `json:"session_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writePaymentRouteError(w, http.StatusBadRequest, "invalid_json_payload", "Invalid JSON payload", "/v1/payment/chargeback/reversal", false, "")
			return
		}
		if req.SessionID == "" {
			writePaymentRouteError(w, http.StatusBadRequest, "invalid_request", "session_id is required", "/v1/payment/chargeback/reversal", false, "")
			return
		}
		if err := cs.HandleReversal(r.Context(), req.SessionID); err != nil {
			writePaymentRouteError(w, http.StatusBadRequest, "reversal_record_failed", err.Error(), "/v1/payment/chargeback/reversal", false, "")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "reversal_recorded"})
	}
}

func writePaymentRouteError(w http.ResponseWriter, status int, code, message, endpoint string, deprecated bool, migrateTo string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	payload := map[string]interface{}{
		// Keep "error" as human-readable text for backward compatibility.
		"error":      message,
		"code":       code,
		"message":    message,
		"endpoint":   endpoint,
		"deprecated": deprecated,
	}
	if migrateTo != "" {
		payload["migrate_to"] = migrateTo
	}
	_ = json.NewEncoder(w).Encode(payload)
}
