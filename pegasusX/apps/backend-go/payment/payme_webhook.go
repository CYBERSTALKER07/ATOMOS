package payment

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type paymeWebhookRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	ID      any    `json:"id"`
	Params  struct {
		ID          string `json:"id"`
		Transaction string `json:"transaction"`
		OrderID     string `json:"order_id"`
		Amount      int64  `json:"amount"`
		State       int    `json:"state"`
		Reason      int    `json:"reason"`
	} `json:"params"`
}

// HandlePaymeWebhook serves POST /v1/webhooks/payme.
func (s *Service) HandlePaymeWebhook(w http.ResponseWriter, r *http.Request) {
	const endpoint = "/v1/webhooks/payme"
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

	if !verifyGlobalPayBasicAuth(r.Header.Get("Authorization"), s.paymeWebhookSecret) {
		writeJSONError(w, http.StatusUnauthorized, "invalid_signature", "Invalid webhook signature", endpoint, false, "")
		return
	}

	var req paymeWebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body", endpoint, false, "")
		return
	}

	transactionID := strings.TrimSpace(req.Params.ID)
	if transactionID == "" {
		transactionID = strings.TrimSpace(req.Params.Transaction)
	}
	if transactionID == "" {
		writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "params.id is required", endpoint, false, "")
		return
	}

	status := "PENDING"
	switch req.Params.State {
	case 2:
		status = "PAID"
	case -1, -2:
		status = "FAILED"
	}

	bodyHash := sha256Hex(body)
	webhookKey := "webhook:payme:" + transactionID + ":" + status
	if replayed := s.writeWebhookReplayIfExists(w, r, endpoint, webhookKey, bodyHash); replayed {
		return
	}

	now := s.now()
	row := WebhookRecord{
		WebhookID:      s.newID("webhook"),
		Gateway:        "PAYME",
		TransactionID:  transactionID,
		OrderID:        strings.TrimSpace(req.Params.OrderID),
		SupplierID:     s.supplierID,
		Status:         status,
		AmountMinor:    req.Params.Amount,
		Currency:       s.currency,
		ReceivedAt:     now,
		SignatureValid: true,
	}
	if err := s.persistWebhookWithOutbox(r.Context(), row, "payment.webhook.payme", now); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "webhook_process_failed", err.Error(), endpoint, false, "")
		return
	}

	resp := map[string]any{
		"result": map[string]any{
			"status":         "accepted",
			"transaction_id": transactionID,
		},
		"id": req.ID,
	}
	respBytes, _ := json.Marshal(resp)
	s.persistIdempotencyRecord(r.Context(), webhookKey, bodyHash, http.StatusOK, respBytes, 7*24*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}
