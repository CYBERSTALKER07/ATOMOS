package replenishment

import (
	"cloud.google.com/go/civil"
	"time"
)

type ReorderSuggestion struct {
	RetailerId      string
	Sku             string
	SuggestedQty    int64
	AdjustedDemand  float64
	CurrentStock    int64
	InFlightQty     int64
	SafetyStock     float64
	SuggestedByDate civil.Date
	ComputedAt      time.Time
	Status          string
}
