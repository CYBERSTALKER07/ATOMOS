package factory

import (
	"math"
	"sort"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

// SupplyLane is one factory→warehouse edge used by SelectOptimalFactory.
type SupplyLane struct {
	LaneID               string
	SupplierID           string
	FactoryID            string
	WarehouseID          string
	DampenedTransitHours float64
	FreightCostMinor     int64
	CarbonScoreKg        float64
	IsActive             bool
	Priority             int64
}

// FactoryCandidate is a fallback Haversine candidate (capacity observer-only).
type FactoryCandidate struct {
	FactoryID string
	Lat       float64
	Lng       float64
	IsActive  bool
}

// FactorySelection is the optimizer result. Capacity never blocks.
type FactorySelection struct {
	FactoryID     string
	LoadPercent   float64
	IsOverloaded  bool
	CapacityLabel string
}

// SelectOptimalFactoryFromLanes picks a factory for warehouse+SKU using mode.
// MANUAL_ONLY returns an empty selection. Empty lanes → caller should Haversine fallback.
func SelectOptimalFactoryFromLanes(mode string, lanes []SupplyLane) FactorySelection {
	mode = normalizeNetworkMode(mode)
	if mode == NetworkModeManualOnly {
		return FactorySelection{}
	}
	active := make([]SupplyLane, 0, len(lanes))
	for _, lane := range lanes {
		if !lane.IsActive || strings.TrimSpace(lane.FactoryID) == "" {
			continue
		}
		active = append(active, lane)
	}
	if len(active) == 0 {
		return FactorySelection{}
	}
	sortLanesByMode(mode, active)
	return unlimitedSelection(active[0].FactoryID)
}

func sortLanesByMode(mode string, lanes []SupplyLane) {
	sort.SliceStable(lanes, func(i, j int) bool {
		a, b := lanes[i], lanes[j]
		switch mode {
		case NetworkModeEconomy:
			if a.FreightCostMinor != b.FreightCostMinor {
				return a.FreightCostMinor < b.FreightCostMinor
			}
		case NetworkModeLowCarbon:
			if a.CarbonScoreKg != b.CarbonScoreKg {
				return a.CarbonScoreKg < b.CarbonScoreKg
			}
		case NetworkModeBalanced:
			sa := balancedScore(a)
			sb := balancedScore(b)
			if sa != sb {
				return sa < sb
			}
		default: // SPEED
			if a.DampenedTransitHours != b.DampenedTransitHours {
				return a.DampenedTransitHours < b.DampenedTransitHours
			}
		}
		if a.Priority != b.Priority {
			return a.Priority > b.Priority
		}
		return a.FactoryID < b.FactoryID
	})
}

func balancedScore(lane SupplyLane) float64 {
	return lane.DampenedTransitHours*0.5 + float64(lane.FreightCostMinor)*0.0003 + lane.CarbonScoreKg*0.2
}

func unlimitedSelection(factoryID string) FactorySelection {
	return FactorySelection{
		FactoryID:     factoryID,
		CapacityLabel: "UNLIMITED",
	}
}

// SelectFallbackFactory picks the nearest active factory by Haversine.
func SelectFallbackFactory(warehouseLat, warehouseLng float64, primaryID, secondaryID string, candidates []FactoryCandidate) FactorySelection {
	pool := make([]FactoryCandidate, 0, len(candidates))
	for _, c := range candidates {
		if !c.IsActive || strings.TrimSpace(c.FactoryID) == "" {
			continue
		}
		pool = append(pool, c)
	}
	if len(pool) == 0 {
		return FactorySelection{}
	}
	hasCoords := math.Abs(warehouseLat) > 1e-9 || math.Abs(warehouseLng) > 1e-9
	if hasCoords {
		best := pool[0]
		bestDist := proximity.HaversineDistance(warehouseLat, warehouseLng, best.Lat, best.Lng)
		for i := 1; i < len(pool); i++ {
			d := proximity.HaversineDistance(warehouseLat, warehouseLng, pool[i].Lat, pool[i].Lng)
			if d < bestDist {
				best = pool[i]
				bestDist = d
			}
		}
		return unlimitedSelection(best.FactoryID)
	}
	primaryID = strings.TrimSpace(primaryID)
	if primaryID != "" {
		for _, c := range pool {
			if c.FactoryID == primaryID {
				return unlimitedSelection(c.FactoryID)
			}
		}
	}
	secondaryID = strings.TrimSpace(secondaryID)
	if secondaryID != "" {
		for _, c := range pool {
			if c.FactoryID == secondaryID {
				return unlimitedSelection(c.FactoryID)
			}
		}
	}
	return unlimitedSelection(pool[0].FactoryID)
}
