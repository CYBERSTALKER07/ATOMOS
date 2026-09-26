package platformadmin

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryRepository is an in-memory store for tests and local gates.
type MemoryRepository struct {
	mu      sync.Mutex
	tenants map[string]Tenant
	audit   []AuditRow
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{tenants: map[string]Tenant{}}
}

func tenantKey(tt, id string) string {
	return strings.ToUpper(tt) + "|" + id
}

func (r *MemoryRepository) UpsertTenant(_ context.Context, t Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tenants[tenantKey(t.TenantType, t.TenantID)] = t
	return nil
}

func (r *MemoryRepository) GetTenant(_ context.Context, tenantType, tenantID string) (Tenant, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tenants[tenantKey(tenantType, tenantID)]
	return t, ok, nil
}

func (r *MemoryRepository) ListTenants(_ context.Context, status string, limit int) ([]Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 {
		limit = 100
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	out := make([]Tenant, 0, len(r.tenants))
	for _, t := range r.tenants {
		if status != "" && t.Status != status {
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *MemoryRepository) InsertAudit(_ context.Context, row AuditRow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	r.audit = append(r.audit, row)
	return nil
}

func (r *MemoryRepository) ListAudit(_ context.Context, limit int) ([]AuditRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 {
		limit = 100
	}
	out := make([]AuditRow, len(r.audit))
	copy(out, r.audit)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
