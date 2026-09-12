package stocklots

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// PutawayRequest credits a lot into a bin location.
type PutawayRequest struct {
	SupplierID       string
	WarehouseID      string
	ProductID        string
	LocationID       string
	LotCode          string
	Quantity         int64
	ExpiryDate       *time.Time
	ManufacturedDate *time.Time
	LotID            string // optional client-supplied for idempotency
}

// PutawayResult is the created/updated lot.
type PutawayResult struct {
	LotID            string    `json:"lot_id"`
	ProductID        string    `json:"product_id"`
	LocationID       string    `json:"location_id"`
	LotCode          string    `json:"lot_code,omitempty"`
	QuantityOnHand   int64     `json:"quantity_on_hand"`
	QuantityReserved int64     `json:"quantity_reserved"`
	Status           string    `json:"status"`
	ExpiryDate       string    `json:"expiry_date,omitempty"`
	ReceivedAt       time.Time `json:"received_at"`
}

// PutawayInTxn inserts or increments a StockLot and rolls up SupplierInventoryV2.
func PutawayInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, req PutawayRequest) (*PutawayResult, error) {
	req.SupplierID = strings.TrimSpace(req.SupplierID)
	req.WarehouseID = strings.TrimSpace(req.WarehouseID)
	req.ProductID = strings.TrimSpace(req.ProductID)
	req.LocationID = strings.TrimSpace(req.LocationID)
	if req.SupplierID == "" || req.WarehouseID == "" || req.ProductID == "" || req.LocationID == "" {
		return nil, fmt.Errorf("putaway: supplier_id, warehouse_id, product_id, location_id required")
	}
	if req.Quantity <= 0 {
		return nil, fmt.Errorf("putaway: quantity must be positive")
	}

	locRow, err := txn.ReadRow(ctx, "WarehouseLocations",
		spanner.Key{req.WarehouseID, req.LocationID},
		[]string{"LocationType", "IsActive"})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			return nil, fmt.Errorf("putaway: location not found")
		}
		return nil, err
	}
	var locType string
	var active bool
	if err := locRow.Columns(&locType, &active); err != nil {
		return nil, err
	}
	if !active {
		return nil, fmt.Errorf("putaway: location inactive")
	}

	perishable, _, err := loadProductShelfMeta(ctx, txn, req.ProductID)
	if err != nil {
		return nil, err
	}
	if perishable && (req.ExpiryDate == nil || req.ExpiryDate.IsZero()) {
		return nil, fmt.Errorf("putaway: expiry_date required for perishable product")
	}

	status := "AVAILABLE"
	if strings.EqualFold(locType, "QUARANTINE") {
		status = "QUARANTINE"
	}

	lotID := strings.TrimSpace(req.LotID)
	if lotID == "" {
		lotID = uuid.NewString()
	}

	var expiry, mfg spanner.NullDate
	if req.ExpiryDate != nil && !req.ExpiryDate.IsZero() {
		expiry = spanner.NullDate{Date: civil.DateOf(req.ExpiryDate.UTC()), Valid: true}
	}
	if req.ManufacturedDate != nil && !req.ManufacturedDate.IsZero() {
		mfg = spanner.NullDate{Date: civil.DateOf(req.ManufacturedDate.UTC()), Valid: true}
	}

	existing, err := txn.ReadRow(ctx, "StockLots", spanner.Key{lotID},
		[]string{"QuantityOnHand", "QuantityReserved", "Status", "ProductId", "WarehouseId"})
	if err != nil && spanner.ErrCode(err) != 5 {
		return nil, err
	}

	var qoh, qr int64
	if err == nil {
		var existingPID, existingWID, existingStatus string
		if err := existing.Columns(&qoh, &qr, &existingStatus, &existingPID, &existingWID); err != nil {
			return nil, err
		}
		if existingPID != req.ProductID || existingWID != req.WarehouseID {
			return nil, fmt.Errorf("putaway: lot_id collision with different product/warehouse")
		}
		qoh += req.Quantity
		if existingStatus == "DEPLETED" {
			status = "AVAILABLE"
			if strings.EqualFold(locType, "QUARANTINE") {
				status = "QUARANTINE"
			}
		} else {
			status = existingStatus
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("StockLots", map[string]any{
			"LotId":          lotID,
			"QuantityOnHand": qoh,
			"LocationId":     req.LocationID,
			"Status":         status,
			"UpdatedAt":      spanner.CommitTimestamp,
		})}); err != nil {
			return nil, err
		}
	} else {
		qoh = req.Quantity
		cols := map[string]any{
			"LotId":            lotID,
			"SupplierId":       req.SupplierID,
			"WarehouseId":      req.WarehouseID,
			"ProductId":        req.ProductID,
			"LocationId":       req.LocationID,
			"LotCode":          strings.TrimSpace(req.LotCode),
			"QuantityOnHand":   qoh,
			"QuantityReserved": int64(0),
			"Status":           status,
			"ReceivedAt":       spanner.CommitTimestamp,
			"CreatedAt":        spanner.CommitTimestamp,
			"UpdatedAt":        spanner.CommitTimestamp,
		}
		if expiry.Valid {
			cols["ExpiryDate"] = expiry.Date
		}
		if mfg.Valid {
			cols["ManufacturedDate"] = mfg.Date
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("StockLots", cols)}); err != nil {
			return nil, err
		}
	}

	if status == "AVAILABLE" {
		if err := RollupInventoryV2InTxn(ctx, txn, req.SupplierID, req.WarehouseID, req.ProductID); err != nil {
			return nil, err
		}
	}

	res := &PutawayResult{
		LotID:            lotID,
		ProductID:        req.ProductID,
		LocationID:       req.LocationID,
		LotCode:          strings.TrimSpace(req.LotCode),
		QuantityOnHand:   qoh,
		QuantityReserved: qr,
		Status:           status,
		ReceivedAt:       time.Now().UTC(),
	}
	if expiry.Valid {
		res.ExpiryDate = expiry.Date.String()
	}
	return res, nil
}

// StockLotView is a list/detail DTO.
type StockLotView struct {
	LotID            string `json:"lot_id"`
	SupplierID       string `json:"supplier_id"`
	WarehouseID      string `json:"warehouse_id"`
	ProductID        string `json:"product_id"`
	LocationID       string `json:"location_id"`
	LotCode          string `json:"lot_code,omitempty"`
	ExpiryDate       string `json:"expiry_date,omitempty"`
	ManufacturedDate string `json:"manufactured_date,omitempty"`
	QuantityOnHand   int64  `json:"quantity_on_hand"`
	QuantityReserved int64  `json:"quantity_reserved"`
	Status           string `json:"status"`
	ReceivedAt       string `json:"received_at,omitempty"`
}

// ListLotsInTxn lists lots for a warehouse with optional filters.
func ListLots(ctx context.Context, client *spanner.Client, warehouseID, productID, locationID, status string, limit int) ([]StockLotView, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner required")
	}
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" {
		return nil, fmt.Errorf("warehouse_id required")
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	sql := `SELECT LotId, SupplierId, WarehouseId, ProductId, LocationId, LotCode,
	               ExpiryDate, ManufacturedDate, QuantityOnHand, QuantityReserved, Status, ReceivedAt
	        FROM StockLots WHERE WarehouseId = @wid`
	params := map[string]any{"wid": warehouseID}
	if p := strings.TrimSpace(productID); p != "" {
		sql += ` AND ProductId = @pid`
		params["pid"] = p
	}
	if l := strings.TrimSpace(locationID); l != "" {
		sql += ` AND LocationId = @lid`
		params["lid"] = l
	}
	if s := strings.TrimSpace(status); s != "" {
		sql += ` AND Status = @st`
		params["st"] = strings.ToUpper(s)
	}
	sql += ` ORDER BY ExpiryDate ASC, ReceivedAt ASC LIMIT @lim`
	params["lim"] = int64(limit)

	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []StockLotView
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		v, err := scanLot(row)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// GetLot loads one lot.
func GetLot(ctx context.Context, client *spanner.Client, lotID string) (*StockLotView, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner required")
	}
	row, err := client.Single().ReadRow(ctx, "StockLots", spanner.Key{strings.TrimSpace(lotID)},
		[]string{"LotId", "SupplierId", "WarehouseId", "ProductId", "LocationId", "LotCode",
			"ExpiryDate", "ManufacturedDate", "QuantityOnHand", "QuantityReserved", "Status", "ReceivedAt"})
	if err != nil {
		return nil, err
	}
	v, err := scanLot(row)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func scanLot(row *spanner.Row) (StockLotView, error) {
	var v StockLotView
	var lotCode spanner.NullString
	var expiry, mfg spanner.NullDate
	var received time.Time
	if err := row.Columns(
		&v.LotID, &v.SupplierID, &v.WarehouseID, &v.ProductID, &v.LocationID, &lotCode,
		&expiry, &mfg, &v.QuantityOnHand, &v.QuantityReserved, &v.Status, &received,
	); err != nil {
		return v, err
	}
	if lotCode.Valid {
		v.LotCode = lotCode.StringVal
	}
	if expiry.Valid {
		v.ExpiryDate = expiry.Date.String()
	}
	if mfg.Valid {
		v.ManufacturedDate = mfg.Date.String()
	}
	v.ReceivedAt = received.UTC().Format(time.RFC3339)
	return v, nil
}
