package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	"google.golang.org/api/iterator"
)

type spannerSupplyRow struct {
	RequestID    string
	WarehouseID  string
	SupplierID   string
	State        string
	FactoryID    string
	TransferMode string
	ProjectedVU  int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (s *Service) listSupplyRequestsFromSpanner(ctx context.Context) ([]SupplyRequest, error) {
	if s.spannerClient == nil {
		return nil, fmt.Errorf("spanner_not_configured")
	}
	factoryID := strings.TrimSpace(s.factoryNodeID)
	stmt := spanner.Statement{
		SQL: `SELECT sr.RequestId, sr.WarehouseId, sr.SupplierId, sr.State, sr.ProjectedUnits, sr.CreatedAt, sr.UpdatedAt,
		             COALESCE(sr.FactoryId, w.PrimaryFactoryId, ''), COALESCE(sr.TransferMode, w.TransferMode, 'TRUCK')
		      FROM WarehouseSupplyRequests sr
		      INNER JOIN Warehouses w ON sr.WarehouseId = w.WarehouseId
		      WHERE sr.SupplierId = @supplierId
		        AND (COALESCE(sr.FactoryId, w.PrimaryFactoryId, w.CoLocateWithFactoryId, '') = @factoryId
		             OR w.PrimaryFactoryId = @factoryId
		             OR w.CoLocateWithFactoryId = @factoryId)
		      ORDER BY sr.UpdatedAt DESC
		      LIMIT 100`,
		Params: map[string]any{
			"supplierId": s.supplierID,
			"factoryId":  factoryID,
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	rows := make([]SupplyRequest, 0, 16)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list factory supply requests: %w", err)
		}
		var rec spannerSupplyRow
		if err := row.Columns(&rec.RequestID, &rec.WarehouseID, &rec.SupplierID, &rec.State, &rec.ProjectedVU, &rec.CreatedAt, &rec.UpdatedAt, &rec.FactoryID, &rec.TransferMode); err != nil {
			return nil, fmt.Errorf("scan factory supply request: %w", err)
		}
		rows = append(rows, SupplyRequest{
			RequestID:   rec.RequestID,
			WarehouseID: rec.WarehouseID,
			Status:      rec.State,
			CreatedAt:   rec.CreatedAt.UTC().Format(time.RFC3339Nano),
			UpdatedAt:   rec.UpdatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return rows, nil
}

func (s *Service) getSupplyRequestFromSpanner(ctx context.Context, requestID string) (spannerSupplyRow, error) {
	if s.spannerClient == nil {
		return spannerSupplyRow{}, fmt.Errorf("spanner_not_configured")
	}
	stmt := spanner.Statement{
		SQL: `SELECT sr.RequestId, sr.WarehouseId, sr.SupplierId, sr.State, sr.ProjectedUnits, sr.CreatedAt, sr.UpdatedAt,
		             COALESCE(sr.FactoryId, w.PrimaryFactoryId, ''), COALESCE(sr.TransferMode, w.TransferMode, 'TRUCK')
		      FROM WarehouseSupplyRequests sr
		      INNER JOIN Warehouses w ON sr.WarehouseId = w.WarehouseId
		      WHERE sr.RequestId = @requestId`,
		Params: map[string]any{"requestId": requestID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return spannerSupplyRow{}, fmt.Errorf("request_not_found")
	}
	if err != nil {
		return spannerSupplyRow{}, err
	}
	var rec spannerSupplyRow
	if err := row.Columns(&rec.RequestID, &rec.WarehouseID, &rec.SupplierID, &rec.State, &rec.ProjectedVU, &rec.CreatedAt, &rec.UpdatedAt, &rec.FactoryID, &rec.TransferMode); err != nil {
		return spannerSupplyRow{}, err
	}
	return rec, nil
}

func (s *Service) transitionSupplyRequestSpanner(ctx context.Context, requestID, nextState string, emit func(outbox.TxnBuffer) error) error {
	if s.spannerClient == nil {
		return fmt.Errorf("spanner_not_configured")
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "WarehouseSupplyRequests", spanner.Key{requestID}, []string{"RequestId", "SupplierId", "State"})
		if err != nil {
			return err
		}
		var id, supplierID, state string
		if err := row.Columns(&id, &supplierID, &state); err != nil {
			return err
		}
		if supplierID != s.supplierID {
			return fmt.Errorf("forbidden")
		}
		muts := []*spanner.Mutation{spanner.UpdateMap("WarehouseSupplyRequests", map[string]any{
			"RequestId": requestID,
			"State":     nextState,
			"UpdatedAt": spanner.CommitTimestamp,
		})}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			for _, e := range buf.events {
				createdAt := e.CreatedAt.UTC()
				if createdAt.IsZero() {
					createdAt = time.Now().UTC()
				}
				muts = append(muts, spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
					"EventId": e.EventID, "AggregateType": e.AggregateType, "AggregateId": e.AggregateID,
					"TopicName": e.TopicName, "Payload": e.Payload, "CreatedAt": createdAt, "PublishedAt": nil,
				}))
			}
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func (s *Service) fulfillSupplyRequestSpanner(ctx context.Context, requestID string) (string, error) {
	rec, err := s.getSupplyRequestFromSpanner(ctx, requestID)
	if err != nil {
		return "", err
	}
	transferMode := supplier.NormalizeTransferMode(rec.TransferMode)
	totalVU := float64(rec.ProjectedVU)
	if totalVU <= 0 {
		totalVU = 1
	}
	factoryID := strings.TrimSpace(rec.FactoryID)
	if factoryID == "" {
		factoryID = strings.TrimSpace(s.factoryNodeID)
	}
	transferID := uuid.NewString()
	initialTransferState := "APPROVED"
	if transferMode == supplier.TransferModeInternal {
		initialTransferState = "RECEIVED"
	}

	var linkedTransferID string
	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{
			spanner.UpdateMap("WarehouseSupplyRequests", map[string]any{
				"RequestId":        requestID,
				"State":            "FULFILLED",
				"LinkedTransferId": transferID,
				"UpdatedAt":        spanner.CommitTimestamp,
			}),
			spanner.InsertOrUpdateMap("FactoryInternalTransfers", map[string]any{
				"TransferId":    transferID,
				"FactoryId":       factoryID,
				"SupplierId":      rec.SupplierID,
				"WarehouseId":     rec.WarehouseID,
				"SupplyRequestId": requestID,
				"TransferMode":    transferMode,
				"State":           initialTransferState,
				"TotalVolumeVU":   totalVU,
				"CreatedAt":       spanner.CommitTimestamp,
				"UpdatedAt":       spanner.CommitTimestamp,
			}),
		}
		evt := events.WarehouseEvent{
			BaseEvent:        events.BaseEvent{Type: events.EventWarehouseTransferCreated, Timestamp: s.now().Format(time.RFC3339Nano)},
			TransferID:       transferID,
			WarehouseID:      rec.WarehouseID,
			SupplierID:       rec.SupplierID,
			FactoryID:        factoryID,
			RequestID:        requestID,
			TransferMode:     transferMode,
			LinkedTransferID: transferID,
			Status:           initialTransferState,
			Units:            rec.ProjectedVU,
		}
		payload, err := json.Marshal(evt)
		if err != nil {
			return err
		}
		muts = append(muts, spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
			"EventId":       uuid.NewString(),
			"AggregateType": events.AggregateWarehouse,
			"AggregateId":   transferID,
			"TopicName":     events.TopicMain,
			"Payload":       payload,
			"CreatedAt":     spanner.CommitTimestamp,
			"PublishedAt":   nil,
		}))
		if initialTransferState == "RECEIVED" {
			if err := inventory.CreditBulkVUInTxn(ctx, txn, rec.WarehouseID, rec.SupplierID, rec.ProjectedVU); err != nil {
				return err
			}
			recv := events.WarehouseEvent{
				BaseEvent:   events.BaseEvent{Type: events.EventWarehouseTransferReceived, Timestamp: s.now().Format(time.RFC3339Nano)},
				TransferID:  transferID,
				WarehouseID: rec.WarehouseID,
				SupplierID:  rec.SupplierID,
				RequestID:   requestID,
				Units:       rec.ProjectedVU,
			}
			recvPayload, err := json.Marshal(recv)
			if err != nil {
				return err
			}
			muts = append(muts, spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
				"EventId": uuid.NewString(), "AggregateType": events.AggregateWarehouse, "AggregateId": transferID,
				"TopicName": events.TopicMain, "Payload": recvPayload, "CreatedAt": spanner.CommitTimestamp, "PublishedAt": nil,
			}))
		}
		linkedTransferID = transferID
		return txn.BufferWrite(muts)
	})
	return linkedTransferID, err
}
