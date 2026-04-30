package replenishment

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	internalKafka "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ═══════════════════════════════════════════════════════════════════════════════
// REPLENISHMENT ENGINE — THRESHOLD + PREDICTIVE DEFICIT ANALYSIS
//
// Math:
//   R = (V × L) + S   where S = V × L × 0.15 (safety buffer)
//   → R = V × L × 1.15
//   V = 7-day rolling burn rate (units/day)
//   L = factory lead time (days)
//   TTE = current_stock / V (time to empty, days)
//
// Urgency:
//   TTE ≤ L×1.3  → CRITICAL (auto-draft transfer)
//   TTE ≤ L×2.0  → WARNING
//   Otherwise     → STABLE (no insight)
//
// Deficit:
//   Q = R - (current_stock + in_transit - unfulfilled)
// ═══════════════════════════════════════════════════════════════════════════════

const (
	safetyBufferMultiplier = 0.15
	criticalLeadMultiplier = 1.3
	warningLeadMultiplier  = 2.0
	burnRateWindowDays     = 7
)

// ReplenishmentEngine runs periodic stock deficit analysis and auto-generates
// InternalTransferOrders when warehouse inventory drops below safety thresholds.
type ReplenishmentEngine struct {
	Spanner  *spanner.Client
	Producer *kafkago.Writer
}

// warehouseInfo holds warehouse + factory assignment data.
type warehouseInfo struct {
	WarehouseId        string
	SupplierId         string
	PrimaryFactoryId   string
	SecondaryFactoryId string
}

// skuStock holds per-SKU inventory state for a warehouse.
type skuStock struct {
	SkuId           string
	CurrentStock    int64
	DailyBurnRate   float64
	InTransitQty    int64
	UnfulfilledQty  int64
	FactoryLeadDays int64
	VolumetricUnit  float64
}

// StartReplenishmentCron runs every 4 hours, scanning all active warehouses.
func (e *ReplenishmentEngine) StartReplenishmentCron() {
	fmt.Println("[REPLENISHMENT] Stock deficit analysis cron initiated (4h interval)...")

	ticker := time.NewTicker(4 * time.Hour)

	go func() {
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			e.runCycle(ctx)
			cancel()
		}
	}()
}

// HandleManualTrigger — POST /v1/admin/replenishment/trigger
// Allows admin to manually trigger a replenishment cycle.
func (e *ReplenishmentEngine) HandleManualTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	e.runCycle(ctx)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "CYCLE_COMPLETE"})
}

func (e *ReplenishmentEngine) runCycle(ctx context.Context) {
	log.Println("[REPLENISHMENT] Starting deficit analysis cycle...")

	warehouses, err := e.fetchActiveWarehouses(ctx)
	if err != nil {
		log.Printf("[REPLENISHMENT] Failed to fetch warehouses: %v", err)
		return
	}

	totalInsights := 0
	totalTransfers := 0

	for _, wh := range warehouses {
		if wh.PrimaryFactoryId == "" {
			continue // no factory assigned → skip
		}

		insights, transfers, err := e.analyzeWarehouse(ctx, wh)
		if err != nil {
			log.Printf("[REPLENISHMENT] warehouse %s analysis failed: %v", wh.WarehouseId, err)
			continue
		}
		totalInsights += insights
		totalTransfers += transfers
	}

	log.Printf("[REPLENISHMENT] Cycle complete: %d warehouses scanned, %d insights generated, %d auto-transfers created",
		len(warehouses), totalInsights, totalTransfers)
}

func (e *ReplenishmentEngine) fetchActiveWarehouses(ctx context.Context) ([]warehouseInfo, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, SupplierId,
		             COALESCE(PrimaryFactoryId, ''), COALESCE(SecondaryFactoryId, '')
		      FROM Warehouses WHERE IsActive = true`,
	}
	iter := e.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var result []warehouseInfo
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var w warehouseInfo
		if err := row.Columns(&w.WarehouseId, &w.SupplierId, &w.PrimaryFactoryId, &w.SecondaryFactoryId); err != nil {
			return nil, err
		}
		result = append(result, w)
	}
	return result, nil
}

func (e *ReplenishmentEngine) analyzeWarehouse(ctx context.Context, wh warehouseInfo) (int, int, error) {
	leadTimeDays, err := e.getFactoryLeadTime(ctx, wh.PrimaryFactoryId)
	if err != nil {
		return 0, 0, fmt.Errorf("lead time lookup: %w", err)
	}

	stock, err := e.getWarehouseStock(ctx, wh.WarehouseId, wh.SupplierId)
	if err != nil {
		return 0, 0, fmt.Errorf("stock lookup: %w", err)
	}

	burnRates, err := e.get7DayBurnRates(ctx, wh.WarehouseId)
	if err != nil {
		return 0, 0, fmt.Errorf("burn rate lookup: %w", err)
	}

	inTransit, err := e.getInTransitStock(ctx, wh.WarehouseId)
	if err != nil {
		return 0, 0, fmt.Errorf("in-transit lookup: %w", err)
	}

	unfulfilled, err := e.getUnfulfilledDemand(ctx, wh.WarehouseId)
	if err != nil {
		return 0, 0, fmt.Errorf("unfulfilled demand lookup: %w", err)
	}

	vuMap, err := e.getVolumetricUnits(ctx, wh.SupplierId)
	if err != nil {
		return 0, 0, fmt.Errorf("VU lookup: %w", err)
	}

	// Merge all data into single SKU map
	allSkus := make(map[string]*skuStock)
	for skuId, qty := range stock {
		allSkus[skuId] = &skuStock{
			SkuId:           skuId,
			CurrentStock:    qty,
			FactoryLeadDays: leadTimeDays,
			VolumetricUnit:  vuMap[skuId],
		}
	}
	for skuId, rate := range burnRates {
		if _, ok := allSkus[skuId]; !ok {
			allSkus[skuId] = &skuStock{
				SkuId:           skuId,
				FactoryLeadDays: leadTimeDays,
				VolumetricUnit:  vuMap[skuId],
			}
		}
		allSkus[skuId].DailyBurnRate = rate
	}
	for skuId, qty := range inTransit {
		if _, ok := allSkus[skuId]; !ok {
			allSkus[skuId] = &skuStock{
				SkuId:           skuId,
				FactoryLeadDays: leadTimeDays,
				VolumetricUnit:  vuMap[skuId],
			}
		}
		allSkus[skuId].InTransitQty = qty
	}
	for skuId, qty := range unfulfilled {
		if _, ok := allSkus[skuId]; !ok {
			allSkus[skuId] = &skuStock{
				SkuId:           skuId,
				FactoryLeadDays: leadTimeDays,
				VolumetricUnit:  vuMap[skuId],
			}
		}
		allSkus[skuId].UnfulfilledQty = qty
	}

	insightCount := 0
	transferCount := 0

	for _, sku := range allSkus {
		if sku.DailyBurnRate <= 0 {
			continue // no demand → no insight
		}

		L := float64(sku.FactoryLeadDays)
		V := sku.DailyBurnRate
		S := V * L * safetyBufferMultiplier
		R := V*L + S // reorder point

		TTE := float64(sku.CurrentStock) / V // time to empty in days

		var urgency string
		if TTE <= L*criticalLeadMultiplier {
			urgency = "CRITICAL"
		} else if TTE <= L*warningLeadMultiplier {
			urgency = "WARNING"
		} else {
			continue // stock is healthy
		}

		// Compute suggested quantity: cover reorder point minus what's available/coming
		effectiveStock := float64(sku.CurrentStock) + float64(sku.InTransitQty) - float64(sku.UnfulfilledQty)
		suggestedQty := int64(math.Ceil(R - effectiveStock))
		if suggestedQty <= 0 {
			suggestedQty = int64(math.Ceil(V * L)) // minimum = cover lead time
		}

		reason := "LOW_STOCK"
		if V > float64(sku.CurrentStock)/L {
			reason = "HIGH_VELOCITY"
		}

		breakdown := map[string]interface{}{
			"unfulfilled":   sku.UnfulfilledQty,
			"in_transit":    sku.InTransitQty,
			"current_stock": sku.CurrentStock,
			"burn_rate_7d":  V,
			"reorder_point": R,
			"safety_stock":  S,
		}
		breakdownJSON, _ := json.Marshal(breakdown)

		insightID := uuid.New().String()
		targetFactory := wh.PrimaryFactoryId

		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdate("ReplenishmentInsights",
				[]string{"InsightId", "WarehouseId", "ProductId", "SupplierId",
					"CurrentStock", "DailyBurnRate", "TimeToEmptyDays",
					"SuggestedQuantity", "UrgencyLevel", "ReasonCode", "Status",
					"TargetFactoryId", "DemandBreakdown", "CreatedAt"},
				[]interface{}{insightID, wh.WarehouseId, sku.SkuId, wh.SupplierId,
					sku.CurrentStock, V, TTE,
					suggestedQty, urgency, reason, "PENDING",
					targetFactory, string(breakdownJSON), spanner.CommitTimestamp},
			),
		}

		if _, err := e.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite(mutations)
		}); err != nil {
			log.Printf("[REPLENISHMENT] Failed to write insight for %s/%s: %v",
				wh.WarehouseId, sku.SkuId, err)
			continue
		}
		insightCount++

		// Auto-draft transfer for CRITICAL urgency
		if urgency == "CRITICAL" {
			if err := e.autoCreateTransfer(ctx, wh, sku.SkuId, suggestedQty, sku.VolumetricUnit, targetFactory); err != nil {
				log.Printf("[REPLENISHMENT] Auto-transfer failed for %s/%s: %v",
					wh.WarehouseId, sku.SkuId, err)
			} else {
				transferCount++
			}
		}
	}

	return insightCount, transferCount, nil
}

// ── Data Fetchers ───────────────────────────────────────────────────────────

func (e *ReplenishmentEngine) getFactoryLeadTime(ctx context.Context, factoryID string) (int64, error) {
	row, err := e.Spanner.Single().ReadRow(ctx, "Factories",
		spanner.Key{factoryID}, []string{"LeadTimeDays"})
	if err != nil {
		return 2, nil // default 2 days
	}
	var lead int64
	if err := row.Columns(&lead); err != nil {
		return 2, nil
	}
	return lead, nil
}

func (e *ReplenishmentEngine) getWarehouseStock(ctx context.Context, whID, supplierID string) (map[string]int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ProductId, QuantityAvailable FROM SupplierInventory
		      WHERE SupplierId = @sid AND WarehouseId = @whId`,
		Params: map[string]interface{}{"sid": supplierID, "whId": whID},
	}
	iter := e.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	result := make(map[string]int64)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var pid string
		var qty int64
		if err := row.Columns(&pid, &qty); err != nil {
			continue
		}
		result[pid] = qty
	}
	return result, nil
}

func (e *ReplenishmentEngine) get7DayBurnRates(ctx context.Context, whID string) (map[string]float64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT li.SkuId, SUM(li.Quantity) AS TotalQty
		      FROM OrderLineItems li
		      JOIN Orders o ON li.OrderId = o.OrderId
		      WHERE o.WarehouseId = @whId
		        AND o.State = 'COMPLETED'
		        AND o.UpdatedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)
		      GROUP BY li.SkuId`,
		Params: map[string]interface{}{"whId": whID},
	}
	iter := e.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	result := make(map[string]float64)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var skuId string
		var totalQty int64
		if err := row.Columns(&skuId, &totalQty); err != nil {
			continue
		}
		result[skuId] = float64(totalQty) / float64(burnRateWindowDays)
	}
	return result, nil
}

func (e *ReplenishmentEngine) getInTransitStock(ctx context.Context, whID string) (map[string]int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT iti.ProductId, SUM(iti.Quantity) AS InTransitQty
		      FROM InternalTransferItems iti
		      JOIN InternalTransferOrders ito ON iti.TransferId = ito.TransferId
		      WHERE ito.WarehouseId = @whId
		        AND ito.State IN ('APPROVED', 'LOADING', 'DISPATCHED', 'IN_TRANSIT', 'ARRIVED')
		      GROUP BY iti.ProductId`,
		Params: map[string]interface{}{"whId": whID},
	}
	iter := e.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	result := make(map[string]int64)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var pid string
		var qty int64
		if err := row.Columns(&pid, &qty); err != nil {
			continue
		}
		result[pid] = qty
	}
	return result, nil
}

func (e *ReplenishmentEngine) getUnfulfilledDemand(ctx context.Context, whID string) (map[string]int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT li.SkuId, SUM(li.Quantity) AS UnfulfilledQty
		      FROM OrderLineItems li
		      JOIN Orders o ON li.OrderId = o.OrderId
		      WHERE o.WarehouseId = @whId
		        AND o.State IN ('PENDING', 'LOADED', 'IN_TRANSIT')
		        AND li.Status = 'PENDING'
		      GROUP BY li.SkuId`,
		Params: map[string]interface{}{"whId": whID},
	}
	iter := e.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	result := make(map[string]int64)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var skuId string
		var qty int64
		if err := row.Columns(&skuId, &qty); err != nil {
			continue
		}
		result[skuId] = qty
	}
	return result, nil
}

func (e *ReplenishmentEngine) getVolumetricUnits(ctx context.Context, supplierID string) (map[string]float64, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT SkuId, VolumetricUnit FROM SupplierProducts WHERE SupplierId = @sid`,
		Params: map[string]interface{}{"sid": supplierID},
	}
	iter := e.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	result := make(map[string]float64)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var skuId string
		var vu float64
		if err := row.Columns(&skuId, &vu); err != nil {
			continue
		}
		result[skuId] = vu
	}
	return result, nil
}

func (e *ReplenishmentEngine) autoCreateTransfer(ctx context.Context, wh warehouseInfo, skuId string, qty int64, vu float64, factoryId string) error {
	transferID := uuid.New().String()
	itemID := uuid.New().String()
	totalVU := float64(qty) * vu

	mutations := []*spanner.Mutation{
		spanner.Insert("InternalTransferOrders",
			[]string{"TransferId", "FactoryId", "WarehouseId", "SupplierId", "State", "TotalVolumeVU", "Source", "CreatedAt"},
			[]interface{}{transferID, factoryId, wh.WarehouseId, wh.SupplierId, "DRAFT", totalVU, "SYSTEM_THRESHOLD", spanner.CommitTimestamp},
		),
		spanner.Insert("InternalTransferItems",
			[]string{"TransferId", "ItemId", "ProductId", "Quantity", "VolumeVU"},
			[]interface{}{transferID, itemID, skuId, qty, totalVU},
		),
	}

	if _, err := e.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		evt := map[string]interface{}{
			"event":        "REPLENISHMENT_TRANSFER_CREATED",
			"transfer_id":  transferID,
			"factory_id":   factoryId,
			"warehouse_id": wh.WarehouseId,
			"supplier_id":  wh.SupplierId,
			"sku_id":       skuId,
			"quantity":     qty,
			"source":       "SYSTEM_THRESHOLD",
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
		}
		return outbox.EmitJSON(txn, "InternalTransferOrder", transferID, "REPLENISHMENT_TRANSFER_CREATED", internalKafka.TopicMain, evt, telemetry.TraceIDFromContext(ctx))
	}); err != nil {
		return err
	}

	log.Printf("[REPLENISHMENT] Auto-created transfer %s: %s → %s (SKU %s, qty %d)",
		transferID, factoryId, wh.WarehouseId, skuId, qty)
	return nil
}
