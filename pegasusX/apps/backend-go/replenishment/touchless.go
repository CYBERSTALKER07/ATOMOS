package replenishment

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
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

	_, err = e.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.UpdateMap("ReplenishmentInsights", map[string]any{
				"InsightId": insightID,
				"Status":    "APPROVED",
			}),
		}
		payload := events.PlanningEvent{
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
		buf := &spannerTxnBuffer{}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregatePlanning, supplierID, events.TopicMain, payload); emitErr != nil {
			return emitErr
		}
		mutations = append(mutations, outboxMutations(buf.events)...)
		_ = breakdown // retained for audit in insight row
		return txn.BufferWrite(mutations)
	})
	return err
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
