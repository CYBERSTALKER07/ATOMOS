// Package warehouse owns warehouse-role handlers and local scaffold state.
package warehouse

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

// Service stores additive in-memory data for warehouse operational surfaces.
type Service struct {
	repo         Repository
	cache        *cache.Cache
	supplierHub  *ws.Hub
	warehouseHub *ws.Hub
	log          *slog.Logger

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
	Repo         Repository
	Cache        *cache.Cache
	SupplierHub  *ws.Hub
	WarehouseHub *ws.Hub
	Log          *slog.Logger

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
	if c.Log == nil {
		c.Log = slog.Default()
	}
	if c.Currency == "" {
		c.Currency = "UZS"
	}
	return &Service{
		repo:         c.Repo,
		cache:        c.Cache,
		supplierHub:  c.SupplierHub,
		warehouseHub: c.WarehouseHub,
		log:          c.Log,
		supplierID:   c.SupplierID,
		currency:     c.Currency,
		now:          c.Now,
		inventory:    make(map[string]InventoryRow),
		locks:        make(map[string]DispatchLock),
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
		nowTS := s.now().Format(time.RFC3339Nano)
		claims, _ := auth.FromContext(r.Context())
		requestedBy := strings.TrimSpace(r.URL.Query().Get("requested_by"))
		if requestedBy == "" {
			requestedBy = claims.Subject
		}
		warehouseID := strings.TrimSpace(r.URL.Query().Get("warehouse_id"))
		if warehouseID == "" {
			warehouseID = strings.TrimSpace(claims.HomeNodeID)
		}

		req := SupplyRequest{
			RequestID:   "sreq_" + strings.ReplaceAll(nowTS, ":", ""),
			Status:      "OPEN",
			CreatedAt:   nowTS,
			UpdatedAt:   nowTS,
			RequestedBy: requestedBy,
		}

		eventPayload := map[string]any{
			"type":         events.EventWarehouseSupplyRequestOpened,
			"supplier_id":  s.supplierID,
			"warehouse_id": warehouseID,
			"request_id":   req.RequestID,
			"status":       req.Status,
			"requested_by": req.RequestedBy,
			"timestamp":    nowTS,
		}

		if err := s.repo.Apply(r.Context(), func() error {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.supplyRequests = append(s.supplyRequests, req)
			return nil
		}, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateWarehouse, req.RequestID, events.TopicMain, eventPayload)
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_supply_request_failed"})
			return
		}

		if s.cache != nil {
			s.cache.Invalidate(r.Context(), warehouseSupplyRequestsKey(s.supplierID))
		}

		s.broadcastWarehouseEvent(r.Context(), warehouseID, eventPayload)
		s.log.Info("warehouse supply request opened", "supplier_id", s.supplierID, "warehouse_id", warehouseID, "request_id", req.RequestID)
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
		nowTS := s.now().Format(time.RFC3339Nano)
		claims, _ := auth.FromContext(r.Context())
		warehouseID := strings.TrimSpace(r.URL.Query().Get("warehouse_id"))
		if warehouseID == "" {
			warehouseID = strings.TrimSpace(claims.HomeNodeID)
		}
		lock := DispatchLock{
			LockID:     "lock_" + strings.ReplaceAll(nowTS, ":", ""),
			EntityType: strings.TrimSpace(payload.EntityType),
			EntityID:   strings.TrimSpace(payload.EntityID),
			Reason:     strings.TrimSpace(payload.Reason),
			CreatedAt:  nowTS,
		}

		eventPayload := map[string]any{
			"type":         events.EventWarehouseDispatchLockChanged,
			"supplier_id":  s.supplierID,
			"warehouse_id": warehouseID,
			"lock_id":      lock.LockID,
			"entity_type":  lock.EntityType,
			"entity_id":    lock.EntityID,
			"reason":       lock.Reason,
			"action":       "ACQUIRED",
			"timestamp":    lock.CreatedAt,
		}

		if err := s.repo.Apply(r.Context(), func() error {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.locks[lock.LockID] = lock
			return nil
		}, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateWarehouse, lock.LockID, events.TopicMain, eventPayload)
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_dispatch_lock_failed"})
			return
		}

		if s.cache != nil {
			s.cache.Invalidate(r.Context(), warehouseDispatchLocksKey(s.supplierID))
		}

		s.broadcastWarehouseEvent(r.Context(), warehouseID, eventPayload)
		s.log.Info("warehouse dispatch lock acquired", "supplier_id", s.supplierID, "warehouse_id", warehouseID, "lock_id", lock.LockID)
		writeJSON(w, http.StatusCreated, lock)
	case http.MethodDelete:
		lockID := strings.TrimSpace(r.URL.Query().Get("lock_id"))
		if lockID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lock_id required"})
			return
		}

		claims, _ := auth.FromContext(r.Context())
		warehouseID := strings.TrimSpace(r.URL.Query().Get("warehouse_id"))
		if warehouseID == "" {
			warehouseID = strings.TrimSpace(claims.HomeNodeID)
		}

		var released DispatchLock
		if err := s.repo.Apply(r.Context(), func() error {
			s.mu.Lock()
			defer s.mu.Unlock()
			released = s.locks[lockID]
			delete(s.locks, lockID)
			return nil
		}, func(txn outbox.TxnBuffer) error {
			eventPayload := map[string]any{
				"type":         events.EventWarehouseDispatchLockChanged,
				"supplier_id":  s.supplierID,
				"warehouse_id": warehouseID,
				"lock_id":      lockID,
				"entity_type":  released.EntityType,
				"entity_id":    released.EntityID,
				"reason":       released.Reason,
				"action":       "RELEASED",
				"timestamp":    s.now().Format(time.RFC3339Nano),
			}
			return outbox.EmitJSON(r.Context(), txn, events.AggregateWarehouse, lockID, events.TopicMain, eventPayload)
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "release_dispatch_lock_failed"})
			return
		}

		if s.cache != nil {
			s.cache.Invalidate(r.Context(), warehouseDispatchLocksKey(s.supplierID))
		}

		s.broadcastWarehouseEvent(r.Context(), warehouseID, map[string]any{
			"type":         events.EventWarehouseDispatchLockChanged,
			"supplier_id":  s.supplierID,
			"warehouse_id": warehouseID,
			"lock_id":      lockID,
			"entity_type":  released.EntityType,
			"entity_id":    released.EntityID,
			"reason":       released.Reason,
			"action":       "RELEASED",
			"timestamp":    s.now().Format(time.RFC3339Nano),
		})
		s.log.Info("warehouse dispatch lock released", "supplier_id", s.supplierID, "warehouse_id", warehouseID, "lock_id", lockID)
		writeJSON(w, http.StatusOK, map[string]any{"status": "released", "lock_id": lockID})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) broadcastWarehouseEvent(ctx context.Context, warehouseID string, payload map[string]any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if s.supplierHub != nil && s.supplierID != "" {
		s.supplierHub.Broadcast(ctx, "supplier:"+s.supplierID, raw)
	}
	if s.warehouseHub != nil && warehouseID != "" {
		s.warehouseHub.Broadcast(ctx, "warehouse:"+warehouseID, raw)
	}
}

func warehouseSupplyRequestsKey(supplierID string) string {
	return "warehouse:supply-requests:" + supplierID
}

func warehouseDispatchLocksKey(supplierID string) string {
	return "warehouse:dispatch-locks:" + supplierID
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
