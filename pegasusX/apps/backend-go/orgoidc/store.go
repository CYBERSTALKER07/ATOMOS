package orgoidc

import (
	"context"
	"sync"
)

// Store persists per-supplier OIDC config. Isolation key is SupplierId.
type Store interface {
	Get(ctx context.Context, supplierID string) (Config, bool, error)
	Put(ctx context.Context, c Config) error
	Delete(ctx context.Context, supplierID string) error
}

// MemoryStore is the test / nil-Spanner implementation.
type MemoryStore struct {
	mu   sync.RWMutex
	rows map[string]Config
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: map[string]Config{}}
}

func (m *MemoryStore) Get(_ context.Context, supplierID string) (Config, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.rows[supplierID]
	return c, ok, nil
}

func (m *MemoryStore) Put(_ context.Context, c Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rows == nil {
		m.rows = map[string]Config{}
	}
	m.rows[c.SupplierID] = c
	return nil
}

func (m *MemoryStore) Delete(_ context.Context, supplierID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rows, supplierID)
	return nil
}
