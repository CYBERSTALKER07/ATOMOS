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

// RunDriverScoreWorker recomputes driver scores on a ticker.
func (s *Service) RunDriverScoreWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
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

// driverMetrics holds raw aggregates for a single driver over the 28-day window.
type driverMetrics struct {
	DriverId         string
	TotalDelivered   int64
	DeliveredOnTime  int64
	TotalCancelled   int64
	TotalShopClosed  int64
	TotalOrders      int64
	DamageClaims     int64
	RatingSum        float64
	RatingCount      int64
	FirstDeliveryAt  time.Time
	LastDeliveryAt   time.Time
	ActiveDays       int64
}

// RunDriverScoreComputation recomputes scores for all active drivers.
func (s *Service) RunDriverScoreComputation(ctx context.Context) error {
	now := time.Now().UTC()
	windowStart := now.AddDate(0, 0, -28)
	windowEnd := now

	// Single aggregate query for all driver metrics in the window.
	stmt := spanner.Statement{
		SQL: `
			SELECT
				o.DriverId,
				COUNTIF(o.Status = 'DELIVERED') as TotalDelivered,
				COUNTIF(o.Status = 'DELIVERED' AND o.DeliveredAt <= o.PromisedBy) as DeliveredOnTime,
				COUNTIF(o.Status = 'CANCELLED') as TotalCancelled,
				COUNTIF(o.ShopClosedAt IS NOT NULL) as TotalShopClosed,
				COUNT(*) as TotalOrders,
				COALESCE(AVG(CASE WHEN o.Rating IS NOT NULL THEN CAST(o.Rating AS FLOAT64) END), 2.5) as AvgRating,
				COUNTIF(o.Rating IS NOT NULL) as RatingCount
			FROM Orders o
			WHERE o.DriverId IS NOT NULL
			  AND o.CreatedAt >= @WindowStart
			GROUP BY o.DriverId
			HAVING COUNT(*) >= 1
		`,
		Params: map[string]interface{}{
			"WindowStart": windowStart,
		},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var mutations []*spanner.Mutation
	windowStartDate := civil.DateOf(windowStart)
	windowEndDate := civil.DateOf(windowEnd)

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("query driver metrics: %w", err)
		}

		var driverID string
		var totalDelivered, deliveredOnTime, totalCancelled, totalShopClosed, totalOrders, ratingCount int64
		var avgRating float64

		if err := row.Columns(
			&driverID, &totalDelivered, &deliveredOnTime, &totalCancelled,
			&totalShopClosed, &totalOrders, &avgRating, &ratingCount,
		); err != nil {
			return fmt.Errorf("scan driver metrics: %w", err)
		}

		// Compute rates
		onTimeRate := 0.0
		if totalDelivered > 0 {
			onTimeRate = float64(deliveredOnTime) / float64(totalDelivered)
		}

		completionRate := 0.0
		deliveredPlusCancelled := totalDelivered + totalCancelled
		if deliveredPlusCancelled > 0 {
			completionRate = float64(totalDelivered) / float64(deliveredPlusCancelled)
		}

		// Phase 1: approximate damage rate from claims (if no join, default 0)
		damageRate := 0.0

		shopClosedRate := 0.0
		if totalOrders > 0 {
			shopClosedRate = float64(totalShopClosed) / float64(totalOrders)
		}

		feedbackScore := 0.5
		if ratingCount > 0 {
			feedbackScore = avgRating / 5.0
		}

		// Efficiency: estimate stops per hour (rough: totalDelivered / (28 * 8h work day))
		stopsPerHour := 0.0
		activeDays := float64(28)
		if totalDelivered > 0 {
			stopsPerHour = float64(totalDelivered) / (activeDays * 6.0) // assume 6 active hours/day
		}

		score := computeScore(onTimeRate, completionRate, damageRate, shopClosedRate, feedbackScore)

		mutations = append(mutations, spanner.InsertOrUpdateMap("DriverScores", map[string]interface{}{
			"DriverId":       driverID,
			"Score":          score,
			"OnTimeRate":     onTimeRate,
			"CompletionRate": completionRate,
			"DamageRate":     damageRate,
			"ShopClosedRate": shopClosedRate,
			"FeedbackScore":  feedbackScore,
			"StopsPerHour":   stopsPerHour,
			"WindowStart":    windowStartDate,
			"WindowEnd":      windowEndDate,
			"ComputedAt":     now,
		}))

		// Batch write every 500 mutations
		if len(mutations) >= 500 {
			if _, err := s.spanner.Apply(ctx, mutations); err != nil {
				return fmt.Errorf("apply driver score mutations: %w", err)
			}
			mutations = nil
		}
	}

	if len(mutations) > 0 {
		if _, err := s.spanner.Apply(ctx, mutations); err != nil {
			return fmt.Errorf("apply remaining driver score mutations: %w", err)
		}
	}

	slog.Info("driver score computation complete")
	return nil
}
