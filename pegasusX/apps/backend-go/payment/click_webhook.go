package payment

import (
	"encoding/json"
	"io"
	"net/http"
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

// HandleClickWebhook is the Click SHOP API Prepare (action=0) / Complete
// (action=1) callback. Signatures follow docs.click.uz SHOP API (complete
// includes merchant_prepare_id). Amount is so'm at the edge → int64 minor.
//
// UNWIRED: webhookroutes does not mount POST /v1/webhooks/click. Tests call
// this handler directly. Live card checkout for CLICK stays catalog honesty 501.
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

	req, err := decodeClickShopBody(r.Header.Get("Content-Type"), body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body", endpoint, false, "")
		return
	}

	transactionID := strings.TrimSpace(req.ClickTransID)
	if transactionID == "" {
		writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "click_trans_id is required", endpoint, false, "")
		return
	}

	var signatureValid bool
	secrets := strings.Split(s.clickWebhookSecret, ",")
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret == "" {
			continue
		}
		if verifyClickSignature(req, secret) {
			signatureValid = true
			break
		}
	}
	if !signatureValid {
		writeJSONError(w, http.StatusUnauthorized, "invalid_signature", "Invalid webhook signature", endpoint, false, "")
		return
	}

	action := clickNormalizedAction(req.Action)
	if action != clickActionPrepare && action != clickActionComplete {
		s.writeClickShop(w, req, "", clickErrAction, "Action not found")
		return
	}

	orderID := strings.TrimSpace(req.MerchantTransID)
	if s.repo == nil {
		s.writeClickShop(w, req, "", clickErrUserMissing, "User does not exist")
		return
	}
	session, found, err := s.repo.GetSessionByOrderID(r.Context(), orderID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "database_error", err.Error(), endpoint, false, "")
		return
	}
	if !found {
		s.writeClickShop(w, req, "", clickErrUserMissing, "User does not exist")
		return
	}

	amountMinor, err := clickSomToMinor(req.Amount)
	if err != nil {
		s.writeClickShop(w, req, "", clickErrAmount, "Incorrect parameter amount")
		return
	}
	if session.AmountMinor != amountMinor {
		s.writeClickShop(w, req, "", clickErrAmount, "Incorrect parameter amount")
		return
	}

	prepareID := clickPrepareID(session)
	if action == clickActionComplete {
		gotPrepare := strings.TrimSpace(req.MerchantPrepareID)
		if gotPrepare == "" {
			s.writeClickShop(w, req, "", clickErrRequest, "Error in request from click")
			return
		}
		if gotPrepare != prepareID {
			s.writeClickShop(w, req, "", clickErrTxMissing, "Transaction does not exist")
			return
		}
		if strings.EqualFold(session.Status, "PAID") {
			s.writeClickShop(w, req, prepareID, clickErrAlreadyPaid, "Already paid")
			return
		}
	}

	status := "PREPARED"
	if action == clickActionComplete {
		status = "PAID"
		if strings.TrimSpace(req.Error) != "" && req.Error != "0" {
			status = "FAILED"
		}
	}

	bodyHash := sha256Hex(body)
	webhookKey := "webhook:click:" + transactionID + ":" + status
	if replayed := s.writeWebhookReplayIfExists(w, r, endpoint, webhookKey, bodyHash); replayed {
		return
	}

	now := s.now()
	row := WebhookRecord{
		WebhookID:      s.newID("webhook"),
		Gateway:        "CLICK",
		TransactionID:  transactionID,
		OrderID:        orderID,
		SessionID:      prepareID,
		SupplierID:     s.resolveWebhookSupplierID(r.Context(), orderID),
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

	errorCode := clickErrOK
	note := "Success"
	if status == "FAILED" {
		errorCode = clickErrCancelled
		note = "Transaction cancelled"
	}
	resp := s.clickShopResponse(req, prepareID, errorCode, note)
	respBytes, _ := json.Marshal(resp)
	s.persistIdempotencyRecord(r.Context(), webhookKey, bodyHash, http.StatusOK, respBytes, 7*24*time.Hour)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

func (s *Service) writeClickShop(w http.ResponseWriter, req clickWebhookRequest, prepareID string, code int, note string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(s.clickShopResponse(req, prepareID, code, note))
}

func (s *Service) clickShopResponse(req clickWebhookRequest, prepareID string, code int, note string) map[string]any {
	resp := map[string]any{
		"click_trans_id":    jsonValueIntOrString(req.ClickTransID),
		"merchant_trans_id": req.MerchantTransID,
		"error":             code,
		"error_note":        note,
	}
	if prepareID != "" {
		resp["merchant_prepare_id"] = prepareID
		if clickNormalizedAction(req.Action) == clickActionComplete {
			resp["merchant_confirm_id"] = prepareID
		}
	}
	return resp
}

func jsonValueIntOrString(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if n, err := json.Number(s).Int64(); err == nil && json.Number(s).String() == s {
		return n
	}
	return s
}
