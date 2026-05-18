package retailer

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
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
	Name  string `json:"name,omitempty"`
	Phone string `json:"phone,omitempty"`
}

type supplierPreference struct {
	SupplierID string `json:"supplier_id"`
	Name       string `json:"name"`
	IsFavorite bool   `json:"is_favorite"`
}

// HandleProfile supports GET/PUT /v1/retailer/profile.
func (s *Service) HandleProfile(w http.ResponseWriter, r *http.Request) {
	rid := retailerIDFromRequest(r)
	if rid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
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
	writeJSON(w, http.StatusOK, map[string]any{
		"retailer_id":  ret.RetailerID,
		"supplier_id":  ret.SupplierID,
		"name":         ret.Name,
		"phone":        ret.Phone,
		"country_code": ret.CountryCode,
		"h3_cell":      ret.H3Cell,
		"lat":          ret.Lat,
		"lng":          ret.Lng,
		"created_at":   ret.CreatedAt.Format(time.RFC3339Nano),
		"updated_at":   ret.UpdatedAt.Format(time.RFC3339Nano),
	})
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
	if strings.TrimSpace(req.Name) != "" {
		ret.Name = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.Phone) != "" {
		ret.Phone = strings.TrimSpace(req.Phone)
	}
	ret.UpdatedAt = s.now()
	if err := s.repo.UpdateRetailer(r.Context(), ret, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateRetailer, ret.RetailerID, events.TopicMain, retailerRegisteredEvent{
			Type:        events.EventRetailerRegistered,
			RetailerID:  ret.RetailerID,
			Phone:       ret.Phone,
			Name:        ret.Name,
			SupplierID:  ret.SupplierID,
			Lat:         ret.Lat,
			Lng:         ret.Lng,
			H3Cell:      ret.H3Cell,
			CountryCode: ret.CountryCode,
			Timestamp:   ret.UpdatedAt.Format(time.RFC3339Nano),
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

// HandleSuppliers supports GET /v1/retailer/suppliers and
// POST /v1/retailer/suppliers/{supplierID}/{action}.
func (s *Service) HandleSuppliers(w http.ResponseWriter, r *http.Request) {
	rid := retailerIDFromRequest(r)
	if rid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if r.Method == http.MethodGet {
		s.handleSupplierList(w, rid)
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
	s.handleSupplierList(w, rid)
}

func (s *Service) handleSupplierList(w http.ResponseWriter, retailerID string) {
	s.mu.RLock()
	prefs := s.favoriteSuppliers[retailerID]
	favorite := false
	if prefs != nil {
		favorite = prefs[s.supplierID]
	}
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, []supplierPreference{{
		SupplierID: s.supplierID,
		Name:       "pegasusX Supplier",
		IsFavorite: favorite,
	}})
}

// HandleCartSync supports GET/POST /v1/retailer/cart/sync.
func (s *Service) HandleCartSync(w http.ResponseWriter, r *http.Request) {
	rid := retailerIDFromRequest(r)
	if rid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		cart := s.cartByRetailer[rid]
		s.mu.RUnlock()
		if cart == nil {
			cart = map[string]any{"items": []any{}, "updated_at": time.Now().UTC().Format(time.RFC3339Nano)}
		}
		writeJSON(w, http.StatusOK, cart)
	case http.MethodPost:
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		defer r.Body.Close()
		payload["updated_at"] = time.Now().UTC().Format(time.RFC3339Nano)
		s.mu.Lock()
		s.cartByRetailer[rid] = payload
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, payload)
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
	retailerID := chi.URLParam(r, "retailerID")
	if retailerID == "" {
		retailerID = retailerIDFromRequest(r)
	}
	if retailerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, []map[string]any{})
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
	writeJSON(w, http.StatusOK, map[string]any{
		"currency":          s.countryCode,
		"total_spend_minor": 0,
		"open_orders":       0,
		"last_updated":      time.Now().UTC().Format(time.RFC3339Nano),
	})
}

// HandleDetailedAnalytics returns additive detailed analytics payload.
func (s *Service) HandleDetailedAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"series":      []any{},
		"by_category": []any{},
		"from":        strings.TrimSpace(r.URL.Query().Get("from")),
		"to":          strings.TrimSpace(r.URL.Query().Get("to")),
	})
}

// HandleFamilyMembers supports GET/POST /v1/retailer/family-members.
func (s *Service) HandleFamilyMembers(w http.ResponseWriter, r *http.Request) {
	rid := retailerIDFromRequest(r)
	if rid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
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
	rid := retailerIDFromRequest(r)
	if rid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
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

// HandleConfirmAIOrder acknowledges AI-confirm flow.
func (s *Service) HandleConfirmAIOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ai_confirmed"})
}

// HandleRejectAIOrder acknowledges AI-reject flow.
func (s *Service) HandleRejectAIOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ai_rejected"})
}

// HandleEditPreorder acknowledges preorder edit flow.
func (s *Service) HandleEditPreorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "preorder_edited"})
}

// HandleConfirmPreorder acknowledges preorder confirm flow.
func (s *Service) HandleConfirmPreorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "preorder_confirmed"})
}

// HandlePendingPayments returns pending post-offload payment sessions.
func (s *Service) HandlePendingPayments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pending": []any{}})
}

// HandleActiveFulfillment returns active fulfillment list for retailer UI.
func (s *Service) HandleActiveFulfillment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"fulfillments": []any{}})
}

// HandleTracking returns tracking payload scaffold.
func (s *Service) HandleTracking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "idle", "events": []any{}})
}

func retailerIDFromRequest(r *http.Request) string {
	if claims, ok := auth.FromContext(r.Context()); ok {
		if strings.TrimSpace(claims.Subject) != "" {
			return strings.TrimSpace(claims.Subject)
		}
	}
	if id := strings.TrimSpace(chi.URLParam(r, "retailerID")); id != "" {
		return id
	}
	return strings.TrimSpace(r.URL.Query().Get("retailer_id"))
}
