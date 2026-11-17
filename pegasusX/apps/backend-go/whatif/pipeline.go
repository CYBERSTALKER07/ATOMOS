package whatif

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ScenarioRecord captures a PlanningScenarios row.
type ScenarioRecord struct {
	ScenarioID           string
	Status               string
	Label                string
	HorizonDays          int64
	FactoryDowntimeHours int64
}

// ScenarioPatchRecord captures a ScenarioObjectPatches overlay row.
type ScenarioPatchRecord struct {
	ScenarioID string
	ObjectType string
	ObjectID   string
	Operation  string
	PatchJSON  string
}

// ForecastRecord captures a ForecastValues row.
type ForecastRecord struct {
	SKU        string
	ProductID  string
	LocationID string
	IsWinner   bool
	P50Milli   int64
}

// InventoryRecord captures a SupplierInventoryV2 row.
type InventoryRecord struct {
	WarehouseID      string
	ProductID        string
	QuantityOnHand   int64
	QuantityReserved int64
	ReorderThreshold int64
}

// ProductPriceRecord captures Products.PriceMinor and currency.
type ProductPriceRecord struct {
	ProductID  string
	Name       string
	PriceMinor int64
	Currency   string
}

// CostLayerRecord captures InventoryCostLayers unit cost.
type CostLayerRecord struct {
	ProductID     string
	UnitCostMinor int64
}

// OrderDispatchRecord captures an Orders row for demand fallback and dispatch costing.
type OrderDispatchRecord struct {
	OrderID          string
	WarehouseID      string
	RouteID          string
	ManifestID       string
	Status           string
	TotalMinor       int64
	DeliveryFeeMinor int64
	LineItemsJSON    string
}

// RouteRecord captures RoutePerformanceAnalytics / SupplierTruckManifests route summary.
type RouteRecord struct {
	RouteID     string
	ReplanCount int64
}

// CreditProfileRecord captures RetailerCreditProfiles.
type CreditProfileRecord struct {
	RetailerID          string
	CreditLimitMinor    int64
	CurrentBalanceMinor int64
	ReservedMinor       int64
	RiskScore           int64
	DelinquencyCount    int64
	Status              string
}

// ArInvoiceRecord captures open ArInvoices.
type ArInvoiceRecord struct {
	InvoiceID    string
	RetailerID   string
	BalanceMinor int64
	Status       string
}

// SimulationDataset holds a consistent point-in-time read-only Spanner snapshot.
type SimulationDataset struct {
	TenantID        string
	EntityID        string
	Scenario        *ScenarioRecord
	ScenarioPatches []ScenarioPatchRecord
	Forecasts       []ForecastRecord
	Inventory       []InventoryRecord
	Products        []ProductPriceRecord
	CostLayers      []CostLayerRecord
	Orders          []OrderDispatchRecord
	Routes          []RouteRecord
	CreditProfiles  []CreditProfileRecord
	ArInvoices      []ArInvoiceRecord
}

// SimulationOutcome holds computed integer outputs and stage breakdown.
type SimulationOutcome struct {
	HorizonDays              int64
	ForecastDeltaMinor       int64
	ReplenishmentCostMinor   int64
	DispatchCostDeltaMinor   int64
	CreditExposureDeltaMinor int64
	NetPnLDeltaMinor         int64
	Breakdown                WhatIfBreakdown
}

// EvaluatePipeline executes the multi-stage E2E What-If simulator in pure int64 arithmetic.
// It NEVER mutates any domain or ledger table.
func EvaluatePipeline(ds SimulationDataset, in SimulateInput) SimulationOutcome {
	horizon := in.HorizonDays
	if horizon <= 0 {
		if ds.Scenario != nil && ds.Scenario.HorizonDays > 0 {
			horizon = ds.Scenario.HorizonDays
		} else {
			horizon = DefaultHorizonDays
		}
	}
	downtimeHours := in.FactoryDowntimeHours
	if downtimeHours == 0 && ds.Scenario != nil && ds.Scenario.FactoryDowntimeHours > 0 {
		downtimeHours = ds.Scenario.FactoryDowntimeHours
	}
	currency := nonEmpty(strings.ToUpper(in.Currency), "UZS")

	// Build base price, cost, inventory, forecast, and credit maps, then apply ScenarioObjectPatches overlay.
	priceBySKU := make(map[string]int64, len(ds.Products))
	for _, p := range ds.Products {
		if p.PriceMinor > 0 && p.ProductID != "" {
			priceBySKU[p.ProductID] = p.PriceMinor
		}
	}
	costBySKU := make(map[string]int64, len(ds.CostLayers))
	for _, c := range ds.CostLayers {
		if c.UnitCostMinor > 0 && c.ProductID != "" && costBySKU[c.ProductID] == 0 {
			costBySKU[c.ProductID] = c.UnitCostMinor
		}
	}

	type invAgg struct {
		availableUnits   int64
		reorderThreshold int64
		warehouseID      string
	}
	invBySKU := make(map[string]*invAgg, len(ds.Inventory))
	var primaryWarehouseID string
	for _, iv := range ds.Inventory {
		if primaryWarehouseID == "" && iv.WarehouseID != "" {
			primaryWarehouseID = iv.WarehouseID
		}
		avail := max64(0, iv.QuantityOnHand-iv.QuantityReserved)
		entry := invBySKU[iv.ProductID]
		if entry == nil {
			entry = &invAgg{warehouseID: iv.WarehouseID}
			invBySKU[iv.ProductID] = entry
		}
		entry.availableUnits += avail
		entry.reorderThreshold += max64(0, iv.ReorderThreshold)
	}

	// Stage 1: Forecast demand aggregation (prefer winner forecasts if present).
	hasWinners := false
	for _, f := range ds.Forecasts {
		if f.IsWinner {
			hasWinners = true
			break
		}
	}
	forecastBySKU := make(map[string]int64)
	for _, f := range ds.Forecasts {
		if hasWinners && !f.IsWinner {
			continue
		}
		sku := nonEmpty(f.ProductID, f.SKU)
		if sku == "" {
			continue
		}
		forecastBySKU[sku] += max64(0, f.P50Milli)
	}

	// Fallback baseline demand from Orders when ForecastValues is empty.
	var totalOrderRevenueMinor int64
	var totalDeliveryFeesMinor int64
	var activeOrderCount int
	routeSet := make(map[string]struct{})
	for _, r := range ds.Routes {
		if r.RouteID != "" {
			routeSet[r.RouteID] = struct{}{}
		}
	}
	for _, o := range ds.Orders {
		if strings.EqualFold(o.Status, "CANCELLED") {
			continue
		}
		activeOrderCount++
		totalOrderRevenueMinor += max64(0, o.TotalMinor)
		totalDeliveryFeesMinor += max64(0, o.DeliveryFeeMinor)
		if o.RouteID != "" {
			routeSet[o.RouteID] = struct{}{}
		}
		if primaryWarehouseID == "" && o.WarehouseID != "" {
			primaryWarehouseID = o.WarehouseID
		}
		if len(ds.Forecasts) == 0 {
			for _, l := range decodeOrderLines(o.LineItemsJSON) {
				sku := nonEmpty(l.productID, l.sku)
				if sku == "" || l.quantity <= 0 {
					continue
				}
				forecastBySKU[sku] += l.quantity * 1000
				if l.unitPriceMinor > 0 && priceBySKU[sku] == 0 {
					priceBySKU[sku] = l.unitPriceMinor
				}
			}
		}
	}

	// If neither ForecastValues nor order lines populated SKU demand, seed from inventory/products or order revenue.
	if len(forecastBySKU) == 0 && len(ds.Inventory) > 0 {
		for _, iv := range ds.Inventory {
			if iv.ProductID != "" {
				baselineUnits := max64(10, iv.ReorderThreshold+iv.QuantityReserved+10)
				forecastBySKU[iv.ProductID] += baselineUnits * 1000
			}
		}
	}
	if len(forecastBySKU) == 0 && totalOrderRevenueMinor > 0 {
		forecastBySKU["TENANT_BASKET"] = 1000
		priceBySKU["TENANT_BASKET"] = totalOrderRevenueMinor
	}

	// Apply ScenarioObjectPatches overlay (Copy-on-Write non-mutating).
	creditProfiles := make([]CreditProfileRecord, len(ds.CreditProfiles))
	copy(creditProfiles, ds.CreditProfiles)
	creditIdxByRetailer := make(map[string]int, len(creditProfiles))
	for i, cp := range creditProfiles {
		creditIdxByRetailer[cp.RetailerID] = i
	}

	patchCount := 0
	for _, p := range ds.ScenarioPatches {
		patchCount++
		props := decodePatchJSON(p.PatchJSON)
		op := strings.ToUpper(strings.TrimSpace(p.Operation))
		ot := strings.ToUpper(strings.TrimSpace(p.ObjectType))

		switch {
		case strings.Contains(ot, "PRODUCT") || strings.Contains(ot, "SKU"):
			if op == "DELETE" {
				delete(forecastBySKU, p.ObjectID)
				continue
			}
			if pm := parseInt64Prop(props, "price_minor", "unit_price_minor"); pm > 0 {
				priceBySKU[p.ObjectID] = pm
			}
			if cm := parseInt64Prop(props, "unit_cost_minor", "cost_minor"); cm > 0 {
				costBySKU[p.ObjectID] = cm
			}
			if qm := parseInt64Prop(props, "p50_milli", "forecast_milli"); qm > 0 {
				forecastBySKU[p.ObjectID] = qm
			}
		case strings.Contains(ot, "INVENTORY") || strings.Contains(ot, "STOCK"):
			if op == "DELETE" {
				delete(invBySKU, p.ObjectID)
				continue
			}
			avail := parseInt64Prop(props, "quantity_on_hand", "available_qty")
			thresh := parseInt64Prop(props, "reorder_threshold")
			if avail > 0 || thresh > 0 {
				entry := invBySKU[p.ObjectID]
				if entry == nil {
					entry = &invAgg{warehouseID: nonEmpty(primaryWarehouseID, "default")}
					invBySKU[p.ObjectID] = entry
				}
				if avail > 0 {
					entry.availableUnits = avail
				}
				if thresh > 0 {
					entry.reorderThreshold = thresh
				}
			}
		case strings.Contains(ot, "CREDIT"):
			if idx, ok := creditIdxByRetailer[p.ObjectID]; ok {
				if lim := parseInt64Prop(props, "credit_limit_minor"); lim > 0 {
					creditProfiles[idx].CreditLimitMinor = lim
				}
				if bal := parseInt64Prop(props, "current_balance_minor"); bal >= 0 {
					creditProfiles[idx].CurrentBalanceMinor = bal
				}
			}
		}
	}

	// Collect union of SKUs deterministically.
	skuSet := make(map[string]struct{}, len(forecastBySKU)+len(invBySKU))
	for sku := range forecastBySKU {
		skuSet[sku] = struct{}{}
	}
	for sku := range invBySKU {
		skuSet[sku] = struct{}{}
	}
	skus := make([]string, 0, len(skuSet))
	for sku := range skuSet {
		skus = append(skus, sku)
	}
	sort.Strings(skus)

	// Stage 1 & 2 Calculations: Demand & Replenishment.
	var (
		baseRevMinor              int64
		shockedRevMinor           int64
		baseUnitsMilli            int64
		shockedUnitsMilli         int64
		baseCogsMinor             int64
		shockedCogsMinor          int64
		baseReplenishCostMinor    int64
		shockedReplenishCostMinor int64
		totalShortfallMilli       int64
		affectedSkus              []string
		topShortfallSKU           string
		topShortfallMinor         int64
		topShortfallQtyUnits      int64
		topShortfallWH            string
	)

	horizonHours := horizon * 24
	downtimeLossBps := mulDivInt64(max64(0, downtimeHours), 10000, horizonHours)
	if downtimeLossBps > 10000 {
		downtimeLossBps = 10000
	}
	expediteBps := (max64(0, downtimeHours) * 50) / 8
	if expediteBps > 2500 {
		expediteBps = 2500
	}

	for _, sku := range skus {
		p50Milli := forecastBySKU[sku]
		unitPrice := priceBySKU[sku]
		if unitPrice <= 0 {
			unitPrice = 10_000 // fallback 100 UZS
		}
		unitCost := costBySKU[sku]
		if unitCost <= 0 {
			unitCost = mulDivInt64(unitPrice, 7000, 10000) // default 70% COGS ratio when no cost layer
		}

		shockedQtyMilli := max64(0, mulDivInt64(p50Milli, 10000+in.DemandDeltaBps, 10000))
		shockedPrice := max64(0, mulDivInt64(unitPrice, 10000+in.UnitPriceDeltaBps, 10000))
		shockedCost := max64(0, mulDivInt64(unitCost, 10000+in.FxRateShockBps, 10000))

		baseRevMinor += mulDivInt64(p50Milli, unitPrice, 1000)
		shockedRevMinor += mulDivInt64(shockedQtyMilli, shockedPrice, 1000)
		baseUnitsMilli += p50Milli
		shockedUnitsMilli += shockedQtyMilli

		baseCogsMinor += mulDivInt64(p50Milli, unitCost, 1000)
		shockedCogsMinor += mulDivInt64(shockedQtyMilli, shockedCost, 1000)

		var availUnits, reorderThresh int64
		wh := primaryWarehouseID
		if iv := invBySKU[sku]; iv != nil {
			availUnits = iv.availableUnits
			reorderThresh = iv.reorderThreshold
			if iv.warehouseID != "" {
				wh = iv.warehouseID
			}
		}
		availMilli := availUnits * 1000
		effectiveAvailMilli := mulDivInt64(availMilli, 10000-downtimeLossBps, 10000)
		threshMilli := reorderThresh * 1000

		baseShortfallMilli := max64(0, p50Milli+threshMilli-availMilli)
		shockedShortfallMilli := max64(0, shockedQtyMilli+threshMilli-effectiveAvailMilli)

		baseReplenishCostMinor += mulDivInt64(baseShortfallMilli, unitCost, 1000)
		effectiveReplCost := mulDivInt64(shockedCost, 10000+expediteBps, 10000)
		skuReplCostMinor := mulDivInt64(shockedShortfallMilli, effectiveReplCost, 1000)
		shockedReplenishCostMinor += skuReplCostMinor

		if shockedShortfallMilli > 0 {
			totalShortfallMilli += shockedShortfallMilli
			affectedSkus = append(affectedSkus, sku)
			if skuReplCostMinor > topShortfallMinor {
				topShortfallMinor = skuReplCostMinor
				topShortfallSKU = sku
				topShortfallQtyUnits = (shockedShortfallMilli + 999) / 1000
				topShortfallWH = wh
			}
		}
	}
	if affectedSkus == nil {
		affectedSkus = []string{}
	}

	forecastDeltaMinor := shockedRevMinor - baseRevMinor
	replenishCostDeltaMinor := shockedReplenishCostMinor - baseReplenishCostMinor

	// Stage 3: Dispatch & Fleet impact.
	baseDispatchCost := totalDeliveryFeesMinor
	if baseDispatchCost == 0 && (activeOrderCount > 0 || len(routeSet) > 0) {
		baseDispatchCost = int64(activeOrderCount)*150_000 + int64(len(routeSet))*500_000
	}
	if baseDispatchCost == 0 && baseUnitsMilli > 0 {
		estimatedOrders := max64(1, baseUnitsMilli/10000)
		baseDispatchCost = estimatedOrders * 150_000
	}
	fxFuelBps := mulDivInt64(in.FxRateShockBps, 4000, 10000)
	downtimeDispatchBps := min64(5000, max64(0, downtimeHours)*50)
	volumeFactorBps := max64(0, 10000+in.DemandDeltaBps)
	costFactorBps := max64(0, 10000+fxFuelBps+downtimeDispatchBps)
	shockedDispatchCost := mulDivInt64(baseDispatchCost, volumeFactorBps*costFactorBps, 10000*10000)
	dispatchCostDeltaMinor := shockedDispatchCost - baseDispatchCost

	// Stage 4: AR & Credit Exposure impact.
	openARByRetailer := make(map[string]int64)
	var totalOpenARMinor int64
	for _, inv := range ds.ArInvoices {
		openARByRetailer[inv.RetailerID] += max64(0, inv.BalanceMinor)
		totalOpenARMinor += max64(0, inv.BalanceMinor)
	}

	var baseCreditExposure int64
	var limitDeltaExposure int64
	var overLimitRetailers int
	var riskWeightSum int64
	var topExposedRetailer string
	var topRetailerExposure int64

	seenRetailer := make(map[string]bool)
	for _, cp := range creditProfiles {
		seenRetailer[cp.RetailerID] = true
		profExposure := cp.CurrentBalanceMinor + cp.ReservedMinor
		if ar := openARByRetailer[cp.RetailerID]; ar > profExposure {
			profExposure = ar
		}
		baseCreditExposure += profExposure
		newLimit := max64(0, mulDivInt64(cp.CreditLimitMinor, 10000+in.CreditLimitDeltaBps, 10000))
		if profExposure > newLimit && cp.CreditLimitMinor > 0 {
			overLimitRetailers++
		}
		// 50% drawdown weight on credit limit deltas.
		limitDelta := newLimit - cp.CreditLimitMinor
		limitDeltaExposure += mulDivInt64(limitDelta, 5000, 10000)
		riskBps := clampBps(200 + cp.RiskScore*50 + cp.DelinquencyCount*300)
		if riskBps > riskWeightSum {
			riskWeightSum = riskBps
		}
		if profExposure > topRetailerExposure || topExposedRetailer == "" {
			topRetailerExposure = profExposure
			topExposedRetailer = cp.RetailerID
		}
	}
	for rid, arBal := range openARByRetailer {
		if !seenRetailer[rid] {
			baseCreditExposure += arBal
			if arBal > topRetailerExposure || topExposedRetailer == "" {
				topRetailerExposure = arBal
				topExposedRetailer = rid
			}
		}
	}
	if riskWeightSum == 0 {
		riskWeightSum = 300 // baseline 3.00% expected loss weight
	}
	salesShockBps := in.DemandDeltaBps + in.UnitPriceDeltaBps + mulDivInt64(in.DemandDeltaBps, in.UnitPriceDeltaBps, 10000)
	arShiftMinor := mulDivInt64(baseCreditExposure, salesShockBps, 10000)
	creditExposureDeltaMinor := arShiftMinor + limitDeltaExposure
	shockedCreditExposure := max64(0, baseCreditExposure+creditExposureDeltaMinor)
	creditRiskCostDelta := int64(0)
	if creditExposureDeltaMinor > 0 {
		creditRiskCostDelta = mulDivInt64(creditExposureDeltaMinor, riskWeightSum, 10000)
	}

	// Stage 5: Net P&L Delta.
	cogsDeltaMinor := shockedCogsMinor - baseCogsMinor
	grossMarginDeltaMinor := forecastDeltaMinor - cogsDeltaMinor
	netPnLDeltaMinor := grossMarginDeltaMinor - replenishCostDeltaMinor - dispatchCostDeltaMinor - creditRiskCostDelta

	// Stage 6: Recommended governed L1 actions (never executed directly!).
	actions := buildRecommendedActions(
		ds,
		in,
		horizon,
		downtimeHours,
		primaryWarehouseID,
		topShortfallSKU,
		topShortfallWH,
		topShortfallQtyUnits,
		shockedReplenishCostMinor,
		dispatchCostDeltaMinor,
		topExposedRetailer,
		creditExposureDeltaMinor,
		overLimitRetailers,
	)

	return SimulationOutcome{
		HorizonDays:              horizon,
		ForecastDeltaMinor:       forecastDeltaMinor,
		ReplenishmentCostMinor:   shockedReplenishCostMinor,
		DispatchCostDeltaMinor:   dispatchCostDeltaMinor,
		CreditExposureDeltaMinor: creditExposureDeltaMinor,
		NetPnLDeltaMinor:         netPnLDeltaMinor,
		Breakdown: WhatIfBreakdown{
			Currency:        currency,
			ScenarioApplied: ds.Scenario != nil || patchCount > 0,
			PatchCount:      patchCount,
			Demand: DemandStageBreakdown{
				BaselineRevenueMinor: baseRevMinor,
				ShockedRevenueMinor:  shockedRevMinor,
				ForecastDeltaMinor:   forecastDeltaMinor,
				BaselineUnitsMilli:   baseUnitsMilli,
				ShockedUnitsMilli:    shockedUnitsMilli,
				SkuCount:             len(skus),
			},
			Replenishment: ReplenishmentStageBreakdown{
				BaselineCostMinor:   baseReplenishCostMinor,
				ShockedCostMinor:    shockedReplenishCostMinor,
				CostDeltaMinor:      replenishCostDeltaMinor,
				ShortfallUnitsMilli: totalShortfallMilli,
				AffectedSkus:        affectedSkus,
			},
			Dispatch: DispatchStageBreakdown{
				BaselineCostMinor:      baseDispatchCost,
				ShockedCostMinor:       shockedDispatchCost,
				DispatchCostDeltaMinor: dispatchCostDeltaMinor,
				RouteCount:             len(routeSet),
				OrderCount:             activeOrderCount,
			},
			Credit: CreditStageBreakdown{
				BaselineExposureMinor:    baseCreditExposure,
				ShockedExposureMinor:     shockedCreditExposure,
				CreditExposureDeltaMinor: creditExposureDeltaMinor,
				OverLimitRetailers:       overLimitRetailers,
				RiskCostDeltaMinor:       creditRiskCostDelta,
			},
			PnL: PnLStageBreakdown{
				CogsDeltaMinor:        cogsDeltaMinor,
				GrossMarginDeltaMinor: grossMarginDeltaMinor,
				NetPnLDeltaMinor:      netPnLDeltaMinor,
			},
			RecommendedActions: actions,
		},
	}
}

func buildRecommendedActions(
	ds SimulationDataset,
	in SimulateInput,
	horizon int64,
	downtimeHours int64,
	primaryWH string,
	topShortfallSKU string,
	topShortfallWH string,
	topShortfallQtyUnits int64,
	replenishCostMinor int64,
	dispatchDeltaMinor int64,
	topRetailerID string,
	creditExposureDeltaMinor int64,
	overLimitRetailers int,
) []RecommendedAction {
	var actions []RecommendedAction
	wh := nonEmpty(topShortfallWH, primaryWH, ds.EntityID, ds.TenantID, "wh-primary")

	if topShortfallSKU != "" || replenishCostMinor > 0 || in.DemandDeltaBps > 0 {
		sku := nonEmpty(topShortfallSKU, "SKU-PRIMARY")
		qty := max64(1, topShortfallQtyUnits)
		impact := max64(0, replenishCostMinor)
		actions = append(actions, RecommendedAction{
			ProposalType:         "REORDER",
			AgentID:              "REORDER_AGENT",
			TargetObjectType:     "SupplierInventory",
			TargetObjectID:       wh + ":" + sku,
			Title:                fmt.Sprintf("What-If Reorder %s (+%d units) at %s", sku, qty, wh),
			Summary:              fmt.Sprintf("Replenish projected %d-day shortfall under demand shock %+d bps and downtime %dh", horizon, in.DemandDeltaBps, downtimeHours),
			EstimatedImpactMinor: impact,
			ConfidenceBps:        8200,
			Payload: map[string]any{
				"warehouse_id":           wh,
				"product_id":             sku,
				"suggested_qty":          qty,
				"demand_delta_bps":       in.DemandDeltaBps,
				"fx_rate_shock_bps":      in.FxRateShockBps,
				"factory_downtime_hours": downtimeHours,
			},
		})
	}

	if topRetailerID != "" && (creditExposureDeltaMinor > 0 || overLimitRetailers > 0 || in.CreditLimitDeltaBps != 0) {
		propType := "CREDIT_HOLD"
		if creditExposureDeltaMinor <= 0 && overLimitRetailers == 0 {
			propType = "COLLECTIONS_NUDGE"
		}
		actions = append(actions, RecommendedAction{
			ProposalType:         propType,
			AgentID:              "COLLECTIONS_AGENT",
			TargetObjectType:     "RetailerCreditProfile",
			TargetObjectID:       topRetailerID,
			Title:                fmt.Sprintf("What-If Credit Review for Retailer %s", topRetailerID),
			Summary:              fmt.Sprintf("Projected credit exposure delta %+d minor (%d over-limit retailers)", creditExposureDeltaMinor, overLimitRetailers),
			EstimatedImpactMinor: max64(0, abs64(creditExposureDeltaMinor)),
			ConfidenceBps:        7800,
			Payload: map[string]any{
				"retailer_id":                 topRetailerID,
				"credit_limit_delta_bps":      in.CreditLimitDeltaBps,
				"credit_exposure_delta_minor": creditExposureDeltaMinor,
				"over_limit_retailers":        overLimitRetailers,
			},
		})
	}

	if dispatchDeltaMinor > 0 || downtimeHours > 0 {
		actions = append(actions, RecommendedAction{
			ProposalType:         "SUPPLY_REQUEST",
			AgentID:              "REORDER_AGENT",
			TargetObjectType:     "Warehouse",
			TargetObjectID:       wh,
			Title:                fmt.Sprintf("What-If Dispatch Rebalance for Warehouse %s", wh),
			Summary:              fmt.Sprintf("Rebalance fleet/supply allocation across %d-day horizon (dispatch delta %+d minor, downtime %dh)", horizon, dispatchDeltaMinor, downtimeHours),
			EstimatedImpactMinor: max64(0, abs64(dispatchDeltaMinor)),
			ConfidenceBps:        7600,
			Payload: map[string]any{
				"warehouse_id":              wh,
				"dispatch_cost_delta_minor": dispatchDeltaMinor,
				"factory_downtime_hours":    downtimeHours,
				"horizon_days":              horizon,
			},
		})
	}

	if len(actions) == 0 {
		actions = append(actions, RecommendedAction{
			ProposalType:         "SUPPLY_REQUEST",
			AgentID:              "REORDER_AGENT",
			TargetObjectType:     "Warehouse",
			TargetObjectID:       wh,
			Title:                fmt.Sprintf("What-If Scenario Review: %s", nonEmpty(in.Name, "Simulation")),
			Summary:              fmt.Sprintf("Governed L1 review for %d-day What-If simulation", horizon),
			EstimatedImpactMinor: 0,
			ConfidenceBps:        7000,
			Payload: map[string]any{
				"warehouse_id":     wh,
				"horizon_days":     horizon,
				"demand_delta_bps": in.DemandDeltaBps,
			},
		})
	}
	return actions
}

type decodedOrderLine struct {
	sku            string
	productID      string
	quantity       int64
	unitPriceMinor int64
}

func decodeOrderLines(raw string) []decodedOrderLine {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "[]" || trimmed == "null" {
		return nil
	}
	type rawLine struct {
		SKU            string `json:"sku"`
		ProductID      string `json:"product_id"`
		Quantity       int64  `json:"quantity"`
		UnitPriceMinor int64  `json:"unit_price_minor"`
	}
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.UseNumber()
	var lines []rawLine
	if err := dec.Decode(&lines); err != nil {
		return nil
	}
	out := make([]decodedOrderLine, 0, len(lines))
	for _, l := range lines {
		out = append(out, decodedOrderLine{
			sku:            l.SKU,
			productID:      l.ProductID,
			quantity:       l.Quantity,
			unitPriceMinor: l.UnitPriceMinor,
		})
	}
	return out
}

func decodePatchJSON(raw string) map[string]any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(trimmed)))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil
	}
	return m
}

func parseInt64Prop(m map[string]any, keys ...string) int64 {
	if m == nil {
		return 0
	}
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch tv := v.(type) {
		case int64:
			return tv
		case int:
			return int64(tv)
		case json.Number:
			if n, err := tv.Int64(); err == nil {
				return n
			}
		case string:
			if n, err := strconv.ParseInt(strings.TrimSpace(tv), 10, 64); err == nil {
				return n
			}
		}
	}
	return 0
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
