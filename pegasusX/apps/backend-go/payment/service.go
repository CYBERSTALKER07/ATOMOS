// chargeback/reversal, and gateway webhooks.
package payment

import (
	"context"
	"crypto/sha256"
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
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
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
	FindStuckSessions(ctx context.Context, cutoff time.Time, limit int) ([]SessionRecord, error)
	GetSession(ctx context.Context, sessionID string) (SessionRecord, bool, error)
	GetSessionByOrderID(ctx context.Context, orderID string) (SessionRecord, bool, error)
	HasChargebackForOrder(ctx context.Context, orderID string) (bool, error)

	ListLedgerEntries(ctx context.Context, q LedgerQuery) ([]LedgerEntryRecord, error)
	SummarizeLedgerEntries(ctx context.Context, q SettlementAuthorityQuery) ([]SettlementAuthorityRow, error)

	CreatePayer(ctx context.Context, p Payer) error
	GetPayer(ctx context.Context, payerID string) (Payer, error)
	UpdatePayer(ctx context.Context, p Payer) error
	ListPayers(ctx context.Context, limit, offset int) ([]Payer, error)
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
	// Source is optional ledger source tag (e.g. claims.settle_chargeback:clm_…).
	Source string
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
	// seedSupplierID is fixtures/bootstrap only — request paths use resolveSupplierID.
	seedSupplierID string
	currency       string
	execution      *ProviderExecutionRouter

	cartCheckout    CartCheckoutHandler
	checkoutPreview CheckoutPreviewHandler
	orderReader  OrderCheckoutReader
	policy       PolicyResolver

	globalPayEnv           string
	globalPayUsername      string
	globalPayPassword      string
	globalPayWebhookSecret string
	adyenWebhookSecret     string
	stripeWebhookSecret    string
	paymeWebhookSecret     string
	clickWebhookSecret     string

	webhookInbox *WebhookInboxStore

	fx     *fxrates.Service
	log    *slog.Logger
	now    func() time.Time
	newID  func(prefix string) string
}

// ServiceConfig is constructor input.
type ServiceConfig struct {
	Repo                            Repository
	Cache                           *cache.Cache
	Idem                            idempotency.Store
	// SeedSupplierID is bootstrap/fixture fallback only (Gate 5 Week 11).
	SeedSupplierID                  string
	// SupplierID is deprecated; use SeedSupplierID.
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
	PaymeWebhookSecret     string
	ClickWebhookSecret     string
	WebhookInbox           *WebhookInboxStore

	Log   *slog.Logger
	Now   func() time.Time
	NewID func(prefix string) string

	Policy PolicyResolver

	Fx *fxrates.Service
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

// Stripe types removed in favor of SDK

// Adyen types removed in favor of SDK

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
	if c.PaymeWebhookSecret == "" {
		c.PaymeWebhookSecret = "dev-payme-secret"
	}
	if c.ClickWebhookSecret == "" {
		c.ClickWebhookSecret = "dev-click-secret"
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
	seedID := strings.TrimSpace(c.SeedSupplierID)
	if seedID == "" {
		seedID = strings.TrimSpace(c.SupplierID)
	}
	return &Service{
		repo:                   c.Repo,
		cache:                  c.Cache,
		idem:                   c.Idem,
		seedSupplierID:         seedID,
		currency:               c.Currency,
		execution:              c.Execution,
		globalPayEnv:           c.GlobalPayEnv,
		globalPayUsername:      c.GlobalPayUsername,
		globalPayPassword:      c.GlobalPayPassword,
		globalPayWebhookSecret: c.GlobalPayWebhookSecret,
		adyenWebhookSecret:     c.AdyenWebhookSecret,
		stripeWebhookSecret:    c.StripeWebhookSecret,
		paymeWebhookSecret:     c.PaymeWebhookSecret,
		clickWebhookSecret:     c.ClickWebhookSecret,
		webhookInbox:           c.WebhookInbox,
		policy:                 c.Policy,
		fx:                     c.Fx,
		log:                    c.Log,
		now:                    c.Now,
		newID:                  c.NewID,
	}
}

// SetFxRates attaches theatre #13 ConvertMinor for settlement operating rollups.
func (s *Service) SetFxRates(fx *fxrates.Service) {
	if s == nil {
		return
	}
	s.fx = fx
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

// HandleCheckoutPreview serves POST /v1/checkout/preview (inventory dry-run).
func (s *Service) HandleCheckoutPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "/v1/checkout/preview", false, "")
		return
	}
	if s.checkoutPreview == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "checkout_preview_unavailable", "checkout preview handler not configured", "/v1/checkout/preview", false, "")
		return
	}
	s.checkoutPreview.HandleCheckoutPreview(w, r)
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
	// Theatre #13: reject chargeback currency ≠ existing session currency when known.
	if session, ok, err := s.repo.GetSessionByOrderID(r.Context(), req.OrderID); err == nil && ok {
		sessCur := strings.TrimSpace(session.Currency)
		if sessCur != "" {
			if err := fxrates.AssertSameCurrency(sessCur, req.Currency); err != nil {
				writeJSONError(w, http.StatusUnprocessableEntity, "currency_mismatch", "chargeback currency must match payment session currency", "/v1/payment/chargeback", false, "")
				return
			}
		}
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
		SupplierID:   s.resolveSupplierID(r.Context()),
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

	session, found, err := s.repo.GetSession(r.Context(), req.SessionID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to retrieve session", "/v1/payment/chargeback/reversal", false, "")
		return
	}
	if !found {
		writeJSONError(w, http.StatusNotFound, "session_not_found", "Session not found", "/v1/payment/chargeback/reversal", false, "")
		return
	}

	hasChargeback, err := s.repo.HasChargebackForOrder(r.Context(), session.OrderID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to verify chargeback status", "/v1/payment/chargeback/reversal", false, "")
		return
	}
	if !hasChargeback {
		writeJSONError(w, http.StatusConflict, "chargeback_required", "Cannot reverse a chargeback that has not been recorded", "/v1/payment/chargeback/reversal", false, "")
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
		SupplierID:  s.resolveSupplierID(r.Context()),
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
// It resolves the checkout session for the order (gateway + provider payment id) and
// short-circuits when the payment is already settled, so retries never double-capture.
// Returns the provider reference of the confirmed capture.
func (s *Service) CaptureCardPayment(ctx context.Context, orderID string, amountMinor int64, currency string) (string, error) {
	gateway := "GLOBAL_PAY"
	sessionID := ""
	if s.repo != nil {
		if session, ok, err := s.repo.GetSessionByOrderID(ctx, orderID); err == nil && ok {
			if g := strings.ToUpper(strings.TrimSpace(session.Gateway)); g != "" {
				gateway = g
			}
			sessionID = session.SessionID
			switch strings.ToUpper(strings.TrimSpace(session.Status)) {
			case "PAID", "CAPTURED", "SUCCESS":
				return sessionID, nil
			}
		}
	}
	// Query-before-capture: if the provider already settled this payment (a prior
	// capture succeeded but our confirmation write was lost), do not capture twice.
	if res, err := s.execution.Execute(ctx, ExecutionRequest{
		Gateway:     gateway,
		Action:      ExecutionActionStatusCheck,
		OrderID:     orderID,
		SessionID:   sessionID,
		AmountMinor: amountMinor,
		Currency:    currency,
	}); err == nil {
		switch strings.ToUpper(strings.TrimSpace(res.ProviderRef)) {
		case "PAID", "CAPTURED", "SUCCESS":
			if res.ProviderRef != "" {
				return res.ProviderRef, nil
			}
			return sessionID, nil
		}
	}
	result, err := s.execution.Execute(ctx, ExecutionRequest{
		Gateway:     gateway,
		Action:      ExecutionActionCheckoutCapture,
		OrderID:     orderID,
		SessionID:   sessionID,
		AmountMinor: amountMinor,
		Currency:    currency,
	})
	if err != nil {
		return "", err
	}
	return result.ProviderRef, nil
}

// RefundCardPayment reverses a captured card payment (full or partial) via the
// gateway. Implements order.PaymentRefunder. Retry-safety comes from the
// execution layer's idempotency keys plus the caller's refund idempotency key.
func (s *Service) RefundCardPayment(ctx context.Context, orderID string, amountMinor int64, currency string) (string, error) {
	if s == nil || s.execution == nil {
		return "", fmt.Errorf("payment execution unavailable")
	}
	if strings.TrimSpace(orderID) == "" || amountMinor <= 0 {
		return "", fmt.Errorf("order_id and positive amount_minor required")
	}
	gateway := "GLOBAL_PAY"
	sessionID := ""
	if s.repo != nil {
		if session, ok, err := s.repo.GetSessionByOrderID(ctx, orderID); err == nil && ok {
			if g := strings.ToUpper(strings.TrimSpace(session.Gateway)); g != "" {
				gateway = g
			}
			sessionID = session.SessionID
		}
	}
	if currency == "" {
		currency = s.currency
	}
	if currency == "" {
		currency = "UZS"
	}
	result, err := s.execution.Execute(ctx, ExecutionRequest{
		Gateway:     gateway,
		Action:      ExecutionActionRefund,
		OrderID:     orderID,
		SessionID:   sessionID,
		AmountMinor: amountMinor,
		Currency:    currency,
	})
	if err != nil {
		return "", err
	}
	return result.ProviderRef, nil
}

// ClaimChargebackInput is used by claims.Service for marketplace chargebacks.
type ClaimChargebackInput struct {
	ClaimID           string
	OrderID           string
	SupplierID        string
	RetailerID        string
	AmountMinor       int64
	Currency          string
	SkipGatewayRefund bool
}

// ClaimChargebackResult is the settlement outcome for a logistics claim.
type ClaimChargebackResult struct {
	ChargebackID    string
	AmountMinor     int64
	Currency        string
	Gateway         string
	GatewayRefunded bool
	ProviderRef     string
	Mode            string
}

// SettleClaimChargeback debits the supplier (ledger chargeback) and, when the
// order was paid by Global Pay card, attempts a partial PSP refund to the retailer.
//
// Cash / credit deliveries settle as LEDGER_ONLY — money never left the retailer's
// bank; we reduce supplier settlement authority for the returned qty instead.
func (s *Service) SettleClaimChargeback(ctx context.Context, in ClaimChargebackInput) (ClaimChargebackResult, error) {
	if s == nil || s.repo == nil {
		return ClaimChargebackResult{}, fmt.Errorf("payment service unavailable")
	}
	if strings.TrimSpace(in.OrderID) == "" || in.AmountMinor <= 0 {
		return ClaimChargebackResult{}, fmt.Errorf("order_id and positive amount_minor required")
	}
	currency := strings.TrimSpace(in.Currency)
	if currency == "" {
		currency = s.currency
	}
	if currency == "" {
		currency = "UZS"
	}
	supplierID := strings.TrimSpace(in.SupplierID)
	if supplierID == "" {
		supplierID = s.resolveSupplierID(ctx)
	}
	now := s.now()
	gateway := "INTERNAL"
	sessionID := ""
	if session, ok, err := s.repo.GetSessionByOrderID(ctx, in.OrderID); err == nil && ok {
		gateway = strings.ToUpper(strings.TrimSpace(session.Gateway))
		sessionID = session.SessionID
		// Cap chargeback/refund at session amount (cannot reverse more than was paid).
		if session.AmountMinor > 0 && in.AmountMinor > session.AmountMinor {
			in.AmountMinor = session.AmountMinor
		}
	}
	if gateway == "" {
		gateway = "INTERNAL"
	}

	// Deterministic id so approve retries InsertOrUpdate the same chargeback row.
	chargebackID := strings.TrimSpace(in.ClaimID)
	if chargebackID == "" {
		chargebackID = s.newID("chargeback")
	} else if strings.HasPrefix(chargebackID, "clm_") {
		chargebackID = "chargeback_" + chargebackID
	} else {
		chargebackID = "chargeback_clm_" + chargebackID
	}

	// 1) Always record immutable ledger chargeback (supplier debit / settlement clawback).
	if _, err := s.execution.Execute(ctx, ExecutionRequest{
		Gateway:     gateway,
		Action:      ExecutionActionChargebackRecord,
		OrderID:     in.OrderID,
		SessionID:   sessionID,
		AmountMinor: in.AmountMinor,
		Currency:    currency,
	}); err != nil {
		return ClaimChargebackResult{}, err
	}
	claimTag := "claims.settle_chargeback:" + strings.TrimSpace(in.ClaimID)
	rec := ChargebackRecord{
		ChargebackID: chargebackID,
		OrderID:      strings.TrimSpace(in.OrderID),
		SupplierID:   supplierID,
		RetailerID:   strings.TrimSpace(in.RetailerID),
		Gateway:      gateway,
		AmountMinor:  in.AmountMinor,
		Currency:     currency,
		CreatedAt:    now,
		Source:       claimTag,
	}
	if err := s.repo.SaveChargeback(ctx, rec, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateSession, rec.ChargebackID, events.TopicMain, events.FinanceEvent{
			BaseEvent:       events.BaseEvent{Type: events.EventPaymentRequired, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:         rec.OrderID,
			SupplierID:      rec.SupplierID,
			RetailerID:      rec.RetailerID,
			Gateway:         rec.Gateway,
			Status:          "CLAIM_CHARGEBACK_RECORDED",
			ExecutionAction: string(ExecutionActionChargebackRecord),
			AmountMinor:     rec.AmountMinor,
			Currency:        rec.Currency,
			Source:          claimTag,
		})
	}); err != nil {
		return ClaimChargebackResult{}, err
	}

	out := ClaimChargebackResult{
		ChargebackID: rec.ChargebackID,
		AmountMinor:  rec.AmountMinor,
		Currency:     currency,
		Gateway:      gateway,
		Mode:         "LEDGER_ONLY",
	}

	// 2) Optional card refund via Global Pay for partial returns.
	if in.SkipGatewayRefund {
		return out, nil
	}
	if !isGlobalPayGateway(gateway) {
		return out, nil
	}
	if s.execution == nil {
		return out, nil
	}
	refundRes, err := s.execution.Execute(ctx, ExecutionRequest{
		Gateway:     "GLOBAL_PAY",
		Action:      ExecutionActionRefund,
		OrderID:     in.OrderID,
		SessionID:   sessionID,
		AmountMinor: in.AmountMinor,
		Currency:    currency,
	})
	if err != nil {
		// Ledger already debited; surface soft-fail so ops can manual refund.
		out.Mode = "LEDGER_ONLY_GATEWAY_REFUND_FAILED"
		out.ProviderRef = err.Error()
		return out, nil
	}
	out.GatewayRefunded = true
	out.ProviderRef = refundRes.ProviderRef
	out.Mode = "LEDGER_AND_GATEWAY_REFUND"
	out.Gateway = refundRes.ResolvedGateway
	return out, nil
}

func isGlobalPayGateway(g string) bool {
	g = strings.ToUpper(strings.TrimSpace(g))
	return g == "GLOBAL_PAY" || g == "GLOBALPAY" || g == "GP"
}

// HandleClaimChargebacks serves GET /v1/supplier/claim-chargebacks?limit=&order_id=.
// Lists ledger CHARGEBACK_RECORDED rows originating from claims settlement for the admin's supplier.
func (s *Service) HandleClaimChargebacks(w http.ResponseWriter, r *http.Request) {
	const endpoint = "/v1/supplier/claim-chargebacks"
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", endpoint, false, "")
		return
	}
	if s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "ledger_repository_unavailable", "Ledger repository is unavailable", endpoint, false, "")
		return
	}
	supplierID, ok := resolvePaymentSupplierScope(w, r, s.seedSupplierID, endpoint)
	if !ok {
		return
	}
	limit, err := parseBoundedIntQuery(strings.TrimSpace(r.URL.Query().Get("limit")), 50, 1, 200)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 200", endpoint, false, "")
		return
	}
	// Pull a wider page then filter to claim-originated chargebacks.
	items, err := s.repo.ListLedgerEntries(r.Context(), LedgerQuery{
		SupplierID: supplierID,
		OrderID:    strings.TrimSpace(r.URL.Query().Get("order_id")),
		EntryType:  "CHARGEBACK_RECORDED",
		Limit:      limit * 3,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "ledger_query_failed", err.Error(), endpoint, false, "")
		return
	}
	out := make([]LedgerEntryRecord, 0, len(items))
	for _, it := range items {
		src := strings.ToLower(it.Source)
		ref := strings.ToLower(it.ReferenceID)
		if strings.Contains(src, "claims.settle") ||
			strings.HasPrefix(ref, "chargeback_clm_") ||
			strings.Contains(ref, "clm_") {
			out = append(out, it)
			if len(out) >= limit {
				break
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":       out,
		"count":       len(out),
		"limit":       limit,
		"supplier_id": supplierID,
	})
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

	supplierID, ok := resolvePaymentSupplierScope(w, r, s.seedSupplierID, endpoint)
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

	supplierID, ok := resolvePaymentSupplierScope(w, r, s.seedSupplierID, endpoint)
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

	operating := fxrates.NormalizeCurrency(s.currency)
	if operating == "" {
		operating = "UZS"
	}
	opTotal, opPartial := rollupOperatingCurrencyMinor(r.Context(), s.fx, operating, rows, s.now)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":                          rows,
		"count":                          len(rows),
		"group_limit":                    groupLimit,
		"supplier_id":                    supplierID,
		"entry_count_total":              entryCountTotal,
		"totals_by_currency":             currencyTotals,
		"operating_currency":             operating,
		"operating_currency_total_minor": opTotal,
		"operating_conversion_partial":   opPartial,
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

	supplierID, ok := resolvePaymentSupplierScope(w, r, s.seedSupplierID, endpoint)
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
	path := "/v1/checkout/unified"
	if mode == "B2B" {
		path = "/v1/checkout/b2b"
	}
	writeJSONError(w, http.StatusGone, "payment_before_delivery_removed",
		"Pre-delivery payment checkout is disabled; pay at delivery after offload via /v1/order/card-checkout or /v1/order/cash-checkout",
		path, true, "/v1/order/card-checkout")
}

func (s *Service) persistWebhookWithOutbox(ctx context.Context, row WebhookRecord, source string, now time.Time) error {
	eventType := events.EventPaymentRequired
	if isPaymentSuccessStatus(row.Status) {
		eventType = events.EventPaymentCleared
	} else if isPaymentFailureStatus(row.Status) {
		eventType = events.EventPaymentFailed
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
		if s.webhookInbox != nil {
			if qErr := s.webhookInbox.Enqueue(ctx, row, source); qErr == nil {
				return nil
			}
		}
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
	supplierID := auth.PreferTenantSupplierID(r.Context(), fallbackSupplierID)

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

// resolveSupplierID prefers request TenantContext over the bootstrap seed.
func (s *Service) resolveSupplierID(ctx context.Context) string {
	return auth.PreferTenantSupplierID(ctx, s.seedSupplierID)
}

// resolveWebhookSupplierID prefers payment-session tenant, then seed (webhooks lack JWT).
func (s *Service) resolveWebhookSupplierID(ctx context.Context, orderID string) string {
	if oid := strings.TrimSpace(orderID); oid != "" && s.repo != nil {
		if session, ok, err := s.repo.GetSessionByOrderID(ctx, oid); err == nil && ok {
			if sid := strings.TrimSpace(session.SupplierID); sid != "" {
				return sid
			}
		}
	}
	return strings.TrimSpace(s.seedSupplierID)
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

func isPaymentFailureStatus(status string) bool {
	s := strings.ToUpper(strings.TrimSpace(status))
	return s == "FAILED" || s == "DECLINED" || s == "CANCELLED" || s == "VOIDED"
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

// assertSessionCurrency rejects requestCurrency when a known session currency differs (theatre #13).
func (s *Service) assertSessionCurrency(ctx context.Context, sessionID, orderID, requestCurrency string) error {
	if s == nil || s.repo == nil {
		return nil
	}
	requestCurrency = strings.TrimSpace(requestCurrency)
	if requestCurrency == "" {
		return nil
	}
	var session SessionRecord
	var ok bool
	var err error
	if sid := strings.TrimSpace(sessionID); sid != "" {
		session, ok, err = s.repo.GetSession(ctx, sid)
	} else if oid := strings.TrimSpace(orderID); oid != "" {
		session, ok, err = s.repo.GetSessionByOrderID(ctx, oid)
	}
	if err != nil || !ok {
		return nil
	}
	return fxrates.AssertSameCurrency(session.Currency, requestCurrency)
}
