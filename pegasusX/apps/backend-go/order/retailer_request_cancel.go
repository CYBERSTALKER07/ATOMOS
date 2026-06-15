package order

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// HandleRetailerRequestCancel serves POST /v1/orders/request-cancel for in-flight
// orders where immediate cancellation requires supplier/driver coordination.
func (s *Service) HandleRetailerRequestCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleRetailer {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
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
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req struct {
		OrderID    string `json:"order_id"`
		RetailerID string `json:"retailer_id"`
		Reason     string `json:"reason,omitempty"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}
	retailerID := strings.TrimSpace(req.RetailerID)
	if retailerID == "" {
		retailerID = strings.TrimSpace(claims.Subject)
	}
	if retailerID == "" || retailerID != strings.TrimSpace(claims.Subject) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	current, found, err := s.repo.GetOrder(r.Context(), orderID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	if current.RetailerID != retailerID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if current.Status == StatusCancelRequested {
		idemCommitted = true
		s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
			"status":       "cancel_requested",
			"order_id":     current.OrderID,
			"order_status": string(current.Status),
		})
		return
	}
	if !canRequestCancel(current.Status) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_status_transition"})
		return
	}

	prevStatus := current.Status
	reason := strings.TrimSpace(req.Reason)
	current.Status = StatusCancelRequested
	current.UpdatedAt = s.now()

	if err := s.repo.UpdateOrder(r.Context(), current, nil, func(txn outbox.TxnBuffer) error {
		ts := current.UpdatedAt.Format(time.RFC3339Nano)
		if err := outbox.EmitJSON(r.Context(), txn, events.AggregateOrder, current.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:      events.BaseEvent{Type: "CANCEL_REQUESTED", Timestamp: ts},
			OrderID:        current.OrderID,
			SupplierID:     current.SupplierID,
			RetailerID:     current.RetailerID,
			DriverID:       current.DriverID,
			PreviousStatus: string(prevStatus),
			Status:         string(current.Status),
			Reason:         reason,
			ActorRole:      string(auth.RoleRetailer),
			ActorID:        retailerID,
		}); err != nil {
			return err
		}
		return outbox.EmitJSON(r.Context(), txn, events.AggregateOrder, current.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:      events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: ts},
			OrderID:        current.OrderID,
			SupplierID:     current.SupplierID,
			RetailerID:     current.RetailerID,
			DriverID:       current.DriverID,
			PreviousStatus: string(prevStatus),
			Status:         string(current.Status),
			Reason:         reason,
			ActorRole:      string(auth.RoleRetailer),
			ActorID:        retailerID,
		})
	}); err != nil {
		s.log.ErrorContext(r.Context(), "request cancel failed", "order_id", orderID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_failed"})
		return
	}

	s.afterOrderMutation(r.Context(), current)

	idemCommitted = true
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"status":          "cancel_requested",
		"order_id":        current.OrderID,
		"previous_status": string(prevStatus),
		"order_status":    string(current.Status),
		"version":         current.Version,
		"updated_at":      current.UpdatedAt.Format(time.RFC3339Nano),
	})
}

func canRequestCancel(status Status) bool {
	switch status {
	case StatusLoaded, StatusInTransit, StatusArrived:
		return true
	default:
		return false
	}
}
