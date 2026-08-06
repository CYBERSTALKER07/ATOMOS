package stocklots

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// TemperatureReadingView is one cold-chain sample.
type TemperatureReadingView struct {
	ReadingID  string  `json:"reading_id"`
	ManifestID string  `json:"manifest_id"`
	SensorID   string  `json:"sensor_id,omitempty"`
	RecordedAt string  `json:"recorded_at"`
	TempC      float64 `json:"temp_c"`
	Lat        float64 `json:"lat,omitempty"`
	Lng        float64 `json:"lng,omitempty"`
	Excursion  bool    `json:"excursion,omitempty"`
}

// IngestTemperatureInTxn stores a reading; on excursion quarantines AVAILABLE lots on the manifest's warehouse.
func IngestTemperatureInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, manifestID, sensorID string, tempC, lat, lng float64, minC, maxC float64) (*TemperatureReadingView, error) {
	manifestID = strings.TrimSpace(manifestID)
	if manifestID == "" {
		return nil, fmt.Errorf("manifest_id required")
	}
	if maxC == 0 && minC == 0 {
		minC, maxC = 0, 8 // default chilled band
	}
	excursion := tempC < minC || tempC > maxC
	id := uuid.NewString()
	now := time.Now().UTC()
	cols := map[string]any{
		"ReadingId":  id,
		"ManifestId": manifestID,
		"RecordedAt": now,
		"TempC":      tempC,
		"CreatedAt":  spanner.CommitTimestamp,
	}
	if s := strings.TrimSpace(sensorID); s != "" {
		cols["SensorId"] = s
	}
	if lat != 0 || lng != 0 {
		cols["Lat"] = lat
		cols["Lng"] = lng
	}
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("TemperatureReadings", cols)}); err != nil {
		return nil, err
	}
	if excursion {
		if err := quarantineManifestLotsInTxn(ctx, txn, manifestID); err != nil {
			return nil, err
		}
	}
	return &TemperatureReadingView{
		ReadingID: id, ManifestID: manifestID, SensorID: strings.TrimSpace(sensorID),
		RecordedAt: now.Format(time.RFC3339Nano), TempC: tempC, Lat: lat, Lng: lng, Excursion: excursion,
	}, nil
}

func quarantineManifestLotsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, manifestID string) error {
	mRow, err := txn.ReadRow(ctx, "SupplierTruckManifests", spanner.Key{manifestID}, []string{"WarehouseId", "SupplierId"})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			return nil
		}
		return err
	}
	var wid, sid string
	if err := mRow.Columns(&wid, &sid); err != nil {
		return err
	}
	orderIDs, err := loadManifestOrderIDs(ctx, txn, manifestID)
	if err != nil {
		return err
	}
	products := map[string]struct{}{}
	for _, oid := range orderIDs {
		lines, err := loadOrderLineQtys(ctx, txn, oid)
		if err != nil {
			continue
		}
		for _, l := range lines {
			products[l.SKU] = struct{}{}
		}
	}
	for pid := range products {
		iter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT LotId FROM StockLots WHERE WarehouseId = @wid AND ProductId = @pid AND Status = 'AVAILABLE'`,
			Params: map[string]any{"wid": wid, "pid": pid},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return err
			}
			var lotID string
			if err := row.Columns(&lotID); err != nil {
				iter.Stop()
				return err
			}
			if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("StockLots", map[string]any{
				"LotId":     lotID,
				"Status":    "QUARANTINE",
				"UpdatedAt": spanner.CommitTimestamp,
			})}); err != nil {
				iter.Stop()
				return err
			}
		}
		iter.Stop()
		if err := RollupInventoryV2InTxn(ctx, txn, sid, wid, pid); err != nil {
			return err
		}
	}
	return nil
}

// ListTemperatureReadings lists recent readings for a manifest.
func ListTemperatureReadings(ctx context.Context, client *spanner.Client, manifestID string) ([]TemperatureReadingView, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ReadingId, ManifestId, SensorId, RecordedAt, TempC, Lat, Lng
		      FROM TemperatureReadings WHERE ManifestId = @mid ORDER BY RecordedAt DESC LIMIT 100`,
		Params: map[string]any{"mid": strings.TrimSpace(manifestID)},
	})
	defer iter.Stop()
	var out []TemperatureReadingView
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var v TemperatureReadingView
		var sensor spanner.NullString
		var recorded spanner.NullTime
		var lat, lng spanner.NullFloat64
		if err := row.Columns(&v.ReadingID, &v.ManifestID, &sensor, &recorded, &v.TempC, &lat, &lng); err != nil {
			return nil, err
		}
		if sensor.Valid {
			v.SensorID = sensor.StringVal
		}
		if recorded.Valid {
			v.RecordedAt = recorded.Time.UTC().Format(time.RFC3339Nano)
		}
		if lat.Valid {
			v.Lat = lat.Float64
		}
		if lng.Valid {
			v.Lng = lng.Float64
		}
		out = append(out, v)
	}
	if out == nil {
		out = []TemperatureReadingView{}
	}
	return out, nil
}

// InventoryReconcileReport compares V2 vs lot sums.
type InventoryReconcileReport struct {
	WarehouseID string                   `json:"warehouse_id"`
	SupplierID  string                   `json:"supplier_id"`
	Matched     int                      `json:"matched"`
	Mismatches  []InventoryReconcileRow  `json:"mismatches"`
}

// InventoryReconcileRow is one SKU drift.
type InventoryReconcileRow struct {
	ProductID     string `json:"product_id"`
	V2OnHand      int64  `json:"v2_on_hand"`
	LotsOnHand    int64  `json:"lots_on_hand"`
	V2Reserved    int64  `json:"v2_reserved"`
	LotsReserved  int64  `json:"lots_reserved"`
}

// ReconcileInventoryV2 asserts SupplierInventoryV2 ≡ sum AVAILABLE lots.
func ReconcileInventoryV2(ctx context.Context, client *spanner.Client, supplierID, warehouseID string) (*InventoryReconcileReport, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT v.ProductId, v.QuantityOnHand, v.QuantityReserved,
		             COALESCE(l.qoh, 0), COALESCE(l.qr, 0)
		      FROM SupplierInventoryV2 v
		      LEFT JOIN (
		        SELECT ProductId, SUM(QuantityOnHand) AS qoh, SUM(QuantityReserved) AS qr
		        FROM StockLots
		        WHERE SupplierId = @sid AND WarehouseId = @wid AND Status = 'AVAILABLE'
		        GROUP BY ProductId
		      ) l ON l.ProductId = v.ProductId
		      WHERE v.SupplierId = @sid AND v.WarehouseId = @wid`,
		Params: map[string]any{"sid": supplierID, "wid": warehouseID},
	})
	defer iter.Stop()
	rep := &InventoryReconcileReport{WarehouseID: warehouseID, SupplierID: supplierID, Mismatches: []InventoryReconcileRow{}}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var pid string
		var v2q, v2r, lq, lr int64
		if err := row.Columns(&pid, &v2q, &v2r, &lq, &lr); err != nil {
			return nil, err
		}
		if v2q == lq && v2r == lr {
			rep.Matched++
			continue
		}
		rep.Mismatches = append(rep.Mismatches, InventoryReconcileRow{
			ProductID: pid, V2OnHand: v2q, LotsOnHand: lq, V2Reserved: v2r, LotsReserved: lr,
		})
	}
	return rep, nil
}
