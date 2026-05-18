// Package idempotency provides the canonical idempotency-key guard used on
// mutating HTTP endpoints. Replays with the same key + same body hash return
// the stored response; same key + different body returns 409 Conflict.
package idempotency

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrConflict is returned when an idempotency key is reused with a different
// request body hash.
var ErrConflict = errors.New("idempotency: key reused with different payload")

// Record is what we store per key.
type Record struct {
	BodyHash    string
	StatusCode  int
	Response    []byte
	StoredAt    time.Time
}

// Store is the persistence seam. The scaffold ships an in-memory impl; bind
// to Redis (SETNX with TTL) in bootstrap for cross-pod correctness.
type Store interface {
	// Load returns (record, found, error). A clean miss is (zero, false, nil).
	Load(ctx context.Context, key string) (Record, bool, error)
	// Save records a (key → record) mapping. TTL controls retention.
	Save(ctx context.Context, key string, rec Record, ttl time.Duration) error
}

// Guard is invoked from a handler. If the key was seen with a different body
// hash it returns ErrConflict. If the key was seen with the same body it
// returns the prior record (and ok=true) so the handler can replay it.
// Otherwise it returns (zero, false, nil) and the caller MUST proceed with
// real work then call Save with the produced response.
func Guard(ctx context.Context, store Store, key, bodyHash string) (Record, bool, error) {
	if store == nil || key == "" {
		return Record{}, false, nil
	}
	rec, found, err := store.Load(ctx, key)
	if err != nil {
		return Record{}, false, err
	}
	if !found {
		return Record{}, false, nil
	}
	if rec.BodyHash != bodyHash {
		return Record{}, true, ErrConflict
	}
	return rec, true, nil
}

// InMemoryStore is the scaffold-default Store. Safe for concurrent use; not
// shared across pods.
type InMemoryStore struct {
	mu      sync.Mutex
	records map[string]inMemEntry
}

type inMemEntry struct {
	rec       Record
	expiresAt time.Time
}

// NewInMemoryStore returns an empty store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{records: make(map[string]inMemEntry)}
}

// Load implements Store.
func (s *InMemoryStore) Load(_ context.Context, key string) (Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.records[key]
	if !ok {
		return Record{}, false, nil
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		delete(s.records, key)
		return Record{}, false, nil
	}
	return e.rec, true, nil
}

// Save implements Store.
func (s *InMemoryStore) Save(_ context.Context, key string, rec Record, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := inMemEntry{rec: rec}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}
	s.records[key] = entry
	return nil
}
