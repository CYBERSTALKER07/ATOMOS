package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
)

const maxUzumWebhookBodyBytes = 256 * 1024

type uzumWebhookParams struct {
	OrderID   string `json:"order_id"`
	OrderIDCC string `json:"orderId"`
	Account   string `json:"account"`
}

type uzumWebhookRequest struct {
	ServiceID     json.RawMessage   `json:"serviceId"`
	Timestamp     int64             `json:"timestamp"`
	TransID       string            `json:"transId"`
	TransIDSnake  string            `json:"trans_id"`
	TransactionID string            `json:"transaction_id"`
	ID            string            `json:"id"`
	OrderID       string            `json:"orderId"`
	OrderIDSnake  string            `json:"order_id"`
	Amount        int64             `json:"amount"`
	AmountMinor   int64             `json:"amount_minor"`
	AmountMinorCC int64             `json:"amountMinor"`
	Currency      string            `json:"currency"`
	Status        string            `json:"status"`
	TargetType    string            `json:"targetType"`
	TargetTypeSn  string            `json:"target_type"`
	Params        uzumWebhookParams `json:"params"`
}

// HandleUzumWebhook serves POST /v1/webhooks/uzum.
// Verifies the HMAC-SHA256 webhook signature (fail-closed when unconfigured),
// checks amount and currency consistency when a payment session exists,
// enforces idempotency replay/conflict semantics, and persists the webhook +
// finance outbox event.
func (s *Service) HandleUzumWebhook(w http.ResponseWriter, r *http.Request) {
	const endpoint = "/v1/webhooks/uzum"
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", endpoint, false, "")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxUzumWebhookBodyBytes+1))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "Unable to read request body", endpoint, false, "")
		return
	}
	defer r.Body.Close()
	if len(body) > maxUzumWebhookBodyBytes {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "Webhook body exceeds 256KiB limit", endpoint, false, "")
		return
	}

	sigHeader := coalesceString(
		strings.TrimSpace(r.Header.Get("X-Uzum-Signature")),
		strings.TrimSpace(r.Header.Get("X-Signature")),
	)
	if !s.verifyUzumSignature(body, sigHeader) {
		writeJSONError(w, http.StatusUnauthorized, "invalid_signature", "Invalid webhook signature", endpoint, false, "")
		return
	}

	var req uzumWebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body", endpoint, false, "")
		return
	}

	transID := strings.TrimSpace(coalesceString(req.TransID, req.TransIDSnake, req.TransactionID, req.ID))
	orderID := strings.TrimSpace(coalesceString(req.OrderID, req.OrderIDSnake, req.Params.OrderID, req.Params.OrderIDCC, req.Params.Account))
	amountMinor := req.Amount
	if amountMinor == 0 {
		amountMinor = req.AmountMinor
	}
	if amountMinor == 0 {
		amountMinor = req.AmountMinorCC
	}
	normalizedStatus, statusOK := normalizeUzumWebhookStatus(req.Status)
	if transID == "" || orderID == "" || amountMinor <= 0 || !statusOK {
		writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "transId, orderId, positive amount, and valid status are required", endpoint, false, "")
		return
	}

	currency := strings.ToUpper(strings.TrimSpace(coalesceString(req.Currency, s.currency, "UZS")))

	var (
		sessionID  string
		retailerID string
		supplierID string
	)
	if s.repo != nil {
		session, found, err := s.repo.GetSessionByOrderID(r.Context(), orderID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database_error", err.Error(), endpoint, false, "")
			return
		}
		if found {
			if session.AmountMinor > 0 && session.AmountMinor != amountMinor {
				writeJSONError(w, http.StatusUnprocessableEntity, "amount_mismatch", "webhook amount must match payment session amount", endpoint, false, "")
				return
			}
			if err := fxrates.AssertSameCurrency(session.Currency, currency); err != nil {
				writeJSONError(w, http.StatusUnprocessableEntity, "currency_mismatch", "webhook currency must match payment session currency", endpoint, false, "")
				return
			}
			sessionID = session.SessionID
			retailerID = session.RetailerID
			supplierID = session.SupplierID
		} else if err := s.assertSessionCurrency(r.Context(), "", orderID, currency); err != nil {
			writeJSONError(w, http.StatusUnprocessableEntity, "currency_mismatch", "webhook currency must match payment session currency", endpoint, false, "")
			return
		}
	}
	if supplierID == "" {
		supplierID = s.resolveWebhookSupplierID(r.Context(), orderID)
	}

	webhookKey := "webhook:uzum:" + transID + ":" + normalizedStatus
	bodyHash := sha256Hex(body)
	if replayed := s.writeWebhookReplayIfExists(w, r, endpoint, webhookKey, bodyHash); replayed {
		return
	}

	now := s.now()
	row := WebhookRecord{
		WebhookID:      s.newID("webhook"),
		Gateway:        "UZUM",
		TransactionID:  transID,
		SessionID:      sessionID,
		OrderID:        orderID,
		SupplierID:     supplierID,
		RetailerID:     retailerID,
		Status:         normalizedStatus,
		AmountMinor:    amountMinor,
		Currency:       currency,
		ReceivedAt:     now,
		SignatureValid: true,
	}
	if err := s.persistWebhookWithOutbox(r.Context(), row, "payment.webhook.uzum", now); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "webhook_process_failed", err.Error(), endpoint, false, "")
		return
	}

	resp := map[string]string{
		"status":         "accepted",
		"gateway":        "UZUM",
		"transaction_id": transID,
		"order_id":       orderID,
		"webhook_status": normalizedStatus,
	}
	respBytes, _ := json.Marshal(resp)
	s.persistIdempotencyRecord(r.Context(), webhookKey, bodyHash, http.StatusOK, respBytes, 7*24*time.Hour)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

func (s *Service) resolveUzumWebhookSecrets() string {
	if s != nil && strings.TrimSpace(s.uzumWebhookSecret) != "" {
		return s.uzumWebhookSecret
	}
	return normalizeWebhookSecret(os.Getenv("UZUM_WEBHOOK_SECRET"))
}

func (s *Service) verifyUzumSignature(body []byte, header string) bool {
	if header == "" {
		return false
	}
	secrets := strings.Split(s.resolveUzumWebhookSecrets(), ",")
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret == "" {
			continue
		}
		if verifyUzumHMACSignature(body, header, secret) {
			return true
		}
	}
	return false
}

func verifyUzumHMACSignature(body []byte, sigHeader, secret string) bool {
	if strings.TrimSpace(secret) == "" {
		return false
	}
	cand := strings.TrimSpace(sigHeader)
	if len(cand) >= 7 && strings.EqualFold(cand[:7], "sha256=") {
		cand = strings.TrimSpace(cand[7:])
	}
	if cand == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)

	if decodedHex, err := hex.DecodeString(cand); err == nil && len(decodedHex) == sha256.Size {
		return hmac.Equal(decodedHex, expected)
	}
	if decodedB64, err := base64.StdEncoding.DecodeString(cand); err == nil && len(decodedB64) == sha256.Size {
		return hmac.Equal(decodedB64, expected)
	}
	if decodedRawB64, err := base64.RawStdEncoding.DecodeString(cand); err == nil && len(decodedRawB64) == sha256.Size {
		return hmac.Equal(decodedRawB64, expected)
	}
	return false
}

func normalizeUzumWebhookStatus(raw string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "CONFIRMED", "PAID", "COMPLETED", "SUCCEEDED", "SUCCESS":
		return "PAID", true
	case "FAILED", "REJECTED", "ERROR", "DECLINED":
		return "FAILED", true
	case "REVERSED", "CANCELLED", "CANCELED", "REFUNDED":
		return "REVERSED", true
	case "CREATED", "PENDING", "PROCESSING":
		return "PENDING", true
	default:
		return "", false
	}
}
