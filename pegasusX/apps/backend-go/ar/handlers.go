package ar

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("ar response encode failed", "err", err)
	}
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
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	list, nextCursor, err := s.ListRetailerInvoicesPage(r.Context(), rid, status, limit, cursor)
	if err != nil {
		if errors.Is(err, ErrInvalidCursor) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_cursor"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	resp := map[string]any{"invoices": list}
	if nextCursor != "" {
		resp["next_cursor"] = nextCursor
	}
	writeJSON(w, http.StatusOK, resp)
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
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	list, nextCursor, err := s.ListSupplierInvoicesPage(r.Context(), sid, status, limit, cursor)
	if err != nil {
		if errors.Is(err, ErrInvalidCursor) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_cursor"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	resp := map[string]any{"invoices": list}
	if nextCursor != "" {
		resp["next_cursor"] = nextCursor
	}
	writeJSON(w, http.StatusOK, resp)
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
	existingInv, err := s.repo.GetByID(r.Context(), invoiceID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "invoice_not_found"})
		return
	}
	supplierID := strings.TrimSpace(claims.SupplierID)
	if supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}
	if existingInv.SupplierID != supplierID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	var req WriteOffRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	}

	inv, err := s.WriteOffInvoice(r.Context(), invoiceID, claims.Subject, req)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
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
	if claims.Role != auth.RoleRetailer && claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
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
	if claims.Role == auth.RoleAdmin {
		if strings.TrimSpace(claims.SupplierID) == "" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
			return
		}
		retailerID = strings.TrimSpace(r.URL.Query().Get("retailer_id"))
	} else {
		retailerID = auth.ResolveRetailerOrgID(claims)
		if retailerID == "" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
	}

	idemKey := r.Header.Get("Idempotency-Key")
	inv, err := s.RetailerPayInvoice(r.Context(), retailerID, invoiceID, req, idemKey)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
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

func resolveRetailerScopeForHandler(claims auth.Claims, r *http.Request, allowAdminEmpty bool) (retailerID string, status int, errCode string) {
	if claims.Role == auth.RoleAdmin {
		rid := strings.TrimSpace(r.URL.Query().Get("retailer_id"))
		if rid == "" && !allowAdminEmpty {
			return "", http.StatusBadRequest, "retailer_id_required"
		}
		return rid, 0, ""
	}
	rid := auth.ResolveRetailerOrgID(claims)
	if rid == "" {
		return "", http.StatusForbidden, "forbidden"
	}
	return rid, 0, ""
}

// HandleRetailerStatement GET /v1/retailer/ar/statement
func (s *Service) HandleRetailerStatement(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.Role != auth.RoleRetailer && claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	rid, stCode, errCode := resolveRetailerScopeForHandler(claims, r, false)
	if stCode != 0 {
		writeJSON(w, stCode, map[string]string{"error": errCode})
		return
	}
	supplierID := strings.TrimSpace(r.URL.Query().Get("supplier_id"))
	if claims.Role == auth.RoleAdmin && strings.TrimSpace(claims.SupplierID) != "" {
		if supplierID != "" && supplierID != strings.TrimSpace(claims.SupplierID) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		supplierID = strings.TrimSpace(claims.SupplierID)
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	stmt, err := s.GetRetailerStatement(r.Context(), rid, supplierID, status, limit, cursor)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		case errors.Is(err, ErrInvalidCursor):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_cursor"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return
	}
	writeJSON(w, http.StatusOK, stmt)
}

// HandleRetailerPayFIFO POST /v1/retailer/ar/pay-fifo
func (s *Service) HandleRetailerPayFIFO(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.Role != auth.RoleRetailer && claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	var req PayFIFORequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if idemHeader := strings.TrimSpace(r.Header.Get("Idempotency-Key")); idemHeader != "" {
		req.IdempotencyKey = idemHeader
	}
	if claims.Role == auth.RoleAdmin && strings.TrimSpace(req.RetailerID) == "" {
		req.RetailerID = strings.TrimSpace(r.URL.Query().Get("retailer_id"))
	}

	resp, err := s.ApplyPaymentFIFO(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		case errors.Is(err, ErrInvoiceNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "invoice_not_found"})
		case errors.Is(err, ErrInvoiceAlreadySettled):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "invoice_already_settled"})
		case errors.Is(err, ErrInvalidFIFORequest):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleRetailerListPromises GET /v1/retailer/ar/promises
func (s *Service) HandleRetailerListPromises(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.Role != auth.RoleRetailer && claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	rid, stCode, errCode := resolveRetailerScopeForHandler(claims, r, false)
	if stCode != 0 {
		writeJSON(w, stCode, map[string]string{"error": errCode})
		return
	}
	supplierID := strings.TrimSpace(r.URL.Query().Get("supplier_id"))
	if claims.Role == auth.RoleAdmin && strings.TrimSpace(claims.SupplierID) != "" {
		if supplierID != "" && supplierID != strings.TrimSpace(claims.SupplierID) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		supplierID = strings.TrimSpace(claims.SupplierID)
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	promises, err := s.ListPromises(r.Context(), rid, supplierID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	qKey := QuarterKey(s.now().UTC(), s.resolveSupplierTimezone(r.Context(), supplierID))
	used, err := s.IsSelfServePromiseUsed(r.Context(), rid, supplierID, qKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !used && supplierID == "" {
		for _, p := range promises {
			if p.QuarterKey == qKey && p.ApprovedBy == PromiseApprovedBySystem {
				used = true
				break
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"promises":                promises,
		"current_quarter_key":     qKey,
		"self_serve_promise_used": used,
		"max_extension_days":      MaxPromiseDays(),
	})
}

type createPromiseHTTPRequest struct {
	SupplierID    string `json:"supplier_id"`
	RetailerID    string `json:"retailer_id,omitempty"`
	InvoiceID     string `json:"invoice_id,omitempty"`
	ExtensionDays int    `json:"extension_days"`
}

func writePromiseCreateError(w http.ResponseWriter, err error) {
	var qErr *QuarterlyLimitError
	switch {
	case errors.As(err, &qErr):
		writeJSON(w, http.StatusConflict, map[string]any{
			"code":        "quarterly_promise_limit_reached",
			"error":       "quarterly_promise_limit_reached",
			"quarter_key": qErr.QuarterKey,
			"message":     qErr.Error(),
		})
	case errors.Is(err, ErrQuarterlyPromiseLimitReached):
		writeJSON(w, http.StatusConflict, map[string]any{
			"code":    "quarterly_promise_limit_reached",
			"error":   "quarterly_promise_limit_reached",
			"message": err.Error(),
		})
	case errors.Is(err, ErrPromisePrepayLocked):
		writeJSON(w, http.StatusConflict, map[string]string{
			"code":    "prepay_only_locked",
			"error":   "prepay_only_locked",
			"message": err.Error(),
		})
	case errors.Is(err, ErrPromiseProfileFrozen):
		writeJSON(w, http.StatusConflict, map[string]string{
			"code":    "credit_profile_frozen",
			"error":   "credit_profile_frozen",
			"message": err.Error(),
		})
	case errors.Is(err, ErrForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	case errors.Is(err, ErrInvoiceNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "invoice_not_found"})
	case errors.Is(err, ErrInvoiceAlreadySettled):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invoice_already_settled"})
	case errors.Is(err, ErrPromiseInvalidExtension), errors.Is(err, ErrPromiseNoBalance):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
}

// HandleRetailerCreatePromise POST /v1/retailer/ar/promises
func (s *Service) HandleRetailerCreatePromise(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.Role != auth.RoleRetailer && claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	var req createPromiseHTTPRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if strings.TrimSpace(req.SupplierID) == "" && strings.TrimSpace(claims.SupplierID) != "" {
		req.SupplierID = strings.TrimSpace(claims.SupplierID)
	}
	if strings.TrimSpace(req.SupplierID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supplier_id_required"})
		return
	}
	rid := strings.TrimSpace(req.RetailerID)
	if claims.Role == auth.RoleAdmin && rid == "" {
		rid = strings.TrimSpace(r.URL.Query().Get("retailer_id"))
	}
	promise, err := s.CreateRetailerPromise(r.Context(), rid, req.SupplierID, req.InvoiceID, req.ExtensionDays)
	if err != nil {
		writePromiseCreateError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, promise)
}

// HandleSupplierListPromises GET /v1/supplier/ar/promises
func (s *Service) HandleSupplierListPromises(w http.ResponseWriter, r *http.Request) {
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
	rid := strings.TrimSpace(r.URL.Query().Get("retailer_id"))
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	promises, err := s.ListPromises(r.Context(), rid, sid, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"promises":            promises,
		"current_quarter_key": QuarterKey(s.now().UTC(), s.resolveSupplierTimezone(r.Context(), sid)),
		"max_extension_days":  MaxPromiseDays(),
	})
}

// HandleSupplierGrantPromise POST /v1/supplier/ar/promises
func (s *Service) HandleSupplierGrantPromise(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.Role != auth.RoleAdmin && claims.Role != auth.RoleWarehouseAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	sid := strings.TrimSpace(claims.SupplierID)
	if sid == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}
	var req createPromiseHTTPRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if strings.TrimSpace(req.RetailerID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
		return
	}
	promise, err := s.GrantSupplierPromise(r.Context(), sid, req.RetailerID, req.InvoiceID, req.ExtensionDays, claims.Subject)
	if err != nil {
		writePromiseCreateError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, promise)
}
