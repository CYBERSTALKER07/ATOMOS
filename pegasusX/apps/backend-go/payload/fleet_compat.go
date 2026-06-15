package payload

import (
	"encoding/json"
	"net/http"
	"strings"
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
	}, nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "reassign_failed"})
		return
	}

	if reassigned > 0 {
		s.invalidatePayloadKeys(r.Context(), payloadOrderListKey(s.supplierID))
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
