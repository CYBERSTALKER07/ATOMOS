package outbox

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// SpannerStore persists and reads outbox events from OutboxEvents.
// Relay Fetch/MarkPublished authority is bound to this store when enabled.
type SpannerStore struct {
	client *spanner.Client
}

// NewSpannerStore constructs a Spanner-backed outbox store.
func NewSpannerStore(client *spanner.Client) *SpannerStore {
	return &SpannerStore{client: client}
}

// Append persists emitted events into OutboxEvents.
func (s *SpannerStore) Append(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}
	if s == nil || s.client == nil {
		return fmt.Errorf("outbox spanner store: nil client")
	}

	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := make([]*spanner.Mutation, 0, len(events))
		for _, e := range events {
			if strings.TrimSpace(e.EventID) == "" {
				return fmt.Errorf("outbox spanner store: event_id required")
			}
			if strings.TrimSpace(e.AggregateType) == "" {
				return fmt.Errorf("outbox spanner store: aggregate_type required")
			}
			if strings.TrimSpace(e.AggregateID) == "" {
				return fmt.Errorf("outbox spanner store: aggregate_id required")
			}
			if strings.TrimSpace(e.TopicName) == "" {
				return fmt.Errorf("outbox spanner store: topic_name required")
			}
			if len(e.Payload) == 0 {
				return fmt.Errorf("outbox spanner store: payload required")
			}

			createdAt := e.CreatedAt.UTC()
			if createdAt.IsZero() {
				createdAt = time.Now().UTC()
			}

			row := map[string]interface{}{
				"EventId":       e.EventID,
				"AggregateType": e.AggregateType,
				"AggregateId":   e.AggregateID,
				"TopicName":     e.TopicName,
				"Payload":       e.Payload,
				"CreatedAt":     createdAt,
				"PublishedAt":   nil,
				"ClaimedBy":     nil,
				"ClaimedUntil":  nil,
			}
			if e.PublishedAt != nil {
				row["PublishedAt"] = e.PublishedAt.UTC()
			}

			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
		}

		if len(mutations) == 0 {
			return nil
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("outbox spanner store: append: %w", err)
	}
	return nil
}

const defaultOutboxLease = 2 * time.Minute

// Fetch claims unpublished events with a short lease (ClaimedBy/ClaimedUntil)
// inside a RW transaction so multi-replica relays cannot double-publish.
func (s *SpannerStore) Fetch(ctx context.Context, limit int) ([]Event, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("outbox spanner store: nil client")
	}
	if limit <= 0 {
		limit = 100
	}
	claimant := "relay-" + uuid.NewString()
	if len(claimant) > 64 {
		claimant = claimant[:64]
	}
	now := time.Now().UTC()
	leaseUntil := now.Add(defaultOutboxLease)
	var events []Event

	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `
SELECT EventId, AggregateType, AggregateId, TopicName, Payload, CreatedAt, PublishedAt
FROM OutboxEvents@{FORCE_INDEX=Idx_OutboxEvents_Unpublished}
WHERE PublishedAt IS NULL
  AND (ClaimedUntil IS NULL OR ClaimedUntil < @now)
ORDER BY CreatedAt
LIMIT @limit`,
			Params: map[string]interface{}{
				"limit": int64(limit),
				"now":   now,
			},
		}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()

		claimed := make([]Event, 0, limit)
		mutations := make([]*spanner.Mutation, 0, limit)
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("outbox spanner store: fetch: %w", err)
			}

			var (
				eventID       string
				aggregateType string
				aggregateID   string
				topicName     string
				payload       []byte
				createdAt     time.Time
				publishedAt   spanner.NullTime
			)
			if err := row.Columns(&eventID, &aggregateType, &aggregateID, &topicName, &payload, &createdAt, &publishedAt); err != nil {
				return fmt.Errorf("outbox spanner store: fetch scan: %w", err)
			}

			e := Event{
				EventID:       eventID,
				AggregateType: aggregateType,
				AggregateID:   aggregateID,
				TopicName:     topicName,
				Payload:       payload,
				CreatedAt:     createdAt.UTC(),
			}
			if publishedAt.Valid {
				ts := publishedAt.Time.UTC()
				e.PublishedAt = &ts
			}
			claimed = append(claimed, e)
			mutations = append(mutations, spanner.UpdateMap("OutboxEvents", map[string]interface{}{
				"EventId":      eventID,
				"ClaimedBy":    claimant,
				"ClaimedUntil": leaseUntil,
			}))
		}
		if len(mutations) > 0 {
			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}
		}
		events = claimed
		return nil
	})
	if err != nil {
		return nil, err
	}
	return events, nil
}

// CountUnpublished returns the number of rows with PublishedAt IS NULL.
func (s *SpannerStore) CountUnpublished(ctx context.Context) (int64, error) {
	if s == nil || s.client == nil {
		return 0, fmt.Errorf("outbox spanner store: nil client")
	}
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM OutboxEvents@{FORCE_INDEX=Idx_OutboxEvents_Unpublished} WHERE PublishedAt IS NULL`,
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, fmt.Errorf("outbox spanner store: count: %w", err)
	}
	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, fmt.Errorf("outbox spanner store: count scan: %w", err)
	}
	return count, nil
}

// MarkPublished marks event IDs as published at the given timestamp.
func (s *SpannerStore) MarkPublished(ctx context.Context, eventIDs []string, at time.Time) error {
	if len(eventIDs) == 0 {
		return nil
	}
	if s == nil || s.client == nil {
		return fmt.Errorf("outbox spanner store: nil client")
	}

	publishedAt := at.UTC()
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := make([]*spanner.Mutation, 0, len(eventIDs))
		for _, id := range eventIDs {
			trimmed := strings.TrimSpace(id)
			if trimmed == "" {
				continue
			}
			mutations = append(mutations, spanner.UpdateMap("OutboxEvents", map[string]interface{}{
				"EventId":     trimmed,
				"PublishedAt": publishedAt,
			}))
		}
		if len(mutations) == 0 {
			return nil
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("outbox spanner store: mark published: %w", err)
	}
	return nil
}
