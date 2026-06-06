package driver

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func demoFleetOrders(driverID string) []map[string]any {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	return []map[string]any{
		{
			"id": "ord_factory_1", "retailer_id": "ret_demo", "retailer_name": "Demo Retailer",
			"state": "IN_TRANSIT", "total_amount": int64(24000),
			"delivery_address": "Tashkent, Chilonzor",
			"latitude": 41.285, "longitude": 69.203,
			"qr_token": "demo-token-ord-1", "payment_gateway": "CASH",
			"created_at": now, "updated_at": now,
			"route_id": "route_veh_factory_1", "sequence_index": 1,
			"items": []map[string]any{
				{"product_id": "prod-milk-1l", "product_name": "Whole Milk 1L", "quantity": 2, "unit_price": 12000},
			},
		},
		{
			"id": "ord_factory_2", "retailer_id": "ret_demo", "retailer_name": "Demo Retailer 2",
			"state": "DISPATCHED", "total_amount": int64(18000),
			"delivery_address": "Tashkent, Yunusabad",
			"latitude": 41.335, "longitude": 69.288,
			"qr_token": "demo-token-ord-2", "payment_gateway": "CASH",
			"created_at": now, "updated_at": now,
			"route_id": "route_veh_factory_1", "sequence_index": 2,
			"items": []map[string]any{},
		},
	}
}

// HandleFleetOrders serves GET /v1/fleet/orders.
func (s *Service) HandleFleetOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.orderList != nil {
		orders, err := s.orderList(r.Context(), driverID)
		if err != nil {
			s.log.ErrorContext(r.Context(), "fleet orders query failed", "err", err, "driver_id", driverID)
			w.Header().Set("Retry-After", "30")
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "fleet_orders_unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, orders)
		return
	}
	if allowDriverDemoFallback() {
		writeJSON(w, http.StatusOK, demoFleetOrders(driverID))
		return
	}
	w.Header().Set("Retry-After", "30")
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "fleet_orders_unavailable"})
}

// HandleWSAck serves POST /v1/ws/ack.
func (s *Service) HandleWSAck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	var req struct {
		CommandID string `json:"command_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "command_id": strings.TrimSpace(req.CommandID)})
}

// HandleDriverDepart serves POST /v1/fleet/driver/depart.
func (s *Service) HandleDriverDepart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "departed"})
}

// HandleDriverReturnComplete serves POST /v1/fleet/driver/return-complete.
func (s *Service) HandleDriverReturnComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "returned"})
}

// HandleOrderDeliver serves POST /v1/order/deliver.
func (s *Service) HandleOrderDeliver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "message": "delivered", "new_state": "COMPLETED"})
}

// HandleOrderValidateQR serves POST /v1/order/validate-qr.
func (s *Service) HandleOrderValidateQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"valid": true, "message": "ok"})
}

// HandleOrderConfirmOffload serves POST /v1/order/confirm-offload.
func (s *Service) HandleOrderConfirmOffload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "offloaded", "state": "AWAITING_PAYMENT"})
}

// HandleOrderComplete serves POST /v1/order/complete.
func (s *Service) HandleOrderComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
}

// HandleOrderCollectCash serves POST /v1/order/collect-cash.
func (s *Service) HandleOrderCollectCash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "collected", "state": "COMPLETED"})
}

// HandleOrderStatePatch serves PATCH /v1/orders/{orderID}/state.
func (s *Service) HandleOrderStatePatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	var req struct {
		State string `json:"state"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	state := strings.TrimSpace(req.State)
	if state == "" {
		state = "IN_TRANSIT"
	}
	if s.orderGet != nil {
		current, found, _ := s.orderGet(r.Context(), orderID)
		if found {
			current.Status = state
			writeJSON(w, http.StatusOK, current)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": orderID, "state": state})
}

// HandleDeliveryArrive serves POST /v1/delivery/arrive.
func (s *Service) HandleDeliveryArrive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "arrived"})
}

// HandleDeliveryShopClosed is a fallback when order.Service is not wired.
func (s *Service) HandleDeliveryShopClosed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeCompatNotImplemented(w)
}

// HandleDeliveryBypass serves POST /v1/delivery/bypass-offload and confirm-payment-bypass.
func (s *Service) HandleDeliveryBypass(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeCompatNotImplemented(w)
}

// HandleFleetRouteReorder serves POST /v1/fleet/route/reorder.
func (s *Service) HandleFleetRouteReorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeCompatNotImplemented(w)
}

// HandleFleetEarlyComplete serves POST /v1/fleet/route/request-early-complete.
func (s *Service) HandleFleetEarlyComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeCompatNotImplemented(w)
}

// HandleOrderAmend serves POST /v1/order/amend.
func (s *Service) HandleOrderAmend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "amended"})
}

// HandleOrderGet serves GET /v1/orders/{orderID}.
func (s *Service) HandleOrderGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if s.orderGet != nil {
		order, found, err := s.orderGet(r.Context(), orderID)
		if err != nil {
			s.log.ErrorContext(r.Context(), "order get failed", "err", err, "order_id", orderID)
		} else if found {
			writeJSON(w, http.StatusOK, order)
			return
		} else {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
			return
		}
	}
	for _, row := range demoFleetOrders("") {
		if row["id"] == orderID {
			writeJSON(w, http.StatusOK, row)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
}

// HandleUserNotifications serves GET /v1/user/notifications (driver inbox).
func (s *Service) HandleUserNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.notifSvc == nil {
		writeJSON(w, http.StatusOK, map[string]any{"notifications": []any{}, "unread_count": 0})
		return
	}
	notifs, listErr := s.notifSvc.ListForRecipient(r.Context(), driverID, 50)
	if listErr != nil {
		s.log.ErrorContext(r.Context(), "list notifications failed", "err", listErr, "driver_id", driverID)
		writeJSON(w, http.StatusOK, map[string]any{"notifications": []any{}, "unread_count": 0})
		return
	}
	unread, _ := s.notifSvc.UnreadCount(r.Context(), driverID)
	writeJSON(w, http.StatusOK, map[string]any{"notifications": notifs, "unread_count": unread})
}

// HandleMarkNotificationsRead serves POST /v1/user/notifications/read.
func (s *Service) HandleMarkNotificationsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req struct {
		NotificationIDs []string `json:"notification_ids"`
	}
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if s.notifSvc != nil && len(req.NotificationIDs) > 0 {
		if markErr := s.notifSvc.MarkRead(r.Context(), driverID, req.NotificationIDs); markErr != nil {
			s.log.ErrorContext(r.Context(), "mark notifications read failed", "err", markErr, "driver_id", driverID)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func allowDriverDemoFallback() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("ALLOW_DRIVER_DEMO_FALLBACK")), "true")
}

func writeCompatNotImplemented(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{
		"error":        "not_implemented",
		"feature_flag": "driver_compat_pending",
	})
}

// HandleDeliveryCompatOK accepts POST bodies for extended delivery edges.
func (s *Service) HandleDeliveryCompatOK(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeCompatNotImplemented(w)
}

func iosRouteManifest(driverID, date string) map[string]any {
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02")
	}
	return map[string]any{
		"driver_id":   driverID,
		"date":        date,
		"expires_at":  time.Now().UTC().Add(24 * time.Hour).Unix(),
		"hashes": map[string]string{
			"ord_factory_1": "demo-token-ord-1",
			"ord_factory_2": "demo-token-ord-2",
		},
	}
}
