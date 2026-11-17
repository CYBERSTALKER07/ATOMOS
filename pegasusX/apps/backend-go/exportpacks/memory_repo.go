package exportpacks

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"cloud.google.com/go/spanner"
)

// MemoryRepository implements Repository in memory for unit/HTTP tests.
type MemoryRepository struct {
	mu     sync.RWMutex
	byKey  map[string]*ExportPack
	byPack map[string]*ExportPack
}

// NewMemoryRepository returns an initialized in-memory pack repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byKey:  make(map[string]*ExportPack),
		byPack: make(map[string]*ExportPack),
	}
}

func clonePack(p *ExportPack) *ExportPack {
	if p == nil {
		return nil
	}
	cp := *p
	if p.Artifacts != nil {
		cp.Artifacts = make([]ArtifactMeta, len(p.Artifacts))
		copy(cp.Artifacts, p.Artifacts)
	}
	return &cp
}

func (r *MemoryRepository) CreatePack(
	ctx context.Context,
	pack *ExportPack,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) error {
	if pack == nil || strings.TrimSpace(pack.TenantID) == "" || strings.TrimSpace(pack.PackID) == "" {
		return fmt.Errorf("%w: tenant_id and pack_id required", ErrInvalidInput)
	}
	r.mu.Lock()
	cp := clonePack(pack)
	r.byKey[cp.TenantID+"|"+cp.PackID] = cp
	r.byPack[cp.PackID] = cp
	r.mu.Unlock()

	if auditFn != nil {
		return auditFn(ctx, nil)
	}
	return nil
}

func (r *MemoryRepository) GetPack(_ context.Context, tenantID, packID string) (*ExportPack, error) {
	tid := strings.TrimSpace(tenantID)
	pid := strings.TrimSpace(packID)
	r.mu.RLock()
	p, ok := r.byKey[tid+"|"+pid]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, pid)
	}
	return clonePack(p), nil
}

func (r *MemoryRepository) GetPackByID(_ context.Context, packID string) (*ExportPack, error) {
	pid := strings.TrimSpace(packID)
	r.mu.RLock()
	p, ok := r.byPack[pid]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, pid)
	}
	return clonePack(p), nil
}

func (r *MemoryRepository) ListPacks(_ context.Context, tenantID, entityID string, limit int) ([]*ExportPack, error) {
	tid := strings.TrimSpace(tenantID)
	eid := strings.TrimSpace(entityID)
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*ExportPack
	for _, p := range r.byKey {
		if tid != "" && p.TenantID != tid {
			continue
		}
		if eid != "" && p.EntityID != eid {
			continue
		}
		out = append(out, clonePack(p))
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].PackID > out[j].PackID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	if out == nil {
		out = []*ExportPack{}
	}
	return out, nil
}
