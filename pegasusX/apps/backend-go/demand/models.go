package demand

import (
	"encoding/json"
	"strings"
	"time"
)

type SignalType string

const (
	SignalHoliday            SignalType = "HOLIDAY"
	SignalWeather            SignalType = "WEATHER"
	SignalEvent              SignalType = "EVENT"
	SignalPromo              SignalType = "PROMO"
	SignalPayday             SignalType = "PAYDAY"
	SignalEventDensity       SignalType = "EVENT_DENSITY"
	SignalCompetitorPressure SignalType = "COMPETITOR_PRESSURE"
)

// PlatformSupplierID stamps global/weather signals that are not tenant-owned.
const PlatformSupplierID = "_platform"

// ResolveSupplierID returns explicit id, or PlatformSupplierID when empty.
func ResolveSupplierID(explicit string) string {
	if sid := strings.TrimSpace(explicit); sid != "" {
		return sid
	}
	return PlatformSupplierID
}

type DemandSignal struct {
	SignalId   string          `json:"signalId"`
	Type       SignalType      `json:"type"`
	Scope      string          `json:"scope"`
	Sku        *string         `json:"sku,omitempty"`
	SupplierId string          `json:"supplierId,omitempty"`
	StartAt    time.Time       `json:"startAt"`
	EndAt      time.Time       `json:"endAt"`
	Multiplier float64         `json:"multiplier"`
	Meta       json.RawMessage `json:"meta,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
	CreatedBy  string          `json:"createdBy"`
}

type DemandAdjustment struct {
	RetailerId     string             `json:"retailerId"`
	Sku            string             `json:"sku"`
	SupplierId     string             `json:"supplierId,omitempty"`
	Date           time.Time          `json:"date"`
	BaseVelocity   float64            `json:"baseVelocity"`
	Adjustment     float64            `json:"adjustment"`
	AdjustedDemand float64            `json:"adjustedDemand"`
	Factors        map[string]float64 `json:"factors"`
	ComputedAt     time.Time          `json:"computedAt"`
}
