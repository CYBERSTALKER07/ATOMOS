package payment

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type globalPayWebhookRequest struct {
	SessionID     string `json:"session_id,omitempty"`
	ServiceToken  string `json:"service_token,omitempty"`
	PaymentID     string `json:"payment_id,omitempty"`
	TransactionID string `json:"transaction_id,omitempty"`
	OrderID       string `json:"order_id,omitempty"`
	Status        string `json:"status,omitempty"`
	AmountMinor   int64  `json:"amount_minor,omitempty"`
	Amount        int64  `json:"amount,omitempty"`
	Currency      string `json:"currency,omitempty"`
}

// HandleGlobalPayWebhook serves POST /v1/webhooks/global-pay.
func (s *Service) HandleGlobalPayWebhook(w http.ResponseWriter, r *http.Request) {
	const endpoint = "/v1/webhooks/global-pay"
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

	if !verifyGlobalPayBasicAuth(r.Header.Get("Authorization"), s.globalPayWebhookSecret) {
		writeJSONError(w, http.StatusUnauthorized, "invalid_signature", "Invalid webhook signature", endpoint, false, "")
		return
	}

	req, err := parseGlobalPayWebhookRequest(body, r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", err.Error(), endpoint, false, "")
		return
	}

	req.Status = strings.ToUpper(strings.TrimSpace(req.Status))
	if req.TransactionID == "" || req.Status == "" {
		writeJSONError(w, http.StatusUnprocessableEntity, "invalid_request", "transaction_id and status are required", endpoint, false, "")
		return
	}

	bodyHash := sha256Hex(body)
	webhookKey := "webhook:global_pay:" + req.TransactionID + ":" + req.Status
	if replayed := s.writeWebhookReplayIfExists(w, r, endpoint, webhookKey, bodyHash); replayed {
		return
	}

	// Docs-only integration requirement: Authoritative status verification
	if req.Status == "SUCCESS" || req.Status == "SETTLED" {
		verifiedStatus, err := s.verifyGlobalPayPaymentStatus(r.Context(), req.TransactionID)
		if err != nil {
			writeJSONError(w, http.StatusBadGateway, "verification_failed", fmt.Sprintf("failed to verify payment status: %v", err), endpoint, false, "")
			return
		}
		if verifiedStatus != req.Status {
			writeJSONError(w, http.StatusConflict, "status_mismatch", "webhook status does not match authoritative status", endpoint, false, "")
			return
		}
	}

	amount := req.AmountMinor
	if amount <= 0 {
		amount = req.Amount
	}
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = s.currency
	}
	now := s.now()
	row := WebhookRecord{
		WebhookID:      s.newID("webhook"),
		Gateway:        "GLOBAL_PAY",
		TransactionID:  req.TransactionID,
		SessionID:      req.SessionID,
		OrderID:        strings.TrimSpace(req.OrderID),
		SupplierID:     s.supplierID,
		Status:         req.Status,
		AmountMinor:    amount,
		Currency:       currency,
		ReceivedAt:     now,
		SignatureValid: true,
	}
	if err := s.persistWebhookWithOutbox(r.Context(), row, "payment.webhook.global_pay", now); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "webhook_process_failed", err.Error(), endpoint, false, "")
		return
	}

	resp := map[string]string{
		"status":         "accepted",
		"gateway":        "global-pay",
		"transaction_id": row.TransactionID,
	}
	respBytes, _ := json.Marshal(resp)
	s.persistIdempotencyRecord(r.Context(), webhookKey, bodyHash, http.StatusOK, respBytes, 7*24*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

func (s *Service) verifyGlobalPayPaymentStatus(ctx context.Context, transactionID string) (string, error) {
	if s.globalPayUsername == "" || s.globalPayPassword == "" {
		return "", fmt.Errorf("globalpay credentials missing (username or password)")
	}

	env := strings.ToLower(s.globalPayEnv)
	var baseURL string
	switch env {
	case "production":
		baseURL = "https://checkout-api.globalpay.uz/checkout"
	case "staging":
		baseURL = "https://checkout-api-staging.globalpay.uz/checkout"
	default:
		baseURL = "https://checkout-api-dev.globalpay.uz/checkout"
	}

	// 1. Authenticate to get access token
	authURL := fmt.Sprintf("%s/v1/merchant/auth", baseURL)
	reqBody, _ := json.Marshal(map[string]string{
		"username": s.globalPayUsername,
		"password": s.globalPayPassword,
	})

	authReq, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	authReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	authResp, err := client.Do(authReq)
	if err != nil {
		return "", fmt.Errorf("auth request failed: %w", err)
	}
	defer authResp.Body.Close()

	if authResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(authResp.Body)
		return "", fmt.Errorf("auth failed with status %d: %s", authResp.StatusCode, string(b))
	}

	var authData struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(authResp.Body).Decode(&authData); err != nil {
		return "", fmt.Errorf("auth decode failed: %w", err)
	}
	if authData.AccessToken == "" {
		return "", fmt.Errorf("empty access token")
	}

	// 2. Get payment status
	statusURL := fmt.Sprintf("%s/v1/payment/%s", baseURL, transactionID)
	statusReq, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return "", err
	}
	statusReq.Header.Set("Authorization", "Bearer "+authData.AccessToken)

	statusResp, err := client.Do(statusReq)
	if err != nil {
		return "", fmt.Errorf("status request failed: %w", err)
	}
	defer statusResp.Body.Close()

	if statusResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(statusResp.Body)
		return "", fmt.Errorf("status query failed with status %d: %s", statusResp.StatusCode, string(b))
	}

	var statusData struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(statusResp.Body).Decode(&statusData); err != nil {
		return "", fmt.Errorf("status decode failed: %w", err)
	}

	return strings.ToUpper(strings.TrimSpace(statusData.Status)), nil
}

func parseGlobalPayWebhookRequest(body []byte, r *http.Request) (globalPayWebhookRequest, error) {
	var req globalPayWebhookRequest
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			return globalPayWebhookRequest{}, fmt.Errorf("invalid JSON payload")
		}
	}

	q := r.URL.Query()
	req.SessionID = coalesceString(strings.TrimSpace(req.SessionID), strings.TrimSpace(q.Get("session_id")))
	req.ServiceToken = coalesceString(strings.TrimSpace(req.ServiceToken), strings.TrimSpace(q.Get("service_token")), strings.TrimSpace(q.Get("serviceToken")), strings.TrimSpace(q.Get("provider_reference")))
	req.PaymentID = coalesceString(strings.TrimSpace(req.PaymentID), strings.TrimSpace(q.Get("payment_id")), strings.TrimSpace(q.Get("paymentId")))
	req.TransactionID = coalesceString(strings.TrimSpace(req.TransactionID), req.PaymentID, req.ServiceToken, strings.TrimSpace(q.Get("transaction_id")))
	req.OrderID = coalesceString(strings.TrimSpace(req.OrderID), strings.TrimSpace(q.Get("order_id")), strings.TrimSpace(q.Get("merchant_reference")))
	req.Status = coalesceString(strings.TrimSpace(req.Status), strings.TrimSpace(q.Get("status")), strings.TrimSpace(q.Get("state")))

	if req.AmountMinor <= 0 {
		if raw := strings.TrimSpace(q.Get("amount_minor")); raw != "" {
			if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
				req.AmountMinor = parsed
			}
		}
	}
	if req.Amount <= 0 {
		if raw := strings.TrimSpace(q.Get("amount")); raw != "" {
			if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
				req.Amount = parsed
			}
		}
	}
	req.Currency = coalesceString(strings.TrimSpace(req.Currency), strings.TrimSpace(q.Get("currency")))

	if req.SessionID == "" {
		return globalPayWebhookRequest{}, fmt.Errorf("session_id is required")
	}
	if req.TransactionID == "" {
		return globalPayWebhookRequest{}, fmt.Errorf("transaction_id (or payment_id/service_token) is required")
	}
	if strings.TrimSpace(req.Status) == "" {
		return globalPayWebhookRequest{}, fmt.Errorf("status is required")
	}
	return req, nil
}

func verifyGlobalPayBasicAuth(rawHeader, secret string) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}
	header := strings.TrimSpace(rawHeader)
	if len(header) < len("Basic ") || !strings.EqualFold(header[:len("Basic ")], "Basic ") {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(header[len("Basic "):]))
	if err != nil {
		return false
	}
	expected := []byte("Paycom:" + secret)
	if len(decoded) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare(decoded, expected) == 1
}
