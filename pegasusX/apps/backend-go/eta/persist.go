package eta

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/spanner"
)

func (s *Service) PersistETAs(ctx context.Context, routeId string, etas []RouteETA) error {
	var muts []*spanner.Mutation
	
	// First, delete existing uncompleted ETAs for this route to avoid stale data
	muts = append(muts, spanner.Delete("RouteETAs", spanner.KeyRange{
		Start: spanner.Key{routeId},
		End:   spanner.Key{routeId},
		Kind:  spanner.ClosedClosed,
	}))

	for _, eta := range etas {
		factorsJson, _ := json.Marshal(eta.Factors)
		muts = append(muts, spanner.Insert("RouteETAs",
			[]string{"RouteId", "StopId", "Sequence", "PredictedArrival", "WindowStart", "WindowEnd", "Confidence", "ComputedAt", "FactorsJson"},
			[]interface{}{eta.RouteId, eta.StopId, eta.Sequence, eta.PredictedArrival, eta.WindowStart, eta.WindowEnd, eta.Confidence, eta.ComputedAt, spanner.NullJSON{Value: factorsJson, Valid: true}},
		))
	}

	_, err := s.spanner.Apply(ctx, muts)
	if err != nil {
		return fmt.Errorf("persist etas: %w", err)
	}
	return nil
}
