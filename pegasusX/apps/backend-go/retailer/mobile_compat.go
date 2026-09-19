package retailer

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
)

// Catalog handlers (HandleCatalogCategories, HandleCatalogProducts,
// HandleCategorySuppliers) removed — replaced by catalog.Service in
// catalogroutes. Demo data eliminated.

// HandleCreateOrder is a dead fallback (not mounted when orderroutes owns create).
// B7 R-P0-3: never fabricate order_id without Spanner.
func (s *Service) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{
		"error":   "order_service_unwired",
		"message": "Use POST /v1/order/create via order.Service (Spanner + outbox)",
	})
}

// HandleUnifiedCheckout is a dead fallback (not mounted when paymentroutes owns unified).
// B7 R-P0-3: never fabricate checkout success.
func (s *Service) HandleUnifiedCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{
		"error":   "order_service_unwired",
		"message": "Use POST /v1/checkout/unified via order/payment service",
	})
}

// HandleCashCheckout serves POST /v1/order/cash-checkout when PaymentService is absent.
// B1 M-P0-5: never fake PENDING_CASH_COLLECTION without Spanner/outbox.
func (s *Service) HandleCashCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusConflict, map[string]any{
		"error":         "use_confirm_cash",
		"message":       "Use POST /v1/delivery/confirm-cash (or wired payment cash-checkout) — silent cash selection is forbidden",
		"checkout_path": "/v1/delivery/confirm-cash",
	})
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

// HandleAIPredictionsAlias serves GET /v1/ai/predictions (legacy client path).
// P1: fail-closed. Clients expect DemandForecast[] (id/product_name/qty).
// The real list is GET /v1/retailer/ai/predictions ({items: RetailerAIPrediction}).
// Proxying would be a schema lie; P4 retargets apps. Never return silent [].
func (s *Service) HandleAIPredictionsAlias(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusGone, map[string]string{
		"error":            "use_retailer_ai_predictions",
		"message":          "GET /v1/ai/predictions is not a product list; use GET /v1/retailer/ai/predictions",
		"predictions_path": "/v1/retailer/ai/predictions",
	})
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
// P1: no persist path exists — never {status:ok}.
func (s *Service) HandleCorrectPrediction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusGone, map[string]string{
		"error":   "prediction_correct_unwired",
		"message": "PATCH /v1/ai/predictions/correct does not persist a correction",
	})
}

// HandleRetailerCards serves GET /v1/retailer/cards.
// P1: no vault — 410 saved_cards_not_product (default; not a silent empty list).
func (s *Service) HandleRetailerCards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeSavedCardsGone(w)
}

// HandleRetailerCardMutation serves POST card lifecycle endpoints.
func (s *Service) HandleRetailerCardMutation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeSavedCardsGone(w)
}

func writeSavedCardsGone(w http.ResponseWriter) {
	writeJSON(w, http.StatusGone, map[string]string{
		"error":   "saved_cards_not_product",
		"message": "Saved cards are not a PegasusX product; pay at delivery via /v1/order/card-checkout",
	})
}

// HandleLoyaltyNotProduct serves GET /v1/retailer/loyalty/tier — 410, never a fake tier.
func (s *Service) HandleLoyaltyNotProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusGone, map[string]string{
		"error":   "loyalty_not_product",
		"message": "Loyalty tiers are not a PegasusX product",
	})
}

// HandleUserNotifications is not mounted. Live GET is notifications.InboxHandlers.HandleList
// (main.go last registration). Kept fail-closed so a remount cannot return empty [] as success.
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
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "inbox_unavailable"})
		return
	}
	limit, offset := notifications.ParseInboxPagination(r)
	notifs, listErr := s.notifSvc.ListForRecipient(r.Context(), rid, limit, offset)
	if listErr != nil {
		s.log.ErrorContext(r.Context(), "list notifications failed", "err", listErr, "retailer_id", rid)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "inbox_list_failed"})
		return
	}
	unread, unreadErr := s.notifSvc.UnreadCount(r.Context(), rid)
	if unreadErr != nil {
		s.log.ErrorContext(r.Context(), "unread count failed", "err", unreadErr, "retailer_id", rid)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "inbox_unread_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"notifications": notifications.ToInboxWireFromAnyList(notifs),
		"unread_count":  unread,
		"limit":         limit,
		"offset":        offset,
		"has_more":      len(notifs) == limit,
	})
}

// HandleMarkNotificationsRead is not mounted. Live POST is notifications.InboxHandlers.HandleMarkRead.
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
	if s.notifSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "inbox_unavailable"})
		return
	}
	if markErr := notifications.ApplyMarkRead(r.Context(), s.notifSvc, rid, req); markErr != nil {
		s.log.ErrorContext(r.Context(), "mark notifications read failed", "err", markErr, "retailer_id", rid)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mark_read_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleDeviceToken is intentionally not mounted. Clients must use platform
// POST /v1/user/device-token (durable DeviceTokens). Kept as a comment trap:
// do not re-register a silent OK handler here.

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
		SupplierID           string  `json:"supplier_id"`
		InviteToken          string  `json:"invite_token"`
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
		SupplierID:           strings.TrimSpace(req.SupplierID),
		InviteToken:          strings.TrimSpace(req.InviteToken),
		Lat:                  req.Latitude,
		Lng:                  req.Longitude,
		DeliveryAddress:      deliveryAddr,
		PlaceID:              strings.TrimSpace(req.PlaceID),
		ReceivingWindowOpen:  req.ReceivingWindowOpen,
		ReceivingWindowClose: req.ReceivingWindowClose,
	})
	if err != nil {
		writeAttachError(w, err)
		return
	}
	ret := Retailer{
		RetailerID: reg.RetailerID,
		Phone:      reg.Phone,
		Name:       name,
		SupplierID: reg.SupplierID,
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
		writeAttachError(w, err)
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
	if fr := strings.TrimSpace(order.LocationFreshness); fr != "" {
		m["location_freshness"] = fr
	} else if order.LiveLocationAvailable {
		m["location_freshness"] = "LIVE"
	} else if order.DriverLocation != nil {
		m["location_freshness"] = "LAST_KNOWN"
	} else {
		m["location_freshness"] = "AWAITING_TELEMETRY"
	}
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
