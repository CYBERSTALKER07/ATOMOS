package factory

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// SpannerRepository provides a Spanner transaction seam for factory outbox events.
// Note: Factory domain entities remain in-memory in pegasusX scaffold for now,
// but their events are written durably to OutboxEvents.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository constructs the Spanner backend for factory.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

type spannerTxnBuffer struct {
	events []outbox.Event
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

// Apply executes the in-memory mutation and durably persists any emitted outbox events.
func (r *SpannerRepository) Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner factory repository: nil client")
	}

	// Scaffold note: factory state is in-memory and not retry-safe.
	// Mutate first, then persist events. If Spanner fails, in-memory state is dirty
	// (acceptable for scaffold).
	if mutate != nil {
		if err := mutate(); err != nil {
			return err
		}
	}

	buf := &spannerTxnBuffer{}
	if emit != nil {
		if err := emit(buf); err != nil {
			return err
		}
	}

	if len(buf.events) == 0 {
		return nil
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var mutations []*spanner.Mutation
		for _, e := range buf.events {
			createdAt := e.CreatedAt.UTC()
			if createdAt.IsZero() {
				createdAt = time.Now().UTC()
			}

			row := map[string]any{
				"EventId":       e.EventID,
				"AggregateType": e.AggregateType,
				"AggregateId":   e.AggregateID,
				"TopicName":     e.TopicName,
				"Payload":       e.Payload,
				"CreatedAt":     createdAt,
				"PublishedAt":   nil,
			}
			if e.PublishedAt != nil {
				row["PublishedAt"] = e.PublishedAt.UTC()
			}

			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("factory outbox transaction: %w", err)
	}

	return nil
}
