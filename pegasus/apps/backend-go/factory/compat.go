package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/spannerx"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type factoryDashboardCompatResponse struct {
	PendingTransfers  int64 `json:"pending_transfers"`
	LoadingTransfers  int64 `json:"loading_transfers"`
	ActiveManifests   int64 `json:"active_manifests"`
	DispatchedToday   int64 `json:"dispatched_today"`
	VehiclesTotal     int64 `json:"vehicles_total"`
	VehiclesAvailable int64 `json:"vehicles_available"`
	StaffOnShift      int64 `json:"staff_on_shift"`
	CriticalInsights  int64 `json:"critical_insights"`
}

type factoryFleetCompatResponse struct {
	Vehicles []factoryFleetCompatVehicle `json:"vehicles"`
}

type factoryFleetCompatVehicle struct {
	ID             string  `json:"id"`
	PlateNumber    string  `json:"plate_number"`
	CapacityM3     float64 `json:"capacity_m3"`
	CapacityKg     float64 `json:"capacity_kg"`
	CapacityL      float64 `json:"capacity_l"`
	Status         string  `json:"status"`
	DriverName     string  `json:"driver_name"`
	CurrentRouteID string  `json:"current_route_id"`
	CurrentRoute   string  `json:"current_route"`
}

type factoryDriverCompat struct {
	Name        string
	VehicleID   string
	TruckStatus string
}

type factoryManifestCompat struct {
	ManifestID string
	VehicleID  string
	State      string
	CreatedAt  time.Time
}

func HandleFactoryDashboardCompat(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		factoryID, ok := auth.MustFactoryID(w, r.Context())
		if !ok {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		supplierID, err := lookupFactorySupplierID(ctx, spannerClient, factoryID)
		if err != nil {
			log.Printf("[FACTORY COMPAT] dashboard supplier lookup failed: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		fleet, err := loadFactoryFleetCompat(ctx, spannerClient, factoryID, supplierID)
		if err != nil {
			log.Printf("[FACTORY COMPAT] dashboard fleet snapshot failed: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		resp := factoryDashboardCompatResponse{
			VehiclesTotal: int64(len(fleet)),
		}
		for _, vehicle := range fleet {
			if isFactoryVehicleAvailable(vehicle) {
				resp.VehiclesAvailable++
			}
		}

		if err := fillFactoryTransferStats(ctx, spannerClient, factoryID, &resp); err != nil {
			log.Printf("[FACTORY COMPAT] dashboard transfer stats failed: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if err := fillFactoryManifestStats(ctx, spannerClient, factoryID, &resp); err != nil {
			log.Printf("[FACTORY COMPAT] dashboard manifest stats failed: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if err := fillFactoryStaffStats(ctx, spannerClient, factoryID, &resp); err != nil {
			log.Printf("[FACTORY COMPAT] dashboard staff stats failed: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if err := fillFactoryInsightStats(ctx, spannerClient, factoryID, supplierID, &resp); err != nil {
			log.Printf("[FACTORY COMPAT] dashboard insight stats failed: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func HandleFactoryFleetCompat(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		factoryID, ok := auth.MustFactoryID(w, r.Context())
		if !ok {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		supplierID, err := lookupFactorySupplierID(ctx, spannerClient, factoryID)
		if err != nil {
			log.Printf("[FACTORY COMPAT] fleet supplier lookup failed: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		vehicles, err := loadFactoryFleetCompat(ctx, spannerClient, factoryID, supplierID)
		if err != nil {
			log.Printf("[FACTORY COMPAT] fleet snapshot failed: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(factoryFleetCompatResponse{Vehicles: vehicles})
	}
}

func lookupFactorySupplierID(ctx context.Context, spannerClient *spanner.Client, factoryID string) (string, error) {
	row, err := spannerx.StaleReadRow(ctx, spannerClient, "Factories", spanner.Key{factoryID}, []string{"SupplierId"})
	if err != nil {
		return "", fmt.Errorf("read factory %s supplier: %w", factoryID, err)
	}
	var supplierID spanner.NullString
	if err := row.Columns(&supplierID); err != nil {
		return "", fmt.Errorf("scan factory %s supplier: %w", factoryID, err)
	}
	if !supplierID.Valid || supplierID.StringVal == "" {
		return "", fmt.Errorf("factory %s missing supplier id", factoryID)
	}
	return supplierID.StringVal, nil
}

func fillFactoryTransferStats(ctx context.Context, spannerClient *spanner.Client, factoryID string, resp *factoryDashboardCompatResponse) error {
	stmt := spanner.Statement{
		SQL: `SELECT
				COUNTIF(State IN ('DRAFT', 'APPROVED')),
				COUNTIF(State = 'LOADING'),
				COUNTIF(State = 'DISPATCHED' AND DATE(COALESCE(UpdatedAt, CreatedAt)) = CURRENT_DATE())
			  FROM InternalTransferOrders@{FORCE_INDEX=Idx_Transfers_ByFactoryId}
			  WHERE FactoryId = @factoryId`,
		Params: map[string]interface{}{"factoryId": factoryID},
	}
	iter := spannerx.StaleQuery(ctx, spannerClient, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return fmt.Errorf("query transfer stats: %w", err)
	}
	if err := row.Columns(&resp.PendingTransfers, &resp.LoadingTransfers, &resp.DispatchedToday); err != nil {
		return fmt.Errorf("scan transfer stats: %w", err)
	}
	return nil
}

func fillFactoryManifestStats(ctx context.Context, spannerClient *spanner.Client, factoryID string, resp *factoryDashboardCompatResponse) error {
	stmt := spanner.Statement{
		SQL: `SELECT COUNTIF(State IN ('READY_FOR_LOADING', 'LOADING', 'DISPATCHED'))
			  FROM FactoryTruckManifests@{FORCE_INDEX=Idx_FactoryManifests_ByFactoryId}
			  WHERE FactoryId = @factoryId`,
		Params: map[string]interface{}{"factoryId": factoryID},
	}
	iter := spannerx.StaleQuery(ctx, spannerClient, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return fmt.Errorf("query manifest stats: %w", err)
	}
	if err := row.Columns(&resp.ActiveManifests); err != nil {
		return fmt.Errorf("scan manifest stats: %w", err)
	}
	return nil
}

func fillFactoryStaffStats(ctx context.Context, spannerClient *spanner.Client, factoryID string, resp *factoryDashboardCompatResponse) error {
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*)
			  FROM FactoryStaff@{FORCE_INDEX=Idx_FactoryStaff_ByFactoryId}
			  WHERE FactoryId = @factoryId AND IsActive = true`,
		Params: map[string]interface{}{"factoryId": factoryID},
	}
	iter := spannerx.StaleQuery(ctx, spannerClient, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return fmt.Errorf("query staff stats: %w", err)
	}
	if err := row.Columns(&resp.StaffOnShift); err != nil {
		return fmt.Errorf("scan staff stats: %w", err)
	}
	return nil
}

func fillFactoryInsightStats(ctx context.Context, spannerClient *spanner.Client, factoryID, supplierID string, resp *factoryDashboardCompatResponse) error {
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*)
			  FROM ReplenishmentInsights@{FORCE_INDEX=Idx_Insights_BySupplierId}
			  WHERE SupplierId = @supplierId
			    AND Status = 'PENDING'
			    AND UrgencyLevel = 'CRITICAL'
			    AND (TargetFactoryId = @factoryId OR TargetFactoryId IS NULL)`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
			"factoryId":  factoryID,
		},
	}
	iter := spannerx.StaleQuery(ctx, spannerClient, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return fmt.Errorf("query insight stats: %w", err)
	}
	if err := row.Columns(&resp.CriticalInsights); err != nil {
		return fmt.Errorf("scan insight stats: %w", err)
	}
	return nil
}

func loadFactoryFleetCompat(ctx context.Context, spannerClient *spanner.Client, factoryID, supplierID string) ([]factoryFleetCompatVehicle, error) {
	vehicles, err := readFactoryCompatVehicles(ctx, spannerClient, supplierID)
	if err != nil {
		return nil, err
	}
	driversByVehicle, err := readFactoryCompatDrivers(ctx, spannerClient, supplierID)
	if err != nil {
		return nil, err
	}
	manifestsByVehicle, err := readFactoryCompatManifests(ctx, spannerClient, factoryID)
	if err != nil {
		return nil, err
	}

	result := make([]factoryFleetCompatVehicle, 0, len(vehicles))
	for _, vehicle := range vehicles {
		driver := driversByVehicle[vehicle.ID]
		manifest := manifestsByVehicle[vehicle.ID]

		status := "AVAILABLE"
		switch {
		case !vehicle.Active:
			status = "INACTIVE"
		case driver.TruckStatus != "" && driver.TruckStatus != "AVAILABLE":
			status = driver.TruckStatus
		case manifest.State != "":
			status = manifest.State
		}

		result = append(result, factoryFleetCompatVehicle{
			ID:             vehicle.ID,
			PlateNumber:    vehicle.PlateNumber,
			CapacityM3:     vehicle.Capacity,
			CapacityKg:     vehicle.Capacity,
			CapacityL:      vehicle.Capacity,
			Status:         status,
			DriverName:     driver.Name,
			CurrentRouteID: manifest.ManifestID,
			CurrentRoute:   manifest.ManifestID,
		})
	}

	return result, nil
}

type factoryVehicleCompat struct {
	ID          string
	PlateNumber string
	Capacity    float64
	Active      bool
}

func readFactoryCompatVehicles(ctx context.Context, spannerClient *spanner.Client, supplierID string) ([]factoryVehicleCompat, error) {
	stmt := spanner.Statement{
		SQL: `SELECT VehicleId, COALESCE(LicensePlate, ''), MaxVolumeVU, IsActive
			  FROM Vehicles@{FORCE_INDEX=Idx_Vehicles_BySupplier}
			  WHERE SupplierId = @supplierId
			  ORDER BY CreatedAt DESC`,
		Params: map[string]interface{}{"supplierId": supplierID},
	}
	iter := spannerx.StaleQuery(ctx, spannerClient, stmt)
	defer iter.Stop()

	vehicles := []factoryVehicleCompat{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query vehicles: %w", err)
		}
		var vehicle factoryVehicleCompat
		if err := row.Columns(&vehicle.ID, &vehicle.PlateNumber, &vehicle.Capacity, &vehicle.Active); err != nil {
			return nil, fmt.Errorf("scan vehicle: %w", err)
		}
		vehicles = append(vehicles, vehicle)
	}
	return vehicles, nil
}

func readFactoryCompatDrivers(ctx context.Context, spannerClient *spanner.Client, supplierID string) (map[string]factoryDriverCompat, error) {
	stmt := spanner.Statement{
		SQL: `SELECT Name, COALESCE(VehicleId, ''), COALESCE(TruckStatus, 'AVAILABLE')
			  FROM Drivers@{FORCE_INDEX=Idx_Drivers_BySupplierId}
			  WHERE SupplierId = @supplierId AND IsActive = true`,
		Params: map[string]interface{}{"supplierId": supplierID},
	}
	iter := spannerx.StaleQuery(ctx, spannerClient, stmt)
	defer iter.Stop()

	drivers := make(map[string]factoryDriverCompat)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query drivers: %w", err)
		}
		var driver factoryDriverCompat
		if err := row.Columns(&driver.Name, &driver.VehicleID, &driver.TruckStatus); err != nil {
			return nil, fmt.Errorf("scan driver: %w", err)
		}
		if driver.VehicleID == "" {
			continue
		}
		if _, exists := drivers[driver.VehicleID]; !exists {
			drivers[driver.VehicleID] = driver
		}
	}
	return drivers, nil
}

func readFactoryCompatManifests(ctx context.Context, spannerClient *spanner.Client, factoryID string) (map[string]factoryManifestCompat, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId, COALESCE(VehicleId, ''), State, CreatedAt
			  FROM FactoryTruckManifests@{FORCE_INDEX=Idx_FactoryManifests_ByFactoryId}
			  WHERE FactoryId = @factoryId AND State IN ('READY_FOR_LOADING', 'LOADING', 'DISPATCHED')
			  ORDER BY CreatedAt DESC`,
		Params: map[string]interface{}{"factoryId": factoryID},
	}
	iter := spannerx.StaleQuery(ctx, spannerClient, stmt)
	defer iter.Stop()

	manifests := make(map[string]factoryManifestCompat)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query manifests: %w", err)
		}
		var manifest factoryManifestCompat
		if err := row.Columns(&manifest.ManifestID, &manifest.VehicleID, &manifest.State, &manifest.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan manifest: %w", err)
		}
		if manifest.VehicleID == "" {
			continue
		}
		if _, exists := manifests[manifest.VehicleID]; !exists {
			manifests[manifest.VehicleID] = manifest
		}
	}
	return manifests, nil
}

func isFactoryVehicleAvailable(vehicle factoryFleetCompatVehicle) bool {
	return vehicle.Status == "AVAILABLE" || vehicle.Status == "RETURNING"
}
