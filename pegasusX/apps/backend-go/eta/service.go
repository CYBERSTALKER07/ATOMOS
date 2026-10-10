package eta

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

type Service struct {
	spanner *spanner.Client
}

func NewService(spannerClient *spanner.Client) *Service {
	return &Service{spanner: spannerClient}
}

func (s *Service) GetRouteETAs(ctx context.Context, routeId string) ([]RouteETA, error) {
	stmt := spanner.Statement{
		SQL: `SELECT RouteId, StopId, Sequence, PredictedArrival, WindowStart, WindowEnd, Confidence, ComputedAt, FactorsJson
			  FROM RouteETAs WHERE RouteId = @RouteId ORDER BY Sequence`,
		Params: map[string]interface{}{"RouteId": routeId},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var etas []RouteETA
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query route etas: %w", err)
		}
		var eta RouteETA
		var factors spanner.NullJSON
		if err := row.Columns(&eta.RouteId, &eta.StopId, &eta.Sequence, &eta.PredictedArrival, &eta.WindowStart, &eta.WindowEnd, &eta.Confidence, &eta.ComputedAt, &factors); err != nil {
			return nil, fmt.Errorf("scan route eta: %w", err)
		}
		if factors.Valid && factors.Value != nil {
			if factorsBytes, err := json.Marshal(factors.Value); err == nil {
				_ = json.Unmarshal(factorsBytes, &eta.Factors)
			}
		}
		etas = append(etas, eta)
	}
	return etas, nil
}

func (s *Service) GetStopETA(ctx context.Context, stopId string) (*RouteETA, error) {
	stmt := spanner.Statement{
		SQL: `SELECT RouteId, StopId, Sequence, PredictedArrival, WindowStart, WindowEnd, Confidence, ComputedAt, FactorsJson
			  FROM RouteETAs WHERE StopId = @StopId LIMIT 1`,
		Params: map[string]interface{}{"StopId": stopId},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("query stop eta: %w", err)
	}
	var eta RouteETA
	var factors spanner.NullJSON
	if err := row.Columns(&eta.RouteId, &eta.StopId, &eta.Sequence, &eta.PredictedArrival, &eta.WindowStart, &eta.WindowEnd, &eta.Confidence, &eta.ComputedAt, &factors); err != nil {
		return nil, fmt.Errorf("scan stop eta: %w", err)
	}
	if factors.Valid && factors.Value != nil {
		if factorsBytes, err := json.Marshal(factors.Value); err == nil {
			_ = json.Unmarshal(factorsBytes, &eta.Factors)
		}
	}
	return &eta, nil
}

type RecalculateRequest struct {
	RouteId         string             `json:"routeId"`
	RouteIDSnake    string             `json:"route_id"`
	Now             time.Time          `json:"now"`
	DriverLat       float64            `json:"driverLat"`
	DriverLng       float64            `json:"driverLng"`
	Profile         DriverProfile      `json:"profile"`
	Stops           []StopInput        `json:"stops"`
	ShopClosedRates map[string]float64 `json:"shopClosedRates"`
}

func (r RecalculateRequest) EffectiveRouteID() string {
	if id := strings.TrimSpace(r.RouteId); id != "" {
		return id
	}
	return strings.TrimSpace(r.RouteIDSnake)
}

func (s *Service) RecalculateETAs(ctx context.Context, req RecalculateRequest) error {
	routeID := req.EffectiveRouteID()
	etas := CalculateETAs(req.Now, req.DriverLat, req.DriverLng, req.Profile, req.Stops, req.ShopClosedRates)
	if len(etas) > 0 {
		for i := range etas {
			etas[i].RouteId = routeID
		}
		return s.PersistETAs(ctx, routeID, etas)
	}
	return nil
}

type txBuf struct {
	mutations *[]*spanner.Mutation
}

func (b *txBuf) BufferOutbox(_ context.Context, e outbox.Event) error {
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
		"SupplierId":    e.SupplierID,
	}
	if e.PublishedAt != nil {
		row["PublishedAt"] = e.PublishedAt.UTC()
	}
	*b.mutations = append(*b.mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
	return nil
}

type etaUpdatedEvent struct {
	RouteId string     `json:"routeId"`
	Reason  string     `json:"reason"`
	Stops   []RouteETA `json:"stops"`
}

// RecalculateRoute is the primary entry point to recalculate ETAs for the remaining stops on a route.
func (s *Service) RecalculateRoute(ctx context.Context, routeID string, reason string) error {
	now := time.Now().UTC()

	// 1. Load Route and DriverId
	routeStmt := spanner.Statement{
		SQL:    `SELECT DriverId FROM SupplierTruckManifests WHERE ManifestId = @RouteId LIMIT 1`,
		Params: map[string]interface{}{"RouteId": routeID},
	}
	rIter := s.spanner.Single().Query(ctx, routeStmt)
	rRow, err := rIter.Next()
	rIter.Stop()
	if err == iterator.Done {
		return fmt.Errorf("route not found: %s", routeID)
	}
	if err != nil {
		return fmt.Errorf("query route: %w", err)
	}
	var driverId spanner.NullString
	if err := rRow.Columns(&driverId); err != nil {
		return fmt.Errorf("scan route: %w", err)
	}

	// 2. Load Driver Profile
	profile := DriverProfile{DriverId: driverId.StringVal}
	if driverId.Valid && driverId.StringVal != "" {
		dsStmt := spanner.Statement{
			SQL:    `SELECT StopsPerHour, TotalOrders FROM DriverScores WHERE DriverId = @DriverId LIMIT 1`,
			Params: map[string]interface{}{"DriverId": driverId.StringVal},
		}
		dsIter := s.spanner.Single().Query(ctx, dsStmt)
		if dsRow, dsErr := dsIter.Next(); dsErr == nil {
			var stopsPerHour float64
			var totalOrders int64
			if dsRow.Columns(&stopsPerHour, &totalOrders) == nil {
				profile.RecentStopCount = totalOrders
				if stopsPerHour > 0 {
					profile.AvgStopDuration = 60.0 / stopsPerHour
				}
			}
		}
		dsIter.Stop()
	}

	// 3. Load remaining stops
	stopsStmt := spanner.Statement{
		SQL: `
			SELECT mo.OrderId as StopId, mo.OrderId, mo.SequenceIndex as Sequence, r.RetailerId, r.Lat, r.Lng
			FROM ManifestOrders mo
			JOIN Orders o ON mo.OrderId = o.OrderId
			JOIN Retailers r ON o.RetailerId = r.RetailerId
			WHERE mo.ManifestId = @RouteId AND mo.State NOT IN ('DELIVERED', 'REJECTED', 'CANCELLED')
			ORDER BY mo.SequenceIndex ASC
		`,
		Params: map[string]interface{}{"RouteId": routeID},
	}
	sIter := s.spanner.Single().Query(ctx, stopsStmt)
	defer sIter.Stop()

	var stops []StopInput
	var retailerIds []string
	for {
		row, err := sIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("query route stops: %w", err)
		}
		var stop StopInput
		var sLat, sLng spanner.NullFloat64
		if err := row.Columns(&stop.StopId, &stop.OrderId, &stop.Sequence, &stop.RetailerId, &sLat, &sLng); err != nil {
			return fmt.Errorf("scan route stop: %w", err)
		}
		stop.Lat = sLat.Float64
		stop.Lng = sLng.Float64
		stop.IsCompleted = false
		stops = append(stops, stop)
		retailerIds = append(retailerIds, stop.RetailerId)
	}

	if len(stops) == 0 {
		return nil // No remaining stops to recalculate
	}

	// 4. Load driver's current location (mocked/fallback to first stop if missing)
	driverLat, driverLng := stops[0].Lat, stops[0].Lng

	// 5. Shop closed rates - RetailerScores table not yet provisioned.
	// Gracefully default to 0 until the table is deployed (Finding 5.6).
	shopClosedRates := make(map[string]float64)

	// Calculate new ETAs
	etas := CalculateETAs(now, driverLat, driverLng, profile, stops, shopClosedRates)
	for i := range etas {
		etas[i].RouteId = routeID
	}

	// Perform transaction: Delete old, Insert new, Emit Outbox event
	_, txnErr := s.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var muts []*spanner.Mutation
		buf := &txBuf{mutations: &muts}

		muts = append(muts, spanner.Delete("RouteETAs", spanner.KeyRange{
			Start: spanner.Key{routeID},
			End:   spanner.Key{routeID},
			Kind:  spanner.ClosedClosed,
		}))

		for _, eta := range etas {
			factorsJson, _ := json.Marshal(eta.Factors)
			muts = append(muts, spanner.Insert("RouteETAs",
				[]string{"RouteId", "StopId", "Sequence", "PredictedArrival", "WindowStart", "WindowEnd", "Confidence", "ComputedAt", "FactorsJson"},
				[]interface{}{eta.RouteId, eta.StopId, eta.Sequence, eta.PredictedArrival, eta.WindowStart, eta.WindowEnd, eta.Confidence, eta.ComputedAt, spanner.NullJSON{Value: factorsJson, Valid: true}},
			))
		}

		ev := etaUpdatedEvent{
			RouteId: routeID,
			Reason:  reason,
			Stops:   etas,
		}
		_ = outbox.EmitJSON(ctx, buf, "RouteETA", routeID, "route.eta.updated", ev)

		return txn.BufferWrite(muts)
	})

	return txnErr
}
