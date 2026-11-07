package planning

import (
	"math"
	"sort"
)

// ProjectionOutput is the twin-backed scenario projection result.
type ProjectionOutput struct {
	SLARiskPct         float64
	BaselineSLARiskPct float64
	FleetVolume        int64
	StockoutSKUs       []string
	CapacityBreach     bool
	RevenueAtRiskMinor int64
	UnitValueSource    string
	Mode               string
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
		UnitValueSource:    shocked.unitValueSource,
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
	unitValueSource    string
}

func projectOnce(snap NetworkSnapshot, downtimeHours int, demandDeltaPct float64) projectionScratch {
	downtimeFactor := 1.0 - math.Min(float64(downtimeHours)/168.0, 0.9)
	demandFactor := 1.0 + demandDeltaPct/100.0
	fallback := scenarioUnitValueFallbackMinor()

	var stockouts []string
	var revenueAtRisk int64
	usedProduct, usedFallback := false, false
	for sku, demand := range snap.OpenDemand {
		projected := int64(math.Ceil(float64(demand) * demandFactor))
		avail := snap.Inventory[sku]
		if projected > avail {
			stockouts = append(stockouts, sku)
			shortfall := projected - avail
			uv, fromProduct := unitValueForSKU(snap.UnitValueMinor, sku, fallback)
			if fromProduct {
				usedProduct = true
			} else {
				usedFallback = true
			}
			revenueAtRisk += shortfall * uv
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

	source := ""
	if len(stockouts) > 0 {
		source = classifyUnitValueSource(usedProduct, usedFallback)
	}

	return projectionScratch{
		slaRisk:            slaRisk,
		fleetVolume:        fleetVolume,
		stockoutSKUs:       stockouts,
		capacityBreach:     capacityBreach,
		revenueAtRiskMinor: revenueAtRisk,
		unitValueSource:    source,
	}
}

// heuristicRevenueAtRisk estimates RaR for critical SKUs using product prices.
// shortfallQtyBySKU may be empty; missing qty defaults to 1 unit per SKU.
func heuristicRevenueAtRisk(unitValues map[string]int64, stockoutSKUs []string, shortfallQtyBySKU map[string]int64) (rar int64, source string) {
	if len(stockoutSKUs) == 0 {
		return 0, ""
	}
	fallback := scenarioUnitValueFallbackMinor()
	usedProduct, usedFallback := false, false
	for _, sku := range stockoutSKUs {
		qty := int64(1)
		if shortfallQtyBySKU != nil {
			if q := shortfallQtyBySKU[sku]; q > 0 {
				qty = q
			}
		}
		uv, fromProduct := unitValueForSKU(unitValues, sku, fallback)
		if fromProduct {
			usedProduct = true
		} else {
			usedFallback = true
		}
		rar += qty * uv
	}
	return rar, classifyUnitValueSource(usedProduct, usedFallback)
}
