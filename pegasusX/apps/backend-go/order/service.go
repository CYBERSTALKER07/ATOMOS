// Package order owns the order aggregate: creation, state transitions,
// reassignment, and lifecycle event emission. Every mutating method runs
// inside a Repository.* method that wraps a ReadWriteTransaction so the row
// mutation and the OutboxEvents row commit atomically.
//
// Gate 5: request paths resolve supplier via TenantContext
// (resolveSupplierScope); bootstrap SeedSupplierID is fixtures/fallback only.
package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/allocation"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/payout"
	"github.com/pegasusx/pegasusx/apps/backend-go/pricing"
	"github.com/pegasusx/pegasusx/apps/backend-go/promotion"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"github.com/pegasusx/pegasusx/apps/backend-go/tax"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
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
	StatusShopClosedPending      Status = "SHOP_CLOSED_PENDING"
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
	StatusMigrationOpen          Status = "MIGRATION_OPEN"
)

// OrderSource captures how an order entered the system.
type OrderSource string

const (
	OrderSourceManual            OrderSource = "MANUAL"
	OrderSourceManualPreorder    OrderSource = "MANUAL_PREORDER"
	OrderSourceAIPreorder        OrderSource = "AI_PREORDER"
	OrderSourceBackorder         OrderSource = "BACKORDER"
	OrderSourceAutoOrder         OrderSource = "AUTO_ORDER"         // L3.5 place mode
	OrderSourcePartnerEDI        OrderSource = "PARTNER_EDI"        // Gate-3 Wave 2B EDI-lite ORDERS
	OrderSourcePartnerSandbox    OrderSource = "PARTNER_SANDBOX"    // partner sandbox API key orders
	OrderSourceMigration         OrderSource = "MIGRATION"          // historical or unreleased migrated order
	OrderSourceMigrationReleased OrderSource = "MIGRATION_RELEASED" // released live after cutover
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
	ErrProximityRequired         = errors.New("proximity_required")
)

// LineItem is one line on an order.
// Partial offload fields are additive JSON on LineItemsJson (no OrderLines table).
type LineItem struct {
	SKU           string  `json:"sku"`
	Name          string  `json:"name"`
	Quantity      int64   `json:"quantity"`
	UnitPrice     int64   `json:"unit_price_minor"` // minor units (tiyin / cents)
	UnitVolumeVU  float64 `json:"unit_volume_vu,omitempty"`
	PromotionID   string  `json:"promotion_id,omitempty"`
	HandlingClass string  `json:"handling_class,omitempty"`
	IsHazardous   bool    `json:"is_hazardous,omitempty"`
	IsPerishable  bool    `json:"is_perishable,omitempty"`
	// Partial offload (doorstep): DeliveredQty + RemainingQty == Quantity when set.
	DeliveredQty  int64  `json:"delivered_qty,omitempty"`
	RemainingQty  int64  `json:"remaining_qty,omitempty"`
	OffloadStatus string `json:"offload_status,omitempty"` // FULL|PARTIAL|NONE|RETURNED
	OffloadReason string `json:"offload_reason,omitempty"`
	// Additive fiscal fields (Soliq OFD compliance):
	VatRateBps  *int64 `json:"vat_rate_bps,omitempty"`
	TaxCategory string `json:"tax_category,omitempty"` // "STANDARD" | "ZERO" | "EXEMPT"
	// Operating-model substitution & shortage preferences + disclosure fields:
	BrandID               string         `json:"brand_id,omitempty"`
	MustHave              bool           `json:"must_have,omitempty"`
	AllowSubstitutes      bool           `json:"allow_substitutes,omitempty"`
	AllowedSubstituteSKUs []string       `json:"allowed_substitute_skus,omitempty"`
	ShortageAction        ShortageAction `json:"shortage_action,omitempty"`
	OriginalSKU           string         `json:"original_sku,omitempty"`
	SubstituteSKU         string         `json:"substitute_sku,omitempty"`
	PriceGapMinor         int64          `json:"price_gap_minor,omitempty"`
	SubstitutionReason    string         `json:"substitution_reason,omitempty"`
}

// Order is the persisted aggregate.
type Order struct {
	OrderID                string
	SupplierID             string
	RetailerID             string
	ParentOrderID          string
	IdempotencyKey         string
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
	FiscalStatus          string
	LatestFiscalReceiptID string
	LatestFiscalAttemptID string
	FiscalizedAt          *time.Time

	// Shop-closed / proximity / partial (additive Orders columns; 2026-07-29).
	ShopClosedAt                *time.Time
	ShopClosedReason            string
	ShopClosedGraceEndsAt       *time.Time
	ShopClosedResolution        string
	RecentClosedShopLeavesCount int
	PartialDelivery             bool
	ProximityUnlockedAt         *time.Time
	ProximityMethod             string

	BuyerAcceptanceStatus   string
	BuyerAcceptanceDeadline *time.Time

	// Claim window snapshot (immutable after first COMPLETED write; G3).
	ClaimWindowHours        int64
	ClaimWindowEndsAt       *time.Time
	ClaimWindowPolicySource string

	// Preorder auto-confirm opt-in (Orders.AutoConfirmOptIn DDL column).
	AutoConfirmOptIn bool `json:"auto_confirm_opt_in"`

	// Non-persisted operating-model evaluation metadata (optional runtime projection).
	HoldReason    string                     `json:"-"`
	Substitutions []ShortageEvaluationResult `json:"-"`
	CutLines      []ShortageEvaluationResult `json:"-"`

	// PendingSupplierReturns is written in the same UpdateOrder transaction and not stored on Orders.
	PendingSupplierReturns []SupplierReturn `json:"-"`
	// ConditionReports is written in the same UpdateOrder transaction and not stored on Orders.
	ConditionReports []ConditionReport `json:"-"`
	// PendingFiscalReceipts are inserted in the same UpdateOrder transaction (ADR-009).
	PendingFiscalReceipts []FiscalReceiptRow `json:"-"`

	// Ephemeral fields for F-05: Status transition buffered into UpdateOrder txn.
	TransitionReason    string         `json:"-"`
	TransitionActorRole string         `json:"-"`
	TransitionActorID   string         `json:"-"`
	TransitionEventKind string         `json:"-"`
	TransitionMetadata  map[string]any `json:"-"`

	// FiscalReceiptUpdate updates an existing attempt row (PENDING → SUCCESS|FAILED).
	FiscalReceiptUpdate *FiscalReceiptRow `json:"-"`
	// PendingFiscalSnapshots are inserted in the same UpdateOrder transaction at COMPLETED time.
	PendingFiscalSnapshots []tax.OrderLineFiscalSnapshot `json:"-"`
	// Migration & cutover linkage (M6 / Orders DDL additive columns).
	ExternalOrderNo      string `json:"external_order_no,omitempty"`
	ExternalFiscalRef    string `json:"external_fiscal_ref,omitempty"`
	SupersededByRollback bool   `json:"superseded_by_rollback,omitempty"`
	// PendingExceptionTicket is written in the same UpdateOrder transaction.
	PendingExceptionTicket *ExceptionTicket `json:"-"`
}

type ExceptionTicket struct {
	TicketID     string
	Type         string
	OrderID      string
	EhfID        string
	Severity     string
	Status       string
	Title        string
	Description  string
	AssignedRole string
	CreatedAt    time.Time
	CreatedBy    string
	Payload      json.RawMessage
}

// DeliveryProofType classifies one immutable delivery handoff proof row.
type DeliveryProofType string

const (
	DeliveryProofTypeQRHandoff         DeliveryProofType = "QR_HANDOFF"
	DeliveryProofTypeFinalizationGeo   DeliveryProofType = "FINALIZATION_GEOFENCE"
	DeliveryProofTypeCashCollectionGeo DeliveryProofType = "CASH_COLLECTION_GEOFENCE"
	DeliveryProofTypeOfflinePoD        DeliveryProofType = "OFFLINE_CRYPTOGRAPHIC_POD"
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
	CreateOrder(ctx context.Context, o *Order, emit func(outbox.TxnBuffer) error, stockOpts StockReservationOpts) error
	CreateOrderWithBackorder(ctx context.Context, o *Order, backorder *Order, inTxn func(context.Context, *spanner.ReadWriteTransaction) error, emit func(outbox.TxnBuffer) error, stockOpts StockReservationOpts) error
	UpdateOrder(ctx context.Context, o Order, proofs []DeliveryProofArtifact, emit func(outbox.TxnBuffer) error) error
	UpdateOrderWithTxn(ctx context.Context, o Order, proofs []DeliveryProofArtifact, inTxn func(context.Context, *spanner.ReadWriteTransaction) error, emit func(outbox.TxnBuffer) error) error
	GetOrder(ctx context.Context, orderID string) (Order, bool, error)
	GetOrderTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (Order, bool, error)
	// GetFiscalAttempt loads one OrderFiscalReceipts row (ADR-009 worker idempotency).
	GetFiscalAttempt(ctx context.Context, orderID, attemptID string) (FiscalReceiptRow, bool, error)
	// GetFiscalByReceiptID loads a successful attempt by public FiscalReceiptId.
	GetFiscalByReceiptID(ctx context.Context, receiptID string) (FiscalReceiptRow, bool, error)
	// FindPendingBuyerAcceptance returns orders in SUCCESS fiscal state waiting for buyer acceptance.
	FindPendingBuyerAcceptance(ctx context.Context, limit int) ([]*Order, error)
	// CountFiscalAttemptsByStatus counts attempts for retry-budget decisions.
	CountFiscalAttemptsByStatus(ctx context.Context, orderID, status string) (int64, error)
	ListRetailerOrders(ctx context.Context, retailerID string, limit int) ([]Order, error)
	ListWarehouseOrdersByDeliveryWindow(ctx context.Context, warehouseID string, from, to time.Time, limit int) ([]Order, error)
	ListDueAutoConfirmOrders(ctx context.Context, before time.Time, limit int) ([]Order, error)
	ListManifestOrders(ctx context.Context, manifestID string) ([]Order, error)
	ListWarehousePreorders(ctx context.Context, warehouseID string, limit, offset int) ([]Order, error)
	ListOrdersForStockCommitment(ctx context.Context, warehouseID string, limit int) ([]Order, error)
	ListBackorderedOrders(ctx context.Context, limit int) ([]Order, error)
	ClearBackorder(ctx context.Context, orderID string, emit func(outbox.TxnBuffer) error, stockOpts StockReservationOpts) error
	ListOrdersByStatus(ctx context.Context, supplierID, status string, limit int) ([]Order, error)
	CreateConditionReport(ctx context.Context, report ConditionReport, emit func(outbox.TxnBuffer) error) error
	ListConditionReports(ctx context.Context, orderID string) ([]ConditionReport, error)
	FindSiblingDriversForOrder(ctx context.Context, orderID string) ([]string, error)
}

// WarehouseResolver resolves the best supplier warehouse for retailer
// coordinates at order-create time.
type WarehouseResolver interface {
	ResolveNearestWarehouseID(ctx context.Context, supplierID string, store proximity.StorePoint) (string, error)
}

// PaymentCapturer allows the order service to trigger synchronous card captures.
// Implementations must be retry-safe: a repeated capture for an already-settled
// payment returns success with the provider reference instead of double-charging.
type PaymentCapturer interface {
	CaptureCardPayment(ctx context.Context, orderID string, amountMinor int64, currency string) (providerRef string, err error)
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
	repo               Repository
	cache              *cache.Cache
	warehouse          WarehouseResolver
	paymentCapturer    PaymentCapturer
	paymentRefunder    PaymentRefunder
	promotions         *promotion.Service
	credit             *credit.Service
	ar                 *ar.Service
	replanner          RouteReplanner
	commissionResolver payout.CommissionResolver

	// seedSupplierID is fixtures/bootstrap only — request paths use resolveSupplierScope.
	seedSupplierID      string
	supplierName        string
	currency            string
	retailerHub         *ws.Hub
	supplierHub         *ws.Hub
	driverHub           *ws.Hub
	spannerClient       *spanner.Client
	manifestStore       *manifest.Store
	idem                idempotency.Store
	shopGrace           time.Duration
	log                 *slog.Logger
	now                 func() time.Time
	newID               func() string
	jwtSecret           string
	handoff             *handoff.Engine
	gatewayPolicy       GatewayPolicyReader
	dispatchPlanWarm    func(ctx context.Context, warehouseID string)
	previewRateLimiter  RateLimiter
	ofd                 FiscalProvider        // optional; nil → ProviderFromEnv()
	fiscalNotifier      FiscalSuccessNotifier // optional; async SMS + Telegram QR receipt delivery
	fiscalNotifyWG      sync.WaitGroup
	claimsBridge        claimsBridge    // optional logistics claims domain
	taxSvc              *tax.Service    // optional tax regime service
	pricingSvc          pricing.Service // optional pricing engine
	draftSoDCheck       func(ctx context.Context, in DraftSoDCheckInput) error
	draftApprovalBridge DraftApprovalBridge
	proximityCfg        ProximityConfig
	allocator           Allocator
	allocationRequired  bool
	packRuleReader      PackRuleReader
	shortageStockReader ShortageStockReader
	locations           telemetry.LastLocationReader
	returnPolicies      ReturnPolicyStore
	doorstepShortfall   func(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID, supplierID, driverID, currency string, expectedMinor, receivedMinor int64, occurredAt time.Time) error
	creditTermsLookup   func(ctx context.Context, retailerID, supplierID string) (credit.ResolvedTerms, error)
	perOrderFeeFn       func(ctx context.Context, supplierID string) (feeMinor int64, currency string, err error)

	currencyPickerEnabled bool
	currencyAllowlist     []string

	mandateReader          MandateReader
	usageMu                sync.Mutex
	sellerDailyUsage       map[sellerUsageKey]SellerDailyUsage
	recordedOrderUsage     map[string]orderUsageDelta
	approvalRequests       map[string]ApprovalRequestDTO
	approvalsByClientOrder map[string]string
	approvedReplayResults  map[string]ApprovalDecisionResponse
	ordersByIdemKey        map[string]Order
	parentSagaStates       map[string]string
	timeoutCfg             TimeoutConfig
	approvalApproveMu      map[string]*sync.Mutex
	lockOrderTracer        func(string)
}

// RouteReplanner handles continuous dynamic resequencing.
type RouteReplanner interface {
	ReplanRoute(ctx context.Context, routeID, reason, actor string) error
}

type Allocator interface {
	AllocateOrder(ctx context.Context, req *allocation.AllocationRequest) (*allocation.AllocationResult, error)
	AllocateOrderTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, req *allocation.AllocationRequest) (*allocation.AllocationResult, error)
}

// ShortageContext supplies live warehouse stock and same-brand substitution candidates.
type ShortageContext struct {
	AvailableBySKU   map[string]int64
	CandidatesBySKU  map[string][]SubstituteCandidate
	EnforceAllowlist bool
}

// ShortageStockReader loads live SKU availability and approved substitution candidates
// inside an active Spanner ReadWriteTransaction (or in-memory test seam).
type ShortageStockReader func(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID string, items []LineItem) (ShortageContext, error)

// ServiceConfig is the constructor input.
type ServiceConfig struct {
	Repo               Repository
	Cache              *cache.Cache
	Warehouse          WarehouseResolver
	Promotions         *promotion.Service
	Credit             *credit.Service
	Replanner          RouteReplanner
	CommissionResolver payout.CommissionResolver
	// SeedSupplierID is bootstrap/fixture fallback only (Gate 5 Week 11). Prefer TenantContext.
	SeedSupplierID string
	// SupplierID is deprecated; use SeedSupplierID. Kept for bootstrap/test call sites.
	SupplierID      string
	SupplierName    string
	Currency        string
	ProximityCfg    ProximityConfig
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
	Allocator       Allocator
	Locations       telemetry.LastLocationReader
	ReturnPolicies  ReturnPolicyStore
	MandateReader   MandateReader

	// CurrencyPickerEnabled + CurrencyAllowlist gate CreateRequest.currency (theatre #13 Wave 2+).
	CurrencyPickerEnabled bool
	CurrencyAllowlist     []string
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
		if d, err := auth.ShopClosedGraceFromContext(context.Background(), ""); err == nil {
			grace = d
		}
	}
	if c.Currency == "" {
		if cur, err := auth.CurrencyFromContext(context.Background(), strings.TrimSpace(c.SeedSupplierID)); err == nil {
			c.Currency = cur
		}
	}
	allow := c.CurrencyAllowlist
	if len(allow) == 0 {
		allow = ParseCurrencyAllowlist("", c.Currency)
	}
	seedID := strings.TrimSpace(c.SeedSupplierID)
	if seedID == "" {
		seedID = strings.TrimSpace(c.SupplierID)
	}
	svc := &Service{
		repo:                   c.Repo,
		cache:                  c.Cache,
		warehouse:              c.Warehouse,
		promotions:             c.Promotions,
		replanner:              c.Replanner,
		commissionResolver:     c.CommissionResolver,
		seedSupplierID:         seedID,
		supplierName:           strings.TrimSpace(c.SupplierName),
		currency:               c.Currency,
		retailerHub:            c.RetailerHub,
		supplierHub:            c.SupplierHub,
		driverHub:              c.DriverHub,
		spannerClient:          c.SpannerClient,
		shopGrace:              grace,
		log:                    c.Log,
		now:                    c.Now,
		newID:                  c.NewID,
		jwtSecret:              c.JWTSecret,
		handoff:                c.Handoff,
		idem:                   c.Idem,
		credit:                 c.Credit,
		previewRateLimiter:     newSimpleRateLimiter(100 * time.Millisecond),
		allocator:              c.Allocator,
		allocationRequired:     false,
		locations:              c.Locations,
		returnPolicies:         c.ReturnPolicies,
		currencyPickerEnabled:  c.CurrencyPickerEnabled,
		currencyAllowlist:      allow,
		mandateReader:          c.MandateReader,
		sellerDailyUsage:       make(map[sellerUsageKey]SellerDailyUsage),
		recordedOrderUsage:     make(map[string]orderUsageDelta),
		approvalRequests:       make(map[string]ApprovalRequestDTO),
		approvalsByClientOrder: make(map[string]string),
		approvedReplayResults:  make(map[string]ApprovalDecisionResponse),
		ordersByIdemKey:        make(map[string]Order),
		parentSagaStates:       make(map[string]string),
		approvalApproveMu:      make(map[string]*sync.Mutex),
	}
	if svc.handoff == nil {
		svc.handoff = handoff.FromEnv()
	}
	if svc.commissionResolver == nil && c.SpannerClient != nil {
		svc.commissionResolver = billing.NewFeeScheduleResolver(c.SpannerClient)
	}
	if svc.returnPolicies == nil {
		if rp, ok := c.Repo.(ReturnPolicyStore); ok {
			svc.returnPolicies = rp
		}
	}
	return svc
}

// SetDoorstepShortfall writes a balanced shortfall journal in the cash-collect transaction.
func (s *Service) SetDoorstepShortfall(fn func(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID, supplierID, driverID, currency string, expectedMinor, receivedMinor int64, occurredAt time.Time) error) {
	if s != nil {
		s.doorstepShortfall = fn
	}
}

// SetReturnPolicyStore wires claim-window policy persistence (optional).
func (s *Service) SetReturnPolicyStore(store ReturnPolicyStore) {
	if s != nil {
		s.returnPolicies = store
	}
}

// SetLocationReader wires the telemetry driver location store for anti-spoofing velocity verification.
func (s *Service) SetLocationReader(r telemetry.LastLocationReader) {
	if s != nil {
		s.locations = r
	}
}

// SetDraftSoDCheck wires runtime four-eyes SoD & violation recording for financial draft reviews.
func (s *Service) SetDraftSoDCheck(fn func(ctx context.Context, in DraftSoDCheckInput) error) {
	if s != nil {
		s.draftSoDCheck = fn
	}
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

// SetARService wires AR invoice creation at credit leave.
func (s *Service) SetARService(svc *ar.Service) {
	s.ar = svc
}

// SetCreditTermsLookup wires Net terms and grace-period lookup for credit-leave AR invoices.
func (s *Service) SetCreditTermsLookup(fn func(ctx context.Context, retailerID, supplierID string) (credit.ResolvedTerms, error)) {
	s.creditTermsLookup = fn
}

func (s *Service) resolveARTerms(ctx context.Context, retailerID, supplierID string, leaveAt time.Time) (int64, int64, time.Time) {
	termsDays := int64(30)
	var graceDays int64
	tzName := ""
	if s.creditTermsLookup != nil {
		if terms, err := s.creditTermsLookup(ctx, retailerID, supplierID); err == nil {
			if terms.TermsDays > 0 {
				termsDays = terms.TermsDays
			}
			if terms.GracePeriodDays >= 0 {
				graceDays = terms.GracePeriodDays
			}
			tzName = terms.Timezone
		}
	} else if s.credit != nil {
		if check, cerr := s.credit.CheckCreditPath(ctx, retailerID, supplierID, 0); cerr == nil && check.TermsDays > 0 {
			termsDays = check.TermsDays
		}
	}
	base := leaveAt
	if tzName != "" {
		if loc, err := time.LoadLocation(tzName); err == nil {
			base = leaveAt.In(loc)
		}
	}
	return termsDays, graceDays, base.AddDate(0, 0, int(termsDays)).UTC()
}

func creditReserveAtCreateEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("CREDIT_RESERVE_AT_CREATE")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// SetTaxService wires the tax domain (optional).
func (s *Service) SetTaxService(t *tax.Service) {
	s.taxSvc = t
}

func (s *Service) SetTimeoutConfig(cfg TimeoutConfig) {
	s.timeoutCfg = cfg
}

func (s *Service) SetAllocator(a Allocator) {
	s.allocator = a
}

func (s *Service) SetPricingService(p pricing.Service) {
	s.pricingSvc = p
}

// SetGatewayPolicyReader wires payment gateway policy for PAYMENT_REQUIRED fanout.
func (s *Service) SetGatewayPolicyReader(reader GatewayPolicyReader) {
	s.gatewayPolicy = reader
}

func (s *Service) SetAllocationRequired(req bool) {
	s.allocationRequired = req
}

// SetPackRuleReader overrides Spanner Products lookup for pack validation (used in tests).
func (s *Service) SetPackRuleReader(reader PackRuleReader) {
	s.packRuleReader = reader
}

// SetShortageStockReader overrides live stock + candidate lookup during shortage evaluation (used in tests).
func (s *Service) SetShortageStockReader(reader ShortageStockReader) {
	s.shortageStockReader = reader
}

// GetOrder returns a single order payload snapshot by ID (used by claims and other domains).
func (s *Service) GetOrder(ctx context.Context, orderID string) (Order, bool, error) {
	if s == nil || s.repo == nil {
		return Order{}, false, fmt.Errorf("order service unavailable")
	}
	return s.repo.GetOrder(ctx, strings.TrimSpace(orderID))
}

// GetOrderForTenant returns an order only when it belongs to the request TenantContext.
// Cross-tenant IDs fail closed as not found (IDOR-safe).
func (s *Service) GetOrderForTenant(ctx context.Context, orderID string) (Order, bool, error) {
	o, ok, err := s.GetOrder(ctx, orderID)
	if err != nil || !ok {
		return o, ok, err
	}
	if t, tok := auth.TenantFromContext(ctx); tok {
		if strings.TrimSpace(o.SupplierID) != strings.TrimSpace(t.SupplierID) {
			return Order{}, false, nil
		}
	}
	return o, true, nil
}

// loadOrderForRequest loads an order with tenant IDOR when TenantContext is present.
func (s *Service) loadOrderForRequest(ctx context.Context, orderID string) (Order, bool, error) {
	if _, ok := auth.TenantFromContext(ctx); ok {
		return s.GetOrderForTenant(ctx, orderID)
	}
	return s.GetOrder(ctx, orderID)
}

// resolveSupplierScope is the Gate 5 request-scoped supplier (seed = fixtures only).
func (s *Service) resolveSupplierScope(ctx context.Context) string {
	return auth.PreferTenantSupplierID(ctx, s.seedSupplierID)
}

// resolveSupplierIDForCreate prefers TenantContext over body/seed.
func (s *Service) resolveSupplierIDForCreate(ctx context.Context, reqSupplier string) (string, error) {
	reqSupplier = strings.TrimSpace(reqSupplier)
	if t, ok := auth.TenantFromContext(ctx); ok {
		tid := strings.TrimSpace(t.SupplierID)
		if reqSupplier != "" && reqSupplier != tid {
			return "", fmt.Errorf("%w: supplier_id mismatch with tenant", ErrOrderForbidden)
		}
		return tid, nil
	}
	if reqSupplier != "" {
		return reqSupplier, nil
	}
	sid := s.resolveSupplierScope(ctx)
	if sid == "" {
		return "", fmt.Errorf("%w: tenant required", ErrOrderForbidden)
	}
	return sid, nil
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
	// Retail OS Phase 2: optional store branch; clients may send active location.
	// Server still uses lat/lng/h3 for routing — location_id is correlation metadata
	// until Orders.LocationId column ships in a later migration.
	LocationID string `json:"location_id,omitempty"`
	// SupplierID overrides the service default supplier (auto-order multi-supplier place).
	SupplierID string `json:"supplier_id,omitempty"`
	// ParentOrderID links a child order to a Gate 5 Phase 2 ParentOrders rollup row.
	ParentOrderID string `json:"parent_order_id,omitempty"`
	// Source forces order provenance when set (e.g. AUTO_ORDER from auto-order place).
	// When empty, ClassifyDelivery assigns MANUAL / preorder sources.
	Source OrderSource `json:"order_source,omitempty"`
	// Currency is optional ISO-4217. Honoured only when ORDER_CURRENCY_PICKER_ENABLED
	// and value is on ORDER_CURRENCY_ALLOWLIST; otherwise operating currency is stamped.
	Currency string `json:"currency,omitempty"`
	// AutoConfirmOptIn controls whether unconfirmed future/auto orders auto-confirm at cutoff.
	AutoConfirmOptIn bool `json:"auto_confirm_opt_in,omitempty"`
	// PaymentMethod optionally specifies CREDIT | CASH | CARD for seller mandate checks.
	PaymentMethod string `json:"payment_method,omitempty"`
	// ClientOrderID optionally provides caller-assigned idempotency key stored in Orders.IdempotencyKey.
	ClientOrderID string `json:"client_order_id,omitempty"`
}

// CreateResponse is what callers get back.
type CreateResponse struct {
	OrderID               string                     `json:"order_id"`
	WarehouseID           string                     `json:"warehouse_id,omitempty"`
	Status                Status                     `json:"status"`
	Source                OrderSource                `json:"order_source"`
	ConfirmationStatus    ConfirmationStatus         `json:"confirmation_status"`
	AutoConfirmOptIn      bool                       `json:"auto_confirm_opt_in"`
	HoldReason            string                     `json:"hold_reason,omitempty"`
	RequestedDeliveryDate string                     `json:"requested_delivery_date,omitempty"`
	DeliverBefore         string                     `json:"deliver_before,omitempty"`
	DeliveryPriority      string                     `json:"delivery_priority,omitempty"`
	DeliveryMode          string                     `json:"delivery_mode,omitempty"`
	TotalMinor            int64                      `json:"total_minor"`
	Currency              string                     `json:"currency"`
	CreatedAt             string                     `json:"created_at"`
	ReceivingWindowOpen   string                     `json:"receiving_window_open,omitempty"`
	ReceivingWindowClose  string                     `json:"receiving_window_close,omitempty"`
	BackorderOrderID      string                     `json:"backorder_order_id,omitempty"`
	BackorderedItemCount  int                        `json:"backordered_item_count,omitempty"`
	StockWarnings         []StockWarning             `json:"stock_warnings,omitempty"`
	LineItems             []LineItem                 `json:"line_items,omitempty"`
	Substitutions         []ShortageEvaluationResult `json:"substitutions,omitempty"`
	CutLines              []ShortageEvaluationResult `json:"cut_lines,omitempty"`
}

// ConfirmAIOrderRequest confirms an AI-created future order and optionally
// applies retailer edits before confirming it.
type ConfirmAIOrderRequest struct {
	OrderID               string     `json:"order_id"`
	LineItems             []LineItem `json:"line_items,omitempty"`
	RequestedDeliveryDate string     `json:"requested_delivery_date,omitempty"`
	AutoConfirmOptIn      *bool      `json:"auto_confirm_opt_in,omitempty"`
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
	AutoConfirmOptIn      *bool      `json:"auto_confirm_opt_in,omitempty"`
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
	OrderID                 string                     `json:"order_id"`
	Status                  Status                     `json:"status"`
	CustomerStage           CustomerStage              `json:"customer_stage,omitempty"`
	CustomerProgressPercent int                        `json:"customer_progress_percent,omitempty"`
	Source                  OrderSource                `json:"order_source"`
	ConfirmationStatus      ConfirmationStatus         `json:"confirmation_status"`
	AutoConfirmOptIn        bool                       `json:"auto_confirm_opt_in"`
	HoldReason              string                     `json:"hold_reason,omitempty"`
	RequestedDeliveryDate   string                     `json:"requested_delivery_date,omitempty"`
	DeliverBefore           string                     `json:"deliver_before,omitempty"`
	DeliveryPriority        string                     `json:"delivery_priority,omitempty"`
	DeliveryMode            string                     `json:"delivery_mode,omitempty"`
	PreorderBadge           string                     `json:"preorder_badge,omitempty"`
	ProposedDeliveryDate    string                     `json:"proposed_delivery_date,omitempty"`
	DeliveryProposalReason  string                     `json:"delivery_proposal_reason,omitempty"`
	AutoConfirmAt           string                     `json:"auto_confirm_at,omitempty"`
	TotalMinor              int64                      `json:"total_minor"`
	Currency                string                     `json:"currency"`
	Version                 int64                      `json:"version"`
	UpdatedAt               string                     `json:"updated_at"`
	Created                 bool                       `json:"created,omitempty"`
	Substitutions           []ShortageEvaluationResult `json:"substitutions,omitempty"`
	CutLines                []ShortageEvaluationResult `json:"cut_lines,omitempty"`
}

// RetailerAIPrediction projects a pending AI preorder for retailer review.
type RetailerAIPrediction struct {
	OrderID               string             `json:"order_id"`
	SupplierID            string             `json:"supplier_id,omitempty"`
	Source                OrderSource        `json:"order_source"`
	ConfirmationStatus    ConfirmationStatus `json:"confirmation_status"`
	AutoConfirmOptIn      bool               `json:"auto_confirm_opt_in"`
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
	OrderID             string     `json:"order_id"`
	Latitude            float64    `json:"latitude"`
	Longitude           float64    `json:"longitude"`
	AmountReceivedMinor *int64     `json:"amount_received_minor,omitempty"`
	Note                string     `json:"note,omitempty"`
	ClientTimestamp     *time.Time `json:"client_timestamp,omitempty"`
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
	ID                      string                `json:"id"`
	OrderID                 string                `json:"order_id"`
	RetailerID              string                `json:"retailer_id"`
	RetailerName            string                `json:"retailer_name"`
	State                   Status                `json:"state"`
	Status                  string                `json:"status"`
	CustomerStage           CustomerStage         `json:"customer_stage,omitempty"`
	CustomerProgressPercent int                   `json:"customer_progress_percent,omitempty"`
	TotalAmount             int64                 `json:"total_amount"`
	Currency                string                `json:"currency"`
	DeliveryAddress         string                `json:"delivery_address"`
	Latitude                float64               `json:"latitude,omitempty"`
	Longitude               float64               `json:"longitude,omitempty"`
	CreatedAt               string                `json:"created_at"`
	UpdatedAt               string                `json:"updated_at"`
	Items                   []DriverOrderLineItem `json:"items"`
	Message                 string                `json:"message,omitempty"`
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
	InTxn        func(context.Context, *spanner.ReadWriteTransaction) error
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
	case StatusPending, StatusLoaded, StatusInTransit, StatusArrived, StatusAwaitingPayment, StatusPendingCashCollection, StatusCompleted, StatusCancelled, StatusCancelRequested:
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

func applyLineItemPreferences(dst *LineItem, src LineItem) {
	if dst == nil {
		return
	}
	dst.UnitVolumeVU = src.UnitVolumeVU
	dst.PromotionID = strings.TrimSpace(src.PromotionID)
	dst.VatRateBps = src.VatRateBps
	dst.TaxCategory = strings.TrimSpace(src.TaxCategory)
	dst.BrandID = strings.TrimSpace(src.BrandID)
	dst.MustHave = src.MustHave
	dst.AllowSubstitutes = src.AllowSubstitutes
	if len(src.AllowedSubstituteSKUs) > 0 {
		allow := make([]string, 0, len(src.AllowedSubstituteSKUs))
		for _, cand := range src.AllowedSubstituteSKUs {
			if trimmed := strings.TrimSpace(cand); trimmed != "" {
				allow = append(allow, trimmed)
			}
		}
		if len(allow) > 0 {
			dst.AllowedSubstituteSKUs = allow
		}
	}
}

func (s *Service) normalizeAndQuoteLineItems(ctx context.Context, items []LineItem, reqDate *time.Time) ([]LineItem, int64, error) {
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
				SKU:           sku,
				Name:          strings.TrimSpace(li.Name),
				Quantity:      li.Quantity,
				UnitPrice:     li.UnitPrice,
				HandlingClass: li.HandlingClass,
				IsHazardous:   li.IsHazardous,
				IsPerishable:  li.IsPerishable,
			}
			applyLineItemPreferences(&item, li)
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

	iter := s.spannerClient.Single().Read(ctx, "Products", spanner.KeySetFromKeys(keys...), []string{"ProductId", "Name", "PriceMinor", "HandlingClass", "IsHazardous", "IsPerishable"})
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
		var handlingClass spanner.NullString
		if err := row.Columns(&id, &name, &price, &handlingClass, &snap.isHazardous, &snap.isPerishable); err != nil {
			return nil, 0, err
		}
		prices[id] = price
		names[id] = name
		snap.handlingClass = handlingClass.StringVal
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

		var serverPrice int64
		if s.pricingSvc != nil {
			var effectiveDate time.Time
			if reqDate != nil {
				effectiveDate = *reqDate
			} else {
				effectiveDate = s.now()
			}
			p, err := s.pricingSvc.ResolvePrice(ctx, s.resolveSupplierScope(ctx), sku, effectiveDate)
			if err != nil {
				// Fall back to Products.PriceMinor when PriceLists miss (multi-supplier
				// seed/smoke paths and catalog-only SKUs).
				if sp, ok := prices[sku]; ok {
					serverPrice = sp
				} else {
					return nil, 0, fmt.Errorf("line_items[%d].sku %s failed to resolve price: %w", i, sku, err)
				}
			} else {
				serverPrice = p
			}
		} else {
			sp, ok := prices[sku]
			if !ok {
				return nil, 0, fmt.Errorf("line_items[%d].sku %s not found in catalog", i, sku)
			}
			serverPrice = sp
		}

		snap := snapshots[sku]
		item := LineItem{
			SKU:           sku,
			Name:          names[sku],
			Quantity:      li.Quantity,
			UnitPrice:     serverPrice,
			HandlingClass: snap.handlingClass,
			IsHazardous:   snap.isHazardous,
			IsPerishable:  snap.isPerishable,
		}
		applyLineItemPreferences(&item, li)

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
	stage := ToCustomerStage(orderRecord.Status)
	return RetailerOrderLifecycleResponse{
		OrderID:                 orderRecord.OrderID,
		Status:                  orderRecord.Status,
		CustomerStage:           stage,
		CustomerProgressPercent: CustomerStageProgressPercent(stage),
		Source:                  orderRecord.Source,
		ConfirmationStatus:      orderRecord.ConfirmationStatus,
		AutoConfirmOptIn:        orderRecord.AutoConfirmOptIn,
		HoldReason:              orderRecord.HoldReason,
		RequestedDeliveryDate:   formatOptionalRFC3339(orderRecord.RequestedDeliveryDate),
		DeliverBefore:           formatOptionalRFC3339(orderRecord.DeliverBefore),
		DeliveryPriority:        string(orderRecord.DeliveryPriority),
		DeliveryMode:            deliveryModeLabel(orderRecord),
		PreorderBadge:           preorderBadgeLabel(orderRecord),
		ProposedDeliveryDate:    formatOptionalRFC3339(orderRecord.ProposedDeliveryDate),
		DeliveryProposalReason:  orderRecord.DeliveryProposalReason,
		AutoConfirmAt:           formatOptionalRFC3339(orderRecord.AutoConfirmAt),
		TotalMinor:              orderRecord.TotalMinor,
		Currency:                orderRecord.Currency,
		Version:                 version,
		UpdatedAt:               orderRecord.UpdatedAt.Format(time.RFC3339Nano),
		Created:                 created,
		Substitutions:           append([]ShortageEvaluationResult(nil), orderRecord.Substitutions...),
		CutLines:                append([]ShortageEvaluationResult(nil), orderRecord.CutLines...),
	}
}

func retailerPrediction(orderRecord Order) RetailerAIPrediction {
	return RetailerAIPrediction{
		OrderID:               orderRecord.OrderID,
		SupplierID:            orderRecord.SupplierID,
		Source:                orderRecord.Source,
		ConfirmationStatus:    orderRecord.ConfirmationStatus,
		AutoConfirmOptIn:      orderRecord.AutoConfirmOptIn,
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
