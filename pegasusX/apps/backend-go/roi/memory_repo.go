package roi

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryRepository provides thread-safe in-memory storage for RoiBaselines, RoiEvents,
// optional platform fees, and historical metrics in non-Spanner unit tests.
type MemoryRepository struct {
	mu           sync.RWMutex
	baselines    map[string][]Baseline // tenantID -> baselines
	events       map[string][]RoiEvent // tenantID -> events
	eventsByIdem map[string]RoiEvent   // tenantID + ":" + idempotencyKey -> event
	platformFees map[string]int64      // tenantID -> feesMinor
	histMetrics  map[string]HistoricalMetrics
}

// NewMemoryRepository constructs an empty in-memory ROI repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		baselines:    make(map[string][]Baseline),
		events:       make(map[string][]RoiEvent),
		eventsByIdem: make(map[string]RoiEvent),
		platformFees: make(map[string]int64),
		histMetrics:  make(map[string]HistoricalMetrics),
	}
}

// SetPlatformFeesMinor seeds deterministic platform fees for unit tests.
func (m *MemoryRepository) SetPlatformFeesMinor(tenantID string, feesMinor int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.platformFees[strings.TrimSpace(tenantID)] = feesMinor
}

// SetHistoricalMetrics seeds historical AR/bad-debt/expired metrics for unit tests.
func (m *MemoryRepository) SetHistoricalMetrics(tenantID string, hm HistoricalMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.histMetrics[strings.TrimSpace(tenantID)] = hm
}

func (m *MemoryRepository) ReadPlatformFeesMinor(_ context.Context, tenantID string, _, _ time.Time) (int64, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.platformFees[strings.TrimSpace(tenantID)], "UZS", nil
}

func (m *MemoryRepository) ReadHistoricalMetrics(_ context.Context, tenantID string, _, _ time.Time) (HistoricalMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	hm, ok := m.histMetrics[strings.TrimSpace(tenantID)]
	if !ok {
		return HistoricalMetrics{Currency: "UZS"}, nil
	}
	if hm.Currency == "" {
		hm.Currency = "UZS"
	}
	return hm, nil
}

func (m *MemoryRepository) UpsertBaseline(_ context.Context, b Baseline) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	tid := strings.TrimSpace(b.TenantID)
	list := m.baselines[tid]
	for i := range list {
		if list[i].BaselineID == b.BaselineID {
			list[i] = cloneBaseline(b)
			m.baselines[tid] = list
			return nil
		}
	}
	m.baselines[tid] = append(list, cloneBaseline(b))
	return nil
}

func (m *MemoryRepository) GetLatestBaseline(_ context.Context, tenantID string) (Baseline, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := m.baselines[strings.TrimSpace(tenantID)]
	if len(list) == 0 {
		return Baseline{}, false, nil
	}
	latest := list[0]
	for _, b := range list[1:] {
		if b.CreatedAt.After(latest.CreatedAt) || (b.CreatedAt.Equal(latest.CreatedAt) && b.BaselineID > latest.BaselineID) {
			latest = b
		}
	}
	return cloneBaseline(latest), true, nil
}

func (m *MemoryRepository) FindEventByIdempotency(_ context.Context, tenantID, idempotencyKey string) (RoiEvent, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ev, ok := m.eventsByIdem[strings.TrimSpace(tenantID)+":"+strings.TrimSpace(idempotencyKey)]
	return ev, ok, nil
}

func (m *MemoryRepository) CreateEvent(_ context.Context, ev RoiEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	tid := strings.TrimSpace(ev.TenantID)
	if idem := strings.TrimSpace(ev.IdempotencyKey); idem != "" {
		key := tid + ":" + idem
		if _, exists := m.eventsByIdem[key]; exists {
			return nil
		}
		m.eventsByIdem[key] = ev
	}
	m.events[tid] = append(m.events[tid], ev)
	return nil
}

func (m *MemoryRepository) ListEvents(_ context.Context, f EventsFilter) ([]RoiEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tid := strings.TrimSpace(f.TenantID)
	cat := strings.ToUpper(strings.TrimSpace(f.Category))
	src := m.events[tid]
	out := make([]RoiEvent, 0, len(src))
	for _, ev := range src {
		if cat != "" && !strings.EqualFold(ev.Category, cat) {
			continue
		}
		if f.Since != nil && ev.OccurredAt.Before(*f.Since) {
			continue
		}
		if f.Until != nil && ev.OccurredAt.After(*f.Until) {
			continue
		}
		out = append(out, ev)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].OccurredAt.Equal(out[j].OccurredAt) {
			return out[i].OccurredAt.After(out[j].OccurredAt)
		}
		return out[i].EventID > out[j].EventID
	})
	limit := f.Limit
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func cloneBaseline(b Baseline) Baseline {
	if b.SurveyJSON != "" && b.Survey == nil {
		var m map[string]any
		if err := json.Unmarshal([]byte(b.SurveyJSON), &m); err == nil {
			b.Survey = m
		}
	}
	if b.Survey != nil {
		cp := make(map[string]any, len(b.Survey))
		for k, v := range b.Survey {
			cp[k] = v
		}
		b.Survey = cp
	}
	return b
}
