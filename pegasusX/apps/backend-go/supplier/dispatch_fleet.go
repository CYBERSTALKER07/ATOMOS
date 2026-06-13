package supplier

import (
	"context"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
)

func (s *Service) driversOnActiveManifests(ctx context.Context, supplierID, warehouseID string, driverIDs []string) (map[string]bool, error) {
	if s.portalSpanner == nil || len(driverIDs) == 0 {
		return map[string]bool{}, nil
	}
	store := s.manifestStore
	if store == nil {
		store = manifest.NewStore(s.portalSpanner)
	}
	return store.DriversOnActiveManifests(ctx, supplierID, warehouseID, driverIDs)
}

func supplierDriverTruckStatus(isActive bool, onActiveManifest bool) (truckStatus string, unavailable bool) {
	if !isActive {
		return "UNAVAILABLE", true
	}
	if onActiveManifest {
		return "IN_TRANSIT", true
	}
	return "AVAILABLE", false
}

func collectSupplierDriverIDs(drivers []SupplierFleetDriver, warehouseID string) []string {
	ids := make([]string, 0, len(drivers))
	for _, driver := range drivers {
		if warehouseID != "" && !strings.EqualFold(strings.TrimSpace(driver.HomeNodeID), warehouseID) {
			continue
		}
		if id := strings.TrimSpace(driver.DriverID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
