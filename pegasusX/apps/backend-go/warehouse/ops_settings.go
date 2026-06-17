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
	row, err := s.spannerClient.Single().ReadRow(r.Context(), "Warehouses", spanner.Key{warehouseID},
		[]string{"WarehouseId", "Name", "RegionId", "DefaultOutOfStockPolicy", "ShowStockCountsToRetailers", "OperatingSchedule", "IsOnShift"})
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "warehouse_not_found"})
		return
	}
	var id, name string
	var regionID spanner.NullString
	var policy spanner.NullString
	var showCounts bool
	var schedule spanner.NullJSON
	var isOnShift bool
	if err := row.Columns(&id, &name, &regionID, &policy, &showCounts, &schedule, &isOnShift); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "decode_warehouse_failed"})
		return
	}
	var sched any
	if schedule.Valid {
		_ = json.Unmarshal([]byte(schedule.String()), &sched)
	}
	expressEnabled, expressStockFloor := readExpressOps(sched)
	writeJSON(w, http.StatusOK, map[string]any{
		"warehouse_id":                  id,
		"name":                          name,
		"region_id":                     regionID.StringVal,
		"default_out_of_stock_policy":   ResolveOutOfStockPolicy(policy.StringVal, ""),
		"show_stock_counts_to_retailers": showCounts,
		"operating_schedule":            sched,
		"is_on_shift":                   isOnShift,
		"ops_always_available":          true,
		"express_enabled":               expressEnabled,
		"express_stock_floor":           expressStockFloor,
	})
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
		DefaultOutOfStockPolicy      *string          `json:"default_out_of_stock_policy"`
		ShowStockCountsToRetailers *bool            `json:"show_stock_counts_to_retailers"`
		OperatingSchedule          *json.RawMessage `json:"operating_schedule"`
		RegionID                   *string          `json:"region_id"`
		IsOnShift                  *bool            `json:"is_on_shift"`
		ExpressEnabled             *bool            `json:"express_enabled"`
		ExpressStockFloor          *int64           `json:"express_stock_floor"`
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
