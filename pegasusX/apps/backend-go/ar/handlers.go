package ar

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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
	rid := claims.Subject
	if claims.Role == auth.RoleAdmin {
		if q := strings.TrimSpace(r.URL.Query().Get("retailer_id")); q != "" {
			rid = q
		}
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
