// Package payment — Click & Payme Webhook Handlers (Phase 7: Payment Settlement)
//
// POST /v1/webhooks/click   — Server-to-server callback from Click Up
// POST /v1/webhooks/payme   — Server-to-server JSON-RPC callback from Payme
//
// These endpoints are UNAUTHENTICATED (no JWT). Security is enforced via:
//   - Click:  MD5 sign_string verification against CLICK_SECRET_KEY
//   - Payme:  Basic Auth header verification against PAYME_MERCHANT_KEY
//
// Idempotency:
//   - Click:  Keyed by click_trans_id — duplicate action=1 returns success without re-crediting.
//   - Payme:  Keyed by payme_transaction_id — PerformTransaction on an already-SETTLED invoice
//     returns the original result without double-crediting.
//
// Both handlers settle invoices inside a Spanner ReadWriteTransaction.
// Kafka INVOICE_SETTLED event fires ONLY after commit.
package payment

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"backend-go/cache"
	"backend-go/workers"
	wsEvents "backend-go/ws"

	"cloud.google.com/go/spanner"
	"github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ─── Webhook Service ─────────────────────────────────────────────────────────

// Kafka event type constants — local mirrors of kafka.Event* to avoid import cycle.
const (
	eventPaymentSettled = "PAYMENT_SETTLED"
	eventPaymentFailed  = "PAYMENT_FAILED"
	eventOrderCompleted = "ORDER_COMPLETED"
)

// WebhookService holds the Spanner + Kafka handles for webhook processing.
type WebhookService struct {
	Spanner       *spanner.Client
	Producer      *kafka.Writer
	DriverHub     DriverPusher    // Push PAYMENT_SETTLED to driver after Payme settlement
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

// InvoiceSettledEvent is emitted to Kafka after a successful settlement.
type InvoiceSettledEvent struct {
	InvoiceID  string    `json:"invoice_id"`
	Gateway    string    `json:"gateway"`
	Amount     int64     `json:"amount"`
	RetailerID string    `json:"retailer_id"`
	Timestamp  time.Time `json:"timestamp"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// CLICK WEBHOOK
// ═══════════════════════════════════════════════════════════════════════════════

// Click sends form-encoded POST with these fields on action=0 (Prepare) and
// action=1 (Complete). We only settle on action=1.
type clickWebhookRequest struct {
	ClickTransID    string `json:"click_trans_id"`
	ServiceID       string `json:"service_id"`
	MerchantTransID string `json:"merchant_trans_id"` // Our InvoiceId
	Amount          int64  `json:"amount"`
	Action          int    `json:"action"` // 0 = Prepare, 1 = Complete
	SignTime        string `json:"sign_time"`
	SignString      string `json:"sign_string"`
	Error           int    `json:"error"`
	ErrorNote       string `json:"error_note"`
}

type clickWebhookResponse struct {
	ClickTransID    string `json:"click_trans_id"`
	MerchantTransID string `json:"merchant_trans_id"`
	Error           int    `json:"error"`
	ErrorNote       string `json:"error_note"`
}

type globalPayWebhookRequest struct {
	SessionID         string `json:"session_id"`
	ProviderReference string `json:"service_token"`
	ProviderPaymentID string `json:"payment_id"`
	Status            string `json:"status"`
}

// HandleClickWebhook processes Click Up server-to-server callbacks.
func (ws *WebhookService) HandleClickWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16)) // 64KB max
	if err != nil {
		writeClickError(w, "", "", -1, "failed to read body")
		return
	}
	defer r.Body.Close()

	var req clickWebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeClickError(w, "", "", -1, "malformed JSON payload")
		return
	}

	// ── Signature Verification ──────────────────────────────────────────────
	// Try per-supplier vault credentials first, fall back to global ENV
	secretKey := ""
	if ws.VaultResolver != nil {
		// MerchantTransID is our InvoiceId → resolve supplier secret
		orderID := ws.resolveOrderFromInvoice(r.Context(), req.MerchantTransID)
		if orderID != "" {
			cfg, vErr := ws.VaultResolver.GetDecryptedConfigByOrder(r.Context(), orderID, "CLICK")
			if vErr == nil {
				secretKey = cfg.SecretKey
			} else {
				slog.Warn("click_webhook.vault_lookup_failed", "err", vErr)
			}
		}
	}
	if secretKey == "" {
		secretKey = os.Getenv("CLICK_SECRET_KEY")
	}
	if secretKey == "" {
		slog.Error("click_webhook.no_secret_key", "detail", "vault + ENV both empty")
		writeClickError(w, req.ClickTransID, req.MerchantTransID, -1, "server configuration error")
		return
	}

	expectedSign := computeClickSignature(
		req.ClickTransID, req.ServiceID, secretKey,
		req.MerchantTransID, req.Amount, req.Action, req.SignTime,
	)
	if !secureCompare(req.SignString, expectedSign) {
		slog.Error("click_webhook.signature_mismatch", "click_trans_id", req.ClickTransID)
		ws.trackWebhookSigFailure("CLICK", req.ClickTransID, r.RemoteAddr)
		writeClickError(w, req.ClickTransID, req.MerchantTransID, -1, "signature verification failed")
		return
	}

	// ── Action 0: Prepare (validate invoice exists) ─────────────────────────
	if req.Action == 0 {
		exists, invoiceTotal, err := ws.lookupInvoice(r.Context(), req.MerchantTransID)
		if err != nil {
			slog.Error("click_webhook.prepare_lookup_error", "err", err)
			writeClickError(w, req.ClickTransID, req.MerchantTransID, -3, "database error")
			return
		}
		if !exists {
			writeClickError(w, req.ClickTransID, req.MerchantTransID, -5, "invoice not found")
			return
		}
		if invoiceTotal != req.Amount {
			writeClickError(w, req.ClickTransID, req.MerchantTransID, -2, "amount mismatch")
			return
		}
		writeClickSuccess(w, req.ClickTransID, req.MerchantTransID)
		return
	}

	// ── Action 1: Complete (settle the invoice) ─────────────────────────────
	if req.Action == 1 {
		retailerID, err := ws.settleInvoice(r.Context(), req.MerchantTransID, req.Amount, "CLICK")
		if err != nil {
			if strings.Contains(err.Error(), "already settled") {
				// Idempotency: Click retried — return success without re-crediting
				slog.Info("click_webhook.idempotent_replay", "invoice", req.MerchantTransID)
				writeClickSuccess(w, req.ClickTransID, req.MerchantTransID)
				return
			}
			slog.Error("click_webhook.settlement_failed", "err", err)
			// Push PAYMENT_FAILED to retailer + driver
			orderID := ws.resolveOrderFromInvoice(r.Context(), req.MerchantTransID)
			if orderID != "" {
				workers.EventPool.Submit(func() { ws.notifyRetailerPaymentFailed(orderID, "CLICK", err.Error()) })
				workers.EventPool.Submit(func() { ws.notifyDriverPaymentFailed(orderID, "Click payment settlement failed") })
				workers.EventPool.Submit(func() { ws.emitPaymentFailedEvent(orderID, req.MerchantTransID, retailerID, "CLICK", err.Error()) })
			}
			writeClickError(w, req.ClickTransID, req.MerchantTransID, -4, err.Error())
			return
		}

		// Kafka INVOICE_SETTLED — fires ONLY after Spanner commit
		ws.emitSettledEvent(req.MerchantTransID, "CLICK", req.Amount, retailerID)

		// Settle durable payment session (non-fatal)
		ws.settlePaymentSession(r.Context(), req.MerchantTransID, "CLICK", req.ClickTransID)

		// Notify driver + emit PAYMENT_SETTLED for notification dispatcher
		orderID := ws.resolveOrderFromInvoice(r.Context(), req.MerchantTransID)
		if orderID != "" {
			ws.notifyDriverPaymentSettled(orderID, req.Amount)
			workers.EventPool.Submit(func() { ws.emitPaymentSettledEvent(orderID, req.MerchantTransID, retailerID, "CLICK", req.Amount) })
		}

		writeClickSuccess(w, req.ClickTransID, req.MerchantTransID)
		return
	}

	writeClickError(w, req.ClickTransID, req.MerchantTransID, -3, "unknown action")
}

func computeClickSignature(clickTransID, serviceID, secretKey, merchantTransID string, amount int64, action int, signTime string) string {
	raw := fmt.Sprintf("%s%s%s%s%d%d%s", clickTransID, serviceID, secretKey, merchantTransID, amount, action, signTime)
	hash := md5.Sum([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func writeClickSuccess(w http.ResponseWriter, clickTransID, merchantTransID string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clickWebhookResponse{
		ClickTransID:    clickTransID,
		MerchantTransID: merchantTransID,
		Error:           0,
		ErrorNote:       "Success",
	})
}

func writeClickError(w http.ResponseWriter, clickTransID, merchantTransID string, code int, note string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clickWebhookResponse{
		ClickTransID:    clickTransID,
		MerchantTransID: merchantTransID,
		Error:           code,
		ErrorNote:       note,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// PAYME WEBHOOK (JSON-RPC)
// ═══════════════════════════════════════════════════════════════════════════════

type paymeRPCRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
	ID     interface{}            `json:"id"`
}

type paymeRPCResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  interface{} `json:"error,omitempty"`
	ID     interface{} `json:"id"`
}

// HandlePaymeWebhook processes Payme JSON-RPC server-to-server callbacks.
// Methods: CheckPerformTransaction, CreateTransaction, PerformTransaction, CancelTransaction, CheckTransaction
func (ws *WebhookService) HandlePaymeWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// ── Basic Auth Verification ─────────────────────────────────────────────
	// Fast path: verify Authorization header against the primary merchant key
	// before any JSON parsing. This rejects obvious unauthenticated traffic
	// early and avoids unnecessary decode work.
	authHeader := r.Header.Get("Authorization")
	primaryMerchantKey := os.Getenv("PAYME_MERCHANT_KEY")

	if primaryMerchantKey != "" && ws.VaultResolver == nil && !validatePaymeAuth(authHeader, primaryMerchantKey) {
		slog.Error("payme_webhook.auth_failed", "remote_addr", r.RemoteAddr)
		ws.trackWebhookSigFailure("PAYME", r.RemoteAddr, r.RemoteAddr)
		writePaymeError(w, nil, -32504, "Insufficient privilege to perform this method")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writePaymeError(w, nil, -32700, "failed to read body")
		return
	}
	defer r.Body.Close()

	var rpcReq paymeRPCRequest
	if err := json.Unmarshal(body, &rpcReq); err != nil {
		writePaymeError(w, nil, -32700, "parse error")
		return
	}

	authenticated := primaryMerchantKey != "" && validatePaymeAuth(authHeader, primaryMerchantKey)

	if !authenticated && primaryMerchantKey == "" && ws.VaultResolver == nil {
		slog.Error("payme_webhook.no_merchant_key", "detail", "vault + ENV both empty")
		http.Error(w, "server configuration error", http.StatusInternalServerError)
		return
	}

	// Fallback path: resolve per-supplier key from vault only when primary key
	// auth fails and a vault resolver is available.
	if !authenticated && ws.VaultResolver != nil {
		invoiceID, _, _ := extractPaymeOrderParams(rpcReq.Params)
		if invoiceID != "" {
			orderID := ws.resolveOrderFromInvoice(r.Context(), invoiceID)
			if orderID != "" {
				cfg, vErr := ws.VaultResolver.GetDecryptedConfigByOrder(r.Context(), orderID, "PAYME")
				if vErr == nil && cfg != nil && cfg.SecretKey != "" {
					authenticated = validatePaymeAuth(authHeader, cfg.SecretKey)
				} else if vErr != nil {
					slog.Warn("payme_webhook.vault_lookup_failed", "err", vErr)
				}
			}
		}
	}

	if !authenticated {
		slog.Error("payme_webhook.auth_failed", "remote_addr", r.RemoteAddr)
		ws.trackWebhookSigFailure("PAYME", r.RemoteAddr, r.RemoteAddr)
		writePaymeError(w, nil, -32504, "Insufficient privilege to perform this method")
		return
	}

	ctx := r.Context()

	switch rpcReq.Method {
	case "CheckPerformTransaction":
		ws.paymeCheckPerform(ctx, w, rpcReq)
	case "CreateTransaction":
		ws.paymeCreateTransaction(ctx, w, rpcReq)
	case "PerformTransaction":
		ws.paymePerformTransaction(ctx, w, rpcReq)
	case "CancelTransaction":
		ws.paymeCancelTransaction(ctx, w, rpcReq)
	case "CheckTransaction":
		ws.paymeCheckTransaction(ctx, w, rpcReq)
	default:
		writePaymeError(w, rpcReq.ID, -32601, "method not found")
	}
}

// paymeCheckPerform validates that the invoice exists and the amount matches.
func (ws *WebhookService) paymeCheckPerform(ctx context.Context, w http.ResponseWriter, req paymeRPCRequest) {
	invoiceID, amountTiyins, err := extractPaymeOrderParams(req.Params)
	if err != nil {
		writePaymeError(w, req.ID, -31050, err.Error())
		return
	}
	amount := amountTiyins / 100

	exists, invoiceTotal, err := ws.lookupInvoice(ctx, invoiceID)
	if err != nil {
		writePaymeError(w, req.ID, -31001, "database error")
		return
	}
	if !exists {
		writePaymeError(w, req.ID, -31050, "invoice not found")
		return
	}
	if invoiceTotal != amount {
		writePaymeError(w, req.ID, -31001, fmt.Sprintf("amount mismatch: expected %d got %d", invoiceTotal, amount))
		return
	}

	writePaymeResult(w, req.ID, map[string]interface{}{"allow": true})
}

// paymeCreateTransaction registers the Payme transaction ID against our invoice.
// This is a prerequisite for PerformTransaction.
func (ws *WebhookService) paymeCreateTransaction(ctx context.Context, w http.ResponseWriter, req paymeRPCRequest) {
	paymeID, _ := req.Params["id"].(string)
	invoiceID, amountTiyins, err := extractPaymeOrderParams(req.Params)
	if err != nil {
		writePaymeError(w, req.ID, -31050, err.Error())
		return
	}
	amount := amountTiyins / 100

	exists, invoiceTotal, err := ws.lookupInvoice(ctx, invoiceID)
	if err != nil {
		writePaymeError(w, req.ID, -31001, "database error")
		return
	}
	if !exists {
		writePaymeError(w, req.ID, -31050, "invoice not found")
		return
	}
	if invoiceTotal != amount {
		writePaymeError(w, req.ID, -31001, "amount mismatch")
		return
	}

	// Store the Payme transaction ID on the invoice for idempotency tracking
	_, err = ws.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "MasterInvoices", spanner.Key{invoiceID},
			[]string{"State", "PaymeTransactionId"})
		if readErr != nil {
			return readErr
		}

		var state string
		var existingPaymeID spanner.NullString
		if colErr := row.Columns(&state, &existingPaymeID); colErr != nil {
			return colErr
		}

		// Idempotency: if this exact Payme transaction already created, return success
		if existingPaymeID.Valid && existingPaymeID.StringVal == paymeID {
			return nil
		}

		if state != "PENDING" {
			return fmt.Errorf("invoice state is %s, expected PENDING", state)
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("MasterInvoices",
				[]string{"InvoiceId", "PaymeTransactionId"},
				[]interface{}{invoiceID, spanner.NullString{StringVal: paymeID, Valid: true}},
			),
		})
	})
	if err != nil {
		slog.Error("payme_webhook.create_transaction_failed", "err", err)
		writePaymeError(w, req.ID, -31008, err.Error())
		return
	}

	now := time.Now().UnixMilli()
	writePaymeResult(w, req.ID, map[string]interface{}{
		"create_time": now,
		"transaction": paymeID,
		"state":       1, // 1 = Created
	})
}

// paymePerformTransaction settles the invoice. Idempotent — double calls return success.
func (ws *WebhookService) paymePerformTransaction(ctx context.Context, w http.ResponseWriter, req paymeRPCRequest) {
	paymeID, _ := req.Params["id"].(string)
	if paymeID == "" {
		writePaymeError(w, req.ID, -31050, "transaction id required")
		return
	}

	var retailerID string
	var amount int64
	var invoiceID string
	var orderID string
	var alreadySettled bool

	_, err := ws.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Find the invoice by Payme transaction ID
		stmt := spanner.Statement{
			SQL: `SELECT InvoiceId, RetailerId, Total, State, OrderId FROM MasterInvoices
			      WHERE PaymeTransactionId = @paymeId`,
			Params: map[string]interface{}{"paymeId": paymeID},
		}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()

		row, iterErr := iter.Next()
		if iterErr == iterator.Done {
			return fmt.Errorf("payme transaction not found: %s", paymeID)
		}
		if iterErr != nil {
			return iterErr
		}

		var state string
		var nullOrderID spanner.NullString
		if colErr := row.Columns(&invoiceID, &retailerID, &amount, &state, &nullOrderID); colErr != nil {
			return colErr
		}
		orderID = nullOrderID.StringVal

		if state == "SETTLED" {
			alreadySettled = true
			return nil // Idempotent — no double credit
		}
		if state != "PENDING" {
			return fmt.Errorf("invoice %s in state %s, cannot settle", invoiceID, state)
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("MasterInvoices",
				[]string{"InvoiceId", "State"},
				[]interface{}{invoiceID, "SETTLED"},
			),
		})
	})

	if err != nil {
		slog.Error("payme_webhook.perform_transaction_failed", "err", err)
		if orderID != "" {
			workers.EventPool.Submit(func() { ws.emitPaymentFailedEvent(orderID, invoiceID, retailerID, "PAYME", err.Error()) })
		}
		writePaymeError(w, req.ID, -31008, err.Error())
		return
	}

	if !alreadySettled {
		ws.emitSettledEvent(invoiceID, "PAYME", amount, retailerID)
		slog.Info("payme_webhook.invoice_settled", "invoice_id", invoiceID, "amount", amount, "retailer_id", retailerID)

		// Settle durable payment session (non-fatal)
		ws.settlePaymentSession(ctx, invoiceID, "PAYME", paymeID)

		// Push PAYMENT_SETTLED to driver via WebSocket so they can tap "Completed"
		if ws.DriverHub != nil && orderID != "" {
			workers.EventPool.Submit(func() { ws.notifyDriverPaymentSettled(orderID, amount) })
		}

		// Emit PAYMENT_SETTLED for notification dispatcher (inbox + Telegram)
		if orderID != "" {
			workers.EventPool.Submit(func() { ws.emitPaymentSettledEvent(orderID, invoiceID, retailerID, "PAYME", amount) })
		}
	} else {
		slog.Info("payme_webhook.idempotent_replay", "invoice_id", invoiceID)
	}

	now := time.Now().UnixMilli()
	writePaymeResult(w, req.ID, map[string]interface{}{
		"perform_time": now,
		"transaction":  paymeID,
		"state":        2, // 2 = Completed
	})
}

// paymeCancelTransaction cancels a pending or settled invoice.
func (ws *WebhookService) paymeCancelTransaction(ctx context.Context, w http.ResponseWriter, req paymeRPCRequest) {
	paymeID, _ := req.Params["id"].(string)
	reason, _ := req.Params["reason"].(float64)

	if paymeID == "" {
		writePaymeError(w, req.ID, -31050, "transaction id required")
		return
	}

	_, err := ws.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `SELECT InvoiceId, State FROM MasterInvoices
			      WHERE PaymeTransactionId = @paymeId`,
			Params: map[string]interface{}{"paymeId": paymeID},
		}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()

		row, iterErr := iter.Next()
		if iterErr == iterator.Done {
			return fmt.Errorf("payme transaction not found: %s", paymeID)
		}
		if iterErr != nil {
			return iterErr
		}

		var invoiceID, state string
		if colErr := row.Columns(&invoiceID, &state); colErr != nil {
			return colErr
		}

		if state == "CANCELLED" {
			return nil // Already cancelled — idempotent
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("MasterInvoices",
				[]string{"InvoiceId", "State"},
				[]interface{}{invoiceID, "CANCELLED"},
			),
		})
	})

	if err != nil {
		slog.Error("payme_webhook.cancel_transaction_failed", "err", err)
		writePaymeError(w, req.ID, -31008, err.Error())
		return
	}

	slog.Info("payme_webhook.transaction_cancelled", "payme_id", paymeID, "reason", reason)

	// Push PAYMENT_FAILED to retailer + driver for cancelled transaction
	workers.EventPool.Submit(func() { ws.notifyPaymeCancelled(ctx, paymeID) })

	now := time.Now().UnixMilli()
	writePaymeResult(w, req.ID, map[string]interface{}{
		"cancel_time": now,
		"transaction": paymeID,
		"state":       -1, // -1 = Cancelled
	})
}

// notifyPaymeCancelled resolves the order from a Payme transaction and pushes PAYMENT_FAILED.
func (ws *WebhookService) notifyPaymeCancelled(ctx context.Context, paymeID string) {
	stmt := spanner.Statement{
		SQL:    `SELECT InvoiceId FROM MasterInvoices WHERE PaymeTransactionId = @paymeId`,
		Params: map[string]interface{}{"paymeId": paymeID},
	}
	iter := ws.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, iterErr := iter.Next()
	if iterErr != nil {
		return
	}
	var invoiceID string
	if row.Columns(&invoiceID) != nil {
		return
	}
	orderID := ws.resolveOrderFromInvoice(ctx, invoiceID)
	if orderID == "" {
		return
	}
	ws.notifyRetailerPaymentFailed(orderID, "PAYME", "Transaction cancelled by Payme")
	ws.notifyDriverPaymentFailed(orderID, "Payme transaction cancelled")
}

// paymeCheckTransaction returns the current state of a Payme transaction.
func (ws *WebhookService) paymeCheckTransaction(ctx context.Context, w http.ResponseWriter, req paymeRPCRequest) {
	paymeID, _ := req.Params["id"].(string)
	if paymeID == "" {
		writePaymeError(w, req.ID, -31050, "transaction id required")
		return
	}

	stmt := spanner.Statement{
		SQL: `SELECT InvoiceId, State FROM MasterInvoices
		      WHERE PaymeTransactionId = @paymeId`,
		Params: map[string]interface{}{"paymeId": paymeID},
	}
	iter := ws.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, iterErr := iter.Next()
	if iterErr == iterator.Done {
		writePaymeError(w, req.ID, -31050, "transaction not found")
		return
	}
	if iterErr != nil {
		writePaymeError(w, req.ID, -31001, "database error")
		return
	}

	var invoiceID, state string
	if colErr := row.Columns(&invoiceID, &state); colErr != nil {
		writePaymeError(w, req.ID, -31001, "row parse error")
		return
	}

	paymeState := 1 // Created
	switch state {
	case "SETTLED":
		paymeState = 2
	case "CANCELLED":
		paymeState = -1
	}

	writePaymeResult(w, req.ID, map[string]interface{}{
		"create_time": time.Now().UnixMilli(),
		"transaction": paymeID,
		"state":       paymeState,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// GLOBAL PAY WEBHOOK
// ═══════════════════════════════════════════════════════════════════════════════

// HandleGlobalPayWebhook verifies a Global Pay callback by re-querying the
// provider status endpoint before settling any invoice.
func (ws *WebhookService) HandleGlobalPayWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
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
		retailerID, settleErr := ws.settleInvoice(r.Context(), session.InvoiceID, session.LockedAmount, "GLOBAL_PAY")
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

		ws.emitSettledEvent(session.InvoiceID, "GLOBAL_PAY", session.LockedAmount, retailerID)
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
// Concurrent webhook deliveries (e.g. Click + GlobalPay retry, or duplicate
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

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("MasterInvoices",
				[]string{"InvoiceId", "State"},
				[]interface{}{invoiceID, "SETTLED"},
			),
		})
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

// emitSettledEvent fires the INVOICE_SETTLED Kafka event asynchronously.
func (ws *WebhookService) emitSettledEvent(invoiceID, gateway string, amount int64, retailerID string) {
	event := InvoiceSettledEvent{
		InvoiceID:  invoiceID,
		Gateway:    gateway,
		Amount:     amount,
		RetailerID: retailerID,
		Timestamp:  time.Now().UTC(),
	}

	workers.EventPool.Submit(func() {
		data, _ := json.Marshal(event)
		err := ws.Producer.WriteMessages(context.Background(), kafka.Message{
			Key:   []byte(invoiceID),
			Value: data,
		})
		if err != nil {
			slog.Error("kafka.emit_invoice_settled_failed", "invoice_id", invoiceID, "err", err)
		} else {
			slog.Info("kafka.invoice_settled_emitted", "invoice_id", invoiceID, "gateway", gateway, "amount", amount)
		}
	})
}

// emitPaymentSettledEvent fires a PAYMENT_SETTLED Kafka event for the notification dispatcher.
func (ws *WebhookService) emitPaymentSettledEvent(orderID, invoiceID, retailerID, gateway string, amount int64) {
	driverID := ws.resolveDriverFromOrder(orderID)
	event := map[string]interface{}{
		"order_id":    orderID,
		"invoice_id":  invoiceID,
		"retailer_id": retailerID,
		"driver_id":   driverID,
		"gateway":     gateway,
		"amount":      amount,
		"timestamp":   time.Now().UTC(),
	}
	data, _ := json.Marshal(event)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ws.Producer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(eventPaymentSettled),
		Value: data,
	}); err != nil {
		slog.Error("kafka.emit_payment_settled_failed", "order_id", orderID, "err", err)
	}
}

// emitPaymentFailedEvent fires a PAYMENT_FAILED Kafka event for the notification dispatcher.
func (ws *WebhookService) emitPaymentFailedEvent(orderID, invoiceID, retailerID, gateway, reason string) {
	event := map[string]interface{}{
		"order_id":    orderID,
		"invoice_id":  invoiceID,
		"retailer_id": retailerID,
		"gateway":     gateway,
		"reason":      reason,
		"timestamp":   time.Now().UTC(),
	}
	data, _ := json.Marshal(event)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ws.Producer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(eventPaymentFailed),
		Value: data,
	}); err != nil {
		slog.Error("kafka.emit_payment_failed_failed", "order_id", orderID, "err", err)
	}
}

// resolveDriverFromOrder looks up the DriverId from an order for event emissions.
func (ws *WebhookService) resolveDriverFromOrder(orderID string) string {
	row, err := ws.Spanner.Single().ReadRow(context.Background(), "Orders", spanner.Key{orderID}, []string{"DriverId"})
	if err != nil {
		return ""
	}
	var driverID spanner.NullString
	if colErr := row.Columns(&driverID); colErr != nil || !driverID.Valid {
		return ""
	}
	return driverID.StringVal
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

// validatePaymeAuth verifies the Basic Auth header against the merchant key.
// Payme sends: Authorization: Basic base64("Paycom:" + merchantKey)
func validatePaymeAuth(authHeader, merchantKey string) bool {
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

// extractPaymeOrderParams extracts invoice_id and amount from Payme's nested params.
// Payme sends: {"account": {"order_id": "INV-123"}, "amount": 1500000} (amount in tiyins)
func extractPaymeOrderParams(params map[string]interface{}) (invoiceID string, amountTiyins int64, err error) {
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

// writePaymeResult sends a successful JSON-RPC response.
func writePaymeResult(w http.ResponseWriter, id interface{}, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(paymeRPCResponse{
		Result: result,
		ID:     id,
	})
}

// writePaymeError sends a JSON-RPC error response.
func writePaymeError(w http.ResponseWriter, id interface{}, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(paymeRPCResponse{
		Error: map[string]interface{}{
			"code":    code,
			"message": map[string]string{"en": message},
		},
		ID: id,
	})
}

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
