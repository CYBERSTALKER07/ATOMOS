package kafka

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// EventDedupStore drops duplicate Kafka events across pods.
type EventDedupStore interface {
	ShouldProcess(ctx context.Context, key string) (bool, error)
	Release(ctx context.Context, key string) error
}

// DedupKeyForMessage builds a stable dedup key from topic/partition/offset.
func DedupKeyForMessage(topic string, partition int, offset int64) string {
	return fmt.Sprintf("%s|%d|%d", topic, partition, offset)
}

// DedupKeyForEventID prefers Event.EventID (Kafka header) over broker offset so
// multi-replica outbox relays that re-publish the same logical event are dropped.
func DedupKeyForEventID(consumerGroup, topic, eventID string) string {
	eid := strings.TrimSpace(eventID)
	if eid == "" {
		return ""
	}
	group := strings.TrimSpace(consumerGroup)
	if group == "" {
		return fmt.Sprintf("%s|eid:%s", topic, eid)
	}
	return fmt.Sprintf("%s:%s|eid:%s", group, topic, eid)
}

// DedupKeyForConsumerGroup scopes dedup to one consumer group so independent
// groups reading the same topic/partition/offset do not suppress each other.
func DedupKeyForConsumerGroup(consumerGroup, topic string, partition int, offset int64) string {
	group := strings.TrimSpace(consumerGroup)
	if group == "" {
		return DedupKeyForMessage(topic, partition, offset)
	}
	return fmt.Sprintf("%s:%s", group, DedupKeyForMessage(topic, partition, offset))
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

// Release removes the key from the seen map.
func (d *InMemoryEventDedup) Release(_ context.Context, key string) error {
	if key == "" {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.seen, key)
	return nil
}
