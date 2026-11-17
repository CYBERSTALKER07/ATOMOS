package anomalies

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type scanHTTPReq struct {
	TenantID       string   `json:"tenant_id,omitempty"`
	EntityID       string   `json:"entity_id,omitempty"`
	RuleCodes      []string `json:"rule_codes,omitempty"`
	ThresholdBps   *int64   `json:"threshold_bps,omitempty"`
	ThresholdMinor *int64   `json:"threshold_minor,omitempty"`
}

type resolveCaseHTTPReq struct {
	TenantID       string `json:"tenant_id,omitempty"`
	Status         string `json:"status"`
	ResolutionNote string `json:"resolution_note"`
	PlaybookAction string `json:"playbook_action,omitempty"`
	AssignedTo     string `json:"assigned_to,omitempty"`
	CreateProposal bool   `json:"create_proposal,omitempty"`
}

// HandleCatalogue serves GET /v1/anomalies/catalogue.
func (s *Service) HandleCatalogue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !ensureAuditorActive(w, r) {
		return
	}
	tenantID, ok := resolveTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	rules := s.Catalogue()
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"rules":     rules,
		"count":     len(rules),
	})
}

// HandleScan serves POST /v1/anomalies/scan.
func (s *Service) HandleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !ensureAuditorActive(w, r) {
		return
	}
	if !rejectAuditorOrAgentMutation(w, r) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "anomalies_unavailable", "anomalies service unavailable")
		return
	}
	var req scanHTTPReq
	if !decodeOptionalJSONBody(w, r, &req) {
		return
	}
	tenantID, ok := resolveTenantScope(w, r, nonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	actorID, actorType, actorRole := callerIdentity(r)
	res, err := s.Scan(r.Context(), ScanInput{
		TenantID:       tenantID,
		EntityID:       req.EntityID,
		RuleCodes:      req.RuleCodes,
		ThresholdBps:   req.ThresholdBps,
		ThresholdMinor: req.ThresholdMinor,
		ActorID:        actorID,
		ActorType:      actorType,
		ActorRole:      actorRole,
		TraceID:        strings.TrimSpace(r.Header.Get("X-Trace-Id")),
		IP:             strings.TrimSpace(r.RemoteAddr),
		DeviceID:       strings.TrimSpace(r.Header.Get("X-Device-Id")),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "anomalies.scan",
		ObjectType: "AnomalyCase",
		ObjectID:   tenantID,
	})
	writeJSON(w, http.StatusOK, res)
}

// HandleListCases serves GET /v1/anomalies/cases?status=&severity=&category=&rule_code=&limit=.
func (s *Service) HandleListCases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !ensureAuditorActive(w, r) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "anomalies_unavailable", "anomalies service unavailable")
		return
	}
	tenantID, ok := resolveTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	cases, err := s.ListCases(r.Context(), ListFilter{
		TenantID: tenantID,
		Status:   r.URL.Query().Get("status"),
		Severity: r.URL.Query().Get("severity"),
		Category: r.URL.Query().Get("category"),
		RuleCode: r.URL.Query().Get("rule_code"),
		Limit:    parseQueryLimit(r),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if cases == nil {
		cases = []AnomalyCase{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"cases":     cases,
		"count":     len(cases),
	})
}

// HandleResolveCase serves POST /v1/anomalies/cases/{caseID}/resolve.
func (s *Service) HandleResolveCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !ensureAuditorActive(w, r) {
		return
	}
	if !rejectAuditorOrAgentMutation(w, r) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "anomalies_unavailable", "anomalies service unavailable")
		return
	}
	var req resolveCaseHTTPReq
	if !decodeJSONBody(w, r, &req) {
		return
	}
	tenantID, ok := resolveTenantScope(w, r, nonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	caseID := strings.TrimSpace(chi.URLParam(r, "caseID"))
	actorID, actorType, actorRole := callerIdentity(r)
	res, err := s.ResolveCase(r.Context(), ResolveInput{
		TenantID:       tenantID,
		CaseID:         caseID,
		Status:         req.Status,
		ResolutionNote: req.ResolutionNote,
		PlaybookAction: req.PlaybookAction,
		AssignedTo:     req.AssignedTo,
		CreateProposal: req.CreateProposal,
		ActorID:        actorID,
		ActorType:      actorType,
		ActorRole:      actorRole,
		IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		TraceID:        strings.TrimSpace(r.Header.Get("X-Trace-Id")),
		IP:             strings.TrimSpace(r.RemoteAddr),
		DeviceID:       strings.TrimSpace(r.Header.Get("X-Device-Id")),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "anomalies.case.resolve",
		ObjectType: "AnomalyCase",
		ObjectID:   res.Case.CaseID,
		ReasonCode: res.Case.Status,
	})
	writeJSON(w, http.StatusOK, res)
}

func ensureAuditorActive(w http.ResponseWriter, r *http.Request) bool {
	if auth.IsAuditorExpiredInContext(r.Context()) {
		writeJSONError(w, http.StatusUnauthorized, "auditor_access_expired", "auditor access has expired")
		return false
	}
	if claims, ok := auth.FromContext(r.Context()); ok && auth.IsAuditorExpired(claims, time.Now().UTC()) {
		writeJSONError(w, http.StatusUnauthorized, "auditor_access_expired", "auditor access has expired")
		return false
	}
	return true
}

func rejectAuditorOrAgentMutation(w http.ResponseWriter, r *http.Request) bool {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return true
	}
	if auth.IsReadOnlyAuditor(claims) {
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", "AUDITOR role is strictly read-only")
		return false
	}
	if auth.IsAgentActor(claims) {
		writeJSONError(w, http.StatusForbidden, "agent_approval_forbidden", "AGENT actor cannot execute anomaly mutations")
		return false
	}
	return true
}

func resolveTenantScope(w http.ResponseWriter, r *http.Request, requested string) (string, bool) {
	req := strings.TrimSpace(requested)
	ctx := r.Context()

	var callerTenant string
	if t, ok := auth.TenantFromContext(ctx); ok && strings.TrimSpace(t.SupplierID) != "" {
		callerTenant = strings.TrimSpace(t.SupplierID)
	} else if !auth.TenantContextEnforced() {
		if sid, ok := auth.ResolveSupplierID(ctx); ok {
			callerTenant = strings.TrimSpace(sid)
		}
	}

	if auth.IsPlatformAdmin(ctx) {
		target := nonEmpty(req, callerTenant)
		if target == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_required", "tenant_id parameter is required for platform admin")
			return "", false
		}
		return target, true
	}
	if callerTenant == "" {
		writeJSONError(w, http.StatusUnauthorized, "tenant_required", "authenticated supplier tenant context is required")
		return "", false
	}
	if req != "" && req != callerTenant {
		writeJSONError(w, http.StatusForbidden, "tenant_scope_mismatch", "requested tenant_id does not match authenticated supplier tenant")
		return "", false
	}
	return callerTenant, true
}

func callerIdentity(r *http.Request) (actorID, actorType, actorRole string) {
	actorID = auth.ActorFromContext(r.Context())
	if c, ok := auth.FromContext(r.Context()); ok {
		actorType = auth.EffectiveActorType(c)
		actorRole = auth.EffectiveSupplierRole(c)
	}
	if actorType == "" {
		actorType = auth.ActorTypeHuman
	}
	return actorID, actorType, actorRole
}

func parseQueryLimit(r *http.Request) int {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return DefaultListLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return DefaultListLimit
	}
	return clampLimit(n)
}

func decodeOptionalJSONBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Body == nil {
		return true
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 512*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return false
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return true
	}
	return decodeAndCheckNoFloats(w, raw, dst)
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Body == nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "request body is required")
		return false
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 512*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return false
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "request body is required")
		return false
	}
	return decodeAndCheckNoFloats(w, raw, dst)
}

func decodeAndCheckNoFloats(w http.ResponseWriter, raw []byte, dst any) bool {
	var generic any
	gdec := json.NewDecoder(bytes.NewReader(raw))
	gdec.UseNumber()
	if err := gdec.Decode(&generic); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return false
	}
	if err := validateNoFloats(generic); err != nil {
		writeJSONError(w, http.StatusBadRequest, "float_forbidden", err.Error())
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
	case errors.Is(err, ErrFloatForbidden):
		writeJSONError(w, http.StatusBadRequest, "float_forbidden", err.Error())
	case errors.Is(err, ErrInvalidInput):
		writeJSONError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrForbidden):
		writeJSONError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, ErrConflict):
		writeJSONError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, ErrProposalUnavailable), errors.Is(err, ErrRepoUnavailable):
		writeJSONError(w, http.StatusServiceUnavailable, "anomalies_unavailable", err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "anomalies_internal_error", err.Error())
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
