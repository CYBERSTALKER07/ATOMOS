package forecastv2

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryRepository is an in-memory Repository implementation for deterministic unit testing.
type MemoryRepository struct {
	mu           sync.RWMutex
	runs         map[string]map[string]ForecastRun         // entityID -> runID -> run
	values       map[string][]ForecastValue                // entityID -> values
	explanations map[string]map[string]ForecastExplanation // entityID -> explanationID -> exp
	seriesSeed   map[string][]SeriesHistoryInput           // entityID -> series
	windowsSeed  map[string][]SeasonalWindow               // entityID -> overrides
	actualSeed   map[string]int64                          // entityID|loc|sku -> actual milli
	linkedPairs  map[string]bool                           // supplierID|retailerID -> true
}

// NewMemoryRepository constructs an empty MemoryRepository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		runs:         make(map[string]map[string]ForecastRun),
		values:       make(map[string][]ForecastValue),
		explanations: make(map[string]map[string]ForecastExplanation),
		seriesSeed:   make(map[string][]SeriesHistoryInput),
		windowsSeed:  make(map[string][]SeasonalWindow),
		actualSeed:   make(map[string]int64),
		linkedPairs:  make(map[string]bool),
	}
}

// SeedSeries configures historical series for unit tests.
func (m *MemoryRepository) SeedSeries(entityID string, series []SeriesHistoryInput, windows []SeasonalWindow) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seriesSeed[entityID] = append([]SeriesHistoryInput(nil), series...)
	m.windowsSeed[entityID] = append([]SeasonalWindow(nil), windows...)
}

// SeedActualMilli seeds observed actual demand in milli-units for FVA tests.
func (m *MemoryRepository) SeedActualMilli(entityID, locationID, sku string, actualMilli int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.actualSeed[entityID+"|"+locationID+"|"+sku] = actualMilli
}

// SeedRetailerLink marks retailerID as linked to supplierID for memory-mode RBAC checks.
func (m *MemoryRepository) SeedRetailerLink(supplierID, retailerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.linkedPairs[supplierID+"|"+retailerID] = true
}

func (m *MemoryRepository) SaveRunAndValues(_ context.Context, run ForecastRun, values []ForecastValue, explanations []ForecastExplanation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.runs[run.EntityID] == nil {
		m.runs[run.EntityID] = make(map[string]ForecastRun)
	}
	m.runs[run.EntityID][run.RunID] = run
	m.values[run.EntityID] = append(m.values[run.EntityID], values...)
	if m.explanations[run.EntityID] == nil {
		m.explanations[run.EntityID] = make(map[string]ForecastExplanation)
	}
	for _, exp := range explanations {
		m.explanations[run.EntityID][exp.ExplanationID] = exp
	}
	return nil
}

func (m *MemoryRepository) GetRun(_ context.Context, entityID, runID string) (ForecastRun, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.runs[entityID][runID]
	return r, ok, nil
}

func (m *MemoryRepository) GetLatestRun(_ context.Context, entityID string) (ForecastRun, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	byID := m.runs[entityID]
	if len(byID) == 0 {
		return ForecastRun{}, false, nil
	}
	var latest ForecastRun
	found := false
	for _, r := range byID {
		if !found || r.CreatedAt.After(latest.CreatedAt) || (r.CreatedAt.Equal(latest.CreatedAt) && r.RunID > latest.RunID) {
			latest = r
			found = true
		}
	}
	return latest, found, nil
}

func (m *MemoryRepository) ListForecastValues(_ context.Context, q ForecastQuery) ([]ForecastValue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	vals := m.values[q.EntityID]
	out := make([]ForecastValue, 0, len(vals))
	for _, v := range vals {
		if q.RunID != "" && v.RunID != q.RunID {
			continue
		}
		if q.LocationID != "" && v.LocationID != q.LocationID {
			continue
		}
		if q.SKU != "" && v.SKU != q.SKU {
			continue
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ForecastDate != out[j].ForecastDate {
			return out[i].ForecastDate < out[j].ForecastDate
		}
		if out[i].SKU != out[j].SKU {
			return out[i].SKU < out[j].SKU
		}
		if out[i].LocationID != out[j].LocationID {
			return out[i].LocationID < out[j].LocationID
		}
		return out[i].Model < out[j].Model
	})
	if q.Limit > 0 && len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out, nil
}

func (m *MemoryRepository) ListExplanations(_ context.Context, entityID, runID, locationID, sku string, limit int) ([]ForecastExplanation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	byID := m.explanations[entityID]
	out := make([]ForecastExplanation, 0, len(byID))
	for _, exp := range byID {
		if runID != "" && exp.RunID != runID {
			continue
		}
		if locationID != "" && exp.LocationID != locationID {
			continue
		}
		if sku != "" && exp.SKU != sku {
			continue
		}
		out = append(out, exp)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].ExplanationID > out[j].ExplanationID
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemoryRepository) GetExplanation(_ context.Context, entityID, explanationID string) (ForecastExplanation, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	exp, ok := m.explanations[entityID][explanationID]
	return exp, ok, nil
}

func (m *MemoryRepository) SaveExplanation(_ context.Context, exp ForecastExplanation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.explanations[exp.EntityID] == nil {
		m.explanations[exp.EntityID] = make(map[string]ForecastExplanation)
	}
	m.explanations[exp.EntityID][exp.ExplanationID] = exp
	return nil
}

func (m *MemoryRepository) UpdateExplanation(_ context.Context, exp ForecastExplanation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.explanations[exp.EntityID][exp.ExplanationID]; !ok {
		return ErrExplanationNotFound
	}
	m.explanations[exp.EntityID][exp.ExplanationID] = exp
	return nil
}

func (m *MemoryRepository) LoadHistoricalSeries(_ context.Context, entityID, locationID, sku string, _ time.Time, _ int) ([]SeriesHistoryInput, []SeasonalWindow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	raw := m.seriesSeed[entityID]
	var out []SeriesHistoryInput
	for _, s := range raw {
		if locationID != "" && s.LocationID != locationID {
			continue
		}
		if sku != "" && s.SKU != sku {
			continue
		}
		out = append(out, s)
	}
	return out, append([]SeasonalWindow(nil), m.windowsSeed[entityID]...), nil
}

func (m *MemoryRepository) LoadActualMilli(_ context.Context, entityID, locationID, sku string, _, _ time.Time) (int64, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.actualSeed[entityID+"|"+locationID+"|"+sku]
	return v, ok, nil
}

func (m *MemoryRepository) IsRetailerLinked(_ context.Context, supplierID, retailerID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.linkedPairs[strings.TrimSpace(supplierID)+"|"+strings.TrimSpace(retailerID)], nil
}
