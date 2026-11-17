package approvals

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

type createPolicyHTTPReq struct {
	TenantID      string         `json:"tenant_id,omitempty"`
	PolicyID      string         `json:"policy_id,omitempty"`
	Name          string         `json:"name"`
	ObjectType    string         `json:"object_type"`
	Action        string         `json:"action"`
	ConditionExpr string         `json:"condition_expr"`
	Steps         []ApprovalStep `json:"steps"`
	Enabled       *bool          `json:"enabled,omitempty"`
}

type updatePolicyHTTPReq struct {
	TenantID      string          `json:"tenant_id,omitempty"`
	Name          *string         `json:"name,omitempty"`
	ObjectType    *string         `json:"object_type,omitempty"`
	Action        *string         `json:"action,omitempty"`
	ConditionExpr *string         `json:"condition_expr,omitempty"`
	Steps         *[]ApprovalStep `json:"steps,omitempty"`
	Enabled       *bool           `json:"enabled,omitempty"`
}

type decideHTTPReq struct {
	TenantID         string `json:"tenant_id,omitempty"`
	Comment          string `json:"comment,omitempty"`
	PayloadHash      string `json:"payload_hash,omitempty"`
	TOTPCode         string `json:"totp_code,omitempty"`
	BreakGlass       bool   `json:"break_glass,omitempty"`
	BreakGlassReason string `json:"break_glass_reason,omitempty"`
}

type createDelegationHTTPReq struct {
	TenantID    string `json:"tenant_id,omitempty"`
	DelegatorID string `json:"delegator_id,omitempty"`
	DelegateID  string `json:"delegate_id"`
	RoleScope   string `json:"role_scope,omitempty"`
	StartsAt    string `json:"starts_at,omitempty"`
	EndsAt      string `json:"ends_at"`
	Reason      string `json:"reason"`
}

type submitRequestHTTPReq struct {
	TenantID         string         `json:"tenant_id,omitempty"`
	ObjectType       string         `json:"object_type"`
	ObjectID         string         `json:"object_id"`
	Action           string         `json:"action"`
	AmountMinor      int64          `json:"amount_minor"`
	Context          map[string]any `json:"context,omitempty"`
	Payload          map[string]any `json:"payload,omitempty"`
	IdempotencyKey   string         `json:"idempotency_key,omitempty"`
	MinApprovers     int64          `json:"min_approvers,omitempty"`
	ForceCreate      bool           `json:"force_create,omitempty"`
	BreakGlass       bool           `json:"break_glass,omitempty"`
	BreakGlassReason string         `json:"break_glass_reason,omitempty"`
	ExpiresInHours   int            `json:"expires_in_hours,omitempty"`
}

// HandleListPolicies serves GET /v1/approvals/policies.
func (s *Service) HandleListPolicies(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "approvals_unavailable", "approvals service unavailable")
		return
	}
	tenantID, ok := resolveApprovalTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	policies, err := s.ListPolicies(r.Context(), tenantID, r.URL.Query().Get("object_type"), r.URL.Query().Get("action"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if policies == nil {
		policies = []Policy{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"policies":  policies,
		"count":     len(policies),
	})
}

// HandleCreatePolicy serves POST /v1/approvals/policies.
func (s *Service) HandleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "approvals_unavailable", "approvals service unavailable")
		return
	}
	if !requireNonAuditorNonAgent(w, r, auth.PermConfigManage) {
		return
	}
	var req createPolicyHTTPReq
	if !decodeJSONBody(w, r, &req) {
		return
	}
	tenantID, ok := resolveApprovalTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	pol, err := s.CreatePolicy(r.Context(), CreatePolicyInput{
		TenantID:      tenantID,
		PolicyID:      req.PolicyID,
		Name:          req.Name,
		ObjectType:    req.ObjectType,
		Action:        req.Action,
		ConditionExpr: req.ConditionExpr,
		Steps:         req.Steps,
		Enabled:       req.Enabled,
		CreatedBy:     auth.ActorFromContext(r.Context()),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "approvals.policy.create",
		ObjectType: "ApprovalPolicy",
		ObjectID:   pol.PolicyID,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"policy": pol})
}

// HandleUpdatePolicy serves PUT/PATCH /v1/approvals/policies/{id}.
func (s *Service) HandleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "approvals_unavailable", "approvals service unavailable")
		return
	}
	if !requireNonAuditorNonAgent(w, r, auth.PermConfigManage) {
		return
	}
	policyID := strings.TrimSpace(chi.URLParam(r, "id"))
	if policyID == "" {
		writeJSONError(w, http.StatusBadRequest, "policy_id_required", "policy id is required")
		return
	}
	var req updatePolicyHTTPReq
	if !decodeJSONBody(w, r, &req) {
		return
	}
	tenantID, ok := resolveApprovalTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	pol, err := s.UpdatePolicy(r.Context(), UpdatePolicyInput{
		TenantID:      tenantID,
		PolicyID:      policyID,
		Name:          req.Name,
		ObjectType:    req.ObjectType,
		Action:        req.Action,
		ConditionExpr: req.ConditionExpr,
		Steps:         req.Steps,
		Enabled:       req.Enabled,
		UpdatedBy:     auth.ActorFromContext(r.Context()),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "approvals.policy.update",
		ObjectType: "ApprovalPolicy",
		ObjectID:   pol.PolicyID,
	})
	writeJSON(w, http.StatusOK, map[string]any{"policy": pol})
}

// HandleSubmitRequest serves POST /v1/approvals/requests.
func (s *Service) HandleSubmitRequest(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "approvals_unavailable", "approvals service unavailable")
		return
	}
	claims, _ := auth.FromContext(r.Context())
	if auth.IsReadOnlyAuditor(claims) {
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", "AUDITOR role is strictly read-only")
		return
	}
	var req submitRequestHTTPReq
	if !decodeJSONBody(w, r, &req) {
		return
	}
	tenantID, ok := resolveApprovalTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	idem := coalesceNonEmpty(req.IdempotencyKey, r.Header.Get("Idempotency-Key"), r.Header.Get("X-Idempotency-Key"))
	res, err := s.SubmitRequest(r.Context(), SubmitRequestInput{
		TenantID:         tenantID,
		ObjectType:       req.ObjectType,
		ObjectID:         req.ObjectID,
		Action:           req.Action,
		AmountMinor:      req.AmountMinor,
		RequestedBy:      auth.ActorFromContext(r.Context()),
		ActorType:        auth.EffectiveActorType(claims),
		ApproverRole:     auth.EffectiveSupplierRole(claims),
		Context:          req.Context,
		Payload:          req.Payload,
		IdempotencyKey:   idem,
		MinApprovers:     req.MinApprovers,
		ForceCreate:      req.ForceCreate,
		BreakGlass:       req.BreakGlass,
		BreakGlassReason: req.BreakGlassReason,
		MFAVerified:      claims.MFAVerified,
		ExpiresInHours:   req.ExpiresInHours,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if res.Request != nil {
		audit.SetMutationContext(r.Context(), audit.MutationContext{
			TenantID:   tenantID,
			Action:     "approvals.request.submit",
			ObjectType: "ApprovalRequest",
			ObjectID:   res.Request.RequestID,
			ApprovalID: res.Request.RequestID,
		})
		writeJSON(w, http.StatusCreated, res)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// HandleInbox serves GET /v1/approvals/inbox.
func (s *Service) HandleInbox(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "approvals_unavailable", "approvals service unavailable")
		return
	}
	tenantID, ok := resolveApprovalTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 200 {
			writeJSONError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 200")
			return
		}
		limit = n
	}
	var bgPtr *bool
	if rawBG := strings.TrimSpace(r.URL.Query().Get("break_glass")); rawBG != "" {
		b, err := strconv.ParseBool(rawBG)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_break_glass", "break_glass must be true or false")
			return
		}
		bgPtr = &b
	}
	reqs, err := s.ListInbox(r.Context(), InboxFilter{
		TenantID:   tenantID,
		Status:     r.URL.Query().Get("status"),
		ObjectType: r.URL.Query().Get("object_type"),
		Action:     r.URL.Query().Get("action"),
		BreakGlass: bgPtr,
		Limit:      limit,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if reqs == nil {
		reqs = []ApprovalRequest{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"requests":  reqs,
		"count":     len(reqs),
	})
}

// HandleGetRequest serves GET /v1/approvals/{id}.
func (s *Service) HandleGetRequest(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "approvals_unavailable", "approvals service unavailable")
		return
	}
	reqID := strings.TrimSpace(chi.URLParam(r, "id"))
	if reqID == "" {
		writeJSONError(w, http.StatusBadRequest, "request_id_required", "request id is required")
		return
	}
	tenantID, ok := resolveApprovalTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	detail, err := s.GetRequest(r.Context(), tenantID, reqID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if detail.Decisions == nil {
		detail.Decisions = []ApprovalDecision{}
	}
	writeJSON(w, http.StatusOK, detail)
}

// HandleApprove serves POST /v1/approvals/{id}/approve.
func (s *Service) HandleApprove(w http.ResponseWriter, r *http.Request) {
	s.handleDecision(w, r, DecisionApprove)
}

// HandleReject serves POST /v1/approvals/{id}/reject.
func (s *Service) HandleReject(w http.ResponseWriter, r *http.Request) {
	s.handleDecision(w, r, DecisionReject)
}

func (s *Service) handleDecision(w http.ResponseWriter, r *http.Request, decision string) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "approvals_unavailable", "approvals service unavailable")
		return
	}
	reqID := strings.TrimSpace(chi.URLParam(r, "id"))
	if reqID == "" {
		writeJSONError(w, http.StatusBadRequest, "request_id_required", "request id is required")
		return
	}
	var req decideHTTPReq
	if r.Body != nil {
		body, err := io.ReadAll(io.LimitReader(r.Body, 128*1024))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
			return
		}
		defer r.Body.Close()
		if len(strings.TrimSpace(string(body))) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
				return
			}
		}
	}

	claims, hasClaims := auth.FromContext(r.Context())
	if !hasClaims {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "authenticated caller claims are required")
		return
	}
	if auth.IsReadOnlyAuditor(claims) {
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", "AUDITOR role is strictly read-only")
		return
	}
	if auth.IsAgentActor(claims) {
		writeJSONError(w, http.StatusForbidden, "agent_approval_forbidden", "AGENT actor cannot approve or reject requests")
		return
	}

	tenantID, ok := resolveApprovalTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}

	approverID := auth.ActorFromContext(r.Context())
	actorType := auth.EffectiveActorType(claims)
	approverRole := auth.EffectiveSupplierRole(claims)

	res, err := s.Decide(r.Context(), DecideInput{
		TenantID:         tenantID,
		RequestID:        reqID,
		ApproverID:       approverID,
		ActorType:        actorType,
		ApproverRole:     approverRole,
		SupplierPerms:    claims.SupplierPerms,
		Decision:         decision,
		Comment:          req.Comment,
		ExpectedHash:     req.PayloadHash,
		MFAVerified:      claims.MFAVerified,
		TOTPCode:         req.TOTPCode,
		BreakGlass:       req.BreakGlass,
		BreakGlassReason: req.BreakGlassReason,
		RoutePath:        r.URL.Path,
		TraceID:          strings.TrimSpace(r.Header.Get("X-Trace-Id")),
		IP:               r.RemoteAddr,
		DeviceID:         strings.TrimSpace(r.Header.Get("X-Device-Id")),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "approvals." + strings.ToLower(decision),
		ObjectType: "ApprovalRequest",
		ObjectID:   reqID,
		ApprovalID: reqID,
	})
	writeJSON(w, http.StatusOK, res)
}

// HandleListDelegations serves GET /v1/approvals/delegations.
func (s *Service) HandleListDelegations(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "approvals_unavailable", "approvals service unavailable")
		return
	}
	tenantID, ok := resolveApprovalTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	dels, err := s.ListDelegations(r.Context(), tenantID, r.URL.Query().Get("actor_id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if dels == nil {
		dels = []ApprovalDelegation{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id":   tenantID,
		"delegations": dels,
		"count":       len(dels),
	})
}

// HandleCreateDelegation serves POST /v1/approvals/delegations.
func (s *Service) HandleCreateDelegation(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "approvals_unavailable", "approvals service unavailable")
		return
	}
	if !requireNonAuditorNonAgent(w, r, "") {
		return
	}
	var req createDelegationHTTPReq
	if !decodeJSONBody(w, r, &req) {
		return
	}
	tenantID, ok := resolveApprovalTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	claims, _ := auth.FromContext(r.Context())
	delegatorID := auth.ActorFromContext(r.Context())
	if strings.TrimSpace(req.DelegatorID) != "" && (auth.IsPlatformAdmin(r.Context()) || auth.EffectiveSupplierRole(claims) == auth.SupplierRoleOwner) {
		delegatorID = strings.TrimSpace(req.DelegatorID)
	}
	var startsAt time.Time
	if strings.TrimSpace(req.StartsAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartsAt))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_starts_at", "starts_at must be RFC3339")
			return
		}
		startsAt = t.UTC()
	}
	if strings.TrimSpace(req.EndsAt) == "" {
		writeJSONError(w, http.StatusBadRequest, "ends_at_required", "ends_at (RFC3339) is required")
		return
	}
	endsAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EndsAt))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_ends_at", "ends_at must be RFC3339")
		return
	}
	del, err := s.CreateDelegation(r.Context(), CreateDelegationInput{
		TenantID:    tenantID,
		DelegatorID: delegatorID,
		DelegateID:  req.DelegateID,
		RoleScope:   req.RoleScope,
		StartsAt:    startsAt,
		EndsAt:      endsAt.UTC(),
		Reason:      req.Reason,
		ActorType:   auth.EffectiveActorType(claims),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "approvals.delegation.create",
		ObjectType: "ApprovalDelegation",
		ObjectID:   del.DelegationID,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"delegation": del})
}

// HandleRevokeDelegation serves POST /v1/approvals/delegations/{id}/revoke.
func (s *Service) HandleRevokeDelegation(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "approvals_unavailable", "approvals service unavailable")
		return
	}
	if !requireNonAuditorNonAgent(w, r, "") {
		return
	}
	delID := strings.TrimSpace(chi.URLParam(r, "id"))
	if delID == "" {
		writeJSONError(w, http.StatusBadRequest, "delegation_id_required", "delegation id is required")
		return
	}
	tenantID, ok := resolveApprovalTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	del, err := s.RevokeDelegation(r.Context(), tenantID, delID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "approvals.delegation.revoke",
		ObjectType: "ApprovalDelegation",
		ObjectID:   del.DelegationID,
	})
	writeJSON(w, http.StatusOK, map[string]any{"delegation": del})
}

func requireNonAuditorNonAgent(w http.ResponseWriter, r *http.Request, perm string) bool {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return true
	}
	if auth.IsReadOnlyAuditor(claims) {
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", "AUDITOR role is strictly read-only")
		return false
	}
	if auth.IsAgentActor(claims) {
		writeJSONError(w, http.StatusForbidden, "agent_approval_forbidden", "AGENT actor cannot modify approval governance")
		return false
	}
	if perm != "" && !auth.HasSupplierPerm(claims, perm) {
		writeJSONError(w, http.StatusForbidden, "missing_permission", "missing_permission: "+perm)
		return false
	}
	return true
}

func resolveApprovalTenantScope(w http.ResponseWriter, r *http.Request, requested string) (string, bool) {
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
	case errors.Is(err, ErrInvalidCEL):
		writeJSONError(w, http.StatusBadRequest, "invalid_cel_expression", err.Error())
	case errors.Is(err, ErrFloatForbidden), errors.Is(err, ErrInvalidInput):
		writeJSONError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, ErrPolicyNotFound), errors.Is(err, ErrRequestNotFound), errors.Is(err, ErrDelegationNotFound):
		writeJSONError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrAgentApprovalForbidden):
		writeJSONError(w, http.StatusForbidden, "agent_approval_forbidden", err.Error())
	case errors.Is(err, ErrAuditorReadOnly):
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", err.Error())
	case errors.Is(err, ErrSoDViolation):
		writeJSONError(w, http.StatusForbidden, "sod_violation", err.Error())
	case errors.Is(err, ErrRoleNotAuthorized):
		writeJSONError(w, http.StatusForbidden, "approval_role_forbidden", err.Error())
	case errors.Is(err, ErrMFARequired):
		writeJSONError(w, http.StatusForbidden, "mfa_verification_required", err.Error())
	case errors.Is(err, ErrPayloadHashMismatch):
		writeJSONError(w, http.StatusConflict, "payload_hash_mismatch", err.Error())
	case errors.Is(err, ErrRequestAlreadyResolved):
		writeJSONError(w, http.StatusConflict, "request_already_resolved", err.Error())
	case errors.Is(err, ErrRequestExpired):
		writeJSONError(w, http.StatusConflict, "request_expired", err.Error())
	case errors.Is(err, ErrAlreadyDecidedByActor):
		writeJSONError(w, http.StatusConflict, "already_decided", err.Error())
	case errors.Is(err, ErrWebhookUnauthorized):
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", err.Error())
	case errors.Is(err, ErrRepoUnavailable):
		writeJSONError(w, http.StatusServiceUnavailable, "approvals_unavailable", err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "approvals_internal_error", err.Error())
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
