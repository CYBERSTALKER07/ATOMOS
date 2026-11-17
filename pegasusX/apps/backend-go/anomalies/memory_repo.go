package anomalies

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
	datasets map[string]DetectorDataset
	cases    map[string]map[string]AnomalyCase // tenantID -> caseID -> AnomalyCase
}

// NewMemoryRepository constructs an empty in-memory anomaly repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		datasets: make(map[string]DetectorDataset),
		cases:    make(map[string]map[string]AnomalyCase),
	}
}

// SeedDataset stores a deterministic DetectorDataset snapshot for a tenant in unit tests.
func (r *MemoryRepository) SeedDataset(tenantID string, ds DetectorDataset) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	ds.TenantID = strings.TrimSpace(tenantID)
	r.datasets[ds.TenantID] = ds
}

func (r *MemoryRepository) LoadDetectorDataset(ctx context.Context, tenantID, entityID string) (DetectorDataset, error) {
	if err := ctx.Err(); err != nil {
		return DetectorDataset{}, err
	}
	if r == nil {
		return DetectorDataset{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	if ds, ok := r.datasets[tid]; ok {
		ds.EntityID = strings.TrimSpace(entityID)
		return ds, nil
	}
	return DetectorDataset{TenantID: tid, EntityID: strings.TrimSpace(entityID)}, nil
}

func (r *MemoryRepository) SaveDetectedCases(
	ctx context.Context,
	tenantID string,
	candidates []AnomalyCase,
	auditRec AuditRecorder,
	auditIn AuditRecordInput,
) ([]AnomalyCase, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	if r == nil {
		return nil, 0, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cases[tid] == nil {
		r.cases[tid] = make(map[string]AnomalyCase)
	}
	existingKeys := make(map[string]struct{}, len(r.cases[tid]))
	for _, c := range r.cases[tid] {
		existingKeys[FindingKey(c.RuleCode, c.ObjectType, c.ObjectID)] = struct{}{}
	}

	var created []AnomalyCase
	var deduped int
	for _, cand := range candidates {
		fk := FindingKey(cand.RuleCode, cand.ObjectType, cand.ObjectID)
		if _, exists := existingKeys[fk]; exists {
			deduped++
			continue
		}
		existingKeys[fk] = struct{}{}
		cp := cloneCase(cand)
		r.cases[tid][cp.CaseID] = cp
		created = append(created, cloneCase(cp))
	}

	if auditRec != nil && len(created) > 0 {
		if err := auditRec.RecordEvent(ctx, auditIn); err != nil {
			return nil, 0, fmt.Errorf("audit anomaly.scan: %w", err)
		}
	}
	return created, deduped, nil
}

func (r *MemoryRepository) ListCases(ctx context.Context, f ListFilter) ([]AnomalyCase, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(f.TenantID)
	limit := clampLimit(f.Limit)
	st := strings.ToUpper(strings.TrimSpace(f.Status))
	sev := strings.ToUpper(strings.TrimSpace(f.Severity))
	cat := strings.ToUpper(strings.TrimSpace(f.Category))
	rc := strings.ToUpper(strings.TrimSpace(f.RuleCode))

	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []AnomalyCase
	for _, c := range r.cases[tid] {
		if st != "" && strings.ToUpper(c.Status) != st {
			continue
		}
		if sev != "" && strings.ToUpper(c.Severity) != sev {
			continue
		}
		if cat != "" && strings.ToUpper(c.Category) != cat {
			continue
		}
		if rc != "" && strings.ToUpper(c.RuleCode) != rc {
			continue
		}
		out = append(out, cloneCase(c))
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].DetectedAt.Equal(out[j].DetectedAt) {
			return out[i].DetectedAt.After(out[j].DetectedAt)
		}
		return out[i].CaseID < out[j].CaseID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *MemoryRepository) GetCase(ctx context.Context, tenantID, caseID string) (AnomalyCase, bool, error) {
	if err := ctx.Err(); err != nil {
		return AnomalyCase{}, false, err
	}
	if r == nil {
		return AnomalyCase{}, false, ErrRepoUnavailable
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.cases[strings.TrimSpace(tenantID)][strings.TrimSpace(caseID)]
	if !ok {
		return AnomalyCase{}, false, nil
	}
	return cloneCase(c), true, nil
}

func (r *MemoryRepository) ResolveCase(
	ctx context.Context,
	tenantID, caseID string,
	mutate func(cur AnomalyCase) (AnomalyCase, error),
	auditRec AuditRecorder,
	auditIn func(updated AnomalyCase) AuditRecordInput,
	_ time.Time,
) (AnomalyCase, error) {
	if err := ctx.Err(); err != nil {
		return AnomalyCase{}, err
	}
	if r == nil {
		return AnomalyCase{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	cid := strings.TrimSpace(caseID)

	r.mu.Lock()
	defer r.mu.Unlock()

	cur, ok := r.cases[tid][cid]
	if !ok {
		return AnomalyCase{}, ErrNotFound
	}
	updated, err := mutate(cloneCase(cur))
	if err != nil {
		return AnomalyCase{}, err
	}
	r.cases[tid][cid] = cloneCase(updated)
	if auditRec != nil && auditIn != nil {
		if err := auditRec.RecordEvent(ctx, auditIn(updated)); err != nil {
			return AnomalyCase{}, fmt.Errorf("audit anomaly.resolve: %w", err)
		}
	}
	return cloneCase(updated), nil
}

func cloneCase(c AnomalyCase) AnomalyCase {
	cp := c
	if c.ResolvedAt != nil {
		t := *c.ResolvedAt
		cp.ResolvedAt = &t
	}
	if c.Evidence != nil {
		ev := make(map[string]any, len(c.Evidence))
		for k, v := range c.Evidence {
			ev[k] = v
		}
		cp.Evidence = ev
	} else if c.EvidenceJSON != "" {
		cp.Evidence = decodeEvidenceMap(c.EvidenceJSON)
	}
	return cp
}
