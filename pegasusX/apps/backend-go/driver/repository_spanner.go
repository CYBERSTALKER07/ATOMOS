package driver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Repository is the persistence seam for the driver module.
type Repository interface {
	Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error
}

// AvailabilityUpdate is the durable driver shift/offline row patch.
type AvailabilityUpdate struct {
	DriverID  string
	OnShift   bool
	Reason    string
	Note      string
	UpdatedAt time.Time
}

// AvailabilityWriter persists driver on-shift state in Spanner.
type AvailabilityWriter interface {
	ApplyAvailability(ctx context.Context, upd AvailabilityUpdate, emit func(outbox.TxnBuffer) error) error
}

// AvailabilityReader loads durable on-shift state from Spanner.
type AvailabilityReader func(ctx context.Context, driverID string) (onShift bool, reason, note string, ok bool, err error)

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
	audits []outbox.AuditEntry
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func (b *spannerTxnBuffer) BufferAudit(_ context.Context, e outbox.AuditEntry) error {
	b.audits = append(b.audits, e)
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
		for _, a := range buf.audits {
			mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("driver outbox persist: %w", err)
	}

	return nil
}

func (r *SpannerRepository) ApplyAvailability(ctx context.Context, upd AvailabilityUpdate, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner driver repository: nil client")
	}
	buf := &spannerTxnBuffer{}
	if emit != nil {
		if err := emit(buf); err != nil {
			return err
		}
	}
	row := map[string]any{
		"DriverId":          upd.DriverID,
		"OnShift":           upd.OnShift,
		"UpdatedAt":         upd.UpdatedAt,
		"UnavailableReason": spanner.NullString{},
		"UnavailableNote":   spanner.NullString{},
	}
	if !upd.OnShift {
		if strings.TrimSpace(upd.Reason) != "" {
			row["UnavailableReason"] = strings.TrimSpace(upd.Reason)
		}
		if strings.TrimSpace(upd.Note) != "" {
			row["UnavailableNote"] = strings.TrimSpace(upd.Note)
		}
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{spanner.UpdateMap("Drivers", row)}
		for _, e := range buf.events {
			createdAt := e.CreatedAt.UTC()
			if createdAt.IsZero() {
				createdAt = time.Now().UTC()
			}
			outboxRow := map[string]any{
				"EventId":       e.EventID,
				"AggregateType": e.AggregateType,
				"AggregateId":   e.AggregateID,
				"TopicName":     e.TopicName,
				"Payload":       e.Payload,
				"CreatedAt":     createdAt,
				"PublishedAt":   nil,
			}
			if e.PublishedAt != nil {
				outboxRow["PublishedAt"] = e.PublishedAt.UTC()
			}
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", outboxRow))
		}
		for _, a := range buf.audits {
			mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("driver availability persist: %w", err)
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

func (r *inMemoryRepository) ApplyAvailability(ctx context.Context, upd AvailabilityUpdate, emit func(outbox.TxnBuffer) error) error {
	return r.Apply(ctx, nil, emit)
}
