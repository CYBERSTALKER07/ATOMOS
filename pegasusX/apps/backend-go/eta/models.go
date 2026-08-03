package eta

import (
	"time"
)

type RouteETA struct {
	RouteId          string             `json:"routeId"`
	StopId           string             `json:"stopId"`
	Sequence         int64              `json:"sequence"`
	PredictedArrival time.Time          `json:"predictedArrival"`
	WindowStart      time.Time          `json:"windowStart"`
	WindowEnd        time.Time          `json:"windowEnd"`
	Confidence       float64            `json:"confidence"`
	ComputedAt       time.Time          `json:"computedAt"`
	Factors          map[string]float64 `json:"factors,omitempty"`
}

type ETAFactors struct {
	TravelMinutes          float64 `json:"travel_minutes"`
	RemainingStopsDuration float64 `json:"remaining_stops_duration_minutes"`
	CongestionFactor       float64 `json:"congestion_factor"`
	ShopClosedBuffer       float64 `json:"shop_closed_buffer_minutes"`
	HistoricalSpeedKmH     float64 `json:"historical_speed_km_h"`
	AvgStopDuration        float64 `json:"avg_stop_duration_minutes"`
}

type StopInput struct {
	StopId      string
	OrderId     string
	RetailerId  string
	Lat         float64
	Lng         float64
	Sequence    int64
	IsCompleted bool
}

type DriverProfile struct {
	DriverId           string
	HistoricalSpeedKmH float64 // avg speed from recent routes
	AvgStopDuration    float64 // minutes per stop
	RecentStopCount    int64   // for confidence calc
}
