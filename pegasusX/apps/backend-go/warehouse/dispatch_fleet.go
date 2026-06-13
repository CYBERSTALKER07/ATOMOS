package warehouse

import (
	"context"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
)

func (s *Service) driversOnActiveManifests(ctx context.Context, supplierID, warehouseID string, driverIDs []string) (map[string]bool, error) {
	if s.spannerClient == nil || len(driverIDs) == 0 {
		return map[string]bool{}, nil
	}
	store := s.manifestStore
	if store == nil {
		store = manifest.NewStore(s.spannerClient)
	}
	return store.DriversOnActiveManifests(ctx, supplierID, warehouseID, driverIDs)
}

func warehouseDriverTruckStatus(isActive bool, onActiveManifest bool) (truckStatus string, unavailable bool) {
	if !isActive {
		return "UNAVAILABLE", true
	}
	if onActiveManifest {
		return "IN_TRANSIT", true
	}
	return "AVAILABLE", false
}

func collectWarehouseDriverIDs(drivers []PortalDriver) []string {
	ids := make([]string, 0, len(drivers))
	for _, driver := range drivers {
		if id := strings.TrimSpace(driver.DriverID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
