package factory

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// PlanPredictiveTransfers builds SYSTEM_PREDICTED drafts from DemandForecastBaseline
// projected breaches. X AIPredictions is ORDER-grain (PENDING, no RetailerId /
// TriggerDate / AIPredictionItems) — that table is not used here.
func PlanPredictiveTransfers(mode string, breached []breachedSKU, pick factoryPicker, acquire lockFn) []plannedTransfer {
	mode = normalizeNetworkMode(mode)
	if mode == NetworkModeManualOnly {
		return nil
	}
	if acquire == nil {
		acquire = func(_, _, _, _ string) bool { return true }
	}
	type key struct{ WarehouseID, FactoryID string }
	grouped := map[key]*plannedTransfer{}
	order := make([]key, 0)
	for _, b := range breached {
		if b.Deficit <= 0 || strings.TrimSpace(b.WarehouseID) == "" {
			continue
		}
		factoryID, err := pick(b.SupplierID, b.WarehouseID, b.ProductID, mode)
		if err != nil || strings.TrimSpace(factoryID) == "" {
			continue
		}
		if !acquire(b.SupplierID, b.WarehouseID, b.ProductID, factoryID) {
			continue
		}
		k := key{WarehouseID: b.WarehouseID, FactoryID: factoryID}
		row, ok := grouped[k]
		if !ok {
			row = &plannedTransfer{
				SupplierID:  b.SupplierID,
				WarehouseID: b.WarehouseID,
				FactoryID:   factoryID,
				Source:      TransferSourcePredicted,
				State:       TransferStateCreated,
			}
			grouped[k] = row
			order = append(order, k)
		}
		vu := b.UnitVU
		if vu <= 0 {
			vu = 1
		}
		row.TotalVU += float64(b.Deficit) * vu
		row.ProductIDs = append(row.ProductIDs, b.ProductID)
	}
	out := make([]plannedTransfer, 0, len(order))
	for _, k := range order {
		out = append(out, *grouped[k])
	}
	return out
}

func projectedDeficit(currentQty, safetyLevel, predictedQty int64) int64 {
	projected := currentQty - predictedQty
	if projected > safetyLevel {
		return 0
	}
	deficit := safetyLevel - projected
	if deficit <= 0 {
		return predictedQty
	}
	return deficit
}

func (p *PlanningService) findPredictedBreaches(ctx context.Context, supplierID string, horizonDays int64) ([]breachedSKU, error) {
	if horizonDays <= 0 {
		horizonDays = 3
	}
	sql := `SELECT i.SupplierId, i.WarehouseId, i.ProductId,
	               i.QuantityOnHand, i.QuantityReserved, i.ReorderThreshold,
	               COALESCE(d.pred, 0) AS predicted
	        FROM SupplierInventoryV2 i
	        JOIN (
	          SELECT WarehouseId, ProductId, SUM(BaselineQty) AS pred
	          FROM DemandForecastBaseline
	          WHERE SupplierId = @sid
	            AND ForecastDate >= CURRENT_DATE()
	            AND ForecastDate < DATE_ADD(CURRENT_DATE(), INTERVAL @days DAY)
	          GROUP BY WarehouseId, ProductId
	        ) d ON d.WarehouseId = i.WarehouseId AND d.ProductId = i.ProductId
	        WHERE i.SupplierId = @sid AND i.ReorderThreshold > 0
	        LIMIT 1000`
	iter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: map[string]any{"sid": supplierID, "days": horizonDays},
	})
	defer iter.Stop()
	vu, _ := p.unitVolumes(ctx, supplierID)
	var out []breachedSKU
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var b breachedSKU
		var reserved, predicted int64
		if err := row.Columns(&b.SupplierID, &b.WarehouseID, &b.ProductID, &b.CurrentQty, &reserved, &b.SafetyLevel, &predicted); err != nil {
			continue
		}
		available := b.CurrentQty - reserved
		if available < 0 {
			available = 0
		}
		b.CurrentQty = available
		b.Deficit = projectedDeficit(available, b.SafetyLevel, predicted)
		if b.Deficit <= 0 {
			continue
		}
		b.UnitVU = vu[b.ProductID]
		out = append(out, b)
	}
	return out, nil
}

// RunPredictivePushForSupplier writes SYSTEM_PREDICTED transfers from DemandForecastBaseline.
func (p *PlanningService) RunPredictivePushForSupplier(ctx context.Context, supplierID, source string) (int, int, error) {
	if p == nil || p.Spanner == nil {
		return 0, 0, errors.New("planning_unavailable")
	}
	if !PlanningEnabled() {
		return 0, 0, errors.New("factory_planning_disabled")
	}
	supplierID = strings.TrimSpace(supplierID)
	mode, _ := p.GetNetworkMode(ctx, supplierID)
	if mode == NetworkModeManualOnly {
		return 0, 0, nil
	}
	breached, err := p.findPredictedBreaches(ctx, supplierID, 3)
	if err != nil {
		return 0, 0, err
	}
	planned := PlanPredictiveTransfers(mode, breached, p.picker(ctx), p.locker(ctx))
	n, err := p.insertPlannedTransfers(ctx, planned)
	return n, len(breached), err
}

// HandlePredictivePush serves POST /v1/supplier/planning/predictive-push.
func (p *PlanningService) HandlePredictivePush(w http.ResponseWriter, r *http.Request, supplierID string) {
	if r.Method != http.MethodPost {
		writePlanningJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if !PlanningEnabled() {
		writePlanningJSON(w, http.StatusConflict, map[string]string{"error": "factory_planning_disabled"})
		return
	}
	transfers, skus, err := p.RunPredictivePushForSupplier(r.Context(), supplierID, "MANUAL")
	if err != nil {
		writePlanningJSON(w, http.StatusInternalServerError, map[string]string{"error": "predictive_push_failed"})
		return
	}
	writePlanningJSON(w, http.StatusOK, map[string]any{
		"transfers":    transfers,
		"skus":         skus,
		"source":       "MANUAL",
		"grain":        "demand_forecast_baseline",
		"not_from":     "AIPredictions",
	})
}
