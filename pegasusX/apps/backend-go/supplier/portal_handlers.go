package supplier

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// InventoryItem is the wire/storage shape for supplier inventory rows.
type InventoryItem struct {
	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`
	Quantity    int64  `json:"quantity"`
	UpdatedAt   string `json:"updated_at"`
}

// InventoryPatchRequest is the mutation payload for PATCH /v1/supplier/inventory.
type InventoryPatchRequest struct {
	SKUID         string `json:"sku_id,omitempty"`
	SKU           string `json:"sku,omitempty"`
	ProductName   string `json:"product_name,omitempty"`
	Quantity      *int64 `json:"quantity,omitempty"`
	QuantityDelta *int64 `json:"quantity_delta,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

// InventoryAuditEntry tracks additive inventory mutations for supplier audit UI.
type InventoryAuditEntry struct {
	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`
	Before      int64  `json:"before"`
	After       int64  `json:"after"`
	Delta       int64  `json:"delta"`
	Reason      string `json:"reason"`
	Timestamp   string `json:"timestamp"`
}

// SupplierOrder is the lightweight supplier vetting/order queue shape.
type SupplierOrder struct {
	OrderID    string `json:"order_id"`
	RetailerID string `json:"retailer_id"`
	Status     string `json:"status"`
	Decision   string `json:"decision,omitempty"`
	Note       string `json:"note,omitempty"`
	TotalMinor int64  `json:"total_minor"`
	Currency   string `json:"currency"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type configureRequest struct {
	LegalName string `json:"legal_name,omitempty"`
}

type supplierProfileResponse struct {
	SupplierID       string   `json:"supplier_id"`
	LegalName        string   `json:"legal_name"`
	ContactName      string   `json:"contact_name"`
	Email            string   `json:"email"`
	Phone            string   `json:"phone"`
	Country          string   `json:"country"`
	Currency         string   `json:"currency"`
	Categories       []string `json:"categories"`
	IsRegistered     bool     `json:"is_registered"`
	IsConfigured     bool     `json:"is_configured"`
	SelectedGateways []string `json:"selected_gateways"`
	UpdatedAt        string   `json:"updated_at"`
}

type supplierProfileUpdateRequest struct {
	LegalName   string   `json:"legal_name,omitempty"`
	ContactName string   `json:"contact_name,omitempty"`
	Email       string   `json:"email,omitempty"`
	Phone       string   `json:"phone,omitempty"`
	Categories  []string `json:"categories,omitempty"`
}

type supplierDashboardResponse struct {
	SupplierID    string `json:"supplier_id"`
	IsConfigured  bool   `json:"is_configured"`
	InventorySKUs int    `json:"inventory_skus"`
	PendingOrders int    `json:"pending_orders"`
	UpdatedAt     string `json:"updated_at"`
}

type supplierEarningsResponse struct {
	Currency   string `json:"currency"`
	TodayMinor int64  `json:"today_minor"`
	WeekMinor  int64  `json:"week_minor"`
	MonthMinor int64  `json:"month_minor"`
}

type vetOrderRequest struct {
	OrderID    string `json:"order_id"`
	RetailerID string `json:"retailer_id,omitempty"`
	Decision   string `json:"decision"`
	Note       string `json:"note,omitempty"`
	TotalMinor int64  `json:"total_minor,omitempty"`
	Currency   string `json:"currency,omitempty"`
}

// HandleConfigure marks onboarding as configured/registered and supports the
// supplier portal completion handoff.
func (s *Service) HandleConfigure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	var req configureRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	defer r.Body.Close()

	current, found, err := s.repo.GetProfile(r.Context(), s.supplierID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_failed"})
		return
	}
	if !found {
		current = Profile{SupplierID: s.supplierID, Country: s.country, Currency: s.currency}
	}
	now := s.now()
	if strings.TrimSpace(req.LegalName) != "" {
		current.LegalName = strings.TrimSpace(req.LegalName)
	}
	current.IsRegistered = true
	current.UpdatedAt = now
	if err := s.repo.UpdateProfile(r.Context(), current, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateSupplier, s.supplierID, events.TopicMain, supplierUpdatedEvent{
			Type:         events.EventSupplierUpdated,
			SupplierID:   s.supplierID,
			LegalName:    current.LegalName,
			ContactName:  current.ContactName,
			Email:        current.Email,
			Phone:        current.Phone,
			Country:      current.Country,
			Categories:   current.Categories,
			IsRegistered: current.IsRegistered,
			IsConfigured: current.IsConfigured,
			Action:       "CONFIGURE",
			Timestamp:    now.Format(time.RFC3339Nano),
		})
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_supplier_failed"})
		return
	}
	if s.cache != nil {
		s.cache.Invalidate(r.Context(), supplierCacheKey(s.supplierID))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"supplier_id":   s.supplierID,
		"is_registered": current.IsRegistered,
		"is_configured": current.IsConfigured,
		"completed_at":  now.Format(time.RFC3339Nano),
	})
}

// HandleProfile supports GET/PUT /v1/supplier/profile.
func (s *Service) HandleProfile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetProfile(w, r)
	case http.MethodPut:
		s.handleUpdateProfile(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	current, found, err := s.repo.GetProfile(r.Context(), s.supplierID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_failed"})
		return
	}
	if !found {
		current = Profile{SupplierID: s.supplierID, Country: s.country, Currency: s.currency}
	}
	updated := ""
	if !current.UpdatedAt.IsZero() {
		updated = current.UpdatedAt.Format(time.RFC3339Nano)
	}
	writeJSON(w, http.StatusOK, supplierProfileResponse{
		SupplierID:       current.SupplierID,
		LegalName:        current.LegalName,
		ContactName:      current.ContactName,
		Email:            current.Email,
		Phone:            current.Phone,
		Country:          current.Country,
		Currency:         current.Currency,
		Categories:       append([]string(nil), current.Categories...),
		IsRegistered:     current.IsRegistered,
		IsConfigured:     current.IsConfigured,
		SelectedGateways: append([]string(nil), current.SelectedGateways...),
		UpdatedAt:        updated,
	})
}

func (s *Service) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req supplierProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

	current, found, err := s.repo.GetProfile(r.Context(), s.supplierID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_failed"})
		return
	}
	if !found {
		current = Profile{SupplierID: s.supplierID, Country: s.country, Currency: s.currency}
	}
	if strings.TrimSpace(req.LegalName) != "" {
		current.LegalName = strings.TrimSpace(req.LegalName)
	}
	if strings.TrimSpace(req.ContactName) != "" {
		current.ContactName = strings.TrimSpace(req.ContactName)
	}
	if strings.TrimSpace(req.Email) != "" {
		current.Email = strings.TrimSpace(req.Email)
	}
	if strings.TrimSpace(req.Phone) != "" {
		current.Phone = strings.TrimSpace(req.Phone)
	}
	if len(req.Categories) > 0 {
		current.Categories = append([]string(nil), req.Categories...)
	}
	now := s.now()
	current.UpdatedAt = now
	if err := s.repo.UpdateProfile(r.Context(), current, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateSupplier, s.supplierID, events.TopicMain, supplierUpdatedEvent{
			Type:         events.EventSupplierUpdated,
			SupplierID:   s.supplierID,
			LegalName:    current.LegalName,
			ContactName:  current.ContactName,
			Email:        current.Email,
			Phone:        current.Phone,
			Country:      current.Country,
			Categories:   current.Categories,
			IsRegistered: current.IsRegistered,
			IsConfigured: current.IsConfigured,
			Action:       "PROFILE_UPDATED",
			Timestamp:    now.Format(time.RFC3339Nano),
		})
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_supplier_failed"})
		return
	}
	if s.cache != nil {
		s.cache.Invalidate(r.Context(), supplierCacheKey(s.supplierID))
	}
	s.handleGetProfile(w, r)
}

// HandleDashboard returns supplier-facing operational counters.
func (s *Service) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	current, found, _ := s.repo.GetProfile(r.Context(), s.supplierID)
	if !found {
		current = Profile{SupplierID: s.supplierID, Country: s.country, Currency: s.currency}
	}
	s.mu.RLock()
	invCount := len(s.inventory)
	pending := 0
	for _, o := range s.orders {
		if strings.EqualFold(o.Status, "PENDING") || strings.EqualFold(o.Status, "AWAITING_REVIEW") {
			pending++
		}
	}
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, supplierDashboardResponse{
		SupplierID:    s.supplierID,
		IsConfigured:  current.IsConfigured,
		InventorySKUs: invCount,
		PendingOrders: pending,
		UpdatedAt:     s.now().Format(time.RFC3339Nano),
	})
}

// HandleEarnings returns scaffold supplier earnings summaries.
func (s *Service) HandleEarnings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	now := s.now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	weekStart := dayStart.AddDate(0, 0, -int(dayStart.Weekday()))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var today, week, month int64
	s.mu.RLock()
	for _, o := range s.orders {
		if !strings.EqualFold(o.Decision, "APPROVED") && !strings.EqualFold(o.Status, "COMPLETED") {
			continue
		}
		ts, err := time.Parse(time.RFC3339Nano, o.UpdatedAt)
		if err != nil {
			continue
		}
		if !ts.Before(dayStart) {
			today += o.TotalMinor
		}
		if !ts.Before(weekStart) {
			week += o.TotalMinor
		}
		if !ts.Before(monthStart) {
			month += o.TotalMinor
		}
	}
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, supplierEarningsResponse{
		Currency:   s.currency,
		TodayMinor: today,
		WeekMinor:  week,
		MonthMinor: month,
	})
}

// HandleInventory supports GET/PATCH /v1/supplier/inventory.
func (s *Service) HandleInventory(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleInventoryList(w, r)
	case http.MethodPatch:
		s.handleInventoryPatch(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleInventoryList(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	items := make([]InventoryItem, 0, len(s.inventory))
	for _, it := range s.inventory {
		items = append(items, it)
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].SKU < items[j].SKU })
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Service) handleInventoryPatch(w http.ResponseWriter, r *http.Request) {
	var req InventoryPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

	sku := strings.TrimSpace(req.SKUID)
	if sku == "" {
		sku = strings.TrimSpace(req.SKU)
	}
	if sku == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sku_id_or_sku_required"})
		return
	}
	if req.Quantity == nil && req.QuantityDelta == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "quantity_or_quantity_delta_required"})
		return
	}

	now := s.now()
	s.mu.Lock()
	item := s.inventory[sku]
	if item.SKU == "" {
		item.SKU = sku
		if strings.TrimSpace(req.ProductName) != "" {
			item.ProductName = strings.TrimSpace(req.ProductName)
		} else {
			item.ProductName = sku
		}
	}
	if strings.TrimSpace(req.ProductName) != "" {
		item.ProductName = strings.TrimSpace(req.ProductName)
	}
	before := item.Quantity
	if req.Quantity != nil {
		item.Quantity = *req.Quantity
	} else {
		item.Quantity += *req.QuantityDelta
		if item.Quantity < 0 {
			item.Quantity = 0
		}
	}
	item.UpdatedAt = now.Format(time.RFC3339Nano)
	s.inventory[sku] = item
	delta := item.Quantity - before
	entry := InventoryAuditEntry{
		SKU:         item.SKU,
		ProductName: item.ProductName,
		Before:      before,
		After:       item.Quantity,
		Delta:       delta,
		Reason:      strings.TrimSpace(req.Reason),
		Timestamp:   now.Format(time.RFC3339Nano),
	}
	s.inventoryAudit = append(s.inventoryAudit, entry)
	s.mu.Unlock()

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), "supplier:inventory:"+s.supplierID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": item, "audit": entry})
}

// HandleInventoryAudit returns additive inventory mutation history.
func (s *Service) HandleInventoryAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.RLock()
	audit := append([]InventoryAuditEntry(nil), s.inventoryAudit...)
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"entries": audit})
}

// HandleOrders returns supplier queue entries for vetting/review.
func (s *Service) HandleOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
	s.mu.RLock()
	orders := make([]SupplierOrder, 0, len(s.orders))
	for _, o := range s.orders {
		if statusFilter != "" && !strings.EqualFold(o.Status, statusFilter) {
			continue
		}
		orders = append(orders, o)
	}
	s.mu.RUnlock()
	sort.Slice(orders, func(i, j int) bool { return orders[i].UpdatedAt > orders[j].UpdatedAt })
	writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

// HandleVetOrder applies APPROVED/REJECTED decisions for supplier queue items.
func (s *Service) HandleVetOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	var req vetOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

	req.OrderID = strings.TrimSpace(req.OrderID)
	req.Decision = strings.ToUpper(strings.TrimSpace(req.Decision))
	if req.OrderID == "" || (req.Decision != "APPROVED" && req.Decision != "REJECTED") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id and decision(APPROVED|REJECTED) required"})
		return
	}

	now := s.now().Format(time.RFC3339Nano)
	s.mu.Lock()
	o := s.orders[req.OrderID]
	if o.OrderID == "" {
		o = SupplierOrder{
			OrderID:    req.OrderID,
			RetailerID: strings.TrimSpace(req.RetailerID),
			Status:     "AWAITING_REVIEW",
			TotalMinor: req.TotalMinor,
			Currency:   strings.TrimSpace(req.Currency),
			CreatedAt:  now,
		}
		if o.Currency == "" {
			o.Currency = s.currency
		}
	}
	o.Decision = req.Decision
	o.Note = strings.TrimSpace(req.Note)
	if req.Decision == "APPROVED" {
		o.Status = "APPROVED"
	} else {
		o.Status = "REJECTED"
	}
	o.UpdatedAt = now
	s.orders[o.OrderID] = o
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{"order": o})
}
