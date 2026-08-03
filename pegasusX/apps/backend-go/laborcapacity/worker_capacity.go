package laborcapacity

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// RunCapacitySnapshotWorker triggers recomputation of today and the next 2 days.
func (s *Service) RunCapacitySnapshotWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().In(time.FixedZone("Asia/Tashkent", 5*3600))
			// Compute for today + 2 days
			if err := s.Recompute(ctx, now, now.AddDate(0, 0, 2)); err != nil {
				slog.Error("capacity snapshot failed", "error", err)
			}
		}
	}
}

type capacityOutbox struct {
	ZoneH3        string    `json:"zoneH3"`
	Date          string    `json:"date"`
	TotalCapacity float64   `json:"totalCapacity"`
	UsedCapacity  float64   `json:"usedCapacity"`
	ComputedAt    time.Time `json:"computedAt"`
}

// Recompute recalculates zone capacity for a date range globally.
func (s *Service) Recompute(ctx context.Context, from, to time.Time) error {
	return s.doRecompute(ctx, civil.DateOf(from), civil.DateOf(to), "")
}

// RecomputeZoneDay recalculates capacity for a single zone on a specific date.
func (s *Service) RecomputeZoneDay(ctx context.Context, zoneH3 string, date time.Time) error {
	d := civil.DateOf(date)
	return s.doRecompute(ctx, d, d, zoneH3)
}

func (s *Service) doRecompute(ctx context.Context, from, to civil.Date, filterZone string) error {
	now := time.Now().UTC()

	// 1. Fetch available driver capacity
	capStmt := spanner.Statement{
		SQL: `
			SELECT
				da.ZoneH3,
				da.Date,
				da.AvailableHours,
				da.Status,
				COALESCE(ds.StopsPerHour, 3.0) as StopsPerHour,
				COALESCE(ds.Score, 70.0) as Score
			FROM DriverAvailability da
			LEFT JOIN DriverScores ds ON da.DriverId = ds.DriverId
			WHERE da.Date >= @From AND da.Date <= @To
			  AND da.Status IN ('AVAILABLE', 'LIMITED')
		`,
		Params: map[string]interface{}{
			"From": from,
			"To":   to,
		},
	}
	if filterZone != "" {
		capStmt.SQL += " AND da.ZoneH3 = @Zone"
		capStmt.Params["Zone"] = filterZone
	}

	iter := s.spanner.Single().Query(ctx, capStmt)
	defer iter.Stop()

	type zoneDate struct {
		zone string
		date civil.Date
	}
	type agg struct {
		totalCapacity float64
		usedCapacity  float64
	}

	aggregates := make(map[zoneDate]*agg)

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("query capacity: %w", err)
		}

		var zoneH3 spanner.NullString
		var date civil.Date
		var availableHours, stopsPerHour, score float64
		var status string

		if err := row.Columns(&zoneH3, &date, &availableHours, &status, &stopsPerHour, &score); err != nil {
			return fmt.Errorf("scan capacity row: %w", err)
		}

		if !zoneH3.Valid || zoneH3.StringVal == "" {
			continue
		}
		
		zd := zoneDate{zone: zoneH3.StringVal, date: date}
		if _, ok := aggregates[zd]; !ok {
			aggregates[zd] = &agg{}
		}

		// Apply rules: Score fallback=70, stopsPerHour fallback=3.0 (done in SQL COALESCE)
		// LIMITED status reduces hours by 0.6
		hours := availableHours
		if status == "LIMITED" {
			hours *= 0.6
		}

		// Avoid 0 division if stopsPerHour was recorded as 0
		if stopsPerHour <= 0 {
			stopsPerHour = 3.0
		}

		scoreFactor := score / 100.0
		aggregates[zd].totalCapacity += hours * stopsPerHour * scoreFactor
	}

	// 2. Compute UsedCapacity from assigned orders
	usedStmt := spanner.Statement{
		SQL: `
			SELECT 
				ZoneH3, 
				DATE(PromisedBy, 'Asia/Tashkent') as AssignedDate,
				COUNT(*) as AssignedStops
			FROM Orders
			WHERE Status IN ('ASSIGNED', 'IN_TRANSIT')
			  AND DATE(PromisedBy, 'Asia/Tashkent') >= CAST(@From AS STRING)
			  AND DATE(PromisedBy, 'Asia/Tashkent') <= CAST(@To AS STRING)
			GROUP BY ZoneH3, AssignedDate
		`,
		Params: map[string]interface{}{
			"From": from.String(),
			"To":   to.String(),
		},
	}
	if filterZone != "" {
		usedStmt.SQL += " AND ZoneH3 = @Zone"
		usedStmt.Params["Zone"] = filterZone
	}
	
	uIter := s.spanner.Single().Query(ctx, usedStmt)
	defer uIter.Stop()
	for {
		row, err := uIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("query used capacity: %w", err)
		}

		var zoneH3 spanner.NullString
		var assignedDate spanner.NullDate
		var assignedStops int64

		if err := row.Columns(&zoneH3, &assignedDate, &assignedStops); err != nil {
			return fmt.Errorf("scan used capacity row: %w", err)
		}

		if !zoneH3.Valid || zoneH3.StringVal == "" || !assignedDate.Valid {
			continue
		}

		zd := zoneDate{zone: zoneH3.StringVal, date: assignedDate.Date}
		if _, ok := aggregates[zd]; !ok {
			aggregates[zd] = &agg{} // Track zones that have demand but no capacity (0 TotalCapacity explicitly)
		}
		aggregates[zd].usedCapacity += float64(assignedStops)
	}

	// 3. Write mutations & outbox
	var mutations []*spanner.Mutation
	var events []capacityOutbox

	for zd, a := range aggregates {
		mutations = append(mutations, spanner.InsertOrUpdateMap("ZoneCapacity", map[string]interface{}{
			"ZoneH3":        zd.zone,
			"Date":          zd.date,
			"TotalCapacity": a.totalCapacity,
			"UsedCapacity":  a.usedCapacity,
			"ComputedAt":    now,
		}))
		
		events = append(events, capacityOutbox{
			ZoneH3:        zd.zone,
			Date:          zd.date.String(),
			TotalCapacity: a.totalCapacity,
			UsedCapacity:  a.usedCapacity,
			ComputedAt:    now,
		})

		if len(mutations) >= 500 {
			if err := s.flushCapacityMutations(ctx, mutations, events); err != nil {
				return err
			}
			mutations = nil
			events = nil
		}
	}

	if len(mutations) > 0 {
		if err := s.flushCapacityMutations(ctx, mutations, events); err != nil {
			return err
		}
	}

	slog.Info("zone capacity recompute complete", "zones", len(aggregates))
	return nil
}

func (s *Service) flushCapacityMutations(ctx context.Context, mutations []*spanner.Mutation, events []capacityOutbox) error {
	_, err := s.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &txBuf{mutations: &mutations}
		for _, ev := range events {
			// capacity.zone.updated
			aggregateID := fmt.Sprintf("%s|%s", ev.ZoneH3, ev.Date)
			_ = outbox.EmitJSON(ctx, buf, "ZoneCapacity", aggregateID, "capacity.zone.updated", ev)
		}
		return txn.BufferWrite(mutations)
	})
	return err
}

