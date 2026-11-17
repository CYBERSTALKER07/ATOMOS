package periods

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
)

// MemoryRepository is an in-memory Repository for unit tests and dev fallback without Spanner.
type MemoryRepository struct {
	mu         sync.RWMutex
	periods    map[string]Period // entityID + "|" + periodID
	checklists map[string]ChecklistReport
	snapshots  map[string]CloseSnapshot
	outbox     []string
}

// NewMemoryRepository initializes an empty in-memory Period repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		periods:    make(map[string]Period),
		checklists: make(map[string]ChecklistReport),
		snapshots:  make(map[string]CloseSnapshot),
	}
}

func memKey(entityID, periodID string) string {
	return strings.TrimSpace(entityID) + "|" + strings.TrimSpace(periodID)
}

// SetChecklistForTest seeds a custom ChecklistReport for unit tests using MemoryRepository.
func (m *MemoryRepository) SetChecklistForTest(entityID, periodID string, report ChecklistReport) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checklists[memKey(entityID, periodID)] = report
}

// SetSnapshotDataForTest seeds snapshot numbers for unit tests using MemoryRepository.
func (m *MemoryRepository) SetSnapshotDataForTest(entityID, periodID string, snap CloseSnapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshots[memKey(entityID, periodID)] = snap
}

// OutboxEventTypes returns emitted period event types for memory-mode test assertions.
func (m *MemoryRepository) OutboxEventTypes() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, len(m.outbox))
	copy(out, m.outbox)
	return out
}

func (m *MemoryRepository) CreatePeriod(ctx context.Context, p Period, auditRec AuditRecorder, auditIn AuditRecordInput) (Period, error) {
	m.mu.Lock()
	k := memKey(p.EntityID, p.PeriodID)
	if _, exists := m.periods[k]; exists {
		m.mu.Unlock()
		return Period{}, fmt.Errorf("%w: %s for %s", ErrPeriodAlreadyExists, p.PeriodID, p.EntityID)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	p.CreatedAt = now
	p.UpdatedAt = now
	EnrichPeriodWindow(&p)
	m.periods[k] = p
	m.mu.Unlock()

	if auditRec != nil {
		_ = auditRec.RecordEvent(ctx, auditIn)
	}
	return p, nil
}

func (m *MemoryRepository) GetPeriod(_ context.Context, entityID, periodID string) (Period, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.periods[memKey(entityID, periodID)]
	if !ok {
		return Period{}, false, nil
	}
	EnrichPeriodWindow(&p)
	return p, true, nil
}

func (m *MemoryRepository) ListPeriods(_ context.Context, entityID, statusFilter string) ([]Period, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entityID = strings.TrimSpace(entityID)
	statusFilter = strings.ToUpper(strings.TrimSpace(statusFilter))
	out := make([]Period, 0)
	for _, p := range m.periods {
		if p.EntityID != entityID {
			continue
		}
		if statusFilter != "" && strings.ToUpper(p.Status) != statusFilter {
			continue
		}
		cp := p
		EnrichPeriodWindow(&cp)
		out = append(out, cp)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].PeriodID > out[j].PeriodID
	})
	return out, nil
}

func (m *MemoryRepository) TransitionPeriod(
	ctx context.Context,
	entityID, periodID string,
	mutateFn func(existing *Period, found bool) (PeriodTransition, error),
	auditRec AuditRecorder,
	auditIn AuditRecordInput,
) (Period, error) {
	m.mu.Lock()
	k := memKey(entityID, periodID)
	cur, found := m.periods[k]
	var curPtr *Period
	if found {
		cp := cur
		curPtr = &cp
	}
	tr, err := mutateFn(curPtr, found)
	if err != nil {
		m.mu.Unlock()
		return Period{}, err
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	next := tr.NextPeriod
	if !found || next.CreatedAt.IsZero() {
		next.CreatedAt = now
	}
	next.UpdatedAt = now
	if next.SoftClosedAt != nil && next.SoftClosedAt.IsZero() {
		t := now
		next.SoftClosedAt = &t
	}
	if next.ClosedAt != nil && next.ClosedAt.IsZero() {
		t := now
		next.ClosedAt = &t
	}
	if next.ReopenedAt != nil && next.ReopenedAt.IsZero() {
		t := now
		next.ReopenedAt = &t
	}
	EnrichPeriodWindow(&next)
	m.periods[k] = next
	if tr.EventType != "" {
		m.outbox = append(m.outbox, tr.EventType)
	}
	m.mu.Unlock()

	if auditRec != nil {
		_ = auditRec.RecordEvent(ctx, auditIn)
	}
	return next, nil
}

func (m *MemoryRepository) EvaluateChecklist(ctx context.Context, entityID, periodID, tzName string) (ChecklistReport, error) {
	m.mu.RLock()
	if rep, ok := m.checklists[memKey(entityID, periodID)]; ok {
		m.mu.RUnlock()
		return rep, nil
	}
	m.mu.RUnlock()

	if strings.TrimSpace(tzName) == "" {
		resolved, _, err := ResolveEntityTimezone(ctx, entityID, "")
		if err != nil {
			return ChecklistReport{}, err
		}
		tzName = resolved
	}
	startUTC, endUTC, _, resolvedTZ, err := ComputePeriodWindow(periodID, tzName)
	if err != nil {
		return ChecklistReport{}, err
	}
	items := defaultChecklistZeroItems()
	return ChecklistReport{
		EntityID:       entityID,
		PeriodID:       periodID,
		Timezone:       resolvedTZ,
		PeriodStartUTC: startUTC,
		PeriodEndUTC:   endUTC,
		ReadyToClose:   true,
		BlockingCount:  0,
		EvaluatedAt:    time.Now().UTC().Truncate(time.Microsecond),
		Items:          items,
	}, nil
}

func (m *MemoryRepository) BuildCloseSnapshotData(ctx context.Context, entityID, periodID, tzName, closedBy string) (CloseSnapshot, error) {
	m.mu.RLock()
	if snap, ok := m.snapshots[memKey(entityID, periodID)]; ok {
		m.mu.RUnlock()
		return snap, nil
	}
	m.mu.RUnlock()

	if strings.TrimSpace(tzName) == "" {
		resolved, _, err := ResolveEntityTimezone(ctx, entityID, "")
		if err != nil {
			return CloseSnapshot{}, err
		}
		tzName = resolved
	}
	startUTC, endUTC, _, resolvedTZ, err := ComputePeriodWindow(periodID, tzName)
	if err != nil {
		return CloseSnapshot{}, err
	}
	curr := gl.DefaultCurrencyForPack("")
	return CloseSnapshot{
		EntityID:       entityID,
		PeriodID:       periodID,
		Timezone:       resolvedTZ,
		PeriodStartUTC: startUTC,
		PeriodEndUTC:   endUTC,
		Currency:       curr,
		ClosedBy:       closedBy,
		GeneratedAt:    time.Now().UTC().Truncate(time.Microsecond),
		TrialBalance: gl.TrialBalanceReport{
			EntityID: entityID,
			PeriodID: periodID,
			Currency: curr,
			Balanced: true,
			Accounts: []gl.TrialBalanceLine{},
		},
		ARSummary: SnapshotARSummary{Currency: curr},
		ValuationSummary: SnapshotValuationSummary{
			Currency: curr,
			Lines:    []SnapshotValuationLine{},
		},
		VATSummary: SnapshotVATSummary{
			PeriodID: periodID,
			Currency: curr,
		},
	}, nil
}
