package order

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/fiscal"
	"github.com/pegasusx/pegasusx/apps/backend-go/soliq"
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

func (FakeFiscalProvider) GetSoliqClient() soliq.SoliqClient { return nil }

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

func (p hardFailProvider) GetSoliqClient() soliq.SoliqClient { return nil }

func (p hardFailProvider) CreateReceipt(_ context.Context, _ FiscalCreateRequest) (FiscalCreateResult, error) {
	return FiscalCreateResult{}, fmt.Errorf("ofd_misconfigured: %s", p.reason)
}

// MySoliqProvider is the HTTP OFD adapter for my.soliq.uz-class endpoints.
// Request/response shapes are intentionally thin so sandbox and production
// gateways can map without changing the worker contract.
type MySoliqProvider struct {
	TIN         string
	soliqClient soliq.SoliqClient
	signer      fiscal.EDSSigner
}

// NewMySoliqProvider builds a MY_SOLIQ adapter with an explicit signer and Soliq client.
// Prefer NewMySoliqProviderFromEnv in production; this constructor is for SSMR/contract
// harnesses that inject a mock BaseURL + DevHMACSigner.
func NewMySoliqProvider(baseURL, apiKey, tin string, signer fiscal.EDSSigner) (*MySoliqProvider, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	apiKey = strings.TrimSpace(apiKey)
	tin = strings.TrimSpace(tin)
	if baseURL == "" || apiKey == "" || tin == "" {
		return nil, fmt.Errorf("mysoliq: baseURL, apiKey, and tin required")
	}
	if signer == nil {
		return nil, fmt.Errorf("mysoliq: EDSSigner required")
	}
	return &MySoliqProvider{
		TIN: tin,
		soliqClient: soliq.NewClient(soliq.SoliqConfig{
			BaseURL: baseURL,
			APIKey:  apiKey,
			TIN:     tin,
		}),
		signer: signer,
	}, nil
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
	// EDS signing is mandatory: MY_SOLIQ issues legal tax receipts (EHF), and an
	// unsigned receipt must never reach the gateway. Fail-closed at construction
	// instead of per-receipt so misconfiguration surfaces at deploy, not at sale.
	signer, err := fiscal.SignerFromEnv(os.Getenv("PEGASUSX_ENV"))
	if err != nil {
		return nil, err
	}
	cfg := soliq.SoliqConfig{
		BaseURL: base,
		APIKey:  key,
		TIN:     tin,
		Timeout: timeout,
	}
	return &MySoliqProvider{
		TIN:         tin,
		soliqClient: soliq.NewClient(cfg),
		signer:      signer,
	}, nil
}

// SetSigner injects the EDS signer (contract tests wire the dev-hmac signer here).
func (p *MySoliqProvider) SetSigner(signer fiscal.EDSSigner) {
	p.signer = signer
}

type mySoliqReceiptRequest struct {
	AttemptID      string                   `json:"attempt_id"`
	OrderID        string                   `json:"order_id"`
	SupplierID     string                   `json:"supplier_id"`
	RetailerID     string                   `json:"retailer_id,omitempty"`
	TIN            string                   `json:"tin"`
	AmountMinor    int64                    `json:"amount_minor"`
	Currency       string                   `json:"currency"`
	PaymentMethod  string                   `json:"payment_method"`
	LineItems      []mySoliqReceiptLineItem `json:"line_items,omitempty"`
	IdempotencyKey string                   `json:"idempotency_key"`
	// Corrective chain: set on refund EHF that reverses an earlier receipt.
	CorrectsEhfID  string                   `json:"corrects_ehf_id,omitempty"`
	CorrectReason  string                   `json:"correct_reason,omitempty"`
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
	if p.signer == nil {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq: no EDSSigner configured")
	}

	signedPayload, err := fiscal.AttachSignature(ctx, p.signer, body)
	if err != nil {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq attach signature: %w", err)
	}

	resp, err := p.soliqClient.Submit(ctx, signedPayload, req.AttemptID)
	if err != nil {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq submit: %w", err)
	}
	if !resp.Success {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq error (code=%s): %s", resp.ErrorCode, resp.ErrorMessage)
	}
	
	if resp.EhfID == "" {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq missing receipt_id in response")
	}
	return FiscalCreateResult{
		FiscalReceiptID: resp.EhfID,
		RawPayload:      resp.RawBody,
	}, nil
}

func (p *MySoliqProvider) GetSoliqClient() soliq.SoliqClient {
	return p.soliqClient
}

// CreateCorrectiveReceipt submits the corrective (refund) EHF referencing the
// original receipt. Same signed envelope plus corrects_ehf_id; idempotency key
// is the corrective attempt id so retries never double-issue.
func (p *MySoliqProvider) CreateCorrectiveReceipt(ctx context.Context, req FiscalCorrectiveRequest) (FiscalCreateResult, error) {
	if p == nil {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq: nil provider")
	}
	if strings.TrimSpace(req.OriginalReceiptID) == "" {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq corrective: original receipt id required")
	}
	if p.signer == nil {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq: no EDSSigner configured")
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
		PaymentMethod:  "REFUND",
		IdempotencyKey: req.AttemptID,
		CorrectsEhfID:  req.OriginalReceiptID,
		CorrectReason:  req.ReasonCode,
	}
	signedPayload, err := fiscal.AttachSignature(ctx, p.signer, body)
	if err != nil {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq attach signature: %w", err)
	}
	resp, err := p.soliqClient.Submit(ctx, signedPayload, req.AttemptID)
	if err != nil {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq submit corrective: %w", err)
	}
	if !resp.Success {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq corrective error (code=%s): %s", resp.ErrorCode, resp.ErrorMessage)
	}
	if resp.EhfID == "" {
		return FiscalCreateResult{}, fmt.Errorf("mysoliq corrective missing receipt_id in response")
	}
	return FiscalCreateResult{FiscalReceiptID: resp.EhfID, RawPayload: resp.RawBody}, nil
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

func (p *GlobalPayReceiptProvider) GetSoliqClient() soliq.SoliqClient { return nil }
func (PegasusReceiptProvider) GetSoliqClient() soliq.SoliqClient { return nil }
func (multiReceiptProvider) GetSoliqClient() soliq.SoliqClient { return nil }

