// Package driver owns driver-role handlers and local scaffold state.
package driver

import (
	"context"
	"encoding/json"
	"io"
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
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

// DriverNotificationReader provides read access to the notification inbox.
type DriverNotificationReader interface {
	ListForRecipient(ctx context.Context, recipientID string, limit int) ([]any, error)
	MarkRead(ctx context.Context, recipientID string, notificationIDs []string) error
	UnreadCount(ctx context.Context, recipientID string) (int64, error)
}

// DriverOrderView is a single order row projected for the driver fleet surface.
type DriverOrderView struct {
	OrderID         string  `json:"id"`
	RetailerID      string  `json:"retailer_id"`
	RetailerName    string  `json:"retailer_name"`
	Status          string  `json:"state"`
	TotalMinor      int64   `json:"total_amount"`
	DeliveryAddress string  `json:"delivery_address,omitempty"`
	Lat             float64 `json:"latitude"`
	Lng             float64 `json:"longitude"`
	PaymentGateway  string  `json:"payment_gateway"`
	RouteID         string  `json:"route_id,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// DriverOrderQuery lists active orders assigned to a driver from Spanner.
type DriverOrderQuery func(ctx context.Context, driverID string) ([]DriverOrderView, error)

// DriverOrderGetQuery retrieves a single order by ID from Spanner.
type DriverOrderGetQuery func(ctx context.Context, orderID string) (DriverOrderView, bool, error)

// Service keeps additive in-memory driver state for scaffold routes.
type Service struct {
	repo         Repository
	cache        *cache.Cache
	notifSvc     DriverNotificationReader
	orderList    DriverOrderQuery
	orderGet     DriverOrderGetQuery
	supplierHub  *ws.Hub
	driverHub    *ws.Hub
	log          *slog.Logger
	manifestGate ManifestGateLookup
	manifest     ManifestLookup
	pendingQuery PendingCollectionsLookup
	earnings     EarningsLookup
	depart       DepartFn

	supplierID string
	currency   string
	jwtSecret  string
	jwtIssuer  string

	mu                 sync.RWMutex
	availability       map[string]bool
	history            map[string][]HistoryRow
	earningsMinor      map[string]int64
	pendingCollections map[string][]PendingCollection
	now                func() time.Time
}

// ServiceConfig is the constructor input.
type ServiceConfig struct {
	Repo         Repository
	Cache        *cache.Cache
	NotifSvc     DriverNotificationReader
	OrderList    DriverOrderQuery
	OrderGet     DriverOrderGetQuery
	SupplierHub  *ws.Hub
	DriverHub    *ws.Hub
	Log          *slog.Logger
	ManifestGate ManifestGateLookup
	Manifest     ManifestLookup
	PendingQuery PendingCollectionsLookup
	Earnings     EarningsLookup
	Depart       DepartFn
	SupplierID   string
	Currency     string
	JWTSecret    string
	JWTIssuer    string
	Now          func() time.Time
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
	return &Service{
		availability:       make(map[string]bool),
		history:            make(map[string][]HistoryRow),
		earningsMinor:      make(map[string]int64),
		pendingCollections: make(map[string][]PendingCollection),
		repo:               c.Repo,
		cache:              c.Cache,
		notifSvc:           c.NotifSvc,
		orderList:          c.OrderList,
		orderGet:           c.OrderGet,
		supplierHub:        c.SupplierHub,
		driverHub:          c.DriverHub,
		log:                c.Log,
		manifestGate:       c.ManifestGate,
		manifest:           c.Manifest,
		pendingQuery:       c.PendingQuery,
		earnings:           c.Earnings,
		depart:             c.Depart,
		supplierID:         c.SupplierID,
		currency:           strings.ToUpper(strings.TrimSpace(c.Currency)),
		jwtSecret:          strings.TrimSpace(c.JWTSecret),
		jwtIssuer:          strings.TrimSpace(c.JWTIssuer),
		now:                c.Now,
	}
}

type availabilityPatchRequest struct {
	OnShift bool `json:"on_shift"`
}

// HandleProfile serves GET /v1/driver/profile.
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
	s.mu.RLock()
	onShift := s.availability[driverID]
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"driver_id":      driverID,
		"home_node_type": claims.HomeNodeType,
		"home_node_id":   claims.HomeNodeID,
		"on_shift":       onShift,
		"updated_at":     s.now().Format(time.RFC3339Nano),
	})
}

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
	s.mu.RLock()
	rows := append([]HistoryRow(nil), s.history[driverID]...)
	s.mu.RUnlock()
	sort.Slice(rows, func(i, j int) bool { return rows[i].CompletedAt > rows[j].CompletedAt })
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
		s.mu.RLock()
		onShift := s.availability[driverID]
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"driver_id": driverID, "on_shift": onShift, "available": onShift})
	case http.MethodPost:
		var req struct {
			Available bool   `json:"available"`
			Reason    string `json:"reason"`
			Note      string `json:"note"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		defer r.Body.Close()
		r2 := r.Clone(r.Context())
		r2.Method = http.MethodPatch
		r2.Body = http.NoBody
		patchBody, _ := json.Marshal(availabilityPatchRequest{OnShift: req.Available})
		r2.Body = io.NopCloser(strings.NewReader(string(patchBody)))
		r2.ContentLength = int64(len(patchBody))
		s.HandleAvailability(w, r2)
	case http.MethodPatch:
		var req availabilityPatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		defer r.Body.Close()

		s.mu.RLock()
		current, existed := s.availability[driverID]
		s.mu.RUnlock()
		if existed && current == req.OnShift {
			// Idempotent no-op: state already at target. No outbox emit, no cache
			// invalidation, no ws broadcast — mirrors warehouse dispatch-lock
			// missing-release negative-path contract.
			writeJSON(w, http.StatusOK, map[string]any{"driver_id": driverID, "on_shift": req.OnShift, "no_change": true})
			return
		}

		nowTS := s.now().Format(time.RFC3339Nano)
		claims, _ := auth.FromContext(r.Context())
		eventPayload := map[string]any{
			"type":           events.EventDriverAvailabilityChanged,
			"trace_id":       outbox.TraceIDFromContext(r.Context()),
			"timestamp":      nowTS,
			"v":              1,
			"schema_version": 1,
			"driver_id":      driverID,
			"available":      req.OnShift,
			"on_shift":       req.OnShift,
			"supplier_id":    s.supplierID,
			"home_node_type": strings.TrimSpace(string(claims.HomeNodeType)),
			"home_node_id":   strings.TrimSpace(claims.HomeNodeID),
		}

		if err := s.repo.Apply(r.Context(), func() error {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.availability[driverID] = req.OnShift
			return nil
		}, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateDriver, driverID, events.TopicMain, eventPayload)
		}); err != nil {
			s.log.Error("driver availability persist failed", "driver_id", driverID, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_availability_failed"})
			return
		}

		if s.cache != nil {
			s.cache.Invalidate(r.Context(), driverAvailabilityKey(driverID))
		}
		s.broadcastDriverEvent(r.Context(), driverID, eventPayload)
		s.log.Info("driver availability changed", "driver_id", driverID, "on_shift", req.OnShift)
		writeJSON(w, http.StatusOK, map[string]any{"driver_id": driverID, "on_shift": req.OnShift})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
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
			"driver_id":   driverID,
			"manifest_id": gate.ManifestID,
			"state":       gate.State,
			"allowed":     true,
			"cleared":     true,
			"reason":      "ok",
			"stop_count":  gate.StopCount,
			"volume_vu":   gate.VolumeVU,
		})
		return
	}
	writeJSON(w, http.StatusForbidden, map[string]any{
		"driver_id":   driverID,
		"manifest_id": gate.ManifestID,
		"state":       gate.State,
		"allowed":     false,
		"cleared":     false,
		"reason":      events.EventManifestSealed,
		"error":       "AWAITING_PAYLOAD_SEAL",
		"message":     "Manifest is in " + gate.State + " state. Wait for Payloader to complete loading and seal.",
	})
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
		"hashes":                  []string{},
		"legacy_hashes_available": false,
	})
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

func (s *Service) broadcastDriverEvent(ctx context.Context, driverID string, payload map[string]any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if s.driverHub != nil && driverID != "" {
		s.driverHub.Broadcast(ctx, "driver:"+driverID, raw)
	}
	if s.supplierHub != nil && s.supplierID != "" {
		s.supplierHub.Broadcast(ctx, "supplier:"+s.supplierID, raw)
	}
}
