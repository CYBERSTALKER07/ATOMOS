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
	// SplitGroupID is set when this order belongs to a multi-truck split group.
	SplitGroupID string
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
	// SplitGroupID is non-empty when this route is one leg of a split-retailer delivery.
	SplitGroupID string
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
	// OverflowWarnings lists retailer orders whose combined volume exceeds any
	// single truck. The warehouse admin must decide to split across trucks or
	// cancel individual orders before dispatch can commit.
	OverflowWarnings []RetailerOverflowWarning
	// SplitShipmentGroups is populated by dispatch_execute after commit; it
	// describes which manifests share a payment-coordination group.
	SplitShipmentGroups []SplitShipmentGroup
}

// RetailerOverflowWarning is surfaced to the warehouse admin when a retailer's
// consolidated order volume exceeds the maximum single-truck effective capacity.
// The UI must present this before allowing dispatch, giving the admin the choice
// to split the load across multiple trucks or cancel orders from the list.
type RetailerOverflowWarning struct {
	RetailerID    string   `json:"retailer_id"`
	RetailerName  string   `json:"retailer_name"`
	OrderIDs      []string `json:"order_ids"`
	TotalVolumeVU float64  `json:"total_volume_vu"`
	MaxTruckVU    float64  `json:"max_truck_vu"`
	ExcessVU      float64  `json:"excess_vu"`
	// SplitRequired is true — the admin must act before dispatch proceeds.
	SplitRequired bool `json:"split_required"`
}

// SplitShipmentGroup describes a single retailer's order set that was split
// across multiple trucks by the warehouse admin. All trucks in a group share:
//   - The same canonical SharedRouteID (driver/retailer apps receive one route)
//   - The same order list visible to every driver in the group
//   - Payment coordination: the system accepts exactly one payment event
//     (cash or card) and marks the order complete for all drivers in the group.
type SplitShipmentGroup struct {
	SplitGroupID  string   `json:"split_group_id"`
	RetailerID    string   `json:"retailer_id"`
	OrderIDs      []string `json:"order_ids"`
	DriverIDs     []string `json:"driver_ids"`
	ManifestIDs   []string `json:"manifest_ids"`
	SharedRouteID string   `json:"shared_route_id"`
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
	Offset      int
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
