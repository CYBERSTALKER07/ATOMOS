package proximity

import (
	"errors"
	"fmt"
	"log"
	"math"
	"sort"

	h3 "github.com/uber/h3-go/v4"
)

// ─── H3 Hexagonal Grid — uber/h3-go/v4 ─────────────────────────────────────────
//
// Cell IDs are 15-char lowercase hex strings produced by Cell.String() —
// directly decodable by h3-js on the MapLibre frontend.
//
// Resolution 7 ≈ 5.16 km² per hex; neighbor center-to-center ≈ 2.11 km.

const (
	EarthRadiusKm = 6371.0
	H3Resolution  = 7
	// MaxCoveragePolygonPoints caps polygon complexity to protect CPU and heap.
	MaxCoveragePolygonPoints = 512
	// MaxCoverageAreaRes7Km2 bounds accepted polygon footprint at res7.
	MaxCoverageAreaRes7Km2 = 12000.0
	// MaxCoverageAreaRes8Km2 bounds accepted polygon footprint at res8.
	MaxCoverageAreaRes8Km2 = 3500.0
	// MaxCoverageCells caps total generated coverage cells from one polygon.
	MaxCoverageCells = 20000
	// MaxCoverageFallbackIterations bounds fallback grid sampler work.
	MaxCoverageFallbackIterations = 250000
	// MaxCoverageResponseCells caps emitted cells in large JSON payloads.
	MaxCoverageResponseCells = 8000
	// MaxCompactedEgressCells caps compacted cell arrays per egress payload.
	MaxCompactedEgressCells = 5000
	// Approximate hex edge length at resolution 7 in km.
	H3Res7EdgeKm = 1.22
	// Neighbor center-to-center distance at resolution 7 (edge * sqrt(3)).
	h3Res7CenterKm = 2.11
)

var (
	ErrCoveragePolygonTooFewPoints   = errors.New("coverage polygon must contain at least 3 points")
	ErrCoveragePolygonTooManyPoints  = errors.New("coverage polygon exceeds max point count")
	ErrCoverageCoordinatesOutOfRange = errors.New("coverage polygon coordinates out of range")
	ErrCoveragePolygonAreaTooLarge   = errors.New("coverage polygon area exceeds allowed bounds")
	ErrCoverageCellLimitExceeded     = errors.New("coverage cell count exceeds allowed bounds")
	ErrCoverageSamplerLimitExceeded  = errors.New("coverage fallback sampler exceeded iteration budget")
	ErrCoverageResponseLimitExceeded = errors.New("coverage response size exceeds allowed bounds")
)

// NormalizeCoverageResolution coerces unsupported values to the canonical default.
func NormalizeCoverageResolution(resolution int) int {
	if resolution != 7 && resolution != 8 {
		return H3Resolution
	}
	return resolution
}

// MaxCoverageAreaKm2 returns the polygon area budget for a requested resolution.
func MaxCoverageAreaKm2(resolution int) float64 {
	if NormalizeCoverageResolution(resolution) == 8 {
		return MaxCoverageAreaRes8Km2
	}
	return MaxCoverageAreaRes7Km2
}

// ValidateCoveragePolygon enforces polygon complexity, coordinates, and area limits.
func ValidateCoveragePolygon(polygon [][2]float64, resolution int) error {
	if len(polygon) < 3 {
		return ErrCoveragePolygonTooFewPoints
	}
	if len(polygon) > MaxCoveragePolygonPoints {
		return fmt.Errorf("%w: got %d max %d", ErrCoveragePolygonTooManyPoints, len(polygon), MaxCoveragePolygonPoints)
	}

	for _, point := range polygon {
		if point[0] < -90 || point[0] > 90 || point[1] < -180 || point[1] > 180 {
			return ErrCoverageCoordinatesOutOfRange
		}
	}

	areaKm2 := estimatePolygonBoundingAreaKm2(polygon)
	maxArea := MaxCoverageAreaKm2(resolution)
	if areaKm2 > maxArea {
		return fmt.Errorf("%w: area=%.2f max=%.2f", ErrCoveragePolygonAreaTooLarge, areaKm2, maxArea)
	}

	return nil
}

// ValidateCoverageCellsCount enforces coverage-set size limits.
func ValidateCoverageCellsCount(count int) error {
	if count > MaxCoverageCells {
		return fmt.Errorf("%w: got %d max %d", ErrCoverageCellLimitExceeded, count, MaxCoverageCells)
	}
	return nil
}

// ValidateCoverageResponseCount enforces response payload guardrails.
func ValidateCoverageResponseCount(count int) error {
	if count > MaxCoverageResponseCells {
		return fmt.Errorf("%w: got %d max %d", ErrCoverageResponseLimitExceeded, count, MaxCoverageResponseCells)
	}
	return nil
}

// LimitCoverageCellsForResponse returns at most max cells plus truncation state.
func LimitCoverageCellsForResponse(cells []string, max int) ([]string, bool) {
	if max <= 0 || len(cells) <= max {
		return append([]string(nil), cells...), false
	}
	return append([]string(nil), cells[:max]...), true
}

// CoverageErrorStatus maps coverage validation errors to stable HTTP status codes.
func CoverageErrorStatus(err error) int {
	switch {
	case errors.Is(err, ErrCoveragePolygonTooFewPoints),
		errors.Is(err, ErrCoverageCoordinatesOutOfRange):
		return 400
	case errors.Is(err, ErrCoveragePolygonTooManyPoints),
		errors.Is(err, ErrCoveragePolygonAreaTooLarge):
		return 422
	case errors.Is(err, ErrCoverageCellLimitExceeded),
		errors.Is(err, ErrCoverageSamplerLimitExceeded),
		errors.Is(err, ErrCoverageResponseLimitExceeded):
		return 413
	default:
		return 500
	}
}

// CoverageErrorMessage maps coverage validation errors to API-safe error labels.
func CoverageErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrCoveragePolygonTooFewPoints):
		return "polygon must have at least 3 points"
	case errors.Is(err, ErrCoveragePolygonTooManyPoints):
		return "polygon exceeds maximum allowed point count"
	case errors.Is(err, ErrCoverageCoordinatesOutOfRange):
		return "coordinates out of range"
	case errors.Is(err, ErrCoveragePolygonAreaTooLarge):
		return "polygon area exceeds allowed bounds"
	case errors.Is(err, ErrCoverageCellLimitExceeded):
		return "coverage cell count exceeds allowed bounds"
	case errors.Is(err, ErrCoverageSamplerLimitExceeded):
		return "coverage computation budget exceeded"
	case errors.Is(err, ErrCoverageResponseLimitExceeded):
		return "coverage response exceeds payload limits"
	default:
		return "internal server error"
	}
}

func estimatePolygonBoundingAreaKm2(polygon [][2]float64) float64 {
	if len(polygon) < 3 {
		return 0
	}

	minLat, maxLat := polygon[0][0], polygon[0][0]
	minLng, maxLng := polygon[0][1], polygon[0][1]
	for _, point := range polygon[1:] {
		if point[0] < minLat {
			minLat = point[0]
		}
		if point[0] > maxLat {
			maxLat = point[0]
		}
		if point[1] < minLng {
			minLng = point[1]
		}
		if point[1] > maxLng {
			maxLng = point[1]
		}
	}

	latSpanKm := math.Abs(maxLat-minLat) * 111.0
	midLat := (minLat + maxLat) / 2.0
	lngSpanKm := math.Abs(maxLng-minLng) * 111.0 * math.Cos(degreesToRadians(midLat))
	if lngSpanKm < 0 {
		lngSpanKm = -lngSpanKm
	}

	return latSpanKm * lngSpanKm
}

// HaversineKm returns the great-circle distance in km between two lat/lng points.
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := degreesToRadians(lat2 - lat1)
	dLng := degreesToRadians(lng2 - lng1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return EarthRadiusKm * c
}

// IsWithinRadius checks if point (lat2, lng2) is within radiusKm of (lat1, lng1).
func IsWithinRadius(lat1, lng1, lat2, lng2, radiusKm float64) bool {
	return HaversineKm(lat1, lng1, lat2, lng2) <= radiusKm
}

// ComputeGridCoverage returns the H3 res-7 cell IDs whose centers lie within
// radiusKm of (lat, lng). Cell IDs are 15-char lowercase hex strings directly
// consumable by h3-js on the frontend and Spanner ARRAY<STRING> columns.
func ComputeGridCoverage(lat, lng, radiusKm float64) []string {
	origin, err := safeLatLngToCell(lat, lng, H3Resolution)
	if err != nil {
		return nil
	}

	k := int(math.Ceil(radiusKm/h3Res7CenterKm)) + 1
	if k < 1 {
		k = 1
	}

	disk, err := safeGridDisk(origin, k)
	if err != nil {
		return nil
	}

	out := make([]string, 0, len(disk))
	for _, c := range disk {
		ll, err := safeCellLatLng(c)
		if err != nil {
			continue
		}
		if HaversineKm(lat, lng, ll.Lat, ll.Lng) <= radiusKm {
			out = append(out, c.String())
		}
	}
	return out
}

// LookupCell returns the H3 res-7 cell ID containing the point.
func LookupCell(lat, lng float64) string {
	c, err := safeLatLngToCell(lat, lng, H3Resolution)
	if err != nil {
		return ""
	}
	return c.String()
}

// CellToLatLng returns the geographic center of an H3 cell ID. Returns
// (0, 0, false) if the cell ID is invalid.
func CellToLatLng(cellID string) (lat, lng float64, ok bool) {
	c := h3.CellFromString(cellID)
	if !c.IsValid() {
		return 0, 0, false
	}
	ll, err := safeCellLatLng(c)
	if err != nil {
		return 0, 0, false
	}
	return ll.Lat, ll.Lng, true
}

// GridDisk returns all H3 cell IDs within k rings of the given cell.
func GridDisk(cellID string, k int) []string {
	c := h3.CellFromString(cellID)
	if !c.IsValid() {
		return nil
	}
	disk, err := safeGridDisk(c, k)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(disk))
	for _, d := range disk {
		out = append(out, d.String())
	}
	return out
}

// polygonToCells returns the H3 cells covering the polygon at the requested
// resolution using the native H3 polygon fill implementation.
func polygonToCells(polygon [][2]float64, resolution int) ([]string, error) {
	if len(polygon) < 3 {
		return nil, nil
	}

	loop := make(h3.GeoLoop, 0, len(polygon))
	for _, point := range polygon {
		loop = append(loop, h3.NewLatLng(point[0], point[1]))
	}

	cells, err := safePolygonToCells(polygon, loop, resolution)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(cells))
	for _, cell := range cells {
		out = append(out, cell.String())
	}
	sort.Strings(out)
	return out, nil
}

// CompactCells compacts same-resolution H3 cell IDs into their coarsest valid
// representation for transport-friendly payloads.
func CompactCells(cells []string) ([]string, error) {
	indexes, err := cellsFromStrings(cells)
	if err != nil {
		return nil, err
	}
	if len(indexes) == 0 {
		return []string{}, nil
	}

	compacted, err := h3.CompactCells(indexes)
	if err != nil {
		return nil, err
	}
	return stringsFromCells(compacted), nil
}

// UncompactCells expands compacted H3 cell IDs to the requested resolution.
func UncompactCells(cells []string, resolution int) ([]string, error) {
	indexes, err := cellsFromStrings(cells)
	if err != nil {
		return nil, err
	}
	if len(indexes) == 0 {
		return []string{}, nil
	}

	uncompacted, err := h3.UncompactCells(indexes, resolution)
	if err != nil {
		return nil, err
	}
	return stringsFromCells(uncompacted), nil
}

func cellsFromStrings(cells []string) ([]h3.Cell, error) {
	seen := make(map[string]struct{}, len(cells))
	indexes := make([]h3.Cell, 0, len(cells))
	for _, cellID := range cells {
		if _, exists := seen[cellID]; exists {
			continue
		}
		seen[cellID] = struct{}{}

		cell := h3.CellFromString(cellID)
		if !cell.IsValid() {
			return nil, fmt.Errorf("invalid H3 cell: %s", cellID)
		}
		indexes = append(indexes, cell)
	}
	return indexes, nil
}

func stringsFromCells(cells []h3.Cell) []string {
	out := make([]string, 0, len(cells))
	for _, cell := range cells {
		out = append(out, cell.String())
	}
	sort.Strings(out)
	return out
}

// CoverageResolution returns the resolution of the first valid H3 cell in the
// provided set, or the canonical default when the set is empty.
func CoverageResolution(cells []string) int {
	for _, cellID := range cells {
		cell := h3.CellFromString(cellID)
		if cell.IsValid() {
			return cell.Resolution()
		}
	}
	return H3Resolution
}

func runH3BoundaryCall(operation string, fn func() error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%s panic: %v", operation, recovered)
		}
	}()

	if callErr := fn(); callErr != nil {
		return fmt.Errorf("%s failed: %w", operation, callErr)
	}

	return nil
}

func safeLatLngToCell(lat, lng float64, resolution int) (h3.Cell, error) {
	var cell h3.Cell
	err := runH3BoundaryCall("lat_lng_to_cell", func() error {
		var callErr error
		cell, callErr = h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lng}, resolution)
		return callErr
	})
	if err != nil {
		return h3.Cell(0), err
	}
	return cell, nil
}

func safeCellLatLng(cell h3.Cell) (h3.LatLng, error) {
	var ll h3.LatLng
	err := runH3BoundaryCall("cell_lat_lng", func() error {
		var callErr error
		ll, callErr = cell.LatLng()
		return callErr
	})
	if err != nil {
		return h3.LatLng{}, err
	}
	return ll, nil
}

func cellTouchesPentagon(cell h3.Cell) bool {
	if cell.IsPentagon() {
		return true
	}

	var neighbors []h3.Cell
	err := runH3BoundaryCall("grid_disk_probe", func() error {
		var callErr error
		neighbors, callErr = cell.GridDisk(1)
		return callErr
	})
	if err != nil {
		return false
	}

	for _, neighbor := range neighbors {
		if neighbor.IsPentagon() {
			return true
		}
	}

	return false
}

func safeGridDisk(origin h3.Cell, k int) ([]h3.Cell, error) {
	var disk []h3.Cell
	err := runH3BoundaryCall("grid_disk", func() error {
		var callErr error
		disk, callErr = origin.GridDisk(k)
		return callErr
	})
	if err == nil {
		return disk, nil
	}

	if cellTouchesPentagon(origin) {
		fallback, fallbackErr := gridDiskHaversineFallback(origin, k)
		if fallbackErr == nil {
			logPentagonFallbackAlert("grid_disk", fmt.Sprintf("origin=%s k=%d", origin.String(), k), err)
			return fallback, nil
		}
		return nil, fmt.Errorf("grid_disk fallback failed: %w", fallbackErr)
	}

	return nil, err
}

func edgeLengthKmForResolution(resolution int) float64 {
	if resolution == 8 {
		return H3Res7EdgeKm / 2.6457
	}
	return H3Res7EdgeKm
}

func centerDistanceKmForResolution(resolution int) float64 {
	if resolution == 8 {
		return h3Res7CenterKm / 2.6457
	}
	return h3Res7CenterKm
}

func gridDiskHaversineFallback(origin h3.Cell, k int) ([]h3.Cell, error) {
	originCenter, err := safeCellLatLng(origin)
	if err != nil {
		return nil, err
	}

	resolution := origin.Resolution()
	edgeKm := edgeLengthKmForResolution(resolution)
	radiusKm := float64(k+1) * centerDistanceKmForResolution(resolution)

	stepLat := edgeKm / 111.0
	cosLat := math.Cos(degreesToRadians(originCenter.Lat))
	if cosLat < 1e-10 {
		cosLat = 1e-10
	}
	stepLng := edgeKm / (111.0 * cosLat)
	if stepLat <= 0 || stepLng <= 0 || math.IsNaN(stepLat) || math.IsNaN(stepLng) || math.IsInf(stepLng, 0) {
		return nil, ErrCoverageSamplerLimitExceeded
	}

	minLat := originCenter.Lat - (radiusKm+edgeKm)/111.0
	maxLat := originCenter.Lat + (radiusKm+edgeKm)/111.0
	lngDelta := (radiusKm + edgeKm) / (111.0 * cosLat)
	minLng := originCenter.Lng - lngDelta
	maxLng := originCenter.Lng + lngDelta

	latSteps := int(math.Ceil((maxLat-minLat)/stepLat)) + 1
	lngSteps := int(math.Ceil((maxLng-minLng)/stepLng)) + 1
	iterations := latSteps * lngSteps
	if iterations > MaxCoverageFallbackIterations {
		return nil, fmt.Errorf("%w: iterations=%d max=%d", ErrCoverageSamplerLimitExceeded, iterations, MaxCoverageFallbackIterations)
	}

	cells := make(map[string]h3.Cell)
	cells[origin.String()] = origin

	for lat := minLat; lat <= maxLat; lat += stepLat {
		for lng := minLng; lng <= maxLng; lng += stepLng {
			if HaversineKm(originCenter.Lat, originCenter.Lng, lat, lng) > radiusKm+edgeKm {
				continue
			}

			cell, cellErr := safeLatLngToCell(lat, lng, resolution)
			if cellErr != nil {
				continue
			}

			if len(cells) >= MaxCoverageCells {
				return nil, fmt.Errorf("%w: got >=%d", ErrCoverageCellLimitExceeded, MaxCoverageCells)
			}
			cells[cell.String()] = cell
		}
	}

	out := make([]h3.Cell, 0, len(cells))
	for _, cell := range cells {
		out = append(out, cell)
	}

	return out, nil
}

func safePolygonToCells(polygon [][2]float64, loop h3.GeoLoop, resolution int) ([]h3.Cell, error) {
	pentagonRisk := polygonTouchesPentagon(polygon, resolution)

	var cells []h3.Cell
	err := runH3BoundaryCall("polygon_to_cells", func() error {
		var callErr error
		cells, callErr = h3.PolygonToCells(h3.GeoPolygon{GeoLoop: loop}, resolution)
		return callErr
	})
	if err == nil {
		return cells, nil
	}

	if pentagonRisk {
		fallback, fallbackErr := polygonToCellsHaversineFallback(polygon, resolution)
		if fallbackErr == nil {
			logPentagonFallbackAlert("polygon_to_cells", fmt.Sprintf("resolution=%d", resolution), err)
			return fallback, nil
		}
		return nil, fmt.Errorf("polygon_to_cells fallback failed: %w", fallbackErr)
	}

	return nil, err
}

func polygonTouchesPentagon(polygon [][2]float64, resolution int) bool {
	if len(polygon) == 0 {
		return false
	}

	step := 1
	if len(polygon) > 24 {
		step = len(polygon) / 24
	}

	for i := 0; i < len(polygon); i += step {
		cell, err := safeLatLngToCell(polygon[i][0], polygon[i][1], resolution)
		if err != nil {
			continue
		}
		if cellTouchesPentagon(cell) {
			return true
		}
	}

	centerLat, centerLng := polygonCentroid(polygon)
	cell, err := safeLatLngToCell(centerLat, centerLng, resolution)
	if err != nil {
		return false
	}

	return cellTouchesPentagon(cell)
}

func polygonCentroid(polygon [][2]float64) (float64, float64) {
	if len(polygon) == 0 {
		return 0, 0
	}

	var sumLat, sumLng float64
	for _, point := range polygon {
		sumLat += point[0]
		sumLng += point[1]
	}

	count := float64(len(polygon))
	return sumLat / count, sumLng / count
}

func polygonToCellsHaversineFallback(polygon [][2]float64, resolution int) ([]h3.Cell, error) {
	edgeKm := edgeLengthKmForResolution(resolution)

	minLat, maxLat := polygon[0][0], polygon[0][0]
	minLng, maxLng := polygon[0][1], polygon[0][1]
	for _, point := range polygon[1:] {
		if point[0] < minLat {
			minLat = point[0]
		}
		if point[0] > maxLat {
			maxLat = point[0]
		}
		if point[1] < minLng {
			minLng = point[1]
		}
		if point[1] > maxLng {
			maxLng = point[1]
		}
	}

	centerLat, centerLng := polygonCentroid(polygon)
	radiusKm := 0.0
	for _, point := range polygon {
		distance := HaversineKm(centerLat, centerLng, point[0], point[1])
		if distance > radiusKm {
			radiusKm = distance
		}
	}

	stepLat := edgeKm / 111.0
	cosLat := math.Cos(degreesToRadians(centerLat))
	if cosLat < 1e-10 {
		cosLat = 1e-10
	}
	stepLng := edgeKm / (111.0 * cosLat)
	if stepLat <= 0 || stepLng <= 0 || math.IsNaN(stepLat) || math.IsNaN(stepLng) || math.IsInf(stepLng, 0) {
		return nil, ErrCoverageSamplerLimitExceeded
	}

	latSpan := (maxLat + stepLat) - (minLat - stepLat)
	lngSpan := (maxLng + stepLng) - (minLng - stepLng)
	latSteps := int(math.Ceil(latSpan/stepLat)) + 1
	lngSteps := int(math.Ceil(lngSpan/stepLng)) + 1
	iterations := latSteps * lngSteps
	if iterations > MaxCoverageFallbackIterations {
		return nil, fmt.Errorf("%w: iterations=%d max=%d", ErrCoverageSamplerLimitExceeded, iterations, MaxCoverageFallbackIterations)
	}

	cells := make(map[string]h3.Cell)
	for lat := minLat - stepLat; lat <= maxLat+stepLat; lat += stepLat {
		for lng := minLng - stepLng; lng <= maxLng+stepLng; lng += stepLng {
			if !pointInPolygon(lat, lng, polygon) {
				continue
			}
			if HaversineKm(centerLat, centerLng, lat, lng) > radiusKm+edgeKm {
				continue
			}

			cell, cellErr := safeLatLngToCell(lat, lng, resolution)
			if cellErr != nil {
				continue
			}

			if len(cells) >= MaxCoverageCells {
				return nil, fmt.Errorf("%w: got >=%d", ErrCoverageCellLimitExceeded, MaxCoverageCells)
			}
			cells[cell.String()] = cell
		}
	}

	out := make([]h3.Cell, 0, len(cells))
	for _, cell := range cells {
		out = append(out, cell)
	}

	return out, nil
}

func safeGridDistanceCells(a, b h3.Cell) (int, error) {
	var distance int
	err := runH3BoundaryCall("grid_distance", func() error {
		var callErr error
		distance, callErr = h3.GridDistance(a, b)
		return callErr
	})
	if err == nil {
		return distance, nil
	}

	if cellTouchesPentagon(a) || cellTouchesPentagon(b) {
		fallback, fallbackErr := haversineGridDistanceFallback(a, b)
		if fallbackErr == nil {
			logPentagonFallbackAlert("grid_distance", fmt.Sprintf("a=%s b=%s", a.String(), b.String()), err)
			return fallback, nil
		}
		return 0, fmt.Errorf("grid_distance fallback failed: %w", fallbackErr)
	}

	return 0, err
}

func haversineGridDistanceFallback(a, b h3.Cell) (int, error) {
	aCenter, err := safeCellLatLng(a)
	if err != nil {
		return 0, err
	}
	bCenter, err := safeCellLatLng(b)
	if err != nil {
		return 0, err
	}

	resolution := a.Resolution()
	if b.Resolution() > resolution {
		resolution = b.Resolution()
	}

	stepKm := centerDistanceKmForResolution(resolution)
	if stepKm <= 0 {
		return 0, ErrCoverageSamplerLimitExceeded
	}

	distanceKm := HaversineKm(aCenter.Lat, aCenter.Lng, bCenter.Lat, bCenter.Lng)
	rings := int(math.Ceil(distanceKm / stepKm))
	if rings < 0 {
		rings = 0
	}

	return rings, nil
}

func logPentagonFallbackAlert(operation, details string, cause error) {
	log.Printf("[ALERT][H3-PENTAGON-FALLBACK] operation=%s details=%s action=haversine_fallback cause=%v", operation, details, cause)
}

func degreesToRadians(d float64) float64 {
	return d * math.Pi / 180
}
