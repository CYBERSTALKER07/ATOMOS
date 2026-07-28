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

// Fiscal provider selection (ADR-009 hard-gate).
//
//	FISCAL_PROVIDER=PEGASUS    (default product path — platform commercial receipts, no Soliq)
//	FISCAL_PROVIDER=FAKE       SSMR test hooks (amount=13 / fiscal-fail ids)
//	FISCAL_PROVIDER=MY_SOLIQ   tax OFD HTTP adapter (when Soliq sandbox/prod creds arrive)
//	FISCAL_PROVIDER=GLOBAL_PAY payment-provider receipts only (rarely used alone)
//
// Optional secondary payment receipt (best-effort, never blocks COMPLETED):
//
//	FISCAL_GLOBAL_PAY_RECEIPT_ENABLED=true
//	FISCAL_GLOBAL_PAY_RECEIPT_BASE_URL=...
//	FISCAL_GLOBAL_PAY_RECEIPT_API_KEY=...
//
// My.soliq / OFD credentials (required when MY_SOLIQ):
//
//	FISCAL_MY_SOLIQ_BASE_URL   e.g. https://sandbox.ofd.example/api
//	FISCAL_MY_SOLIQ_API_KEY    bearer or API key
//	FISCAL_MY_SOLIQ_TIN        supplier taxpayer ID (STIR)
//	FISCAL_MY_SOLIQ_PATH       optional path, default /v1/receipts
//	FISCAL_MY_SOLIQ_TIMEOUT_MS optional, default 8000
const (
	envFiscalProvider   = "FISCAL_PROVIDER"
	envMySoliqBaseURL   = "FISCAL_MY_SOLIQ_BASE_URL"
	envMySoliqAPIKey    = "FISCAL_MY_SOLIQ_API_KEY"
	envMySoliqTIN       = "FISCAL_MY_SOLIQ_TIN"
	envMySoliqPath      = "FISCAL_MY_SOLIQ_PATH"
	envMySoliqTimeoutMS = "FISCAL_MY_SOLIQ_TIMEOUT_MS"
	// SSMR fail hook: FakeFiscalProvider rejects amount_minor == FiscalFakeFailAmountMinor.
	FiscalFakeFailAmountMinor int64 = 13
)

// ProviderFromEnv selects PEGASUS (default), FAKE, MY_SOLIQ, or GLOBAL_PAY.
// When PEGASUS is primary and Global Pay receipt env is enabled, wraps multi-receipt.
func ProviderFromEnv() FiscalProvider {
	switch strings.ToUpper(strings.TrimSpace(os.Getenv(envFiscalProvider))) {
	case FiscalProviderMySoliq, "MYSOLIQ", "SOLIQ", "OFD":
		p, err := NewMySoliqProviderFromEnv()
		if err != nil {
			// Misconfigured production adapter must not silently fall back in
			// a way that invents fiscal success — surface as hard-fail provider.
			return hardFailProvider{reason: err.Error()}
		}
		return p
	case FiscalProviderFake:
		return FakeFiscalProvider{}
	case FiscalProviderGlobalPay:
		p, err := NewGlobalPayReceiptProviderFromEnv()
		if err != nil {
			return hardFailProvider{reason: err.Error()}
		}
		if p == nil {
			return hardFailProvider{reason: "GLOBAL_PAY receipt provider not enabled — set FISCAL_GLOBAL_PAY_RECEIPT_ENABLED=true and credentials"}
		}
		return p
	default:
		// PEGASUS (and empty / unknown aliases) — product default without Soliq.
		primary := PegasusReceiptProvider{PublicBaseURL: strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL"))}
		gp, err := NewGlobalPayReceiptProviderFromEnv()
		if err != nil {
			// Misconfigured secondary must not block platform receipts.
			// hard-fail only when GLOBAL_PAY is the primary provider (handled above).
			return primary
		}
		if gp != nil {
			return multiReceiptProvider{
				primary: primary,
				payment: gp,
				name:    FiscalProviderPegasus,
			}
		}
		return primary
	}
}

// FakeFiscalProvider succeeds unless OrderID contains "fiscal-fail" or amount is the SSMR fail hook.
type FakeFiscalProvider struct{}

func (FakeFiscalProvider) CreateReceipt(_ context.Context, req FiscalCreateRequest) (FiscalCreateResult, error) {
	if strings.Contains(strings.ToLower(req.OrderID), "fiscal-fail") {
		return FiscalCreateResult{}, fmt.Errorf("fake_ofd_rejected: order_id=%s", req.OrderID)
	}
	if strings.Contains(strings.ToLower(req.RetailerID), "fiscal-fail") {
		return FiscalCreateResult{}, fmt.Errorf("fake_ofd_rejected: retailer_id=%s", req.RetailerID)
	}
	if req.AmountMinor == FiscalFakeFailAmountMinor {
		return FiscalCreateResult{}, fmt.Errorf("fake_ofd_rejected: amount_minor=%d (ssmr fail hook)", FiscalFakeFailAmountMinor)
	}
	id := "FAKE-RCPT-" + req.AttemptID
	if len(id) > 64 {
		id = id[:64]
	}
	return FiscalCreateResult{
		FiscalReceiptID: id,
		FiscalQR:        "https://fake.ofd.local/qr/" + req.AttemptID,
		RawPayload:      []byte(`{"provider":"FAKE","ok":true}`),
	}, nil
}

// hardFailProvider always fails — used when MY_SOLIQ is selected but misconfigured.
type hardFailProvider struct{ reason string }

func (p hardFailProvider) CreateReceipt(_ context.Context, _ FiscalCreateRequest) (FiscalCreateResult, error) {
	return FiscalCreateResult{}, fmt.Errorf("ofd_misconfigured: %s", p.reason)
}

// MySoliqProvider is the HTTP OFD adapter for my.soliq.uz-class endpoints.
// Request/response shapes are intentionally thin so sandbox and production
// gateways can map without changing the worker contract.
type MySoliqProvider struct {
	BaseURL    string
	APIKey     string
	TIN        string
	Path       string
	HTTPClient *http.Client
}

// NewMySoliqProviderFromEnv builds a provider from env; errors if required vars missing.
func NewMySoliqProviderFromEnv() (*MySoliqProvider, error) {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv(envMySoliqBaseURL)), "/")
	key := strings.TrimSpace(os.Getenv(envMySoliqAPIKey))
	tin := strings.TrimSpace(os.Getenv(envMySoliqTIN))
	if base == "" {
		return nil, fmt.Errorf("%s required when FISCAL_PROVIDER=MY_SOLIQ", envMySoliqBaseURL)
	}
	if key == "" {
		return nil, fmt.Errorf("%s required when FISCAL_PROVIDER=MY_SOLIQ", envMySoliqAPIKey)
	}
	if tin == "" {
		return nil, fmt.Errorf("%s required when FISCAL_PROVIDER=MY_SOLIQ", envMySoliqTIN)
	}
	path := strings.TrimSpace(os.Getenv(envMySoliqPath))
	if path == "" {
		path = "/v1/receipts"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	timeout := FiscalOFDTimeout
	if ms := strings.TrimSpace(os.Getenv(envMySoliqTimeoutMS)); ms != "" {
		var n int
		if _, err := fmt.Sscanf(ms, "%d", &n); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Millisecond
		}
	}
	return &MySoliqProvider{
		BaseURL: base,
		APIKey:  key,
		TIN:     tin,
		Path:    path,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

type mySoliqReceiptRequest struct {
	AttemptID     string                   `json:"attempt_id"`
	OrderID       string                   `json:"order_id"`
	SupplierID    string                   `json:"supplier_id"`
	RetailerID    string                   `json:"retailer_id,omitempty"`
	TIN           string                   `json:"tin"`
	AmountMinor   int64                    `json:"amount_minor"`
	Currency      string                   `json:"currency"`
	PaymentMethod string                   `json:"payment_method"`
	LineItems     []mySoliqReceiptLineItem `json:"line_items,omitempty"`
	IdempotencyKey string                  `json:"idempotency_key"`
}

type mySoliqReceiptLineItem struct {
	SKU       string `json:"sku,omitempty"`
	Name      string `json:"name,omitempty"`
	Quantity  int64  `json:"quantity"`
	UnitPrice int64  `json:"unit_price_minor"`
}

type mySoliqReceiptResponse struct {
	ReceiptID string `json:"receipt_id"`
	// Alternate field names seen on sandbox gateways.
	FiscalReceiptID string `json:"fiscal_receipt_id"`
	QR              string `json:"qr"`
	QRURL           string `json:"qr_url"`
	FiscalQR        string `json:"fiscal_qr"`
	Error           string `json:"error"`
	Message         string `json:"message"`
	Code            string `json:"code"`
}

func (p *MySoliqProvider) CreateReceipt(ctx context.Context, req FiscalCreateRequest) (FiscalCreateResult, error) {
	if p == nil {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq: nil provider")
	}
	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "UZS"
	}
	body := mySoliqReceiptRequest{
		AttemptID:      req.AttemptID,
		OrderID:        req.OrderID,
		SupplierID:     req.SupplierID,
		RetailerID:     req.RetailerID,
		TIN:            p.TIN,
		AmountMinor:    req.AmountMinor,
		Currency:       currency,
		PaymentMethod:  req.PaymentMethod,
		IdempotencyKey: req.AttemptID, // provider-side dedupe key
	}
	for _, li := range req.LineItems {
		body.LineItems = append(body.LineItems, mySoliqReceiptLineItem{
			SKU:       li.SKU,
			Name:      li.Name,
			Quantity:  li.Quantity,
			UnitPrice: li.UnitPrice,
		})
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq marshal: %w", err)
	}
	url := p.BaseURL + p.Path
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("X-Idempotency-Key", req.AttemptID)
	httpReq.Header.Set("X-Taxpayer-TIN", p.TIN)

	client := p.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: FiscalOFDTimeout}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq http: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var parsed mySoliqReceiptResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq decode: %w body=%s", err, truncate(string(respBody), 200))
	}
	if parsed.Error != "" || parsed.Code != "" && strings.EqualFold(parsed.Code, "error") {
		msg := parsed.Error
		if msg == "" {
			msg = parsed.Message
		}
		return FiscalCreateResult{}, fmt.Errorf("mysoliq error: %s", msg)
	}
	receiptID := firstNonEmpty(parsed.FiscalReceiptID, parsed.ReceiptID)
	qr := firstNonEmpty(parsed.FiscalQR, parsed.QRURL, parsed.QR)
	if receiptID == "" {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq missing receipt_id in response")
	}
	return FiscalCreateResult{
		FiscalReceiptID: receiptID,
		FiscalQR:        qr,
		RawPayload:      respBody,
	}, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
