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

// ErrInProgress is returned when the same idempotency key is already being processed.
var ErrInProgress = errors.New("idempotency: request in progress")

// Record is what we store per key.
type Record struct {
	BodyHash   string
	StatusCode int
	Response   []byte
	StoredAt   time.Time
}

// Store is the persistence seam. The scaffold ships an in-memory impl; bind
// to Redis (SETNX with TTL) in bootstrap for cross-pod correctness.
type Store interface {
	// Load returns (record, found, error). A clean miss is (zero, false, nil).
	Load(ctx context.Context, key string) (Record, bool, error)
	// Save records a (key → record) mapping. TTL controls retention.
	Save(ctx context.Context, key string, rec Record, ttl time.Duration) error
	// Acquire claims the key for in-flight processing. Returns ErrInProgress when
	// another worker already holds the key.
	Acquire(ctx context.Context, key, bodyHash string, ttl time.Duration) error
	// Release drops an in-flight claim without recording a replayable response.
	Release(ctx context.Context, key string) error
}

const inProgressStatusCode = -1

// Guard is invoked from a handler. If the key was seen with a different body
// hash it returns ErrConflict. If the key was seen with the same body it
// returns the prior record (and ok=true) so the handler can replay it.
// Otherwise it acquires the key and returns (zero, false, nil); the caller
// MUST proceed with real work then call Save with the produced response.
func Guard(ctx context.Context, store Store, key, bodyHash string) (Record, bool, error) {
	if store == nil || key == "" {
		return Record{}, false, nil
	}
	if Claimed(ctx) {
		return Record{}, false, nil
	}
	rec, found, err := store.Load(ctx, key)
	if err != nil {
		return Record{}, false, err
	}
	if found {
		if rec.StatusCode == inProgressStatusCode {
			return Record{}, false, ErrInProgress
		}
		if rec.BodyHash != bodyHash {
			return Record{}, true, ErrConflict
		}
		return rec, true, nil
	}
	if err := store.Acquire(ctx, key, bodyHash, 60*time.Second); err != nil {
		return Record{}, false, err
	}
	return Record{}, false, nil
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

// Acquire implements Store.
func (s *InMemoryStore) Acquire(_ context.Context, key, bodyHash string, ttl time.Duration) error {
	if key == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.records[key]; ok {
		if e.expiresAt.IsZero() || time.Now().Before(e.expiresAt) {
			return ErrInProgress
		}
		delete(s.records, key)
	}
	entry := inMemEntry{
		rec: Record{
			BodyHash:   bodyHash,
			StatusCode: inProgressStatusCode,
			StoredAt:   time.Now(),
		},
	}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}
	s.records[key] = entry
	return nil
}

// Release implements Store.
func (s *InMemoryStore) Release(_ context.Context, key string) error {
	if key == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, key)
	return nil
}
