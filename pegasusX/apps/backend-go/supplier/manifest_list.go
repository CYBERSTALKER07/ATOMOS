package supplier

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
)

func portalStatusFromManifestState(state string) string {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "LOADING", "SEALED":
		return "LOADING"
	case "DISPATCHED", "COMPLETED":
		return "DISPATCHED"
	default:
		return "DRAFT"
	}
}

func volumeVuToInt64(volume float64) int64 {
	if volume <= 0 {
		return 0
	}
	return int64(math.Round(volume))
}

func (s *Service) listSupplierManifests(ctx context.Context, supplierID string) ([]SupplierManifestRow, error) {
	if s.portalSpanner != nil {
		rows, err := s.listSupplierManifestsFromStore(ctx, supplierID)
		if err != nil {
			return nil, err
		}
		if bucket, ok, err := s.unassignedManifestBucket(ctx, supplierID); err != nil {
			return nil, err
		} else if ok {
			rows = append(rows, bucket)
		}
		return rows, nil
	}
	orders, err := s.listSupplierOrders(ctx, supplierID, "", "")
	if err != nil {
		return nil, err
	}
	return s.aggregateManifestsLegacy(ctx, supplierID, orders)
}

func (s *Service) listSupplierManifestsFromStore(ctx context.Context, supplierID string) ([]SupplierManifestRow, error) {
	store := s.manifestStore
	if store == nil {
		store = manifest.NewStore(s.portalSpanner)
	}
	manifests, err := store.ListSupplierManifests(ctx, supplierID)
	if err != nil {
		return nil, fmt.Errorf("list supplier truck manifests: %w", err)
	}

	driverNames, vehiclePlates := s.manifestFleetLabels(ctx, supplierID)

	rows := make([]SupplierManifestRow, 0, len(manifests))
	for _, row := range manifests {
		state := strings.ToUpper(strings.TrimSpace(row.State))
		status := portalStatusFromManifestState(state)
		vehicleID := strings.TrimSpace(row.TruckID)
		driverID := strings.TrimSpace(row.DriverID)
		driverName := driverNames[driverID]
		if driverName == "" && driverID != "" {
			driverName = driverID
		}
		if driverName == "" {
			driverName = "Unassigned"
		}
		rows = append(rows, SupplierManifestRow{
			ManifestID:    row.ManifestID,
			Status:        status,
			State:         state,
			OrdersCount:   int(row.StopCount),
			StopCount:     int(row.StopCount),
			DriverID:      driverID,
			DriverName:    driverName,
			VehicleID:     vehicleID,
			VehiclePlate:  vehiclePlates[vehicleID],
			TruckID:       vehicleID,
			TotalVu:       volumeVuToInt64(row.TotalVolumeVU),
			TotalVolumeVU: row.TotalVolumeVU,
			MaxVolumeVU:   row.MaxVolumeVU,
			UpdatedAt:     row.UpdatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].UpdatedAt > rows[j].UpdatedAt })
	return rows, nil
}

func (s *Service) manifestFleetLabels(ctx context.Context, supplierID string) (map[string]string, map[string]string) {
	driverNames := map[string]string{}
	if drivers, err := s.repo.ListFleetDrivers(ctx, supplierID); err == nil {
		for _, driver := range drivers {
			driverNames[driver.DriverID] = strings.TrimSpace(driver.Name)
		}
	}
	vehiclePlates := map[string]string{}
	if vehicles, err := s.repo.ListFleetVehicles(ctx, supplierID); err == nil {
		for _, vehicle := range vehicles {
			vehiclePlates[vehicle.VehicleID] = strings.TrimSpace(vehicle.LicensePlate)
		}
	}
	return driverNames, vehiclePlates
}

func (s *Service) unassignedManifestBucket(ctx context.Context, supplierID string) (SupplierManifestRow, bool, error) {
	orders, err := s.listSupplierOrders(ctx, supplierID, "", "")
	if err != nil {
		return SupplierManifestRow{}, false, err
	}
	unassigned := make([]SupplierOrder, 0)
	for _, order := range orders {
		if strings.TrimSpace(order.ManifestID) == "" {
			unassigned = append(unassigned, order)
		}
	}
	if len(unassigned) == 0 {
		return SupplierManifestRow{}, false, nil
	}
	return SupplierManifestRow{
		ManifestID:  "unassigned",
		Status:      "DRAFT",
		State:       "DRAFT",
		OrdersCount: len(unassigned),
		StopCount:   len(unassigned),
		DriverName:  "Unassigned",
		UpdatedAt:   unassigned[0].UpdatedAt,
	}, true, nil
}

func supplierManifestToWire(row SupplierManifestRow) manifest.Wire {
	vehicleID := strings.TrimSpace(row.VehicleID)
	driverName := strings.TrimSpace(row.DriverName)
	if driverName == "" {
		driverName = "Unassigned"
	}
	state := strings.TrimSpace(row.State)
	if state == "" {
		state = row.Status
	}
	return manifest.Wire{
		ManifestID:    row.ManifestID,
		Status:        row.Status,
		State:         state,
		OrdersCount:   row.OrdersCount,
		DriverID:      row.DriverID,
		DriverName:    driverName,
		VehiclePlate:  row.VehiclePlate,
		TruckID:       vehicleID,
		VehicleID:     vehicleID,
		TotalVu:       row.TotalVu,
		TotalVolumeVU: row.TotalVolumeVU,
		MaxVolumeVU:   row.MaxVolumeVU,
		StopCount:     row.StopCount,
		UpdatedAt:     row.UpdatedAt,
	}
}
