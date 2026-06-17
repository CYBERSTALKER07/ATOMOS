package warehouse

import "strings"

func portalVehicleStatus(isActive bool) string {
	if isActive {
		return "ACTIVE"
	}
	return "UNAVAILABLE"
}

func portalVehicleCapacityVU(v PortalVehicle) float64 {
	if v.MaxVolumeVU > 0 {
		return v.MaxVolumeVU
	}
	return resolveVehicleMaxVU(v.VehicleClass, 0)
}

func wirePortalVehicle(v PortalVehicle) map[string]any {
	capacity := portalVehicleCapacityVU(v)
	out := map[string]any{
		"vehicle_id":    v.VehicleID,
		"label":         v.Label,
		"license_plate": v.LicensePlate,
		"vehicle_class": v.VehicleClass,
		"max_volume_vu": capacity,
		"capacity_vu":   capacity,
		"is_active":     v.IsActive,
		"status":        portalVehicleStatus(v.IsActive),
	}
	if strings.TrimSpace(v.UnavailableReason) != "" {
		out["unavailable_reason"] = v.UnavailableReason
	}
	if strings.TrimSpace(v.UnavailableNote) != "" {
		out["unavailable_note"] = v.UnavailableNote
	}
	if strings.TrimSpace(v.AssignedDriverID) != "" {
		out["assigned_driver_id"] = v.AssignedDriverID
	}
	if strings.TrimSpace(v.AssignedDriverName) != "" {
		out["assigned_driver_name"] = v.AssignedDriverName
	}
	return out
}

func wirePortalVehicles(vehicles []PortalVehicle) []map[string]any {
	out := make([]map[string]any, 0, len(vehicles))
	for _, v := range vehicles {
		out = append(out, wirePortalVehicle(v))
	}
	return out
}
