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

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
)

// FamilyMember models retailer family-member records.
type FamilyMember struct {
	MemberID  string `json:"member_id"`
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	CreatedAt string `json:"created_at"`
}

type retailerProfileUpdateRequest struct {
	Name                 string  `json:"name,omitempty"`
	Phone                string  `json:"phone,omitempty"`
	Company              string  `json:"company,omitempty"`
	CountryCode          string  `json:"country_code,omitempty"`
	ReceivingWindowOpen  *string `json:"receiving_window_open"`
	ReceivingWindowClose *string `json:"receiving_window_close"`
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
	writeJSON(w, http.StatusOK, retailerProfileDTO(ret, s.supplierID))
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
		"location":     retailerLocationLabel(ret.Lat, ret.Lng),
		"country_code": ret.CountryCode,
		"status":       "ACTIVE",
		"h3_cell":      ret.H3Cell,
		"lat":          ret.Lat,
		"lng":          ret.Lng,
		"created_at":   ret.CreatedAt.Format(time.RFC3339Nano),
		"updated_at":   ret.UpdatedAt.Format(time.RFC3339Nano),
	}
	if open := strings.TrimSpace(ret.ReceivingWindowOpen); open != "" {
		dto["receiving_window_open"] = open
	}
	if close := strings.TrimSpace(ret.ReceivingWindowClose); close != "" {
		dto["receiving_window_close"] = close
	}
	return dto
}

func retailerLocationLabel(lat, lng float64) string {
	if lat == 0 && lng == 0 {
		return ""
	}
	return fmt.Sprintf("%.5f,%.5f", lat, lng)
}

func (s *Service) handleUpdateProfile(w http.ResponseWriter, r *http.Request, retailerID string) {
	var req retailerProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

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
		ret.SupplierID = s.supplierID
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
	s.handleGetProfile(w, r, retailerID)
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
	s.mu.Lock()
	prefs := s.favoriteSuppliers[rid]
	if prefs == nil {
		prefs = map[string]bool{}
	}
	if action == "add" {
		prefs[supplierID] = true
	} else {
		delete(prefs, supplierID)
	}
	s.favoriteSuppliers[rid] = prefs
	s.mu.Unlock()
	s.handleSupplierList(r.Context(), w, rid)
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

	s.mu.Lock()
	prefs := s.favoriteSuppliers[rid]
	if prefs == nil {
		prefs = map[string]bool{}
	}
	if action == "add" {
		prefs[supplierID] = true
	} else {
		delete(prefs, supplierID)
	}
	s.favoriteSuppliers[rid] = prefs
	s.mu.Unlock()
	s.handleSupplierList(r.Context(), w, rid)
}

func (s *Service) handleSupplierList(ctx context.Context, w http.ResponseWriter, retailerID string) {
	s.mu.RLock()
	prefs := s.favoriteSuppliers[retailerID]
	favorite := false
	if prefs != nil {
		favorite = prefs[s.supplierID]
	}
	s.mu.RUnlock()

	var pricing *retailerPricingSummary
	rule, found, err := s.repo.GetSupplierPricingRule(ctx, s.supplierID)
	if err != nil {
		s.log.Warn("retailer supplier pricing read failed", "supplier_id", s.supplierID, "err", err)
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

	writeJSON(w, http.StatusOK, []supplierPreference{{
		SupplierID: s.supplierID,
		Name:       "pegasusX Supplier",
		IsFavorite: favorite,
		Pricing:    pricing,
	}})
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

	rule, found, err := s.repo.GetSupplierPricingRule(r.Context(), s.supplierID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_pricing_rule_failed"})
		return
	}

	resp := retailerPricingRuleResponse{
		SupplierID: s.supplierID,
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
		items, listErr := s.cartRepo.ListByRetailer(r.Context(), rid, s.supplierID)
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
			for i := range payload.Items {
				if payload.Items[i].CartItemID == "" {
					payload.Items[i].CartItemID = s.newID()
				}
				payload.Items[i].RetailerID = rid
				payload.Items[i].SupplierID = s.supplierID
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
	if len(orders) == 0 {
		orders = demoTrackingOrdersForRetailer(retailerID, s.supplierID)
	}
	out := make([]map[string]any, 0, len(orders))
	for i := range orders {
		out = append(out, mobileTrackingOrder(orders[i]))
	}
	return out
}

// HandleRequestCancel is POST /v1/orders/request-cancel.
func (s *Service) HandleRequestCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "cancel_requested"})
}

// HandleCancelOrder is POST /v1/order/cancel.
func (s *Service) HandleCancelOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "cancelled"})
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
		writeJSON(w, http.StatusOK, map[string]any{"members": members})
	case http.MethodPost:
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		defer r.Body.Close()
		name := strings.TrimSpace(payload["name"])
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
		writeJSON(w, http.StatusCreated, m)
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
	var req order.ConfirmAIOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.ConfirmAIOrder(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
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
	var req order.RejectAIOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.RejectAIOrder(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
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
	var req order.EditPreorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.EditPreorder(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
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
	var req order.ConfirmPreorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.ConfirmPreorder(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeRetailerOrderLifecycleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, order.ErrOrderNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
	case errors.Is(err, order.ErrOrderForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	case errors.Is(err, order.ErrInvalidStatusTransition):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_status_transition"})
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
	return normalizeTrackingOrders(s.attachLiveLocations(ctx, all)), nil
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
			if proximity.HaversineDistance(driverLat, driverLng, orders[i].DeliveryLat, orders[i].DeliveryLng) < 0.100 {
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

func retailerIDFromRequest(r *http.Request) (string, error) {
	if claims, ok := auth.FromContext(r.Context()); ok {
		subject := strings.TrimSpace(claims.Subject)
		if subject != "" {
			pathRetailerID := strings.TrimSpace(chi.URLParam(r, "retailerID"))
			if pathRetailerID != "" && pathRetailerID != subject {
				return "", errRetailerScopeMismatch
			}
			queryRetailerID := strings.TrimSpace(r.URL.Query().Get("retailer_id"))
			if queryRetailerID != "" && queryRetailerID != subject {
				return "", errRetailerScopeMismatch
			}
			return subject, nil
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
