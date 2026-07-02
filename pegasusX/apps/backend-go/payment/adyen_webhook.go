package payment

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	adyenhmac "github.com/adyen/adyen-go-api-library/v21/src/hmacvalidator"
	adyenwebhook "github.com/adyen/adyen-go-api-library/v21/src/webhook"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

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

	adyenWebhook, err := adyenwebhook.HandleRequest(string(body))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json_payload", "Invalid JSON payload", endpoint, false, "")
		return
	}

	if len(*adyenWebhook.NotificationItems) == 0 {
		writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "notificationItems is required", endpoint, false, "")
		return
	}

	secrets := strings.Split(s.adyenWebhookSecret, ",")
	for _, itemWrapper := range *adyenWebhook.NotificationItems {
		item := itemWrapper.NotificationRequestItem
		var signatureValid bool
		for _, secret := range secrets {
			secret = strings.TrimSpace(secret)
			if secret == "" {
				continue
			}
			if adyenhmac.ValidateHmac(item, secret) {
				signatureValid = true
				break
			}
		}
		if !signatureValid {
			writeJSONError(w, http.StatusUnauthorized, "invalid_signature", "Invalid webhook signature", endpoint, false, "")
			return
		}
	}

	processed := 0
	now := s.now()
	for _, itemWrapper := range *adyenWebhook.NotificationItems {
		item := itemWrapper.NotificationRequestItem

		eventCode := strings.ToUpper(strings.TrimSpace(item.EventCode))
		success := strings.EqualFold(strings.TrimSpace(item.Success), "true")
		var status string
		switch eventCode {
		case "REFUND", "REFUND_FAILED", "REFUNDED_REVERSED":
			if success {
				status = "REFUNDED"
			} else {
				status = "FAILED"
			}
		case "CANCELLATION", "CANCEL_OR_REFUND", "VOID_PENDING_REFUND", "CANCELLED":
			if success {
				status = "CANCELLED"
			} else {
				status = "FAILED"
			}
		case "CAPTURE", "CAPTURE_FAILED":
			if success {
				status = "PAID"
			} else {
				status = "FAILED"
			}
		default:
			if success {
				status = "PAID"
			} else {
				status = "FAILED"
			}
		}

		transactionID := strings.TrimSpace(item.PspReference)
		webhookKey := "webhook:adyen:" + transactionID + ":" + eventCode

		bodyHash := sha256Hex([]byte(adyenhmac.GetDataToSign(item)))
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

		var sessionID string
		if item.AdditionalData != nil {
			if val, ok := (*item.AdditionalData)["session_id"].(string); ok {
				sessionID = strings.TrimSpace(val)
			}
		}

		// Map specific Adyen events to PaymentAttempt metadata transitions
		var executionAction string
		switch eventCode {
		case "CAPTURE", "CAPTURE_FAILED":
			executionAction = string(ExecutionActionCheckoutCapture)
		case "CANCELLATION":
			executionAction = "CANCELLATION"
		case "REFUND", "REFUND_FAILED":
			executionAction = "REFUND"
		}

		if executionAction != "" {
			attempt := PaymentAttemptRecord{
				AttemptID:         s.newID("attempt"),
				SessionID:         sessionID,
				Gateway:           "ADYEN",
				ExecutionAction:   executionAction,
				ExecutionMode:     "WEBHOOK",
				ProviderReference: transactionID,
				Status:            status,
				CreatedAt:         now,
				UpdatedAt:         now,
			}
			// Additive metadata transition via SaveAttempt
			if err := s.repo.SaveAttempt(r.Context(), attempt, nil); err != nil {
				s.log.Warn("failed to save adyen webhook attempt", "transaction_id", transactionID, "event", eventCode, "err", err)
			}
		}

		row := WebhookRecord{
			WebhookID:      s.newID("webhook"),
			Gateway:        "ADYEN",
			TransactionID:  transactionID,
			SessionID:      sessionID,
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

