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
	attempts    map[string]int64
	deadLetters map[string]outbox.Event
}

// NewOutboxStore returns an in-memory outbox store for local scaffold runs.
func NewOutboxStore() *OutboxStore {
	return &OutboxStore{
		publishedAt: make(map[string]time.Time),
		attempts:    make(map[string]int64),
		deadLetters: make(map[string]outbox.Event),
	}
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
		if _, ok := s.deadLetters[e.EventID]; ok {
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
			if _, dead := s.deadLetters[e.EventID]; !dead {
				count++
			}
		}
	}
	return count, nil
}

// RecordPublishFailures implements outbox.Store: increments per-event attempts and
// dead-letters events reaching maxAttempts.
func (s *OutboxStore) RecordPublishFailures(_ context.Context, eventIDs []string, _ string, maxAttempts int64) ([]string, error) {
	if len(eventIDs) == 0 {
		return nil, nil
	}
	if maxAttempts <= 0 {
		maxAttempts = 20
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var deadLettered []string
	for _, id := range eventIDs {
		s.attempts[id]++
		if s.attempts[id] < maxAttempts {
			continue
		}
		for _, e := range s.events {
			if e.EventID == id {
				s.deadLetters[id] = e
				deadLettered = append(deadLettered, id)
				break
			}
		}
	}
	return deadLettered, nil
}
