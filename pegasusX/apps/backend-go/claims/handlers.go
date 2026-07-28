package claims

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
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

// HandleListOrderClaims serves GET /v1/orders/{orderID}/claims.
func (s *Service) HandleListOrderClaims(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if _, ok := auth.FromContext(r.Context()); !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		orderID = strings.TrimSpace(chi.URLParam(r, "id"))
	}
	list, err := s.repo.ListByOrder(r.Context(), orderID)
	if err != nil {
		slog.ErrorContext(r.Context(), "list claims failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
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
