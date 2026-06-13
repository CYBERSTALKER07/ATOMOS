package routing

import "math"

const defaultPassedWaypointMeters = 50.0

// DistanceToPolylineMeters returns the shortest distance from a point to a polyline path.
func DistanceToPolylineMeters(lat, lng float64, path []LatLng) float64 {
	if len(path) == 0 {
		return math.MaxFloat64
	}
	if len(path) == 1 {
		return haversineMeters(lat, lng, path[0].Lat, path[0].Lng)
	}
	minDist := math.MaxFloat64
	for i := 0; i < len(path)-1; i++ {
		d := distancePointToSegmentMeters(lat, lng, path[i], path[i+1])
		if d < minDist {
			minDist = d
		}
	}
	return minDist
}

// WaypointsAhead rebuilds stop waypoints from the driver's current position.
func WaypointsAhead(from LatLng, waypoints []LatLng, passedMeters float64) []LatLng {
	if passedMeters <= 0 {
		passedMeters = defaultPassedWaypointMeters
	}
	if len(waypoints) == 0 {
		return []LatLng{from}
	}
	startIdx := 0
	for i, wp := range waypoints {
		if haversineMeters(from.Lat, from.Lng, wp.Lat, wp.Lng) <= passedMeters {
			startIdx = i + 1
		}
	}
	if startIdx >= len(waypoints) {
		last := waypoints[len(waypoints)-1]
		return []LatLng{from, last}
	}
	out := make([]LatLng, 0, len(waypoints)-startIdx+1)
	out = append(out, from)
	out = append(out, waypoints[startIdx:]...)
	return out
}

func distancePointToSegmentMeters(lat, lng float64, start, end LatLng) float64 {
	dx := end.Lng - start.Lng
	dy := end.Lat - start.Lat
	if dx == 0 && dy == 0 {
		return haversineMeters(lat, lng, start.Lat, start.Lng)
	}
	t := ((lng-start.Lng)*dx + (lat-start.Lat)*dy) / (dx*dx + dy*dy)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	closestLat := start.Lat + t*dy
	closestLng := start.Lng + t*dx
	return haversineMeters(lat, lng, closestLat, closestLng)
}
