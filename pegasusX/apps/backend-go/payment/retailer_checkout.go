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

// ErrCurrencyMismatch is returned when request currency differs from the order currency.
var ErrCurrencyMismatch = errors.New("currency_mismatch")

// OrderCheckoutReader loads persisted order totals for retailer payment flows.
type OrderCheckoutReader interface {
	CheckoutSnapshot(ctx context.Context, orderID, retailerID string) (totalMinor int64, currency string, err error)
	CheckoutOrderContext(ctx context.Context, orderID, retailerID string) (order.CheckoutOrderContext, error)
}

// OrderCashSelector marks an order as cash-at-delivery (durable Spanner + outbox).
// Implemented by order.Service.SelectCashAtDelivery.
type OrderCashSelector interface {
	SelectCashAtDelivery(ctx context.Context, orderID, retailerID, actorID string) (status string, amountMinor int64, currency string, err error)
}

// BindOrderCheckoutReader wires order totals for card/cash checkout handlers.
func (s *Service) BindOrderCheckoutReader(reader OrderCheckoutReader) {
	s.orderReader = reader
}

// BindOrderCashSelector wires durable cash selection (B1 M-P0-5).
func (s *Service) BindOrderCashSelector(sel OrderCashSelector) {
	s.orderCash = sel
}

type retailerCardCheckoutRequest struct {
	OrderID     string `json:"order_id"`
	Gateway     string `json:"gateway"`
	Amount      int64  `json:"amount"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
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

	// B3 M-P0-4: multi-user JWT — checkout ownership is org id.
	retailerID := ""
	if claims, ok := auth.FromContext(r.Context()); ok {
		retailerID = auth.ResolveRetailerOrgID(claims)
	}
	if retailerID == "" {
		writeJSONError(w, http.StatusUnprocessableEntity, "retailer_scope_missing", "retailer context is required", "/v1/order/card-checkout", false, "")
		return
	}

	amountMinor := req.AmountMinor
	if amountMinor <= 0 {
		amountMinor = req.Amount
	}
	currency := strings.TrimSpace(req.Currency)
	if amountMinor <= 0 && s.orderReader != nil {
		total, orderCurrency, snapErr := s.orderReader.CheckoutSnapshot(r.Context(), req.OrderID, retailerID)
		if snapErr != nil {
			s.writeOrderCheckoutError(w, "/v1/order/card-checkout", snapErr)
			return
		}
		amountMinor = total
		if currency == "" && strings.TrimSpace(orderCurrency) != "" {
			currency = orderCurrency
		}
	}
	if currency == "" {
		currency = s.currency
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
		if errors.Is(err, ErrCurrencyMismatch) {
			writeJSONError(w, http.StatusUnprocessableEntity, "currency_mismatch", "request currency must match order currency", "/v1/order/card-checkout", false, "")
			return
		}
		if writeCheckoutPackError(w, "/v1/order/card-checkout", err) {
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
// B1 M-P0-5: must mutate Spanner to PENDING_CASH_COLLECTION + outbox (via orderCash).
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

	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "unauthorized", "/v1/order/cash-checkout", false, "")
		return
	}
	// B3 M-P0-4: org-scoped cash selection; actor remains staff user id.
	retailerID := auth.ResolveRetailerOrgID(claims)
	if retailerID == "" {
		writeJSONError(w, http.StatusUnprocessableEntity, "retailer_scope_missing", "retailer context is required", "/v1/order/cash-checkout", false, "")
		return
	}
	actorID := auth.ResolveRetailerUserID(claims)
	if actorID == "" {
		actorID = claims.Subject
	}

	pack, packErr := auth.RequireCheckoutPack(claims)
	if packErr != nil {
		writeCheckoutPackError(w, "/v1/order/cash-checkout", packErr)
		return
	}
	if err := auth.AssertPackPSP(pack, DefaultPaymentMethod); err != nil {
		writeCheckoutPackError(w, "/v1/order/cash-checkout", err)
		return
	}

	if s.orderCash == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "cash_selector_unwired",
			"Cash selection requires order service wiring; use POST /v1/delivery/confirm-cash",
			"/v1/order/cash-checkout", false, "")
		return
	}

	status, amountMinor, _, err := s.orderCash.SelectCashAtDelivery(r.Context(), req.OrderID, retailerID, actorID)
	if err != nil {
		s.writeOrderCheckoutError(w, "/v1/order/cash-checkout", err)
		return
	}

	resp := retailerCashCheckoutResponse{
		OrderID:    req.OrderID,
		State:      status,
		Amount:     amountMinor,
		RetailerID: retailerID,
		Message:    "awaiting_driver_cash_collection",
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
	case errors.Is(err, order.ErrInvalidStatusTransition):
		writeJSONError(w, http.StatusConflict, "invalid_status_for_cash_selection", err.Error(), endpoint, false, "")
	default:
		writeJSONError(w, http.StatusInternalServerError, "order_lookup_failed", err.Error(), endpoint, false, "")
	}
}

func writeCheckoutPackError(w http.ResponseWriter, endpoint string, err error) bool {
	if errors.Is(err, auth.ErrMarketPackUnknown) || errors.Is(err, auth.ErrMarketPackNotShipped) ||
		errors.Is(err, auth.ErrPackGatewayForbidden) || errors.Is(err, auth.ErrPackCurrencyMismatch) {
		st, code := auth.CheckoutPackHTTPStatus(err)
		writeJSONError(w, st, code, err.Error(), endpoint, false, "")
		return true
	}
	return false
}

func (s *Service) initCheckoutSession(ctx context.Context, mode string, req CheckoutRequest) (SessionRecord, PaymentAttemptRecord, ExecutionResult, error) {
	pack, err := auth.CheckoutPackFromContext(ctx)
	if err != nil {
		return SessionRecord{}, PaymentAttemptRecord{}, ExecutionResult{}, err
	}
	resolvedCurrency, err := auth.ResolveCheckoutCurrency(pack, req.Currency)
	if err != nil {
		return SessionRecord{}, PaymentAttemptRecord{}, ExecutionResult{}, err
	}
	requestHadCurrency := strings.TrimSpace(req.Currency) != ""

	warehouseID := ""
	orderCurrency := ""
	if s.orderReader != nil {
		if orderCtx, err := s.orderReader.CheckoutOrderContext(ctx, req.OrderID, req.RetailerID); err == nil {
			orderCurrency = strings.ToUpper(strings.TrimSpace(orderCtx.Currency))
			warehouseID = orderCtx.WarehouseID
		}
	}
	if orderCurrency != "" {
		if requestHadCurrency && resolvedCurrency != orderCurrency {
			return SessionRecord{}, PaymentAttemptRecord{}, ExecutionResult{}, ErrCurrencyMismatch
		}
		if !strings.EqualFold(orderCurrency, pack.CurrencyCode) {
			return SessionRecord{}, PaymentAttemptRecord{}, ExecutionResult{}, auth.ErrPackCurrencyMismatch
		}
	}

	supplierID := s.resolveSupplierID(ctx)
	policy := NormalizeGatewayPolicy(PaymentAcceptorSupplier, nil, "SUPPLIER_DEFAULT")
	if s.policy != nil {
		resolved, err := s.policy.Resolve(ctx, supplierID, warehouseID)
		if err != nil {
			return SessionRecord{}, PaymentAttemptRecord{}, ExecutionResult{}, err
		}
		policy = resolved
	}
	policy = applyPackToGatewayPolicy(policy, pack)

	req.Gateway = auth.CanonicalPSP(req.Gateway)
	if mode == "CASH" {
		req.Gateway = DefaultPaymentMethod
	} else {
		req.Gateway = policy.ResolveCardGateway(req.Gateway)
	}
	if err := auth.AssertPackPSP(pack, req.Gateway); err != nil {
		return SessionRecord{}, PaymentAttemptRecord{}, ExecutionResult{}, err
	}
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
		SupplierID:  supplierID,
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
