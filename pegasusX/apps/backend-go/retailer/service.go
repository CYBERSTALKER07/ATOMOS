// Package retailer owns the retailer-domain handlers, services and repository
// boundaries. In pegasusX every retailer is scoped to the single seeded
// supplier, so the registration handler does NOT accept a supplier_id from the
// body — it resolves the seeded supplier id from the application context.
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
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
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
	ListTrackingOrders(ctx context.Context, retailerID string, limit int) ([]TrackingOrder, error)
	// ListRecentReceipts returns recent completed-order snapshots for retailer
	// receipt visibility on the tracking surface.
	ListRecentReceipts(ctx context.Context, retailerID string, limit int) ([]TrackingOrder, error)
}

// Retailer is the persisted aggregate.
type Retailer struct {
	RetailerID           string
	SupplierID           string
	Phone                string
	Name                 string
	CountryCode          string
	Lat                  float64
	Lng                  float64
	H3Cell               string
	ReceivingWindowOpen  string
	ReceivingWindowClose string
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
	OrderID               string                   `json:"order_id"`
	SupplierID            string                   `json:"supplier_id"`
	RetailerID            string                   `json:"retailer_id"`
	WarehouseID           string                   `json:"warehouse_id,omitempty"`
	DriverID              string                   `json:"driver_id,omitempty"`
	VehicleID             string                   `json:"vehicle_id,omitempty"`
	RouteID               string                   `json:"route_id,omitempty"`
	ManifestID            string                   `json:"manifest_id,omitempty"`
	Status                string                   `json:"status"`
	TrackingStatus        string                   `json:"tracking_status"`
	TotalMinor            int64                    `json:"total_minor"`
	Currency              string                   `json:"currency"`
	LiveLocationAvailable bool                     `json:"live_location_available"`
	DriverLocation        *TrackingLocation        `json:"driver_location,omitempty"`
	PaymentEvidence       *TrackingPaymentEvidence `json:"payment_evidence,omitempty"`
	ReceiptDossier        *TrackingReceiptDossier  `json:"receipt_dossier,omitempty"`
	CreatedAt             string                   `json:"created_at"`
	UpdatedAt             string                   `json:"updated_at"`
	Items                 []TrackingLineItem       `json:"items"`
	DeliveryToken         string                   `json:"delivery_token,omitempty"`
	IsApproaching         bool                     `json:"is_approaching"`
	PaymentStatus         string                   `json:"payment_status,omitempty"`
	DeliveryLat           float64                  `json:"-"`
	DeliveryLng           float64                  `json:"-"`
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
	ListRetailerAIPredictions(ctx context.Context, retailerID string, limit int) ([]order.RetailerAIPrediction, error)
}

// NotificationReader provides read access to the notification inbox.
type NotificationReader interface {
	ListForRecipient(ctx context.Context, recipientID string, limit int) ([]any, error)
	MarkRead(ctx context.Context, recipientID string, notificationIDs []string) error
	UnreadCount(ctx context.Context, recipientID string) (int64, error)
}

// Service wires repository, cache, idempotency and outbox dependencies.
type Service struct {
	repo        Repository
	orders      OrderLifecycle
	cartRepo    CartRepository
	notifSvc    NotificationReader
	cache       *cache.Cache
	idem        idempotency.Store
	proximity   *RetailerProximityService
	locations   telemetry.LastLocationReader
	supplierID  string
	countryCode string
	jwtSecret   string
	jwtIssuer   string
	log         *slog.Logger
	now         func() time.Time
	newID       func() string

	mu                  sync.RWMutex
	autoOrderMu         sync.RWMutex
	favoriteSuppliers   map[string]map[string]bool
	familyByRetailer    map[string][]FamilyMember
	autoOrderByRetailer map[string]*AutoOrderSettings

	firebaseVerifier auth.FirebaseVerifier
	spannerClient    *spanner.Client
}

// ServiceConfig is the constructor input.
type ServiceConfig struct {
	Repo        Repository
	CartRepo    CartRepository
	NotifSvc    NotificationReader
	Orders      OrderLifecycle
	Cache       *cache.Cache
	Idem        idempotency.Store
	Proximity   *RetailerProximityService
	Locations   telemetry.LastLocationReader
	SupplierID  string
	CountryCode string
	JWTSecret   string
	JWTIssuer   string
	Log         *slog.Logger
	Now         func() time.Time
	NewID       func() string
	FirebaseVerifier auth.FirebaseVerifier
	Spanner          *spanner.Client
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
	return &Service{
		repo:                c.Repo,
		cartRepo:            c.CartRepo,
		notifSvc:            c.NotifSvc,
		orders:              c.Orders,
		cache:               c.Cache,
		idem:                c.Idem,
		proximity:           c.Proximity,
		locations:           c.Locations,
		supplierID:          c.SupplierID,
		countryCode:         c.CountryCode,
		jwtSecret:           c.JWTSecret,
		jwtIssuer:           c.JWTIssuer,
		log:                 c.Log,
		now:                 c.Now,
		newID:               c.NewID,
		favoriteSuppliers:   make(map[string]map[string]bool),
		familyByRetailer:    make(map[string][]FamilyMember),
		autoOrderByRetailer: make(map[string]*AutoOrderSettings),
		firebaseVerifier:    c.FirebaseVerifier,
		spannerClient:       c.Spanner,
	}
}

// SetOrderLifecycle wires the retailer-facing order aggregate after service construction.
func (s *Service) SetOrderLifecycle(orders OrderLifecycle) {
	if s == nil {
		return
	}
	s.orders = orders
}

// RegisterRequest is the wire shape for POST /v1/auth/retailer/register.
type RegisterRequest struct {
	Phone                string  `json:"phone"`
	Name                 string  `json:"name,omitempty"`
	Lat                  float64 `json:"lat"`
	Lng                  float64 `json:"lng"`
	H3Cell               string  `json:"h3_cell"`
	ReceivingWindowOpen  string  `json:"receiving_window_open,omitempty"`
	ReceivingWindowClose string  `json:"receiving_window_close,omitempty"`
}

// RegisterResponse is what callers get back.
type RegisterResponse struct {
	RetailerID string `json:"retailer_id"`
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

// Register performs the registration mutation: dedupe by phone, write retailer
// row + RETAILER_REGISTERED outbox event in the same RW transaction, then
// invalidate any cached retailer-by-phone entry.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (RegisterResponse, error) {
	if err := req.Validate(); err != nil {
		return RegisterResponse{}, err
	}

	// Dedupe: phone is uniquely indexed. A retry MAY hit a row that already
	// exists; treat that as idempotent success.
	if existing, found, err := s.repo.FindByPhone(ctx, req.Phone); err != nil {
		return RegisterResponse{}, fmt.Errorf("lookup retailer: %w", err)
	} else if found {
		return RegisterResponse{
			RetailerID: existing.RetailerID,
			Phone:      existing.Phone,
			H3Cell:     existing.H3Cell,
			CreatedAt:  existing.CreatedAt.Format(time.RFC3339Nano),
		}, nil
	}

	h3Cell := strings.TrimSpace(req.H3Cell)
	if s.proximity != nil {
		cell, err := s.proximity.CellForCoordinate(req.Lat, req.Lng)
		if err != nil {
			return RegisterResponse{}, fmt.Errorf("derive retailer h3_cell: %w", err)
		}
		h3Cell = cell
	}
	if h3Cell == "" {
		return RegisterResponse{}, errors.New("h3_cell required")
	}

	windowOpen, err := validateReceivingWindowField(req.ReceivingWindowOpen)
	if err != nil {
		return RegisterResponse{}, err
	}
	windowClose, err := validateReceivingWindowField(req.ReceivingWindowClose)
	if err != nil {
		return RegisterResponse{}, err
	}

	r := Retailer{
		RetailerID:           s.newID(),
		SupplierID:           s.supplierID,
		Phone:                req.Phone,
		Name:                 req.Name,
		CountryCode:          s.countryCode,
		Lat:                  req.Lat,
		Lng:                  req.Lng,
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
			SupplierID:  s.supplierID,
			Lat:         r.Lat,
			Lng:         r.Lng,
			H3Cell:      r.H3Cell,
			CountryCode: r.CountryCode,
		})
	})
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("persist retailer: %w", err)
	}

	// Post-commit cache invalidation. Pre-commit invalidation races with
	// rollback — TTL is the safety net but never the correctness mechanism.
	s.cache.Invalidate(ctx, retailerByPhoneKey(r.Phone))

	s.log.Info("retailer registered",
		"retailer_id", r.RetailerID,
		"supplier_id", s.supplierID,
		"h3_cell", r.H3Cell,
	)
	return RegisterResponse{
		RetailerID: r.RetailerID,
		Phone:      r.Phone,
		H3Cell:     r.H3Cell,
		CreatedAt:  r.CreatedAt.Format(time.RFC3339Nano),
	}, nil
}

func retailerByPhoneKey(phone string) string {
	return "retailer:phone:" + phone
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
			_, _ = w.Write(rec.Response)
			return
		}
	}

	var req RegisterRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.Register(r.Context(), req)
	if err != nil {
		s.log.Warn("retailer registration failed", "err", err)
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	respBytes, _ := json.Marshal(resp)
	if key := r.Header.Get("Idempotency-Key"); key != "" && s.idem != nil {
		_ = s.idem.Save(r.Context(), key, idempotency.Record{
			BodyHash:   sha256Hex(body),
			StatusCode: http.StatusCreated,
			Response:   respBytes,
			StoredAt:   s.now(),
		}, 24*time.Hour)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func defaultRetailerID() string {
	// Scaffold: timestamp-based id. Production swaps for uuid.NewV7.
	return fmt.Sprintf("ret_%d", time.Now().UnixNano())
}
