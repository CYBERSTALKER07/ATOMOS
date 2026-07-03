package supplier

import (
	"context"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
)

func filterDispatchRowsByManualRoutes(rows []dispatch.DispatchableOrder, routes []dispatch.ManualRouteInput) []dispatch.DispatchableOrder {
	selected := make(map[string]struct{})
	for _, route := range routes {
		for _, oid := range route.OrderIDs {
			if oid = strings.TrimSpace(oid); oid != "" {
				selected[oid] = struct{}{}
			}
		}
	}
	if len(selected) == 0 {
		return nil
	}
	filtered := make([]dispatch.DispatchableOrder, 0, len(selected))
	for _, row := range rows {
		if _, ok := selected[row.OrderID]; ok {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func supplierDriverMaxVUMap(ctx context.Context, s *Service, supplierID, warehouseID string, vehiclesByID map[string]dispatch.VehicleSpec) map[string]float64 {
	out := make(map[string]float64)
	if s == nil || s.repo == nil {
		return out
	}
	drivers, err := s.repo.ListFleetDrivers(ctx, supplierID)
	if err != nil {
		return out
	}
	for _, driver := range drivers {
		if !driver.IsActive {
			continue
		}
		if warehouseID != "" && !strings.EqualFold(strings.TrimSpace(driver.HomeNodeID), warehouseID) {
			continue
		}
		driverID := strings.TrimSpace(driver.DriverID)
		if driverID == "" {
			continue
		}
		vehicleID := strings.TrimSpace(driver.VehicleID)
		if spec, ok := vehiclesByID[vehicleID]; ok && spec.MaxVolumeVU > 0 {
			out[driverID] = spec.MaxVolumeVU
		}
	}
	return out
}

func supplierVehicleByDriverMap(ctx context.Context, s *Service, supplierID, warehouseID string) map[string]string {
	out := make(map[string]string)
	if s == nil || s.repo == nil {
		return out
	}
	drivers, err := s.repo.ListFleetDrivers(ctx, supplierID)
	if err != nil {
		return out
	}
	for _, driver := range drivers {
		if !driver.IsActive {
			continue
		}
		if warehouseID != "" && !strings.EqualFold(strings.TrimSpace(driver.HomeNodeID), warehouseID) {
			continue
		}
		driverID := strings.TrimSpace(driver.DriverID)
		if driverID == "" {
			continue
		}
		out[driverID] = strings.TrimSpace(driver.VehicleID)
	}
	return out
}
