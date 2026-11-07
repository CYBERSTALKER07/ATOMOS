package outbox

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// SpannerStore persists and reads outbox events from OutboxEvents.
// Relay Fetch/MarkPublished authority is bound to this store when enabled.
type SpannerStore struct {
	client *spanner.Client
}

var (
	_ Store          = (*SpannerStore)(nil)
	_ ClaimReleaser  = (*SpannerStore)(nil)
	_ StuckInspector = (*SpannerStore)(nil)
)

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

			row := map[string]interface{}{
				"EventId":       e.EventID,
				"AggregateType": e.AggregateType,
				"AggregateId":   e.AggregateID,
				"TopicName":     e.TopicName,
				"Payload":       e.Payload,
				"CreatedAt":     spanner.CommitTimestamp,
				"PublishedAt":   nil,
				"UnpublishedAt": spanner.CommitTimestamp,
				"ClaimedBy":     nil,
				"ClaimedUntil":  nil,
			}
			if e.PublishedAt != nil {
				row["PublishedAt"] = e.PublishedAt.UTC()
				row["UnpublishedAt"] = nil
			}
			sid := ResolveSupplierID(e.SupplierID, e.Payload)
			row["SupplierId"] = sid
			e.SupplierID = sid

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
// Fetch claims unpublished events using a Two-Phase Claiming strategy:
// Phase 1: Lock-free snapshot read (Single().Query) to discover candidates without taking shared range locks on Idx_OutboxEvents_Unpublished.
// Phase 2: Targeted point-read CAS claim inside a ReadWriteTransaction by primary key (EventId KeySet) to eliminate Spanner abort storms.
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

	fetchLimit := limit * 4
	if fetchLimit < 100 {
		fetchLimit = 100
	}
	if fetchLimit > 500 {
		fetchLimit = 500
	}

	// Phase 1: Lock-Free Snapshot Read
	// Single().Query runs as a read-only snapshot transaction which NEVER acquires shared locks on Idx_OutboxEvents_Pending_BySupplier.
	stmt := spanner.Statement{
		SQL: `
SELECT EventId, AggregateType, AggregateId, TopicName, Payload, CreatedAt, PublishedAt, SupplierId
FROM OutboxEvents@{FORCE_INDEX=Idx_OutboxEvents_Pending_BySupplier}
WHERE UnpublishedAt IS NOT NULL
  AND (ClaimedUntil IS NULL OR ClaimedUntil < @now)
ORDER BY CreatedAt
LIMIT @limit`,
		Params: map[string]interface{}{
			"limit": int64(fetchLimit),
			"now":   now,
		},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	candidates := make([]Event, 0, fetchLimit)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("outbox spanner store: snapshot fetch candidates: %w", err)
		}

		var (
			eventID       string
			aggregateType string
			aggregateID   string
			topicName     string
			payload       []byte
			createdAt     time.Time
			publishedAt   spanner.NullTime
			supplierID    spanner.NullString
		)
		if err := row.Columns(&eventID, &aggregateType, &aggregateID, &topicName, &payload, &createdAt, &publishedAt, &supplierID); err != nil {
			return nil, fmt.Errorf("outbox spanner store: snapshot fetch scan: %w", err)
		}

		stored := ""
		if supplierID.Valid {
			stored = supplierID.StringVal
		}
		e := Event{
			EventID:       eventID,
			AggregateType: aggregateType,
			AggregateID:   aggregateID,
			TopicName:     topicName,
			Payload:       payload,
			CreatedAt:     createdAt.UTC(),
			SupplierID:    ResolveSupplierID(stored, payload),
		}
		if publishedAt.Valid {
			ts := publishedAt.Time.UTC()
			e.PublishedAt = &ts
		}
		candidates = append(candidates, e)
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// Interleave candidates across suppliers and aggregates
	selected := FairInterleave(candidates, limit)
	if len(selected) == 0 {
		return nil, nil
	}

	// Phase 2: Targeted Point-Read CAS Claim inside ReadWriteTransaction
	// Reads ONLY the candidate primary keys (KeySet). No index range scans are executed.
	var claimedEvents []Event
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		claimedEvents = claimedEvents[:0]
		mutations := make([]*spanner.Mutation, 0, len(selected))

		candidateMap := make(map[string]Event, len(selected))
		keys := make([]spanner.Key, 0, len(selected))
		for _, e := range selected {
			candidateMap[e.EventID] = e
			keys = append(keys, spanner.Key{e.EventID})
		}

		rows := txn.Read(ctx, "OutboxEvents", spanner.KeySetFromKeys(keys...), []string{"EventId", "ClaimedUntil", "PublishedAt"})
		defer rows.Stop()

		for {
			row, err := rows.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			var (
				eventID      string
				claimedUntil spanner.NullTime
				publishedAt  spanner.NullTime
			)
			if err := row.Columns(&eventID, &claimedUntil, &publishedAt); err != nil {
				return err
			}

			// Verify still unpublished and claim lease available
			if publishedAt.Valid {
				continue
			}
			if claimedUntil.Valid && !claimedUntil.Time.Before(now) {
				continue // Already claimed by a concurrent worker
			}

			mutations = append(mutations, spanner.UpdateMap("OutboxEvents", map[string]interface{}{
				"EventId":      eventID,
				"ClaimedBy":    claimant,
				"ClaimedUntil": leaseUntil,
			}))
			if e, ok := candidateMap[eventID]; ok {
				claimedEvents = append(claimedEvents, e)
			}
		}

		if len(mutations) > 0 {
			return txn.BufferWrite(mutations)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("outbox spanner store: claim transaction: %w", err)
	}

	return claimedEvents, nil
}

// CountUnpublished returns the number of rows with UnpublishedAt IS NOT NULL.
func (s *SpannerStore) CountUnpublished(ctx context.Context) (int64, error) {
	if s == nil || s.client == nil {
		return 0, fmt.Errorf("outbox spanner store: nil client")
	}
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM OutboxEvents@{FORCE_INDEX=Idx_OutboxEvents_Pending_BySupplier} WHERE UnpublishedAt IS NOT NULL`,
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

// MarkPublished marks event IDs as published at the given timestamp and clears
// active leases (ClaimedBy/ClaimedUntil) so published rows leave the claimed pool.
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
				"EventId":       trimmed,
				"PublishedAt":   publishedAt,
				"UnpublishedAt": spanner.NullTime{},
				"ClaimedBy":     spanner.NullString{},
				"ClaimedUntil":  spanner.NullTime{},
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

// Unclaim releases claimed leases on the given event IDs without incrementing
// PublishAttempts (e.g. for causal ordering skips), returning them to the pool immediately.
func (s *SpannerStore) Unclaim(ctx context.Context, eventIDs []string) error {
	if len(eventIDs) == 0 {
		return nil
	}
	if s == nil || s.client == nil {
		return fmt.Errorf("outbox spanner store: nil client")
	}

	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := make([]*spanner.Mutation, 0, len(eventIDs))
		for _, id := range eventIDs {
			trimmed := strings.TrimSpace(id)
			if trimmed == "" {
				continue
			}
			mutations = append(mutations, spanner.UpdateMap("OutboxEvents", map[string]interface{}{
				"EventId":      trimmed,
				"ClaimedBy":    spanner.NullString{},
				"ClaimedUntil": spanner.NullTime{},
			}))
		}
		if len(mutations) == 0 {
			return nil
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("outbox spanner store: unclaim: %w", err)
	}
	return nil
}

// InspectStuck queries unpublished events in read-only mode without claiming leases,
// allowing the watchdog to inspect backlog staleness without locking out relay workers.
func (s *SpannerStore) InspectStuck(ctx context.Context, limit int) ([]Event, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("outbox spanner store: nil client")
	}
	if limit <= 0 {
		limit = 100
	}
	stmt := spanner.Statement{
		SQL: `
SELECT EventId, AggregateType, AggregateId, TopicName, Payload, CreatedAt, PublishedAt, SupplierId
FROM OutboxEvents@{FORCE_INDEX=Idx_OutboxEvents_Pending_BySupplier}
WHERE UnpublishedAt IS NOT NULL
ORDER BY CreatedAt
LIMIT @limit`,
		Params: map[string]interface{}{
			"limit": int64(limit),
		},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var events []Event
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("outbox spanner store: inspect stuck: %w", err)
		}

		var (
			eventID       string
			aggregateType string
			aggregateID   string
			topicName     string
			payload       []byte
			createdAt     time.Time
			publishedAt   spanner.NullTime
			supplierID    spanner.NullString
		)
		if err := row.Columns(&eventID, &aggregateType, &aggregateID, &topicName, &payload, &createdAt, &publishedAt, &supplierID); err != nil {
			return nil, fmt.Errorf("outbox spanner store: inspect stuck scan: %w", err)
		}

		stored := ""
		if supplierID.Valid {
			stored = supplierID.StringVal
		}
		e := Event{
			EventID:       eventID,
			AggregateType: aggregateType,
			AggregateID:   aggregateID,
			TopicName:     topicName,
			Payload:       payload,
			CreatedAt:     createdAt.UTC(),
			SupplierID:    ResolveSupplierID(stored, payload),
		}
		if publishedAt.Valid {
			ts := publishedAt.Time.UTC()
			e.PublishedAt = &ts
		}
		events = append(events, e)
	}
	return events, nil
}

// RecordPublishFailures increments PublishAttempts per event and atomically moves
// events that reach maxAttempts into OutboxDeadLetters (copy + delete), so poison
// events leave the retry set with a full forensic record instead of retrying
// forever or being dropped.
func (s *SpannerStore) RecordPublishFailures(ctx context.Context, eventIDs []string, lastErr string, maxAttempts int64) ([]string, error) {
	if len(eventIDs) == 0 {
		return nil, nil
	}
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("outbox spanner store: nil client")
	}
	if maxAttempts <= 0 {
		maxAttempts = 20
	}
	if len(lastErr) > 1024 {
		lastErr = lastErr[:1024]
	}
	var deadLettered []string
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var mutations []*spanner.Mutation
		for _, id := range eventIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			row, readErr := txn.ReadRow(ctx, "OutboxEvents", spanner.Key{id}, []string{
				"EventId", "AggregateType", "AggregateId", "TopicName", "Payload", "CreatedAt", "PublishAttempts", "SupplierId",
			})
			if readErr != nil {
				if spanner.ErrCode(readErr) == codes.NotFound {
					continue
				}
				return fmt.Errorf("outbox spanner store: read for failure record %s: %w", id, readErr)
			}
			var (
				eventID, aggregateType, aggregateID, topicName string
				payload                                      []byte
				createdAt                                    time.Time
				attempts                                     int64
				supplierID                                   spanner.NullString
			)
			if scanErr := row.Columns(&eventID, &aggregateType, &aggregateID, &topicName, &payload, &createdAt, &attempts, &supplierID); scanErr != nil {
				return fmt.Errorf("outbox spanner store: scan for failure record %s: %w", id, scanErr)
			}
			attempts++
			if attempts >= maxAttempts {
				storedSupplier := ""
				if supplierID.Valid {
					storedSupplier = supplierID.StringVal
				}
				resolvedSupplier := ResolveSupplierID(storedSupplier, payload)

				cols := map[string]interface{}{
					"EventId":        eventID,
					"AggregateType":  aggregateType,
					"AggregateId":    aggregateID,
					"TopicName":      topicName,
					"Payload":        payload,
					"CreatedAt":      createdAt.UTC(),
					"DeadLetteredAt": spanner.CommitTimestamp,
					"Attempts":       attempts,
					"LastError":      lastErr,
				}
				if resolvedSupplier != "" {
					cols["SupplierId"] = resolvedSupplier
				}
				mutations = append(mutations,
					spanner.InsertOrUpdateMap("OutboxDeadLetters", cols),
					spanner.Delete("OutboxEvents", spanner.Key{id}),
				)
				deadLettered = append(deadLettered, eventID)
				continue
			}
			mutations = append(mutations, spanner.UpdateMap("OutboxEvents", map[string]interface{}{
				"EventId":         eventID,
				"PublishAttempts": attempts,
				"ClaimedBy":       spanner.NullString{},
				"ClaimedUntil":    spanner.NullTime{},
			}))
		}
		if len(mutations) == 0 {
			return nil
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return nil, fmt.Errorf("outbox spanner store: record publish failures: %w", err)
	}
	return deadLettered, nil
}
