package warehouse

import "strings"

// Warehouse vehicle unavailable reason codes (admin).
const (
	VehicleReasonMaintenance    = "MAINTENANCE"
	VehicleReasonTruckDamaged   = "TRUCK_DAMAGED"
	VehicleReasonRegulatoryHold = "REGULATORY_HOLD"
	VehicleReasonManualHold     = "MANUAL_HOLD"
	VehicleReasonOther          = "OTHER"
)

var warehouseVehicleReasons = map[string]struct{}{
	VehicleReasonMaintenance:    {},
	VehicleReasonTruckDamaged:   {},
	VehicleReasonRegulatoryHold: {},
	VehicleReasonManualHold:     {},
	VehicleReasonOther:          {},
}

func normalizeWarehouseVehicleReason(reason string) string {
	r := strings.ToUpper(strings.TrimSpace(reason))
	if r == "" {
		return VehicleReasonManualHold
	}
	if _, ok := warehouseVehicleReasons[r]; ok {
		return r
	}
	return VehicleReasonOther
}

func vehicleUnavailableDisplayReason(reason, note string) string {
	reason = strings.TrimSpace(reason)
	note = strings.TrimSpace(note)
	if reason == VehicleReasonOther && note != "" {
		return note
	}
	return reason
}

func driverOffShiftTruckStatus(reason string) string {
	if strings.EqualFold(strings.TrimSpace(reason), "RETURNING_TO_WAREHOUSE") {
		return "RETURNING_TO_WAREHOUSE"
	}
	return "OFF_SHIFT"
}

func driverUnavailableDisplayReason(reason, note string) string {
	reason = strings.TrimSpace(reason)
	note = strings.TrimSpace(note)
	if reason == "" {
		return "OFF_SHIFT"
	}
	if strings.EqualFold(reason, "OTHER") && note != "" {
		return note
	}
	return reason
}
