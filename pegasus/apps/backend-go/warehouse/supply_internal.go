package warehouse

import (
	"context"
	"fmt"
	"time"

	internalKafka "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

type warehouseNearbyConfig struct {
	IsNearby       bool
	PrimaryFactory string
}

func loadWarehouseNearbyConfig(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID string) (warehouseNearbyConfig, error) {
	row, err := txn.ReadRow(ctx, "Warehouses", spanner.Key{warehouseID},
		[]string{"IsNearbyFactory", "PrimaryFactoryId"})
	if err != nil {
		return warehouseNearbyConfig{}, fmt.Errorf("read warehouse %s: %w", warehouseID, err)
	}
	var isNearby spanner.NullBool
	var primaryFactory spanner.NullString
	if err := row.Columns(&isNearby, &primaryFactory); err != nil {
		return warehouseNearbyConfig{}, err
	}
	cfg := warehouseNearbyConfig{}
	if isNearby.Valid {
		cfg.IsNearby = isNearby.Bool
	}
	if primaryFactory.Valid {
		cfg.PrimaryFactory = primaryFactory.StringVal
	}
	return cfg, nil
}

func isInternalSupplyLane(cfg warehouseNearbyConfig, factoryID string) bool {
	return cfg.IsNearby && cfg.PrimaryFactory != "" && cfg.PrimaryFactory == factoryID
}

type supplyRequestLine struct {
	ProductID    string
	Quantity     int64
	UnitVolumeVU float64
}

func applyInternalTransferFromSupplyRequest(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	requestID, warehouseID, factoryID, supplierID string,
) (string, error) {
	items, totalVU, err := readSupplyRequestLines(ctx, txn, requestID)
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", fmt.Errorf("supply request has no line items")
	}

	transferID := uuid.New().String()
	mutations := []*spanner.Mutation{
		spanner.Insert("InternalTransferOrders",
			[]string{"TransferId", "FactoryId", "WarehouseId", "SupplierId", "State", "TotalVolumeVU",
				"Source", "SupplyRequestId", "LaneType", "CreatedAt", "UpdatedAt"},
			[]interface{}{transferID, factoryID, warehouseID, supplierID, "RECEIVED", totalVU,
				"WAREHOUSE_REQUEST", requestID, "INTERNAL", spanner.CommitTimestamp, spanner.CommitTimestamp},
		),
	}

	for _, item := range items {
		itemID := uuid.New().String()
		lineVolume := item.UnitVolumeVU * float64(item.Quantity)
		mutations = append(mutations, spanner.Insert("InternalTransferItems",
			[]string{"TransferId", "ItemId", "ProductId", "Quantity", "VolumeVU"},
			[]interface{}{transferID, itemID, item.ProductID, item.Quantity, lineVolume},
		))
		if err := incrementWarehouseStock(ctx, txn, supplierID, warehouseID, item.ProductID, item.Quantity); err != nil {
			return "", err
		}
	}

	if err := txn.BufferWrite(mutations); err != nil {
		return "", err
	}

	evt := map[string]interface{}{
		"event":             internalKafka.EventTransferReceived,
		"transfer_id":       transferID,
		"factory_id":        factoryID,
		"warehouse_id":      warehouseID,
		"supplier_id":       supplierID,
		"supply_request_id": requestID,
		"lane_type":         "INTERNAL",
		"items_count":       len(items),
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
	}
	if err := outbox.EmitJSON(txn, "InternalTransferOrder", transferID,
		internalKafka.EventTransferReceived, internalKafka.TopicMain, evt,
		telemetry.TraceIDFromContext(ctx)); err != nil {
		return "", err
	}
	return transferID, nil
}

func readSupplyRequestLines(ctx context.Context, txn *spanner.ReadWriteTransaction, requestID string) ([]supplyRequestLine, float64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ProductId, RequestedQuantity, UnitVolumeVU
		      FROM SupplyRequestItems WHERE RequestId = @rid`,
		Params: map[string]interface{}{"rid": requestID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	var items []supplyRequestLine
	var totalVU float64
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		var line supplyRequestLine
		if err := row.Columns(&line.ProductID, &line.Quantity, &line.UnitVolumeVU); err != nil {
			return nil, 0, err
		}
		totalVU += line.UnitVolumeVU * float64(line.Quantity)
		items = append(items, line)
	}
	return items, totalVU, nil
}

func incrementWarehouseStock(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID, productID string, qty int64) error {
	var currentQty int64
	stockRow, stockErr := txn.ReadRow(ctx, "SupplierInventory",
		spanner.Key{productID}, []string{"QuantityAvailable"})
	if stockErr == nil {
		if err := stockRow.Columns(&currentQty); err != nil {
			return err
		}
	} else if spanner.ErrCode(stockErr) != codes.NotFound {
		return stockErr
	}

	var currentQtyV2 int64
	stockV2Row, stockV2Err := txn.ReadRow(ctx, "SupplierInventoryV2",
		spanner.Key{supplierID, warehouseID, productID}, []string{"QuantityAvailable"})
	if stockV2Err == nil {
		if err := stockV2Row.Columns(&currentQtyV2); err != nil {
			return err
		}
	} else if spanner.ErrCode(stockV2Err) != codes.NotFound {
		return stockV2Err
	}

	return txn.BufferWrite([]*spanner.Mutation{
		spanner.InsertOrUpdate("SupplierInventory",
			[]string{"ProductId", "SupplierId", "WarehouseId", "QuantityAvailable", "UpdatedAt"},
			[]interface{}{productID, supplierID, warehouseID, currentQty + qty, spanner.CommitTimestamp},
		),
		spanner.InsertOrUpdate("SupplierInventoryV2",
			[]string{"SupplierId", "WarehouseId", "ProductId", "QuantityAvailable", "UpdatedAt"},
			[]interface{}{supplierID, warehouseID, productID, currentQtyV2 + qty, spanner.CommitTimestamp},
		),
	})
}
