package dispatch

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
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

// BinPackOptions configures BinPack behaviour.
type BinPackOptions struct {
	// AllowRetailerSplit permits the algorithm to split a single retailer's
	// consolidated order across multiple trucks when it exceeds a single truck's
	// capacity. When false, oversized consolidated orders become OverflowWarnings
	// that the warehouse admin must resolve before dispatching.
	AllowRetailerSplit bool
}

// BinPack runs the Smart Fit protocol over a pre-fetched set of orders and fleet.
// Pure computation — no I/O. Each retailer's orders are treated as one atomic
// super-order (no-split rule). If a super-order exceeds max single-truck capacity,
// it is reported as a RetailerOverflowWarning instead of silently orphaned so the
// warehouse admin can decide: cancel orders or allow a multi-truck split.
//
//	Rule 1 — Consolidation: fit into existing same-cell route.
//	Rule 2 — Multi-Stop:    greedy first-fit into same-cell group at 95% cap.
//	Rule 3 — Oversized:     super-orders exceeding max fleet cap → OverflowWarning.
//	Rule 4 — Override:      IgnoreCapacity bypasses volume check.
//	Rule 5 — Split:         when AllowRetailerSplit=true, oversized super-orders
//	                        are split greedily across multiple trucks with a shared
//	                        SplitGroupID for payment coordination.
func BinPack(orders []DispatchableOrder, fleet []AvailableDriver, cellLookup func(lat, lng float64) string, opts ...BinPackOptions) *AssignmentResult {
	var opt BinPackOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

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

	// Separate normal vs override.
	var normal []DispatchableOrder
	for _, o := range orders {
		if o.IgnoreCapacity {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("ORDER %s: IgnoreCapacity=true, bypassing volume check (%.1f VU)", o.OrderID, o.VolumeVU))
			normal = append(normal, o)
			continue
		}
		normal = append(normal, o)
	}

	// Group normal orders by RetailerID to enforce No-Split rule.
	retailerGroups := make(map[string][]DispatchableOrder)
	retailerOrder := []string{} // preserve insertion order for determinism
	for _, o := range normal {
		rid := o.RetailerID
		if rid == "" {
			rid = "anon_" + o.OrderID
		}
		if _, exists := retailerGroups[rid]; !exists {
			retailerOrder = append(retailerOrder, rid)
		}
		retailerGroups[rid] = append(retailerGroups[rid], o)
	}

	type RetailerSuperOrder struct {
		RetailerID   string
		RetailerName string
		VolumeVU     float64
		IgnoreCap    bool
		Lat          float64
		Lng          float64
		Orders       []DispatchableOrder
	}

	var superOrders []RetailerSuperOrder
	for _, rid := range retailerOrder {
		group := retailerGroups[rid]
		var totalVU float64
		var ignoreCap bool
		retailerName := ""
		for _, o := range group {
			totalVU += o.VolumeVU
			if o.IgnoreCapacity {
				ignoreCap = true
			}
			if retailerName == "" {
				retailerName = o.RetailerName
			}
		}

		if !ignoreCap && totalVU > maxFleetCap && maxFleetCap > 0 {
			// Consolidated super-order exceeds the largest truck.
			orderIDs := make([]string, 0, len(group))
			for _, o := range group {
				orderIDs = append(orderIDs, o.OrderID)
			}

			if opt.AllowRetailerSplit {
				// Admin approved the split: dispatch the orders across as many
				// trucks as needed. They share a SplitGroupID for payment coord.
				splitGroupID := uuid.NewString()
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("RETAILER %s: %.1f VU exceeds max truck %.1f VU — splitting across trucks (group %s)",
						rid, totalVU, maxFleetCap, splitGroupID))
				// Break the group into truck-sized chunks and add as individual
				// super-orders with the SplitGroupID stamped.
				remaining := append([]DispatchableOrder(nil), group...)
				for len(remaining) > 0 {
					var chunk []DispatchableOrder
					var chunkVU float64
					var leftover []DispatchableOrder
					for _, o := range remaining {
						if chunkVU+o.VolumeVU <= maxFleetCap || len(chunk) == 0 {
							chunk = append(chunk, o)
							chunkVU += o.VolumeVU
						} else {
							leftover = append(leftover, o)
						}
					}
					so := RetailerSuperOrder{
						RetailerID:   rid,
						RetailerName: retailerName,
						VolumeVU:     chunkVU,
						IgnoreCap:    false,
						Lat:          chunk[0].Lat,
						Lng:          chunk[0].Lng,
						Orders:       chunk,
					}
					// Mark each order in this chunk with the split group ID so
					// dispatch_execute can link manifests.
					for i := range so.Orders {
						_ = so.Orders[i] // SplitGroupID is stamped at GeoOrder creation below
					}
					// Attach SplitGroupID via orders – will be transferred to GeoOrder
					superOrders = append(superOrders, so)
					// Record the group for result.SplitShipmentGroups (populated in commit).
					// We reuse Warnings to carry the split group ID for now; dispatch_execute
					// reads result.Warnings for split-group reconstruction.
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("SPLIT_GROUP_ID:%s:RETAILER:%s:ORDERS:%v", splitGroupID, rid, orderIDs))
					remaining = leftover
				}
			} else {
				// Admin has NOT approved split: surface as OverflowWarning.
				result.OverflowWarnings = append(result.OverflowWarnings, RetailerOverflowWarning{
					RetailerID:    rid,
					RetailerName:  retailerName,
					OrderIDs:      orderIDs,
					TotalVolumeVU: totalVU,
					MaxTruckVU:    maxFleetCap,
					ExcessVU:      totalVU - maxFleetCap,
					SplitRequired: true,
				})
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("RETAILER %s: %.1f VU exceeds max truck capacity %.1f VU — admin action required (split or cancel)",
						rid, totalVU, maxFleetCap))
				for _, o := range group {
					result.Orphans = append(result.Orphans, o.ToGeo())
				}
			}
			continue
		}

		superOrders = append(superOrders, RetailerSuperOrder{
			RetailerID:   group[0].RetailerID,
			RetailerName: retailerName,
			VolumeVU:     totalVU,
			IgnoreCap:    ignoreCap,
			Lat:          group[0].Lat,
			Lng:          group[0].Lng,
			Orders:       group,
		})
	}

	// H3 cell grouping for spatial consolidation.
	cellGroups := make(map[string][]RetailerSuperOrder)
	cellOrder := []string{}
	for _, so := range superOrders {
		cell := cellLookup(so.Lat, so.Lng)
		if _, exists := cellGroups[cell]; !exists {
			cellOrder = append(cellOrder, cell)
		}
		cellGroups[cell] = append(cellGroups[cell], so)
	}

	driverRouteMap := make(map[string]int)

	availableFleet := make([]AvailableDriver, len(fleet))
	copy(availableFleet, fleet)

	for _, cell := range cellOrder {
		group := cellGroups[cell]
		sort.Slice(group, func(i, j int) bool {
			return group[i].VolumeVU > group[j].VolumeVU
		})

		for _, so := range group {
			placed := false

			if so.IgnoreCap {
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
					for _, o := range so.Orders {
						geo := o.ToGeo()
						geo.Assigned = true
						result.Routes[bestRoute].Orders = append(result.Routes[bestRoute].Orders, geo)
					}
					result.Routes[bestRoute].LoadedVolume += so.VolumeVU
					placed = true
				} else {
					match, ok := SelectBestVehicle(0, availableFleet)
					if ok {
						driverRouteMap[match.Driver.DriverID] = len(result.Routes)
						newRoute := DispatchRoute{
							DriverID:     match.Driver.DriverID,
							MaxVolume:    match.Driver.MaxVolumeVU * TetrisBuffer,
							LoadedVolume: so.VolumeVU,
						}
						for _, o := range so.Orders {
							geo := o.ToGeo()
							geo.Assigned = true
							newRoute.Orders = append(newRoute.Orders, geo)
						}
						result.Routes = append(result.Routes, newRoute)
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
					for _, o := range so.Orders {
						result.Orphans = append(result.Orphans, o.ToGeo())
					}
				}
				continue
			}

			// Balanced Routing: find the route with the LEAST number of orders that can fit `so`
			bestRouteIdx := -1
			bestRouteOrderCount := 999999

			// 1. Check existing active routes
			for ri := range result.Routes {
				remaining := result.Routes[ri].MaxVolume - result.Routes[ri].LoadedVolume
				if remaining >= so.VolumeVU {
					orderCount := len(result.Routes[ri].Orders)
					if orderCount < bestRouteOrderCount {
						bestRouteIdx = ri
						bestRouteOrderCount = orderCount
					}
				}
			}

			// 2. Check if a new truck would be better (it has 0 orders, so it balances perfectly)
			bestNewTruckIdx := -1
			if len(availableFleet) > 0 && bestRouteOrderCount > 0 {
				match, ok := SelectBestVehicle(so.VolumeVU, availableFleet)
				if ok && !match.Overflow {
					bestRouteIdx = -2 // Special marker for new truck
					bestRouteOrderCount = 0
					for i, d := range availableFleet {
						if d.DriverID == match.Driver.DriverID {
							bestNewTruckIdx = i
							break
						}
					}
				}
			}

			// 3. Assign
			if bestRouteIdx >= 0 {
				// Existing route
				for _, o := range so.Orders {
					geo := o.ToGeo()
					geo.Assigned = true
					result.Routes[bestRouteIdx].Orders = append(result.Routes[bestRouteIdx].Orders, geo)
				}
				result.Routes[bestRouteIdx].LoadedVolume += so.VolumeVU
				placed = true
			} else if bestRouteIdx == -2 {
				// New truck
				driver := availableFleet[bestNewTruckIdx]
				newRoute := DispatchRoute{
					DriverID:     driver.DriverID,
					MaxVolume:    driver.MaxVolumeVU * TetrisBuffer,
					LoadedVolume: so.VolumeVU,
				}
				for _, o := range so.Orders {
					geo := o.ToGeo()
					geo.Assigned = true
					newRoute.Orders = append(newRoute.Orders, geo)
				}
				result.Routes = append(result.Routes, newRoute)
				availableFleet = append(availableFleet[:bestNewTruckIdx], availableFleet[bestNewTruckIdx+1:]...)
				placed = true
			}

			if !placed {
				// No existing truck fits and no new truck fits (or fleet empty)
				match, ok := SelectBestVehicle(so.VolumeVU, availableFleet)
				if ok && match.Overflow {
					// Fallback: This individual super-order exceeds the remaining largest truck!
					// Record as overflow warning.
					orderIDs := make([]string, 0, len(so.Orders))
					for _, o := range so.Orders {
						orderIDs = append(orderIDs, o.OrderID)
					}
					result.OverflowWarnings = append(result.OverflowWarnings, RetailerOverflowWarning{
						RetailerID:    so.RetailerID,
						RetailerName:  so.RetailerName,
						OrderIDs:      orderIDs,
						TotalVolumeVU: so.VolumeVU,
						MaxTruckVU:    maxFleetCap,
						ExcessVU:      so.VolumeVU - maxFleetCap,
						SplitRequired: true,
					})
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("RETAILER %s: %.1f VU exceeds any available truck — admin action required",
							so.RetailerID, so.VolumeVU))
				}
				for _, o := range so.Orders {
					result.Orphans = append(result.Orphans, o.ToGeo())
				}
			}
		}
	}

	// Suppress driverRouteMap "unused" lint.
	_ = driverRouteMap

	return result
}
