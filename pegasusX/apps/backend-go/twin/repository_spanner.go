package twin

import (
	"context"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) GetRouteTwin(ctx context.Context, routeID string) (*RouteTwinView, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT RouteId, SupplierId, DriverId, Status, CurrentLat, CurrentLng, CurrentH3, LocationAt, RemainingStops, CapacityUsedWeight, CapacityUsedVolume, LastEventAt, UpdatedAt FROM RouteTwins WHERE RouteId = @routeID`,
		Params: map[string]interface{}{"routeID": routeID},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, err
	}

	var rt RouteTwin
	if err := row.ToStruct(&rt); err != nil {
		return nil, err
	}

	stopsStmt := spanner.Statement{
		SQL:    `SELECT RouteID, StopID, Sequence, Status, PredictedArrival, WindowStart, WindowEnd, DeliveredGrossMinor, RemainingGrossMinor, UpdatedAt FROM StopTwins WHERE RouteID = @routeID ORDER BY Sequence ASC`,
		Params: map[string]interface{}{"routeID": routeID},
	}
	var stops []StopTwin
	stopsIter := r.client.Single().Query(ctx, stopsStmt)
	defer stopsIter.Stop()
	err = stopsIter.Do(func(row *spanner.Row) error {
		var st StopTwin
		if err := row.ToStruct(&st); err != nil {
			return err
		}
		stops = append(stops, st)
		return nil
	})
	if err != nil {
		return nil, err
	}

	inv, err := r.GetVehicleInventory(ctx, routeID)
	if err != nil {
		return nil, err
	}

	return &RouteTwinView{
		RouteTwin: rt,
		Stops:     stops,
		Inventory: inv,
	}, nil
}

func (r *SpannerRepository) ListActiveRouteTwins(ctx context.Context, zoneH3 string) ([]RouteTwinView, error) {
	var query string
	params := map[string]interface{}{}
	if zoneH3 != "" {
		query = `SELECT RouteId, SupplierId, DriverId, Status, CurrentLat, CurrentLng, CurrentH3, LocationAt, RemainingStops, CapacityUsedWeight, CapacityUsedVolume, LastEventAt, UpdatedAt FROM RouteTwins WHERE Status = 'ACTIVE' AND CurrentH3 = @zoneH3`
		params["zoneH3"] = zoneH3
	} else {
		query = `SELECT RouteId, SupplierId, DriverId, Status, CurrentLat, CurrentLng, CurrentH3, LocationAt, RemainingStops, CapacityUsedWeight, CapacityUsedVolume, LastEventAt, UpdatedAt FROM RouteTwins WHERE Status = 'ACTIVE'`
	}

	stmt := spanner.Statement{
		SQL:    query,
		Params: params,
	}

	var activeRoutes []RouteTwin
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	err := iter.Do(func(row *spanner.Row) error {
		var rt RouteTwin
		if err := row.ToStruct(&rt); err != nil {
			return err
		}
		activeRoutes = append(activeRoutes, rt)
		return nil
	})
	if err != nil {
		return nil, err
	}

	var routes []RouteTwinView
	for _, rt := range activeRoutes {
		stopsStmt := spanner.Statement{
			SQL:    `SELECT RouteID, StopID, Sequence, Status, PredictedArrival, WindowStart, WindowEnd, DeliveredGrossMinor, RemainingGrossMinor, UpdatedAt FROM StopTwins WHERE RouteID = @routeID ORDER BY Sequence ASC`,
			Params: map[string]interface{}{"routeID": rt.RouteID},
		}
		var stops []StopTwin
		stopsIter := r.client.Single().Query(ctx, stopsStmt)
		defer stopsIter.Stop()
		err = stopsIter.Do(func(sRow *spanner.Row) error {
			var st StopTwin
			if err := sRow.ToStruct(&st); err != nil {
				return err
			}
			stops = append(stops, st)
			return nil
		})
		if err != nil {
			return nil, err
		}

		inv, err := r.GetVehicleInventory(ctx, rt.RouteID)
		if err != nil {
			return nil, err
		}

		routes = append(routes, RouteTwinView{
			RouteTwin: rt,
			Stops:     stops,
			Inventory: inv,
		})
	}

	return routes, nil
}

func (r *SpannerRepository) GetVehicleInventory(ctx context.Context, routeID string) ([]VehicleInventory, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT RouteID, Sku, QtyOnVehicle, UpdatedAt FROM VehicleInventory WHERE RouteID = @routeID`,
		Params: map[string]interface{}{"routeID": routeID},
	}
	var invs []VehicleInventory
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	err := iter.Do(func(row *spanner.Row) error {
		var inv VehicleInventory
		if err := row.ToStruct(&inv); err != nil {
			return err
		}
		invs = append(invs, inv)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if invs == nil {
		invs = []VehicleInventory{}
	}
	return invs, nil
}

func (r *SpannerRepository) GetStopTwin(ctx context.Context, routeID, stopID string) (*StopTwin, error) {
	row, err := r.client.Single().ReadRow(ctx, "StopTwins", spanner.Key{routeID, stopID},
		[]string{"RouteID", "StopID", "Sequence", "Status", "PredictedArrival", "WindowStart", "WindowEnd", "DeliveredGrossMinor", "RemainingGrossMinor", "UpdatedAt"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	var st StopTwin
	if err := row.ToStruct(&st); err != nil {
		return nil, err
	}
	return &st, nil
}

func (r *SpannerRepository) SaveRouteTwin(ctx context.Context, rt RouteTwin) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		existingRow, err := txn.ReadRow(ctx, "RouteTwins", spanner.Key{rt.RouteID}, []string{
			"RouteId", "SupplierId", "DriverId", "Status", "CurrentLat", "CurrentLng", "CurrentH3",
			"LocationAt", "RemainingStops", "CapacityUsedWeight", "CapacityUsedVolume", "LastEventAt", "UpdatedAt",
		})
		if err == nil {
			var existing RouteTwin
			if err := existingRow.ToStruct(&existing); err == nil {
				if rt.SupplierID == "" {
					rt.SupplierID = existing.SupplierID
				}
				if rt.DriverID == "" {
					rt.DriverID = existing.DriverID
				}
				if rt.Status == "" {
					rt.Status = existing.Status
				}
				if rt.RemainingStops == 0 && existing.RemainingStops != 0 {
					rt.RemainingStops = existing.RemainingStops
				}
				if rt.CurrentLat == 0 && rt.CurrentLng == 0 && (existing.CurrentLat != 0 || existing.CurrentLng != 0) {
					rt.CurrentLat = existing.CurrentLat
					rt.CurrentLng = existing.CurrentLng
					rt.CurrentH3 = existing.CurrentH3
					rt.LocationAt = existing.LocationAt
				}
				if rt.CapacityUsedWeight == 0 && existing.CapacityUsedWeight != 0 {
					rt.CapacityUsedWeight = existing.CapacityUsedWeight
				}
				if rt.CapacityUsedVolume == 0 && existing.CapacityUsedVolume != 0 {
					rt.CapacityUsedVolume = existing.CapacityUsedVolume
				}
			}
		} else if spanner.ErrCode(err) != codes.NotFound {
			return err
		}

		if rt.SupplierID == "" || rt.DriverID == "" {
			stmt := spanner.Statement{
				SQL:    `SELECT SupplierId, DriverId FROM Orders WHERE RouteId = @routeID LIMIT 1`,
				Params: map[string]interface{}{"routeID": rt.RouteID},
			}
			iter := txn.Query(ctx, stmt)
			defer iter.Stop()
			if row, err := iter.Next(); err == nil {
				var supID, drvID spanner.NullString
				_ = row.ColumnByName("SupplierId", &supID)
				_ = row.ColumnByName("DriverId", &drvID)
				if rt.SupplierID == "" && supID.Valid {
					rt.SupplierID = supID.StringVal
				}
				if rt.DriverID == "" && drvID.Valid {
					rt.DriverID = drvID.StringVal
				}
			}
		}

		if rt.SupplierID == "" {
			rt.SupplierID = "unknown_supplier"
		}
		if rt.DriverID == "" {
			rt.DriverID = "unknown_driver"
		}
		if rt.Status == "" {
			rt.Status = "ACTIVE"
		}

		m, err := spanner.InsertOrUpdateStruct("RouteTwins", rt)
		if err != nil {
			return err
		}
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, "RouteTwin", rt.RouteID, events.TopicRealtime, map[string]any{
			"type":        "ROUTE_TWIN_SAVED",
			"route_id":    rt.RouteID,
			"supplier_id": rt.SupplierID,
			"driver_id":   rt.DriverID,
			"status":      rt.Status,
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
	return err
}

func (r *SpannerRepository) SaveStopTwin(ctx context.Context, st StopTwin) error {
	m, err := spanner.InsertOrUpdateStruct("StopTwins", st)
	if err != nil {
		return err
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, "StopTwin", st.StopID, events.TopicRealtime, map[string]any{
			"type":     "STOP_TWIN_SAVED",
			"route_id": st.RouteID,
			"stop_id":  st.StopID,
			"status":   st.Status,
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
	return err
}

func (r *SpannerRepository) SaveVehicleInventory(ctx context.Context, inv VehicleInventory) error {
	if inv.QtyOnVehicle < 0 {
		inv.QtyOnVehicle = 0
	}
	m, err := spanner.InsertOrUpdateStruct("VehicleInventory", inv)
	if err != nil {
		return err
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, "VehicleInventory", inv.RouteID+":"+inv.Sku, events.TopicRealtime, map[string]any{
			"type":           "VEHICLE_INVENTORY_SAVED",
			"route_id":       inv.RouteID,
			"sku":            inv.Sku,
			"qty_on_vehicle": inv.QtyOnVehicle,
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
	return err
}

func (r *SpannerRepository) RebuildRouteTwin(ctx context.Context, routeID string, view RouteTwinView) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var muts []*spanner.Mutation

		muts = append(muts, spanner.Delete("StopTwins", spanner.Key{routeID}))
		muts = append(muts, spanner.Delete("VehicleInventory", spanner.Key{routeID}))

		if view.RouteTwin.SupplierID == "" {
			view.RouteTwin.SupplierID = "unknown_supplier"
		}
		if view.RouteTwin.DriverID == "" {
			view.RouteTwin.DriverID = "unknown_driver"
		}
		if view.RouteTwin.Status == "" {
			view.RouteTwin.Status = "ACTIVE"
		}

		mRoute, err := spanner.InsertOrUpdateStruct("RouteTwins", view.RouteTwin)
		if err != nil {
			return err
		}
		muts = append(muts, mRoute)

		for _, st := range view.Stops {
			m, err := spanner.InsertOrUpdateStruct("StopTwins", st)
			if err != nil {
				return err
			}
			muts = append(muts, m)
		}
		for _, inv := range view.Inventory {
			if inv.QtyOnVehicle < 0 {
				inv.QtyOnVehicle = 0
			}
			m, err := spanner.InsertOrUpdateStruct("VehicleInventory", inv)
			if err != nil {
				return err
			}
			muts = append(muts, m)
		}

		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, "RouteTwin", routeID, events.TopicRealtime, map[string]any{
			"type":     "ROUTE_TWIN_REBUILT",
			"route_id": routeID,
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
	return err
}
