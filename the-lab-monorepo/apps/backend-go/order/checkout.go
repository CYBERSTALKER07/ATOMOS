// Package order — B2B Checkout Bridge (Vector G)
// HandleB2BCheckout orchestrates: cart pricing → CreateOrder → HTTP 201.
package order

import (
	"encoding/json"
	"log"
	"net/http"

	"backend-go/cart"
)

// B2BCheckoutRequest is the payload from the Retailer App's [ AUTHORIZE PROCUREMENT ] button.
type B2BCheckoutRequest struct {
	RetailerID     string               `json:"retailer_id"`
	PaymentGateway string               `json:"payment_gateway"` // "CLICK" | "PAYME" | "UZCARD"
	Latitude       float64              `json:"latitude"`
	Longitude      float64              `json:"longitude"`
	Items          []cart.OrderLineItem `json:"items"`
}

// B2BCheckoutResponse is returned on HTTP 201.
type B2BCheckoutResponse struct {
	Status  string `json:"status"`
	OrderID string `json:"order_id"`
	Total   int64  `json:"total"`
}

// HandleB2BCheckout handles POST /v1/checkout/b2b.
// Flow: decode → price → create order → respond.
func (s *OrderService) HandleB2BCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req B2BCheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Malformed checkout payload", http.StatusBadRequest)
		return
	}

	// ── Validation ────────────────────────────────────────────────────────────
	if req.RetailerID == "" {
		http.Error(w, "retailer_id is required", http.StatusUnprocessableEntity)
		return
	}
	if len(req.Items) == 0 {
		http.Error(w, "items must not be empty", http.StatusUnprocessableEntity)
		return
	}
	for _, item := range req.Items {
		if item.Quantity <= 0 || item.UnitPrice <= 0 {
			http.Error(w, "each item must have positive quantity and unit_price", http.StatusUnprocessableEntity)
			return
		}
	}

	ctx := r.Context()

	// ── Step 1: Price the cart ─────────────────────────────────────────────
	total, err := cart.CalculateB2BTotal(ctx, s.Client, req.Items)
	if err != nil {
		log.Printf("[ERROR] cart.CalculateB2BTotal: %v", err)
		http.Error(w, "Pricing engine error", http.StatusInternalServerError)
		return
	}

	// ── Step 2: Persist the order ──────────────────────────────────────────
	orderID, err := s.CreateOrder(ctx, CreateOrderRequest{
		RetailerID:     req.RetailerID,
		Amount:         total,
		PaymentGateway: req.PaymentGateway,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		OrderSource:    "B2B_CHECKOUT",
		State:          "PENDING",
	})
	if err != nil {
		log.Printf("[ERROR] order.CreateOrder (B2B): %v", err)
		http.Error(w, "Order creation failed", http.StatusInternalServerError)
		return
	}

	log.Printf("[B2B_CHECKOUT] OrderID=%s RetailerId=%s Total=%d Gateway=%s",
		orderID, req.RetailerID, total, req.PaymentGateway)

	// ── Step 3: Respond ────────────────────────────────────────────────────
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(B2BCheckoutResponse{
		Status:  "CHECKOUT_LOCKED",
		OrderID: orderID,
		Total:   total,
	}); err != nil {
		log.Printf("[ERROR] B2B response encode: %v", err)
	}
}
