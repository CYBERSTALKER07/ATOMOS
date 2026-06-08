package factory

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

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

type spannerTxnBuffer struct {
	events []outbox.Event
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

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
	return nil
}

type spannerFactoryTx struct {
	txn         *spanner.ReadWriteTransaction
	supplierID  string
	factoryNode string
}

func (tx *spannerFactoryTx) GetManifest(ctx context.Context, manifestID string) (ManifestRow, error) {
	row, err := tx.txn.ReadRow(ctx, "FactoryTruckManifests", spanner.Key{manifestID}, []string{
		"State", "TotalVolumeVU", "MaxVolumeVU", "StopCount", "TransferCount",
		"DriverId", "VehicleId", "CreatedAt", "UpdatedAt",
		"LoadingStartedAt", "SealedAt", "DispatchedAt", "CompletedAt", "CancelledAt",
	})
	if err != nil {
		return ManifestRow{}, err
	}
	var m ManifestRow
	m.ManifestID = manifestID
	var totalVolume, maxVolume float64
	var stopCount, transferCount int64
	var driverID, vehicleID spanner.NullString
	var createdAt, updatedAt time.Time
	var loadingAt, sealedAt, dispatchedAt, completedAt, cancelledAt spanner.NullTime

	if err := row.Columns(&m.State, &totalVolume, &maxVolume, &stopCount, &transferCount,
		&driverID, &vehicleID, &createdAt, &updatedAt,
		&loadingAt, &sealedAt, &dispatchedAt, &completedAt, &cancelledAt); err != nil {
		return ManifestRow{}, err
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

	return m, nil
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

func (tx *spannerFactoryTx) GetTransfer(ctx context.Context, transferID string) (TransferRow, error) {
	row, err := tx.txn.ReadRow(ctx, "FactoryInternalTransfers", spanner.Key{transferID}, []string{
		"OrderId", "ManifestId", "State", "TotalVolumeVU", "DriverId", "VehicleId",
		"ReassignDepth", "ExceptionCount", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		return TransferRow{}, err
	}
	var t TransferRow
	t.TransferID = transferID
	var orderID, manifestID, driverID, vehicleID spanner.NullString
	var totalVolume float64
	var reassignDepth, exceptionCount int64
	var createdAt, updatedAt time.Time

	if err := row.Columns(&orderID, &manifestID, &t.State, &totalVolume, &driverID, &vehicleID,
		&reassignDepth, &exceptionCount, &createdAt, &updatedAt); err != nil {
		return TransferRow{}, err
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

	return t, nil
}

func (tx *spannerFactoryTx) ListTransfers(ctx context.Context) ([]TransferRow, error) {
	return tx.queryTransfers(ctx, `SELECT TransferId, OrderId, ManifestId, State, TotalVolumeVU,
		DriverId, VehicleId, ReassignDepth, ExceptionCount, CreatedAt, UpdatedAt
		FROM FactoryInternalTransfers WHERE FactoryId = @fid`, map[string]interface{}{"fid": tx.factoryNode})
}

func (tx *spannerFactoryTx) ListManifestTransfers(ctx context.Context, manifestID string) ([]TransferRow, error) {
	return tx.queryTransfers(ctx, `SELECT TransferId, OrderId, ManifestId, State, TotalVolumeVU,
		DriverId, VehicleId, ReassignDepth, ExceptionCount, CreatedAt, UpdatedAt
		FROM FactoryInternalTransfers WHERE ManifestId = @mid AND FactoryId = @fid`, 
		map[string]interface{}{"mid": manifestID, "fid": tx.factoryNode})
}

func (tx *spannerFactoryTx) GetUnassignedTransfers(ctx context.Context) ([]TransferRow, error) {
	return tx.queryTransfers(ctx, `SELECT TransferId, OrderId, ManifestId, State, TotalVolumeVU,
		DriverId, VehicleId, ReassignDepth, ExceptionCount, CreatedAt, UpdatedAt
		FROM FactoryInternalTransfers WHERE State = 'CREATED' AND FactoryId = @fid`, 
		map[string]interface{}{"fid": tx.factoryNode})
}

func (tx *spannerFactoryTx) queryTransfers(ctx context.Context, query string, params map[string]interface{}) ([]TransferRow, error) {
	stmt := spanner.Statement{SQL: query, Params: params}
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

func (tx *spannerFactoryTx) SaveManifestTransition(ctx context.Context, manifestID string, t ManifestTransition) error {
	return nil // Not strictly required yet
}

func (tx *spannerFactoryTx) SaveManifestReassignment(ctx context.Context, r ManifestReassignment) error {
	return nil 
}

func (tx *spannerFactoryTx) SaveManifestException(ctx context.Context, e ManifestException) error {
	return nil 
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
