package retailer

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type disputeOpeningBalanceRequest struct {
	Reason string `json:"reason"`
}

// HandleListNasiyaOpeningBalances serves GET /v1/retailer/nasiya/opening-balances.
func (s *Service) HandleListNasiyaOpeningBalances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if s == nil || s.migrationAR == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "nasiya_opening_balances_unavailable"})
		return
	}
	items, err := s.migrationAR.ListOpeningBalances(r.Context(), rid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_opening_balances_failed"})
		return
	}
	if items == nil {
		items = []MigrationOpeningInvoice{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"retailer_id": rid,
		"invoices":    items,
	})
}

// HandleAcknowledgeARInvoice serves POST /v1/retailer/ar-invoices/{invoiceID}/acknowledge.
func (s *Service) HandleAcknowledgeARInvoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if s == nil || s.migrationAR == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "nasiya_opening_balances_unavailable"})
		return
	}
	invoiceID := strings.TrimSpace(chi.URLParam(r, "invoiceID"))
	if invoiceID == "" {
		invoiceID = strings.TrimSpace(chi.URLParam(r, "invoiceId"))
	}
	if invoiceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invoice_id_required"})
		return
	}
	body, ok := readLimitedBody(w, r, 16*1024)
	if !ok {
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	inv, err := s.migrationAR.AcknowledgeOpeningInvoice(r.Context(), rid, invoiceID)
	if err != nil {
		writeMigrationARError(w, err)
		return
	}
	if s.migrationNotifier != nil && strings.TrimSpace(inv.SupplierID) != "" {
		_ = s.migrationNotifier(
			r.Context(),
			inv.SupplierID,
			string(auth.RoleAdmin),
			"AR_MIGRATION_ACKNOWLEDGED",
			"Retailer confirmed opening Nasiya balance",
			"Invoice "+inv.InvoiceID+" acknowledged by retailer "+rid,
			"/nasiya?invoiceId="+inv.InvoiceID,
		)
	}
	respBytes, _ := json.Marshal(map[string]any{
		"retailer_id": rid,
		"invoice":     inv,
	})
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleDisputeARInvoice serves POST /v1/retailer/ar-invoices/{invoiceID}/dispute.
func (s *Service) HandleDisputeARInvoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if s == nil || s.migrationAR == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "nasiya_opening_balances_unavailable"})
		return
	}
	invoiceID := strings.TrimSpace(chi.URLParam(r, "invoiceID"))
	if invoiceID == "" {
		invoiceID = strings.TrimSpace(chi.URLParam(r, "invoiceId"))
	}
	if invoiceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invoice_id_required"})
		return
	}
	body, ok := readLimitedBody(w, r, 16*1024)
	if !ok {
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req disputeOpeningBalanceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "dispute_reason_required"})
		return
	}

	inv, err := s.migrationAR.DisputeOpeningInvoice(r.Context(), rid, invoiceID, reason)
	if err != nil {
		writeMigrationARError(w, err)
		return
	}
	if s.migrationNotifier != nil && strings.TrimSpace(inv.SupplierID) != "" {
		_ = s.migrationNotifier(
			r.Context(),
			inv.SupplierID,
			string(auth.RoleAdmin),
			"AR_MIGRATION_DISPUTED",
			"Retailer disputed migrated Nasiya balance",
			"Invoice "+inv.InvoiceID+" disputed by retailer "+rid+": "+reason,
			"/nasiya?tab=disputes&invoiceId="+inv.InvoiceID,
		)
	}
	respBytes, _ := json.Marshal(map[string]any{
		"retailer_id": rid,
		"invoice":     inv,
	})
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

func writeMigrationARError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not_found"), strings.Contains(msg, "not found"):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "invoice_not_found"})
	case strings.Contains(msg, "forbidden"), strings.Contains(msg, "mismatch"), strings.Contains(msg, "unauthorized"):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invoice_forbidden"})
	case strings.Contains(msg, "reason"):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "dispute_reason_required"})
	case strings.Contains(msg, "unavailable"):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "nasiya_opening_balances_unavailable"})
	case strings.Contains(msg, "settled"), strings.Contains(msg, "void"), strings.Contains(msg, "conflict"), strings.Contains(msg, "already"):
		writeJSON(w, http.StatusConflict, map[string]string{"error": msg})
	default:
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": msg})
	}
}
