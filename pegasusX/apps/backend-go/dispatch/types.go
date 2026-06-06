package dispatch

// GeoOrder is the atomic routing unit consumed by optimiser clients.
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
	IgnoreCapacity       bool
	IsRecovery           bool
	ReceivingWindowOpen  string
	ReceivingWindowClose string
}

// DispatchableOrder is the Spanner-backed dispatch candidate row.
type DispatchableOrder struct {
	OrderID              string
	RetailerID           string
	RetailerName         string
	WarehouseID          string
	Status               string
	TotalMinor           int64
	Currency             string
	H3Cell               string
	Lat                  float64
	Lng                  float64
	VolumeVU             float64
	IgnoreCapacity       bool
	IsRecovery           bool
	ReceivingWindowOpen  string
	ReceivingWindowClose string
}

// DispatchRoute tracks a single truck's load state.
type DispatchRoute struct {
	DriverID     string
	MaxVolume    float64
	LoadedVolume float64
	Orders       []GeoOrder
}

// VehicleMatch is the result of SelectBestVehicle.
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

// AssignmentResult is the output of the bin-packing / optimiser pipeline.
type AssignmentResult struct {
	Routes   []DispatchRoute
	Splits   []SplitOrder
	Orphans  []GeoOrder
	Warnings []string
}

// SplitOrder records an order split across multiple trucks.
type SplitOrder struct {
	OriginalOrderID string
	TotalVolumeVU   float64
	Chunks          []OrderChunk
	Reason          string
}

// OrderChunk is a portion of a split order.
type OrderChunk struct {
	ChunkIndex int
	VolumeVU   float64
	TruckID    string
}

// FetchParams scopes a dispatchable-order query.
type FetchParams struct {
	SupplierID  string
	WarehouseID string
	FilterIDs   []string
	Limit       int
	StrongRead  bool
}

// Preview bundles dispatch planning rows for portal JSON responses.
type Preview struct {
	UndispatchedOrders []map[string]any
	GeoOrders          []GeoOrder
	WindowConstrained  int
}

// ToGeo converts a DispatchableOrder to a GeoOrder.
func (o DispatchableOrder) ToGeo() GeoOrder {
	return GeoOrder{
		OrderID:              o.OrderID,
		RetailerID:           o.RetailerID,
		RetailerName:         o.RetailerName,
		Amount:               o.TotalMinor,
		Lat:                  o.Lat,
		Lng:                  o.Lng,
		Volume:               o.VolumeVU,
		IsRecovery:           o.IsRecovery,
		ReceivingWindowOpen:  o.ReceivingWindowOpen,
		ReceivingWindowClose: o.ReceivingWindowClose,
	}
}
