package warehouse

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

var (
	errDriverNotFound  = errors.New("driver_not_found")
	errVehicleNotFound = errors.New("vehicle_not_found")
)

// FleetMutationError carries an HTTP status for fleet assignment guards.
type FleetMutationError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *FleetMutationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type driverAssignmentState struct {
	DriverID     string
	SupplierID   string
	HomeNodeType string
	HomeNodeID   string
	VehicleID    string
}

func readDriverAssignmentState(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID, driverID string) (driverAssignmentState, error) {
	row, err := txn.ReadRow(ctx, "Drivers", spanner.Key{driverID},
		[]string{"SupplierId", "HomeNodeType", "HomeNodeId", "VehicleId"})
	if err != nil {
		return driverAssignmentState{}, errDriverNotFound
	}
	var state driverAssignmentState
	var homeNodeType, homeNodeID, vehicleID spanner.NullString
	if err := row.Columns(&state.SupplierID, &homeNodeType, &homeNodeID, &vehicleID); err != nil {
		return driverAssignmentState{}, err
	}
	state.DriverID = driverID
	state.HomeNodeType = strings.TrimSpace(homeNodeType.StringVal)
	state.HomeNodeID = strings.TrimSpace(homeNodeID.StringVal)
	state.VehicleID = strings.TrimSpace(vehicleID.StringVal)
	if state.SupplierID != supplierID {
		return driverAssignmentState{}, errDriverNotFound
	}
	if warehouseID != "" && (!strings.EqualFold(state.HomeNodeType, "WAREHOUSE") || state.HomeNodeID != warehouseID) {
		return driverAssignmentState{}, errDriverNotFound
	}
	return state, nil
}

func driverAssignmentGuard(state driverAssignmentState, activeOrders int64) error {
	if activeOrders > 0 {
		return &FleetMutationError{
			StatusCode: http.StatusConflict,
			Code:       "driver_active_orders",
			Message:    fmt.Sprintf("driver %s has %d active orders and cannot change vehicle assignment", state.DriverID, activeOrders),
		}
	}
	return nil
}

func countActiveOrdersForDriver(ctx context.Context, txn *spanner.ReadWriteTransaction, driverID string) (int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM Orders
		      WHERE DriverId = @driverId
		        AND Status IN ('LOADED','IN_TRANSIT','ARRIVED','DISPATCHED','ARRIVING','EN_ROUTE','AWAITING_PAYMENT','PENDING_CASH_COLLECTION')`,
		Params: map[string]any{"driverId": driverID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, err
	}
	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func countActiveOrdersForVehicle(ctx context.Context, txn *spanner.ReadWriteTransaction, vehicleID string) (int64, error) {
	if vehicleID == "" {
		return 0, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM Orders
		      WHERE VehicleId = @vehicleId
		        AND Status IN ('LOADED','IN_TRANSIT','ARRIVED','DISPATCHED','ARRIVING','EN_ROUTE','AWAITING_PAYMENT','PENDING_CASH_COLLECTION')`,
		Params: map[string]any{"vehicleId": vehicleID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, err
	}
	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func readDriverByVehicle(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID, vehicleID, excludeDriverID string) (*driverAssignmentState, error) {
	stmt := spanner.Statement{
		SQL: `SELECT DriverId, SupplierId, COALESCE(HomeNodeType,''), COALESCE(HomeNodeId,''), COALESCE(VehicleId,'')
		      FROM Drivers
		      WHERE SupplierId = @sid AND VehicleId = @vid
		        AND HomeNodeType = 'WAREHOUSE' AND HomeNodeId = @wid
		        AND DriverId != @exclude
		      LIMIT 1`,
		Params: map[string]any{
			"sid":     supplierID,
			"vid":     vehicleID,
			"wid":     warehouseID,
			"exclude": excludeDriverID,
		},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var state driverAssignmentState
	if err := row.Columns(&state.DriverID, &state.SupplierID, &state.HomeNodeType, &state.HomeNodeID, &state.VehicleID); err != nil {
		return nil, err
	}
	return &state, nil
}

func vehicleClassMaxVU(class string) float64 {
	switch strings.ToUpper(strings.TrimSpace(class)) {
	case "CLASS_A":
		return 50
	case "CLASS_C":
		return 400
	default:
		return 150
	}
}

func resolveVehicleMaxVU(class string, explicit float64) float64 {
	if explicit > 0 {
		return explicit
	}
	return vehicleClassMaxVU(class)
}
