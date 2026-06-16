package returns

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"google.golang.org/api/iterator"
)

// AmendReturnLine is one return row created during order amend.
type AmendReturnLine struct {
	ReturnID    string
	SKU         string
	RejectedQty int64
	Reason      string
	DriverNotes string
}

// AmendContext carries order metadata stamped onto new SupplierReturns rows.
type AmendContext struct {
	OrderID     string
	ManifestID  string
	DriverID    string
	WarehouseID string
	SupplierID  string
	Returns     []AmendReturnLine
}

// StampAmendReturns enriches SupplierReturns inserts with manifest context.
func StampAmendReturns(ctx AmendContext) map[string]map[string]any {
	out := make(map[string]map[string]any, len(ctx.Returns))
	for _, line := range ctx.Returns {
		returnID := strings.TrimSpace(line.ReturnID)
		if returnID == "" {
			continue
		}
		out[returnID] = map[string]any{
			"ManifestId":     nullableString(ctx.ManifestID),
			"DriverId":       nullableString(ctx.DriverID),
			"WarehouseId":    nullableString(ctx.WarehouseID),
			"ExpectedQty":    line.RejectedQty,
			"ReceivedQty":    int64(0),
			"PhysicalStatus": PhysicalPending,
		}
	}
	return out
}

// OnDriverReturnComplete marks pending returns as ON_TRUCK for the completed manifest.
func (s *Service) OnDriverReturnComplete(ctx context.Context, returned manifest.ReturnedManifest) error {
	if s == nil || s.spanner == nil {
		return nil
	}
	manifestID := strings.TrimSpace(returned.ManifestID)
	driverID := strings.TrimSpace(returned.DriverID)
	if manifestID == "" && driverID == "" {
		return nil
	}
	_, err := s.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		sql := `SELECT ReturnId FROM SupplierReturns
		        WHERE PhysicalStatus = @pending
		          AND Status = @fin_pending`
		params := map[string]any{
			"pending":     PhysicalPending,
			"fin_pending": FinancialPending,
		}
		if manifestID != "" {
			sql += " AND ManifestId = @manifest_id"
			params["manifest_id"] = manifestID
		} else {
			sql += " AND DriverId = @driver_id"
			params["driver_id"] = driverID
		}
		iter := txn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
		defer iter.Stop()
		var mutations []*spanner.Mutation
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			var returnID string
			if err := row.Columns(&returnID); err != nil {
				return err
			}
			mutations = append(mutations, spanner.UpdateMap("SupplierReturns", map[string]any{
				"ReturnId":       returnID,
				"PhysicalStatus": PhysicalOnTruck,
				"ManifestId":     nullableString(manifestID),
				"DriverId":       nullableString(driverID),
				"WarehouseId":    nullableString(returned.WarehouseID),
			}))
		}
		if len(mutations) == 0 {
			return nil
		}
		return txn.BufferWrite(mutations)
	})
	return err
}

// CheckWarehouseApproach evaluates driver telemetry against warehouse depot and
// broadcasts DRIVER_RETURN_APPROACHING when a truck with ON_TRUCK returns is near.
func (s *Service) CheckWarehouseApproach(ctx context.Context, driverID, supplierID string, lat, lng float64) error {
	if s == nil || s.spanner == nil || lat == 0 || lng == 0 {
		return nil
	}
	driverID = strings.TrimSpace(driverID)
	if driverID == "" {
		return nil
	}
	if s.shouldDedupApproach(driverID) {
		return nil
	}

	warehouseID, manifestID, count, err := s.pendingOnTruckSummary(ctx, driverID)
	if err != nil || count == 0 || warehouseID == "" {
		return err
	}
	depot, err := fetchWarehouseDepot(ctx, s.spanner, warehouseID)
	if err != nil {
		return nil
	}
	if depot.Lat == 0 && depot.Lng == 0 {
		return nil
	}
	dist := proximity.HaversineDistance(lat, lng, depot.Lat, depot.Lng)
	if dist >= 0.100 {
		return nil
	}

	_, err = s.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `UPDATE SupplierReturns SET PhysicalStatus = @arrived
			      WHERE DriverId = @driver_id AND PhysicalStatus = @on_truck`,
			Params: map[string]any{
				"arrived":   PhysicalArrived,
				"driver_id": driverID,
				"on_truck":  PhysicalOnTruck,
			},
		}
		_, err := txn.Update(ctx, stmt)
		return err
	})
	if err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]any{
		"type":         events.EventDriverReturnApproaching,
		"driver_id":    driverID,
		"supplier_id":  strings.TrimSpace(supplierID),
		"warehouse_id": warehouseID,
		"manifest_id":  manifestID,
		"return_count": count,
		"distance_km":  dist,
		"timestamp":    s.now().UTC().Format(time.RFC3339Nano),
	})
	if s.payloadHub != nil {
		s.payloadHub.Broadcast(ctx, "warehouse:"+warehouseID, payload)
	}
	if s.warehouseHub != nil {
		s.warehouseHub.Broadcast(ctx, "warehouse:"+warehouseID, payload)
	}
	if s.supplierHub != nil && supplierID != "" {
		s.supplierHub.Broadcast(ctx, "supplier:"+supplierID, payload)
	}
	s.markApproachDedup(driverID)
	return nil
}

func (s *Service) shouldDedupApproach(driverID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	last, ok := s.approachDedup[driverID]
	if !ok {
		return false
	}
	return s.now().Sub(last) < s.approachDedupTTL
}

func (s *Service) markApproachDedup(driverID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.approachDedup[driverID] = s.now()
}

func (s *Service) pendingOnTruckSummary(ctx context.Context, driverID string) (warehouseID, manifestID string, count int64, err error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, ManifestId, COUNT(*) AS Cnt
		      FROM SupplierReturns
		      WHERE DriverId = @driver_id AND PhysicalStatus = @on_truck
		      GROUP BY WarehouseId, ManifestId
		      ORDER BY Cnt DESC
		      LIMIT 1`,
		Params: map[string]any{
			"driver_id": driverID,
			"on_truck":  PhysicalOnTruck,
		},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return "", "", 0, nil
	}
	if err != nil {
		return "", "", 0, err
	}
	var whNull, manNull spanner.NullString
	if err := row.Columns(&whNull, &manNull, &count); err != nil {
		return "", "", 0, err
	}
	if whNull.Valid {
		warehouseID = whNull.StringVal
	}
	if manNull.Valid {
		manifestID = manNull.StringVal
	}
	return warehouseID, manifestID, count, nil
}

func nullableString(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

type depotCoords struct {
	Lat float64
	Lng float64
}

func fetchWarehouseDepot(ctx context.Context, client *spanner.Client, warehouseID string) (depotCoords, error) {
	if client == nil || strings.TrimSpace(warehouseID) == "" {
		return depotCoords{}, fmt.Errorf("missing warehouse id")
	}
	stmt := spanner.Statement{
		SQL:    `SELECT COALESCE(Lat, 0), COALESCE(Lng, 0) FROM Warehouses WHERE WarehouseId = @wid`,
		Params: map[string]any{"wid": warehouseID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return depotCoords{}, fmt.Errorf("warehouse not found")
	}
	if err != nil {
		return depotCoords{}, err
	}
	var depot depotCoords
	if err := row.Columns(&depot.Lat, &depot.Lng); err != nil {
		return depotCoords{}, err
	}
	return depot, nil
}
