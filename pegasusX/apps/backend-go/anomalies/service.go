package anomalies

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Service orchestrates §7.7 Anomaly & Fraud Detector Catalogue scanning,
// deduplicated AnomalyCases management, and optional escalation into L1 Proposals.
type Service struct {
	repo      Repository
	audit     AuditRecorder
	proposals ProposalCreator
	now       func() time.Time
}

// NewService constructs an anomalies Service backed by a custom Repository (e.g. MemoryRepository).
func NewService(repo Repository, cfg ServiceConfig) *Service {
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	return &Service{
		repo:      repo,
		audit:     cfg.Audit,
		proposals: cfg.Proposals,
		now:       nowFn,
	}
}

// SetAuditRecorder attaches an immutable AuditRecorder.
func (s *Service) SetAuditRecorder(rec AuditRecorder) {
	if s != nil {
		s.audit = rec
	}
}

// SetProposalCreator attaches an L1 ProposalCreator for anomaly escalations.
func (s *Service) SetProposalCreator(pc ProposalCreator) {
	if s != nil {
		s.proposals = pc
	}
}

func (s *Service) currentNow() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

// Catalogue returns the 5 deterministic §7.7 detector rules and default integer thresholds.
func (s *Service) Catalogue() []CatalogueRule {
	return DefaultCatalogue()
}

// Scan executes deterministic integer-scaled detectors in a read-only Spanner snapshot,
// persists deduplicated AnomalyCases, buffers anomaly.case.detected outbox events, and records AuditEvents.
func (s *Service) Scan(ctx context.Context, in ScanInput) (ScanResult, error) {
	if s == nil || s.repo == nil {
		return ScanResult{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	if tid == "" {
		return ScanResult{}, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	if in.ThresholdBps != nil && *in.ThresholdBps < 0 {
		return ScanResult{}, fmt.Errorf("%w: threshold_bps must be non-negative", ErrInvalidInput)
	}
	if in.ThresholdMinor != nil && *in.ThresholdMinor < 0 {
		return ScanResult{}, fmt.Errorf("%w: threshold_minor must be non-negative", ErrInvalidInput)
	}
	for _, rc := range in.RuleCodes {
		if strings.TrimSpace(rc) == "" {
			continue
		}
		if _, ok := GetRule(rc); !ok {
			return ScanResult{}, fmt.Errorf("%w: unknown rule_code %q", ErrInvalidInput, rc)
		}
	}

	ds, err := s.repo.LoadDetectorDataset(ctx, tid, strings.TrimSpace(in.EntityID))
	if err != nil {
		return ScanResult{}, err
	}

	now := s.currentNow()
	findings, scannedRules := EvaluateDetectors(ds, EvalOptions{
		RuleCodes:      in.RuleCodes,
		ThresholdBps:   in.ThresholdBps,
		ThresholdMinor: in.ThresholdMinor,
		Now:            now,
	})

	candidates := make([]AnomalyCase, 0, len(findings))
	for _, f := range findings {
		detectedAt := f.DetectedAt.UTC()
		if detectedAt.IsZero() {
			detectedAt = now
		}
		evJSON := fmtEvidenceJSON(f.Evidence)
		candidates = append(candidates, AnomalyCase{
			TenantID:       tid,
			CaseID:         "anom_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24],
			RuleCode:       f.RuleCode,
			Category:       f.Category,
			Severity:       f.Severity,
			Status:         StatusOpen,
			ObjectType:     f.ObjectType,
			ObjectID:       f.ObjectID,
			ExposureMinor:  f.ExposureMinor,
			ScoreBps:       f.ScoreBps,
			EvidenceJSON:   evJSON,
			Evidence:       f.Evidence,
			PlaybookAction: f.PlaybookAction,
			DetectedAt:     detectedAt,
			UpdatedAt:      now,
		})
	}

	actorID := nonEmpty(in.ActorID, "system:anomaly-scanner")
	actorType := nonEmpty(in.ActorType, auth.ActorTypeHuman)
	afterRaw, _ := json.Marshal(map[string]any{
		"scanned_rules":   scannedRules,
		"matched_count":   len(candidates),
		"threshold_bps":   in.ThresholdBps,
		"threshold_minor": in.ThresholdMinor,
	})
	auditIn := AuditRecordInput{
		TenantID:   tid,
		ActorID:    actorID,
		ActorType:  actorType,
		Role:       in.ActorRole,
		Action:     "anomalies.scan",
		ObjectType: "AnomalyCase",
		ObjectID:   tid,
		AfterJSON:  string(afterRaw),
		TraceID:    in.TraceID,
		IP:         in.IP,
		DeviceID:   in.DeviceID,
	}

	created, deduped, err := s.repo.SaveDetectedCases(ctx, tid, candidates, s.audit, auditIn)
	if err != nil {
		return ScanResult{}, err
	}
	if created == nil {
		created = []AnomalyCase{}
	}
	if scannedRules == nil {
		scannedRules = []string{}
	}
	return ScanResult{
		TenantID:          tid,
		Cases:             created,
		CreatedCount:      len(created),
		DeduplicatedCount: deduped,
		MatchedCount:      len(candidates),
		ScannedRules:      scannedRules,
		ScanAt:            now,
	}, nil
}

// ListCases returns filtered AnomalyCases for a tenant.
func (s *Service) ListCases(ctx context.Context, f ListFilter) ([]AnomalyCase, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	f.TenantID = strings.TrimSpace(f.TenantID)
	if f.TenantID == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	if st := strings.ToUpper(strings.TrimSpace(f.Status)); st != "" {
		switch st {
		case StatusOpen, StatusResolved, StatusDismissed, StatusEscalated:
			f.Status = st
		default:
			return nil, fmt.Errorf("%w: invalid status %q", ErrInvalidInput, f.Status)
		}
	}
	f.Limit = clampLimit(f.Limit)
	return s.repo.ListCases(ctx, f)
}

// GetCase retrieves a single AnomalyCase by tenant and case ID.
func (s *Service) GetCase(ctx context.Context, tenantID, caseID string) (AnomalyCase, error) {
	if s == nil || s.repo == nil {
		return AnomalyCase{}, ErrRepoUnavailable
	}
	c, ok, err := s.repo.GetCase(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(caseID))
	if err != nil {
		return AnomalyCase{}, err
	}
	if !ok {
		return AnomalyCase{}, ErrNotFound
	}
	return c, nil
}

// ResolveCase transitions an AnomalyCase to RESOLVED, DISMISSED, or ESCALATED,
// optionally creating a governed L1 Proposal when escalated with create_proposal=true.
func (s *Service) ResolveCase(ctx context.Context, in ResolveInput) (ResolveResult, error) {
	if s == nil || s.repo == nil {
		return ResolveResult{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	cid := strings.TrimSpace(in.CaseID)
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	note := strings.TrimSpace(in.ResolutionNote)

	if tid == "" || cid == "" {
		return ResolveResult{}, fmt.Errorf("%w: tenant_id and case_id are required", ErrInvalidInput)
	}
	switch status {
	case StatusResolved, StatusDismissed, StatusEscalated:
	default:
		return ResolveResult{}, fmt.Errorf("%w: status must be RESOLVED, DISMISSED, or ESCALATED", ErrInvalidInput)
	}
	if note == "" {
		return ResolveResult{}, fmt.Errorf("%w: resolution_note is required", ErrInvalidInput)
	}
	if strings.EqualFold(in.ActorRole, auth.SupplierRoleAuditor) {
		return ResolveResult{}, fmt.Errorf("%w: AUDITOR cannot resolve anomaly cases", ErrForbidden)
	}
	if strings.EqualFold(in.ActorRole, auth.SupplierRoleAgent) || strings.EqualFold(in.ActorType, auth.ActorTypeAgent) {
		return ResolveResult{}, fmt.Errorf("%w: AGENT cannot resolve anomaly cases", ErrForbidden)
	}

	cur, found, err := s.repo.GetCase(ctx, tid, cid)
	if err != nil {
		return ResolveResult{}, err
	}
	if !found {
		return ResolveResult{}, ErrNotFound
	}
	if cur.Status == StatusResolved || cur.Status == StatusDismissed {
		return ResolveResult{}, fmt.Errorf("%w: anomaly case %s is already %s", ErrConflict, cid, cur.Status)
	}

	var createdProp *CreatedProposal
	var proposalID string
	if status == StatusEscalated && in.CreateProposal {
		if cur.ProposalID != "" {
			proposalID = cur.ProposalID
		} else {
			if s.proposals == nil {
				return ResolveResult{}, ErrProposalUnavailable
			}
			action := nonEmpty(in.PlaybookAction, cur.PlaybookAction, "INVESTIGATE_ANOMALY")
			propType := "ANOMALY_ESCALATION"
			if cur.RuleCode == RuleBasketSpikeSharedINN {
				propType = "CREDIT_HOLD"
			}
			impact := cur.ExposureMinor
			if impact < 0 {
				impact = 0
			}
			prop, pErr := s.proposals.CreateProposal(ctx, ProposalCreateInput{
				TenantID:         tid,
				AgentID:          "HUMAN",
				ProposalType:     propType,
				TargetObjectType: cur.ObjectType,
				TargetObjectID:   cur.ObjectID,
				Title:            fmt.Sprintf("[%s] Escalate %s %s", cur.RuleCode, cur.ObjectType, cur.ObjectID),
				Summary:          note,
				Rationale:        fmt.Sprintf("Anomaly case %s (%s, score %d bps): %s", cur.CaseID, cur.RuleCode, cur.ScoreBps, note),
				Payload: map[string]any{
					"case_id":         cur.CaseID,
					"rule_code":       cur.RuleCode,
					"category":        cur.Category,
					"severity":        cur.Severity,
					"object_type":     cur.ObjectType,
					"object_id":       cur.ObjectID,
					"exposure_minor":  cur.ExposureMinor,
					"score_bps":       cur.ScoreBps,
					"playbook_action": action,
					"resolution_note": note,
				},
				EvidenceRefs: []ProposalEvidenceRef{
					{Type: "AnomalyCase", ID: cur.CaseID, Label: cur.RuleCode},
					{Type: cur.ObjectType, ID: cur.ObjectID, Label: cur.Category},
				},
				ConfidenceBps:        clampBps(cur.ScoreBps),
				EstimatedImpactMinor: impact,
				Currency:             "UZS",
				CreatedBy:            nonEmpty(in.ActorID, auth.ActorFromContext(ctx), "human"),
				ActorType:            nonEmpty(in.ActorType, auth.ActorTypeHuman),
				IdempotencyKey:       "anomaly-escalate:" + cur.CaseID,
			})
			if pErr != nil {
				return ResolveResult{}, fmt.Errorf("create escalation proposal: %w", pErr)
			}
			createdProp = &prop
			proposalID = prop.ProposalID
		}
	}

	now := s.currentNow()
	actorID := nonEmpty(in.ActorID, auth.ActorFromContext(ctx), "system")
	actorType := nonEmpty(in.ActorType, auth.ActorTypeHuman)
	beforeRaw, _ := json.Marshal(cur)

	updated, err := s.repo.ResolveCase(ctx, tid, cid, func(latest AnomalyCase) (AnomalyCase, error) {
		if latest.Status == StatusResolved || latest.Status == StatusDismissed {
			return AnomalyCase{}, fmt.Errorf("%w: anomaly case %s is already %s", ErrConflict, cid, latest.Status)
		}
		latest.Status = status
		latest.ResolutionNote = note
		if act := strings.TrimSpace(in.PlaybookAction); act != "" {
			latest.PlaybookAction = act
		}
		if ass := strings.TrimSpace(in.AssignedTo); ass != "" {
			latest.AssignedTo = ass
		}
		if proposalID != "" {
			latest.ProposalID = proposalID
		}
		latest.ResolvedBy = actorID
		t := now
		latest.ResolvedAt = &t
		latest.UpdatedAt = now
		return latest, nil
	}, s.audit, func(u AnomalyCase) AuditRecordInput {
		afterRaw, _ := json.Marshal(u)
		return AuditRecordInput{
			TenantID:   tid,
			ActorID:    actorID,
			ActorType:  actorType,
			Role:       in.ActorRole,
			Action:     "anomalies.case.resolve",
			ObjectType: "AnomalyCase",
			ObjectID:   u.CaseID,
			BeforeJSON: string(beforeRaw),
			AfterJSON:  string(afterRaw),
			ReasonCode: status,
			TraceID:    in.TraceID,
			IP:         in.IP,
			DeviceID:   in.DeviceID,
		}
	}, now)
	if err != nil {
		return ResolveResult{}, err
	}

	return ResolveResult{
		Case:       updated,
		Proposal:   createdProp,
		ProposalID: nonEmpty(proposalID, updated.ProposalID),
	}, nil
}
