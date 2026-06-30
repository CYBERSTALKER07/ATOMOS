package payment

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// ErrGatewayMismatch is returned when a session exists but the requested gateway differs.
var ErrGatewayMismatch = errors.New("gateway mismatch")

// OrderCheckoutReader loads persisted order totals for retailer payment flows.
type OrderCheckoutReader interface {
	CheckoutSnapshot(ctx context.Context, orderID, retailerID string) (totalMinor int64, currency string, err error)
	CheckoutOrderContext(ctx context.Context, orderID, retailerID string) (order.CheckoutOrderContext, error)
}

// BindOrderCheckoutReader wires order totals for card/cash checkout handlers.
func (s *Service) BindOrderCheckoutReader(reader OrderCheckoutReader) {
	s.orderReader = reader
}

type retailerCardCheckoutRequest struct {
	OrderID     string `json:"order_id"`
	Gateway     string `json:"gateway"`
	Amount      int64  `json:"amount"`
	AmountMinor int64  `json:"amount_minor"`
	ReturnURL   string `json:"return_url"`
	InvoiceID   string `json:"invoice_id"`
}

type retailerCardCheckoutResponse struct {
	OrderID         string   `json:"order_id"`
	State           string   `json:"state"`
	Amount          int64    `json:"amount"`
	Gateway         string   `json:"gateway"`
	ResolvedGateway string   `json:"resolved_gateway,omitempty"`
	PolicySource    string   `json:"policy_source,omitempty"`
	AllowedGateways []string `json:"allowed_gateways,omitempty"`
	PolicyReason    string   `json:"policy_reason,omitempty"`
	PaymentURL      string   `json:"payment_url"`
	InvoiceID       string   `json:"invoice_id"`
	SessionID       string   `json:"session_id,omitempty"`
	AttemptID       string   `json:"attempt_id,omitempty"`
	RetailerID      string   `json:"retailer_id"`
	Message         string   `json:"message"`
}

type retailerCashCheckoutRequest struct {
	OrderID string `json:"order_id"`
}

type retailerCashCheckoutResponse struct {
	OrderID    string `json:"order_id"`
	State      string `json:"state"`
	Amount     int64  `json:"amount"`
	DriverID   string `json:"driver_id,omitempty"`
	RetailerID string `json:"retailer_id"`
	Message    string `json:"message"`
}

// HandleOrderCardCheckout serves POST /v1/order/card-checkout.
func (s *Service) HandleOrderCardCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "/v1/order/card-checkout", false, "")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "Unable to read request body", "/v1/order/card-checkout", false, "")
		return
	}
	defer r.Body.Close()

	if rec, ok := s.handleIdempotentHit(w, r, body); ok {
		_ = rec
		return
	}

	var req retailerCardCheckoutRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json_payload", "Invalid JSON payload", "/v1/order/card-checkout", false, "")
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	if req.OrderID == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "order_id is required", "/v1/order/card-checkout", false, "")
		return
	}

	retailerID := ""
	if claims, ok := auth.FromContext(r.Context()); ok && claims.Subject != "" {
		retailerID = claims.Subject
	}
	if retailerID == "" {
		writeJSONError(w, http.StatusUnprocessableEntity, "retailer_scope_missing", "retailer context is required", "/v1/order/card-checkout", false, "")
		return
	}

	amountMinor := req.AmountMinor
	if amountMinor <= 0 {
		amountMinor = req.Amount
	}
	currency := s.currency
	if amountMinor <= 0 && s.orderReader != nil {
		total, orderCurrency, snapErr := s.orderReader.CheckoutSnapshot(r.Context(), req.OrderID, retailerID)
		if snapErr != nil {
			s.writeOrderCheckoutError(w, "/v1/order/card-checkout", snapErr)
			return
		}
		amountMinor = total
		if strings.TrimSpace(orderCurrency) != "" {
			currency = orderCurrency
		}
	}
	if amountMinor <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "amount is required", "/v1/order/card-checkout", false, "")
		return
	}

	checkoutReq := CheckoutRequest{
		OrderID:     req.OrderID,
		RetailerID:  retailerID,
		Gateway:     req.Gateway,
		Currency:    currency,
		AmountMinor: amountMinor,
	}
	session, attempt, executionResult, err := s.initCheckoutSession(r.Context(), "CARD", checkoutReq)
	if err != nil {
		if errors.Is(err, ErrGatewayMismatch) {
			writeJSONError(w, http.StatusConflict, "gateway_mismatch", "cannot change gateway for an active session", "/v1/order/card-checkout", false, "")
			return
		}
		s.writeExecutionError(w, "/v1/order/card-checkout", err)
		return
	}

	invoiceID := strings.TrimSpace(req.InvoiceID)
	if invoiceID == "" {
		invoiceID = s.newID("inv")
	}
	resp := retailerCardCheckoutResponse{
		OrderID:         session.OrderID,
		State:           "PAYMENT_REQUIRED",
		Amount:          session.AmountMinor,
		Gateway:         req.Gateway,
		ResolvedGateway: session.Gateway,
		PolicySource:    executionResult.PolicySource,
		PaymentURL:      executionResult.RedirectURL,
		InvoiceID:       invoiceID,
		SessionID:       session.SessionID,
		AttemptID:       attempt.AttemptID,
		RetailerID:      retailerID,
		Message:         "payment session created",
	}
	if strings.TrimSpace(resp.PaymentURL) == "" {
		resp.PaymentURL = "/v1/payment/session/" + session.SessionID
	}
	if strings.TrimSpace(resp.Gateway) == "" {
		resp.Gateway = session.Gateway
	}

	respBytes, _ := json.Marshal(resp)
	s.persistIdempotencyRecord(r.Context(), r.Header.Get("Idempotency-Key"), sha256Hex(body), http.StatusOK, respBytes, 24*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

// HandleOrderCashCheckout serves POST /v1/order/cash-checkout.
func (s *Service) HandleOrderCashCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "/v1/order/cash-checkout", false, "")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "Unable to read request body", "/v1/order/cash-checkout", false, "")
		return
	}
	defer r.Body.Close()

	if rec, ok := s.handleIdempotentHit(w, r, body); ok {
		_ = rec
		return
	}

	var req retailerCashCheckoutRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json_payload", "Invalid JSON payload", "/v1/order/cash-checkout", false, "")
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	if req.OrderID == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "order_id is required", "/v1/order/cash-checkout", false, "")
		return
	}

	retailerID := ""
	if claims, ok := auth.FromContext(r.Context()); ok && claims.Subject != "" {
		retailerID = claims.Subject
	}
	if retailerID == "" {
		writeJSONError(w, http.StatusUnprocessableEntity, "retailer_scope_missing", "retailer context is required", "/v1/order/cash-checkout", false, "")
		return
	}

	amountMinor := int64(0)
	if s.orderReader != nil {
		total, _, snapErr := s.orderReader.CheckoutSnapshot(r.Context(), req.OrderID, retailerID)
		if snapErr != nil {
			s.writeOrderCheckoutError(w, "/v1/order/cash-checkout", snapErr)
			return
		}
		amountMinor = total
	}

	resp := retailerCashCheckoutResponse{
		OrderID:    req.OrderID,
		State:      "PENDING",
		Amount:     amountMinor,
		RetailerID: retailerID,
		Message:    "cash checkout accepted",
	}
	respBytes, _ := json.Marshal(resp)
	s.persistIdempotencyRecord(r.Context(), r.Header.Get("Idempotency-Key"), sha256Hex(body), http.StatusOK, respBytes, 24*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

func (s *Service) writeOrderCheckoutError(w http.ResponseWriter, endpoint string, err error) {
	switch {
	case errors.Is(err, order.ErrOrderNotFound):
		writeJSONError(w, http.StatusNotFound, "order_not_found", "order not found", endpoint, false, "")
	case errors.Is(err, order.ErrOrderForbidden):
		writeJSONError(w, http.StatusForbidden, "forbidden", "forbidden", endpoint, false, "")
	case errors.Is(err, order.ErrBackorderPaymentDeferred):
		writeJSONError(w, http.StatusUnprocessableEntity, "backorder_payment_deferred", "backorder payment is deferred until fulfillment", endpoint, false, "")
	case errors.Is(err, order.ErrPaymentBeforeDelivery):
		writeJSONError(w, http.StatusUnprocessableEntity, "payment_before_delivery_not_allowed", "payment is collected at delivery after offload", endpoint, false, "")
	default:
		writeJSONError(w, http.StatusInternalServerError, "order_lookup_failed", err.Error(), endpoint, false, "")
	}
}

func (s *Service) initCheckoutSession(ctx context.Context, mode string, req CheckoutRequest) (SessionRecord, PaymentAttemptRecord, ExecutionResult, error) {
	resolvedCurrency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if resolvedCurrency == "" {
		resolvedCurrency = s.currency
	}

	warehouseID := ""
	if s.orderReader != nil {
		if orderCtx, err := s.orderReader.CheckoutOrderContext(ctx, req.OrderID, req.RetailerID); err == nil {
			if orderCtx.Currency != "" && resolvedCurrency == s.currency {
				resolvedCurrency = strings.ToUpper(strings.TrimSpace(orderCtx.Currency))
			}
			warehouseID = orderCtx.WarehouseID
		}
	}

	policy := NormalizeGatewayPolicy(PaymentAcceptorSupplier, nil, "SUPPLIER_DEFAULT")
	if s.policy != nil {
		resolved, err := s.policy.Resolve(ctx, s.supplierID, warehouseID)
		if err != nil {
			return SessionRecord{}, PaymentAttemptRecord{}, ExecutionResult{}, err
		}
		policy = resolved
	}

	req.Gateway = policy.ResolveCardGateway(req.Gateway)
	if mode != "CASH" {
		if err := policy.ValidateCardGateway(req.Gateway); err != nil {
			return SessionRecord{}, PaymentAttemptRecord{}, ExecutionResult{}, err
		}
	}

	if existing, ok, err := s.repo.GetSessionByOrderID(ctx, req.OrderID); err == nil && ok {
		if existing.Status == "PAYMENT_REQUIRED" || existing.Status == "AWAITING_PAYMENT" {
			if existing.Gateway != "" && req.Gateway != "" && !strings.EqualFold(existing.Gateway, req.Gateway) {
				return SessionRecord{}, PaymentAttemptRecord{}, ExecutionResult{}, ErrGatewayMismatch
			}
		}
	}

	executionResult, err := s.execution.Execute(ctx, ExecutionRequest{
		Gateway:     req.Gateway,
		Action:      ExecutionActionCheckoutInit,
		OrderID:     req.OrderID,
		AmountMinor: req.AmountMinor,
		Currency:    resolvedCurrency,
	})
	if err != nil {
		return SessionRecord{}, PaymentAttemptRecord{}, ExecutionResult{}, err
	}
	if executionResult.PolicySource == "" || executionResult.PolicySource == "SUPPLIER_DEFAULT" {
		executionResult.PolicySource = policy.PolicySource
	}

	now := s.now()
	session := SessionRecord{
		SessionID:   s.newID("psess"),
		OrderID:     req.OrderID,
		SupplierID:  s.supplierID,
		RetailerID:  req.RetailerID,
		Gateway:     executionResult.ResolvedGateway,
		Currency:    resolvedCurrency,
		AmountMinor: req.AmountMinor,
		Mode:        mode,
		Status:      "PAYMENT_REQUIRED",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	attempt := PaymentAttemptRecord{
		AttemptID:         s.newID("pattempt"),
		SessionID:         session.SessionID,
		Gateway:           session.Gateway,
		ExecutionAction:   string(ExecutionActionCheckoutInit),
		ExecutionMode:     string(executionResult.Mode),
		ProviderReference: executionResult.ProviderRef,
		Status:            "INITIATED",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.repo.CreateSessionWithAttempt(ctx, session, attempt, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateSession, session.SessionID, events.TopicMain, events.FinanceEvent{
			BaseEvent:         events.BaseEvent{Type: events.EventPaymentRequired, Timestamp: now.Format(time.RFC3339Nano)},
			SessionID:         session.SessionID,
			AttemptID:         attempt.AttemptID,
			OrderID:           session.OrderID,
			SupplierID:        session.SupplierID,
			RetailerID:        session.RetailerID,
			Gateway:           session.Gateway,
			Status:            session.Status,
			ExecutionAction:   string(ExecutionActionCheckoutInit),
			ExecutionMode:     string(executionResult.Mode),
			PolicySource:      executionResult.PolicySource,
			ProviderReference: executionResult.ProviderRef,
			AmountMinor:       session.AmountMinor,
			Currency:          session.Currency,
			Source:            "payment.checkout",
		})
	}); err != nil {
		return SessionRecord{}, PaymentAttemptRecord{}, ExecutionResult{}, err
	}
	s.cache.Invalidate(ctx, paymentOrderKey(session.OrderID), paymentRetailerKey(session.RetailerID), paymentSessionKey(session.SessionID))
	return session, attempt, executionResult, nil
}
