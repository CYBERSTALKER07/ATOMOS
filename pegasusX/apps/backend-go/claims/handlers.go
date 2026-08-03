package claims

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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// HandleFileOrderClaim serves POST /v1/orders/{orderID}/claims.
func (s *Service) HandleFileOrderClaim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
		orderID = strings.TrimSpace(chi.URLParam(r, "id"))
	}
	var req FileClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	created, err := s.FileRetailerClaim(r.Context(), claims, orderID, req)
	if err != nil {
		writeClaimError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// HandleApproveClaim serves POST /v1/claims/{claimID}/approve.
func (s *Service) HandleApproveClaim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	actor, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	claimID := strings.TrimSpace(chi.URLParam(r, "claimID"))
	var req ApproveClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && r.ContentLength != 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	c, settlement, err := s.ApproveClaim(r.Context(), actor, claimID, req)
	if err != nil {
		writeClaimError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"claim":      c,
		"settlement": settlement,
	})
}

// HandleRejectClaim serves POST /v1/claims/{claimID}/reject.
func (s *Service) HandleRejectClaim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	actor, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	claimID := strings.TrimSpace(chi.URLParam(r, "claimID"))
	var req RejectClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && r.ContentLength != 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	c, err := s.RejectClaim(r.Context(), actor, claimID, req)
	if err != nil {
		writeClaimError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// HandleListSupplierClaims serves GET /v1/supplier/claims?status=&limit=.
func (s *Service) HandleListSupplierClaims(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	actor, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	status := Status(strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status"))))
	if status != "" && !status.Valid() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_status"})
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	list, err := s.ListSupplierClaims(r.Context(), actor, status, limit)
	if err != nil {
		writeClaimError(w, r, err)
		return
	}
	if list == nil {
		list = []Claim{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"claims": list})
}

// HandleListOrderClaims serves GET /v1/orders/{orderID}/claims.
// Retailers may only list claims on their own orders; admin/warehouse admin may list any.
func (s *Service) HandleListOrderClaims(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	actor, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		orderID = strings.TrimSpace(chi.URLParam(r, "id"))
	}
	list, err := s.ListOrderClaims(r.Context(), actor, orderID)
	if err != nil {
		writeClaimError(w, r, err)
		return
	}
	if list == nil {
		list = []Claim{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"claims": list})
}

func writeClaimError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrOrderNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
	case errors.Is(err, ErrClaimNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "claim_not_found"})
	case errors.Is(err, ErrForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	case errors.Is(err, ErrClaimNotAllowed):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "claim_not_allowed", "message": err.Error()})
	case errors.Is(err, ErrClaimWindowExpired):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "claim_window_expired"})
	case errors.Is(err, ErrEvidenceRequired):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "photo_evidence_required"})
	case errors.Is(err, ErrInvalidEvidenceURI):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_evidence_uri"})
	case errors.Is(err, ErrClaimStockHoldFailed):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "claim_stock_hold_failed", "message": err.Error()})
	case errors.Is(err, ErrInvalidClaimState):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_claim_state", "message": err.Error()})
	case errors.Is(err, ErrPricingFailed), errors.Is(err, ErrInvalidClaimType), errors.Is(err, ErrInvalidLineItems):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pricing_failed", "message": err.Error()})
	case errors.Is(err, ErrAlreadySettled):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "claim_already_settled"})
	default:
		slog.ErrorContext(r.Context(), "claim operation failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	}
}
