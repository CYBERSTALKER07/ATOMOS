package driver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Repository is the persistence seam for the driver module.
type Repository interface {
	Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error
	CreateDriver(ctx context.Context, d Driver, emit func(outbox.TxnBuffer) error) error
	GetDriver(ctx context.Context, driverID string) (Driver, error)
	UpdateDriver(ctx context.Context, d Driver, emit func(outbox.TxnBuffer) error) error
	ListDrivers(ctx context.Context, supplierID string, limit, offset int) ([]Driver, error)
	CreateVehicle(ctx context.Context, v Vehicle, emit func(outbox.TxnBuffer) error) error
	GetVehicle(ctx context.Context, vehicleID string) (Vehicle, error)
	UpdateVehicle(ctx context.Context, v Vehicle, emit func(outbox.TxnBuffer) error) error
	ListVehicles(ctx context.Context, supplierID string, limit, offset int) ([]Vehicle, error)
	FindSiblingDriversForOrder(ctx context.Context, orderID string) ([]string, error)
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

func outboxMutations(eventsList []outbox.Event) []*spanner.Mutation {
	muts := make([]*spanner.Mutation, 0, len(eventsList))
	for _, e := range eventsList {
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
		}
		muts = append(muts, spanner.InsertOrUpdateMap("OutboxEvents", row))
	}
	return muts
}

func (r *SpannerRepository) Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner driver repository: nil client")
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
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
		if len(mutations) == 0 {
			return nil
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
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}
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

func (b *inMemoryTxnBuffer) BufferAudit(_ context.Context, e outbox.AuditEntry) error {
	return nil
}

func (r *inMemoryRepository) Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error {
	if mutate != nil {
		if err := mutate(); err != nil {
			return err
		}
	}
	if emit != nil {
		return emit(&inMemoryTxnBuffer{})
	}
	return nil
}

func (r *inMemoryRepository) CreateDriver(ctx context.Context, d Driver, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *inMemoryRepository) GetDriver(ctx context.Context, driverID string) (Driver, error) {
	return Driver{}, nil
}
func (r *inMemoryRepository) UpdateDriver(ctx context.Context, d Driver, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *inMemoryRepository) ListDrivers(ctx context.Context, supplierID string, limit, offset int) ([]Driver, error) {
	return nil, nil
}
func (r *inMemoryRepository) CreateVehicle(ctx context.Context, v Vehicle, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *inMemoryRepository) GetVehicle(ctx context.Context, vehicleID string) (Vehicle, error) {
	return Vehicle{}, nil
}
func (r *inMemoryRepository) UpdateVehicle(ctx context.Context, v Vehicle, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *inMemoryRepository) ListVehicles(ctx context.Context, supplierID string, limit, offset int) ([]Vehicle, error) {
	return nil, nil
}

func (r *inMemoryRepository) ApplyAvailability(ctx context.Context, upd AvailabilityUpdate, emit func(outbox.TxnBuffer) error) error {
	return r.Apply(ctx, nil, emit)
}

func (r *inMemoryRepository) FindSiblingDriversForOrder(ctx context.Context, orderID string) ([]string, error) {
	return nil, nil
}

func (r *SpannerRepository) FindSiblingDriversForOrder(ctx context.Context, orderID string) ([]string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT DISTINCT DriverId FROM SupplierTruckManifests
			  WHERE ManifestId IN (
				  SELECT ManifestId FROM ManifestOrders WHERE OrderId = @orderId
			  )`,
		Params: map[string]interface{}{"orderId": orderID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var driverIDs []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var driverID string
		if err := row.Column(0, &driverID); err != nil {
			return nil, err
		}
		driverIDs = append(driverIDs, driverID)
	}
	return driverIDs, nil
}
