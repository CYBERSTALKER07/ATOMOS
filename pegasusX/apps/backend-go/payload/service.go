package payload

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/drivercmd"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"google.golang.org/api/iterator"
)

const (
	payloadManifestStateDraft      = "DRAFT"
	payloadManifestStateLoading    = "LOADING"
	payloadManifestStateSealed     = "SEALED"
	payloadManifestStateDispatched = "DISPATCHED"

	payloadExceptionEscalationThreshold = 3
)

// Service stores additive in-memory payload operations state.
//
// Production note: Spanner (SupplierTruckManifests + manifest.Store) is the source of
// truth after warehouse dispatch execute. The in-memory overlay below accelerates
// payload-terminal dev/SSMR flows when Spanner is partial. For production hardening:
//   - Persist every seal/reassign/exception through manifest.Store only
//   - Remove dual-write paths once payload terminal reads Spanner exclusively
//   - Keep this overlay behind PAYLOAD_DEV_OVERLAY=true for Docker simulation only
type Service struct {
	repo        Repository
	cache       *cache.Cache
	supplierHub *ws.Hub
	payloadHub  *ws.Hub
	driverHub   *ws.Hub
	notifSvc    *notifications.Service
	log         *slog.Logger
	idem        idempotency.Store

	seedSupplierID   string
	currency         string
	jwtSecret        string
	jwtIssuer        string
	now              func() time.Time
	firebaseVerifier auth.FirebaseVerifier
	staffLookup      PayloadStaffLookup

	mu             sync.RWMutex
	spannerLoaded  bool
	trucks         []TruckRow
	orders         []OrderRow
	manifests      []ManifestRow
	manifestOrders map[string][]ManifestOrder
	shipUnits      map[string][]ShipUnit
	overflowCount  map[string]int64
	exceptions     []ManifestException
	reassignments  []Reassignment
	seq            int64

	portalLister  PortalManifestLister
	manifestStore *manifest.Store
	orderReader   OrderExpectationReader
	locations     telemetry.LastLocationReader
}

// ServiceConfig is the constructor input.
type ServiceConfig struct {
	Repo        Repository
	Cache       *cache.Cache
	SupplierHub *ws.Hub
	PayloadHub  *ws.Hub
	DriverHub   *ws.Hub
	NotifSvc    *notifications.Service
	Log         *slog.Logger

	// SeedSupplierID is bootstrap/fixture fallback only (Gate 5 Week 11).
	SeedSupplierID string
	// SupplierID is deprecated; use SeedSupplierID.
	SupplierID       string
	Currency         string
	JWTSecret        string
	JWTIssuer        string
	Now              func() time.Time
	FirebaseVerifier auth.FirebaseVerifier
	ManifestStore    *manifest.Store
	Idem             idempotency.Store
	Locations        telemetry.LastLocationReader
}

type payloaderTruckWire struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	LicensePlate string `json:"license_plate"`
	VehicleClass string `json:"vehicle_class"`
	// TruckStatus is the current open manifest state (DRAFT/LOADING/SEALED/DISPATCHED).
	// Empty when the vehicle has no board-state manifest — do not invent DRAFT.
	TruckStatus  string `json:"truck_status,omitempty"`
	UsedVolumeVU int64  `json:"used_volume_vu,omitempty"`
	MaxVolumeVU  int64  `json:"max_volume_vu,omitempty"`
	StopCount    int    `json:"stop_count,omitempty"`
}

type liveOrderLineWire struct {
	SKUID       string `json:"sku_id"`
	ProductName string `json:"product_name"`
	Quantity    int64  `json:"quantity"`
}

type liveOrderWire struct {
	OrderID        string              `json:"order_id"`
	RetailerID     string              `json:"retailer_id,omitempty"`
	Amount         int64               `json:"amount,omitempty"`
	PaymentGateway string              `json:"payment_gateway,omitempty"`
	State          string              `json:"state"`
	RouteID        string              `json:"route_id,omitempty"`
	WarehouseID    string              `json:"warehouse_id,omitempty"`
	Items          []liveOrderLineWire `json:"items,omitempty"`
}

// TruckRow represents one payloader-visible truck row.
type TruckRow struct {
	VehicleID string `json:"vehicle_id"`
	PlateNo   string `json:"plate_no"`
	State     string `json:"state"`
}

// OrderRow represents one payloader-visible order row.
type OrderRow struct {
	OrderID          string  `json:"order_id"`
	Status           string  `json:"status"`
	TotalMinor       int64   `json:"total_minor"`
	Currency         string  `json:"currency"`
	ManifestID       string  `json:"manifest_id,omitempty"`
	RouteID          string  `json:"route_id,omitempty"`
	VehicleID        string  `json:"vehicle_id,omitempty"`
	DispatchPriority int     `json:"dispatch_priority,omitempty"`
	OverflowCount    int64   `json:"overflow_count,omitempty"`
	ReassignDepth    int     `json:"reassign_depth,omitempty"`
	SplitGroupID     string  `json:"split_group_id,omitempty"`
	Lat              float64 `json:"lat,omitempty"`
	Lng              float64 `json:"lng,omitempty"`
	VolumeVU         int64   `json:"volume_vu,omitempty"`
	UpdatedAt        string  `json:"updated_at"`
}

// ManifestRow represents one payloader-visible manifest.
type ManifestRow struct {
	ManifestID    string `json:"manifest_id"`
	VehicleID     string `json:"vehicle_id,omitempty"`
	DriverID      string `json:"driver_id,omitempty"`
	State         string `json:"state"`
	TotalVolumeVU int64  `json:"total_volume_vu"`
	MaxVolumeVU   int64  `json:"max_volume_vu"`
	StopCount     int    `json:"stop_count"`
	// WarehouseID is stamped when known (JWT home node / Spanner). B7 PL-P0-6 scope.
	WarehouseID      string `json:"warehouse_id,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	LoadingStartedAt string `json:"loading_started_at,omitempty"`
	SealedAt         string `json:"sealed_at,omitempty"`
}

// ManifestOrder is one order assignment in a manifest.
type ManifestOrder struct {
	ManifestID   string `json:"manifest_id"`
	OrderID      string `json:"order_id"`
	State        string `json:"state"`
	VolumeVU     int64  `json:"volume_vu"`
	Reason       string `json:"reason,omitempty"`
	SplitGroupID string `json:"split_group_id,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// ManifestException tracks order removal exceptions during loading.
type ManifestException struct {
	ExceptionID  string `json:"exception_id"`
	ManifestID   string `json:"manifest_id"`
	OrderID      string `json:"order_id"`
	Reason       string `json:"reason"`
	Metadata     string `json:"metadata,omitempty"`
	AttemptCount int64  `json:"attempt_count"`
	Escalated    bool   `json:"escalated"`
	CreatedAt    string `json:"created_at"`
}

// ReassignRecommendation models one target route candidate.
type ReassignRecommendation struct {
	OrderID    string  `json:"order_id"`
	FromRoute  string  `json:"from_route_id,omitempty"`
	ToRoute    string  `json:"to_route_id"`
	ToDriverID string  `json:"to_driver_id,omitempty"`
	Score      float64 `json:"score"`
	Reason     string  `json:"reason"`
}

// Reassignment tracks one applied reassignment.
type Reassignment struct {
	OrderID        string `json:"order_id"`
	FromRouteID    string `json:"from_route_id,omitempty"`
	ToRouteID      string `json:"to_route_id"`
	FromManifestID string `json:"from_manifest_id,omitempty"`
	ManifestID     string `json:"manifest_id,omitempty"`
	ToDriverID     string `json:"to_driver_id,omitempty"`
	Reason         string `json:"reason,omitempty"`
	Depth          int    `json:"depth"`
	AppliedAt      string `json:"applied_at"`
}

// NewService constructs the payload service.
func NewService(c ServiceConfig) *Service {
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	if c.Log == nil {
		c.Log = slog.Default()
	}
	if c.Repo == nil {
		panic("repo is required")
	}
	seedID := strings.TrimSpace(c.SeedSupplierID)
	if seedID == "" {
		seedID = strings.TrimSpace(c.SupplierID)
	}
	if c.Currency == "" {
		if cur, err := auth.CurrencyFromContext(context.Background(), seedID); err == nil {
			c.Currency = cur
		}
	}
	return &Service{
		repo:             c.Repo,
		cache:            c.Cache,
		supplierHub:      c.SupplierHub,
		payloadHub:       c.PayloadHub,
		driverHub:        c.DriverHub,
		notifSvc:         c.NotifSvc,
		log:              c.Log,
		idem:             c.Idem,
		seedSupplierID:   seedID,
		currency:         c.Currency,
		jwtSecret:        c.JWTSecret,
		jwtIssuer:        c.JWTIssuer,
		now:              c.Now,
		firebaseVerifier: c.FirebaseVerifier,
		manifestStore:    c.ManifestStore,
		locations:        c.Locations,
		manifestOrders:   make(map[string][]ManifestOrder),
		overflowCount:    make(map[string]int64),
	}
}

// resolveSupplierScope prefers request TenantContext over the bootstrap seed.
// AuthIssuer returns the JWT iss claim used when minting role tokens / WS tickets.
func (s *Service) AuthIssuer() string {
	if s == nil {
		return ""
	}
	return s.jwtIssuer
}

// resolveWarehouseScope prefers JWT home-node warehouse for PAYLOAD role (B2 M-P1-9 / B7 PL-P0-6).
func (s *Service) resolveWarehouseScope(ctx context.Context) string {
	if claims, ok := auth.FromContext(ctx); ok {
		if claims.HomeNodeType == auth.HomeNodeWarehouse && strings.TrimSpace(claims.HomeNodeID) != "" {
			return strings.TrimSpace(claims.HomeNodeID)
		}
		if claims.Role == auth.RolePayload && strings.TrimSpace(claims.HomeNodeID) != "" {
			return strings.TrimSpace(claims.HomeNodeID)
		}
	}
	return ""
}

// manifestMatchesWarehouseScope allows historical rows with empty warehouse when JWT is set
// only if both non-empty and equal; mismatch with both set is forbidden (B7 PL-P0-6).
func manifestMatchesWarehouseScope(rowWarehouseID, scopeWarehouseID string) bool {
	scope := strings.TrimSpace(scopeWarehouseID)
	if scope == "" {
		return true
	}
	row := strings.TrimSpace(rowWarehouseID)
	if row == "" {
		// Historical manifests without WarehouseId remain visible until backfilled.
		return true
	}
	return row == scope
}

// assertManifestWarehouseScope rejects mutate when JWT warehouse and row warehouse both set and differ.
func (s *Service) assertManifestWarehouseScope(ctx context.Context, m ManifestRow) error {
	scope := s.resolveWarehouseScope(ctx)
	if scope == "" {
		return nil
	}
	row := strings.TrimSpace(m.WarehouseID)
	if row != "" && row != scope {
		return errManifestWarehouseScope
	}
	return nil
}

// errManifestWarehouseScope is returned when a payload mutate targets a foreign warehouse.
var errManifestWarehouseScope = errString("warehouse_scope_forbidden")

type errString string

func (e errString) Error() string { return string(e) }

func (s *Service) resolveSupplierScope(ctx context.Context) string {
	return auth.PreferTenantSupplierID(ctx, s.seedSupplierID)
}

type recommendReassignRequest struct {
	OrderID string `json:"order_id"`
}

type injectOrderRequest struct {
	OrderID  string `json:"order_id"`
	VolumeVU int64  `json:"volume_vu"`
}

type manifestExceptionRequest struct {
	ManifestID string `json:"manifest_id"`
	OrderID    string `json:"order_id"`
	Reason     string `json:"reason"`
	Metadata   string `json:"metadata"`
}

type applyReassignRequest struct {
	OrderID         string `json:"order_id"`
	ToRouteID       string `json:"to_route_id"`
	ToManifestID    string `json:"to_manifest_id"`
	ToDriverID      string `json:"to_driver_id"`
	Reason          string `json:"reason"`
	IsPartial       bool   `json:"is_partial"`
	PartialVolumeVU int64  `json:"partial_volume_vu"`
}

type sealRequest struct {
	OrderID    string `json:"order_id"`
	ManifestID string `json:"manifest_id"`
}

func (s *Service) nextIDLocked(prefix string) string {
	s.seq++
	return prefix + "_" + strconv.FormatInt(s.now().UnixNano(), 10) + "_" + strconv.FormatInt(s.seq, 10)
}

func routeIDForManifest(m ManifestRow) string {
	if m.VehicleID != "" {
		return "route_" + m.VehicleID
	}
	return "route_" + m.ManifestID
}

func (s *Service) driverIDForRouteLocked(routeID string) string {
	routeID = strings.TrimSpace(routeID)
	if routeID == "" {
		return ""
	}
	for i := range s.manifests {
		if routeIDForManifest(s.manifests[i]) == routeID {
			return strings.TrimSpace(s.manifests[i].DriverID)
		}
	}
	return ""
}

// portalSeedEnabled gates in-memory demo seed data. Disabled when Spanner is wired
// unless PAYLOAD_PORTAL_SEED=true is set explicitly for local scaffold runs.
func (s *Service) portalSeedEnabled() bool {
	if _, ok := s.repo.(*SpannerRepository); ok {
		switch strings.ToLower(strings.TrimSpace(os.Getenv("PAYLOAD_PORTAL_SEED"))) {
		case "1", "true", "yes":
			return true
		default:
			return false
		}
	}
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PAYLOAD_PORTAL_SEED")))
	return v == "1" || v == "true" || v == "yes"
}

func (s *Service) ensureDemoDataLocked() {
	if !s.portalSeedEnabled() {
		return
	}
	if len(s.trucks) == 0 {
		s.trucks = []TruckRow{
			{VehicleID: "veh_payload_1", PlateNo: "01P111AA", State: "READY"},
			{VehicleID: "veh_payload_2", PlateNo: "01P222AA", State: "READY"},
			{VehicleID: "veh_payload_3", PlateNo: "01P333AA", State: "READY"},
		}
	}
	if len(s.orders) == 0 {
		now := s.now().Format(time.RFC3339Nano)
		s.orders = []OrderRow{
			{OrderID: "ord_payload_1", Status: "PENDING", TotalMinor: 120000, Currency: s.currency, Lat: 41.3111, Lng: 69.2797, UpdatedAt: now},
			{OrderID: "ord_payload_2", Status: "PENDING", TotalMinor: 98000, Currency: s.currency, Lat: 41.3150, Lng: 69.2850, UpdatedAt: now},
			{OrderID: "ord_payload_3", Status: "PENDING", TotalMinor: 143000, Currency: s.currency, Lat: 41.3200, Lng: 69.2900, UpdatedAt: now},
		}
	}
	if len(s.manifests) == 0 {
		now := s.now().Format(time.RFC3339Nano)
		manifest := ManifestRow{
			ManifestID:    "mf_payload_1",
			VehicleID:     "veh_payload_1",
			DriverID:      "drv_payload_1",
			State:         payloadManifestStateDraft,
			TotalVolumeVU: 75,
			MaxVolumeVU:   140,
			StopCount:     2,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		s.manifests = append(s.manifests, manifest)
		s.manifestOrders[manifest.ManifestID] = []ManifestOrder{
			{ManifestID: manifest.ManifestID, OrderID: "ord_payload_1", State: "ASSIGNED", VolumeVU: 41, UpdatedAt: now},
			{ManifestID: manifest.ManifestID, OrderID: "ord_payload_2", State: "ASSIGNED", VolumeVU: 34, UpdatedAt: now},
		}
		for i := range s.orders {
			if s.orders[i].OrderID == "ord_payload_1" || s.orders[i].OrderID == "ord_payload_2" {
				s.orders[i].ManifestID = manifest.ManifestID
				s.orders[i].VehicleID = manifest.VehicleID
				s.orders[i].RouteID = routeIDForManifest(manifest)
				s.orders[i].UpdatedAt = now
			}
		}
	}
	if s.findManifestIndexLocked("mf_payload_2") < 0 {
		now := s.now().Format(time.RFC3339Nano)
		sibling := ManifestRow{
			ManifestID:    "mf_payload_2",
			VehicleID:     "veh_payload_2",
			DriverID:      "drv_payload_2",
			State:         payloadManifestStateDraft,
			TotalVolumeVU: 0,
			MaxVolumeVU:   140,
			StopCount:     0,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		s.manifests = append(s.manifests, sibling)
		if s.manifestOrders == nil {
			s.manifestOrders = make(map[string][]ManifestOrder)
		}
		s.manifestOrders[sibling.ManifestID] = []ManifestOrder{}
	}
}

func (s *Service) findManifestIndexLocked(manifestID string) int {
	for i := range s.manifests {
		if s.manifests[i].ManifestID == manifestID {
			return i
		}
	}
	return -1
}

func (s *Service) findOrderIndexLocked(orderID string) int {
	for i := range s.orders {
		if s.orders[i].OrderID == orderID {
			return i
		}
	}
	return -1
}

func (s *Service) findManifestOrderIndexLocked(manifestID, orderID string) int {
	orders := s.manifestOrders[manifestID]
	for i := range orders {
		if orders[i].OrderID == orderID {
			return i
		}
	}
	return -1
}

// assertPickWaveReadyForSeal enforces §8.7 Wave 1B seal gate when enabled.
// Soft-warn mode returns nil and stashes warning on context via side channel is not used;
// callers should use assertPickWaveReadyForSealResult.
func (s *Service) assertPickWaveReadyForSeal(ctx context.Context, manifestID string) error {
	_, err := s.assertPickWaveReadyForSealResult(ctx, manifestID)
	return err
}

func (s *Service) assertPickWaveReadyForSealResult(ctx context.Context, manifestID string) (warn string, err error) {
	return stocklots.AssertManifestPickReady(ctx, s.spannerClient(), manifestID)
}

// assertPhysicalSealGates runs G2 pick + load ledger + cold baseline before seal.
func (s *Service) assertPhysicalSealGates(ctx context.Context, manifestID string) (warn string, err error) {
	warn, err = s.assertPickWaveReadyForSealResult(ctx, manifestID)
	if err != nil {
		return warn, err
	}
	if err := stocklots.AssertLoadLedgerReady(ctx, s.spannerClient(), manifestID); err != nil {
		return warn, err
	}
	if err := stocklots.AssertColdChainBaselineReady(ctx, s.spannerClient(), manifestID); err != nil {
		return warn, err
	}
	return warn, nil
}

func (s *Service) sealManifestLocked(manifestID string) (ManifestRow, error) {
	idx := s.findManifestIndexLocked(manifestID)
	if idx < 0 {
		return ManifestRow{}, http.ErrMissingFile
	}
	manifest := s.manifests[idx]
	if manifest.State != payloadManifestStateLoading && manifest.State != payloadManifestStateDraft {
		return ManifestRow{}, http.ErrBodyNotAllowed
	}
	now := s.now().Format(time.RFC3339Nano)
	if manifest.State == payloadManifestStateDraft {
		manifest.LoadingStartedAt = now
	}
	manifest.State = payloadManifestStateSealed
	manifest.SealedAt = now
	manifest.UpdatedAt = now
	s.manifests[idx] = manifest

	orders := s.manifestOrders[manifestID]
	for i := range orders {
		if orders[i].State == "ASSIGNED" || orders[i].State == "LOADED" {
			orders[i].State = "SEALED"
			orders[i].UpdatedAt = now
		}
		oIdx := s.findOrderIndexLocked(orders[i].OrderID)
		if oIdx >= 0 {
			s.orders[oIdx].Status = "DISPATCHED"
			s.orders[oIdx].ManifestID = manifestID
			s.orders[oIdx].RouteID = routeIDForManifest(manifest)
			s.orders[oIdx].VehicleID = manifest.VehicleID
			s.orders[oIdx].UpdatedAt = now
		}
	}
	s.manifestOrders[manifestID] = orders
	return manifest, nil
}

type wsEnvelope struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Data      any    `json:"data"`
}

func (s *Service) broadcastPayloadEvent(ctx context.Context, eventType string, data map[string]any) {
	ts := s.now().Format(time.RFC3339Nano)
	manifestID, _ := data["manifest_id"].(string)
	syncFrame, err := json.Marshal(map[string]any{
		"type":        "PAYLOAD_SYNC",
		"channel":     "SYNC",
		"manifest_id": manifestID,
		"timestamp":   ts,
	})
	if err != nil {
		s.log.Warn("payload ws sync marshal failed", "event_type", eventType, "err", err)
		return
	}
	title := strings.ReplaceAll(eventType, "_", " ")
	notifyFrame, notifyErr := json.Marshal(map[string]any{
		"type":        eventType,
		"title":       title,
		"body":        title,
		"channel":     "PUSH",
		"timestamp":   ts,
		"manifest_id": manifestID,
	})
	legacyEnvelope, legacyErr := json.Marshal(map[string]any{
		"type":      eventType,
		"timestamp": ts,
		"data":      data,
	})
	if s.payloadHub != nil && s.resolveSupplierScope(ctx) != "" {
		s.payloadHub.Broadcast(ctx, "payload:"+s.resolveSupplierScope(ctx), syncFrame)
		if notifyErr == nil {
			s.payloadHub.Broadcast(ctx, "payload:"+s.resolveSupplierScope(ctx), notifyFrame)
		}
	}
	if s.supplierHub != nil && s.resolveSupplierScope(ctx) != "" {
		s.supplierHub.Broadcast(ctx, "supplier:"+s.resolveSupplierScope(ctx), syncFrame)
		if notifyErr == nil {
			s.supplierHub.Broadcast(ctx, "supplier:"+s.resolveSupplierScope(ctx), notifyFrame)
		}
		if legacyErr == nil {
			s.supplierHub.Broadcast(ctx, "supplier:"+s.resolveSupplierScope(ctx), legacyEnvelope)
		}
	}
}

func (s *Service) broadcastDriverEvent(ctx context.Context, driverID string, data map[string]any) {
	if s.driverHub == nil {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"type":      data["type"],
		"timestamp": s.now().UTC().Format(time.RFC3339Nano),
		"data":      data,
	})
	if err != nil {
		s.log.Warn("driver ws marshal failed", "err", err)
		return
	}
	payload = drivercmd.PrepareAndRecordJSON(ctx, s.spannerClient(), driverID, payload)
	s.driverHub.Broadcast(ctx, "driver:"+driverID, payload)
}

func (s *Service) invalidatePayloadKeys(ctx context.Context, keys ...string) {
	if s.cache == nil {
		return
	}
	if sid := strings.TrimSpace(s.resolveSupplierScope(ctx)); sid != "" {
		keys = append(keys, cache.DashboardKey("supplier", sid))
	}
	if len(keys) == 0 {
		return
	}
	s.cache.Invalidate(ctx, keys...)
}

func payloadManifestKey(manifestID string) string {
	return "payload:manifest:" + manifestID
}

func payloadManifestListKey(supplierID string) string {
	return "payload:manifests:" + supplierID
}

func payloadOrderListKey(supplierID string) string {
	return "payload:orders:" + supplierID
}

func payloadExceptionListKey(supplierID string) string {
	return "payload:manifest_exceptions:" + supplierID
}

// HandleTrucks serves GET /v1/payloader/trucks.
// truck_status + VU come from the vehicle's current board-state manifest, never GET /v1/payload/capacity/{id}.
func (s *Service) HandleTrucks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	_ = s.hydrateFromRepo(r.Context())
	s.mu.Lock()
	s.ensureManifestStateLocked()
	rows := append([]TruckRow(nil), s.trucks...)
	wire := make([]payloaderTruckWire, len(rows))
	for i := range rows {
		wire[i] = truckWireFromRowLocked(s, rows[i])
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, wire)
}

// HandleOrders serves GET /v1/payloader/orders.
func (s *Service) HandleOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	stateFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("state")))
	manifestFilter := strings.TrimSpace(r.URL.Query().Get("manifest_id"))
	vehicleFilter := strings.TrimSpace(r.URL.Query().Get("vehicle_id"))

	_ = s.hydrateFromRepo(r.Context())
	s.mu.Lock()
	s.ensureManifestStateLocked()
	rows := append([]OrderRow(nil), s.orders...)
	lineItemsByOrder := s.orderLineItemsLocked()
	s.mu.Unlock()

	if stateFilter != "" || manifestFilter != "" || vehicleFilter != "" {
		filtered := make([]OrderRow, 0, len(rows))
		for i := range rows {
			if stateFilter != "" && strings.ToUpper(rows[i].Status) != stateFilter {
				continue
			}
			if manifestFilter != "" && rows[i].ManifestID != manifestFilter {
				continue
			}
			if vehicleFilter != "" && rows[i].VehicleID != vehicleFilter {
				continue
			}
			filtered = append(filtered, rows[i])
		}
		rows = filtered
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].UpdatedAt > rows[j].UpdatedAt })
	wire := make([]liveOrderWire, len(rows))
	for i := range rows {
		wire[i] = liveOrderWire{
			OrderID: rows[i].OrderID,
			Amount:  rows[i].TotalMinor,
			State:   rows[i].Status,
			RouteID: rows[i].RouteID,
			Items:   lineItemsByOrder[rows[i].OrderID],
		}
	}
	writeJSON(w, http.StatusOK, wire)
}

func (s *Service) orderLineItemsLocked() map[string][]liveOrderLineWire {
	out := make(map[string][]liveOrderLineWire)
	if s.repo == nil {
		return out
	}
	spannerRepo, ok := s.repo.(*SpannerRepository)
	if !ok || spannerRepo.client == nil {
		return out
	}
	orderIDs := make([]string, 0, len(s.orders))
	for _, row := range s.orders {
		if row.OrderID != "" {
			orderIDs = append(orderIDs, row.OrderID)
		}
	}
	if len(orderIDs) == 0 {
		for _, rows := range s.manifestOrders {
			for _, mo := range rows {
				orderIDs = append(orderIDs, mo.OrderID)
			}
		}
	}
	if len(orderIDs) == 0 {
		return out
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stmt := spanner.Statement{
		SQL:    `SELECT OrderId, LineItemsJson FROM Orders WHERE OrderId IN UNNEST(@oids)`,
		Params: map[string]interface{}{"oids": orderIDs},
	}
	iter := spannerRepo.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		var orderID string
		var raw []byte
		if err := row.Columns(&orderID, &raw); err != nil {
			continue
		}
		out[orderID] = decodeLiveOrderLineItems(raw)
	}
	return out
}

func decodeLiveOrderLineItems(raw []byte) []liveOrderLineWire {
	if len(raw) == 0 {
		return nil
	}
	var source []struct {
		SKU      string `json:"sku"`
		SKUID    string `json:"sku_id"`
		Name     string `json:"name"`
		Quantity int64  `json:"quantity"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil
	}
	items := make([]liveOrderLineWire, 0, len(source))
	for _, item := range source {
		sku := item.SKU
		if sku == "" {
			sku = item.SKUID
		}
		items = append(items, liveOrderLineWire{
			SKUID:       sku,
			ProductName: item.Name,
			Quantity:    item.Quantity,
		})
	}
	return items
}

// HandleStartLoading serves POST /v1/payloader/manifests/{manifestID}/start-loading.
func (s *Service) HandleStartLoading(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, err := readLimitedBody(r, 8*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	manifestID := manifestIDParam(r)
	if manifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}

	// B7 PL-P0-6: pin start-loading to JWT warehouse when row warehouse is set.
	_ = s.hydrateFromRepo(r.Context())
	s.mu.Lock()
	s.ensureManifestStateLocked()
	preIdx := s.findManifestIndexLocked(manifestID)
	var preRow ManifestRow
	if preIdx >= 0 {
		preRow = s.manifests[preIdx]
	}
	s.mu.Unlock()
	if preIdx >= 0 {
		if scopeErr := s.assertManifestWarehouseScope(r.Context(), preRow); scopeErr != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_forbidden"})
			return
		}
	}

	var manifest ManifestRow
	now := ""
	err = s.apply(r.Context(), func(_ PayloadTx) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		idx := s.findManifestIndexLocked(manifestID)
		if idx < 0 {
			return fmt.Errorf("manifest_not_found")
		}
		manifest = s.manifests[idx]
		if manifest.State != payloadManifestStateDraft {
			return fmt.Errorf("manifest_not_draft")
		}
		now = s.now().Format(time.RFC3339Nano)
		manifest.State = payloadManifestStateLoading
		manifest.LoadingStartedAt = now
		manifest.UpdatedAt = now
		s.manifests[idx] = manifest

		manifestRows := s.manifestOrders[manifestID]
		for i := range manifestRows {
			if manifestRows[i].State == "ASSIGNED" {
				manifestRows[i].State = "LOADED"
				manifestRows[i].UpdatedAt = now
			}
			oIdx := s.findOrderIndexLocked(manifestRows[i].OrderID)
			if oIdx >= 0 {
				s.orders[oIdx].Status = "LOADED"
				s.orders[oIdx].ManifestID = manifestID
				s.orders[oIdx].VehicleID = manifest.VehicleID
				s.orders[oIdx].RouteID = routeIDForManifest(manifest)
				s.orders[oIdx].UpdatedAt = now
			}
		}
		s.manifestOrders[manifestID] = manifestRows
		return nil
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, manifestID, events.TopicMain, events.ManifestEvent{
			ManifestDomain: events.ManifestDomainSupplier,
			BaseEvent:      events.BaseEvent{Type: events.EventManifestLoadingStarted, Timestamp: now},
			ManifestID:     manifestID,
			SupplierID:     s.resolveSupplierScope(r.Context()),
			State:          payloadManifestStateLoading,
		})
	})
	if err != nil {
		switch err.Error() {
		case "manifest_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
			return
		case "manifest_not_draft":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "manifest_not_draft"})
			return
		default:
			s.log.ErrorContext(r.Context(), "payload.start_loading_failed", "err", err, "manifest_id", manifestID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "start_loading_failed"})
			return
		}
	}

	// G2.B: seed durable load ledger from order lines (memory + optional Spanner later in apply).
	s.seedLoadLedgerForManifest(r.Context(), manifestID)

	s.invalidatePayloadKeys(r.Context(), payloadManifestKey(manifestID), payloadManifestListKey(s.resolveSupplierScope(r.Context())), payloadOrderListKey(s.resolveSupplierScope(r.Context())))
	s.broadcastPayloadEvent(r.Context(), events.EventManifestLoadingStarted, map[string]any{
		"manifest_id": manifestID,
		"state":       payloadManifestStateLoading,
		"updated_at":  now,
	})

	resp := map[string]any{
		"status":      "loading_started",
		"manifest_id": manifestID,
		"state":       payloadManifestStateLoading,
		"updated_at":  now,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleInjectOrder serves POST /v1/payloader/manifests/{manifestID}/inject-order.
func (s *Service) HandleInjectOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, err := readLimitedBody(r, 32*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	manifestID := manifestIDParam(r)
	if manifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}

	var req injectOrderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	if req.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}
	if req.VolumeVU <= 0 {
		req.VolumeVU = 12
	}

	var manifest ManifestRow
	now := ""
	err = s.apply(r.Context(), func(_ PayloadTx) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		mIdx := s.findManifestIndexLocked(manifestID)
		if mIdx < 0 {
			return fmt.Errorf("manifest_not_found")
		}
		manifest = s.manifests[mIdx]
		if manifest.State != payloadManifestStateLoading {
			return fmt.Errorf("manifest_not_loading")
		}
		if manifest.TotalVolumeVU+req.VolumeVU > manifest.MaxVolumeVU {
			return fmt.Errorf("volume_conflict")
		}

		oIdx := s.findOrderIndexLocked(req.OrderID)
		if oIdx < 0 {
			now = s.now().Format(time.RFC3339Nano)
			s.orders = append(s.orders, OrderRow{OrderID: req.OrderID, Status: "PENDING", TotalMinor: 0, Currency: s.currency, UpdatedAt: now})
			oIdx = len(s.orders) - 1
		}
		if s.orders[oIdx].ManifestID != "" && s.orders[oIdx].ManifestID != manifestID {
			return fmt.Errorf("order_already_assigned")
		}

		now = s.now().Format(time.RFC3339Nano)
		manifestOrders := s.manifestOrders[manifestID]
		if s.findManifestOrderIndexLocked(manifestID, req.OrderID) < 0 {
			manifestOrders = append(manifestOrders, ManifestOrder{
				ManifestID: manifestID,
				OrderID:    req.OrderID,
				State:      "ASSIGNED",
				VolumeVU:   req.VolumeVU,
				UpdatedAt:  now,
			})
			manifest.StopCount++
			manifest.TotalVolumeVU += req.VolumeVU
		}
		s.manifestOrders[manifestID] = manifestOrders

		s.orders[oIdx].ManifestID = manifestID
		s.orders[oIdx].Status = "LOADED"
		s.orders[oIdx].VehicleID = manifest.VehicleID
		s.orders[oIdx].RouteID = routeIDForManifest(manifest)
		s.orders[oIdx].UpdatedAt = now

		manifest.UpdatedAt = now
		s.manifests[mIdx] = manifest
		return nil
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, manifestID, events.TopicMain, events.ManifestEvent{
			ManifestDomain: events.ManifestDomainSupplier,
			BaseEvent:      events.BaseEvent{Type: events.EventManifestOrderInjected, Timestamp: now},
			ManifestID:     manifestID,
			OrderID:        req.OrderID,
			SupplierID:     s.resolveSupplierScope(r.Context()),
			TotalVolumeVU:  manifest.TotalVolumeVU,
			StopCount:      int64(manifest.StopCount),
		})
	})
	if err != nil {
		switch err.Error() {
		case "manifest_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
			return
		case "manifest_not_loading":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "manifest_not_loading"})
			return
		case "volume_conflict":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "volume_conflict"})
			return
		case "order_already_assigned":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "order_already_assigned"})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "inject_order_failed"})
			return
		}
	}

	s.invalidatePayloadKeys(r.Context(), payloadManifestKey(manifestID), payloadManifestListKey(s.resolveSupplierScope(r.Context())), payloadOrderListKey(s.resolveSupplierScope(r.Context())))
	s.broadcastPayloadEvent(r.Context(), events.EventManifestOrderInjected, map[string]any{
		"manifest_id":     manifestID,
		"order_id":        req.OrderID,
		"total_volume_vu": manifest.TotalVolumeVU,
		"stop_count":      manifest.StopCount,
	})

	resp := map[string]any{
		"status":          "order_injected",
		"manifest_id":     manifestID,
		"order_id":        req.OrderID,
		"total_volume_vu": manifest.TotalVolumeVU,
		"stop_count":      manifest.StopCount,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleManifestException serves POST /v1/payload/manifest-exception.
