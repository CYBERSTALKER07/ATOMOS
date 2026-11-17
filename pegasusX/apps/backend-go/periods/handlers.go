package periods

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type openPeriodHTTPReq struct {
	EntityID string `json:"entity_id,omitempty"`
	PeriodID string `json:"period_id"`
	Timezone string `json:"timezone,omitempty"`
}

type periodEntityHTTPReq struct {
	EntityID string `json:"entity_id,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

type reopenPeriodHTTPReq struct {
	EntityID     string `json:"entity_id,omitempty"`
	Reason       string `json:"reason"`
	ApprovalID   string `json:"approval_id,omitempty"`
	CoApproverID string `json:"co_approver_id,omitempty"`
}

// HandleListPeriods serves GET /v1/finance/periods?entity=&status=.
func (s *Service) HandleListPeriods(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "periods_unavailable", "periods service unavailable")
		return
	}
	entityID, ok := resolvePeriodEntityScope(w, r, r.URL.Query().Get("entity"))
	if !ok {
		return
	}
	list, err := s.ListPeriods(r.Context(), entityID, r.URL.Query().Get("status"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if list == nil {
		list = []Period{}
	}
	tzName, _, _ := ResolveEntityTimezone(r.Context(), entityID, "")
	writeJSON(w, http.StatusOK, map[string]any{
		"entity_id": entityID,
		"timezone":  tzName,
		"periods":   list,
	})
}

// HandleOpenPeriod serves POST /v1/finance/periods.
func (s *Service) HandleOpenPeriod(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "periods_unavailable", "periods service unavailable")
		return
	}
	if !requirePerm(w, r, auth.PermPeriodClose) {
		return
	}
	var req openPeriodHTTPReq
	if !decodeJSONBody(w, r, &req, false) {
		return
	}
	entityID, ok := resolvePeriodEntityScope(w, r, coalesceNonEmpty(req.EntityID, r.URL.Query().Get("entity")))
	if !ok {
		return
	}
	claims, _ := auth.FromContext(r.Context())
	actorID := auth.ActorFromContext(r.Context())
	actorType := auth.EffectiveActorType(claims)
	actorRole := auth.EffectiveSupplierRole(claims)

	p, err := s.OpenPeriod(r.Context(), entityID, req.PeriodID, req.Timezone, actorID, actorType, actorRole)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   entityID,
		Action:     "period.opened",
		ObjectType: "Period",
		ObjectID:   entityID + ":" + p.PeriodID,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"period": p})
}

// HandleSoftClose serves POST /v1/finance/periods/{id}/soft-close.
func (s *Service) HandleSoftClose(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "periods_unavailable", "periods service unavailable")
		return
	}
	if !requirePerm(w, r, auth.PermPeriodClose) {
		return
	}
	var req periodEntityHTTPReq
	if !decodeJSONBody(w, r, &req, true) {
		return
	}
	entityID, ok := resolvePeriodEntityScope(w, r, coalesceNonEmpty(req.EntityID, r.URL.Query().Get("entity")))
	if !ok {
		return
	}
	periodID := strings.TrimSpace(chi.URLParam(r, "id"))
	claims, _ := auth.FromContext(r.Context())
	p, err := s.SoftClose(r.Context(), entityID, periodID, req.Timezone, auth.ActorFromContext(r.Context()), auth.EffectiveActorType(claims), auth.EffectiveSupplierRole(claims))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   entityID,
		Action:     "period.soft_closed",
		ObjectType: "Period",
		ObjectID:   entityID + ":" + p.PeriodID,
	})
	writeJSON(w, http.StatusOK, map[string]any{"period": p})
}

// HandleClose serves POST /v1/finance/periods/{id}/close.
func (s *Service) HandleClose(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "periods_unavailable", "periods service unavailable")
		return
	}
	if !requirePerm(w, r, auth.PermPeriodClose) {
		return
	}
	var req periodEntityHTTPReq
	if !decodeJSONBody(w, r, &req, true) {
		return
	}
	entityID, ok := resolvePeriodEntityScope(w, r, coalesceNonEmpty(req.EntityID, r.URL.Query().Get("entity")))
	if !ok {
		return
	}
	periodID := strings.TrimSpace(chi.URLParam(r, "id"))
	claims, _ := auth.FromContext(r.Context())
	res, err := s.Close(r.Context(), entityID, periodID, req.Timezone, auth.ActorFromContext(r.Context()), auth.EffectiveActorType(claims), auth.EffectiveSupplierRole(claims))
	if err != nil {
		if errors.Is(err, ErrChecklistFailed) {
			writeJSON(w, http.StatusConflict, map[string]any{
				"code":      "period_checklist_failed",
				"error":     err.Error(),
				"message":   err.Error(),
				"checklist": res.Checklist,
			})
			return
		}
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   entityID,
		Action:     "period.closed",
		ObjectType: "Period",
		ObjectID:   entityID + ":" + res.Period.PeriodID,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"period":    res.Period,
		"checklist": res.Checklist,
		"snapshot":  res.Snapshot,
	})
}

// HandleReopen serves POST /v1/finance/periods/{id}/reopen.
func (s *Service) HandleReopen(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "periods_unavailable", "periods service unavailable")
		return
	}
	if !requirePerm(w, r, auth.PermPeriodReopen) {
		return
	}
	var req reopenPeriodHTTPReq
	if !decodeJSONBody(w, r, &req, false) {
		return
	}
	entityID, ok := resolvePeriodEntityScope(w, r, coalesceNonEmpty(req.EntityID, r.URL.Query().Get("entity")))
	if !ok {
		return
	}
	periodID := strings.TrimSpace(chi.URLParam(r, "id"))
	claims, _ := auth.FromContext(r.Context())
	res, err := s.Reopen(r.Context(), ReopenInput{
		EntityID:     entityID,
		PeriodID:     periodID,
		Reason:       req.Reason,
		ActorID:      auth.ActorFromContext(r.Context()),
		ActorType:    auth.EffectiveActorType(claims),
		ActorRole:    auth.EffectiveSupplierRole(claims),
		ApprovalID:   coalesceNonEmpty(req.ApprovalID, r.Header.Get("X-Approval-Id")),
		CoApproverID: req.CoApproverID,
		RoutePath:    r.URL.Path,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if res.PendingApproval {
		audit.SetMutationContext(r.Context(), audit.MutationContext{
			TenantID:   entityID,
			Action:     "period.reopen.approval_requested",
			ObjectType: "ApprovalRequest",
			ObjectID:   res.ApprovalRequestID,
			ReasonCode: req.Reason,
			ApprovalID: res.ApprovalRequestID,
		})
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":              "PENDING_APPROVAL",
			"code":                "approval_required",
			"approval_request_id": res.ApprovalRequestID,
			"required_approvers":  res.RequiredApprovers,
			"payload_hash":        res.PayloadHash,
		})
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   entityID,
		Action:     "period.reopened",
		ObjectType: "Period",
		ObjectID:   entityID + ":" + periodID,
		ReasonCode: req.Reason,
		ApprovalID: res.ApprovalRequestID,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"period":              res.Period,
		"approval_request_id": res.ApprovalRequestID,
	})
}

// HandleChecklist serves GET /v1/finance/periods/{id}/checklist?entity=.
func (s *Service) HandleChecklist(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "periods_unavailable", "periods service unavailable")
		return
	}
	entityID, ok := resolvePeriodEntityScope(w, r, r.URL.Query().Get("entity"))
	if !ok {
		return
	}
	periodID := strings.TrimSpace(chi.URLParam(r, "id"))
	report, err := s.EvaluateChecklist(r.Context(), entityID, periodID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// HandleSnapshot serves GET /v1/finance/periods/{id}/snapshot?entity=&format=json|csv.
func (s *Service) HandleSnapshot(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "periods_unavailable", "periods service unavailable")
		return
	}
	entityID, ok := resolvePeriodEntityScope(w, r, r.URL.Query().Get("entity"))
	if !ok {
		return
	}
	periodID := strings.TrimSpace(chi.URLParam(r, "id"))
	snap, err := s.GetSnapshot(r.Context(), entityID, periodID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "csv" {
		csvBytes := FormatSnapshotCSV(snap)
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="snapshot-%s-%s.csv"`, sanitizePathSeg(entityID), sanitizePathSeg(snap.PeriodID)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(csvBytes)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func requirePerm(w http.ResponseWriter, r *http.Request, perm string) bool {
	if !auth.TenantContextEnforced() {
		return true
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "authenticated session is required")
		return false
	}
	if auth.IsPlatformAdmin(r.Context()) {
		return true
	}
	if !auth.HasSupplierPerm(claims, perm) {
		writeJSONError(w, http.StatusForbidden, "forbidden", "missing required permission: "+perm)
		return false
	}
	return true
}

func resolvePeriodEntityScope(w http.ResponseWriter, r *http.Request, requestedEntity string) (string, bool) {
	ctx := r.Context()
	req := strings.TrimSpace(requestedEntity)
	callerTenant := ""
	if claims, ok := auth.FromContext(ctx); ok && strings.TrimSpace(claims.SupplierID) != "" {
		callerTenant = strings.TrimSpace(claims.SupplierID)
	} else if !auth.TenantContextEnforced() {
		if sid, ok := auth.ResolveSupplierID(ctx); ok {
			callerTenant = strings.TrimSpace(sid)
		}
	}

	if auth.IsPlatformAdmin(ctx) {
		target := coalesceNonEmpty(req, callerTenant)
		if target == "" {
			writeJSONError(w, http.StatusBadRequest, "entity_required", "entity_id parameter is required for platform admin")
			return "", false
		}
		return target, true
	}
	if callerTenant == "" {
		writeJSONError(w, http.StatusUnauthorized, "entity_required", "authenticated supplier entity context is required")
		return "", false
	}
	if req != "" && req != callerTenant {
		writeJSONError(w, http.StatusForbidden, "entity_scope_mismatch", "requested entity does not match authenticated supplier tenant")
		return "", false
	}
	return callerTenant, true
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any, allowEmpty bool) bool {
	if r.Body == nil {
		if allowEmpty {
			return true
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "request body is required")
		return false
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 256*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return false
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		if allowEmpty {
			return true
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "request body is required")
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(dst); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return false
	}
	return true
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		writeJSONError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, ErrPeriodNotFound), errors.Is(err, ErrSnapshotNotFound):
		writeJSONError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrPeriodAlreadyExists):
		writeJSONError(w, http.StatusConflict, "period_already_exists", err.Error())
	case errors.Is(err, ErrInvalidStatusTransition):
		writeJSONError(w, http.StatusConflict, "invalid_status_transition", err.Error())
	case IsPeriodClosedError(err):
		writeJSONError(w, http.StatusConflict, "period_closed", err.Error())
	case errors.Is(err, ErrChecklistFailed):
		writeJSONError(w, http.StatusConflict, "period_checklist_failed", err.Error())
	case errors.Is(err, ErrPayloadMismatch):
		writeJSONError(w, http.StatusConflict, "payload_hash_mismatch", err.Error())
	case errors.Is(err, ErrApprovalRequired), errors.Is(err, ErrSoDViolation):
		writeJSONError(w, http.StatusForbidden, "approval_required", err.Error())
	case errors.Is(err, ErrSnapshotWormRequired), errors.Is(err, ErrRepoUnavailable):
		writeJSONError(w, http.StatusServiceUnavailable, "periods_unavailable", err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "periods_internal_error", err.Error())
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error":   message,
		"code":    code,
		"message": message,
	})
}

func coalesceNonEmpty(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}
