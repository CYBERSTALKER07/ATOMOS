package replenishment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// tryTouchlessApprove auto-approves insights that match supplier policy.
func tryTouchlessApprove(
	ctx context.Context,
	e *Engine,
	supplierID, insightID string,
	wh warehouseInfo,
	sku *skuStock,
	qty int64,
	urgency, reasonCode, breakdown string,
) error {
	if e == nil || e.Spanner == nil {
		return nil
	}
	policy, err := LoadPolicy(ctx, e.Spanner, supplierID)
	if err != nil {
		return err
	}
	approvedToday, err := CountAutoApprovedToday(ctx, e.Spanner, supplierID, e.Now())
	if err != nil {
		return err
	}
	if approvedToday+qty > policy.MaxDailyTransferUnits {
		return nil
	}

	allow := false
	switch {
	case strings.EqualFold(reasonCode, "PREDICTIVE_PUSH") && policy.AutoApprovePredictivePush:
		allow = true
	case strings.EqualFold(urgency, "STABLE") && policy.AutoApproveStable:
		allow = true
	case strings.EqualFold(reasonCode, "MEIO_NETWORK") && policy.AutoApproveStable:
		allow = true
	}
	if !allow || strings.EqualFold(urgency, "CRITICAL") {
		return nil
	}
	factoryID := resolveInsightFactory(wh)
	if factoryID == "" {
		return nil
	}

	transferID := uuid.NewString()
	totalVU := float64(qty) * sku.UnitVolumeVU
	if totalVU <= 0 {
		totalVU = float64(qty)
	}
	if totalVU <= 0 {
		totalVU = 1
	}

	_, err = e.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.UpdateMap("ReplenishmentInsights", map[string]any{
				"InsightId": insightID,
				"Status":    "APPROVED",
			}),
			spanner.InsertOrUpdateMap("FactoryInternalTransfers", map[string]any{
				"TransferId":    transferID,
				"FactoryId":     factoryID,
				"SupplierId":    supplierID,
				"WarehouseId":   wh.WarehouseId,
				"State":         "APPROVED",
				"TotalVolumeVU": totalVU,
				"CreatedAt":     spanner.CommitTimestamp,
				"UpdatedAt":     spanner.CommitTimestamp,
			}),
		}
		buf := &spannerTxnBuffer{}
		planPayload := events.PlanningEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventReplenishmentAutoApproved,
				Timestamp: e.Now().UTC().Format(time.RFC3339Nano),
			},
			SupplierID:  supplierID,
			WarehouseID: wh.WarehouseId,
			InsightID:   insightID,
			ProductID:   sku.SkuId,
			BaselineQty: qty,
			Action:      reasonCode,
		}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregatePlanning, supplierID, events.TopicMain, planPayload); emitErr != nil {
			return emitErr
		}
		transferPayload := events.WarehouseEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventWarehouseTransferCreated, Timestamp: e.Now().UTC().Format(time.RFC3339Nano)},
			TransferID:  transferID,
			WarehouseID: wh.WarehouseId,
			SupplierID:  supplierID,
			Units:       qty,
		}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, wh.WarehouseId, events.TopicMain, transferPayload); emitErr != nil {
			return emitErr
		}
		mutations = append(mutations, outboxMutations(buf.events)...)
		_ = breakdown // retained for audit in insight row
		return txn.BufferWrite(mutations)
	})
	return err
}

func resolveInsightFactory(wh warehouseInfo) string {
	if factoryID := strings.TrimSpace(wh.PrimaryFactoryId); factoryID != "" {
		return factoryID
	}
	return strings.TrimSpace(wh.SecondaryFactoryId)
}

// FulfillApprovedInsight opens a factory transfer for a touchless- or agent-approved insight.
func FulfillApprovedInsight(ctx context.Context, client *spanner.Client, insightID string, now func() time.Time) (transferID string, err error) {
	if client == nil || strings.TrimSpace(insightID) == "" {
		return "", fmt.Errorf("fulfill insight: invalid input")
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	row, err := loadInsightFulfillRow(ctx, client, insightID)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(row.Status, "PENDING") {
		return "", fmt.Errorf("insight_not_pending")
	}
	factoryID := strings.TrimSpace(row.TargetFactoryID)
	if factoryID == "" {
		factoryID = resolveInsightFactory(warehouseInfo{
			PrimaryFactoryId:   row.PrimaryFactoryID,
			SecondaryFactoryId: row.SecondaryFactoryID,
		})
	}
	if factoryID == "" {
		return "", fmt.Errorf("insight_no_target_factory")
	}
	transferID = uuid.NewString()
	totalVU := float64(row.SuggestedQuantity)
	if totalVU <= 0 {
		totalVU = 1
	}
	ts := now().UTC().Format(time.RFC3339Nano)
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.UpdateMap("ReplenishmentInsights", map[string]any{
				"InsightId": insightID,
				"Status":    "APPROVED",
			}),
			spanner.InsertOrUpdateMap("FactoryInternalTransfers", map[string]any{
				"TransferId":    transferID,
				"FactoryId":     factoryID,
				"SupplierId":    row.SupplierID,
				"WarehouseId":   row.WarehouseID,
				"State":         "APPROVED",
				"TotalVolumeVU": totalVU,
				"CreatedAt":     spanner.CommitTimestamp,
				"UpdatedAt":     spanner.CommitTimestamp,
			}),
		}
		buf := &spannerTxnBuffer{}
		planPayload := events.PlanningEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventReplenishmentAutoApproved, Timestamp: ts},
			SupplierID:  row.SupplierID,
			WarehouseID: row.WarehouseID,
			InsightID:   insightID,
			ProductID:   row.ProductID,
			BaselineQty: row.SuggestedQuantity,
			Action:      row.ReasonCode,
		}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregatePlanning, row.SupplierID, events.TopicMain, planPayload); emitErr != nil {
			return emitErr
		}
		transferPayload := events.WarehouseEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventWarehouseTransferCreated, Timestamp: ts},
			TransferID:  transferID,
			WarehouseID: row.WarehouseID,
			SupplierID:  row.SupplierID,
			Units:       row.SuggestedQuantity,
		}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, row.WarehouseID, events.TopicMain, transferPayload); emitErr != nil {
			return emitErr
		}
		mutations = append(mutations, outboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	return transferID, err
}

type insightFulfillRow struct {
	WarehouseID        string
	ProductID          string
	SupplierID         string
	SuggestedQuantity  int64
	Status             string
	ReasonCode         string
	TargetFactoryID    string
	PrimaryFactoryID   string
	SecondaryFactoryID string
}

func loadInsightFulfillRow(ctx context.Context, client *spanner.Client, insightID string) (insightFulfillRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ri.WarehouseId, ri.ProductId, ri.SupplierId, ri.SuggestedQuantity, ri.Status,
		             COALESCE(ri.ReasonCode, ''), COALESCE(ri.TargetFactoryId, ''),
		             COALESCE(w.PrimaryFactoryId, ''), COALESCE(w.SecondaryFactoryId, '')
		      FROM ReplenishmentInsights ri
		      LEFT JOIN Warehouses w ON ri.WarehouseId = w.WarehouseId
		      WHERE ri.InsightId = @iid`,
		Params: map[string]any{"iid": insightID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return insightFulfillRow{}, err
	}
	var out insightFulfillRow
	if err := row.Columns(
		&out.WarehouseID, &out.ProductID, &out.SupplierID, &out.SuggestedQuantity, &out.Status,
		&out.ReasonCode, &out.TargetFactoryID, &out.PrimaryFactoryID, &out.SecondaryFactoryID,
	); err != nil {
		return insightFulfillRow{}, err
	}
	return out, nil
}

// TouchlessEligible reports whether an insight would auto-approve under policy.
func TouchlessEligible(policy Policy, urgency, reasonCode string, qty int64, approvedToday int64) bool {
	if approvedToday+qty > policy.MaxDailyTransferUnits {
		return false
	}
	if strings.EqualFold(urgency, "CRITICAL") {
		return false
	}
	if strings.EqualFold(reasonCode, "PREDICTIVE_PUSH") {
		return policy.AutoApprovePredictivePush
	}
	return policy.AutoApproveStable
}

// ParseDemandBreakdown decodes insight breakdown JSON for warehouse portal display.
func ParseDemandBreakdown(raw string) map[string]any {
	out := map[string]any{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}
