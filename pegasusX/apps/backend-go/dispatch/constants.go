package dispatch

const (
	// TetrisBuffer is the volumetric safety margin applied to truck capacity.
	TetrisBuffer = 0.95

	// DefaultTruckVolumeVU is the CLASS_B tier capacity (Transit Van).
	DefaultTruckVolumeVU = 150.0

	// H3DispatchResolution matches the ecosystem wire format (resolution 7).
	H3DispatchResolution = 7
)
