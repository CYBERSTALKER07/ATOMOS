package recon

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryRepository provides an in-memory Repository implementation for unit tests and harness fallback.
type MemoryRepository struct {
	mu        sync.Mutex
	snapshots map[string]Snapshot
	runs      map[string][]ReconciliationRun
	breaks    map[string]map[string]ReconciliationBreak
	suppliers []string
}

// NewMemoryRepository constructs an initialized in-memory reconciliation repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		snapshots: make(map[string]Snapshot),
		runs:      make(map[string][]ReconciliationRun),
		breaks:    make(map[string]map[string]ReconciliationBreak),
	}
}

// SeedSnapshot registers a snapshot for an entity (intended for unit/harness tests).
func (m *MemoryRepository) SeedSnapshot(snap Snapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshots[snap.EntityID] = snap
	for _, s := range m.suppliers {
		if s == snap.EntityID {
			return
		}
	}
	m.suppliers = append(m.suppliers, snap.EntityID)
}

func (m *MemoryRepository) LoadSnapshot(_ context.Context, entityID, periodID, currency string, asOf time.Time) (Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if snap, ok := m.snapshots[entityID]; ok {
		snap.EntityID = entityID
		snap.PeriodID = periodID
		if snap.Currency == "" {
			snap.Currency = currency
		}
		if snap.AsOf.IsZero() {
			snap.AsOf = asOf
		}
		return snap, nil
	}
	return Snapshot{
		EntityID: entityID,
		PeriodID: periodID,
		Currency: currency,
		AsOf:     asOf,
	}, nil
}

func (m *MemoryRepository) SaveRunAndBreaks(_ context.Context, run ReconciliationRun, breaks []ReconciliationBreak) (ReconciliationRun, []ReconciliationBreak, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	savedBreaks := make([]ReconciliationBreak, len(breaks))
	if m.breaks[run.EntityID] == nil {
		m.breaks[run.EntityID] = make(map[string]ReconciliationBreak)
	}
	for i, b := range breaks {
		if b.ControlTowerExceptionID == "" {
			b.ControlTowerExceptionID = uuid.NewString()
		}
		m.breaks[run.EntityID][b.BreakID] = b
		savedBreaks[i] = b
	}
	m.runs[run.EntityID] = append(m.runs[run.EntityID], run)
	return run, savedBreaks, nil
}

func (m *MemoryRepository) ListRuns(_ context.Context, filter ListRunsFilter) ([]ReconciliationRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	all := m.runs[filter.EntityID]
	out := make([]ReconciliationRun, 0, len(all))
	for _, r := range all {
		if filter.PeriodID != "" && r.PeriodID != filter.PeriodID {
			continue
		}
		if filter.Status != "" && !strings.EqualFold(r.Status, filter.Status) {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt.After(out[j].StartedAt)
	})
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemoryRepository) ListBreaks(_ context.Context, filter ListBreaksFilter) ([]ReconciliationBreak, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	byID := m.breaks[filter.EntityID]
	out := make([]ReconciliationBreak, 0, len(byID))
	for _, b := range byID {
		if filter.RunID != "" && b.RunID != filter.RunID {
			continue
		}
		if filter.Dimension != "" && !strings.EqualFold(b.Dimension, filter.Dimension) {
			continue
		}
		if filter.Status != "" && !strings.EqualFold(filter.Status, "ALL") && !strings.EqualFold(b.Status, filter.Status) {
			continue
		}
		if filter.Severity != "" && !strings.EqualFold(b.Severity, filter.Severity) {
			continue
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool {
		if absInt64(out[i].DeltaMinor) != absInt64(out[j].DeltaMinor) {
			return absInt64(out[i].DeltaMinor) > absInt64(out[j].DeltaMinor)
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemoryRepository) GetBreak(_ context.Context, entityID, breakID string) (ReconciliationBreak, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if byID := m.breaks[entityID]; byID != nil {
		if b, ok := byID[breakID]; ok {
			return b, nil
		}
	}
	return ReconciliationBreak{}, fmt.Errorf("%w: break %s", ErrNotFound, breakID)
}

func (m *MemoryRepository) UpdateBreak(_ context.Context, entityID, breakID string, req PatchBreakRequest, now time.Time) (ReconciliationBreak, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	byID := m.breaks[entityID]
	if byID == nil {
		return ReconciliationBreak{}, fmt.Errorf("%w: break %s", ErrNotFound, breakID)
	}
	b, ok := byID[breakID]
	if !ok {
		return ReconciliationBreak{}, fmt.Errorf("%w: break %s", ErrNotFound, breakID)
	}
	nextStatus, err := resolveBreakTransition(b.Status, req)
	if err != nil {
		return ReconciliationBreak{}, err
	}
	b.Status = nextStatus
	if owner := strings.TrimSpace(req.OwnerID); owner != "" {
		b.OwnerID = owner
	}
	if note := strings.TrimSpace(coalesce(req.Note, req.ResolutionNote)); note != "" {
		b.ResolutionNote = note
	}
	b.UpdatedAt = now
	byID[breakID] = b
	return b, nil
}

func (m *MemoryRepository) ListActiveSupplierIDs(_ context.Context, limit int) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 || limit >= len(m.suppliers) {
		out := make([]string, len(m.suppliers))
		copy(out, m.suppliers)
		return out, nil
	}
	out := make([]string, limit)
	copy(out, m.suppliers[:limit])
	return out, nil
}
