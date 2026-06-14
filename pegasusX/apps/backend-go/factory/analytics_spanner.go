package factory

import (
	"context"
	"fmt"
	"math"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type analyticsOverview struct {
	DailyActivity    []map[string]any
	TransfersTotal   int64
	ManifestsActive  int64
	ExceptionQueue   int64
	AvgLeadTimeMins  float64
}

func (s *Service) loadAnalyticsOverview(ctx context.Context) (analyticsOverview, error) {
	if s == nil || s.spannerClient == nil {
		return analyticsOverview{}, fmt.Errorf("spanner unavailable")
	}
	factoryID := s.factoryNodeID
	supplierID := s.supplierID
	readCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	txn := s.spannerClient.Single().WithTimestampBound(spanner.MaxStaleness(15 * time.Second))
	defer txn.Close()

	overview := analyticsOverview{DailyActivity: []map[string]any{}}

	if err := txn.Query(readCtx, spanner.Statement{
		SQL: `SELECT COUNT(*) FROM FactoryInternalTransfers WHERE FactoryId = @fid`,
		Params: map[string]any{"fid": factoryID},
	}).Do(func(row *spanner.Row) error {
		return row.Columns(&overview.TransfersTotal)
	}); err != nil {
		return analyticsOverview{}, fmt.Errorf("count transfers: %w", err)
	}

	if err := txn.Query(readCtx, spanner.Statement{
		SQL: `SELECT COUNT(*) FROM FactoryTruckManifests
		      WHERE FactoryId = @fid AND State NOT IN ('COMPLETED', 'CANCELLED')`,
		Params: map[string]any{"fid": factoryID},
	}).Do(func(row *spanner.Row) error {
		return row.Columns(&overview.ManifestsActive)
	}); err != nil {
		return analyticsOverview{}, fmt.Errorf("count active manifests: %w", err)
	}

	if err := txn.Query(readCtx, spanner.Statement{
		SQL: `SELECT COUNT(*) FROM ManifestExceptions
		      WHERE SupplierId = @sid AND ResolvedAt IS NULL AND EscalatedAt IS NOT NULL`,
		Params: map[string]any{"sid": supplierID},
	}).Do(func(row *spanner.Row) error {
		return row.Columns(&overview.ExceptionQueue)
	}); err != nil {
		return analyticsOverview{}, fmt.Errorf("count manifest exceptions: %w", err)
	}

	var avgMinutes spanner.NullFloat64
	if err := txn.Query(readCtx, spanner.Statement{
		SQL: `SELECT AVG(TIMESTAMP_DIFF(CompletedAt, DispatchedAt, MINUTE))
		      FROM FactoryTruckManifests
		      WHERE FactoryId = @fid
		        AND CompletedAt IS NOT NULL
		        AND DispatchedAt IS NOT NULL
		        AND CompletedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)`,
		Params: map[string]any{"fid": factoryID},
	}).Do(func(row *spanner.Row) error {
		return row.Columns(&avgMinutes)
	}); err != nil {
		return analyticsOverview{}, fmt.Errorf("avg lead time: %w", err)
	}
	if avgMinutes.Valid {
		overview.AvgLeadTimeMins = math.Round(avgMinutes.Float64*10) / 10
	}

	iter := txn.Query(readCtx, spanner.Statement{
		SQL: `SELECT FORMAT_TIMESTAMP('%Y-%m-%d', CreatedAt) AS day, COUNT(*) AS cnt
		      FROM FactoryInternalTransfers
		      WHERE FactoryId = @fid
		        AND CreatedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)
		      GROUP BY day
		      ORDER BY day ASC`,
		Params: map[string]any{"fid": factoryID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return analyticsOverview{}, fmt.Errorf("daily activity: %w", err)
		}
		var day string
		var count int64
		if err := row.Columns(&day, &count); err != nil {
			return analyticsOverview{}, fmt.Errorf("scan daily activity: %w", err)
		}
		overview.DailyActivity = append(overview.DailyActivity, map[string]any{
			"date":      day,
			"transfers": count,
		})
	}

	return overview, nil
}
