package eta

import (
	"math"
	"testing"
	"time"
)

func TestHaversine(t *testing.T) {
	// Tashkent to Samarkand: ~268 km
	// Tashkent: 41.2995 N, 69.2401 E
	// Samarkand: 39.6270 N, 66.9750 E
	dist := haversineKm(41.2995, 69.2401, 39.6270, 66.9750)
	if math.Abs(dist-268.0) > 10.0 { // Allow some error due to exact coordinates
		t.Errorf("expected ~268km, got %f", dist)
	}
}

func TestCalculateETAs_ThreeStops(t *testing.T) {
	now := time.Now()
	profile := DriverProfile{
		DriverId:           "d1",
		HistoricalSpeedKmH: 30.0,
		AvgStopDuration:    10.0,
		RecentStopCount:    20, // max confidence
	}

	stops := []StopInput{
		{StopId: "s1", Sequence: 1, Lat: 41.3000, Lng: 69.2400, IsCompleted: false}, // ~0 km initially (assuming driver starts here)
		{StopId: "s2", Sequence: 2, Lat: 41.3500, Lng: 69.2400, IsCompleted: false}, // ~5.5 km north
		{StopId: "s3", Sequence: 3, Lat: 41.3500, Lng: 69.3000, IsCompleted: false}, // ~5 km east
	}

	etas := CalculateETAs(now, 41.3000, 69.2400, profile, stops, nil)

	if len(etas) != 3 {
		t.Fatalf("expected 3 etas, got %d", len(etas))
	}

	if etas[0].Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", etas[0].Confidence)
	}

	// First stop is right here. Distance 0.
	if etas[0].Factors["travel_minutes"] != 0 {
		t.Errorf("expected 0 travel time for first stop, got %f", etas[0].Factors["travel_minutes"])
	}

	// Second stop is ~5.5km away. Travel time ~11 minutes. + 10 min stop duration for first stop.
	// So predicted arrival should be ~21 mins from now.
	diff := etas[1].PredictedArrival.Sub(now).Minutes()
	if diff < 20 || diff > 22 {
		t.Errorf("expected ~21 mins to stop 2, got %f", diff)
	}
}

func TestCalculateETAs_HighShopClosedRate(t *testing.T) {
	now := time.Now()
	profile := DriverProfile{HistoricalSpeedKmH: 25, AvgStopDuration: 5, RecentStopCount: 15}
	
	stops := []StopInput{
		{StopId: "s1", RetailerId: "r1", Sequence: 1, Lat: 41.3, Lng: 69.2, IsCompleted: false},
	}
	rates := map[string]float64{
		"r1": 0.25, // > 0.2, so 10 min buffer
	}

	etas := CalculateETAs(now, 41.3, 69.2, profile, stops, rates)
	if etas[0].Factors["shop_closed_buffer_minutes"] != 10.0 {
		t.Errorf("expected 10 min buffer, got %f", etas[0].Factors["shop_closed_buffer_minutes"])
	}
}

func TestCalculateETAs_LowConfidence(t *testing.T) {
	etas := CalculateETAs(time.Now(), 0, 0, DriverProfile{RecentStopCount: 5}, []StopInput{{StopId: "s1"}}, nil)
	expected := 5.0 / 15.0
	if math.Abs(etas[0].Confidence-expected) > 0.01 {
		t.Errorf("expected confidence ~%f, got %f", expected, etas[0].Confidence)
	}
}

func TestCalculateETAs_DefaultSpeed(t *testing.T) {
	etas := CalculateETAs(time.Now(), 0, 0, DriverProfile{HistoricalSpeedKmH: 0}, []StopInput{{StopId: "s1"}}, nil)
	if etas[0].Factors["historical_speed_km_h"] != 25.0 {
		t.Errorf("expected default speed 25, got %f", etas[0].Factors["historical_speed_km_h"])
	}
}

func TestCalculateETAs_AllCompleted(t *testing.T) {
	stops := []StopInput{
		{StopId: "s1", IsCompleted: true},
	}
	etas := CalculateETAs(time.Now(), 0, 0, DriverProfile{}, stops, nil)
	if len(etas) != 0 {
		t.Errorf("expected 0 etas for completed stops, got %d", len(etas))
	}
}
