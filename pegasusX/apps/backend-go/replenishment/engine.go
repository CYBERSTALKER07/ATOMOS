package replenishment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/segment"
	"google.golang.org/api/iterator"
)

const (
	safetyBufferMultiplier = 0.15
	criticalLeadMultiplier = 1.3
	warningLeadMultiplier  = 2.0
	burnRateWindowDays     = 7
	defaultLeadTimeDays    = 2
)

// CycleResult summarizes one replenishment analysis pass.
type CycleResult struct {
	WarehousesScanned int `json:"warehouses_scanned"`
	InsightsGenerated int `json:"insights_generated"`
	TransfersCreated  int `json:"transfers_created"`
}

// Engine scans warehouse inventory against burn rates and persists insights.
type Engine struct {
	Spanner                *spanner.Client
	Log                    *slog.Logger
	Now                    func() time.Time
	EchelonTargetsEnabled  bool
	SegmentSvc             *segment.Service
}

// NewEngine returns an engine bound to Spanner.
func NewEngine(client *spanner.Client, log *slog.Logger) *Engine {
	if log == nil {
		log = slog.Default()
	}
	return &Engine{
		Spanner: client,
		Log:     log,
		Now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// StartCron runs analysis on a fixed interval until ctx is cancelled.
func (e *Engine) StartCron(ctx context.Context) {
	if e == nil || e.Spanner == nil {
		return
	}
	interval := cronInterval()
	e.Log.Info("replenishment.engine.cron_started", "interval", interval.String())
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
				result, err := e.RunCycle(runCtx)
				cancel()
				if err != nil {
					e.Log.Error("replenishment.engine.cycle_failed", "err", err)
					continue
				}
				e.Log.Info("replenishment.engine.cycle_completed",
					"warehouses_scanned", result.WarehousesScanned,
					"insights_generated", result.InsightsGenerated,
					"transfers_created", result.TransfersCreated,
				)
			}
		}
	}()
}

// RunCycle analyzes every active warehouse.
func (e *Engine) RunCycle(ctx context.Context) (CycleResult, error) {
	return e.runCycle(ctx, "")
}

// RunForSupplier analyzes warehouses owned by one supplier.
func (e *Engine) RunForSupplier(ctx context.Context, supplierID string) (CycleResult, error) {
	return e.runCycle(ctx, strings.TrimSpace(supplierID))
}

func (e *Engine) runCycle(ctx context.Context, supplierID string) (CycleResult, error) {
	if e == nil || e.Spanner == nil {
		return CycleResult{}, errors.New("replenishment engine unavailable")
	}
	warehouses, err := e.fetchActiveWarehouses(ctx, supplierID)
	if err != nil {
		return CycleResult{}, err
	}
	result := CycleResult{WarehousesScanned: len(warehouses)}
	for _, wh := range warehouses {
		if wh.PrimaryFactoryId == "" {
			continue
		}
		insights, transfers, analyzeErr := e.analyzeWarehouse(ctx, wh)
		if analyzeErr != nil {
			e.Log.Error("replenishment.engine.warehouse_analysis_failed",
				"warehouse_id", wh.WarehouseId,
				"supplier_id", wh.SupplierId,
				"err", analyzeErr,
			)
			continue
		}
		result.InsightsGenerated += insights
		result.TransfersCreated += transfers
	}
	if supplierID != "" {
		meiSummary, meiErr := e.RunMEIONetwork(ctx, supplierID)
		if meiErr != nil {
			e.Log.Warn("replenishment.mei_network_failed", "supplier_id", supplierID, "err", meiErr)
		} else {
			result.InsightsGenerated += meiSummary.InsightsGenerated
		}
	}
	return result, nil
}

type warehouseInfo struct {
	WarehouseId        string
	SupplierId         string
	PrimaryFactoryId   string
	SecondaryFactoryId string
}

type skuStock struct {
	SkuId           string
	CurrentStock    int64
	DailyBurnRate   float64
	InTransitQty    int64
	UnfulfilledQty  int64
	FactoryLeadDays int64
	UnitVolumeVU    float64
}

func classifyUrgency(tte, leadDays float64) string {
	if tte <= leadDays*criticalLeadMultiplier {
		return "CRITICAL"
	}
	if tte <= leadDays*warningLeadMultiplier {
		return "WARNING"
	}
	return "STABLE"
}

func computeSuggestedQty(stock skuStock, reorderPoint float64) int64 {
	effectiveStock := float64(stock.CurrentStock) + float64(stock.InTransitQty) - float64(stock.UnfulfilledQty)
	suggested := int64(math.Ceil(reorderPoint - effectiveStock))
	if suggested <= 0 {
		suggested = int64(math.Ceil(stock.DailyBurnRate * float64(stock.FactoryLeadDays)))
	}
	if suggested < 1 {
		suggested = 1
	}
	return suggested
}

func (e *Engine) fetchActiveWarehouses(ctx context.Context, supplierID string) ([]warehouseInfo, error) {
	sql := `SELECT WarehouseId, SupplierId,
	               COALESCE(PrimaryFactoryId, ''), COALESCE(SecondaryFactoryId, '')
	        FROM Warehouses WHERE IsActive = true`
	params := map[string]any{}
	if supplierID != "" {
		sql += ` AND SupplierId = @sid`
		params["sid"] = supplierID
	}
	iter := e.Spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var result []warehouseInfo
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var wh warehouseInfo
		if err := row.Columns(&wh.WarehouseId, &wh.SupplierId, &wh.PrimaryFactoryId, &wh.SecondaryFactoryId); err != nil {
			return nil, err
		}
		result = append(result, wh)
	}
	return result, nil
}

func (e *Engine) analyzeWarehouse(ctx context.Context, wh warehouseInfo) (int, int, error) {
	leadDays := int64(defaultLeadTimeDays)
	stock, err := e.getWarehouseStock(ctx, wh.WarehouseId, wh.SupplierId)
	if err != nil {
		return 0, 0, fmt.Errorf("stock lookup: %w", err)
	}
	burnRates, err := e.get7DayBurnRates(ctx, wh.WarehouseId)
	if err != nil {
		return 0, 0, fmt.Errorf("burn rate lookup: %w", err)
	}
	unfulfilled, err := e.getUnfulfilledDemand(ctx, wh.WarehouseId)
	if err != nil {
		return 0, 0, fmt.Errorf("unfulfilled demand lookup: %w", err)
	}
	vuMap, err := e.getUnitVolumes(ctx, wh.SupplierId)
	if err != nil {
		return 0, 0, fmt.Errorf("unit volume lookup: %w", err)
	}

	allSkus := make(map[string]*skuStock)
	for skuID, qty := range stock {
		allSkus[skuID] = &skuStock{
			SkuId:           skuID,
			CurrentStock:    qty,
			FactoryLeadDays: leadDays,
			UnitVolumeVU:    vuMap[skuID],
		}
	}
	for skuID, rate := range burnRates {
		sku := allSkus[skuID]
		if sku == nil {
			sku = &skuStock{SkuId: skuID, FactoryLeadDays: leadDays, UnitVolumeVU: vuMap[skuID]}
			allSkus[skuID] = sku
		}
		sku.DailyBurnRate = rate
	}
	for skuID, qty := range unfulfilled {
		sku := allSkus[skuID]
		if sku == nil {
			sku = &skuStock{SkuId: skuID, FactoryLeadDays: leadDays, UnitVolumeVU: vuMap[skuID]}
			allSkus[skuID] = sku
		}
		sku.UnfulfilledQty = qty
	}

	insightCount := 0
	transferCount := 0
	for _, sku := range allSkus {
		if sku.DailyBurnRate <= 0 {
			continue
		}
		lead := float64(sku.FactoryLeadDays)
		burn := sku.DailyBurnRate
		reorderPoint := burn*lead + burn*lead*safetyBufferMultiplier
		tte := float64(sku.CurrentStock) / burn
		urgency := classifyUrgency(tte, lead)
		if urgency == "STABLE" {
			continue
		}
		pending, err := e.hasPendingInsight(ctx, wh.WarehouseId, sku.SkuId)
		if err != nil {
			return insightCount, transferCount, err
		}
		if pending {
			continue
		}

		suggestedQty := computeSuggestedQty(*sku, reorderPoint)
		if e.EchelonTargetsEnabled {
			if target, ok, err := e.getEchelonTarget(ctx, wh.SupplierId, sku.SkuId, wh.WarehouseId); err == nil && ok {
				suggestedQty = computeSuggestedQtyWithEchelon(*sku, reorderPoint, target.TargetQty, true)
			}
		}
		seasonMul := seasonalMultiplierFor(e.Now())
		if seasonMul > 0 && seasonMul != 1.0 {
			suggestedQty = int64(math.Ceil(float64(suggestedQty) * seasonMul))
			if suggestedQty < 1 {
				suggestedQty = 1
			}
		}
		reason := "LOW_STOCK"
		if burn > float64(sku.CurrentStock)/lead {
			reason = "HIGH_VELOCITY"
		}
		breakdownJSON, _ := json.Marshal(map[string]any{
			"unfulfilled":         sku.UnfulfilledQty,
			"in_transit":          sku.InTransitQty,
			"current_stock":       sku.CurrentStock,
			"burn_rate_7d":        burn,
			"reorder_point":       reorderPoint,
			"seasonal_multiplier": seasonMul,
		})

		insightID := uuid.NewString()
		if err := e.writeInsight(ctx, insightID, wh, sku, burn, tte, suggestedQty, urgency, reason, string(breakdownJSON)); err != nil {
			e.Log.Error("replenishment.engine.write_insight_failed",
				"warehouse_id", wh.WarehouseId,
				"sku_id", sku.SkuId,
				"err", err,
			)
			continue
		}
		insightCount++

		if urgency == "CRITICAL" {
			if err := e.autoCreateTransfer(ctx, wh, insightID, sku.SkuId, suggestedQty, sku.UnitVolumeVU, wh.PrimaryFactoryId); err != nil {
				e.Log.Error("replenishment.engine.auto_transfer_failed",
					"warehouse_id", wh.WarehouseId,
					"sku_id", sku.SkuId,
					"err", err,
				)
			} else {
				transferCount++
			}
		} else if err := tryTouchlessApprove(ctx, e, wh.SupplierId, insightID, wh, sku, suggestedQty, urgency, reason, string(breakdownJSON)); err != nil {
			e.Log.Error("replenishment.engine.touchless_failed",
				"warehouse_id", wh.WarehouseId,
				"sku_id", sku.SkuId,
				"err", err,
			)
		}
	}
	return insightCount, transferCount, nil
}

func (e *Engine) hasPendingInsight(ctx context.Context, warehouseID, productID string) (bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT InsightId FROM ReplenishmentInsights
		      WHERE WarehouseId = @wh AND ProductId = @pid AND Status = 'PENDING'
		      LIMIT 1`,
		Params: map[string]any{"wh": warehouseID, "pid": productID},
	}
	iter := e.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	_, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return false, nil
	}
	return err == nil, err
}

func (e *Engine) writeInsight(
	ctx context.Context,
	insightID string,
	wh warehouseInfo,
	sku *skuStock,
	burn, tte float64,
	suggestedQty int64,
	urgency, reason, breakdown string,
) error {
	_, err := e.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdateMap("ReplenishmentInsights", map[string]any{
				"InsightId":         insightID,
				"WarehouseId":       wh.WarehouseId,
				"ProductId":         sku.SkuId,
				"SupplierId":        wh.SupplierId,
				"CurrentStock":      sku.CurrentStock,
				"DailyBurnRate":     burn,
				"TimeToEmptyDays":   tte,
				"SuggestedQuantity": suggestedQty,
				"UrgencyLevel":      urgency,
				"ReasonCode":        reason,
				"Status":            "PENDING",
				"TargetFactoryId":   wh.PrimaryFactoryId,
				"DemandBreakdown":   breakdown,
				"CreatedAt":         spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

func (e *Engine) autoCreateTransfer(ctx context.Context, wh warehouseInfo, insightID, skuID string, qty int64, vu float64, factoryID string) error {
	if vu <= 0 {
		vu = 1
	}
	transferID := uuid.NewString()
	totalVU := float64(qty) * vu
	_, err := e.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.UpdateMap("ReplenishmentInsights", map[string]any{
				"InsightId": insightID,
				"Status":    "APPROVED",
			}),
			spanner.InsertOrUpdateMap("FactoryInternalTransfers", map[string]any{
				"TransferId":      transferID,
				"FactoryId":       factoryID,
				"SupplierId":      wh.SupplierId,
				"WarehouseId":     wh.WarehouseId,
				"SourceInsightId": insightID,
				"State":           "APPROVED",
				"TotalVolumeVU":   totalVU,
				"CreatedAt":       spanner.CommitTimestamp,
				"UpdatedAt":       spanner.CommitTimestamp,
			}),
		}
		payload := events.WarehouseEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventWarehouseTransferCreated, Timestamp: e.Now().Format(time.RFC3339Nano)},
			TransferID:  transferID,
			WarehouseID: wh.WarehouseId,
			SupplierID:  wh.SupplierId,
			Units:       qty,
		}
		buf := &spannerTxnBuffer{}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, wh.WarehouseId, events.TopicMain, payload); emitErr != nil {
			return emitErr
		}
		mutations = append(mutations, outboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	return err
}

type spannerTxnBuffer struct {
	events []outbox.Event
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func outboxMutations(eventsList []outbox.Event) []*spanner.Mutation {
	mutations := make([]*spanner.Mutation, 0, len(eventsList))
	for _, event := range eventsList {
		createdAt := event.CreatedAt.UTC()
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		row := map[string]any{
			"EventId":       event.EventID,
			"AggregateType": event.AggregateType,
			"AggregateId":   event.AggregateID,
			"TopicName":     event.TopicName,
			"Payload":       event.Payload,
			"CreatedAt":     createdAt,
			"PublishedAt":   nil,
		}
		if event.PublishedAt != nil {
			row["PublishedAt"] = event.PublishedAt.UTC()
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
	}
	return mutations
}

func (e *Engine) getWarehouseStock(ctx context.Context, warehouseID, supplierID string) (map[string]int64, error) {
	result := make(map[string]int64)
	iter := e.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, QuantityOnHand FROM SupplierInventoryV2
		      WHERE SupplierId = @sid AND WarehouseId = @whId`,
		Params: map[string]any{"sid": supplierID, "whId": warehouseID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var productID string
		var qty int64
		if err := row.Columns(&productID, &qty); err != nil {
			continue
		}
		result[productID] = qty
	}
	return result, nil
}

func (e *Engine) get7DayBurnRates(ctx context.Context, warehouseID string) (map[string]float64, error) {
	totals, err := e.sumOrderLineQuantities(ctx, warehouseID, `Status = 'COMPLETED'`, true)
	if err != nil {
		return nil, err
	}
	result := make(map[string]float64, len(totals))
	for sku, qty := range totals {
		result[sku] = float64(qty) / float64(burnRateWindowDays)
	}
	return result, nil
}

func (e *Engine) getUnfulfilledDemand(ctx context.Context, warehouseID string) (map[string]int64, error) {
	return e.sumOrderLineQuantities(ctx, warehouseID, `Status IN ('PENDING', 'LOADED', 'IN_TRANSIT')`, false)
}

func (e *Engine) sumOrderLineQuantities(ctx context.Context, warehouseID, statusPredicate string, last7Days bool) (map[string]int64, error) {
	sql := fmt.Sprintf(`SELECT LineItemsJson FROM Orders
	                    WHERE WarehouseId = @whId AND %s`, statusPredicate)
	if last7Days {
		sql += ` AND UpdatedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)`
	}
	iter := e.Spanner.Single().Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: map[string]any{"whId": warehouseID},
	})
	defer iter.Stop()

	totals := make(map[string]int64)
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var raw []byte
		if err := row.Columns(&raw); err != nil {
			continue
		}
		items, err := parseLineItems(raw)
		if err != nil {
			continue
		}
		for _, item := range items {
			sku := strings.TrimSpace(item.SKU)
			if sku == "" || item.Quantity <= 0 {
				continue
			}
			totals[sku] += item.Quantity
		}
	}
	return totals, nil
}

func parseLineItems(raw []byte) ([]order.LineItem, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var items []order.LineItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (e *Engine) getUnitVolumes(ctx context.Context, supplierID string) (map[string]float64, error) {
	iter := e.Spanner.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT ProductId, UnitVolumeVU FROM Products WHERE SupplierId = @sid AND IsActive = true`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()

	result := make(map[string]float64)
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var productID string
		var vu float64
		if err := row.Columns(&productID, &vu); err != nil {
			continue
		}
		result[productID] = vu
	}
	return result, nil
}

func cronInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("REPLENISHMENT_CRON_INTERVAL_HOURS"))
	if raw == "" {
		return 4 * time.Hour
	}
	hours, err := strconv.Atoi(raw)
	if err != nil || hours < 1 {
		return 4 * time.Hour
	}
	return time.Duration(hours) * time.Hour
}
