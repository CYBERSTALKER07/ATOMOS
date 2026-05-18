package bootstrap

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type outboxEventAppender interface {
	Append(ctx context.Context, events []outbox.Event) error
}

// inMemoryOutboxStore is the scaffold outbox store used until Spanner-backed
// OutboxEvents tables are wired in pegasusX.
type inMemoryOutboxStore struct {
	mu          sync.RWMutex
	events      []outbox.Event
	publishedAt map[string]time.Time
}

func newInMemoryOutboxStore() *inMemoryOutboxStore {
	return &inMemoryOutboxStore{publishedAt: make(map[string]time.Time)}
}

func (s *inMemoryOutboxStore) Append(_ context.Context, events []outbox.Event) error {
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

func (s *inMemoryOutboxStore) Fetch(_ context.Context, limit int) ([]outbox.Event, error) {
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

func (s *inMemoryOutboxStore) MarkPublished(_ context.Context, eventIDs []string, at time.Time) error {
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

// loggingOutboxPublisher is a scaffold publisher seam; production wiring swaps
// this for Kafka writer-backed Publisher.
type loggingOutboxPublisher struct {
	log *slog.Logger
}

func (p *loggingOutboxPublisher) Publish(_ context.Context, topic string, key []byte, value []byte) error {
	if p.log != nil {
		p.log.Debug("outbox published",
			"topic", topic,
			"aggregate_id", string(key),
			"payload_bytes", len(value),
		)
	}
	return nil
}
