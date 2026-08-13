package order

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// PublicReceiptView is the redacted commercial receipt returned to clients/QR JSON.
// It is not a Soliq tax fiscal document unless Provider == MY_SOLIQ.
type PublicReceiptView struct {
	ReceiptID     string        `json:"receipt_id"`
	Provider      string        `json:"provider"`
	LegalClass    string        `json:"legal_class"`
	TaxOFD        bool          `json:"tax_ofd"`
	Status        string        `json:"status"`
	OrderID       string        `json:"order_id"`
	SupplierID    string        `json:"supplier_id,omitempty"`
	RetailerID    string        `json:"retailer_id,omitempty"`
	AmountMinor   int64         `json:"amount_minor"`
	Currency      string        `json:"currency"`
	PaymentMethod string        `json:"payment_method,omitempty"`
	QRURL         string        `json:"qr_url,omitempty"`
	HTMLURL       string        `json:"html_url,omitempty"`
	PDFURL        string        `json:"pdf_url,omitempty"`
	IssuedAt      time.Time     `json:"issued_at,omitempty"`
	CountryCode   string        `json:"country_code,omitempty"`
	PartyCopy     string        `json:"party_copy,omitempty"`
	CompanyName   string        `json:"company_name,omitempty"`
	LineItems     []ReceiptLine `json:"line_items,omitempty"`
}

func resolveReceiptFormat(r *http.Request) string {
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	switch q {
	case "html", "pdf", "json":
		return q
	}
	accept := strings.ToLower(r.Header.Get("Accept"))
	switch {
	case strings.Contains(accept, "text/html"):
		return "html"
	case strings.Contains(accept, "application/pdf"):
		return "pdf"
	default:
		return "json"
	}
}

func (s *Service) loadSuccessfulFiscal(receiptID string, r *http.Request) (FiscalReceiptRow, bool, error) {
	if s == nil || s.repo == nil {
		return FiscalReceiptRow{}, false, nil
	}
	fr, ok, err := s.repo.GetFiscalByReceiptID(r.Context(), receiptID)
	if err != nil || !ok {
		return fr, ok, err
	}
	switch fr.Status {
	case FiscalAttemptSuccess, FiscalAttemptForceSkipped:
		return fr, true, nil
	default:
		return FiscalReceiptRow{}, false, nil
	}
}

func (s *Service) orderForReceipt(r *http.Request, orderID string) (*Order, error) {
	if s == nil || s.repo == nil || strings.TrimSpace(orderID) == "" {
		return nil, nil
	}
	o, ok, err := s.loadOrderForRequest(r.Context(), orderID)
	if err != nil || !ok {
		return nil, err
	}
	return &o, nil
}

func writeReceiptResponse(w http.ResponseWriter, r *http.Request, doc ReceiptDocument) {
	switch resolveReceiptFormat(r) {
	case "html":
		body, err := RenderReceiptHTML(doc)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "receipt_render_failed"})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "private, max-age=60")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	case "pdf":
		body, err := RenderReceiptPDF(doc)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "receipt_pdf_failed"})
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `inline; filename="`+doc.ReceiptID+`.pdf"`)
		w.Header().Set("Cache-Control", "private, max-age=60")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	default:
		view := PublicReceiptView{
			ReceiptID:     doc.ReceiptID,
			Provider:      doc.Provider,
			LegalClass:    doc.LegalClass,
			TaxOFD:        doc.TaxOFD,
			Status:        doc.Status,
			OrderID:       doc.OrderID,
			SupplierID:    doc.SupplierID,
			RetailerID:    doc.RetailerID,
			AmountMinor:   doc.AmountMinor,
			Currency:      doc.Currency,
			PaymentMethod: doc.PaymentMethod,
			QRURL:         doc.QRURL,
			HTMLURL:       doc.HTMLURL,
			PDFURL:        doc.PDFURL,
			IssuedAt:      doc.IssuedAt,
			CountryCode:   doc.CountryCode,
			PartyCopy:     string(doc.PartyCopy),
			CompanyName:   doc.CompanyName,
			LineItems:     doc.LineItems,
		}
		writeJSON(w, http.StatusOK, view)
	}
}

// HandleGetPlatformReceipt serves GET /v1/platform/receipts/{receiptID}.
// Public read of a successful platform (or OFD) receipt by id — used by QR deep links.
// Supports ?format=json|html|pdf (default json; browsers with Accept: text/html get HTML).
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
	fr, ok, err := s.loadSuccessfulFiscal(receiptID, r)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "receipt_not_found"})
		return
	}
	ord, _ := s.orderForReceipt(r, fr.OrderID)
	doc := BuildReceiptDocument(fr, ord, PartyCopyPublic)
	writeReceiptResponse(w, r, doc)
}

func (s *Service) loadOrderReceiptForParty(w http.ResponseWriter, r *http.Request, party ReceiptPartyCopy) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}
	if s == nil || s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	orderRecord, found, err := s.repo.GetOrder(r.Context(), orderID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	if !authorizeReceiptParty(claims, orderRecord, party) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	receiptID := strings.TrimSpace(orderRecord.LatestFiscalReceiptID)
	var fr FiscalReceiptRow
	var frOK bool
	if receiptID != "" {
		fr, frOK, err = s.loadSuccessfulFiscal(receiptID, r)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
	}
	if !frOK && strings.TrimSpace(orderRecord.LatestFiscalAttemptID) != "" {
		fr, frOK, err = s.repo.GetFiscalAttempt(r.Context(), orderID, orderRecord.LatestFiscalAttemptID)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		if frOK {
			switch fr.Status {
			case FiscalAttemptSuccess, FiscalAttemptForceSkipped:
			default:
				frOK = false
			}
		}
	}
	if !frOK {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "receipt_not_found"})
		return
	}
	doc := BuildReceiptDocument(fr, &orderRecord, party)
	writeReceiptResponse(w, r, doc)
}

func authorizeReceiptParty(claims auth.Claims, o Order, party ReceiptPartyCopy) bool {
	switch party {
	case PartyCopyRetailer:
		if claims.Role == auth.RoleRetailer {
			// B3 M-P0-4: receipt party scoped to org, not staff Subject.
			orgID := auth.ResolveRetailerOrgID(claims)
			return orgID != "" && orgID == o.RetailerID
		}
		// Admin may inspect retailer copy within supplier tenant.
		if claims.Role == auth.RoleAdmin {
			return claims.SupplierID == "" || claims.SupplierID == o.SupplierID
		}
		return false
	case PartyCopySupplier:
		if claims.Role == auth.RoleAdmin {
			return claims.SupplierID == "" || claims.SupplierID == o.SupplierID
		}
		return false
	case PartyCopyWarehouse:
		switch claims.Role {
		case auth.RoleWarehouse, auth.RoleWarehouseAdmin, auth.RoleAdmin:
			return claims.SupplierID == "" || claims.SupplierID == o.SupplierID
		default:
			return false
		}
	default:
		return false
	}
}

// HandleGetRetailerOrderReceipt serves GET /v1/retailer/orders/{orderID}/receipt
func (s *Service) HandleGetRetailerOrderReceipt(w http.ResponseWriter, r *http.Request) {
	s.loadOrderReceiptForParty(w, r, PartyCopyRetailer)
}

// HandleGetSupplierOrderReceipt serves GET /v1/supplier/orders/{orderID}/receipt
func (s *Service) HandleGetSupplierOrderReceipt(w http.ResponseWriter, r *http.Request) {
	s.loadOrderReceiptForParty(w, r, PartyCopySupplier)
}

// HandleGetWarehouseOrderReceipt serves GET /v1/warehouse/orders/{orderID}/receipt
func (s *Service) HandleGetWarehouseOrderReceipt(w http.ResponseWriter, r *http.Request) {
	s.loadOrderReceiptForParty(w, r, PartyCopyWarehouse)
}
