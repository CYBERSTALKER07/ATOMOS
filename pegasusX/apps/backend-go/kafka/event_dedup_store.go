package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EventDedupStore drops duplicate Kafka events across pods.
type EventDedupStore interface {
	ShouldProcess(ctx context.Context, key string) (bool, error)
}

// DedupKeyForMessage builds a stable dedup key from topic/partition/offset.
func DedupKeyForMessage(topic string, partition int, offset int64) string {
	return fmt.Sprintf("%s|%d|%d", topic, partition, offset)
}

// InMemoryEventDedup is a process-local fallback dedup store.
type InMemoryEventDedup struct {
	mu   sync.Mutex
	seen map[string]time.Time
	ttl  time.Duration
}

// NewInMemoryEventDedup returns an in-memory dedup store.
func NewInMemoryEventDedup(ttl time.Duration) *InMemoryEventDedup {
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	return &InMemoryEventDedup{
		seen: make(map[string]time.Time),
		ttl:  ttl,
	}
}

// ShouldProcess reports false when the key was seen within TTL.
func (d *InMemoryEventDedup) ShouldProcess(_ context.Context, key string) (bool, error) {
	if key == "" {
		return true, nil
	}
	now := time.Now().UTC()
	d.mu.Lock()
	defer d.mu.Unlock()
	for k, expires := range d.seen {
		if now.After(expires) {
			delete(d.seen, k)
		}
	}
	if expires, ok := d.seen[key]; ok && now.Before(expires) {
		return false, nil
	}
	d.seen[key] = now.Add(d.ttl)
	return true, nil
}
