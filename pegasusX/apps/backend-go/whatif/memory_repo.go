package whatif

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryRepository provides a thread-safe in-memory Repository for unit and route tests.
type MemoryRepository struct {
	mu       sync.RWMutex
	datasets map[string]SimulationDataset
	sims     map[string]map[string]WhatIfSimulation // tenantID -> simID -> WhatIfSimulation
}

// NewMemoryRepository constructs an empty in-memory What-If repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		datasets: make(map[string]SimulationDataset),
		sims:     make(map[string]map[string]WhatIfSimulation),
	}
}

// SeedDataset stores a deterministic SimulationDataset snapshot for unit tests.
func (r *MemoryRepository) SeedDataset(tenantID string, ds SimulationDataset) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	ds.TenantID = strings.TrimSpace(tenantID)
	r.datasets[ds.TenantID] = ds
}

func (r *MemoryRepository) LoadSimulationDataset(ctx context.Context, tenantID, scenarioID, entityID string) (SimulationDataset, error) {
	if err := ctx.Err(); err != nil {
		return SimulationDataset{}, err
	}
	if r == nil {
		return SimulationDataset{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	if ds, ok := r.datasets[tid]; ok {
		ds.EntityID = strings.TrimSpace(entityID)
		if strings.TrimSpace(scenarioID) != "" && ds.Scenario != nil && ds.Scenario.ScenarioID != strings.TrimSpace(scenarioID) {
			ds.Scenario = nil
			ds.ScenarioPatches = nil
		}
		return ds, nil
	}
	return SimulationDataset{TenantID: tid, EntityID: strings.TrimSpace(entityID)}, nil
}

func (r *MemoryRepository) SaveSimulation(
	ctx context.Context,
	sim WhatIfSimulation,
	auditRec AuditRecorder,
	auditIn AuditRecordInput,
) (WhatIfSimulation, bool, error) {
	if err := ctx.Err(); err != nil {
		return WhatIfSimulation{}, false, err
	}
	if r == nil {
		return WhatIfSimulation{}, false, ErrRepoUnavailable
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sims[sim.TenantID] == nil {
		r.sims[sim.TenantID] = make(map[string]WhatIfSimulation)
	}
	if existing, ok := r.sims[sim.TenantID][sim.SimulationID]; ok {
		return cloneSim(existing), true, nil
	}
	r.sims[sim.TenantID][sim.SimulationID] = cloneSim(sim)
	if auditRec != nil {
		if err := auditRec.RecordEvent(ctx, auditIn); err != nil {
			return WhatIfSimulation{}, false, fmt.Errorf("audit whatif.simulate: %w", err)
		}
	}
	return cloneSim(sim), false, nil
}

func (r *MemoryRepository) ListSimulations(ctx context.Context, f ListFilter) ([]WhatIfSimulation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(f.TenantID)
	st := strings.ToUpper(strings.TrimSpace(f.Status))
	limit := clampLimit(f.Limit)

	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []WhatIfSimulation
	for _, s := range r.sims[tid] {
		if st != "" && strings.ToUpper(s.Status) != st {
			continue
		}
		out = append(out, cloneSim(s))
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].SimulationID < out[j].SimulationID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *MemoryRepository) GetSimulation(ctx context.Context, tenantID, simID string) (WhatIfSimulation, bool, error) {
	if err := ctx.Err(); err != nil {
		return WhatIfSimulation{}, false, err
	}
	if r == nil {
		return WhatIfSimulation{}, false, ErrRepoUnavailable
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sims[strings.TrimSpace(tenantID)][strings.TrimSpace(simID)]
	if !ok {
		return WhatIfSimulation{}, false, nil
	}
	return cloneSim(s), true, nil
}

func (r *MemoryRepository) CommitSimulationProposals(
	ctx context.Context,
	tenantID, simID string,
	mutate func(cur WhatIfSimulation) (WhatIfSimulation, error),
	auditRec AuditRecorder,
	auditIn func(updated WhatIfSimulation) AuditRecordInput,
	_ time.Time,
) (WhatIfSimulation, error) {
	if err := ctx.Err(); err != nil {
		return WhatIfSimulation{}, err
	}
	if r == nil {
		return WhatIfSimulation{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	sid := strings.TrimSpace(simID)

	r.mu.Lock()
	defer r.mu.Unlock()
	cur, ok := r.sims[tid][sid]
	if !ok {
		return WhatIfSimulation{}, ErrNotFound
	}
	updated, err := mutate(cloneSim(cur))
	if err != nil {
		return WhatIfSimulation{}, err
	}
	r.sims[tid][sid] = cloneSim(updated)
	if auditRec != nil && auditIn != nil {
		if err := auditRec.RecordEvent(ctx, auditIn(updated)); err != nil {
			return WhatIfSimulation{}, fmt.Errorf("audit whatif.commit_proposals: %w", err)
		}
	}
	return cloneSim(updated), nil
}

func cloneSim(s WhatIfSimulation) WhatIfSimulation {
	cp := s
	cp.CommittedProposalIDs = append([]string(nil), s.CommittedProposalIDs...)
	if s.Breakdown != nil {
		b := *s.Breakdown
		b.Replenishment.AffectedSkus = append([]string(nil), s.Breakdown.Replenishment.AffectedSkus...)
		b.RecommendedActions = append([]RecommendedAction(nil), s.Breakdown.RecommendedActions...)
		cp.Breakdown = &b
	}
	return cp
}
