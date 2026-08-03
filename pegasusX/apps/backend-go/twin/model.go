package twin

import "time"

type RouteTwin struct {
	RouteID            string
	DriverID           string
	Status             string
	CurrentLat         float64
	CurrentLng         float64
	CurrentH3          string
	LocationAt         time.Time
	RemainingStops     int64
	CapacityUsedWeight float64
	CapacityUsedVolume float64
	LastEventAt        time.Time
	UpdatedAt          time.Time
}

type StopTwin struct {
	RouteID             string
	StopID              string // usually OrderId
	Sequence            int64
	Status              string
	PredictedArrival    *time.Time
	WindowStart         *time.Time
	WindowEnd           *time.Time
	DeliveredGrossMinor int64
	RemainingGrossMinor int64
	UpdatedAt           time.Time
}

type VehicleInventory struct {
	RouteID      string
	Sku          string
	QtyOnVehicle int64
	UpdatedAt    time.Time
}

type RouteTwinView struct {
	RouteTwin
	Stops     []StopTwin
	Inventory []VehicleInventory
}
