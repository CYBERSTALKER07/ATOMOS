package proximity

import (
	"log"
	
	"github.com/uber/h3-go/v4"
)

// ZonePreviewResponse represents the preview of a spatial zone
type ZonePreviewResponse struct {
	HexesCompacted      []string `json:"hexes_compacted"`
	H3IndexesCompacted  []string `json:"h3_indexes_compacted"`
	H3Resolution        int      `json:"h3_resolution"`
	CoverageTruncated   bool     `json:"coverage_truncated"`
	OriginalCellsCount  int      `json:"original_cells_count"`
	CompactedCellsCount int      `json:"compacted_cells_count"`
}

// GenerateZonePreview generates a safe zone preview for a polygon.
func GenerateZonePreview(geoLoop []h3.LatLng, resolution int) (*ZonePreviewResponse, error) {
	// Validate complexity and generate cells
	cells, err := PanicSafePolygonToCells(geoLoop, resolution)
	if err != nil {
		return nil, err
	}

	originalCount := len(cells)

	// Compact the cells
	compactedCells, err := CompactCells(cells)
	if err != nil {
		return nil, err
	}

	compactedCount := len(compactedCells)
	RecordCompactionRatio("preview", resolution, originalCount, compactedCount)

	// Egress Circuit Breaker
	truncated := false
	if len(compactedCells) > MaxCompactedEgressCells {
		log.Printf("WARN: compacted egress circuit-breaker triggered. Truncating %d cells to %d", len(compactedCells), MaxCompactedEgressCells)
		compactedCells = compactedCells[:MaxCompactedEgressCells]
		truncated = true
		RecordEgressOverflow("preview", resolution)
	}

	var hexes []string
	var indexes []string
	for _, cell := range compactedCells {
		hexes = append(hexes, cell.String())
		// In a real system, you might format this specifically. We return the hex string for both in this stub.
		indexes = append(indexes, cell.String())
	}

	return &ZonePreviewResponse{
		HexesCompacted:      hexes,
		H3IndexesCompacted:  indexes,
		H3Resolution:        resolution,
		CoverageTruncated:   truncated,
		OriginalCellsCount:  originalCount,
		CompactedCellsCount: compactedCount,
	}, nil
}
