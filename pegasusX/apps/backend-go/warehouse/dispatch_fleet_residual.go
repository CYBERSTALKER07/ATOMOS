package warehouse

import (
	"context"
	"math"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
)

// fleetDispatchContext holds manifest-aware fleet eligibility for dispatch.
type fleetDispatchContext struct {
	InTransit map[string]bool
	TopOff    map[string]manifest.DriverManifestCapacity
}

func (s *Service) loadFleetDispatchContext(ctx context.Context, supplierID, warehouseID string, driverIDs []string) (fleetDispatchContext, error) {
	out := fleetDispatchContext{
		InTransit: map[string]bool{},
		TopOff:    map[string]manifest.DriverManifestCapacity{},
	}
	if len(driverIDs) == 0 {
		return out, nil
	}
	store := s.manifestStore
	if store == nil {
		store = manifest.NewStore(s.spannerClient)
	}
	inTransit, err := store.DriversOnInTransitManifests(ctx, supplierID, warehouseID, driverIDs)
	if err != nil {
		return out, err
	}
	topOff, err := store.ActiveManifestCapacityByDrivers(ctx, supplierID, warehouseID, driverIDs)
	if err != nil {
		return out, err
	}
	out.InTransit = inTransit
	out.TopOff = topOff
	return out, nil
}

func (fc fleetDispatchContext) topOffFor(driverID string) *manifest.DriverManifestCapacity {
	if cap, ok := fc.TopOff[driverID]; ok && cap.ManifestID != "" {
		c := cap
		return &c
	}
	return nil
}

func driverResidualVolumes(driver PortalDriver, topOff *manifest.DriverManifestCapacity) (usedVU, freeVU, effectiveMaxVU float64) {
	maxVU := driver.MaxVolumeVU
	if maxVU <= 0 {
		return 0, 0, 0
	}
	if topOff != nil && topOff.ManifestID != "" {
		usedVU = topOff.TotalVolumeVU
		if topOff.MaxVolumeVU > 0 {
			maxVU = topOff.MaxVolumeVU
		}
	}
	freeVU = math.Max(0, maxVU-usedVU)
	effectiveMaxVU = freeVU * dispatch.TetrisBuffer
	return usedVU, freeVU, effectiveMaxVU
}

func driverDispatchEligible(driver PortalDriver, fleetCtx fleetDispatchContext) bool {
	driverID := strings.TrimSpace(driver.DriverID)
	if !driver.IsActive || !driver.OnShift || driverID == "" {
		return false
	}
	if fleetCtx.InTransit[driverID] {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(driver.TruckStatus), "AVAILABLE") {
		return false
	}
	_, freeVU, _ := driverResidualVolumes(driver, fleetCtx.topOffFor(driverID))
	return freeVU > 0
}

func fleetEffectiveCapacityVU(drivers []PortalDriver, fleetCtx fleetDispatchContext) float64 {
	var total float64
	for _, driver := range drivers {
		if !driverDispatchEligible(driver, fleetCtx) {
			continue
		}
		_, _, effective := driverResidualVolumes(driver, fleetCtx.topOffFor(driver.DriverID))
		total += effective
	}
	return total
}

func buildFleetDriverInputs(drivers []PortalDriver, fleetCtx fleetDispatchContext, homeNodeID string) []dispatch.FleetDriverInput {
	inputs := make([]dispatch.FleetDriverInput, 0, len(drivers))
	for _, driver := range drivers {
		if !driverDispatchEligible(driver, fleetCtx) {
			continue
		}
		_, freeVU, _ := driverResidualVolumes(driver, fleetCtx.topOffFor(driver.DriverID))
		inputs = append(inputs, dispatch.FleetDriverInput{
			DriverID:     driver.DriverID,
			DriverName:   driver.Name,
			VehicleID:    driver.VehicleID,
			VehicleClass: driver.VehicleClass,
			MaxVolumeVU:  freeVU,
			IsActive:     driver.IsActive,
			TruckStatus:  driver.TruckStatus,
			HomeNodeID:   homeNodeID,
		})
	}
	return inputs
}

func driverPreviewEntry(driver PortalDriver, fleetCtx fleetDispatchContext) map[string]any {
	driverID := strings.TrimSpace(driver.DriverID)
	usedVU, freeVU, _ := driverResidualVolumes(driver, fleetCtx.topOffFor(driverID))
	entry := map[string]any{
		"driver_id":     driver.DriverID,
		"name":          driver.Name,
		"vehicle_id":    driver.VehicleID,
		"vehicle_label": driver.VehicleLabel,
		"vehicle_class": driver.VehicleClass,
		"max_volume_vu": driver.MaxVolumeVU,
	}
	if usedVU > 0 {
		entry["used_volume_vu"] = usedVU
		entry["free_volume_vu"] = freeVU
		if top := fleetCtx.topOffFor(driverID); top != nil {
			entry["active_manifest_id"] = top.ManifestID
		}
	}
	return entry
}

func warehouseDriverAvailability(driver PortalDriver, fleetCtx fleetDispatchContext) (truckStatus string, unavailable bool, reason string) {
	driverID := strings.TrimSpace(driver.DriverID)
	if !driver.IsActive {
		return "UNAVAILABLE", true, "INACTIVE"
	}
	if !driver.OnShift {
		status := driverOffShiftTruckStatus(driver.UnavailableReason)
		return status, true, driverUnavailableDisplayReason(driver.UnavailableReason, driver.UnavailableNote)
	}
	if fleetCtx.InTransit[driverID] {
		return "IN_TRANSIT", true, "IN_TRANSIT"
	}
	if !strings.EqualFold(strings.TrimSpace(driver.TruckStatus), "AVAILABLE") {
		display := strings.TrimSpace(driver.TruckStatus)
		if strings.EqualFold(display, "VEHICLE_INACTIVE") {
			if vr := strings.TrimSpace(driver.VehicleUnavailableReason); vr != "" {
				display = vehicleUnavailableDisplayReason(vr, driver.VehicleUnavailableNote)
			}
		}
		return driver.TruckStatus, true, display
	}
	_, freeVU, _ := driverResidualVolumes(driver, fleetCtx.topOffFor(driverID))
	if freeVU <= 0 {
		return "FULL", true, "MANIFEST_FULL"
	}
	return "AVAILABLE", false, ""
}

func filterDispatchRowsByOrderIDs(rows []dispatch.DispatchableOrder, orderIDs []string) []dispatch.DispatchableOrder {
	if len(orderIDs) == 0 {
		return rows
	}
	allowed := make(map[string]struct{}, len(orderIDs))
	for _, id := range orderIDs {
		if id = strings.TrimSpace(id); id != "" {
			allowed[id] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return rows
	}
	filtered := make([]dispatch.DispatchableOrder, 0, len(orderIDs))
	for _, row := range rows {
		if _, ok := allowed[row.OrderID]; ok {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func sumOrderVolumeVU(rows []dispatch.DispatchableOrder) float64 {
	var total float64
	for _, row := range rows {
		total += row.VolumeVU
	}
	return total
}
