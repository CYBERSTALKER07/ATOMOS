package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"google.golang.org/api/iterator"
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
	if s.guardIdempotency(w, r, body) {
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
		if s.proximity != nil {
			if cell, err := s.proximity.CellForCoordinate(ret.Lat, ret.Lng); err == nil {
				ret.H3Cell = cell
			}
		}
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
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()
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
			writeJSON(w, http.StatusOK, map[string]any{"items": []any{}, "updated_at": s.now().Format(time.RFC3339Nano)})
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
			writeJSON(w, http.StatusOK, map[string]any{"items": []any{}, "updated_at": s.now().Format(time.RFC3339Nano)})
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
		writeJSON(w, http.StatusOK, map[string]any{"cleared": true})
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
func (s *Service) HandleAIPredictions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	limit := 25
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be a positive integer"})
			return
		}
		limit = parsed
	}
	items, err := s.orders.ListRetailerAIPredictions(r.Context(), retailerID, limit)
	if err != nil {
		s.log.Warn("retailer ai predictions read failed", "retailer_id", retailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_ai_predictions_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// HandleConfirmAIOrder confirms a retailer AI preorder.
func (s *Service) HandleConfirmAIOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
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
	var req order.ConfirmAIOrderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.ConfirmAIOrder(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleRejectAIOrder rejects a retailer AI preorder.
func (s *Service) HandleRejectAIOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
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
	var req order.RejectAIOrderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.RejectAIOrder(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleEditPreorder edits a retailer draft preorder.
func (s *Service) HandleEditPreorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
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
	var req order.EditPreorderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.EditPreorder(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleConfirmPreorder confirms a retailer draft preorder.
func (s *Service) HandleConfirmPreorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
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
	var req order.ConfirmPreorderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.ConfirmPreorder(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleAcceptDeliveryProposal applies a warehouse-proposed delivery date.
func (s *Service) HandleAcceptDeliveryProposal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
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
	var req order.AcceptDeliveryProposalRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.AcceptDeliveryProposal(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleRejectDeliveryProposal cancels an order when the retailer declines a warehouse date proposal.
func (s *Service) HandleRejectDeliveryProposal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
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
	var req order.RejectDeliveryProposalRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.RejectDeliveryProposal(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleRejectPreorder lets a retailer cancel a draft or scheduled manual pre-order.
func (s *Service) HandleRejectPreorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
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
	var req order.RejectPreorderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.RejectPreorder(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

func writeRetailerOrderLifecycleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, order.ErrOrderNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
	case errors.Is(err, order.ErrOrderForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	case errors.Is(err, order.ErrInvalidStatusTransition):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_status_transition"})
	case errors.Is(err, order.ErrDeliveryProposalRequired):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "delivery_proposal_required"})
	case errors.Is(err, order.ErrDeliveryProposalPending):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "delivery_proposal_pending"})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
}

// HandlePendingPayments returns pending post-offload payment sessions.
func (s *Service) HandlePendingPayments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	orders, err := s.listRetailerTrackingOrders(r.Context(), retailerID)
	if err != nil {
		s.log.Warn("retailer pending payments read failed", "retailer_id", retailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_pending_payments_failed"})
		return
	}
	pending := filterTrackingOrders(orders, isPendingPaymentStatus)
	sessions := make([]map[string]any, 0, len(pending))
	for i := range pending {
		sessions = append(sessions, map[string]any{
			"session_id":    "sess_" + pending[i].OrderID,
			"order_id":      pending[i].OrderID,
			"retailer_id":   retailerID,
			"supplier_id":   pending[i].SupplierID,
			"gateway":       "payme",
			"locked_amount": pending[i].TotalMinor,
			"currency":      pending[i].Currency,
			"status":        pending[i].Status,
			"created_at":    pending[i].CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "pending",
		"pending": sessions,
		"count":   len(sessions),
	})
}

// HandleActiveFulfillment returns active fulfillment list for retailer UI.
func (s *Service) HandleActiveFulfillment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	orders, err := s.listRetailerTrackingOrders(r.Context(), retailerID)
	if err != nil {
		s.log.Warn("retailer active fulfillment read failed", "retailer_id", retailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_active_fulfillment_failed"})
		return
	}
	fulfillments := make([]map[string]any, 0, len(orders))
	for i := range orders {
		fulfillments = append(fulfillments, mobileActiveFulfillment(orders[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "active",
		"fulfillments": fulfillments,
		"count":        len(fulfillments),
	})
}

// HandleTracking returns active assigned-order tracking projections.
func (s *Service) HandleTracking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	orders, err := s.listRetailerTrackingOrders(r.Context(), retailerID)
	if err != nil {
		s.log.Warn("retailer tracking read failed", "retailer_id", retailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_tracking_failed"})
		return
	}

	receipts, err := s.listRetailerRecentReceipts(r.Context(), retailerID)
	if err != nil {
		s.log.Warn("retailer recent receipts read failed", "retailer_id", retailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_receipts_failed"})
		return
	}

	merged := mergeTrackingOrders(orders, receipts)
	events := deriveTrackingEvents(merged)

	status := "idle"
	if len(orders) > 0 {
		status = "active"
	}

	mobileOrders := make([]map[string]any, 0, len(orders))
	for i := range orders {
		mobileOrders = append(mobileOrders, mobileTrackingOrder(orders[i]))
	}

	mobileReceipts := make([]map[string]any, 0, len(receipts))
	for i := range receipts {
		mobileReceipts = append(mobileReceipts, mobileTrackingOrder(receipts[i]))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":          status,
		"orders":          mobileOrders,
		"recent_receipts": mobileReceipts,
		"events":          events,
	})
}

func (s *Service) listRetailerTrackingOrders(ctx context.Context, retailerID string) ([]TrackingOrder, error) {
	const pageSize = 500
	all := make([]TrackingOrder, 0)
	for offset := 0; ; offset += pageSize {
		batch, err := s.repo.ListTrackingOrders(ctx, retailerID, pageSize, offset)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < pageSize {
			break
		}
	}
	return normalizeTrackingOrders(s.attachRouteGeometry(ctx, s.attachLiveLocations(ctx, all))), nil
}

func (s *Service) listRetailerRecentReceipts(ctx context.Context, retailerID string) ([]TrackingOrder, error) {
	receipts, err := s.repo.ListRecentReceipts(ctx, retailerID, recentReceiptLimit)
	if err != nil {
		return nil, err
	}
	return normalizeTrackingOrders(receipts), nil
}

func mergeTrackingOrders(primary []TrackingOrder, secondary []TrackingOrder) []TrackingOrder {
	if len(primary) == 0 && len(secondary) == 0 {
		return []TrackingOrder{}
	}
	merged := make([]TrackingOrder, 0, len(primary)+len(secondary))
	seen := make(map[string]struct{}, len(primary)+len(secondary))
	appendUnique := func(orders []TrackingOrder) {
		for _, order := range orders {
			orderID := strings.TrimSpace(order.OrderID)
			if orderID != "" {
				if _, ok := seen[orderID]; ok {
					continue
				}
				seen[orderID] = struct{}{}
			}
			merged = append(merged, order)
		}
	}
	appendUnique(primary)
	appendUnique(secondary)
	return merged
}

func normalizeTrackingOrders(orders []TrackingOrder) []TrackingOrder {
	if len(orders) == 0 {
		return []TrackingOrder{}
	}
	return orders
}

func filterTrackingOrders(orders []TrackingOrder, keep func(TrackingOrder) bool) []TrackingOrder {
	if len(orders) == 0 {
		return []TrackingOrder{}
	}
	filtered := make([]TrackingOrder, 0, len(orders))
	for _, order := range orders {
		if keep(order) {
			filtered = append(filtered, order)
		}
	}
	if len(filtered) == 0 {
		return []TrackingOrder{}
	}
	return filtered
}

func isPendingPaymentStatus(order TrackingOrder) bool {
	switch strings.TrimSpace(order.Status) {
	case "AWAITING_PAYMENT", "PENDING_CASH_COLLECTION":
		return true
	default:
		return false
	}
}

func deriveTrackingEvents(orders []TrackingOrder) []TrackingEvent {
	if len(orders) == 0 {
		return []TrackingEvent{}
	}
	events := make([]TrackingEvent, 0, len(orders)*2)
	for _, order := range orders {
		createdAt := strings.TrimSpace(order.CreatedAt)
		updatedAt := strings.TrimSpace(order.UpdatedAt)
		status := strings.TrimSpace(order.Status)
		if createdAt != "" {
			events = append(events, TrackingEvent{
				EventType:  TrackingEventOrderCreated,
				OrderID:    order.OrderID,
				OccurredAt: createdAt,
				Derived:    true,
				Source:     trackingEventSourceOrderRow,
			})
		}
		if updatedAt == "" || status == "" {
			continue
		}
		if updatedAt == createdAt && status == "PENDING" {
			continue
		}
		if updatedAt == createdAt && createdAt == "" {
			continue
		}
		events = append(events, TrackingEvent{
			EventType:  TrackingEventOrderStatusSnapshot,
			OrderID:    order.OrderID,
			Status:     status,
			OccurredAt: updatedAt,
			Derived:    true,
			Source:     trackingEventSourceOrderRow,
		})
	}
	sort.SliceStable(events, func(i, j int) bool {
		left := parseTrackingEventTime(events[i].OccurredAt)
		right := parseTrackingEventTime(events[j].OccurredAt)
		if left.Equal(right) {
			if events[i].OrderID == events[j].OrderID {
				return events[i].EventType < events[j].EventType
			}
			return events[i].OrderID < events[j].OrderID
		}
		return left.After(right)
	})
	if len(events) == 0 {
		return []TrackingEvent{}
	}
	return events
}

func parseTrackingEventTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}
	}
	return parsed
}

type trackingLocationLookup struct {
	location telemetry.DriverLocation
	found    bool
}

func (s *Service) attachLiveLocations(ctx context.Context, orders []TrackingOrder) []TrackingOrder {
	if s.locations == nil || len(orders) == 0 {
		return orders
	}
	now := s.now()
	lookups := make(map[string]trackingLocationLookup, len(orders))
	for i := range orders {
		orders[i].LiveLocationAvailable = false
		orders[i].DriverLocation = nil
		driverID := strings.TrimSpace(orders[i].DriverID)
		if driverID == "" {
			continue
		}
		lookup, ok := lookups[driverID]
		if !ok {
			location, found, err := s.locations.GetDriverLocation(ctx, driverID)
			if err != nil {
				s.log.Warn("retailer tracking location read failed", "driver_id", driverID, "err", err)
				continue
			}
			lookup = trackingLocationLookup{location: location, found: found}
			lookups[driverID] = lookup
		}
		if !lookup.found || !lookup.location.IsLive(now) {
			continue
		}
		if strings.TrimSpace(lookup.location.SupplierID) != strings.TrimSpace(orders[i].SupplierID) {
			continue
		}
		orders[i].LiveLocationAvailable = true
		orders[i].DriverLocation = trackingLocationFromTelemetry(lookup.location)
		if orders[i].DeliveryLat != 0 && orders[i].DeliveryLng != 0 {
			driverLat := lookup.location.Lat
			driverLng := lookup.location.Lng
			if driverLat == 0 && lookup.location.Latitude != 0 {
				driverLat = lookup.location.Latitude
			}
			if driverLng == 0 && lookup.location.Longitude != 0 {
				driverLng = lookup.location.Longitude
			}
			if proximity.WithinDeliveryApproach(proximity.HaversineDistance(driverLat, driverLng, orders[i].DeliveryLat, orders[i].DeliveryLng)) {
				orders[i].IsApproaching = true
			}
		}
	}
	return orders
}

func trackingLocationFromTelemetry(location telemetry.DriverLocation) *TrackingLocation {
	return &TrackingLocation{
		DriverID:          location.DriverID,
		SupplierID:        location.SupplierID,
		Lat:               location.Lat,
		Lng:               location.Lng,
		Latitude:          location.Latitude,
		Longitude:         location.Longitude,
		Velocity:          location.Velocity,
		Heading:           location.Heading,
		ReportedAt:        location.ReportedAt.UTC().Format(time.RFC3339Nano),
		ReceivedAt:        location.ReceivedAt.UTC().Format(time.RFC3339Nano),
		StaleAfterSeconds: location.StaleAfterSeconds,
	}
}

// attachRouteGeometry hydrates planned polylines from SupplierTruckManifests (fail-open).
func (s *Service) attachRouteGeometry(ctx context.Context, orders []TrackingOrder) []TrackingOrder {
	if s == nil || s.spannerClient == nil || len(orders) == 0 {
		return orders
	}
	manifestIDs := make([]string, 0, len(orders))
	seen := make(map[string]struct{}, len(orders))
	for i := range orders {
		mid := strings.TrimSpace(orders[i].ManifestID)
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}
		manifestIDs = append(manifestIDs, mid)
	}
	if len(manifestIDs) == 0 {
		return orders
	}
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId, RouteId, EncodedRoutePolyline, RouteGeometrySource, StopCount
		      FROM SupplierTruckManifests
		      WHERE ManifestId IN UNNEST(@ids)
		        AND EncodedRoutePolyline IS NOT NULL
		        AND EncodedRoutePolyline != ''`,
		Params: map[string]any{"ids": manifestIDs},
	}
	iter := s.spannerClient.Single().
		WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).
		Query(ctx, stmt)
	defer iter.Stop()

	byManifest := make(map[string]routing.RouteGeometryWire, len(manifestIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			s.log.Warn("retailer tracking route geometry query failed", "err", err)
			return orders
		}
		var manifestID, routeID string
		var encoded, source spanner.NullString
		var stopCount int64
		if err := row.Columns(&manifestID, &routeID, &encoded, &source, &stopCount); err != nil {
			s.log.Warn("retailer tracking route geometry scan failed", "err", err)
			continue
		}
		if !encoded.Valid || encoded.StringVal == "" {
			continue
		}
		geometry, decodeErr := routing.GeometryFromStoredPolyline(
			routeID,
			encoded.StringVal,
			source.StringVal,
			int(stopCount),
		)
		if decodeErr != nil {
			continue
		}
		wire := routing.ToRouteGeometryWire(geometry)
		byManifest[strings.TrimSpace(manifestID)] = wire
	}
	for i := range orders {
		mid := strings.TrimSpace(orders[i].ManifestID)
		if mid == "" {
			continue
		}
		if wire, ok := byManifest[mid]; ok {
			copy := wire
			orders[i].RouteGeometry = &copy
		}
	}
	return orders
}

func retailerIDFromRequest(r *http.Request) (string, error) {
	if claims, ok := auth.FromContext(r.Context()); ok {
		// Retail OS v2: tenant id is RetailerOrgID; legacy v1 uses Subject as retailer id.
		orgID := auth.ResolveRetailerOrgID(claims)
		if orgID != "" {
			pathRetailerID := strings.TrimSpace(chi.URLParam(r, "retailerID"))
			if pathRetailerID != "" && pathRetailerID != orgID {
				return "", errRetailerScopeMismatch
			}
			queryRetailerID := strings.TrimSpace(r.URL.Query().Get("retailer_id"))
			if queryRetailerID != "" && queryRetailerID != orgID {
				return "", errRetailerScopeMismatch
			}
			return orgID, nil
		}
	}
	if id := strings.TrimSpace(chi.URLParam(r, "retailerID")); id != "" {
		return id, nil
	}
	if id := strings.TrimSpace(r.URL.Query().Get("retailer_id")); id != "" {
		return id, nil
	}
	return "", errRetailerUnauthorized
}

func writeRetailerIdentityError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errRetailerScopeMismatch):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	default:
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
}
