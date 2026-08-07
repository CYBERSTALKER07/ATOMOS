package order

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// PegasusReceiptProvider issues platform commercial receipts owned by PegasusX.
// These are NOT Soliq/OFD tax fiscal receipts. They satisfy the ADR-009 hard-gate
// (order may COMPLETE after a successful attempt) while tax OFD (MY_SOLIQ) remains
// deferred until credentials arrive.
//
// Receipt identity is deterministic per attempt so retries stay idempotent.
type PegasusReceiptProvider struct {
	// PublicBaseURL is used for QR deep links (env PUBLIC_BASE_URL when empty).
	PublicBaseURL string
}

// CreateReceipt builds a durable platform receipt payload stored on OrderFiscalReceipts.
func (p PegasusReceiptProvider) CreateReceipt(_ context.Context, req FiscalCreateRequest) (FiscalCreateResult, error) {
	// Optional SSMR fail hooks (only when explicitly enabled) so fiscal smoke
	// can still exercise FISCAL_FAILED without switching back to FAKE.
	if pegasusSSMRHooksEnabled() {
		if strings.Contains(strings.ToLower(req.OrderID), "fiscal-fail") {
			return FiscalCreateResult{}, fmt.Errorf("pegasus_receipt_rejected: order_id=%s", req.OrderID)
		}
		if strings.Contains(strings.ToLower(req.RetailerID), "fiscal-fail") {
			return FiscalCreateResult{}, fmt.Errorf("pegasus_receipt_rejected: retailer_id=%s", req.RetailerID)
		}
		if req.AmountMinor == FiscalFakeFailAmountMinor {
			return FiscalCreateResult{}, fmt.Errorf("pegasus_receipt_rejected: amount_minor=%d (ssmr fail hook)", FiscalFakeFailAmountMinor)
		}
	}

	attemptID := strings.TrimSpace(req.AttemptID)
	if attemptID == "" {
		return FiscalCreateResult{}, fmt.Errorf("pegasus_receipt: attempt_id required")
	}
	receiptID := "PX-RCPT-" + attemptID
	if len(receiptID) > 128 {
		receiptID = receiptID[:128]
	}
	base := strings.TrimRight(strings.TrimSpace(p.PublicBaseURL), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")
	}
	if base == "" {
		base = "https://api-ssmr.pegasusx.app"
	}
	// QR opens the branded HTML document (JSON remains the default Accept/API form).
	qr := base + "/v1/platform/receipts/" + receiptID + "?format=html"

	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "UZS"
	}
	country := countryFromCurrency(currency)
	layout := receiptLayoutForCountry(country)
	company := pegasusCompanyName()
	tin := pegasusIssuerTIN()

	lines := make([]map[string]any, 0, len(req.LineItems))
	for _, li := range req.LineItems {
		lines = append(lines, map[string]any{
			"sku":              li.SKU,
			"name":             li.Name,
			"quantity":         li.Quantity,
			"unit_price_minor": li.UnitPrice,
		})
	}
	payload := map[string]any{
		"provider":       FiscalProviderPegasus,
		"legal_class":    "platform_receipt",
		"tax_ofd":        false,
		"tax_ofd_note":   layout.OFDDeferredNote,
		"receipt_id":     receiptID,
		"attempt_id":     attemptID,
		"order_id":       req.OrderID,
		"supplier_id":    req.SupplierID,
		"retailer_id":    req.RetailerID,
		"amount_minor":   req.AmountMinor,
		"currency":       currency,
		"country_code":   country,
		"payment_method": req.PaymentMethod,
		"line_items":     lines,
		"issued_at":      time.Now().UTC().Format(time.RFC3339),
		"qr_url":         qr,
		"company_name":   company,
		"issuer_tin":     tin,
		"logo_path":      "order/receipt_assets/logo.svg",
		"branding": map[string]any{
			"company_name": company,
			"issuer_tin":   tin,
			"logo_path":    "order/receipt_assets/logo.svg",
			"title":        layout.Title,
			"subtitle":     layout.Subtitle,
			"style":        "pegasus_settlement_v1",
		},
		"payment_receipt": map[string]any{
			"provider": FiscalProviderGlobalPay,
			"status":   "deferred",
			"note":     "wire when Global Pay receipt API credentials arrive",
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return FiscalCreateResult{}, fmt.Errorf("pegasus_receipt marshal: %w", err)
	}
	return FiscalCreateResult{
		FiscalReceiptID: receiptID,
		FiscalQR:        qr,
		RawPayload:      raw,
	}, nil
}

// CreateCorrectiveReceipt issues a platform credit receipt referencing the
// original platform receipt. Not a tax document — the tax corrective chain is
// MY_SOLIQ's job once EDS credentials land; this keeps the refund audit trail
// honest in the interim.
func (p PegasusReceiptProvider) CreateCorrectiveReceipt(_ context.Context, req FiscalCorrectiveRequest) (FiscalCreateResult, error) {
	if strings.TrimSpace(req.AttemptID) == "" {
		return FiscalCreateResult{}, fmt.Errorf("pegasus corrective: attempt_id required")
	}
	if strings.TrimSpace(req.OriginalReceiptID) == "" {
		return FiscalCreateResult{}, fmt.Errorf("pegasus corrective: original receipt id required")
	}
	receiptID := "PX-CN-" + req.AttemptID
	if len(receiptID) > 128 {
		receiptID = receiptID[:128]
	}
	base := strings.TrimRight(strings.TrimSpace(p.PublicBaseURL), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")
	}
	if base == "" {
		base = "https://api-ssmr.pegasusx.app"
	}
	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "UZS"
	}
	payload := map[string]any{
		"provider":             FiscalProviderPegasus,
		"legal_class":          "platform_credit_receipt",
		"tax_ofd":              false,
		"tax_ofd_note":         "platform credit note; tax corrective EHF deferred until Soliq EDS credentials arrive",
		"receipt_id":           receiptID,
		"attempt_id":           req.AttemptID,
		"order_id":             req.OrderID,
		"supplier_id":          req.SupplierID,
		"retailer_id":          req.RetailerID,
		"corrects_receipt_id":  req.OriginalReceiptID,
		"correct_reason":       req.ReasonCode,
		"amount_minor":         req.AmountMinor,
		"currency":             currency,
		"issued_at":            time.Now().UTC().Format(time.RFC3339),
		"qr_url":               base + "/v1/platform/receipts/" + receiptID + "?format=html",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return FiscalCreateResult{}, fmt.Errorf("pegasus corrective marshal: %w", err)
	}
	return FiscalCreateResult{FiscalReceiptID: receiptID, RawPayload: raw}, nil
}

func pegasusSSMRHooksEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("FISCAL_PEGASUS_SSMR_HOOKS")))
	if v == "1" || v == "true" || v == "yes" {
		return true
	}
	// SSMR tenant default: keep fiscal e2e fail hooks without FAKE.
	return strings.EqualFold(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")), "ssmr")
}
