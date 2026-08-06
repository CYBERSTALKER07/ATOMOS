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
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
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

	supplierID       string
	currency         string
	jwtSecret        string
	jwtIssuer        string
	now              func() time.Time
	firebaseVerifier auth.FirebaseVerifier

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

	SupplierID       string
	Currency         string
	JWTSecret        string
	JWTIssuer        string
	Now              func() time.Time
	FirebaseVerifier auth.FirebaseVerifier
	ManifestStore    *manifest.Store
	Idem               idempotency.Store
	Locations          telemetry.LastLocationReader
}

type payloaderTruckWire struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	LicensePlate string `json:"license_plate"`
	VehicleClass string `json:"vehicle_class"`
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
	OrderID          string `json:"order_id"`
	Status           string `json:"status"`
	TotalMinor       int64  `json:"total_minor"`
	Currency         string `json:"currency"`
	ManifestID       string `json:"manifest_id,omitempty"`
	RouteID          string `json:"route_id,omitempty"`
	VehicleID        string `json:"vehicle_id,omitempty"`
	DispatchPriority int    `json:"dispatch_priority,omitempty"`
	OverflowCount    int64  `json:"overflow_count,omitempty"`
	ReassignDepth    int    `json:"reassign_depth,omitempty"`
	SplitGroupID     string `json:"split_group_id,omitempty"`
	UpdatedAt        string `json:"updated_at"`
}

// ManifestRow represents one payloader-visible manifest.
type ManifestRow struct {
	ManifestID       string `json:"manifest_id"`
	VehicleID        string `json:"vehicle_id,omitempty"`
	DriverID         string `json:"driver_id,omitempty"`
	State            string `json:"state"`
	TotalVolumeVU    int64  `json:"total_volume_vu"`
	MaxVolumeVU      int64  `json:"max_volume_vu"`
	StopCount        int    `json:"stop_count"`
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
	if c.Currency == "" {
		c.Currency = "UZS"
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
		supplierID:       c.SupplierID,
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
	OrderID      string `json:"order_id"`
	ToRouteID    string `json:"to_route_id"`
	ToManifestID string `json:"to_manifest_id"`
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
			{OrderID: "ord_payload_1", Status: "PENDING", TotalMinor: 120000, Currency: s.currency, UpdatedAt: now},
			{OrderID: "ord_payload_2", Status: "PENDING", TotalMinor: 98000, Currency: s.currency, UpdatedAt: now},
			{OrderID: "ord_payload_3", Status: "PENDING", TotalMinor: 143000, Currency: s.currency, UpdatedAt: now},
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
	if s.payloadHub != nil && s.supplierID != "" {
		s.payloadHub.Broadcast(ctx, "payload:"+s.supplierID, syncFrame)
		if notifyErr == nil {
			s.payloadHub.Broadcast(ctx, "payload:"+s.supplierID, notifyFrame)
		}
	}
	if s.supplierHub != nil && s.supplierID != "" {
		s.supplierHub.Broadcast(ctx, "supplier:"+s.supplierID, syncFrame)
		if notifyErr == nil {
			s.supplierHub.Broadcast(ctx, "supplier:"+s.supplierID, notifyFrame)
		}
		if legacyErr == nil {
			s.supplierHub.Broadcast(ctx, "supplier:"+s.supplierID, legacyEnvelope)
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
	s.driverHub.Broadcast(ctx, "driver:"+driverID, payload)
}

func (s *Service) invalidatePayloadKeys(ctx context.Context, keys ...string) {
	if s.cache == nil || len(keys) == 0 {
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
func (s *Service) HandleTrucks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]TruckRow(nil), s.trucks...)
	s.mu.Unlock()
	wire := make([]payloaderTruckWire, len(rows))
	for i := range rows {
		wire[i] = payloaderTruckWire{
			ID:           rows[i].VehicleID,
			Label:        rows[i].PlateNo,
			LicensePlate: rows[i].PlateNo,
			VehicleClass: "TRUCK",
		}
	}
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
	ctx := context.Background()
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
	if s.guardIdempotency(w, r, body) {
		return
	}
	manifestID := manifestIDParam(r)
	if manifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}

	var manifest ManifestRow
	now := ""
	err = s.apply(r.Context(), func() error {
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
			BaseEvent:  events.BaseEvent{Type: events.EventManifestLoadingStarted, Timestamp: now},
			ManifestID: manifestID,
			SupplierID: s.supplierID,
			State:      payloadManifestStateLoading,
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
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "start_loading_failed"})
			return
		}
	}

	s.invalidatePayloadKeys(r.Context(), payloadManifestKey(manifestID), payloadManifestListKey(s.supplierID), payloadOrderListKey(s.supplierID))
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
	if s.guardIdempotency(w, r, body) {
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
	err = s.apply(r.Context(), func() error {
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
			BaseEvent:     events.BaseEvent{Type: events.EventManifestOrderInjected, Timestamp: now},
			ManifestID:    manifestID,
			OrderID:       req.OrderID,
			SupplierID:    s.supplierID,
			TotalVolumeVU: manifest.TotalVolumeVU,
			StopCount:     int64(manifest.StopCount),
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

	s.invalidatePayloadKeys(r.Context(), payloadManifestKey(manifestID), payloadManifestListKey(s.supplierID), payloadOrderListKey(s.supplierID))
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
func (s *Service) HandleManifestException(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
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

	var req manifestExceptionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.ManifestID = strings.TrimSpace(req.ManifestID)
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.Reason = strings.ToUpper(strings.TrimSpace(req.Reason))
	if req.ManifestID == "" || req.OrderID == "" || req.Reason == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_order_id_reason_required"})
		return
	}
	if req.Reason != "OVERFLOW" && req.Reason != "DAMAGED" && req.Reason != "MANUAL" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_reason"})
		return
	}

	var exception ManifestException
	err = s.apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		mIdx := s.findManifestIndexLocked(req.ManifestID)
		if mIdx < 0 {
			return fmt.Errorf("manifest_not_found")
		}
		manifest := s.manifests[mIdx]
		if manifest.State != payloadManifestStateLoading {
			return fmt.Errorf("manifest_not_loading")
		}

		moIdx := s.findManifestOrderIndexLocked(req.ManifestID, req.OrderID)
		if moIdx < 0 {
			return fmt.Errorf("manifest_order_not_found")
		}

		now := s.now().Format(time.RFC3339Nano)
		manifestOrders := s.manifestOrders[req.ManifestID]
		removedVolume := manifestOrders[moIdx].VolumeVU
		manifestOrders[moIdx].State = "REMOVED_" + req.Reason
		manifestOrders[moIdx].Reason = req.Reason
		manifestOrders[moIdx].UpdatedAt = now
		s.manifestOrders[req.ManifestID] = manifestOrders

		if manifest.TotalVolumeVU >= removedVolume {
			manifest.TotalVolumeVU -= removedVolume
		}
		if manifest.StopCount > 0 {
			manifest.StopCount--
		}
		manifest.UpdatedAt = now
		s.manifests[mIdx] = manifest

		oIdx := s.findOrderIndexLocked(req.OrderID)
		if oIdx >= 0 {
			s.orders[oIdx].ManifestID = ""
			s.orders[oIdx].Status = "PENDING"
			s.orders[oIdx].DispatchPriority = 10
			s.orders[oIdx].UpdatedAt = now
			if req.Reason == "OVERFLOW" {
				s.overflowCount[req.OrderID]++
				s.orders[oIdx].OverflowCount = s.overflowCount[req.OrderID]
			}
		}

		attemptCount := s.overflowCount[req.OrderID]
		if req.Reason != "OVERFLOW" {
			attemptCount = 1
		}
		escalated := req.Reason == "OVERFLOW" && attemptCount >= payloadExceptionEscalationThreshold
		exception = ManifestException{
			ExceptionID:  s.nextIDLocked("mex"),
			ManifestID:   req.ManifestID,
			OrderID:      req.OrderID,
			Reason:       req.Reason,
			Metadata:     strings.TrimSpace(req.Metadata),
			AttemptCount: attemptCount,
			Escalated:    escalated,
			CreatedAt:    now,
		}
		s.exceptions = append(s.exceptions, exception)
		return nil
	}, func(txn outbox.TxnBuffer) error {
		if err := outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, events.ManifestEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventManifestOrderException, Timestamp: exception.CreatedAt},
			ManifestID:   req.ManifestID,
			OrderID:      req.OrderID,
			SupplierID:   s.supplierID,
			Reason:       req.Reason,
			AttemptCount: exception.AttemptCount,
			Escalated:    exception.Escalated,
		}); err != nil {
			return err
		}
		if exception.Escalated {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, events.ManifestEvent{
				BaseEvent:    events.BaseEvent{Type: events.EventManifestDLQEscalation, Timestamp: exception.CreatedAt},
				ManifestID:   req.ManifestID,
				OrderID:      req.OrderID,
				SupplierID:   s.supplierID,
				Reason:       req.Reason,
				AttemptCount: exception.AttemptCount,
			})
		}
		return nil
	})
	if err != nil {
		switch err.Error() {
		case "manifest_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
			return
		case "manifest_not_loading":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "manifest_not_loading"})
			return
		case "manifest_order_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_order_not_found"})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "manifest_exception_failed"})
			return
		}
	}

	s.invalidatePayloadKeys(r.Context(), payloadManifestKey(req.ManifestID), payloadManifestListKey(s.supplierID), payloadOrderListKey(s.supplierID), payloadExceptionListKey(s.supplierID))
	s.broadcastPayloadEvent(r.Context(), events.EventManifestOrderException, map[string]any{
		"manifest_id":   req.ManifestID,
		"order_id":      req.OrderID,
		"reason":        req.Reason,
		"attempt_count": exception.AttemptCount,
		"escalated":     exception.Escalated,
		"exception_id":  exception.ExceptionID,
	})
	if exception.Escalated {
		s.broadcastPayloadEvent(r.Context(), events.EventManifestDLQEscalation, map[string]any{
			"manifest_id":   req.ManifestID,
			"order_id":      req.OrderID,
			"reason":        req.Reason,
			"attempt_count": exception.AttemptCount,
			"exception_id":  exception.ExceptionID,
		})
	}

	resp := map[string]any{
		"exception_id":   exception.ExceptionID,
		"manifest_id":    exception.ManifestID,
		"order_id":       exception.OrderID,
		"reason":         exception.Reason,
		"overflow_count": exception.AttemptCount,
		"escalated":      exception.Escalated,
		"reinjected":     true,
		"new_priority":   10,
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encode_response_failed"})
		return
	}
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleManifestExceptions serves GET /v1/payloader/manifest-exceptions.
func (s *Service) HandleManifestExceptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	escalatedOnly := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("escalated")), "true")

	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]ManifestException(nil), s.exceptions...)
	s.mu.Unlock()

	if escalatedOnly {
		filtered := make([]ManifestException, 0, len(rows))
		for i := range rows {
			if rows[i].Escalated {
				filtered = append(filtered, rows[i])
			}
		}
		rows = filtered
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].CreatedAt > rows[j].CreatedAt })
	writeJSON(w, http.StatusOK, map[string]any{"exceptions": rows})
}

// HandleRecommendReassign serves POST /v1/payloader/recommend-reassign.
func (s *Service) HandleRecommendReassign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
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

	var req recommendReassignRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)

	s.mu.Lock()
	s.ensureDemoDataLocked()
	if req.OrderID == "" && len(s.orders) > 0 {
		for i := range s.orders {
			if s.orders[i].Status == "LOADED" || s.orders[i].Status == "PENDING" {
				req.OrderID = s.orders[i].OrderID
				break
			}
		}
		if req.OrderID == "" {
			req.OrderID = s.orders[0].OrderID
		}
	}

	oIdx := s.findOrderIndexLocked(req.OrderID)
	if oIdx < 0 {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	order := s.orders[oIdx]

	recommendations := make([]ReassignRecommendation, 0)
	for i := range s.manifests {
		manifest := s.manifests[i]
		if manifest.State != payloadManifestStateDraft && manifest.State != payloadManifestStateLoading {
			continue
		}
		toRoute := routeIDForManifest(manifest)
		if toRoute == order.RouteID {
			continue
		}
		loadPenalty := 0.0
		if manifest.MaxVolumeVU > 0 {
			loadPenalty = (float64(manifest.TotalVolumeVU) / float64(manifest.MaxVolumeVU)) * 40.0
		}
		score := 100.0 - float64(manifest.StopCount*6) - loadPenalty
		if score < 1 {
			score = 1
		}
		recommendations = append(recommendations, ReassignRecommendation{
			OrderID:    order.OrderID,
			FromRoute:  order.RouteID,
			ToRoute:    toRoute,
			ToDriverID: manifest.DriverID,
			Score:      score,
			Reason:     "lower_route_load",
		})
	}
	truckRecs := buildTruckRecommendationsLocked(s, order, recommendations)
	retailerName := "Retailer"
	orderVolume := float64(10)
	if oIdx >= 0 {
		orderVolume = float64(order.TotalMinor) / 100.0
	}
	currentDriver := order.RouteID
	s.mu.Unlock()

	sort.Slice(truckRecs, func(i, j int) bool { return truckRecs[i].Score > truckRecs[j].Score })
	idemCommitted = true
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"order_id":        req.OrderID,
		"retailer_name":   retailerName,
		"order_volume_vu": orderVolume,
		"current_driver":  currentDriver,
		"recommendations": truckRecs,
	})
}

// HandleApplyReassign serves POST /v1/payloader/reassign-order.
func (s *Service) HandleApplyReassign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
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
	var req applyReassignRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.ToRouteID = strings.TrimSpace(req.ToRouteID)
	req.ToManifestID = strings.TrimSpace(req.ToManifestID)
	req.ToDriverID = strings.TrimSpace(req.ToDriverID)
	if req.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}

	var reassignment Reassignment
	err = s.apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		oIdx := s.findOrderIndexLocked(req.OrderID)
		if oIdx < 0 {
			return fmt.Errorf("order_not_found")
		}
		order := s.orders[oIdx]
		fromRoute := order.RouteID
		fromManifestID := order.ManifestID
		targetManifestID := req.ToManifestID
		targetRouteID := req.ToRouteID
		targetDriverID := req.ToDriverID
		reassignedVolume := int64(10)
		if req.IsPartial && req.PartialVolumeVU > 0 {
			reassignedVolume = req.PartialVolumeVU
		}
		if fromManifestID != "" {
			fromManifestOrders := s.manifestOrders[fromManifestID]
			fromManifestOrderIdx := s.findManifestOrderIndexLocked(fromManifestID, order.OrderID)
			if fromManifestOrderIdx >= 0 && fromManifestOrders[fromManifestOrderIdx].VolumeVU > 0 {
				if req.IsPartial && req.PartialVolumeVU == 0 {
					reassignedVolume = fromManifestOrders[fromManifestOrderIdx].VolumeVU / 2
				} else if !req.IsPartial {
					reassignedVolume = fromManifestOrders[fromManifestOrderIdx].VolumeVU
				}
			}
		}

		if targetManifestID != "" {
			if targetManifestID == order.ManifestID && (targetRouteID == "" || targetRouteID == order.RouteID) {
				return fmt.Errorf("order_already_assigned")
			}
			targetManifestIdx := s.findManifestIndexLocked(targetManifestID)
			if targetManifestIdx < 0 {
				return fmt.Errorf("target_manifest_not_found")
			}
			targetManifest := s.manifests[targetManifestIdx]
			if targetManifest.State != payloadManifestStateDraft && targetManifest.State != payloadManifestStateLoading {
				return fmt.Errorf("target_manifest_not_mutable")
			}
			resolvedTargetRoute := routeIDForManifest(targetManifest)
			if targetRouteID != "" && targetRouteID != resolvedTargetRoute {
				return fmt.Errorf("target_route_mismatch")
			}
			targetRouteID = resolvedTargetRoute
			if targetManifest.MaxVolumeVU > 0 && targetManifest.TotalVolumeVU+reassignedVolume > targetManifest.MaxVolumeVU {
				return fmt.Errorf("target_manifest_capacity_exceeded")
			}
			if targetDriverID != "" && targetManifest.DriverID != targetDriverID {
				return fmt.Errorf("target_driver_manifest_mismatch")
			}
			if targetDriverID == "" {
				targetDriverID = targetManifest.DriverID
			}
		} else {
			sourceManifestMutable := false
			sourceManifestDriverID := ""
			if order.ManifestID != "" {
				sourceManifestIdx := s.findManifestIndexLocked(order.ManifestID)
				if sourceManifestIdx >= 0 {
					sourceManifest := s.manifests[sourceManifestIdx]
					sourceManifestDriverID = sourceManifest.DriverID
					if sourceManifest.State == payloadManifestStateDraft || sourceManifest.State == payloadManifestStateLoading {
						sourceManifestMutable = true
					}
				}
			}
			if targetRouteID != "" && targetRouteID == order.RouteID && sourceManifestMutable {
				if targetDriverID != "" && targetDriverID != sourceManifestDriverID {
					return fmt.Errorf("target_driver_manifest_mismatch")
				}
				return fmt.Errorf("order_already_assigned")
			}
			driverMismatchOnTargetRoute := false
			for i := range s.manifests {
				candidateManifest := s.manifests[i]
				if candidateManifest.State != payloadManifestStateDraft && candidateManifest.State != payloadManifestStateLoading {
					continue
				}
				if candidateManifest.ManifestID == order.ManifestID {
					continue
				}
				candidate := routeIDForManifest(candidateManifest)
				if candidate == order.RouteID {
					if targetRouteID == "" || sourceManifestMutable {
						continue
					}
				}
				if targetRouteID != "" && candidate != targetRouteID {
					continue
				}
				if targetDriverID != "" && candidateManifest.DriverID != targetDriverID {
					if targetRouteID != "" && candidate == targetRouteID {
						driverMismatchOnTargetRoute = true
					}
					continue
				}
				if candidateManifest.MaxVolumeVU > 0 && candidateManifest.TotalVolumeVU+reassignedVolume > candidateManifest.MaxVolumeVU {
					continue
				}
				targetRouteID = candidate
				targetManifestID = candidateManifest.ManifestID
				if targetDriverID == "" {
					targetDriverID = candidateManifest.DriverID
				}
				break
			}
			if targetManifestID == "" {
				if targetRouteID != "" {
					if targetDriverID != "" && driverMismatchOnTargetRoute {
						return fmt.Errorf("target_driver_manifest_mismatch")
					}
					return fmt.Errorf("target_manifest_not_found")
				}
				if sourceManifestMutable {
					return fmt.Errorf("order_already_assigned")
				}
				return fmt.Errorf("reassign_target_unavailable")
			}
		}
		if targetManifestID == order.ManifestID {
			return fmt.Errorf("order_already_assigned")
		}

		sourceManifestIdx := -1
		if order.ManifestID != "" {
			sourceManifestIdx = s.findManifestIndexLocked(order.ManifestID)
			if sourceManifestIdx < 0 {
				return fmt.Errorf("source_manifest_not_found")
			}
			sourceRouteID := routeIDForManifest(s.manifests[sourceManifestIdx])
			if order.RouteID != "" && order.RouteID != sourceRouteID {
				return fmt.Errorf("source_route_manifest_mismatch")
			}
			if s.findManifestOrderIndexLocked(order.ManifestID, order.OrderID) < 0 {
				return fmt.Errorf("source_manifest_order_missing")
			}
		}

		now := s.now().Format(time.RFC3339Nano)
		splitGroupID := order.SplitGroupID
		if req.IsPartial && splitGroupID == "" {
			splitGroupID = uuid.NewString()
			s.orders[oIdx].SplitGroupID = splitGroupID
		}

		if sourceManifestIdx >= 0 {
			fromManifestOrders := s.manifestOrders[order.ManifestID]
			moIdx := s.findManifestOrderIndexLocked(order.ManifestID, order.OrderID)
			removedVol := reassignedVolume
			if req.IsPartial {
				fromManifestOrders[moIdx].VolumeVU -= removedVol
				if fromManifestOrders[moIdx].VolumeVU < 0 {
					fromManifestOrders[moIdx].VolumeVU = 0
				}
				fromManifestOrders[moIdx].SplitGroupID = splitGroupID
			} else {
				fromManifestOrders[moIdx].State = "REMOVED_REASSIGNED"
				fromManifestOrders[moIdx].Reason = "REASSIGNED"
			}
			fromManifestOrders[moIdx].UpdatedAt = now
			s.manifestOrders[order.ManifestID] = fromManifestOrders
			if !req.IsPartial && s.manifests[sourceManifestIdx].StopCount > 0 {
				s.manifests[sourceManifestIdx].StopCount--
			}
			if s.manifests[sourceManifestIdx].TotalVolumeVU >= removedVol {
				s.manifests[sourceManifestIdx].TotalVolumeVU -= removedVol
			}
			s.manifests[sourceManifestIdx].UpdatedAt = now
		}

		if targetManifestID != "" {
			targetManifestIdx := s.findManifestIndexLocked(targetManifestID)
			if targetManifestIdx < 0 {
				return fmt.Errorf("target_manifest_not_found")
			}
			targetManifest := s.manifests[targetManifestIdx]
			if targetManifest.MaxVolumeVU > 0 && targetManifest.TotalVolumeVU+reassignedVolume > targetManifest.MaxVolumeVU {
				return fmt.Errorf("target_manifest_capacity_exceeded")
			}
			targetManifestOrders := s.manifestOrders[targetManifestID]
			targetManifestOrders = append(targetManifestOrders, ManifestOrder{
				ManifestID: targetManifestID,
				OrderID:    order.OrderID,
				State:      "ASSIGNED",
				VolumeVU:   reassignedVolume,
				SplitGroupID: splitGroupID,
				UpdatedAt:  now,
			})
			s.manifestOrders[targetManifestID] = targetManifestOrders
			targetManifest.StopCount++
			targetManifest.TotalVolumeVU += reassignedVolume
			targetManifest.UpdatedAt = now
			s.manifests[targetManifestIdx] = targetManifest
		}

		s.orders[oIdx].RouteID = targetRouteID
		s.orders[oIdx].ManifestID = targetManifestID
		s.orders[oIdx].Status = "LOADED"
		s.orders[oIdx].ReassignDepth++
		s.orders[oIdx].UpdatedAt = now

		reassignment = Reassignment{
			OrderID:        order.OrderID,
			FromRouteID:    fromRoute,
			ToRouteID:      targetRouteID,
			FromManifestID: fromManifestID,
			ManifestID:     targetManifestID,
			ToDriverID:     targetDriverID,
			Reason:         strings.TrimSpace(req.Reason),
			Depth:          s.orders[oIdx].ReassignDepth,
			AppliedAt:      now,
		}
		s.reassignments = append(s.reassignments, reassignment)
		return nil
	}, func(txn outbox.TxnBuffer) error {
		aggregateID := coalesceString(reassignment.ManifestID, reassignment.FromManifestID, "payload-reassign")
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, aggregateID, events.TopicMain, events.ManifestEvent{
			BaseEvent:      events.BaseEvent{Type: events.EventManifestRebalanced, Timestamp: reassignment.AppliedAt},
			ManifestID:     aggregateID,
			FromManifestID: reassignment.FromManifestID,
			ToManifestID:   reassignment.ManifestID,
			OrderID:        req.OrderID,
			SupplierID:     s.supplierID,
			FromRouteID:    reassignment.FromRouteID,
			ToRouteID:      reassignment.ToRouteID,
			ToDriverID:     reassignment.ToDriverID,
			Depth:          reassignment.Depth,
			Reason:         reassignment.Reason,
		})
	})
	if err != nil {
		switch err.Error() {
		case "order_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
			return
		case "target_manifest_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "target_manifest_not_found"})
			return
		case "target_manifest_not_mutable":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "target_manifest_not_mutable"})
			return
		case "source_manifest_not_found":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "source_manifest_not_found"})
			return
		case "source_route_manifest_mismatch":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "source_route_manifest_mismatch"})
			return
		case "source_manifest_order_missing":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "source_manifest_order_missing"})
			return
		case "target_route_mismatch":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "target_route_mismatch"})
			return
		case "target_driver_manifest_mismatch":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "target_driver_manifest_mismatch"})
			return
		case "target_manifest_capacity_exceeded":
			platform.WriteErrorWithExplain(w, http.StatusConflict, "target_manifest_capacity_exceeded", err)
			return
		case "reassign_target_unavailable":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "reassign_target_unavailable"})
			return
		case "order_already_assigned":
			s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{"status": "already_assigned", "order_id": req.OrderID})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "reassign_failed"})
			return
		}
	}

	keys := []string{payloadManifestListKey(s.supplierID), payloadOrderListKey(s.supplierID)}
	if reassignment.FromManifestID != "" {
		keys = append(keys, payloadManifestKey(reassignment.FromManifestID))
	}
	if reassignment.ManifestID != "" {
		keys = append(keys, payloadManifestKey(reassignment.ManifestID))
	}
	s.invalidatePayloadKeys(r.Context(), keys...)
	s.broadcastPayloadEvent(r.Context(), events.EventManifestRebalanced, map[string]any{
		"order_id":         reassignment.OrderID,
		"from_route_id":    reassignment.FromRouteID,
		"to_route_id":      reassignment.ToRouteID,
		"from_manifest_id": reassignment.FromManifestID,
		"to_manifest_id":   reassignment.ManifestID,
		"manifest_id":      reassignment.ManifestID,
		"to_driver_id":     reassignment.ToDriverID,
		"reassign_depth":   reassignment.Depth,
		"applied_at":       reassignment.AppliedAt,
	})

	if reassignment.ToDriverID != "" {
		s.broadcastDriverEvent(r.Context(), reassignment.ToDriverID, map[string]any{
			"type":        "MANIFEST_AMENDED",
			"driver_id":   reassignment.ToDriverID,
			"manifest_id": reassignment.ManifestID,
			"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
		})
	}

	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"status":         "order_reassigned",
		"order_id":       reassignment.OrderID,
		"from_route_id":  reassignment.FromRouteID,
		"to_route_id":    reassignment.ToRouteID,
		"to_manifest_id": reassignment.ManifestID,
		"reassign_depth": reassignment.Depth,
		"applied_at":     reassignment.AppliedAt,
	})
}

// HandleSealCompletedManifests serves POST /v1/payloader/manifests/seal-completed.
func (s *Service) HandleSealCompletedManifests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
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
	var req struct {
		ManifestIDs []string `json:"manifest_ids"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	results := make([]map[string]any, 0, len(req.ManifestIDs))
	sealedCount := 0
	for _, manifestID := range req.ManifestIDs {
		manifestID = strings.TrimSpace(manifestID)
		if manifestID == "" {
			continue
		}
		if gateErr := s.assertPickWaveReadyForSeal(r.Context(), manifestID); gateErr != nil {
			results = append(results, map[string]any{
				"manifest_id": manifestID,
				"status":      "pick_wave_blocked",
				"error":       gateErr.Error(),
			})
			continue
		}
		var manifest ManifestRow
		err := s.apply(r.Context(), func() error {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.ensureDemoDataLocked()
			var sealErr error
			manifest, sealErr = s.sealManifestLocked(manifestID)
			return sealErr
		}, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, manifestID, events.TopicMain, events.ManifestEvent{
				BaseEvent:  events.BaseEvent{Type: events.EventManifestSealed, Timestamp: manifest.UpdatedAt},
				ManifestID: manifestID,
				SupplierID: s.supplierID,
				State:      payloadManifestStateSealed,
				RouteID:    routeIDForManifest(manifest),
				DriverID:   manifest.DriverID,
				VehicleID:  manifest.VehicleID,
				OrderCount: manifest.StopCount,
			})
		})
		switch {
		case err == http.ErrMissingFile:
			row := map[string]any{"manifest_id": manifestID, "status": "not_found"}
			platform.AttachExplainToMap(row, "manifest_not_found")
			results = append(results, row)
		case err == http.ErrBodyNotAllowed:
			row := map[string]any{"manifest_id": manifestID, "status": "not_sealable"}
			platform.AttachExplainToMap(row, "manifest_not_sealable")
			results = append(results, row)
		case err != nil:
			row := map[string]any{"manifest_id": manifestID, "status": "seal_failed"}
			platform.AttachExplainToMap(row, "manifest_seal_failed")
			results = append(results, row)
		default:
			sealedCount++
			if s.manifestStore != nil {
				if geomErr := s.manifestStore.PersistRouteGeometryForManifest(r.Context(), manifestID, "manifest_sealed"); geomErr != nil {
					s.log.WarnContext(r.Context(), "manifest route geometry persist failed",
						"manifest_id", manifestID, "err", geomErr)
				}
			}
			s.invalidatePayloadKeys(r.Context(), payloadManifestKey(manifestID), payloadManifestListKey(s.supplierID), payloadOrderListKey(s.supplierID))
			s.broadcastPayloadEvent(r.Context(), events.EventManifestSealed, map[string]any{
				"manifest_id": manifestID,
				"state":       payloadManifestStateSealed,
				"route_id":    routeIDForManifest(manifest),
				"driver_id":   manifest.DriverID,
				"vehicle_id":  manifest.VehicleID,
				"order_count": manifest.StopCount,
				"updated_at":  manifest.UpdatedAt,
			})
			if manifest.DriverID != "" {
				s.broadcastDriverEvent(r.Context(), manifest.DriverID, map[string]any{
					"type":        "MANIFEST_DISPATCHED",
					"driver_id":   manifest.DriverID,
					"manifest_id": manifestID,
					"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
				})
			}
			s.afterSealAssignSSCC(r.Context(), manifestID)
			results = append(results, map[string]any{
				"manifest_id": manifestID,
				"status":      "sealed",
				"state":       manifest.State,
				"sealed_at":   manifest.SealedAt,
				"driver_id":   manifest.DriverID,
				"vehicle_id":  manifest.VehicleID,
			})
		}
	}

	resp := map[string]any{
		"status":       "batch_seal_complete",
		"sealed_count": sealedCount,
		"results":      results,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleSealManifest serves POST /v1/payloader/manifests/{manifestID}/seal.
func (s *Service) HandleSealManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, err := readLimitedBody(r, 8*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	manifestID := manifestIDParam(r)
	if manifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}
	sealPickWarn, gateErr := s.assertPickWaveReadyForSealResult(r.Context(), manifestID)
	if gateErr != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": gateErr.Error()})
		return
	}

	var manifest ManifestRow
	err = s.apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		var sealErr error
		manifest, sealErr = s.sealManifestLocked(manifestID)
		return sealErr
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, manifestID, events.TopicMain, events.ManifestEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventManifestSealed, Timestamp: manifest.UpdatedAt},
			ManifestID: manifestID,
			SupplierID: s.supplierID,
			State:      payloadManifestStateSealed,
			RouteID:    routeIDForManifest(manifest),
			DriverID:   manifest.DriverID,
			VehicleID:  manifest.VehicleID,
			OrderCount: manifest.StopCount,
		})
	})
	if err == http.ErrMissingFile {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
		return
	}
	if err == http.ErrBodyNotAllowed {
		platform.WriteErrorWithExplain(w, http.StatusConflict, "manifest_not_sealable", err)
		return
	}
	if err != nil {
		platform.WriteErrorWithExplain(w, http.StatusInternalServerError, "manifest_seal_failed", err)
		return
	}

	if s.manifestStore != nil {
		if geomErr := s.manifestStore.PersistRouteGeometryForManifest(r.Context(), manifestID, "manifest_sealed"); geomErr != nil {
			s.log.WarnContext(r.Context(), "manifest route geometry persist failed",
				"manifest_id", manifestID, "err", geomErr)
		}
	}

	s.invalidatePayloadKeys(r.Context(), payloadManifestKey(manifestID), payloadManifestListKey(s.supplierID), payloadOrderListKey(s.supplierID))
	s.broadcastPayloadEvent(r.Context(), events.EventManifestSealed, map[string]any{
		"manifest_id": manifestID,
		"state":       payloadManifestStateSealed,
		"route_id":    routeIDForManifest(manifest),
		"driver_id":   manifest.DriverID,
		"vehicle_id":  manifest.VehicleID,
		"order_count": manifest.StopCount,
		"updated_at":  manifest.UpdatedAt,
	})

	if manifest.DriverID != "" {
		s.broadcastDriverEvent(r.Context(), manifest.DriverID, map[string]any{
			"type":        "MANIFEST_DISPATCHED",
			"driver_id":   manifest.DriverID,
			"manifest_id": manifestID,
			"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
		})
	}

	resp := map[string]any{
		"status":      "PAYLOAD_MANIFEST_SEALED",
		"manifest_id": manifest.ManifestID,
		"state":       manifest.State,
		"sealed_at":   manifest.SealedAt,
		"stop_count":  manifest.StopCount,
		"volume_vu":   float64(manifest.TotalVolumeVU),
		"max_vu":      float64(manifest.MaxVolumeVU),
	}
	if sealPickWarn != "" {
		resp["pick_wave_warning"] = sealPickWarn
	}
	s.afterSealAssignSSCC(r.Context(), manifestID)

	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleSeal serves POST /v1/payload/seal.
func (s *Service) HandleSeal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
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
	var req sealRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.ManifestID = strings.TrimSpace(req.ManifestID)
	if req.OrderID == "" && req.ManifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id required"})
		return
	}
	if req.ManifestID != "" {
		if gateErr := s.assertPickWaveReadyForSeal(r.Context(), req.ManifestID); gateErr != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": gateErr.Error()})
			return
		}
	}

	s.mu.Lock()
	s.ensureDemoDataLocked()

	if req.ManifestID != "" {
		var manifest ManifestRow
		err := s.apply(r.Context(), func() error {
			var sealErr error
			manifest, sealErr = s.sealManifestLocked(req.ManifestID)
			return sealErr
		}, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, events.ManifestEvent{
				BaseEvent:  events.BaseEvent{Type: events.EventManifestSealed, Timestamp: manifest.UpdatedAt},
				ManifestID: req.ManifestID,
				SupplierID: s.supplierID,
				State:      payloadManifestStateSealed,
				RouteID:    routeIDForManifest(manifest),
				DriverID:   manifest.DriverID,
				VehicleID:  manifest.VehicleID,
				OrderCount: manifest.StopCount,
			})
		})
		s.mu.Unlock()
		if err == http.ErrMissingFile {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
			return
		}
		if err == http.ErrBodyNotAllowed {
			platform.WriteErrorWithExplain(w, http.StatusConflict, "manifest_not_sealable", err)
			return
		}
		if err != nil {
			platform.WriteErrorWithExplain(w, http.StatusInternalServerError, "manifest_seal_failed", err)
			return
		}

		if s.manifestStore != nil {
			if geomErr := s.manifestStore.PersistRouteGeometryForManifest(r.Context(), req.ManifestID, "manifest_sealed"); geomErr != nil {
				s.log.WarnContext(r.Context(), "manifest route geometry persist failed",
					"manifest_id", req.ManifestID, "err", geomErr)
			}
		}

		s.invalidatePayloadKeys(r.Context(), payloadManifestKey(req.ManifestID), payloadManifestListKey(s.supplierID), payloadOrderListKey(s.supplierID))
		s.broadcastPayloadEvent(r.Context(), events.EventManifestSealed, map[string]any{
			"manifest_id": req.ManifestID,
			"state":       payloadManifestStateSealed,
			"route_id":    routeIDForManifest(manifest),
			"driver_id":   manifest.DriverID,
			"vehicle_id":  manifest.VehicleID,
			"order_count": manifest.StopCount,
			"updated_at":  manifest.UpdatedAt,
		})
		s.afterSealAssignSSCC(r.Context(), req.ManifestID)
		resp := map[string]any{
			"status":      "PAYLOAD_MANIFEST_SEALED",
			"manifest_id": manifest.ManifestID,
			"state":       manifest.State,
			"sealed_at":   manifest.SealedAt,
		}
		respBytes, _ := json.Marshal(resp)
		s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
		writeJSONBytes(w, http.StatusOK, respBytes)
		return
	}

	oIdx := s.findOrderIndexLocked(req.OrderID)
	now := s.now().Format(time.RFC3339Nano)
	if oIdx < 0 {
		s.orders = append(s.orders, OrderRow{
			OrderID:    req.OrderID,
			Status:     "DISPATCHED",
			TotalMinor: 0,
			Currency:   s.currency,
			UpdatedAt:  now,
		})
	} else {
		s.orders[oIdx].Status = "DISPATCHED"
		s.orders[oIdx].UpdatedAt = now
	}

	var sealedOrderManifestID string
	if oIdx >= 0 && s.orders[oIdx].ManifestID != "" {
		manifestID := s.orders[oIdx].ManifestID
		sealedOrderManifestID = manifestID
		orders := s.manifestOrders[manifestID]
		moIdx := s.findManifestOrderIndexLocked(manifestID, req.OrderID)
		if moIdx >= 0 {
			orders[moIdx].State = "SEALED"
			orders[moIdx].UpdatedAt = now
			s.manifestOrders[manifestID] = orders
		}
	}
	s.mu.Unlock()
	if sealedOrderManifestID != "" {
		s.afterSealAssignSSCC(r.Context(), sealedOrderManifestID)
	}
	s.invalidatePayloadKeys(r.Context(), payloadOrderListKey(s.supplierID))
	s.broadcastPayloadEvent(r.Context(), "ORDER_DISPATCHED", map[string]any{"order_id": req.OrderID})
	resp := map[string]any{
		"status":        "PAYLOAD_SEALED_AND_DISPATCHED",
		"order_id":      req.OrderID,
		"sealed_at":     now,
		"dispatch_code": dispatchCodeForOrder(req.OrderID),
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func coalesceString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
func (s *Service) HandleSealAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, err := readLimitedBody(r, 4*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}

	var req struct {
		// StateFilter, if set, limits sealing to manifests in that state only.
		// Valid values: "DRAFT", "LOADING". Empty means both are sealed.
		StateFilter string `json:"state_filter"`
	}
	// body may be empty for a no-args call — treat as "seal everything".
	if len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}
	stateFilter := strings.ToUpper(strings.TrimSpace(req.StateFilter))

	// Collect candidate manifest IDs under the read lock.
	s.mu.RLock()
	var candidateIDs []string
	for _, m := range s.manifests {
		state := strings.ToUpper(m.State)
		if stateFilter != "" && state != stateFilter {
			continue
		}
		if state == payloadManifestStateDraft || state == payloadManifestStateLoading {
			candidateIDs = append(candidateIDs, m.ManifestID)
		}
	}
	s.mu.RUnlock()

	if len(candidateIDs) == 0 {
		resp := map[string]any{
			"status":       "no_sealable_manifests",
			"sealed_count": 0,
			"results":      []any{},
		}
		respBytes, _ := json.Marshal(resp)
		s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
		writeJSONBytes(w, http.StatusOK, respBytes)
		return
	}

	results := make([]map[string]any, 0, len(candidateIDs))
	sealedCount := 0

	for _, manifestID := range candidateIDs {
		if gateErr := s.assertPickWaveReadyForSeal(r.Context(), manifestID); gateErr != nil {
			results = append(results, map[string]any{
				"manifest_id": manifestID,
				"status":      "pick_wave_blocked",
				"error":       gateErr.Error(),
			})
			continue
		}
		var sealed ManifestRow
		sealErr := s.apply(r.Context(), func() error {
			s.mu.Lock()
			defer s.mu.Unlock()
			var e error
			sealed, e = s.sealManifestLocked(manifestID)
			return e
		}, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, manifestID, events.TopicMain, events.ManifestEvent{
				BaseEvent:  events.BaseEvent{Type: events.EventManifestSealed, Timestamp: sealed.UpdatedAt},
				ManifestID: manifestID,
				SupplierID: s.supplierID,
				State:      payloadManifestStateSealed,
				RouteID:    routeIDForManifest(sealed),
				DriverID:   sealed.DriverID,
				VehicleID:  sealed.VehicleID,
				OrderCount: sealed.StopCount,
			})
		})
		switch {
		case sealErr == http.ErrMissingFile:
			row := map[string]any{"manifest_id": manifestID, "status": "not_found"}
			platform.AttachExplainToMap(row, "manifest_not_found")
			results = append(results, row)
		case sealErr == http.ErrBodyNotAllowed:
			// Already sealed or dispatched — skip silently in seal-all context.
			row := map[string]any{"manifest_id": manifestID, "status": "already_sealed"}
			results = append(results, row)
		case sealErr != nil:
			row := map[string]any{"manifest_id": manifestID, "status": "seal_failed", "error": sealErr.Error()}
			platform.AttachExplainToMap(row, "manifest_seal_failed")
			results = append(results, row)
		default:
			sealedCount++
			if s.manifestStore != nil {
				if geomErr := s.manifestStore.PersistRouteGeometryForManifest(r.Context(), manifestID, "manifest_sealed"); geomErr != nil {
					s.log.WarnContext(r.Context(), "seal-all: route geometry persist failed",
						"manifest_id", manifestID, "err", geomErr)
				}
			}
			s.invalidatePayloadKeys(r.Context(),
				payloadManifestKey(manifestID),
				payloadManifestListKey(s.supplierID),
				payloadOrderListKey(s.supplierID),
			)
			s.broadcastPayloadEvent(r.Context(), events.EventManifestSealed, map[string]any{
				"manifest_id": manifestID,
				"state":       payloadManifestStateSealed,
				"route_id":    routeIDForManifest(sealed),
				"driver_id":   sealed.DriverID,
				"vehicle_id":  sealed.VehicleID,
				"order_count": sealed.StopCount,
				"updated_at":  sealed.UpdatedAt,
			})
			if sealed.DriverID != "" {
				s.broadcastDriverEvent(r.Context(), sealed.DriverID, map[string]any{
					"type":        "MANIFEST_DISPATCHED",
					"driver_id":   sealed.DriverID,
					"manifest_id": manifestID,
					"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
				})
			}
			s.afterSealAssignSSCC(r.Context(), manifestID)
			results = append(results, map[string]any{
				"manifest_id": manifestID,
				"status":      "sealed",
				"state":       sealed.State,
				"sealed_at":   sealed.SealedAt,
				"driver_id":   sealed.DriverID,
				"vehicle_id":  sealed.VehicleID,
			})
		}
	}

	resp := map[string]any{
		"status":          "seal_all_complete",
		"sealed_count":    sealedCount,
		"candidate_count": len(candidateIDs),
		"results":         results,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}
