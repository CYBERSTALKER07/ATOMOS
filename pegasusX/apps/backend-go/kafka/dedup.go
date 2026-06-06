package kafka

import (
	"sync"
	"time"
)

const defaultFanoutDedupTTL = 30 * time.Second

// fanoutDedup drops duplicate notification fan-outs within a short window.
type fanoutDedup struct {
	mu   sync.Mutex
	seen map[string]time.Time
	ttl  time.Duration
}

func newFanoutDedup(ttl time.Duration) *fanoutDedup {
	if ttl <= 0 {
		ttl = defaultFanoutDedupTTL
	}
	return &fanoutDedup{
		seen: make(map[string]time.Time),
		ttl:  ttl,
	}
}

func (d *fanoutDedup) shouldDrop(key string) bool {
	if key == "" {
		return false
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
		return true
	}
	d.seen[key] = now.Add(d.ttl)
	return false
}

func fanoutDedupKey(eventType, traceID, aggregateID string) string {
	if aggregateID == "" {
		aggregateID = traceID
	}
	if eventType == "" {
		return ""
	}
	return eventType + "|" + aggregateID
}
