package dispatch

// ─── Domain Types ─────────────────────────────────────────────────────────────
// These types are the shared vocabulary for the dispatch pipeline. They are
// consumed by supplier/, warehouse/, and factory/ domain packages.

// GeoOrder is the atomic unit — one order, one invoice, indivisible.
type GeoOrder struct {
	OrderID              string
	RetailerID           string
	RetailerName         string
	Amount               int64
	Lat                  float64
	Lng                  float64
	Volume               float64
	Assigned             bool
	ForceAssigned        bool
	CapacityOverflow     bool
	LogisticsIsolated    bool
	IgnoreCapacity       bool   // Manual payload override — skip volume check
	IsRecovery           bool   // Overflow-bounced; gets priority savings boost in solver
	ReceivingWindowOpen  string // "HH:MM" or ""
	ReceivingWindowClose string // "HH:MM" or ""
}

// DispatchRoute tracks a single truck's load state.
type DispatchRoute struct {
	DriverID     string
	MaxVolume    float64
	LoadedVolume float64
	Orders       []GeoOrder
}

// VehicleMatch is the result of SelectBestVehicle — contains the matched
// driver/vehicle and whether the order overflows the fleet's capacity.
type VehicleMatch struct {
	Driver   AvailableDriver
	Overflow bool
}

// AvailableDriver is the dispatch-relevant view of a driver+vehicle pair.
type AvailableDriver struct {
	DriverID     string
	DriverName   string
	VehicleID    string
	VehicleClass string
	MaxVolumeVU  float64
}

// AssignmentResult is the output of the bin-packing pipeline.
type AssignmentResult struct {
	Routes   []DispatchRoute
	Splits   []SplitOrder
	Orphans  []GeoOrder
	Warnings []string
}

// SplitOrder records an order whose TotalVolumeVU exceeded the largest
// available truck's effective capacity, requiring volumetric splitting.
type SplitOrder struct {
	OriginalOrderID string
	TotalVolumeVU   float64
	Chunks          []OrderChunk
	Reason          string
}

// OrderChunk is a portion of a split order that fits a single truck.
type OrderChunk struct {
	ChunkIndex int
	VolumeVU   float64
	TruckID    string
}

// ManifestChunk represents a single sub-manifest within a split group.
type ManifestChunk struct {
	RouteID  string
	Orders   []GeoOrder
	VolumeVU float64
	Suffix   string
}

// ManifestGroup wraps the result of splitting a single driver's orders
// into one or more route-legal manifest chunks.
type ManifestGroup struct {
	DriverID    string
	TruckID     string
	Chunks      []ManifestChunk
	TotalOrders int
}
