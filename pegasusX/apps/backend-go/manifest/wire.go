// Package manifest defines the unified manifest JSON contract shared by the
// supplier portal (order projection) and payloader surfaces (loading gate).
package manifest

import "strings"

// Wire is the additive manifest DTO every list/detail endpoint emits.
// Portal clients read status/orders_count/driver_name; tablet clients read
// state/truck_id/total_volume_vu/orders. Both field sets are always present.
type Wire struct {
	ManifestID    string           `json:"manifest_id"`
	Status        string           `json:"status"`
	State         string           `json:"state"`
	OrdersCount   int              `json:"orders_count"`
	DriverID      string           `json:"driver_id,omitempty"`
	DriverName    string           `json:"driver_name"`
	VehiclePlate  string           `json:"vehicle_plate,omitempty"`
	TruckID       string           `json:"truck_id"`
	VehicleID     string           `json:"vehicle_id,omitempty"`
	TotalVu       int64            `json:"total_vu"`
	TotalVolumeVU float64          `json:"total_volume_vu"`
	MaxVolumeVU   float64          `json:"max_volume_vu"`
	StopCount     int              `json:"stop_count"`
	RegionCode    string           `json:"region_code,omitempty"`
	SealedAt      string           `json:"sealed_at,omitempty"`
	DispatchedAt  string           `json:"dispatched_at,omitempty"`
	CreatedAt     string           `json:"created_at,omitempty"`
	UpdatedAt     string           `json:"updated_at,omitempty"`
	Orders        []OrderWire      `json:"orders,omitempty"`
	OverflowCount int              `json:"overflow_count"`
}

// OrderWire is the per-stop order shape embedded in manifest detail responses.
type OrderWire struct {
	OrderID        string          `json:"order_id"`
	RetailerID     string          `json:"retailer_id,omitempty"`
	Amount         int64           `json:"amount"`
	PaymentGateway string          `json:"payment_gateway,omitempty"`
	State          string          `json:"state"`
	Status         string          `json:"status"`
	RouteID        string          `json:"route_id,omitempty"`
	WarehouseID    string          `json:"warehouse_id,omitempty"`
	Items          []OrderItemWire `json:"items"`
}

// OrderItemWire is one catalog line on a manifest order.
type OrderItemWire struct {
	LineItemID string `json:"line_item_id"`
	SkuID      string `json:"sku_id"`
	SkuName    string `json:"sku_name"`
	Quantity   int    `json:"quantity"`
	UnitPrice  int64  `json:"unit_price"`
	Status     string `json:"status"`
}

// PortalRow is the supplier order-aggregation projection input.
type PortalRow struct {
	ManifestID   string
	Status       string
	OrdersCount  int
	DriverID     string
	DriverName   string
	VehiclePlate string
	VehicleID    string
	TotalVu      int64
	UpdatedAt    string
}

// PayloadRow is the payloader operational manifest input.
type PayloadRow struct {
	ManifestID       string
	VehicleID        string
	DriverID         string
	State            string
	TotalVolumeVU    int64
	MaxVolumeVU      int64
	StopCount        int
	RegionCode       string
	SealedAt         string
	DispatchedAt     string
	CreatedAt        string
	UpdatedAt        string
	OverflowCount    int
	Orders           []PayloadOrderRow
	DriverName       string
	VehiclePlate     string
}

// PayloadOrderRow is one manifest-order join row for detail hydration.
type PayloadOrderRow struct {
	OrderID      string
	State        string
	Amount       int64
	RouteID      string
	WarehouseID  string
	RetailerID   string
	LineItemID   string
	SkuID        string
	SkuName      string
	Quantity     int
	UnitPrice    int64
}

// FromPortalRow maps a supplier portal projection into the unified wire shape.
func FromPortalRow(row PortalRow) Wire {
	status := normalizeStatus(row.Status)
	vehicleID := strings.TrimSpace(row.VehicleID)
	return Wire{
		ManifestID:    row.ManifestID,
		Status:        status,
		State:         status,
		OrdersCount:   row.OrdersCount,
		DriverID:      row.DriverID,
		DriverName:    row.DriverName,
		VehiclePlate:  row.VehiclePlate,
		TruckID:       vehicleID,
		VehicleID:     vehicleID,
		TotalVu:       row.TotalVu,
		TotalVolumeVU: float64(row.TotalVu),
		StopCount:     row.OrdersCount,
		UpdatedAt:     row.UpdatedAt,
		Orders:        nil,
	}
}

// FromPayloadRow maps payloader store rows into the unified wire shape.
func FromPayloadRow(row PayloadRow) Wire {
	status := normalizeStatus(row.State)
	driverName := strings.TrimSpace(row.DriverName)
	if driverName == "" {
		driverName = strings.TrimSpace(row.DriverID)
	}
	if driverName == "" {
		driverName = "Unassigned"
	}
	plate := strings.TrimSpace(row.VehiclePlate)
	vehicleID := strings.TrimSpace(row.VehicleID)
	orders := make([]OrderWire, 0, len(row.Orders))
	for i := range row.Orders {
		orders = append(orders, orderFromPayload(row.Orders[i]))
	}
	return Wire{
		ManifestID:    row.ManifestID,
		Status:        status,
		State:         status,
		OrdersCount:   row.StopCount,
		DriverID:      row.DriverID,
		DriverName:    driverName,
		VehiclePlate:  plate,
		TruckID:       vehicleID,
		VehicleID:     vehicleID,
		TotalVu:       row.TotalVolumeVU,
		TotalVolumeVU: float64(row.TotalVolumeVU),
		MaxVolumeVU:   float64(row.MaxVolumeVU),
		StopCount:     row.StopCount,
		RegionCode:    row.RegionCode,
		SealedAt:      row.SealedAt,
		DispatchedAt:  row.DispatchedAt,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		Orders:        orders,
		OverflowCount: row.OverflowCount,
	}
}

func orderFromPayload(row PayloadOrderRow) OrderWire {
	state := normalizeStatus(row.State)
	return OrderWire{
		OrderID: row.OrderID,
		RetailerID: row.RetailerID,
		Amount:  row.Amount,
		State:   state,
		Status:  state,
		RouteID: row.RouteID,
		WarehouseID: row.WarehouseID,
		Items: []OrderItemWire{{
			LineItemID: coalesce(row.LineItemID, row.OrderID+"-line-1"),
			SkuID:      coalesce(row.SkuID, "SKU-DEMO"),
			SkuName:    coalesce(row.SkuName, "Demo SKU"),
			Quantity:   max1(row.Quantity),
			UnitPrice:  row.UnitPrice,
			Status:     state,
		}},
	}
}

func normalizeStatus(raw string) string {
	s := strings.ToUpper(strings.TrimSpace(raw))
	switch s {
	case "", "ASSIGNED":
		return "DRAFT"
	default:
		return s
	}
}

func coalesce(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func max1(v int) int {
	if v <= 0 {
		return 1
	}
	return v
}

// MatchesStateFilter returns true when filter is empty or equals status/state.
func MatchesStateFilter(w Wire, stateFilter string) bool {
	if stateFilter == "" {
		return true
	}
	return strings.ToUpper(strings.TrimSpace(w.State)) == stateFilter
}

// MatchesTruckFilter returns true when filter is empty or matches truck/vehicle id.
func MatchesTruckFilter(w Wire, truckFilter string) bool {
	if truckFilter == "" {
		return true
	}
	truckFilter = strings.TrimSpace(truckFilter)
	return w.TruckID == truckFilter || w.VehicleID == truckFilter
}
