package proposals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/pegasusx/pegasusx/apps/backend-go/roi"
)

// Canonical L2 soak-gating defaults (§7.5 & §11 D4).
const (
	DefaultMinSoakDays        int64 = 14
	DefaultMinExecutedCount   int64 = 10
	DefaultMaxRollbackRateBps int64 = 500
	DefaultMaxL2AmountMinor   int64 = 1_000_000_000 // 10,000,000 UZS in minor units (Tiyin)
)

// L2BlockedProposal explains why a pending proposal was excluded from L2 auto-execution.
type L2BlockedProposal struct {
	ProposalID   string `json:"proposal_id"`
	ProposalType string `json:"proposal_type"`
	AgentID      string `json:"agent_id"`
	ReasonCode   string `json:"reason_code"`
	Reason       string `json:"reason"`
}

// L2EvaluateInput carries parameters for L2 readiness evaluation and optional auto-execution.
type L2EvaluateInput struct {
	TenantID         string
	AutoExecute      bool
	MaxL2AmountMinor int64
	ProposalIDs      []string
	CallerID         string
	ActorType        string
	CallerRole       string
	RoutePath        string
	TraceID          string
	IP               string
	DeviceID         string
}

// L2EvaluationResult summarizes tenant soak readiness and L2 auto-execution outcome.
type L2EvaluationResult struct {
	TenantID                string              `json:"tenant_id"`
	SoakDays                int64               `json:"soak_days"`
	MinSoakDays             int64               `json:"min_soak_days"`
	ExecutedCount           int64               `json:"executed_count"`
	MinExecutedCount        int64               `json:"min_executed_count"`
	RollbackCount           int64               `json:"rollback_count"`
	RollbackRateBps         int64               `json:"rollback_rate_bps"`
	MaxRollbackRateBps      int64               `json:"max_rollback_rate_bps"`
	MaxL2AmountMinor        int64               `json:"max_l2_amount_minor"`
	Eligible                bool                `json:"eligible"`
	Blockers                []string            `json:"blockers"`
	EligibleProposalIDs     []string            `json:"eligible_proposal_ids"`
	EligibleCount           int                 `json:"eligible_count"`
	AutoExecutedCount       int                 `json:"auto_executed_count"`
	AutoExecutedProposalIDs []string            `json:"auto_executed_proposal_ids"`
	ExecutedProposals       []Proposal          `json:"executed_proposals,omitempty"`
	BlockedProposals        []L2BlockedProposal `json:"blocked_proposals,omitempty"`
}

type evaluateL2HTTPReq struct {
	TenantID         string   `json:"tenant_id,omitempty"`
	AutoExecute      bool     `json:"auto_execute"`
	MaxL2AmountMinor int64    `json:"max_l2_amount_minor,omitempty"`
	ProposalIDs      []string `json:"proposal_ids,omitempty"`
}

// IsHighRiskL2Blocked reports whether a proposal touches credit limits, write-offs,
// payouts, accounting periods, journals, or other high-risk actions unconditionally banned from L2 (§11 D4).
func IsHighRiskL2Blocked(p Proposal) (bool, string) {
	payload := decodePayloadMap(p.PayloadJSON)
	action := strings.ToUpper(strField(payload, "action"))
	pType := strings.ToUpper(strings.TrimSpace(p.ProposalType))
	tType := strings.ToUpper(strings.TrimSpace(p.TargetObjectType))

	tokens := []string{pType, action, tType}
	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		if strings.Contains(tok, "CREDIT_LIMIT") ||
			strings.Contains(tok, "CREDIT_HOLD") ||
			strings.Contains(tok, "FREEZE_CREDIT") ||
			strings.Contains(tok, "CREDIT_NOTE") ||
			strings.Contains(tok, "WRITE_OFF") ||
			strings.Contains(tok, "WRITEOFF") ||
			strings.Contains(tok, "PAYOUT") ||
			strings.Contains(tok, "PERIOD") ||
			strings.Contains(tok, "JOURNAL") ||
			strings.Contains(tok, "REVERSAL") ||
			strings.Contains(tok, "CUSTOM_ROLE") ||
			strings.Contains(tok, "SCIM") {
			return true, fmt.Sprintf("high-risk action %q is strictly forbidden from L2 auto-execution", tok)
		}
	}
	return false, ""
}

func classifyL2ProposalEligibility(p Proposal, maxAmountMinor int64, now time.Time) (bool, string, string) {
	if blocked, reason := IsHighRiskL2Blocked(p); blocked {
		return false, "high_risk_action_forbidden_for_l2", reason
	}
	if p.Status != StatusPending {
		return false, "not_pending", fmt.Sprintf("proposal status is %s", p.Status)
	}
	if p.ExpiresAt != nil && !now.Before(*p.ExpiresAt) {
		return false, "proposal_expired", "proposal has expired"
	}

	pType := strings.ToUpper(strings.TrimSpace(p.ProposalType))
	agentID := strings.ToUpper(strings.TrimSpace(p.AgentID))
	payload := decodePayloadMap(p.PayloadJSON)
	action := strings.ToLower(strField(payload, "action"))

	isReorder := pType == TypeReorder && (agentID == AgentReorder || agentID == AgentHuman || agentID == "")
	isLowRiskNotify := (pType == "NOTIFICATION" || action == "notify" || action == "broadcast_template") &&
		(agentID == AgentReorder || agentID == "NOTIFY_AGENT" || agentID == AgentHuman || agentID == "")

	if isReorder {
		switch action {
		case "",
			string(planning.AgentApproveInsight),
			string(planning.AgentOpenSupplyRequest),
			string(planning.AgentBroadcastTemplate),
			"approve_reorder_suggestion",
			"notify":
		default:
			return false, "action_not_l2_allowlisted", fmt.Sprintf("reorder action %q is not allowlisted for L2", action)
		}
	} else if !isLowRiskNotify {
		return false, "proposal_type_not_l2_allowlisted", fmt.Sprintf("only low-risk REORDER_AGENT or notification proposals may execute at L2 (got %s/%s)", pType, agentID)
	}

	effectiveMinor := p.EstimatedImpactMinor
	if amt := int64Field(payload, "amount_minor"); amt > effectiveMinor {
		effectiveMinor = amt
	}
	qty := int64Field(payload, "suggested_qty")
	unitPrice := int64Field(payload, "unit_price_minor")
	if qty > 0 && unitPrice > 0 {
		if orderVal := qty * unitPrice; orderVal > effectiveMinor {
			effectiveMinor = orderVal
		}
	}
	if isReorder && effectiveMinor <= 0 {
		return false, "invalid_amount_minor", "reorder proposal requires positive amount/impact ceiling check"
	}
	if maxAmountMinor > 0 && effectiveMinor > maxAmountMinor {
		return false, "amount_exceeds_max_l2_ceiling", fmt.Sprintf("effective amount %d minor exceeds max_l2_amount_minor %d", effectiveMinor, maxAmountMinor)
	}
	return true, "", ""
}

// EvaluateL2 evaluates tenant L2 soak readiness and, when AutoExecute is true, promotes and
// executes eligible low-value REORDER_AGENT / notification proposals within max_l2_amount_minor.
func (s *Service) EvaluateL2(ctx context.Context, in L2EvaluateInput) (L2EvaluationResult, error) {
	if s == nil || s.repo == nil {
		return L2EvaluationResult{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	if tid == "" {
		return L2EvaluationResult{}, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	if in.MaxL2AmountMinor < 0 {
		return L2EvaluationResult{}, fmt.Errorf("%w: max_l2_amount_minor must be non-negative", ErrInvalidInput)
	}
	maxAmount := in.MaxL2AmountMinor
	if maxAmount == 0 {
		maxAmount = DefaultMaxL2AmountMinor
	}

	actorType := strings.ToLower(strings.TrimSpace(in.ActorType))
	callerRole := strings.ToUpper(strings.TrimSpace(in.CallerRole))
	callerID := strings.TrimSpace(in.CallerID)
	if in.AutoExecute {
		if callerRole == auth.SupplierRoleAuditor {
			return L2EvaluationResult{}, ErrAuditorReadOnly
		}
		if actorType == auth.ActorTypeAgent || callerRole == auth.SupplierRoleAgent {
			return L2EvaluationResult{}, ErrAgentApprovalForbidden
		}
		if err := planning.ValidateAgentGovernance(actorType, true); err != nil {
			return L2EvaluationResult{}, ErrAgentApprovalForbidden
		}
	}

	now := s.currentNow()
	stats, err := s.repo.GetL2SoakStats(ctx, tid, now)
	if err != nil {
		return L2EvaluationResult{}, err
	}

	var blockers []string
	if stats.SoakDays < DefaultMinSoakDays {
		blockers = append(blockers, fmt.Sprintf("min_soak_days_not_met: %d < %d", stats.SoakDays, DefaultMinSoakDays))
	}
	if stats.ExecutedCount < DefaultMinExecutedCount {
		blockers = append(blockers, fmt.Sprintf("min_executed_count_not_met: %d < %d", stats.ExecutedCount, DefaultMinExecutedCount))
	}
	if stats.RollbackRateBps > DefaultMaxRollbackRateBps {
		blockers = append(blockers, fmt.Sprintf("rollback_rate_exceeds_max: %d > %d bps", stats.RollbackRateBps, DefaultMaxRollbackRateBps))
	}
	if blockers == nil {
		blockers = []string{}
	}

	res := L2EvaluationResult{
		TenantID:                tid,
		SoakDays:                stats.SoakDays,
		MinSoakDays:             DefaultMinSoakDays,
		ExecutedCount:           stats.ExecutedCount,
		MinExecutedCount:        DefaultMinExecutedCount,
		RollbackCount:           stats.RollbackCount,
		RollbackRateBps:         stats.RollbackRateBps,
		MaxRollbackRateBps:      DefaultMaxRollbackRateBps,
		MaxL2AmountMinor:        maxAmount,
		Eligible:                len(blockers) == 0,
		Blockers:                blockers,
		EligibleProposalIDs:     []string{},
		AutoExecutedProposalIDs: []string{},
		ExecutedProposals:       []Proposal{},
		BlockedProposals:        []L2BlockedProposal{},
	}

	var candidates []Proposal
	if len(in.ProposalIDs) > 0 {
		seen := make(map[string]struct{}, len(in.ProposalIDs))
		for _, rawID := range in.ProposalIDs {
			pid := strings.TrimSpace(rawID)
			if pid == "" {
				continue
			}
			if _, dup := seen[pid]; dup {
				continue
			}
			seen[pid] = struct{}{}
			p, ok, gerr := s.repo.Get(ctx, tid, pid)
			if gerr != nil {
				return L2EvaluationResult{}, gerr
			}
			if !ok {
				res.BlockedProposals = append(res.BlockedProposals, L2BlockedProposal{
					ProposalID: pid,
					ReasonCode: "proposal_not_found",
					Reason:     "proposal not found for tenant",
				})
				continue
			}
			candidates = append(candidates, p)
		}
	} else {
		pending, lerr := s.repo.List(ctx, ListFilter{TenantID: tid, Status: StatusPending, Limit: 200})
		if lerr != nil {
			return L2EvaluationResult{}, lerr
		}
		candidates = pending
	}

	var toExecute []Proposal
	for _, p := range candidates {
		ok, code, reason := classifyL2ProposalEligibility(p, maxAmount, now)
		if !ok {
			res.BlockedProposals = append(res.BlockedProposals, L2BlockedProposal{
				ProposalID:   p.ProposalID,
				ProposalType: p.ProposalType,
				AgentID:      p.AgentID,
				ReasonCode:   code,
				Reason:       reason,
			})
			continue
		}
		res.EligibleProposalIDs = append(res.EligibleProposalIDs, p.ProposalID)
		toExecute = append(toExecute, p)
	}
	res.EligibleCount = len(res.EligibleProposalIDs)

	if !in.AutoExecute || !res.Eligible {
		return res, nil
	}

	for _, cand := range toExecute {
		executedProp, blockItem, execErr := s.executeProposalAtL2(ctx, tid, cand.ProposalID, maxAmount, callerID, actorType, callerRole, stats, in)
		if execErr != nil {
			return L2EvaluationResult{}, execErr
		}
		if blockItem != nil {
			res.BlockedProposals = append(res.BlockedProposals, *blockItem)
			continue
		}
		res.ExecutedProposals = append(res.ExecutedProposals, executedProp)
		res.AutoExecutedProposalIDs = append(res.AutoExecutedProposalIDs, executedProp.ProposalID)
	}
	res.AutoExecutedCount = len(res.AutoExecutedProposalIDs)
	return res, nil
}

func (s *Service) executeProposalAtL2(
	ctx context.Context,
	tid, pid string,
	maxAmount int64,
	callerID, actorType, callerRole string,
	soakStats L2SoakMetrics,
	in L2EvaluateInput,
) (Proposal, *L2BlockedProposal, error) {
	unlock := s.lockTarget(tid, "l2_lookup", pid)
	p, ok, err := s.repo.Get(ctx, tid, pid)
	unlock()
	if err != nil {
		return Proposal{}, nil, err
	}
	if !ok {
		return Proposal{}, &L2BlockedProposal{
			ProposalID: pid,
			ReasonCode: "proposal_not_found",
			Reason:     "proposal disappeared before execution",
		}, nil
	}

	targetUnlock := s.lockTarget(tid, p.TargetObjectType, p.TargetObjectID)
	defer targetUnlock()

	p, ok, err = s.repo.Get(ctx, tid, pid)
	if err != nil {
		return Proposal{}, nil, err
	}
	if !ok {
		return Proposal{}, &L2BlockedProposal{
			ProposalID: pid,
			ReasonCode: "proposal_not_found",
			Reason:     "proposal disappeared before execution",
		}, nil
	}

	now := s.currentNow()
	if okElig, code, reason := classifyL2ProposalEligibility(p, maxAmount, now); !okElig {
		return Proposal{}, &L2BlockedProposal{
			ProposalID:   p.ProposalID,
			ProposalType: p.ProposalType,
			AgentID:      p.AgentID,
			ReasonCode:   code,
			Reason:       reason,
		}, nil
	}
	if callerID != "" && strings.EqualFold(p.CreatedBy, callerID) {
		return Proposal{}, &L2BlockedProposal{
			ProposalID:   p.ProposalID,
			ProposalType: p.ProposalType,
			AgentID:      p.AgentID,
			ReasonCode:   "sod_self_approval_forbidden_for_l2",
			Reason:       "creator cannot trigger L2 auto-execution of own proposal",
		}, nil
	}

	_, recomputedHash, err := CanonicalizeJSONBytes([]byte(p.PayloadJSON))
	if err != nil || !strings.EqualFold(recomputedHash, p.PayloadHash) {
		return Proposal{}, &L2BlockedProposal{
			ProposalID:   p.ProposalID,
			ProposalType: p.ProposalType,
			AgentID:      p.AgentID,
			ReasonCode:   "payload_hash_mismatch",
			Reason:       "payload hash integrity verification failed",
		}, nil
	}

	overlap, err := s.repo.HasOverlappingExecuted(ctx, tid, p.TargetObjectType, p.TargetObjectID, p.ProposalType, p.ProposalID, p.CreatedAt)
	if err != nil {
		return Proposal{}, nil, err
	}
	if overlap {
		return Proposal{}, &L2BlockedProposal{
			ProposalID:   p.ProposalID,
			ProposalType: p.ProposalType,
			AgentID:      p.AgentID,
			ReasonCode:   "target_execution_conflict",
			Reason:       "overlapping proposal already executed for target",
		}, nil
	}

	if err := s.verifyPreconditions(ctx, tid, p); err != nil {
		p.Status = StatusFailed
		p.ExecutionError = err.Error()
		p.UpdatedAt = now
		_ = s.repo.Update(ctx, p)
		return Proposal{}, &L2BlockedProposal{
			ProposalID:   p.ProposalID,
			ProposalType: p.ProposalType,
			AgentID:      p.AgentID,
			ReasonCode:   "precondition_failed",
			Reason:       err.Error(),
		}, nil
	}

	// Resolve linked approval request cleanly if present.
	if s.approvals != nil && p.ApprovalRequestID != "" {
		decActor := nonEmptyOr(callerID, "l2:soak-policy")
		if _, dErr := s.approvals.Decide(ctx, approvals.DecideInput{
			TenantID:     tid,
			RequestID:    p.ApprovalRequestID,
			ApproverID:   decActor,
			ActorType:    auth.ActorTypeHuman,
			ApproverRole: nonEmptyOr(callerRole, auth.SupplierRoleOwner),
			Decision:     approvals.DecisionApprove,
			Comment:      "L2 soak-gated auto-execution",
			ExpectedHash: p.PayloadHash,
			RoutePath:    in.RoutePath,
			TraceID:      in.TraceID,
			IP:           in.IP,
			DeviceID:     in.DeviceID,
		}); dErr != nil && !errors.Is(dErr, approvals.ErrRequestAlreadyResolved) {
			return Proposal{}, nil, dErr
		}
	}

	executorLabel := "l2:" + nonEmptyOr(callerID, "soak-policy")
	execOutcome, err := s.dispatchGovernedExecution(ctx, p, executorLabel)
	if err != nil {
		p.Status = StatusFailed
		p.ExecutionError = err.Error()
		p.UpdatedAt = now
		_ = s.repo.Update(ctx, p)
		return Proposal{}, &L2BlockedProposal{
			ProposalID:   p.ProposalID,
			ProposalType: p.ProposalType,
			AgentID:      p.AgentID,
			ReasonCode:   "execution_failed",
			Reason:       err.Error(),
		}, nil
	}
	if execOutcome == nil {
		execOutcome = map[string]any{}
	}
	execOutcome["autonomy_level"] = AutonomyLevelL2

	if s.roiSvc != nil && p.EstimatedImpactMinor > 0 {
		cat := nonEmptyOr(p.RoiCategory, defaultRoiCategoryForProposal(p.ProposalType))
		ev, rErr := s.roiSvc.RecordEvent(ctx, roi.RecordEventInput{
			TenantID:        tid,
			Category:        cat,
			SavingsMinor:    p.EstimatedImpactMinor,
			Currency:        nonEmptyOr(p.Currency, "UZS"),
			Methodology:     roi.DefaultMethodology(cat) + " (L2 soak auto-executed " + p.AgentID + ")",
			ConfidenceBps:   p.ConfidenceBps,
			ProposalID:      p.ProposalID,
			EvidenceRefType: p.TargetObjectType,
			EvidenceRefID:   p.TargetObjectID,
			Notes:           p.Title,
			RecordedBy:      executorLabel,
			IdempotencyKey:  "proposal-l2-exec:" + p.ProposalID,
			OccurredAt:      now,
		})
		if rErr != nil {
			return Proposal{}, nil, rErr
		}
		p.RoiEventID = ev.EventID
		execOutcome["roi_event_id"] = ev.EventID
	}

	resRaw, _ := json.Marshal(execOutcome)
	p.AutonomyLevel = AutonomyLevelL2
	soakCopy := soakStats
	p.L2SoakContext = &soakCopy
	p.Status = StatusExecuted
	p.DecidedBy = "policy:l2-soak"
	p.DecisionComment = "L2 soak-gated auto-execution"
	p.DecidedAt = &now
	p.ExecutedBy = executorLabel
	p.ExecutedAt = &now
	p.UpdatedAt = now
	p.ExecutionResultJSON = string(resRaw)
	p.ExecutionResult = execOutcome
	p.ExecutionError = ""
	if err := s.repo.Update(ctx, p); err != nil {
		return Proposal{}, nil, err
	}

	if s.audit != nil {
		_ = s.audit.RecordEvent(ctx, AuditRecordInput{
			TenantID:   tid,
			ActorID:    nonEmptyOr(callerID, executorLabel),
			ActorType:  nonEmptyOr(actorType, auth.ActorTypeHuman),
			Role:       callerRole,
			Action:     "proposal.l2.auto_executed",
			ObjectType: "Proposal",
			ObjectID:   p.ProposalID,
			BeforeJSON: p.PayloadJSON,
			AfterJSON:  p.ExecutionResultJSON,
			ReasonCode: "L2_SOAK_AUTO_EXECUTE",
			ApprovalID: p.ApprovalRequestID,
			TraceID:    in.TraceID,
			IP:         in.IP,
			DeviceID:   in.DeviceID,
		})
	}
	return p, nil, nil
}

// HandleEvaluateL2 serves GET/POST /v1/proposals/l2-evaluate.
func (s *Service) HandleEvaluateL2(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "proposals_unavailable", "proposals service unavailable")
		return
	}
	claims, _ := auth.FromContext(r.Context())

	switch r.Method {
	case http.MethodGet:
		tenantID, ok := resolveProposalTenantScope(w, r, r.URL.Query().Get("tenant_id"))
		if !ok {
			return
		}
		var maxAmt int64
		if raw := strings.TrimSpace(r.URL.Query().Get("max_l2_amount_minor")); raw != "" {
			v, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || v < 0 {
				writeJSONError(w, http.StatusBadRequest, "invalid_request", "max_l2_amount_minor must be a non-negative integer")
				return
			}
			maxAmt = v
		}
		res, err := s.EvaluateL2(r.Context(), L2EvaluateInput{
			TenantID:         tenantID,
			AutoExecute:      false,
			MaxL2AmountMinor: maxAmt,
			CallerID:         auth.ActorFromContext(r.Context()),
			ActorType:        auth.EffectiveActorType(claims),
			CallerRole:       auth.EffectiveSupplierRole(claims),
		})
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, res)

	case http.MethodPost:
		if !requireNonAuditorNonAgent(w, r, auth.PermAgentProposalApprove) {
			return
		}
		var req evaluateL2HTTPReq
		if !decodeJSONBody(w, r, &req, true) {
			return
		}
		tenantID, ok := resolveProposalTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
		if !ok {
			return
		}
		res, err := s.EvaluateL2(r.Context(), L2EvaluateInput{
			TenantID:         tenantID,
			AutoExecute:      req.AutoExecute,
			MaxL2AmountMinor: req.MaxL2AmountMinor,
			ProposalIDs:      req.ProposalIDs,
			CallerID:         auth.ActorFromContext(r.Context()),
			ActorType:        auth.EffectiveActorType(claims),
			CallerRole:       auth.EffectiveSupplierRole(claims),
			RoutePath:        r.URL.Path,
			TraceID:          claims.TraceID,
			IP:               r.RemoteAddr,
			DeviceID:         strings.TrimSpace(r.Header.Get("X-Device-Id")),
		})
		if err != nil {
			writeDomainError(w, err)
			return
		}
		if req.AutoExecute {
			afterRaw, _ := json.Marshal(map[string]any{
				"eligible":                   res.Eligible,
				"auto_executed_count":        res.AutoExecutedCount,
				"auto_executed_proposal_ids": res.AutoExecutedProposalIDs,
				"blocked_count":              len(res.BlockedProposals),
			})
			audit.SetMutationContext(r.Context(), audit.MutationContext{
				TenantID:   tenantID,
				Action:     "proposal.l2.evaluate",
				ObjectType: "L2Policy",
				ObjectID:   tenantID,
				AfterJSON:  string(afterRaw),
				ReasonCode: "L2_SOAK_EVALUATE",
			})
		}
		writeJSON(w, http.StatusOK, res)

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}
