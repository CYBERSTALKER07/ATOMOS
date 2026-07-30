package laborcapacity

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// RunDriverScoreWorker recomputes driver scores on a ticker.
// Scheduled to run nightly (e.g. 02:00 Asia/Tashkent).
func (s *Service) RunDriverScoreWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	// Initial run
	_ = s.RunDriverScoreComputation(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.RunDriverScoreComputation(ctx); err != nil {
				slog.Error("driver score computation failed", "error", err)
			}
		}
	}
}

type driverScoreOutbox struct {
	DriverId   string    `json:"driverId"`
	Score      float64   `json:"score"`
	ComputedAt time.Time `json:"computedAt"`
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
	}
	if e.PublishedAt != nil {
		row["PublishedAt"] = e.PublishedAt.UTC()
	}

	*b.mutations = append(*b.mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
	return nil
}

// RunDriverScoreComputation recomputes scores for all active drivers.
func (s *Service) RunDriverScoreComputation(ctx context.Context) error {
	now := time.Now().UTC()
	windowEnd := now.Truncate(24 * time.Hour)
	windowStart := windowEnd.Add(-28 * 24 * time.Hour)

	// Analytical query across operational tables using exact metric definitions
	stmt := spanner.Statement{
		SQL: `
			WITH AssignedOrders AS (
				SELECT DriverId, COUNT(OrderId) as TotalAssigned, 
				       COUNTIF(Status = 'DELIVERED') as TotalCompleted,
				       COUNTIF(Status = 'DELIVERED' AND DeliveredAt <= PromisedBy) as OnTimeCompleted
				FROM Orders
				WHERE CreatedAt >= @Start AND CreatedAt < @End AND DriverId IS NOT NULL
				GROUP BY DriverId
			),
			DamageLines AS (
				SELECT o.DriverId, COALESCE(SUM(l.DeliveredQty), 0) as TotalDelivered, COALESCE(SUM(c.Quantity), 0) as Damaged
				FROM Orders o
				JOIN OrderLines l ON o.OrderId = l.OrderId
				LEFT JOIN Claims c ON c.OrderLineId = l.Id AND c.Type = 'DAMAGE'
				WHERE o.Status = 'DELIVERED' AND o.CreatedAt >= @Start AND o.CreatedAt < @End
				GROUP BY o.DriverId
			),
			ShopClosed AS (
				SELECT o.DriverId, COUNT(e.Id) as ShopClosedTotal,
				       COUNTIF(e.Resolution IN ('RETURN_TO_WAREHOUSE', 'FORCE')) as ShopClosedBad
				FROM Orders o
				JOIN OrderEvents e ON e.OrderId = o.OrderId AND e.Type = 'SHOP_CLOSED'
				WHERE o.CreatedAt >= @Start AND o.CreatedAt < @End
				GROUP BY o.DriverId
			),
			Feedback AS (
				SELECT DriverId, AVG(CAST(Rating AS FLOAT64)) as AvgRating
				FROM Orders
				WHERE CreatedAt >= @Start AND CreatedAt < @End AND Rating IS NOT NULL
				GROUP BY DriverId
			),
			HoursLogged AS (
				SELECT DriverId, SUM(DurationHours) as TotalHours
				FROM DriverShifts
				WHERE StartAt >= @Start AND StartAt < @End
				GROUP BY DriverId
			)
			SELECT 
				a.DriverId,
				a.TotalAssigned,
				a.TotalCompleted,
				a.OnTimeCompleted,
				COALESCE(d.TotalDelivered, 0) as TotalDelivered,
				COALESCE(d.Damaged, 0) as Damaged,
				COALESCE(s.ShopClosedTotal, 0) as ShopClosedTotal,
				COALESCE(s.ShopClosedBad, 0) as ShopClosedBad,
				COALESCE(f.AvgRating, 0.0) as AvgRating,
				COALESCE(h.TotalHours, 0.0) as TotalHours
			FROM AssignedOrders a
			LEFT JOIN DamageLines d ON a.DriverId = d.DriverId
			LEFT JOIN ShopClosed s ON a.DriverId = s.DriverId
			LEFT JOIN Feedback f ON a.DriverId = f.DriverId
			LEFT JOIN HoursLogged h ON a.DriverId = h.DriverId
			WHERE a.TotalCompleted > 0
		`,
		Params: map[string]interface{}{
			"Start": civil.DateOf(windowStart),
			"End":   civil.DateOf(windowEnd),
		},
	}
	
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var mutations []*spanner.Mutation
	var outboxEvents []driverScoreOutbox

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("query metrics: %w", err)
		}

		var (
			driverId        string
			totalAssigned   int64
			totalCompleted  int64
			onTimeCompleted int64
			totalDelivered  int64
			damaged         int64
			shopClosedTotal int64
			shopClosedBad   int64
			avgRating       float64
			totalHours      float64
		)
		if err := row.Columns(
			&driverId, &totalAssigned, &totalCompleted, &onTimeCompleted,
			&totalDelivered, &damaged, &shopClosedTotal, &shopClosedBad,
			&avgRating, &totalHours,
		); err != nil {
			return fmt.Errorf("scan columns: %w", err)
		}

		onTimeRate := 0.0
		completionRate := 0.0
		damageRate := 0.0
		shopClosedRate := 0.0
		feedbackScore := 0.75 // Default 0.75 if no feedback

		if totalCompleted > 0 {
			onTimeRate = float64(onTimeCompleted) / float64(totalCompleted)
		}
		if totalAssigned > 0 {
			completionRate = float64(totalCompleted) / float64(totalAssigned)
		}
		if totalDelivered > 0 {
			damageRate = float64(damaged) / float64(totalDelivered)
		}
		if shopClosedTotal > 0 {
			shopClosedRate = float64(shopClosedBad) / float64(shopClosedTotal)
		}
		if avgRating > 0 {
			feedbackScore = (avgRating - 1.0) / 4.0 // normalize 1-5 to 0-1
		}

		stopsPerHour := 0.0
		if totalHours > 0 {
			stopsPerHour = float64(totalCompleted) / totalHours
		}

		// Calculate Score
		score := 70.0 // Edge Rule: New driver (< 15 stops) -> 70 neutral
		if totalCompleted >= 15 {
			score = 100.0 * (
				0.35*onTimeRate +
				0.25*completionRate +
				0.20*(1.0-damageRate) +
				0.10*(1.0-shopClosedRate) +
				0.10*feedbackScore)
		}

		// Clamp to [0, 100]
		score = math.Max(0.0, math.Min(100.0, score))

		mutations = append(mutations, spanner.InsertOrUpdateMap("DriverScores", map[string]any{
			"DriverId":       driverId,
			"Score":          score,
			"OnTimeRate":     onTimeRate,
			"CompletionRate": completionRate,
			"DamageRate":     damageRate,
			"ShopClosedRate": shopClosedRate,
			"FeedbackScore":  feedbackScore,
			"StopsPerHour":   stopsPerHour,
			"WindowStart":    civil.DateOf(windowStart),
			"WindowEnd":      civil.DateOf(windowEnd),
			"ComputedAt":     now,
		}))

		outboxEvents = append(outboxEvents, driverScoreOutbox{
			DriverId:   driverId,
			Score:      score,
			ComputedAt: now,
		})

		if len(mutations) >= 500 {
			if err := s.flushScoreMutations(ctx, mutations, outboxEvents); err != nil {
				return err
			}
			mutations = nil
			outboxEvents = nil
		}
	}

	if len(mutations) > 0 {
		if err := s.flushScoreMutations(ctx, mutations, outboxEvents); err != nil {
			return err
		}
	}

	slog.Info("driver score computation complete")
	return nil
}

func (s *Service) flushScoreMutations(ctx context.Context, mutations []*spanner.Mutation, events []driverScoreOutbox) error {
	_, err := s.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &txBuf{mutations: &mutations}
		for _, ev := range events {
			_ = outbox.EmitJSON(ctx, buf, "DriverScore", ev.DriverId, "driver.score.updated", ev)
		}
		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}
		return nil
	})
	return err
}

