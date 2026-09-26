package analytics

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type RoutePerformance struct {
	RouteId            string
	SupplierId         spanner.NullString
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
	Query(ctx context.Context, stmt spanner.Statement) *spanner.RowIterator
}

type spannerClientApplier struct {
	client *spanner.Client
}

func (s *spannerClientApplier) Apply(ctx context.Context, ms []*spanner.Mutation, opts ...spanner.ApplyOption) (time.Time, error) {
	return s.client.Apply(ctx, ms, opts...)
}

func (s *spannerClientApplier) Query(ctx context.Context, stmt spanner.Statement) *spanner.RowIterator {
	return s.client.Single().Query(ctx, stmt)
}

// NewRouteAnalyticsWorkerFromClient builds a worker backed by a Spanner client.
func NewRouteAnalyticsWorkerFromClient(client *spanner.Client) *RouteAnalyticsWorker {
	return &RouteAnalyticsWorker{applier: &spannerClientApplier{client: client}}
}

type RouteAnalyticsWorker struct {
	applier SpannerApplier
}

func NewRouteAnalyticsWorker(applier SpannerApplier) *RouteAnalyticsWorker {
	return &RouteAnalyticsWorker{applier: applier}
}

func (w *RouteAnalyticsWorker) ComputeAndSave(ctx context.Context, routeId string) error {
	if w == nil || w.applier == nil {
		return nil
	}
	perf := RoutePerformance{
		RouteId:    routeId,
		ComputedAt: time.Now().UTC(),
	}

	// Manifest metadata for supplier, planned stops, driver, replans, duration bounds.
	manifestIter := w.applier.Query(ctx, spanner.Statement{
		SQL: `
			SELECT SupplierId, DriverId, StopCount, ReplanCount, DispatchedAt, CompletedAt
			FROM SupplierTruckManifests
			WHERE RouteId = @routeId
			ORDER BY UpdatedAt DESC LIMIT 1`,
		Params: map[string]any{"routeId": routeId},
	})
	if manifestIter != nil {
		if row, err := manifestIter.Next(); err == nil {
			var supplier, driver spanner.NullString
			var stopCount, replan spanner.NullInt64
			var dispatched, completed spanner.NullTime
			if err := row.Columns(&supplier, &driver, &stopCount, &replan, &dispatched, &completed); err == nil {
				perf.SupplierId = supplier
				perf.DriverId = driver
				perf.PlannedStops = stopCount
				perf.ReplanCount = replan
				if dispatched.Valid && completed.Valid {
					sec := completed.Time.Sub(dispatched.Time).Seconds()
					if sec > 0 {
						perf.ActualDurationSec = spanner.NullInt64{Int64: int64(sec), Valid: true}
					}
				}
			}
		}
		manifestIter.Stop()
	}

	if !perf.SupplierId.Valid || perf.SupplierId.StringVal == "" {
		sidIter := w.applier.Query(ctx, spanner.Statement{
			SQL:    `SELECT SupplierId FROM Orders WHERE RouteId = @routeId LIMIT 1`,
			Params: map[string]any{"routeId": routeId},
		})
		if sidIter != nil {
			if row, err := sidIter.Next(); err == nil {
				var sid string
				if err := row.Column(0, &sid); err == nil && sid != "" {
					perf.SupplierId = spanner.NullString{StringVal: sid, Valid: true}
				}
			}
			sidIter.Stop()
		}
	}

	actualIter := w.applier.Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM Orders WHERE RouteId = @routeId AND Status = 'COMPLETED'`,
		Params: map[string]any{"routeId": routeId},
	})
	if actualIter != nil {
		if row, err := actualIter.Next(); err == nil {
			var cnt int64
			if err := row.Column(0, &cnt); err == nil {
				perf.ActualStops = spanner.NullInt64{Int64: cnt, Valid: true}
			}
		}
		actualIter.Stop()
	}

	etaIter := w.applier.Query(ctx, spanner.Statement{
		SQL:    `SELECT MIN(WindowStart), MAX(WindowEnd) FROM RouteETAs WHERE RouteId = @routeId`,
		Params: map[string]any{"routeId": routeId},
	})
	if etaIter != nil {
		if row, err := etaIter.Next(); err == nil {
			var start, end spanner.NullTime
			if err := row.Columns(&start, &end); err == nil && start.Valid && end.Valid {
				sec := end.Time.Sub(start.Time).Seconds()
				if sec > 0 {
					perf.PlannedDurationSec = spanner.NullInt64{Int64: int64(sec), Valid: true}
				}
			}
		}
		etaIter.Stop()
	}

	if !perf.ActualDurationSec.Valid && perf.PlannedDurationSec.Valid {
		perf.ActualDurationSec = perf.PlannedDurationSec
	}

	m, err := spanner.InsertOrUpdateStruct("RoutePerformanceAnalytics", perf)
	if err != nil {
		return err
	}
	_, err = w.applier.Apply(ctx, []*spanner.Mutation{m})
	return err
}

// RunNightlyWorker computes analytics for recently completed routes.
func (w *RouteAnalyticsWorker) RunNightlyWorker(ctx context.Context, interval time.Duration) {
	if w == nil || w.applier == nil {
		return
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	_ = w.RunBatch(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = w.RunBatch(ctx)
		}
	}
}

func (w *RouteAnalyticsWorker) RunBatch(ctx context.Context) error {
	if w == nil || w.applier == nil {
		return nil
	}
	since := time.Now().UTC().Add(-48 * time.Hour)
	iter := w.applier.Query(ctx, spanner.Statement{
		SQL: `
			SELECT RouteId FROM SupplierTruckManifests
			WHERE State = 'COMPLETED' AND RouteId IS NOT NULL
			  AND CompletedAt >= @since
			LIMIT 500`,
		Params: map[string]any{"since": since},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var routeID string
		if err := row.Column(0, &routeID); err != nil {
			return err
		}
		if err := w.ComputeAndSave(ctx, routeID); err != nil {
			continue
		}
	}
	return nil
}
