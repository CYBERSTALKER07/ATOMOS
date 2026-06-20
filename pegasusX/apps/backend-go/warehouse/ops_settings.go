package warehouse

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// HandleOpsSettings serves GET/PATCH /v1/warehouse/ops/settings (stock policy + operating hours).
func (s *Service) HandleOpsSettings(w http.ResponseWriter, r *http.Request) {
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "spanner_unavailable"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleGetOpsSettings(w, r, whID)
	case http.MethodPatch:
		s.handlePatchOpsSettings(w, r, whID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleGetOpsSettings(w http.ResponseWriter, r *http.Request, warehouseID string) {
	policy, err := LoadOpsPolicy(r.Context(), s.spannerClient, warehouseID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "warehouse_not_found"})
		return
	}
	row, err := s.spannerClient.Single().ReadRow(r.Context(), "Warehouses", spanner.Key{warehouseID},
		[]string{"OperatingSchedule"})
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "warehouse_not_found"})
		return
	}
	var schedule spanner.NullJSON
	if err := row.Columns(&schedule); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "decode_warehouse_failed"})
		return
	}
	var sched any
	if schedule.Valid {
		_ = json.Unmarshal([]byte(schedule.String()), &sched)
	}
	expressEnabled, expressStockFloor := readExpressOps(sched)
	writeJSON(w, http.StatusOK, opsPolicyToJSONMap(policy, sched, expressEnabled, expressStockFloor))
}

func (s *Service) handlePatchOpsSettings(w http.ResponseWriter, r *http.Request, warehouseID string) {
	body, ok := readMutationBody(w, r, 32*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseMutationReplay(r.Context(), key)
		}
	}()

	var req struct {
		DefaultOutOfStockPolicy    *string          `json:"default_out_of_stock_policy"`
		ShowStockCountsToRetailers *bool            `json:"show_stock_counts_to_retailers"`
		OperatingSchedule          *json.RawMessage `json:"operating_schedule"`
		RegionID                   *string          `json:"region_id"`
		IsOnShift                  *bool            `json:"is_on_shift"`
		ExpressEnabled             *bool            `json:"express_enabled"`
		ExpressStockFloor          *int64           `json:"express_stock_floor"`
		PreorderMinLeadDays        *int64           `json:"preorder_min_lead_days"`
		PreorderMaxLeadDays        *int64           `json:"preorder_max_lead_days"`
		OrderLineMinQuantity       *int64           `json:"order_line_min_quantity"`
		OrderLineMaxQuantity       *int64           `json:"order_line_max_quantity"`
		ClearOrderLineMinQuantity  *bool            `json:"clear_order_line_min_quantity"`
		ClearOrderLineMaxQuantity  *bool            `json:"clear_order_line_max_quantity"`
		DeliveryFeeRules         *json.RawMessage `json:"delivery_fee_rules"`
		ClearDeliveryFeeRules      *bool            `json:"clear_delivery_fee_rules"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	update := map[string]any{
		"WarehouseId": warehouseID,
		"UpdatedAt":   spanner.CommitTimestamp,
	}
	if req.DefaultOutOfStockPolicy != nil {
		p := ResolveOutOfStockPolicy(*req.DefaultOutOfStockPolicy, "")
		if p != OutOfStockPolicyReject && p != OutOfStockPolicyAcceptBackorder {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_out_of_stock_policy"})
			return
		}
		update["DefaultOutOfStockPolicy"] = p
	}
	if req.ShowStockCountsToRetailers != nil {
		update["ShowStockCountsToRetailers"] = *req.ShowStockCountsToRetailers
	}
	if req.OperatingSchedule != nil {
		update["OperatingSchedule"] = spanner.NullJSON{Value: *req.OperatingSchedule, Valid: true}
	}
	if req.RegionID != nil {
		update["RegionId"] = strings.TrimSpace(*req.RegionID)
	}
	if req.IsOnShift != nil {
		update["IsOnShift"] = *req.IsOnShift
	}
	if req.PreorderMinLeadDays != nil || req.PreorderMaxLeadDays != nil {
		current, err := LoadOpsPolicy(r.Context(), s.spannerClient, warehouseID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "warehouse_not_found"})
			return
		}
		minLead := current.PreorderMinLeadDays
		maxLead := current.PreorderMaxLeadDays
		if req.PreorderMinLeadDays != nil {
			minLead = *req.PreorderMinLeadDays
		}
		if req.PreorderMaxLeadDays != nil {
			maxLead = *req.PreorderMaxLeadDays
		}
		minLead, maxLead, err = normalizePreorderLeadDays(minLead, maxLead)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		update["PreorderMinLeadDays"] = minLead
		update["PreorderMaxLeadDays"] = maxLead
	}
	if req.ClearOrderLineMinQuantity != nil && *req.ClearOrderLineMinQuantity {
		update["OrderLineMinQuantity"] = nil
	} else if req.OrderLineMinQuantity != nil {
		update["OrderLineMinQuantity"] = *req.OrderLineMinQuantity
	}
	if req.ClearOrderLineMaxQuantity != nil && *req.ClearOrderLineMaxQuantity {
		update["OrderLineMaxQuantity"] = nil
	} else if req.OrderLineMaxQuantity != nil {
		update["OrderLineMaxQuantity"] = *req.OrderLineMaxQuantity
	}
	if minQty, ok := update["OrderLineMinQuantity"]; ok && minQty != nil {
		maxQty := update["OrderLineMaxQuantity"]
		var minPtr, maxPtr *int64
		if v, ok := minQty.(int64); ok {
			minPtr = &v
		}
		if v, ok := maxQty.(int64); ok {
			maxPtr = &v
		}
		if err := validateOrderLineLimits(minPtr, maxPtr); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}
	if req.ClearDeliveryFeeRules != nil && *req.ClearDeliveryFeeRules {
		update["DeliveryFeeRules"] = nil
	} else if req.DeliveryFeeRules != nil {
		rules, err := ParseDeliveryFeeRulesJSON(*req.DeliveryFeeRules)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		update["DeliveryFeeRules"] = spanner.NullJSON{Value: *req.DeliveryFeeRules, Valid: true}
		_ = rules
	}
	if req.ExpressEnabled != nil || req.ExpressStockFloor != nil {
		schedJSON := loadOperatingScheduleJSON(r.Context(), s.spannerClient, warehouseID)
		express := map[string]any{}
		if existing, ok := schedJSON["express"].(map[string]any); ok {
			express = existing
		}
		if req.ExpressEnabled != nil {
			express["enabled"] = *req.ExpressEnabled
		}
		if req.ExpressStockFloor != nil {
			express["stock_floor"] = *req.ExpressStockFloor
		}
		schedJSON["express"] = express
		raw, _ := json.Marshal(schedJSON)
		update["OperatingSchedule"] = spanner.NullJSON{Value: json.RawMessage(raw), Valid: true}
	}

	_, err := s.spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("Warehouses", update)})
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_warehouse_settings_failed"})
		return
	}

	claims, _ := auth.FromContext(r.Context())
	s.log.InfoContext(r.Context(), "warehouse.settings.updated",
		"warehouse_id", warehouseID,
		"actor", strings.TrimSpace(claims.Subject),
	)

	resp := map[string]string{"status": "updated"}
	respBytes, _ := json.Marshal(resp)
	s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSON(w, http.StatusOK, resp)
}
