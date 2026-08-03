package order

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Global Pay receipt env (optional secondary payment-provider receipt).
// Not required for ADR-009 hard-gate — PEGASUS platform receipts cover that.
// When configured, ProviderFromEnv can wrap PEGASUS + GLOBAL_PAY.
const (
	envGPReceiptEnabled  = "FISCAL_GLOBAL_PAY_RECEIPT_ENABLED"
	envGPReceiptBaseURL  = "FISCAL_GLOBAL_PAY_RECEIPT_BASE_URL"
	envGPReceiptPath     = "FISCAL_GLOBAL_PAY_RECEIPT_PATH"
	envGPReceiptAPIKey   = "FISCAL_GLOBAL_PAY_RECEIPT_API_KEY"
	envGPReceiptTimeout  = "FISCAL_GLOBAL_PAY_RECEIPT_TIMEOUT_MS"
)

// GlobalPayReceiptProvider calls Global Pay's receipt API when credentials exist.
// Until Global Pay sends API docs/credentials, leave disabled — CreateReceipt
// returns a clear deferred error so misconfiguration is never silent success.
type GlobalPayReceiptProvider struct {
	BaseURL    string
	Path       string
	APIKey     string
	HTTPClient *http.Client
}

// NewGlobalPayReceiptProviderFromEnv builds the provider when explicitly enabled.
// Returns (nil, nil) when disabled (normal). Returns error only if enabled but incomplete.
func NewGlobalPayReceiptProviderFromEnv() (*GlobalPayReceiptProvider, error) {
	if !envTruthy(os.Getenv(envGPReceiptEnabled)) {
		return nil, nil
	}
	base := strings.TrimRight(strings.TrimSpace(os.Getenv(envGPReceiptBaseURL)), "/")
	key := strings.TrimSpace(os.Getenv(envGPReceiptAPIKey))
	if base == "" {
		return nil, fmt.Errorf("%s required when %s=true", envGPReceiptBaseURL, envGPReceiptEnabled)
	}
	if key == "" {
		return nil, fmt.Errorf("%s required when %s=true", envGPReceiptAPIKey, envGPReceiptEnabled)
	}
	path := strings.TrimSpace(os.Getenv(envGPReceiptPath))
	if path == "" {
		path = "/v1/receipts"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	timeout := FiscalOFDTimeout
	if ms := strings.TrimSpace(os.Getenv(envGPReceiptTimeout)); ms != "" {
		var n int
		if _, err := fmt.Sscanf(ms, "%d", &n); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Millisecond
		}
	}
	return &GlobalPayReceiptProvider{
		BaseURL: base,
		Path:    path,
		APIKey:  key,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (p *GlobalPayReceiptProvider) CreateReceipt(ctx context.Context, req FiscalCreateRequest) (FiscalCreateResult, error) {
	if p == nil {
		return FiscalCreateResult{}, fmt.Errorf("global_pay_receipt: nil provider")
	}
	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "UZS"
	}
	body := map[string]any{
		"attempt_id":      req.AttemptID,
		"order_id":        req.OrderID,
		"supplier_id":     req.SupplierID,
		"retailer_id":     req.RetailerID,
		"amount_minor":    req.AmountMinor,
		"currency":        currency,
		"payment_method":  req.PaymentMethod,
		"idempotency_key": req.AttemptID,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return FiscalCreateResult{}, fmt.Errorf("global_pay_receipt marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+p.Path, bytes.NewReader(raw))
	if err != nil {
		return FiscalCreateResult{}, fmt.Errorf("global_pay_receipt request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("X-Idempotency-Key", req.AttemptID)

	client := p.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: FiscalOFDTimeout}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return FiscalCreateResult{}, fmt.Errorf("global_pay_receipt http: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return FiscalCreateResult{}, fmt.Errorf("global_pay_receipt status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var parsed struct {
		ReceiptID       string `json:"receipt_id"`
		FiscalReceiptID string `json:"fiscal_receipt_id"`
		QR              string `json:"qr"`
		QRURL           string `json:"qr_url"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return FiscalCreateResult{}, fmt.Errorf("global_pay_receipt decode: %w", err)
	}
	id := firstNonEmpty(parsed.FiscalReceiptID, parsed.ReceiptID)
	if id == "" {
		return FiscalCreateResult{}, fmt.Errorf("global_pay_receipt missing receipt_id")
	}
	return FiscalCreateResult{
		FiscalReceiptID: id,
		FiscalQR:        firstNonEmpty(parsed.QRURL, parsed.QR),
		RawPayload:      respBody,
	}, nil
}

// multiReceiptProvider runs primary (PEGASUS) for hard-gate success, then optionally
// attaches a Global Pay payment receipt into the payload (best-effort; never fails the gate).
type multiReceiptProvider struct {
	primary  FiscalProvider
	payment  FiscalProvider
	name     string
}

func (m multiReceiptProvider) CreateReceipt(ctx context.Context, req FiscalCreateRequest) (FiscalCreateResult, error) {
	if m.primary == nil {
		return FiscalCreateResult{}, fmt.Errorf("multi_receipt: nil primary")
	}
	primary, err := m.primary.CreateReceipt(ctx, req)
	if err != nil {
		return FiscalCreateResult{}, err
	}
	if m.payment == nil {
		return primary, nil
	}
	payRes, payErr := m.payment.CreateReceipt(ctx, req)
	// Enrich payload; payment failure is recorded but does not block COMPLETED.
	var envelope map[string]any
	if len(primary.RawPayload) > 0 {
		_ = json.Unmarshal(primary.RawPayload, &envelope)
	}
	if envelope == nil {
		envelope = map[string]any{}
	}
	if payErr != nil {
		envelope["payment_receipt"] = map[string]any{
			"provider": FiscalProviderGlobalPay,
			"status":   "failed",
			"error":    payErr.Error(),
		}
	} else {
		envelope["payment_receipt"] = map[string]any{
			"provider":   FiscalProviderGlobalPay,
			"status":     "success",
			"receipt_id": payRes.FiscalReceiptID,
			"qr":         payRes.FiscalQR,
		}
	}
	raw, _ := json.Marshal(envelope)
	primary.RawPayload = raw
	return primary, nil
}

func envTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
