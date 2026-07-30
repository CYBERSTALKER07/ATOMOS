package eta

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
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
	RouteId         string
	Now             time.Time
	DriverLat       float64
	DriverLng       float64
	Profile         DriverProfile
	Stops           []StopInput
	ShopClosedRates map[string]float64
}

func (s *Service) RecalculateETAs(ctx context.Context, req RecalculateRequest) error {
	etas := CalculateETAs(req.Now, req.DriverLat, req.DriverLng, req.Profile, req.Stops, req.ShopClosedRates)
	if len(etas) > 0 {
		for i := range etas {
			etas[i].RouteId = req.RouteId
		}
		return s.PersistETAs(ctx, req.RouteId, etas)
	}
	return nil
}
