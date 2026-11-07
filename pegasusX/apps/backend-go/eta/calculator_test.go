package eta

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
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

func TestFetchOSRMTable_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/table/v1/driving/69.240000,41.300000;69.250000,41.310000" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("annotations") != "distance,duration" {
			t.Errorf("unexpected annotations: %s", r.URL.Query().Get("annotations"))
		}
		resp := OSRMTableResponse{
			Code: "Ok",
			Durations: [][]float64{
				{0, 720},
				{720, 0},
			},
			Distances: [][]float64{
				{0, 6000},
				{6000, 0},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	coords := [][2]float64{
		{41.3000, 69.2400},
		{41.3100, 69.2500},
	}

	tbl, err := FetchOSRMTable(context.Background(), mockServer.URL, coords)
	if err != nil {
		t.Fatalf("fetch error: %v", err)
	}
	if tbl.Code != "Ok" {
		t.Fatalf("expected code Ok, got %s", tbl.Code)
	}
	if tbl.Durations[0][1] != 720 {
		t.Fatalf("expected duration 720, got %v", tbl.Durations[0][1])
	}
	if tbl.Distances[0][1] != 6000 {
		t.Fatalf("expected distance 6000, got %v", tbl.Distances[0][1])
	}
}

func TestFetchOSRMTable_Error(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	coords := [][2]float64{
		{41.3000, 69.2400},
		{41.3100, 69.2500},
	}

	_, err := FetchOSRMTable(context.Background(), mockServer.URL, coords)
	if err == nil {
		t.Fatal("expected error on 500 status")
	}

	_, err = FetchOSRMTable(context.Background(), "", coords)
	if err == nil {
		t.Fatal("expected error on empty url")
	}

	_, err = FetchOSRMTable(context.Background(), mockServer.URL, [][2]float64{{41.3, 69.2}})
	if err == nil {
		t.Fatal("expected error on insufficient coords")
	}
}

func TestCalculateETAs_WithOSRMTable(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 3 waypoints: driver, stop 1, stop 2
		resp := OSRMTableResponse{
			Code: "Ok",
			Durations: [][]float64{
				{0, 600, 1200},
				{600, 0, 900},
				{1200, 900, 0},
			},
			Distances: [][]float64{
				{0, 5000, 10000},
				{5000, 0, 7500},
				{10000, 7500, 0},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	now := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	profile := DriverProfile{
		DriverId:           "d1",
		HistoricalSpeedKmH: 30.0,
		AvgStopDuration:    10.0,
		RecentStopCount:    20,
	}

	stops := []StopInput{
		{StopId: "s1", Sequence: 1, Lat: 41.31, Lng: 69.25, IsCompleted: false},
		{StopId: "s2", Sequence: 2, Lat: 41.35, Lng: 69.29, IsCompleted: false},
	}

	etas := CalculateETAsWithContext(context.Background(), now, 41.30, 69.24, profile, stops, nil, mockServer.URL)
	if len(etas) != 2 {
		t.Fatalf("expected 2 etas, got %d", len(etas))
	}

	// Stop 1: OSRM duration index 0->1 is 600s = 10 mins. Distance = 5 km.
	if etas[0].Factors["travel_minutes"] != 10.0 {
		t.Fatalf("expected 10.0 travel minutes for stop 1, got %v", etas[0].Factors["travel_minutes"])
	}
	if etas[0].Factors["distance_km"] != 5.0 {
		t.Fatalf("expected 5.0 km for stop 1, got %v", etas[0].Factors["distance_km"])
	}

	// Stop 2: OSRM duration index 1->2 is 900s = 15 mins. Distance = 7.5 km.
	if etas[1].Factors["travel_minutes"] != 15.0 {
		t.Fatalf("expected 15.0 travel minutes for stop 2, got %v", etas[1].Factors["travel_minutes"])
	}
	if etas[1].Factors["distance_km"] != 7.5 {
		t.Fatalf("expected 7.5 km for stop 2, got %v", etas[1].Factors["distance_km"])
	}
}

func TestCalculateETAs_OSRM_Fallback(t *testing.T) {
	now := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	profile := DriverProfile{
		DriverId:           "d1",
		HistoricalSpeedKmH: 30.0,
		AvgStopDuration:    10.0,
		RecentStopCount:    20,
	}

	stops := []StopInput{
		{StopId: "s1", Sequence: 1, Lat: 41.31, Lng: 69.25, IsCompleted: false},
	}

	// Invalid OSRM URL should gracefully fall back to Haversine
	etas := CalculateETAsWithContext(context.Background(), now, 41.30, 69.24, profile, stops, nil, "http://127.0.0.1:54321")
	if len(etas) != 1 {
		t.Fatalf("expected 1 eta on fallback, got %d", len(etas))
	}
	if etas[0].Factors["travel_minutes"] <= 0 {
		t.Fatalf("expected positive travel minutes on fallback, got %v", etas[0].Factors["travel_minutes"])
	}
}
