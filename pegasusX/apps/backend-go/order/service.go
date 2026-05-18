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
	GetOrder(ctx context.Context, orderID string) (Order, bool, error)
}

// Service wires repository + cache + ws hubs + supplier scope.
type Service struct {
	repo        Repository
	cache       *cache.Cache
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
	OrderID    string `json:"order_id"`
	Status     Status `json:"status"`
	TotalMinor int64  `json:"total_minor"`
	Currency   string `json:"currency"`
	CreatedAt  string `json:"created_at"`
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

	o := Order{
		OrderID:    s.newID(),
		SupplierID: s.supplierID,
		RetailerID: retailerID,
		Status:     StatusPending,
		LineItems:  req.LineItems,
		TotalMinor: total,
		Currency:   s.currency,
		H3Cell:     req.H3Cell,
		Lat:        req.Lat,
		Lng:        req.Lng,
		Version:    1,
		CreatedAt:  s.now(),
		UpdatedAt:  s.now(),
	}

	err := s.repo.CreateOrder(ctx, o, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, o.OrderID, events.TopicMain, orderCreatedEvent{
			Type:       events.EventOrderCreated,
			OrderID:    o.OrderID,
			SupplierID: o.SupplierID,
			RetailerID: o.RetailerID,
			Status:     string(o.Status),
			TotalMinor: o.TotalMinor,
			Currency:   o.Currency,
			H3Cell:     o.H3Cell,
			Lat:        o.Lat,
			Lng:        o.Lng,
			Timestamp:  o.CreatedAt.Format(time.RFC3339Nano),
		})
	})
	if err != nil {
		return CreateResponse{}, fmt.Errorf("persist order: %w", err)
	}

	// Post-commit cache invalidation: any retailer-orders or supplier-orders
	// list cache MUST be dropped so the next read sees the new row.
	s.cache.Invalidate(ctx,
		retailerOrdersKey(o.RetailerID),
		supplierOrdersKey(o.SupplierID),
	)

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
		"total_minor", o.TotalMinor,
		"currency", o.Currency,
		"h3_cell", o.H3Cell,
	)
	return CreateResponse{
		OrderID:    o.OrderID,
		Status:     o.Status,
		TotalMinor: o.TotalMinor,
		Currency:   o.Currency,
		CreatedAt:  o.CreatedAt.Format(time.RFC3339Nano),
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

// ── wire shapes ────────────────────────────────────────────────────────────

type orderCreatedEvent struct {
	Type       string  `json:"type"`
	OrderID    string  `json:"order_id"`
	SupplierID string  `json:"supplier_id"`
	RetailerID string  `json:"retailer_id"`
	Status     string  `json:"status"`
	TotalMinor int64   `json:"total_minor"`
	Currency   string  `json:"currency"`
	H3Cell     string  `json:"h3_cell"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	Timestamp  string  `json:"timestamp"`
}

type wsEnvelope struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Data      any    `json:"data"`
}

func orderCreatedData(o Order) map[string]any {
	return map[string]any{
		"order_id":    o.OrderID,
		"supplier_id": o.SupplierID,
		"retailer_id": o.RetailerID,
		"status":      string(o.Status),
		"total_minor": o.TotalMinor,
		"currency":    o.Currency,
		"h3_cell":     o.H3Cell,
	}
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
