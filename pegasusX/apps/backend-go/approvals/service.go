package approvals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// OnApprovedHook is invoked post-commit when an ApprovalRequest reaches StatusApproved.
type OnApprovedHook func(ctx context.Context, req ApprovalRequest, decisions []ApprovalDecision) error

// Service orchestrates CEL policy evaluation, multi-step approval workflows,
// payload hash pinning, delegations, SoD, MFA thresholds, and break-glass controls.
type Service struct {
	repo                     Repository
	audit                    AuditRecorder
	sod                      SoDChecker
	mfa                      MFAVerifier
	telegramResolver         TelegramApproverResolver
	telegramSecret           string
	defaultMFAThresholdMinor int64
	now                      func() time.Time

	hooksMu    sync.RWMutex
	onApproved map[string]OnApprovedHook // key: objectType + ":" + action
}

// NewService constructs a Spanner-backed approvals Service (after repository_spanner.go is wired).
func NewService(client *spanner.Client) *Service {
	if client == nil {
		return &Service{
			now:        func() time.Time { return time.Now().UTC() },
			onApproved: make(map[string]OnApprovedHook),
		}
	}
	return &Service{
		repo:       NewSpannerRepository(client),
		now:        func() time.Time { return time.Now().UTC() },
		onApproved: make(map[string]OnApprovedHook),
	}
}

// NewServiceWithRepository constructs an approvals Service with a custom Repository (e.g., MemoryRepository).
func NewServiceWithRepository(repo Repository) *Service {
	return &Service{
		repo:       repo,
		now:        func() time.Time { return time.Now().UTC() },
		onApproved: make(map[string]OnApprovedHook),
	}
}

// Repo exposes the underlying Repository.
func (s *Service) Repo() Repository {
	if s == nil {
		return nil
	}
	return s.repo
}

// SetAuditRecorder attaches an immutable AuditEvents recorder.
func (s *Service) SetAuditRecorder(rec AuditRecorder) {
	if s != nil {
		s.audit = rec
	}
}

// SetSoDChecker attaches an iam.Service four-eyes checker.
func (s *Service) SetSoDChecker(sod SoDChecker) {
	if s != nil {
		s.sod = sod
	}
}

// SetMFAVerifier attaches an MFA TOTP verifier for step-up threshold checks.
func (s *Service) SetMFAVerifier(v MFAVerifier) {
	if s != nil {
		s.mfa = v
	}
}

// SetTelegramWebhook configures the Telegram webhook secret and optional chat->actor resolver.
func (s *Service) SetTelegramWebhook(secret string, resolver TelegramApproverResolver) {
	if s == nil {
		return
	}
	s.telegramSecret = strings.TrimSpace(secret)
	s.telegramResolver = resolver
}

// SetDefaultMFAThresholdMinor configures an entity-wide minor-unit threshold requiring MFA.
func (s *Service) SetDefaultMFAThresholdMinor(thresholdMinor int64) {
	if s != nil && thresholdMinor >= 0 {
		s.defaultMFAThresholdMinor = thresholdMinor
	}
}

// RegisterOnApprovedHook registers an idempotent post-approval callback for (objectType, action).
func (s *Service) RegisterOnApprovedHook(objectType, action string, fn OnApprovedHook) {
	if s == nil || fn == nil {
		return
	}
	s.hooksMu.Lock()
	defer s.hooksMu.Unlock()
	if s.onApproved == nil {
		s.onApproved = make(map[string]OnApprovedHook)
	}
	key := strings.ToLower(strings.TrimSpace(objectType)) + ":" + strings.ToLower(strings.TrimSpace(action))
	s.onApproved[key] = fn
}

func (s *Service) currentNow() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func normalizeSteps(steps []ApprovalStep) ([]ApprovalStep, error) {
	if len(steps) == 0 {
		return []ApprovalStep{{StepIndex: 0, Name: "Primary Approval", RequiredApprovers: 1}}, nil
	}
	out := make([]ApprovalStep, len(steps))
	for i, st := range steps {
		st.StepIndex = i
		st.Name = strings.TrimSpace(st.Name)
		if st.RequiredApprovers <= 0 {
			st.RequiredApprovers = 1
		}
		if st.MFAThresholdMinor < 0 {
			return nil, fmt.Errorf("%w: step %d mfa_threshold_minor must be >= 0", ErrInvalidInput, i)
		}
		if len(st.AllowedRoles) > 0 {
			normRoles := make([]string, 0, len(st.AllowedRoles))
			seen := make(map[string]struct{}, len(st.AllowedRoles))
			for _, r := range st.AllowedRoles {
				nr := strings.ToUpper(strings.TrimSpace(r))
				if nr == "" {
					continue
				}
				if _, ok := seen[nr]; ok {
					continue
				}
				seen[nr] = struct{}{}
				normRoles = append(normRoles, nr)
			}
			st.AllowedRoles = normRoles
		}
		out[i] = st
	}
	return out, nil
}

// CreatePolicy validates the CEL condition expression and stores a new ApprovalPolicy at Version 1.
func (s *Service) CreatePolicy(ctx context.Context, in CreatePolicyInput) (Policy, error) {
	if s == nil || s.repo == nil {
		return Policy{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	name := strings.TrimSpace(in.Name)
	objType := strings.TrimSpace(in.ObjectType)
	action := strings.TrimSpace(in.Action)
	expr := strings.TrimSpace(in.ConditionExpr)
	if tid == "" || name == "" || objType == "" || action == "" {
		return Policy{}, fmt.Errorf("%w: tenant_id, name, object_type, and action are required", ErrInvalidInput)
	}
	if _, err := CompileCEL(expr); err != nil {
		return Policy{}, err
	}
	steps, err := normalizeSteps(in.Steps)
	if err != nil {
		return Policy{}, err
	}
	pid := strings.TrimSpace(in.PolicyID)
	if pid == "" {
		pid = "apol_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	createdBy := strings.TrimSpace(in.CreatedBy)
	if createdBy == "" {
		createdBy = auth.ActorFromContext(ctx)
	}
	now := s.currentNow()
	pol := Policy{
		TenantID:      tid,
		PolicyID:      pid,
		Name:          name,
		ObjectType:    objType,
		Action:        action,
		ConditionExpr: expr,
		Steps:         steps,
		Version:       1,
		Enabled:       enabled,
		CreatedBy:     createdBy,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.repo.CreatePolicy(ctx, pol); err != nil {
		return Policy{}, err
	}
	return pol, nil
}

// UpdatePolicy applies updates to an existing ApprovalPolicy and increments its Version.
func (s *Service) UpdatePolicy(ctx context.Context, in UpdatePolicyInput) (Policy, error) {
	if s == nil || s.repo == nil {
		return Policy{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	pid := strings.TrimSpace(in.PolicyID)
	if tid == "" || pid == "" {
		return Policy{}, fmt.Errorf("%w: tenant_id and policy_id are required", ErrInvalidInput)
	}
	existing, ok, err := s.repo.GetPolicy(ctx, tid, pid)
	if err != nil {
		return Policy{}, err
	}
	if !ok {
		return Policy{}, ErrPolicyNotFound
	}
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" {
			return Policy{}, fmt.Errorf("%w: policy name cannot be empty", ErrInvalidInput)
		}
		existing.Name = n
	}
	if in.ObjectType != nil {
		ot := strings.TrimSpace(*in.ObjectType)
		if ot == "" {
			return Policy{}, fmt.Errorf("%w: object_type cannot be empty", ErrInvalidInput)
		}
		existing.ObjectType = ot
	}
	if in.Action != nil {
		act := strings.TrimSpace(*in.Action)
		if act == "" {
			return Policy{}, fmt.Errorf("%w: action cannot be empty", ErrInvalidInput)
		}
		existing.Action = act
	}
	if in.ConditionExpr != nil {
		expr := strings.TrimSpace(*in.ConditionExpr)
		if _, err := CompileCEL(expr); err != nil {
			return Policy{}, err
		}
		existing.ConditionExpr = expr
	}
	if in.Steps != nil {
		steps, err := normalizeSteps(*in.Steps)
		if err != nil {
			return Policy{}, err
		}
		existing.Steps = steps
	}
	if in.Enabled != nil {
		existing.Enabled = *in.Enabled
	}
	existing.Version++
	existing.UpdatedAt = s.currentNow()
	if err := s.repo.UpdatePolicy(ctx, existing); err != nil {
		return Policy{}, err
	}
	return existing, nil
}

// ListPolicies lists policies for a tenant filtered optionally by objectType and action.
func (s *Service) ListPolicies(ctx context.Context, tenantID, objectType, action string) ([]Policy, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	return s.repo.ListPolicies(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(objectType), strings.TrimSpace(action), false)
}

// GetPolicy fetches one policy by (tenantID, policyID).
func (s *Service) GetPolicy(ctx context.Context, tenantID, policyID string) (Policy, error) {
	if s == nil || s.repo == nil {
		return Policy{}, ErrRepoUnavailable
	}
	pol, ok, err := s.repo.GetPolicy(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(policyID))
	if err != nil {
		return Policy{}, err
	}
	if !ok {
		return Policy{}, ErrPolicyNotFound
	}
	return pol, nil
}

// SubmitRequest evaluates enabled CEL policies for (ObjectType, Action) and creates an ApprovalRequest
// when matched (or when ForceCreate is true).
func (s *Service) SubmitRequest(ctx context.Context, in SubmitRequestInput) (SubmitRequestResult, error) {
	if s == nil || s.repo == nil {
		return SubmitRequestResult{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	objType := strings.TrimSpace(in.ObjectType)
	objID := strings.TrimSpace(in.ObjectID)
	action := strings.TrimSpace(in.Action)
	reqBy := strings.TrimSpace(in.RequestedBy)
	if reqBy == "" {
		reqBy = auth.ActorFromContext(ctx)
	}
	actorType := strings.ToLower(strings.TrimSpace(in.ActorType))
	if actorType == "" {
		actorType = auth.ActorTypeHuman
	}
	if tid == "" || objType == "" || objID == "" || action == "" {
		return SubmitRequestResult{}, fmt.Errorf("%w: tenant_id, object_type, object_id, and action are required", ErrInvalidInput)
	}

	if idem := strings.TrimSpace(in.IdempotencyKey); idem != "" {
		if existing, found, err := s.repo.FindRequestByIdempotency(ctx, tid, idem); err != nil {
			return SubmitRequestResult{}, err
		} else if found {
			var pol *Policy
			if existing.PolicyID != "" {
				if p, ok, _ := s.repo.GetPolicy(ctx, tid, existing.PolicyID); ok {
					pol = &p
				}
			}
			return SubmitRequestResult{Required: true, Matched: existing.PolicyID != "", Request: &existing, Policy: pol}, nil
		}
	}

	var canonJSON, payloadHash string
	var err error
	if strings.TrimSpace(in.PayloadJSON) != "" {
		canonJSON, payloadHash, err = CanonicalizeJSONBytes([]byte(in.PayloadJSON))
	} else if in.Payload != nil {
		canonJSON, payloadHash, err = CanonicalizePayload(in.Payload)
	} else {
		canonJSON, payloadHash, err = CanonicalizePayload(in.Context)
	}
	if err != nil {
		return SubmitRequestResult{}, err
	}

	evalCtx := make(map[string]any, len(in.Context)+8)
	for k, v := range in.Context {
		evalCtx[k] = v
	}
	if in.Payload != nil && len(evalCtx) == 0 {
		for k, v := range in.Payload {
			evalCtx[k] = v
		}
	}
	if _, ok := evalCtx["amount_minor"]; !ok {
		evalCtx["amount_minor"] = in.AmountMinor
	}
	if _, ok := evalCtx["object_type"]; !ok {
		evalCtx["object_type"] = objType
	}
	if _, ok := evalCtx["object_id"]; !ok {
		evalCtx["object_id"] = objID
	}
	if _, ok := evalCtx["action"]; !ok {
		evalCtx["action"] = action
	}
	if _, ok := evalCtx["requested_by"]; !ok {
		evalCtx["requested_by"] = reqBy
	}
	if _, ok := evalCtx["actor_type"]; !ok {
		evalCtx["actor_type"] = actorType
	}
	if _, ok := evalCtx["tenant_id"]; !ok {
		evalCtx["tenant_id"] = tid
	}

	policies, err := s.repo.ListPolicies(ctx, tid, objType, action, true)
	if err != nil {
		return SubmitRequestResult{}, err
	}
	matched := make([]Policy, 0)
	for _, pol := range policies {
		ok, err := EvaluateCEL(pol.ConditionExpr, evalCtx)
		if err != nil {
			return SubmitRequestResult{}, err
		}
		if ok {
			matched = append(matched, pol)
		}
	}

	if len(matched) == 0 && !in.ForceCreate {
		return SubmitRequestResult{Required: false, Matched: false}, nil
	}

	var chosen *Policy
	if len(matched) > 0 {
		sort.Slice(matched, func(i, j int) bool {
			mi := maxStepApprovers(matched[i].Steps)
			mj := maxStepApprovers(matched[j].Steps)
			if mi != mj {
				return mi > mj
			}
			if len(matched[i].Steps) != len(matched[j].Steps) {
				return len(matched[i].Steps) > len(matched[j].Steps)
			}
			return matched[i].PolicyID < matched[j].PolicyID
		})
		cp := matched[0]
		chosen = &cp
	}

	reqApprovers := in.RequiredApprovers
	if reqApprovers <= 0 {
		reqApprovers = 1
	}
	if chosen != nil && len(chosen.Steps) > 0 {
		if int64(chosen.Steps[0].RequiredApprovers) > reqApprovers {
			reqApprovers = int64(chosen.Steps[0].RequiredApprovers)
		}
		if in.MinApprovers > reqApprovers && totalStepApprovers(chosen.Steps) < int(in.MinApprovers) {
			reqApprovers = in.MinApprovers
		}
	} else if in.MinApprovers > reqApprovers {
		reqApprovers = in.MinApprovers
	}

	now := s.currentNow()
	status := StatusPending
	var resolvedAt *time.Time
	breakGlass := false
	breakGlassReason := ""
	if in.BreakGlass {
		if actorType == auth.ActorTypeAgent || strings.EqualFold(strings.TrimSpace(in.ApproverRole), auth.SupplierRoleAgent) {
			return SubmitRequestResult{}, ErrAgentApprovalForbidden
		}
		if strings.EqualFold(strings.TrimSpace(in.ApproverRole), auth.SupplierRoleAuditor) {
			return SubmitRequestResult{}, ErrAuditorReadOnly
		}
		breakGlassReason = strings.TrimSpace(in.BreakGlassReason)
		if breakGlassReason == "" {
			return SubmitRequestResult{}, fmt.Errorf("%w: break_glass_reason is required for break-glass approval", ErrInvalidInput)
		}
		if !isBreakGlassRole(in.ApproverRole) {
			return SubmitRequestResult{}, ErrRoleNotAuthorized
		}
		breakGlass = true
		status = StatusApproved
		resolvedAt = &now
	}

	var expiresAt *time.Time
	timeoutHrs := in.ExpiresInHours
	if timeoutHrs <= 0 && chosen != nil && len(chosen.Steps) > 0 && chosen.Steps[0].TimeoutHours > 0 {
		timeoutHrs = chosen.Steps[0].TimeoutHours
	}
	if timeoutHrs > 0 {
		t := now.Add(time.Duration(timeoutHrs) * time.Hour)
		expiresAt = &t
	}

	var polID string
	var polVersion int64 = 1
	if chosen != nil {
		polID = chosen.PolicyID
		polVersion = chosen.Version
	}

	req := ApprovalRequest{
		TenantID:          tid,
		RequestID:         "apr_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24],
		PolicyID:          polID,
		PolicyVersion:     polVersion,
		ObjectType:        objType,
		ObjectID:          objID,
		Action:            action,
		AmountMinor:       in.AmountMinor,
		RequestedBy:       reqBy,
		ActorType:         actorType,
		PayloadJSON:       canonJSON,
		PayloadHash:       payloadHash,
		Status:            status,
		CurrentStep:       0,
		RequiredApprovers: reqApprovers,
		BreakGlass:        breakGlass,
		BreakGlassReason:  breakGlassReason,
		ExpiresAt:         expiresAt,
		IdempotencyKey:    strings.TrimSpace(in.IdempotencyKey),
		ResolvedAt:        resolvedAt,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.repo.CreateRequest(ctx, req, s.audit); err != nil {
		return SubmitRequestResult{}, err
	}

	if req.Status == StatusApproved {
		s.triggerOnApproved(ctx, req, nil)
	}

	return SubmitRequestResult{
		Required: true,
		Matched:  chosen != nil,
		Request:  &req,
		Policy:   chosen,
	}, nil
}

func totalStepApprovers(steps []ApprovalStep) int {
	total := 0
	for _, st := range steps {
		n := st.RequiredApprovers
		if n <= 0 {
			n = 1
		}
		total += n
	}
	return total
}

func maxStepApprovers(steps []ApprovalStep) int {
	m := 1
	for _, st := range steps {
		if st.RequiredApprovers > m {
			m = st.RequiredApprovers
		}
	}
	return m
}

func isBreakGlassRole(role string) bool {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case string(auth.RoleAdmin), string(auth.RolePlatformAdmin), auth.SupplierRoleOwner, auth.SupplierRoleFinanceManager:
		return true
	default:
		return false
	}
}

// GetRequest retrieves an approval request, its decisions, and its policy.
func (s *Service) GetRequest(ctx context.Context, tenantID, requestID string) (RequestDetail, error) {
	if s == nil || s.repo == nil {
		return RequestDetail{}, ErrRepoUnavailable
	}
	req, decs, ok, err := s.repo.GetRequest(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(requestID))
	if err != nil {
		return RequestDetail{}, err
	}
	if !ok {
		return RequestDetail{}, ErrRequestNotFound
	}
	var pol *Policy
	if req.PolicyID != "" {
		if p, pOK, _ := s.repo.GetPolicy(ctx, req.TenantID, req.PolicyID); pOK {
			pol = &p
		}
	}
	return RequestDetail{Request: req, Decisions: decs, Policy: pol}, nil
}

// ListInbox lists approval requests matching f.
func (s *Service) ListInbox(ctx context.Context, f InboxFilter) ([]ApprovalRequest, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	return s.repo.ListInbox(ctx, f)
}

// Decide records APPROVE or REJECT on an ApprovalRequest enforcing AGENT bans, AUDITOR bans,
// maker!=checker SoD, PayloadHash integrity, active delegations, step role checks, and MFA thresholds.
func (s *Service) Decide(ctx context.Context, in DecideInput) (DecideResult, error) {
	if s == nil || s.repo == nil {
		return DecideResult{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	reqID := strings.TrimSpace(in.RequestID)
	approverID := strings.TrimSpace(in.ApproverID)
	actorType := strings.ToLower(strings.TrimSpace(in.ActorType))
	approverRole := strings.ToUpper(strings.TrimSpace(in.ApproverRole))
	decision := strings.ToUpper(strings.TrimSpace(in.Decision))
	if decision != DecisionApprove && decision != DecisionReject {
		return DecideResult{}, fmt.Errorf("%w: decision must be APPROVE or REJECT", ErrInvalidInput)
	}
	if reqID == "" || approverID == "" {
		return DecideResult{}, fmt.Errorf("%w: request_id and approver_id are required", ErrInvalidInput)
	}
	if tid == "" {
		req, _, found, err := s.repo.GetRequestByID(ctx, reqID)
		if err != nil {
			return DecideResult{}, err
		}
		if !found {
			return DecideResult{}, ErrRequestNotFound
		}
		tid = req.TenantID
	}

	// Hard ban 1: AGENT actor cannot approve or reject anything.
	if actorType == auth.ActorTypeAgent || approverRole == auth.SupplierRoleAgent {
		s.recordSoD(ctx, tid, approverID, actorType, approverRole, decision, reqID, "", in)
		return DecideResult{}, ErrAgentApprovalForbidden
	}
	// Hard ban 2: AUDITOR is strictly read-only.
	if approverRole == auth.SupplierRoleAuditor {
		s.recordSoD(ctx, tid, approverID, actorType, approverRole, decision, reqID, "", in)
		return DecideResult{}, ErrAuditorReadOnly
	}

	// Verify TOTP code once before entering transaction if supplied.
	mfaVerified := in.MFAVerified
	if !mfaVerified && strings.TrimSpace(in.TOTPCode) != "" && s.mfa != nil {
		if err := s.mfa.Verify(ctx, approverID, strings.TrimSpace(in.TOTPCode)); err != nil {
			return DecideResult{}, fmt.Errorf("%w: %v", ErrMFARequired, err)
		}
		mfaVerified = true
	}

	now := s.currentNow()
	delegations, err := s.repo.FindActiveDelegationsForDelegate(ctx, tid, approverID, now)
	if err != nil {
		return DecideResult{}, err
	}

	var allDecisions []ApprovalDecision
	updatedReq, recordedDec, err := s.repo.ApplyDecision(ctx, tid, reqID, func(req ApprovalRequest, existing []ApprovalDecision, policy *Policy) (ApprovalRequest, ApprovalDecision, error) {
		if req.Status != StatusPending {
			return ApprovalRequest{}, ApprovalDecision{}, ErrRequestAlreadyResolved
		}
		if req.ExpiresAt != nil && !now.Before(*req.ExpiresAt) {
			return ApprovalRequest{}, ApprovalDecision{}, ErrRequestExpired
		}
		if expected := strings.TrimSpace(in.ExpectedHash); expected != "" && !strings.EqualFold(expected, req.PayloadHash) {
			return ApprovalRequest{}, ApprovalDecision{}, ErrPayloadHashMismatch
		}
		if strings.EqualFold(approverID, req.RequestedBy) {
			s.recordSoD(ctx, req.TenantID, approverID, actorType, approverRole, decision, req.RequestID, req.RequestedBy, in)
			return ApprovalRequest{}, ApprovalDecision{}, ErrSoDViolation
		}

		// Prevent duplicate decision by same approver on current step.
		for _, d := range existing {
			if d.StepIndex == req.CurrentStep && strings.EqualFold(d.ApproverID, approverID) {
				return ApprovalRequest{}, ApprovalDecision{}, ErrAlreadyDecidedByActor
			}
		}

		var step *ApprovalStep
		if policy != nil && int(req.CurrentStep) >= 0 && int(req.CurrentStep) < len(policy.Steps) {
			st := policy.Steps[req.CurrentStep]
			step = &st
		}

		// Role & delegation check.
		delegatedFrom := ""
		if step != nil && len(step.AllowedRoles) > 0 {
			if !roleAllowed(approverRole, step.AllowedRoles) {
				matchedDel := findMatchingDelegation(delegations, step.AllowedRoles, req.RequestedBy)
				if matchedDel == nil {
					return ApprovalRequest{}, ApprovalDecision{}, ErrRoleNotAuthorized
				}
				delegatedFrom = matchedDel.DelegatorID
			}
		} else if len(delegations) > 0 {
			if matchedDel := findMatchingDelegation(delegations, nil, req.RequestedBy); matchedDel != nil {
				delegatedFrom = matchedDel.DelegatorID
			}
		}

		// MFA threshold check (required on APPROVE; also enforce if step strictly marks RequireMFA).
		if decision == DecisionApprove {
			needMFA := s.defaultMFAThresholdMinor > 0 && req.AmountMinor >= s.defaultMFAThresholdMinor
			if step != nil && (step.RequireMFA || (step.MFAThresholdMinor > 0 && req.AmountMinor >= step.MFAThresholdMinor)) {
				needMFA = true
			}
			if needMFA && !mfaVerified {
				return ApprovalRequest{}, ApprovalDecision{}, ErrMFARequired
			}
		}

		// Break-glass post-review tagging on decision if requested.
		if in.BreakGlass {
			reason := strings.TrimSpace(in.BreakGlassReason)
			if reason == "" {
				return ApprovalRequest{}, ApprovalDecision{}, fmt.Errorf("%w: break_glass_reason is required", ErrInvalidInput)
			}
			if !isBreakGlassRole(approverRole) {
				return ApprovalRequest{}, ApprovalDecision{}, ErrRoleNotAuthorized
			}
			req.BreakGlass = true
			req.BreakGlassReason = reason
		}

		dec := ApprovalDecision{
			TenantID:      req.TenantID,
			RequestID:     req.RequestID,
			DecisionID:    "apd_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24],
			StepIndex:     req.CurrentStep,
			ApproverID:    approverID,
			DelegatedFrom: delegatedFrom,
			Decision:      decision,
			Comment:       strings.TrimSpace(in.Comment),
			MfaVerified:   mfaVerified,
			CreatedAt:     now,
		}

		req.UpdatedAt = now
		if decision == DecisionReject {
			req.Status = StatusRejected
			t := now
			req.ResolvedAt = &t
			allDecisions = append(append([]ApprovalDecision(nil), existing...), dec)
			return req, dec, nil
		}

		if req.BreakGlass {
			req.Status = StatusApproved
			t := now
			req.ResolvedAt = &t
			allDecisions = append(append([]ApprovalDecision(nil), existing...), dec)
			return req, dec, nil
		}

		// Count approvals for CurrentStep (including this one).
		stepApprovals := 1
		for _, d := range existing {
			if d.StepIndex == req.CurrentStep && d.Decision == DecisionApprove {
				stepApprovals++
			}
		}
		needApprovers := int(req.RequiredApprovers)
		if step != nil && step.RequiredApprovers > needApprovers {
			needApprovers = step.RequiredApprovers
		}
		if needApprovers <= 0 {
			needApprovers = 1
		}

		if stepApprovals >= needApprovers {
			totalSteps := 1
			if policy != nil && len(policy.Steps) > 0 {
				totalSteps = len(policy.Steps)
			}
			if int(req.CurrentStep)+1 < totalSteps {
				req.CurrentStep++
				nextStep := policy.Steps[req.CurrentStep]
				if nextStep.RequiredApprovers > 0 {
					req.RequiredApprovers = int64(nextStep.RequiredApprovers)
				} else {
					req.RequiredApprovers = 1
				}
			} else {
				req.Status = StatusApproved
				t := now
				req.ResolvedAt = &t
			}
		}
		allDecisions = append(append([]ApprovalDecision(nil), existing...), dec)
		return req, dec, nil
	}, s.audit, in.TraceID)
	if err != nil {
		return DecideResult{}, err
	}

	if updatedReq.Status == StatusApproved {
		s.triggerOnApproved(ctx, updatedReq, allDecisions)
	}

	return DecideResult{
		Request:  updatedReq,
		Decision: recordedDec,
	}, nil
}

func (s *Service) triggerOnApproved(ctx context.Context, req ApprovalRequest, decs []ApprovalDecision) {
	if s == nil {
		return
	}
	s.hooksMu.RLock()
	key := strings.ToLower(strings.TrimSpace(req.ObjectType)) + ":" + strings.ToLower(strings.TrimSpace(req.Action))
	fn := s.onApproved[key]
	s.hooksMu.RUnlock()
	if fn != nil {
		_ = fn(ctx, req, decs)
	}
}

func (s *Service) recordSoD(ctx context.Context, tenantID, actorID, actorType, role, decision, reqID, makerID string, in DecideInput) {
	if s == nil || s.sod == nil {
		return
	}
	_ = s.sod.CheckFourEyes(ctx, SoDCheckInput{
		TenantID:    tenantID,
		ActorID:     actorID,
		ActorType:   actorType,
		Role:        role,
		Action:      "approval." + strings.ToLower(decision),
		MakerAction: "approval.request",
		ObjectType:  "ApprovalRequest",
		ObjectID:    reqID,
		MakerID:     makerID,
		RoutePath:   in.RoutePath,
		TraceID:     in.TraceID,
		IP:          in.IP,
		DeviceID:    in.DeviceID,
		IsApproval:  true,
	})
}

func roleAllowed(actorRole string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	ar := strings.ToUpper(strings.TrimSpace(actorRole))
	for _, r := range allowed {
		if strings.EqualFold(ar, strings.TrimSpace(r)) {
			return true
		}
	}
	return false
}

func findMatchingDelegation(delegations []ApprovalDelegation, allowedRoles []string, requesterID string) *ApprovalDelegation {
	for i := range delegations {
		d := &delegations[i]
		if !d.Enabled {
			continue
		}
		if requesterID != "" && strings.EqualFold(d.DelegatorID, requesterID) {
			// Prevent SoD bypass via self-delegation of maker's own request.
			continue
		}
		if len(allowedRoles) == 0 {
			return d
		}
		if strings.TrimSpace(d.RoleScope) == "" {
			return d
		}
		if roleAllowed(d.RoleScope, allowedRoles) {
			return d
		}
	}
	return nil
}

// VerifyApprovedRequest verifies that requestID exists, is APPROVED, matches (tenantID, objectType, action),
// matches payloadHash, and satisfies minApprovers distinct human approvers.
func (s *Service) VerifyApprovedRequest(ctx context.Context, tenantID, requestID, objectType, action, payloadHash string, minApprovers int) (ApprovalRequest, []ApprovalDecision, error) {
	if s == nil || s.repo == nil {
		return ApprovalRequest{}, nil, ErrRepoUnavailable
	}
	req, decs, ok, err := s.repo.GetRequest(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(requestID))
	if err != nil {
		return ApprovalRequest{}, nil, err
	}
	if !ok {
		return ApprovalRequest{}, nil, ErrRequestNotFound
	}
	if objectType != "" && !strings.EqualFold(req.ObjectType, objectType) {
		return ApprovalRequest{}, nil, fmt.Errorf("%w: approval request object_type mismatch", ErrInvalidInput)
	}
	if action != "" && !strings.EqualFold(req.Action, action) {
		return ApprovalRequest{}, nil, fmt.Errorf("%w: approval request action mismatch", ErrInvalidInput)
	}
	if payloadHash != "" && !strings.EqualFold(req.PayloadHash, payloadHash) {
		return ApprovalRequest{}, nil, ErrPayloadHashMismatch
	}
	if req.Status != StatusApproved {
		return ApprovalRequest{}, nil, fmt.Errorf("%w: approval request status is %s", ErrRequestAlreadyResolved, req.Status)
	}
	if minApprovers > 1 && !req.BreakGlass {
		distinct := make(map[string]struct{})
		for _, d := range decs {
			if d.Decision == DecisionApprove && !strings.EqualFold(d.ApproverID, req.RequestedBy) {
				distinct[strings.ToLower(strings.TrimSpace(d.ApproverID))] = struct{}{}
			}
		}
		if len(distinct) < minApprovers {
			return ApprovalRequest{}, nil, fmt.Errorf("%w: requires %d distinct approvers, got %d", ErrSoDViolation, minApprovers, len(distinct))
		}
	}
	return req, decs, nil
}

// CreateDelegation registers a time-bounded delegation while forbidding self-delegation and AGENT/AUDITOR delegates.
func (s *Service) CreateDelegation(ctx context.Context, in CreateDelegationInput) (ApprovalDelegation, error) {
	if s == nil || s.repo == nil {
		return ApprovalDelegation{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	delegator := strings.TrimSpace(in.DelegatorID)
	delegate := strings.TrimSpace(in.DelegateID)
	reason := strings.TrimSpace(in.Reason)
	if tid == "" || delegator == "" || delegate == "" || reason == "" {
		return ApprovalDelegation{}, fmt.Errorf("%w: tenant_id, delegator_id, delegate_id, and reason are required", ErrInvalidInput)
	}
	if strings.EqualFold(delegator, delegate) {
		return ApprovalDelegation{}, fmt.Errorf("%w: cannot delegate to self", ErrInvalidInput)
	}
	if strings.EqualFold(in.ActorType, auth.ActorTypeAgent) || strings.EqualFold(in.DelegateType, auth.ActorTypeAgent) {
		return ApprovalDelegation{}, ErrAgentApprovalForbidden
	}
	if strings.EqualFold(strings.TrimSpace(in.RoleScope), auth.SupplierRoleAgent) {
		return ApprovalDelegation{}, ErrAgentApprovalForbidden
	}
	if strings.EqualFold(strings.TrimSpace(in.RoleScope), auth.SupplierRoleAuditor) {
		return ApprovalDelegation{}, ErrAuditorReadOnly
	}
	now := s.currentNow()
	startsAt := in.StartsAt.UTC()
	if startsAt.IsZero() {
		startsAt = now
	}
	endsAt := in.EndsAt.UTC()
	if endsAt.IsZero() || !endsAt.After(startsAt) {
		return ApprovalDelegation{}, fmt.Errorf("%w: ends_at must be strictly after starts_at", ErrInvalidInput)
	}
	del := ApprovalDelegation{
		TenantID:     tid,
		DelegationID: "apdel_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:22],
		DelegatorID:  delegator,
		DelegateID:   delegate,
		RoleScope:    strings.ToUpper(strings.TrimSpace(in.RoleScope)),
		StartsAt:     startsAt,
		EndsAt:       endsAt,
		Reason:       reason,
		Enabled:      true,
		CreatedAt:    now,
	}
	if err := s.repo.CreateDelegation(ctx, del); err != nil {
		return ApprovalDelegation{}, err
	}
	return del, nil
}

// ListDelegations returns delegations scoped to tenantID and optional actorID.
func (s *Service) ListDelegations(ctx context.Context, tenantID, actorID string) ([]ApprovalDelegation, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	return s.repo.ListDelegations(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(actorID))
}

// RevokeDelegation disables an existing delegation.
func (s *Service) RevokeDelegation(ctx context.Context, tenantID, delegationID string) (ApprovalDelegation, error) {
	if s == nil || s.repo == nil {
		return ApprovalDelegation{}, ErrRepoUnavailable
	}
	return s.repo.SetDelegationEnabled(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(delegationID), false)
}

// MarshalPayloadJSON helper converts any Go value into canonical zero-float JSON + SHA-256 hex.
func MarshalPayloadJSON(v any) (string, string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", "", fmt.Errorf("%w: marshal payload: %v", ErrInvalidInput, err)
	}
	return CanonicalizeJSONBytes(raw)
}

func isServiceUnavailableErr(err error) bool {
	return errors.Is(err, ErrRepoUnavailable)
}
