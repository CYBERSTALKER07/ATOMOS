package retailer

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// Catalog handlers (HandleCatalogCategories, HandleCatalogProducts,
// HandleCategorySuppliers) removed — replaced by catalog.Service in
// catalogroutes. Demo data eliminated.

// HandleCreateOrder serves POST /v1/order/create.
func (s *Service) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	orderID := "ord_" + s.newID()
	writeJSON(w, http.StatusOK, map[string]any{
		"order_id": orderID,
		"status":   "PENDING",
		"message":  "order accepted",
	})
}

// HandleUnifiedCheckout serves POST /v1/checkout/unified.
func (s *Service) HandleUnifiedCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	orderID := "ord_" + s.newID()
	writeJSON(w, http.StatusOK, map[string]any{
		"order_id":    orderID,
		"status":      "PENDING",
		"total_minor": int64(28000),
		"currency":    "UZS",
	})
}

// HandleCashCheckout serves POST /v1/order/cash-checkout.
func (s *Service) HandleCashCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "state": "PENDING_CASH_COLLECTION"})
}

// HandleCardCheckout serves POST /v1/order/card-checkout.
func (s *Service) HandleCardCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusConflict, map[string]any{
		"error":            "use_checkout_unified",
		"message":          "Use POST /v1/checkout/unified for card checkout",
		"checkout_path":    "/v1/checkout/unified",
	})
}

// HandleAIPredictionsAlias serves GET /v1/ai/predictions (mobile catalog path).
func (s *Service) HandleAIPredictionsAlias(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if _, err := retailerIDFromRequest(r); err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, []any{})
}

// HandleAIPreorder serves POST /v1/ai/preorder.
func (s *Service) HandleAIPreorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleCorrectPrediction serves PATCH /v1/ai/predictions/correct.
func (s *Service) HandleCorrectPrediction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleRetailerCards serves GET /v1/retailer/cards.
func (s *Service) HandleRetailerCards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cards": []any{}})
}

// HandleRetailerCardMutation serves POST card lifecycle endpoints.
func (s *Service) HandleRetailerCardMutation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleUserNotifications serves GET /v1/user/notifications.
func (s *Service) HandleUserNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if s.notifSvc == nil {
		writeJSON(w, http.StatusOK, map[string]any{"notifications": []any{}, "unread_count": 0})
		return
	}
	notifs, listErr := s.notifSvc.ListForRecipient(r.Context(), rid, 50)
	if listErr != nil {
		s.log.ErrorContext(r.Context(), "list notifications failed", "err", listErr, "retailer_id", rid)
		writeJSON(w, http.StatusOK, map[string]any{"notifications": []any{}, "unread_count": 0})
		return
	}
	unread, _ := s.notifSvc.UnreadCount(r.Context(), rid)
	writeJSON(w, http.StatusOK, map[string]any{"notifications": notifs, "unread_count": unread})
}

// HandleMarkNotificationsRead serves POST /v1/user/notifications/read.
func (s *Service) HandleMarkNotificationsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
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
		if markErr := s.notifSvc.MarkRead(r.Context(), rid, req.NotificationIDs); markErr != nil {
			s.log.ErrorContext(r.Context(), "mark notifications read failed", "err", markErr, "retailer_id", rid)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleDeviceToken serves POST /v1/user/device-token.
func (s *Service) HandleDeviceToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleMobileRegister accepts the legacy mobile registration body and returns AuthResponse.
func (s *Service) HandleMobileRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_failed"})
		return
	}
	defer r.Body.Close()

	if s.writeCanonicalRegisterResponse(w, r, body) {
		return
	}

	var req struct {
		PhoneNumber          string  `json:"phone_number"`
		Password             string  `json:"password"`
		OwnerName            string  `json:"owner_name"`
		StoreName            string  `json:"store_name"`
		Latitude             float64 `json:"latitude"`
		Longitude            float64 `json:"longitude"`
		ReceivingWindowOpen  string  `json:"receiving_window_open"`
		ReceivingWindowClose string  `json:"receiving_window_close"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	name := strings.TrimSpace(req.StoreName)
	if name == "" {
		name = strings.TrimSpace(req.OwnerName)
	}
	reg, err := s.Register(r.Context(), RegisterRequest{
		Phone:                strings.TrimSpace(req.PhoneNumber),
		Name:                 name,
		Lat:                  req.Latitude,
		Lng:                  req.Longitude,
		ReceivingWindowOpen:  req.ReceivingWindowOpen,
		ReceivingWindowClose: req.ReceivingWindowClose,
	})
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	ret := Retailer{RetailerID: reg.RetailerID, Phone: reg.Phone, Name: name, SupplierID: s.supplierID}
	s.writeMobileAuthResponse(w, ret)
}

// writeCanonicalRegisterResponse handles POST bodies that include top-level "phone"
// (SSMR smoke, API clients) and returns RegisterResponse. Mobile apps send
// phone_number instead and continue through HandleMobileRegister.
func (s *Service) writeCanonicalRegisterResponse(w http.ResponseWriter, r *http.Request, body []byte) bool {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(body, &probe); err != nil {
		return false
	}
	if _, ok := probe["phone"]; !ok {
		return false
	}
	var req RegisterRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return true
	}
	resp, err := s.Register(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return true
	}
	writeJSON(w, http.StatusCreated, resp)
	return true
}

func mobileTrackingOrder(order TrackingOrder) map[string]any {
	state := strings.TrimSpace(order.Status)
	if state == "" {
		state = strings.TrimSpace(order.TrackingStatus)
	}
	return map[string]any{
		"order_id":                order.OrderID,
		"supplier_id":             order.SupplierID,
		"supplier_name":           "pegasusX Supplier",
		"warehouse_id":            order.WarehouseID,
		"warehouse_name":          "Demo Warehouse",
		"driver_id":               order.DriverID,
		"state":                   state,
		"total_amount":            order.TotalMinor,
		"created_at":              order.CreatedAt,
		"driver_latitude":         driverLat(order),
		"driver_longitude":        driverLng(order),
		"items":                   order.Items,
		"live_location_available": order.LiveLocationAvailable,
	}
}

func driverLat(order TrackingOrder) any {
	if order.DriverLocation == nil {
		return nil
	}
	return order.DriverLocation.Lat
}

func driverLng(order TrackingOrder) any {
	if order.DriverLocation == nil {
		return nil
	}
	return order.DriverLocation.Lng
}

func mobileActiveFulfillment(order TrackingOrder) map[string]any {
	state := strings.TrimSpace(order.Status)
	return map[string]any{
		"order_id":        order.OrderID,
		"supplier_id":     order.SupplierID,
		"supplier_name":   "pegasusX Supplier",
		"state":           state,
		"adjusted_amount": order.TotalMinor,
		"item_count":      len(order.Items),
	}
}

func demoTrackingOrdersForRetailer(retailerID, supplierID string) []TrackingOrder {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	return []TrackingOrder{
		{
			OrderID:    "ord_retailer_demo_1",
			SupplierID: supplierID,
			RetailerID: retailerID,
			Status:     "IN_TRANSIT",
			TotalMinor: 28000,
			Currency:   "UZS",
			CreatedAt:  now,
			UpdatedAt:  now,
			DriverID:   "drv_demo_1",
			Items: []TrackingLineItem{
				{ProductID: "prod-milk-1l", ProductName: "Whole Milk 1L", Quantity: 2, UnitPrice: 12000, LineTotal: 24000},
			},
			DriverLocation: &TrackingLocation{
				DriverID: supplierID,
				Lat:      41.312,
				Lng:      69.241,
				Latitude: 41.312,
				Longitude: 69.241,
			},
		},
	}
}
