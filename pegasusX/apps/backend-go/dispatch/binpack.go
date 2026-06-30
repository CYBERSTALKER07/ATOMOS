package dispatch

import (
	"fmt"
	"sort"
)

// SelectBestVehicle implements smallest-fit escalation.
// Returns the smallest vehicle whose effective capacity >= orderVolumeVU.
// If none fits, returns the largest vehicle with Overflow=true.
// Returns (nil, false) only when fleet is empty.
func SelectBestVehicle(orderVolumeVU float64, fleet []AvailableDriver) (*VehicleMatch, bool) {
	if len(fleet) == 0 {
		return nil, false
	}

	sorted := make([]AvailableDriver, len(fleet))
	copy(sorted, fleet)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].MaxVolumeVU < sorted[j].MaxVolumeVU
	})

	for _, d := range sorted {
		if d.MaxVolumeVU*TetrisBuffer >= orderVolumeVU {
			return &VehicleMatch{Driver: d, Overflow: false}, true
		}
	}

	largest := sorted[len(sorted)-1]
	return &VehicleMatch{Driver: largest, Overflow: true}, true
}

// ComputeOrderVolume calculates TotalVolumeVU = Σ(qty_i × vol_i).
// Uses Kahan compensated summation for floating-point accuracy.
func ComputeOrderVolume(quantities []int, volumes []float64) float64 {
	if len(quantities) != len(volumes) {
		return 0
	}
	var sum, compensation float64
	for i := range quantities {
		y := float64(quantities[i])*volumes[i] - compensation
		t := sum + y
		compensation = (t - sum) - y
		sum = t
	}
	return sum
}

// BinPack runs the Smart Fit protocol over a pre-fetched set of orders and fleet.
// Pure computation — no I/O. Each order is atomic: it is assigned whole to one
// truck or left as an orphan; partial loads and cross-truck splits are not used.
//
//	Rule 1 — Consolidation: fit into existing same-cell route.
//	Rule 2 — Multi-Stop:    greedy first-fit into same-cell group at 95% cap.
//	Rule 3 — Oversized:     orders exceeding max fleet cap become orphans.
//	Rule 4 — Override:      IgnoreCapacity bypasses volume check.
func BinPack(orders []DispatchableOrder, fleet []AvailableDriver, cellLookup func(lat, lng float64) string) *AssignmentResult {
	result := &AssignmentResult{
		Routes:  []DispatchRoute{},
		Splits:  []SplitOrder{},
		Orphans: []GeoOrder{},
	}

	if len(orders) == 0 || len(fleet) == 0 {
		for _, o := range orders {
			result.Orphans = append(result.Orphans, o.ToGeo())
		}
		return result
	}

	// Max fleet capacity for split threshold.
	maxFleetCap := 0.0
	for _, d := range fleet {
		if eff := d.MaxVolumeVU * TetrisBuffer; eff > maxFleetCap {
			maxFleetCap = eff
		}
	}

	// Separate normal vs split-required vs override.
	var normal []DispatchableOrder
	for _, o := range orders {
		if o.IgnoreCapacity {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("ORDER %s: IgnoreCapacity=true, bypassing volume check (%.1f VU)", o.OrderID, o.VolumeVU))
			normal = append(normal, o)
			continue
		}

		if o.VolumeVU > maxFleetCap && maxFleetCap > 0 {
			geo := o.ToGeo()
			result.Orphans = append(result.Orphans, geo)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("ORDER %s: %.1f VU exceeds max fleet capacity %.1f VU — left undispatched (atomic order)",
					o.OrderID, o.VolumeVU, maxFleetCap))
			continue
		}

		normal = append(normal, o)
	}

	// H3 cell grouping for spatial consolidation.
	cellGroups := make(map[string][]DispatchableOrder)
	cellOrder := []string{}
	for _, o := range normal {
		cell := cellLookup(o.Lat, o.Lng)
		if _, exists := cellGroups[cell]; !exists {
			cellOrder = append(cellOrder, cell)
		}
		cellGroups[cell] = append(cellGroups[cell], o)
	}

	driverRouteMap := make(map[string]int)

	availableFleet := make([]AvailableDriver, len(fleet))
	copy(availableFleet, fleet)

	for _, cell := range cellOrder {
		group := cellGroups[cell]
		sort.Slice(group, func(i, j int) bool {
			return group[i].VolumeVU > group[j].VolumeVU
		})

		for _, o := range group {
			geo := o.ToGeo()
			placed := false

			if o.IgnoreCapacity {
				bestRoute := -1
				bestRemaining := -1.0
				for ri := range result.Routes {
					rem := result.Routes[ri].MaxVolume - result.Routes[ri].LoadedVolume
					if rem > bestRemaining {
						bestRoute = ri
						bestRemaining = rem
					}
				}
				if bestRoute >= 0 {
					geo.Assigned = true
					result.Routes[bestRoute].Orders = append(result.Routes[bestRoute].Orders, geo)
					result.Routes[bestRoute].LoadedVolume += o.VolumeVU
					placed = true
				} else {
					match, ok := SelectBestVehicle(0, availableFleet)
					if ok {
						geo.Assigned = true
						driverRouteMap[match.Driver.DriverID] = len(result.Routes)
						result.Routes = append(result.Routes, DispatchRoute{
							DriverID:     match.Driver.DriverID,
							MaxVolume:    match.Driver.MaxVolumeVU * TetrisBuffer,
							LoadedVolume: o.VolumeVU,
							Orders:       []GeoOrder{geo},
						})
						
						// Remove from available fleet
						for i, d := range availableFleet {
							if d.DriverID == match.Driver.DriverID {
								availableFleet = append(availableFleet[:i], availableFleet[i+1:]...)
								break
							}
						}
						placed = true
					}
				}
				if !placed {
					result.Orphans = append(result.Orphans, geo)
				}
				continue
			}

			// Consolidation: try existing route.
			for ri := range result.Routes {
				remaining := result.Routes[ri].MaxVolume - result.Routes[ri].LoadedVolume
				if remaining >= o.VolumeVU {
					geo.Assigned = true
					result.Routes[ri].Orders = append(result.Routes[ri].Orders, geo)
					result.Routes[ri].LoadedVolume += o.VolumeVU
					placed = true
					break
				}
			}
			if placed {
				continue
			}

			match, ok := SelectBestVehicle(o.VolumeVU, availableFleet)
			if !ok {
				result.Orphans = append(result.Orphans, geo)
				continue
			}

			if match.Overflow {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("ORDER %s: %.1f VU exceeds any single truck — left undispatched (atomic order)",
						o.OrderID, o.VolumeVU))
				result.Orphans = append(result.Orphans, geo)
				continue
			}

			geo.Assigned = true
			driverRouteMap[match.Driver.DriverID] = len(result.Routes)
			result.Routes = append(result.Routes, DispatchRoute{
				DriverID:     match.Driver.DriverID,
				MaxVolume:    match.Driver.MaxVolumeVU * TetrisBuffer,
				LoadedVolume: o.VolumeVU,
				Orders:       []GeoOrder{geo},
			})
			
			// Remove from available fleet
			for i, d := range availableFleet {
				if d.DriverID == match.Driver.DriverID {
					availableFleet = append(availableFleet[:i], availableFleet[i+1:]...)
					break
				}
			}
		}
	}

	return result
}
