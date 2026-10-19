package auth

import (
	"context"
	"sync"
	"time"
)

// RevocationStore tracks revoked JWT IDs (jti) until natural expiry.
type RevocationStore interface {
	Revoke(ctx context.Context, jti string, ttl time.Duration) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

var (
	revocationMu    sync.RWMutex
	revocationStore RevocationStore = NewMemoryRevocationStore()
)

// SetRevocationStore installs the process-wide JWT denylist (Redis in prod, memory in tests).
func SetRevocationStore(store RevocationStore) {
	revocationMu.Lock()
	defer revocationMu.Unlock()
	if store == nil {
		revocationStore = NewMemoryRevocationStore()
		return
	}
	revocationStore = store
}

// GetRevocationStore returns the active denylist.
func GetRevocationStore() RevocationStore {
	revocationMu.RLock()
	defer revocationMu.RUnlock()
	return revocationStore
}

// MemoryRevocationStore is an in-process denylist (tests / single-pod fallback).
type MemoryRevocationStore struct {
	mu   sync.Mutex
	byID map[string]time.Time
}

// NewMemoryRevocationStore constructs an empty memory denylist.
func NewMemoryRevocationStore() *MemoryRevocationStore {
	return &MemoryRevocationStore{byID: make(map[string]time.Time)}
}

// Revoke marks jti revoked until now+ttl (minimum 1s).
func (m *MemoryRevocationStore) Revoke(_ context.Context, jti string, ttl time.Duration) error {
	jti = trimJTI(jti)
	if jti == "" {
		return nil
	}
	if ttl < time.Second {
		ttl = time.Second
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID[jti] = time.Now().UTC().Add(ttl)
	m.gcLocked()
	return nil
}

// IsRevoked reports whether jti is currently on the denylist.
func (m *MemoryRevocationStore) IsRevoked(_ context.Context, jti string) (bool, error) {
	jti = trimJTI(jti)
	if jti == "" {
		return false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	exp, ok := m.byID[jti]
	if !ok {
		return false, nil
	}
	if time.Now().UTC().After(exp) {
		delete(m.byID, jti)
		return false, nil
	}
	return true, nil
}

func (m *MemoryRevocationStore) gcLocked() {
	now := time.Now().UTC()
	for id, exp := range m.byID {
		if now.After(exp) {
			delete(m.byID, id)
		}
	}
}

func trimJTI(jti string) string {
	for len(jti) > 0 && (jti[0] == ' ' || jti[0] == '\t') {
		jti = jti[1:]
	}
	for len(jti) > 0 && (jti[len(jti)-1] == ' ' || jti[len(jti)-1] == '\t') {
		jti = jti[:len(jti)-1]
	}
	return jti
}
