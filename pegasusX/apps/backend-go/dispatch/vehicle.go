package dispatch

import "strings"

const (
	VehicleClassA = "CLASS_A"
	VehicleClassB = "CLASS_B"
	VehicleClassC = "CLASS_C"
)

// VehicleSpec is per-truck capacity metadata used by dispatch hydration.
type VehicleSpec struct {
	VehicleClass string
	MaxVolumeVU  float64
}

// VolumeVUForClass returns the canonical capacity for a vehicle class tier.
func VolumeVUForClass(class string) float64 {
	switch strings.ToUpper(strings.TrimSpace(class)) {
	case VehicleClassA:
		return 50.0
	case VehicleClassC:
		return 400.0
	default:
		return DefaultTruckVolumeVU
	}
}

// ResolveMaxVolumeVU prefers an explicit VU value, then class defaults.
func ResolveMaxVolumeVU(class string, explicit float64) float64 {
	if explicit > 0 {
		return explicit
	}
	return VolumeVUForClass(class)
}

// ResolveVehicleClass normalises empty input to CLASS_B.
func ResolveVehicleClass(class string) string {
	class = strings.ToUpper(strings.TrimSpace(class))
	if class == "" {
		return VehicleClassB
	}
	return class
}
