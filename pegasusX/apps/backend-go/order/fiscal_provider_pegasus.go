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
	qr := base + "/v1/platform/receipts/" + receiptID

	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "UZS"
	}
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
		"provider":          FiscalProviderPegasus,
		"legal_class":       "platform_receipt",
		"tax_ofd":           false,
		"tax_ofd_note":      "Soliq/OFD deferred; this is a PegasusX commercial receipt for delivery settlement hard-gate",
		"receipt_id":        receiptID,
		"attempt_id":        attemptID,
		"order_id":          req.OrderID,
		"supplier_id":       req.SupplierID,
		"retailer_id":       req.RetailerID,
		"amount_minor":      req.AmountMinor,
		"currency":          currency,
		"payment_method":    req.PaymentMethod,
		"line_items":        lines,
		"issued_at":         time.Now().UTC().Format(time.RFC3339),
		"qr_url":            qr,
		"payment_receipt":   map[string]any{"provider": FiscalProviderGlobalPay, "status": "deferred", "note": "wire when Global Pay receipt API credentials arrive"},
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

func pegasusSSMRHooksEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("FISCAL_PEGASUS_SSMR_HOOKS")))
	if v == "1" || v == "true" || v == "yes" {
		return true
	}
	// SSMR tenant default: keep fiscal e2e fail hooks without FAKE.
	return strings.EqualFold(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")), "ssmr")
}
