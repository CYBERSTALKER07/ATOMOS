package gamification

import (
	"context"
	"log/slog"
)

type DriverScoreEvent struct {
	DriverID     string
	PointsEarned int
	Reason       string // e.g., "ON_TIME_DELIVERY", "ZERO_DAMAGE"
}

// ProcessCompletedOrder evaluates an order completion and awards gamification points
// based on on-time delivery rates and damage reports.
func ProcessCompletedOrder(ctx context.Context, driverID string, onTime bool, damageReported bool) DriverScoreEvent {
	points := 0
	reason := "COMPLETED_DELIVERY"

	// Base points for delivery
	points += 10

	if onTime {
		points += 5
		reason = "ON_TIME_DELIVERY"
	}

	if !damageReported {
		points += 15
		if reason == "ON_TIME_DELIVERY" {
			reason = "PERFECT_HANDOVER"
		} else {
			reason = "ZERO_DAMAGE"
		}
	} else {
		// Penalty for damage
		points -= 10
	}

	slog.InfoContext(ctx, "driver awarded gamification points", 
		"driver_id", driverID, 
		"points", points, 
		"reason", reason,
	)

	return DriverScoreEvent{
		DriverID:     driverID,
		PointsEarned: points,
		Reason:       reason,
	}
}
