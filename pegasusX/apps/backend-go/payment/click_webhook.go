package payment

import (
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type clickWebhookRequest struct {
	ClickTransID      string `json:"click_trans_id"`
	MerchantTransID   string `json:"merchant_trans_id"`
	MerchantPrepareID string `json:"merchant_prepare_id"`
	ServiceID         string `json:"service_id"`
	Amount            string `json:"amount"`
	Action            string `json:"action"`
	SignTime          string `json:"sign_time"`
	SignString        string `json:"sign_string"`
	Error             string `json:"error"`
	ErrorNote         string `json:"error_note"`
}

// HandleClickWebhook serves POST /v1/webhooks/click.
func (s *Service) HandleClickWebhook(w http.ResponseWriter, r *http.Request) {
	const endpoint = "/v1/webhooks/click"
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

	var req clickWebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body", endpoint, false, "")
		return
	}

	transactionID := strings.TrimSpace(req.ClickTransID)
	if transactionID == "" {
		writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "click_trans_id is required", endpoint, false, "")
		return
	}
	if !verifyClickSignature(req, s.clickWebhookSecret) {
		writeJSONError(w, http.StatusUnauthorized, "invalid_signature", "Invalid webhook signature", endpoint, false, "")
		return
	}

	status := "PAID"
	if strings.TrimSpace(req.Error) != "" && req.Error != "0" {
		status = "FAILED"
	}

	bodyHash := sha256Hex(body)
	webhookKey := "webhook:click:" + transactionID + ":" + status
	if replayed := s.writeWebhookReplayIfExists(w, r, endpoint, webhookKey, bodyHash); replayed {
		return
	}

	amountMinor, _ := strconv.ParseInt(strings.TrimSpace(req.Amount), 10, 64)
	orderID := strings.TrimSpace(req.MerchantTransID)

	session, found, err := s.repo.GetSessionByOrderID(r.Context(), orderID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "database_error", err.Error(), endpoint, false, "")
		return
	}
	if !found {
		resp := map[string]any{
			"error":          -5,
			"error_note":     "User does not exist",
			"click_trans_id": transactionID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	if session.AmountMinor != amountMinor {
		resp := map[string]any{
			"error":          -2,
			"error_note":     "Incorrect parameter amount",
			"click_trans_id": transactionID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	now := s.now()
	row := WebhookRecord{
		WebhookID:      s.newID("webhook"),
		Gateway:        "CLICK",
		TransactionID:  transactionID,
		OrderID:        strings.TrimSpace(req.MerchantTransID),
		SupplierID:     s.supplierID,
		Status:         status,
		AmountMinor:    amountMinor,
		Currency:       s.currency,
		ReceivedAt:     now,
		SignatureValid: true,
	}
	if err := s.persistWebhookWithOutbox(r.Context(), row, "payment.webhook.click", now); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "webhook_process_failed", err.Error(), endpoint, false, "")
		return
	}

	resp := map[string]any{
		"error":      0,
		"error_note": "accepted",
		"click_trans_id": transactionID,
	}
	respBytes, _ := json.Marshal(resp)
	s.persistIdempotencyRecord(r.Context(), webhookKey, bodyHash, http.StatusOK, respBytes, 7*24*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

func verifyClickSignature(req clickWebhookRequest, secret string) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}
	provided := strings.TrimSpace(req.SignString)
	if provided == "" {
		return false
	}
	payload := strings.TrimSpace(req.ClickTransID) +
		strings.TrimSpace(req.ServiceID) +
		secret +
		strings.TrimSpace(req.MerchantTransID) +
		strings.TrimSpace(req.Amount) +
		strings.TrimSpace(req.Action) +
		strings.TrimSpace(req.SignTime)
	sum := md5.Sum([]byte(payload))
	expected := hex.EncodeToString(sum[:])
	
	// Convert both to lowercase byte slices for safe constant-time comparison
	bProvided := []byte(strings.ToLower(provided))
	bExpected := []byte(expected)
	
	if len(bProvided) != len(bExpected) {
		return false
	}
	return subtle.ConstantTimeCompare(bProvided, bExpected) == 1
}
