package credit

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// writeJSON is a small helper for JSON responses.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
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
	retailerID := claims.Subject
	if claims.Role == auth.RoleAdmin {
		retailerID = strings.TrimSpace(r.URL.Query().Get("retailer_id"))
	}
	if retailerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
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

	var req struct {
		RetailerID       string   `json:"retailer_id"`
		CreditLimitMinor int64    `json:"credit_limit_minor"`
		RiskTier         RiskTier `json:"risk_tier,omitempty"`
		Status           Status   `json:"status,omitempty"`
		Reason           string   `json:"reason,omitempty"`
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
	if req.Status != "" && !req.Status.Valid() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_status"})
		return
	}

	existing, found, err := s.repo.GetProfile(r.Context(), req.RetailerID, claims.SupplierID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get credit profile failed", "err", err, "retailer_id", req.RetailerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	p := Profile{
		RetailerID:       req.RetailerID,
		SupplierID:       claims.SupplierID,
		CreditLimitMinor: req.CreditLimitMinor,
	}
	if found {
		p = existing
		if req.CreditLimitMinor > 0 || req.CreditLimitMinor == 0 {
			p.CreditLimitMinor = req.CreditLimitMinor
		}
		if req.Status != "" {
			p.Status = req.Status
		}
		if req.RiskTier != "" {
			p.RiskTier = req.RiskTier
		}
	} else {
		p.Status = StatusActive
		if req.Status != "" {
			p.Status = req.Status
		}
		if req.RiskTier != "" {
			p.RiskTier = req.RiskTier
		}
	}
	p.RiskTier = s.EvaluateRisk(p.DelinquencyCount, p.CurrentBalanceMinor, p.CreditLimitMinor)

	if err := s.UpsertProfile(r.Context(), p, claims.Subject, req.Reason); err != nil {
		slog.ErrorContext(r.Context(), "upsert credit profile failed", "err", err, "retailer_id", req.RetailerID)
		writeJSON(w, http.StatusConflict, map[string]string{"error": "upsert_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
