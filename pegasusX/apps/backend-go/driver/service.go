// Package driver owns driver-role handlers and local scaffold state.
package driver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/factory"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

// DriverNotificationReader provides read access to the notification inbox.
type DriverNotificationReader interface {
	ListForRecipient(ctx context.Context, recipientID string, limit, offset int) ([]any, error)
	MarkRead(ctx context.Context, recipientID string, notificationIDs []string) error
	MarkAllRead(ctx context.Context, recipientID string) error
	UnreadCount(ctx context.Context, recipientID string) (int64, error)
}

// DriverOrderLineView is a line item on the driver fleet order projection.
type DriverOrderLineView struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int64  `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
}

// DriverOrderView is a single order row projected for the driver fleet surface.
type DriverOrderView struct {
	OrderID            string                `json:"id"`
	RetailerID         string                `json:"retailer_id"`
	RetailerName       string                `json:"retailer_name"`
	Status             string                `json:"state"`
	TotalMinor         int64                 `json:"total_amount"`
	DeliveryFeeMinor   int64                 `json:"delivery_fee_minor,omitempty"`
	DeliveryDistanceKm float64               `json:"delivery_distance_km,omitempty"`
	DeliveryAddress    string                `json:"delivery_address,omitempty"`
	Lat                float64               `json:"latitude"`
	Lng                float64               `json:"longitude"`
	PaymentGateway     string                `json:"payment_gateway"`
	RouteID            string                `json:"route_id,omitempty"`
	SequenceIndex      int64                 `json:"sequence_index,omitempty"`
	Items              []DriverOrderLineView `json:"items,omitempty"`
	SplitGroupID       string                `json:"split_group_id,omitempty"`
	CreatedAt          string                `json:"created_at"`
	UpdatedAt          string                `json:"updated_at"`

	// AssignedDriverID is the order's assigned driver, used for the fail-closed
	// ownership check in HandleOrderGet. Never serialized to clients.
	AssignedDriverID string `json:"-"`
}

// DriverOrderQuery lists active orders assigned to a driver from Spanner.
type DriverOrderQuery func(ctx context.Context, driverID string) ([]DriverOrderView, error)

// DriverHistoryQuery lists completed-window orders for GET /v1/driver/history.
type DriverHistoryQuery func(ctx context.Context, driverID string, since time.Time, limit int) ([]HistoryRow, error)

// DriverOrderGetQuery retrieves a single order by ID from Spanner.
type DriverOrderGetQuery func(ctx context.Context, orderID string) (DriverOrderView, bool, error)

// DriverProfileSnapshot is the durable driver row slice exposed on profile.
type DriverProfileSnapshot struct {
	VehicleID   string
	TruckStatus string
	RouteID     string
}

// DriverProfileLookup reads durable driver assignment fields from Spanner.
type DriverProfileLookup func(ctx context.Context, driverID string) (DriverProfileSnapshot, bool, error)

// FleetAvailabilityBroadcaster fans out warehouse-scoped fleet availability WS frames.
type FleetAvailabilityBroadcaster func(ctx context.Context, warehouseID string, payload map[string]any)

// ManifestDeliveryTokenLookup resolves persisted delivery tokens for manifest orders.
type ManifestDeliveryTokenLookup func(ctx context.Context, orderIDs []string) map[string]string

// Service keeps additive in-memory driver state for scaffold routes.
type Service struct {
	repo              Repository
	cache             *cache.Cache
	notifSvc          DriverNotificationReader
	orderList         DriverOrderQuery
	orderGet          DriverOrderGetQuery
	supplierHub       *ws.Hub
	driverHub         *ws.Hub
	log               *slog.Logger
	manifestGate      ManifestGateLookup
	manifest          ManifestLookup
	manifestTokens    ManifestDeliveryTokenLookup
	pendingQuery      PendingCollectionsLookup
	earnings          EarningsLookup
	depart            DepartFn
	returnComplete    ReturnCompleteFn
	openFiscal        OpenFiscalLookup
	cashReconRequired bool
	cashReconGate     CashReconciliationGateLookup
	routeGeometry     RouteGeometryLookup
	profileLookup     DriverProfileLookup
	availReader       AvailabilityReader
	planInvalidate    func(ctx context.Context, warehouseID string)
	fleetBroadcast    FleetAvailabilityBroadcaster
	idem              idempotency.Store

	seedSupplierID string
	currency       string
	jwtSecret      string
	jwtIssuer      string

	historyQuery DriverHistoryQuery

	mu                 sync.RWMutex
	availability       map[string]bool
	earningsMinor      map[string]int64
	pendingCollections map[string][]PendingCollection
	now                func() time.Time
	firebaseVerifier   auth.FirebaseVerifier
	loginLookup        DriverLoginLookup
}

// ServiceConfig is the constructor input.
type ServiceConfig struct {
	Repo                         Repository
	Cache                        *cache.Cache
	NotifSvc                     DriverNotificationReader
	OrderList                    DriverOrderQuery
	HistoryQuery                 DriverHistoryQuery
	OrderGet                     DriverOrderGetQuery
	SupplierHub                  *ws.Hub
	DriverHub                    *ws.Hub
	Log                          *slog.Logger
	ManifestGate                 ManifestGateLookup
	Manifest                     ManifestLookup
	ManifestTokens               ManifestDeliveryTokenLookup
	PendingQuery                 PendingCollectionsLookup
	Earnings                     EarningsLookup
	Depart                       DepartFn
	ReturnComplete               ReturnCompleteFn
	OpenFiscal                   OpenFiscalLookup
	CashReconciliationRequired   bool
	CashReconciliationGate       CashReconciliationGateLookup
	RouteGeometry                RouteGeometryLookup
	ProfileLookup                DriverProfileLookup
	AvailabilityReader           AvailabilityReader
	DispatchPlanInvalidate       func(ctx context.Context, warehouseID string)
	FleetAvailabilityBroadcaster FleetAvailabilityBroadcaster
	// SeedSupplierID is bootstrap/fixture fallback only (Gate 5 Week 11).
	SeedSupplierID string
	// SupplierID is deprecated; use SeedSupplierID.
	SupplierID       string
	Currency         string
	JWTSecret        string
	JWTIssuer        string
	Now              func() time.Time
	FirebaseVerifier auth.FirebaseVerifier
	Idem             idempotency.Store
}

// ManifestGateResult is the read-model response for driver ghost-stop checks.
type ManifestGateResult struct {
	ManifestID string
	State      string
	StopCount  int
	VolumeVU   int64
}

// ManifestGateLookup resolves the current manifest gate state for a manifest id.
type ManifestGateLookup func(manifestID string) (ManifestGateResult, bool)

// ManifestLookup resolves the current manifest detail for a driver-scoped read.
type ManifestLookup func(driverID, manifestID, date string) (factory.ManifestDetailSnapshot, bool)

// PendingCollectionsLookup resolves the current pending cash-collection read model.
type PendingCollectionsLookup func(driverID string) []PendingCollection

// EarningsLookup resolves the current driver earnings read model.
type EarningsLookup func(ctx context.Context, driverID string) (DriverEarningsResponse, error)

// DepartResult summarizes a committed driver-depart transition.
type DepartResult struct {
	ManifestID string   `json:"manifest_id"`
	OrderIDs   []string `json:"order_ids"`
	Count      int      `json:"orders_dispatched"`
}

// DepartFn flips a driver's SEALED manifest to DISPATCHED and rolls its LOADED
// orders to IN_TRANSIT atomically. ok=false means the driver had no sealed
// manifest (idempotent no-op).
type DepartFn func(ctx context.Context, driverID string) (DepartResult, bool, error)

// ReturnCompleteResult summarizes the manifest closed when a driver returns to depot.
type ReturnCompleteResult struct {
	ManifestID string   `json:"manifest_id"`
	OrderIDs   []string `json:"order_ids"`
	Count      int      `json:"orders_returned"`
}

// ReturnCompleteFn flips a driver's DISPATCHED manifest to COMPLETED and marks
// the driver as off-shift atomically. ok=false means no DISPATCHED manifest
// found (idempotent no-op for double-tap return).
type ReturnCompleteFn func(ctx context.Context, driverID string) (ReturnCompleteResult, bool, error)

// OpenFiscalSnapshot is the Phase 6 soft-freeze signal (cash bag / shift-end).
type OpenFiscalSnapshot struct {
	Count    int64    `json:"open_fiscal_count"`
	OrderIDs []string `json:"order_ids,omitempty"`
	Frozen   bool     `json:"cash_bag_frozen"`
}

// OpenFiscalLookup counts orders still in FISCALIZING / FISCAL_FAILED for a driver.
type OpenFiscalLookup func(ctx context.Context, driverID string) (OpenFiscalSnapshot, error)

// CashReconciliationGateLookup returns true when the driver has an accepted cash reconciliation for today.
type CashReconciliationGateLookup func(ctx context.Context, driverID string) (bool, error)

// ErrOpenFiscalBlock is returned when shift-end is blocked by open fiscal attempts.
var ErrOpenFiscalBlock = errors.New("open_fiscal_block")

// DailyEarning is one day of driver delivery volume.
type DailyEarning struct {
	Date          string `json:"date"`
	DeliveryCount int64  `json:"delivery_count"`
	Volume        int64  `json:"volume"`
	Currency      string `json:"currency,omitempty"`
	VolumeMinor   int64  `json:"volume_minor,omitempty"`
}

// DriverEarningsResponse mirrors the driver mobile earnings contract.
type DriverEarningsResponse struct {
	DriverID        string         `json:"driver_id,omitempty"`
	Currency        string         `json:"currency,omitempty"`
	TotalDeliveries int64          `json:"total_deliveries"`
	TotalVolume     int64          `json:"total_volume"`
	TotalRoutes     int64          `json:"total_routes"`
	Last30Days      []DailyEarning `json:"last_30_days"`
	TodayMinor      int64          `json:"today_minor"`
	WeekMinor       int64          `json:"week_minor"`
	MonthMinor      int64          `json:"month_minor"`
}

// HistoryRow represents one completed or in-progress delivery history row.
type HistoryRow struct {
	OrderID     string `json:"order_id"`
	Status      string `json:"status"`
	TotalMinor  int64  `json:"total_minor"`
	Currency    string `json:"currency"`
	CompletedAt string `json:"completed_at,omitempty"`
}

// PendingCollection is one outstanding cash collection task.
type PendingCollection struct {
	OrderID     string `json:"order_id"`
	RetailerID  string `json:"retailer_id,omitempty"`
	Amount      int64  `json:"amount"`
	State       string `json:"state"`
	UpdatedAt   string `json:"updated_at"`
	AmountMinor int64  `json:"amount_minor,omitempty"`
	Currency    string `json:"currency,omitempty"`
	DueAt       string `json:"due_at,omitempty"`
}

// NewService constructs the driver service.
func NewService(c ServiceConfig) *Service {
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	if c.Log == nil {
		c.Log = slog.Default()
	}
	if strings.TrimSpace(c.Currency) == "" {
		c.Currency = "UZS"
	}
	seedID := strings.TrimSpace(c.SeedSupplierID)
	if seedID == "" {
		seedID = strings.TrimSpace(c.SupplierID)
	}
	return &Service{
		availability:       make(map[string]bool),
		earningsMinor:      make(map[string]int64),
		pendingCollections: make(map[string][]PendingCollection),
		repo:               c.Repo,
		cache:              c.Cache,
		notifSvc:           c.NotifSvc,
		orderList:          c.OrderList,
		historyQuery:       c.HistoryQuery,
		orderGet:           c.OrderGet,
		supplierHub:        c.SupplierHub,
		driverHub:          c.DriverHub,
		log:                c.Log,
		manifestGate:       c.ManifestGate,
		manifest:           c.Manifest,
		manifestTokens:     c.ManifestTokens,
		pendingQuery:       c.PendingQuery,
		earnings:           c.Earnings,
		depart:             c.Depart,
		returnComplete:     c.ReturnComplete,
		openFiscal:         c.OpenFiscal,
		cashReconRequired:  c.CashReconciliationRequired,
		cashReconGate:      c.CashReconciliationGate,
		routeGeometry:      c.RouteGeometry,
		profileLookup:      c.ProfileLookup,
		availReader:        c.AvailabilityReader,
		planInvalidate:     c.DispatchPlanInvalidate,
		fleetBroadcast:     c.FleetAvailabilityBroadcaster,
		seedSupplierID:     seedID,
		currency:           strings.ToUpper(strings.TrimSpace(c.Currency)),
		jwtSecret:          strings.TrimSpace(c.JWTSecret),
		jwtIssuer:          strings.TrimSpace(c.JWTIssuer),
		now:                c.Now,
		firebaseVerifier:   c.FirebaseVerifier,
		idem:               c.Idem,
	}
}

// resolveSupplierScope prefers request TenantContext over the bootstrap seed.
func (s *Service) resolveSupplierScope(ctx context.Context) string {
	return auth.PreferTenantSupplierID(ctx, s.seedSupplierID)
}

type availabilityPatchRequest struct {
	OnShift bool   `json:"on_shift"`
	Reason  string `json:"reason,omitempty"`
	Note    string `json:"note,omitempty"`
}

func (s *Service) driverOnShiftSnapshot(ctx context.Context, driverID string) bool {
	onShift, known := s.driverOnShiftKnown(ctx, driverID)
	if known {
		return onShift
	}
	return true
}

func (s *Service) driverOnShiftKnown(ctx context.Context, driverID string) (onShift bool, known bool) {
	if s.availReader != nil {
		onShift, _, _, ok, err := s.availReader(ctx, driverID)
		if err == nil && ok {
			return onShift, true
		}
	}
	s.mu.RLock()
	onShift, existed := s.availability[driverID]
	s.mu.RUnlock()
	return onShift, existed
}

func (s *Service) persistDriverAvailability(ctx context.Context, driverID string, onShift bool, reason, note string, event events.DriverEvent) error {
	now := s.now()
	emit := func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateDriver, driverID, events.TopicMain, event)
	}
	if writer, ok := s.repo.(AvailabilityWriter); ok {
		return writer.ApplyAvailability(ctx, AvailabilityUpdate{
			DriverID:  driverID,
			OnShift:   onShift,
			Reason:    reason,
			Note:      note,
			UpdatedAt: now,
		}, emit)
	}
	return s.repo.Apply(ctx, func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.availability[driverID] = onShift
		return nil
	}, emit)
}

// SetDispatchPlanInvalidate wires warehouse dispatch plan cache busting after bootstrap constructs warehouse.
func (s *Service) SetDispatchPlanInvalidate(fn func(ctx context.Context, warehouseID string)) {
	s.planInvalidate = fn
}

// SetFleetAvailabilityBroadcaster wires immediate warehouse WS fan-out for driver availability.
func (s *Service) SetFleetAvailabilityBroadcaster(fn FleetAvailabilityBroadcaster) {
	s.fleetBroadcast = fn
}
func (s *Service) HandleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	claims, _ := auth.FromContext(r.Context())
	onShift := s.driverOnShiftSnapshot(r.Context(), driverID)
	resp := map[string]any{
		"driver_id":      driverID,
		"home_node_type": claims.HomeNodeType,
		"home_node_id":   claims.HomeNodeID,
		"on_shift":       onShift,
		"updated_at":     s.now().Format(time.RFC3339Nano),
	}
	if s.profileLookup != nil {
		if snap, ok, err := s.profileLookup(r.Context(), driverID); err == nil && ok {
			if strings.TrimSpace(snap.VehicleID) != "" {
				resp["vehicle_id"] = strings.TrimSpace(snap.VehicleID)
			}
			if strings.TrimSpace(snap.TruckStatus) != "" {
				resp["truck_status"] = strings.TrimSpace(snap.TruckStatus)
			}
			if strings.TrimSpace(snap.RouteID) != "" {
				resp["route_id"] = strings.TrimSpace(snap.RouteID)
			}
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

const driverHistoryWindow = 30 * 24 * time.Hour
const driverHistoryLimit = 50

// HandleHistory serves GET /v1/driver/history.
func (s *Service) HandleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.historyQuery == nil {
		writeJSON(w, http.StatusOK, map[string]any{"rows": []HistoryRow{}})
		return
	}
	since := s.now().Add(-driverHistoryWindow)
	rows, err := s.historyQuery(r.Context(), driverID, since, driverHistoryLimit)
	if err != nil {
		s.log.ErrorContext(r.Context(), "driver history query failed", "err", err, "driver_id", driverID)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "history_unavailable"})
		return
	}
	if rows == nil {
		rows = []HistoryRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

// HandleEarnings serves GET /v1/driver/earnings.
func (s *Service) HandleEarnings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	earnings, err := s.earningsSnapshot(r.Context(), driverID)
	if err != nil {
		s.log.Error("driver earnings read failed", "driver_id", driverID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "earnings_read_failed"})
		return
	}
	writeJSON(w, http.StatusOK, earnings)
}

func (s *Service) earningsSnapshot(ctx context.Context, driverID string) (DriverEarningsResponse, error) {
	if s.earnings != nil {
		response, err := s.earnings(ctx, driverID)
		if err != nil {
			return DriverEarningsResponse{}, err
		}
		return s.normalizeEarningsResponse(driverID, response), nil
	}
	s.mu.RLock()
	total := s.earningsMinor[driverID]
	s.mu.RUnlock()
	return DriverEarningsResponse{
		DriverID:    driverID,
		Currency:    s.currency,
		TotalVolume: total,
		Last30Days:  []DailyEarning{},
		TodayMinor:  total,
		WeekMinor:   total,
		MonthMinor:  total,
	}, nil
}

func (s *Service) normalizeEarningsResponse(driverID string, response DriverEarningsResponse) DriverEarningsResponse {
	if strings.TrimSpace(response.DriverID) == "" {
		response.DriverID = driverID
	}
	if strings.TrimSpace(response.Currency) == "" {
		response.Currency = s.currency
	}
	if response.Last30Days == nil {
		response.Last30Days = []DailyEarning{}
	}
	for i := range response.Last30Days {
		if response.Last30Days[i].Currency == "" {
			response.Last30Days[i].Currency = response.Currency
		}
		if response.Last30Days[i].VolumeMinor == 0 && response.Last30Days[i].Volume != 0 {
			response.Last30Days[i].VolumeMinor = response.Last30Days[i].Volume
		}
	}
	return response
}

// HandleAvailability serves GET/PATCH /v1/driver/availability.
func (s *Service) HandleAvailability(w http.ResponseWriter, r *http.Request) {
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		onShift := s.driverOnShiftSnapshot(r.Context(), driverID)
		writeJSON(w, http.StatusOK, map[string]any{"driver_id": driverID, "on_shift": onShift, "available": onShift})
	case http.MethodPost:
		body, err := readLimitedBody(r, 8*1024)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
			return
		}
		if s.guardIdempotency(w, r, body) {
			return
		}
		var req struct {
			Available bool   `json:"available"`
			Reason    string `json:"reason"`
			Note      string `json:"note"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		s.patchDriverAvailability(w, r, driverID, body, availabilityPatchRequest{
			OnShift: req.Available,
			Reason:  req.Reason,
			Note:    req.Note,
		})
	case http.MethodPatch:
		body, err := readLimitedBody(r, 8*1024)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
			return
		}
		if s.guardIdempotency(w, r, body) {
			return
		}
		var req availabilityPatchRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		s.patchDriverAvailability(w, r, driverID, body, req)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) patchDriverAvailability(w http.ResponseWriter, r *http.Request, driverID string, idemBody []byte, req availabilityPatchRequest) {
	current, known := s.driverOnShiftKnown(r.Context(), driverID)
	if known && current == req.OnShift {
		resp := map[string]any{"driver_id": driverID, "on_shift": req.OnShift, "no_change": true}
		respBytes, _ := json.Marshal(resp)
		s.saveIdempotency(r.Context(), r, idemBody, http.StatusOK, respBytes)
		writeJSONBytes(w, http.StatusOK, respBytes)
		return
	}

	reason := ""
	note := strings.TrimSpace(req.Note)
	if !req.OnShift {
		reason = normalizeDriverUnavailableReason(req.Reason)
		if reason == ReasonOther && note == "" {
			note = strings.TrimSpace(req.Reason)
		}
	}

	nowTS := s.now().Format(time.RFC3339Nano)
	claims, _ := auth.FromContext(r.Context())
	homeNodeType := strings.TrimSpace(string(claims.HomeNodeType))
	homeNodeID := strings.TrimSpace(claims.HomeNodeID)
	eventPayload := events.DriverEvent{
		BaseEvent:    events.BaseEvent{Type: events.EventDriverAvailabilityChanged, Timestamp: nowTS, Version: 1},
		DriverID:     driverID,
		Available:    req.OnShift,
		OnShift:      req.OnShift,
		Reason:       reason,
		Note:         note,
		SupplierID:   s.resolveSupplierScope(r.Context()),
		HomeNodeType: homeNodeType,
		HomeNodeID:   homeNodeID,
	}

	if err := s.persistDriverAvailability(r.Context(), driverID, req.OnShift, reason, note, eventPayload); err != nil {
		s.log.Error("driver availability persist failed", "driver_id", driverID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_availability_failed"})
		return
	}

	s.mu.Lock()
	s.availability[driverID] = req.OnShift
	s.mu.Unlock()

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), driverAvailabilityKey(driverID))
	}
	if s.planInvalidate != nil && strings.EqualFold(homeNodeType, "WAREHOUSE") && homeNodeID != "" {
		s.planInvalidate(r.Context(), homeNodeID)
	}
	if s.fleetBroadcast != nil && strings.EqualFold(homeNodeType, "WAREHOUSE") && homeNodeID != "" {
		raw, _ := json.Marshal(eventPayload)
		formatted := notifications.FormatFromEvent(events.EventDriverAvailabilityChanged, raw)
		s.fleetBroadcast(r.Context(), homeNodeID, map[string]any{
			"type":           events.EventDriverAvailabilityChanged,
			"driver_id":      driverID,
			"on_shift":       req.OnShift,
			"available":      req.OnShift,
			"reason":         reason,
			"note":           note,
			"home_node_id":   homeNodeID,
			"home_node_type": homeNodeType,
			"title":          formatted.Title,
			"body":           formatted.Body,
			"deep_link":      formatted.DeepLink,
			"timestamp":      nowTS,
		})
	}
	s.broadcastDriverEvent(r.Context(), driverID, eventPayload)
	s.log.Info("driver availability changed", "driver_id", driverID, "on_shift", req.OnShift, "reason", reason)
	resp := map[string]any{"driver_id": driverID, "on_shift": req.OnShift, "reason": reason}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, idemBody, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandlePendingCollections serves GET /v1/driver/pending-collections.
func (s *Service) HandlePendingCollections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	pending := s.pendingCollectionsSnapshot(driverID)
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("envelope")), "1") ||
		strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("envelope")), "true") {
		writeJSON(w, http.StatusOK, map[string]any{
			"pending_collections": pending,
			"count":               len(pending),
			"pending":             pending,
		})
		return
	}
	writeJSON(w, http.StatusOK, pending)
}

func (s *Service) pendingCollectionsSnapshot(driverID string) []PendingCollection {
	var pending []PendingCollection
	if s.pendingQuery != nil {
		pending = append([]PendingCollection(nil), s.pendingQuery(driverID)...)
	} else {
		s.mu.RLock()
		pending = append([]PendingCollection(nil), s.pendingCollections[driverID]...)
		s.mu.RUnlock()
	}
	if len(pending) == 0 {
		return []PendingCollection{}
	}
	normalized := make([]PendingCollection, 0, len(pending))
	for _, item := range pending {
		normalized = append(normalized, normalizePendingCollection(item))
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		return normalized[i].UpdatedAt > normalized[j].UpdatedAt
	})
	return normalized
}

func normalizePendingCollection(item PendingCollection) PendingCollection {
	if item.Amount == 0 && item.AmountMinor != 0 {
		item.Amount = item.AmountMinor
	}
	if item.AmountMinor == 0 && item.Amount != 0 {
		item.AmountMinor = item.Amount
	}
	if item.State == "" {
		item.State = "PENDING_CASH_COLLECTION"
	}
	if item.UpdatedAt == "" {
		item.UpdatedAt = item.DueAt
	}
	return item
}

// HandleManifestGate serves GET /v1/driver/manifest-gate.
func (s *Service) HandleManifestGate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	manifestID := strings.TrimSpace(r.URL.Query().Get("manifest_id"))
	if manifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}
	if s.manifestGate == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "manifest_gate_unavailable"})
		return
	}
	gate, ok := s.manifestGate(manifestID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"driver_id":   driverID,
			"manifest_id": manifestID,
			"allowed":     false,
			"cleared":     false,
			"error":       "manifest_not_found",
		})
		return
	}
	if strings.TrimSpace(gate.ManifestID) == "" {
		gate.ManifestID = manifestID
	}
	if manifestGateCleared(gate.State) {
		writeJSON(w, http.StatusOK, map[string]any{
			"driver_id":     driverID,
			"manifest_id":   gate.ManifestID,
			"state":         gate.State,
			"allowed":       true,
			"cleared":       true,
			"reason":        "ok",
			"stop_count":    gate.StopCount,
			"volume_vu":     gate.VolumeVU,
			"offline_nonce": s.GenerateOfflineNonce(gate.ManifestID, driverID),
		})
		return
	}
	gateBody := map[string]any{
		"driver_id":   driverID,
		"manifest_id": gate.ManifestID,
		"state":       gate.State,
		"allowed":     false,
		"cleared":     false,
		"reason":      events.EventManifestSealed,
		"error":       "AWAITING_PAYLOAD_SEAL",
		"message":     "Manifest is in " + gate.State + " state. Wait for Payloader to complete loading and seal.",
	}
	platform.AttachExplain(gateBody, nil)
	writeJSON(w, http.StatusForbidden, gateBody)
}

// HandleManifest serves GET /v1/driver/manifest and legacy GET /v1/fleet/manifest.
func (s *Service) HandleManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.manifest == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "manifest_lookup_unavailable"})
		return
	}
	manifestID := strings.TrimSpace(r.URL.Query().Get("manifest_id"))
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if manifestID == "" {
		writeJSON(w, http.StatusOK, iosRouteManifest(driverID, date))
		return
	}
	detail, ok := s.manifest(driverID, manifestID, date)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"driver_id":   driverID,
			"manifest_id": manifestID,
			"date":        date,
			"error":       "manifest_not_found",
		})
		return
	}
	hashes := s.offlineManifestHashes(r.Context(), detail)
	writeJSON(w, http.StatusOK, map[string]any{
		"driver_id":               driverID,
		"manifest_id":             detail.Manifest.ManifestID,
		"date":                    date,
		"route_id":                detail.RouteID,
		"stop_count":              detail.StopCount,
		"order_count":             detail.OrderCount,
		"manifest":                detail.Manifest,
		"transfers":               detail.Transfers,
		"transitions":             detail.Transitions,
		"reassignments":           detail.Reassignments,
		"exceptions":              detail.Exceptions,
		"hashes":                  hashes,
		"legacy_hashes_available": len(hashes) > 0,
		"offline_nonce":           s.GenerateOfflineNonce(detail.Manifest.ManifestID, driverID),
	})
}

func (s *Service) offlineManifestHashes(ctx context.Context, detail factory.ManifestDetailSnapshot) map[string]string {
	orderIDs := make([]string, 0, len(detail.Transfers))
	seen := make(map[string]struct{})
	for _, transfer := range detail.Transfers {
		orderID := strings.TrimSpace(transfer.OrderID)
		if orderID == "" {
			continue
		}
		if _, ok := seen[orderID]; ok {
			continue
		}
		seen[orderID] = struct{}{}
		orderIDs = append(orderIDs, orderID)
	}

	tokens := map[string]string{}
	if s.manifestTokens != nil && len(orderIDs) > 0 {
		tokens = s.manifestTokens(ctx, orderIDs)
	}
	if len(tokens) == 0 && allowDriverDemoFallback() {
		tokens = demoOrderDeliveryTokens()
	}

	hashes := make(map[string]string, len(tokens))
	for orderID, token := range tokens {
		if hashed := hashDeliveryToken(token); hashed != "" {
			hashes[orderID] = hashed
		}
	}
	return hashes
}

func demoOrderDeliveryTokens() map[string]string {
	out := make(map[string]string)
	for _, row := range demoFleetOrders("") {
		id, _ := row["id"].(string)
		token, _ := row["qr_token"].(string)
		if strings.TrimSpace(id) != "" && strings.TrimSpace(token) != "" {
			out[id] = token
		}
	}
	return out
}

func hashDeliveryToken(token string) string {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])
}

// GenerateOfflineNonce deterministically derives an offline signing secret.
func (s *Service) GenerateOfflineNonce(manifestID, driverID string) string {
	h := sha256.New()
	h.Write([]byte(manifestID))
	h.Write([]byte(driverID))
	h.Write([]byte(s.jwtSecret))
	return hex.EncodeToString(h.Sum(nil))
}

func driverIDFromRequest(r *http.Request) string {
	if claims, ok := auth.FromContext(r.Context()); ok {
		if strings.TrimSpace(claims.Subject) != "" {
			return strings.TrimSpace(claims.Subject)
		}
	}
	return strings.TrimSpace(r.URL.Query().Get("driver_id"))
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func driverAvailabilityKey(driverID string) string {
	return "driver:availability:" + driverID
}

func driverManifestKey(driverID string) string {
	return "driver:manifest:" + driverID
}

func manifestGateCleared(state string) bool {
	trimmed := strings.TrimSpace(state)
	return trimmed == "SEALED" || trimmed == "DISPATCHED" || trimmed == "COMPLETED"
}

func (s *Service) broadcastDriverEvent(ctx context.Context, driverID string, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if s.driverHub != nil && driverID != "" {
		s.driverHub.Broadcast(ctx, "driver:"+driverID, raw)
	}
	if s.supplierHub != nil && s.resolveSupplierScope(ctx) != "" {
		s.supplierHub.Broadcast(ctx, "supplier:"+s.resolveSupplierScope(ctx), raw)
	}
}
