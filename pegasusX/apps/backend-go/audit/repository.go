package audit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Repository defines durable storage operations for M0.4 audit events, hash-chain
// verification, daily WORM anchors, and legal holds.
type Repository interface {
	Record(ctx context.Context, in EventInput) (AuditEvent, error)
	RecordInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, in EventInput) (AuditEvent, error)
	ListEvents(ctx context.Context, filter Filter) ([]AuditEvent, error)
	GetObjectHistory(ctx context.Context, tenantID, objectType, objectID, viewerRole string, limit int) ([]AuditEvent, error)
	VerifyDayChain(ctx context.Context, tenantID, dayBucket string) (ChainVerifyResult, error)
	WriteDailyAnchor(ctx context.Context, tenantID, dayBucket, anchoredBy string, sink WormAnchorSink) (Anchor, error)
	ListAnchors(ctx context.Context, tenantID, dayBucket string, limit int) ([]Anchor, error)
	CreateLegalHold(ctx context.Context, in LegalHoldInput) (LegalHold, error)
	ListLegalHolds(ctx context.Context, tenantID, objectType, objectID, status string, limit int) ([]LegalHold, error)
	ListTenantsForDay(ctx context.Context, dayBucket string) ([]string, error)
}

// Service coordinates immutable hash-chained audit logging, verification, WORM anchoring, and legal holds.
type Service struct {
	repo       Repository
	diffWriter DiffWriter
	anchorSink WormAnchorSink
	now        func() time.Time
}

// NewService constructs a Spanner-backed audit Service.
func NewService(client *spanner.Client) *Service {
	if client == nil {
		return &Service{now: func() time.Time { return time.Now().UTC() }}
	}
	return &Service{
		repo: NewSpannerRepository(client),
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// NewServiceWithRepository constructs an audit Service over a custom Repository (e.g., MemoryRepository).
func NewServiceWithRepository(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// WithDiffWriter configures an explicit WORM DiffWriter for >32KB payloads.
func (s *Service) WithDiffWriter(w DiffWriter) *Service {
	if s != nil {
		s.diffWriter = w
	}
	return s
}

// WithAnchorSink configures a default WormAnchorSink for WriteDailyAnchor when callers pass sink==nil.
func (s *Service) WithAnchorSink(sink WormAnchorSink) *Service {
	if s != nil {
		s.anchorSink = sink
	}
	return s
}

// Repo returns the backing Repository.
func (s *Service) Repo() Repository {
	if s == nil {
		return nil
	}
	return s.repo
}

// Record appends one hash-chained AuditEvents row in its own transaction.
func (s *Service) Record(ctx context.Context, in EventInput) (AuditEvent, error) {
	if s == nil || s.repo == nil {
		return AuditEvent{}, ErrRepoUnavailable
	}
	in, err := s.prepareInput(ctx, in)
	if err != nil {
		return AuditEvent{}, err
	}
	return s.repo.Record(ctx, in)
}

// RecordInTxn appends one hash-chained AuditEvents row inside an existing Spanner ReadWriteTransaction.
// For multiple audit events inside the same RW transaction, prefer NewBatchRecorder.
func (s *Service) RecordInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, in EventInput) (AuditEvent, error) {
	if s == nil || s.repo == nil {
		return AuditEvent{}, ErrRepoUnavailable
	}
	in, err := s.prepareInput(ctx, in)
	if err != nil {
		return AuditEvent{}, err
	}
	return s.repo.RecordInTxn(ctx, txn, in)
}

// ListEvents queries AuditEvents (with optional legacy AuditLog fallback) and masks PII for non-finance/non-admin viewers.
func (s *Service) ListEvents(ctx context.Context, filter Filter) ([]AuditEvent, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	if filter.DayBucket != "" && !IsValidDayBucket(filter.DayBucket) {
		return nil, ErrInvalidDayBucket
	}
	rows, err := s.repo.ListEvents(ctx, filter)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(filter.ViewerRole) != "" {
		rows = applyMaskingToEvents(filter.ViewerRole, rows)
	}
	return rows, nil
}

// GetObjectHistory returns combined AuditEvents (+ readable AuditLog fallback) for one object,
// applying PII masking when viewerRole is not finance/admin.
func (s *Service) GetObjectHistory(ctx context.Context, tenantID, objectType, objectID, viewerRole string) ([]AuditEvent, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	tenantID = strings.TrimSpace(tenantID)
	objectType = strings.TrimSpace(objectType)
	objectID = strings.TrimSpace(objectID)
	if tenantID == "" || objectType == "" || objectID == "" {
		return nil, fmt.Errorf("%w: tenant_id, object_type, and object_id are required", ErrInvalidAuditInput)
	}
	rows, err := s.repo.GetObjectHistory(ctx, tenantID, objectType, objectID, viewerRole, 200)
	if err != nil {
		return nil, err
	}
	return applyMaskingToEvents(viewerRole, rows), nil
}

// VerifyDayChain verifies every PrevHash -> EntryHash link for (tenantID, dayBucket).
func (s *Service) VerifyDayChain(ctx context.Context, tenantID, dayBucket string) (ChainVerifyResult, error) {
	if s == nil || s.repo == nil {
		return ChainVerifyResult{}, ErrRepoUnavailable
	}
	tenantID = strings.TrimSpace(tenantID)
	dayBucket = strings.TrimSpace(dayBucket)
	if tenantID == "" {
		return ChainVerifyResult{}, fmt.Errorf("%w: tenant_id is required", ErrInvalidAuditInput)
	}
	if !IsValidDayBucket(dayBucket) {
		return ChainVerifyResult{}, ErrInvalidDayBucket
	}
	return s.repo.VerifyDayChain(ctx, tenantID, dayBucket)
}

// WriteDailyAnchor verifies the day's chain, writes the signed canonical root JSON to sink (or env WORM sink),
// persists AuditChainAnchors, and emits events.EventAuditAnchorWritten.
func (s *Service) WriteDailyAnchor(ctx context.Context, tenantID, dayBucket, anchoredBy string, sink WormAnchorSink) (Anchor, error) {
	if s == nil || s.repo == nil {
		return Anchor{}, ErrRepoUnavailable
	}
	tenantID = strings.TrimSpace(tenantID)
	dayBucket = strings.TrimSpace(dayBucket)
	anchoredBy = strings.TrimSpace(anchoredBy)
	if tenantID == "" {
		return Anchor{}, fmt.Errorf("%w: tenant_id is required", ErrInvalidAuditInput)
	}
	if !IsValidDayBucket(dayBucket) {
		return Anchor{}, ErrInvalidDayBucket
	}
	if anchoredBy == "" {
		anchoredBy = "system"
	}
	if sink == nil {
		sink = s.anchorSink
	}
	if sink == nil {
		var err error
		sink, err = NewDefaultWormSinkFromEnv(ctx)
		if err != nil {
			return Anchor{}, err
		}
	}
	return s.repo.WriteDailyAnchor(ctx, tenantID, dayBucket, anchoredBy, sink)
}

// ListAnchors returns daily anchors for tenantID (or cross-tenant when tenantID=="" for platform governance).
func (s *Service) ListAnchors(ctx context.Context, tenantID, dayBucket string, limit int) ([]Anchor, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	if strings.TrimSpace(dayBucket) != "" && !IsValidDayBucket(dayBucket) {
		return nil, ErrInvalidDayBucket
	}
	return s.repo.ListAnchors(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(dayBucket), limit)
}

// CreateLegalHold records a new AuditLegalHolds entry and appends an immutable audit event.
func (s *Service) CreateLegalHold(ctx context.Context, in LegalHoldInput) (LegalHold, error) {
	if s == nil || s.repo == nil {
		return LegalHold{}, ErrRepoUnavailable
	}
	in.TenantID = strings.TrimSpace(in.TenantID)
	in.ObjectType = strings.TrimSpace(in.ObjectType)
	in.ObjectID = strings.TrimSpace(in.ObjectID)
	in.Reason = strings.TrimSpace(in.Reason)
	in.CreatedBy = strings.TrimSpace(in.CreatedBy)
	if in.TenantID == "" || in.ObjectType == "" || in.ObjectID == "" || in.Reason == "" {
		return LegalHold{}, fmt.Errorf("%w: tenant_id, object_type, object_id, and reason are required", ErrInvalidAuditInput)
	}
	if in.CreatedBy == "" {
		in.CreatedBy = "system"
	}
	return s.repo.CreateLegalHold(ctx, in)
}

// ListLegalHolds lists legal holds matching optional filters.
func (s *Service) ListLegalHolds(ctx context.Context, tenantID, objectType, objectID, status string, limit int) ([]LegalHold, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	return s.repo.ListLegalHolds(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(objectType), strings.TrimSpace(objectID), strings.TrimSpace(status), limit)
}

// ListTenantsForDay returns distinct TenantIds with AuditEvents in dayBucket.
func (s *Service) ListTenantsForDay(ctx context.Context, dayBucket string) ([]string, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	if !IsValidDayBucket(dayBucket) {
		return nil, ErrInvalidDayBucket
	}
	return s.repo.ListTenantsForDay(ctx, dayBucket)
}

func (s *Service) prepareInput(ctx context.Context, in EventInput) (EventInput, error) {
	in.TenantID = strings.TrimSpace(in.TenantID)
	in.ActorID = strings.TrimSpace(in.ActorID)
	in.Action = strings.TrimSpace(in.Action)
	if in.TenantID == "" {
		return EventInput{}, fmt.Errorf("%w: tenant_id is required", ErrInvalidAuditInput)
	}
	if in.Action == "" {
		return EventInput{}, fmt.Errorf("%w: action is required", ErrInvalidAuditInput)
	}
	if in.ActorID == "" {
		in.ActorID = "system"
	}
	in.EventID = strings.TrimSpace(in.EventID)
	if in.EventID == "" {
		in.EventID = "aud_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	in.ActorType = NormalizeActorType(string(in.ActorType))
	if in.DayBucket = strings.TrimSpace(in.DayBucket); in.DayBucket == "" {
		nowFn := s.now
		if nowFn == nil {
			nowFn = func() time.Time { return time.Now().UTC() }
		}
		in.DayBucket = nowFn().UTC().Format("2006-01-02")
	} else if !IsValidDayBucket(in.DayBucket) {
		return EventInput{}, ErrInvalidDayBucket
	}

	if len(in.BeforeJSON) > MaxInlineDiffBytes || len(in.AfterJSON) > MaxInlineDiffBytes {
		writer := s.diffWriter
		if writer == nil {
			if dw, ok := s.anchorSink.(DiffWriter); ok {
				writer = dw
			} else if envDw, err := ResolveDiffWriterFromEnv(ctx); err != nil {
				if auth.IsProduction() {
					return EventInput{}, err
				}
			} else {
				writer = envDw
			}
		}
		ev := AuditEvent{
			TenantID:   in.TenantID,
			DayBucket:  in.DayBucket,
			EventID:    in.EventID,
			BeforeJSON: in.BeforeJSON,
			AfterJSON:  in.AfterJSON,
			DiffGcsURI: in.DiffGcsURI,
		}
		if err := ApplyLargeDiffOffload(ctx, &ev, writer); err != nil {
			return EventInput{}, err
		}
		in.BeforeJSON = ev.BeforeJSON
		in.AfterJSON = ev.AfterJSON
		in.DiffGcsURI = ev.DiffGcsURI
	}
	return in, nil
}

func applyMaskingToEvents(viewerRole string, events []AuditEvent) []AuditEvent {
	if len(events) == 0 {
		return events
	}
	maskedRole := !IsFinanceOrAdminRole(viewerRole)
	out := make([]AuditEvent, len(events))
	copy(out, events)
	if !maskedRole {
		return out
	}
	for i := range out {
		origBefore := out[i].BeforeJSON
		origAfter := out[i].AfterJSON
		out[i].BeforeJSON = MaskPIIForRole(viewerRole, origBefore)
		out[i].AfterJSON = MaskPIIForRole(viewerRole, origAfter)
		if out[i].BeforeJSON != origBefore || out[i].AfterJSON != origAfter {
			out[i].PIIMasked = true
		}
	}
	return out
}
