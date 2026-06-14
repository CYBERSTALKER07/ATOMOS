package dispatch

import "sort"

// SuggestOrdersToUnselect returns a minimal set of order IDs to remove from an
// overloaded route so loaded volume fits within the Tetris-buffered capacity.
func SuggestOrdersToUnselect(route DispatchRoute) (orderIDs []string, excessVU float64) {
	maxVU := route.MaxVolume
	if maxVU <= 0 {
		return nil, 0
	}
	effective := maxVU * TetrisBuffer
	if route.LoadedVolume <= effective {
		return nil, 0
	}
	excessVU = route.LoadedVolume - effective

	type scored struct {
		orderID string
		volume  float64
	}
	scoredOrders := make([]scored, 0, len(route.Orders))
	for _, order := range route.Orders {
		if order.OrderID == "" {
			continue
		}
		scoredOrders = append(scoredOrders, scored{orderID: order.OrderID, volume: order.Volume})
	}
	sort.Slice(scoredOrders, func(i, j int) bool {
		if scoredOrders[i].volume == scoredOrders[j].volume {
			return scoredOrders[i].orderID < scoredOrders[j].orderID
		}
		return scoredOrders[i].volume > scoredOrders[j].volume
	})

	remaining := route.LoadedVolume
	selected := make([]string, 0)
	for _, item := range scoredOrders {
		if remaining <= effective {
			break
		}
		selected = append(selected, item.orderID)
		remaining -= item.volume
	}
	return selected, excessVU
}
