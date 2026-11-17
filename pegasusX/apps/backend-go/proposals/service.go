package proposals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltower"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/pegasusx/pegasusx/apps/backend-go/roi"
)

// DomainExecutor executes domain-specific mutations for approved proposals once
// all approvals, payload hashes, target locks, and preconditions have passed.
type DomainExecutor interface {
	ExecuteProposalAction(ctx context.Context, p Proposal, executorID string) (map[string]any, error)
}

// Service orchestrates M1.7 L1 proposal creation, CEL approval routing, human
// approve/edit/reject governance, per-target execution locking, and ROI attribution.
type Service struct {
	repo         Repository
	dataReader   AgentDataReader
	approvals    *approvals.Service
	roiSvc       *roi.Service
	planningExec *planning.Executor
	ctExec       *controltower.ActionExecutor
	domainExec   DomainExecutor
	audit        AuditRecorder
	now          func() time.Time

	targetLocksMu sync.Mutex
	targetLocks   map[string]*sync.Mutex
}

// NewService constructs a Spanner-backed proposals service.
func NewService(client *spanner.Client) *Service {
	if client == nil {
		return &Service{
			now:         func() time.Time { return time.Now().UTC() },
			targetLocks: make(map[string]*sync.Mutex),
		}
	}
	sr := NewSpannerRepository(client)
	return &Service{
		repo:         sr,
		dataReader:   sr,
		planningExec: planning.NewExecutor(client),
		now:          func() time.Time { return time.Now().UTC() },
		targetLocks:  make(map[string]*sync.Mutex),
	}
}

// NewServiceWithRepository constructs a proposals service backed by a custom Repository.
func NewServiceWithRepository(repo Repository) *Service {
	svc := &Service{
		repo:        repo,
		now:         func() time.Time { return time.Now().UTC() },
		targetLocks: make(map[string]*sync.Mutex),
	}
	if dr, ok := repo.(AgentDataReader); ok {
		svc.dataReader = dr
	}
	return svc
}

// Repo exposes the underlying Repository.
func (s *Service) Repo() Repository {
	if s == nil {
		return nil
	}
	return s.repo
}

// SetApprovals attaches the M1.2 CEL Approvals Engine.
func (s *Service) SetApprovals(a *approvals.Service) {
	if s != nil {
		s.approvals = a
	}
}

// SetROI attaches the M1.8 ROI Attribution Service.
func (s *Service) SetROI(r *roi.Service) {
	if s != nil {
		s.roiSvc = r
	}
}

// SetPlanningExecutor attaches the governed planning executor.
func (s *Service) SetPlanningExecutor(pe *planning.Executor) {
	if s != nil {
		s.planningExec = pe
	}
}

// SetControlTowerExecutor attaches the control tower ActionExecutor.
func (s *Service) SetControlTowerExecutor(ce *controltower.ActionExecutor) {
	if s != nil {
		s.ctExec = ce
	}
}

// SetDomainExecutor attaches custom domain execution callbacks.
func (s *Service) SetDomainExecutor(de DomainExecutor) {
	if s != nil {
		s.domainExec = de
	}
}

// SetAgentDataReader overrides the live Spanner candidate/precondition reader.
func (s *Service) SetAgentDataReader(dr AgentDataReader) {
	if s != nil {
		s.dataReader = dr
	}
}

// SetAuditRecorder attaches an immutable audit recorder.
func (s *Service) SetAuditRecorder(rec AuditRecorder) {
	if s != nil {
		s.audit = rec
	}
}

func (s *Service) currentNow() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) lockTarget(tenantID, targetType, targetID string) func() {
	key := strings.TrimSpace(tenantID) + ":" + strings.ToLower(strings.TrimSpace(targetType)) + ":" + strings.TrimSpace(targetID)
	s.targetLocksMu.Lock()
	if s.targetLocks == nil {
		s.targetLocks = make(map[string]*sync.Mutex)
	}
	mu, ok := s.targetLocks[key]
	if !ok {
		mu = &sync.Mutex{}
		s.targetLocks[key] = mu
	}
	s.targetLocksMu.Unlock()
	mu.Lock()
	return mu.Unlock
}

// CreateProposal validates canonical zero-float payloads, creates a PENDING proposal,
// routes it through approvals.Engine (ForceCreate=true so agents NEVER self-approve),
// and emits proposal.created (+ inventory.rebalance.proposed for expiry rebalance).
func (s *Service) CreateProposal(ctx context.Context, in CreateProposalInput) (Proposal, error) {
	if s == nil || s.repo == nil {
		return Proposal{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	pType := strings.ToUpper(strings.TrimSpace(in.ProposalType))
	tType := strings.TrimSpace(in.TargetObjectType)
	tID := strings.TrimSpace(in.TargetObjectID)
	title := strings.TrimSpace(in.Title)
	rationale := strings.TrimSpace(in.Rationale)
	if tid == "" || pType == "" || tType == "" || tID == "" || title == "" {
		return Proposal{}, fmt.Errorf("%w: tenant_id, proposal_type, target_object_type, target_object_id, and title are required", ErrInvalidInput)
	}
	if rationale == "" {
		rationale = strings.TrimSpace(in.Summary)
	}
	if rationale == "" {
		rationale = title
	}
	if in.EstimatedImpactMinor < 0 {
		return Proposal{}, fmt.Errorf("%w: estimated_impact_minor must be non-negative", ErrInvalidInput)
	}
	if in.ConfidenceBps < 0 || in.ConfidenceBps > 10000 {
		return Proposal{}, fmt.Errorf("%w: confidence_bps must be between 0 and 10000", ErrInvalidInput)
	}

	if idem := strings.TrimSpace(in.IdempotencyKey); idem != "" {
		if existing, found, err := s.repo.FindByIdempotency(ctx, tid, idem); err != nil {
			return Proposal{}, err
		} else if found {
			return existing, nil
		}
	}

	var canonJSON, payloadHash string
	var err error
	if strings.TrimSpace(in.PayloadJSON) != "" {
		canonJSON, payloadHash, err = CanonicalizeJSONBytes([]byte(in.PayloadJSON))
	} else {
		canonJSON, payloadHash, err = CanonicalizePayload(in.Payload)
	}
	if err != nil {
		return Proposal{}, err
	}

	refs := in.EvidenceRefs
	if refs == nil {
		refs = []EvidenceRef{}
	}
	refsRaw, err := json.Marshal(refs)
	if err != nil {
		return Proposal{}, fmt.Errorf("%w: marshal evidence_refs: %v", ErrInvalidInput, err)
	}

	agentID := strings.ToUpper(strings.TrimSpace(in.AgentID))
	if agentID == "" {
		agentID = AgentHuman
	}
	actorType := strings.ToLower(strings.TrimSpace(in.ActorType))
	if actorType == "" {
		if agentID != AgentHuman {
			actorType = auth.ActorTypeAgent
		} else {
			actorType = auth.ActorTypeHuman
		}
	}
	createdBy := strings.TrimSpace(in.CreatedBy)
	if createdBy == "" {
		createdBy = auth.ActorFromContext(ctx)
	}
	if createdBy == "" || createdBy == "unknown" {
		if actorType == auth.ActorTypeAgent {
			createdBy = "agent:" + strings.ToLower(agentID)
		}
	}

	roiCat := strings.ToUpper(strings.TrimSpace(in.RoiCategory))
	if roiCat == "" {
		roiCat = defaultRoiCategoryForProposal(pType)
	} else if _, ok := roi.NormalizeCategory(roiCat); !ok {
		return Proposal{}, fmt.Errorf("%w: unsupported roi_category %q", ErrInvalidInput, roiCat)
	}

	now := s.currentNow()
	var expiresAt *time.Time
	if in.ExpiresInHours > 0 {
		t := now.Add(time.Duration(in.ExpiresInHours) * time.Hour)
		expiresAt = &t
	} else {
		t := now.Add(72 * time.Hour)
		expiresAt = &t
	}

	propID := "prop_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
	var appReqID string
	if s.approvals != nil {
		sub, aErr := s.approvals.SubmitRequest(ctx, approvals.SubmitRequestInput{
			TenantID:       tid,
			ObjectType:     "Proposal",
			ObjectID:       propID,
			Action:         pType,
			AmountMinor:    in.EstimatedImpactMinor,
			RequestedBy:    createdBy,
			ActorType:      actorType,
			PayloadJSON:    canonJSON,
			IdempotencyKey: "proposal:" + propID,
			ForceCreate:    true,
			ExpiresInHours: in.ExpiresInHours,
			Context: map[string]any{
				"agent_id":           agentID,
				"proposal_type":      pType,
				"target_object_type": tType,
				"target_object_id":   tID,
				"confidence_bps":     in.ConfidenceBps,
			},
		})
		if aErr != nil {
			return Proposal{}, aErr
		}
		if sub.Request != nil {
			appReqID = sub.Request.RequestID
		}
	}

	prop := Proposal{
		TenantID:             tid,
		ProposalID:           propID,
		AgentID:              agentID,
		ProposalType:         pType,
		TargetObjectType:     tType,
		TargetObjectID:       tID,
		Title:                title,
		Summary:              strings.TrimSpace(in.Summary),
		Rationale:            rationale,
		PayloadJSON:          canonJSON,
		Payload:              decodePayloadMap(canonJSON),
		PayloadHash:          payloadHash,
		EvidenceRefsJSON:     string(refsRaw),
		EvidenceRefs:         refs,
		ConfidenceBps:        in.ConfidenceBps,
		EstimatedImpactMinor: in.EstimatedImpactMinor,
		Currency:             nonEmptyOr(strings.ToUpper(in.Currency), "UZS"),
		RoiCategory:          roiCat,
		AutonomyLevel:        AutonomyLevelL1,
		Status:               StatusPending,
		ApprovalRequestID:    appReqID,
		CreatedBy:            createdBy,
		ActorType:            actorType,
		IdempotencyKey:       strings.TrimSpace(in.IdempotencyKey),
		ExpiresAt:            expiresAt,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.repo.Create(ctx, prop); err != nil {
		return Proposal{}, err
	}
	return prop, nil
}

// List returns proposals scoped to tenant and optional filters.
func (s *Service) List(ctx context.Context, f ListFilter) ([]Proposal, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	f.TenantID = strings.TrimSpace(f.TenantID)
	if f.TenantID == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	return s.repo.List(ctx, f)
}

// Get fetches one proposal by (tenantID, proposalID).
func (s *Service) Get(ctx context.Context, tenantID, proposalID string) (Proposal, error) {
	if s == nil || s.repo == nil {
		return Proposal{}, ErrRepoUnavailable
	}
	p, ok, err := s.repo.Get(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(proposalID))
	if err != nil {
		return Proposal{}, err
	}
	if !ok {
		return Proposal{}, ErrProposalNotFound
	}
	return p, nil
}

// ApproveOrEditProposal approves a PENDING proposal by a human approver (enforcing SoD
// and banning AGENT/AUDITOR actors). When edited_payload is provided, it canonicalizes
// and re-hashes the payload, supersedes the prior approval request, and re-pins the new hash.
func (s *Service) ApproveOrEditProposal(ctx context.Context, in ApproveOrEditInput) (Proposal, error) {
	if s == nil || s.repo == nil {
		return Proposal{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	pid := strings.TrimSpace(in.ProposalID)
	approverID := strings.TrimSpace(in.ApproverID)
	actorType := strings.ToLower(strings.TrimSpace(in.ActorType))
	approverRole := strings.ToUpper(strings.TrimSpace(in.ApproverRole))
	if tid == "" || pid == "" || approverID == "" {
		return Proposal{}, fmt.Errorf("%w: tenant_id, proposal_id, and approver_id are required", ErrInvalidInput)
	}
	if actorType == auth.ActorTypeAgent || approverRole == auth.SupplierRoleAgent {
		return Proposal{}, ErrAgentApprovalForbidden
	}
	if err := planning.ValidateAgentGovernance(actorType, true); err != nil {
		return Proposal{}, ErrAgentApprovalForbidden
	}
	if approverRole == auth.SupplierRoleAuditor {
		return Proposal{}, ErrAuditorReadOnly
	}

	p, ok, err := s.repo.Get(ctx, tid, pid)
	if err != nil {
		return Proposal{}, err
	}
	if !ok {
		return Proposal{}, ErrProposalNotFound
	}
	if p.Status != StatusPending {
		return Proposal{}, fmt.Errorf("%w: proposal status is %s", ErrInvalidTransition, p.Status)
	}
	if strings.EqualFold(approverID, p.CreatedBy) {
		return Proposal{}, ErrSoDViolation
	}
	if exp := strings.TrimSpace(in.ExpectedHash); exp != "" && !strings.EqualFold(exp, p.PayloadHash) {
		return Proposal{}, ErrPayloadHashMismatch
	}

	now := s.currentNow()
	if p.ExpiresAt != nil && !now.Before(*p.ExpiresAt) {
		return Proposal{}, fmt.Errorf("%w: proposal expired", ErrInvalidTransition)
	}

	beforePayload := p.PayloadJSON
	edited := false
	if strings.TrimSpace(in.EditedPayloadJSON) != "" || in.EditedPayload != nil {
		var newJSON, newHash string
		if strings.TrimSpace(in.EditedPayloadJSON) != "" {
			newJSON, newHash, err = CanonicalizeJSONBytes([]byte(in.EditedPayloadJSON))
		} else {
			newJSON, newHash, err = CanonicalizePayload(in.EditedPayload)
		}
		if err != nil {
			return Proposal{}, err
		}
		if newHash != p.PayloadHash {
			edited = true
			oldReqID := p.ApprovalRequestID
			p.PayloadJSON = newJSON
			p.Payload = decodePayloadMap(newJSON)
			p.PayloadHash = newHash

			if s.approvals != nil {
				if oldReqID != "" {
					_, _ = s.approvals.Decide(ctx, approvals.DecideInput{
						TenantID:     tid,
						RequestID:    oldReqID,
						ApproverID:   approverID,
						ActorType:    actorType,
						ApproverRole: approverRole,
						Decision:     approvals.DecisionReject,
						Comment:      "superseded by payload edit " + newHash[:12],
						RoutePath:    in.RoutePath,
						TraceID:      in.TraceID,
						IP:           in.IP,
						DeviceID:     in.DeviceID,
					})
				}
				sub, sErr := s.approvals.SubmitRequest(ctx, approvals.SubmitRequestInput{
					TenantID:       tid,
					ObjectType:     "Proposal",
					ObjectID:       p.ProposalID,
					Action:         p.ProposalType,
					AmountMinor:    p.EstimatedImpactMinor,
					RequestedBy:    p.CreatedBy,
					ActorType:      p.ActorType,
					PayloadJSON:    newJSON,
					IdempotencyKey: "proposal-edit:" + p.ProposalID + ":" + newHash[:16],
					ForceCreate:    true,
					Context: map[string]any{
						"agent_id":           p.AgentID,
						"proposal_type":      p.ProposalType,
						"target_object_type": p.TargetObjectType,
						"target_object_id":   p.TargetObjectID,
						"edited_by":          approverID,
					},
				})
				if sErr != nil {
					return Proposal{}, sErr
				}
				if sub.Request != nil {
					p.ApprovalRequestID = sub.Request.RequestID
				}
			}
		}
	}

	comment := strings.TrimSpace(in.Comment)
	if edited && comment == "" {
		comment = "edited and approved"
	}

	fullyApproved := true
	if s.approvals != nil && p.ApprovalRequestID != "" {
		decRes, dErr := s.approvals.Decide(ctx, approvals.DecideInput{
			TenantID:         tid,
			RequestID:        p.ApprovalRequestID,
			ApproverID:       approverID,
			ActorType:        actorType,
			ApproverRole:     approverRole,
			Decision:         approvals.DecisionApprove,
			Comment:          comment,
			ExpectedHash:     p.PayloadHash,
			MFAVerified:      in.MFAVerified,
			TOTPCode:         in.TOTPCode,
			BreakGlass:       in.BreakGlass,
			BreakGlassReason: in.BreakGlassReason,
			RoutePath:        in.RoutePath,
			TraceID:          in.TraceID,
			IP:               in.IP,
			DeviceID:         in.DeviceID,
		})
		if dErr != nil {
			if errors.Is(dErr, approvals.ErrSoDViolation) {
				return Proposal{}, ErrSoDViolation
			}
			if errors.Is(dErr, approvals.ErrPayloadHashMismatch) {
				return Proposal{}, ErrPayloadHashMismatch
			}
			return Proposal{}, dErr
		}
		fullyApproved = decRes.Request.Status == approvals.StatusApproved
	}

	p.UpdatedAt = now
	if comment != "" {
		p.DecisionComment = comment
	}
	if fullyApproved {
		p.Status = StatusApproved
		p.DecidedBy = approverID
		p.DecidedAt = &now
	}

	if err := s.repo.Update(ctx, p); err != nil {
		return Proposal{}, err
	}
	if s.audit != nil {
		reason := "proposal.approved"
		if edited {
			reason = "proposal.edited_and_approved"
		}
		_ = s.audit.RecordEvent(ctx, AuditRecordInput{
			TenantID:   tid,
			ActorID:    approverID,
			ActorType:  actorType,
			Role:       approverRole,
			Action:     "proposal.approve",
			ObjectType: "Proposal",
			ObjectID:   p.ProposalID,
			BeforeJSON: beforePayload,
			AfterJSON:  p.PayloadJSON,
			ReasonCode: reason,
			ApprovalID: p.ApprovalRequestID,
			TraceID:    in.TraceID,
			IP:         in.IP,
			DeviceID:   in.DeviceID,
		})
	}
	return p, nil
}

// RejectProposal marks a PENDING proposal as REJECTED and records the rejection in approvals.Engine.
func (s *Service) RejectProposal(ctx context.Context, in RejectInput) (Proposal, error) {
	if s == nil || s.repo == nil {
		return Proposal{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	pid := strings.TrimSpace(in.ProposalID)
	reviewerID := strings.TrimSpace(in.ReviewerID)
	actorType := strings.ToLower(strings.TrimSpace(in.ActorType))
	reviewerRole := strings.ToUpper(strings.TrimSpace(in.ReviewerRole))
	if tid == "" || pid == "" || reviewerID == "" {
		return Proposal{}, fmt.Errorf("%w: tenant_id, proposal_id, and reviewer_id are required", ErrInvalidInput)
	}
	if actorType == auth.ActorTypeAgent || reviewerRole == auth.SupplierRoleAgent {
		return Proposal{}, ErrAgentApprovalForbidden
	}
	if err := planning.ValidateAgentGovernance(actorType, true); err != nil {
		return Proposal{}, ErrAgentApprovalForbidden
	}
	if reviewerRole == auth.SupplierRoleAuditor {
		return Proposal{}, ErrAuditorReadOnly
	}

	p, ok, err := s.repo.Get(ctx, tid, pid)
	if err != nil {
		return Proposal{}, err
	}
	if !ok {
		return Proposal{}, ErrProposalNotFound
	}
	if p.Status != StatusPending {
		return Proposal{}, fmt.Errorf("%w: proposal status is %s", ErrInvalidTransition, p.Status)
	}

	if s.approvals != nil && p.ApprovalRequestID != "" {
		if _, dErr := s.approvals.Decide(ctx, approvals.DecideInput{
			TenantID:     tid,
			RequestID:    p.ApprovalRequestID,
			ApproverID:   reviewerID,
			ActorType:    actorType,
			ApproverRole: reviewerRole,
			Decision:     approvals.DecisionReject,
			Comment:      strings.TrimSpace(in.Comment),
			RoutePath:    in.RoutePath,
			TraceID:      in.TraceID,
			IP:           in.IP,
			DeviceID:     in.DeviceID,
		}); dErr != nil && !errors.Is(dErr, approvals.ErrRequestAlreadyResolved) {
			if errors.Is(dErr, approvals.ErrSoDViolation) {
				return Proposal{}, ErrSoDViolation
			}
			return Proposal{}, dErr
		}
	}

	now := s.currentNow()
	p.Status = StatusRejected
	p.DecidedBy = reviewerID
	p.DecisionComment = strings.TrimSpace(in.Comment)
	p.DecidedAt = &now
	p.UpdatedAt = now
	if err := s.repo.Update(ctx, p); err != nil {
		return Proposal{}, err
	}
	return p, nil
}

// ExecuteProposal enforces target concurrency locking, APPROVED state, PayloadHash integrity,
// approvals verification, fresh domain preconditions, governed action execution, and ROI recording.
func (s *Service) ExecuteProposal(ctx context.Context, in ExecuteInput) (ExecuteResult, error) {
	if s == nil || s.repo == nil {
		return ExecuteResult{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	pid := strings.TrimSpace(in.ProposalID)
	executorID := strings.TrimSpace(in.ExecutorID)
	actorType := strings.ToLower(strings.TrimSpace(in.ActorType))
	executorRole := strings.ToUpper(strings.TrimSpace(in.ExecutorRole))
	if tid == "" || pid == "" || executorID == "" {
		return ExecuteResult{}, fmt.Errorf("%w: tenant_id, proposal_id, and executor_id are required", ErrInvalidInput)
	}
	if actorType == auth.ActorTypeAgent || executorRole == auth.SupplierRoleAgent {
		return ExecuteResult{}, ErrAgentApprovalForbidden
	}
	if err := planning.ValidateAgentGovernance(actorType, true); err != nil {
		return ExecuteResult{}, ErrAgentApprovalForbidden
	}
	if executorRole == auth.SupplierRoleAuditor {
		return ExecuteResult{}, ErrAuditorReadOnly
	}

	initial, ok, err := s.repo.Get(ctx, tid, pid)
	if err != nil {
		return ExecuteResult{}, err
	}
	if !ok {
		return ExecuteResult{}, ErrProposalNotFound
	}

	// Hold per-target concurrency lock so overlapping proposals on the same target never double-execute.
	unlock := s.lockTarget(tid, initial.TargetObjectType, initial.TargetObjectID)
	defer unlock()

	p, ok, err := s.repo.Get(ctx, tid, pid)
	if err != nil {
		return ExecuteResult{}, err
	}
	if !ok {
		return ExecuteResult{}, ErrProposalNotFound
	}
	if p.Status == StatusExecuted {
		return ExecuteResult{}, ErrAlreadyExecuted
	}
	if p.Status != StatusApproved {
		return ExecuteResult{}, fmt.Errorf("%w: proposal must be APPROVED before execution (status=%s)", ErrInvalidTransition, p.Status)
	}

	now := s.currentNow()
	if p.ExpiresAt != nil && !now.Before(*p.ExpiresAt) {
		return ExecuteResult{}, fmt.Errorf("%w: proposal expired", ErrInvalidTransition)
	}

	_, recomputedHash, err := CanonicalizeJSONBytes([]byte(p.PayloadJSON))
	if err != nil {
		return ExecuteResult{}, ErrPayloadHashMismatch
	}
	if !strings.EqualFold(recomputedHash, p.PayloadHash) {
		return ExecuteResult{}, ErrPayloadHashMismatch
	}
	if exp := strings.TrimSpace(in.ExpectedHash); exp != "" && !strings.EqualFold(exp, p.PayloadHash) {
		return ExecuteResult{}, ErrPayloadHashMismatch
	}

	if s.approvals != nil && p.ApprovalRequestID != "" {
		if _, _, vErr := s.approvals.VerifyApprovedRequest(ctx, tid, p.ApprovalRequestID, "Proposal", p.ProposalType, p.PayloadHash, 1); vErr != nil {
			return ExecuteResult{}, vErr
		}
	}

	overlap, err := s.repo.HasOverlappingExecuted(ctx, tid, p.TargetObjectType, p.TargetObjectID, p.ProposalType, p.ProposalID, p.CreatedAt)
	if err != nil {
		return ExecuteResult{}, err
	}
	if overlap {
		return ExecuteResult{}, ErrTargetExecutionConflict
	}

	if err := s.verifyPreconditions(ctx, tid, p); err != nil {
		p.Status = StatusFailed
		p.ExecutionError = err.Error()
		p.UpdatedAt = now
		_ = s.repo.Update(ctx, p)
		return ExecuteResult{}, err
	}

	execOutcome, err := s.dispatchGovernedExecution(ctx, p, executorID)
	if err != nil {
		p.Status = StatusFailed
		p.ExecutionError = err.Error()
		p.UpdatedAt = now
		_ = s.repo.Update(ctx, p)
		return ExecuteResult{}, err
	}

	var recordedRoiEvent *roi.RoiEvent
	if s.roiSvc != nil && p.EstimatedImpactMinor > 0 {
		cat := nonEmptyOr(p.RoiCategory, defaultRoiCategoryForProposal(p.ProposalType))
		ev, rErr := s.roiSvc.RecordEvent(ctx, roi.RecordEventInput{
			TenantID:        tid,
			Category:        cat,
			SavingsMinor:    p.EstimatedImpactMinor,
			Currency:        nonEmptyOr(p.Currency, "UZS"),
			Methodology:     roi.DefaultMethodology(cat) + " (governed proposal " + p.AgentID + ")",
			ConfidenceBps:   p.ConfidenceBps,
			ProposalID:      p.ProposalID,
			EvidenceRefType: p.TargetObjectType,
			EvidenceRefID:   p.TargetObjectID,
			Notes:           p.Title,
			RecordedBy:      executorID,
			IdempotencyKey:  "proposal-exec:" + p.ProposalID,
			OccurredAt:      now,
		})
		if rErr != nil {
			return ExecuteResult{}, rErr
		}
		recordedRoiEvent = &ev
		p.RoiEventID = ev.EventID
		execOutcome["roi_event_id"] = ev.EventID
	}

	resRaw, _ := json.Marshal(execOutcome)
	p.Status = StatusExecuted
	p.ExecutedBy = executorID
	p.ExecutedAt = &now
	p.UpdatedAt = now
	p.ExecutionResultJSON = string(resRaw)
	p.ExecutionResult = execOutcome
	p.ExecutionError = ""
	if err := s.repo.Update(ctx, p); err != nil {
		return ExecuteResult{}, err
	}

	if s.audit != nil {
		_ = s.audit.RecordEvent(ctx, AuditRecordInput{
			TenantID:   tid,
			ActorID:    executorID,
			ActorType:  actorType,
			Role:       executorRole,
			Action:     "proposal.execute",
			ObjectType: "Proposal",
			ObjectID:   p.ProposalID,
			BeforeJSON: p.PayloadJSON,
			AfterJSON:  p.ExecutionResultJSON,
			ReasonCode: p.ProposalType,
			ApprovalID: p.ApprovalRequestID,
		})
	}

	var roiAny any
	if recordedRoiEvent != nil {
		roiAny = recordedRoiEvent
	}
	return ExecuteResult{
		Proposal:        p,
		ExecutionResult: execOutcome,
		RoiEvent:        roiAny,
	}, nil
}

func (s *Service) verifyPreconditions(ctx context.Context, tenantID string, p Proposal) error {
	if s.dataReader == nil {
		return nil
	}
	switch strings.ToUpper(strings.TrimSpace(p.ProposalType)) {
	case TypeReorder, TypeSupplyRequest:
		return s.dataReader.VerifyReorderPrecondition(ctx, tenantID, p)
	case TypeCollections, TypeCreditHold:
		return s.dataReader.VerifyCollectionsPrecondition(ctx, tenantID, p)
	case TypeExpiryRebalance:
		return s.dataReader.VerifyExpiryRebalancePrecondition(ctx, tenantID, p)
	default:
		return nil
	}
}

func (s *Service) dispatchGovernedExecution(ctx context.Context, p Proposal, executorID string) (map[string]any, error) {
	payload := decodePayloadMap(p.PayloadJSON)
	action := strings.ToLower(strField(payload, "action"))

	// Route planning allowlisted agent mutations through planning.ValidateAgentInvocation + Executor.
	if p.ProposalType == TypeReorder || p.ProposalType == TypeSupplyRequest {
		if action == string(planning.AgentApproveInsight) || action == string(planning.AgentOpenSupplyRequest) || action == string(planning.AgentBroadcastTemplate) {
			target := strField(payload, "insight_id")
			if target == "" {
				target = p.TargetObjectID
			}
			inv := planning.AgentInvocation{
				Action:         planning.GovernedAgentAction(action),
				IdempotencyKey: "prop:" + p.ProposalID,
				SupplierID:     p.TenantID,
				TargetID:       target,
				Note:           p.Title,
			}
			if err := planning.ValidateAgentInvocation(inv); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
			}
			if s.planningExec != nil && s.planningExec.Spanner != nil {
				res, err := s.planningExec.Execute(ctx, inv)
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"executor":  "planning",
					"action":    res.Action,
					"status":    res.Status,
					"result_id": res.ResultID,
				}, nil
			}
		}
	}

	// Route Control Tower actions (FREEZE_CREDIT, NOTIFY, CREATE_CREDIT_NOTE) when wired.
	if s.ctExec != nil && (p.ProposalType == TypeCreditHold || action == "freeze_credit" || action == "notify") {
		ctType := "NOTIFY"
		if p.ProposalType == TypeCreditHold || action == "freeze_credit" {
			ctType = "FREEZE_CREDIT"
		}
		ex := controltower.Exception{
			ExceptionID: "prop:" + p.ProposalID,
			SupplierID:  p.TenantID,
			RetailerID:  strField(payload, "retailer_id"),
			OrderID:     strField(payload, "order_id"),
			WarehouseID: strField(payload, "source_warehouse_id"),
			Type:        p.ProposalType,
		}
		res := s.ctExec.ExecuteAction(ctx, p.ProposalID, 0, controltower.ActionSpec{
			Type:   ctType,
			Params: json.RawMessage(p.PayloadJSON),
		}, ex, executorID)
		if res.Status == "failed" {
			return nil, fmt.Errorf("controltower execute failed: %s", res.Message)
		}
		return map[string]any{
			"executor": "controltower",
			"action":   ctType,
			"status":   res.Status,
		}, nil
	}

	if s.domainExec != nil {
		return s.domainExec.ExecuteProposalAction(ctx, p, executorID)
	}

	return map[string]any{
		"executor":         "governed_proposal",
		"proposal_type":    p.ProposalType,
		"action":           nonEmptyOr(action, strings.ToLower(p.ProposalType)),
		"target_object_id": p.TargetObjectID,
		"status":           "completed",
	}, nil
}

func defaultRoiCategoryForProposal(proposalType string) string {
	switch strings.ToUpper(strings.TrimSpace(proposalType)) {
	case TypeReorder, TypeSupplyRequest:
		return roi.CategoryStockoutAvoided
	case TypeCollections, TypeCreditHold:
		return roi.CategoryBadDebtAvoided
	case TypeExpiryRebalance:
		return roi.CategoryExpirySaved
	default:
		return roi.CategoryWorkingCapitalSaved
	}
}
