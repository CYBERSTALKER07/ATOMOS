package retailer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// SpannerRepository persists retailer rows in Spanner and writes emitted outbox
// events in the same RW transaction.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository builds a Spanner-backed retailer repository.
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

// CreateRetailer writes the Retailers row and any emitted outbox events atomically.
func (r *SpannerRepository) CreateRetailer(ctx context.Context, ret Retailer, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner retailer repository: nil client")
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := []*spanner.Mutation{
			spanner.InsertMap("Retailers", map[string]any{
				"RetailerId":  ret.RetailerID,
				"Phone":       ret.Phone,
				"Name":        ret.Name,
				"CountryCode": ret.CountryCode,
				"Lat":         ret.Lat,
				"Lng":         ret.Lng,
				"H3Cell":      ret.H3Cell,
				"CreatedAt":   ret.CreatedAt.UTC(),
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
		return fmt.Errorf("create retailer transaction: %w", err)
	}

	return nil
}

// FindByPhone looks up a retailer by its unique phone number.
func (r *SpannerRepository) FindByPhone(ctx context.Context, phone string) (Retailer, bool, error) {
	if r == nil || r.client == nil {
		return Retailer{}, false, fmt.Errorf("spanner retailer repository: nil client")
	}

	stmt := spanner.Statement{
		SQL: `SELECT RetailerId, Phone, Name, CountryCode, Lat, Lng, H3Cell, CreatedAt
			  FROM Retailers
			  WHERE Phone = @Phone
			  LIMIT 1`,
		Params: map[string]interface{}{
			"Phone": phone,
		},
	}
	
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return Retailer{}, false, nil
	}
	if err != nil {
		return Retailer{}, false, fmt.Errorf("query retailer by phone: %w", err)
	}

	var ret Retailer
	if err := row.Columns(
		&ret.RetailerID,
		&ret.Phone,
		&ret.Name,
		&ret.CountryCode,
		&ret.Lat,
		&ret.Lng,
		&ret.H3Cell,
		&ret.CreatedAt,
	); err != nil {
		return Retailer{}, false, fmt.Errorf("scan retailer by phone: %w", err)
	}
	
	return ret, true, nil
}

// GetRetailer fetches one retailer aggregate by id.
func (r *SpannerRepository) GetRetailer(ctx context.Context, retailerID string) (Retailer, bool, error) {
	if r == nil || r.client == nil {
		return Retailer{}, false, fmt.Errorf("spanner retailer repository: nil client")
	}

	row, err := r.client.Single().ReadRow(ctx, "Retailers", spanner.Key{retailerID}, []string{
		"RetailerId",
		"Phone",
		"Name",
		"CountryCode",
		"Lat",
		"Lng",
		"H3Cell",
		"CreatedAt",
	})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return Retailer{}, false, nil
		}
		return Retailer{}, false, fmt.Errorf("read retailer %s: %w", retailerID, err)
	}

	var ret Retailer
	if err := row.Columns(
		&ret.RetailerID,
		&ret.Phone,
		&ret.Name,
		&ret.CountryCode,
		&ret.Lat,
		&ret.Lng,
		&ret.H3Cell,
		&ret.CreatedAt,
	); err != nil {
		return Retailer{}, false, fmt.Errorf("scan retailer %s: %w", retailerID, err)
	}

	return ret, true, nil
}

// UpdateRetailer updates the Retailers row and any emitted outbox events atomically.
func (r *SpannerRepository) UpdateRetailer(ctx context.Context, ret Retailer, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner retailer repository: nil client")
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("Retailers", map[string]any{
				"RetailerId":  ret.RetailerID,
				"Phone":       ret.Phone,
				"Name":        ret.Name,
				"CountryCode": ret.CountryCode,
				"Lat":         ret.Lat,
				"Lng":         ret.Lng,
				"H3Cell":      ret.H3Cell,
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
		return fmt.Errorf("update retailer transaction: %w", err)
	}

	return nil
}

// ListRetailersBySupplier lists all retailers. (Note: in PegasusX, SupplierId is not saved in Retailers table)
func (r *SpannerRepository) ListRetailersBySupplier(ctx context.Context, supplierID string) ([]Retailer, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner retailer repository: nil client")
	}

	stmt := spanner.Statement{
		SQL: `SELECT RetailerId, Phone, Name, CountryCode, Lat, Lng, H3Cell, CreatedAt
			  FROM Retailers`,
	}
	
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var retailers []Retailer
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query all retailers: %w", err)
		}

		var ret Retailer
		if err := row.Columns(
			&ret.RetailerID,
			&ret.Phone,
			&ret.Name,
			&ret.CountryCode,
			&ret.Lat,
			&ret.Lng,
			&ret.H3Cell,
			&ret.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan list retailers: %w", err)
		}
		
		// Map supplier ID since the schema does not have it. In single-tenant pegasusX this is safe.
		ret.SupplierID = supplierID
		retailers = append(retailers, ret)
	}
	
	return retailers, nil
}
