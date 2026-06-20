package factory

import (
	"context"
	"math"
	"strings"
)

// factoryIsConfigured reports whether a factory has a usable facility location for ops.
func (s *Service) factoryIsConfigured(ctx context.Context, factoryID string) bool {
	factoryID = strings.TrimSpace(factoryID)
	if factoryID == "" || s.spannerClient == nil {
		return false
	}
	loc, err := s.loadFactoryLocation(ctx, factoryID)
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
