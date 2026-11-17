// Package retailer owns the retailer-domain handlers, services and repository
// boundaries. Gate 5 Phase 1: registration stamps TenantContext when present,
// otherwise the bootstrap seed (retailer multi-supplier cart is Phase 2).
package retailer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/innlookup"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"golang.org/x/crypto/bcrypt"
)

// Repository is the storage seam. Production binds this to a Spanner-backed
// implementation that runs every mutation inside a ReadWriteTransaction and
// uses the supplied TxnBuffer to write the OutboxEvents row atomically.
type Repository interface {
	// CreateRetailer inserts the row + outbox event inside one RW transaction.
	// emit is invoked with a TxnBuffer scoped to the same transaction so the
	// caller can append an outbox event atomically.
	CreateRetailer(ctx context.Context, r Retailer, emit func(outbox.TxnBuffer) error) error
	// FindByPhone returns the retailer for a phone number if present.
	FindByPhone(ctx context.Context, phone string) (Retailer, bool, error)
	// GetRetailer returns the retailer by id.
	GetRetailer(ctx context.Context, retailerID string) (Retailer, bool, error)
	// UpdateRetailer mutates a retailer row and optionally emits outbox payload.
	UpdateRetailer(ctx context.Context, r Retailer, emit func(outbox.TxnBuffer) error) error
	// ListRetailersBySupplier lists retailer rows by supplier scope.
	ListRetailersBySupplier(ctx context.Context, supplierID string) ([]Retailer, error)
	// GetSupplierPricingRule reads the effective supplier pricing authority row
	// used by retailer display/checkout surfaces.
	GetSupplierPricingRule(ctx context.Context, supplierID string) (SupplierPricingRule, bool, error)
	// ListTrackingOrders returns active orders for the retailer tracking surface.
	ListTrackingOrders(ctx context.Context, retailerID string, limit, offset int) ([]TrackingOrder, error)
	// ListRecentReceipts returns recent completed-order snapshots for retailer
	// receipt visibility on the tracking surface.
	ListRecentReceipts(ctx context.Context, retailerID string, limit int) ([]TrackingOrder, error)
}

// Legal entity onboarding classification persisted on Retailers.LegalEntityStatus.
const (
	LegalEntityVerified      = "LEGAL_VERIFIED"
	LegalEntityInformalPhone = "INFORMAL_PHONE"
)

// MigrationOpeningInvoice is the retailer-facing view of a migrated Nasiya opening balance.
type MigrationOpeningInvoice struct {
	InvoiceID       string `json:"invoice_id"`
	SupplierID      string `json:"supplier_id"`
	SupplierName    string `json:"supplier_name,omitempty"`
	RetailerID      string `json:"retailer_id"`
	OrderID         string `json:"order_id,omitempty"`
	ExternalID      string `json:"external_id,omitempty"`
	PrincipalMinor  int64  `json:"principal_minor"`
	BalanceMinor    int64  `json:"balance_minor"`
	Currency        string `json:"currency"`
	Status          string `json:"status"`
	Source          string `json:"source"`
	DunningFrozen   bool   `json:"dunning_frozen"`
	DueAt           string `json:"due_at,omitempty"`
	AcknowledgedAt  string `json:"acknowledged_at,omitempty"`
	DisputedAt      string `json:"disputed_at,omitempty"`
	DisputeReason   string `json:"dispute_reason,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
}

// MigrationARManager decouples retailer nasiya handlers from ar package imports.
type MigrationARManager interface {
	ListOpeningBalances(ctx context.Context, retailerID string) ([]MigrationOpeningInvoice, error)
	AcknowledgeOpeningInvoice(ctx context.Context, retailerID, invoiceID string) (MigrationOpeningInvoice, error)
	DisputeOpeningInvoice(ctx context.Context, retailerID, invoiceID, reason string) (MigrationOpeningInvoice, error)
}

// MigrationNotifier sends notifications on AR dispute / join requests.
type MigrationNotifier func(ctx context.Context, recipientID, recipientRole, eventType, title, body, deepLink string) error

// Retailer is the persisted aggregate.
type Retailer struct {
	RetailerID           string
	SupplierID           string
	Phone                string
	Name                 string
	CountryCode          string
	TaxID                string
	NormalizedTaxID      string
	HomeCell             string
	LegalEntityStatus    string
	InnVerifiedAt        *time.Time
	Lat                  float64
	Lng                  float64
	H3Cell               string
	ReceivingWindowOpen  string
	ReceivingWindowClose string
	DeliveryAddress      string
	PlaceID              string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// SupplierPricingRule is the retailer-facing read projection of supplier
// pricing authority.
type SupplierPricingRule struct {
	SupplierID          string
	BaseMarkupBps       int64
	RetailerDiscountBps int64
	MinMarginBps        int64
	Currency            string
	RuleVersion         int64
	UpdatedBy           string
	UpdatedAt           time.Time
}

// TrackingLineItem is the retailer tracking line-item projection.
type TrackingLineItem struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int64  `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	LineTotal   int64  `json:"line_total"`
}

// TrackingLocation is the retailer-safe driver last-location projection.
type TrackingLocation struct {
	DriverID          string   `json:"driver_id"`
	SupplierID        string   `json:"supplier_id"`
	Lat               float64  `json:"lat"`
	Lng               float64  `json:"lng"`
	Latitude          float64  `json:"latitude"`
	Longitude         float64  `json:"longitude"`
	Velocity          *float64 `json:"velocity,omitempty"`
	Heading           *float64 `json:"heading,omitempty"`
	ReportedAt        string   `json:"reported_at"`
	ReceivedAt        string   `json:"received_at"`
	StaleAfterSeconds int      `json:"stale_after_seconds"`
}

// TrackingPaymentEvidence is the latest immutable payment-ledger snapshot
// attached additively to receipt rows.
type TrackingPaymentEvidence struct {
	EntryType   string `json:"entry_type"`
	Gateway     string `json:"gateway"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	ReferenceID string `json:"reference_id,omitempty"`
	OccurredAt  string `json:"occurred_at"`
}

const trackingMissingDeliveryHandoffProof = "DELIVERY_HANDOFF_PROOF_NOT_PERSISTED"

// TrackingReceiptPaymentRecord is one immutable payment-ledger row attached
// to a receipt dossier.
type TrackingReceiptPaymentRecord struct {
	LedgerEntryID string `json:"ledger_entry_id"`
	SessionID     string `json:"session_id,omitempty"`
	OrderID       string `json:"order_id,omitempty"`
	Gateway       string `json:"gateway"`
	EntryType     string `json:"entry_type"`
	AmountMinor   int64  `json:"amount_minor"`
	Currency      string `json:"currency"`
	ReferenceID   string `json:"reference_id,omitempty"`
	Source        string `json:"source"`
	OccurredAt    string `json:"occurred_at"`
	CreatedAt     string `json:"created_at"`
}

// TrackingReceiptGatewayWebhook is one recorded gateway webhook attached
// additively to a receipt dossier.
type TrackingReceiptGatewayWebhook struct {
	WebhookID      string `json:"webhook_id"`
	SessionID      string `json:"session_id,omitempty"`
	Gateway        string `json:"gateway"`
	TransactionID  string `json:"transaction_id"`
	Status         string `json:"status"`
	AmountMinor    int64  `json:"amount_minor"`
	Currency       string `json:"currency"`
	SignatureValid bool   `json:"signature_valid"`
	ReceivedAt     string `json:"received_at"`
}

// TrackingReceiptDeliveryProof is one immutable delivery-proof artifact row.
type TrackingReceiptDeliveryProof struct {
	ProofID                 string   `json:"proof_id"`
	ProofType               string   `json:"proof_type"`
	Latitude                *float64 `json:"latitude,omitempty"`
	Longitude               *float64 `json:"longitude,omitempty"`
	DistanceM               *float64 `json:"distance_m,omitempty"`
	QRTokenHashPresent      bool     `json:"qr_token_hash_present"`
	ScannedTokenHashPresent bool     `json:"scanned_token_hash_present"`
	CapturedAt              string   `json:"captured_at"`
}

// TrackingReceiptChargebackRecord is one durable chargeback row attached to a
// receipt dossier.
type TrackingReceiptChargebackRecord struct {
	ChargebackID string `json:"chargeback_id"`
	Gateway      string `json:"gateway"`
	AmountMinor  int64  `json:"amount_minor"`
	Currency     string `json:"currency"`
	CreatedAt    string `json:"created_at"`
}

// TrackingReceiptReversalRecord is one durable chargeback-reversal row
// enriched from immutable session timeline evidence when available.
type TrackingReceiptReversalRecord struct {
	ReversalID    string `json:"reversal_id"`
	SessionID     string `json:"session_id,omitempty"`
	Gateway       string `json:"gateway"`
	AmountMinor   int64  `json:"amount_minor"`
	Currency      string `json:"currency"`
	LedgerEntryID string `json:"ledger_entry_id,omitempty"`
	CreatedAt     string `json:"created_at"`
}

// TrackingReceiptProofStatus is an explicit truth summary of which immutable
// receipt artifacts are available on the current pegasusX contract.
type TrackingReceiptProofStatus struct {
	PaymentTimelineAvailable bool     `json:"payment_timeline_available"`
	GatewayWebhooksAvailable bool     `json:"gateway_webhooks_available"`
	DeliveryProofAvailable   bool     `json:"delivery_proof_available"`
	MissingArtifacts         []string `json:"missing_artifacts,omitempty"`
}

// TrackingReceiptDossier is the retailer-facing dispute-support read model for
// a completed receipt row.
type TrackingReceiptDossier struct {
	SessionID       string                            `json:"session_id,omitempty"`
	PaymentTimeline []TrackingReceiptPaymentRecord    `json:"payment_timeline"`
	GatewayWebhooks []TrackingReceiptGatewayWebhook   `json:"gateway_webhooks"`
	DeliveryProofs  []TrackingReceiptDeliveryProof    `json:"delivery_proofs"`
	Chargebacks     []TrackingReceiptChargebackRecord `json:"chargebacks"`
	Reversals       []TrackingReceiptReversalRecord   `json:"reversals"`
	ProofStatus     TrackingReceiptProofStatus        `json:"proof_status"`
}

// TrackingOrder is the retailer-facing active delivery tracking projection.
type TrackingOrder struct {
	OrderID               string `json:"order_id"`
	SupplierID            string `json:"supplier_id"`
	RetailerID            string `json:"retailer_id"`
	ParentOrderID         string `json:"parent_order_id,omitempty"`
	WarehouseID           string `json:"warehouse_id,omitempty"`
	DriverID              string `json:"driver_id,omitempty"`
	VehicleID             string `json:"vehicle_id,omitempty"`
	LicensePlate          string `json:"license_plate,omitempty"`
	RouteID               string `json:"route_id,omitempty"`
	ManifestID            string `json:"manifest_id,omitempty"`
	Status                string `json:"status"`
	TrackingStatus        string `json:"tracking_status"`
	TotalMinor            int64  `json:"total_minor"`
	Currency              string `json:"currency"`
	LiveLocationAvailable bool   `json:"live_location_available"`
	// LocationFreshness: LIVE | LAST_KNOWN | AWAITING_TELEMETRY (G3.C honesty).
	LocationFreshness string            `json:"location_freshness,omitempty"`
	DriverLocation    *TrackingLocation `json:"driver_location,omitempty"`
	// RouteGeometry is the planned sealed-route overlay (from SupplierTruckManifests).
	RouteGeometry       *routing.RouteGeometryWire `json:"route_geometry,omitempty"`
	PaymentEvidence     *TrackingPaymentEvidence   `json:"payment_evidence,omitempty"`
	ReceiptDossier      *TrackingReceiptDossier    `json:"receipt_dossier,omitempty"`
	CreatedAt           string                     `json:"created_at"`
	UpdatedAt           string                     `json:"updated_at"`
	Items               []TrackingLineItem         `json:"items"`
	DeliveryToken       string                     `json:"delivery_token,omitempty"`
	DeliveryExpectation *order.DeliveryExpectation `json:"delivery_expectation,omitempty"`
	IsApproaching       bool                       `json:"is_approaching"`
	PaymentStatus       string                     `json:"payment_status,omitempty"`
	// ADR-009 fiscal hard-gate fields for receipt / tracking surfaces.
	FiscalStatus          string  `json:"fiscal_status,omitempty"`
	FiscalQR              string  `json:"fiscal_qr,omitempty"`
	LatestFiscalReceiptID string  `json:"latest_fiscal_receipt_id,omitempty"`
	DeliveryLat           float64 `json:"-"`
	DeliveryLng           float64 `json:"-"`
}

type TrackingEventType string

const (
	TrackingEventOrderCreated        TrackingEventType = "ORDER_CREATED"
	TrackingEventOrderStatusSnapshot TrackingEventType = "ORDER_STATUS_SNAPSHOT"
	trackingEventSourceOrderRow                        = "ORDER_ROW"
)

// TrackingEvent is the retailer-facing derived event timeline item.
// These events are additive snapshots derived from durable order rows, not a
// replay of the full outbox stream.
type TrackingEvent struct {
	EventType  TrackingEventType `json:"event_type"`
	OrderID    string            `json:"order_id"`
	Status     string            `json:"status,omitempty"`
	OccurredAt string            `json:"occurred_at"`
	Derived    bool              `json:"derived"`
	Source     string            `json:"source"`
}

// OrderLifecycle captures the retailer-facing order mutation surface owned by
// the order aggregate.
type OrderLifecycle interface {
	ConfirmAIOrder(ctx context.Context, retailerID string, req order.ConfirmAIOrderRequest) (order.RetailerOrderLifecycleResponse, error)
	RejectAIOrder(ctx context.Context, retailerID string, req order.RejectAIOrderRequest) (order.RetailerOrderLifecycleResponse, error)
	EditPreorder(ctx context.Context, retailerID string, req order.EditPreorderRequest) (order.RetailerOrderLifecycleResponse, error)
	ConfirmPreorder(ctx context.Context, retailerID string, req order.ConfirmPreorderRequest) (order.RetailerOrderLifecycleResponse, error)
	AcceptDeliveryProposal(ctx context.Context, retailerID string, req order.AcceptDeliveryProposalRequest) (order.RetailerOrderLifecycleResponse, error)
	RejectDeliveryProposal(ctx context.Context, retailerID string, req order.RejectDeliveryProposalRequest) (order.RetailerOrderLifecycleResponse, error)
	RejectPreorder(ctx context.Context, retailerID string, req order.RejectPreorderRequest) (order.RetailerOrderLifecycleResponse, error)
	ListRetailerAIPredictions(ctx context.Context, retailerID string, limit int) ([]order.RetailerAIPrediction, error)
}

// OrderCreator places real procurement orders (enterprise auto-order place mode).
// Implemented by order.Service.Create in bootstrap — never mobile_compat stubs.
type OrderCreator interface {
	Create(ctx context.Context, retailerID string, req order.CreateRequest) (order.CreateResponse, error)
}

// NotificationReader provides read access to the notification inbox.
// Optional CreateNotification enables Phase 5 variance alerts to owners.
type NotificationReader interface {
	ListForRecipient(ctx context.Context, recipientID string, limit, offset int) ([]any, error)
	MarkRead(ctx context.Context, recipientID string, notificationIDs []string) error
	MarkAllRead(ctx context.Context, recipientID string) error
	UnreadCount(ctx context.Context, recipientID string) (int64, error)
}

// NotificationWriter is optional; implemented by bootstrap adapter for alerts.
type NotificationWriter interface {
	CreateNotification(ctx context.Context, recipientID, recipientRole, eventType, title, body, deepLink string) error
}

// Service wires repository, cache, idempotency and outbox dependencies.
type Service struct {
	repo           Repository
	orders         OrderLifecycle
	cartRepo       CartRepository
	notifSvc       NotificationReader
	cache          *cache.Cache
	idem           idempotency.Store
	proximity      *RetailerProximityService
	locations      telemetry.LastLocationReader
	seedSupplierID string
	partners       TradingPartnerLookup
	countryCode    string
	jwtSecret      string
	jwtIssuer      string
	log            *slog.Logger
	now            func() time.Time
	newID          func() string

	mu                  sync.RWMutex
	autoOrderMu         sync.RWMutex
	favoriteSuppliers   map[string]map[string]bool
	familyByRetailer    map[string][]FamilyMember
	familyWritesGone    map[string]bool // orgIDs that completed Family→Team migrate policy
	autoOrderByRetailer map[string]*AutoOrderSettings
	// Phase 0/1/2 memory fallbacks when Spanner client is nil (unit tests).
	ownerByRetailer     map[string]RetailerUser
	staffByRetailer     map[string][]RetailerUser
	packsByRetailer     map[string]map[string]bool
	locationsByRetailer map[string][]RetailerLocation
	userLocations       map[string][]string // userID -> locationIDs
	// Phase 3 store stock (memory fallback)
	stockBalances   map[stockBalanceKey]memStockBalance
	stockMovements  []StockMovementDTO
	receiveSessions map[string]ReceiveSessionDTO
	receiveByOrder  map[string]string
	// Phase 4 POS memory
	posRegisters       map[string]RegisterDTO
	posSessions        map[string]PosSessionDTO
	posSales           map[string]PosSaleDTO
	posSalesByClient   map[string]string // retailerID|clientSaleID -> saleID
	posAnomaliesByOrg  map[string]map[string]PosAnomalyReview
	mandatesByRetailer map[string]map[string]SellerMandate
	// Phase 5 shifts memory
	timeEntries map[string]TimeEntryDTO // entryID -> entry
	shifts      map[string]ShiftDTO
	// Phase 6 sections + assist memory
	sections      map[string]SectionDTO
	sectionSkus   map[string]map[string]bool // sectionID -> sku set
	staffSections map[string]map[string]bool // sectionID -> userID set
	assistTickets map[string]AssistTicketDTO
	// Close-out: auto-order execution
	autoOrderWorker     *autoOrderWorkerState
	autoOrderCandidates map[string][]AutoOrderCandidate
	shadowProposalsMem  map[string][]AutoOrderShadowProposal // tests without Spanner
	// L3 sell-through (memory)
	sellThroughDaily   map[sellThroughKey]SellThroughDayDTO
	sellThroughFactors map[string]float64 // retailer|sku|day → SELL_THROUGH
	// L3.5 AutoOrder from reorder suggestions (test seed)
	reorderSuggestionSeed map[string][]RetailerReorderSuggestion
	// Place mode
	orderCreator          OrderCreator
	autoOrderPlaceEnabled bool // env AUTO_ORDER_PLACE_ENABLED (fallback when flags unset)
	placeFlags            PlaceFlagEvaluator
	soakGateDisabled      bool // test seam: bypass the soak-evidence gate
	soakBypassAuditor     SoakBypassAuditor
	soakBypassLogged      sync.Map // orgID|source → true (dedupe runtime audits)
	soakBypassAudits      []map[string]any
	// Durable-ish place/draft bucket for multi-run tests (retailer|day|sku|mode -> orderID)
	autoOrderBucket map[string]string
	// Local POS catalog (memory fallback)
	localCatalog map[string][]LocalSKU
	// B4 DEMAND_SIGNAL memory capture (tests + no-Spanner emit path)
	demandSignalsEmitted []events.DemandSignalEvent
	// C1.1 multi-org memberships (memory dual-write when Spanner nil)
	// key: userID → memberships by retailerID
	membershipsByUser map[string]map[string]RetailerMembership
	// C1.2 test override for MULTI_ORG_LOGIN_ENABLED (nil = read env)
	multiOrgLoginOverride *bool
	// C3.1 parked POS holds (memory when Spanner nil)
	posHolds         map[string]PosHoldDTO // retailerID|holdID
	posHoldsOverride *bool                 // test override for POS_HOLDS_ENABLED
	// C2.1 HQ analytics memory projections
	hqSalesDaily map[hqSalesKey]HqSalesDayDTO
	hqStockSnap  map[hqStockKey]hqStockSnap
	// C2.2 test override for HQ_ANALYTICS_ENABLED (nil = env)
	hqAnalyticsOverride *bool
	// C3.3 offline count version etag + force audit (memory)
	stockVersionByLoc    map[stockVersionKey]int64
	countForceAudits     []map[string]any
	offlineCountOverride *bool
	// C4.1 assist SLA worker
	assistSLAOverride *bool
	assistSLANotified map[string]bool // ticketID → notified (pre-column fallback)

	firebaseVerifier      auth.FirebaseVerifier
	spannerClient         *spanner.Client
	paymentSessionByOrder PaymentSessionByOrder
	dashboardOrders       RetailerDashboardOrdersQuery
	stockBalancesQuery    func(context.Context, string, string) ([]StockBalanceDTO, error)
	shiftsQuery           func(context.Context, string, string, int) ([]ShiftDTO, error)
	stockMovementsQuery   func(context.Context, string, string, string, int) ([]StockMovementDTO, error)
	enabledPacksQuery     func(context.Context, string) (EnabledSet, error)
	setPackEnabledFn      func(context.Context, string, string, string, bool, map[string]any) error
	posSalesQuery         func(context.Context, string, string, time.Time, time.Time) ([]PosSaleDTO, error)
	localSKUsQuery        func(context.Context, string, string, bool) ([]LocalSKU, error)
	sectionSkusQuery      func(context.Context, string) ([]string, error)
	sectionStaffQuery     func(context.Context, string) ([]string, error)
	replaceSectionSkusFn  func(context.Context, SectionDTO, []string) error
	addSectionSkusFn      func(context.Context, SectionDTO, []string) error
	removeSectionSkusFn   func(context.Context, string, []string) error
	replaceSectionStaffFn func(context.Context, SectionDTO, []string) error
	// W5: de-stubbed AI auto-order, fast-home BFF, and Telegram OTP binding
	predictDemandFn      func(context.Context, DemandPredictionRequest) (*DemandPredictionResponse, error)
	historicalSalesQuery func(context.Context, string, string, int) ([]int, error)
	fastHomeQuery        func(context.Context, string, string) (*FastHomeResponse, error)
	telegramLinksMem     map[string]TelegramLinkRecord // key: retailerID|userID
	botWebhookSecret     string
	innLookup            *innlookup.Service
	migrationAR          MigrationARManager
	migrationNotifier    MigrationNotifier
	OnRegistered         func(ctx context.Context, retailerID, name string) error
}

// ServiceConfig is the constructor input.
type ServiceConfig struct {
	Repo                  Repository
	CartRepo              CartRepository
	NotifSvc              NotificationReader
	INNLookup             *innlookup.Service
	MigrationAR           MigrationARManager
	MigrationNotifier     MigrationNotifier
	OnRegistered          func(ctx context.Context, retailerID, name string) error
	Orders                OrderLifecycle
	OrderCreator          OrderCreator // optional; required for mode=place
	AutoOrderPlaceEnabled bool
	Cache                 *cache.Cache
	Idem                  idempotency.Store
	Proximity             *RetailerProximityService
	Locations             telemetry.LastLocationReader
	// SeedSupplierID is bootstrap/fixture fallback only (Gate 5 Week 11).
	SeedSupplierID string
	// SupplierID is deprecated; use SeedSupplierID.
	SupplierID       string
	CountryCode      string
	JWTSecret        string
	JWTIssuer        string
	Log              *slog.Logger
	Now              func() time.Time
	NewID            func() string
	FirebaseVerifier auth.FirebaseVerifier
	Spanner          *spanner.Client
	// MultiOrgLoginEnabled overrides MULTI_ORG_LOGIN_ENABLED for tests (nil = env).
	MultiOrgLoginEnabled *bool
	// PosHoldsEnabled overrides POS_HOLDS_ENABLED for tests (nil = env).
	PosHoldsEnabled *bool
	// HqAnalyticsEnabled overrides HQ_ANALYTICS_ENABLED for tests (nil = env).
	HqAnalyticsEnabled *bool
	// OfflineCountEnabled overrides OFFLINE_COUNT_ENABLED for tests (nil = env).
	OfflineCountEnabled *bool
	// AssistSLAEnabled overrides ASSIST_SLA_ENABLED for tests (nil = env).
	AssistSLAEnabled *bool
	// PaymentSessionByOrder looks up a real PaymentSessions row by order id.
	PaymentSessionByOrder PaymentSessionByOrder
	// DashboardOrders injects child-order status counts for tests (Spanner-shaped).
	DashboardOrders RetailerDashboardOrdersQuery
	// BotWebhookSecret configures X-Bot-Webhook-Secret validation when set (defaults to env).
	BotWebhookSecret string
}

// NewService constructs a Service with sensible defaults for Now/NewID.
func NewService(c ServiceConfig) *Service {
	if c.Log == nil {
		c.Log = slog.Default()
	}
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	if c.NewID == nil {
		c.NewID = defaultRetailerID
	}
	seedID := strings.TrimSpace(c.SeedSupplierID)
	if seedID == "" {
		seedID = strings.TrimSpace(c.SupplierID)
	}
	return &Service{
		repo:                  c.Repo,
		cartRepo:              c.CartRepo,
		notifSvc:              c.NotifSvc,
		orders:                c.Orders,
		orderCreator:          c.OrderCreator,
		autoOrderPlaceEnabled: c.AutoOrderPlaceEnabled,
		autoOrderBucket:       make(map[string]string),
		cache:                 c.Cache,
		idem:                  c.Idem,
		proximity:             c.Proximity,
		locations:             c.Locations,
		seedSupplierID:        seedID,
		countryCode:           c.CountryCode,
		jwtSecret:             c.JWTSecret,
		jwtIssuer:             c.JWTIssuer,
		log:                   c.Log,
		now:                   c.Now,
		newID:                 c.NewID,
		multiOrgLoginOverride: c.MultiOrgLoginEnabled,
		posHoldsOverride:      c.PosHoldsEnabled,
		posHolds:              make(map[string]PosHoldDTO),
		hqSalesDaily:          make(map[hqSalesKey]HqSalesDayDTO),
		hqStockSnap:           make(map[hqStockKey]hqStockSnap),
		hqAnalyticsOverride:   c.HqAnalyticsEnabled,
		offlineCountOverride:  c.OfflineCountEnabled,
		stockVersionByLoc:     make(map[stockVersionKey]int64),
		assistSLAOverride:     c.AssistSLAEnabled,
		assistSLANotified:     make(map[string]bool),
		favoriteSuppliers:     make(map[string]map[string]bool),
		familyByRetailer:      make(map[string][]FamilyMember),
		familyWritesGone:      make(map[string]bool),
		autoOrderByRetailer:   make(map[string]*AutoOrderSettings),
		ownerByRetailer:       make(map[string]RetailerUser),
		staffByRetailer:       make(map[string][]RetailerUser),
		packsByRetailer:       make(map[string]map[string]bool),
		locationsByRetailer:   make(map[string][]RetailerLocation),
		userLocations:         make(map[string][]string),
		stockBalances:         make(map[stockBalanceKey]memStockBalance),
		receiveSessions:       make(map[string]ReceiveSessionDTO),
		receiveByOrder:        make(map[string]string),
		posRegisters:          make(map[string]RegisterDTO),
		posSessions:           make(map[string]PosSessionDTO),
		posSales:              make(map[string]PosSaleDTO),
		posSalesByClient:      make(map[string]string),
		posAnomaliesByOrg:     make(map[string]map[string]PosAnomalyReview),
		mandatesByRetailer:    make(map[string]map[string]SellerMandate),
		timeEntries:           make(map[string]TimeEntryDTO),
		shifts:                make(map[string]ShiftDTO),
		sections:              make(map[string]SectionDTO),
		sectionSkus:           make(map[string]map[string]bool),
		staffSections:         make(map[string]map[string]bool),
		assistTickets:         make(map[string]AssistTicketDTO),
		sellThroughDaily:      make(map[sellThroughKey]SellThroughDayDTO),
		sellThroughFactors:    make(map[string]float64),
		reorderSuggestionSeed: make(map[string][]RetailerReorderSuggestion),
		firebaseVerifier:      c.FirebaseVerifier,
		spannerClient:         c.Spanner,
		paymentSessionByOrder: c.PaymentSessionByOrder,
		dashboardOrders:       c.DashboardOrders,
		telegramLinksMem:      make(map[string]TelegramLinkRecord),
		botWebhookSecret:      strings.TrimSpace(c.BotWebhookSecret),
		innLookup:             c.INNLookup,
		migrationAR:           c.MigrationAR,
		migrationNotifier:     c.MigrationNotifier,
		OnRegistered:          c.OnRegistered,
	}
}

// SetINNLookup sets the INN lookup & JoinRequests service.
func (s *Service) SetINNLookup(svc *innlookup.Service) {
	if s == nil {
		return
	}
	s.innLookup = svc
}

// INNLookup returns the configured INN lookup service, lazily wrapping Spanner when set.
func (s *Service) INNLookup() *innlookup.Service {
	if s == nil {
		return nil
	}
	if s.innLookup != nil {
		return s.innLookup
	}
	if s.spannerClient != nil {
		store := innlookup.NewSpannerStore(s.spannerClient)
		return innlookup.NewService(innlookup.Config{
			Cache:        store,
			Tenants:      store,
			JoinRequests: store,
		})
	}
	return nil
}

// SetMigrationAR wires migrated opening-balance inspection/acknowledge/dispute backing.
func (s *Service) SetMigrationAR(m MigrationARManager) {
	if s == nil {
		return
	}
	s.migrationAR = m
}

// SetMigrationNotifier wires optional supplier/retailer notification delivery on disputes.
func (s *Service) SetMigrationNotifier(fn MigrationNotifier) {
	if s == nil {
		return
	}
	s.migrationNotifier = fn
}

// HasFirebaseVerifier reports whether OTP id_token login can verify tokens.
func (s *Service) HasFirebaseVerifier() bool {
	return s != nil && s.firebaseVerifier != nil
}

// PaymentSessionByOrder resolves a durable checkout session for an order.
type PaymentSessionByOrder func(ctx context.Context, orderID string) (sessionID, gateway string, ok bool, err error)

// SetPaymentSessionByOrder wires session lookup after payment repo construction.
func (s *Service) SetPaymentSessionByOrder(fn PaymentSessionByOrder) {
	if s == nil {
		return
	}
	s.paymentSessionByOrder = fn
}

// resolveSupplierScope prefers request TenantContext over the bootstrap seed.
func (s *Service) resolveSupplierScope(ctx context.Context) string {
	return auth.PreferTenantSupplierID(ctx, s.seedSupplierID)
}

// SetOrderLifecycle wires the retailer-facing order aggregate after service construction.
func (s *Service) SetOrderLifecycle(orders OrderLifecycle) {
	if s == nil {
		return
	}
	s.orders = orders
}

// SetOrderCreator wires real order placement for auto-order place mode.
// Call after order.Service is constructed (bootstrap two-phase wire).
func (s *Service) SetOrderCreator(creator OrderCreator) {
	if s == nil {
		return
	}
	s.orderCreator = creator
}

// SetAutoOrderPlaceEnabled toggles place mode process-wide (env AUTO_ORDER_PLACE_ENABLED).
func (s *Service) SetAutoOrderPlaceEnabled(enabled bool) {
	if s == nil {
		return
	}
	s.autoOrderPlaceEnabled = enabled
}

// PlaceFlagEvaluator is the dual-control flag surface used at place time
// (featureflags.Evaluate). Empty/nil falls back to the process-wide env bit.
type PlaceFlagEvaluator interface {
	Evaluate(ctx context.Context, flagKey, tenantType, tenantID string) (bool, string, error)
}

// SoakBypassAuditor records runtime soak-gate break-glass usage (env or flag).
type SoakBypassAuditor interface {
	RecordFlagAudit(ctx context.Context, actor, action, tenantType, tenantID, detailJSON string) error
}

// SetPlaceFlagEvaluator wires runtime dual-control evaluation for AUTO_ORDER_PLACE_ENABLED
// and AUTO_ORDER_SOAK_GATE_DISABLED.
func (s *Service) SetPlaceFlagEvaluator(eval PlaceFlagEvaluator) {
	if s == nil {
		return
	}
	s.placeFlags = eval
}

// SetSoakBypassAuditor wires PlatformAdminAudit for soak-gate break-glass (P2-7).
func (s *Service) SetSoakBypassAuditor(a SoakBypassAuditor) {
	if s == nil {
		return
	}
	s.soakBypassAuditor = a
}

// RegisterRequest is the wire shape for POST /v1/auth/retailer/register.
type RegisterRequest struct {
	Phone                string  `json:"phone"`
	Name                 string  `json:"name,omitempty"`
	SupplierID           string  `json:"supplier_id,omitempty"`
	InviteToken          string  `json:"invite_token,omitempty"`
	CountryCode          string  `json:"country_code,omitempty"`
	TaxIDTIN             string  `json:"tax_id_tin,omitempty"`
	Lat                  float64 `json:"lat"`
	Lng                  float64 `json:"lng"`
	DeliveryAddress      string  `json:"delivery_address,omitempty"`
	PlaceID              string  `json:"place_id,omitempty"`
	H3Cell               string  `json:"h3_cell"`
	ReceivingWindowOpen  string  `json:"receiving_window_open,omitempty"`
	ReceivingWindowClose string  `json:"receiving_window_close,omitempty"`
}

// RegisterResponse is what callers get back.
type RegisterResponse struct {
	RetailerID string `json:"retailer_id"`
	SupplierID string `json:"supplier_id,omitempty"`
	Phone      string `json:"phone"`
	H3Cell     string `json:"h3_cell"`
	CreatedAt  string `json:"created_at"`
}

// Validate enforces input invariants. Returns a human-readable error suitable
// for direct JSON surfacing.
func (r RegisterRequest) Validate() error {
	if strings.TrimSpace(r.Phone) == "" {
		return errors.New("phone required")
	}
	if strings.TrimSpace(r.H3Cell) != "" && len(r.H3Cell) != 15 {
		return fmt.Errorf("h3_cell must be 15-char hex, got %d", len(r.H3Cell))
	}
	if r.Lat == 0 && r.Lng == 0 {
		return errors.New("lat/lng required")
	}
	return nil
}

// RegisterWithCreated performs the registration mutation: dedupe by phone, write retailer
// row + RETAILER_REGISTERED outbox event in the same RW transaction, then
// invalidate any cached retailer-by-phone entry. It returns createdNew=false if the phone already exists.
func (s *Service) RegisterWithCreated(ctx context.Context, req RegisterRequest) (RegisterResponse, bool, error) {
	if err := req.Validate(); err != nil {
		return RegisterResponse{}, false, err
	}
	partnerID, inviteMarket, err := s.resolveTradingPartnerPack(ctx, req.SupplierID, req.InviteToken)
	if err != nil {
		return RegisterResponse{}, false, err
	}
	pack, err := auth.FiscalPackForSupplier(partnerID)
	if err != nil {
		return RegisterResponse{}, false, err
	}
	if err := assertAttachMarket(pack, req.CountryCode, inviteMarket); err != nil {
		return RegisterResponse{}, false, err
	}

	requestedCountry := strings.TrimSpace(req.CountryCode)
	if requestedCountry == "" {
		requestedCountry = s.countryCode
	}
	geo, err := proximity.StampNodeGeography(pack, req.Lat, req.Lng, requestedCountry)
	if err != nil {
		return RegisterResponse{}, false, err
	}
	if geo.H3Cell == "" {
		return RegisterResponse{}, false, auth.ErrGeographyIncomplete
	}

	return s.persistRetailerRegistration(ctx, req, partnerID, geo.CountryCode, geo.H3Cell, "")
}

// Register performs the registration mutation for callers that do not require createdNew.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (RegisterResponse, error) {
	resp, _, err := s.RegisterWithCreated(ctx, req)
	return resp, err
}

// registerSelfOnboard handles standalone desktop/mobile retailer registration without a pre-bound
// trading partner or map pin, while preserving market-law checks and never attaching to seedSupplierID.
func (s *Service) registerSelfOnboard(ctx context.Context, req RegisterRequest, rawPassword string) (RegisterResponse, bool, error) {
	if strings.TrimSpace(req.Phone) == "" {
		return RegisterResponse{}, false, errors.New("phone required")
	}
	if strings.TrimSpace(req.H3Cell) != "" && len(req.H3Cell) != 15 {
		return RegisterResponse{}, false, fmt.Errorf("h3_cell must be 15-char hex, got %d", len(req.H3Cell))
	}
	requestedCountry := strings.TrimSpace(req.CountryCode)
	if requestedCountry == "" {
		requestedCountry = s.countryCode
	}
	country, h3Cell, err := stampRetailerOptionalCoords(ctx, req.Lat, req.Lng, requestedCountry)
	if err != nil {
		return RegisterResponse{}, false, err
	}
	var passwordHash string
	if pwd := strings.TrimSpace(rawPassword); pwd != "" {
		hashed, herr := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		if herr != nil {
			return RegisterResponse{}, false, fmt.Errorf("hash password: %w", herr)
		}
		passwordHash = string(hashed)
	}
	return s.persistRetailerRegistration(ctx, req, "", country, h3Cell, passwordHash)
}

func (s *Service) persistRetailerRegistration(ctx context.Context, req RegisterRequest, partnerID, countryCode, h3Cell, passwordHash string) (RegisterResponse, bool, error) {
	// Dedupe: phone is uniquely indexed. A retry MAY hit a row that already
	// exists; treat that as idempotent success.
	if existing, found, err := s.repo.FindByPhone(ctx, req.Phone); err != nil {
		return RegisterResponse{}, false, fmt.Errorf("lookup retailer: %w", err)
	} else if found {
		return RegisterResponse{
			RetailerID: existing.RetailerID,
			SupplierID: existing.SupplierID,
			Phone:      existing.Phone,
			H3Cell:     existing.H3Cell,
			CreatedAt:  existing.CreatedAt.Format(time.RFC3339Nano),
		}, false, nil
	}

	windowOpen, err := validateReceivingWindowField(req.ReceivingWindowOpen)
	if err != nil {
		return RegisterResponse{}, false, err
	}
	windowClose, err := validateReceivingWindowField(req.ReceivingWindowClose)
	if err != nil {
		return RegisterResponse{}, false, err
	}

	r := Retailer{
		RetailerID:           s.newID(),
		SupplierID:           partnerID,
		Phone:                req.Phone,
		Name:                 req.Name,
		CountryCode:          countryCode,
		Lat:                  req.Lat,
		Lng:                  req.Lng,
		DeliveryAddress:      strings.TrimSpace(req.DeliveryAddress),
		PlaceID:              strings.TrimSpace(req.PlaceID),
		H3Cell:               h3Cell,
		ReceivingWindowOpen:  windowOpen,
		ReceivingWindowClose: windowClose,
		CreatedAt:            s.now(),
		UpdatedAt:            s.now(),
	}

	err = s.repo.CreateRetailer(ctx, r, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateRetailer, r.RetailerID, events.TopicMain, events.RetailerEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventRetailerRegistered, Timestamp: r.CreatedAt.Format(time.RFC3339Nano)},
			RetailerID:  r.RetailerID,
			Phone:       r.Phone,
			Name:        r.Name,
			SupplierID:  r.SupplierID,
			Lat:         r.Lat,
			Lng:         r.Lng,
			H3Cell:      r.H3Cell,
			CountryCode: r.CountryCode,
		})
	})
	if err != nil {
		return RegisterResponse{}, false, fmt.Errorf("persist retailer: %w", err)
	}

	// Post-commit cache invalidation. Pre-commit invalidation races with
	// rollback — TTL is the safety net but never the correctness mechanism.
	s.cache.Invalidate(ctx, retailerByPhoneKey(r.Phone))

	// Phase 0: bootstrap OWNER user for multi-user identity spine.
	if _, err := s.ensureOwnerUserWithPassword(ctx, r, passwordHash); err != nil {
		s.log.Warn("owner bootstrap after register failed", "retailer_id", r.RetailerID, "err", err)
	}
	if strings.TrimSpace(r.SupplierID) != "" {
		if _, _, lerr := s.SyncSupplierRetailerLink(ctx, RetailerLinkInput{
			SupplierID: r.SupplierID,
			RetailerID: r.RetailerID,
			TaxIDTIN:   strings.TrimSpace(req.TaxIDTIN),
			Phone:      r.Phone,
			LegalName:  r.Name,
		}); lerr != nil {
			s.log.Warn("sync supplier retailer link failed", "retailer_id", r.RetailerID, "supplier_id", r.SupplierID, "err", lerr)
		}
	}

	if s.OnRegistered != nil {
		if err := s.OnRegistered(ctx, r.RetailerID, r.Name); err != nil {
			s.log.Warn("on registered callback failed", "retailer_id", r.RetailerID, "err", err)
		}
	}

	s.log.Info("retailer registered",
		"retailer_id", r.RetailerID,
		"supplier_id", r.SupplierID,
		"h3_cell", r.H3Cell,
	)
	return RegisterResponse{
		RetailerID: r.RetailerID,
		SupplierID: r.SupplierID,
		Phone:      r.Phone,
		H3Cell:     r.H3Cell,
		CreatedAt:  r.CreatedAt.Format(time.RFC3339Nano),
	}, true, nil
}

func retailerByPhoneKey(phone string) string {
	return "retailer:phone:" + phone
}

type supplierRetailerLinkChecker interface {
	HasSupplierRetailerLink(ctx context.Context, supplierID, retailerID string) (bool, error)
}

// HasSupplierRetailerLink checks if supplierID is authorized for retailerID
// (via retailer.SupplierID, active link in SupplierRetailerLinks, or checker interface).
func (s *Service) HasSupplierRetailerLink(ctx context.Context, supplierID, retailerID string) (bool, error) {
	supplierID = strings.TrimSpace(supplierID)
	retailerID = strings.TrimSpace(retailerID)
	if supplierID == "" || retailerID == "" {
		return false, nil
	}

	if checker, ok := s.repo.(supplierRetailerLinkChecker); ok {
		return checker.HasSupplierRetailerLink(ctx, supplierID, retailerID)
	}

	// 1. Direct retailer ownership check
	if s.repo != nil {
		ret, found, err := s.repo.GetRetailer(ctx, retailerID)
		if err == nil && found {
			if strings.TrimSpace(ret.SupplierID) == supplierID {
				return true, nil
			}
		}
	}

	// 2. Spanner check in SupplierRetailerLinks
	if s.spannerClient != nil {
		stmt := spanner.Statement{
			SQL: `SELECT RetailerId FROM SupplierRetailerLinks 
			      WHERE SupplierId = @sid AND RetailerId = @rid 
			      AND (DedupStatus IS NULL OR DedupStatus != 'MERGED') 
			      LIMIT 1`,
			Params: map[string]interface{}{
				"sid": supplierID,
				"rid": retailerID,
			},
		}
		iter := s.spannerClient.Single().Query(ctx, stmt)
		defer iter.Stop()
		_, err := iter.Next()
		if err == nil {
			return true, nil
		}
		if err != iterator.Done {
			return false, err
		}
	}

	return false, nil
}

// HandleRegister is the HTTP entry-point for POST /v1/auth/retailer/register.
// Wired by retailerroutes.RegisterRoutes.
func (s *Service) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	// Optional Idempotency-Key flow.
	if key := r.Header.Get("Idempotency-Key"); key != "" && s.idem != nil {
		hash := sha256Hex(body)
		rec, hit, err := idempotency.Guard(r.Context(), s.idem, key, hash)
		switch {
		case errors.Is(err, idempotency.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "idempotency_key_payload_mismatch",
			})
			return
		case err != nil:
			s.log.Warn("idempotency guard failed", "err", err)
		case hit:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(rec.StatusCode)
			if _, err := w.Write(rec.Response); err != nil && s.log != nil {
				s.log.Warn("idempotent response write failed", "err", err)
			}
			return
		}
	}

	var req RegisterRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, createdNew, err := s.RegisterWithCreated(r.Context(), req)
	if err != nil {
		s.log.Warn("retailer registration failed", "err", err)
		writeAttachError(w, err)
		return
	}
	if !createdNew {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "phone_already_registered"})
		return
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "marshal_failed"})
		return
	}
	if key := r.Header.Get("Idempotency-Key"); key != "" && s.idem != nil {
		if err := s.idem.Save(r.Context(), key, idempotency.Record{
			BodyHash:   sha256Hex(body),
			StatusCode: http.StatusCreated,
			Response:   respBytes,
			StoredAt:   s.now(),
		}, 24*time.Hour); err != nil && s.log != nil {
			s.log.Warn("idempotency save failed", "err", err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(respBytes); err != nil && s.log != nil {
		s.log.Warn("register response write failed", "err", err)
	}
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Warn("json response encode failed", "err", err)
	}
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func defaultRetailerID() string {
	return "ret_" + strings.ReplaceAll(uuid.NewString(), "-", "")
}
