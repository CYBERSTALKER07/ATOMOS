package proximity

import (
	"log"
	
	"github.com/uber/h3-go/v4"
)

// GeoReportResponse represents a geographic coverage report
type GeoReportResponse struct {
	HexesCompacted      []string `json:"hexes_compacted"`
	H3IndexesCompacted  []string `json:"h3_indexes_compacted"`
	H3Resolution        int      `json:"h3_resolution"`
	CoverageTruncated   bool     `json:"coverage_truncated"`
	UniqueOverlapSample int      `json:"unique_overlap_sample"`
}

// GenerateGeoReport generates a geo report with bounds checking and overlap sampling.
func GenerateGeoReport(geoLoop []h3.LatLng, resolution int) (*GeoReportResponse, error) {
	cells, err := PanicSafePolygonToCells(geoLoop, resolution)
	if err != nil {
		return nil, err
	}

	originalCount := len(cells)

	compactedCells, err := CompactCells(cells)
	if err != nil {
		return nil, err
	}

	compactedCount := len(compactedCells)
	RecordCompactionRatio("geo_report", resolution, originalCount, compactedCount)

	truncated := false
	if len(compactedCells) > MaxCompactedEgressCells {
		log.Printf("WARN: geo_report compacted egress circuit-breaker triggered. Truncating %d cells to %d", len(compactedCells), MaxCompactedEgressCells)
		compactedCells = compactedCells[:MaxCompactedEgressCells]
		truncated = true
		RecordEgressOverflow("geo_report", resolution)
	}

	var hexes []string
	var indexes []string
	for _, cell := range compactedCells {
		hexes = append(hexes, cell.String())
		indexes = append(indexes, cell.String())
	}

	return &GeoReportResponse{
		HexesCompacted:      hexes,
		H3IndexesCompacted:  indexes,
		H3Resolution:        resolution,
		CoverageTruncated:   truncated,
		// Example placeholder for unique overlap sample to prevent unbounded egress
		UniqueOverlapSample: compactedCount, 
	}, nil
}
