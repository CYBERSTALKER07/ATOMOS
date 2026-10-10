package twin

import "time"

type RouteTwin struct {
	RouteID            string    `json:"route_id" spanner:"RouteId"`
	SupplierID         string    `json:"supplier_id" spanner:"SupplierId"`
	DriverID           string    `json:"driver_id" spanner:"DriverId"`
	Status             string    `json:"status" spanner:"Status"`
	CurrentLat         float64   `json:"current_lat" spanner:"CurrentLat"`
	CurrentLng         float64   `json:"current_lng" spanner:"CurrentLng"`
	CurrentH3          string    `json:"current_h3" spanner:"CurrentH3"`
	LocationAt         time.Time `json:"location_at" spanner:"LocationAt"`
	RemainingStops     int64     `json:"remaining_stops" spanner:"RemainingStops"`
	CapacityUsedWeight float64   `json:"capacity_used_weight" spanner:"CapacityUsedWeight"`
	CapacityUsedVolume float64   `json:"capacity_used_volume" spanner:"CapacityUsedVolume"`
	LastEventAt        time.Time `json:"last_event_at" spanner:"LastEventAt"`
	UpdatedAt          time.Time `json:"updated_at" spanner:"UpdatedAt"`
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
