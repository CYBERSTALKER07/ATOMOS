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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/promotion"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"github.com/pegasusx/pegasusx/packages/handoff"
	"google.golang.org/api/iterator"
)

// Status is the canonical order state. Matches packages/types OrderStatus and
// events.schema.json OrderStatus enum.
type Status string

const (
	StatusPending                Status = "PENDING"
	StatusLoaded                 Status = "LOADED"
	StatusInTransit              Status = "IN_TRANSIT"
	StatusArrived                Status = "ARRIVED"
	StatusArrivedShopClosed      Status = "ARRIVED_SHOP_CLOSED"
	StatusAwaitingPayment        Status = "AWAITING_PAYMENT"
	StatusPendingCashCollection  Status = "PENDING_CASH_COLLECTION"
	StatusDeliveredOnCredit      Status = "DELIVERED_ON_CREDIT"
	StatusFiscalizing            Status = "FISCALIZING"   // ADR-009: payment captured, OFD in flight
	StatusFiscalFailed           Status = "FISCAL_FAILED" // ADR-009: last OFD attempt failed
	StatusCompleted              Status = "COMPLETED"
	StatusCancelled              Status = "CANCELLED"
	StatusCancelRequested        Status = "CANCEL_REQUESTED"
	StatusReconciliationRequired Status = "RECONCILIATION_REQUIRED"
	StatusDelayed                Status = "DELAYED"
	StatusBackordered            Status = "BACKORDERED"
	StatusScheduled              Status = "SCHEDULED"
	StatusAutoAccepted           Status = "AUTO_ACCEPTED"

	deliveryGeofenceMeters = proximity.DeliveryApproachRadiusM
)

// OrderSource captures how an order entered the system.
type OrderSource string

const (
	OrderSourceManual         OrderSource = "MANUAL"
	OrderSourceManualPreorder OrderSource = "MANUAL_PREORDER"
	OrderSourceAIPreorder     OrderSource = "AI_PREORDER"
	OrderSourceBackorder      OrderSource = "BACKORDER"
)

// ConfirmationStatus captures whether a future-dated order still needs a
// retailer decision before downstream planning should treat it as committed.
type ConfirmationStatus string

const (
	ConfirmationStatusConfirmed        ConfirmationStatus = "CONFIRMED"
	ConfirmationStatusDraft            ConfirmationStatus = "DRAFT"
	ConfirmationStatusPending          ConfirmationStatus = "PENDING"
	ConfirmationStatusRejected         ConfirmationStatus = "REJECTED"
	ConfirmationStatusAutoConfirmed    ConfirmationStatus = "AUTO_CONFIRMED"
	ConfirmationStatusPendingWarehouse ConfirmationStatus = "PENDING_WAREHOUSE"
)

var (
	ErrOrderNotFound             = errors.New("order_not_found")
	ErrOrderForbidden            = errors.New("order_forbidden")
	ErrOrderCancelLocked         = errors.New("order_cancel_locked")
	ErrInvalidStatusTransition   = errors.New("invalid_status_transition")
	ErrGeofenceViolation         = errors.New("geofence_violation")
	ErrZoneMiss                  = errors.New("zone_miss")
	ErrServiceabilityUnavailable = errors.New("delivery_perimeter_unavailable")
	ErrAssignmentRequired        = errors.New("assignment_required")
	ErrInventoryExhausted        = errors.New("inventory_exhausted")
	ErrBackorderPaymentDeferred  = errors.New("backorder_payment_deferred")
	ErrPaymentBeforeDelivery     = errors.New("payment_before_delivery_not_allowed")
	ErrCreditLimitBreached       = errors.New("credit_limit_breached")
	ErrFiscalNotFailed           = errors.New("fiscal_not_failed")
	ErrForceCompleteForbidden    = errors.New("force_complete_forbidden")
	ErrForceReasonRequired       = errors.New("force_reason_required")
	ErrForceReasonInvalid        = errors.New("force_reason_invalid")
	ErrFiscalAlreadySucceeded    = errors.New("fiscal_already_succeeded")
	ErrCashAmountRequired        = errors.New("cash_amount_required")
	ErrCashAmountNegative        = errors.New("cash_amount_negative")
)

// LineItem is one line on an order.
type LineItem struct {
	SKU               string   `json:"sku"`
	Name              string   `json:"name"`
	Quantity          int64    `json:"quantity"`
	UnitPrice         int64    `json:"unit_price_minor"` // minor units (tiyin / cents)
	UnitVolumeVU      float64  `json:"unit_volume_vu,omitempty"`
	PromotionID       string   `json:"promotion_id,omitempty"`
	HandlingClass     string   `json:"handling_class,omitempty"`
	RequiresColdChain bool     `json:"requires_cold_chain,omitempty"`
	IsHazardous       bool     `json:"is_hazardous,omitempty"`
	IsPerishable      bool     `json:"is_perishable,omitempty"`
	StorageTempMinC   *float64 `json:"storage_temp_min_c,omitempty"`
	StorageTempMaxC   *float64 `json:"storage_temp_max_c,omitempty"`
}

// Order is the persisted aggregate.
type Order struct {
	OrderID                string
	SupplierID             string
	RetailerID             string
	WarehouseID            string
	DriverID               string
	VehicleID              string
	RouteID                string
	ManifestID             string
	Status                 Status
	Source                 OrderSource
	ConfirmationStatus     ConfirmationStatus
	LineItems              []LineItem
	TotalMinor             int64
	OriginalTotalMinor     int64
	Currency               string
	H3Cell                 string
	Lat                    float64
	Lng                    float64
	QRToken                string
	RequestedDeliveryDate  *time.Time
	AutoConfirmAt          *time.Time
	DecisionAt             *time.Time
	DecisionBy             string
	DerivedFromOrderID     string
	ReceivingWindowOpen    string
	ReceivingWindowClose   string
	Timezone               string
	DeliverBefore          *time.Time
	DeliveryPriority       DeliveryPriority
	DeliveryFeeMinor       int64
	WarehouseNotes         string
	PreorderReminderSentAt *time.Time
	NudgeNotifiedAt        *time.Time
	ConfirmationNotifiedAt *time.Time
	CancelLockedAt         *time.Time
	CancelLockReason       string
	CancelLockExpiresAt    *time.Time
	ProposedDeliveryDate   *time.Time
	DeliveryProposalAt     *time.Time
	DeliveryProposalBy     string
	DeliveryProposalReason string
	Version                int64
	CreatedAt              time.Time
	UpdatedAt              time.Time

	// Denormalized fiscal rollup (Orders columns; ADR-009).
	FiscalStatus           string
	LatestFiscalReceiptID  string
	LatestFiscalAttemptID  string
	FiscalizedAt           *time.Time

	// PendingSupplierReturns is written in the same UpdateOrder transaction and not stored on Orders.
	PendingSupplierReturns []SupplierReturn `json:"-"`
	// ConditionReports is written in the same UpdateOrder transaction and not stored on Orders.
	ConditionReports []ConditionReport `json:"-"`
	// PendingFiscalReceipts are inserted in the same UpdateOrder transaction (ADR-009).
	PendingFiscalReceipts []FiscalReceiptRow `json:"-"`
	// FiscalReceiptUpdate updates an existing attempt row (PENDING → SUCCESS|FAILED).
	FiscalReceiptUpdate *FiscalReceiptRow `json:"-"`
}

// DeliveryProofType classifies one immutable delivery handoff proof row.
type DeliveryProofType string

const (
	DeliveryProofTypeQRHandoff         DeliveryProofType = "QR_HANDOFF"
	DeliveryProofTypeFinalizationGeo   DeliveryProofType = "FINALIZATION_GEOFENCE"
	DeliveryProofTypeCashCollectionGeo DeliveryProofType = "CASH_COLLECTION_GEOFENCE"
)

// DeliveryProofArtifact is an immutable handoff-evidence row written in the
// same transaction as a driver delivery transition.
type DeliveryProofArtifact struct {
	ProofID          string
	OrderID          string
	SupplierID       string
	RetailerID       string
	DriverID         string
	ProofType        DeliveryProofType
	QRTokenHash      string
	ScannedTokenHash string
	Latitude         *float64
	Longitude        *float64
	DistanceM        *float64
	CapturedAt       time.Time
}

// Repository is the storage seam. Production binds to Spanner; the emit
// callback receives a TxnBuffer scoped to the same RW transaction so the row
// + outbox event commit atomically.
type Repository interface {
	CreateOrder(ctx context.Context, o *Order, emit func(outbox.TxnBuffer) error) error
	UpdateOrder(ctx context.Context, o Order, proofs []DeliveryProofArtifact, emit func(outbox.TxnBuffer) error) error
	GetOrder(ctx context.Context, orderID string) (Order, bool, error)
	// GetFiscalAttempt loads one OrderFiscalReceipts row (ADR-009 worker idempotency).
	GetFiscalAttempt(ctx context.Context, orderID, attemptID string) (FiscalReceiptRow, bool, error)
	// CountFiscalAttemptsByStatus counts attempts for retry-budget decisions.
	CountFiscalAttemptsByStatus(ctx context.Context, orderID, status string) (int64, error)
	ListRetailerOrders(ctx context.Context, retailerID string, limit int) ([]Order, error)
	ListWarehouseOrdersByDeliveryWindow(ctx context.Context, warehouseID string, from, to time.Time, limit int) ([]Order, error)
	ListDueAutoConfirmOrders(ctx context.Context, before time.Time, limit int) ([]Order, error)
	ListManifestOrders(ctx context.Context, manifestID string) ([]Order, error)
	ListWarehousePreorders(ctx context.Context, warehouseID string, limit, offset int) ([]Order, error)
	ListOrdersForStockCommitment(ctx context.Context, warehouseID string, limit int) ([]Order, error)
	ListBackorderedOrders(ctx context.Context, limit int) ([]Order, error)
	ClearBackorder(ctx context.Context, orderID string, emit func(outbox.TxnBuffer) error) error
	ListOrdersByStatus(ctx context.Context, supplierID, status string, limit int) ([]Order, error)
	CreateConditionReport(ctx context.Context, report ConditionReport, emit func(outbox.TxnBuffer) error) error
	ListConditionReports(ctx context.Context, orderID string) ([]ConditionReport, error)
	FindSiblingDriversForOrder(ctx context.Context, orderID string) ([]string, error)
}

// WarehouseResolver resolves the best supplier warehouse for retailer
// coordinates at order-create time.
type WarehouseResolver interface {
	ResolveNearestWarehouseID(ctx context.Context, supplierID string, retailerLat, retailerLng float64) (string, error)
}

// PaymentCapturer allows the order service to trigger synchronous card captures.
type PaymentCapturer interface {
	CaptureCardPayment(ctx context.Context, orderID string, amountMinor int64, currency string) error
}

// RateLimiter is a simple interface for rate limiting.
type RateLimiter interface {
	Allow(key string) bool
}

// simpleRateLimiter is a basic in-memory token bucket rate limiter.
type simpleRateLimiter struct {
	mu       sync.Mutex
	lastTime map[string]time.Time
	interval time.Duration
}

func (r *simpleRateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if last, ok := r.lastTime[key]; ok && now.Sub(last) < r.interval {
		return false
	}
	r.lastTime[key] = now
	return true
}

func newSimpleRateLimiter(interval time.Duration) *simpleRateLimiter {
	return &simpleRateLimiter{
		lastTime: make(map[string]time.Time),
		interval: interval,
	}
}

// Service wires repository + cache + ws hubs + supplier scope.
type Service struct {
	repo            Repository
	cache           *cache.Cache
	warehouse       WarehouseResolver
	paymentCapturer PaymentCapturer
	promotions      *promotion.Service
	credit          *credit.Service

	supplierID         string
	supplierName       string
	currency           string
	retailerHub        *ws.Hub
	supplierHub        *ws.Hub
	driverHub          *ws.Hub
	spannerClient      *spanner.Client
	manifestStore      *manifest.Store
	idem               idempotency.Store
	shopGrace          time.Duration
	log                *slog.Logger
	now                func() time.Time
	newID              func() string
	jwtSecret          string
	handoff            *handoff.Engine
	gatewayPolicy      GatewayPolicyReader
	dispatchPlanWarm   func(ctx context.Context, warehouseID string)
	previewRateLimiter RateLimiter
	ofd                FiscalProvider // optional; nil → ProviderFromEnv()
}

// ServiceConfig is the constructor input.
type ServiceConfig struct {
	Repo            Repository
	Cache           *cache.Cache
	Warehouse       WarehouseResolver
	Promotions      *promotion.Service
	Credit          *credit.Service
	SupplierID      string
	SupplierName    string
	Currency        string
	RetailerHub     *ws.Hub
	SupplierHub     *ws.Hub
	DriverHub       *ws.Hub
	SpannerClient   *spanner.Client
	ShopClosedGrace time.Duration
	Log             *slog.Logger
	Now             func() time.Time
	NewID           func() string
	JWTSecret       string
	Handoff         *handoff.Engine
	Idem            idempotency.Store
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
	grace := c.ShopClosedGrace
	if grace <= 0 {
		grace = 5 * time.Minute
	}
	svc := &Service{
		repo:               c.Repo,
		cache:              c.Cache,
		warehouse:          c.Warehouse,
		promotions:         c.Promotions,
		supplierID:         c.SupplierID,
		supplierName:       strings.TrimSpace(c.SupplierName),
		currency:           c.Currency,
		retailerHub:        c.RetailerHub,
		supplierHub:        c.SupplierHub,
		driverHub:          c.DriverHub,
		spannerClient:      c.SpannerClient,
		shopGrace:          grace,
		log:                c.Log,
		now:                c.Now,
		newID:              c.NewID,
		jwtSecret:          c.JWTSecret,
		handoff:            c.Handoff,
		idem:               c.Idem,
		credit:             c.Credit,
		previewRateLimiter: newSimpleRateLimiter(100 * time.Millisecond),
	}
	if svc.handoff == nil {
		svc.handoff = handoff.FromEnv()
	}
	return svc
}

// SetPaymentCapturer sets the capturer after construction.
func (s *Service) SetPaymentCapturer(pc PaymentCapturer) {
	s.paymentCapturer = pc
}

// SetManifestStore wires manifest persistence for route geometry refresh after reorder.
func (s *Service) SetManifestStore(store *manifest.Store) {
	s.manifestStore = store
}

// SetCreditService wires the credit service for order-time limit checks and
// credit-delivery balance updates.
func (s *Service) SetCreditService(svc *credit.Service) {
	s.credit = svc
}

// SetGatewayPolicyReader wires payment gateway policy for PAYMENT_REQUIRED fanout.
func (s *Service) SetGatewayPolicyReader(reader GatewayPolicyReader) {
	s.gatewayPolicy = reader
}

// CreateRequest is the wire shape for POST /v1/order/create.
type CreateRequest struct {
	LineItems             []LineItem `json:"line_items"`
	H3Cell                string     `json:"h3_cell"`
	Lat                   float64    `json:"lat"`
	Lng                   float64    `json:"lng"`
	DeliveryMode          string     `json:"delivery_mode,omitempty"`
	RequestedDeliveryDate string     `json:"requested_delivery_date,omitempty"`
	DeliverBefore         string     `json:"deliver_before,omitempty"`
	DeliveryPriority      string     `json:"delivery_priority,omitempty"`
	CheckoutPolicyToken   string     `json:"checkout_policy_token,omitempty"`
}

// CreateResponse is what callers get back.
type CreateResponse struct {
	OrderID               string             `json:"order_id"`
	WarehouseID           string             `json:"warehouse_id,omitempty"`
	Status                Status             `json:"status"`
	Source                OrderSource        `json:"order_source"`
	ConfirmationStatus    ConfirmationStatus `json:"confirmation_status"`
	RequestedDeliveryDate string             `json:"requested_delivery_date,omitempty"`
	DeliverBefore         string             `json:"deliver_before,omitempty"`
	DeliveryPriority      string             `json:"delivery_priority,omitempty"`
	DeliveryMode          string             `json:"delivery_mode,omitempty"`
	TotalMinor            int64              `json:"total_minor"`
	Currency              string             `json:"currency"`
	CreatedAt             string             `json:"created_at"`
	ReceivingWindowOpen   string             `json:"receiving_window_open,omitempty"`
	ReceivingWindowClose  string             `json:"receiving_window_close,omitempty"`
	BackorderOrderID      string             `json:"backorder_order_id,omitempty"`
	BackorderedItemCount  int                `json:"backordered_item_count,omitempty"`
	StockWarnings         []StockWarning     `json:"stock_warnings,omitempty"`
}

// ConfirmAIOrderRequest confirms an AI-created future order and optionally
// applies retailer edits before confirming it.
type ConfirmAIOrderRequest struct {
	OrderID               string     `json:"order_id"`
	LineItems             []LineItem `json:"line_items,omitempty"`
	RequestedDeliveryDate string     `json:"requested_delivery_date,omitempty"`
}

// RejectAIOrderRequest rejects an AI-created future order.
type RejectAIOrderRequest struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason,omitempty"`
}

// EditPreorderRequest edits a scheduled manual preorder.
type EditPreorderRequest struct {
	OrderID               string     `json:"order_id"`
	LineItems             []LineItem `json:"line_items"`
	RequestedDeliveryDate string     `json:"requested_delivery_date"`
}

// ConfirmPreorderRequest confirms a draft manual preorder.
type ConfirmPreorderRequest struct {
	OrderID string `json:"order_id"`
}

// AcceptDeliveryProposalRequest accepts a warehouse-proposed delivery date.
type AcceptDeliveryProposalRequest struct {
	OrderID string `json:"order_id"`
}

// RejectDeliveryProposalRequest rejects a warehouse proposal and cancels the order.
type RejectDeliveryProposalRequest struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason,omitempty"`
}

// RejectPreorderRequest lets a retailer cancel a draft/scheduled pre-order.
type RejectPreorderRequest struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason,omitempty"`
}

// ProposeDeliveryDateRequest is the warehouse body for proposing a new delivery date.
type ProposeDeliveryDateRequest struct {
	ProposedDeliveryDate string `json:"proposed_delivery_date"`
	Reason               string `json:"reason"`
}

// RetailerOrderLifecycleResponse returns a durable order-side snapshot for AI
// and preorder actions.
type RetailerOrderLifecycleResponse struct {
	OrderID                string             `json:"order_id"`
	Status                 Status             `json:"status"`
	Source                 OrderSource        `json:"order_source"`
	ConfirmationStatus     ConfirmationStatus `json:"confirmation_status"`
	RequestedDeliveryDate  string             `json:"requested_delivery_date,omitempty"`
	DeliverBefore          string             `json:"deliver_before,omitempty"`
	DeliveryPriority       string             `json:"delivery_priority,omitempty"`
	DeliveryMode           string             `json:"delivery_mode,omitempty"`
	PreorderBadge          string             `json:"preorder_badge,omitempty"`
	ProposedDeliveryDate   string             `json:"proposed_delivery_date,omitempty"`
	DeliveryProposalReason string             `json:"delivery_proposal_reason,omitempty"`
	AutoConfirmAt          string             `json:"auto_confirm_at,omitempty"`
	TotalMinor             int64              `json:"total_minor"`
	Currency               string             `json:"currency"`
	Version                int64              `json:"version"`
	UpdatedAt              string             `json:"updated_at"`
	Created                bool               `json:"created,omitempty"`
}

// RetailerAIPrediction projects a pending AI preorder for retailer review.
type RetailerAIPrediction struct {
	OrderID               string             `json:"order_id"`
	Source                OrderSource        `json:"order_source"`
	ConfirmationStatus    ConfirmationStatus `json:"confirmation_status"`
	RequestedDeliveryDate string             `json:"requested_delivery_date,omitempty"`
	AutoConfirmAt         string             `json:"auto_confirm_at,omitempty"`
	TotalMinor            int64              `json:"total_minor"`
	Currency              string             `json:"currency"`
	DerivedFromOrderID    string             `json:"derived_from_order_id,omitempty"`
	UpdatedAt             string             `json:"updated_at"`
	Items                 []LineItem         `json:"line_items"`
}

// WarehouseDemandDay is one forecast bucket for warehouse planning.
type WarehouseDemandDay struct {
	Date                     string `json:"date"`
	ProjectedUnits           int64  `json:"projected_units"`
	ProjectedRevenue         int64  `json:"projected_revenue"`
	CommittedUnits           int64  `json:"committed_units"`
	PendingConfirmationUnits int64  `json:"pending_confirmation_units"`
	Currency                 string `json:"currency"`
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

// AssignOrderRequest is the wire shape for POST /v1/orders/{orderID}/assign.
type AssignOrderRequest struct {
	DriverID   string `json:"driver_id"`
	VehicleID  string `json:"vehicle_id,omitempty"`
	RouteID    string `json:"route_id"`
	ManifestID string `json:"manifest_id,omitempty"`
}

// AssignOrderResponse returns the durable assignment snapshot.
type AssignOrderResponse struct {
	OrderID    string `json:"order_id"`
	SupplierID string `json:"supplier_id"`
	RetailerID string `json:"retailer_id"`
	DriverID   string `json:"driver_id"`
	VehicleID  string `json:"vehicle_id,omitempty"`
	RouteID    string `json:"route_id"`
	ManifestID string `json:"manifest_id,omitempty"`
	EventType  string `json:"event_type"`
	Version    int64  `json:"version"`
	UpdatedAt  string `json:"updated_at"`
	NoChange   bool   `json:"no_change,omitempty"`
}

type DeliverySubmitRequest struct {
	OrderID         string     `json:"order_id"`
	QRToken         string     `json:"qr_token"`
	ScannedToken    string     `json:"scanned_token"`
	Latitude        float64    `json:"latitude"`
	Longitude       float64    `json:"longitude"`
	ClientTimestamp *time.Time `json:"client_timestamp,omitempty"`
	BypassGeofence  bool       `json:"-"`
}

// DeliverySubmitResponse confirms QR/offline-token delivery submission.
type DeliverySubmitResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	NewState Status `json:"new_state,omitempty"`
}

// ConfirmOffloadRequest is the wire shape for POST /v1/order/confirm-offload.
type ConfirmOffloadRequest struct {
	OrderID         string     `json:"order_id"`
	ClientTimestamp *time.Time `json:"client_timestamp,omitempty"`
}

// ConfirmOffloadResponse matches the driver mobile offload-review contract.
type ConfirmOffloadResponse struct {
	OrderID       string `json:"order_id"`
	State         Status `json:"state"`
	PaymentMethod string `json:"payment_method"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	InvoiceID     string `json:"invoice_id,omitempty"`
	RetailerID    string `json:"retailer_id"`
	Message       string `json:"message"`
}

// ValidateQRRequest is the wire shape for POST /v1/order/validate-qr.
type ValidateQRRequest struct {
	OrderID      string `json:"order_id"`
	ScannedToken string `json:"scanned_token"`
}

// ValidateQRResponse matches the driver scanner contract without mutating order state.
type ValidateQRResponse struct {
	OrderID      string                `json:"order_id"`
	RetailerName string                `json:"retailer_name"`
	TotalAmount  int64                 `json:"total_amount"`
	State        Status                `json:"state"`
	Items        []DriverOrderLineItem `json:"items"`
}

// CompleteOrderRequest is the wire shape for POST /v1/order/complete.
type CompleteOrderRequest struct {
	OrderID         string     `json:"order_id"`
	Latitude        *float64   `json:"latitude,omitempty"`
	Longitude       *float64   `json:"longitude,omitempty"`
	ClientTimestamp *time.Time `json:"client_timestamp,omitempty"`
}

// CollectCashRequest is the wire shape for POST /v1/order/collect-cash.
// amount_received_minor is the cash actually taken (Tiyin). Fiscal uses this amount.
// When omitted, expected order total is used (compat); shortfall/overage still computed when provided.
type CollectCashRequest struct {
	OrderID              string     `json:"order_id"`
	Latitude             float64    `json:"latitude"`
	Longitude            float64    `json:"longitude"`
	AmountReceivedMinor  *int64     `json:"amount_received_minor,omitempty"`
	Note                 string     `json:"note,omitempty"`
	ClientTimestamp      *time.Time `json:"client_timestamp,omitempty"`
}

// CollectCashResponse matches the driver mobile cash-collection contract.
// ADR-009: State is FISCALIZING after capture (not COMPLETED until fiscal SUCCESS).
type CollectCashResponse struct {
	OrderID             string  `json:"order_id"`
	State               Status  `json:"state"`
	Amount              int64   `json:"amount"` // expected order total
	AmountReceivedMinor int64   `json:"amount_received_minor,omitempty"`
	ShortfallMinor      int64   `json:"shortfall_minor,omitempty"`
	OverageMinor        int64   `json:"overage_minor,omitempty"`
	Currency            string  `json:"currency"`
	DistanceM           float64 `json:"distance_m"`
	Message             string  `json:"message"`
	AttemptID           string  `json:"attempt_id,omitempty"`
	FiscalStatus        string  `json:"fiscal_status,omitempty"`
}

// DriverOrderLineItem is a driver-mobile compatible line-item snapshot.
type DriverOrderLineItem struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int64  `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	LineTotal   int64  `json:"line_total"`
}

// DriverOrderResponse is compatible with the driver native Order models.
type DriverOrderResponse struct {
	ID              string                `json:"id"`
	OrderID         string                `json:"order_id"`
	RetailerID      string                `json:"retailer_id"`
	RetailerName    string                `json:"retailer_name"`
	State           Status                `json:"state"`
	Status          string                `json:"status"`
	TotalAmount     int64                 `json:"total_amount"`
	Currency        string                `json:"currency"`
	DeliveryAddress string                `json:"delivery_address"`
	Latitude        float64               `json:"latitude,omitempty"`
	Longitude       float64               `json:"longitude,omitempty"`
	CreatedAt       string                `json:"created_at"`
	UpdatedAt       string                `json:"updated_at"`
	Items           []DriverOrderLineItem `json:"items"`
	Message         string                `json:"message,omitempty"`
}

type driverTransitionRequest struct {
	OrderID             string
	NextStatus          Status
	Reason              string
	ClientTimestamp     *time.Time
	Precheck            func(Order) error
	TransformNextStatus func(Order, Status) Status
	// PrepareOrder mutates the post-transition order before persist (fiscal rows, denorm).
	PrepareOrder func(*Order, Status) // previousStatus
	BuildProofs  func(Order) []DeliveryProofArtifact
	EmitExtra    func(outbox.TxnBuffer, Order, Status) error
}

type driverTransitionResult struct {
	Order          Order
	PreviousStatus Status
	Version        int64
	UpdatedAt      time.Time
	NoChange       bool
}

type orderStatusEmitParams struct {
	Claims         auth.Claims
	Order          Order
	PreviousStatus Status
	Reason         string
	ActorID        string
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
	if r.Lat == 0 && r.Lng == 0 {
		return errors.New("lat/lng required")
	}
	if strings.TrimSpace(r.RequestedDeliveryDate) != "" {
		if _, err := parseOptionalRFC3339(r.RequestedDeliveryDate); err != nil {
			return fmt.Errorf("requested_delivery_date must be RFC3339: %w", err)
		}
	}
	if strings.TrimSpace(r.DeliverBefore) != "" {
		if _, err := parseOptionalRFC3339(r.DeliverBefore); err != nil {
			return fmt.Errorf("deliver_before must be RFC3339: %w", err)
		}
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
	case StatusPending, StatusLoaded, StatusInTransit, StatusArrived, StatusAwaitingPayment, StatusPendingCashCollection, StatusCompleted, StatusCancelled:
		return next, nil
	default:
		return "", fmt.Errorf("unsupported status: %s", next)
	}
}

// Validate enforces assignment input invariants.
func (r AssignOrderRequest) Validate() (AssignOrderRequest, error) {
	normalized := AssignOrderRequest{
		DriverID:   strings.TrimSpace(r.DriverID),
		VehicleID:  strings.TrimSpace(r.VehicleID),
		RouteID:    strings.TrimSpace(r.RouteID),
		ManifestID: strings.TrimSpace(r.ManifestID),
	}
	if normalized.DriverID == "" {
		return AssignOrderRequest{}, errors.New("driver_id required")
	}
	if normalized.RouteID == "" {
		return AssignOrderRequest{}, errors.New("route_id required")
	}
	return normalized, nil
}

func (s *Service) normalizeAndQuoteLineItems(ctx context.Context, items []LineItem) ([]LineItem, int64, error) {
	if len(items) == 0 {
		return nil, 0, errors.New("line_items required")
	}
	if s.spannerClient == nil {
		var total int64
		normalized := make([]LineItem, 0, len(items))
		for i, li := range items {
			sku := strings.TrimSpace(li.SKU)
			if sku == "" {
				return nil, 0, fmt.Errorf("line_items[%d].sku required", i)
			}
			if li.Quantity <= 0 {
				return nil, 0, fmt.Errorf("line_items[%d].quantity must be > 0", i)
			}
			if li.UnitPrice < 0 {
				return nil, 0, fmt.Errorf("line_items[%d].unit_price_minor must be >= 0", i)
			}
			item := LineItem{
				SKU:               sku,
				Name:              strings.TrimSpace(li.Name),
				Quantity:          li.Quantity,
				UnitPrice:         li.UnitPrice,
				UnitVolumeVU:      li.UnitVolumeVU,
				HandlingClass:     li.HandlingClass,
				RequiresColdChain: li.RequiresColdChain,
				IsHazardous:       li.IsHazardous,
				IsPerishable:      li.IsPerishable,
				StorageTempMinC:   li.StorageTempMinC,
				StorageTempMaxC:   li.StorageTempMaxC,
			}
			total += item.UnitPrice * item.Quantity
			normalized = append(normalized, item)
		}
		return normalized, total, nil
	}
	var total int64
	normalized := make([]LineItem, 0, len(items))

	keys := make([]spanner.Key, 0, len(items))
	for _, li := range items {
		sku := strings.TrimSpace(li.SKU)
		if sku != "" {
			keys = append(keys, spanner.Key{sku})
		}
	}

	iter := s.spannerClient.Single().Read(ctx, "Products", spanner.KeySetFromKeys(keys...), []string{"ProductId", "Name", "PriceMinor", "UnitVolumeVU", "HandlingClass", "RequiresColdChain", "IsHazardous", "IsPerishable", "StorageTempMinC", "StorageTempMaxC"})
	defer iter.Stop()

	prices := make(map[string]int64)
	names := make(map[string]string)
	snapshots := make(map[string]productSnapshot)

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read products: %v", err)
		}
		var id, name string
		var price int64
		var snap productSnapshot
		var vu, storageTempMinC, storageTempMaxC spanner.NullFloat64
		var handlingClass spanner.NullString
		if err := row.Columns(&id, &name, &price, &vu, &handlingClass, &snap.requiresColdChain, &snap.isHazardous, &snap.isPerishable, &storageTempMinC, &storageTempMaxC); err != nil {
			return nil, 0, err
		}
		prices[id] = price
		names[id] = name
		if vu.Valid {
			snap.unitVolumeVU = vu.Float64
		} else {
			snap.unitVolumeVU = defaultUnitVolumeVU
		}
		snap.handlingClass = handlingClass.StringVal
		if storageTempMinC.Valid {
			v := storageTempMinC.Float64
			snap.storageTempMinC = &v
		}
		if storageTempMaxC.Valid {
			v := storageTempMaxC.Float64
			snap.storageTempMaxC = &v
		}
		snapshots[id] = snap
	}

	for i, li := range items {
		sku := strings.TrimSpace(li.SKU)
		if sku == "" {
			return nil, 0, fmt.Errorf("line_items[%d].sku required", i)
		}
		if li.Quantity <= 0 {
			return nil, 0, fmt.Errorf("line_items[%d].quantity must be > 0", i)
		}

		serverPrice, ok := prices[sku]
		if !ok {
			return nil, 0, fmt.Errorf("line_items[%d].sku %s not found in catalog", i, sku)
		}

		snap := snapshots[sku]
		item := LineItem{
			SKU:               sku,
			Name:              names[sku],
			Quantity:          li.Quantity,
			UnitPrice:         serverPrice,
			UnitVolumeVU:      snap.unitVolumeVU,
			HandlingClass:     snap.handlingClass,
			RequiresColdChain: snap.requiresColdChain,
			IsHazardous:       snap.isHazardous,
			IsPerishable:      snap.isPerishable,
			StorageTempMinC:   snap.storageTempMinC,
			StorageTempMaxC:   snap.storageTempMaxC,
		}

		total += item.UnitPrice * item.Quantity
		normalized = append(normalized, item)
	}
	return normalized, total, nil
}

func parseOptionalRFC3339(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, trimmed)
		if err != nil {
			return nil, err
		}
	}
	utc := parsed.UTC()
	return &utc, nil
}

func formatOptionalRFC3339(ts *time.Time) string {
	if ts == nil || ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339Nano)
}

func lifecycleResponse(orderRecord Order, version int64, created bool) RetailerOrderLifecycleResponse {
	return RetailerOrderLifecycleResponse{
		OrderID:                orderRecord.OrderID,
		Status:                 orderRecord.Status,
		Source:                 orderRecord.Source,
		ConfirmationStatus:     orderRecord.ConfirmationStatus,
		RequestedDeliveryDate:  formatOptionalRFC3339(orderRecord.RequestedDeliveryDate),
		DeliverBefore:          formatOptionalRFC3339(orderRecord.DeliverBefore),
		DeliveryPriority:       string(orderRecord.DeliveryPriority),
		DeliveryMode:           deliveryModeLabel(orderRecord),
		PreorderBadge:          preorderBadgeLabel(orderRecord),
		ProposedDeliveryDate:   formatOptionalRFC3339(orderRecord.ProposedDeliveryDate),
		DeliveryProposalReason: orderRecord.DeliveryProposalReason,
		AutoConfirmAt:          formatOptionalRFC3339(orderRecord.AutoConfirmAt),
		TotalMinor:             orderRecord.TotalMinor,
		Currency:               orderRecord.Currency,
		Version:                version,
		UpdatedAt:              orderRecord.UpdatedAt.Format(time.RFC3339Nano),
		Created:                created,
	}
}

func retailerPrediction(orderRecord Order) RetailerAIPrediction {
	return RetailerAIPrediction{
		OrderID:               orderRecord.OrderID,
		Source:                orderRecord.Source,
		ConfirmationStatus:    orderRecord.ConfirmationStatus,
		RequestedDeliveryDate: formatOptionalRFC3339(orderRecord.RequestedDeliveryDate),
		AutoConfirmAt:         formatOptionalRFC3339(orderRecord.AutoConfirmAt),
		TotalMinor:            orderRecord.TotalMinor,
		Currency:              orderRecord.Currency,
		DerivedFromOrderID:    orderRecord.DerivedFromOrderID,
		UpdatedAt:             orderRecord.UpdatedAt.Format(time.RFC3339Nano),
		Items:                 append([]LineItem(nil), orderRecord.LineItems...),
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

	lineItems, total, err := s.normalizeAndQuoteLineItems(ctx, req.LineItems)
	if err != nil {
		return CreateResponse{}, err
	}
	requestedDeliveryDate, err := parseOptionalRFC3339(req.RequestedDeliveryDate)
	if err != nil {
		return CreateResponse{}, fmt.Errorf("parse requested_delivery_date: %w", err)
	}
	deliverBefore, err := parseOptionalRFC3339(req.DeliverBefore)
	if err != nil {
		return CreateResponse{}, fmt.Errorf("parse deliver_before: %w", err)
	}
	now := s.now()
	priority := normalizeDeliveryPriority(req.DeliveryPriority)

	if s.warehouse == nil {
		return CreateResponse{}, fmt.Errorf("%w: warehouse_resolver_unavailable", ErrServiceabilityUnavailable)
	}

	resolvedWarehouseID, err := s.warehouse.ResolveNearestWarehouseID(ctx, s.supplierID, req.Lat, req.Lng)
	if err != nil {
		return CreateResponse{}, fmt.Errorf("%w: resolve nearest warehouse: %v", ErrServiceabilityUnavailable, err)
	}
	warehouseID := strings.TrimSpace(resolvedWarehouseID)
	if warehouseID == "" {
		return CreateResponse{}, fmt.Errorf("%w: no_eligible_warehouse", ErrZoneMiss)
	}

	var whPolicy WarehouseOpsPolicy
	var deliveryFeeMinor int64
	if s.spannerClient != nil {
		whPolicy, err = LoadWarehouseOpsPolicy(ctx, s.spannerClient, warehouseID)
		if err != nil {
			return CreateResponse{}, fmt.Errorf("load warehouse policy: %w", err)
		}
		if lineErrs := ValidateLineQuantities(lineItems, whPolicy); len(lineErrs) > 0 {
			return CreateResponse{}, fmt.Errorf("%w: %v", ErrLineQuantityOutOfRange, lineErrs)
		}
		if packErrs := validatePackMultiples(ctx, s.spannerClient, lineItems); len(packErrs) > 0 {
			return CreateResponse{}, fmt.Errorf("%w: %v", ErrLineQuantityOutOfRange, packErrs)
		}
		if open, _, _, closedMsg := checkOrderAcceptanceGate(whPolicy, now); !open {
			return CreateResponse{}, fmt.Errorf("%w: %s", ErrOrderAcceptanceClosed, closedMsg)
		}
		deliveryFeeMinor, _ = ComputeOrderDeliveryFee(whPolicy, req.Lat, req.Lng)
	}

	loc := proximity.TashkentLocation
	if whPolicy.OperatingSchedule.Timezone != "" {
		if l, err := time.LoadLocation(whPolicy.OperatingSchedule.Timezone); err == nil {
			loc = l
		}
	}

	source, status, confirmation, committedDate, latestDelivery, classifyErr := ClassifyDelivery(
		now, loc, req.DeliveryMode, requestedDeliveryDate, deliverBefore,
		whPolicy.PreorderMinLeadDays, whPolicy.PreorderMaxLeadDays,
	)
	if classifyErr != nil {
		return CreateResponse{}, classifyErr
	}

	var invPlan InventoryPlan
	if s.spannerClient != nil {
		policyOverride, _ := s.resolveCheckoutPolicyOverride(whPolicy, req.CheckoutPolicyToken)
		invPlan, err = PlanInventoryCheckout(ctx, s.spannerClient, s.supplierID, warehouseID, lineItems, policyOverride)
		if err != nil {
			return CreateResponse{}, err
		}
		lineItems = invPlan.Fulfillable
		total = 0
		for _, li := range lineItems {
			total += li.Quantity * li.UnitPrice
		}
		total += deliveryFeeMinor
	} else {
		invPlan.Fulfillable = lineItems
		total += deliveryFeeMinor
	}

	if s.credit != nil && total > 0 {
		check, err := s.credit.CheckOrder(ctx, retailerID, s.supplierID, total)
		if err != nil {
			return CreateResponse{}, fmt.Errorf("credit check failed: %w", err)
		}
		if !check.Allowed {
			if emitErr := s.emitCreditLimitBreached(ctx, retailerID, total, check); emitErr != nil {
				s.log.Warn("failed to emit credit limit breached event", "err", emitErr, "retailer_id", retailerID)
			}
			return CreateResponse{}, fmt.Errorf("%w: %s (shortfall %d)", ErrCreditLimitBreached, check.Reason, check.Shortfall)
		}
	}

	if len(lineItems) == 0 && len(invPlan.Backorder) == 0 {
		return CreateResponse{}, ErrInventoryExhausted
	}

	// All lines are backordered — single BACKORDERED order (no empty primary).
	if len(lineItems) == 0 && len(invPlan.Backorder) > 0 {
		boID, err := s.createBackorderOrder(ctx, retailerID, warehouseID, "", invPlan.Backorder, Order{
			H3Cell: req.H3Cell,
			Lat:    req.Lat,
			Lng:    req.Lng,
		})
		if err != nil {
			return CreateResponse{}, fmt.Errorf("persist backorder: %w", err)
		}
		var boTotal int64
		for _, li := range invPlan.Backorder {
			boTotal += li.Quantity * li.UnitPrice
		}
		return CreateResponse{
			OrderID:              boID,
			WarehouseID:          warehouseID,
			Status:               StatusBackordered,
			Source:               OrderSourceBackorder,
			ConfirmationStatus:   ConfirmationStatusConfirmed,
			TotalMinor:           boTotal,
			Currency:             s.currency,
			CreatedAt:            s.now().Format(time.RFC3339Nano),
			BackorderedItemCount: invPlan.BackorderCount,
			StockWarnings:        invPlan.Warnings,
		}, nil
	}

	o := Order{
		OrderID:               s.newID(),
		SupplierID:            s.supplierID,
		RetailerID:            retailerID,
		WarehouseID:           warehouseID,
		Status:                status,
		Source:                source,
		ConfirmationStatus:    confirmation,
		LineItems:             lineItems,
		TotalMinor:            total,
		Currency:              s.currency,
		H3Cell:                req.H3Cell,
		Lat:                   req.Lat,
		Lng:                   req.Lng,
		RequestedDeliveryDate: committedDate,
		DeliverBefore:         latestDelivery,
		DeliveryPriority:      priority,
		DeliveryFeeMinor:      deliveryFeeMinor,
		Version:               1,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	err = s.repo.CreateOrder(ctx, &o, func(txn outbox.TxnBuffer) error {
		if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, o.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:             events.BaseEvent{Type: events.EventOrderCreated, Timestamp: o.CreatedAt.Format(time.RFC3339Nano)},
			OrderID:               o.OrderID,
			SupplierID:            o.SupplierID,
			RetailerID:            o.RetailerID,
			WarehouseID:           o.WarehouseID,
			Status:                string(o.Status),
			OrderSource:           string(o.Source),
			ConfirmationStatus:    string(o.ConfirmationStatus),
			TotalMinor:            o.TotalMinor,
			Currency:              o.Currency,
			H3Cell:                o.H3Cell,
			Lat:                   o.Lat,
			Lng:                   o.Lng,
			RequestedDeliveryDate: formatOptionalRFC3339(o.RequestedDeliveryDate),
			ReceivingWindowOpen:   o.ReceivingWindowOpen,
			ReceivingWindowClose:  o.ReceivingWindowClose,
			LineItems:             o.LineItems,
		}); err != nil {
			return err
		}
		if o.Status == StatusScheduled {
			if err := emitPreorderEvent(ctx, txn, events.EventPreOrderNotified, o, string(auth.RoleRetailer), retailerID); err != nil {
				return err
			}
		}
		if ab, ok := txn.(outbox.AuditBuffer); ok {
			return outbox.WriteAudit(ctx, ab, o.SupplierID, retailerID, "RETAILER", "ORDER_CREATED", "Order", o.OrderID, map[string]any{
				"receiving_window_open":  o.ReceivingWindowOpen,
				"receiving_window_close": o.ReceivingWindowClose,
			})
		}
		return nil
	})
	if err != nil {
		return CreateResponse{}, fmt.Errorf("persist order: %w", err)
	}

	var backorderOrderID string
	if len(invPlan.Backorder) > 0 {
		backorderOrderID, err = s.createBackorderOrder(ctx, retailerID, warehouseID, o.OrderID, invPlan.Backorder, o)
		if err != nil {
			s.log.Warn("backorder create failed", "parent_order_id", o.OrderID, "err", err)
		}
	}

	// Post-commit cache invalidation: any retailer-orders or supplier-orders
	// list cache MUST be dropped so the next read sees the new row.
	if s.cache != nil {
		s.cache.Invalidate(ctx,
			retailerOrdersKey(o.RetailerID),
			supplierOrdersKey(o.SupplierID),
			"catalog:products:"+o.SupplierID,
		)
	}

	s.log.Info("order created",
		"order_id", o.OrderID,
		"supplier_id", o.SupplierID,
		"retailer_id", o.RetailerID,
		"warehouse_id", o.WarehouseID,
		"total_minor", o.TotalMinor,
		"currency", o.Currency,
		"h3_cell", o.H3Cell,
		"receiving_window_open", o.ReceivingWindowOpen,
		"receiving_window_close", o.ReceivingWindowClose,
	)
	return CreateResponse{
		OrderID:               o.OrderID,
		WarehouseID:           o.WarehouseID,
		Status:                o.Status,
		Source:                o.Source,
		ConfirmationStatus:    o.ConfirmationStatus,
		RequestedDeliveryDate: formatOptionalRFC3339(o.RequestedDeliveryDate),
		DeliverBefore:         formatOptionalRFC3339(o.DeliverBefore),
		DeliveryPriority:      string(o.DeliveryPriority),
		DeliveryMode:          normalizeDeliveryMode(req.DeliveryMode),
		TotalMinor:            o.TotalMinor,
		Currency:              o.Currency,
		CreatedAt:             o.CreatedAt.Format(time.RFC3339Nano),
		ReceivingWindowOpen:   o.ReceivingWindowOpen,
		ReceivingWindowClose:  o.ReceivingWindowClose,
		BackorderOrderID:      backorderOrderID,
		BackorderedItemCount:  invPlan.BackorderCount,
		StockWarnings:         invPlan.Warnings,
	}, nil
}

func (s *Service) createBackorderOrder(
	ctx context.Context,
	retailerID, warehouseID, parentOrderID string,
	lines []LineItem,
	parent Order,
) (string, error) {
	if len(lines) == 0 {
		return "", nil
	}
	var total int64
	for _, li := range lines {
		total += li.Quantity * li.UnitPrice
	}
	bo := Order{
		OrderID:            s.newID(),
		SupplierID:         s.supplierID,
		RetailerID:         retailerID,
		WarehouseID:        warehouseID,
		Status:             StatusBackordered,
		Source:             OrderSourceBackorder,
		ConfirmationStatus: ConfirmationStatusConfirmed,
		LineItems:          lines,
		TotalMinor:         total,
		Currency:           s.currency,
		H3Cell:             parent.H3Cell,
		Lat:                parent.Lat,
		Lng:                parent.Lng,
		DerivedFromOrderID: parentOrderID,
		Version:            1,
		CreatedAt:          s.now(),
		UpdatedAt:          s.now(),
	}
	err := s.repo.CreateOrder(ctx, &bo, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, bo.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:          events.BaseEvent{Type: events.EventOrderCreated, Timestamp: bo.CreatedAt.Format(time.RFC3339Nano)},
			OrderID:            bo.OrderID,
			SupplierID:         bo.SupplierID,
			RetailerID:         bo.RetailerID,
			WarehouseID:        bo.WarehouseID,
			Status:             string(bo.Status),
			OrderSource:        string(bo.Source),
			ConfirmationStatus: string(bo.ConfirmationStatus),
			TotalMinor:         bo.TotalMinor,
			Currency:           bo.Currency,
			LineItems:          bo.LineItems,
		})
	})
	if err != nil {
		return "", err
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, retailerOrdersKey(bo.RetailerID), supplierOrdersKey(bo.SupplierID))
	}
	return bo.OrderID, nil
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
		if PreorderCancelLocked(s.now(), current) {
			return UpdateStatusResponse{}, ErrOrderCancelLocked
		}
	}

	if err := ValidateStatusTransition(current.Status, nextStatus); err != nil {
		return UpdateStatusResponse{}, err
	}

	// ADR-009 hard-gate: no COMPLETED without reconstructible fiscal SUCCESS or audited FORCE.
	// Force-complete and fiscal worker set FiscalStatus in the same mutation as COMPLETED;
	// generic status patches (incl. reconciliation resolve) must not soft-complete.
	if nextStatus == StatusCompleted {
		fs := strings.TrimSpace(current.FiscalStatus)
		if fs != FiscalStatusSuccess && fs != FiscalStatusForceSkipped {
			return UpdateStatusResponse{}, fmt.Errorf(
				"%w: COMPLETED requires fiscal SUCCESS or FORCE_SKIPPED (use force-complete with reason_code); fiscal_status=%s",
				ErrInvalidStatusTransition, fs,
			)
		}
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
	previousDriverID := strings.TrimSpace(current.DriverID)
	current.Status = nextStatus
	current.UpdatedAt = s.now()
	s.applyHandoffLifecycle(&current, prevStatus, previousDriverID)

	actorID := claims.Subject
	if actorID == "" {
		actorID = claims.SupplierID
	}

	err = s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:      events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: current.UpdatedAt.Format(time.RFC3339Nano)},
			OrderID:        current.OrderID,
			SupplierID:     current.SupplierID,
			RetailerID:     current.RetailerID,
			DriverID:       current.DriverID,
			PreviousStatus: string(prevStatus),
			Status:         string(current.Status),
			Reason:         strings.TrimSpace(req.Reason),
			ActorRole:      string(claims.Role),
			ActorID:        actorID,
		}); err != nil {
			return err
		}
		if ab, ok := txn.(outbox.AuditBuffer); ok {
			return outbox.WriteAudit(ctx, ab, current.SupplierID, actorID, string(claims.Role), "ORDER_STATUS_CHANGED", "Order", current.OrderID, map[string]string{"from": string(prevStatus), "to": string(current.Status)})
		}
		return nil
	})
	if err != nil {
		return UpdateStatusResponse{}, fmt.Errorf("update order status %s: %w", orderID, err)
	}

	s.recordStatusTransitionFromOrder(current, prevStatus, strings.TrimSpace(req.Reason), string(claims.Role), actorID, "", nil)

	if s.cache != nil {
		s.cache.Invalidate(ctx,
			retailerOrdersKey(current.RetailerID),
			supplierOrdersKey(current.SupplierID),
		)
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

// AssignOrder durably binds an order to the driver/route authority used by
// driver execution and retailer tracking.
func (s *Service) AssignOrder(ctx context.Context, claims auth.Claims, orderID string, req AssignOrderRequest) (AssignOrderResponse, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return AssignOrderResponse{}, errors.New("order_id required")
	}
	if !canAssignOrders(claims.Role) {
		return AssignOrderResponse{}, ErrOrderForbidden
	}

	normalized, err := req.Validate()
	if err != nil {
		return AssignOrderResponse{}, err
	}

	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return AssignOrderResponse{}, fmt.Errorf("get order %s: %w", orderID, err)
	}
	if !ok {
		return AssignOrderResponse{}, ErrOrderNotFound
	}
	if strings.TrimSpace(claims.SupplierID) != "" && claims.SupplierID != current.SupplierID {
		return AssignOrderResponse{}, ErrOrderForbidden
	}

	previousDriverID := strings.TrimSpace(current.DriverID)
	previousRouteID := strings.TrimSpace(current.RouteID)
	if current.DriverID == normalized.DriverID && current.VehicleID == normalized.VehicleID && current.RouteID == normalized.RouteID && current.ManifestID == normalized.ManifestID {
		return assignmentResponse(current, assignmentEventType(previousDriverID), current.Version, true), nil
	}

	previousStatus := current.Status
	current.DriverID = normalized.DriverID
	current.VehicleID = normalized.VehicleID
	current.RouteID = normalized.RouteID
	current.ManifestID = normalized.ManifestID
	current.UpdatedAt = s.now()
	s.applyHandoffLifecycle(&current, previousStatus, previousDriverID)
	eventType := assignmentEventType(previousDriverID)

	err = s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		if eventType == events.EventOrderReassigned {
			return outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, orderReassignedData(current, previousDriverID, previousRouteID))
		}
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, orderAssignedData(current))
	})
	if err != nil {
		return AssignOrderResponse{}, fmt.Errorf("assign order %s: %w", orderID, err)
	}

	s.afterOrderMutation(ctx, current)
	s.log.Info("order assignment updated",
		"order_id", current.OrderID,
		"supplier_id", current.SupplierID,
		"retailer_id", current.RetailerID,
		"driver_id", current.DriverID,
		"route_id", current.RouteID,
		"event_type", eventType,
	)

	return assignmentResponse(current, eventType, current.Version+1, false), nil
}

// MarkArrived moves a driver's active order from IN_TRANSIT to ARRIVED.
func (s *Service) MarkArrived(ctx context.Context, claims auth.Claims, orderID string) (UpdateStatusResponse, error) {
	result, err := s.transitionDriverOrder(ctx, claims, driverTransitionRequest{
		OrderID:    orderID,
		NextStatus: StatusArrived,
		Reason:     "driver_arrived",
	})
	if err != nil {
		return UpdateStatusResponse{}, err
	}
	return UpdateStatusResponse{
		OrderID:        result.Order.OrderID,
		PreviousStatus: result.PreviousStatus,
		Status:         result.Order.Status,
		Version:        result.Version,
		UpdatedAt:      result.UpdatedAt.Format(time.RFC3339Nano),
		EventType:      events.EventOrderStatusChanged,
	}, nil
}

// SubmitDelivery records QR/offline-token handoff and opens payment settlement (ADR-009).
// Does not complete the order — money + fiscal hard-gate still required.
func (s *Service) SubmitDelivery(ctx context.Context, claims auth.Claims, req DeliverySubmitRequest) (DeliverySubmitResponse, error) {
	if strings.TrimSpace(req.OrderID) == "" {
		return DeliverySubmitResponse{}, errors.New("order_id required")
	}
	if strings.TrimSpace(req.token()) == "" {
		return DeliverySubmitResponse{}, errors.New("qr_token required")
	}

	var distanceM float64
	result, err := s.transitionDriverOrder(ctx, claims, driverTransitionRequest{
		OrderID:         req.OrderID,
		NextStatus:      StatusAwaitingPayment,
		Reason:          "driver_delivery_submit",
		ClientTimestamp: req.ClientTimestamp,
		TransformNextStatus: func(orderRecord Order, next Status) Status {
			if orderRecord.Status == StatusCancelled {
				return StatusReconciliationRequired
			}
			// Already past handoff — leave payment/fiscal states alone.
			switch orderRecord.Status {
			case StatusAwaitingPayment, StatusPendingCashCollection, StatusFiscalizing, StatusFiscalFailed, StatusCompleted, StatusDeliveredOnCredit:
				return orderRecord.Status
			default:
				return next
			}
		},
		Precheck: func(orderRecord Order) error {
			if token := req.token(); token != "" {
				if err := s.validateDeliveryToken(orderRecord, token); err != nil {
					if s.jwtSecret == "" || orderRecord.ManifestID == "" ||
						s.validateOfflineQR(orderRecord.ManifestID, claims.Subject, orderRecord.OrderID, token) != nil {
						return err
					}
				}
			}

			if !req.BypassGeofence && (req.Latitude != 0 || req.Longitude != 0) {
				computedDistance, err := validateOptionalGeofence(req.Latitude, req.Longitude, orderRecord)
				if err == nil {
					distanceM = computedDistance
				}
				return err
			}
			return nil
		},
		BuildProofs: func(orderRecord Order) []DeliveryProofArtifact {
			latitude, longitude := deliveryProofCoordinatesFromValues(req.Latitude, req.Longitude)
			return buildDeliveryProofArtifacts(
				s.newID(),
				orderRecord,
				claims.Subject,
				DeliveryProofTypeQRHandoff,
				strings.TrimSpace(req.QRToken),
				strings.TrimSpace(req.ScannedToken),
				latitude,
				longitude,
				deliveryProofDistance(distanceM, latitude, longitude),
			)
		},
		EmitExtra: func(txn outbox.TxnBuffer, orderRecord Order, previousStatus Status) error {
			if orderRecord.Status != StatusAwaitingPayment || previousStatus == StatusAwaitingPayment {
				return nil
			}
			if err := s.emitSettlementRequired(ctx, txn, orderRecord); err != nil {
				return err
			}
			return s.emitPaymentRequired(ctx, txn, orderRecord)
		},
	})
	if err != nil {
		return DeliverySubmitResponse{}, err
	}

	return DeliverySubmitResponse{
		Success:  true,
		Message:  "Handoff recorded. Awaiting payment and fiscal receipt.",
		NewState: result.Order.Status,
	}, nil
}

// ConfirmOffload records the driver handoff and opens retailer payment settlement.
func (s *Service) ConfirmOffload(ctx context.Context, claims auth.Claims, req ConfirmOffloadRequest) (ConfirmOffloadResponse, error) {
	result, err := s.transitionDriverOrder(ctx, claims, driverTransitionRequest{
		OrderID:         req.OrderID,
		NextStatus:      StatusAwaitingPayment,
		Reason:          "confirm_offload",
		ClientTimestamp: req.ClientTimestamp,
		EmitExtra: func(txn outbox.TxnBuffer, orderRecord Order, _ Status) error {
			if err := s.emitSettlementRequired(ctx, txn, orderRecord); err != nil {
				return err
			}
			return s.emitPaymentRequired(ctx, txn, orderRecord)
		},
	})
	if err != nil {
		return ConfirmOffloadResponse{}, err
	}

	paymentMethod := resolveOrderPaymentMethod(ctx, s.spannerClient, result.Order.OrderID)
	return ConfirmOffloadResponse{
		OrderID:       result.Order.OrderID,
		State:         result.Order.Status,
		PaymentMethod: paymentMethod,
		Amount:        result.Order.TotalMinor,
		Currency:      result.Order.Currency,
		RetailerID:    result.Order.RetailerID,
		Message:       fmt.Sprintf("Collect %d %s", result.Order.TotalMinor, result.Order.Currency),
	}, nil
}

// CompleteOrder captures non-cash settlement intent and enters fiscal hard-gate (ADR-009).
// Order becomes FISCALIZING; COMPLETED only after fiscal SUCCESS (or audited force-complete).
func (s *Service) CompleteOrder(ctx context.Context, claims auth.Claims, req CompleteOrderRequest) (DriverOrderResponse, error) {
	var distanceM float64
	var attemptID string
	result, err := s.transitionDriverOrder(ctx, claims, driverTransitionRequest{
		OrderID:         req.OrderID,
		NextStatus:      StatusFiscalizing,
		Reason:          "complete_order",
		ClientTimestamp: req.ClientTimestamp,
		Precheck: func(orderRecord Order) error {
			switch orderRecord.Status {
			case StatusAwaitingPayment, StatusDeliveredOnCredit, StatusFiscalizing:
			default:
				return fmt.Errorf("%w: complete requires AWAITING_PAYMENT or credit settlement (current %s)", ErrInvalidStatusTransition, orderRecord.Status)
			}
			if orderRecord.Status == StatusFiscalizing {
				return nil
			}
			computedDistance, err := validatePointerGeofence(req.Latitude, req.Longitude, orderRecord)
			if err == nil {
				distanceM = computedDistance
			}
			return err
		},
		PrepareOrder: func(o *Order, _ Status) {
			row := s.newFiscalPendingRow(*o, "CARD", s.newID(), o.TotalMinor)
			attemptID = row.AttemptID
			o.PendingFiscalReceipts = []FiscalReceiptRow{row}
			o.FiscalStatus = FiscalStatusPending
			o.LatestFiscalAttemptID = row.AttemptID
		},
		BuildProofs: func(orderRecord Order) []DeliveryProofArtifact {
			latitude, longitude := deliveryProofCoordinatesFromPointers(req.Latitude, req.Longitude)
			return buildDeliveryProofArtifacts(
				s.newID(),
				orderRecord,
				claims.Subject,
				DeliveryProofTypeFinalizationGeo,
				"",
				"",
				latitude,
				longitude,
				deliveryProofDistance(distanceM, latitude, longitude),
			)
		},
		EmitExtra: func(txn outbox.TxnBuffer, orderRecord Order, previousStatus Status) error {
			if previousStatus == StatusDeliveredOnCredit {
				if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, events.CreditDeliveryEvent{
					BaseEvent:  events.BaseEvent{Type: events.EventCreditDeliveryResolved, Timestamp: s.now().UTC().Format(time.RFC3339Nano)},
					OrderID:    orderRecord.OrderID,
					DriverID:   claims.Subject,
					SupplierID: orderRecord.SupplierID,
					RetailerID: orderRecord.RetailerID,
					Status:     string(StatusFiscalizing),
				}); err != nil {
					return err
				}
			}
			row := orderRecord.PendingFiscalReceipts[0]
			return emitPaymentCaptureFiscal(ctx, txn, orderRecord, row, "CARD")
		},
	})
	if err != nil {
		return DriverOrderResponse{}, err
	}
	if !result.NoChange {
		if s.credit != nil && result.PreviousStatus == StatusDeliveredOnCredit && result.Order.TotalMinor > 0 {
			if clearErr := s.credit.ClearBalance(ctx, result.Order.RetailerID, result.Order.SupplierID, result.Order.TotalMinor, result.Order.OrderID); clearErr != nil {
				s.log.Error("clear credit balance failed", "order_id", result.Order.OrderID, "err", clearErr)
			}
		}
		if s.paymentCapturer != nil && result.Order.TotalMinor > 0 {
			go func() {
				captureCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := s.paymentCapturer.CaptureCardPayment(captureCtx, result.Order.OrderID, result.Order.TotalMinor, result.Order.Currency); err != nil {
					s.log.Error("failed to capture card payment", "order_id", result.Order.OrderID, "error", err)
				}
			}()
		}
	}

	msg := "Payment captured. Waiting for fiscal receipt."
	if attemptID != "" {
		_ = attemptID
	}
	return driverOrderResponse(result.Order, msg), nil
}

// CollectCash geofence-confirms cash collection, captures payment, and starts fiscal hard-gate (ADR-009).
// Fiscal amount = amount_received_minor when provided (P0 T3); shortfall/overage emitted as audit events.
func (s *Service) CollectCash(ctx context.Context, claims auth.Claims, req CollectCashRequest) (CollectCashResponse, error) {
	var distanceM float64
	var attemptID string
	var receivedMinor, shortfallMinor, overageMinor int64
	result, err := s.transitionDriverOrder(ctx, claims, driverTransitionRequest{
		OrderID:         req.OrderID,
		NextStatus:      StatusFiscalizing,
		Reason:          "collect_cash",
		ClientTimestamp: req.ClientTimestamp,
		Precheck: func(orderRecord Order) error {
			switch orderRecord.Status {
			case StatusPendingCashCollection, StatusAwaitingPayment, StatusDeliveredOnCredit, StatusFiscalizing:
			default:
				return fmt.Errorf("%w: collect cash requires payment capture state (current %s)", ErrInvalidStatusTransition, orderRecord.Status)
			}
			if orderRecord.Status == StatusFiscalizing {
				return nil
			}
			if req.AmountReceivedMinor != nil {
				if *req.AmountReceivedMinor < 0 {
					return ErrCashAmountNegative
				}
				receivedMinor = *req.AmountReceivedMinor
			} else {
				// Compat: default to expected total when driver omits received amount.
				receivedMinor = orderRecord.TotalMinor
			}
			if orderRecord.TotalMinor > receivedMinor {
				shortfallMinor = orderRecord.TotalMinor - receivedMinor
			} else if receivedMinor > orderRecord.TotalMinor {
				overageMinor = receivedMinor - orderRecord.TotalMinor
			}
			computedDistance, err := validateRequiredGeofence(req.Latitude, req.Longitude, orderRecord)
			if err != nil {
				return err
			}
			distanceM = computedDistance
			return nil
		},
		PrepareOrder: func(o *Order, _ Status) {
			if receivedMinor == 0 && req.AmountReceivedMinor == nil {
				receivedMinor = o.TotalMinor
			}
			row := s.newFiscalPendingRow(*o, "CASH", s.newID(), receivedMinor)
			attemptID = row.AttemptID
			o.PendingFiscalReceipts = []FiscalReceiptRow{row}
			o.FiscalStatus = FiscalStatusPending
			o.LatestFiscalAttemptID = row.AttemptID
		},
		BuildProofs: func(orderRecord Order) []DeliveryProofArtifact {
			latitude, longitude := deliveryProofCoordinatesFromValues(req.Latitude, req.Longitude)
			return buildDeliveryProofArtifacts(
				s.newID(),
				orderRecord,
				claims.Subject,
				DeliveryProofTypeCashCollectionGeo,
				"",
				"",
				latitude,
				longitude,
				deliveryProofDistance(distanceM, latitude, longitude),
			)
		},
		EmitExtra: func(txn outbox.TxnBuffer, orderRecord Order, previousStatus Status) error {
			if previousStatus == StatusDeliveredOnCredit {
				if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, events.CreditDeliveryEvent{
					BaseEvent:  events.BaseEvent{Type: events.EventCreditDeliveryResolved, Timestamp: s.now().UTC().Format(time.RFC3339Nano)},
					OrderID:    orderRecord.OrderID,
					DriverID:   claims.Subject,
					SupplierID: orderRecord.SupplierID,
					RetailerID: orderRecord.RetailerID,
					Status:     string(StatusFiscalizing),
				}); err != nil {
					return err
				}
			}
			if err := emitCashVariance(ctx, txn, orderRecord, claims.Subject, orderRecord.TotalMinor, receivedMinor, req.Note); err != nil {
				return err
			}
			row := orderRecord.PendingFiscalReceipts[0]
			return emitPaymentCaptureFiscal(ctx, txn, orderRecord, row, "CASH")
		},
	})
	if err != nil {
		return CollectCashResponse{}, err
	}
	if result.NoChange {
		distanceM = 0
		return CollectCashResponse{
			OrderID:             result.Order.OrderID,
			State:               result.Order.Status,
			Amount:              result.Order.TotalMinor,
			AmountReceivedMinor: receivedMinor,
			Currency:            result.Order.Currency,
			DistanceM:           distanceM,
			Message:             "Cash collection already fiscalizing.",
			AttemptID:           result.Order.LatestFiscalAttemptID,
			FiscalStatus:        result.Order.FiscalStatus,
		}, nil
	}

	msg := "Cash collected. Waiting for fiscal receipt."
	if shortfallMinor > 0 {
		msg = fmt.Sprintf("Cash collected with shortfall %d. Fiscalizing received amount.", shortfallMinor)
	} else if overageMinor > 0 {
		msg = fmt.Sprintf("Cash collected with overage %d. Fiscalizing received amount.", overageMinor)
	}

	return CollectCashResponse{
		OrderID:             result.Order.OrderID,
		State:               result.Order.Status,
		Amount:              result.Order.TotalMinor,
		AmountReceivedMinor: receivedMinor,
		ShortfallMinor:      shortfallMinor,
		OverageMinor:        overageMinor,
		Currency:            result.Order.Currency,
		DistanceM:           distanceM,
		Message:             msg,
		AttemptID:           attemptID,
		FiscalStatus:        FiscalStatusPending,
	}, nil
}

func (s *Service) transitionDriverOrder(ctx context.Context, claims auth.Claims, req driverTransitionRequest) (driverTransitionResult, error) {
	current, err := s.loadDriverTransitionOrder(ctx, claims, req.OrderID)
	if err != nil {
		return driverTransitionResult{}, err
	}

	if req.TransformNextStatus != nil {
		req.NextStatus = req.TransformNextStatus(current, req.NextStatus)
	}

	if current.Status == req.NextStatus {
		return driverTransitionResult{
			Order:          current,
			PreviousStatus: current.Status,
			Version:        current.Version,
			UpdatedAt:      current.UpdatedAt,
			NoChange:       true,
		}, nil
	}
	if err := ValidateStatusTransition(current.Status, req.NextStatus); err != nil {
		return driverTransitionResult{}, err
	}
	if req.Precheck != nil {
		if err := req.Precheck(current); err != nil {
			return driverTransitionResult{}, err
		}
	}

	previousStatus := current.Status
	previousDriverID := strings.TrimSpace(current.DriverID)
	current.Status = req.NextStatus
	if req.ClientTimestamp != nil {
		current.UpdatedAt = *req.ClientTimestamp
	} else {
		current.UpdatedAt = s.now()
	}
	s.applyHandoffLifecycle(&current, previousStatus, previousDriverID)
	if req.PrepareOrder != nil {
		req.PrepareOrder(&current, previousStatus)
	}
	if err := s.persistDriverTransition(ctx, claims, req, current, previousStatus); err != nil {
		return driverTransitionResult{}, err
	}

	s.recordDriverTransitionSuccess(ctx, claims, req, current, previousStatus)

	return driverTransitionResult{
		Order:          current,
		PreviousStatus: previousStatus,
		Version:        current.Version + 1,
		UpdatedAt:      current.UpdatedAt,
	}, nil
}

func (s *Service) loadDriverTransitionOrder(ctx context.Context, claims auth.Claims, orderID string) (Order, error) {
	trimmedOrderID := strings.TrimSpace(orderID)
	if trimmedOrderID == "" {
		return Order{}, errors.New("order_id required")
	}
	if claims.Role != auth.RoleDriver {
		return Order{}, ErrOrderForbidden
	}

	current, ok, err := s.repo.GetOrder(ctx, trimmedOrderID)
	if err != nil {
		return Order{}, fmt.Errorf("get order %s: %w", trimmedOrderID, err)
	}
	if !ok {
		return Order{}, ErrOrderNotFound
	}
	if strings.TrimSpace(current.DriverID) == "" {
		return Order{}, ErrAssignmentRequired
	}
	if strings.TrimSpace(claims.Subject) == "" || strings.TrimSpace(current.DriverID) != strings.TrimSpace(claims.Subject) {
		return Order{}, ErrOrderForbidden
	}

	return current, nil
}

func canAssignOrders(role auth.Role) bool {
	switch role {
	case auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleFactoryAdmin:
		return true
	default:
		return false
	}
}

func (s *Service) persistDriverTransition(ctx context.Context, claims auth.Claims, req driverTransitionRequest, current Order, previousStatus Status) error {
	actorID := claims.Subject
	var proofs []DeliveryProofArtifact
	if req.BuildProofs != nil {
		proofs = req.BuildProofs(current)
	}
	err := s.repo.UpdateOrder(ctx, current, proofs, func(txn outbox.TxnBuffer) error {
		params := orderStatusEmitParams{
			Claims:         claims,
			Order:          current,
			PreviousStatus: previousStatus,
			Reason:         req.Reason,
			ActorID:        actorID,
		}
		if err := emitOrderStatusChanged(ctx, txn, params); err != nil {
			return err
		}
		if req.EmitExtra != nil {
			if err := req.EmitExtra(txn, current, previousStatus); err != nil {
				return err
			}
		}
		if ab, ok := txn.(outbox.AuditBuffer); ok {
			return outbox.WriteAudit(ctx, ab, current.SupplierID, actorID, string(claims.Role), "ORDER_STATUS_CHANGED", "Order", current.OrderID, map[string]string{"from": string(previousStatus), "to": string(current.Status)})
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("transition driver order %s: %w", current.OrderID, err)
	}

	return nil
}

func emitOrderStatusChanged(ctx context.Context, txn outbox.TxnBuffer, params orderStatusEmitParams) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, params.Order.OrderID, events.TopicMain, events.OrderEvent{
		BaseEvent:             events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: params.Order.UpdatedAt.Format(time.RFC3339Nano)},
		OrderID:               params.Order.OrderID,
		SupplierID:            params.Order.SupplierID,
		RetailerID:            params.Order.RetailerID,
		WarehouseID:           params.Order.WarehouseID,
		DriverID:              params.Order.DriverID,
		PreviousStatus:        string(params.PreviousStatus),
		Status:                string(params.Order.Status),
		Reason:                params.Reason,
		ActorRole:             string(params.Claims.Role),
		ActorID:               params.ActorID,
		OrderSource:           string(params.Order.Source),
		ConfirmationStatus:    string(params.Order.ConfirmationStatus),
		RequestedDeliveryDate: formatOptionalRFC3339(params.Order.RequestedDeliveryDate),
		Version:               params.Order.Version,
	})
}

func (s *Service) recordDriverTransitionSuccess(ctx context.Context, claims auth.Claims, req driverTransitionRequest, current Order, previousStatus Status) {
	s.afterOrderMutation(ctx, current)
	s.log.Info("driver order status updated",
		"order_id", current.OrderID,
		"supplier_id", current.SupplierID,
		"retailer_id", current.RetailerID,
		"prev_status", previousStatus,
		"status", current.Status,
		"actor_id", claims.Subject,
	)

	if (current.Status == StatusInTransit || current.Status == StatusCompleted) && s.driverHub != nil {
		siblings, err := s.repo.FindSiblingDriversForOrder(ctx, current.OrderID)
		if err == nil && len(siblings) > 1 {
			for _, sib := range siblings {
				if sib != claims.Subject {
					eventType := "OTHER_TRUCK_ON_WAY"
					message := "Another truck is on the way to this route."
					if current.Status == StatusCompleted {
						eventType = "PAYMENT_COMPLETED"
						message = "Payment has been collected by another driver."
					}
					
					payload := map[string]any{
						"type":     eventType,
						"order_id": current.OrderID,
						"message":  message,
					}
					b, _ := json.Marshal(payload)
					
					go s.driverHub.Broadcast(context.Background(), "driver:"+sib, b)
				}
			}
		}
	}
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

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
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

	var req CreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	// Subject IS the retailer id for RETAILER-role callers.
	resp, err := s.Create(r.Context(), claims.Subject, req)
	if err != nil {
		s.log.Warn("order create failed",
			"retailer_id", claims.Subject, "err", err)
		switch {
		case errors.Is(err, ErrZoneMiss):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": ErrZoneMiss.Error()})
		case errors.Is(err, ErrOrderAcceptanceClosed):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error(), "code": ErrOrderAcceptanceClosed.Error()})
		case errors.Is(err, ErrServiceabilityUnavailable):
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": ErrServiceabilityUnavailable.Error()})
		default:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusCreated, respBytes)
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

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
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

	var req UpdateStatusRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

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
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// ValidateQR checks a scanned delivery token against the order without advancing lifecycle.
func (s *Service) ValidateQR(ctx context.Context, claims auth.Claims, req ValidateQRRequest) (ValidateQRResponse, error) {
	orderID := strings.TrimSpace(req.OrderID)
	token := strings.TrimSpace(req.ScannedToken)
	if orderID == "" || token == "" {
		return ValidateQRResponse{}, errors.New("order_id and scanned_token required")
	}

	orderRecord, found, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return ValidateQRResponse{}, fmt.Errorf("load order %s: %w", orderID, err)
	}
	if !found {
		return ValidateQRResponse{}, ErrOrderNotFound
	}
	if claims.Role == auth.RoleDriver && strings.TrimSpace(orderRecord.DriverID) != strings.TrimSpace(claims.Subject) {
		return ValidateQRResponse{}, ErrOrderForbidden
	}
	if err := s.validateDeliveryToken(orderRecord, token); err != nil {
		return ValidateQRResponse{}, err
	}

	resp := driverOrderResponse(orderRecord, "")
	retailerName := resolveRetailerDisplayName(ctx, s.spannerClient, orderRecord.RetailerID)
	return ValidateQRResponse{
		OrderID:      orderRecord.OrderID,
		RetailerName: retailerName,
		TotalAmount:  orderRecord.TotalMinor,
		State:        orderRecord.Status,
		Items:        resp.Items,
	}, nil
}

// HandleValidateQR is POST /v1/order/validate-qr.
func (s *Service) HandleValidateQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req ValidateQRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

	resp, err := s.ValidateQR(r.Context(), claims, req)
	if err != nil {
		s.writeOrderMutationError(w, "validate qr failed", req.OrderID, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleGetQRPayload serves GET /v1/order/{orderID}/qr-payload.
func (s *Service) HandleGetQRPayload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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

	current, found, err := s.repo.GetOrder(r.Context(), orderID)
	if err != nil {
		s.log.ErrorContext(r.Context(), "get order for qr payload failed", "err", err, "order_id", orderID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	if claims.Role == auth.RoleRetailer && current.RetailerID != claims.Subject {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	qrToken := s.publicDeliveryToken(current)

	writeJSON(w, http.StatusOK, map[string]string{
		"order_id": orderID,
		"qr_token": qrToken,
	})
}

// HandleDeliveryScanQR serves POST /v1/delivery/scan-qr.
func (s *Service) HandleDeliveryScanQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		OrderID         string     `json:"order_id"`
		QRToken         string     `json:"qr_token"`
		Latitude        *float64   `json:"latitude"`
		Longitude       *float64   `json:"longitude"`
		ClientTimestamp *time.Time `json:"client_timestamp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

	if strings.TrimSpace(req.OrderID) == "" || strings.TrimSpace(req.QRToken) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id and qr_token required"})
		return
	}

	result, err := s.transitionDriverOrder(r.Context(), claims, driverTransitionRequest{
		OrderID:         req.OrderID,
		NextStatus:      StatusAwaitingPayment,
		Reason:          "qr_scanned",
		ClientTimestamp: req.ClientTimestamp,
		Precheck: func(orderRecord Order) error {
			return s.validateDeliveryToken(orderRecord, req.QRToken)
		},
		EmitExtra: func(txn outbox.TxnBuffer, orderRecord Order, _ Status) error {
			if err := s.emitSettlementRequired(r.Context(), txn, orderRecord); err != nil {
				return err
			}
			return s.emitPaymentRequired(r.Context(), txn, orderRecord)
		},
	})

	if err != nil {
		s.log.Warn("delivery scan qr failed", "order_id", req.OrderID, "err", err)
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"valid":    true,
		"order_id": result.Order.OrderID,
		"state":    result.Order.Status,
	})
}

// HandleAssignOrder is POST /v1/orders/{orderID}/assign.
func (s *Service) HandleAssignOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
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

	var req AssignOrderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.AssignOrder(r.Context(), claims, orderID, req)
	if err != nil {
		s.log.Warn("order assignment failed", "order_id", orderID, "err", err)
		switch {
		case errors.Is(err, ErrOrderNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		case errors.Is(err, ErrOrderForbidden):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		default:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		return
	}

	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleMarkArrived is POST /v1/delivery/arrive.
func (s *Service) HandleMarkArrived(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
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

	var req struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.MarkArrived(r.Context(), claims, req.OrderID)
	if err != nil {
		s.writeOrderMutationError(w, "driver mark arrived failed", req.OrderID, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleSubmitDelivery is POST /v1/order/deliver.
func (s *Service) HandleSubmitDelivery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
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

	var req DeliverySubmitRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.SubmitDelivery(r.Context(), claims, req)
	if err != nil {
		s.writeOrderMutationError(w, "driver delivery submit failed", req.OrderID, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleConfirmOffload is POST /v1/order/confirm-offload.
func (s *Service) HandleConfirmOffload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
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

	var req ConfirmOffloadRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.ConfirmOffload(r.Context(), claims, req)
	if err != nil {
		s.writeOrderMutationError(w, "driver confirm offload failed", req.OrderID, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleCompleteOrder is POST /v1/order/complete.
func (s *Service) HandleCompleteOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
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

	var req CompleteOrderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.CompleteOrder(r.Context(), claims, req)
	if err != nil {
		s.writeOrderMutationError(w, "driver complete order failed", req.OrderID, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleCollectCash is POST /v1/order/collect-cash.
func (s *Service) HandleCollectCash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
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

	var req CollectCashRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.CollectCash(r.Context(), claims, req)
	if err != nil {
		s.writeOrderMutationError(w, "driver collect cash failed", req.OrderID, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

func (s *Service) writeOrderMutationError(w http.ResponseWriter, operation string, orderID string, err error) {
	s.log.Warn(operation, "order_id", orderID, "err", err)
	switch {
	case errors.Is(err, ErrOrderNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
	case errors.Is(err, ErrOrderForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	case errors.Is(err, ErrInvalidStatusTransition):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_status_transition"})
	case errors.Is(err, ErrGeofenceViolation):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "geofence_violation"})
	case errors.Is(err, ErrAssignmentRequired):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "assignment_required"})
	case errors.Is(err, ErrServiceabilityUnavailable):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "delivery_perimeter_unavailable"})
	case errors.Is(err, ErrFiscalNotFailed):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "fiscal_not_failed"})
	case errors.Is(err, ErrForceCompleteForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "force_complete_forbidden"})
	case errors.Is(err, ErrForceReasonRequired):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "force_reason_required"})
	case errors.Is(err, ErrForceReasonInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "force_reason_invalid"})
	case errors.Is(err, ErrFiscalAlreadySucceeded):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "fiscal_already_succeeded"})
	case errors.Is(err, ErrCashAmountNegative):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cash_amount_negative"})
	default:
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
	}
}

// HandleFiscalRetry is POST /v1/order/{orderID}/fiscal/retry
func (s *Service) HandleFiscalRetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := strings.TrimSpace(chiURLParam(r, "orderID"))
	if orderID == "" {
		// fallback for body-based clients
		var body struct {
			OrderID string `json:"order_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		orderID = strings.TrimSpace(body.OrderID)
	}
	resp, err := s.RetryFiscal(r.Context(), claims, orderID)
	if err != nil {
		s.writeOrderMutationError(w, "fiscal retry failed", orderID, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleForceComplete is POST /v1/order/{orderID}/force-complete
func (s *Service) HandleForceComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := strings.TrimSpace(chiURLParam(r, "orderID"))
	var body struct {
		OrderID    string `json:"order_id"`
		ReasonCode string `json:"reason_code"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if orderID == "" {
		orderID = strings.TrimSpace(body.OrderID)
	}
	resp, err := s.ForceCompleteOrder(r.Context(), claims, orderID, body.ReasonCode)
	if err != nil {
		s.writeOrderMutationError(w, "force complete failed", orderID, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func chiURLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

// ── wire shapes ────────────────────────────────────────────────────────────

func orderAssignedData(orderRecord Order) map[string]any {
	data := map[string]any{
		"type":         events.EventOrderAssigned,
		"order_id":     orderRecord.OrderID,
		"supplier_id":  orderRecord.SupplierID,
		"retailer_id":  orderRecord.RetailerID,
		"warehouse_id": orderRecord.WarehouseID,
		"driver_id":    orderRecord.DriverID,
		"route_id":     orderRecord.RouteID,
		"timestamp":    orderRecord.UpdatedAt.Format(time.RFC3339Nano),
	}
	addOptionalAssignmentFields(data, orderRecord)
	return data
}

func orderReassignedData(orderRecord Order, previousDriverID string, previousRouteID string) map[string]any {
	data := map[string]any{
		"type":           events.EventOrderReassigned,
		"order_id":       orderRecord.OrderID,
		"supplier_id":    orderRecord.SupplierID,
		"retailer_id":    orderRecord.RetailerID,
		"warehouse_id":   orderRecord.WarehouseID,
		"from_driver_id": previousDriverID,
		"to_driver_id":   orderRecord.DriverID,
		"from_route_id":  previousRouteID,
		"to_route_id":    orderRecord.RouteID,
		"timestamp":      orderRecord.UpdatedAt.Format(time.RFC3339Nano),
	}
	addOptionalAssignmentFields(data, orderRecord)
	return data
}

func addOptionalAssignmentFields(data map[string]any, orderRecord Order) {
	if strings.TrimSpace(orderRecord.VehicleID) != "" {
		data["vehicle_id"] = orderRecord.VehicleID
	}
	if strings.TrimSpace(orderRecord.ManifestID) != "" {
		data["manifest_id"] = orderRecord.ManifestID
	}
}

func assignmentEventType(previousDriverID string) string {
	if strings.TrimSpace(previousDriverID) != "" {
		return events.EventOrderReassigned
	}
	return events.EventOrderAssigned
}

func assignmentResponse(orderRecord Order, eventType string, version int64, noChange bool) AssignOrderResponse {
	return AssignOrderResponse{
		OrderID:    orderRecord.OrderID,
		SupplierID: orderRecord.SupplierID,
		RetailerID: orderRecord.RetailerID,
		DriverID:   orderRecord.DriverID,
		VehicleID:  orderRecord.VehicleID,
		RouteID:    orderRecord.RouteID,
		ManifestID: orderRecord.ManifestID,
		EventType:  eventType,
		Version:    version,
		UpdatedAt:  orderRecord.UpdatedAt.Format(time.RFC3339Nano),
		NoChange:   noChange,
	}
}

func (s *Service) afterOrderMutation(ctx context.Context, orderRecord Order) {
	if s.cache != nil {
		keys := []string{
			retailerOrdersKey(orderRecord.RetailerID),
			supplierOrdersKey(orderRecord.SupplierID),
		}
		if orderRecord.Status == StatusCancelled {
			keys = append(keys, "catalog:products:"+orderRecord.SupplierID)
		}
		s.cache.Invalidate(ctx, keys...)
	}
	if s.dispatchPlanWarm != nil {
		if wh := strings.TrimSpace(orderRecord.WarehouseID); wh != "" {
			s.dispatchPlanWarm(ctx, wh)
		}
	}
}

// SetDispatchPlanWarm schedules background smart-dispatch preview recompute for a warehouse.
func (s *Service) SetDispatchPlanWarm(fn func(ctx context.Context, warehouseID string)) {
	s.dispatchPlanWarm = fn
}

func (s *Service) emitSettlementRequired(ctx context.Context, txn outbox.TxnBuffer, orderRecord Order) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, s.settlementRequiredData(ctx, orderRecord))
}

func (s *Service) emitPaymentRequired(ctx context.Context, txn outbox.TxnBuffer, orderRecord Order) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, s.paymentRequiredData(ctx, orderRecord))
}

func emitPaymentCleared(ctx context.Context, txn outbox.TxnBuffer, orderRecord Order, method string) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, paymentClearedData(orderRecord, method))
}

func emitOrderFinalized(ctx context.Context, txn outbox.TxnBuffer, orderRecord Order) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, orderFinalizedData(orderRecord))
}

func (s *Service) paymentRequiredData(ctx context.Context, orderRecord Order) map[string]any {
	data := map[string]any{
		"type":           events.EventPaymentRequired,
		"order_id":       orderRecord.OrderID,
		"supplier_id":    orderRecord.SupplierID,
		"retailer_id":    orderRecord.RetailerID,
		"amount":         moneyData(orderRecord),
		"amount_minor":   orderRecord.TotalMinor,
		"currency":       orderRecord.Currency,
		"payment_method": "CASH",
		"gateway":        "CASH",
		"status":         string(orderRecord.Status),
		"timestamp":      orderRecord.UpdatedAt.Format(time.RFC3339Nano),
	}
	if driverID := strings.TrimSpace(orderRecord.DriverID); driverID != "" {
		data["driver_id"] = driverID
	}
	originalAmount := orderOriginalAmountMinor(orderRecord)
	if originalAmount > orderRecord.TotalMinor {
		data["original_amount"] = originalAmount
	}
	if s != nil && s.gatewayPolicy != nil {
		gateways, acceptor, err := s.gatewayPolicy.AllowedGateways(ctx, orderRecord.SupplierID, orderRecord.WarehouseID)
		if err == nil {
			data["payment_acceptor"] = acceptor
			data["available_card_gateways"] = cardGatewaysOnly(gateways)
		}
	}
	return data
}

func cardGatewaysOnly(gateways []string) []string {
	out := make([]string, 0, len(gateways))
	for _, gateway := range gateways {
		if strings.EqualFold(strings.TrimSpace(gateway), "CASH") {
			continue
		}
		out = append(out, gateway)
	}
	if len(out) == 0 {
		return []string{"GLOBAL_PAY"}
	}
	return out
}

func (s *Service) settlementRequiredData(ctx context.Context, orderRecord Order) map[string]any {
	data := s.paymentRequiredData(ctx, orderRecord)
	data["type"] = events.EventSettlementRequired
	return data
}

func paymentClearedData(orderRecord Order, method string) map[string]any {
	data := map[string]any{
		"type":           events.EventPaymentCleared,
		"order_id":       orderRecord.OrderID,
		"supplier_id":    orderRecord.SupplierID,
		"retailer_id":    orderRecord.RetailerID,
		"amount":         moneyData(orderRecord),
		"amount_minor":   orderRecord.TotalMinor,
		"currency":       orderRecord.Currency,
		"payment_method": method,
		"gateway":        method,
		"status":         string(orderRecord.Status),
		"timestamp":      orderRecord.UpdatedAt.Format(time.RFC3339Nano),
	}
	if driverID := strings.TrimSpace(orderRecord.DriverID); driverID != "" {
		data["driver_id"] = driverID
	}
	return data
}

func orderFinalizedData(orderRecord Order) map[string]any {
	data := map[string]any{
		"type":         events.EventOrderFinalized,
		"order_id":     orderRecord.OrderID,
		"supplier_id":  orderRecord.SupplierID,
		"retailer_id":  orderRecord.RetailerID,
		"total":        moneyData(orderRecord),
		"amount_minor": orderRecord.TotalMinor,
		"currency":     orderRecord.Currency,
		"status":       string(orderRecord.Status),
		"timestamp":    orderRecord.UpdatedAt.Format(time.RFC3339Nano),
	}
	if driverID := strings.TrimSpace(orderRecord.DriverID); driverID != "" {
		data["driver_id"] = driverID
	}
	return data
}

func moneyData(orderRecord Order) map[string]any {
	return map[string]any{
		"amount":   orderRecord.TotalMinor,
		"currency": orderRecord.Currency,
	}
}

func (s *Service) emitCreditLimitBreached(ctx context.Context, retailerID string, requestedAmount int64, check credit.CheckResult) error {
	if s.spannerClient == nil {
		return nil
	}
	return spannerutils.RunReadWriteTransaction(ctx, s.spannerClient, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateCreditProfile, retailerID, events.TopicMain, events.CreditLimitEvent{
			BaseEvent:        events.BaseEvent{Type: events.EventRetailerCreditLimitBreached, Timestamp: s.now().Format(time.RFC3339Nano)},
			OrderID:          "", // unknown at this pre-order stage; consumers key on retailer_id
			RetailerID:       retailerID,
			SupplierID:       s.supplierID,
			RequestedAmount:  requestedAmount,
			CreditLimitMinor: check.CreditLimitMinor,
			CurrentBalance:   check.CurrentBalance,
		}); err != nil {
			return err
		}
		var mutations []*spanner.Mutation
		for _, e := range buf.events {
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
				"EventId":       e.EventID,
				"AggregateType": e.AggregateType,
				"AggregateId":   e.AggregateID,
				"TopicName":     e.TopicName,
				"Payload":       e.Payload,
				"CreatedAt":     e.CreatedAt,
			}))
		}
		return txn.BufferWrite(mutations)
	})
}

func driverOrderResponse(orderRecord Order, message string) DriverOrderResponse {
	items := make([]DriverOrderLineItem, 0, len(orderRecord.LineItems))
	for _, lineItem := range orderRecord.LineItems {
		items = append(items, DriverOrderLineItem{
			ProductID:   lineItem.SKU,
			ProductName: lineItem.Name,
			Quantity:    lineItem.Quantity,
			UnitPrice:   lineItem.UnitPrice,
			LineTotal:   lineItem.UnitPrice * lineItem.Quantity,
		})
	}

	return DriverOrderResponse{
		ID:              orderRecord.OrderID,
		OrderID:         orderRecord.OrderID,
		RetailerID:      orderRecord.RetailerID,
		RetailerName:    orderRecord.RetailerID,
		State:           orderRecord.Status,
		Status:          string(orderRecord.Status),
		TotalAmount:     orderRecord.TotalMinor,
		Currency:        orderRecord.Currency,
		DeliveryAddress: "",
		Latitude:        orderRecord.Lat,
		Longitude:       orderRecord.Lng,
		CreatedAt:       orderRecord.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:       orderRecord.UpdatedAt.Format(time.RFC3339Nano),
		Items:           items,
		Message:         message,
	}
}

func (r DeliverySubmitRequest) token() string {
	if strings.TrimSpace(r.QRToken) != "" {
		return r.QRToken
	}
	return r.ScannedToken
}

func (s *Service) validateOfflineQR(manifestID, driverID, orderID, token string) error {
	h := sha256.New()
	h.Write([]byte(manifestID))
	h.Write([]byte(driverID))
	h.Write([]byte(s.jwtSecret))
	offlineNonce := hex.EncodeToString(h.Sum(nil))

	expected := sha256.New()
	expected.Write([]byte(offlineNonce))
	expected.Write([]byte(orderID))
	expectedHash := hex.EncodeToString(expected.Sum(nil))

	if token != expectedHash && token != offlineNonce {
		return errors.New("invalid offline qr token")
	}
	return nil
}

func buildDeliveryProofArtifacts(proofID string, orderRecord Order, driverID string, proofType DeliveryProofType, qrToken string, scannedToken string, latitude *float64, longitude *float64, distanceM *float64) []DeliveryProofArtifact {
	resolvedDriverID := strings.TrimSpace(driverID)
	if resolvedDriverID == "" {
		resolvedDriverID = strings.TrimSpace(orderRecord.DriverID)
	}
	qrTokenHash := hashDeliveryProofToken(qrToken)
	scannedTokenHash := hashDeliveryProofToken(scannedToken)
	if strings.TrimSpace(proofID) == "" || strings.TrimSpace(orderRecord.OrderID) == "" || resolvedDriverID == "" {
		return nil
	}
	if qrTokenHash == "" && scannedTokenHash == "" && (latitude == nil || longitude == nil) {
		return nil
	}
	capturedAt := orderRecord.UpdatedAt.UTC()
	if capturedAt.IsZero() {
		capturedAt = time.Now().UTC()
	}
	return []DeliveryProofArtifact{{
		ProofID:          strings.TrimSpace(proofID),
		OrderID:          orderRecord.OrderID,
		SupplierID:       orderRecord.SupplierID,
		RetailerID:       orderRecord.RetailerID,
		DriverID:         resolvedDriverID,
		ProofType:        proofType,
		QRTokenHash:      qrTokenHash,
		ScannedTokenHash: scannedTokenHash,
		Latitude:         latitude,
		Longitude:        longitude,
		DistanceM:        distanceM,
		CapturedAt:       capturedAt,
	}}
}

func hashDeliveryProofToken(token string) string {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])
}

func deliveryProofCoordinatesFromValues(latitude, longitude float64) (*float64, *float64) {
	if latitude == 0 && longitude == 0 {
		return nil, nil
	}
	lat := latitude
	lng := longitude
	return &lat, &lng
}

func deliveryProofCoordinatesFromPointers(latitude, longitude *float64) (*float64, *float64) {
	if latitude == nil || longitude == nil {
		return nil, nil
	}
	lat := *latitude
	lng := *longitude
	return &lat, &lng
}

func deliveryProofDistance(distanceM float64, latitude, longitude *float64) *float64 {
	if latitude == nil || longitude == nil {
		return nil
	}
	value := distanceM
	return &value
}

func validatePointerGeofence(latitude, longitude *float64, orderRecord Order) (float64, error) {
	if latitude == nil || longitude == nil {
		return 0, errors.New("latitude and longitude required")
	}
	return validateRequiredGeofence(*latitude, *longitude, orderRecord)
}

func validateOptionalGeofence(latitude, longitude float64, orderRecord Order) (float64, error) {
	return validateRequiredGeofence(latitude, longitude, orderRecord)
}

func validateRequiredGeofence(latitude, longitude float64, orderRecord Order) (float64, error) {
	if latitude == 0 && longitude == 0 {
		return 0, errors.New("latitude and longitude required")
	}
	if orderRecord.Lat == 0 && orderRecord.Lng == 0 {
		return 0, fmt.Errorf("%w: order coordinates unavailable", ErrServiceabilityUnavailable)
	}
	distanceM := distanceMeters(latitude, longitude, orderRecord.Lat, orderRecord.Lng)
	if distanceM > deliveryGeofenceMeters {
		return distanceM, fmt.Errorf("%w: %.0fm from delivery point", ErrGeofenceViolation, distanceM)
	}
	return distanceM, nil
}

func distanceMeters(latA, lngA, latB, lngB float64) float64 {
	const earthRadiusMeters = 6371000.0
	latARadians := latA * math.Pi / 180
	latBRadians := latB * math.Pi / 180
	deltaLat := (latB - latA) * math.Pi / 180
	deltaLng := (lngB - lngA) * math.Pi / 180

	angle := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(latARadians)*math.Cos(latBRadians)*math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	circle := 2 * math.Atan2(math.Sqrt(angle), math.Sqrt(1-angle))
	return earthRadiusMeters * circle
}

// AmendItemRequest is one line adjustment from the driver offload review surface.
type AmendItemRequest struct {
	ProductID    string `json:"product_id"`
	AcceptedQty  int64  `json:"accepted_qty"`
	RejectedQty  int64  `json:"rejected_qty"`
	Reason       string `json:"reason"`
	CustomReason string `json:"custom_reason"`
}

// AmendOrderRequest is POST /v1/order/amend.
type AmendOrderRequest struct {
	OrderID     string             `json:"order_id"`
	Items       []AmendItemRequest `json:"items"`
	DriverNotes string             `json:"driver_notes"`
}

// AmendOrderResponse matches native driver amend contracts.
type AmendOrderResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	AdjustedTotal int64  `json:"adjusted_total"`
}

// AmendOrder applies driver-reported rejections and recomputes order totals.
func (s *Service) AmendOrder(ctx context.Context, claims auth.Claims, req AmendOrderRequest) (AmendOrderResponse, error) {
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return AmendOrderResponse{}, errors.New("order_id required")
	}
	if len(req.Items) == 0 {
		return AmendOrderResponse{}, errors.New("items required")
	}

	current, err := s.loadDriverTransitionOrder(ctx, claims, orderID)
	if err != nil {
		return AmendOrderResponse{}, err
	}

	return s.applyOrderAmendments(ctx, current, req, claims.Subject)
}

func (s *Service) applyOrderAmendments(ctx context.Context, current Order, req AmendOrderRequest, actorID string) (AmendOrderResponse, error) {
	orderID := strings.TrimSpace(current.OrderID)
	if !orderAmendable(current.Status) {
		return AmendOrderResponse{}, fmt.Errorf("order %s cannot be amended from state %s", orderID, current.Status)
	}

	amendByProduct := make(map[string]AmendItemRequest, len(req.Items))
	for _, item := range req.Items {
		key := strings.TrimSpace(item.ProductID)
		if key == "" {
			continue
		}
		amendByProduct[key] = item
	}
	if len(amendByProduct) == 0 {
		return AmendOrderResponse{}, errors.New("items required")
	}

	origQtyBySKU := make(map[string]int64, len(current.LineItems))
	for _, line := range current.LineItems {
		origQtyBySKU[strings.TrimSpace(line.SKU)] = line.Quantity
	}

	updatedItems := make([]LineItem, 0, len(current.LineItems))
	pendingReturns := make([]SupplierReturn, 0, len(req.Items))
	var adjustedTotal int64
	for _, line := range current.LineItems {
		key := strings.TrimSpace(line.SKU)
		origQty := origQtyBySKU[key]
		if amend, ok := amendByProduct[key]; ok {
			acceptedQty, rejectedQty, err := resolveAmendQuantities(origQty, amend.AcceptedQty, amend.RejectedQty)
			if err != nil {
				return AmendOrderResponse{}, fmt.Errorf("%s for sku %s", err.Error(), key)
			}
			if rejectedQty > 0 {
				reason := normalizeAmendReason(amend.Reason)
				if reason == "" {
					return AmendOrderResponse{}, fmt.Errorf("reason required for rejected quantity on sku %s", key)
				}
				if err := validateAmendReason(reason, amend.CustomReason); err != nil {
					return AmendOrderResponse{}, fmt.Errorf("sku %s: %w", key, err)
				}
				pendingReturns = append(pendingReturns, SupplierReturn{
					ReturnID:    s.newID(),
					SKU:         key,
					RejectedQty: rejectedQty,
					Reason:      reason,
					DriverNotes: supplierReturnNotes(reason, amend.CustomReason, req.DriverNotes),
					ManifestID:  strings.TrimSpace(current.ManifestID),
					DriverID:    strings.TrimSpace(current.DriverID),
					WarehouseID: strings.TrimSpace(current.WarehouseID),
				})
			}
			line.Quantity = acceptedQty
		}
		adjustedTotal += line.UnitPrice * line.Quantity
		updatedItems = append(updatedItems, line)
	}

	previousTotal := current.TotalMinor
	if current.OriginalTotalMinor == 0 {
		current.OriginalTotalMinor = previousTotal
	}
	current.LineItems = updatedItems
	current.TotalMinor = adjustedTotal
	current.PendingSupplierReturns = pendingReturns
	current.UpdatedAt = s.now()

	if err := s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		payload := map[string]any{
			"type":            events.EventOrderAmended,
			"order_id":        current.OrderID,
			"driver_id":       strings.TrimSpace(actorID),
			"previous_total":  previousTotal,
			"adjusted_total":  adjustedTotal,
			"original_amount": current.OriginalTotalMinor,
			"currency":        current.Currency,
			"driver_notes":    strings.TrimSpace(req.DriverNotes),
			"amended_items":   req.Items,
			"timestamp":       current.UpdatedAt.Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, payload); err != nil {
			return err
		}
		if len(pendingReturns) > 0 {
			for _, pr := range pendingReturns {
				returnPayload, err := json.Marshal(map[string]any{
					"type":            events.EventSupplierReturnCreated,
					"return_id":       pr.ReturnID,
					"order_id":        current.OrderID,
					"sku_id":          pr.SKU,
					"quantity":        pr.RejectedQty,
					"reason":          pr.Reason,
					"manifest_id":     strings.TrimSpace(current.ManifestID),
					"driver_id":       strings.TrimSpace(current.DriverID),
					"warehouse_id":    strings.TrimSpace(current.WarehouseID),
					"supplier_id":     strings.TrimSpace(current.SupplierID),
					"physical_status": "PENDING",
					"timestamp":       current.UpdatedAt.Format(time.RFC3339Nano),
				})
				if err != nil {
					return err
				}
				if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, returnPayload); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return AmendOrderResponse{}, fmt.Errorf("amend order %s: %w", orderID, err)
	}

	current.PendingSupplierReturns = nil
	s.afterOrderMutation(ctx, current)

	return AmendOrderResponse{
		Success:       true,
		Message:       "amended",
		AdjustedTotal: adjustedTotal,
	}, nil
}

// HandleAmendOrder is POST /v1/order/amend.
func (s *Service) HandleAmendOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req AmendOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

	resp, err := s.AmendOrder(r.Context(), claims, req)
	if err != nil {
		s.writeOrderMutationError(w, "amend order failed", req.OrderID, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleReportDamage serves POST /v1/delivery/report-damage.
func (s *Service) HandleReportDamage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
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

	var req struct {
		OrderID      string `json:"order_id"`
		DamagedItems []struct {
			SKU      string `json:"sku"`
			Quantity int64  `json:"quantity"`
		} `json:"damaged_items"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}
	if len(req.DamagedItems) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "damaged_items_required"})
		return
	}

	current, found, err := s.repo.GetOrder(r.Context(), orderID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	if current.DriverID != claims.Subject {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	if current.Status != StatusInTransit && current.Status != StatusArrived {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_status_for_damage_report"})
		return
	}

	origQtyBySKU := make(map[string]int64, len(current.LineItems))
	for _, line := range current.LineItems {
		origQtyBySKU[strings.TrimSpace(line.SKU)] = line.Quantity
	}
	amendItems := make([]AmendItemRequest, 0, len(req.DamagedItems))
	conditionReports := make([]ConditionReport, 0, len(req.DamagedItems))
	for _, item := range req.DamagedItems {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_damaged_item"})
			return
		}
		origQty, ok := origQtyBySKU[sku]
		if !ok || item.Quantity > origQty {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("damage quantity exceeds item quantity for sku %s", sku)})
			return
		}
		amendItems = append(amendItems, AmendItemRequest{
			ProductID:   sku,
			AcceptedQty: origQty - item.Quantity,
			RejectedQty: item.Quantity,
			Reason:      "DAMAGED",
		})
		conditionReports = append(conditionReports, ConditionReport{
			ReportID:         s.newID(),
			OrderID:          current.OrderID,
			SupplierID:       current.SupplierID,
			RetailerID:       current.RetailerID,
			SKU:              sku,
			ConditionType:    ConditionTypeDamaged,
			Severity:         SeverityHigh,
			Description:      strings.TrimSpace(req.Reason),
			ReportedBy:       claims.Subject,
			ReportedByRole:   string(claims.Role),
			ResolutionStatus: ResolutionStatusOpen,
			CreatedAt:        s.now(),
		})
	}
	current.ConditionReports = conditionReports

	amendResp, err := s.applyOrderAmendments(r.Context(), current, AmendOrderRequest{
		OrderID:     orderID,
		Items:       amendItems,
		DriverNotes: req.Reason,
	}, claims.Subject)
	if err != nil {
		s.writeOrderMutationError(w, "report damage amend failed", orderID, err)
		return
	}

	idemCommitted = true
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"success":        true,
		"message":        "damage_reported",
		"adjusted_total": amendResp.AdjustedTotal,
	})
}

// HandleRetailerConfirmCash serves POST /v1/delivery/confirm-cash for Retailers.
func (s *Service) HandleRetailerConfirmCash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
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

	var req struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}

	current, found, err := s.repo.GetOrder(r.Context(), orderID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	if current.RetailerID != claims.Subject {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	if current.Status != StatusAwaitingPayment {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_status_for_cash_selection"})
		return
	}

	previousStatus := current.Status
	current.Status = StatusPendingCashCollection
	current.UpdatedAt = s.now()

	if err := s.repo.UpdateOrder(r.Context(), current, nil, func(txn outbox.TxnBuffer) error {
		if err := emitOrderStatusChanged(r.Context(), txn, orderStatusEmitParams{
			Claims:         claims,
			Order:          current,
			PreviousStatus: previousStatus,
			Reason:         "retailer_selected_cash",
			ActorID:        claims.Subject,
		}); err != nil {
			return err
		}
		cashPayment := s.paymentRequiredData(r.Context(), current)
		cashPayment["payment_method"] = "CASH"
		cashPayment["status"] = string(StatusPendingCashCollection)
		return outbox.EmitJSON(r.Context(), txn, events.AggregateOrder, current.OrderID, events.TopicMain, cashPayment)
	}); err != nil {
		s.log.ErrorContext(r.Context(), "failed to select cash payment", "order_id", orderID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_failed"})
		return
	}

	s.afterOrderMutation(r.Context(), current)

	resp := map[string]any{
		"success":  true,
		"order_id": orderID,
		"state":    current.Status,
		"message":  "awaiting_driver_cash_collection",
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
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

func resolveOrderPaymentMethod(ctx context.Context, client *spanner.Client, orderID string) string {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" || client == nil {
		return "CASH"
	}
	stmt := spanner.Statement{
		SQL: `SELECT Gateway FROM PaymentLedgerEntries
		      WHERE OrderId = @oid
		      ORDER BY OccurredAt DESC
		      LIMIT 1`,
		Params: map[string]any{"oid": orderID},
	}
	iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return "CASH"
	}
	var gateway spanner.NullString
	if err := row.Columns(&gateway); err != nil || !gateway.Valid {
		return "CASH"
	}
	switch strings.ToUpper(strings.TrimSpace(gateway.StringVal)) {
	case "CASH", "":
		return "CASH"
	default:
		return "CARD"
	}
}

func resolveRetailerDisplayName(ctx context.Context, client *spanner.Client, retailerID string) string {
	retailerID = strings.TrimSpace(retailerID)
	if retailerID == "" {
		return ""
	}
	if client == nil {
		return retailerID
	}
	row, err := client.Single().ReadRow(ctx, "Retailers", spanner.Key{retailerID}, []string{"Name"})
	if err != nil {
		return retailerID
	}
	var name spanner.NullString
	if err := row.Column(0, &name); err != nil || !name.Valid || strings.TrimSpace(name.StringVal) == "" {
		return retailerID
	}
	return strings.TrimSpace(name.StringVal)
}
