package laborcapacity

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// RunCapacitySnapshotWorker aggregates driver availability + scores into zone capacity.
func (s *Service) RunCapacitySnapshotWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.RunCapacitySnapshot(ctx); err != nil {
				slog.Error("capacity snapshot failed", "error", err)
			}
		}
	}
}

type zoneAgg struct {
	totalCapacity float64
	usedCapacity  float64
}

// RunCapacitySnapshot computes zone capacity from availability + scores.
func (s *Service) RunCapacitySnapshot(ctx context.Context) error {
	now := time.Now().UTC()
	today := civil.DateOf(now)

	// Join DriverAvailability with DriverScores for today.
	stmt := spanner.Statement{
		SQL: `
			SELECT
				da.ZoneH3,
				da.AvailableHours,
				COALESCE(ds.StopsPerHour, 3.0) as StopsPerHour,
				COALESCE(ds.Score, 50.0) as Score
			FROM DriverAvailability da
			LEFT JOIN DriverScores ds ON da.DriverId = ds.DriverId
			WHERE da.Date = @Today AND da.Status != 'OFF'
		`,
		Params: map[string]interface{}{
			"Today": today,
		},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	zones := make(map[string]*zoneAgg)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("query zone capacity: %w", err)
		}

		var zoneH3 spanner.NullString
		var availableHours, stopsPerHour, score float64
		if err := row.Columns(&zoneH3, &availableHours, &stopsPerHour, &score); err != nil {
			return fmt.Errorf("scan zone row: %w", err)
		}

		zone := "UNKNOWN"
		if zoneH3.Valid && zoneH3.StringVal != "" {
			zone = zoneH3.StringVal
		}

		if _, ok := zones[zone]; !ok {
			zones[zone] = &zoneAgg{}
		}

		// AvailableCapacity = hours × stopsPerHour × (score/100)
		scoreFactor := score / 100.0
		zones[zone].totalCapacity += availableHours * stopsPerHour * scoreFactor
	}

	// Write zone capacities
	var mutations []*spanner.Mutation
	for zone, agg := range zones {
		mutations = append(mutations, spanner.InsertOrUpdateMap("ZoneCapacity", map[string]interface{}{
			"ZoneH3":        zone,
			"Date":          today,
			"TotalCapacity": agg.totalCapacity,
			"UsedCapacity":  agg.usedCapacity, // Phase 1: 0 until dispatch integration
			"ComputedAt":    now,
		}))
	}

	if len(mutations) > 0 {
		if _, err := s.spanner.Apply(ctx, mutations); err != nil {
			return fmt.Errorf("apply zone capacity mutations: %w", err)
		}
	}

	slog.Info("capacity snapshot complete", "zones", len(zones))
	return nil
}
