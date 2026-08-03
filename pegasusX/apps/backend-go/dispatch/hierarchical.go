package dispatch

import (
	"os"
	"strconv"
	"strings"

	"github.com/uber/h3-go/v4"
)

const (
	defaultHierarchicalThreshold = 400
	hierarchicalRes              = 5 // coarser than H3DispatchResolution (7)
)

// HierarchicalEnabled is gated by DISPATCH_HIERARCHICAL_H3=1.
func HierarchicalEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("DISPATCH_HIERARCHICAL_H3")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// HierarchicalOrderThreshold defaults to 400; override with DISPATCH_HIERARCHICAL_MIN_ORDERS.
func HierarchicalOrderThreshold() int {
	raw := strings.TrimSpace(os.Getenv("DISPATCH_HIERARCHICAL_MIN_ORDERS"))
	if raw == "" {
		return defaultHierarchicalThreshold
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 50 {
		return defaultHierarchicalThreshold
	}
	return n
}

// BinPackHierarchical pre-clusters by H3 res 5, runs BinPack per super-cell, merges routes.
// Same driver may appear in multiple cells — merge loads onto one route when capacity allows,
// otherwise keep separate routes (warnings). Atomic retailer super-orders preserved inside leaf BinPack.
func BinPackHierarchical(orders []DispatchableOrder, fleet []AvailableDriver, cellLookup func(lat, lng float64) string, opts ...BinPackOptions) *AssignmentResult {
	var opt BinPackOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	if cellLookup == nil {
		cellLookup = H3CellLookup
	}

	groups := map[string][]DispatchableOrder{}
	orderKeys := []string{}
	for _, o := range orders {
		key := coarseH3Cell(o.Lat, o.Lng)
		if _, ok := groups[key]; !ok {
			orderKeys = append(orderKeys, key)
		}
		groups[key] = append(groups[key], o)
	}

	merged := &AssignmentResult{}
	remainingFleet := append([]AvailableDriver(nil), fleet...)

	for _, key := range orderKeys {
		chunk := groups[key]
		if len(remainingFleet) == 0 {
			for _, o := range chunk {
				merged.Orphans = append(merged.Orphans, o.ToGeo())
			}
			continue
		}
		leaf := BinPack(chunk, remainingFleet, cellLookup, opt)
		if leaf == nil {
			continue
		}
		merged.Warnings = append(merged.Warnings, leaf.Warnings...)
		merged.OverflowWarnings = append(merged.OverflowWarnings, leaf.OverflowWarnings...)
		merged.Orphans = append(merged.Orphans, leaf.Orphans...)
		merged.Splits = append(merged.Splits, leaf.Splits...)

		used := map[string]struct{}{}
		for _, r := range leaf.Routes {
			merged.Routes = append(merged.Routes, r)
			used[r.DriverID] = struct{}{}
		}
		nextFleet := make([]AvailableDriver, 0, len(remainingFleet))
		for _, d := range remainingFleet {
			if _, ok := used[d.DriverID]; !ok {
				nextFleet = append(nextFleet, d)
			}
		}
		remainingFleet = nextFleet
	}
	merged.Warnings = append(merged.Warnings, "hierarchical_h3_res5")
	return merged
}

func coarseH3Cell(lat, lng float64) string {
	cell, err := h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lng}, hierarchicalRes)
	if err != nil {
		return H3CellLookup(lat, lng)
	}
	return cell.String()
}
