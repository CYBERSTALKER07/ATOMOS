// Package payment — Global Pay Canonical Reconciliation Path
//
// This file provides the single shared path for resolving Global Pay payment
// sessions. Webhooks, background sweepers, and manual reconciliation endpoints
// all call ReconcileSession to apply the provider's status to local state.
//
// The path: resolve credentials → verify with provider → settle, fail, or expire
// the session → update invoice, order projection, and notify connected clients.
package payment

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"backend-go/cache"
	"backend-go/ws"

	"cloud.google.com/go/spanner"
	"github.com/segmentio/kafka-go"
)

// RetailerPusher abstracts the retailer WebSocket hub for pushing payment events.
type RetailerPusher interface {
	PushToRetailer(retailerID string, payload interface{}) bool
}

// GlobalPayReconciler provides the canonical path for resolving Global Pay
// payment sessions — used by webhooks, background sweepers, and manual
// reconciliation endpoints.
type GlobalPayReconciler struct {
	Spanner       *spanner.Client
	SessionSvc    *SessionService
	VaultResolver VaultResolver
	Producer      *kafka.Writer
	DriverHub     DriverPusher
	RetailerHub   RetailerPusher
}

// GlobalPayReconcileResult captures the outcome of a session reconciliation attempt.
type GlobalPayReconcileResult struct {
	SessionID  string `json:"session_id"`
	OrderID    string `json:"order_id"`
	Status     string `json:"status"` // SETTLED | FAILED | EXPIRED | PENDING | ALREADY_TERMINAL
	ProviderID string `json:"provider_id,omitempty"`
	Message    string `json:"message,omitempty"`
}

// ReconcileSession is the canonical Global Pay status-application path.
// It verifies payment status with the provider, then either settles, fails,
// or expires the session and updates all connected state.
//
// providerPaymentIDHint is optional — webhooks may supply it from the callback
// payload. Background sweepers pass "".
func (r *GlobalPayReconciler) ReconcileSession(ctx context.Context, session *PaymentSession, providerPaymentIDHint string) (*GlobalPayReconcileResult, error) {
	if session.Gateway != "GLOBAL_PAY" {
		return nil, fmt.Errorf("session %s is gateway %s, not GLOBAL_PAY", session.SessionID, session.Gateway)
	}

	// Already terminal — idempotent skip
	if session.Status == SessionSettled || session.Status == SessionCancelled || session.Status == SessionExpired {
		return &GlobalPayReconcileResult{
			SessionID: session.SessionID,
			OrderID:   session.OrderID,
			Status:    "ALREADY_TERMINAL",
			Message:   fmt.Sprintf("session already %s", session.Status),
		}, nil
	}

	if session.ProviderReference == "" {
		return nil, fmt.Errorf("session %s has no provider reference for status verification", session.SessionID)
	}

	// Resolve per-supplier credentials (vault-backed with ENV fallback)
	creds, err := r.resolveCredentials(ctx, session.OrderID)
	if err != nil {
		return nil, fmt.Errorf("credential resolution failed for session %s: %w", session.SessionID, err)
	}

	// Query provider status
	status, err := VerifyGlobalPayPayment(ctx, creds, session.ProviderReference, providerPaymentIDHint)
	if err != nil {
		return nil, fmt.Errorf("provider verification failed for session %s: %w", session.SessionID, err)
	}

	// Path 1: Provider confirms payment
	if status.Paid {
		return r.applySettlement(ctx, session, status)
	}

	// Path 2: Provider declares failure
	if status.Failed() {
		return r.applyFailure(ctx, session, status)
	}

	// Path 3: Session past expiry and provider still not settled
	if session.ExpiresAt != nil && time.Now().UTC().After(*session.ExpiresAt) {
		return r.applyExpiry(ctx, session)
	}

	// Still pending — no action
	return &GlobalPayReconcileResult{
		SessionID: session.SessionID,
		OrderID:   session.OrderID,
		Status:    "PENDING",
		Message:   fmt.Sprintf("provider status: %s", status.RawStatus),
	}, nil
}

// applySettlement settles the invoice, session, and connected state when
// the provider confirms payment.
func (r *GlobalPayReconciler) applySettlement(ctx context.Context, session *PaymentSession, status *GlobalPayPaymentStatus) (*GlobalPayReconcileResult, error) {
	// Settle the master invoice
	if session.InvoiceID != "" {
		_, settleErr := r.settleInvoice(ctx, session.InvoiceID, session.LockedAmount)
		if settleErr != nil {
			if strings.Contains(settleErr.Error(), "already settled") {
				log.Printf("[GP_RECONCILE] Invoice %s already settled — idempotent", session.InvoiceID)
			} else {
				return nil, fmt.Errorf("invoice settlement failed for session %s: %w", session.SessionID, settleErr)
			}
		}
	}

	// Settle durable payment session
	if err := r.SessionSvc.SettleSession(ctx, session.SessionID, status.ProviderPaymentID); err != nil {
		log.Printf("[GP_RECONCILE] Failed to settle session %s: %v", session.SessionID, err)
	}

	// Notify driver
	if r.DriverHub != nil && session.OrderID != "" {
		go r.notifyDriverPaymentSettled(session.OrderID, session.LockedAmount)
	}

	// Notify retailer
	if r.RetailerHub != nil && session.RetailerID != "" {
		r.RetailerHub.PushToRetailer(session.RetailerID, map[string]interface{}{
			"type":       ws.EventPaymentSettled,
			"order_id":   session.OrderID,
			"session_id": session.SessionID,
			"amount":     session.LockedAmount,
			"gateway":    "GLOBAL_PAY",
			"message":    "Payment confirmed",
		})
	}

	log.Printf("[GP_RECONCILE] Session %s SETTLED (provider_payment=%s)", session.SessionID, status.ProviderPaymentID)

	return &GlobalPayReconcileResult{
		SessionID:  session.SessionID,
		OrderID:    session.OrderID,
		Status:     "SETTLED",
		ProviderID: status.ProviderPaymentID,
		Message:    "payment confirmed by provider",
	}, nil
}

// applyFailure marks the session failed when the provider declares failure.
func (r *GlobalPayReconciler) applyFailure(ctx context.Context, session *PaymentSession, status *GlobalPayPaymentStatus) (*GlobalPayReconcileResult, error) {
	errorCode := firstNonEmpty(status.FailureCode, "GLOBAL_PAY_FAILED")
	errorMessage := firstNonEmpty(status.FailureMessage, status.RawStatus, "payment failed at provider")

	if err := r.SessionSvc.FailSession(ctx, session.SessionID, errorCode, errorMessage); err != nil {
		log.Printf("[GP_RECONCILE] Failed to mark session %s as failed: %v", session.SessionID, err)
	}

	// Notify retailer of payment failure
	if r.RetailerHub != nil && session.RetailerID != "" {
		r.RetailerHub.PushToRetailer(session.RetailerID, map[string]interface{}{
			"type":       ws.EventPaymentFailed,
			"order_id":   session.OrderID,
			"session_id": session.SessionID,
			"gateway":    "GLOBAL_PAY",
			"message":    errorMessage,
		})
	}

	log.Printf("[GP_RECONCILE] Session %s FAILED: %s — %s", session.SessionID, errorCode, errorMessage)

	return &GlobalPayReconcileResult{
		SessionID: session.SessionID,
		OrderID:   session.OrderID,
		Status:    "FAILED",
		Message:   errorMessage,
	}, nil
}

// applyExpiry marks the session expired when the checkout window has passed
// and the provider still has not confirmed payment.
func (r *GlobalPayReconciler) applyExpiry(ctx context.Context, session *PaymentSession) (*GlobalPayReconcileResult, error) {
	if err := r.SessionSvc.ExpireSession(ctx, session.SessionID); err != nil {
		log.Printf("[GP_RECONCILE] Failed to mark session %s as expired: %v", session.SessionID, err)
	}

	// Notify retailer of expiry
	if r.RetailerHub != nil && session.RetailerID != "" {
		r.RetailerHub.PushToRetailer(session.RetailerID, map[string]interface{}{
			"type":       ws.EventPaymentExpired,
			"order_id":   session.OrderID,
			"session_id": session.SessionID,
			"gateway":    "GLOBAL_PAY",
			"message":    "Payment session expired",
		})
	}

	log.Printf("[GP_RECONCILE] Session %s EXPIRED (deadline: %v)", session.SessionID, session.ExpiresAt)

	return &GlobalPayReconcileResult{
		SessionID: session.SessionID,
		OrderID:   session.OrderID,
		Status:    "EXPIRED",
		Message:   "checkout window expired without provider confirmation",
	}, nil
}

// ─── Internal Helpers ────────────────────────────────────────────────────────

func (r *GlobalPayReconciler) resolveCredentials(ctx context.Context, orderID string) (GlobalPayCredentials, error) {
	if r.VaultResolver != nil && orderID != "" {
		cfg, err := r.VaultResolver.GetDecryptedConfigByOrder(ctx, orderID, "GLOBAL_PAY")
		if err == nil {
			return ResolveGlobalPayCredentials(cfg.MerchantId, cfg.ServiceId, cfg.SecretKey)
		}
		log.Printf("[GP_RECONCILE] Vault lookup failed for order %s, trying ENV: %v", orderID, err)
	}
	return ResolveGlobalPayCredentials("", "", "")
}

func (r *GlobalPayReconciler) settleInvoice(ctx context.Context, invoiceID string, expectedAmount int64) (string, error) {
	var retailerID string

	_, err := r.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
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
			return fmt.Errorf("amount mismatch: invoice=%d expected=%d", total, expectedAmount)
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("MasterInvoices",
				[]string{"InvoiceId", "State"},
				[]interface{}{invoiceID, "SETTLED"},
			),
		}); err != nil {
			return err
		}

		return emitInvoiceSettledOutbox(ctx, txn, invoiceID, "GLOBAL_PAY", total, retailerID)
	})
	if err == nil && retailerID != "" {
		cache.Invalidate(ctx, cache.PrefixActiveOrders+retailerID)
	}

	return retailerID, err
}

func (r *GlobalPayReconciler) notifyDriverPaymentSettled(orderID string, amount int64) {
	row, err := r.Spanner.Single().ReadRow(context.Background(), "Orders", spanner.Key{orderID}, []string{"DriverId"})
	if err != nil {
		log.Printf("[GP_RECONCILE] Failed to read driver for order %s: %v", orderID, err)
		return
	}
	var driverID spanner.NullString
	if err := row.Columns(&driverID); err != nil || !driverID.Valid {
		return
	}

	type settledPayload struct {
		Type    string `json:"type"`
		OrderID string `json:"order_id"`
		Amount  int64  `json:"amount"`
		Message string `json:"message"`
	}
	r.DriverHub.PushToDriver(driverID.StringVal, settledPayload{
		Type:    "PAYMENT_SETTLED",
		OrderID: orderID,
		Amount:  amount,
		Message: "Payment confirmed by Global Pay",
	})
}
