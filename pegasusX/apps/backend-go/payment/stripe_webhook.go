package payment

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
	stripe "github.com/stripe/stripe-go/v76"
	stripewebhook "github.com/stripe/stripe-go/v76/webhook"
)

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

	event, err := stripewebhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), s.stripeWebhookSecret)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid_signature", "Invalid webhook signature", endpoint, false, "")
		return
	}

	eventID := strings.TrimSpace(event.ID)
	eventType := strings.TrimSpace(string(event.Type))
	if eventID == "" || eventType == "" || len(event.Data.Raw) == 0 {
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

	switch eventType {
	case "payment_intent.succeeded", "payment_intent.payment_failed":
		var intent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &intent); err != nil {
			writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "invalid payment_intent payload", endpoint, false, "")
			return
		}
		intentID := strings.TrimSpace(intent.ID)
		if intentID == "" {
			writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "payment_intent.id is required", endpoint, false, "")
			return
		}
		row.TransactionID = intentID
		if intent.Metadata != nil {
			row.SessionID = strings.TrimSpace(intent.Metadata["session_id"])
			row.OrderID = strings.TrimSpace(intent.Metadata["order_id"])
			row.RetailerID = strings.TrimSpace(intent.Metadata["retailer_id"])
		}
		if row.OrderID == "" && intent.Metadata != nil {
			row.OrderID = strings.TrimSpace(intent.Metadata["merchant_reference"])
		}
		if eventType == "payment_intent.succeeded" {
			row.Status = "PAID"
		} else {
			row.Status = "FAILED"
		}
		row.AmountMinor = intent.Amount
		row.Currency = strings.ToUpper(strings.TrimSpace(string(intent.Currency)))
	case "charge.refunded":
		var charge stripe.Charge
		if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
			writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "invalid charge payload", endpoint, false, "")
			return
		}
		chargeID := strings.TrimSpace(charge.ID)
		if chargeID == "" {
			writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "charge.id is required", endpoint, false, "")
			return
		}
		row.TransactionID = chargeID
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
		row.Currency = strings.ToUpper(strings.TrimSpace(string(charge.Currency)))
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "ignored",
			"gateway":    "stripe",
			"event_type": eventType,
		})
		return
	}

	if row.Currency == "" {
		row.Currency = s.currency
	}

	bodyHash := sha256Hex(body)
	webhookKey := "webhook:stripe:" + eventID
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
