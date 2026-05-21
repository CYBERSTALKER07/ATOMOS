package driver

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Repository is the persistence seam for the driver module.
type Repository interface {
	Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error
}

// SpannerRepository durably persists outbox events within a Spanner transaction.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository configures a Spanner backend seam.
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

func (r *SpannerRepository) Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner driver repository: nil client")
	}

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

	_, err := r.client.ReadWriteTransaction(ctx, func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
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
		return fmt.Errorf("driver outbox persist: %w", err)
	}

	return nil
}

type inMemoryRepository struct{}

func NewInMemoryRepository() Repository {
	return &inMemoryRepository{}
}

type inMemoryTxnBuffer struct {
	events []outbox.Event
}

func (b *inMemoryTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func (r *inMemoryRepository) Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error {
	if mutate != nil {
		if err := mutate(); err != nil {
			return err
		}
	}
	buf := &inMemoryTxnBuffer{}
	if emit != nil {
		_ = emit(buf)
	}
	return nil
}
