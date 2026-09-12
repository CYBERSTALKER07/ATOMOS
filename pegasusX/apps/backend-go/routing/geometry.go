package routing

import (
	"fmt"
	"math"
)

// LatLng is a WGS84 coordinate on the wire.
type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// RouteGeometry is the driver map overlay contract.
type RouteGeometry struct {
	RouteID         string      `json:"route_id"`
	EncodedPolyline string      `json:"encoded_polyline"`
	Coordinates     []LatLng    `json:"coordinates"`
	Source          string      `json:"source"`
	StopCount       int         `json:"stop_count"`
	Steps           []RouteStep `json:"steps,omitempty"`
}

const defaultDensifyStepMeters = 25.0

// BuildDenseRouteGeometry connects waypoints with haversine-densified segments.
// When fewer than two waypoints exist, returns an empty geometry.
func BuildDenseRouteGeometry(routeID string, waypoints []LatLng) RouteGeometry {
	if len(waypoints) < 2 {
		return RouteGeometry{
			RouteID:     routeID,
			Coordinates: []LatLng{},
			Source:      "insufficient_waypoints",
			StopCount:   len(waypoints),
		}
	}

	dense := DensifyPath(waypoints, defaultDensifyStepMeters)
	return RouteGeometry{
		RouteID:         routeID,
		EncodedPolyline: EncodePolyline(dense),
		Coordinates:     dense,
		Source:          "computed_dense",
		StopCount:       len(waypoints),
	}
}

// GeometryFromStoredPolyline hydrates the driver map contract from a persisted polyline.
func GeometryFromStoredPolyline(routeID, encoded, source string, stopCount int) (RouteGeometry, error) {
	coords, err := DecodePolyline(encoded)
	if err != nil {
		return RouteGeometry{}, fmt.Errorf("decode stored polyline: %w", err)
	}
	if len(coords) < 2 {
		return RouteGeometry{}, fmt.Errorf("stored polyline has insufficient points")
	}
	if source == "" {
		source = "manifest_sealed"
	}
	return RouteGeometry{
		RouteID:         routeID,
		EncodedPolyline: encoded,
		Coordinates:     coords,
		Source:          source,
		StopCount:       stopCount,
	}, nil
}

// DensifyPath inserts intermediate points along each segment.
func DensifyPath(waypoints []LatLng, stepMeters float64) []LatLng {
	if len(waypoints) == 0 {
		return nil
	}
	if stepMeters <= 0 {
		stepMeters = defaultDensifyStepMeters
	}

	out := make([]LatLng, 0, len(waypoints)*4)
	out = append(out, waypoints[0])
	for i := 1; i < len(waypoints); i++ {
		start := waypoints[i-1]
		end := waypoints[i]
		segment := densifySegment(start, end, stepMeters)
		if len(segment) > 1 {
			out = append(out, segment[1:]...)
		}
	}
	return out
}

func densifySegment(start, end LatLng, stepMeters float64) []LatLng {
	distance := haversineMeters(start.Lat, start.Lng, end.Lat, end.Lng)
	if distance <= stepMeters {
		return []LatLng{start, end}
	}
	steps := int(math.Ceil(distance / stepMeters))
	out := make([]LatLng, 0, steps+1)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		out = append(out, LatLng{
			Lat: start.Lat + (end.Lat-start.Lat)*t,
			Lng: start.Lng + (end.Lng-start.Lng)*t,
		})
	}
	return out
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6_371_000.0
	dLat := degreesToRadians(lat2 - lat1)
	dLon := degreesToRadians(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadius * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// HaversineDistanceMatrixM builds an NxN meter matrix from haversine distances.
func HaversineDistanceMatrixM(points []LatLng) [][]int {
	n := len(points)
	out := make([][]int, n)
	for i := 0; i < n; i++ {
		out[i] = make([]int, n)
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			m := int(haversineMeters(points[i].Lat, points[i].Lng, points[j].Lat, points[j].Lng))
			if m < 1 {
				m = 1
			}
			out[i][j] = m
		}
	}
	return out
}

// MergeDistanceMatrix fills zero off-diagonal cells in primary from fallback.
func MergeDistanceMatrix(primary, fallback [][]int) [][]int {
	if len(primary) == 0 {
		return fallback
	}
	if len(fallback) != len(primary) {
		return primary
	}
	out := make([][]int, len(primary))
	for i := range primary {
		out[i] = make([]int, len(primary[i]))
		copy(out[i], primary[i])
		if len(fallback[i]) != len(primary[i]) {
			continue
		}
		for j := range primary[i] {
			if i != j && out[i][j] <= 0 {
				out[i][j] = fallback[i][j]
			}
		}
	}
	return out
}

func degreesToRadians(v float64) float64 {
	return v * math.Pi / 180
}
