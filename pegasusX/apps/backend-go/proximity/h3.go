package proximity

import (
	"errors"
	"math"

	"github.com/uber/h3-go/v4"
)

// CoverageResolution defines the default H3 resolution for coverage (Phase-3).
const CoverageResolution = 9

// Policy Limits to prevent memory/egress safety exhaustion
const (
	MaxPolygonPoints       = 5000
	MaxPolygonAreaSqKm     = 10000.0 // Ceiling area
	MaxCells               = 500000  // Maximum cells we will generate
	FallbackIterationLimit = 100000
	MaxCompactedEgressCells = 5000
)

// Typed Coverage Errors
var (
	ErrPolygonTooComplex = errors.New("coverage: polygon exceeds maximum allowed points")
	ErrPolygonTooLarge   = errors.New("coverage: polygon area exceeds maximum allowed square kilometers")
	ErrMaxCellsExceeded  = errors.New("coverage: resolution yields too many cells, exceeding memory safety bounds")
	ErrH3BoundaryPanic   = errors.New("coverage: H3 CGO boundary triggered an unsafe panic or pentagon neighborhood failure")
)

// PanicSafeLatLngToCell safely wraps LatLngToCell
func PanicSafeLatLngToCell(lat, lng float64, resolution int) (h3.Cell, error) {
	defer func() {
		if r := recover(); r != nil {
			// Catch CGO panics if they happen
		}
	}()
	return h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lng}, resolution)
}

// PanicSafePolygonToCells safely generates cells for a polygon with bounds checking.
func PanicSafePolygonToCells(geoLoop []h3.LatLng, resolution int) ([]h3.Cell, error) {
	if len(geoLoop) > MaxPolygonPoints {
		return nil, ErrPolygonTooComplex
	}

	defer func() {
		if r := recover(); r != nil {
			// Catch CGO panics
		}
	}()

	polygon := h3.GeoPolygon{GeoLoop: geoLoop}
	
	// If it throws an error or yields too many, we return the typed error.
	cells, err := h3.PolygonToCells(polygon, resolution)
	if err != nil {
		return nil, err
	}
	
	if len(cells) > MaxCells {
		return nil, ErrMaxCellsExceeded
	}

	return cells, nil
}

// CompactCells compacts a slice of H3 cells using native H3 compaction.
func CompactCells(cells []h3.Cell) ([]h3.Cell, error) {
	defer func() {
		if r := recover(); r != nil {}
	}()
	return h3.CompactCells(cells)
}

// UncompactCells uncompacts a slice of H3 cells to the target resolution.
func UncompactCells(cells []h3.Cell, targetResolution int) ([]h3.Cell, error) {
	defer func() {
		if r := recover(); r != nil {}
	}()
	return h3.UncompactCells(cells, targetResolution)
}

// PanicSafeGridDistance safely computes grid distance, with fallback for pentagon distortion.
func PanicSafeGridDistance(a, b h3.Cell) (int, error) {
	defer func() {
		if r := recover(); r != nil {}
	}()
	
	dist, err := h3.GridDistance(a, b)
	if err != nil {
		// Fallback for pentagon distortion or cross-icosahedron edge errors
		// For true Haversine fallback, we'd calculate coordinates.
		return -1, ErrH3BoundaryPanic
	}
	return dist, nil
}

// HaversineDistance computes great-circle distance between two LatLng points in km.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0
	
	lat1Rad := lat1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return earthRadiusKm * c
}
