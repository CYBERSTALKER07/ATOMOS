package recon

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// HandlePostRun handles POST /v1/finance/recon/run.
func (s *Service) HandlePostRun(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeReconError(w, http.StatusServiceUnavailable, "recon_unavailable", "reconciliation service not initialized")
		return
	}

	var req RunRequest
	if r.Body != nil {
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeReconError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}

	entityID, countryPack, err := resolveReconEntityScope(r, req.EntityID)
	if err != nil {
		writeScopeError(w, err)
		return
	}
	req.EntityID = entityID
	req.CountryPack = countryPack
	req.TriggeredBy = coalesce(strings.TrimSpace(auth.ActorFromContext(r.Context())), "admin")
	if p := strings.TrimSpace(req.PeriodID); p != "" && !IsValidPeriodID(p) {
		writeReconError(w, http.StatusBadRequest, "invalid_period", "period_id must be formatted YYYY-MM")
		return
	}

	out, err := s.RunReconciliation(r.Context(), req)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	if out.Breaks == nil {
		out.Breaks = []ReconciliationBreak{}
	}
	writeReconJSON(w, http.StatusOK, out)
}

// HandleListRuns handles GET /v1/finance/recon/runs.
func (s *Service) HandleListRuns(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeReconError(w, http.StatusServiceUnavailable, "recon_unavailable", "reconciliation service not initialized")
		return
	}
	entityID, _, err := resolveReconEntityScope(r, "")
	if err != nil {
		writeScopeError(w, err)
		return
	}

	q := r.URL.Query()
	periodID := strings.TrimSpace(q.Get("period_id"))
	if periodID != "" && !IsValidPeriodID(periodID) {
		writeReconError(w, http.StatusBadRequest, "invalid_period", "period_id must be formatted YYYY-MM")
		return
	}
	limit := parsePositiveLimit(q.Get("limit"), 50, 200)
	runs, err := s.ListRuns(r.Context(), ListRunsFilter{
		EntityID: entityID,
		PeriodID: periodID,
		Status:   strings.TrimSpace(q.Get("status")),
		Limit:    limit,
	})
	if err != nil {
		mapServiceError(w, err)
		return
	}
	if runs == nil {
		runs = []ReconciliationRun{}
	}
	writeReconJSON(w, http.StatusOK, map[string]any{
		"entity_id": entityID,
		"runs":      runs,
		"count":     len(runs),
	})
}

// HandleListBreaks handles GET /v1/finance/recon/breaks.
func (s *Service) HandleListBreaks(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeReconError(w, http.StatusServiceUnavailable, "recon_unavailable", "reconciliation service not initialized")
		return
	}
	entityID, _, err := resolveReconEntityScope(r, "")
	if err != nil {
		writeScopeError(w, err)
		return
	}

	q := r.URL.Query()
	limit := parsePositiveLimit(q.Get("limit"), 100, 500)
	breaks, err := s.ListBreaks(r.Context(), ListBreaksFilter{
		EntityID:  entityID,
		RunID:     strings.TrimSpace(q.Get("run_id")),
		Dimension: strings.TrimSpace(q.Get("dimension")),
		Status:    strings.TrimSpace(q.Get("status")),
		Severity:  strings.TrimSpace(q.Get("severity")),
		Limit:     limit,
	})
	if err != nil {
		mapServiceError(w, err)
		return
	}
	if breaks == nil {
		breaks = []ReconciliationBreak{}
	}
	writeReconJSON(w, http.StatusOK, map[string]any{
		"entity_id": entityID,
		"breaks":    breaks,
		"count":     len(breaks),
	})
}

// HandlePatchBreak handles PATCH /v1/finance/recon/breaks/{id}.
func (s *Service) HandlePatchBreak(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeReconError(w, http.StatusServiceUnavailable, "recon_unavailable", "reconciliation service not initialized")
		return
	}
	breakID := strings.TrimSpace(chi.URLParam(r, "id"))
	if breakID == "" {
		writeReconError(w, http.StatusBadRequest, "invalid_request", "break id is required")
		return
	}

	var req PatchBreakRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeReconError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	entityID, _, err := resolveReconEntityScope(r, "")
	if err != nil {
		writeScopeError(w, err)
		return
	}
	req.ActorID = coalesce(strings.TrimSpace(auth.ActorFromContext(r.Context())), "admin")

	updated, err := s.PatchBreak(r.Context(), entityID, breakID, req)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeReconJSON(w, http.StatusOK, map[string]any{"break": updated})
}

func resolveReconEntityScope(r *http.Request, bodyEntity string) (string, string, error) {
	ctx := r.Context()
	claims, hasClaims := auth.FromContext(ctx)
	tenant, hasTenant := auth.TenantFromContext(ctx)
	if !hasClaims && !hasTenant {
		return "", "", fmt.Errorf("%w: missing authentication context", ErrForbidden)
	}
	if !hasClaims {
		return "", "", fmt.Errorf("%w: claims required", ErrForbidden)
	}

	role := strings.ToUpper(strings.TrimSpace(string(claims.Role)))
	if role != string(auth.RoleAdmin) && role != string(auth.RolePlatformAdmin) {
		return "", "", fmt.Errorf("%w: admin role required", ErrForbidden)
	}

	countryPack := strings.ToUpper(strings.TrimSpace(claims.MarketCode))
	requested := strings.TrimSpace(coalesce(bodyEntity, r.URL.Query().Get("entity"), r.URL.Query().Get("entity_id")))

	callerSupplier := ""
	if hasTenant && strings.TrimSpace(tenant.SupplierID) != "" {
		callerSupplier = strings.TrimSpace(tenant.SupplierID)
	} else if !auth.TenantContextEnforced() {
		if sid, ok := auth.ResolveSupplierID(ctx); ok {
			callerSupplier = strings.TrimSpace(sid)
		}
	}

	if auth.IsPlatformAdmin(ctx) {
		if requested == "" {
			requested = callerSupplier
		}
		if requested == "" {
			return "", "", fmt.Errorf("%w: entity query or entity_id field is required", ErrInvalidInput)
		}
		return requested, countryPack, nil
	}

	if callerSupplier == "" {
		return "", "", fmt.Errorf("%w: missing tenant supplier scope", ErrForbidden)
	}
	if requested != "" && requested != callerSupplier {
		return "", "", fmt.Errorf("%w: cross-tenant reconciliation access denied", ErrForbidden)
	}
	return callerSupplier, countryPack, nil
}

func writeScopeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrInvalidInput) {
		writeReconError(w, http.StatusBadRequest, "entity_required", err.Error())
		return
	}
	writeReconError(w, http.StatusForbidden, "forbidden", err.Error())
}

func mapServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeReconError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrInvalidTransition):
		writeReconError(w, http.StatusConflict, "invalid_transition", err.Error())
	case errors.Is(err, ErrInvalidInput):
		writeReconError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, ErrForbidden):
		writeReconError(w, http.StatusForbidden, "forbidden", err.Error())
	default:
		writeReconError(w, http.StatusInternalServerError, "internal_error", err.Error())
	}
}

func parsePositiveLimit(raw string, def, max int) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v <= 0 {
		return def
	}
	if v > max {
		return max
	}
	return v
}

func writeReconJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeReconError(w http.ResponseWriter, status int, code, msg string) {
	writeReconJSON(w, status, map[string]string{
		"error":   code,
		"message": msg,
	})
}
