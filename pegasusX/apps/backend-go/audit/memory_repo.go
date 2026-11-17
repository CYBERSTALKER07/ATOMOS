package audit

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

// MemoryRepository is a thread-safe in-memory audit repository for unit tests and harness runs.
type MemoryRepository struct {
	mu      sync.RWMutex
	events  []AuditEvent
	legacy  []AuditEvent
	anchors map[string]Anchor
	holds   []LegalHold
	diffs   map[string][]byte
	now     func() time.Time
}

// NewMemoryRepository constructs an initialized in-memory audit Repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		anchors: make(map[string]Anchor),
		diffs:   make(map[string][]byte),
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// WriteDiff implements DiffWriter for in-memory large diff tests.
func (r *MemoryRepository) WriteDiff(_ context.Context, tenantID, dayBucket, eventID, field string, raw []byte) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	uri := fmt.Sprintf("mem://worm-diffs/%s/%s/%s-%s.json", tenantID, dayBucket, eventID, field)
	cp := make([]byte, len(raw))
	copy(cp, raw)
	r.diffs[uri] = cp
	return uri, nil
}

// GetOffloadedDiffForTest retrieves an offloaded diff payload stored by WriteDiff.
func (r *MemoryRepository) GetOffloadedDiffForTest(uri string) ([]byte, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.diffs[uri]
	if !ok {
		return nil, false
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp, true
}

// SeedLegacyAuditLogForTest inserts a legacy AuditLog-shaped entry for fallback history tests.
func (r *MemoryRepository) SeedLegacyAuditLogForTest(ev AuditEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ev.Source == "" {
		ev.Source = "legacy_audit_log"
	}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = r.now()
	}
	r.legacy = append(r.legacy, ev)
}

// TamperEventForTest mutates a stored event's AfterJSON in memory without updating its EntryHash
// so tests can verify chain tamper detection.
func (r *MemoryRepository) TamperEventForTest(tenantID, dayBucket, eventID, newAfterJSON string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.events {
		if r.events[i].TenantID == tenantID && r.events[i].DayBucket == dayBucket && r.events[i].EventID == eventID {
			r.events[i].AfterJSON = newAfterJSON
			return true
		}
	}
	return false
}

// Record appends one hash-chained AuditEvent in memory.
func (r *MemoryRepository) Record(ctx context.Context, in EventInput) (AuditEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now().UTC()
	dayBucket := strings.TrimSpace(in.DayBucket)
	if dayBucket == "" {
		dayBucket = now.Format("2006-01-02")
	}
	eventID := strings.TrimSpace(in.EventID)
	if eventID == "" {
		eventID = "aud_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	}

	var prevSeq int64
	prevHash := GenesisHash
	for _, existing := range r.events {
		if existing.TenantID == in.TenantID && existing.DayBucket == dayBucket && existing.SeqNum > prevSeq {
			prevSeq = existing.SeqNum
			prevHash = existing.EntryHash
		}
	}

	legalHold := in.LegalHold
	legalHoldReason := strings.TrimSpace(in.LegalHoldReason)
	if !legalHold && in.ObjectType != "" && in.ObjectID != "" {
		for _, h := range r.holds {
			if h.TenantID == in.TenantID &&
				strings.EqualFold(h.ObjectType, in.ObjectType) &&
				h.ObjectID == in.ObjectID &&
				h.Status == LegalHoldStatusActive &&
				(h.ExpiresAt == nil || h.ExpiresAt.After(now)) {
				legalHold = true
				legalHoldReason = h.Reason
				break
			}
		}
	}

	ev := AuditEvent{
		TenantID:        in.TenantID,
		DayBucket:       dayBucket,
		EventID:         eventID,
		SeqNum:          prevSeq + 1,
		ActorID:         in.ActorID,
		ActorType:       NormalizeActorType(string(in.ActorType)),
		Role:            strings.TrimSpace(in.Role),
		Action:          in.Action,
		ObjectType:      strings.TrimSpace(in.ObjectType),
		ObjectID:        strings.TrimSpace(in.ObjectID),
		BeforeJSON:      in.BeforeJSON,
		AfterJSON:       in.AfterJSON,
		DiffGcsURI:      strings.TrimSpace(in.DiffGcsURI),
		ReasonCode:      strings.TrimSpace(in.ReasonCode),
		ApprovalID:      strings.TrimSpace(in.ApprovalID),
		TraceID:         strings.TrimSpace(in.TraceID),
		IP:              strings.TrimSpace(in.IP),
		DeviceID:        strings.TrimSpace(in.DeviceID),
		PrevHash:        prevHash,
		LegalHold:       legalHold,
		LegalHoldReason: legalHoldReason,
		RetentionUntil:  in.RetentionUntil,
		CreatedAt:       now,
		Source:          "audit_events",
	}
	if len(ev.BeforeJSON) > MaxInlineDiffBytes || len(ev.AfterJSON) > MaxInlineDiffBytes {
		r.mu.Unlock()
		err := ApplyLargeDiffOffload(ctx, &ev, r)
		r.mu.Lock()
		if err != nil {
			return AuditEvent{}, err
		}
	}
	canonical, err := CanonicalAuditBytes(ev)
	if err != nil {
		return AuditEvent{}, err
	}
	ev.EntryHash = ComputeEntryHash(ev.PrevHash, canonical)
	r.events = append(r.events, ev)
	return ev, nil
}

// RecordInTxn delegates to Record for in-memory test configurations.
func (r *MemoryRepository) RecordInTxn(ctx context.Context, _ *spanner.ReadWriteTransaction, in EventInput) (AuditEvent, error) {
	return r.Record(ctx, in)
}

// ListEvents filters in-memory AuditEvents (and optional legacy AuditLog entries).
func (r *MemoryRepository) ListEvents(_ context.Context, filter Filter) ([]AuditEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	out := make([]AuditEvent, 0, len(r.events))
	collect := func(src []AuditEvent) {
		for _, ev := range src {
			if filter.TenantID != "" && ev.TenantID != filter.TenantID {
				continue
			}
			if filter.ObjectType != "" && !strings.EqualFold(ev.ObjectType, filter.ObjectType) {
				continue
			}
			if filter.ObjectID != "" && ev.ObjectID != filter.ObjectID {
				continue
			}
			if filter.ActorID != "" && ev.ActorID != filter.ActorID {
				continue
			}
			if filter.ActorType != "" && ev.ActorType != filter.ActorType {
				continue
			}
			if filter.Action != "" && ev.Action != filter.Action {
				continue
			}
			if filter.DayBucket != "" && ev.DayBucket != filter.DayBucket {
				continue
			}
			if filter.From != nil && ev.CreatedAt.Before(*filter.From) {
				continue
			}
			if filter.To != nil && ev.CreatedAt.After(*filter.To) {
				continue
			}
			out = append(out, ev)
		}
	}
	collect(r.events)
	if filter.IncludeLegacy || (filter.ObjectType != "" && filter.ObjectID != "") {
		collect(r.legacy)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		if out[i].DayBucket != out[j].DayBucket {
			return out[i].DayBucket > out[j].DayBucket
		}
		return out[i].SeqNum > out[j].SeqNum
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// GetObjectHistory returns AuditEvents + legacy AuditLog entries for one object.
func (r *MemoryRepository) GetObjectHistory(ctx context.Context, tenantID, objectType, objectID, viewerRole string, limit int) ([]AuditEvent, error) {
	return r.ListEvents(ctx, Filter{
		TenantID:      tenantID,
		ObjectType:    objectType,
		ObjectID:      objectID,
		Limit:         limit,
		IncludeLegacy: true,
		ViewerRole:    viewerRole,
	})
}

// VerifyDayChain verifies every hash link for (tenantID, dayBucket).
func (r *MemoryRepository) VerifyDayChain(_ context.Context, tenantID, dayBucket string) (ChainVerifyResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var dayEvents []AuditEvent
	for _, ev := range r.events {
		if ev.TenantID == tenantID && ev.DayBucket == dayBucket {
			dayEvents = append(dayEvents, ev)
		}
	}
	anchor, hasAnchor := r.anchors[tenantID+"|"+dayBucket]
	return verifyOrderedEvents(tenantID, dayBucket, dayEvents, anchor, hasAnchor), nil
}

// WriteDailyAnchor verifies the chain, writes to sink if configured, and records the anchor.
func (r *MemoryRepository) WriteDailyAnchor(ctx context.Context, tenantID, dayBucket, anchoredBy string, sink WormAnchorSink) (Anchor, error) {
	verifyRes, err := r.VerifyDayChain(ctx, tenantID, dayBucket)
	if err != nil {
		return Anchor{}, err
	}
	if !verifyRes.Valid {
		return Anchor{}, fmt.Errorf("%w: %s", ErrChainBroken, verifyRes.FailureReason)
	}

	r.mu.RLock()
	existing, ok := r.anchors[tenantID+"|"+dayBucket]
	r.mu.RUnlock()
	if ok && existing.RootHash == verifyRes.RootHash && existing.EventCount == verifyRes.EventCount {
		return existing, nil
	}

	now := r.now().UTC()
	anchor := Anchor{
		TenantID:    tenantID,
		DayBucket:   dayBucket,
		HeadEventID: verifyRes.HeadEventID,
		EventCount:  verifyRes.EventCount,
		RootHash:    verifyRes.RootHash,
		AnchoredBy:  anchoredBy,
		AnchoredAt:  now,
	}
	if anchor.HeadEventID == "" {
		anchor.HeadEventID = "genesis"
	}

	blob, raw, err := SignAnchorBlob(anchor)
	if err != nil {
		return Anchor{}, err
	}
	anchor.SignerKeyID = blob.SignerKeyID
	if sink != nil {
		uri, err := sink.WriteAnchor(ctx, blob, raw)
		if err != nil {
			return Anchor{}, err
		}
		anchor.WormURI = uri
	}

	r.mu.Lock()
	r.anchors[tenantID+"|"+dayBucket] = anchor
	r.mu.Unlock()
	return anchor, nil
}

// ListAnchors returns stored daily anchors filtered by optional tenantID/dayBucket.
func (r *MemoryRepository) ListAnchors(_ context.Context, tenantID, dayBucket string, limit int) ([]Anchor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	out := make([]Anchor, 0, len(r.anchors))
	for _, a := range r.anchors {
		if tenantID != "" && a.TenantID != tenantID {
			continue
		}
		if dayBucket != "" && a.DayBucket != dayBucket {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DayBucket != out[j].DayBucket {
			return out[i].DayBucket > out[j].DayBucket
		}
		return out[i].AnchoredAt.After(out[j].AnchoredAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// CreateLegalHold stores an active legal hold and appends an audit event.
func (r *MemoryRepository) CreateLegalHold(ctx context.Context, in LegalHoldInput) (LegalHold, error) {
	now := r.now().UTC()
	hold := LegalHold{
		TenantID:   in.TenantID,
		HoldID:     "hold_" + strings.ReplaceAll(uuid.NewString(), "-", ""),
		ObjectType: in.ObjectType,
		ObjectID:   in.ObjectID,
		Reason:     in.Reason,
		Status:     LegalHoldStatusActive,
		CreatedBy:  in.CreatedBy,
		ExpiresAt:  in.ExpiresAt,
		CreatedAt:  now,
	}
	r.mu.Lock()
	r.holds = append(r.holds, hold)
	r.mu.Unlock()

	_, _ = r.Record(ctx, EventInput{
		TenantID:        in.TenantID,
		ActorID:         in.CreatedBy,
		ActorType:       ActorTypeHuman,
		Action:          "audit.legal_hold.created",
		ObjectType:      in.ObjectType,
		ObjectID:        in.ObjectID,
		AfterJSON:       fmt.Sprintf(`{"hold_id":%q,"status":%q,"reason":%q}`, hold.HoldID, hold.Status, hold.Reason),
		ReasonCode:      "LEGAL_HOLD",
		LegalHold:       true,
		LegalHoldReason: in.Reason,
	})
	return hold, nil
}

// ListLegalHolds lists legal holds matching optional filters.
func (r *MemoryRepository) ListLegalHolds(_ context.Context, tenantID, objectType, objectID, status string, limit int) ([]LegalHold, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	out := make([]LegalHold, 0, len(r.holds))
	for _, h := range r.holds {
		if tenantID != "" && h.TenantID != tenantID {
			continue
		}
		if objectType != "" && !strings.EqualFold(h.ObjectType, objectType) {
			continue
		}
		if objectID != "" && h.ObjectID != objectID {
			continue
		}
		if status != "" && !strings.EqualFold(h.Status, status) {
			continue
		}
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ListTenantsForDay returns distinct tenants with events in dayBucket.
func (r *MemoryRepository) ListTenantsForDay(_ context.Context, dayBucket string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]struct{})
	var out []string
	for _, ev := range r.events {
		if ev.DayBucket == dayBucket {
			if _, ok := seen[ev.TenantID]; !ok {
				seen[ev.TenantID] = struct{}{}
				out = append(out, ev.TenantID)
			}
		}
	}
	sort.Strings(out)
	return out, nil
}
