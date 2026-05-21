// Package order owns the order aggregate: creation, state transitions,
// reassignment, and lifecycle event emission. Every mutating method runs
// inside a Repository.* method that wraps a ReadWriteTransaction so the row
// mutation and the OutboxEvents row commit atomically.
//
// In pegasusX every order is scoped to the single seeded supplier; the
// supplier id is bound at Service construction and never read from request
// bodies.
package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

// Status is the canonical order state. Matches packages/types OrderStatus and
// events.schema.json OrderStatus enum.
type Status string

const (
	StatusPending   Status = "PENDING"
	StatusLoaded    Status = "LOADED"
	StatusInTransit Status = "IN_TRANSIT"
	StatusArrived   Status = "ARRIVED"
	StatusCompleted Status = "COMPLETED"
	StatusCancelled Status = "CANCELLED"
)

var (
	ErrOrderNotFound           = errors.New("order_not_found")
	ErrOrderForbidden          = errors.New("order_forbidden")
	ErrInvalidStatusTransition = errors.New("invalid_status_transition")
)

// LineItem is one line on an order.
type LineItem struct {
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Quantity  int64  `json:"quantity"`
	UnitPrice int64  `json:"unit_price_minor"` // minor units (tiyin / cents)
}

// Order is the persisted aggregate.
type Order struct {
	OrderID     string
	SupplierID  string
	RetailerID  string
	WarehouseID string
	Status      Status
	LineItems   []LineItem
	TotalMinor  int64
	Currency    string
	H3Cell      string
	Lat         float64
	Lng         float64
	Version     int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Repository is the storage seam. Production binds to Spanner; the emit
// callback receives a TxnBuffer scoped to the same RW transaction so the row
// + outbox event commit atomically.
type Repository interface {
	CreateOrder(ctx context.Context, o Order, emit func(outbox.TxnBuffer) error) error
	UpdateOrder(ctx context.Context, o Order, emit func(outbox.TxnBuffer) error) error
	GetOrder(ctx context.Context, orderID string) (Order, bool, error)
}

// WarehouseResolver resolves the best supplier warehouse for retailer
// coordinates at order-create time.
type WarehouseResolver interface {
	ResolveNearestWarehouseID(ctx context.Context, supplierID string, retailerLat, retailerLng float64) (string, error)
}

// Service wires repository + cache + ws hubs + supplier scope.
type Service struct {
	repo        Repository
	cache       *cache.Cache
	warehouse   WarehouseResolver
	supplierID  string
	currency    string
	retailerHub *ws.Hub
	supplierHub *ws.Hub
	log         *slog.Logger
	now         func() time.Time
	newID       func() string
}

// ServiceConfig is the constructor input.
type ServiceConfig struct {
	Repo        Repository
	Cache       *cache.Cache
	Warehouse   WarehouseResolver
	SupplierID  string
	Currency    string
	RetailerHub *ws.Hub
	SupplierHub *ws.Hub
	Log         *slog.Logger
	Now         func() time.Time
	NewID       func() string
}

// NewService constructs a Service with default Now/NewID.
func NewService(c ServiceConfig) *Service {
	if c.Log == nil {
		c.Log = slog.Default()
	}
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	if c.NewID == nil {
		c.NewID = defaultOrderID
	}
	return &Service{
		repo:        c.Repo,
		cache:       c.Cache,
		warehouse:   c.Warehouse,
		supplierID:  c.SupplierID,
		currency:    c.Currency,
		retailerHub: c.RetailerHub,
		supplierHub: c.SupplierHub,
		log:         c.Log,
		now:         c.Now,
		newID:       c.NewID,
	}
}

// CreateRequest is the wire shape for POST /v1/order/create.
type CreateRequest struct {
	LineItems []LineItem `json:"line_items"`
	H3Cell    string     `json:"h3_cell"`
	Lat       float64    `json:"lat"`
	Lng       float64    `json:"lng"`
}

// CreateResponse is what callers get back.
type CreateResponse struct {
	OrderID     string `json:"order_id"`
	WarehouseID string `json:"warehouse_id,omitempty"`
	Status      Status `json:"status"`
	TotalMinor  int64  `json:"total_minor"`
	Currency    string `json:"currency"`
	CreatedAt   string `json:"created_at"`
}

// UpdateStatusRequest is the wire shape for PATCH /v1/order/{orderID}/status.
type UpdateStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// UpdateStatusResponse confirms the post-mutation status snapshot.
type UpdateStatusResponse struct {
	OrderID        string `json:"order_id"`
	PreviousStatus Status `json:"previous_status"`
	Status         Status `json:"status"`
	Version        int64  `json:"version"`
	UpdatedAt      string `json:"updated_at"`
	EventType      string `json:"event_type"`
}

// Validate enforces input invariants.
func (r CreateRequest) Validate() error {
	if len(r.LineItems) == 0 {
		return errors.New("line_items required")
	}
	for i, li := range r.LineItems {
		if strings.TrimSpace(li.SKU) == "" {
			return fmt.Errorf("line_items[%d].sku required", i)
		}
		if li.Quantity <= 0 {
			return fmt.Errorf("line_items[%d].quantity must be > 0", i)
		}
		if li.UnitPrice < 0 {
			return fmt.Errorf("line_items[%d].unit_price_minor must be >= 0", i)
		}
	}
	if len(r.H3Cell) != 15 {
		return fmt.Errorf("h3_cell must be 15-char hex, got %d", len(r.H3Cell))
	}
	return nil
}

func (r UpdateStatusRequest) normalizedStatus() Status {
	return Status(strings.ToUpper(strings.TrimSpace(r.Status)))
}

// Validate enforces update-status input invariants and returns normalized status.
func (r UpdateStatusRequest) Validate() (Status, error) {
	next := r.normalizedStatus()
	if next == "" {
		return "", errors.New("status required")
	}
	switch next {
	case StatusPending, StatusLoaded, StatusInTransit, StatusArrived, StatusCompleted, StatusCancelled:
		return next, nil
	default:
		return "", fmt.Errorf("unsupported status: %s", next)
	}
}

// Create runs the order-creation mutation: validate, total, persist row +
// ORDER_CREATED outbox event in one RW txn, then ws fanout to retailer +
// supplier rooms (best-effort, fail-open).
func (s *Service) Create(ctx context.Context, retailerID string, req CreateRequest) (CreateResponse, error) {
	if err := req.Validate(); err != nil {
		return CreateResponse{}, err
	}
	if retailerID == "" {
		return CreateResponse{}, errors.New("retailer_id required from session")
	}

	var total int64
	for _, li := range req.LineItems {
		total += li.UnitPrice * li.Quantity
	}

	warehouseID := ""
	if s.warehouse != nil && (req.Lat != 0 || req.Lng != 0) {
		resolvedWarehouseID, err := s.warehouse.ResolveNearestWarehouseID(ctx, s.supplierID, req.Lat, req.Lng)
		if err != nil {
			return CreateResponse{}, fmt.Errorf("resolve nearest warehouse: %w", err)
		}
		warehouseID = strings.TrimSpace(resolvedWarehouseID)
	}

	o := Order{
		OrderID:     s.newID(),
		SupplierID:  s.supplierID,
		RetailerID:  retailerID,
		WarehouseID: warehouseID,
		Status:      StatusPending,
		LineItems:   req.LineItems,
		TotalMinor:  total,
		Currency:    s.currency,
		H3Cell:      req.H3Cell,
		Lat:         req.Lat,
		Lng:         req.Lng,
		Version:     1,
		CreatedAt:   s.now(),
		UpdatedAt:   s.now(),
	}

	err := s.repo.CreateOrder(ctx, o, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, o.OrderID, events.TopicMain, orderCreatedEvent{
			Type:        events.EventOrderCreated,
			OrderID:     o.OrderID,
			SupplierID:  o.SupplierID,
			RetailerID:  o.RetailerID,
			WarehouseID: o.WarehouseID,
			Status:      string(o.Status),
			TotalMinor:  o.TotalMinor,
			Currency:    o.Currency,
			H3Cell:      o.H3Cell,
			Lat:         o.Lat,
			Lng:         o.Lng,
			Timestamp:   o.CreatedAt.Format(time.RFC3339Nano),
		})
	})
	if err != nil {
		return CreateResponse{}, fmt.Errorf("persist order: %w", err)
	}

	// Post-commit cache invalidation: any retailer-orders or supplier-orders
	// list cache MUST be dropped so the next read sees the new row.
	if s.cache != nil {
		s.cache.Invalidate(ctx,
			retailerOrdersKey(o.RetailerID),
			supplierOrdersKey(o.SupplierID),
		)
	}

	// Best-effort ws fanout. Failures are absorbed by Hub.Broadcast.
	envelope := wsEnvelope{
		Type:      events.EventOrderCreated,
		Timestamp: o.CreatedAt.Format(time.RFC3339Nano),
		Data:      orderCreatedData(o),
	}
	payload, _ := json.Marshal(envelope)
	if s.retailerHub != nil {
		s.retailerHub.Broadcast(ctx, "retailer:"+o.RetailerID, payload)
	}
	if s.supplierHub != nil {
		s.supplierHub.Broadcast(ctx, "supplier:"+o.SupplierID, payload)
	}

	s.log.Info("order created",
		"order_id", o.OrderID,
		"supplier_id", o.SupplierID,
		"retailer_id", o.RetailerID,
		"warehouse_id", o.WarehouseID,
		"total_minor", o.TotalMinor,
		"currency", o.Currency,
		"h3_cell", o.H3Cell,
	)
	return CreateResponse{
		OrderID:     o.OrderID,
		WarehouseID: o.WarehouseID,
		Status:      o.Status,
		TotalMinor:  o.TotalMinor,
		Currency:    o.Currency,
		CreatedAt:   o.CreatedAt.Format(time.RFC3339Nano),
	}, nil
}

// UpdateStatus transitions one order across the canonical lifecycle. Mutations
// and outbox emission happen atomically in repository UpdateOrder.
func (s *Service) UpdateStatus(ctx context.Context, claims auth.Claims, orderID string, req UpdateStatusRequest) (UpdateStatusResponse, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return UpdateStatusResponse{}, errors.New("order_id required")
	}

	nextStatus, err := req.Validate()
	if err != nil {
		return UpdateStatusResponse{}, err
	}

	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return UpdateStatusResponse{}, fmt.Errorf("get order %s: %w", orderID, err)
	}
	if !ok {
		return UpdateStatusResponse{}, ErrOrderNotFound
	}

	if claims.Role != auth.RoleAdmin && claims.Role != auth.RoleRetailer {
		return UpdateStatusResponse{}, ErrOrderForbidden
	}
	if claims.Role == auth.RoleRetailer {
		if claims.Subject == "" || claims.Subject != current.RetailerID || nextStatus != StatusCancelled {
			return UpdateStatusResponse{}, ErrOrderForbidden
		}
	}

	if err := validateStatusTransition(current.Status, nextStatus); err != nil {
		return UpdateStatusResponse{}, err
	}

	if current.Status == nextStatus {
		return UpdateStatusResponse{
			OrderID:        current.OrderID,
			PreviousStatus: current.Status,
			Status:         current.Status,
			Version:        current.Version,
			UpdatedAt:      current.UpdatedAt.Format(time.RFC3339Nano),
			EventType:      events.EventOrderStatusChanged,
		}, nil
	}

	prevStatus := current.Status
	current.Status = nextStatus
	current.UpdatedAt = s.now()

	actorID := claims.Subject
	if actorID == "" {
		actorID = claims.SupplierID
	}

	err = s.repo.UpdateOrder(ctx, current, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, orderStatusChangedEvent{
			Type:           events.EventOrderStatusChanged,
			OrderID:        current.OrderID,
			SupplierID:     current.SupplierID,
			RetailerID:     current.RetailerID,
			PreviousStatus: string(prevStatus),
			Status:         string(current.Status),
			Reason:         strings.TrimSpace(req.Reason),
			ActorRole:      string(claims.Role),
			ActorID:        actorID,
			Timestamp:      current.UpdatedAt.Format(time.RFC3339Nano),
		})
	})
	if err != nil {
		return UpdateStatusResponse{}, fmt.Errorf("update order status %s: %w", orderID, err)
	}

	if s.cache != nil {
		s.cache.Invalidate(ctx,
			retailerOrdersKey(current.RetailerID),
			supplierOrdersKey(current.SupplierID),
		)
	}

	envelope := wsEnvelope{
		Type:      events.EventOrderStatusChanged,
		Timestamp: current.UpdatedAt.Format(time.RFC3339Nano),
		Data:      orderStatusChangedData(current, prevStatus, strings.TrimSpace(req.Reason), current.Version+1),
	}
	payload, _ := json.Marshal(envelope)
	if s.retailerHub != nil {
		s.retailerHub.Broadcast(ctx, "retailer:"+current.RetailerID, payload)
	}
	if s.supplierHub != nil {
		s.supplierHub.Broadcast(ctx, "supplier:"+current.SupplierID, payload)
	}

	s.log.Info("order status updated",
		"order_id", current.OrderID,
		"supplier_id", current.SupplierID,
		"retailer_id", current.RetailerID,
		"prev_status", prevStatus,
		"status", current.Status,
		"actor_role", claims.Role,
		"actor_id", actorID,
	)

	return UpdateStatusResponse{
		OrderID:        current.OrderID,
		PreviousStatus: prevStatus,
		Status:         current.Status,
		Version:        current.Version + 1,
		UpdatedAt:      current.UpdatedAt.Format(time.RFC3339Nano),
		EventType:      events.EventOrderStatusChanged,
	}, nil
}

// HandleCreate is POST /v1/order/create. Wired by orderroutes.RegisterRoutes
// behind auth.RequireRole(RETAILER).
func (s *Service) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Subject == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

	// Subject IS the retailer id for RETAILER-role callers.
	resp, err := s.Create(r.Context(), claims.Subject, req)
	if err != nil {
		s.log.Warn("order create failed",
			"retailer_id", claims.Subject, "err", err)
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// HandleUpdateStatus is PATCH /v1/order/{orderID}/status.
func (s *Service) HandleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := chi.URLParam(r, "orderID")
	if strings.TrimSpace(orderID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

	resp, err := s.UpdateStatus(r.Context(), claims, orderID, req)
	if err != nil {
		s.log.Warn("order status update failed", "order_id", orderID, "err", err)
		switch {
		case errors.Is(err, ErrOrderNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		case errors.Is(err, ErrOrderForbidden):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		case errors.Is(err, ErrInvalidStatusTransition):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_status_transition"})
		default:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ── wire shapes ────────────────────────────────────────────────────────────

type orderCreatedEvent struct {
	Type        string  `json:"type"`
	OrderID     string  `json:"order_id"`
	SupplierID  string  `json:"supplier_id"`
	RetailerID  string  `json:"retailer_id"`
	WarehouseID string  `json:"warehouse_id,omitempty"`
	Status      string  `json:"status"`
	TotalMinor  int64   `json:"total_minor"`
	Currency    string  `json:"currency"`
	H3Cell      string  `json:"h3_cell"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	Timestamp   string  `json:"timestamp"`
}

type orderStatusChangedEvent struct {
	Type           string `json:"type"`
	OrderID        string `json:"order_id"`
	SupplierID     string `json:"supplier_id"`
	RetailerID     string `json:"retailer_id"`
	PreviousStatus string `json:"previous_status"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
	ActorRole      string `json:"actor_role"`
	ActorID        string `json:"actor_id,omitempty"`
	Timestamp      string `json:"timestamp"`
}

type wsEnvelope struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Data      any    `json:"data"`
}

func orderCreatedData(o Order) map[string]any {
	return map[string]any{
		"order_id":     o.OrderID,
		"supplier_id":  o.SupplierID,
		"retailer_id":  o.RetailerID,
		"warehouse_id": o.WarehouseID,
		"status":       string(o.Status),
		"total_minor":  o.TotalMinor,
		"currency":     o.Currency,
		"h3_cell":      o.H3Cell,
	}
}

func orderStatusChangedData(o Order, previous Status, reason string, version int64) map[string]any {
	return map[string]any{
		"order_id":        o.OrderID,
		"supplier_id":     o.SupplierID,
		"retailer_id":     o.RetailerID,
		"previous_status": string(previous),
		"status":          string(o.Status),
		"reason":          reason,
		"version":         version,
		"total_minor":     o.TotalMinor,
		"currency":        o.Currency,
	}
}

func validateStatusTransition(current Status, next Status) error {
	if current == next {
		return nil
	}

	allowed := false
	switch current {
	case StatusPending:
		allowed = next == StatusLoaded || next == StatusCancelled
	case StatusLoaded:
		allowed = next == StatusInTransit || next == StatusCancelled
	case StatusInTransit:
		allowed = next == StatusArrived
	case StatusArrived:
		allowed = next == StatusCompleted
	case StatusCompleted, StatusCancelled:
		allowed = false
	default:
		allowed = false
	}

	if !allowed {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidStatusTransition, current, next)
	}

	return nil
}

// ── helpers ────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func retailerOrdersKey(retailerID string) string {
	return "orders:retailer:" + retailerID
}

func supplierOrdersKey(supplierID string) string {
	return "orders:supplier:" + supplierID
}

func defaultOrderID() string {
	// Scaffold: timestamp-based id. Production swaps for uuid.NewV7.
	return fmt.Sprintf("ord_%d", time.Now().UnixNano())
}
