package payload

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// SpannerRepository implements the payload repository using Spanner ReadWriteTransactions.
type SpannerRepository struct {
	client     *spanner.Client
	supplierID string
}

func NewSpannerRepository(client *spanner.Client, supplierID string) *SpannerRepository {
	return &SpannerRepository{
		client:     client,
		supplierID: supplierID,
	}
}

// spannerTxnBuffer buffers outbox events during a Spanner transaction.
type spannerTxnBuffer struct {
	events []outbox.Event
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

// RunTx executes a function within a Spanner ReadWriteTransaction and flushes the outbox buffer.
func (r *SpannerRepository) RunTx(ctx context.Context, fn func(ctx context.Context, tx PayloadTx) error, emit func(outbox.TxnBuffer) error) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		txImpl := &spannerPayloadTx{
			txn:        txn,
			supplierID: r.supplierID,
		}
		if err := fn(ctx, txImpl); err != nil {
			return err
		}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts := make([]*spanner.Mutation, 0, len(buf.events))
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
				muts = append(muts, spanner.InsertOrUpdateMap("OutboxEvents", row))
			}
			if len(muts) > 0 {
				if err := txn.BufferWrite(muts); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return err
}

func (r *SpannerRepository) Hydrate(ctx context.Context, supplierID string, s *Service) error {
	return r.RunTx(ctx, func(ctx context.Context, tx PayloadTx) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		var err error
		s.manifests, err = tx.ListManifests(ctx)
		if err != nil {
			return err
		}
		s.manifestOrders = make(map[string][]ManifestOrder)
		for _, m := range s.manifests {
			orders, err := tx.ListManifestOrders(ctx, m.ManifestID)
			if err != nil {
				return err
			}
			if len(orders) > 0 {
				s.manifestOrders[m.ManifestID] = orders
			}
		}
		s.exceptions, err = tx.ListExceptions(ctx)
		if err != nil {
			return err
		}
		return nil
	}, nil)
}

type spannerPayloadTx struct {
	txn        *spanner.ReadWriteTransaction
	supplierID string
}

func (tx *spannerPayloadTx) ListManifests(ctx context.Context) ([]ManifestRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId, State, TransferCount, TotalVolumeVU, MaxVolumeVU,
			  DriverId, VehicleId, CreatedAt, UpdatedAt, LoadingStartedAt, SealedAt
			  FROM SupplierTruckManifests WHERE SupplierId = @sid`,
		Params: map[string]interface{}{"sid": tx.supplierID},
	}
	iter := tx.txn.Query(ctx, stmt)
	defer iter.Stop()

	var manifests []ManifestRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var m ManifestRow
		var transferCount int64
		var totalVolume, maxVolume float64
		var driverID, vehicleID spanner.NullString
		var createdAt, updatedAt time.Time
		var loadingAt, sealedAt spanner.NullTime

		if err := row.Columns(&m.ManifestID, &m.State, &transferCount, &totalVolume, &maxVolume,
			&driverID, &vehicleID, &createdAt, &updatedAt,
			&loadingAt, &sealedAt); err != nil {
			return nil, err
		}
		m.StopCount = int(transferCount)
		m.TotalVolumeVU = int64(totalVolume)
		m.MaxVolumeVU = int64(maxVolume)
		m.DriverID = driverID.StringVal
		m.VehicleID = vehicleID.StringVal
		m.CreatedAt = createdAt.Format(time.RFC3339Nano)
		m.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
		if loadingAt.Valid { m.LoadingStartedAt = loadingAt.Time.Format(time.RFC3339Nano) }
		if sealedAt.Valid { m.SealedAt = sealedAt.Time.Format(time.RFC3339Nano) }
		manifests = append(manifests, m)
	}
	return manifests, nil
}

func (tx *spannerPayloadTx) SaveManifest(ctx context.Context, m ManifestRow) error {
	mut := spanner.InsertOrUpdateMap("SupplierTruckManifests", map[string]interface{}{
		"ManifestId": m.ManifestID,
		"SupplierId": tx.supplierID,
		"State": m.State,
		"TransferCount": int64(m.StopCount),
		"TotalVolumeVU": float64(m.TotalVolumeVU),
		"MaxVolumeVU": float64(m.MaxVolumeVU),
		"DriverId": spanner.NullString{StringVal: m.DriverID, Valid: m.DriverID != ""},
		"VehicleId": spanner.NullString{StringVal: m.VehicleID, Valid: m.VehicleID != ""},
		"CreatedAt": parseTime(m.CreatedAt),
		"UpdatedAt": parseTime(m.UpdatedAt),
		"LoadingStartedAt": parseNullTime(m.LoadingStartedAt),
		"SealedAt": parseNullTime(m.SealedAt),
	})
	return tx.txn.BufferWrite([]*spanner.Mutation{mut})
}

func (tx *spannerPayloadTx) ListManifestOrders(ctx context.Context, manifestID string) ([]ManifestOrder, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, State, VolumeVU, RemovedReason, UpdatedAt
			  FROM SupplierManifestOrders WHERE ManifestId = @mid`,
		Params: map[string]interface{}{"mid": manifestID},
	}
	iter := tx.txn.Query(ctx, stmt)
	defer iter.Stop()

	var orders []ManifestOrder
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var o ManifestOrder
		o.ManifestID = manifestID
		var volume float64
		var reason spanner.NullString
		var updatedAt time.Time

		if err := row.Columns(&o.OrderID, &o.State, &volume, &reason, &updatedAt); err != nil {
			return nil, err
		}
		o.VolumeVU = int64(volume)
		o.Reason = reason.StringVal
		o.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
		orders = append(orders, o)
	}
	return orders, nil
}

func (tx *spannerPayloadTx) SaveManifestOrder(ctx context.Context, mo ManifestOrder, seq int64) error {
	mut := spanner.InsertOrUpdateMap("SupplierManifestOrders", map[string]interface{}{
		"ManifestId": mo.ManifestID,
		"OrderId": mo.OrderID,
		"SupplierId": tx.supplierID,
		"SequenceIndex": seq,
		"LoadingOrder": seq,
		"VolumeVU": float64(mo.VolumeVU),
		"State": mo.State,
		"RemovedReason": spanner.NullString{StringVal: mo.Reason, Valid: mo.Reason != ""},
		"UpdatedAt": parseTime(mo.UpdatedAt),
	})
	return tx.txn.BufferWrite([]*spanner.Mutation{mut})
}

func (tx *spannerPayloadTx) ListExceptions(ctx context.Context) ([]ManifestException, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ExceptionId, ManifestId, OrderId, Reason, Metadata, AttemptCount, EscalatedAt, CreatedAt
			  FROM SupplierExceptions WHERE SupplierId = @sid`,
		Params: map[string]interface{}{"sid": tx.supplierID},
	}
	iter := tx.txn.Query(ctx, stmt)
	defer iter.Stop()

	var exceptions []ManifestException
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var e ManifestException
		var metadata spanner.NullString
		var escalatedAt spanner.NullTime
		var createdAt time.Time

		if err := row.Columns(&e.ExceptionID, &e.ManifestID, &e.OrderID, &e.Reason, &metadata, &e.AttemptCount, &escalatedAt, &createdAt); err != nil {
			return nil, err
		}
		e.Metadata = metadata.StringVal
		e.Escalated = escalatedAt.Valid
		e.CreatedAt = createdAt.Format(time.RFC3339Nano)
		exceptions = append(exceptions, e)
	}
	return exceptions, nil
}

func (tx *spannerPayloadTx) SaveException(ctx context.Context, e ManifestException) error {
	row := map[string]interface{}{
		"ExceptionId": e.ExceptionID,
		"ManifestId": e.ManifestID,
		"OrderId": e.OrderID,
		"SupplierId": tx.supplierID,
		"Reason": e.Reason,
		"Metadata": spanner.NullString{StringVal: e.Metadata, Valid: e.Metadata != ""},
		"AttemptCount": e.AttemptCount,
		"CreatedAt": parseTime(e.CreatedAt),
	}
	if e.Escalated {
		row["EscalatedAt"] = parseTime(e.CreatedAt) // Assuming escalated at creation for simplicity
	} else {
		row["EscalatedAt"] = spanner.NullTime{}
	}
	mut := spanner.InsertOrUpdateMap("SupplierExceptions", row)
	return tx.txn.BufferWrite([]*spanner.Mutation{mut})
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t
}

func parseNullTime(s string) spanner.NullTime {
	if s == "" {
		return spanner.NullTime{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil || t.IsZero() {
		return spanner.NullTime{}
	}
	return spanner.NullTime{Time: t, Valid: true}
}
