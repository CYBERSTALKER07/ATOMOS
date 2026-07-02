package memory

import (
	"context"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// OutboxStore is the scaffold outbox store used until Spanner-backed OutboxEvents are wired.
type OutboxStore struct {
	mu          sync.RWMutex
	events      []outbox.Event
	publishedAt map[string]time.Time
}

// NewOutboxStore returns an in-memory outbox store for local scaffold runs.
func NewOutboxStore() *OutboxStore {
	return &OutboxStore{publishedAt: make(map[string]time.Time)}
}

func (s *OutboxStore) Append(_ context.Context, events []outbox.Event) error {
	if len(events) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range events {
		s.events = append(s.events, e)
	}
	return nil
}

func (s *OutboxStore) Fetch(_ context.Context, limit int) ([]outbox.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = 100
	}
	result := make([]outbox.Event, 0, limit)
	for _, e := range s.events {
		if _, ok := s.publishedAt[e.EventID]; ok {
			continue
		}
		result = append(result, e)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (s *OutboxStore) MarkPublished(_ context.Context, eventIDs []string, at time.Time) error {
	if len(eventIDs) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range eventIDs {
		s.publishedAt[id] = at
	}
	return nil
}

func (s *OutboxStore) CountUnpublished(_ context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var count int64
	for _, e := range s.events {
		if _, ok := s.publishedAt[e.EventID]; !ok {
			count++
		}
	}
	return count, nil
}
