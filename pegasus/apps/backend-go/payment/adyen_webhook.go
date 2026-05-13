package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"

	"backend-go/idempotency"

	"github.com/adyen/adyen-go-api-library/v21/src/hmacvalidator"
	adyenwebhook "github.com/adyen/adyen-go-api-library/v21/src/webhook"
)

// HandleAdyenWebhook validates HMAC signatures for all notification items and
// settles payment sessions/invoices for successful AUTHORISATION and CAPTURE events.
func (ws *WebhookService) HandleAdyenWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if ws.SessionSvc == nil {
		http.Error(w, "payment session service unavailable", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	payload := strings.TrimSpace(string(body))
	if payload == "" {
		http.Error(w, "empty payload", http.StatusBadRequest)
		return
	}

	notification, err := adyenwebhook.HandleRequest(payload)
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	items := notification.GetNotificationItems()
	if len(items) == 0 {
		http.Error(w, "notificationItems required", http.StatusBadRequest)
		return
	}

	for _, item := range items {
		if item == nil {
			continue
		}
		hmacKey, keyErr := ws.resolveAdyenHMACKey(r.Context(), item)
		if keyErr != nil {
			slog.Error("adyen_webhook.hmac_key_missing", "merchant_account", item.MerchantAccountCode, "order_id", item.MerchantReference, "err", keyErr)
			http.Error(w, "webhook not configured", http.StatusServiceUnavailable)
			return
		}
		if !hmacvalidator.ValidateHmac(*item, hmacKey) {
			providerRef := strings.TrimSpace(firstNonEmpty(item.PspReference, item.MerchantReference))
			if providerRef == "" {
				providerRef = "unknown"
			}
			ws.trackWebhookSigFailure("adyen", providerRef, r.RemoteAddr)
			http.Error(w, "invalid webhook signature", http.StatusUnauthorized)
			return
		}
	}

	applyAdyenIdempotencyKey(r, items)
	idempotency.Guard(func(w http.ResponseWriter, r *http.Request) {
		ws.handleAdyenWebhookParsed(w, r, items)
	})(w, r)
}

func (ws *WebhookService) handleAdyenWebhookParsed(w http.ResponseWriter, r *http.Request, items []*adyenwebhook.NotificationRequestItem) {
	processed := 0
	for _, item := range items {
		if item == nil {
			continue
		}
		handled, err := ws.processAdyenNotificationItem(r.Context(), item)
		if err != nil {
			slog.Error("adyen_webhook.process_failed", "event_code", item.EventCode, "order_id", item.MerchantReference, "psp_reference", item.PspReference, "err", err)
			http.Error(w, "webhook processing failed", http.StatusBadGateway)
			return
		}
		if handled {
			processed++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "accepted",
		"processed_items": processed,
	})
}

func (ws *WebhookService) processAdyenNotificationItem(ctx context.Context, item *adyenwebhook.NotificationRequestItem) (bool, error) {
	orderID := strings.TrimSpace(item.MerchantReference)
	if orderID == "" {
		slog.Warn("adyen_webhook.order_id_missing", "psp_reference", item.PspReference, "event_code", item.EventCode)
		return false, nil
	}

	session, err := ws.SessionSvc.GetSessionByOrder(ctx, orderID)
	if err != nil {
		slog.Warn("adyen_webhook.session_not_found", "order_id", orderID, "err", err)
		return false, nil
	}
	provider, providerErr := NewProviderClient(session.Gateway)
	if providerErr != nil || provider.GatewayName() != "ADYEN" {
		slog.Warn("adyen_webhook.gateway_mismatch", "order_id", orderID, "session_gateway", session.Gateway, "err", providerErr)
		return false, nil
	}

	eventCode := strings.ToUpper(strings.TrimSpace(item.EventCode))
	success := strings.EqualFold(strings.TrimSpace(item.Success), "true")
	providerTxnID := strings.TrimSpace(item.PspReference)

	if isAdyenSettlementEvent(eventCode) && success {
		if session.InvoiceID == "" {
			return true, fmt.Errorf("payment session %s missing invoice binding", session.SessionID)
		}
		_, settleErr := ws.settleInvoice(ctx, session.InvoiceID, session.LockedAmount, "ADYEN")
		if settleErr != nil && !strings.Contains(strings.ToLower(settleErr.Error()), "already settled") {
			return true, settleErr
		}

		ws.settlePaymentSession(ctx, session.InvoiceID, "ADYEN", providerTxnID)
		if session.OrderID != "" {
			ws.notifyDriverPaymentSettled(session.OrderID, session.LockedAmount)
		}
		return true, nil
	}

	if !success || isAdyenFailureEvent(eventCode) {
		failureCode := firstNonEmpty("ADYEN_"+eventCode, "ADYEN_FAILED")
		failureMessage := firstNonEmpty(strings.TrimSpace(item.Reason), eventCode, "Payment failed")
		if err := ws.SessionSvc.FailSession(ctx, session.SessionID, failureCode, failureMessage); err != nil {
			return true, err
		}
		if session.OrderID != "" {
			ws.notifyDriverPaymentFailed(session.OrderID, failureMessage)
			ws.notifyRetailerPaymentFailed(session.OrderID, "ADYEN", failureMessage)
		}
		return true, nil
	}

	return false, nil
}

func applyAdyenIdempotencyKey(r *http.Request, items []*adyenwebhook.NotificationRequestItem) {
	if strings.TrimSpace(r.Header.Get("Idempotency-Key")) != "" {
		return
	}
	if len(items) == 0 {
		return
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		eventCode := strings.ToUpper(strings.TrimSpace(item.EventCode))
		pspReference := strings.TrimSpace(item.PspReference)
		merchantReference := strings.TrimSpace(item.MerchantReference)
		if eventCode == "" && pspReference == "" && merchantReference == "" {
			continue
		}
		parts = append(parts, eventCode+":"+pspReference+":"+merchantReference)
	}
	if len(parts) == 0 {
		return
	}
	sort.Strings(parts)
	key := strings.Join(parts, ",")
	if len(key) > 180 {
		key = key[:180]
	}
	r.Header.Set("Idempotency-Key", "adyen:"+key)
}

func (ws *WebhookService) resolveAdyenHMACKey(ctx context.Context, item *adyenwebhook.NotificationRequestItem) (string, error) {
	if item == nil {
		return "", fmt.Errorf("notification item is nil")
	}
	if ws.VaultResolver != nil {
		orderID := strings.TrimSpace(item.MerchantReference)
		if orderID != "" {
			cfg, err := ws.VaultResolver.GetDecryptedConfigByOrder(ctx, orderID, "ADYEN")
			if err == nil {
				if key := strings.TrimSpace(cfg.ServiceId); looksLikeAdyenHMACKey(key) {
					return key, nil
				}
			}
		}
	}

	for _, envKey := range adyenHMACEnvCandidates(item.MerchantAccountCode) {
		if value := strings.TrimSpace(os.Getenv(envKey)); value != "" {
			return value, nil
		}
	}

	return "", fmt.Errorf("adyen hmac key not configured")
}

func adyenHMACEnvCandidates(merchantAccount string) []string {
	candidates := make([]string, 0, 3)
	normalizedMerchant := normalizeAdyenMerchantEnvToken(merchantAccount)
	if normalizedMerchant != "" {
		candidates = append(candidates, "ADYEN_HMAC_KEY_"+normalizedMerchant)
	}
	candidates = append(candidates, "ADYEN_HMAC_KEY", "ADYEN_WEBHOOK_HMAC_KEY")
	return candidates
}

func normalizeAdyenMerchantEnvToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r - 32)
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}
	normalized := strings.Trim(builder.String(), "_")
	normalized = strings.ReplaceAll(normalized, "__", "_")
	return normalized
}

func looksLikeAdyenHMACKey(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
			return false
		}
	}
	return true
}

func isAdyenSettlementEvent(eventCode string) bool {
	switch eventCode {
	case adyenwebhook.EventCodeAuthorisation, adyenwebhook.EventCodeCapture:
		return true
	default:
		return false
	}
}

func isAdyenFailureEvent(eventCode string) bool {
	switch eventCode {
	case adyenwebhook.EventCodeCaptureFailed,
		adyenwebhook.EventCodeCancellation,
		adyenwebhook.EventCodeCancelOrRefund,
		adyenwebhook.EventCodeExpire,
		"REFUSED":
		return true
	default:
		return false
	}
}
