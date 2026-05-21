// chargeback/reversal, and gateway webhooks.
package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Repository is the storage seam. Production binds this to Spanner-backed
// implementations with real RW transactions and OutboxEvents writes.
type Repository interface {
	CreateSession(ctx context.Context, s SessionRecord, emit func(outbox.TxnBuffer) error) error
	CreateSessionWithAttempt(ctx context.Context, s SessionRecord, a PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error
	SaveAttempt(ctx context.Context, a PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error
	SaveChargeback(ctx context.Context, c ChargebackRecord, emit func(outbox.TxnBuffer) error) error
	SaveReversal(ctx context.Context, rev ReversalRecord, emit func(outbox.TxnBuffer) error) error
	SaveWebhook(ctx context.Context, w WebhookRecord, emit func(outbox.TxnBuffer) error) error
}

// SessionRecord is the persisted checkout-session aggregate.
type SessionRecord struct {
	SessionID   string
	OrderID     string
	SupplierID  string
	RetailerID  string
	Gateway     string
	Currency    string
	AmountMinor int64
	Mode        string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PaymentAttemptRecord tracks execution metadata for a payment session attempt.
type PaymentAttemptRecord struct {
	AttemptID         string
	SessionID         string
	Gateway           string
	ExecutionAction   string
	ExecutionMode     string
	ProviderReference string
	Status            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ChargebackRecord tracks a chargeback mutation request.
type ChargebackRecord struct {
	ChargebackID string
	OrderID      string
	RetailerID   string
	Gateway      string
	AmountMinor  int64
	Currency     string
	CreatedAt    time.Time
}

// ReversalRecord tracks a chargeback-reversal mutation request.
type ReversalRecord struct {
	ReversalID string
	SessionID  string
	CreatedAt  time.Time
}

// WebhookRecord tracks one accepted gateway webhook.
type WebhookRecord struct {
	WebhookID      string
	Gateway        string
	TransactionID  string
	SessionID      string
	OrderID        string
	Status         string
	AmountMinor    int64
	Currency       string
	ReceivedAt     time.Time
	SignatureValid bool
}

// Service wires repository + cache + idempotency + secrets.
type Service struct {
	repo       Repository
	cache      *cache.Cache
	idem       idempotency.Store
	supplierID string
	currency   string
	execution  *ProviderExecutionRouter

	globalPayWebhookSecret string
	adyenWebhookSecret     string
	stripeWebhookSecret    string

	log   *slog.Logger
	now   func() time.Time
	newID func(prefix string) string
}

// ServiceConfig is constructor input.
type ServiceConfig struct {
	Repo                            Repository
	Cache                           *cache.Cache
	Idem                            idempotency.Store
	SupplierID                      string
	Currency                        string
	Execution                       *ProviderExecutionRouter
	AirwallexDirectExecutionEnabled bool

	GlobalPayWebhookSecret string
	AdyenWebhookSecret     string
	StripeWebhookSecret    string

	Log   *slog.Logger
	Now   func() time.Time
	NewID func(prefix string) string
}

// CheckoutRequest is the wire payload for checkout endpoints.
type CheckoutRequest struct {
	OrderID     string `json:"order_id"`
	RetailerID  string `json:"retailer_id,omitempty"`
	Gateway     string `json:"gateway,omitempty"`
	Currency    string `json:"currency,omitempty"`
	AmountMinor int64  `json:"amount_minor,omitempty"`
}

// CheckoutResponse is the wire response for checkout endpoints.
type CheckoutResponse struct {
	SessionID         string `json:"session_id"`
	OrderID           string `json:"order_id"`
	Status            string `json:"status"`
	ResolvedGateway   string `json:"resolved_gateway"`
	Currency          string `json:"currency"`
	PaymentURL        string `json:"payment_url"`
	PolicySource      string `json:"policy_source"`
	AttemptID         string `json:"attempt_id,omitempty"`
	ExecutionAction   string `json:"execution_action,omitempty"`
	ExecutionMode     string `json:"execution_mode,omitempty"`
	ProviderReference string `json:"provider_reference,omitempty"`
}

// ChargebackRequest is the wire payload for chargeback endpoint.
type ChargebackRequest struct {
	OrderID    string `json:"order_id"`
	RetailerID string `json:"retailer_id"`
	Gateway    string `json:"gateway"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency,omitempty"`
	AmountUZS  int64  `json:"amount_uzs"`
}

// ReversalRequest is the wire payload for chargeback reversal endpoint.
type ReversalRequest struct {
	SessionID string `json:"session_id"`
}

type globalPayWebhookRequest struct {
	SessionID     string `json:"session_id,omitempty"`
	ServiceToken  string `json:"service_token,omitempty"`
	PaymentID     string `json:"payment_id,omitempty"`
	TransactionID string `json:"transaction_id,omitempty"`
	OrderID       string `json:"order_id,omitempty"`
	Status        string `json:"status,omitempty"`
	AmountMinor   int64  `json:"amount_minor,omitempty"`
	Amount        int64  `json:"amount,omitempty"`
	Currency      string `json:"currency,omitempty"`
}

type stripeWebhookEnvelope struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data stripeEventData `json:"data"`
}

type stripeEventData struct {
	Object json.RawMessage `json:"object"`
}

type stripePaymentIntent struct {
	ID       string            `json:"id"`
	Status   string            `json:"status"`
	Amount   int64             `json:"amount"`
	Currency string            `json:"currency"`
	Metadata map[string]string `json:"metadata"`
}

type stripeCharge struct {
	ID             string            `json:"id"`
	Amount         int64             `json:"amount"`
	AmountRefunded int64             `json:"amount_refunded"`
	Currency       string            `json:"currency"`
	Metadata       map[string]string `json:"metadata"`
}

type adyenWebhookEnvelope struct {
	NotificationItems []adyenNotificationItemWrapper `json:"notificationItems"`
}

type adyenNotificationItemWrapper struct {
	Item adyenNotificationItem `json:"NotificationRequestItem"`
}

type adyenNotificationItem struct {
	PspReference        string            `json:"pspReference"`
	OriginalReference   string            `json:"originalReference,omitempty"`
	MerchantReference   string            `json:"merchantReference"`
	MerchantAccountCode string            `json:"merchantAccountCode"`
	EventCode           string            `json:"eventCode"`
	Success             string            `json:"success"`
	Amount              adyenAmount       `json:"amount"`
	AdditionalData      map[string]string `json:"additionalData"`
}

type adyenAmount struct {
	Currency string `json:"currency"`
	Value    int64  `json:"value"`
}

type paymentEvent struct {
	Type              string `json:"type"`
	SessionID         string `json:"session_id,omitempty"`
	AttemptID         string `json:"attempt_id,omitempty"`
	OrderID           string `json:"order_id,omitempty"`
	SupplierID        string `json:"supplier_id,omitempty"`
	RetailerID        string `json:"retailer_id,omitempty"`
	Gateway           string `json:"gateway,omitempty"`
	Status            string `json:"status"`
	ExecutionAction   string `json:"execution_action,omitempty"`
	ExecutionMode     string `json:"execution_mode,omitempty"`
	PolicySource      string `json:"policy_source,omitempty"`
	ProviderReference string `json:"provider_reference,omitempty"`
	AmountMinor       int64  `json:"amount_minor,omitempty"`
	Currency          string `json:"currency,omitempty"`
	TransactionID     string `json:"transaction_id,omitempty"`
	Source            string `json:"source,omitempty"`
	Timestamp         string `json:"timestamp"`
}

// NewService constructs payment service with defaults.
func NewService(c ServiceConfig) *Service {
	if c.Log == nil {
		c.Log = slog.Default()
	}
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	if c.NewID == nil {
		c.NewID = func(prefix string) string {
			return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
		}
	}
	if c.Currency == "" {
		c.Currency = "UZS"
	}
	if c.GlobalPayWebhookSecret == "" {
		c.GlobalPayWebhookSecret = "dev-global-pay-secret"
	}
	if c.AdyenWebhookSecret == "" {
		c.AdyenWebhookSecret = "dev-adyen-secret"
	}
	if c.StripeWebhookSecret == "" {
		c.StripeWebhookSecret = "dev-stripe-secret"
	}
	if c.Execution == nil {
		c.Execution = NewProviderExecutionRouter(ProviderExecutionRouterConfig{
			AirwallexDirectExecutionEnabled: c.AirwallexDirectExecutionEnabled,
		})
	}
	return &Service{
		repo:                   c.Repo,
		cache:                  c.Cache,
		idem:                   c.Idem,
		supplierID:             c.SupplierID,
		currency:               c.Currency,
		execution:              c.Execution,
		globalPayWebhookSecret: c.GlobalPayWebhookSecret,
		adyenWebhookSecret:     c.AdyenWebhookSecret,
		stripeWebhookSecret:    c.StripeWebhookSecret,
		log:                    c.Log,
		now:                    c.Now,
		newID:                  c.NewID,
	}
}

// HandleB2BCheckout serves POST /v1/checkout/b2b.
func (s *Service) HandleB2BCheckout(w http.ResponseWriter, r *http.Request) {
	s.handleCheckout("B2B", w, r)
}

// HandleUnifiedCheckout serves POST /v1/checkout/unified.
func (s *Service) HandleUnifiedCheckout(w http.ResponseWriter, r *http.Request) {
	s.handleCheckout("UNIFIED", w, r)
}

// HandleChargeback serves POST /v1/payment/chargeback.
func (s *Service) HandleChargeback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "/v1/payment/chargeback", false, "")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "Unable to read request body", "/v1/payment/chargeback", false, "")
		return
	}
	defer r.Body.Close()

	if rec, ok := s.handleIdempotentHit(w, r, body); ok {
		_ = rec
		return
	}

	var req ChargebackRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json_payload", "Invalid JSON payload", "/v1/payment/chargeback", false, "")
		return
	}
	amount := req.Amount
	if amount <= 0 {
		amount = req.AmountUZS
	}
	if strings.TrimSpace(req.OrderID) == "" || strings.TrimSpace(req.RetailerID) == "" || strings.TrimSpace(req.Gateway) == "" || amount <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "order_id, retailer_id, gateway, and amount (or amount_uzs) are required", "/v1/payment/chargeback", false, "")
		return
	}
	if strings.TrimSpace(req.Currency) == "" {
		req.Currency = s.currency
	}
	executionResult, err := s.execution.Execute(r.Context(), ExecutionRequest{
		Gateway:     req.Gateway,
		Action:      ExecutionActionChargebackRecord,
		OrderID:     req.OrderID,
		AmountMinor: amount,
		Currency:    req.Currency,
	})
	if err != nil {
		s.writeExecutionError(w, "/v1/payment/chargeback", err)
		return
	}
	now := s.now()
	rec := ChargebackRecord{
		ChargebackID: s.newID("chargeback"),
		OrderID:      strings.TrimSpace(req.OrderID),
		RetailerID:   strings.TrimSpace(req.RetailerID),
		Gateway:      executionResult.ResolvedGateway,
		AmountMinor:  amount,
		Currency:     strings.ToUpper(strings.TrimSpace(req.Currency)),
		CreatedAt:    now,
	}
	if err := s.repo.SaveChargeback(r.Context(), rec, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateSession, rec.ChargebackID, events.TopicMain, paymentEvent{
			Type:            events.EventPaymentRequired,
			OrderID:         rec.OrderID,
			RetailerID:      rec.RetailerID,
			Gateway:         rec.Gateway,
			Status:          "CHARGEBACK_RECORDED",
			ExecutionAction: string(ExecutionActionChargebackRecord),
			ExecutionMode:   string(executionResult.Mode),
			PolicySource:    executionResult.PolicySource,
			AmountMinor:     rec.AmountMinor,
			Currency:        rec.Currency,
			Source:          "payment.chargeback",
			Timestamp:       now.Format(time.RFC3339Nano),
		})
	}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "chargeback_record_failed", err.Error(), "/v1/payment/chargeback", false, "")
		return
	}
	s.cache.Invalidate(r.Context(), paymentOrderKey(rec.OrderID), paymentRetailerKey(rec.RetailerID))

	resp := map[string]string{"status": "chargeback_recorded"}
	respBytes, _ := json.Marshal(resp)
	s.persistIdempotencyRecord(r.Context(), r.Header.Get("Idempotency-Key"), sha256Hex(body), http.StatusOK, respBytes, 24*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

// HandleChargebackReversal serves POST /v1/payment/chargeback/reversal.
func (s *Service) HandleChargebackReversal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "/v1/payment/chargeback/reversal", false, "")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "Unable to read request body", "/v1/payment/chargeback/reversal", false, "")
		return
	}
	defer r.Body.Close()

	if rec, ok := s.handleIdempotentHit(w, r, body); ok {
		_ = rec
		return
	}

	var req ReversalRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json_payload", "Invalid JSON payload", "/v1/payment/chargeback/reversal", false, "")
		return
	}
	if strings.TrimSpace(req.SessionID) == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "session_id is required", "/v1/payment/chargeback/reversal", false, "")
		return
	}
	executionResult, err := s.execution.Execute(r.Context(), ExecutionRequest{
		Action:    ExecutionActionChargebackReversal,
		SessionID: req.SessionID,
	})
	if err != nil {
		s.writeExecutionError(w, "/v1/payment/chargeback/reversal", err)
		return
	}
	now := s.now()
	rev := ReversalRecord{
		ReversalID: s.newID("reversal"),
		SessionID:  strings.TrimSpace(req.SessionID),
		CreatedAt:  now,
	}
	if err := s.repo.SaveReversal(r.Context(), rev, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateSession, rev.SessionID, events.TopicMain, paymentEvent{
			Type:            events.EventPaymentCleared,
			SessionID:       rev.SessionID,
			Status:          "CHARGEBACK_REVERSAL_RECORDED",
			Gateway:         executionResult.ResolvedGateway,
			ExecutionAction: string(ExecutionActionChargebackReversal),
			ExecutionMode:   string(executionResult.Mode),
			PolicySource:    executionResult.PolicySource,
			Source:          "payment.chargeback_reversal",
			Timestamp:       now.Format(time.RFC3339Nano),
		})
	}); err != nil {
		writeJSONError(w, http.StatusBadRequest, "reversal_record_failed", err.Error(), "/v1/payment/chargeback/reversal", false, "")
		return
	}
	s.cache.Invalidate(r.Context(), paymentSessionKey(rev.SessionID))

	resp := map[string]string{"status": "reversal_recorded"}
	respBytes, _ := json.Marshal(resp)
	s.persistIdempotencyRecord(r.Context(), r.Header.Get("Idempotency-Key"), sha256Hex(body), http.StatusOK, respBytes, 24*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

// HandleDeprecatedGlobalPayInitiate serves POST /v1/payment/global_pay/initiate.
func (s *Service) HandleDeprecatedGlobalPayInitiate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "/v1/payment/global_pay/initiate", true, "/v1/order/card-checkout")
		return
	}
	writeJSONError(w, http.StatusGone, "endpoint_deprecated", "Deprecated endpoint", "/v1/payment/global_pay/initiate", true, "/v1/order/card-checkout")
}

// HandleGlobalPayWebhook serves POST /v1/webhooks/global-pay.
func (s *Service) HandleGlobalPayWebhook(w http.ResponseWriter, r *http.Request) {
	const endpoint = "/v1/webhooks/global-pay"
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", endpoint, false, "")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 256*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "Unable to read request body", endpoint, false, "")
		return
	}
	defer r.Body.Close()

	if !verifyGlobalPayBasicAuth(r.Header.Get("Authorization"), s.globalPayWebhookSecret) {
		writeJSONError(w, http.StatusUnauthorized, "invalid_signature", "Invalid webhook signature", endpoint, false, "")
		return
	}

	req, err := parseGlobalPayWebhookRequest(body, r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", err.Error(), endpoint, false, "")
		return
	}

	req.Status = strings.ToUpper(strings.TrimSpace(req.Status))
	if req.TransactionID == "" || req.Status == "" {
		writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "transaction_id and status are required", endpoint, false, "")
		return
	}

	bodyHash := sha256Hex(body)
	webhookKey := "webhook:global_pay:" + req.TransactionID + ":" + req.Status
	if replayed := s.writeWebhookReplayIfExists(w, r, endpoint, webhookKey, bodyHash); replayed {
		return
	}

	amount := req.AmountMinor
	if amount <= 0 {
		amount = req.Amount
	}
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = s.currency
	}
	now := s.now()
	row := WebhookRecord{
		WebhookID:      s.newID("webhook"),
		Gateway:        "GLOBAL_PAY",
		TransactionID:  req.TransactionID,
		SessionID:      req.SessionID,
		OrderID:        strings.TrimSpace(req.OrderID),
		Status:         req.Status,
		AmountMinor:    amount,
		Currency:       currency,
		ReceivedAt:     now,
		SignatureValid: true,
	}
	if err := s.persistWebhookWithOutbox(r.Context(), row, "payment.webhook.global_pay", now); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "webhook_process_failed", err.Error(), endpoint, false, "")
		return
	}

	resp := map[string]string{
		"status":         "accepted",
		"gateway":        "global-pay",
		"transaction_id": row.TransactionID,
	}
	respBytes, _ := json.Marshal(resp)
	s.persistIdempotencyRecord(r.Context(), webhookKey, bodyHash, http.StatusOK, respBytes, 7*24*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

// HandleAdyenWebhook serves POST /v1/webhooks/adyen.
func (s *Service) HandleAdyenWebhook(w http.ResponseWriter, r *http.Request) {
	const endpoint = "/v1/webhooks/adyen"
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", endpoint, false, "")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 512*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "Unable to read request body", endpoint, false, "")
		return
	}
	defer r.Body.Close()

	var envelope adyenWebhookEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json_payload", "Invalid JSON payload", endpoint, false, "")
		return
	}
	if len(envelope.NotificationItems) == 0 {
		writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "notificationItems is required", endpoint, false, "")
		return
	}

	for i := range envelope.NotificationItems {
		item := envelope.NotificationItems[i].Item
		if err := validateAdyenNotificationItem(item); err != nil {
			writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", err.Error(), endpoint, false, "")
			return
		}
		if !verifyAdyenNotificationSignature(item, s.adyenWebhookSecret) {
			writeJSONError(w, http.StatusUnauthorized, "invalid_signature", "Invalid webhook signature", endpoint, false, "")
			return
		}
	}

	processed := 0
	now := s.now()
	for i := range envelope.NotificationItems {
		item := envelope.NotificationItems[i].Item
		status := normalizeAdyenStatus(item)
		transactionID := strings.TrimSpace(item.PspReference)
		webhookKey := "webhook:adyen:" + transactionID + ":" + strings.ToUpper(strings.TrimSpace(item.EventCode))
		bodyHash := sha256Hex([]byte(adyenSigningData(item)))
		if replayed := s.writeWebhookReplayIfExists(w, r, endpoint, webhookKey, bodyHash); replayed {
			continue
		}

		currency := strings.ToUpper(strings.TrimSpace(item.Amount.Currency))
		if currency == "" {
			currency = s.currency
		}
		row := WebhookRecord{
			WebhookID:      s.newID("webhook"),
			Gateway:        "ADYEN",
			TransactionID:  transactionID,
			SessionID:      strings.TrimSpace(item.AdditionalData["session_id"]),
			OrderID:        strings.TrimSpace(item.MerchantReference),
			Status:         status,
			AmountMinor:    item.Amount.Value,
			Currency:       currency,
			ReceivedAt:     now,
			SignatureValid: true,
		}
		if err := s.persistWebhookWithOutbox(r.Context(), row, "payment.webhook.adyen", now); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "webhook_process_failed", err.Error(), endpoint, false, "")
			return
		}

		resp := map[string]string{
			"status":         "accepted",
			"gateway":        "adyen",
			"transaction_id": row.TransactionID,
		}
		respBytes, _ := json.Marshal(resp)
		s.persistIdempotencyRecord(r.Context(), webhookKey, bodyHash, http.StatusOK, respBytes, 7*24*time.Hour)
		processed++
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":          "accepted",
		"gateway":         "adyen",
		"processed_items": processed,
	})
}

// HandleStripeWebhook serves POST /v1/webhooks/stripe.
func (s *Service) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	const endpoint = "/v1/webhooks/stripe"
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", endpoint, false, "")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 512*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "Unable to read request body", endpoint, false, "")
		return
	}
	defer r.Body.Close()

	if !verifyStripeSignatureHeader(body, r.Header.Get("Stripe-Signature"), s.stripeWebhookSecret, s.now()) {
		writeJSONError(w, http.StatusUnauthorized, "invalid_signature", "Invalid webhook signature", endpoint, false, "")
		return
	}

	var envelope stripeWebhookEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json_payload", "Invalid JSON payload", endpoint, false, "")
		return
	}
	envelope.ID = strings.TrimSpace(envelope.ID)
	envelope.Type = strings.TrimSpace(envelope.Type)
	if envelope.ID == "" || envelope.Type == "" || len(envelope.Data.Object) == 0 {
		writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "id, type, and data.object are required", endpoint, false, "")
		return
	}

	var row WebhookRecord
	now := s.now()
	row.WebhookID = s.newID("webhook")
	row.Gateway = "STRIPE"
	row.ReceivedAt = now
	row.SignatureValid = true

	switch envelope.Type {
	case "payment_intent.succeeded", "payment_intent.payment_failed":
		var intent stripePaymentIntent
		if err := json.Unmarshal(envelope.Data.Object, &intent); err != nil {
			writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "invalid payment_intent payload", endpoint, false, "")
			return
		}
		intent.ID = strings.TrimSpace(intent.ID)
		if intent.ID == "" {
			writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "payment_intent.id is required", endpoint, false, "")
			return
		}
		row.TransactionID = intent.ID
		if intent.Metadata != nil {
			row.SessionID = strings.TrimSpace(intent.Metadata["session_id"])
			row.OrderID = strings.TrimSpace(intent.Metadata["order_id"])
		}
		if row.OrderID == "" {
			row.OrderID = strings.TrimSpace(intent.Metadata["merchant_reference"])
		}
		if envelope.Type == "payment_intent.succeeded" {
			row.Status = "PAID"
		} else {
			row.Status = "FAILED"
		}
		row.AmountMinor = intent.Amount
		row.Currency = strings.ToUpper(strings.TrimSpace(intent.Currency))
	case "charge.refunded":
		var charge stripeCharge
		if err := json.Unmarshal(envelope.Data.Object, &charge); err != nil {
			writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "invalid charge payload", endpoint, false, "")
			return
		}
		charge.ID = strings.TrimSpace(charge.ID)
		if charge.ID == "" {
			writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "charge.id is required", endpoint, false, "")
			return
		}
		row.TransactionID = charge.ID
		if charge.Metadata != nil {
			row.SessionID = strings.TrimSpace(charge.Metadata["session_id"])
			row.OrderID = strings.TrimSpace(charge.Metadata["order_id"])
		}
		row.Status = "REFUNDED"
		row.AmountMinor = charge.AmountRefunded
		if row.AmountMinor <= 0 {
			row.AmountMinor = charge.Amount
		}
		row.Currency = strings.ToUpper(strings.TrimSpace(charge.Currency))
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "ignored",
			"gateway":    "stripe",
			"event_type": envelope.Type,
		})
		return
	}

	if row.Currency == "" {
		row.Currency = s.currency
	}

	bodyHash := sha256Hex(body)
	webhookKey := "webhook:stripe:" + envelope.ID
	if replayed := s.writeWebhookReplayIfExists(w, r, endpoint, webhookKey, bodyHash); replayed {
		return
	}

	if err := s.persistWebhookWithOutbox(r.Context(), row, "payment.webhook.stripe", now); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "webhook_process_failed", err.Error(), endpoint, false, "")
		return
	}

	resp := map[string]string{
		"status":         "accepted",
		"gateway":        "stripe",
		"transaction_id": row.TransactionID,
	}
	respBytes, _ := json.Marshal(resp)
	s.persistIdempotencyRecord(r.Context(), webhookKey, bodyHash, http.StatusOK, respBytes, 7*24*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

func (s *Service) handleCheckout(mode string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "/v1/checkout", false, "")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "Unable to read request body", "/v1/checkout", false, "")
		return
	}
	defer r.Body.Close()

	if rec, ok := s.handleIdempotentHit(w, r, body); ok {
		_ = rec
		return
	}

	var req CheckoutRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json_payload", "Invalid JSON payload", "/v1/checkout", false, "")
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	if req.OrderID == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "order_id is required", "/v1/checkout", false, "")
		return
	}
	retailerID := strings.TrimSpace(req.RetailerID)
	if claims, ok := auth.FromContext(r.Context()); ok && claims.Subject != "" {
		retailerID = claims.Subject
	}
	if retailerID == "" {
		writeJSONError(w, http.StatusUnprocessableEntity, "retailer_scope_missing", "retailer context is required", "/v1/checkout", false, "")
		return
	}

	resolvedCurrency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if resolvedCurrency == "" {
		resolvedCurrency = s.currency
	}
	executionResult, err := s.execution.Execute(r.Context(), ExecutionRequest{
		Gateway:     req.Gateway,
		Action:      ExecutionActionCheckoutInit,
		OrderID:     req.OrderID,
		AmountMinor: req.AmountMinor,
		Currency:    resolvedCurrency,
	})
	if err != nil {
		s.writeExecutionError(w, "/v1/checkout", err)
		return
	}
	now := s.now()
	session := SessionRecord{
		SessionID:   s.newID("psess"),
		OrderID:     req.OrderID,
		SupplierID:  s.supplierID,
		RetailerID:  retailerID,
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
	if err := s.repo.CreateSessionWithAttempt(r.Context(), session, attempt, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateSession, session.SessionID, events.TopicMain, paymentEvent{
			Type:              events.EventPaymentRequired,
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
			Timestamp:         now.Format(time.RFC3339Nano),
		})
	}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "checkout_session_attempt_create_failed", err.Error(), "/v1/checkout", false, "")
		return
	}

	s.cache.Invalidate(r.Context(), paymentOrderKey(session.OrderID), paymentRetailerKey(session.RetailerID), paymentSessionKey(session.SessionID))

	resp := CheckoutResponse{
		SessionID:         session.SessionID,
		OrderID:           session.OrderID,
		Status:            session.Status,
		ResolvedGateway:   session.Gateway,
		Currency:          session.Currency,
		PaymentURL:        executionResult.RedirectURL,
		PolicySource:      executionResult.PolicySource,
		AttemptID:         attempt.AttemptID,
		ExecutionAction:   attempt.ExecutionAction,
		ExecutionMode:     attempt.ExecutionMode,
		ProviderReference: attempt.ProviderReference,
	}
	if strings.TrimSpace(resp.PaymentURL) == "" {
		resp.PaymentURL = "/v1/payment/session/" + session.SessionID
	}
	respBytes, _ := json.Marshal(resp)
	s.persistIdempotencyRecord(r.Context(), r.Header.Get("Idempotency-Key"), sha256Hex(body), http.StatusOK, respBytes, 24*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

func (s *Service) persistWebhookWithOutbox(ctx context.Context, row WebhookRecord, source string, now time.Time) error {
	eventType := events.EventPaymentRequired
	if isPaymentSuccessStatus(row.Status) {
		eventType = events.EventPaymentCleared
	}
	if err := s.repo.SaveWebhook(ctx, row, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateSession, row.WebhookID, events.TopicMain, paymentEvent{
			Type:          eventType,
			SessionID:     row.SessionID,
			OrderID:       row.OrderID,
			SupplierID:    s.supplierID,
			Gateway:       row.Gateway,
			Status:        row.Status,
			AmountMinor:   row.AmountMinor,
			Currency:      row.Currency,
			TransactionID: row.TransactionID,
			Source:        source,
			Timestamp:     now.Format(time.RFC3339Nano),
		})
	}); err != nil {
		return err
	}

	keys := make([]string, 0, 2)
	if row.OrderID != "" {
		keys = append(keys, paymentOrderKey(row.OrderID))
	}
	if row.SessionID != "" {
		keys = append(keys, paymentSessionKey(row.SessionID))
	}
	if len(keys) > 0 {
		s.cache.Invalidate(ctx, keys...)
	}
	return nil
}

func (s *Service) writeExecutionError(w http.ResponseWriter, endpoint string, err error) {
	var policyErr *GatewayPolicyError
	if errors.As(err, &policyErr) {
		code := strings.TrimSpace(policyErr.Code)
		if code == "" {
			code = "payment_gateway_policy_violation"
		}
		writeJSONError(w, http.StatusUnprocessableEntity, code, policyErr.Error(), endpoint, false, "")
		return
	}
	writeJSONError(w, http.StatusBadGateway, "payment_gateway_execution_failed", err.Error(), endpoint, false, "")
}

func (s *Service) writeWebhookReplayIfExists(w http.ResponseWriter, r *http.Request, endpoint, webhookKey, bodyHash string) bool {
	replay, hit, err := idempotency.Guard(r.Context(), s.idem, webhookKey, bodyHash)
	if errors.Is(err, idempotency.ErrConflict) {
		writeJSONError(w, http.StatusConflict, "idempotency_key_payload_mismatch", "idempotency key payload mismatch", endpoint, false, "")
		return true
	}
	if err != nil {
		s.log.Warn("webhook idempotency guard failed", "key", webhookKey, "err", err)
		return false
	}
	if !hit {
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(replay.StatusCode)
	_, _ = w.Write(replay.Response)
	return true
}

func (s *Service) handleIdempotentHit(w http.ResponseWriter, r *http.Request, body []byte) (idempotency.Record, bool) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" || s.idem == nil {
		return idempotency.Record{}, false
	}
	rec, hit, err := idempotency.Guard(r.Context(), s.idem, key, sha256Hex(body))
	if errors.Is(err, idempotency.ErrConflict) {
		writeJSONError(w, http.StatusConflict, "idempotency_key_payload_mismatch", "idempotency key payload mismatch", r.URL.Path, false, "")
		return idempotency.Record{}, true
	}
	if err != nil {
		s.log.Warn("idempotency guard failed", "path", r.URL.Path, "err", err)
		return idempotency.Record{}, false
	}
	if !hit {
		return idempotency.Record{}, false
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(rec.StatusCode)
	_, _ = w.Write(rec.Response)
	return rec, true
}

func (s *Service) persistIdempotencyRecord(ctx context.Context, key, bodyHash string, statusCode int, payload []byte, ttl time.Duration) {
	if strings.TrimSpace(key) == "" || s.idem == nil {
		return
	}
	if err := s.idem.Save(ctx, key, idempotency.Record{
		BodyHash:   bodyHash,
		StatusCode: statusCode,
		Response:   payload,
		StoredAt:   s.now(),
	}, ttl); err != nil {
		s.log.Warn("idempotency save failed", "key", key, "err", err)
	}
}

func parseGlobalPayWebhookRequest(body []byte, r *http.Request) (globalPayWebhookRequest, error) {
	var req globalPayWebhookRequest
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			return globalPayWebhookRequest{}, fmt.Errorf("invalid JSON payload")
		}
	}

	q := r.URL.Query()
	req.SessionID = coalesceString(strings.TrimSpace(req.SessionID), strings.TrimSpace(q.Get("session_id")))
	req.ServiceToken = coalesceString(strings.TrimSpace(req.ServiceToken), strings.TrimSpace(q.Get("service_token")), strings.TrimSpace(q.Get("serviceToken")), strings.TrimSpace(q.Get("provider_reference")))
	req.PaymentID = coalesceString(strings.TrimSpace(req.PaymentID), strings.TrimSpace(q.Get("payment_id")), strings.TrimSpace(q.Get("paymentId")))
	req.TransactionID = coalesceString(strings.TrimSpace(req.TransactionID), req.PaymentID, req.ServiceToken, strings.TrimSpace(q.Get("transaction_id")))
	req.OrderID = coalesceString(strings.TrimSpace(req.OrderID), strings.TrimSpace(q.Get("order_id")), strings.TrimSpace(q.Get("merchant_reference")))
	req.Status = coalesceString(strings.TrimSpace(req.Status), strings.TrimSpace(q.Get("status")), strings.TrimSpace(q.Get("state")))

	if req.AmountMinor <= 0 {
		if raw := strings.TrimSpace(q.Get("amount_minor")); raw != "" {
			if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
				req.AmountMinor = parsed
			}
		}
	}
	if req.Amount <= 0 {
		if raw := strings.TrimSpace(q.Get("amount")); raw != "" {
			if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
				req.Amount = parsed
			}
		}
	}
	req.Currency = coalesceString(strings.TrimSpace(req.Currency), strings.TrimSpace(q.Get("currency")))

	if req.SessionID == "" {
		return globalPayWebhookRequest{}, fmt.Errorf("session_id is required")
	}
	if req.TransactionID == "" {
		return globalPayWebhookRequest{}, fmt.Errorf("transaction_id (or payment_id/service_token) is required")
	}
	if strings.TrimSpace(req.Status) == "" {
		return globalPayWebhookRequest{}, fmt.Errorf("status is required")
	}
	return req, nil
}

func verifyGlobalPayBasicAuth(rawHeader, secret string) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}
	header := strings.TrimSpace(rawHeader)
	if len(header) < len("Basic ") || !strings.EqualFold(header[:len("Basic ")], "Basic ") {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(header[len("Basic "):]))
	if err != nil {
		return false
	}
	expected := []byte("Paycom:" + secret)
	if len(decoded) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare(decoded, expected) == 1
}

func verifyStripeSignatureHeader(body []byte, rawHeader, secret string, now time.Time) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}
	parts := strings.Split(strings.TrimSpace(rawHeader), ",")
	if len(parts) == 0 {
		return false
	}

	var timestamp string
	var v1 string
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		switch key {
		case "t":
			timestamp = value
		case "v1":
			v1 = value
		}
	}
	if timestamp == "" || v1 == "" {
		return false
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	if now.Sub(time.Unix(ts, 0)).Abs() > 5*time.Minute {
		return false
	}

	signedPayload := timestamp + "." + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(expected)), []byte(strings.ToLower(v1)))
}

func validateAdyenNotificationItem(item adyenNotificationItem) error {
	if strings.TrimSpace(item.PspReference) == "" {
		return fmt.Errorf("pspReference is required")
	}
	if strings.TrimSpace(item.EventCode) == "" {
		return fmt.Errorf("eventCode is required")
	}
	if strings.TrimSpace(item.MerchantReference) == "" {
		return fmt.Errorf("merchantReference is required")
	}
	if strings.TrimSpace(item.MerchantAccountCode) == "" {
		return fmt.Errorf("merchantAccountCode is required")
	}
	if strings.TrimSpace(item.Success) == "" {
		return fmt.Errorf("success is required")
	}
	if strings.TrimSpace(item.Amount.Currency) == "" {
		return fmt.Errorf("amount.currency is required")
	}
	return nil
}

func verifyAdyenNotificationSignature(item adyenNotificationItem, secret string) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}
	if item.AdditionalData == nil {
		return false
	}
	provided := strings.TrimSpace(item.AdditionalData["hmacSignature"])
	if provided == "" {
		return false
	}
	providedBytes, err := base64.StdEncoding.DecodeString(provided)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(adyenSigningData(item)))
	expected := mac.Sum(nil)
	return hmac.Equal(expected, providedBytes)
}

func adyenSigningData(item adyenNotificationItem) string {
	parts := []string{
		escapeAdyenSignatureValue(item.PspReference),
		escapeAdyenSignatureValue(item.OriginalReference),
		escapeAdyenSignatureValue(item.MerchantAccountCode),
		escapeAdyenSignatureValue(item.MerchantReference),
		strconv.FormatInt(item.Amount.Value, 10),
		escapeAdyenSignatureValue(strings.ToUpper(strings.TrimSpace(item.Amount.Currency))),
		escapeAdyenSignatureValue(strings.ToUpper(strings.TrimSpace(item.EventCode))),
		escapeAdyenSignatureValue(strings.ToLower(strings.TrimSpace(item.Success))),
	}
	return strings.Join(parts, ":")
}

func escapeAdyenSignatureValue(value string) string {
	value = strings.ReplaceAll(value, `\\`, `\\\\`)
	value = strings.ReplaceAll(value, `:`, `\\:`)
	return value
}

func normalizeAdyenStatus(item adyenNotificationItem) string {
	eventCode := strings.ToUpper(strings.TrimSpace(item.EventCode))
	success := strings.EqualFold(strings.TrimSpace(item.Success), "true")
	switch eventCode {
	case "REFUND", "REFUNDED_REVERSED":
		if success {
			return "REFUNDED"
		}
		return "FAILED"
	case "CANCELLATION", "CANCEL_OR_REFUND", "VOID_PENDING_REFUND", "CANCELLED":
		if success {
			return "CANCELLED"
		}
		return "FAILED"
	default:
		if success {
			return "PAID"
		}
		return "FAILED"
	}
}

func coalesceString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func isPaymentSuccessStatus(status string) bool {
	s := strings.ToUpper(strings.TrimSpace(status))
	return s == "PAID" || s == "CAPTURED" || s == "SETTLED" || s == "SUCCESS"
}

func writeJSONError(w http.ResponseWriter, status int, code, message, endpoint string, deprecated bool, migrateTo string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	payload := map[string]any{
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

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func paymentOrderKey(orderID string) string {
	return "payment:order:" + orderID
}

func paymentSessionKey(sessionID string) string {
	return "payment:session:" + sessionID
}

func paymentRetailerKey(retailerID string) string {
	return "payment:retailer:" + retailerID
}
