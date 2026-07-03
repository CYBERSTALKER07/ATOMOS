package dispatch

import "strings"

// ManualRouteInput is one operator-assigned truck route (warehouse + supplier MANUAL execute).
type ManualRouteInput struct {
	DriverID string
	OrderIDs []string
}

// CapacityWarning describes volumetric overflow for a manual dispatch route.
type CapacityWarning struct {
	DriverID                  string   `json:"driver_id"`
	LoadedVU                  float64  `json:"loaded_vu"`
	MaxVolumeVU               float64  `json:"max_volume_vu"`
	EffectiveMaxVU            float64  `json:"effective_max_vu"`
	ExcessVU                  float64  `json:"excess_vu,omitempty"`
	SuggestedUnselectOrderIDs []string `json:"suggested_unselect_order_ids,omitempty"`
}

// BuildManualAssignment maps operator-selected order IDs onto dispatch routes.
func BuildManualAssignment(rows []DispatchableOrder, manualRoutes []ManualRouteInput, driverMaxVU map[string]float64) *AssignmentResult {
	assignment := &AssignmentResult{}
	orderMap := make(map[string]DispatchableOrder, len(rows))
	for _, row := range rows {
		orderMap[row.OrderID] = row
	}
	for _, mr := range manualRoutes {
		driverID := strings.TrimSpace(mr.DriverID)
		if driverID == "" {
			continue
		}
		route := DispatchRoute{DriverID: driverID}
		if maxVU, ok := driverMaxVU[driverID]; ok && maxVU > 0 {
			route.MaxVolume = maxVU
		}
		for _, oid := range mr.OrderIDs {
			oid = strings.TrimSpace(oid)
			if oid == "" {
				continue
			}
			if o, ok := orderMap[oid]; ok {
				route.Orders = append(route.Orders, o.ToGeo())
				route.LoadedVolume += o.VolumeVU
			}
		}
		if len(route.Orders) > 0 {
			assignment.Routes = append(assignment.Routes, route)
		}
	}
	return assignment
}

// ManualCapacityWarnings returns per-route overflow relative to the Tetris buffer.
func ManualCapacityWarnings(routes []DispatchRoute, driverMaxVU map[string]float64) []CapacityWarning {
	warnings := make([]CapacityWarning, 0)
	for _, route := range routes {
		maxVU := route.MaxVolume
		if override, ok := driverMaxVU[route.DriverID]; ok && override > 0 {
			maxVU = override
		}
		if maxVU <= 0 {
			continue
		}
		effective := maxVU * TetrisBuffer
		if route.LoadedVolume <= effective {
			continue
		}
		suggested, excess := SuggestOrdersToUnselect(route)
		warnings = append(warnings, CapacityWarning{
			DriverID:                  route.DriverID,
			LoadedVU:                  route.LoadedVolume,
			MaxVolumeVU:               maxVU,
			EffectiveMaxVU:            effective,
			ExcessVU:                  excess,
			SuggestedUnselectOrderIDs: suggested,
		})
	}
	return warnings
}
