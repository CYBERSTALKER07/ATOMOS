package twin

import (
	"context"

	"cloud.google.com/go/spanner"
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
		SQL:    `SELECT RouteID, DriverID, Status, CurrentLat, CurrentLng, CurrentH3, LocationAt, RemainingStops, CapacityUsedWeight, CapacityUsedVolume, LastEventAt, UpdatedAt FROM RouteTwins WHERE RouteID = @routeID`,
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
		query = `SELECT RouteID, DriverID, Status, CurrentLat, CurrentLng, CurrentH3, LocationAt, RemainingStops, CapacityUsedWeight, CapacityUsedVolume, LastEventAt, UpdatedAt FROM RouteTwins WHERE Status = 'ACTIVE' AND CurrentH3 = @zoneH3`
		params["zoneH3"] = zoneH3
	} else {
		query = `SELECT RouteID, DriverID, Status, CurrentLat, CurrentLng, CurrentH3, LocationAt, RemainingStops, CapacityUsedWeight, CapacityUsedVolume, LastEventAt, UpdatedAt FROM RouteTwins WHERE Status = 'ACTIVE'`
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
	m, err := spanner.InsertOrUpdateStruct("RouteTwins", rt)
	if err != nil {
		return err
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	return err
}

func (r *SpannerRepository) SaveStopTwin(ctx context.Context, st StopTwin) error {
	m, err := spanner.InsertOrUpdateStruct("StopTwins", st)
	if err != nil {
		return err
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
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
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	return err
}

func (r *SpannerRepository) RebuildRouteTwin(ctx context.Context, routeID string, view RouteTwinView) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var muts []*spanner.Mutation

		muts = append(muts, spanner.Delete("StopTwins", spanner.Key{routeID}))
		muts = append(muts, spanner.Delete("VehicleInventory", spanner.Key{routeID}))

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

		return txn.BufferWrite(muts)
	})
	return err
}
