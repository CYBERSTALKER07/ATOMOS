package driver

import (
	"encoding/json"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
)

func demoFleetOrders(driverID string) []map[string]any {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	return []map[string]any{
		{
			"id": "ord_factory_1", "retailer_id": "ret_demo", "retailer_name": "Demo Retailer",
			"state": "IN_TRANSIT", "total_amount": int64(24000), "delivery_fee_minor": int64(100000),
			"delivery_address": "Tashkent, Chilonzor",
			"latitude":         41.285, "longitude": 69.203,
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
			"latitude":         41.335, "longitude": 69.288,
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

// HandleDriverDepart serves POST /v1/fleet/driver/depart. It flips the driver's
// SEALED manifest to DISPATCHED and rolls every LOADED order on it to IN_TRANSIT
// atomically, then broadcasts the transition so supplier + driver surfaces update
// in real time. Without the depart seam wired it degrades to the prior stub.
func (s *Service) HandleDriverDepart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
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

	if s.depart == nil {
		resp := map[string]string{"status": "departed"}
		respBytes, _ := json.Marshal(resp)
		s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
		writeJSONBytes(w, http.StatusOK, respBytes)
		return
	}

	result, ok, err := s.depart(r.Context(), driverID)
	if err != nil {
		s.log.ErrorContext(r.Context(), "driver depart failed", "err", err, "driver_id", driverID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "depart_failed"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusConflict, map[string]string{
			"status": "no_sealed_manifest",
			"error":  "no_sealed_manifest",
		})
		return
	}

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), driverManifestKey(driverID))
	}
	s.broadcastDriverEvent(r.Context(), driverID, map[string]any{
		"type":        "MANIFEST_DISPATCHED",
		"driver_id":   driverID,
		"manifest_id": result.ManifestID,
		"order_ids":   result.OrderIDs,
		"order_count": result.Count,
		"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
	})
	s.log.InfoContext(r.Context(), "driver departed",
		"driver_id", driverID,
		"manifest_id", result.ManifestID,
		"orders_dispatched", result.Count,
	)
	resp := map[string]any{
		"status":            "departed",
		"manifest_id":       result.ManifestID,
		"orders_dispatched": result.Count,
		"order_ids":         result.OrderIDs,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleOpenFiscal serves GET /v1/driver/open-fiscal — Phase 6 cash-bag soft-freeze banner.
func (s *Service) HandleOpenFiscal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	snap := OpenFiscalSnapshot{}
	if s.openFiscal != nil {
		var err error
		snap, err = s.openFiscal(r.Context(), driverID)
		if err != nil {
			s.log.ErrorContext(r.Context(), "open fiscal lookup failed", "err", err, "driver_id", driverID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "open_fiscal_lookup_failed"})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"open_fiscal_count": snap.Count,
		"order_ids":         snap.OrderIDs,
		"cash_bag_frozen":   snap.Frozen || snap.Count > 0,
	})
}

// HandleDriverReturnComplete serves POST /v1/fleet/driver/return-complete.
// It flips the driver's DISPATCHED manifest to COMPLETED and marks the driver
// as off-shift. Without ReturnCompleteFn wired, it degrades gracefully to the
// prior no-op stub so mobile clients never receive a hard error.
//
// Phase 6 / T10: blocks when any assigned order is still FISCALIZING or FISCAL_FAILED
// (cash bag soft-freeze until fiscal SUCCESS or audited force-complete).
func (s *Service) HandleDriverReturnComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
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

	// Soft-freeze cash bag / block shift-end while fiscal is open.
	if s.openFiscal != nil {
		snap, fErr := s.openFiscal(r.Context(), driverID)
		if fErr != nil {
			s.log.ErrorContext(r.Context(), "open fiscal lookup failed", "err", fErr, "driver_id", driverID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "open_fiscal_lookup_failed"})
			return
		}
		if snap.Frozen || snap.Count > 0 {
			writeJSON(w, http.StatusConflict, map[string]any{
				"status":            "open_fiscal_block",
				"error":             "open_fiscal_block",
				"open_fiscal_count": snap.Count,
				"order_ids":         snap.OrderIDs,
				"cash_bag_frozen":   true,
				"message":           "Clear fiscalizing / fiscal-failed orders before ending shift.",
			})
			return
		}
	}

	if s.cashReconRequired && s.cashReconGate != nil {
		ok, gateErr := s.cashReconGate(r.Context(), driverID)
		if gateErr != nil {
			s.log.ErrorContext(r.Context(), "cash reconciliation gate failed", "err", gateErr, "driver_id", driverID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cash_reconciliation_lookup_failed"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusConflict, map[string]any{
				"status":  "cash_reconciliation_required",
				"error":   "cash_reconciliation_required",
				"message": "Submit and reconcile declared cash before ending shift.",
			})
			return
		}
	}

	if s.returnComplete == nil {
		// Graceful degradation — update availability in-memory and return OK.
		s.mu.Lock()
		s.availability[driverID] = false
		s.mu.Unlock()
		resp := map[string]string{"status": "returned"}
		respBytes, _ := json.Marshal(resp)
		s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
		writeJSONBytes(w, http.StatusOK, respBytes)
		return
	}

	result, ok, err := s.returnComplete(r.Context(), driverID)
	if err != nil {
		s.log.ErrorContext(r.Context(), "driver return-complete failed", "err", err, "driver_id", driverID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "return_complete_failed"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusConflict, map[string]string{
			"status": "no_dispatched_manifest",
			"error":  "no_dispatched_manifest",
		})
		return
	}

	// Mark driver off-shift locally.
	s.mu.Lock()
	s.availability[driverID] = false
	s.mu.Unlock()

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), driverManifestKey(driverID))
		s.cache.Invalidate(r.Context(), driverAvailabilityKey(driverID))
	}

	// Broadcast MANIFEST_COMPLETED to supplier + driver hubs so live tracking surfaces update.
	s.broadcastDriverEvent(r.Context(), driverID, map[string]any{
		"type":            "MANIFEST_COMPLETED",
		"driver_id":       driverID,
		"manifest_id":     result.ManifestID,
		"order_ids":       result.OrderIDs,
		"orders_returned": result.Count,
		"timestamp":       s.now().UTC().Format(time.RFC3339Nano),
	})
	// Broadcast availability change so tracking surfaces remove the driver from shift
	s.broadcastDriverEvent(r.Context(), driverID, events.DriverEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventDriverAvailabilityChanged, Timestamp: s.now().UTC().Format(time.RFC3339Nano), Version: 1},
		DriverID:   driverID,
		Available:  false,
		OnShift:    false,
		SupplierID: s.resolveSupplierScope(r.Context()),
	})
	s.log.InfoContext(r.Context(), "driver returned to depot",
		"driver_id", driverID,
		"manifest_id", result.ManifestID,
		"orders_returned", result.Count,
	)
	resp := map[string]any{
		"status":          "returned",
		"manifest_id":     result.ManifestID,
		"orders_returned": result.Count,
		"order_ids":       result.OrderIDs,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
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
	body, err := readLimitedBody(r, 8*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	var req struct {
		State string `json:"state"`
	}
	_ = json.Unmarshal(body, &req)
	state := strings.TrimSpace(req.State)
	if state == "" {
		state = "IN_TRANSIT"
	}
	var resp any
	if s.orderGet != nil {
		current, found, _ := s.orderGet(r.Context(), orderID)
		if found {
			current.Status = state
			resp = current
			
			if state == "IN_TRANSIT" && current.SplitGroupID != "" && s.repo != nil {
				claims, ok := auth.FromContext(r.Context())
				if ok {
					siblings, err := s.repo.FindSiblingDriversForOrder(r.Context(), orderID)
					if err == nil {
						for _, sib := range siblings {
							if sib != claims.Subject {
								payload := map[string]any{
									"type":           "OTHER_TRUCK_ON_WAY",
									"order_id":       orderID,
									"split_group_id": current.SplitGroupID,
									"message":        "Another truck is on the way to this route.",
								}
								s.broadcastDriverEvent(r.Context(), sib, payload)
							}
						}
					}
				}
			}
		}
	}
	if resp == nil {
		resp = map[string]any{"id": orderID, "state": state}
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encode_response_failed"})
		return
	}
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
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
	var req struct {
		OrderID     string `json:"order_id"`
		Items       any    `json:"items"`
		DriverNotes string `json:"driver_notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	eventPayload := events.OrderEvent{
		BaseEvent: events.BaseEvent{Type: "ORDER_AMENDED", Timestamp: s.now().UTC().Format(time.RFC3339Nano), Version: 1},
		OrderID:   req.OrderID,
		DriverID:  driverID,
		Action:    "amend",
		Reason:    req.DriverNotes,
		LineItems: req.Items,
	}

	if err := s.repo.Apply(r.Context(), nil, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateOrder, req.OrderID, events.TopicMain, eventPayload)
	}); err != nil {
		s.log.ErrorContext(r.Context(), "driver order amend failed", "err", err, "order_id", req.OrderID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "amend_failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":        true,
		"message":        "amended",
		"adjusted_total": nil,
	})
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
	if allowDriverDemoFallback() {
		for _, row := range demoFleetOrders("") {
			if row["id"] == orderID {
				writeJSON(w, http.StatusOK, row)
				return
			}
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
	limit, offset := notifications.ParseInboxPagination(r)
	notifs, listErr := s.notifSvc.ListForRecipient(r.Context(), driverID, limit, offset)
	if listErr != nil {
		s.log.ErrorContext(r.Context(), "list notifications failed", "err", listErr, "driver_id", driverID)
		writeJSON(w, http.StatusOK, map[string]any{"notifications": []any{}, "unread_count": 0, "limit": limit, "offset": offset})
		return
	}
	unread, _ := s.notifSvc.UnreadCount(r.Context(), driverID)
	writeJSON(w, http.StatusOK, map[string]any{
		"notifications": notifications.ToInboxWireFromAnyList(notifs),
		"unread_count":  unread,
		"limit":         limit,
		"offset":        offset,
		"has_more":      len(notifs) == limit,
	})
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
	var req notifications.MarkReadRequest
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if markErr := notifications.ApplyMarkRead(r.Context(), s.notifSvc, driverID, req); markErr != nil {
		s.log.ErrorContext(r.Context(), "mark notifications read failed", "err", markErr, "driver_id", driverID)
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
	demoTokens := demoOrderDeliveryTokens()
	hashes := make(map[string]string)
	for _, orderID := range []string{"ord_factory_1", "ord_factory_2"} {
		if token := demoTokens[orderID]; token != "" {
			hashes[orderID] = hashDeliveryToken(token)
		}
	}
	return map[string]any{
		"driver_id":  driverID,
		"date":       date,
		"expires_at": time.Now().UTC().Add(24 * time.Hour).Unix(),
		"hashes":     hashes,
	}
}

func demoRouteGeometry(routeID string) routing.RouteGeometry {
	waypoints := []routing.LatLng{
		{Lat: 41.2995, Lng: 69.2401},
		{Lat: 41.285, Lng: 69.203},
		{Lat: 41.335, Lng: 69.288},
	}
	return routing.BuildDenseRouteGeometry(routeID, waypoints)
}
