package analytics

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
)

type RoutePerformance struct {
	RouteId            string
	DriverId           spanner.NullString
	PlannedStops       spanner.NullInt64
	ActualStops        spanner.NullInt64
	PlannedDurationSec spanner.NullInt64
	ActualDurationSec  spanner.NullInt64
	ReplanCount        spanner.NullInt64
	ComputedAt         time.Time
}

// SpannerApplier makes it easier to mock the spanner client in tests.
type SpannerApplier interface {
	Apply(ctx context.Context, ms []*spanner.Mutation, opts ...spanner.ApplyOption) (time.Time, error)
}

type RouteAnalyticsWorker struct {
	applier SpannerApplier
}

func NewRouteAnalyticsWorker(applier SpannerApplier) *RouteAnalyticsWorker {
	return &RouteAnalyticsWorker{
		applier: applier,
	}
}

func (w *RouteAnalyticsWorker) ComputeAndSave(ctx context.Context, routeId string) error {
	// Stub computing metrics
	perf := RoutePerformance{
		RouteId:            routeId,
		DriverId:           spanner.NullString{StringVal: "driver-123", Valid: true},
		PlannedStops:       spanner.NullInt64{Int64: 10, Valid: true},
		ActualStops:        spanner.NullInt64{Int64: 12, Valid: true},
		PlannedDurationSec: spanner.NullInt64{Int64: 3600, Valid: true},
		ActualDurationSec:  spanner.NullInt64{Int64: 4000, Valid: true},
		ReplanCount:        spanner.NullInt64{Int64: 1, Valid: true},
		ComputedAt:         time.Now(),
	}

	m, err := spanner.InsertOrUpdateStruct("RoutePerformanceAnalytics", perf)
	if err != nil {
		return err
	}

	_, err = w.applier.Apply(ctx, []*spanner.Mutation{m})
	return err
}
