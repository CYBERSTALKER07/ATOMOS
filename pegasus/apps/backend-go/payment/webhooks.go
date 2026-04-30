// Package payment — Cash & GlobalPay Webhook Handlers (Phase 7: Payment Settlement)
//
// POST /v1/webhooks/cash   — Server-to-server callback from Cash Up
// POST /v1/webhooks/global_pay   — Server-to-server JSON-RPC callback from GlobalPay
//
// These endpoints are UNAUTHENTICATED (no JWT). Security is enforced via:
//   - Cash:  MD5 sign_string verification against CLICK_SECRET_KEY
//   - GlobalPay:  Basic Auth header verification against PAYME_MERCHANT_KEY
//
// Idempotency:
//   - Cash:  Keyed by cash_trans_id — duplicate action=1 returns success without re-crediting.
//   - GlobalPay:  Keyed by global_pay_transaction_id — PerformTransaction on an already-SETTLED invoice
//     returns the original result without double-crediting.
//
// Both handlers settle invoices inside a Spanner ReadWriteTransaction.
// Kafka INVOICE_SETTLED events are written through the transactional outbox.
package payment

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"backend-go/cache"
	"backend-go/idempotency"
	"backend-go/outbox"
	"backend-go/telemetry"
	"backend-go/workers"
	wsEvents "backend-go/ws"

	"cloud.google.com/go/spanner"
	"github.com/segmentio/kafka-go"
)

// ─── Webhook Service ─────────────────────────────────────────────────────────

// Kafka event constants are local to avoid the payment -> kafka import cycle.
const (
	eventInvoiceSettled = "INVOICE_SETTLED"
	eventOrderCompleted = "ORDER_COMPLETED"
	topicMain           = "pegasus-logistics-events"
)

// WebhookService holds the Spanner + Kafka handles for webhook processing.
type WebhookService struct {
	Spanner       *spanner.Client
	Producer      *kafka.Writer
	DriverHub     DriverPusher    // Push PAYMENT_SETTLED to driver after GlobalPay settlement
	RetailerHub   RetailerPusher  // Push PAYMENT_FAILED/PAYMENT_SETTLED to retailer
	VaultResolver VaultResolver   // Per-supplier credential vault (nil = ENV-only fallback)
	SessionSvc    *SessionService // Durable payment session engine (nil = legacy-only mode)
}

// VaultResolver fetches decrypted supplier credentials for webhook verification.
type VaultResolver interface {
	GetDecryptedConfigByOrder(ctx context.Context, orderId, gatewayName string) (*VaultConfig, error)
}

// VaultConfig is a minimal credential struct for webhook signature verification
// and split-payment recipient resolution.
type VaultConfig struct {
	SecretKey   string
	MerchantId  string
	ServiceId   string
	RecipientId string // Global Pay split-payment recipient (empty = no split)
}

// DriverPusher abstracts the driver WebSocket hub for testability.
type DriverPusher interface {
	PushToDriver(driverID string, payload interface{}) bool
}

// InvoiceSettledEvent is emitted through the outbox after a successful settlement.
type InvoiceSettledEvent struct {
	InvoiceID  string    `json:"invoice_id"`
	Gateway    string    `json:"gateway"`
	Amount     int64     `json:"amount"`
	RetailerID string    `json:"retailer_id"`
	Timestamp  time.Time `json:"timestamp"`
}

func emitInvoiceSettledOutbox(ctx context.Context, txn *spanner.ReadWriteTransaction, invoiceID, gateway string, amount int64, retailerID string) error {
	event := InvoiceSettledEvent{
		InvoiceID:  invoiceID,
		Gateway:    gateway,
		Amount:     amount,
		RetailerID: retailerID,
		Timestamp:  time.Now().UTC(),
	}
	return outbox.EmitJSON(txn, "MasterInvoice", invoiceID, eventInvoiceSettled, topicMain, event, telemetry.TraceIDFromContext(ctx))
}

// ═══════════════════════════════════════════════════════════════════════════════
type globalPayWebhookRequest struct {
	SessionID         string `json:"session_id"`
	ProviderReference string `json:"service_token"`
	ProviderPaymentID string `json:"payment_id"`
	Status            string `json:"status"`
}

// GLOBAL PAY WEBHOOK
// ═══════════════════════════════════════════════════════════════════════════════

// HandleGlobalPayWebhook verifies a Global Pay callback by re-querying the
// provider status endpoint before settling any invoice.
func (ws *WebhookService) HandleGlobalPayWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Method == http.MethodPost {
		creds, credErr := ResolveGlobalPayCredentials("", "", "")
		if credErr != nil {
			slog.Error("global_pay_webhook.signature_config_missing", "err", credErr)
			http.Error(w, "webhook not configured", http.StatusServiceUnavailable)
			return
		}
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if !validateGlobalPayAuth(authHeader, creds.Password) {
			providerRef := strings.TrimSpace(firstNonEmpty(
				r.URL.Query().Get("service_token"),
				r.URL.Query().Get("serviceToken"),
				r.URL.Query().Get("provider_reference"),
				r.URL.Query().Get("providerReference"),
			))
			if providerRef == "" {
				providerRef = "unknown"
			}
			ws.trackWebhookSigFailure("global_pay", providerRef, r.RemoteAddr)
			http.Error(w, "invalid webhook signature", http.StatusUnauthorized)
			return
		}
	}

	if ws.SessionSvc == nil {
		http.Error(w, "payment session service unavailable", http.StatusServiceUnavailable)
		return
	}

	req, parseErr := parseGlobalPayWebhookRequest(r)
	if parseErr != nil {
		http.Error(w, parseErr.Error(), http.StatusBadRequest)
		return
	}

	applyGlobalPayIdempotencyKey(r, req)
	idempotency.Guard(func(w http.ResponseWriter, r *http.Request) {
		ws.handleGlobalPayWebhookParsed(w, r, req)
	})(w, r)
}

func (ws *WebhookService) handleGlobalPayWebhookParsed(w http.ResponseWriter, r *http.Request, req *globalPayWebhookRequest) {

	session, err := ws.SessionSvc.GetSession(r.Context(), req.SessionID)
	if err != nil {
		http.Error(w, "payment session not found", http.StatusNotFound)
		return
	}
	if session.Gateway != "GLOBAL_PAY" {
		http.Error(w, "payment session gateway mismatch", http.StatusConflict)
		return
	}
	if session.InvoiceID == "" {
		http.Error(w, "payment session missing invoice binding", http.StatusConflict)
		return
	}

	providerReference := firstNonEmpty(req.ProviderReference, session.ProviderReference)
	creds, err := ws.resolveGlobalPayCredentialsForOrder(r.Context(), session.OrderID)
	if err != nil {
		slog.Error("global_pay_webhook.credential_resolution_failed", "order_id", session.OrderID, "err", err)
		http.Error(w, "gateway credentials unavailable", http.StatusServiceUnavailable)
		return
	}

	status, err := VerifyGlobalPayPayment(r.Context(), creds, providerReference, req.ProviderPaymentID)
	if err != nil {
		slog.Error("global_pay_webhook.status_verification_failed", "session_id", session.SessionID, "err", err)
		http.Error(w, "verification failed", http.StatusBadGateway)
		return
	}

	if status.Paid {
		_, settleErr := ws.settleInvoice(r.Context(), session.InvoiceID, session.LockedAmount, "GLOBAL_PAY")
		if settleErr != nil {
			if strings.Contains(settleErr.Error(), "already settled") {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"status": "ALREADY_SETTLED"})
				return
			}
			slog.Error("global_pay_webhook.invoice_settlement_failed", "session_id", session.SessionID, "err", settleErr)
			http.Error(w, settleErr.Error(), http.StatusConflict)
			return
		}

		ws.settlePaymentSession(r.Context(), session.InvoiceID, "GLOBAL_PAY", status.ProviderPaymentID)
		if session.OrderID != "" {
			ws.notifyDriverPaymentSettled(session.OrderID, session.LockedAmount)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":       "SETTLED",
			"session_id":   session.SessionID,
			"invoice_id":   session.InvoiceID,
			"payment_id":   status.ProviderPaymentID,
			"provider_ref": providerReference,
		})
		return
	}

	if status.Failed() {
		if failErr := ws.SessionSvc.FailSession(r.Context(), session.SessionID, firstNonEmpty(status.FailureCode, "GLOBAL_PAY_FAILED"), firstNonEmpty(status.FailureMessage, req.Status, status.RawStatus)); failErr != nil {
			slog.Error("global_pay_webhook.session_fail_mark_error", "session_id", session.SessionID, "err", failErr)
		}

		// Push PAYMENT_FAILED to retailer + driver via WebSocket
		if ws.RetailerHub != nil && session.RetailerID != "" {
			capturedRetailerID := session.RetailerID
			capturedOrderID := session.OrderID
			capturedSessionID := session.SessionID
			capturedMsg := firstNonEmpty(status.FailureMessage, "Payment declined")
			workers.EventPool.Submit(func() {
				ws.RetailerHub.PushToRetailer(capturedRetailerID, map[string]interface{}{
					"type":       wsEvents.EventPaymentFailed,
					"order_id":   capturedOrderID,
					"session_id": capturedSessionID,
					"gateway":    "GLOBAL_PAY",
					"reason":     capturedMsg,
					"message":    "Payment failed — please try again or choose another method",
				})
			})
		}
		if ws.DriverHub != nil && session.OrderID != "" {
			capturedOrderID := session.OrderID
			capturedFailMsg := firstNonEmpty(status.FailureMessage, "Payment failed")
			workers.EventPool.Submit(func() { ws.notifyDriverPaymentFailed(capturedOrderID, capturedFailMsg) })
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":     firstNonEmpty(status.RawStatus, req.Status, "PENDING"),
		"session_id": session.SessionID,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// SHARED INTERNALS
// ═══════════════════════════════════════════════════════════════════════════════

// settleInvoice transitions a MasterInvoice from PENDING → SETTLED inside a
// Spanner ReadWriteTransaction. Returns the RetailerID for Kafka event emission.
// If the invoice is already SETTLED, returns a sentinel error for idempotency.
//
// WEBHOOK RACE SAFETY (F-1):
// Concurrent webhook deliveries (e.g. Cash + GlobalPay retry, or duplicate
// provider callbacks) are safe because Spanner's ReadWriteTransaction provides
// serializable isolation. The first transaction to commit wins; subsequent
// concurrent calls will either:
//   - See state="SETTLED" and return the idempotent sentinel error, or
//   - Be aborted by Spanner's OCC and retried, at which point they see SETTLED.
//
// No external locks or Redis dedup are needed. The same protection applies to
// settlePaymentSession via SessionService.SettleSession which also uses RW txn.
func (ws *WebhookService) settleInvoice(ctx context.Context, invoiceID string, expectedAmount int64, gateway string) (string, error) {
	var retailerID string

	_, err := ws.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "MasterInvoices", spanner.Key{invoiceID},
			[]string{"RetailerId", "Total", "State"})
		if readErr != nil {
			return fmt.Errorf("invoice not found: %s", invoiceID)
		}

		var total int64
		var state string
		if colErr := row.Columns(&retailerID, &total, &state); colErr != nil {
			return fmt.Errorf("invoice row parse error: %w", colErr)
		}

		if state == "SETTLED" {
			return fmt.Errorf("already settled")
		}
		if state != "PENDING" {
			return fmt.Errorf("invoice %s in state %s, cannot settle", invoiceID, state)
		}
		if total != expectedAmount {
			return fmt.Errorf("amount mismatch: invoice=%d webhook=%d", total, expectedAmount)
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("MasterInvoices",
				[]string{"InvoiceId", "State"},
				[]interface{}{invoiceID, "SETTLED"},
			),
		}); err != nil {
			return err
		}

		return emitInvoiceSettledOutbox(ctx, txn, invoiceID, gateway, total, retailerID)
	})

	return retailerID, err
}

// settlePaymentSession resolves and settles the durable payment session for an invoice.
// Called after MasterInvoice is settled. Non-fatal — failures are logged but do not block webhook success.
func (ws *WebhookService) settlePaymentSession(ctx context.Context, invoiceID, gateway, providerTxnID string) {
	if ws.SessionSvc == nil {
		return
	}
	session, err := ws.SessionSvc.ResolveSessionByInvoice(ctx, invoiceID)
	if err != nil {
		slog.Warn("webhook.no_payment_session", "invoice_id", invoiceID, "err", err)
		return
	}
	if err := ws.SessionSvc.SettleSession(ctx, session.SessionID, providerTxnID); err != nil {
		slog.Error("webhook.settle_payment_session_failed", "session_id", session.SessionID, "err", err)
	} else {
		slog.Info("webhook.payment_session_settled", "session_id", session.SessionID, "gateway", gateway)
	}
}

func (ws *WebhookService) resolveGlobalPayCredentialsForOrder(ctx context.Context, orderID string) (GlobalPayCredentials, error) {
	if ws.VaultResolver != nil {
		cfg, err := ws.VaultResolver.GetDecryptedConfigByOrder(ctx, orderID, "GLOBAL_PAY")
		if err == nil {
			return ResolveGlobalPayCredentials(cfg.MerchantId, cfg.ServiceId, cfg.SecretKey)
		}
		slog.Warn("global_pay_webhook.vault_lookup_failed", "order_id", orderID, "err", err)
	}
	return ResolveGlobalPayCredentials("", "", "")
}

func parseGlobalPayWebhookRequest(r *http.Request) (*globalPayWebhookRequest, error) {
	req := &globalPayWebhookRequest{
		SessionID:         strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("session_id"), r.URL.Query().Get("sessionId"))),
		ProviderReference: strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("service_token"), r.URL.Query().Get("serviceToken"), r.URL.Query().Get("provider_reference"))),
		ProviderPaymentID: strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("payment_id"), r.URL.Query().Get("paymentId"))),
		Status:            strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("status"), r.URL.Query().Get("state"))),
	}

	if r.Method == http.MethodPost {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			return nil, fmt.Errorf("failed to read webhook body")
		}
		defer r.Body.Close()
		if len(bytes.TrimSpace(body)) > 0 {
			var raw map[string]interface{}
			if err := json.Unmarshal(body, &raw); err == nil {
				req.SessionID = firstNonEmpty(req.SessionID, globalPayLookupString(raw, "sessionId", "session_id"))
				req.ProviderReference = firstNonEmpty(req.ProviderReference, globalPayLookupString(raw, "serviceToken", "service_token", "providerReference", "provider_reference", "token"))
				req.ProviderPaymentID = firstNonEmpty(req.ProviderPaymentID, globalPayLookupString(raw, "paymentId", "payment_id", "id"))
				req.Status = firstNonEmpty(req.Status, globalPayLookupString(raw, "status", "state", "paymentStatus", "payment_status"))
			}
		}
	}

	if req.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	return req, nil
}

func applyGlobalPayIdempotencyKey(r *http.Request, req *globalPayWebhookRequest) {
	if strings.TrimSpace(r.Header.Get("Idempotency-Key")) != "" {
		return
	}
	if req == nil {
		return
	}
	key := strings.TrimSpace(firstNonEmpty(req.ProviderPaymentID, req.ProviderReference, req.SessionID))
	if key == "" {
		return
	}
	r.Header.Set("Idempotency-Key", "global-pay:"+key)
}

// lookupInvoice checks if a MasterInvoice exists and returns its Total.
func (ws *WebhookService) lookupInvoice(ctx context.Context, invoiceID string) (bool, int64, error) {
	row, err := ws.Spanner.Single().ReadRow(ctx, "MasterInvoices",
		spanner.Key{invoiceID}, []string{"Total"})
	if err != nil {
		if spanner.ErrCode(err) == 5 { // NOT_FOUND
			return false, 0, nil
		}
		return false, 0, err
	}

	var total int64
	if colErr := row.Columns(&total); colErr != nil {
		return false, 0, colErr
	}
	return true, total, nil
}

// notifyDriverPaymentSettled looks up the order's driver and pushes PAYMENT_SETTLED via WebSocket.
func (ws *WebhookService) notifyDriverPaymentSettled(orderID string, amount int64) {
	row, err := ws.Spanner.Single().ReadRow(context.Background(), "Orders", spanner.Key{orderID}, []string{"DriverId"})
	if err != nil {
		slog.Error("payment_push.read_driver_failed", "order_id", orderID, "err", err)
		return
	}
	var driverID spanner.NullString
	if err := row.Columns(&driverID); err != nil || !driverID.Valid {
		slog.Warn("payment_push.no_driver_assigned", "order_id", orderID)
		return
	}

	type settledPayload struct {
		Type    string `json:"type"`
		OrderID string `json:"order_id"`
		Amount  int64  `json:"amount"`
		Message string `json:"message"`
	}
	ws.DriverHub.PushToDriver(driverID.StringVal, settledPayload{
		Type:    "PAYMENT_SETTLED",
		OrderID: orderID,
		Amount:  amount,
		Message: "Payment received. Tap Complete to finalize delivery.",
	})
	slog.Info("payment_push.sent_payment_settled", "driver_id", driverID.StringVal, "order_id", orderID)
}

// notifyDriverPaymentFailed pushes a PAYMENT_FAILED event to the driver assigned to an order.
func (ws *WebhookService) notifyDriverPaymentFailed(orderID, reason string) {
	if ws.DriverHub == nil {
		return
	}
	row, err := ws.Spanner.Single().ReadRow(context.Background(), "Orders", spanner.Key{orderID}, []string{"DriverId"})
	if err != nil {
		return
	}
	var driverID spanner.NullString
	if err := row.Columns(&driverID); err != nil || !driverID.Valid {
		return
	}
	ws.DriverHub.PushToDriver(driverID.StringVal, map[string]interface{}{
		"type":     wsEvents.EventPaymentFailed,
		"order_id": orderID,
		"reason":   reason,
		"message":  "Payment failed for this order. Awaiting retry from retailer.",
	})
}

// notifyRetailerPaymentFailed pushes a PAYMENT_FAILED event to the retailer who owns an order.
func (ws *WebhookService) notifyRetailerPaymentFailed(orderID, gateway, reason string) {
	if ws.RetailerHub == nil {
		return
	}
	row, err := ws.Spanner.Single().ReadRow(context.Background(), "Orders", spanner.Key{orderID}, []string{"RetailerId"})
	if err != nil {
		return
	}
	var retailerID spanner.NullString
	if err := row.Columns(&retailerID); err != nil || !retailerID.Valid {
		return
	}
	ws.RetailerHub.PushToRetailer(retailerID.StringVal, map[string]interface{}{
		"type":     wsEvents.EventPaymentFailed,
		"order_id": orderID,
		"gateway":  gateway,
		"reason":   reason,
		"message":  "Payment failed — please try again or choose another method",
	})
}

// resolveOrderFromInvoice looks up the OrderId from a MasterInvoice for vault credential resolution.
func (ws *WebhookService) resolveOrderFromInvoice(ctx context.Context, invoiceID string) string {
	row, err := ws.Spanner.Single().ReadRow(ctx, "MasterInvoices", spanner.Key{invoiceID}, []string{"OrderId"})
	if err != nil {
		return ""
	}
	var orderID spanner.NullString
	if colErr := row.Columns(&orderID); colErr != nil {
		return ""
	}
	return orderID.StringVal
}

// validateGlobalPayAuth verifies the Basic Auth header against the merchant key.
// GlobalPay sends: Authorization: Basic base64("Paycom:" + merchantKey)
func validateGlobalPayAuth(authHeader, merchantKey string) bool {
	if !strings.HasPrefix(authHeader, "Basic ") {
		return false
	}

	decoded, err := base64.StdEncoding.DecodeString(authHeader[6:])
	if err != nil {
		return false
	}

	expected := "Paycom:" + merchantKey
	return secureCompare(string(decoded), expected)
}

// extractGlobalPayOrderParams extracts invoice_id and amount from GlobalPay's nested params.
// GlobalPay sends: {"account": {"order_id": "INV-123"}, "amount": 1500000} (amount in tiyins)
func extractGlobalPayOrderParams(params map[string]interface{}) (invoiceID string, amountTiyins int64, err error) {
	account, ok := params["account"].(map[string]interface{})
	if !ok {
		return "", 0, fmt.Errorf("missing account field")
	}
	invoiceID, ok = account["order_id"].(string)
	if !ok || invoiceID == "" {
		return "", 0, fmt.Errorf("missing order_id in account")
	}

	amountRaw, ok := params["amount"].(float64) // JSON numbers decode as float64
	if !ok {
		return "", 0, fmt.Errorf("missing amount field")
	}
	amountTiyins = int64(amountRaw)
	return invoiceID, amountTiyins, nil
}

// secureCompare performs constant-time string comparison to prevent timing attacks.
func secureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// writeGlobalPayResult sends a successful JSON-RPC response.

// writeGlobalPayError sends a JSON-RPC error response.

// ─── Webhook DLQ on Repeated Signature Failure (Phase 3.6) ──────────────────

// trackWebhookSigFailure increments a Redis counter for the given provider
// reference. After 3+ failures within 1 hour, it logs a DLQ-level alert. The
// anomaly is written to LedgerAnomalies for admin visibility.
//
// Nil-safe: degrades to log-only when Redis is offline.
func (ws *WebhookService) trackWebhookSigFailure(provider, providerRef, remoteAddr string) {
	redisKey := fmt.Sprintf("%s%s:%s", cache.PrefixWebhookSigFail, provider, providerRef)

	rc := cache.GetClient()
	if rc != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		count, err := rc.Incr(ctx, redisKey).Result()
		if err != nil {
			slog.Error("webhook_dlq.redis_incr_failed", "key", redisKey, "err", err)
			return
		}
		// Set TTL only on first increment
		if count == 1 {
			rc.Expire(ctx, redisKey, cache.TTLWebhookSigFail)
		}

		if count >= 3 {
			slog.Error("webhook_dlq.repeated_sig_failure", "count", count, "provider", provider, "ref", providerRef, "addr", remoteAddr)
		}
	} else {
		slog.Warn("webhook_dlq.sig_failure_redis_offline", "provider", provider, "ref", providerRef, "addr", remoteAddr)
	}
}
