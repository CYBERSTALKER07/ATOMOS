package replenishment

import (
	"context"
	"math"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/segment"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	echelonForward       = "FORWARD"
	echelonSourceMEIO    = "MEIO"
	defaultEchelonHorizon = 14
)

// EchelonTargetRow is a persisted per-warehouse stock target.
type EchelonTargetRow struct {
	SupplierID      string
	Sku             string
	WarehouseID     string
	Echelon         string
	TargetQty       int64
	SafetyQty       int64
	ServiceLevelBps int64
	HorizonDays     int64
	ComputedAt      time.Time
	Source          string
}

// ComputeEchelonTarget derives target and safety quantities from burn and service level.
func ComputeEchelonTarget(burnRate float64, horizonDays int64, serviceLevelBps int64) (targetQty, safetyQty int64) {
	if burnRate <= 0 {
		return 0, 0
	}
	if horizonDays <= 0 {
		horizonDays = defaultEchelonHorizon
	}
	if serviceLevelBps <= 0 {
		serviceLevelBps = 9500
	}
	factor := float64(serviceLevelBps) / 10000.0
	target := math.Ceil(burnRate * float64(horizonDays) * factor)
	safety := math.Ceil(burnRate * float64(defaultLeadTimeDays) * safetyBufferMultiplier)
	return int64(target), int64(safety)
}

// SuggestedQtyFromTarget returns reorder suggestion from echelon target when enabled.
func SuggestedQtyFromTarget(targetQty int64, effectiveStock float64) int64 {
	suggested := int64(math.Ceil(float64(targetQty) - effectiveStock))
	if suggested < 0 {
		return 0
	}
	return suggested
}

func (e *Engine) resolveServiceLevelBps(ctx context.Context, supplierID string) int64 {
	if e == nil || e.SegmentSvc == nil {
		return 9500
	}
	policy, err := e.SegmentSvc.ResolveLineContext(ctx, supplierID, "", segment.SkuWildcard)
	if err != nil {
		return 9500
	}
	if policy.Policy.TargetServiceLevelBps > 0 {
		return policy.Policy.TargetServiceLevelBps
	}
	return 9500
}

func (e *Engine) upsertEchelonTargets(ctx context.Context, supplierID string, balances []meiSkuBalance) error {
	if e == nil || e.Spanner == nil || !e.EchelonTargetsEnabled {
		return nil
	}
	now := e.Now()
	horizon := int64(defaultEchelonHorizon)
	mutations := make([]*spanner.Mutation, 0, len(balances))
	buf := &spannerTxnBuffer{}
	for _, b := range balances {
		if b.burnRate <= 0 {
			continue
		}
		slb := e.resolveServiceLevelBpsForSKU(ctx, supplierID, b.skuID)
		targetQty, safetyQty := ComputeEchelonTarget(b.burnRate, horizon, slb)
		if targetQty <= 0 {
			continue
		}
		row := EchelonTargetRow{
			SupplierID:      supplierID,
			Sku:             b.skuID,
			WarehouseID:     b.warehouseID,
			Echelon:         echelonForward,
			TargetQty:       targetQty,
			SafetyQty:       safetyQty,
			ServiceLevelBps: slb,
			HorizonDays:     horizon,
			ComputedAt:      now,
			Source:          echelonSourceMEIO,
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("EchelonTargets", map[string]any{
			"SupplierId":      row.SupplierID,
			"Sku":             row.Sku,
			"WarehouseId":     row.WarehouseID,
			"Echelon":         row.Echelon,
			"TargetQty":       row.TargetQty,
			"SafetyQty":       row.SafetyQty,
			"ServiceLevelBps": row.ServiceLevelBps,
			"HorizonDays":     row.HorizonDays,
			"ComputedAt":      row.ComputedAt,
			"Source":          row.Source,
		}))
		if err := outbox.EmitJSON(ctx, buf, events.AggregatePlanning, supplierID, events.TopicMain, map[string]any{
			"type":        "echelon.target.updated",
			"timestamp":   now.Format(time.RFC3339Nano),
			"supplier_id": supplierID,
			"sku":         row.Sku,
			"warehouse_id": row.WarehouseID,
			"target_qty":  row.TargetQty,
			"source":      row.Source,
		}); err != nil {
			return err
		}
	}
	if len(mutations) == 0 {
		return nil
	}
	mutations = append(mutations, outboxMutations(buf.events)...)
	_, err := e.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite(mutations)
	})
	return err
}

func (e *Engine) getEchelonTarget(ctx context.Context, supplierID, sku, warehouseID string) (EchelonTargetRow, bool, error) {
	if e == nil || e.Spanner == nil {
		return EchelonTargetRow{}, false, nil
	}
	row, err := e.Spanner.Single().ReadRow(ctx, "EchelonTargets", spanner.Key{supplierID, sku, warehouseID, echelonForward},
		[]string{"SupplierId", "Sku", "WarehouseId", "Echelon", "TargetQty", "SafetyQty", "ServiceLevelBps", "HorizonDays", "ComputedAt", "Source"})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return EchelonTargetRow{}, false, nil
		}
		return EchelonTargetRow{}, false, err
	}
	var target EchelonTargetRow
	if err := row.Columns(&target.SupplierID, &target.Sku, &target.WarehouseID, &target.Echelon,
		&target.TargetQty, &target.SafetyQty, &target.ServiceLevelBps, &target.HorizonDays, &target.ComputedAt, &target.Source); err != nil {
		return EchelonTargetRow{}, false, err
	}
	return target, true, nil
}

func (e *Engine) sumEchelonTargetsByWarehouse(ctx context.Context, supplierID, warehouseID string) (targetSum, onHand int64, err error) {
	if e == nil || e.Spanner == nil {
		return 0, 0, nil
	}
	iter := e.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT SUM(TargetQty) FROM EchelonTargets
		      WHERE SupplierId = @sid AND WarehouseId = @wid AND Echelon = @echelon`,
		Params: map[string]any{"sid": supplierID, "wid": warehouseID, "echelon": echelonForward},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	var sum spanner.NullInt64
	if err := row.Column(0, &sum); err != nil {
		return 0, 0, err
	}
	if sum.Valid {
		targetSum = sum.Int64
	}
	return targetSum, 0, nil
}

// resolveServiceLevelBpsForSKU uses default segment C and velocity class B.
func (e *Engine) resolveServiceLevelBpsForSKU(ctx context.Context, supplierID, sku string) int64 {
	if e == nil || e.SegmentSvc == nil {
		return 9500
	}
	ctxLine, err := e.SegmentSvc.ResolveLineContext(ctx, supplierID, "", sku)
	if err != nil {
		return 9500
	}
	if ctxLine.Policy.TargetServiceLevelBps > 0 {
		return ctxLine.Policy.TargetServiceLevelBps
	}
	return 9500
}

func computeSuggestedQtyWithEchelon(stock skuStock, reorderPoint float64, targetQty int64, enabled bool) int64 {
	effectiveStock := float64(stock.CurrentStock) + float64(stock.InTransitQty) - float64(stock.UnfulfilledQty)
	if enabled && targetQty > 0 {
		if qty := SuggestedQtyFromTarget(targetQty, effectiveStock); qty > 0 {
			return qty
		}
	}
	return computeSuggestedQty(stock, reorderPoint)
}
