package eta

import (
	"math"
	"time"
)

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

// CalculateETAs is a pure function that calculates predicted arrival windows for a sequence of stops.
func CalculateETAs(now time.Time, driverLat, driverLng float64, profile DriverProfile, stops []StopInput, shopClosedRates map[string]float64) []RouteETA {
	var etas []RouteETA

	speed := profile.HistoricalSpeedKmH
	if speed <= 0 {
		speed = 25.0 // Default 25 km/h
	}
	
	stopDuration := profile.AvgStopDuration
	if stopDuration <= 0 {
		stopDuration = 8.0 // Default 8 minutes per stop
	}

	confidence := float64(profile.RecentStopCount) / 15.0
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0 {
		confidence = 0
	}

	currentTime := now
	currentLat := driverLat
	currentLng := driverLng

	for _, stop := range stops {
		if stop.IsCompleted {
			continue
		}

		distKm := haversineKm(currentLat, currentLng, stop.Lat, stop.Lng)
		travelMinutes := (distKm / speed) * 60.0
		
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

		windowStart := predictedArrival.Add(-time.Duration(bufferMinutes) * time.Minute)
		windowEnd := predictedArrival.Add(time.Duration(bufferMinutes) * time.Minute)

		factors := map[string]float64{
			"travel_minutes":                   travelMinutes,
			"remaining_stops_duration_minutes": stopDuration,
			"congestion_factor":                congestionFactor,
			"shop_closed_buffer_minutes":       shopClosedBuffer,
			"historical_speed_km_h":            speed,
			"avg_stop_duration_minutes":        stopDuration,
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
