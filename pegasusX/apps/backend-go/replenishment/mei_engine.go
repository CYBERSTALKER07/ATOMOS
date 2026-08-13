package replenishment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// MEIO solver honesty labels (G6.B) — never claim LP-optimal.
const (
	MEIOSolverGreedyCapitalV1 = "greedy_capital_v1"
	MEIOSolverCostAwareV2     = "cost_aware_v2"
)

// MEIONetworkSummary is the supplier-facing network optimization snapshot.
type MEIONetworkSummary struct {
	SupplierID              string              `json:"supplier_id"`
	WarehousesScanned       int                 `json:"warehouses_scanned"`
	SKUsAnalyzed            int                 `json:"skus_analyzed"`
	InsightsGenerated       int                 `json:"insights_generated"`
	TransferRecommendations int                 `json:"transfer_recommendations"`
	CapitalCapMinor         int64               `json:"capital_cap_minor,omitempty"`
	CapitalUsedMinor        int64               `json:"capital_used_minor,omitempty"`
	TransfersSkippedCapital int                 `json:"transfers_skipped_capital,omitempty"`
	MeioSolver              string              `json:"meio_solver"`
	WarehouseBalances       []MEIOWarehouseNode `json:"warehouse_balances"`
	GeneratedAt             string              `json:"generated_at"`
}

// meiTransferCandidate is a proposed surplus→deficit move evaluated under capital + transport.
type meiTransferCandidate struct {
	skuID             string
	qty               int64
	unitValueMinor    int64
	receiverDaysCover float64
	urgency           string
	donorWarehouseID  string
	transportCostKm   float64 // Haversine proxy between donor/receiver warehouses
	transportCost     float64 // distance_proxy × cost_per_km + fixed handling (minor units)
	bangForBuck       float64 // urgency_weight / (1 + transportCostKm)
	receiver          meiSkuBalance
}

// MEIOWarehouseNode summarizes stock health per warehouse in the network.
type MEIOWarehouseNode struct {
	WarehouseID  string  `json:"warehouse_id"`
	SKUCount     int     `json:"sku_count"`
	CriticalSKUs int     `json:"critical_skus"`
	WarningSKUs  int     `json:"warning_skus"`
	TotalStock   int64   `json:"total_stock"`
	AvgDaysCover float64 `json:"avg_days_cover"`
	TargetStock  int64   `json:"target_stock,omitempty"`
	OnHandStock  int64   `json:"on_hand_stock,omitempty"`
}

type meiSkuBalance struct {
	skuID        string
	warehouseID  string
	stock        int64
	burnRate     float64
	daysCover    float64
	urgency      string
	suggestedQty int64
}

// RunMEIONetwork analyzes all warehouses for one supplier in a single pass.
func (e *Engine) RunMEIONetwork(ctx context.Context, supplierID string) (MEIONetworkSummary, error) {
	summary := MEIONetworkSummary{
		SupplierID:  strings.TrimSpace(supplierID),
		GeneratedAt: e.Now().UTC().Format(time.RFC3339Nano),
		MeioSolver:  MEIOSolverCostAwareV2,
	}
	if e == nil || e.Spanner == nil || summary.SupplierID == "" {
		return summary, errors.New("mei engine unavailable")
	}
	if err := EnsurePolicy(ctx, e.Spanner, summary.SupplierID); err != nil {
		return summary, fmt.Errorf("ensure policy: %w", err)
	}

	warehouses, err := e.fetchActiveWarehouses(ctx, summary.SupplierID)
	if err != nil {
		return summary, err
	}
	summary.WarehousesScanned = len(warehouses)
	// Load warehouse coords for transport-cost scoring (best-effort).
	coords, _ := e.warehouseCoords(ctx, summary.SupplierID)

	type whAgg struct {
		skuCount   int
		critical   int
		warning    int
		totalStock int64
		daysSum    float64
		daysCount  int
	}
	aggByWh := make(map[string]*whAgg)
	var balances []meiSkuBalance

	for _, wh := range warehouses {
		if wh.PrimaryFactoryId == "" {
			continue
		}
		stock, err := e.getWarehouseStock(ctx, wh.WarehouseId, wh.SupplierId)
		if err != nil {
			return summary, err
		}
		burnRates, err := e.get7DayBurnRates(ctx, wh.WarehouseId)
		if err != nil {
			return summary, err
		}
		unfulfilled, err := e.getUnfulfilledDemand(ctx, wh.WarehouseId)
		if err != nil {
			return summary, err
		}
		for skuID, qty := range stock {
			burn := burnRates[skuID]
			if burn <= 0 {
				continue
			}
			effective := float64(qty) + float64(unfulfilled[skuID])
			days := effective / burn
			urgency := classifyUrgency(days, float64(defaultLeadTimeDays))
			sku := meiSkuBalance{
				skuID:       skuID,
				warehouseID: wh.WarehouseId,
				stock:       qty,
				burnRate:    burn,
				daysCover:   days,
				urgency:     urgency,
			}
			if urgency != "STABLE" {
				lead := float64(defaultLeadTimeDays)
				var reorder float64
				if SafetyStockV2Enabled() {
					reorder = ComputeReorderPoint(SafetyStockInputs{
						DBar:             burn,
						SigmaD:           math.Max(burn*0.25, 1),
						SigmaDAssumed:    true,
						L:                lead,
						SigmaL:           1.0,
						LeadSigmaAssumed: true,
						ServiceLevel:     0.98,
					}).ReorderPoint
				} else {
					reorder = LegacyReorderPoint(burn, lead)
				}
				sku.suggestedQty = computeSuggestedQty(skuStock{
					SkuId:           skuID,
					CurrentStock:    qty,
					DailyBurnRate:   burn,
					UnfulfilledQty:  unfulfilled[skuID],
					FactoryLeadDays: defaultLeadTimeDays,
				}, reorder)
			}
			balances = append(balances, sku)
			agg := aggByWh[wh.WarehouseId]
			if agg == nil {
				agg = &whAgg{}
				aggByWh[wh.WarehouseId] = agg
			}
			agg.skuCount++
			agg.totalStock += qty
			if urgency == "CRITICAL" {
				agg.critical++
			} else if urgency == "WARNING" {
				agg.warning++
			}
			if days > 0 && days < 365 {
				agg.daysSum += days
				agg.daysCount++
			}
		}
	}
	summary.SKUsAnalyzed = len(balances)

	if e.EchelonTargetsEnabled {
		if err := e.upsertEchelonTargets(ctx, summary.SupplierID, balances); err != nil {
			e.Log.Warn("mei.echelon_targets_failed", "supplier_id", summary.SupplierID, "err", err)
		}
	}

	// Network transfer recommendations: move stock from surplus to deficit warehouses,
	// constrained by Σ(transfer_qty × unit_value) <= MEIO_CAPITAL_CAP_MINOR (0 = unlimited).
	unitValues, _ := e.productUnitValuesMinor(ctx, summary.SupplierID)
	fallbackUnit := meioUnitValueFallbackMinor()
	capitalCap := meioCapitalCapMinor()
	summary.CapitalCapMinor = capitalCap

	bySKU := make(map[string][]meiSkuBalance)
	for _, b := range balances {
		bySKU[b.skuID] = append(bySKU[b.skuID], b)
	}
	var candidates []meiTransferCandidate
	for skuID, nodes := range bySKU {
		if len(nodes) < 2 {
			continue
		}
		var donor, receiver *meiSkuBalance
		for i := range nodes {
			n := &nodes[i]
			if n.urgency == "CRITICAL" || n.urgency == "WARNING" {
				if receiver == nil || n.daysCover < receiver.daysCover {
					receiver = n
				}
			}
			if n.urgency == "STABLE" && n.daysCover > float64(defaultLeadTimeDays)*warningLeadMultiplier {
				if donor == nil || n.daysCover > donor.daysCover {
					donor = n
				}
			}
		}
		if receiver == nil || donor == nil || donor.warehouseID == receiver.warehouseID {
			continue
		}
		transferQty := int64(math.Min(float64(donor.stock)/2, float64(receiver.suggestedQty)))
		if transferQty < 1 {
			continue
		}
		uv := unitValues[skuID]
		if uv <= 0 {
			uv = fallbackUnit
		}
		km := laneDistanceKm(coords, donor.warehouseID, receiver.warehouseID)
		tc := transportCostMinor(km)
		cand := meiTransferCandidate{
			skuID:             skuID,
			qty:               transferQty,
			unitValueMinor:    uv,
			receiverDaysCover: receiver.daysCover,
			urgency:           receiver.urgency,
			donorWarehouseID:  donor.warehouseID,
			transportCostKm:   km,
			transportCost:     tc,
			bangForBuck:       urgencyWeight(receiver.urgency) / (1.0 + km),
			receiver:          *receiver,
		}
		candidates = append(candidates, cand)
	}

	accepted, used, skipped := selectTransfersCostAware(candidates, capitalCap)
	summary.CapitalUsedMinor = used
	summary.TransfersSkippedCapital = skipped
	for _, c := range accepted {
		summary.TransferRecommendations++
		wh, whOK := warehouseByID(warehouses, c.receiver.warehouseID)
		if !whOK {
			continue
		}
		if err := e.writeMEIOInsight(ctx, wh, c.receiver, c.qty, c.donorWarehouseID); err != nil {
			e.Log.Warn("mei.write_insight_failed", "sku", c.skuID, "err", err)
			continue
		}
		summary.InsightsGenerated++
	}

	for whID, agg := range aggByWh {
		avgDays := 0.0
		if agg.daysCount > 0 {
			avgDays = agg.daysSum / float64(agg.daysCount)
		}
		summary.WarehouseBalances = append(summary.WarehouseBalances, MEIOWarehouseNode{
			WarehouseID:  whID,
			SKUCount:     agg.skuCount,
			CriticalSKUs: agg.critical,
			WarningSKUs:  agg.warning,
			TotalStock:   agg.totalStock,
			AvgDaysCover: avgDays,
			OnHandStock:  agg.totalStock,
		})
	}
	if e.EchelonTargetsEnabled {
		for i := range summary.WarehouseBalances {
			whID := summary.WarehouseBalances[i].WarehouseID
			targetSum, _, err := e.sumEchelonTargetsByWarehouse(ctx, summary.SupplierID, whID)
			if err == nil {
				summary.WarehouseBalances[i].TargetStock = targetSum
			}
		}
	}

	payload := events.PlanningEvent{
		BaseEvent: events.BaseEvent{
			Type:      events.EventPlanningMEIORecommendation,
			Timestamp: summary.GeneratedAt,
		},
		SupplierID:   summary.SupplierID,
		NetworkNodes: summary.WarehousesScanned,
		Transfers:    summary.TransferRecommendations,
	}
	_, _ = e.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregatePlanning, summary.SupplierID, events.TopicMain, payload); err != nil {
			return err
		}
		return txn.BufferWrite(outboxMutations(buf.events))
	})

	return summary, nil
}

func (e *Engine) writeMEIOInsight(ctx context.Context, wh warehouseInfo, receiver meiSkuBalance, qty int64, sourceWarehouseID string) error {
	pending, err := e.hasPendingInsight(ctx, receiver.warehouseID, receiver.skuID)
	if err != nil || pending {
		return err
	}
	breakdown, _ := json.Marshal(map[string]any{
		"mei_network":       true,
		"meio_solver":       MEIOSolverCostAwareV2,
		"source_warehouse":  sourceWarehouseID,
		"burn_rate_7d":      receiver.burnRate,
		"days_cover":        receiver.daysCover,
		"transfer_quantity": qty,
	})
	insightID := uuid.NewString()
	sku := &skuStock{
		SkuId:         receiver.skuID,
		CurrentStock:  receiver.stock,
		DailyBurnRate: receiver.burnRate,
	}
	if err := e.writeInsight(ctx, insightID, wh, sku, receiver.burnRate, receiver.daysCover, qty, receiver.urgency, "MEIO_NETWORK", string(breakdown)); err != nil {
		return err
	}
	return tryTouchlessApprove(ctx, e, wh.SupplierId, insightID, wh, sku, qty, receiver.urgency, "MEIO_NETWORK", string(breakdown))
}

func warehouseByID(warehouses []warehouseInfo, id string) (warehouseInfo, bool) {
	for _, wh := range warehouses {
		if wh.WarehouseId == id {
			return wh, true
		}
	}
	return warehouseInfo{}, false
}

// selectTransfersUnderCapital prioritizes CRITICAL then lowest days-cover receivers,
// accepting transfers while Σ(qty × unit_value) stays within capitalCapMinor.
// capitalCapMinor <= 0 means unlimited. Kept for tests / legacy label greedy_capital_v1.
func selectTransfersUnderCapital(cands []meiTransferCandidate, capitalCapMinor int64) (accepted []meiTransferCandidate, usedMinor int64, skipped int) {
	if len(cands) == 0 {
		return nil, 0, 0
	}
	sorted := append([]meiTransferCandidate(nil), cands...)
	sort.SliceStable(sorted, func(i, j int) bool {
		ui, uj := sorted[i].urgency, sorted[j].urgency
		if ui != uj {
			return urgencyRank(ui) < urgencyRank(uj)
		}
		if sorted[i].receiverDaysCover != sorted[j].receiverDaysCover {
			return sorted[i].receiverDaysCover < sorted[j].receiverDaysCover
		}
		return sorted[i].skuID < sorted[j].skuID
	})
	return acceptUnderCapital(sorted, capitalCapMinor)
}

// selectTransfersCostAware is HEURISTIC multi-commodity selection (G6.B):
// sort by bang-for-buck (urgency_weight / (1+km)) then urgency rank, under capital cap.
// Documented as cost_aware_v2 — not claimed LP-optimal.
func selectTransfersCostAware(cands []meiTransferCandidate, capitalCapMinor int64) (accepted []meiTransferCandidate, usedMinor int64, skipped int) {
	if len(cands) == 0 {
		return nil, 0, 0
	}
	sorted := append([]meiTransferCandidate(nil), cands...)
	for i := range sorted {
		if sorted[i].bangForBuck <= 0 {
			km := sorted[i].transportCostKm
			sorted[i].bangForBuck = urgencyWeight(sorted[i].urgency) / (1.0 + km)
		}
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].bangForBuck != sorted[j].bangForBuck {
			return sorted[i].bangForBuck > sorted[j].bangForBuck
		}
		if urgencyRank(sorted[i].urgency) != urgencyRank(sorted[j].urgency) {
			return urgencyRank(sorted[i].urgency) < urgencyRank(sorted[j].urgency)
		}
		if sorted[i].transportCostKm != sorted[j].transportCostKm {
			return sorted[i].transportCostKm < sorted[j].transportCostKm
		}
		if sorted[i].receiverDaysCover != sorted[j].receiverDaysCover {
			return sorted[i].receiverDaysCover < sorted[j].receiverDaysCover
		}
		return sorted[i].skuID < sorted[j].skuID
	})
	return acceptUnderCapital(sorted, capitalCapMinor)
}

func acceptUnderCapital(sorted []meiTransferCandidate, capitalCapMinor int64) (accepted []meiTransferCandidate, usedMinor int64, skipped int) {
	for _, c := range sorted {
		cost := c.qty * c.unitValueMinor
		if cost < 0 {
			cost = 0
		}
		if capitalCapMinor > 0 && usedMinor+cost > capitalCapMinor {
			skipped++
			continue
		}
		accepted = append(accepted, c)
		usedMinor += cost
	}
	return accepted, usedMinor, skipped
}

func urgencyWeight(u string) float64 {
	switch strings.ToUpper(strings.TrimSpace(u)) {
	case "CRITICAL":
		return 10.0
	case "WARNING":
		return 4.0
	default:
		return 1.0
	}
}

// transportCostMinor = distance_km × MEIO_COST_PER_KM_MINOR + fixed handling (minor units, scoring only).
func transportCostMinor(km float64) float64 {
	perKm := meioCostPerKmMinor()
	fixed := meioHandlingFixedMinor()
	if km < 0 {
		km = 0
	}
	return km*perKm + fixed
}

func meioCostPerKmMinor() float64 {
	v := strings.TrimSpace(os.Getenv("MEIO_COST_PER_KM_MINOR"))
	if v == "" {
		return 100 // 1.00 currency unit per km default
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil || n < 0 {
		return 100
	}
	return n
}

func meioHandlingFixedMinor() float64 {
	v := strings.TrimSpace(os.Getenv("MEIO_HANDLING_FIXED_MINOR"))
	if v == "" {
		return 500
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil || n < 0 {
		return 500
	}
	return n
}

type whCoord struct{ lat, lng float64 }

func (e *Engine) warehouseCoords(ctx context.Context, supplierID string) (map[string]whCoord, error) {
	out := map[string]whCoord{}
	if e == nil || e.Spanner == nil {
		return out, nil
	}
	sql := `SELECT WarehouseId, COALESCE(Lat, 0), COALESCE(Lng, 0) FROM Warehouses WHERE IsActive = true`
	params := map[string]any{}
	if strings.TrimSpace(supplierID) != "" {
		sql += ` AND SupplierId = @sid`
		params["sid"] = supplierID
	}
	iter := e.Spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return out, err
		}
		var id string
		var lat, lng float64
		if err := row.Columns(&id, &lat, &lng); err != nil {
			return out, err
		}
		out[id] = whCoord{lat: lat, lng: lng}
	}
	return out, nil
}

func laneDistanceKm(coords map[string]whCoord, donorID, receiverID string) float64 {
	if coords == nil {
		return 0
	}
	a, okA := coords[donorID]
	b, okB := coords[receiverID]
	if !okA || !okB {
		return 0
	}
	if a.lat == 0 && a.lng == 0 && b.lat == 0 && b.lng == 0 {
		return 0
	}
	return haversineKm(a.lat, a.lng, b.lat, b.lng)
}

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6371.0
	toRad := math.Pi / 180
	dLat := (lat2 - lat1) * toRad
	dLng := (lng2 - lng1) * toRad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*toRad)*math.Cos(lat2*toRad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return r * c
}

func urgencyRank(u string) int {
	switch strings.ToUpper(strings.TrimSpace(u)) {
	case "CRITICAL":
		return 0
	case "WARNING":
		return 1
	default:
		return 2
	}
}

func meioCapitalCapMinor() int64 {
	v := strings.TrimSpace(os.Getenv("MEIO_CAPITAL_CAP_MINOR"))
	if v == "" {
		return 0
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func meioUnitValueFallbackMinor() int64 {
	v := strings.TrimSpace(os.Getenv("MEIO_UNIT_VALUE_MINOR"))
	if v == "" {
		return 10000
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return 10000
	}
	return n
}

func (e *Engine) productUnitValuesMinor(ctx context.Context, supplierID string) (map[string]int64, error) {
	out := make(map[string]int64)
	if e == nil || e.Spanner == nil || strings.TrimSpace(supplierID) == "" {
		return out, nil
	}
	iter := e.Spanner.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT ProductId, PriceMinor FROM Products WHERE SupplierId = @sid AND IsActive = true`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return out, err
		}
		var pid string
		var price int64
		if err := row.Columns(&pid, &price); err != nil {
			return out, err
		}
		if price > 0 {
			out[pid] = price
		}
	}
	return out, nil
}
