package payload

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// HandleFleetReassign serves POST /v1/fleet/reassign (native payload terminal contract).
func (s *Service) HandleFleetReassign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	var req struct {
		OrderIDs   []string `json:"order_ids"`
		NewRouteID string   `json:"new_route_id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.NewRouteID = strings.TrimSpace(req.NewRouteID)
	if len(req.OrderIDs) == 0 || req.NewRouteID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_ids_and_new_route_id_required"})
		return
	}

	reassigned := 0
	conflicts := make([]map[string]string, 0)
	now := s.now().Format("2006-01-02T15:04:05Z07:00")

	// B2: emit ORDER_REASSIGNED on the bus (was emit=nil silent path).
	err = s.apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		for _, orderID := range req.OrderIDs {
			orderID = strings.TrimSpace(orderID)
			if orderID == "" {
				continue
			}
			oIdx := s.findOrderIndexLocked(orderID)
			if oIdx < 0 {
				conflicts = append(conflicts, map[string]string{"order_id": orderID, "reason": "order_not_found"})
				continue
			}
			if s.orders[oIdx].RouteID == req.NewRouteID {
				conflicts = append(conflicts, map[string]string{"order_id": orderID, "reason": "order_already_assigned"})
				continue
			}
			s.orders[oIdx].RouteID = req.NewRouteID
			s.orders[oIdx].UpdatedAt = now
			reassigned++
		}
		return nil
	}, func(txn outbox.TxnBuffer) error {
		if reassigned == 0 {
			return nil
		}
		return outbox.EmitJSON(r.Context(), txn, events.AggregateRoute, req.NewRouteID, events.TopicMain, map[string]any{
			"type":          events.EventOrderReassigned,
			"new_route_id":  req.NewRouteID,
			"reassigned":    reassigned,
			"supplier_id":   s.resolveSupplierScope(r.Context()),
			"warehouse_id":  s.resolveWarehouseScope(r.Context()),
			"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		})
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "reassign_failed"})
		return
	}

	if reassigned > 0 {
		s.invalidatePayloadKeys(r.Context(), payloadOrderListKey(s.resolveSupplierScope(r.Context())))
		s.broadcastPayloadEvent(r.Context(), "ORDER_REASSIGNED", map[string]any{
			"new_route_id": req.NewRouteID,
			"reassigned":   reassigned,
		})
	}

	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"conflicts":    conflicts,
		"total":        len(req.OrderIDs),
		"reassigned":   reassigned,
		"new_route_id": req.NewRouteID,
	})
}
