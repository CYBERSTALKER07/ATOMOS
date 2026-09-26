package ar

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// HandleListRetailerInvoices GET /v1/retailer/ar/invoices
func (s *Service) HandleListRetailerInvoices(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.Role != auth.RoleRetailer && claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	var rid string
	if claims.Role == auth.RoleAdmin {
		q := strings.TrimSpace(r.URL.Query().Get("retailer_id"))
		if q == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
			return
		}
		rid = q
	} else {
		rid = auth.ResolveRetailerOrgID(claims)
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	list, err := s.ListRetailerInvoices(r.Context(), rid, status, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invoices": list})
}

// HandleListSupplierInvoices GET /v1/supplier/ar/invoices
func (s *Service) HandleListSupplierInvoices(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	switch claims.Role {
	case auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleWarehouse:
	default:
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	sid := strings.TrimSpace(claims.SupplierID)
	if sid == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	list, err := s.ListSupplierInvoices(r.Context(), sid, status, 100)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invoices": list})
}

// HandleRunDunningOnce POST /v1/admin/ar/dunning/run-once — ops/e2e trigger.
// B5 M-P0-12: allow PLATFORM_ADMIN (route already gates ADMIN|PLATFORM_ADMIN).
func (w *DunningWorker) HandleRunDunningOnce(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(rw, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || (claims.Role != auth.RoleAdmin && claims.Role != auth.RolePlatformAdmin) {
		writeJSON(rw, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if !DunningEnabled() || !InvoicesEnabled() {
		writeJSON(rw, http.StatusOK, map[string]any{"ok": true, "skipped": true, "reason": "AR_DUNNING_ENABLED or AR_INVOICES_ENABLED off"})
		return
	}
	if err := w.RunOnce(r.Context()); err != nil {
		writeJSON(rw, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(rw, http.StatusOK, map[string]any{"ok": true})
}

// HandleDunningStatus GET /v1/admin/ar/dunning/status — G3.A ops honesty for transports + flags.
func (w *DunningWorker) HandleDunningStatus(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(rw, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || (claims.Role != auth.RoleAdmin && claims.Role != auth.RolePlatformAdmin) {
		writeJSON(rw, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	channels := TransportChannelsConfigured()
	writeJSON(rw, http.StatusOK, map[string]any{
		"ar_invoices_enabled":  InvoicesEnabled(),
		"ar_dunning_enabled":   DunningEnabled(),
		"off_app_channels":     channels,
		"off_app_configured":   len(channels) > 0,
		"in_app_notify":        w != nil && w.notify != nil,
		"note":                 "Off-app SMS/email/WhatsApp require DUNNING_*_PROVIDER + credentials; empty channels means in-app only",
	})
}

// HandleSupplierAgingSummary GET /v1/supplier/ar/aging-summary
func (s *Service) HandleSupplierAgingSummary(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	switch claims.Role {
	case auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleWarehouse:
	default:
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	sid := strings.TrimSpace(claims.SupplierID)
	if sid == "" {
		if q := strings.TrimSpace(r.URL.Query().Get("supplier_id")); q != "" && claims.Role == auth.RoleAdmin {
			sid = q
		}
	}
	if sid == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}

	summary, err := s.GetSupplierAgingSummary(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// HandleInvoiceWriteOff POST /v1/supplier/ar/invoices/{id}/write-off
func (s *Service) HandleInvoiceWriteOff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || (claims.Role != auth.RoleAdmin && claims.Role != auth.RoleWarehouseAdmin) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	invoiceID := chi.URLParam(r, "id")
	if invoiceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invoice_id_required"})
		return
	}

	var req WriteOffRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	inv, err := s.WriteOffInvoice(r.Context(), invoiceID, claims.Subject, req)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// HandleRetailerPayInvoice POST /v1/retailer/ar/invoices/{id}/pay
func (s *Service) HandleRetailerPayInvoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	invoiceID := chi.URLParam(r, "id")
	if invoiceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invoice_id_required"})
		return
	}

	var req RetailerPayInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	var retailerID string
	if claims.Role != auth.RoleAdmin {
		retailerID = auth.ResolveRetailerOrgID(claims)
	}

	idemKey := r.Header.Get("Idempotency-Key")
	inv, err := s.RetailerPayInvoice(r.Context(), retailerID, invoiceID, req, idemKey)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// HandleCheckDelinquencyLock GET /v1/retailer/ar/delinquency-lock or GET /v1/supplier/ar/retailers/{retailerId}/delinquency-lock
func (s *Service) HandleCheckDelinquencyLock(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	retailerID := chi.URLParam(r, "retailerId")
	if retailerID == "" {
		if claims.Role == auth.RoleRetailer {
			retailerID = auth.ResolveRetailerOrgID(claims)
		} else {
			retailerID = strings.TrimSpace(r.URL.Query().Get("retailer_id"))
		}
	}
	if retailerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
		return
	}

	supplierID := strings.TrimSpace(claims.SupplierID)
	if supplierID == "" {
		supplierID = strings.TrimSpace(r.URL.Query().Get("supplier_id"))
	}

	lockStatus, err := s.CheckRetailerDelinquencyLock(r.Context(), retailerID, supplierID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, lockStatus)
}
