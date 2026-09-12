package partner

import (
	"context"
	"sync"
)

// MemoryCoaRepository for tests / local bootstrap without Spanner.
type MemoryCoaRepository struct {
	mu   sync.Mutex
	maps map[string]CoaMap
}

func NewMemoryCoaRepository() *MemoryCoaRepository {
	return &MemoryCoaRepository{maps: map[string]CoaMap{}}
}

func coaKey(tenantType, tenantID string) string {
	return tenantType + "|" + tenantID
}

func (r *MemoryCoaRepository) Get(_ context.Context, tenantType, tenantID string) (CoaMap, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.maps[coaKey(tenantType, tenantID)]
	return m, ok, nil
}

func (r *MemoryCoaRepository) Upsert(_ context.Context, m CoaMap) error {
	NormalizeCoa(&m)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.maps[coaKey(m.TenantType, m.TenantID)] = m
	return nil
}
