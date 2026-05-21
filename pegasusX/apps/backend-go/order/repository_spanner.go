package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// SpannerRepository persists order rows in Spanner and writes emitted outbox
// events in the same RW transaction.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository builds a Spanner-backed order repository.
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

// CreateOrder writes the Orders row and any emitted outbox events atomically.
func (r *SpannerRepository) CreateOrder(ctx context.Context, o Order, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner order repository: nil client")
	}

	lineItemsRaw, err := json.Marshal(o.LineItems)
	if err != nil {
		return fmt.Errorf("marshal order line items: %w", err)
	}

	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := []*spanner.Mutation{
			spanner.InsertMap("Orders", map[string]any{
				"OrderId":       o.OrderID,
				"SupplierId":    o.SupplierID,
				"RetailerId":    o.RetailerID,
				"WarehouseId":   o.WarehouseID,
				"Status":        string(o.Status),
				"LineItemsJson": lineItemsRaw,
				"TotalMinor":    o.TotalMinor,
				"Currency":      o.Currency,
				"H3Cell":        o.H3Cell,
				"Lat":           o.Lat,
				"Lng":           o.Lng,
				"Version":       o.Version,
				"CreatedAt":     o.CreatedAt.UTC(),
				"UpdatedAt":     o.UpdatedAt.UTC(),
			}),
		}

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
		return fmt.Errorf("create order transaction: %w", err)
	}

	return nil
}

// UpdateOrder overrides an order state and emits outbox events automatically.
func (r *SpannerRepository) UpdateOrder(ctx context.Context, o Order, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner order repository: nil client")
	}

	lineItemsRaw, err := json.Marshal(o.LineItems)
	if err != nil {
		return fmt.Errorf("marshal order line items: %w", err)
	}

	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{o.OrderID}, []string{"Version"})
		if err != nil {
			return err
		}
		var version int64
		if err := row.Columns(&version); err != nil {
			return err
		}
		if version != o.Version {
			return fmt.Errorf("optimistic concurrency conflict: expected %d, got %d", o.Version, version)
		}

		o.Version++
		o.UpdatedAt = time.Now().UTC()

		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("Orders", map[string]any{
				"OrderId":       o.OrderID,
				"WarehouseId":   o.WarehouseID,
				"Status":        string(o.Status),
				"LineItemsJson": lineItemsRaw,
				"TotalMinor":    o.TotalMinor,
				"Currency":      o.Currency,
				"H3Cell":        o.H3Cell,
				"Lat":           o.Lat,
				"Lng":           o.Lng,
				"Version":       o.Version,
				"UpdatedAt":     o.UpdatedAt,
			}),
		}

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
		return fmt.Errorf("update order transaction: %w", err)
	}

	return nil
}

// GetOrder fetches one order aggregate by id.
func (r *SpannerRepository) GetOrder(ctx context.Context, orderID string) (Order, bool, error) {
	if r == nil || r.client == nil {
		return Order{}, false, fmt.Errorf("spanner order repository: nil client")
	}

	row, err := r.client.Single().ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{
		"OrderId",
		"SupplierId",
		"RetailerId",
		"WarehouseId",
		"Status",
		"LineItemsJson",
		"TotalMinor",
		"Currency",
		"H3Cell",
		"Lat",
		"Lng",
		"Version",
		"CreatedAt",
		"UpdatedAt",
	})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return Order{}, false, nil
		}
		return Order{}, false, fmt.Errorf("read order %s: %w", orderID, err)
	}

	var (
		statusRaw    string
		lineItemsRaw []byte
		o            Order
	)
	if err := row.Columns(
		&o.OrderID,
		&o.SupplierID,
		&o.RetailerID,
		&o.WarehouseID,
		&statusRaw,
		&lineItemsRaw,
		&o.TotalMinor,
		&o.Currency,
		&o.H3Cell,
		&o.Lat,
		&o.Lng,
		&o.Version,
		&o.CreatedAt,
		&o.UpdatedAt,
	); err != nil {
		return Order{}, false, fmt.Errorf("scan order %s: %w", orderID, err)
	}

	o.Status = Status(statusRaw)
	if len(lineItemsRaw) > 0 {
		if err := json.Unmarshal(lineItemsRaw, &o.LineItems); err != nil {
			return Order{}, false, fmt.Errorf("unmarshal order line items %s: %w", orderID, err)
		}
	}

	return o, true, nil
}
