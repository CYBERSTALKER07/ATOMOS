package factory

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// SpannerRepository implements the Factory repository using Spanner ReadWriteTransactions.
type SpannerRepository struct {
	client      *spanner.Client
	supplierID  string
	factoryNode string
}

func NewSpannerRepository(client *spanner.Client, supplierID, factoryNodeID string) *SpannerRepository {
	return &SpannerRepository{
		client:      client,
		supplierID:  supplierID,
		factoryNode: factoryNodeID,
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
			"PublishedAt":   nil,
		}
		if e.PublishedAt != nil {
			row["PublishedAt"] = e.PublishedAt.UTC()
		}
		muts = append(muts, spanner.InsertOrUpdateMap("OutboxEvents", row))
	}
	return muts
}

// RunTx executes a function within a Spanner ReadWriteTransaction and flushes the outbox buffer.
func (r *SpannerRepository) RunTx(ctx context.Context, fn func(ctx context.Context, tx FactoryTx) error, emit func(outbox.TxnBuffer) error) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		txImpl := &spannerFactoryTx{
			txn:         txn,
			supplierID:  r.supplierID,
			factoryNode: r.factoryNode,
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

func (r *SpannerRepository) UpdateSupplyRequestState(ctx context.Context, requestID, state string, emit func(outbox.TxnBuffer) error) error {
	return r.RunTx(ctx, func(ctx context.Context, tx FactoryTx) error {
		if spTx, ok := tx.(*spannerFactoryTx); ok && spTx.txn != nil {
			return spTx.txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("WarehouseSupplyRequests", map[string]any{
				"RequestId": requestID,
				"State":     state,
				"UpdatedAt": spanner.CommitTimestamp,
			})})
		}
		return nil
	}, emit)
}

// spannerFactoryTx provides granular read/write access during a transaction.
type spannerFactoryTx struct {
	txn         *spanner.ReadWriteTransaction
	supplierID  string
	factoryNode string
}

func (tx *spannerFactoryTx) ListManifests(ctx context.Context) ([]ManifestRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId, State, TotalVolumeVU, MaxVolumeVU, StopCount, TransferCount,
			  DriverId, VehicleId, CreatedAt, UpdatedAt,
			  LoadingStartedAt, SealedAt, DispatchedAt, CompletedAt, CancelledAt
			  FROM FactoryTruckManifests WHERE FactoryId = @fid`,
		Params: map[string]interface{}{"fid": tx.factoryNode},
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
		var totalVolume, maxVolume float64
		var stopCount, transferCount int64
		var driverID, vehicleID spanner.NullString
		var createdAt, updatedAt time.Time
		var loadingAt, sealedAt, dispatchedAt, completedAt, cancelledAt spanner.NullTime

		if err := row.Columns(&m.ManifestID, &m.State, &totalVolume, &maxVolume, &stopCount, &transferCount,
			&driverID, &vehicleID, &createdAt, &updatedAt,
			&loadingAt, &sealedAt, &dispatchedAt, &completedAt, &cancelledAt); err != nil {
			return nil, err
		}
		m.TotalVolumeVU = int64(totalVolume)
		m.MaxVolumeVU = int64(maxVolume)
		m.TransferCnt = int(transferCount)
		m.DriverID = driverID.StringVal
		m.VehicleID = vehicleID.StringVal
		m.CreatedAt = createdAt.Format(time.RFC3339Nano)
		m.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
		if loadingAt.Valid { m.LoadingStartedAt = loadingAt.Time.Format(time.RFC3339Nano) }
		if sealedAt.Valid { m.SealedAt = sealedAt.Time.Format(time.RFC3339Nano) }
		if dispatchedAt.Valid { m.DispatchedAt = dispatchedAt.Time.Format(time.RFC3339Nano) }
		if completedAt.Valid { m.CompletedAt = completedAt.Time.Format(time.RFC3339Nano) }
		if cancelledAt.Valid { m.CancelledAt = cancelledAt.Time.Format(time.RFC3339Nano) }

		manifests = append(manifests, m)
	}
	return manifests, nil
}

func (tx *spannerFactoryTx) SaveManifest(ctx context.Context, m ManifestRow) error {
	mut := spanner.InsertOrUpdateMap("FactoryTruckManifests", map[string]interface{}{
		"ManifestId": m.ManifestID,
		"FactoryId": tx.factoryNode,
		"SupplierId": tx.supplierID,
		"State": m.State,
		"TotalVolumeVU": float64(m.TotalVolumeVU),
		"MaxVolumeVU": float64(m.MaxVolumeVU),
		"StopCount": int64(m.TransferCnt),
		"TransferCount": int64(m.TransferCnt),
		"DriverId": spanner.NullString{StringVal: m.DriverID, Valid: m.DriverID != ""},
		"VehicleId": spanner.NullString{StringVal: m.VehicleID, Valid: m.VehicleID != ""},
		"CreatedAt": parseTime(m.CreatedAt),
		"UpdatedAt": parseTime(m.UpdatedAt),
		"LoadingStartedAt": parseNullTime(m.LoadingStartedAt),
		"SealedAt": parseNullTime(m.SealedAt),
		"DispatchedAt": parseNullTime(m.DispatchedAt),
		"CompletedAt": parseNullTime(m.CompletedAt),
		"CancelledAt": parseNullTime(m.CancelledAt),
	})
	return tx.txn.BufferWrite([]*spanner.Mutation{mut})
}

func (tx *spannerFactoryTx) ListTransfers(ctx context.Context) ([]TransferRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT TransferId, OrderId, ManifestId, State, TotalVolumeVU,
		DriverId, VehicleId, ReassignDepth, ExceptionCount, CreatedAt, UpdatedAt
		FROM FactoryInternalTransfers WHERE FactoryId = @fid`,
		Params: map[string]interface{}{"fid": tx.factoryNode},
	}
	iter := tx.txn.Query(ctx, stmt)
	defer iter.Stop()

	var transfers []TransferRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var t TransferRow
		var orderID, manifestID, driverID, vehicleID spanner.NullString
		var totalVolume float64
		var reassignDepth, exceptionCount int64
		var createdAt, updatedAt time.Time

		if err := row.Columns(&t.TransferID, &orderID, &manifestID, &t.State, &totalVolume, &driverID, &vehicleID,
			&reassignDepth, &exceptionCount, &createdAt, &updatedAt); err != nil {
			return nil, err
		}

		t.OrderID = orderID.StringVal
		t.ManifestID = manifestID.StringVal
		t.TotalVU = int64(totalVolume)
		t.DriverID = driverID.StringVal
		t.VehicleID = vehicleID.StringVal
		t.ReassignDepth = int(reassignDepth)
		t.ExceptionCount = exceptionCount
		t.CreatedAt = createdAt.Format(time.RFC3339Nano)
		t.UpdatedAt = updatedAt.Format(time.RFC3339Nano)

		transfers = append(transfers, t)
	}
	return transfers, nil
}

func (tx *spannerFactoryTx) SaveTransfer(ctx context.Context, t TransferRow) error {
	mut := spanner.InsertOrUpdateMap("FactoryInternalTransfers", map[string]interface{}{
		"TransferId": t.TransferID,
		"FactoryId": tx.factoryNode,
		"SupplierId": tx.supplierID,
		"OrderId": spanner.NullString{StringVal: t.OrderID, Valid: t.OrderID != ""},
		"ManifestId": spanner.NullString{StringVal: t.ManifestID, Valid: t.ManifestID != ""},
		"State": t.State,
		"TotalVolumeVU": float64(t.TotalVU),
		"DriverId": spanner.NullString{StringVal: t.DriverID, Valid: t.DriverID != ""},
		"VehicleId": spanner.NullString{StringVal: t.VehicleID, Valid: t.VehicleID != ""},
		"ReassignDepth": int64(t.ReassignDepth),
		"ExceptionCount": t.ExceptionCount,
		"CreatedAt": parseTime(t.CreatedAt),
		"UpdatedAt": parseTime(t.UpdatedAt),
	})
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

func (r *SpannerRepository) Hydrate(ctx context.Context, factoryID string, s *Service) error {
	return r.RunTx(ctx, func(ctx context.Context, tx FactoryTx) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		var err error
		s.manifests, err = tx.ListManifests(ctx)
		if err != nil {
			return err
		}
		s.transfers, err = tx.ListTransfers(ctx)
		if err != nil {
			return err
		}
		s.rebuildManifestTransfersLocked()
		return nil
	}, nil)
}
