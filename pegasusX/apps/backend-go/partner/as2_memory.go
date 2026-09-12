package partner

import (
	"context"
	"sync"
	"time"
)

// MemoryAs2ConfigRepository for tests / local without Spanner.
type MemoryAs2ConfigRepository struct {
	mu   sync.RWMutex
	cfgs map[string]As2Config
}

func NewMemoryAs2ConfigRepository() *MemoryAs2ConfigRepository {
	return &MemoryAs2ConfigRepository{cfgs: map[string]As2Config{}}
}

func as2Key(tenantType, tenantID string) string {
	return tenantType + "|" + tenantID
}

func (r *MemoryAs2ConfigRepository) Upsert(_ context.Context, c As2Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c.UpdatedAt = time.Now().UTC()
	r.cfgs[as2Key(c.TenantType, c.TenantID)] = c
	return nil
}

func (r *MemoryAs2ConfigRepository) Get(_ context.Context, tenantType, tenantID string) (As2Config, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.cfgs[as2Key(tenantType, tenantID)]
	return c, ok, nil
}

func (r *MemoryAs2ConfigRepository) GetByOurAs2Id(_ context.Context, ourAs2Id string) (As2Config, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.cfgs {
		if c.OurAs2Id == ourAs2Id {
			return c, true, nil
		}
	}
	return As2Config{}, false, nil
}
