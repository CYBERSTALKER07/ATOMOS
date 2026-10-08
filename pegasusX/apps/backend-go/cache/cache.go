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
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Backend is the core contract pegasusX uses to talk to Redis (or any equivalent).
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

// HashBackend defines field-level read/write operations for objects whose
// fields update independently without rewriting the whole object (redis-core).
type HashBackend interface {
	HGet(ctx context.Context, key, field string) ([]byte, bool, error)
	HSet(ctx context.Context, key, field string, value []byte) error
	HGetAll(ctx context.Context, key string) (map[string][]byte, error)
	HDel(ctx context.Context, key string, fields ...string) error
}

// SetBackend defines unique-membership and set manipulation operations.
type SetBackend interface {
	SAdd(ctx context.Context, key string, members ...string) error
	SMembers(ctx context.Context, key string) ([]string, error)
	SIsMember(ctx context.Context, key, member string) (bool, error)
	SRem(ctx context.Context, key string, members ...string) error
	ReplaceSet(ctx context.Context, key string, members []string, ttl time.Duration) error
}

// SortedSetBackend defines score-ordered operations for sliding-window rate limiters,
// rankings, and real-time leaderboards.
type SortedSetBackend interface {
	ZAdd(ctx context.Context, key string, score float64, member string) error
	ZRangeByScore(ctx context.Context, key string, min, max float64, offset, count int64) ([]string, error)
	ZRemRangeByScore(ctx context.Context, key string, min, max float64) (int64, error)
	ZCard(ctx context.Context, key string) (int64, error)
}

// ScanBackend provides cursor-based incremental iteration to prevent blocking the
// Redis single-threaded event loop (redis-connections).
type ScanBackend interface {
	Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error)
	HScan(ctx context.Context, key string, cursor uint64, match string, count int64) ([]string, uint64, error)
}

// EnterpriseBackend combines all Redis data structure and lifecycle capabilities.
type EnterpriseBackend interface {
	Backend
	HashBackend
	SetBackend
	SortedSetBackend
	ScanBackend
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
// any key received locally. It includes an automatic reconnection loop with
// exponential backoff so temporary Redis disconnects do not terminate invalidations permanently.
func (c *Cache) StartInvalidationSubscriber(ctx context.Context) {
	if c == nil || c.backend == nil {
		return
	}

	backoff := 50 * time.Millisecond
	const maxBackoff = 5 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgs, cancel, err := c.backend.Subscribe(ctx, InvalidationChannel)
		if err != nil {
			c.log.Warn("cache invalidate subscribe failed, retrying", "err", err, "backoff", backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
		}

		// Reset backoff on successful subscription
		backoff = 50 * time.Millisecond

		active := true
		for active {
			select {
			case <-ctx.Done():
				cancel()
				return
			case key, ok := <-msgs:
				if !ok {
					active = false
					break
				}
				if err := c.backend.Delete(ctx, string(key)); err != nil {
					c.log.Warn("cache subscriber delete failed", "key", string(key), "err", err)
				}
			}
		}
		cancel()
	}
}

// ── Hash Support ─────────────────────────────────────────────────────────────

func (c *Cache) HGet(ctx context.Context, key, field string) ([]byte, bool, error) {
	if c == nil || c.backend == nil {
		return nil, false, nil
	}
	if hb, ok := c.backend.(HashBackend); ok {
		return hb.HGet(ctx, key, field)
	}
	return nil, false, fmt.Errorf("backend does not support Hash operations")
}

func (c *Cache) HSet(ctx context.Context, key, field string, value []byte) error {
	if c == nil || c.backend == nil {
		return nil
	}
	if hb, ok := c.backend.(HashBackend); ok {
		return hb.HSet(ctx, key, field, value)
	}
	return fmt.Errorf("backend does not support Hash operations")
}

func (c *Cache) HGetAll(ctx context.Context, key string) (map[string][]byte, error) {
	if c == nil || c.backend == nil {
		return nil, nil
	}
	if hb, ok := c.backend.(HashBackend); ok {
		return hb.HGetAll(ctx, key)
	}
	return nil, fmt.Errorf("backend does not support Hash operations")
}

func (c *Cache) HDel(ctx context.Context, key string, fields ...string) error {
	if c == nil || c.backend == nil {
		return nil
	}
	if hb, ok := c.backend.(HashBackend); ok {
		return hb.HDel(ctx, key, fields...)
	}
	return fmt.Errorf("backend does not support Hash operations")
}

// ── Set Support ──────────────────────────────────────────────────────────────

func (c *Cache) SAdd(ctx context.Context, key string, members ...string) error {
	if c == nil || c.backend == nil {
		return nil
	}
	if sb, ok := c.backend.(SetBackend); ok {
		return sb.SAdd(ctx, key, members...)
	}
	return fmt.Errorf("backend does not support Set operations")
}

func (c *Cache) SMembers(ctx context.Context, key string) ([]string, error) {
	if c == nil || c.backend == nil {
		return nil, nil
	}
	if sb, ok := c.backend.(SetBackend); ok {
		return sb.SMembers(ctx, key)
	}
	return nil, fmt.Errorf("backend does not support Set operations")
}

func (c *Cache) SIsMember(ctx context.Context, key, member string) (bool, error) {
	if c == nil || c.backend == nil {
		return false, nil
	}
	if sb, ok := c.backend.(SetBackend); ok {
		return sb.SIsMember(ctx, key, member)
	}
	return false, fmt.Errorf("backend does not support Set operations")
}

func (c *Cache) SRem(ctx context.Context, key string, members ...string) error {
	if c == nil || c.backend == nil {
		return nil
	}
	if sb, ok := c.backend.(SetBackend); ok {
		return sb.SRem(ctx, key, members...)
	}
	return fmt.Errorf("backend does not support Set operations")
}

func (c *Cache) ReplaceSet(ctx context.Context, key string, members []string, ttl time.Duration) error {
	if c == nil || c.backend == nil {
		return nil
	}
	if sb, ok := c.backend.(SetBackend); ok {
		return sb.ReplaceSet(ctx, key, members, ttl)
	}
	return fmt.Errorf("backend does not support Set operations")
}

// ── Sorted Set Support ───────────────────────────────────────────────────────

func (c *Cache) ZAdd(ctx context.Context, key string, score float64, member string) error {
	if c == nil || c.backend == nil {
		return nil
	}
	if zb, ok := c.backend.(SortedSetBackend); ok {
		return zb.ZAdd(ctx, key, score, member)
	}
	return fmt.Errorf("backend does not support SortedSet operations")
}

func (c *Cache) ZRangeByScore(ctx context.Context, key string, min, max float64, offset, count int64) ([]string, error) {
	if c == nil || c.backend == nil {
		return nil, nil
	}
	if zb, ok := c.backend.(SortedSetBackend); ok {
		return zb.ZRangeByScore(ctx, key, min, max, offset, count)
	}
	return nil, fmt.Errorf("backend does not support SortedSet operations")
}

func (c *Cache) ZRemRangeByScore(ctx context.Context, key string, min, max float64) (int64, error) {
	if c == nil || c.backend == nil {
		return 0, nil
	}
	if zb, ok := c.backend.(SortedSetBackend); ok {
		return zb.ZRemRangeByScore(ctx, key, min, max)
	}
	return 0, fmt.Errorf("backend does not support SortedSet operations")
}

func (c *Cache) ZCard(ctx context.Context, key string) (int64, error) {
	if c == nil || c.backend == nil {
		return 0, nil
	}
	if zb, ok := c.backend.(SortedSetBackend); ok {
		return zb.ZCard(ctx, key)
	}
	return 0, fmt.Errorf("backend does not support SortedSet operations")
}

// ── Scanning Support ─────────────────────────────────────────────────────────

func (c *Cache) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	if c == nil || c.backend == nil {
		return nil, 0, nil
	}
	if sc, ok := c.backend.(ScanBackend); ok {
		return sc.Scan(ctx, cursor, match, count)
	}
	return nil, 0, fmt.Errorf("backend does not support Scan operations")
}

// ── InMemoryBackend ─────────────────────────────────────────────────────────
// Full in-memory implementation of EnterpriseBackend for the scaffold and unit tests.

type inMemoryEntry struct {
	value     []byte
	expiresAt time.Time
}

type zEntry struct {
	score  float64
	member string
}

// InMemoryBackend is a thread-safe, process-local EnterpriseBackend.
type InMemoryBackend struct {
	mu          sync.RWMutex
	store       map[string]inMemoryEntry
	hashes      map[string]map[string][]byte
	sets        map[string]map[string]struct{}
	sortedSets  map[string][]zEntry
	subscribers map[string][]chan []byte
}

// NewInMemoryBackend constructs a ready InMemoryBackend.
func NewInMemoryBackend() *InMemoryBackend {
	return &InMemoryBackend{
		store:       make(map[string]inMemoryEntry),
		hashes:      make(map[string]map[string][]byte),
		sets:        make(map[string]map[string]struct{}),
		sortedSets:  make(map[string][]zEntry),
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

// Delete removes one or more keys from store, hashes, sets, and sortedSets.
func (m *InMemoryBackend) Delete(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.store, k)
		delete(m.hashes, k)
		delete(m.sets, k)
		delete(m.sortedSets, k)
	}
	return nil
}

// Exists reports whether the key is present in store, hashes, sets, or sortedSets.
func (m *InMemoryBackend) Exists(_ context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if e, ok := m.store[key]; ok {
		if e.expiresAt.IsZero() || time.Now().Before(e.expiresAt) {
			return true, nil
		}
	}
	if h, ok := m.hashes[key]; ok && len(h) > 0 {
		return true, nil
	}
	if s, ok := m.sets[key]; ok && len(s) > 0 {
		return true, nil
	}
	if z, ok := m.sortedSets[key]; ok && len(z) > 0 {
		return true, nil
	}
	return false, nil
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
	return m.IncrBy(context.Background(), key, -amount)
}

// ── InMemory Hash Implementation ─────────────────────────────────────────────

func (m *InMemoryBackend) HGet(_ context.Context, key, field string) ([]byte, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h, ok := m.hashes[key]
	if !ok {
		return nil, false, nil
	}
	val, ok := h[field]
	if !ok {
		return nil, false, nil
	}
	return append([]byte(nil), val...), true, nil
}

func (m *InMemoryBackend) HSet(_ context.Context, key, field string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.hashes[key]
	if !ok {
		h = make(map[string][]byte)
		m.hashes[key] = h
	}
	h[field] = append([]byte(nil), value...)
	return nil
}

func (m *InMemoryBackend) HGetAll(_ context.Context, key string) (map[string][]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h, ok := m.hashes[key]
	if !ok {
		return map[string][]byte{}, nil
	}
	res := make(map[string][]byte, len(h))
	for k, v := range h {
		res[k] = append([]byte(nil), v...)
	}
	return res, nil
}

func (m *InMemoryBackend) HDel(_ context.Context, key string, fields ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h, ok := m.hashes[key]; ok {
		for _, f := range fields {
			delete(h, f)
		}
	}
	return nil
}

// ── InMemory Set Implementation ──────────────────────────────────────────────

func (m *InMemoryBackend) SAdd(_ context.Context, key string, members ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sets[key]
	if !ok {
		s = make(map[string]struct{})
		m.sets[key] = s
	}
	for _, mbr := range members {
		if strings.TrimSpace(mbr) != "" {
			s[mbr] = struct{}{}
		}
	}
	return nil
}

func (m *InMemoryBackend) SMembers(_ context.Context, key string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sets[key]
	if !ok {
		return []string{}, nil
	}
	res := make([]string, 0, len(s))
	for k := range s {
		res = append(res, k)
	}
	sort.Strings(res)
	return res, nil
}

func (m *InMemoryBackend) SIsMember(_ context.Context, key, member string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.sets[key]; ok {
		_, present := s[member]
		return present, nil
	}
	return false, nil
}

func (m *InMemoryBackend) SRem(_ context.Context, key string, members ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sets[key]; ok {
		for _, mbr := range members {
			delete(s, mbr)
		}
	}
	return nil
}

func (m *InMemoryBackend) ReplaceSet(ctx context.Context, key string, members []string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := make(map[string]struct{}, len(members))
	for _, mbr := range members {
		if strings.TrimSpace(mbr) != "" {
			s[mbr] = struct{}{}
		}
	}
	m.sets[key] = s
	return nil
}

// ── InMemory Sorted Set Implementation ───────────────────────────────────────

func (m *InMemoryBackend) ZAdd(_ context.Context, key string, score float64, member string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := m.sortedSets[key]
	found := false
	for i, entry := range list {
		if entry.member == member {
			list[i].score = score
			found = true
			break
		}
	}
	if !found {
		list = append(list, zEntry{score: score, member: member})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].score < list[j].score
	})
	m.sortedSets[key] = list
	return nil
}

func (m *InMemoryBackend) ZRangeByScore(_ context.Context, key string, min, max float64, offset, count int64) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := m.sortedSets[key]
	var matching []string
	for _, entry := range list {
		if entry.score >= min && entry.score <= max {
			matching = append(matching, entry.member)
		}
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= int64(len(matching)) {
		return []string{}, nil
	}
	matching = matching[offset:]
	if count > 0 && int64(len(matching)) > count {
		matching = matching[:count]
	}
	return matching, nil
}

func (m *InMemoryBackend) ZRemRangeByScore(_ context.Context, key string, min, max float64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := m.sortedSets[key]
	var kept []zEntry
	var removed int64
	for _, entry := range list {
		if entry.score >= min && entry.score <= max {
			removed++
		} else {
			kept = append(kept, entry)
		}
	}
	m.sortedSets[key] = kept
	return removed, nil
}

func (m *InMemoryBackend) ZCard(_ context.Context, key string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.sortedSets[key])), nil
}

// ── InMemory Scan Implementation ─────────────────────────────────────────────

func (m *InMemoryBackend) Scan(_ context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	allKeys := make([]string, 0, len(m.store)+len(m.hashes)+len(m.sets)+len(m.sortedSets))
	for k := range m.store {
		allKeys = append(allKeys, k)
	}
	for k := range m.hashes {
		allKeys = append(allKeys, k)
	}
	for k := range m.sets {
		allKeys = append(allKeys, k)
	}
	for k := range m.sortedSets {
		allKeys = append(allKeys, k)
	}
	sort.Strings(allKeys)

	// Filter by pattern if specified
	var matched []string
	pattern := strings.TrimSpace(match)
	prefix := strings.TrimSuffix(pattern, "*")
	for _, k := range allKeys {
		if pattern == "" || pattern == "*" || (prefix != "" && strings.HasPrefix(k, prefix)) {
			matched = append(matched, k)
		}
	}

	start := int(cursor)
	if start >= len(matched) {
		return []string{}, 0, nil
	}
	end := start + int(count)
	if count <= 0 || end >= len(matched) {
		end = len(matched)
	}
	nextCursor := uint64(end)
	if end >= len(matched) {
		nextCursor = 0
	}
	return matched[start:end], nextCursor, nil
}

func (m *InMemoryBackend) HScan(_ context.Context, key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h, ok := m.hashes[key]
	if !ok {
		return []string{}, 0, nil
	}
	var pairs []string
	for k, v := range h {
		if match == "" || match == "*" || strings.HasPrefix(k, strings.TrimSuffix(match, "*")) {
			pairs = append(pairs, k, string(v))
		}
	}
	start := int(cursor)
	if start >= len(pairs) {
		return []string{}, 0, nil
	}
	end := start + int(count)*2
	if count <= 0 || end >= len(pairs) {
		end = len(pairs)
	}
	nextCursor := uint64(end)
	if end >= len(pairs) {
		nextCursor = 0
	}
	return pairs[start:end], nextCursor, nil
}
