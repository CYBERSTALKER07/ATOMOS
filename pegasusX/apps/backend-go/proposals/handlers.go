package proposals

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type createProposalHTTPReq struct {
	TenantID             string         `json:"tenant_id,omitempty"`
	AgentID              string         `json:"agent_id,omitempty"`
	ProposalType         string         `json:"proposal_type"`
	TargetObjectType     string         `json:"target_object_type"`
	TargetObjectID       string         `json:"target_object_id"`
	Title                string         `json:"title"`
	Summary              string         `json:"summary,omitempty"`
	Rationale            string         `json:"rationale,omitempty"`
	Payload              map[string]any `json:"payload,omitempty"`
	PayloadJSON          string         `json:"payload_json,omitempty"`
	EvidenceRefs         []EvidenceRef  `json:"evidence_refs,omitempty"`
	ConfidenceBps        int64          `json:"confidence_bps,omitempty"`
	EstimatedImpactMinor int64          `json:"estimated_impact_minor,omitempty"`
	Currency             string         `json:"currency,omitempty"`
	RoiCategory          string         `json:"roi_category,omitempty"`
	IdempotencyKey       string         `json:"idempotency_key,omitempty"`
	ExpiresInHours       int            `json:"expires_in_hours,omitempty"`
}

type approveProposalHTTPReq struct {
	TenantID          string         `json:"tenant_id,omitempty"`
	Comment           string         `json:"comment,omitempty"`
	PayloadHash       string         `json:"payload_hash,omitempty"`
	EditedPayload     map[string]any `json:"edited_payload,omitempty"`
	EditedPayloadJSON string         `json:"edited_payload_json,omitempty"`
	TOTPCode          string         `json:"totp_code,omitempty"`
	BreakGlass        bool           `json:"break_glass,omitempty"`
	BreakGlassReason  string         `json:"break_glass_reason,omitempty"`
}

type rejectProposalHTTPReq struct {
	TenantID string `json:"tenant_id,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

type executeProposalHTTPReq struct {
	TenantID    string `json:"tenant_id,omitempty"`
	PayloadHash string `json:"payload_hash,omitempty"`
}

// HandleList serves GET /v1/proposals?agent_id=&status=&limit=.
func (s *Service) HandleList(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "proposals_unavailable", "proposals service unavailable")
		return
	}
	tenantID, ok := resolveProposalTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	list, err := s.List(r.Context(), ListFilter{
		TenantID:   tenantID,
		AgentID:    r.URL.Query().Get("agent_id"),
		Status:     r.URL.Query().Get("status"),
		TargetType: r.URL.Query().Get("target_type"),
		TargetID:   r.URL.Query().Get("target_id"),
		Limit:      limit,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if list == nil {
		list = []Proposal{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"proposals": list,
		"count":     len(list),
	})
}

// HandleCreate serves POST /v1/proposals.
func (s *Service) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "proposals_unavailable", "proposals service unavailable")
		return
	}
	claims, _ := auth.FromContext(r.Context())
	if auth.IsReadOnlyAuditor(claims) {
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", "AUDITOR role is strictly read-only")
		return
	}
	var req createProposalHTTPReq
	if !decodeJSONBody(w, r, &req, false) {
		return
	}
	tenantID, ok := resolveProposalTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	idem := coalesceNonEmpty(req.IdempotencyKey, r.Header.Get("Idempotency-Key"))
	p, err := s.CreateProposal(r.Context(), CreateProposalInput{
		TenantID:             tenantID,
		AgentID:              req.AgentID,
		ProposalType:         req.ProposalType,
		TargetObjectType:     req.TargetObjectType,
		TargetObjectID:       req.TargetObjectID,
		Title:                req.Title,
		Summary:              req.Summary,
		Rationale:            req.Rationale,
		Payload:              req.Payload,
		PayloadJSON:          req.PayloadJSON,
		EvidenceRefs:         req.EvidenceRefs,
		ConfidenceBps:        req.ConfidenceBps,
		EstimatedImpactMinor: req.EstimatedImpactMinor,
		Currency:             req.Currency,
		RoiCategory:          req.RoiCategory,
		CreatedBy:            auth.ActorFromContext(r.Context()),
		ActorType:            auth.EffectiveActorType(claims),
		ApproverRole:         auth.EffectiveSupplierRole(claims),
		IdempotencyKey:       idem,
		ExpiresInHours:       req.ExpiresInHours,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "proposal.create",
		ObjectType: "Proposal",
		ObjectID:   p.ProposalID,
		ApprovalID: p.ApprovalRequestID,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"proposal": p})
}

// HandleRunAgent serves POST /v1/proposals/agents/{agentId}/run.
func (s *Service) HandleRunAgent(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "proposals_unavailable", "proposals service unavailable")
		return
	}
	claims, _ := auth.FromContext(r.Context())
	if auth.IsReadOnlyAuditor(claims) {
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", "AUDITOR role is strictly read-only")
		return
	}
	agentID := strings.TrimSpace(chi.URLParam(r, "agentId"))
	if agentID == "" {
		agentID = "ALL"
	}
	var body struct {
		TenantID string `json:"tenant_id,omitempty"`
	}
	_ = decodeJSONBody(w, r, &body, true)
	tenantID, ok := resolveProposalTenantScope(w, r, coalesceNonEmpty(body.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	list, err := s.RunAgentPass(r.Context(), tenantID, agentID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if list == nil {
		list = []Proposal{}
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "proposal.agent.run",
		ObjectType: "AgentRun",
		ObjectID:   strings.ToUpper(agentID),
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id":     tenantID,
		"agent_id":      strings.ToUpper(agentID),
		"proposals":     list,
		"created_count": len(list),
	})
}

// HandleGet serves GET /v1/proposals/{id}.
func (s *Service) HandleGet(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "proposals_unavailable", "proposals service unavailable")
		return
	}
	propID := strings.TrimSpace(chi.URLParam(r, "id"))
	if propID == "" {
		writeJSONError(w, http.StatusBadRequest, "proposal_id_required", "proposal id is required")
		return
	}
	tenantID, ok := resolveProposalTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	p, err := s.Get(r.Context(), tenantID, propID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"proposal": p})
}

// HandleApprove serves POST /v1/proposals/{id}/approve.
func (s *Service) HandleApprove(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "proposals_unavailable", "proposals service unavailable")
		return
	}
	if !requireNonAuditorNonAgent(w, r, auth.PermAgentProposalApprove) {
		return
	}
	propID := strings.TrimSpace(chi.URLParam(r, "id"))
	if propID == "" {
		writeJSONError(w, http.StatusBadRequest, "proposal_id_required", "proposal id is required")
		return
	}
	var req approveProposalHTTPReq
	if !decodeJSONBody(w, r, &req, true) {
		return
	}
	tenantID, ok := resolveProposalTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	claims, _ := auth.FromContext(r.Context())
	p, err := s.ApproveOrEditProposal(r.Context(), ApproveOrEditInput{
		TenantID:          tenantID,
		ProposalID:        propID,
		ApproverID:        auth.ActorFromContext(r.Context()),
		ActorType:         auth.EffectiveActorType(claims),
		ApproverRole:      auth.EffectiveSupplierRole(claims),
		Comment:           req.Comment,
		ExpectedHash:      req.PayloadHash,
		EditedPayload:     req.EditedPayload,
		EditedPayloadJSON: req.EditedPayloadJSON,
		MFAVerified:       claims.MFAVerified,
		TOTPCode:          req.TOTPCode,
		BreakGlass:        req.BreakGlass,
		BreakGlassReason:  req.BreakGlassReason,
		RoutePath:         r.URL.Path,
		TraceID:           claims.TraceID,
		IP:                r.RemoteAddr,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "proposal.approve",
		ObjectType: "Proposal",
		ObjectID:   p.ProposalID,
		ApprovalID: p.ApprovalRequestID,
	})
	writeJSON(w, http.StatusOK, map[string]any{"proposal": p})
}

// HandleReject serves POST /v1/proposals/{id}/reject.
func (s *Service) HandleReject(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "proposals_unavailable", "proposals service unavailable")
		return
	}
	if !requireNonAuditorNonAgent(w, r, auth.PermAgentProposalApprove) {
		return
	}
	propID := strings.TrimSpace(chi.URLParam(r, "id"))
	if propID == "" {
		writeJSONError(w, http.StatusBadRequest, "proposal_id_required", "proposal id is required")
		return
	}
	var req rejectProposalHTTPReq
	if !decodeJSONBody(w, r, &req, true) {
		return
	}
	tenantID, ok := resolveProposalTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	claims, _ := auth.FromContext(r.Context())
	p, err := s.RejectProposal(r.Context(), RejectInput{
		TenantID:     tenantID,
		ProposalID:   propID,
		ReviewerID:   auth.ActorFromContext(r.Context()),
		ActorType:    auth.EffectiveActorType(claims),
		ReviewerRole: auth.EffectiveSupplierRole(claims),
		Comment:      req.Comment,
		RoutePath:    r.URL.Path,
		TraceID:      claims.TraceID,
		IP:           r.RemoteAddr,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "proposal.reject",
		ObjectType: "Proposal",
		ObjectID:   p.ProposalID,
		ApprovalID: p.ApprovalRequestID,
	})
	writeJSON(w, http.StatusOK, map[string]any{"proposal": p})
}

// HandleExecute serves POST /v1/proposals/{id}/execute.
func (s *Service) HandleExecute(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "proposals_unavailable", "proposals service unavailable")
		return
	}
	if !requireNonAuditorNonAgent(w, r, auth.PermAgentProposalApprove) {
		return
	}
	propID := strings.TrimSpace(chi.URLParam(r, "id"))
	if propID == "" {
		writeJSONError(w, http.StatusBadRequest, "proposal_id_required", "proposal id is required")
		return
	}
	var req executeProposalHTTPReq
	if !decodeJSONBody(w, r, &req, true) {
		return
	}
	tenantID, ok := resolveProposalTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	claims, _ := auth.FromContext(r.Context())
	res, err := s.ExecuteProposal(r.Context(), ExecuteInput{
		TenantID:     tenantID,
		ProposalID:   propID,
		ExecutorID:   auth.ActorFromContext(r.Context()),
		ActorType:    auth.EffectiveActorType(claims),
		ExecutorRole: auth.EffectiveSupplierRole(claims),
		ExpectedHash: req.PayloadHash,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "proposal.execute",
		ObjectType: "Proposal",
		ObjectID:   res.Proposal.ProposalID,
		ApprovalID: res.Proposal.ApprovalRequestID,
	})
	writeJSON(w, http.StatusOK, res)
}

func requireNonAuditorNonAgent(w http.ResponseWriter, r *http.Request, perm string) bool {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "authenticated caller claims are required")
		return false
	}
	if auth.IsReadOnlyAuditor(claims) {
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", "AUDITOR role is strictly read-only")
		return false
	}
	if auth.IsAgentActor(claims) {
		writeJSONError(w, http.StatusForbidden, "agent_approval_forbidden", "AGENT actor cannot approve, reject, or execute proposals")
		return false
	}
	if perm != "" && !auth.HasSupplierPerm(claims, perm) {
		writeJSONError(w, http.StatusForbidden, "missing_permission", "missing_permission: "+perm)
		return false
	}
	return true
}

func resolveProposalTenantScope(w http.ResponseWriter, r *http.Request, requested string) (string, bool) {
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
		target := coalesceNonEmpty(req, callerTenant)
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

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any, allowEmpty bool) bool {
	if r.Body == nil {
		if allowEmpty {
			return true
		}
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
		if allowEmpty {
			return true
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "request body is required")
		return false
	}
	var generic any
	gdec := json.NewDecoder(bytes.NewReader(raw))
	gdec.UseNumber()
	if err := gdec.Decode(&generic); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return false
	}
	if err := validateNoFloats(generic); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", err.Error())
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
	case errors.Is(err, ErrFloatForbidden), errors.Is(err, ErrInvalidInput), errors.Is(err, approvals.ErrInvalidInput):
		writeJSONError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, ErrProposalNotFound), errors.Is(err, approvals.ErrRequestNotFound):
		writeJSONError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrAgentApprovalForbidden), errors.Is(err, approvals.ErrAgentApprovalForbidden):
		writeJSONError(w, http.StatusForbidden, "agent_approval_forbidden", err.Error())
	case errors.Is(err, ErrAuditorReadOnly), errors.Is(err, approvals.ErrAuditorReadOnly):
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", err.Error())
	case errors.Is(err, ErrSoDViolation), errors.Is(err, approvals.ErrSoDViolation):
		writeJSONError(w, http.StatusForbidden, "sod_violation", err.Error())
	case errors.Is(err, approvals.ErrRoleNotAuthorized), errors.Is(err, approvals.ErrMFARequired):
		writeJSONError(w, http.StatusForbidden, "approval_forbidden", err.Error())
	case errors.Is(err, ErrPayloadHashMismatch), errors.Is(err, approvals.ErrPayloadHashMismatch):
		writeJSONError(w, http.StatusConflict, "payload_hash_mismatch", err.Error())
	case errors.Is(err, ErrTargetExecutionConflict):
		writeJSONError(w, http.StatusConflict, "target_execution_conflict", err.Error())
	case errors.Is(err, ErrPreconditionFailed):
		writeJSONError(w, http.StatusConflict, "precondition_failed", err.Error())
	case errors.Is(err, ErrAlreadyExecuted):
		writeJSONError(w, http.StatusConflict, "proposal_already_executed", err.Error())
	case errors.Is(err, ErrInvalidTransition), errors.Is(err, approvals.ErrRequestAlreadyResolved):
		writeJSONError(w, http.StatusConflict, "invalid_transition", err.Error())
	case errors.Is(err, ErrRepoUnavailable):
		writeJSONError(w, http.StatusServiceUnavailable, "proposals_unavailable", err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "proposals_internal_error", err.Error())
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
