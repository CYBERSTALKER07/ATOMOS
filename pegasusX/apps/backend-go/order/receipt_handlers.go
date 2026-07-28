package order

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// PublicReceiptView is the redacted commercial receipt returned to clients/QR.
// It is not a Soliq tax fiscal document unless Provider == MY_SOLIQ.
type PublicReceiptView struct {
	ReceiptID     string    `json:"receipt_id"`
	Provider      string    `json:"provider"`
	LegalClass    string    `json:"legal_class"`
	TaxOFD        bool      `json:"tax_ofd"`
	Status        string    `json:"status"`
	OrderID       string    `json:"order_id"`
	SupplierID    string    `json:"supplier_id,omitempty"`
	AmountMinor   int64     `json:"amount_minor"`
	Currency      string    `json:"currency"`
	PaymentMethod string    `json:"payment_method,omitempty"`
	QRURL         string    `json:"qr_url,omitempty"`
	IssuedAt      time.Time `json:"issued_at,omitempty"`
}

// HandleGetPlatformReceipt serves GET /v1/platform/receipts/{receiptID}.
// Public read of a successful platform (or OFD) receipt by id — used by QR deep links.
func (s *Service) HandleGetPlatformReceipt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}
	receiptID := strings.TrimSpace(chi.URLParam(r, "receiptID"))
	if receiptID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "receipt_id_required"})
		return
	}
	if s == nil || s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	fr, ok, err := s.repo.GetFiscalByReceiptID(r.Context(), receiptID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "receipt_not_found"})
		return
	}
	// Only expose successful / force-skipped receipts publicly.
	switch fr.Status {
	case FiscalAttemptSuccess, FiscalAttemptForceSkipped:
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "receipt_not_found"})
		return
	}
	legal := "platform_receipt"
	taxOFD := false
	if strings.EqualFold(fr.Provider, FiscalProviderMySoliq) {
		legal = "tax_ofd_receipt"
		taxOFD = true
	}
	view := PublicReceiptView{
		ReceiptID:     fr.FiscalReceiptID,
		Provider:      fr.Provider,
		LegalClass:    legal,
		TaxOFD:        taxOFD,
		Status:        fr.Status,
		OrderID:       fr.OrderID,
		SupplierID:    fr.SupplierID,
		AmountMinor:   fr.AmountMinor,
		Currency:      fr.Currency,
		PaymentMethod: fr.PaymentMethod,
		QRURL:         fr.FiscalQR,
		IssuedAt:      fr.CreatedAt,
	}
	writeJSON(w, http.StatusOK, view)
}
