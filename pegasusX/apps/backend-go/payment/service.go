// chargeback/reversal, and gateway webhooks.
package payment

import (
	"bytes"
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
	"sort"
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
	ListLedgerEntries(ctx context.Context, q LedgerQuery) ([]LedgerEntryRecord, error)
	SummarizeLedgerEntries(ctx context.Context, q SettlementAuthorityQuery) ([]SettlementAuthorityRow, error)
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
	SupplierID   string
	RetailerID   string
	Gateway      string
	AmountMinor  int64
	Currency     string
	CreatedAt    time.Time
}

// ReversalRecord tracks a chargeback-reversal mutation request.
type ReversalRecord struct {
	ReversalID  string
	SessionID   string
	SupplierID  string
	Gateway     string
	AmountMinor int64
	Currency    string
	CreatedAt   time.Time
}

// WebhookRecord tracks one accepted gateway webhook.
type WebhookRecord struct {
	WebhookID      string
	Gateway        string
	TransactionID  string
	SessionID      string
	OrderID        string
	SupplierID     string
	RetailerID     string
	Status         string
	AmountMinor    int64
	Currency       string
	ReceivedAt     time.Time
	SignatureValid bool
}

// LedgerQuery defines a bounded payment-ledger read query.
type LedgerQuery struct {
	SupplierID   string
	OrderID      string
	SessionID    string
	Gateway      string
	EntryType    string
	OccurredFrom *time.Time
	OccurredTo   *time.Time
	Limit        int
}

// SettlementAuthorityQuery defines filter criteria for immutable payment
// settlement authority summaries derived from PaymentLedgerEntries.
type SettlementAuthorityQuery struct {
	SupplierID   string
	Gateway      string
	EntryType    string
	OccurredFrom *time.Time
	OccurredTo   *time.Time
	GroupLimit   int
}

// ReconciliationMismatchQuery defines filter criteria for grouped mismatch
// detection over immutable payment ledger summaries.
type ReconciliationMismatchQuery struct {
	SupplierID             string
	Gateway                string
	OccurredFrom           *time.Time
	OccurredTo             *time.Time
	GroupLimit             int
	MismatchThresholdMinor int64
}

// LedgerEntryRecord represents one immutable payment-ledger row.
type LedgerEntryRecord struct {
	LedgerEntryID string    `json:"ledger_entry_id"`
	SessionID     string    `json:"session_id,omitempty"`
	OrderID       string    `json:"order_id,omitempty"`
	SupplierID    string    `json:"supplier_id,omitempty"`
	RetailerID    string    `json:"retailer_id,omitempty"`
	Gateway       string    `json:"gateway"`
	EntryType     string    `json:"entry_type"`
	AmountMinor   int64     `json:"amount_minor"`
	Currency      string    `json:"currency"`
	ReferenceID   string    `json:"reference_id,omitempty"`
	Source        string    `json:"source"`
	OccurredAt    time.Time `json:"occurred_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// SettlementAuthorityRow is one grouped settlement authority summary row.
type SettlementAuthorityRow struct {
	Gateway          string    `json:"gateway"`
	EntryType        string    `json:"entry_type"`
	Currency         string    `json:"currency"`
	EntryCount       int64     `json:"entry_count"`
	AmountMinorTotal int64     `json:"amount_minor_total"`
	FirstOccurredAt  time.Time `json:"first_occurred_at"`
	LastOccurredAt   time.Time `json:"last_occurred_at"`
}

// SettlementCurrencyTotal reports rolled-up totals per currency.
type SettlementCurrencyTotal struct {
	Currency         string `json:"currency"`
	EntryCount       int64  `json:"entry_count"`
	AmountMinorTotal int64  `json:"amount_minor_total"`
}

// ReconciliationEntryTypeTotal exposes one grouped entry-type contribution
// used to classify mismatch net values.
type ReconciliationEntryTypeTotal struct {
	EntryType              string `json:"entry_type"`
	EntryCount             int64  `json:"entry_count"`
	AmountMinorTotal       int64  `json:"amount_minor_total"`
	SignedAmountMinorTotal int64  `json:"signed_amount_minor_total"`
}

// ReconciliationMismatchRow reports one gateway-currency mismatch aggregate.
type ReconciliationMismatchRow struct {
	Gateway         string                         `json:"gateway"`
	Currency        string                         `json:"currency"`
	EntryCountTotal int64                          `json:"entry_count_total"`
	GroupCount      int64                          `json:"group_count"`
	CreditAmount    int64                          `json:"credit_amount_minor_total"`
	DebitAmount     int64                          `json:"debit_amount_minor_total"`
	NetAmount       int64                          `json:"net_amount_minor"`
	FirstOccurredAt time.Time                      `json:"first_occurred_at"`
	LastOccurredAt  time.Time                      `json:"last_occurred_at"`
	EntryTypeTotals []ReconciliationEntryTypeTotal `json:"entry_type_totals"`
}

// Service wires repository + cache + idempotency + secrets.
type Service struct {
	repo       Repository
	cache      *cache.Cache
	idem       idempotency.Store
	supplierID string
	currency   string
	execution  *ProviderExecutionRouter

	cartCheckout CartCheckoutHandler
	orderReader  OrderCheckoutReader

	globalPayEnv           string
	globalPayUsername      string
	globalPayPassword      string
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

	GlobalPayEnv           string
	GlobalPayServiceID     string
	GlobalPayUsername      string
	GlobalPayPassword      string
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

type adyenRawWebhookEnvelope struct {
	NotificationItems []adyenRawNotificationItemWrapper `json:"notificationItems"`
}

type adyenRawNotificationItemWrapper struct {
	Item json.RawMessage `json:"NotificationRequestItem"`
}

type adyenNotificationItemWrapper struct {
	Item adyenNotificationItem `json:"NotificationRequestItem"`
}

type adyenSignedNotificationItem adyenNotificationItem

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
			GlobalPayEnv:                    c.GlobalPayEnv,
			GlobalPayServiceID:              c.GlobalPayServiceID,
			GlobalPayUsername:               c.GlobalPayUsername,
			GlobalPayPassword:               c.GlobalPayPassword,
		})
	}
	return &Service{
		repo:                   c.Repo,
		cache:                  c.Cache,
		idem:                   c.Idem,
		supplierID:             c.SupplierID,
		currency:               c.Currency,
		execution:              c.Execution,
		globalPayEnv:           c.GlobalPayEnv,
		globalPayUsername:      c.GlobalPayUsername,
		globalPayPassword:      c.GlobalPayPassword,
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
// Cart payloads (items[]) route to the bound order handler; order_id payloads
// keep the legacy payment-session checkout path.
func (s *Service) HandleUnifiedCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "/v1/checkout/unified", false, "")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "Unable to read request body", "/v1/checkout/unified", false, "")
		return
	}
	defer r.Body.Close()

	if isCartUnifiedCheckoutBody(body) {
		if s.cartCheckout == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "cart_checkout_unavailable", "cart checkout handler not configured", "/v1/checkout/unified", false, "")
			return
		}
		s.cartCheckout.HandleUnifiedCheckout(w, requestWithBody(r, body))
		return
	}
	s.handleCheckoutWithBody("UNIFIED", w, requestWithBody(r, body), body)
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
		SupplierID:   s.supplierID,
		RetailerID:   strings.TrimSpace(req.RetailerID),
		Gateway:      executionResult.ResolvedGateway,
		AmountMinor:  amount,
		Currency:     strings.ToUpper(strings.TrimSpace(req.Currency)),
		CreatedAt:    now,
	}
	if err := s.repo.SaveChargeback(r.Context(), rec, func(txn outbox.TxnBuffer) error {
		if err := outbox.EmitJSON(r.Context(), txn, events.AggregateSession, rec.ChargebackID, events.TopicMain, events.FinanceEvent{
			BaseEvent:       events.BaseEvent{Type: events.EventPaymentRequired, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:         rec.OrderID,
			SupplierID:      rec.SupplierID,
			RetailerID:      rec.RetailerID,
			Gateway:         rec.Gateway,
			Status:          "CHARGEBACK_RECORDED",
			ExecutionAction: string(ExecutionActionChargebackRecord),
			ExecutionMode:   string(executionResult.Mode),
			PolicySource:    executionResult.PolicySource,
			AmountMinor:     rec.AmountMinor,
			Currency:        rec.Currency,
			Source:          "payment.chargeback",
		}); err != nil {
			return err
		}
		return outbox.EmitJSON(r.Context(), txn, events.AggregateOrder, rec.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventDeliveryDisputed, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:    rec.OrderID,
			SupplierID: rec.SupplierID,
			RetailerID: rec.RetailerID,
			Reason:     "chargeback_recorded",
			Action:     "payment.chargeback",
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
		ReversalID:  s.newID("reversal"),
		SessionID:   strings.TrimSpace(req.SessionID),
		SupplierID:  s.supplierID,
		Gateway:     executionResult.ResolvedGateway,
		AmountMinor: 0,
		Currency:    s.currency,
		CreatedAt:   now,
	}
	if err := s.repo.SaveReversal(r.Context(), rev, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateSession, rev.SessionID, events.TopicMain, events.FinanceEvent{
			BaseEvent:       events.BaseEvent{Type: events.EventPaymentCleared, Timestamp: now.Format(time.RFC3339Nano)},
			SessionID:       rev.SessionID,
			SupplierID:      rev.SupplierID,
			Status:          "CHARGEBACK_REVERSAL_RECORDED",
			Gateway:         executionResult.ResolvedGateway,
			ExecutionAction: string(ExecutionActionChargebackReversal),
			ExecutionMode:   string(executionResult.Mode),
			PolicySource:    executionResult.PolicySource,
			Source:          "payment.chargeback_reversal",
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

// CaptureCardPayment synchronously executes the capture action for a completed card-based order.
func (s *Service) CaptureCardPayment(ctx context.Context, orderID string, amountMinor int64, currency string) error {
	_, err := s.execution.Execute(ctx, ExecutionRequest{
		Gateway:     "GLOBALPAY",
		Action:      ExecutionActionCheckoutCapture,
		OrderID:     orderID,
		AmountMinor: amountMinor,
		Currency:    currency,
	})
	return err
}

// HandleLedger serves GET /v1/payment/ledger.
func (s *Service) HandleLedger(w http.ResponseWriter, r *http.Request) {
	const endpoint = "/v1/payment/ledger"
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", endpoint, false, "")
		return
	}
	if s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "ledger_repository_unavailable", "Ledger repository is unavailable", endpoint, false, "")
		return
	}

	supplierID, ok := resolvePaymentSupplierScope(w, r, s.supplierID, endpoint)
	if !ok {
		return
	}

	limit, err := parseBoundedIntQuery(strings.TrimSpace(r.URL.Query().Get("limit")), 100, 1, 500)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 500", endpoint, false, "")
		return
	}

	occurredFrom, err := parseRFC3339QueryTime(strings.TrimSpace(r.URL.Query().Get("occurred_from")))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_from", "occurred_from must be RFC3339", endpoint, false, "")
		return
	}
	occurredTo, err := parseRFC3339QueryTime(strings.TrimSpace(r.URL.Query().Get("occurred_to")))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_to", "occurred_to must be RFC3339", endpoint, false, "")
		return
	}
	if occurredFrom != nil && occurredTo != nil && occurredFrom.After(*occurredTo) {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_range", "occurred_from must be before or equal to occurred_to", endpoint, false, "")
		return
	}

	items, err := s.repo.ListLedgerEntries(r.Context(), LedgerQuery{
		SupplierID:   supplierID,
		OrderID:      strings.TrimSpace(r.URL.Query().Get("order_id")),
		SessionID:    strings.TrimSpace(r.URL.Query().Get("session_id")),
		Gateway:      strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("gateway"))),
		EntryType:    strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("entry_type"))),
		OccurredFrom: occurredFrom,
		OccurredTo:   occurredTo,
		Limit:        limit,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "ledger_query_failed", err.Error(), endpoint, false, "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":       items,
		"count":       len(items),
		"limit":       limit,
		"supplier_id": supplierID,
		"filters": map[string]any{
			"order_id":      strings.TrimSpace(r.URL.Query().Get("order_id")),
			"session_id":    strings.TrimSpace(r.URL.Query().Get("session_id")),
			"gateway":       strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("gateway"))),
			"entry_type":    strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("entry_type"))),
			"occurred_from": formatOptionalTime(occurredFrom),
			"occurred_to":   formatOptionalTime(occurredTo),
		},
	})
}

// HandleSettlementAuthority serves GET /v1/payment/settlement/authority.
func (s *Service) HandleSettlementAuthority(w http.ResponseWriter, r *http.Request) {
	const endpoint = "/v1/payment/settlement/authority"
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", endpoint, false, "")
		return
	}
	if s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "settlement_repository_unavailable", "Settlement repository is unavailable", endpoint, false, "")
		return
	}

	supplierID, ok := resolvePaymentSupplierScope(w, r, s.supplierID, endpoint)
	if !ok {
		return
	}

	groupLimit, err := parseBoundedIntQuery(strings.TrimSpace(r.URL.Query().Get("group_limit")), 200, 1, 1000)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_group_limit", "group_limit must be between 1 and 1000", endpoint, false, "")
		return
	}

	occurredFrom, err := parseRFC3339QueryTime(strings.TrimSpace(r.URL.Query().Get("occurred_from")))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_from", "occurred_from must be RFC3339", endpoint, false, "")
		return
	}
	occurredTo, err := parseRFC3339QueryTime(strings.TrimSpace(r.URL.Query().Get("occurred_to")))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_to", "occurred_to must be RFC3339", endpoint, false, "")
		return
	}
	if occurredFrom != nil && occurredTo != nil && occurredFrom.After(*occurredTo) {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_range", "occurred_from must be before or equal to occurred_to", endpoint, false, "")
		return
	}

	gateway := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("gateway")))
	entryType := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("entry_type")))

	rows, err := s.repo.SummarizeLedgerEntries(r.Context(), SettlementAuthorityQuery{
		SupplierID:   supplierID,
		Gateway:      gateway,
		EntryType:    entryType,
		OccurredFrom: occurredFrom,
		OccurredTo:   occurredTo,
		GroupLimit:   groupLimit,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "settlement_summary_failed", err.Error(), endpoint, false, "")
		return
	}

	totalsByCurrency := make(map[string]SettlementCurrencyTotal)
	var entryCountTotal int64
	for _, row := range rows {
		entryCountTotal += row.EntryCount
		totals := totalsByCurrency[row.Currency]
		totals.Currency = row.Currency
		totals.EntryCount += row.EntryCount
		totals.AmountMinorTotal += row.AmountMinorTotal
		totalsByCurrency[row.Currency] = totals
	}

	currencyTotals := make([]SettlementCurrencyTotal, 0, len(totalsByCurrency))
	for _, totals := range totalsByCurrency {
		currencyTotals = append(currencyTotals, totals)
	}
	sort.Slice(currencyTotals, func(i, j int) bool {
		return currencyTotals[i].Currency < currencyTotals[j].Currency
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":              rows,
		"count":              len(rows),
		"group_limit":        groupLimit,
		"supplier_id":        supplierID,
		"entry_count_total":  entryCountTotal,
		"totals_by_currency": currencyTotals,
		"filters": map[string]any{
			"gateway":       gateway,
			"entry_type":    entryType,
			"occurred_from": formatOptionalTime(occurredFrom),
			"occurred_to":   formatOptionalTime(occurredTo),
		},
	})
}

// HandleReconciliationMismatches serves GET /v1/payment/reconciliation/mismatches.
func (s *Service) HandleReconciliationMismatches(w http.ResponseWriter, r *http.Request) {
	const endpoint = "/v1/payment/reconciliation/mismatches"
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", endpoint, false, "")
		return
	}
	if s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "reconciliation_repository_unavailable", "Reconciliation repository is unavailable", endpoint, false, "")
		return
	}

	supplierID, ok := resolvePaymentSupplierScope(w, r, s.supplierID, endpoint)
	if !ok {
		return
	}

	groupLimit, err := parseBoundedIntQuery(strings.TrimSpace(r.URL.Query().Get("group_limit")), 200, 1, 1000)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_group_limit", "group_limit must be between 1 and 1000", endpoint, false, "")
		return
	}

	mismatchThresholdMinor, err := parseNonNegativeInt64Query(strings.TrimSpace(r.URL.Query().Get("mismatch_threshold_minor")), 0)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_mismatch_threshold_minor", "mismatch_threshold_minor must be a non-negative integer", endpoint, false, "")
		return
	}

	occurredFrom, err := parseRFC3339QueryTime(strings.TrimSpace(r.URL.Query().Get("occurred_from")))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_from", "occurred_from must be RFC3339", endpoint, false, "")
		return
	}
	occurredTo, err := parseRFC3339QueryTime(strings.TrimSpace(r.URL.Query().Get("occurred_to")))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_to", "occurred_to must be RFC3339", endpoint, false, "")
		return
	}
	if occurredFrom != nil && occurredTo != nil && occurredFrom.After(*occurredTo) {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_range", "occurred_from must be before or equal to occurred_to", endpoint, false, "")
		return
	}

	gateway := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("gateway")))
	rows, err := s.repo.SummarizeLedgerEntries(r.Context(), SettlementAuthorityQuery{
		SupplierID:   supplierID,
		Gateway:      gateway,
		OccurredFrom: occurredFrom,
		OccurredTo:   occurredTo,
		GroupLimit:   groupLimit,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "reconciliation_summary_failed", err.Error(), endpoint, false, "")
		return
	}

	type gatewayCurrencyAggregate struct {
		Gateway         string
		Currency        string
		EntryCountTotal int64
		GroupCount      int64
		CreditAmount    int64
		DebitAmount     int64
		NetAmount       int64
		FirstOccurredAt time.Time
		LastOccurredAt  time.Time
		EntryTypeTotals []ReconciliationEntryTypeTotal
	}

	aggByKey := make(map[string]*gatewayCurrencyAggregate)
	for _, row := range rows {
		key := strings.Join([]string{row.Gateway, row.Currency}, "|")
		agg, exists := aggByKey[key]
		if !exists {
			agg = &gatewayCurrencyAggregate{
				Gateway:         row.Gateway,
				Currency:        row.Currency,
				FirstOccurredAt: row.FirstOccurredAt,
				LastOccurredAt:  row.LastOccurredAt,
			}
			aggByKey[key] = agg
		}

		sign := reconciliationEntrySign(row.EntryType)
		signedAmount := row.AmountMinorTotal * sign
		if signedAmount > 0 {
			agg.CreditAmount += signedAmount
		} else if signedAmount < 0 {
			agg.DebitAmount += -signedAmount
		}

		agg.EntryCountTotal += row.EntryCount
		agg.GroupCount++
		agg.NetAmount += signedAmount
		if row.FirstOccurredAt.Before(agg.FirstOccurredAt) {
			agg.FirstOccurredAt = row.FirstOccurredAt
		}
		if row.LastOccurredAt.After(agg.LastOccurredAt) {
			agg.LastOccurredAt = row.LastOccurredAt
		}
		agg.EntryTypeTotals = append(agg.EntryTypeTotals, ReconciliationEntryTypeTotal{
			EntryType:              row.EntryType,
			EntryCount:             row.EntryCount,
			AmountMinorTotal:       row.AmountMinorTotal,
			SignedAmountMinorTotal: signedAmount,
		})
	}

	mismatches := make([]ReconciliationMismatchRow, 0, len(aggByKey))
	for _, agg := range aggByKey {
		if absInt64(agg.NetAmount) <= mismatchThresholdMinor {
			continue
		}

		sort.Slice(agg.EntryTypeTotals, func(i, j int) bool {
			left := absInt64(agg.EntryTypeTotals[i].SignedAmountMinorTotal)
			right := absInt64(agg.EntryTypeTotals[j].SignedAmountMinorTotal)
			if left == right {
				return agg.EntryTypeTotals[i].EntryType < agg.EntryTypeTotals[j].EntryType
			}
			return left > right
		})

		mismatches = append(mismatches, ReconciliationMismatchRow{
			Gateway:         agg.Gateway,
			Currency:        agg.Currency,
			EntryCountTotal: agg.EntryCountTotal,
			GroupCount:      agg.GroupCount,
			CreditAmount:    agg.CreditAmount,
			DebitAmount:     agg.DebitAmount,
			NetAmount:       agg.NetAmount,
			FirstOccurredAt: agg.FirstOccurredAt,
			LastOccurredAt:  agg.LastOccurredAt,
			EntryTypeTotals: agg.EntryTypeTotals,
		})
	}

	sort.Slice(mismatches, func(i, j int) bool {
		left := absInt64(mismatches[i].NetAmount)
		right := absInt64(mismatches[j].NetAmount)
		if left == right {
			if mismatches[i].Gateway == mismatches[j].Gateway {
				return mismatches[i].Currency < mismatches[j].Currency
			}
			return mismatches[i].Gateway < mismatches[j].Gateway
		}
		return left > right
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":                    mismatches,
		"count":                    len(mismatches),
		"analyzed_group_count":     len(rows),
		"group_limit":              groupLimit,
		"mismatch_threshold_minor": mismatchThresholdMinor,
		"supplier_id":              supplierID,
		"filters": map[string]any{
			"gateway":       gateway,
			"occurred_from": formatOptionalTime(occurredFrom),
			"occurred_to":   formatOptionalTime(occurredTo),
		},
	})
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

	// Docs-only integration requirement: Authoritative status verification
	if req.Status == "SUCCESS" || req.Status == "SETTLED" {
		verifiedStatus, err := s.verifyGlobalPayPaymentStatus(r.Context(), req.TransactionID)
		if err != nil {
			writeJSONError(w, http.StatusBadGateway, "verification_failed", fmt.Sprintf("failed to verify payment status: %v", err), endpoint, false, "")
			return
		}
		if verifiedStatus != req.Status {
			writeJSONError(w, http.StatusConflict, "status_mismatch", "webhook status does not match authoritative status", endpoint, false, "")
			return
		}
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
		SupplierID:     s.supplierID,
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

func (s *Service) verifyGlobalPayPaymentStatus(ctx context.Context, transactionID string) (string, error) {
	if s.globalPayUsername == "" || s.globalPayPassword == "" {
		return "", fmt.Errorf("globalpay credentials missing (username or password)")
	}

	env := strings.ToLower(s.globalPayEnv)
	var baseURL string
	switch env {
	case "production":
		baseURL = "https://checkout-api.globalpay.uz/checkout"
	case "staging":
		baseURL = "https://checkout-api-staging.globalpay.uz/checkout"
	default:
		baseURL = "https://checkout-api-dev.globalpay.uz/checkout"
	}

	// 1. Authenticate to get access token
	authURL := fmt.Sprintf("%s/v1/merchant/auth", baseURL)
	reqBody, _ := json.Marshal(map[string]string{
		"username": s.globalPayUsername,
		"password": s.globalPayPassword,
	})

	authReq, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	authReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	authResp, err := client.Do(authReq)
	if err != nil {
		return "", fmt.Errorf("auth request failed: %w", err)
	}
	defer authResp.Body.Close()

	if authResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(authResp.Body)
		return "", fmt.Errorf("auth failed with status %d: %s", authResp.StatusCode, string(b))
	}

	var authData struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(authResp.Body).Decode(&authData); err != nil {
		return "", fmt.Errorf("auth decode failed: %w", err)
	}
	if authData.AccessToken == "" {
		return "", fmt.Errorf("empty access token")
	}

	// 2. Get payment status
	statusURL := fmt.Sprintf("%s/v1/payment/%s", baseURL, transactionID)
	statusReq, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return "", err
	}
	statusReq.Header.Set("Authorization", "Bearer "+authData.AccessToken)

	statusResp, err := client.Do(statusReq)
	if err != nil {
		return "", fmt.Errorf("status request failed: %w", err)
	}
	defer statusResp.Body.Close()

	if statusResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(statusResp.Body)
		return "", fmt.Errorf("status query failed with status %d: %s", statusResp.StatusCode, string(b))
	}

	var statusData struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(statusResp.Body).Decode(&statusData); err != nil {
		return "", fmt.Errorf("status decode failed: %w", err)
	}

	return strings.ToUpper(strings.TrimSpace(statusData.Status)), nil
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

	envelope, err := parseVerifiedAdyenWebhookEnvelope(body, s.adyenWebhookSecret)
	if err != nil {
		switch {
		case errors.Is(err, errAdyenInvalidJSONPayload):
			writeJSONError(w, http.StatusBadRequest, "invalid_json_payload", "Invalid JSON payload", endpoint, false, "")
		case errors.Is(err, errAdyenMissingNotificationItems):
			writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "notificationItems is required", endpoint, false, "")
		case errors.Is(err, errAdyenInvalidSignature):
			writeJSONError(w, http.StatusUnauthorized, "invalid_signature", "Invalid webhook signature", endpoint, false, "")
		default:
			writeJSONError(w, http.StatusBadRequest, "invalid_json_payload", "Invalid JSON payload", endpoint, false, "")
		}
		return
	}

	for i := range envelope.NotificationItems {
		item := envelope.NotificationItems[i].Item
		if err := validateAdyenNotificationItem(item); err != nil {
			writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", err.Error(), endpoint, false, "")
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
		replayed, err := s.isWebhookReplay(r.Context(), webhookKey, bodyHash)
		if err != nil {
			if errors.Is(err, idempotency.ErrConflict) {
				writeJSONError(w, http.StatusConflict, "idempotency_key_payload_mismatch", "idempotency key payload mismatch", endpoint, false, "")
				return
			}
			s.log.Warn("webhook idempotency guard failed", "key", webhookKey, "err", err)
		}
		if replayed {
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
			SupplierID:     s.supplierID,
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

var (
	errAdyenInvalidJSONPayload       = errors.New("adyen webhook invalid json payload")
	errAdyenMissingNotificationItems = errors.New("adyen webhook missing notification items")
	errAdyenInvalidSignature         = errors.New("adyen webhook invalid signature")
)

func parseVerifiedAdyenWebhookEnvelope(body []byte, secret string) (adyenWebhookEnvelope, error) {
	var rawEnvelope adyenRawWebhookEnvelope
	if err := json.Unmarshal(body, &rawEnvelope); err != nil {
		return adyenWebhookEnvelope{}, errAdyenInvalidJSONPayload
	}
	if len(rawEnvelope.NotificationItems) == 0 {
		return adyenWebhookEnvelope{}, errAdyenMissingNotificationItems
	}
	envelope := adyenWebhookEnvelope{
		NotificationItems: make([]adyenNotificationItemWrapper, 0, len(rawEnvelope.NotificationItems)),
	}
	for _, wrapper := range rawEnvelope.NotificationItems {
		if len(wrapper.Item) == 0 {
			return adyenWebhookEnvelope{}, errAdyenInvalidJSONPayload
		}
		var signedItem adyenSignedNotificationItem
		if err := json.Unmarshal(wrapper.Item, &signedItem); err != nil {
			return adyenWebhookEnvelope{}, errAdyenInvalidJSONPayload
		}
		item := adyenNotificationItem(signedItem)
		if !verifyAdyenNotificationSignature(item, secret) {
			return adyenWebhookEnvelope{}, errAdyenInvalidSignature
		}
		envelope.NotificationItems = append(envelope.NotificationItems, adyenNotificationItemWrapper{Item: item})
	}
	return envelope, nil
}

func (s *Service) isWebhookReplay(ctx context.Context, webhookKey, bodyHash string) (bool, error) {
	if s.idem == nil {
		return false, nil
	}
	_, hit, err := idempotency.Guard(ctx, s.idem, webhookKey, bodyHash)
	if err != nil {
		return false, err
	}
	return hit, nil
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
	row.SupplierID = s.supplierID
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
			row.RetailerID = strings.TrimSpace(intent.Metadata["retailer_id"])
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
			row.RetailerID = strings.TrimSpace(charge.Metadata["retailer_id"])
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
	s.handleCheckoutWithBody(mode, w, r, body)
}

func (s *Service) handleCheckoutWithBody(mode string, w http.ResponseWriter, r *http.Request, body []byte) {
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
	req.RetailerID = retailerID

	session, attempt, executionResult, err := s.initCheckoutSession(r.Context(), mode, req)
	if err != nil {
		s.writeExecutionError(w, "/v1/checkout", err)
		return
	}

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
		return outbox.EmitJSON(ctx, txn, events.AggregateSession, row.WebhookID, events.TopicMain, events.FinanceEvent{
			BaseEvent:     events.BaseEvent{Type: eventType, Timestamp: now.Format(time.RFC3339Nano)},
			SessionID:     row.SessionID,
			OrderID:       row.OrderID,
			SupplierID:    row.SupplierID,
			RetailerID:    row.RetailerID,
			Gateway:       row.Gateway,
			Status:        row.Status,
			AmountMinor:   row.AmountMinor,
			Currency:      row.Currency,
			TransactionID: row.TransactionID,
			Source:        source,
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

func resolvePaymentSupplierScope(w http.ResponseWriter, r *http.Request, fallbackSupplierID string, endpoint string) (string, bool) {
	supplierID := strings.TrimSpace(fallbackSupplierID)
	if scopedSupplierID, ok := auth.ResolveSupplierID(r.Context()); ok {
		scopedSupplierID = strings.TrimSpace(scopedSupplierID)
		if scopedSupplierID != "" {
			supplierID = scopedSupplierID
		}
	}

	requestedSupplierID := strings.TrimSpace(r.URL.Query().Get("supplier_id"))
	if requestedSupplierID != "" {
		if supplierID != "" && requestedSupplierID != supplierID {
			writeJSONError(w, http.StatusForbidden, "forbidden", "supplier scope mismatch", endpoint, false, "")
			return "", false
		}
		supplierID = requestedSupplierID
	}

	return supplierID, true
}

func parseBoundedIntQuery(raw string, defaultValue int, minValue int, maxValue int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < minValue || parsed > maxValue {
		return 0, fmt.Errorf("out of range")
	}
	return parsed, nil
}

func parseNonNegativeInt64Query(raw string, defaultValue int64) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("out of range")
	}
	return parsed, nil
}

func parseRFC3339QueryTime(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func formatOptionalTime(v *time.Time) any {
	if v == nil {
		return nil
	}
	return v.UTC().Format(time.RFC3339Nano)
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// SignedSettlementEntryAmount applies the canonical reconciliation sign rules
// to a grouped settlement authority amount so adjacent read models can stay in
// lockstep with finance mismatch math.
func SignedSettlementEntryAmount(entryType string, amountMinor int64) int64 {
	return amountMinor * reconciliationEntrySign(entryType)
}

func reconciliationEntrySign(entryType string) int64 {
	t := strings.ToUpper(strings.TrimSpace(entryType))
	if t == "" {
		return 0
	}
	if strings.Contains(t, "FAILED") || strings.Contains(t, "CANCEL") || strings.Contains(t, "VOID") || strings.Contains(t, "PENDING") || strings.Contains(t, "REQUIRED") || strings.Contains(t, "UNKNOWN") {
		return 0
	}
	if strings.Contains(t, "REVERSAL") {
		return 1
	}
	if strings.Contains(t, "CHARGEBACK") || strings.Contains(t, "REFUND") {
		return -1
	}
	if strings.Contains(t, "PAID") || strings.Contains(t, "CAPTURED") || strings.Contains(t, "CLEARED") || strings.Contains(t, "SETTLED") || strings.Contains(t, "SUCCESS") || strings.Contains(t, "SUCCEEDED") || strings.Contains(t, "COMPLETED") || strings.Contains(t, "AUTHORIZED") {
		return 1
	}
	return 0
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
