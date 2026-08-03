package laborcapacity

import (
	"time"

	"cloud.google.com/go/civil"
)

// DriverScore holds the computed reliability score for a driver.
type DriverScore struct {
	DriverId       string     `json:"driverId"`
	Score          float64    `json:"score"`
	OnTimeRate     float64    `json:"onTimeRate"`
	CompletionRate float64    `json:"completionRate"`
	DamageRate     float64    `json:"damageRate"`
	ShopClosedRate float64    `json:"shopClosedRate"`
	FeedbackScore  float64    `json:"feedbackScore"`
	StopsPerHour   float64    `json:"stopsPerHour"`
	WindowStart    civil.Date `json:"windowStart"`
	WindowEnd      civil.Date `json:"windowEnd"`
	ComputedAt     time.Time  `json:"computedAt"`
}

// DriverAvailability represents a driver's shift for a given day.
type DriverAvailability struct {
	DriverId       string     `json:"driverId"`
	Date           civil.Date `json:"date"`
	AvailableHours float64    `json:"availableHours"`
	ZoneH3         string     `json:"zoneH3,omitempty"`
	Status         string     `json:"status"` // AVAILABLE | OFF | LIMITED
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// ZoneCapacity is an aggregated capacity snapshot for a zone on a given day.
type ZoneCapacity struct {
	ZoneH3        string    `json:"zoneH3"`
	Date          civil.Date `json:"date"`
	TotalCapacity float64   `json:"totalCapacity"`
	UsedCapacity  float64   `json:"usedCapacity"`
	ComputedAt    time.Time `json:"computedAt"`
}

// SetAvailabilityRequest is the input for setting a driver's availability.
type SetAvailabilityRequest struct {
	DriverId       string  `json:"driverId"`
	Date           string  `json:"date"` // "2006-01-02"
	AvailableHours float64 `json:"availableHours"`
	ZoneH3         string  `json:"zoneH3,omitempty"`
	Status         string  `json:"status"`
}

// computeScore is the pure driver score formula.
// All rates are 0.0–1.0. Returns 0–100.
func computeScore(onTimeRate, completionRate, damageRate, shopClosedRate, feedbackScore float64) float64 {
	raw := 0.35*onTimeRate +
		0.25*completionRate +
		0.20*(1.0-damageRate) +
		0.10*(1.0-shopClosedRate) +
		0.10*feedbackScore
	score := raw * 100.0
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}
