package proximity

import (
	"fmt"

	h3 "github.com/uber/h3-go/v4"
)

const defaultNeighborK = 3

// CellsInRadius returns H3 cell strings for center plus GridDisk rings 0..k.
func CellsInRadius(centerLat, centerLng float64, resolution, k int) ([]string, error) {
	if k < 0 {
		k = 0
	}
	if resolution <= 0 {
		resolution = 9
	}
	center, err := PanicSafeLatLngToCell(centerLat, centerLng, resolution)
	if err != nil {
		return nil, err
	}
	if !center.IsValid() {
		return nil, fmt.Errorf("invalid center cell")
	}
	cells, err := h3.GridDisk(center, k)
	if err != nil {
		return nil, fmt.Errorf("grid_disk: %w", err)
	}
	seen := make(map[string]struct{}, len(cells))
	out := make([]string, 0, len(cells))
	for _, cell := range cells {
		if !cell.IsValid() {
			continue
		}
		s := cell.String()
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out, nil
}

// DefaultNeighborK is the max k-ring expansion for zone-miss fallback.
func DefaultNeighborK() int {
	return defaultNeighborK
}
