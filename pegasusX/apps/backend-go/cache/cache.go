// Package cache wraps the Redis client used for hot reads and exposes the
// canonical Invalidate(ctx, keys...) pattern: DEL local keys, PUBLISH a kill
// signal on "cache:invalidate" so peer pods drop their copies.
//
// Pre-commit invalidation races with rollback — ALWAYS call Invalidate AFTER
// the Spanner ReadWriteTransaction commits.
//
// Storage backend is abstracted behind Backend so the scaffold can swap an
// in-memory implementation for go-redis without import churn.
package cache

import (
	"context"
	"log/slog"
	"sync"
	"strconv"
	"time"

	"golang.org/x/sync/singleflight"
)

// Backend is the contract pegasusX uses to talk to Redis (or any equivalent).
// Production binds this to github.com/redis/go-redis/v9. The in-memory impl
// here keeps the scaffold buildable and unit-testable without a live Redis.
type Backend interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	IncrBy(ctx context.Context, key string, amount int64) (int64, error)
	DecrBy(ctx context.Context, key string, amount int64) (int64, error)
	Publish(ctx context.Context, channel string, payload []byte) error
	// Subscribe returns a channel of payloads for the given Pub/Sub channel.
	// The returned cancel func unsubscribes and closes the channel.
	Subscribe(ctx context.Context, channel string) (<-chan []byte, func(), error)
}

// InvalidationChannel is the canonical Redis Pub/Sub channel.
const InvalidationChannel = "cache:invalidate"

// Cache wraps a Backend with the Invalidate helper plus best-effort logging.
type Cache struct {
	backend Backend
	log     *slog.Logger
	group   singleflight.Group
}

// New wires a Cache. Pass slog.Default() if you have no scoped logger.
func New(backend Backend, log *slog.Logger) *Cache {
	if log == nil {
		log = slog.Default()
	}
	return &Cache{backend: backend, log: log}
}

// Backend returns the underlying cache backend.
func (c *Cache) Backend() Backend {
	if c == nil {
		return nil
	}
	return c.backend
}

// Get reads a key. Returns (value, found, error). A nil error with found=false
// is a clean cache miss.
func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if c == nil || c.backend == nil {
		return nil, false, nil
	}
	return c.backend.Get(ctx, key)
}

// GetOrLoad returns cached data or calls loader on miss. Concurrent misses for
// the same key coalesce through singleflight.
func (c *Cache) GetOrLoad(ctx context.Context, key string, ttl time.Duration, loader func(ctx context.Context) ([]byte, error)) ([]byte, error) {
	if c == nil || c.backend == nil {
		return loader(ctx)
	}
	if data, found, err := c.backend.Get(ctx, key); err == nil && found {
		RecordHit(key)
		return data, nil
	}
	RecordMiss(key)
	val, err, _ := c.group.Do(key, func() (any, error) {
		if data, found, err := c.backend.Get(ctx, key); err == nil && found {
			RecordHit(key)
			return data, nil
		}
		RecordMiss(key)
		data, err := loader(ctx)
		if err != nil {
			return nil, err
		}
		if setErr := c.backend.Set(ctx, key, data, ttl); setErr != nil {
			c.log.Warn("cache set after load failed", "key", key, "err", setErr)
		}
		return data, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]byte), nil
}

// Set writes a key with TTL. Failures are non-fatal at call sites that treat
// the cache as a speed-up rather than the source of truth.
func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c == nil || c.backend == nil {
		return nil
	}
	return c.backend.Set(ctx, key, value, ttl)
}

// Close releases backend resources when the selected backend exposes a Close
// method (for example Redis clients). In-memory backend is a no-op.
func (c *Cache) Close() error {
	if c == nil || c.backend == nil {
		return nil
	}
	if closer, ok := c.backend.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

func (c *Cache) IncrBy(ctx context.Context, key string, amount int64) (int64, error) {
	if c == nil || c.backend == nil {
		return 0, nil
	}
	return c.backend.IncrBy(ctx, key, amount)
}

func (c *Cache) DecrBy(ctx context.Context, key string, amount int64) (int64, error) {
	if c == nil || c.backend == nil {
		return 0, nil
	}
	return c.backend.DecrBy(ctx, key, amount)
}

// Invalidate deletes the given keys locally and publishes them on the
// invalidation channel so peer pods drop their copies. Failures are logged but
// not returned — the caller MUST treat invalidation as best-effort durability,
// backed by TTL as the safety net.
func (c *Cache) Invalidate(ctx context.Context, keys ...string) {
	if c == nil || c.backend == nil || len(keys) == 0 {
		return
	}
	if err := c.backend.Delete(ctx, keys...); err != nil {
		c.log.Warn("cache local delete failed", "keys", keys, "err", err)
	}
	for _, k := range keys {
		if err := c.backend.Publish(ctx, InvalidationChannel, []byte(k)); err != nil {
			c.log.Warn("cache invalidate publish failed",
				"channel", InvalidationChannel, "key", k, "err", err)
		}
	}
}

// StartInvalidationSubscriber subscribes to InvalidationChannel and deletes
// any key received locally. Returns when ctx is cancelled.
func (c *Cache) StartInvalidationSubscriber(ctx context.Context) {
	if c == nil || c.backend == nil {
		return
	}
	msgs, cancel, err := c.backend.Subscribe(ctx, InvalidationChannel)
	if err != nil {
		c.log.Error("cache invalidate subscribe failed", "err", err)
		return
	}
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case key, ok := <-msgs:
			if !ok {
				return
			}
			if err := c.backend.Delete(ctx, string(key)); err != nil {
				c.log.Warn("cache subscriber delete failed", "key", string(key), "err", err)
			}
		}
	}
}

// ── InMemoryBackend ─────────────────────────────────────────────────────────
// Minimal Backend implementation for the scaffold and unit tests. Single
// process only; Pub/Sub is delivered to local subscribers. Replace with
// go-redis in production wiring inside bootstrap.NewApp.

type inMemoryEntry struct {
	value     []byte
	expiresAt time.Time
}

// InMemoryBackend is a process-local Backend. Safe for concurrent use.
type InMemoryBackend struct {
	mu          sync.RWMutex
	store       map[string]inMemoryEntry
	subscribers map[string][]chan []byte
}

// NewInMemoryBackend constructs a ready InMemoryBackend.
func NewInMemoryBackend() *InMemoryBackend {
	return &InMemoryBackend{
		store:       make(map[string]inMemoryEntry),
		subscribers: make(map[string][]chan []byte),
	}
}

// Get returns the stored value and whether it was present (and unexpired).
func (m *InMemoryBackend) Get(_ context.Context, key string) ([]byte, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.store[key]
	if !ok {
		return nil, false, nil
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		return nil, false, nil
	}
	out := make([]byte, len(e.value))
	copy(out, e.value)
	return out, true, nil
}

// Set writes a key with TTL (0 means no expiry).
func (m *InMemoryBackend) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry := inMemoryEntry{value: append([]byte(nil), value...)}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}
	m.store[key] = entry
	return nil
}

// Delete removes one or more keys.
func (m *InMemoryBackend) Delete(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.store, k)
	}
	return nil
}

// Publish fans out to local subscribers. Non-blocking: full subscriber buffers
// drop the message rather than stall the publisher.
func (m *InMemoryBackend) Publish(_ context.Context, channel string, payload []byte) error {
	m.mu.RLock()
	subs := append([]chan []byte(nil), m.subscribers[channel]...)
	m.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- append([]byte(nil), payload...):
		default:
		}
	}
	return nil
}

// Subscribe registers a subscriber. Cancel closes the channel and unregisters.
func (m *InMemoryBackend) Subscribe(_ context.Context, channel string) (<-chan []byte, func(), error) {
	ch := make(chan []byte, 64)
	m.mu.Lock()
	m.subscribers[channel] = append(m.subscribers[channel], ch)
	m.mu.Unlock()
	cancel := func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		subs := m.subscribers[channel]
		for i, c := range subs {
			if c == ch {
				m.subscribers[channel] = append(subs[:i], subs[i+1:]...)
				close(ch)
				return
			}
		}
	}
	return ch, cancel, nil
}


func (m *InMemoryBackend) IncrBy(_ context.Context, key string, amount int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.store[key]
	var current int64
	if ok {
		if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
			// expired
		} else {
			if v, err := strconv.ParseInt(string(entry.value), 10, 64); err == nil {
				current = v
			}
		}
	}
	current += amount
	entry.value = []byte(strconv.FormatInt(current, 10))
	m.store[key] = entry
	return current, nil
}

func (m *InMemoryBackend) DecrBy(_ context.Context, key string, amount int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.store[key]
	var current int64
	if ok {
		if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
			// expired
		} else {
			if v, err := strconv.ParseInt(string(entry.value), 10, 64); err == nil {
				current = v
			}
		}
	}
	current -= amount
	entry.value = []byte(strconv.FormatInt(current, 10))
	m.store[key] = entry
	return current, nil
}
