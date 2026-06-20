package warehouse

import (
	"context"
	"math"
	"strings"
)

// warehouseIsConfigured reports whether a warehouse has a usable depot location for ops.
func (s *Service) warehouseIsConfigured(ctx context.Context, warehouseID string) bool {
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" || s.spannerClient == nil {
		return false
	}
	loc, err := s.loadWarehouseLocation(ctx, warehouseID)
	if err != nil {
		return false
	}
	if strings.TrimSpace(loc.Address) == "" {
		return false
	}
	if loc.Lat < -90 || loc.Lat > 90 || loc.Lng < -180 || loc.Lng > 180 {
		return false
	}
	if math.Abs(loc.Lat) < 1e-6 && math.Abs(loc.Lng) < 1e-6 {
		return false
	}
	return true
}
