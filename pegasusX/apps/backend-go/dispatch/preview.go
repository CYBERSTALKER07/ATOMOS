package dispatch

import "strings"

// BuildPreview maps dispatchable orders into portal wire rows and GeoOrders
// for downstream optimiser handoff. Orders are sorted by receiving-window urgency.
func BuildPreview(orders []DispatchableOrder) Preview {
	if len(orders) == 0 {
		return Preview{
			UndispatchedOrders: []map[string]any{},
			GeoOrders:          []GeoOrder{},
		}
	}

	sorted := append([]DispatchableOrder(nil), orders...)
	SortByWindowUrgency(sorted)

	wire := make([]map[string]any, 0, len(sorted))
	geo := make([]GeoOrder, 0, len(sorted))
	constrained := 0

	for _, order := range sorted {
		hasWindow := HasReceivingWindow(order.ReceivingWindowOpen, order.ReceivingWindowClose)
		if hasWindow {
			constrained++
		}
		retailerLabel := strings.TrimSpace(order.RetailerName)
		if retailerLabel == "" {
			retailerLabel = "Retailer " + order.RetailerID
		}
		wire = append(wire, map[string]any{
			"order_id":               order.OrderID,
			"retailer_id":            order.RetailerID,
			"retailer_name":          retailerLabel,
			"warehouse_id":           order.WarehouseID,
			"status":                 order.Status,
			"total_minor":            order.TotalMinor,
			"currency":               order.Currency,
			"receiving_window_open":  order.ReceivingWindowOpen,
			"receiving_window_close": order.ReceivingWindowClose,
			"has_receiving_window":   hasWindow,
			"volume_vu":              order.VolumeVU,
		})
		geo = append(geo, GeoOrder{
			OrderID:              order.OrderID,
			RetailerID:           order.RetailerID,
			RetailerName:         retailerLabel,
			Amount:               order.TotalMinor,
			Lat:                  order.Lat,
			Lng:                  order.Lng,
			Volume:               order.VolumeVU,
			ReceivingWindowOpen:  order.ReceivingWindowOpen,
			ReceivingWindowClose: order.ReceivingWindowClose,
			HandlingClass:        order.HandlingClass,
			RequiresColdChain:    order.RequiresColdChain,
			IsHazardous:          order.IsHazardous,
			AccessRestriction:    order.AccessRestriction,
		})
	}

	return Preview{
		UndispatchedOrders: wire,
		GeoOrders:          geo,
		WindowConstrained:  constrained,
	}
}
