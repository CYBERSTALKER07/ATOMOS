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
	RequestID             string
	WarehouseID           string
	SupplierID            string
	State                 string
	FactoryID             string
	TransferMode          string
	ProjectedVU           int64
	TotalVolumeVU         float64
	Priority              string
	Notes                 string
	RegionID              string
	RequestedDeliveryDate spanner.NullTime
	LinkedTransferID      string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type factorySupplyItem struct {
	ItemID            string  `json:"item_id"`
	ProductID         string  `json:"product_id"`
	RequestedQuantity int64   `json:"requested_quantity"`
	RecommendedQty    int64   `json:"recommended_qty,omitempty"`
	UnitVolumeVU      float64 `json:"unit_volume_vu,omitempty"`
	ShippedQuantity   int64   `json:"shipped_quantity,omitempty"`
	ReceivedQuantity  int64   `json:"received_quantity,omitempty"`
}

type fulfillLineInput struct {
	ItemID          string
	ShippedQuantity int64
}

func supplyItemsToDTO(items []factorySupplyItem) []SupplyRequestItem {
	out := make([]SupplyRequestItem, 0, len(items))
	for _, item := range items {
		out = append(out, SupplyRequestItem{
			ItemID:            item.ItemID,
			ProductID:         item.ProductID,
			RequestedQuantity: item.RequestedQuantity,
			RecommendedQty:    item.RecommendedQty,
			UnitVolumeVU:      item.UnitVolumeVU,
			ShippedQuantity:   item.ShippedQuantity,
			ReceivedQuantity:  item.ReceivedQuantity,
		})
	}
	return out
}

func (s *Service) listSupplyRequestsFromSpanner(ctx context.Context) ([]SupplyRequest, error) {
	if s.spannerClient == nil {
		return nil, fmt.Errorf("spanner_not_configured")
	}
	factoryID := strings.TrimSpace(s.factoryNodeID)
	stmt := spanner.Statement{
		SQL: `SELECT sr.RequestId, sr.WarehouseId, sr.SupplierId, sr.State, sr.ProjectedUnits, sr.CreatedAt, sr.UpdatedAt,
		             COALESCE(sr.FactoryId, w.PrimaryFactoryId, ''), COALESCE(sr.TransferMode, w.TransferMode, 'TRUCK'),
		             COALESCE(sr.Priority, 'NORMAL'), COALESCE(sr.Notes, ''), COALESCE(sr.RegionId, ''),
		             sr.RequestedDeliveryDate, COALESCE(sr.TotalVolumeVU, 0), COALESCE(sr.LinkedTransferId, '')
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
	requestIDs := make([]string, 0, 16)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list factory supply requests: %w", err)
		}
		var rec spannerSupplyRow
		if err := row.Columns(&rec.RequestID, &rec.WarehouseID, &rec.SupplierID, &rec.State, &rec.ProjectedVU, &rec.CreatedAt, &rec.UpdatedAt, &rec.FactoryID, &rec.TransferMode, &rec.Priority, &rec.Notes, &rec.RegionID, &rec.RequestedDeliveryDate, &rec.TotalVolumeVU, &rec.LinkedTransferID); err != nil {
			return nil, fmt.Errorf("scan factory supply request: %w", err)
		}
		deliveryDate := ""
		if rec.RequestedDeliveryDate.Valid {
			deliveryDate = rec.RequestedDeliveryDate.Time.UTC().Format(time.RFC3339)
		}
		vu := rec.TotalVolumeVU
		if vu <= 0 && rec.ProjectedVU > 0 {
			vu = float64(rec.ProjectedVU)
		}
		rows = append(rows, SupplyRequest{
			RequestID:             rec.RequestID,
			WarehouseID:           rec.WarehouseID,
			Status:                rec.State,
			Priority:              rec.Priority,
			Notes:                 rec.Notes,
			RegionID:              rec.RegionID,
			RequestedDeliveryDate: deliveryDate,
			TotalVolumeVU:         vu,
			LinkedTransferID:      rec.LinkedTransferID,
			CreatedAt:             rec.CreatedAt.UTC().Format(time.RFC3339Nano),
			UpdatedAt:             rec.UpdatedAt.UTC().Format(time.RFC3339Nano),
		})
		requestIDs = append(requestIDs, rec.RequestID)
	}
	if len(requestIDs) > 0 {
		itemsByRequest, err := s.loadSupplyRequestItems(ctx, requestIDs)
		if err != nil {
			return nil, err
		}
		for i := range rows {
			rows[i].Items = supplyItemsToDTO(itemsByRequest[rows[i].RequestID])
		}
	}
	return rows, nil
}

func (s *Service) loadSupplyRequestItems(ctx context.Context, requestIDs []string) (map[string][]factorySupplyItem, error) {
	stmt := spanner.Statement{
		SQL: `SELECT RequestId, ItemId, ProductId, RequestedQuantity, RecommendedQuantity, UnitVolumeVU,
		             COALESCE(ShippedQuantity, 0), COALESCE(ReceivedQuantity, 0)
		      FROM WarehouseSupplyRequestItems
		      WHERE RequestId IN UNNEST(@ids)
		      ORDER BY RequestId, ProductId`,
		Params: map[string]any{"ids": requestIDs},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	out := make(map[string][]factorySupplyItem)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("load supply request items: %w", err)
		}
		var requestID, itemID, productID string
		var requested, recommended, shipped, received int64
		var unitVU float64
		if err := row.Columns(&requestID, &itemID, &productID, &requested, &recommended, &unitVU, &shipped, &received); err != nil {
			continue
		}
		out[requestID] = append(out[requestID], factorySupplyItem{
			ItemID:            itemID,
			ProductID:         productID,
			RequestedQuantity: requested,
			RecommendedQty:    recommended,
			UnitVolumeVU:      unitVU,
			ShippedQuantity:   shipped,
			ReceivedQuantity:  received,
		})
	}
	return out, nil
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

func (s *Service) fulfillSupplyRequestSpanner(ctx context.Context, requestID string, shipped []fulfillLineInput, driverID string) (string, error) {
	rec, err := s.getSupplyRequestFromSpanner(ctx, requestID)
	if err != nil {
		return "", err
	}
	itemsByRequest, err := s.loadSupplyRequestItems(ctx, []string{requestID})
	if err != nil {
		return "", err
	}
	lines := itemsByRequest[requestID]
	if len(lines) == 0 {
		return "", fmt.Errorf("supply_request_items_missing")
	}
	shippedByID := make(map[string]int64, len(shipped))
	for _, row := range shipped {
		if id := strings.TrimSpace(row.ItemID); id != "" && row.ShippedQuantity > 0 {
			shippedByID[id] = row.ShippedQuantity
		}
	}

	transferMode := supplier.NormalizeTransferMode(rec.TransferMode)
	totalVU := float64(0)
	for i := range lines {
		qty := lines[i].RequestedQuantity
		if v, ok := shippedByID[lines[i].ItemID]; ok && v > 0 {
			qty = v
		}
		lines[i].ShippedQuantity = qty
		if lines[i].UnitVolumeVU > 0 {
			totalVU += float64(qty) * lines[i].UnitVolumeVU
		} else {
			totalVU += float64(qty)
		}
	}
	if totalVU <= 0 {
		totalVU = float64(rec.ProjectedVU)
		if totalVU <= 0 {
			totalVU = 1
		}
	}
	factoryID := strings.TrimSpace(rec.FactoryID)
	if factoryID == "" {
		factoryID = strings.TrimSpace(s.factoryNodeID)
	}
	transferID := uuid.NewString()
	initialTransferState := "IN_TRANSIT"
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
				"TotalVolumeVU":    totalVU,
				"UpdatedAt":        spanner.CommitTimestamp,
			}),
		}
		for _, line := range lines {
			muts = append(muts, spanner.UpdateMap("WarehouseSupplyRequestItems", map[string]any{
				"RequestId":       requestID,
				"ItemId":          line.ItemID,
				"ShippedQuantity": line.ShippedQuantity,
			}))
		}
		transferRow := map[string]any{
			"TransferId":      transferID,
			"FactoryId":       factoryID,
			"SupplierId":      rec.SupplierID,
			"WarehouseId":     rec.WarehouseID,
			"SupplyRequestId": requestID,
			"TransferMode":    transferMode,
			"State":           initialTransferState,
			"TotalVolumeVU":   totalVU,
			"CreatedAt":       spanner.CommitTimestamp,
			"UpdatedAt":       spanner.CommitTimestamp,
		}
		if initialTransferState == "RECEIVED" {
			transferRow["ReceivedAt"] = spanner.CommitTimestamp
		}
		if driverID := strings.TrimSpace(driverID); driverID != "" && transferMode != supplier.TransferModeInternal {
			transferRow["DriverId"] = driverID
		}
		muts = append(muts, spanner.InsertOrUpdateMap("FactoryInternalTransfers", transferRow))
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
			for _, line := range lines {
				if err := inventory.CreditSupplierInventoryV2InTxn(ctx, txn, rec.SupplierID, rec.WarehouseID, line.ProductID, line.ShippedQuantity); err != nil {
					return err
				}
				muts = append(muts, spanner.UpdateMap("WarehouseSupplyRequestItems", map[string]any{
					"RequestId":        requestID,
					"ItemId":           line.ItemID,
					"ReceivedQuantity": line.ShippedQuantity,
				}))
			}
			muts = append(muts, spanner.UpdateMap("WarehouseSupplyRequests", map[string]any{
				"RequestId": requestID,
				"State":     "RECEIVED",
				"UpdatedAt": spanner.CommitTimestamp,
			}))
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
