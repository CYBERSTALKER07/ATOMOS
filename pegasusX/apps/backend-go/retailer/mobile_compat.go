package retailer

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
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
		"error":         "use_checkout_unified",
		"message":       "Use POST /v1/checkout/unified for card checkout",
		"checkout_path": "/v1/checkout/unified",
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
	writeJSON(w, http.StatusGone, map[string]string{
		"error":   "ai_preorder_removed",
		"message": "AI pre-orders are not available in PegasusX retailer apps",
	})
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
	limit, offset := notifications.ParseInboxPagination(r)
	notifs, listErr := s.notifSvc.ListForRecipient(r.Context(), rid, limit, offset)
	if listErr != nil {
		s.log.ErrorContext(r.Context(), "list notifications failed", "err", listErr, "retailer_id", rid)
		writeJSON(w, http.StatusOK, map[string]any{"notifications": []any{}, "unread_count": 0, "limit": limit, "offset": offset})
		return
	}
	unread, _ := s.notifSvc.UnreadCount(r.Context(), rid)
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
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	var req notifications.MarkReadRequest
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if markErr := notifications.ApplyMarkRead(r.Context(), s.notifSvc, rid, req); markErr != nil {
		s.log.ErrorContext(r.Context(), "mark notifications read failed", "err", markErr, "retailer_id", rid)
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
		DeliveryAddress      string  `json:"delivery_address"`
		AddressText          string  `json:"address_text"`
		PlaceID              string  `json:"place_id"`
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
	deliveryAddr := strings.TrimSpace(req.DeliveryAddress)
	if deliveryAddr == "" {
		deliveryAddr = strings.TrimSpace(req.AddressText)
	}
	reg, err := s.Register(r.Context(), RegisterRequest{
		Phone:                strings.TrimSpace(req.PhoneNumber),
		Name:                 name,
		Lat:                  req.Latitude,
		Lng:                  req.Longitude,
		DeliveryAddress:      deliveryAddr,
		PlaceID:              strings.TrimSpace(req.PlaceID),
		ReceivingWindowOpen:  req.ReceivingWindowOpen,
		ReceivingWindowClose: req.ReceivingWindowClose,
	})
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	ret := Retailer{
		RetailerID: reg.RetailerID,
		Phone:      reg.Phone,
		Name:       name,
		SupplierID: s.supplierID,
		Lat:        req.Latitude,
		Lng:        req.Longitude,
	}
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

	b, _ := json.Marshal(order)
	var m map[string]any
	json.Unmarshal(b, &m)

	m["supplier_name"] = "pegasusX Supplier"
	m["warehouse_name"] = "Demo Warehouse"
	m["state"] = state
	m["total_amount"] = order.TotalMinor
	m["driver_latitude"] = driverLat(order)
	m["driver_longitude"] = driverLng(order)
	m["live_location_available"] = order.LiveLocationAvailable
	m["delivery_token"] = order.DeliveryToken
	m["is_approaching"] = order.IsApproaching
	if order.PaymentStatus != "" {
		m["payment_status"] = order.PaymentStatus
	}
	// ADR-009 fiscal receipt surface (snake_case for mobile clients).
	if fs := strings.TrimSpace(order.FiscalStatus); fs != "" {
		m["fiscal_status"] = fs
	}
	if qr := strings.TrimSpace(order.FiscalQR); qr != "" {
		m["fiscal_qr"] = qr
	}
	if rid := strings.TrimSpace(order.LatestFiscalReceiptID); rid != "" {
		m["latest_fiscal_receipt_id"] = rid
	}

	if order.DriverLocation != nil {
		m["driver_location"] = order.DriverLocation
	}

	return m
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
	m := map[string]any{
		"order_id":        order.OrderID,
		"supplier_id":     order.SupplierID,
		"supplier_name":   "pegasusX Supplier",
		"state":           state,
		"adjusted_amount": order.TotalMinor,
		"item_count":      len(order.Items),
	}
	if order.DriverLocation != nil {
		m["live_location_available"] = true
	} else {
		m["live_location_available"] = false
	}
	return m
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
				DriverID:  supplierID,
				Lat:       41.312,
				Lng:       69.241,
				Latitude:  41.312,
				Longitude: 69.241,
			},
		},
	}
}
