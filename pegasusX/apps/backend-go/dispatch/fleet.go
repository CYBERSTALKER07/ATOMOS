package dispatch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// FleetDriverInput is a minimal driver row for fleet hydration.
type FleetDriverInput struct {
	DriverID     string
	DriverName   string
	VehicleID    string
	VehicleClass string
	MaxVolumeVU  float64
	IsActive     bool
	TruckStatus  string
	HomeNodeID   string
}

// DepotCoords is the dispatch origin for route solving.
type DepotCoords struct {
	Lat float64
	Lng float64
}

// FetchWarehouseDepot reads warehouse lat/lng for dispatch origin.
func FetchWarehouseDepot(ctx context.Context, client *spanner.Client, warehouseID string) (DepotCoords, error) {
	if client == nil || strings.TrimSpace(warehouseID) == "" {
		return DepotCoords{}, fmt.Errorf("dispatch: missing spanner client or warehouse id")
	}
	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(Lat, 0), COALESCE(Lng, 0)
		      FROM Warehouses WHERE WarehouseId = @wid`,
		Params: map[string]any{"wid": warehouseID},
	}
	iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return DepotCoords{}, fmt.Errorf("dispatch: warehouse %s not found", warehouseID)
	}
	if err != nil {
		return DepotCoords{}, fmt.Errorf("dispatch: warehouse depot query: %w", err)
	}
	var depot DepotCoords
	if err := row.Columns(&depot.Lat, &depot.Lng); err != nil {
		return DepotCoords{}, fmt.Errorf("dispatch: warehouse depot scan: %w", err)
	}
	return depot, nil
}

// ResolveDepot picks warehouse coordinates when available, otherwise fallback.
func ResolveDepot(ctx context.Context, client *spanner.Client, warehouseID string, fallback DepotCoords) DepotCoords {
	if client != nil && strings.TrimSpace(warehouseID) != "" {
		if depot, err := FetchWarehouseDepot(ctx, client, warehouseID); err == nil {
			if depot.Lat != 0 || depot.Lng != 0 {
				return depot
			}
		}
	}
	return fallback
}

// BuildAvailableFleet maps active drivers with vehicle ids into solver fleet rows.
func BuildAvailableFleet(drivers []FleetDriverInput, vehiclesByID map[string]VehicleSpec) []AvailableDriver {
	fleet := make([]AvailableDriver, 0, len(drivers))
	for _, driver := range drivers {
		if !driver.IsActive {
			continue
		}
		status := strings.ToUpper(strings.TrimSpace(driver.TruckStatus))
		if status != "" && status != "AVAILABLE" && status != "IDLE" {
			continue
		}
		vehicleID := strings.TrimSpace(driver.VehicleID)
		if vehicleID == "" {
			continue
		}
		class := driver.VehicleClass
		capacity := driver.MaxVolumeVU
		if spec, ok := vehiclesByID[vehicleID]; ok {
			if strings.TrimSpace(class) == "" {
				class = spec.VehicleClass
			}
			if capacity <= 0 {
				capacity = spec.MaxVolumeVU
			}
		}
		class = ResolveVehicleClass(class)
		capacity = ResolveMaxVolumeVU(class, capacity)
		fleet = append(fleet, AvailableDriver{
			DriverID:     driver.DriverID,
			DriverName:   driver.DriverName,
			VehicleID:    vehicleID,
			VehicleClass: class,
			MaxVolumeVU:  capacity,
		})
	}
	return fleet
}

// VehicleSpecIndex builds a vehicle-id lookup from fleet vehicle rows.
func VehicleSpecIndex(vehicleID string, vehicleClass string, maxVolumeVU float64) (string, VehicleSpec) {
	class := ResolveVehicleClass(vehicleClass)
	return vehicleID, VehicleSpec{
		VehicleClass: class,
		MaxVolumeVU:  ResolveMaxVolumeVU(class, maxVolumeVU),
	}
}
