package periods

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/storage"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// Service orchestrates M1.1 accounting period lifecycle, live close checklists,
// immutable JSON+CSV snapshots, and dual-approver period reopen workflows.
type Service struct {
	repo           Repository
	audit          AuditRecorder
	approvalBridge ApprovalBridge
	snapshotSink   SnapshotSink
}

// NewService constructs a Spanner-backed period close service.
func NewService(client *spanner.Client) *Service {
	var repo Repository
	if client != nil {
		repo = NewSpannerRepository(client)
	}
	return &Service{
		repo:         repo,
		snapshotSink: DefaultSnapshotSink(nil),
	}
}

// NewServiceWithRepository constructs a Service with a custom Repository (e.g., MemoryRepository).
func NewServiceWithRepository(repo Repository) *Service {
	return &Service{
		repo:         repo,
		snapshotSink: DefaultSnapshotSink(nil),
	}
}

// Repo returns the configured Repository.
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

// SetApprovalBridge attaches the M1.2 CEL Approvals bridge for dual-approver period reopen.
func (s *Service) SetApprovalBridge(bridge ApprovalBridge) {
	if s != nil {
		s.approvalBridge = bridge
	}
}

// SetSnapshotSink overrides the immutable snapshot storage sink (e.g., GCS WORM sink or temp test dir).
func (s *Service) SetSnapshotSink(sink SnapshotSink) {
	if s != nil && sink != nil {
		s.snapshotSink = sink
	}
}

// ConfigureGCSSnapshotSink configures a create-only GCS sink when bucket is non-empty.
func (s *Service) ConfigureGCSSnapshotSink(gcsClient *storage.Client, bucket, prefix string) error {
	if s == nil {
		return nil
	}
	sink, err := NewGCSSnapshotSink(gcsClient, bucket, prefix)
	if err != nil {
		return err
	}
	s.snapshotSink = sink
	return nil
}

// OpenPeriod creates a new OPEN accounting period for (entityID, periodID) using the resolved entity timezone.
func (s *Service) OpenPeriod(ctx context.Context, entityID, periodID, explicitTZ, actorID, actorType, actorRole string) (Period, error) {
	if s == nil || s.repo == nil {
		return Period{}, ErrRepoUnavailable
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return Period{}, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	_, _, normPeriod, err := ValidatePeriodID(periodID)
	if err != nil {
		return Period{}, err
	}
	tzName, _, err := ResolveEntityTimezone(ctx, entityID, explicitTZ)
	if err != nil {
		return Period{}, err
	}
	if strings.TrimSpace(actorID) == "" {
		actorID = "system"
	}

	p := Period{
		EntityID: entityID,
		PeriodID: normPeriod,
		Status:   StatusOpen,
		Timezone: tzName,
	}
	EnrichPeriodWindow(&p)
	afterRaw, _ := json.Marshal(p)
	return s.repo.CreatePeriod(ctx, p, s.audit, AuditRecordInput{
		TenantID:   entityID,
		ActorID:    actorID,
		ActorType:  actorType,
		Role:       actorRole,
		Action:     "period.opened",
		ObjectType: "Period",
		ObjectID:   entityID + ":" + normPeriod,
		AfterJSON:  string(afterRaw),
	})
}

// ListPeriods lists persisted accounting periods for entityID.
func (s *Service) ListPeriods(ctx context.Context, entityID, statusFilter string) ([]Period, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return nil, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	return s.repo.ListPeriods(ctx, entityID, statusFilter)
}

// GetPeriod retrieves one Period record if persisted.
func (s *Service) GetPeriod(ctx context.Context, entityID, periodID string) (Period, bool, error) {
	if s == nil || s.repo == nil {
		return Period{}, false, ErrRepoUnavailable
	}
	_, _, normPeriod, err := ValidatePeriodID(periodID)
	if err != nil {
		return Period{}, false, err
	}
	return s.repo.GetPeriod(ctx, strings.TrimSpace(entityID), normPeriod)
}

// SoftClose transitions a period to SOFT_CLOSED (allows MANUAL/PERIOD_ADJUSTMENT, rolls operational events forward).
// If no row exists yet, it is initialized as OPEN first and transitioned atomically.
func (s *Service) SoftClose(ctx context.Context, entityID, periodID, explicitTZ, actorID, actorType, actorRole string) (Period, error) {
	if s == nil || s.repo == nil {
		return Period{}, ErrRepoUnavailable
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return Period{}, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	_, _, normPeriod, err := ValidatePeriodID(periodID)
	if err != nil {
		return Period{}, err
	}
	if strings.TrimSpace(actorID) == "" {
		return Period{}, fmt.Errorf("%w: actor_id is required", ErrInvalidInput)
	}

	tzName, _, err := ResolveEntityTimezone(ctx, entityID, explicitTZ)
	if err != nil {
		return Period{}, err
	}

	now := time.Now().UTC()
	return s.repo.TransitionPeriod(ctx, entityID, normPeriod, func(existing *Period, found bool) (PeriodTransition, error) {
		var next Period
		if found {
			switch existing.Status {
			case StatusOpen, StatusReopened:
				next = *existing
				if strings.TrimSpace(explicitTZ) != "" {
					next.Timezone = tzName
				}
			case StatusSoftClosed:
				return PeriodTransition{}, fmt.Errorf("%w: period %s is already SOFT_CLOSED", ErrInvalidStatusTransition, normPeriod)
			case StatusClosed:
				return PeriodTransition{}, fmt.Errorf("%w: period %s is already CLOSED (use reopen first)", ErrInvalidStatusTransition, normPeriod)
			default:
				return PeriodTransition{}, fmt.Errorf("%w: cannot soft-close period in status %s", ErrInvalidStatusTransition, existing.Status)
			}
		} else {
			next = Period{
				EntityID: entityID,
				PeriodID: normPeriod,
				Timezone: tzName,
			}
		}
		next.Status = StatusSoftClosed
		next.SoftClosedBy = actorID
		next.SoftClosedAt = &now
		return PeriodTransition{
			NextPeriod:   next,
			EventType:    events.EventPeriodSoftClosed,
			AutoInitOpen: !found,
		}, nil
	}, s.audit, AuditRecordInput{
		TenantID:   entityID,
		ActorID:    actorID,
		ActorType:  actorType,
		Role:       actorRole,
		Action:     events.EventPeriodSoftClosed,
		ObjectType: "Period",
		ObjectID:   entityID + ":" + normPeriod,
	})
}

// EvaluateChecklist runs all live close readiness checks for (entityID, periodID).
func (s *Service) EvaluateChecklist(ctx context.Context, entityID, periodID string) (ChecklistReport, error) {
	if s == nil || s.repo == nil {
		return ChecklistReport{}, ErrRepoUnavailable
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return ChecklistReport{}, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	_, _, normPeriod, err := ValidatePeriodID(periodID)
	if err != nil {
		return ChecklistReport{}, err
	}
	tzName := ""
	if existing, found, _ := s.repo.GetPeriod(ctx, entityID, normPeriod); found && strings.TrimSpace(existing.Timezone) != "" {
		tzName = existing.Timezone
	}
	if tzName == "" {
		resolvedTZ, _, tzErr := ResolveEntityTimezone(ctx, entityID, "")
		if tzErr != nil {
			return ChecklistReport{}, tzErr
		}
		tzName = resolvedTZ
	}
	return s.repo.EvaluateChecklist(ctx, entityID, normPeriod, tzName)
}

// Close verifies the live close checklist passes with zero blockers, writes a sealed WORM
// JSON+CSV+SHA256 snapshot, transitions the period to CLOSED, and emits period.closed + AuditEvents.
func (s *Service) Close(ctx context.Context, entityID, periodID, explicitTZ, actorID, actorType, actorRole string) (CloseResult, error) {
	if s == nil || s.repo == nil {
		return CloseResult{}, ErrRepoUnavailable
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return CloseResult{}, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	_, _, normPeriod, err := ValidatePeriodID(periodID)
	if err != nil {
		return CloseResult{}, err
	}
	if strings.TrimSpace(actorID) == "" {
		return CloseResult{}, fmt.Errorf("%w: actor_id is required", ErrInvalidInput)
	}

	existing, found, err := s.repo.GetPeriod(ctx, entityID, normPeriod)
	if err != nil {
		return CloseResult{}, err
	}
	if found && existing.Status == StatusClosed {
		return CloseResult{}, fmt.Errorf("%w: period %s is already CLOSED", ErrInvalidStatusTransition, normPeriod)
	}

	tzName := strings.TrimSpace(explicitTZ)
	if tzName != "" {
		if tzName, _, err = ResolveEntityTimezone(ctx, entityID, tzName); err != nil {
			return CloseResult{}, err
		}
	} else if found && strings.TrimSpace(existing.Timezone) != "" {
		tzName = existing.Timezone
	} else {
		if tzName, _, err = ResolveEntityTimezone(ctx, entityID, ""); err != nil {
			return CloseResult{}, err
		}
	}

	checklist, err := s.repo.EvaluateChecklist(ctx, entityID, normPeriod, tzName)
	if err != nil {
		return CloseResult{}, err
	}
	if !checklist.ReadyToClose {
		return CloseResult{Checklist: checklist}, fmt.Errorf("%w: %d blocking checklist item(s)", ErrChecklistFailed, checklist.BlockingCount)
	}

	sink := s.snapshotSink
	if sink == nil {
		sink = DefaultSnapshotSink(nil)
	}
	snap, err := WriteCloseSnapshot(ctx, s.repo, sink, entityID, normPeriod, tzName, actorID)
	if err != nil {
		return CloseResult{}, err
	}
	checklistRaw, err := json.Marshal(checklist)
	if err != nil {
		return CloseResult{}, fmt.Errorf("marshal close checklist: %w", err)
	}

	now := time.Now().UTC()
	closedPeriod, err := s.repo.TransitionPeriod(ctx, entityID, normPeriod, func(cur *Period, curFound bool) (PeriodTransition, error) {
		var next Period
		if curFound {
			if cur.Status == StatusClosed {
				return PeriodTransition{}, fmt.Errorf("%w: period %s is already CLOSED", ErrInvalidStatusTransition, normPeriod)
			}
			next = *cur
			if strings.TrimSpace(explicitTZ) != "" {
				next.Timezone = tzName
			}
		} else {
			next = Period{
				EntityID: entityID,
				PeriodID: normPeriod,
				Timezone: tzName,
			}
		}
		next.Status = StatusClosed
		next.ClosedBy = actorID
		next.ClosedAt = &now
		next.SnapshotURI = snap.SnapshotURI
		next.SnapshotHash = snap.SnapshotHash
		next.ChecklistJSON = string(checklistRaw)
		return PeriodTransition{
			NextPeriod: next,
			EventType:  events.EventPeriodClosed,
		}, nil
	}, s.audit, AuditRecordInput{
		TenantID:   entityID,
		ActorID:    actorID,
		ActorType:  actorType,
		Role:       actorRole,
		Action:     events.EventPeriodClosed,
		ObjectType: "Period",
		ObjectID:   entityID + ":" + normPeriod,
		AfterJSON:  snap.SnapshotHash,
	})
	if err != nil {
		return CloseResult{}, err
	}
	return CloseResult{
		Period:    closedPeriod,
		Checklist: checklist,
		Snapshot:  snap,
	}, nil
}

// Reopen reopens a CLOSED or SOFT_CLOSED period after verifying a mandatory reason and
// 2 distinct human approvers (via M1.2 CEL Approvals bridge when configured, or distinct CoApproverID in standalone mode).
func (s *Service) Reopen(ctx context.Context, in ReopenInput) (ReopenResult, error) {
	if s == nil || s.repo == nil {
		return ReopenResult{}, ErrRepoUnavailable
	}
	entityID := strings.TrimSpace(in.EntityID)
	if entityID == "" {
		return ReopenResult{}, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	_, _, normPeriod, err := ValidatePeriodID(in.PeriodID)
	if err != nil {
		return ReopenResult{}, err
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return ReopenResult{}, fmt.Errorf("%w: reason is required to reopen a period", ErrInvalidInput)
	}
	actorID := strings.TrimSpace(in.ActorID)
	if actorID == "" {
		return ReopenResult{}, fmt.Errorf("%w: actor_id is required", ErrInvalidInput)
	}
	actorType := strings.ToLower(strings.TrimSpace(in.ActorType))
	if actorType == auth.ActorTypeAgent {
		return ReopenResult{}, fmt.Errorf("%w: agents are forbidden from reopening periods", ErrSoDViolation)
	}

	cur, found, err := s.repo.GetPeriod(ctx, entityID, normPeriod)
	if err != nil {
		return ReopenResult{}, err
	}
	if !found {
		return ReopenResult{}, fmt.Errorf("%w: %s for %s", ErrPeriodNotFound, normPeriod, entityID)
	}
	approvalID := strings.TrimSpace(in.ApprovalID)
	alreadyReopenedReplay := cur.Status == StatusReopened && approvalID != "" && s.approvalBridge != nil
	if !alreadyReopenedReplay && cur.Status != StatusClosed && cur.Status != StatusSoftClosed {
		return ReopenResult{}, fmt.Errorf("%w: period %s is %s (must be SOFT_CLOSED or CLOSED to reopen)", ErrInvalidStatusTransition, normPeriod, cur.Status)
	}

	if s.approvalBridge != nil {
		canonPayload := map[string]any{
			"entity_id": entityID,
			"period_id": normPeriod,
			"reason":    reason,
		}
		res, bErr := s.approvalBridge.EvaluateOrVerifyPeriodReopen(
			ctx, entityID, normPeriod, reason, actorID, in.ActorType, in.ActorRole, approvalID, canonPayload,
		)
		if bErr != nil {
			return ReopenResult{}, bErr
		}
		if !res.Approved {
			return ReopenResult{
				PendingApproval:   true,
				ApprovalRequestID: res.ApprovalRequestID,
				RequiredApprovers: res.RequiredApprovers,
				PayloadHash:       res.PayloadHash,
			}, nil
		}
		if res.ApprovalRequestID != "" {
			approvalID = res.ApprovalRequestID
		}
		if alreadyReopenedReplay {
			if strings.TrimSpace(cur.ReopenReason) == reason {
				return ReopenResult{
					PendingApproval:   false,
					ApprovalRequestID: approvalID,
					RequiredApprovers: 2,
					PayloadHash:       res.PayloadHash,
					Period:            &cur,
				}, nil
			}
			return ReopenResult{}, fmt.Errorf("%w: period %s is already REOPENED", ErrInvalidStatusTransition, normPeriod)
		}
	} else {
		coApprover := strings.TrimSpace(in.CoApproverID)
		if coApprover == "" || strings.EqualFold(coApprover, actorID) {
			return ReopenResult{}, fmt.Errorf("%w: co_approver_id distinct from actor_id is required when approval engine is not attached", ErrApprovalRequired)
		}
	}

	return s.applyReopenTransition(ctx, entityID, normPeriod, reason, actorID, in.ActorType, in.ActorRole, approvalID)
}

// CompleteApprovedReopen is invoked by an M1.2 OnApprovedHook when an ApprovalRequest for
// ("Period", "reopen") reaches APPROVED status with >= 2 distinct human approvers.
func (s *Service) CompleteApprovedReopen(ctx context.Context, entityID, periodID, reason, actorID, approvalID string) (Period, error) {
	if s == nil || s.repo == nil {
		return Period{}, ErrRepoUnavailable
	}
	entityID = strings.TrimSpace(entityID)
	_, _, normPeriod, err := ValidatePeriodID(periodID)
	if err != nil {
		return Period{}, err
	}
	res, err := s.applyReopenTransition(ctx, entityID, normPeriod, strings.TrimSpace(reason), strings.TrimSpace(actorID), auth.ActorTypeHuman, string(auth.RoleAdmin), strings.TrimSpace(approvalID))
	if err != nil {
		return Period{}, err
	}
	if res.Period == nil {
		return Period{}, ErrInvalidStatusTransition
	}
	return *res.Period, nil
}

func (s *Service) applyReopenTransition(ctx context.Context, entityID, normPeriod, reason, actorID, actorType, actorRole, approvalID string) (ReopenResult, error) {
	now := time.Now().UTC()
	reopened, err := s.repo.TransitionPeriod(ctx, entityID, normPeriod, func(cur *Period, curFound bool) (PeriodTransition, error) {
		if !curFound {
			return PeriodTransition{}, fmt.Errorf("%w: %s for %s", ErrPeriodNotFound, normPeriod, entityID)
		}
		if cur.Status == StatusReopened && strings.TrimSpace(cur.ReopenReason) == reason {
			return PeriodTransition{
				NextPeriod: *cur,
			}, nil
		}
		if cur.Status != StatusClosed && cur.Status != StatusSoftClosed {
			return PeriodTransition{}, fmt.Errorf("%w: cannot reopen period in status %s", ErrInvalidStatusTransition, cur.Status)
		}
		next := *cur
		next.Status = StatusReopened
		next.ReopenedBy = actorID
		next.ReopenedAt = &now
		next.ReopenReason = reason
		return PeriodTransition{
			NextPeriod:  next,
			EventType:   events.EventPeriodReopened,
			EventReason: reason,
			ApprovalID:  approvalID,
		}, nil
	}, s.audit, AuditRecordInput{
		TenantID:   entityID,
		ActorID:    actorID,
		ActorType:  actorType,
		Role:       actorRole,
		Action:     events.EventPeriodReopened,
		ObjectType: "Period",
		ObjectID:   entityID + ":" + normPeriod,
		ReasonCode: reason,
		ApprovalID: approvalID,
	})
	if err != nil {
		return ReopenResult{}, err
	}
	return ReopenResult{
		PendingApproval:   false,
		ApprovalRequestID: approvalID,
		RequiredApprovers: 2,
		Period:            &reopened,
	}, nil
}

// GetSnapshot loads the sealed close snapshot JSON artifact for (entityID, periodID) and verifies its hash.
func (s *Service) GetSnapshot(ctx context.Context, entityID, periodID string) (CloseSnapshot, error) {
	if s == nil || s.repo == nil {
		return CloseSnapshot{}, ErrRepoUnavailable
	}
	_, _, normPeriod, err := ValidatePeriodID(periodID)
	if err != nil {
		return CloseSnapshot{}, err
	}
	p, found, err := s.repo.GetPeriod(ctx, strings.TrimSpace(entityID), normPeriod)
	if err != nil {
		return CloseSnapshot{}, err
	}
	if !found || strings.TrimSpace(p.SnapshotURI) == "" {
		return CloseSnapshot{}, ErrSnapshotNotFound
	}
	sink := s.snapshotSink
	if sink == nil {
		sink = DefaultSnapshotSink(nil)
	}
	return ReadCloseSnapshot(ctx, sink, p)
}
