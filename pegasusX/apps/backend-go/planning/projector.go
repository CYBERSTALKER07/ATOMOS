package planning

import (
	"math"
	"sort"
)

// ProjectionOutput is the twin-backed scenario projection result.
type ProjectionOutput struct {
	SLARiskPct           float64
	BaselineSLARiskPct   float64
	FleetVolume          int64
	StockoutSKUs         []string
	CapacityBreach       bool
	RevenueAtRiskMinor   int64
	Mode                 string
}

// ProjectSnapshot applies shocks to a cloned snapshot and returns impact metrics.
func ProjectSnapshot(snap NetworkSnapshot, in ScenarioInput) ProjectionOutput {
	baseline := projectOnce(snap, 0, 0)
	shocked := projectOnce(snap, in.FactoryDowntimeHours, in.DemandDeltaPct)

	out := ProjectionOutput{
		SLARiskPct:         shocked.slaRisk,
		BaselineSLARiskPct: baseline.slaRisk,
		FleetVolume:        shocked.fleetVolume,
		StockoutSKUs:       shocked.stockoutSKUs,
		CapacityBreach:     shocked.capacityBreach,
		RevenueAtRiskMinor: shocked.revenueAtRiskMinor,
		Mode:               "twin_snapshot",
	}
	return out
}

type projectionScratch struct {
	slaRisk            float64
	fleetVolume        int64
	stockoutSKUs       []string
	capacityBreach     bool
	revenueAtRiskMinor int64
}

func projectOnce(snap NetworkSnapshot, downtimeHours int, demandDeltaPct float64) projectionScratch {
	downtimeFactor := 1.0 - math.Min(float64(downtimeHours)/168.0, 0.9)
	demandFactor := 1.0 + demandDeltaPct/100.0

	var stockouts []string
	var revenueAtRisk int64
	for sku, demand := range snap.OpenDemand {
		projected := int64(math.Ceil(float64(demand) * demandFactor))
		avail := snap.Inventory[sku]
		if projected > avail {
			stockouts = append(stockouts, sku)
			shortfall := projected - avail
			revenueAtRisk += shortfall * 10000 // placeholder unit value in minor units
		}
	}
	sort.Strings(stockouts)

	slaRisk := math.Min(95, float64(len(stockouts))*8+float64(downtimeHours)*2)
	slaRisk *= downtimeFactor
	if demandFactor > 1 {
		slaRisk = math.Min(95, slaRisk*demandFactor)
	}

	fleetVolume := int64(float64(snap.OpenOrderCount) * demandFactor)
	capacityBreach := downtimeHours > 24 && snap.WarehouseCount > 0

	return projectionScratch{
		slaRisk:            slaRisk,
		fleetVolume:        fleetVolume,
		stockoutSKUs:       stockouts,
		capacityBreach:     capacityBreach,
		revenueAtRiskMinor: revenueAtRisk,
	}
}
