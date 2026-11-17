package credit

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// writeJSON is a small helper for JSON responses.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("credit response encode failed", "err", err)
	}
}

// HandleGetRetailerProfile serves GET /v1/retailer/credit-profile.
func (s *Service) HandleGetRetailerProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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
	retailerID := auth.ResolveRetailerOrgID(claims)
	if claims.Role == auth.RoleAdmin {
		retailerID = strings.TrimSpace(r.URL.Query().Get("retailer_id"))
	}
	if retailerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
		return
	}
	if s == nil || s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credit_unavailable"})
		return
	}

	profile, found, err := s.repo.GetProfile(r.Context(), retailerID, claims.SupplierID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get credit profile failed", "err", err, "retailer_id", retailerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "credit_profile_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

// HandleListSupplierProfiles serves GET /v1/supplier/credit-profiles?status=&limit=.
// Supplier-scoped collections desk list (IDOR: claims.SupplierID only).
func (s *Service) HandleListSupplierProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	supplierID := strings.TrimSpace(claims.SupplierID)
	if supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}
	if s == nil || s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credit_unavailable"})
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	list, nextCursor, err := s.ListSupplierProfilesPage(r.Context(), supplierID, status, limit, cursor)
	if err != nil {
		if errors.Is(err, ErrInvalidCursor) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_cursor"})
			return
		}
		slog.ErrorContext(r.Context(), "list credit profiles failed", "err", err, "supplier_id", supplierID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	// Desk: status / delinquency / balance + G3 risk score (when CREDIT_SCORING_ENABLED).
	type deskRow struct {
		Profile
		UtilizationBps         int64    `json:"utilization_bps"`
		NeedsAttention         bool     `json:"needs_attention"`
		RiskScore              int64    `json:"risk_score"`
		RiskTier               RiskTier `json:"risk_tier,omitempty"`
		ScoringEnabled         bool     `json:"scoring_enabled"`
		RetailerName           string   `json:"retailer_name,omitempty"`
		DisplayName            string   `json:"display_name,omitempty"`
		TIN                    string   `json:"tin,omitempty"`
		Phone                  string   `json:"phone,omitempty"`
		PendingCreditNoteCount int64    `json:"pending_credit_note_count"`
	}
	retailerIDs := make([]string, 0, len(list))
	seenRIDs := make(map[string]struct{}, len(list))
	for _, p := range list {
		if p.RetailerID != "" {
			if _, exists := seenRIDs[p.RetailerID]; !exists {
				seenRIDs[p.RetailerID] = struct{}{}
				retailerIDs = append(retailerIDs, p.RetailerID)
			}
		}
	}
	metaByRetailer, totalPendingCreditNotes, metaErr := s.loadDeskMetadata(r.Context(), supplierID, retailerIDs)
	if metaErr != nil {
		slog.WarnContext(r.Context(), "load credit desk metadata failed", "supplier_id", supplierID, "err", metaErr)
	}

	rows := make([]deskRow, 0, len(list))
	scoringOn := ScoringEnabled()
	for _, p := range list {
		row := deskRow{Profile: p, ScoringEnabled: scoringOn}
		if p.CreditLimitMinor > 0 {
			row.UtilizationBps = (p.CurrentBalanceMinor * 10000) / p.CreditLimitMinor
		}
		row.RiskScore = p.RiskScore
		row.RiskTier = p.RiskTier
		row.NeedsAttention = p.Status == StatusFrozen || p.Status == StatusBlacklisted ||
			p.PrepayOnlyLocked || p.DelinquencyCount > 0 || p.CurrentBalanceMinor > 0
		if meta, ok := metaByRetailer[p.RetailerID]; ok {
			row.RetailerName = meta.RetailerName
			row.DisplayName = firstNonEmptyCreditStr(meta.DisplayName, meta.RetailerName)
			row.TIN = meta.TIN
			row.Phone = meta.Phone
			row.PendingCreditNoteCount = meta.PendingCreditNoteCount
		}
		rows = append(rows, row)
	}
	resp := map[string]any{
		"profiles":                  rows,
		"supplier_id":               supplierID,
		"count":                     len(rows),
		"scoring_enabled":           scoringOn,
		"scoring_version":           "g3_v1",
		"pending_credit_note_count": totalPendingCreditNotes,
	}
	if nextCursor != "" {
		resp["next_cursor"] = nextCursor
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleGetScores serves GET /v1/supplier/credit-scores?retailer_ids=a,b
func (s *Service) HandleGetScores(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	supplierID := strings.TrimSpace(claims.SupplierID)
	if supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}
	if s == nil || s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credit_unavailable"})
		return
	}
	if !ScoringEnabled() {
		writeJSON(w, http.StatusOK, map[string]any{
			"scores": map[string]any{}, "scoring_enabled": false, "reason": "CREDIT_SCORING_ENABLED off",
		})
		return
	}
	raw := strings.TrimSpace(r.URL.Query().Get("retailer_ids"))
	ids := []string{}
	if raw != "" {
		for _, p := range strings.Split(raw, ",") {
			if id := strings.TrimSpace(p); id != "" {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		// All desk profiles for supplier.
		list, err := s.ListSupplierProfiles(r.Context(), supplierID, "", 100)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
		for _, p := range list {
			ids = append(ids, p.RetailerID)
		}
	}
	scores, err := s.GetScoresForRetailers(r.Context(), supplierID, ids)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"scores":          scores,
		"scoring_enabled": true,
		"scoring_version": "g3_v1",
		"count":           len(scores),
	})
}

// HandleUpsertSupplierProfile serves PATCH /v1/supplier/retailer-credit-profile.
func (s *Service) HandleUpsertSupplierProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if !requireCreditWritePerm(w, claims, auth.PermCreditLimitChange) {
		return
	}

	supplierID := strings.TrimSpace(claims.SupplierID)
	if supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}

	var req struct {
		RetailerID       string `json:"retailer_id"`
		CreditLimitMinor *int64 `json:"credit_limit_minor,omitempty"`
		Status           Status `json:"status,omitempty"`
		PrepayOnlyLocked *bool  `json:"prepay_only_locked,omitempty"`
		Reason           string `json:"reason,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.RetailerID = strings.TrimSpace(req.RetailerID)
	if req.RetailerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
		return
	}
	if req.CreditLimitMinor != nil && *req.CreditLimitMinor < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "credit_limit_minor_non_negative"})
		return
	}
	if req.Status != "" && !req.Status.Valid() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_status"})
		return
	}
	if s == nil || s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credit_unavailable"})
		return
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" && req.PrepayOnlyLocked != nil && req.Status == "" && req.CreditLimitMinor == nil {
		if *req.PrepayOnlyLocked {
			reason = "prepay_only_locked"
		} else {
			reason = "prepay_only_unlocked"
		}
	}

	updated, err := s.patchSupplierProfile(r.Context(), supplierID, req.RetailerID, req.CreditLimitMinor, req.Status, req.PrepayOnlyLocked, claims.Subject, reason)
	if err != nil {
		slog.ErrorContext(r.Context(), "upsert credit profile failed", "err", err, "retailer_id", req.RetailerID)
		writeJSON(w, http.StatusConflict, map[string]string{"error": "upsert_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":             "ok",
		"profile_status":     string(updated.Status),
		"prepay_only_locked": updated.PrepayOnlyLocked,
	})
}

// HandleSetPrepayOnlyLock serves POST /v1/supplier/credit-relationships/{retailerId}/prepay-lock.
func (s *Service) HandleSetPrepayOnlyLock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || (claims.Role != auth.RoleAdmin && claims.Role != auth.RoleWarehouseAdmin) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if !requireCreditWritePerm(w, claims, auth.PermCreditLimitChange) {
		return
	}
	supplierID := strings.TrimSpace(claims.SupplierID)
	if supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}
	retailerID := strings.TrimSpace(chi.URLParam(r, "retailerId"))
	if retailerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
		return
	}
	var req struct {
		Locked           *bool  `json:"locked,omitempty"`
		PrepayOnlyLocked *bool  `json:"prepay_only_locked,omitempty"`
		Reason           string `json:"reason,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	var lockPtr *bool
	if req.PrepayOnlyLocked != nil {
		lockPtr = req.PrepayOnlyLocked
	} else if req.Locked != nil {
		lockPtr = req.Locked
	}
	if lockPtr == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "prepay_only_locked_required"})
		return
	}
	if s == nil || s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credit_unavailable"})
		return
	}
	updated, err := s.SetPrepayOnlyLocked(r.Context(), supplierID, retailerID, *lockPtr, claims.Subject, strings.TrimSpace(req.Reason))
	if err != nil {
		slog.ErrorContext(r.Context(), "set prepay_only_locked failed", "err", err, "retailer_id", retailerID)
		writeJSON(w, http.StatusConflict, map[string]string{"error": "prepay_lock_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":             "ok",
		"retailer_id":        retailerID,
		"profile_status":     string(updated.Status),
		"prepay_only_locked": updated.PrepayOnlyLocked,
	})
}

func requireCreditWritePerm(w http.ResponseWriter, claims auth.Claims, perm string) bool {
	if auth.IsReadOnlyAuditor(claims) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "auditor_read_only", "code": "auditor_read_only"})
		return false
	}
	if auth.IsAgentActor(claims) && auth.IsAgentForbiddenPerm(perm) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "agent_approval_forbidden", "code": "agent_approval_forbidden"})
		return false
	}
	if (claims.SupplierRole != "" || len(claims.SupplierPerms) > 0) && !auth.HasSupplierPerm(claims, perm) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "missing_permission", "code": "missing_permission", "permission": perm})
		return false
	}
	return true
}
