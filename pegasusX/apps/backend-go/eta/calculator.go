package eta

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultOSRMTimeout = 2 * time.Second

// OSRMTableResponse models the response from OSRM /table/v1/driving/
type OSRMTableResponse struct {
	Code      string      `json:"code"`
	Durations [][]float64 `json:"durations"` // seconds
	Distances [][]float64 `json:"distances"` // meters
}

// FetchOSRMTable calls OSRM table service for driving durations and distances.
// Expected coordinates format in OSRM URL: {lng1},{lat1};{lng2},{lat2};...
func FetchOSRMTable(ctx context.Context, osrmURL string, coords [][2]float64) (*OSRMTableResponse, error) {
	osrmURL = strings.TrimRight(strings.TrimSpace(osrmURL), "/")
	if osrmURL == "" || len(coords) < 2 {
		return nil, fmt.Errorf("osrm table: invalid url or insufficient coords")
	}

	var sb strings.Builder
	for i, c := range coords {
		if i > 0 {
			sb.WriteByte(';')
		}
		sb.WriteString(strconv.FormatFloat(c[1], 'f', 6, 64)) // lng
		sb.WriteByte(',')
		sb.WriteString(strconv.FormatFloat(c[0], 'f', 6, 64)) // lat
	}

	url := osrmURL + "/table/v1/driving/" + sb.String() + "?annotations=distance,duration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: defaultOSRMTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("osrm table status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var table OSRMTableResponse
	if err := json.Unmarshal(body, &table); err != nil {
		return nil, err
	}
	if !strings.EqualFold(table.Code, "Ok") || len(table.Durations) == 0 {
		return nil, fmt.Errorf("osrm table code %q", table.Code)
	}
	return &table, nil
}

// haversineKm computes the distance between two points on Earth in kilometers.
func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0 // Earth radius in kilometers
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLng := (lng2 - lng1) * math.Pi / 180.0
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// CalculateETAs is a pure function that calculates predicted arrival windows for a sequence of stops,
// utilizing live OSRM /table/v1/driving/ matrices when available, falling back to Haversine.
func CalculateETAs(now time.Time, driverLat, driverLng float64, profile DriverProfile, stops []StopInput, shopClosedRates map[string]float64) []RouteETA {
	return CalculateETAsWithContext(context.Background(), now, driverLat, driverLng, profile, stops, shopClosedRates, os.Getenv("ROUTING_OSRM_URL"))
}

// CalculateETAsWithContext executes ETA calculation with explicit context and OSRM endpoint.
func CalculateETAsWithContext(ctx context.Context, now time.Time, driverLat, driverLng float64, profile DriverProfile, stops []StopInput, shopClosedRates map[string]float64, osrmURL string) []RouteETA {
	var etas []RouteETA

	speed := profile.HistoricalSpeedKmH
	if speed <= 0 {
		speed = 25.0 // Default 25 km/h
	}
	
	stopDuration := profile.AvgStopDuration
	if stopDuration <= 0 {
		stopDuration = 8.0 // Default 8 minutes per stop
	}

	confidence := math.Min(1.0, math.Max(0.0, float64(profile.RecentStopCount)/15.0))
	
	// Rule: If historical data is thin (< 10 samples) -> widen the window and lower confidence.
	isThinData := profile.RecentStopCount < 10
	if isThinData && confidence > 0.5 {
		confidence = 0.5
	}

	// Filter active stops
	var activeStops []StopInput
	for _, s := range stops {
		if !s.IsCompleted {
			activeStops = append(activeStops, s)
		}
	}
	if len(activeStops) == 0 {
		return etas
	}

	// Build waypoints for OSRM: [driver, stop_1, stop_2, ..., stop_k]
	coords := make([][2]float64, 0, len(activeStops)+1)
	coords = append(coords, [2]float64{driverLat, driverLng})
	for _, s := range activeStops {
		coords = append(coords, [2]float64{s.Lat, s.Lng})
	}

	var osrmTable *OSRMTableResponse
	if osrmURL != "" {
		if tbl, err := FetchOSRMTable(ctx, osrmURL, coords); err == nil {
			osrmTable = tbl
		}
	}

	currentTime := now
	currentLat := driverLat
	currentLng := driverLng

	for idx, stop := range activeStops {
		var travelMinutes float64
		var distKm float64

		// Use OSRM durations and distances if table is valid
		if osrmTable != nil && len(osrmTable.Durations) > idx+1 && len(osrmTable.Durations[idx]) > idx+1 {
			durationSec := osrmTable.Durations[idx][idx+1]
			travelMinutes = durationSec / 60.0
			if len(osrmTable.Distances) > idx+1 && len(osrmTable.Distances[idx]) > idx+1 {
				distKm = osrmTable.Distances[idx][idx+1] / 1000.0
			}
		} else {
			distKm = haversineKm(currentLat, currentLng, stop.Lat, stop.Lng)
			travelMinutes = (distKm / speed) * 60.0
		}
		
		congestionFactor := 1.0 // Phase 1 placeholder

		shopClosedBuffer := 0.0
		rate := shopClosedRates[stop.RetailerId]
		if rate > 0.2 {
			shopClosedBuffer = 10.0
		} else if rate > 0.1 {
			shopClosedBuffer = 5.0
		}

		predictedArrival := currentTime.Add(time.Duration(travelMinutes*congestionFactor+shopClosedBuffer) * time.Minute)
		
		bufferMinutes := travelMinutes * 0.25
		if bufferMinutes < 5.0 {
			bufferMinutes = 5.0
		}
		
		if isThinData {
			bufferMinutes *= 1.5 // Widen the window for thin historical data
		}

		windowStart := predictedArrival.Add(-time.Duration(bufferMinutes) * time.Minute)
		windowEnd := predictedArrival.Add(time.Duration(bufferMinutes) * time.Minute)

		thinDataVal := 0.0
		if isThinData {
			thinDataVal = 1.0
		}
		
		factors := map[string]float64{
			"travel_minutes":                   travelMinutes,
			"remaining_stops_duration_minutes": stopDuration,
			"congestion_factor":                congestionFactor,
			"shop_closed_buffer_minutes":       shopClosedBuffer,
			"historical_speed_km_h":            speed,
			"avg_stop_duration_minutes":        stopDuration,
			"is_thin_data":                     thinDataVal,
			"distance_km":                      distKm,
		}

		eta := RouteETA{
			StopId:           stop.StopId,
			Sequence:         stop.Sequence,
			PredictedArrival: predictedArrival,
			WindowStart:      windowStart,
			WindowEnd:        windowEnd,
			Confidence:       confidence,
			ComputedAt:       now,
			Factors:          factors,
		}
		etas = append(etas, eta)

		currentTime = predictedArrival.Add(time.Duration(stopDuration) * time.Minute)
		currentLat = stop.Lat
		currentLng = stop.Lng
	}

	return etas
}

