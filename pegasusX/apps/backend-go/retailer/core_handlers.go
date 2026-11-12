package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// FamilyMember models retailer family-member records.
type FamilyMember struct {
	MemberID  string `json:"member_id"`
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	CreatedAt string `json:"created_at"`
}

type retailerProfileUpdateRequest struct {
	Name                 string   `json:"name,omitempty"`
	Phone                string   `json:"phone,omitempty"`
	Company              string   `json:"company,omitempty"`
	CountryCode          string   `json:"country_code,omitempty"`
	DeliveryAddress      string   `json:"delivery_address,omitempty"`
	PlaceID              string   `json:"place_id,omitempty"`
	Lat                  *float64 `json:"lat,omitempty"`
	Lng                  *float64 `json:"lng,omitempty"`
	ReceivingWindowOpen  *string  `json:"receiving_window_open"`
	ReceivingWindowClose *string  `json:"receiving_window_close"`
}

type supplierPreference struct {
	SupplierID string                  `json:"supplier_id"`
	Name       string                  `json:"name"`
	IsFavorite bool                    `json:"is_favorite"`
	Pricing    *retailerPricingSummary `json:"pricing,omitempty"`
}

type retailerPricingSummary struct {
	BaseMarkupBps       int64  `json:"base_markup_bps"`
	RetailerDiscountBps int64  `json:"retailer_discount_bps"`
	MinMarginBps        int64  `json:"min_margin_bps"`
	Currency            string `json:"currency"`
	RuleVersion         int64  `json:"rule_version"`
	UpdatedAt           string `json:"updated_at,omitempty"`
}

type retailerPricingRuleResponse struct {
	SupplierID string                 `json:"supplier_id"`
	Configured bool                   `json:"configured"`
	Pricing    retailerPricingSummary `json:"pricing"`
}

const recentReceiptLimit = 10

var (
	errRetailerUnauthorized  = errors.New("retailer_unauthorized")
	errRetailerScopeMismatch = errors.New("retailer_scope_mismatch")
)

// HandleProfile supports GET/PUT /v1/retailer/profile.
func (s *Service) HandleProfile(w http.ResponseWriter, r *http.Request) {
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleGetProfile(w, r, rid)
	case http.MethodPut:
		s.handleUpdateProfile(w, r, rid)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleGetProfile(w http.ResponseWriter, r *http.Request, retailerID string) {
	ret, found, err := s.repo.GetRetailer(r.Context(), retailerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_retailer_failed"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "retailer_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, retailerProfileDTO(ret, s.resolveSupplierScope(r.Context())))
}

func retailerProfileDTO(ret Retailer, boundSupplierID string) map[string]any {
	supplierID := strings.TrimSpace(ret.SupplierID)
	if supplierID == "" {
		supplierID = boundSupplierID
	}
	displayName := strings.TrimSpace(ret.Name)
	dto := map[string]any{
		"retailer_id":  ret.RetailerID,
		"id":           ret.RetailerID,
		"supplier_id":  supplierID,
		"name":         displayName,
		"phone":        ret.Phone,
		"company":      displayName,
		"location":     retailerLocationLabel(ret.DeliveryAddress, ret.Lat, ret.Lng),
		"country_code": ret.CountryCode,
		"status":       "ACTIVE",
		"h3_cell":      ret.H3Cell,
		"lat":          ret.Lat,
		"lng":          ret.Lng,
		"created_at":   ret.CreatedAt.Format(time.RFC3339Nano),
		"updated_at":   ret.UpdatedAt.Format(time.RFC3339Nano),
	}
	if addr := strings.TrimSpace(ret.DeliveryAddress); addr != "" {
		dto["delivery_address"] = addr
	}
	if pid := strings.TrimSpace(ret.PlaceID); pid != "" {
		dto["place_id"] = pid
	}
	if open := strings.TrimSpace(ret.ReceivingWindowOpen); open != "" {
		dto["receiving_window_open"] = open
	}
	if close := strings.TrimSpace(ret.ReceivingWindowClose); close != "" {
		dto["receiving_window_close"] = close
	}
	return dto
}

func retailerLocationLabel(address string, lat, lng float64) string {
	if addr := strings.TrimSpace(address); addr != "" {
		return addr
	}
	if lat == 0 && lng == 0 {
		return ""
	}
	return fmt.Sprintf("%.5f,%.5f", lat, lng)
}

func (s *Service) handleUpdateProfile(w http.ResponseWriter, r *http.Request, retailerID string) {
	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req retailerProfileUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	ret, found, err := s.repo.GetRetailer(r.Context(), retailerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_retailer_failed"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "retailer_not_found"})
		return
	}
	if strings.TrimSpace(ret.SupplierID) == "" {
		ret.SupplierID = s.resolveSupplierScope(r.Context())
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = strings.TrimSpace(req.Company)
	}
	if name != "" {
		ret.Name = name
	}
	if phone := strings.TrimSpace(req.Phone); phone != "" {
		ret.Phone = phone
	}
	if countryCode := strings.TrimSpace(req.CountryCode); countryCode != "" {
		ret.CountryCode = countryCode
	}
	if req.ReceivingWindowOpen != nil {
		open, err := validateReceivingWindowField(*req.ReceivingWindowOpen)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		ret.ReceivingWindowOpen = open
	}
	if req.ReceivingWindowClose != nil {
		closeWindow, err := validateReceivingWindowField(*req.ReceivingWindowClose)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		ret.ReceivingWindowClose = closeWindow
	}
	if addr := strings.TrimSpace(req.DeliveryAddress); addr != "" {
		ret.DeliveryAddress = addr
	}
	if pid := strings.TrimSpace(req.PlaceID); pid != "" {
		ret.PlaceID = pid
	}
	if req.Lat != nil && req.Lng != nil {
		if *req.Lat < -90 || *req.Lat > 90 || *req.Lng < -180 || *req.Lng > 180 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "coordinates_out_of_range"})
			return
		}
		ret.Lat = *req.Lat
		ret.Lng = *req.Lng
	}
	country, cell, stampErr := stampRetailerOptionalCoords(r.Context(), ret.Lat, ret.Lng, ret.CountryCode)
	if stampErr != nil {
		if writeMarketLaw(w, stampErr) {
			return
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": stampErr.Error()})
		return
	}
	ret.CountryCode = country
	if cell != "" {
		ret.H3Cell = cell
	}
	ret.UpdatedAt = s.now()
	if err := s.repo.UpdateRetailer(r.Context(), ret, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateRetailer, ret.RetailerID, events.TopicMain, events.RetailerEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventRetailerRegistered, Timestamp: ret.UpdatedAt.Format(time.RFC3339Nano)},
			RetailerID:  ret.RetailerID,
			Phone:       ret.Phone,
			Name:        ret.Name,
			SupplierID:  ret.SupplierID,
			Lat:         ret.Lat,
			Lng:         ret.Lng,
			H3Cell:      ret.H3Cell,
			CountryCode: ret.CountryCode,
		})
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_retailer_failed"})
		return
	}
	if s.cache != nil {
		s.cache.Invalidate(r.Context(), retailerByPhoneKey(ret.Phone), "retailer:id:"+ret.RetailerID)
	}
	respBytes, err := json.Marshal(retailerProfileDTO(ret, s.resolveSupplierScope(r.Context())))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "marshal_profile_failed"})
		return
	}
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleSupplierAdd serves POST /v1/retailer/suppliers/{supplierID}/add.
func (s *Service) HandleSupplierAdd(w http.ResponseWriter, r *http.Request) {
	s.handleSupplierMutation(w, r, "add")
}

// HandleSupplierRemove serves POST /v1/retailer/suppliers/{supplierID}/remove.
func (s *Service) HandleSupplierRemove(w http.ResponseWriter, r *http.Request) {
	s.handleSupplierMutation(w, r, "remove")
}

func (s *Service) handleSupplierMutation(w http.ResponseWriter, r *http.Request, action string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	supplierID := strings.TrimSpace(chi.URLParam(r, "supplierID"))
	if supplierID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supplier_id_required"})
		return
	}
	body, ok := readLimitedBody(w, r, 8*1024)
	if !ok {
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()
	if action == "add" {
		if _, err := s.guardTradingPartner(r.Context(), supplierID); err != nil {
			writeAttachError(w, err)
			return
		}
	}
	s.applySupplierFavoriteMutationCtx(r.Context(), rid, supplierID, action)
	respBytes, err := s.supplierListResponseBytes(r.Context(), rid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_suppliers_failed"})
		return
	}
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

func (s *Service) applySupplierFavoriteMutation(retailerID, supplierID, action string) {
	// Legacy sync path — prefer durable helper when request context is available.
	s.mu.Lock()
	prefs := s.favoriteSuppliers[retailerID]
	if prefs == nil {
		prefs = map[string]bool{}
	}
	if action == "add" {
		prefs[supplierID] = true
	} else {
		delete(prefs, supplierID)
	}
	s.favoriteSuppliers[retailerID] = prefs
	s.mu.Unlock()
}

func (s *Service) applySupplierFavoriteMutationCtx(ctx context.Context, retailerID, supplierID, action string) {
	_ = s.setFavoriteSupplierDurable(ctx, retailerID, supplierID, action == "add")
}

func (s *Service) supplierListResponseBytes(ctx context.Context, retailerID string) ([]byte, error) {
	prefs, err := s.loadFavoriteSuppliersDurable(ctx, retailerID)
	if err != nil {
		s.log.Warn("favorite suppliers load failed", "err", err)
	}
	favorite := prefs != nil && prefs[s.resolveSupplierScope(ctx)]

	var pricing *retailerPricingSummary
	rule, found, err := s.repo.GetSupplierPricingRule(ctx, s.resolveSupplierScope(ctx))
	if err != nil {
		s.log.Warn("retailer supplier pricing read failed", "supplier_id", s.resolveSupplierScope(ctx), "err", err)
	} else if found {
		summary := retailerPricingSummary{
			BaseMarkupBps:       rule.BaseMarkupBps,
			RetailerDiscountBps: rule.RetailerDiscountBps,
			MinMarginBps:        rule.MinMarginBps,
			Currency:            rule.Currency,
			RuleVersion:         rule.RuleVersion,
		}
		if !rule.UpdatedAt.IsZero() {
			summary.UpdatedAt = rule.UpdatedAt.UTC().Format(time.RFC3339Nano)
		}
		pricing = &summary
	}

	return json.Marshal([]supplierPreference{{
		SupplierID: s.resolveSupplierScope(ctx),
		Name:       "pegasusX Supplier",
		IsFavorite: favorite,
		Pricing:    pricing,
	}})
}

// HandleSuppliers supports GET /v1/retailer/suppliers and
// POST /v1/retailer/suppliers/{supplierID}/{action}.
func (s *Service) HandleSuppliers(w http.ResponseWriter, r *http.Request) {
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if r.Method == http.MethodGet {
		s.handleSupplierList(r.Context(), w, rid)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	supplierID := chi.URLParam(r, "supplierID")
	action := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "action")))
	if supplierID == "" || (action != "add" && action != "remove") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supplierID and action(add|remove) required"})
		return
	}
	if action == "add" {
		s.handleSupplierMutation(w, r, "add")
		return
	}
	s.handleSupplierMutation(w, r, "remove")
}

func (s *Service) handleSupplierList(ctx context.Context, w http.ResponseWriter, retailerID string) {
	respBytes, err := s.supplierListResponseBytes(ctx, retailerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_suppliers_failed"})
		return
	}
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandlePricingRule supports GET /v1/retailer/pricing/rules.
func (s *Service) HandlePricingRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if _, err := retailerIDFromRequest(r); err != nil {
		writeRetailerIdentityError(w, err)
		return
	}

	rule, found, err := s.repo.GetSupplierPricingRule(r.Context(), s.resolveSupplierScope(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_pricing_rule_failed"})
		return
	}

	resp := retailerPricingRuleResponse{
		SupplierID: s.resolveSupplierScope(r.Context()),
		Configured: found,
		Pricing: retailerPricingSummary{
			BaseMarkupBps:       0,
			RetailerDiscountBps: 0,
			MinMarginBps:        0,
			Currency:            "",
			RuleVersion:         0,
			UpdatedAt:           "",
		},
	}

	if found {
		resp.Pricing.BaseMarkupBps = rule.BaseMarkupBps
		resp.Pricing.RetailerDiscountBps = rule.RetailerDiscountBps
		resp.Pricing.MinMarginBps = rule.MinMarginBps
		resp.Pricing.Currency = rule.Currency
		resp.Pricing.RuleVersion = rule.RuleVersion
		if !rule.UpdatedAt.IsZero() {
			resp.Pricing.UpdatedAt = rule.UpdatedAt.UTC().Format(time.RFC3339Nano)
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleCartSync supports GET/POST /v1/retailer/cart/sync.
// GET ?scope=all|all=1 returns cart lines across suppliers (Phase 2).
// POST preserves an explicit per-line supplier_id; otherwise stamps active trading partner.
func (s *Service) HandleCartSync(w http.ResponseWriter, r *http.Request) {
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	switch r.Method {
	case http.MethodGet:
		if s.cartRepo == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "cart_unavailable"})
			return
		}
		var (
			items   []CartItem
			listErr error
		)
		if cartScopeAll(r) {
			items, listErr = s.cartRepo.ListByRetailerAll(r.Context(), rid)
		} else {
			items, listErr = s.cartRepo.ListByRetailer(r.Context(), rid, s.resolveSupplierScope(r.Context()))
		}
		if listErr != nil {
			s.log.ErrorContext(r.Context(), "cart list failed", "err", listErr, "retailer_id", rid)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cart_list_failed"})
			return
		}
		if items == nil {
			items = []CartItem{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "updated_at": s.now().Format(time.RFC3339Nano)})
	case http.MethodPost:
		var payload struct {
			Items []CartItem `json:"items"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		defer r.Body.Close()
		if s.cartRepo != nil {
			fallbackSupplier := s.resolveSupplierScope(r.Context())
			for i := range payload.Items {
				if payload.Items[i].CartItemID == "" {
					payload.Items[i].CartItemID = s.newID()
				}
				payload.Items[i].RetailerID = rid
				// Preserve explicit line SupplierId (mixed-cart / SSMR). Default UI still stamps active partner.
				if strings.TrimSpace(payload.Items[i].SupplierID) == "" {
					payload.Items[i].SupplierID = fallbackSupplier
				}
			}
			if err := s.cartRepo.UpsertItems(r.Context(), payload.Items); err != nil {
				s.log.ErrorContext(r.Context(), "cart sync failed", "err", err, "retailer_id", rid)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
				return
			}
			if s.cache != nil {
				s.cache.Invalidate(r.Context(), "retailer:cart:"+rid)
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": payload.Items, "updated_at": s.now().Format(time.RFC3339Nano)})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleCartClear supports DELETE /v1/retailer/cart (or POST /v1/retailer/cart/clear).
// ?scope=all clears every supplier; otherwise clears the active trading-partner cart.
func (s *Service) HandleCartClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if s.cartRepo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "cart_unavailable"})
		return
	}
	if cartScopeAll(r) {
		if err := s.cartRepo.ClearCartAll(r.Context(), rid); err != nil {
			s.log.ErrorContext(r.Context(), "cart clear-all failed", "err", err, "retailer_id", rid)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
	} else {
		if err := s.cartRepo.ClearCart(r.Context(), rid, s.resolveSupplierScope(r.Context())); err != nil {
			s.log.ErrorContext(r.Context(), "cart clear failed", "err", err, "retailer_id", rid)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
	}
	if s.cache != nil {
		s.cache.Invalidate(r.Context(), "retailer:cart:"+rid)
	}
	writeJSON(w, http.StatusOK, map[string]any{"cleared": true})
}

func cartScopeAll(r *http.Request) bool {
	scope := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("scope")))
	if scope == "all" {
		return true
	}
	all := strings.TrimSpace(r.URL.Query().Get("all"))
	return all == "1" || strings.EqualFold(all, "true")
}

// HandleOrders returns orders scoped to a retailer.
func (s *Service) HandleOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID := strings.TrimSpace(chi.URLParam(r, "retailerID"))
	if retailerID == "" {
		var err error
		retailerID, err = retailerIDFromRequest(r)
		if err != nil {
			writeRetailerIdentityError(w, err)
			return
		}
	}
	orders := s.demoOrdersForRetailer(r.Context(), retailerID)
	writeJSON(w, http.StatusOK, orders)
}

// HandleOrdersAlias serves GET /v1/orders for retailer desktop fallback paths.
func (s *Service) HandleOrdersAlias(w http.ResponseWriter, r *http.Request) {
	s.HandleOrders(w, r)
}

func (s *Service) demoOrdersForRetailer(ctx context.Context, retailerID string) []map[string]any {
	if retailerID == "" {
		return []map[string]any{}
	}
	orders, err := s.listRetailerTrackingOrders(ctx, retailerID)
	if err != nil {
		return []map[string]any{}
	}
	// Empty list is authoritative — never inject demo tracking fixtures.
	out := make([]map[string]any, 0, len(orders))
	for i := range orders {
		out = append(out, mobileTrackingOrder(orders[i]))
	}
	return out
}

// HandleRequestCancel is POST /v1/orders/request-cancel when OrderService is nil.
// B7 R-P0-3: never fake cancel_requested without durable order path.
func (s *Service) HandleRequestCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{
		"error":   "order_service_unwired",
		"message": "Order service required; request-cancel is disabled without Spanner order path",
	})
}

// HandleCancelOrder is POST /v1/order/cancel when OrderService is nil.
// B7 R-P0-3: never fake cancelled without Spanner/outbox.
func (s *Service) HandleCancelOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{
		"error":   "order_service_unwired",
		"message": "Order service required for cancel; use POST /v1/order/cancel with OrderService wired",
	})
}

// HandleExpensesAnalytics returns basic retailer expense analytics.
func (s *Service) HandleExpensesAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	var totalSpend int64
	var openOrders int
	orders, listErr := s.listRetailerTrackingOrders(r.Context(), rid)
	if listErr == nil {
		for _, o := range orders {
			totalSpend += o.TotalMinor
			if o.Status != "COMPLETED" && o.Status != "CANCELLED" {
				openOrders++
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"currency":          s.countryCode,
		"total_spend_minor": totalSpend,
		"open_orders":       openOrders,
		"last_updated":      s.now().Format(time.RFC3339Nano),
	})
}

// HandleDetailedAnalytics returns additive detailed analytics payload.
func (s *Service) HandleDetailedAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	receipts, _ := s.repo.ListRecentReceipts(r.Context(), rid, 100)
	if receipts == nil {
		receipts = []TrackingOrder{}
	}
	series := make([]map[string]any, 0, len(receipts))
	for _, receipt := range receipts {
		series = append(series, map[string]any{
			"order_id":    receipt.OrderID,
			"total_minor": receipt.TotalMinor,
			"currency":    receipt.Currency,
			"created_at":  receipt.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"series":      series,
		"by_category": []any{},
		"from":        strings.TrimSpace(r.URL.Query().Get("from")),
		"to":          strings.TrimSpace(r.URL.Query().Get("to")),
	})
}

// HandleFamilyMembers supports GET/POST /v1/retailer/family-members.
func (s *Service) HandleFamilyMembers(w http.ResponseWriter, r *http.Request) {
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		members := append([]FamilyMember(nil), s.familyByRetailer[rid]...)
		s.mu.RUnlock()
		sort.Slice(members, func(i, j int) bool { return members[i].CreatedAt > members[j].CreatedAt })
		familyWrites := "open"
		if s.isFamilyWritesGone(r.Context(), rid) {
			familyWrites = "gone"
			// Team is SoT after migrate — do not surface RAM residual list as live family.
			members = []FamilyMember{}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"members":       members,
			"family_writes": familyWrites,
			"migrate":       "/v1/retailer/family-members/migrate-to-team",
		})
	case http.MethodPost:
		// After Family→Team migrate, family writes are gone — use Team invites.
		if s.familyPostBlocked(w, r, rid) {
			return
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		defer r.Body.Close()
		// Accept name (desktop) or nickname (mobile legacy).
		name := strings.TrimSpace(payload["name"])
		if name == "" {
			name = strings.TrimSpace(payload["nickname"])
		}
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
			return
		}
		m := FamilyMember{
			MemberID:  "fam_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000"), ".", ""),
			Name:      name,
			Phone:     strings.TrimSpace(payload["phone"]),
			CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		}
		s.mu.Lock()
		s.familyByRetailer[rid] = append(s.familyByRetailer[rid], m)
		s.mu.Unlock()
		writeJSON(w, http.StatusCreated, map[string]any{
			"member":  m,
			"warning": "legacy_family_ram_only_use_migrate_to_team",
			"migrate": "/v1/retailer/family-members/migrate-to-team",
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleDeleteFamilyMember deletes a family member by id.
func (s *Service) HandleDeleteFamilyMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	memberID := strings.TrimSpace(chi.URLParam(r, "memberID"))
	if memberID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "memberID required"})
		return
	}
	s.mu.Lock()
	members := s.familyByRetailer[rid]
	filtered := members[:0]
	for _, m := range members {
		if m.MemberID != memberID {
			filtered = append(filtered, m)
		}
	}
	s.familyByRetailer[rid] = filtered
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "member_id": memberID})
}

// HandleShopClosedResponse stores retailer response to shop-closed event.
func (s *Service) HandleShopClosedResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "acknowledged"})
}

// HandleAIPredictions lists retailer AI preorder suggestions pending review.
