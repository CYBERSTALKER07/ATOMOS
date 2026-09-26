package payment

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// HandlePaymeWebhook is the Payme Merchant API JSON-RPC endpoint
// (CheckPerform / Create / Perform / Cancel / Check / GetStatement / SetFiscalData).
//
// UNWIRED: webhookroutes does not mount POST /v1/webhooks/payme. Tests call
// this handler directly. Live card checkout for PAYME stays catalog honesty 501.
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

	var signatureValid bool
	authHeader := r.Header.Get("Authorization")
	secrets := strings.Split(s.paymeWebhookSecret, ",")
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret == "" {
			continue
		}
		if verifyGlobalPayBasicAuth(authHeader, secret) {
			signatureValid = true
			break
		}
	}
	if !signatureValid {
		writeJSONError(w, http.StatusUnauthorized, "invalid_signature", "Invalid webhook signature", endpoint, false, "")
		return
	}

	var req paymeRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writePaymeRPC(w, nil, nil, paymeError(paymeErrParse, "parse error", ""))
		return
	}
	params, rpcErr := parsePaymeMerchantParams(req.Params)
	if rpcErr != nil {
		s.writePaymeRPC(w, req.ID, nil, rpcErr)
		return
	}

	switch strings.TrimSpace(req.Method) {
	case paymeMethodCheckPerform:
		s.paymeCheckPerform(w, r, req.ID, params)
	case paymeMethodCreate:
		s.paymeCreate(w, r, endpoint, body, req.ID, params)
	case paymeMethodPerform:
		s.paymePerform(w, r, endpoint, body, req.ID, params)
	case paymeMethodCancel:
		s.paymeCancel(w, r, endpoint, body, req.ID, params)
	case paymeMethodCheck:
		s.paymeCheck(w, req.ID, params)
	case paymeMethodStatement:
		s.paymeStatement(w, req.ID, params)
	case paymeMethodSetFiscal:
		s.paymeSetFiscal(w, r, endpoint, body, req.ID, params)
	default:
		s.writePaymeRPC(w, req.ID, nil, paymeError(paymeErrMethod, "method not found", req.Method))
	}
}

func (s *Service) writePaymeRPC(w http.ResponseWriter, id any, result any, rpcErr *paymeRPCError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
	}
	if rpcErr != nil {
		resp["error"] = rpcErr
	} else {
		resp["result"] = result
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Service) paymeLookupOrder(r *http.Request, orderID string) (SessionRecord, *paymeRPCError) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return SessionRecord{}, paymeError(paymeErrAccount, "order_id is required", "account.order_id")
	}
	if s.repo == nil {
		return SessionRecord{}, paymeError(paymeErrAccount, "order not found", "account")
	}
	session, found, err := s.repo.GetSessionByOrderID(r.Context(), orderID)
	if err != nil {
		return SessionRecord{}, paymeError(paymeErrAccount, "order lookup failed", "account")
	}
	if !found {
		return SessionRecord{}, paymeError(paymeErrAccount, "order not found", "account")
	}
	return session, nil
}

func (s *Service) paymeCheckPerform(w http.ResponseWriter, r *http.Request, id any, params paymeMerchantParams) {
	session, rpcErr := s.paymeLookupOrder(r, params.OrderID)
	if rpcErr != nil {
		s.writePaymeRPC(w, id, nil, rpcErr)
		return
	}
	if params.Amount != session.AmountMinor {
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrInvalidAmount, "incorrect amount", "amount"))
		return
	}
	if existing, ok := s.ensurePaymeTx().snapshotByOrder(params.OrderID); ok && existing.State == paymeStatePerformed {
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrOrderState, "order already paid", "account"))
		return
	}
	if strings.EqualFold(session.Status, "PAID") {
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrOrderState, "order already paid", "account"))
		return
	}
	s.writePaymeRPC(w, id, map[string]any{"allow": true}, nil)
}

func (s *Service) paymeCreate(w http.ResponseWriter, r *http.Request, endpoint string, body []byte, id any, params paymeMerchantParams) {
	if strings.TrimSpace(params.PaymeID) == "" {
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrInvalidParams, "params.id is required", "id"))
		return
	}
	session, rpcErr := s.paymeLookupOrder(r, params.OrderID)
	if rpcErr != nil {
		s.writePaymeRPC(w, id, nil, rpcErr)
		return
	}
	if params.Amount != session.AmountMinor {
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrInvalidAmount, "incorrect amount", "amount"))
		return
	}
	store := s.ensurePaymeTx()
	if existing, ok := store.snapshot(params.PaymeID); ok {
		s.writePaymeRPC(w, id, paymeCreateResult(existing), nil)
		return
	}
	if existing, ok := store.snapshotByOrder(params.OrderID); ok {
		if existing.PaymeID != params.PaymeID && existing.State == paymeStateCreated {
			s.writePaymeRPC(w, id, nil, paymeError(paymeErrOrderState, "transaction already exists for order", "account"))
			return
		}
		if existing.State == paymeStatePerformed {
			s.writePaymeRPC(w, id, nil, paymeError(paymeErrCannotPerform, "order already paid", "account"))
			return
		}
	}
	merchantTx := strings.TrimSpace(session.SessionID)
	if merchantTx == "" {
		merchantTx = params.OrderID
	}
	now := s.paymeNowMilli()
	tx := paymeMerchantTx{
		PaymeID:     params.PaymeID,
		OrderID:     params.OrderID,
		AmountMinor: params.Amount,
		State:       paymeStateCreated,
		CreateTime:  now,
		MerchantTx:  merchantTx,
	}
	if params.Time > 0 {
		tx.CreateTime = params.Time
	}
	if err := s.persistPaymeMerchantWebhook(r, endpoint, body, tx); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "webhook_process_failed", err.Error(), endpoint, false, "")
		return
	}
	store.put(tx)
	s.writePaymeRPC(w, id, paymeCreateResult(tx), nil)
}

func (s *Service) paymePerform(w http.ResponseWriter, r *http.Request, endpoint string, body []byte, id any, params paymeMerchantParams) {
	if strings.TrimSpace(params.PaymeID) == "" {
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrInvalidParams, "params.id is required", "id"))
		return
	}
	store := s.ensurePaymeTx()
	tx, ok := store.snapshot(params.PaymeID)
	if !ok {
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrNotFound, "transaction not found", "id"))
		return
	}
	if tx.State == paymeStatePerformed {
		s.writePaymeRPC(w, id, paymePerformResult(tx), nil)
		return
	}
	if tx.State != paymeStateCreated {
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrCannotPerform, "cannot perform transaction", "state"))
		return
	}
	tx.State = paymeStatePerformed
	tx.PerformTime = s.paymeNowMilli()
	if err := s.persistPaymeMerchantWebhook(r, endpoint, body, tx); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "webhook_process_failed", err.Error(), endpoint, false, "")
		return
	}
	store.put(tx)
	s.writePaymeRPC(w, id, paymePerformResult(tx), nil)
}

func (s *Service) paymeCancel(w http.ResponseWriter, r *http.Request, endpoint string, body []byte, id any, params paymeMerchantParams) {
	if strings.TrimSpace(params.PaymeID) == "" {
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrInvalidParams, "params.id is required", "id"))
		return
	}
	store := s.ensurePaymeTx()
	tx, ok := store.snapshot(params.PaymeID)
	if !ok {
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrNotFound, "transaction not found", "id"))
		return
	}
	if tx.State == paymeStateCancelled || tx.State == paymeStateReverted {
		s.writePaymeRPC(w, id, paymeCancelResult(tx), nil)
		return
	}
	now := s.paymeNowMilli()
	switch tx.State {
	case paymeStateCreated:
		tx.State = paymeStateCancelled
	case paymeStatePerformed:
		tx.State = paymeStateReverted
	default:
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrCannotCancel, "cannot cancel transaction", "state"))
		return
	}
	tx.CancelTime = now
	if params.HasReason {
		reason := params.Reason
		tx.Reason = &reason
	}
	if err := s.persistPaymeMerchantWebhook(r, endpoint, body, tx); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "webhook_process_failed", err.Error(), endpoint, false, "")
		return
	}
	store.put(tx)
	s.writePaymeRPC(w, id, paymeCancelResult(tx), nil)
}

func (s *Service) paymeCheck(w http.ResponseWriter, id any, params paymeMerchantParams) {
	if strings.TrimSpace(params.PaymeID) == "" {
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrInvalidParams, "params.id is required", "id"))
		return
	}
	tx, ok := s.ensurePaymeTx().snapshot(params.PaymeID)
	if !ok {
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrNotFound, "transaction not found", "id"))
		return
	}
	s.writePaymeRPC(w, id, paymeCheckResult(tx), nil)
}

func (s *Service) paymeStatement(w http.ResponseWriter, id any, params paymeMerchantParams) {
	items := s.ensurePaymeTx().statement(params.From, params.To)
	rows := make([]map[string]any, 0, len(items))
	for _, tx := range items {
		rows = append(rows, paymeStatementItem(tx))
	}
	s.writePaymeRPC(w, id, map[string]any{"transactions": rows}, nil)
}

func (s *Service) paymeSetFiscal(w http.ResponseWriter, r *http.Request, endpoint string, body []byte, id any, params paymeMerchantParams) {
	if strings.TrimSpace(params.PaymeID) == "" {
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrInvalidParams, "params.id is required", "id"))
		return
	}
	if strings.TrimSpace(string(params.FiscalData)) == "" || strings.TrimSpace(string(params.FiscalData)) == "null" {
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrInvalidParams, "fiscal_data is required", "fiscal_data"))
		return
	}
	tx, ok := s.ensurePaymeTx().snapshot(params.PaymeID)
	if !ok {
		s.writePaymeRPC(w, id, nil, paymeError(paymeErrFiscalNotFound, "receipt not found", "id"))
		return
	}
	status := "FISCAL_PERFORM"
	if strings.EqualFold(strings.TrimSpace(params.Type), "CANCEL") {
		status = "FISCAL_CANCEL"
	}
	now := s.now()
	row := WebhookRecord{
		WebhookID:      s.newID("webhook"),
		Gateway:        "PAYME",
		TransactionID:  tx.PaymeID,
		OrderID:        tx.OrderID,
		SessionID:      tx.MerchantTx,
		SupplierID:     s.resolveWebhookSupplierID(r.Context(), tx.OrderID),
		Status:         status,
		AmountMinor:    tx.AmountMinor,
		Currency:       s.currency,
		ReceivedAt:     now,
		SignatureValid: true,
	}
	if err := s.persistWebhookWithOutbox(r.Context(), row, "payment.webhook.payme", now); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "webhook_process_failed", err.Error(), endpoint, false, "")
		return
	}
	s.persistIdempotencyRecord(r.Context(), "webhook:payme:"+tx.PaymeID+":"+status, sha256Hex(body), http.StatusOK, []byte(`{"result":{"success":true}}`), 7*24*time.Hour)
	s.writePaymeRPC(w, id, map[string]any{"success": true}, nil)
}

func (s *Service) persistPaymeMerchantWebhook(r *http.Request, endpoint string, body []byte, tx paymeMerchantTx) error {
	status := paymeStatusFromState(tx.State)
	bodyHash := sha256Hex(body)
	webhookKey := "webhook:payme:" + tx.PaymeID + ":" + status
	now := s.now()
	row := WebhookRecord{
		WebhookID:      s.newID("webhook"),
		Gateway:        "PAYME",
		TransactionID:  tx.PaymeID,
		OrderID:        tx.OrderID,
		SessionID:      tx.MerchantTx,
		SupplierID:     s.resolveWebhookSupplierID(r.Context(), tx.OrderID),
		Status:         status,
		AmountMinor:    tx.AmountMinor,
		Currency:       s.currency,
		ReceivedAt:     now,
		SignatureValid: true,
	}
	if err := s.persistWebhookWithOutbox(r.Context(), row, "payment.webhook.payme", now); err != nil {
		return err
	}
	resp := map[string]any{"result": map[string]any{"transaction": tx.MerchantTx, "state": tx.State}, "id": tx.PaymeID}
	respBytes, _ := json.Marshal(resp)
	s.persistIdempotencyRecord(r.Context(), webhookKey, bodyHash, http.StatusOK, respBytes, 7*24*time.Hour)
	return nil
}
