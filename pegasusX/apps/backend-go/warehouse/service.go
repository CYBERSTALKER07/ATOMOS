// Package warehouse owns warehouse-role handlers and local scaffold state.
package warehouse

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Service stores additive in-memory data for warehouse operational surfaces.
type Service struct {
	supplierID string
	currency   string
	now        func() time.Time

	mu             sync.RWMutex
	inventory      map[string]InventoryRow
	orders         []OrderRow
	supplyRequests []SupplyRequest
	locks          map[string]DispatchLock
}

// ServiceConfig is the constructor input.
type ServiceConfig struct {
	SupplierID string
	Currency   string
	Now        func() time.Time
}

// InventoryRow represents one stock row.
type InventoryRow struct {
	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`
	Quantity    int64  `json:"quantity"`
	UpdatedAt   string `json:"updated_at"`
}

// OrderRow represents one order summary in warehouse ops list.
type OrderRow struct {
	OrderID    string `json:"order_id"`
	RetailerID string `json:"retailer_id"`
	Status     string `json:"status"`
	TotalMinor int64  `json:"total_minor"`
	Currency   string `json:"currency"`
	UpdatedAt  string `json:"updated_at"`
}

// SupplyRequest represents one replenishment request row.
type SupplyRequest struct {
	RequestID   string `json:"request_id"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	RequestedBy string `json:"requested_by,omitempty"`
}

// DispatchLock represents one active dispatch lock.
type DispatchLock struct {
	LockID     string `json:"lock_id"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Reason     string `json:"reason"`
	CreatedAt  string `json:"created_at"`
}

// NewService constructs the warehouse service.
func NewService(c ServiceConfig) *Service {
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	if c.Currency == "" {
		c.Currency = "UZS"
	}
	return &Service{
		supplierID: c.SupplierID,
		currency:   c.Currency,
		now:        c.Now,
		inventory:  make(map[string]InventoryRow),
		locks:      make(map[string]DispatchLock),
	}
}

// HandleDashboard serves GET /v1/warehouse/ops/dashboard.
func (s *Service) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.RLock()
	inv := len(s.inventory)
	ord := len(s.orders)
	locks := len(s.locks)
	reqs := len(s.supplyRequests)
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"supplier_id":     s.supplierID,
		"inventory_skus":  inv,
		"orders_open":     ord,
		"dispatch_locks":  locks,
		"supply_requests": reqs,
		"updated_at":      s.now().Format(time.RFC3339Nano),
	})
}

// HandleInventory serves GET /v1/warehouse/ops/inventory.
func (s *Service) HandleInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.Lock()
	if len(s.inventory) == 0 {
		now := s.now().Format(time.RFC3339Nano)
		s.inventory["SKU-DEMO-1"] = InventoryRow{SKU: "SKU-DEMO-1", ProductName: "Demo Product", Quantity: 100, UpdatedAt: now}
	}
	rows := make([]InventoryRow, 0, len(s.inventory))
	for _, v := range s.inventory {
		rows = append(rows, v)
	}
	s.mu.Unlock()
	sort.Slice(rows, func(i, j int) bool { return rows[i].SKU < rows[j].SKU })
	writeJSON(w, http.StatusOK, map[string]any{"items": rows})
}

// HandleOrders serves GET /v1/warehouse/ops/orders.
func (s *Service) HandleOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.RLock()
	rows := append([]OrderRow(nil), s.orders...)
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"orders": rows})
}

// HandleDispatchPreview serves POST /v1/warehouse/ops/dispatch/preview.
func (s *Service) HandleDispatchPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":                "preview_ready",
		"recommended_manifests": 0,
		"estimated_eta_minutes": 0,
	})
}

// HandleDemandForecast serves GET /v1/warehouse/demand/forecast.
func (s *Service) HandleDemandForecast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	series := make([]map[string]any, 0, 7)
	start := s.now()
	for i := 0; i < 7; i++ {
		d := start.AddDate(0, 0, i)
		series = append(series, map[string]any{
			"date":              d.Format("2006-01-02"),
			"projected_units":   0,
			"projected_revenue": 0,
			"currency":          s.currency,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"series": series})
}

// HandleSupplyRequests serves GET/POST /v1/warehouse/supply-requests.
func (s *Service) HandleSupplyRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		rows := append([]SupplyRequest(nil), s.supplyRequests...)
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"requests": rows})
	case http.MethodPost:
		now := s.now().Format(time.RFC3339Nano)
		req := SupplyRequest{
			RequestID:   "sreq_" + strings.ReplaceAll(now, ":", ""),
			Status:      "OPEN",
			CreatedAt:   now,
			UpdatedAt:   now,
			RequestedBy: strings.TrimSpace(r.URL.Query().Get("requested_by")),
		}
		s.mu.Lock()
		s.supplyRequests = append(s.supplyRequests, req)
		s.mu.Unlock()
		writeJSON(w, http.StatusCreated, req)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleDispatchLocks serves GET /v1/warehouse/dispatch-locks.
func (s *Service) HandleDispatchLocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.RLock()
	locks := make([]DispatchLock, 0, len(s.locks))
	for _, v := range s.locks {
		locks = append(locks, v)
	}
	s.mu.RUnlock()
	sort.Slice(locks, func(i, j int) bool { return locks[i].CreatedAt > locks[j].CreatedAt })
	writeJSON(w, http.StatusOK, map[string]any{"locks": locks})
}

// HandleDispatchLock serves POST/DELETE /v1/warehouse/dispatch-lock.
func (s *Service) HandleDispatchLock(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var payload struct {
			EntityType string `json:"entity_type"`
			EntityID   string `json:"entity_id"`
			Reason     string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		defer r.Body.Close()
		if strings.TrimSpace(payload.EntityType) == "" || strings.TrimSpace(payload.EntityID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "entity_type and entity_id required"})
			return
		}
		now := s.now().Format(time.RFC3339Nano)
		lock := DispatchLock{
			LockID:     "lock_" + strings.ReplaceAll(now, ":", ""),
			EntityType: strings.TrimSpace(payload.EntityType),
			EntityID:   strings.TrimSpace(payload.EntityID),
			Reason:     strings.TrimSpace(payload.Reason),
			CreatedAt:  now,
		}
		s.mu.Lock()
		s.locks[lock.LockID] = lock
		s.mu.Unlock()
		writeJSON(w, http.StatusCreated, lock)
	case http.MethodDelete:
		lockID := strings.TrimSpace(r.URL.Query().Get("lock_id"))
		if lockID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lock_id required"})
			return
		}
		s.mu.Lock()
		delete(s.locks, lockID)
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"status": "released", "lock_id": lockID})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
