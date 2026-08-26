package factory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/optimizerclient"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const (
	manifestStateDraft      = "DRAFT"
	manifestStateLoading    = "LOADING"
	manifestStateSealed     = "SEALED"
	manifestStateDispatched = "DISPATCHED"
	manifestStateCompleted  = "COMPLETED"
	manifestStateCancelled  = "CANCELLED"

	manifestExceptionEscalationThreshold = 3
)

// Service stores additive in-memory data for factory operational surfaces.
type Service struct {
	repo            Repository
	cache           *cache.Cache
	supplierHub     *ws.Hub
	factoryHub      *ws.Hub
	log             *slog.Logger
	spannerClient   *spanner.Client
	optimizerClient *optimizerclient.Client
	idem            idempotency.Store
	locations       telemetry.LastLocationReader
	planning        *PlanningService
	qcRepo          supplyQCRepo
	exceptionRepo   factoryExceptionBackend
	dashboardQuery  FactoryDashboardQuery

	seedSupplierID   string
	factoryNodeID    string
	currency         string
	jwtSecret        string
	jwtIssuer        string
	now              func() time.Time
	firebaseVerifier auth.FirebaseVerifier

	mu                    sync.RWMutex
	spannerLoaded         bool
	transfers             []TransferRow
	manifests             []ManifestRow
	manifestTransfers     map[string][]TransferRow
	manifestTransitions   map[string][]ManifestTransition
	manifestReassignments map[string][]ManifestReassignment
	manifestExceptions    []ManifestException
	fleetDrivers          []FleetDriver
	fleetVehicles         []FleetVehicle
	staff                 []StaffRow
	supplyRequests        []SupplyRequest
	seq                   int64
}

// ServiceConfig is the constructor input.
type ServiceConfig struct {
	Repo        Repository
	Cache       *cache.Cache
	SupplierHub *ws.Hub
	FactoryHub  *ws.Hub
	Log         *slog.Logger
	Spanner     *spanner.Client
	Locations   telemetry.LastLocationReader

	// SeedSupplierID is bootstrap/fixture fallback only (Gate 5 Week 11).
	SeedSupplierID string
	// SupplierID is deprecated; use SeedSupplierID.
	SupplierID       string
	FactoryNodeID    string
	Currency         string
	JWTSecret        string
	JWTIssuer        string
	Now              func() time.Time
	FirebaseVerifier auth.FirebaseVerifier
	Idem             idempotency.Store
	Planning         *PlanningService
	OptimizerClient  *optimizerclient.Client
	DashboardQuery   FactoryDashboardQuery
}

// TransferRow represents one factory transfer record.
type TransferRow struct {
	TransferID     string `json:"transfer_id"`
	OrderID        string `json:"order_id,omitempty"`
	ManifestID     string `json:"manifest_id,omitempty"`
	State          string `json:"state"`
	TotalVU        int64  `json:"total_vu"`
	DriverID       string `json:"driver_id,omitempty"`
	VehicleID      string `json:"vehicle_id,omitempty"`
	ReassignDepth  int    `json:"reassign_depth,omitempty"`
	ExceptionCount int64  `json:"exception_count,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// ManifestRow represents one manifest record.
type ManifestRow struct {
	ManifestID    string `json:"manifest_id"`
	State         string `json:"state"`
	TransferCnt   int    `json:"transfer_count"`
	TotalVolumeVU int64  `json:"total_volume_vu"`
	MaxVolumeVU   int64  `json:"max_volume_vu"`
	DriverID      string `json:"driver_id,omitempty"`
	VehicleID     string `json:"vehicle_id,omitempty"`
	// TruckID mirrors vehicle_id for payload-terminal / Wire compatibility (P1-18).
	TruckID            string `json:"truck_id,omitempty"`
	StopCount          int    `json:"stop_count,omitempty"`
	RegionCode         string `json:"region_code,omitempty"`
	ReassignmentDepth  int    `json:"reassignment_depth,omitempty"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
	LoadingStartedAt   string `json:"loading_started_at,omitempty"`
	SealedAt           string `json:"sealed_at,omitempty"`
	DispatchedAt       string `json:"dispatched_at,omitempty"`
	CompletedAt        string `json:"completed_at,omitempty"`
	CancelledAt        string `json:"cancelled_at,omitempty"`
	LastExceptionAt    string `json:"last_exception_at,omitempty"`
	EscalatedException bool   `json:"escalated_exception,omitempty"`
}

// ManifestTransition records one manifest lifecycle transition.
type ManifestTransition struct {
	Action    string `json:"action"`
	FromState string `json:"from_state"`
	ToState   string `json:"to_state"`
	Reason    string `json:"reason,omitempty"`
	At        string `json:"at"`
}

// ManifestReassignment records one transfer reassignment inside a manifest.
type ManifestReassignment struct {
	ManifestID      string `json:"manifest_id"`
	TransferID      string `json:"transfer_id"`
	FromDriverID    string `json:"from_driver_id,omitempty"`
	ToDriverID      string `json:"to_driver_id,omitempty"`
	FromVehicleID   string `json:"from_vehicle_id,omitempty"`
	ToVehicleID     string `json:"to_vehicle_id,omitempty"`
	Depth           int    `json:"depth"`
	Reason          string `json:"reason,omitempty"`
	ReassignedAt    string `json:"reassigned_at"`
	Recommendation  string `json:"recommendation,omitempty"`
	AppliedBy       string `json:"applied_by,omitempty"`
	CorrelationHint string `json:"correlation_hint,omitempty"`
}

// ManifestDetailSnapshot is the read-model payload shared by driver and
// factory manifest detail surfaces.
type ManifestDetailSnapshot struct {
	Manifest      ManifestRow            `json:"manifest"`
	Transfers     []TransferRow          `json:"transfers"`
	Transitions   []ManifestTransition   `json:"transitions"`
	Reassignments []ManifestReassignment `json:"reassignments"`
	Exceptions    []ManifestException    `json:"exceptions"`
	RouteID       string                 `json:"route_id,omitempty"`
	StopCount     int                    `json:"stop_count"`
	OrderCount    int                    `json:"order_count"`
}

func routeIDForManifest(m ManifestRow) string {
	if strings.TrimSpace(m.VehicleID) != "" {
		return "route_" + m.VehicleID
	}
	return "route_" + m.ManifestID
}

// ManifestException records transfer-level exceptions during loading/rebalance.
type ManifestException struct {
	ExceptionID   string `json:"exception_id"`
	ManifestID    string `json:"manifest_id"`
	TransferID    string `json:"transfer_id"`
	Reason        string `json:"reason"`
	Metadata      string `json:"metadata,omitempty"`
	AttemptCount  int64  `json:"attempt_count"`
	Escalated     bool   `json:"escalated"`
	CreatedAt     string `json:"created_at"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// FleetDriver represents one factory-scoped driver row.
type FleetDriver struct {
	DriverID string `json:"driver_id"`
	Name     string `json:"name"`
	OnShift  bool   `json:"on_shift"`
}

// FleetVehicle represents one factory-scoped vehicle row.
type FleetVehicle struct {
	VehicleID string `json:"vehicle_id"`
	PlateNo   string `json:"plate_no"`
	State     string `json:"state"`
}

// StaffRow represents one staff member row.
type StaffRow struct {
	StaffID      string `json:"staff_id"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	Phone        string `json:"phone,omitempty"`
	PasswordHash string `json:"-"`
}

// SupplyRequest represents one supply request row.
type SupplyRequest struct {
	RequestID             string              `json:"request_id"`
	Status                string              `json:"status"`
	CreatedAt             string              `json:"created_at"`
	UpdatedAt             string              `json:"updated_at"`
	WarehouseID           string              `json:"warehouse_id,omitempty"`
	Priority              string              `json:"priority,omitempty"`
	Notes                 string              `json:"notes,omitempty"`
	RegionID              string              `json:"region_id,omitempty"`
	RequestedDeliveryDate string              `json:"requested_delivery_date,omitempty"`
	TotalVolumeVU         float64             `json:"total_volume_vu,omitempty"`
	LinkedTransferID      string              `json:"linked_transfer_id,omitempty"`
	Items                 []SupplyRequestItem `json:"items,omitempty"`
	// G7.1 SLA enrichment (optional on wire; filled by handlers).
	SLADueAt          string   `json:"sla_due_at,omitempty"`
	SLAStatus         string   `json:"sla_status,omitempty"`
	SLAHoursRemaining *float64 `json:"sla_hours_remaining,omitempty"`
}

// SupplyRequestItem is one SKU line on a factory supply request.
type SupplyRequestItem struct {
	ItemID            string  `json:"item_id"`
	ProductID         string  `json:"product_id"`
	RequestedQuantity int64   `json:"requested_quantity"`
	ShippedQuantity   int64   `json:"shipped_quantity,omitempty"`
	ReceivedQuantity  int64   `json:"received_quantity,omitempty"`
	VarianceReason    string  `json:"variance_reason,omitempty"`
	RecommendedQty    int64   `json:"recommended_qty,omitempty"`
	UnitVolumeVU      float64 `json:"unit_volume_vu,omitempty"`
}

// NewService constructs the factory service.
func NewService(c ServiceConfig) *Service {
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	if c.Log == nil {
		c.Log = slog.Default()
	}
	if c.Repo == nil {
		c.Repo = NewInMemoryRepository()
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
	factoryNodeID := strings.TrimSpace(c.FactoryNodeID)
	if factoryNodeID == "" {
		factoryNodeID = seedID
	}
	return &Service{
		repo:                  c.Repo,
		cache:                 c.Cache,
		supplierHub:           c.SupplierHub,
		factoryHub:            c.FactoryHub,
		log:                   c.Log,
		seedSupplierID:        seedID,
		factoryNodeID:         factoryNodeID,
		currency:              c.Currency,
		jwtSecret:             c.JWTSecret,
		jwtIssuer:             c.JWTIssuer,
		now:                   c.Now,
		firebaseVerifier:      c.FirebaseVerifier,
		spannerClient:         c.Spanner,
		optimizerClient:       c.OptimizerClient,
		idem:                  c.Idem,
		locations:             c.Locations,
		planning:              c.Planning,
		dashboardQuery:        c.DashboardQuery,
		manifestTransfers:     make(map[string][]TransferRow),
		manifestTransitions:   make(map[string][]ManifestTransition),
		manifestReassignments: make(map[string][]ManifestReassignment),
	}
}

// resolveSupplierScope prefers request TenantContext over the bootstrap seed.
func (s *Service) resolveSupplierScope(ctx context.Context) string {
	return auth.PreferTenantSupplierID(ctx, s.seedSupplierID)
}

type transferCreateRequest struct {
	OrderID   string `json:"order_id"`
	TotalVU   int64  `json:"total_vu"`
	DriverID  string `json:"driver_id"`
	VehicleID string `json:"vehicle_id"`
}

type dispatchRequest struct {
	TransferIDs []string `json:"transfer_ids"`
	DriverID    string   `json:"driver_id"`
	VehicleID   string   `json:"vehicle_id"`
	MaxVolumeVU int64    `json:"max_volume_vu"`
	Reason      string   `json:"reason"`
}

type transitionRequest struct {
	Reason string `json:"reason"`
}

type manifestRebalanceRequest struct {
	ManifestID string `json:"manifest_id"`
	TransferID string `json:"transfer_id"`
	ToDriverID string `json:"to_driver_id"`
	ToVehicle  string `json:"to_vehicle_id"`
	Reason     string `json:"reason"`
	AppliedBy  string `json:"applied_by"`

	SourceManifestID string   `json:"source_manifest_id"`
	TargetManifestID string   `json:"target_manifest_id"`
	TransferIDs      []string `json:"transfer_ids"`
}

type manifestCancelTransferRequest struct {
	ManifestID string `json:"manifest_id"`
	TransferID string `json:"transfer_id"`
	Reason     string `json:"reason"`
	Metadata   string `json:"metadata"`
}

type manifestCancelRequest struct {
	ManifestID string `json:"manifest_id"`
	Reason     string `json:"reason"`
}

func (s *Service) nextIDLocked(prefix string) string {
	s.seq++
	return prefix + "_" + strconv.FormatInt(s.now().UnixNano(), 10) + "_" + strconv.FormatInt(s.seq, 10)
}

// portalSeedEnabled gates in-memory demo seed data. Disabled when Spanner is wired
// unless FACTORY_PORTAL_SEED=true or USE_DEMO_SEED=true is set explicitly for local scaffold runs.
func (s *Service) portalSeedEnabled() bool {
	envOn := func(key string) bool {
		switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
		case "1", "true", "yes":
			return true
		default:
			return false
		}
	}
	if s.spannerClient != nil {
		return envOn("FACTORY_PORTAL_SEED") || envOn("USE_DEMO_SEED")
	}
	return envOn("FACTORY_PORTAL_SEED") || envOn("USE_DEMO_SEED")
}

func (s *Service) ensureDemoDataLocked() {
	if !s.spannerLoaded {
		if r, ok := s.repo.(*SpannerRepository); ok {
			if err := r.Hydrate(context.Background(), s.factoryNodeID, s); err != nil {
				s.log.WarnContext(context.Background(), "factory spanner hydrate failed", "err", err)
			}
		}
	}
	if !s.portalSeedEnabled() {
		return
	}
	nowFn := s.clock()
	if s.manifestTransfers == nil {
		s.manifestTransfers = make(map[string][]TransferRow)
	}
	if s.manifestTransitions == nil {
		s.manifestTransitions = make(map[string][]ManifestTransition)
	}
	if s.manifestReassignments == nil {
		s.manifestReassignments = make(map[string][]ManifestReassignment)
	}
	if len(s.fleetDrivers) == 0 {
		s.fleetDrivers = []FleetDriver{
			{DriverID: "drv_factory_1", Name: "Factory Driver 1", OnShift: true},
			{DriverID: "drv_factory_2", Name: "Factory Driver 2", OnShift: true},
		}
	}
	if len(s.fleetVehicles) == 0 {
		s.fleetVehicles = []FleetVehicle{
			{VehicleID: "veh_factory_1", PlateNo: "01F111AA", State: "READY"},
			{VehicleID: "veh_factory_2", PlateNo: "01F222AA", State: "READY"},
		}
	}
	if len(s.staff) == 0 {
		s.staff = []StaffRow{
			{StaffID: "stf_factory_1", Name: "Factory Lead", Role: "FACTORY_ADMIN"},
			{StaffID: "stf_factory_2", Name: "Bay Operator", Role: "PAYLOAD"},
		}
	}
	if len(s.supplyRequests) == 0 {
		now := nowFn().Format(time.RFC3339Nano)
		s.supplyRequests = []SupplyRequest{
			{RequestID: "srq_factory_1", Status: "SUBMITTED", WarehouseID: "wh-demo-1", CreatedAt: now, UpdatedAt: now},
		}
	}
	if len(s.transfers) == 0 {
		now := nowFn().Format(time.RFC3339Nano)
		s.transfers = []TransferRow{
			{TransferID: "tr_factory_1", OrderID: "ord_factory_1", State: "CREATED", TotalVU: 42, CreatedAt: now, UpdatedAt: now},
			{TransferID: "tr_factory_2", OrderID: "ord_factory_2", State: "CREATED", TotalVU: 37, CreatedAt: now, UpdatedAt: now},
		}
	}
	if len(s.manifests) == 0 {
		now := nowFn().Format(time.RFC3339Nano)
		manifest := ManifestRow{
			ManifestID:    "mf_factory_1",
			State:         manifestStateDraft,
			TransferCnt:   2,
			TotalVolumeVU: 79,
			MaxVolumeVU:   120,
			DriverID:      "drv_factory_1",
			VehicleID:     "veh_factory_1",
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		s.manifests = append(s.manifests, manifest)
		s.manifestTransfers[manifest.ManifestID] = []TransferRow{
			{TransferID: "tr_factory_1", OrderID: "ord_factory_1", ManifestID: manifest.ManifestID, State: "ASSIGNED", TotalVU: 42, DriverID: manifest.DriverID, VehicleID: manifest.VehicleID, CreatedAt: now, UpdatedAt: now},
			{TransferID: "tr_factory_2", OrderID: "ord_factory_2", ManifestID: manifest.ManifestID, State: "ASSIGNED", TotalVU: 37, DriverID: manifest.DriverID, VehicleID: manifest.VehicleID, CreatedAt: now, UpdatedAt: now},
		}
		s.manifestTransitions[manifest.ManifestID] = []ManifestTransition{{
			Action:    "CREATE_DRAFT",
			FromState: "",
			ToState:   manifestStateDraft,
			At:        now,
		}}

		for i := range s.transfers {
			if s.transfers[i].TransferID == "tr_factory_1" || s.transfers[i].TransferID == "tr_factory_2" {
				s.transfers[i].ManifestID = manifest.ManifestID
				s.transfers[i].DriverID = manifest.DriverID
				s.transfers[i].VehicleID = manifest.VehicleID
				s.transfers[i].State = "ASSIGNED"
				s.transfers[i].UpdatedAt = now
			}
		}
	}
	s.ensureDemoManifestExceptionsLocked()
	s.ensureLoadingBayDemoTransfersLocked()
	if _, ok := s.repo.(*SpannerRepository); ok && !s.spannerLoaded {
		s.spannerLoaded = true
	}
}

func (s *Service) clock() func() time.Time {
	if s != nil && s.now != nil {
		return s.now
	}
	return func() time.Time { return time.Now().UTC() }
}

func (s *Service) ensureDemoManifestExceptionsLocked() {
	if len(s.manifestExceptions) > 0 || len(s.manifests) == 0 {
		return
	}
	now := s.clock()().Format(time.RFC3339Nano)
	s.manifestExceptions = []ManifestException{{
		ExceptionID:  "mex_factory_demo_1",
		ManifestID:   s.manifests[0].ManifestID,
		TransferID:   "tr_factory_1",
		Reason:       "OVERFLOW",
		AttemptCount: 1,
		Escalated:    false,
		CreatedAt:    now,
	}}
}

func (s *Service) ensureLoadingBayDemoTransfersLocked() {
	hasBay := false
	for i := range s.transfers {
		id := s.transfers[i].TransferID
		if id == "tr_bay_1" || id == "tr_bay_2" {
			hasBay = true
			break
		}
	}
	if hasBay {
		return
	}
	now := s.clock()().Format(time.RFC3339Nano)
	s.transfers = append(s.transfers,
		TransferRow{TransferID: "tr_bay_1", OrderID: "ord_bay_1", State: "APPROVED", TotalVU: 28, CreatedAt: now, UpdatedAt: now},
		TransferRow{TransferID: "tr_bay_2", OrderID: "ord_bay_2", State: "LOADING", TotalVU: 31, CreatedAt: now, UpdatedAt: now},
	)
}

func (s *Service) findManifestIndexLocked(manifestID string) int {
	for i := range s.manifests {
		if s.manifests[i].ManifestID == manifestID {
			return i
		}
	}
	return -1
}

// ManifestGateSnapshot returns the current manifest state needed for driver gate checks.
func (s *Service) ManifestGateSnapshot(manifestID string) (state string, stopCount int, totalVolumeVU int64, found bool) {
	if s == nil {
		return "", 0, 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureDemoDataLocked()
	idx := s.findManifestIndexLocked(manifestID)
	if idx < 0 {
		return "", 0, 0, false
	}
	manifest := s.manifests[idx]
	return manifest.State, manifest.TransferCnt, manifest.TotalVolumeVU, true
}

// ManifestDetailSnapshotForDriver returns the manifest detail visible to a
// driver-scoped manifest read, optionally narrowed by manifest id and date.
func (s *Service) ManifestDetailSnapshotForDriver(driverID, manifestID, date string) (ManifestDetailSnapshot, bool) {
	if s == nil {
		return ManifestDetailSnapshot{}, false
	}
	driverID = strings.TrimSpace(driverID)
	manifestID = strings.TrimSpace(manifestID)
	date = strings.TrimSpace(date)
	if driverID == "" {
		return ManifestDetailSnapshot{}, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureDemoDataLocked()

	idx := s.findDriverManifestIndexLocked(driverID, manifestID, date)
	if idx < 0 {
		return ManifestDetailSnapshot{}, false
	}
	return s.manifestDetailSnapshotLocked(s.manifests[idx]), true
}

func (s *Service) manifestDetailSnapshotLocked(manifest ManifestRow) ManifestDetailSnapshot {
	manifestID := manifest.ManifestID
	transfers := append([]TransferRow(nil), s.manifestTransfers[manifestID]...)
	transitions := append([]ManifestTransition(nil), s.manifestTransitions[manifestID]...)
	reassignments := append([]ManifestReassignment(nil), s.manifestReassignments[manifestID]...)
	exceptions := make([]ManifestException, 0)
	for i := range s.manifestExceptions {
		if s.manifestExceptions[i].ManifestID == manifestID {
			exceptions = append(exceptions, s.manifestExceptions[i])
		}
	}
	stopCount := manifest.TransferCnt
	if stopCount == 0 {
		stopCount = len(transfers)
	}
	return ManifestDetailSnapshot{
		Manifest:      manifest,
		Transfers:     transfers,
		Transitions:   transitions,
		Reassignments: reassignments,
		Exceptions:    exceptions,
		RouteID:       routeIDForManifest(manifest),
		StopCount:     stopCount,
		OrderCount:    len(transfers),
	}
}

func (s *Service) findDriverManifestIndexLocked(driverID, manifestID, date string) int {
	if manifestID != "" {
		idx := s.findManifestIndexLocked(manifestID)
		if idx < 0 {
			return -1
		}
		manifest := s.manifests[idx]
		if strings.TrimSpace(manifest.DriverID) != driverID {
			return -1
		}
		if !manifestMatchesDate(manifest, date) {
			return -1
		}
		return idx
	}

	bestIdx := -1
	bestRank := -1
	bestUpdatedAt := ""
	for i := range s.manifests {
		manifest := s.manifests[i]
		if strings.TrimSpace(manifest.DriverID) != driverID {
			continue
		}
		if !manifestMatchesDate(manifest, date) {
			continue
		}
		rank := manifestSelectionRank(manifest.State)
		if bestIdx < 0 || rank > bestRank || (rank == bestRank && manifest.UpdatedAt > bestUpdatedAt) {
			bestIdx = i
			bestRank = rank
			bestUpdatedAt = manifest.UpdatedAt
		}
	}
	return bestIdx
}

func manifestMatchesDate(manifest ManifestRow, date string) bool {
	if date == "" {
		return true
	}
	return strings.HasPrefix(manifest.CreatedAt, date) || strings.HasPrefix(manifest.UpdatedAt, date)
}

func manifestSelectionRank(state string) int {
	switch strings.TrimSpace(state) {
	case manifestStateDispatched:
		return 6
	case manifestStateSealed:
		return 5
	case manifestStateLoading:
		return 4
	case manifestStateDraft:
		return 3
	case manifestStateCompleted:
		return 2
	case manifestStateCancelled:
		return 1
	default:
		return 0
	}
}

func (s *Service) findTransferIndexLocked(transfers []TransferRow, transferID string) int {
	for i := range transfers {
		if transfers[i].TransferID == transferID {
			return i
		}
	}
	return -1
}

func (s *Service) findGlobalTransferIndexLocked(transferID string) int {
	for i := range s.transfers {
		if s.transfers[i].TransferID == transferID {
			return i
		}
	}
	return -1
}

func (s *Service) appendTransitionLocked(manifestID, action, fromState, toState, reason string) {
	s.manifestTransitions[manifestID] = append(s.manifestTransitions[manifestID], ManifestTransition{
		Action:    action,
		FromState: fromState,
		ToState:   toState,
		Reason:    strings.TrimSpace(reason),
		At:        s.now().Format(time.RFC3339Nano),
	})
}

func (s *Service) transitionManifestLocked(manifestID, expectedState, toState, action, reason string) (ManifestRow, error) {
	idx := s.findManifestIndexLocked(manifestID)
	if idx < 0 {
		return ManifestRow{}, fmt.Errorf("manifest_not_found")
	}
	row := s.manifests[idx]
	if row.State != expectedState {
		return ManifestRow{}, fmt.Errorf("invalid_state:%s", row.State)
	}
	now := s.now().Format(time.RFC3339Nano)
	row.State = toState
	row.UpdatedAt = now
	if toState == manifestStateLoading {
		row.LoadingStartedAt = now
	}
	if toState == manifestStateSealed {
		row.SealedAt = now
	}
	if toState == manifestStateDispatched {
		row.DispatchedAt = now
	}
	if toState == manifestStateCompleted {
		row.CompletedAt = now
	}

	transfers := s.manifestTransfers[manifestID]
	for i := range transfers {
		switch toState {
		case manifestStateLoading:
			if transfers[i].State == "ASSIGNED" || transfers[i].State == "CREATED" {
				transfers[i].State = "LOADING"
			}
		case manifestStateSealed:
			if transfers[i].State == "LOADING" || transfers[i].State == "ASSIGNED" {
				transfers[i].State = "SEALED"
			}
		case manifestStateDispatched:
			if transfers[i].State == "SEALED" {
				transfers[i].State = "DISPATCHED"
			}
		case manifestStateCompleted:
			if transfers[i].State != "CANCELLED" {
				transfers[i].State = "COMPLETED"
			}
		}
		transfers[i].UpdatedAt = now
	}
	s.manifestTransfers[manifestID] = transfers
	s.manifests[idx] = row
	s.appendTransitionLocked(manifestID, action, expectedState, toState, reason)
	return row, nil
}

type wsEnvelope struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Data      any    `json:"data"`
}

// resolveFactoryNode prefers JWT home-node / FactoryScope over bootstrap demo id (B2 M-P0-8).
func (s *Service) resolveFactoryNode(ctx context.Context) string {
	if id := auth.EffectiveFactoryID(ctx); strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	if claims, ok := auth.FromContext(ctx); ok {
		if claims.HomeNodeType == auth.HomeNodeFactory && strings.TrimSpace(claims.HomeNodeID) != "" {
			return strings.TrimSpace(claims.HomeNodeID)
		}
	}
	return s.factoryNodeID
}

func (s *Service) broadcastFactoryEvent(ctx context.Context, eventType string, data map[string]any) {
	envelope := wsEnvelope{
		Type:      eventType,
		Timestamp: s.now().Format(time.RFC3339Nano),
		Data:      data,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		s.log.Warn("factory ws marshal failed", "event_type", eventType, "err", err)
		return
	}
	if s.supplierHub != nil {
		s.supplierHub.Broadcast(ctx, "supplier:"+s.resolveSupplierScope(ctx), payload)
	}
	if s.factoryHub != nil {
		s.factoryHub.Broadcast(ctx, "factory:"+s.resolveFactoryNode(ctx), payload)
	}
}

func (s *Service) broadcastFactorySupplyEvent(ctx context.Context, envelope map[string]any) {
	eventType, _ := envelope["type"].(string)
	data, _ := envelope["data"].(map[string]any)
	if eventType == "" {
		return
	}
	if data == nil {
		data = map[string]any{}
	}
	s.broadcastFactoryEvent(ctx, eventType, data)
}

func (s *Service) manifestOutboxFields(ctx context.Context, manifest ManifestRow, eventType string) events.ManifestEvent {
	return events.ManifestEvent{
		BaseEvent:      events.BaseEvent{Type: eventType},
		ManifestID:     manifest.ManifestID,
		ManifestDomain: events.ManifestDomainFactory, // G2.D Option B — transfer bay SoT
		SupplierID:     s.resolveSupplierScope(ctx),
		FactoryID:      s.resolveFactoryNode(ctx),
		State:          manifest.State,
		DriverID:       manifest.DriverID,
		VehicleID:      manifest.VehicleID,
		TransferCount:  manifest.TransferCnt,
		RouteID:        routeIDForManifest(manifest),
	}
}

func (s *Service) invalidateFactoryKeys(ctx context.Context, keys ...string) {
	if s.cache == nil {
		return
	}
	if sid := strings.TrimSpace(s.resolveSupplierScope(ctx)); sid != "" {
		keys = append(keys, cache.DashboardKey("factory", sid))
	}
	if len(keys) == 0 {
		return
	}
	s.cache.Invalidate(ctx, keys...)
}

func factoryManifestKey(manifestID string) string {
	return "factory:manifest:" + manifestID
}

func factoryManifestListKey(supplierID string) string {
	return "factory:manifests:" + supplierID
}

func factoryTransferListKey(supplierID string) string {
	return "factory:transfers:" + supplierID
}

func factoryExceptionListKey(supplierID string) string {
	return "factory:manifest_exceptions:" + supplierID
}

func factoryStaffListKey(supplierID string) string {
	return "factory:staff:" + supplierID
}

// HandleAnalyticsOverview serves GET /v1/factory/analytics/overview.
func (s *Service) HandleAnalyticsOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.spannerClient != nil {
		overview, err := s.loadAnalyticsOverview(r.Context())
		if err == nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"daily_activity":     overview.DailyActivity,
				"transfers_total":    overview.TransfersTotal,
				"manifests_active":   overview.ManifestsActive,
				"exception_queue":    overview.ExceptionQueue,
				"avg_lead_time_mins": overview.AvgLeadTimeMins,
			})
			return
		}
		if s.log != nil {
			s.log.WarnContext(r.Context(), "factory analytics spanner read failed, using memory fallback", "err", err)
		}
	}
	s.mu.RLock()
	transfers := len(s.transfers)
	manifests := len(s.manifests)
	escalatedExceptions := 0
	for i := range s.manifestExceptions {
		if s.manifestExceptions[i].Escalated {
			escalatedExceptions++
		}
	}
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"daily_activity":     []any{},
		"transfers_total":    transfers,
		"manifests_active":   manifests,
		"exception_queue":    escalatedExceptions,
		"avg_lead_time_mins": 0,
	})
}

// HandleDashboard serves GET /v1/factory/dashboard.
func (s *Service) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.resolveSupplierScope(r.Context())
	key := cache.DashboardKey("factory", sid)
	body, err := cache.LoadDashboard(s.cache, r.Context(), key, func(ctx context.Context) ([]byte, error) {
		snap, source, loadErr := s.loadDashboardSnapshot(ctx)
		if loadErr != nil {
			return nil, loadErr
		}
		resp := buildFactoryDashboard(snap, source, sid, s.resolveFactoryNode(ctx), s.now())
		return json.Marshal(resp)
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dashboard_unavailable"})
		return
	}
	cache.WriteJSONWithETag(w, r, body)
}

// HandleProfile serves GET /v1/factory/profile.
func (s *Service) HandleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"factory_id":   s.factoryNodeID,
		"factory_name": "PegasusX Demo Factory",
		"supplier_id":  s.resolveSupplierScope(r.Context()),
		"currency":     s.currency,
		"updated_at":   s.now().Format(time.RFC3339Nano),
	})
}

// HandleTransfers serves GET /v1/factory/transfers and POST /v1/factory/transfers/create.
func (s *Service) HandleTransfers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		stateFilters := parseTransferStateFilters(r)
		limit := parseTransferLimit(r.URL.Query().Get("limit"))
		offset := parseTransferOffset(r.URL.Query().Get("offset"))

		if s.spannerClient != nil {
			rows, err := s.loadFactoryTransfersFromSpanner(r.Context())
			if err != nil {
				s.log.WarnContext(r.Context(), "factory transfer list failed", "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "transfer_list_failed"})
				return
			}
			if len(rows) > 0 || !s.portalSeedEnabled() {
				s.writeTransferList(w, filterTransferRows(rows, stateFilters), limit, offset)
				return
			}
		}

		s.mu.Lock()
		s.ensureDemoDataLocked()
		rows := filterTransferRows(append([]TransferRow(nil), s.transfers...), stateFilters)
		s.mu.Unlock()
		if !s.portalSeedEnabled() && s.spannerClient == nil {
			rows = nil
		}
		s.writeTransferList(w, rows, limit, offset)
	case http.MethodPost:
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

		var req transferCreateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		if req.TotalVU <= 0 {
			req.TotalVU = 25
		}
		// FactoryInternalTransfers.OrderId is STRING(36); oversize is a 400, not a Spanner 500.
		if oid := strings.TrimSpace(req.OrderID); len(oid) > 36 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_too_long"})
			return
		}
		var row TransferRow
		now := ""
		factoryID := s.resolveFactoryNode(r.Context())
		supplierID := s.resolveSupplierScope(r.Context())
		err = s.apply(r.Context(), func() error {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.ensureDemoDataLocked()
			now = s.now().Format(time.RFC3339Nano)
			row = TransferRow{
				TransferID: s.nextIDLocked("tr"),
				OrderID:    strings.TrimSpace(req.OrderID),
				State:      "CREATED",
				TotalVU:    req.TotalVU,
				DriverID:   strings.TrimSpace(req.DriverID),
				VehicleID:  strings.TrimSpace(req.VehicleID),
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			s.transfers = append(s.transfers, row)
			return nil
		}, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateFactory, row.TransferID, events.TopicMain, events.WarehouseTransferEvent{
				BaseEvent:  events.BaseEvent{Type: events.EventFactoryTransferCreated},
				TransferID: row.TransferID,
				FactoryID:  factoryID,
				SupplierID: supplierID,
				State:      row.State,
				Status:     row.State,
			})
		})
		if err != nil {
			s.log.ErrorContext(r.Context(), "factory.transfer_create_failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "transfer_create_failed"})
			return
		}
		s.invalidateFactoryKeys(r.Context(), factoryTransferListKey(supplierID))
		s.broadcastFactoryEvent(r.Context(), events.EventFactoryTransferCreated, map[string]any{
			"transfer_id": row.TransferID,
			"factory_id":  factoryID,
			"supplier_id": supplierID,
			"state":       row.State,
		})
		s.log.InfoContext(r.Context(), "factory.transfer_created", "transfer_id", row.TransferID, "factory_id", factoryID)
		idemCommitted = true
		s.writeIdempotentJSON(w, r, body, http.StatusCreated, row)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) writeTransferList(w http.ResponseWriter, rows []TransferRow, limit, offset int) {
	sort.Slice(rows, func(i, j int) bool { return rows[i].UpdatedAt > rows[j].UpdatedAt })
	total := len(rows)
	if offset >= len(rows) {
		rows = nil
	} else {
		rows = rows[offset:]
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	mapped := make([]map[string]any, len(rows))
	for i := range rows {
		mapped[i] = s.iosTransferPayload(rows[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"transfers": mapped, "total": total})
}

func (s *Service) loadFactoryTransfersFromSpanner(ctx context.Context) ([]TransferRow, error) {
	fid := strings.TrimSpace(s.resolveFactoryNode(ctx))
	sid := strings.TrimSpace(s.resolveSupplierScope(ctx))
	if fid == "" || sid == "" || s.spannerClient == nil {
		return nil, fmt.Errorf("factory transfer scope required")
	}
	stmt := spanner.Statement{
		SQL: `SELECT TransferId, OrderId, ManifestId, State, TotalVolumeVU,
		      DriverId, VehicleId, ReassignDepth, ExceptionCount, CreatedAt, UpdatedAt
		      FROM FactoryInternalTransfers@{FORCE_INDEX=Idx_FactoryTransfers_ByFactoryId}
		      WHERE FactoryId = @fid AND SupplierId = @sid
		      ORDER BY UpdatedAt DESC`,
		Params: map[string]any{"fid": fid, "sid": sid},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []TransferRow
	for {
		row, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, fmt.Errorf("factory transfer query: %w", err)
		}
		var t TransferRow
		var orderID, manifestID, driverID, vehicleID spanner.NullString
		var totalVolume float64
		var reassignDepth, exceptionCount int64
		var createdAt, updatedAt time.Time
		if err := row.Columns(&t.TransferID, &orderID, &manifestID, &t.State, &totalVolume, &driverID, &vehicleID,
			&reassignDepth, &exceptionCount, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("factory transfer scan: %w", err)
		}
		t.OrderID = orderID.StringVal
		t.ManifestID = manifestID.StringVal
		t.TotalVU = int64(totalVolume)
		t.DriverID = driverID.StringVal
		t.VehicleID = vehicleID.StringVal
		t.ReassignDepth = int(reassignDepth)
		t.ExceptionCount = exceptionCount
		t.CreatedAt = createdAt.Format(time.RFC3339Nano)
		t.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
		out = append(out, t)
	}
	return out, nil
}

func (s *Service) loadFactoryManifestsFromSpanner(ctx context.Context) ([]ManifestRow, error) {
	fid := strings.TrimSpace(s.resolveFactoryNode(ctx))
	sid := strings.TrimSpace(s.resolveSupplierScope(ctx))
	if fid == "" || sid == "" || s.spannerClient == nil {
		return nil, fmt.Errorf("factory manifest scope required")
	}
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId, State, TotalVolumeVU, MaxVolumeVU, StopCount, TransferCount,
		      DriverId, VehicleId, CreatedAt, UpdatedAt
		      FROM FactoryTruckManifests
		      WHERE FactoryId = @fid AND SupplierId = @sid
		      ORDER BY UpdatedAt DESC`,
		Params: map[string]any{"fid": fid, "sid": sid},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []ManifestRow
	for {
		row, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, fmt.Errorf("factory manifest query: %w", err)
		}
		var m ManifestRow
		var totalVolume, maxVolume float64
		var stopCount, transferCount int64
		var driverID, vehicleID spanner.NullString
		var createdAt, updatedAt time.Time
		if err := row.Columns(&m.ManifestID, &m.State, &totalVolume, &maxVolume, &stopCount, &transferCount,
			&driverID, &vehicleID, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("factory manifest scan: %w", err)
		}
		m.TotalVolumeVU = int64(totalVolume)
		m.MaxVolumeVU = int64(maxVolume)
		m.TransferCnt = int(transferCount)
		m.StopCount = int(stopCount)
		m.DriverID = driverID.StringVal
		m.VehicleID = vehicleID.StringVal
		m.TruckID = m.VehicleID
		m.CreatedAt = createdAt.Format(time.RFC3339Nano)
		m.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
		out = append(out, m)
	}
	return out, nil
}

func (s *Service) hydrateFactoryManifestsFromSpanner(ctx context.Context) error {
	if s == nil || s.spannerClient == nil {
		return nil
	}
	manifests, err := s.loadFactoryManifestsFromSpanner(ctx)
	if err != nil {
		return err
	}
	transfers, err := s.loadFactoryTransfersFromSpanner(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.manifests = manifests
	s.transfers = transfers
	s.rebuildManifestTransfersLocked()
	s.mu.Unlock()
	return nil
}

// HandleManifests serves GET /v1/factory/manifests.
func (s *Service) HandleManifests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	state := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("state")))

	if err := s.hydrateFactoryManifestsFromSpanner(r.Context()); err != nil {
		s.log.WarnContext(r.Context(), "factory manifest list hydrate failed", "err", err)
	}

	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]ManifestRow(nil), s.manifests...)
	s.mu.Unlock()

	if state != "" {
		filtered := make([]ManifestRow, 0, len(rows))
		for i := range rows {
			if rows[i].State == state {
				filtered = append(filtered, rows[i])
			}
		}
		rows = filtered
	}
	for i := range rows {
		if rows[i].TruckID == "" {
			rows[i].TruckID = rows[i].VehicleID
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].UpdatedAt > rows[j].UpdatedAt })
	writeJSON(w, http.StatusOK, map[string]any{"manifests": rows})
}

// HandleManifestDetail serves GET /v1/factory/manifests/{manifestID}.
func (s *Service) HandleManifestDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	manifestID := strings.TrimSpace(chi.URLParam(r, "manifestID"))
	if manifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}

	if err := s.hydrateFactoryManifestsFromSpanner(r.Context()); err != nil {
		s.log.WarnContext(r.Context(), "factory manifest detail hydrate failed", "err", err)
	}

	s.mu.Lock()
	s.ensureDemoDataLocked()
	idx := s.findManifestIndexLocked(manifestID)
	if idx < 0 {
		s.mu.Unlock()
		platform.WriteErrorWithExplain(w, http.StatusNotFound, "manifest_not_found", nil)
		return
	}
	snapshot := s.manifestDetailSnapshotLocked(s.manifests[idx])
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"manifest":      snapshot.Manifest,
		"transfers":     snapshot.Transfers,
		"transitions":   snapshot.Transitions,
		"reassignments": snapshot.Reassignments,
		"exceptions":    snapshot.Exceptions,
		"route_id":      snapshot.RouteID,
		"stop_count":    snapshot.StopCount,
		"order_count":   snapshot.OrderCount,
	})
}

// HandleManifestStartLoading serves POST /v1/factory/manifests/{manifestID}/start-loading.
func (s *Service) HandleManifestStartLoading(w http.ResponseWriter, r *http.Request) {
	s.handleManifestTransition(w, r, manifestStateDraft, manifestStateLoading, "START_LOADING")
}

// HandleManifestSeal serves POST /v1/factory/manifests/{manifestID}/seal.
func (s *Service) HandleManifestSeal(w http.ResponseWriter, r *http.Request) {
	s.handleManifestTransition(w, r, manifestStateLoading, manifestStateSealed, "SEAL")
}

// HandleManifestDispatch serves POST /v1/factory/manifests/{manifestID}/dispatch.
func (s *Service) HandleManifestDispatch(w http.ResponseWriter, r *http.Request) {
	s.handleManifestTransition(w, r, manifestStateSealed, manifestStateDispatched, "DISPATCH")
}

// HandleManifestComplete serves POST /v1/factory/manifests/{manifestID}/complete.
func (s *Service) HandleManifestComplete(w http.ResponseWriter, r *http.Request) {
	s.handleManifestTransition(w, r, manifestStateDispatched, manifestStateCompleted, "COMPLETE")
}

func (s *Service) handleManifestTransition(w http.ResponseWriter, r *http.Request, expectedState, toState, action string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	manifestID := strings.TrimSpace(chi.URLParam(r, "manifestID"))
	if manifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
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

	var req transitionRequest
	if len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}

	eventType := ""
	switch toState {
	case manifestStateLoading:
		eventType = events.EventManifestLoadingStarted
	case manifestStateSealed:
		eventType = events.EventManifestSealed
	case manifestStateDispatched:
		eventType = events.EventManifestDispatched
	case manifestStateCompleted:
		eventType = events.EventManifestCompleted
	}

	var manifest ManifestRow
	err = s.apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		updated, transitionErr := s.transitionManifestLocked(manifestID, expectedState, toState, action, req.Reason)
		if transitionErr != nil {
			return transitionErr
		}
		manifest = updated
		return nil
	}, func(txn outbox.TxnBuffer) error {
		if eventType == "" {
			return nil
		}
		payload := s.manifestOutboxFields(r.Context(), manifest, eventType)
		payload.Reason = strings.TrimSpace(req.Reason)
		payload.Action = action
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, manifest.ManifestID, events.TopicMain, payload)
	})
	if err != nil {
		if strings.HasPrefix(err.Error(), "invalid_state:") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_state", "message": err.Error()})
			return
		}
		if err.Error() == "manifest_not_found" {
			platform.WriteErrorWithExplain(w, http.StatusNotFound, "manifest_not_found", err)
			return
		}
		platform.WriteErrorWithExplain(w, http.StatusInternalServerError, "transition_failed", err)
		return
	}

	s.invalidateFactoryKeys(r.Context(), factoryManifestKey(manifest.ManifestID), factoryManifestListKey(s.resolveSupplierScope(r.Context())), factoryTransferListKey(s.resolveSupplierScope(r.Context())))
	if eventType != "" {
		s.broadcastFactoryEvent(r.Context(), eventType, map[string]any{
			"manifest_id": manifest.ManifestID,
			"state":       manifest.State,
			"action":      action,
			"reason":      strings.TrimSpace(req.Reason),
			"route_id":    routeIDForManifest(manifest),
			"driver_id":   manifest.DriverID,
			"vehicle_id":  manifest.VehicleID,
			"order_count": manifest.TransferCnt,
			"updated_at":  manifest.UpdatedAt,
		})
	}

	resp := map[string]any{
		"status":      strings.ToLower(action) + "_applied",
		"manifest_id": manifest.ManifestID,
		"state":       manifest.State,
		"updated_at":  manifest.UpdatedAt,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

type factoryFleetSpannerResult struct {
	IOSVehicles   []map[string]any
	FleetVehicles []FleetVehicle
}

func (s *Service) loadFactoryFleetFromSpanner(ctx context.Context, factoryID string) (*factoryFleetSpannerResult, error) {
	if s.spannerClient == nil {
		return nil, nil
	}
	factoryID = strings.TrimSpace(factoryID)
	if factoryID == "" {
		factoryID = s.factoryNodeID
	}
	if factoryID == "" {
		return nil, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT 
				v.VehicleId,
				v.LicensePlate,
				v.MaxVolumeVU,
				v.IsActive,
				IFNULL(v.UnavailableReason, ''),
				IFNULL(m.ManifestId, ''),
				IFNULL(m.State, ''),
				IFNULL(m.DriverId, ''),
				IFNULL(d.Name, '')
		      FROM Vehicles v
		      LEFT JOIN FactoryTruckManifests m 
		        ON m.VehicleId = v.VehicleId 
		       AND m.FactoryId = @fid 
		       AND m.State IN ('LOADING', 'SEALED', 'DISPATCHED')
		      LEFT JOIN Drivers d 
		        ON d.DriverId = m.DriverId
		      WHERE v.HomeNodeType = 'FACTORY' 
		        AND v.HomeNodeId = @fid
		      ORDER BY v.VehicleId ASC`,
		Params: map[string]any{"fid": factoryID},
	}
	iter := s.spannerClient.Single().
		WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).
		Query(ctx, stmt)
	defer iter.Stop()

	iosList := make([]map[string]any, 0, 8)
	fleetList := make([]FleetVehicle, 0, 8)

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query factory fleet spanner: %w", err)
		}
		var (
			vehicleID, licensePlate, unavailReason          string
			manifestID, manifestState, driverID, driverName string
			maxVolumeVU                                     float64
			isActive                                        bool
		)
		if err := row.Columns(&vehicleID, &licensePlate, &maxVolumeVU, &isActive, &unavailReason, &manifestID, &manifestState, &driverID, &driverName); err != nil {
			return nil, fmt.Errorf("scan factory fleet spanner: %w", err)
		}

		status := "READY"
		if !isActive {
			if strings.TrimSpace(unavailReason) != "" {
				status = strings.ToUpper(strings.TrimSpace(unavailReason))
			} else {
				status = "UNAVAILABLE"
			}
		} else if strings.TrimSpace(manifestState) != "" {
			status = strings.ToUpper(strings.TrimSpace(manifestState))
		}

		fleetList = append(fleetList, FleetVehicle{
			VehicleID: vehicleID,
			PlateNo:   licensePlate,
			State:     status,
		})

		capM3 := maxVolumeVU * 0.1
		if capM3 <= 0 {
			capM3 = 12.0
		}
		capKg := maxVolumeVU * 25.0
		if capKg <= 0 {
			capKg = 3200.0
		}
		capL := maxVolumeVU * 100.0
		if capL <= 0 {
			capL = 12000.0
		}

		routeID := ""
		currentRoute := ""
		if strings.TrimSpace(manifestID) != "" {
			routeID = "route_" + strings.TrimSpace(manifestID)
			currentRoute = "Manifest " + strings.TrimSpace(manifestID)
		}

		iosList = append(iosList, map[string]any{
			"id":               vehicleID,
			"plate_number":     licensePlate,
			"capacity_m3":      capM3,
			"capacity_kg":      capKg,
			"capacity_l":       capL,
			"status":           status,
			"driver_name":      strings.TrimSpace(driverName),
			"current_route_id": routeID,
			"current_route":    currentRoute,
			"manifest_id":      strings.TrimSpace(manifestID),
			"driver_id":        strings.TrimSpace(driverID),
		})
	}

	return &factoryFleetSpannerResult{
		IOSVehicles:   iosList,
		FleetVehicles: fleetList,
	}, nil
}

// HandleFleetDrivers serves GET /v1/factory/fleet/drivers.
func (s *Service) HandleFleetDrivers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]FleetDriver(nil), s.fleetDrivers...)
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"drivers": rows})
}

// HandleFleetVehicles serves GET /v1/factory/fleet/vehicles.
func (s *Service) HandleFleetVehicles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	factoryID, _ := s.scopedFactoryID(r)
	if factoryID == "" {
		factoryID = s.factoryNodeID
	}
	if s.spannerClient != nil {
		res, err := s.loadFactoryFleetFromSpanner(r.Context(), factoryID)
		if err != nil {
			s.log.ErrorContext(r.Context(), "factory fleet vehicles spanner failed", "err", err, "factory_id", factoryID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fleet_vehicles_failed"})
			return
		}
		if res != nil {
			writeJSON(w, http.StatusOK, map[string]any{"vehicles": res.FleetVehicles})
			return
		}
	}
	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]FleetVehicle(nil), s.fleetVehicles...)
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"vehicles": rows})
}

// HandleStaff serves GET /v1/factory/staff and POST /v1/factory/staff (create).
func (s *Service) HandleStaff(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListStaff(w, r)
		return
	case http.MethodPost:
		s.handleCreateStaff(w, r)
		return
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleListStaff(w http.ResponseWriter, r *http.Request) {
	if s.spannerClient != nil {
		rows, err := s.loadFactoryStaffFromSpanner(r.Context())
		if err != nil {
			s.log.WarnContext(r.Context(), "factory staff list failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "staff_list_failed"})
			return
		}
		if len(rows) > 0 || !s.portalSeedEnabled() {
			mapped := make([]map[string]any, len(rows))
			for i := range rows {
				mapped[i] = iosStaffMemberPayload(rows[i])
			}
			writeJSON(w, http.StatusOK, map[string]any{"staff": mapped})
			return
		}
	}
	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]StaffRow(nil), s.staff...)
	s.mu.Unlock()
	if !s.portalSeedEnabled() && s.spannerClient == nil {
		rows = nil
	}
	mapped := make([]map[string]any, len(rows))
	for i := range rows {
		mapped[i] = iosStaffMemberPayload(rows[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"staff": mapped})
}

func (s *Service) loadFactoryStaffFromSpanner(ctx context.Context) ([]StaffRow, error) {
	fid := strings.TrimSpace(s.resolveFactoryNode(ctx))
	sid := strings.TrimSpace(s.resolveSupplierScope(ctx))
	if fid == "" || sid == "" || s.spannerClient == nil {
		return nil, fmt.Errorf("factory staff scope required")
	}
	stmt := spanner.Statement{
		SQL: `SELECT UserId, Name, COALESCE(Phone, ''), SupplierRole
		      FROM SupplierUsers@{FORCE_INDEX=Idx_SupplierUsers_BySupplierUpdated}
		      WHERE SupplierId = @sid AND AssignedFactoryId = @fid AND IsActive = true
		      ORDER BY UpdatedAt DESC`,
		Params: map[string]any{"sid": sid, "fid": fid},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []StaffRow
	for {
		row, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, fmt.Errorf("factory staff query: %w", err)
		}
		var rec StaffRow
		if err := row.Columns(&rec.StaffID, &rec.Name, &rec.Phone, &rec.Role); err != nil {
			return nil, fmt.Errorf("factory staff scan: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func (s *Service) handleCreateStaff(w http.ResponseWriter, r *http.Request) {
	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	var req struct {
		Name     string `json:"name"`
		Role     string `json:"role"`
		Phone    string `json:"phone"`
		PIN      string `json:"pin"`
		Password string `json:"password"`
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Role = strings.TrimSpace(req.Role)
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name_required"})
		return
	}
	if req.Role == "" {
		req.Role = "FACTORY_OPERATOR"
	}
	if !factoryStaffRoleAllowed(req.Role) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "role_invalid"})
		return
	}
	secret, invite, err := resolveStaffSecret(req.PIN, req.Password)
	if err != nil {
		if errors.Is(err, errStaffSecretTooShort) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pin_or_password_too_short"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "staff_secret_failed"})
		return
	}
	hash, err := hashFactoryStaffSecret(secret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "staff_password_hash_failed"})
		return
	}

	supplierID := s.resolveSupplierScope(r.Context())
	factoryID := s.resolveFactoryNode(r.Context())
	var row StaffRow
	err = s.repo.RunTx(r.Context(), func(ctx context.Context, tx FactoryTx) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		row = StaffRow{
			StaffID:      s.nextIDLocked("stf"),
			Name:         req.Name,
			Role:         req.Role,
			Phone:        req.Phone,
			PasswordHash: hash,
		}
		if err := tx.SaveStaff(ctx, row); err != nil {
			return err
		}
		s.staff = append(s.staff, row)
		return nil
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateFactory, row.StaffID, events.TopicMain, events.FactoryEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventFactoryStaffCreated},
			FactoryID:    factoryID,
			SupplierID:   supplierID,
			UserID:       row.StaffID,
			SupplierRole: row.Role,
		})
	})
	if err != nil {
		s.log.ErrorContext(r.Context(), "factory staff create failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "staff_create_failed"})
		return
	}
	s.invalidateFactoryKeys(r.Context(), factoryStaffListKey(supplierID))
	s.broadcastFactoryEvent(r.Context(), events.EventFactoryStaffCreated, map[string]any{
		"staff_id":    row.StaffID,
		"factory_id":  factoryID,
		"supplier_id": supplierID,
		"role":        row.Role,
	})
	s.log.InfoContext(r.Context(), "factory.staff_created", "staff_id", row.StaffID, "factory_id", factoryID)
	payload := iosStaffMemberPayload(row)
	payload["must_set_password"] = invite != ""
	if invite != "" {
		payload["invite_token"] = invite
	}
	s.writeIdempotentJSON(w, r, body, http.StatusCreated, payload)
}

func factoryStaffRoleAllowed(role string) bool {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case "FACTORY", "FACTORY_ADMIN", "FACTORY_STAFF", "FACTORY_OPERATOR", "FACTORY_DRIVER":
		return true
	default:
		return false
	}
}

var errNoDispatchableTransfers = errors.New("no_dispatchable_transfers")

func pickDispatchTransfers(transfers []TransferRow, req dispatchRequest) []TransferRow {
	available := make([]TransferRow, 0)
	for i := range transfers {
		if transfers[i].State == "CREATED" {
			available = append(available, transfers[i])
		}
	}
	selected := make([]TransferRow, 0)
	if len(req.TransferIDs) > 0 {
		for _, id := range req.TransferIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			for i := range transfers {
				if transfers[i].TransferID != id {
					continue
				}
				if dispatchableTransferState(transfers[i].State) && strings.TrimSpace(transfers[i].ManifestID) == "" {
					selected = append(selected, transfers[i])
				}
				break
			}
		}
	}
	if len(selected) == 0 {
		limit := len(available)
		if limit > 2 {
			limit = 2
		}
		if limit > 0 {
			selected = append(selected, available[:limit]...)
		}
	}
	return selected
}

// HandleDispatch serves POST /v1/factory/dispatch.
func (s *Service) HandleDispatch(w http.ResponseWriter, r *http.Request) {
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
	if s.spannerClient != nil {
		s.handleSolverDispatch(w, r, body)
		return
	}
	var req dispatchRequest
	if len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}

	s.mu.Lock()
	s.ensureDemoDataLocked()
	preview := pickDispatchTransfers(s.transfers, req)
	s.mu.Unlock()
	if len(preview) == 0 {
		nowTS := s.now().Format(time.RFC3339Nano)
		s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
			"status":                 "dispatch_planned",
			"created_manifest_count": 0,
			"manifests_created":      0,
			"manifest_id":            "",
			"transfer_count":         0,
			"updated_at":             nowTS,
			"optimizer_class":        OptimizerHeuristic,
			"dispatch_algo":          DispatchAlgoPickN,
		})
		return
	}

	var manifest ManifestRow
	selectedCount := 0
	now := ""
	err = s.apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		now = s.now().Format(time.RFC3339Nano)

		selected := pickDispatchTransfers(s.transfers, req)
		if len(selected) == 0 {
			return errNoDispatchableTransfers
		}

		if req.MaxVolumeVU <= 0 {
			req.MaxVolumeVU = 180
		}
		if strings.TrimSpace(req.DriverID) == "" {
			if len(s.fleetDrivers) == 0 {
				return errNoDispatchableTransfers
			}
			req.DriverID = s.fleetDrivers[0].DriverID
		}
		if strings.TrimSpace(req.VehicleID) == "" {
			if len(s.fleetVehicles) == 0 {
				return errNoDispatchableTransfers
			}
			req.VehicleID = s.fleetVehicles[0].VehicleID
		}

		manifestID := s.nextIDLocked("mf")
		var total int64
		for i := range selected {
			total += selected[i].TotalVU
		}

		manifest = ManifestRow{
			ManifestID:    manifestID,
			State:         manifestStateDraft,
			TransferCnt:   len(selected),
			TotalVolumeVU: total,
			MaxVolumeVU:   req.MaxVolumeVU,
			DriverID:      req.DriverID,
			VehicleID:     req.VehicleID,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		s.manifests = append(s.manifests, manifest)

		transfers := make([]TransferRow, 0, len(selected))
		for _, transfer := range selected {
			transfer.ManifestID = manifestID
			transfer.DriverID = req.DriverID
			transfer.VehicleID = req.VehicleID
			transfer.State = "ASSIGNED"
			transfer.UpdatedAt = now
			transfers = append(transfers, transfer)
			for i := range s.transfers {
				if s.transfers[i].TransferID == transfer.TransferID {
					s.transfers[i] = transfer
					break
				}
			}
		}
		s.manifestTransfers[manifestID] = transfers
		s.appendTransitionLocked(manifestID, "CREATE_DRAFT", "", manifestStateDraft, req.Reason)
		selectedCount = len(selected)
		return nil
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, manifest.ManifestID, events.TopicMain, events.ManifestEvent{
			BaseEvent:     events.BaseEvent{Type: events.EventManifestDraftCreated},
			ManifestID:    manifest.ManifestID,
			SupplierID:    s.resolveSupplierScope(r.Context()),
			FactoryID:     s.factoryNodeID,
			RouteID:       routeIDForManifest(manifest),
			TransferCount: manifest.TransferCnt,
			TotalVolumeVU: manifest.TotalVolumeVU,
			DriverID:      manifest.DriverID,
			VehicleID:     manifest.VehicleID,
		})
	})
	if errors.Is(err, errNoDispatchableTransfers) {
		nowTS := s.now().Format(time.RFC3339Nano)
		s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
			"status":                 "dispatch_planned",
			"created_manifest_count": 0,
			"manifests_created":      0,
			"manifest_id":            "",
			"transfer_count":         0,
			"updated_at":             nowTS,
			"optimizer_class":        OptimizerHeuristic,
			"dispatch_algo":          DispatchAlgoPickN,
		})
		return
	}
	if err != nil {
		platform.WriteErrorWithExplain(w, http.StatusInternalServerError, "dispatch_failed", err)
		return
	}

	s.invalidateFactoryKeys(r.Context(), factoryManifestKey(manifest.ManifestID), factoryManifestListKey(s.resolveSupplierScope(r.Context())), factoryTransferListKey(s.resolveSupplierScope(r.Context())))
	s.broadcastFactoryEvent(r.Context(), events.EventManifestDraftCreated, map[string]any{
		"manifest_id":       manifest.ManifestID,
		"transfer_count":    manifest.TransferCnt,
		"total_volume_vu":   manifest.TotalVolumeVU,
		"route_id":          routeIDForManifest(manifest),
		"driver_id":         manifest.DriverID,
		"vehicle_id":        manifest.VehicleID,
		"created_manifests": 1,
		"updated_at":        now,
	})

	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"status":                 "dispatch_planned",
		"created_manifest_count": 1,
		"manifests_created":      1,
		"manifest_id":            manifest.ManifestID,
		"transfer_count":         selectedCount,
		"updated_at":             now,
		"optimizer_class":        OptimizerHeuristic,
		"dispatch_algo":          DispatchAlgoPickN,
	})
}

// SetPlanning attaches the P5 planning engine (batcher / SLA / pull matrix).
func (s *Service) SetPlanning(p *PlanningService) {
	if s != nil {
		s.planning = p
	}
}

func (s *Service) handleBatcherDispatch(w http.ResponseWriter, r *http.Request, body []byte) {
	if s.planning == nil || s.planning.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "batcher_unavailable"})
		return
	}
	factoryID := s.resolveFactoryNode(r.Context())
	supplierID := s.resolveSupplierScope(r.Context())
	result, err := s.planning.RunBatchDispatch(r.Context(), factoryID, supplierID)
	if err != nil {
		platform.WriteErrorWithExplain(w, http.StatusInternalServerError, "dispatch_failed", err)
		return
	}
	manifestID := ""
	if len(result.ManifestIDs) > 0 {
		manifestID = result.ManifestIDs[0]
	}
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"status":                 "dispatch_planned",
		"created_manifest_count": result.CreatedManifestCount,
		"manifests_created":      result.CreatedManifestCount,
		"manifest_id":            manifestID,
		"manifest_ids":           result.ManifestIDs,
		"unassigned":             result.Unassigned,
		"optimizer_class":        result.OptimizerClass,
		"dispatch_algo":          result.DispatchAlgo,
	})
}

// HandleManifestRebalance serves POST /v1/factory/manifests/rebalance.
func (s *Service) HandleManifestRebalance(w http.ResponseWriter, r *http.Request) {
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
	var req manifestRebalanceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.ManifestID = strings.TrimSpace(req.ManifestID)
	req.TransferID = strings.TrimSpace(req.TransferID)
	req.ToDriverID = strings.TrimSpace(req.ToDriverID)
	req.ToVehicle = strings.TrimSpace(req.ToVehicle)
	req.SourceManifestID = strings.TrimSpace(req.SourceManifestID)
	req.TargetManifestID = strings.TrimSpace(req.TargetManifestID)
	if req.SourceManifestID == "" {
		req.SourceManifestID = req.ManifestID
	}
	if len(req.TransferIDs) == 0 && req.TransferID != "" {
		req.TransferIDs = []string{req.TransferID}
	}
	if req.TargetManifestID != "" && req.TargetManifestID != req.SourceManifestID {
		s.handleCrossManifestRebalance(w, r, body, req)
		return
	}
	if req.ManifestID == "" || req.TransferID == "" || (req.ToDriverID == "" && req.ToVehicle == "") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_transfer_id_and_target_required"})
		return
	}

	var manifest ManifestRow
	var reassign ManifestReassignment
	err = s.apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		mIdx := s.findManifestIndexLocked(req.ManifestID)
		if mIdx < 0 {
			return fmt.Errorf("manifest_not_found")
		}
		manifest = s.manifests[mIdx]
		if manifest.State != manifestStateDraft && manifest.State != manifestStateLoading {
			return fmt.Errorf("manifest_not_mutable")
		}

		transfers := s.manifestTransfers[req.ManifestID]
		tIdx := s.findTransferIndexLocked(transfers, req.TransferID)
		if tIdx < 0 {
			return fmt.Errorf("transfer_not_found")
		}
		if transfers[tIdx].State != "ASSIGNED" && transfers[tIdx].State != "REASSIGNED" {
			return fmt.Errorf("transfer_not_mutable")
		}
		globalTransferIdx := s.findGlobalTransferIndexLocked(req.TransferID)
		if globalTransferIdx < 0 {
			return fmt.Errorf("transfer_not_found")
		}
		globalTransfer := s.transfers[globalTransferIdx]
		if globalTransfer.ManifestID != "" && globalTransfer.ManifestID != req.ManifestID {
			return fmt.Errorf("transfer_manifest_mismatch")
		}
		if globalTransfer.OrderID != transfers[tIdx].OrderID || globalTransfer.State != transfers[tIdx].State {
			return fmt.Errorf("transfer_ledger_mismatch")
		}
		if globalTransfer.DriverID != "" && transfers[tIdx].DriverID != "" && globalTransfer.DriverID != transfers[tIdx].DriverID {
			return fmt.Errorf("transfer_ledger_mismatch")
		}
		if globalTransfer.VehicleID != "" && transfers[tIdx].VehicleID != "" && globalTransfer.VehicleID != transfers[tIdx].VehicleID {
			return fmt.Errorf("transfer_ledger_mismatch")
		}
		if manifest.VehicleID != "" && transfers[tIdx].VehicleID != "" && transfers[tIdx].VehicleID != manifest.VehicleID {
			return fmt.Errorf("transfer_route_mismatch")
		}

		fromDriver := transfers[tIdx].DriverID
		fromVehicle := transfers[tIdx].VehicleID
		if (req.ToDriverID == "" || req.ToDriverID == fromDriver) && (req.ToVehicle == "" || req.ToVehicle == fromVehicle) {
			return fmt.Errorf("transfer_already_assigned")
		}
		if req.ToDriverID != "" {
			transfers[tIdx].DriverID = req.ToDriverID
		}
		if req.ToVehicle != "" {
			if manifest.VehicleID != "" && req.ToVehicle != manifest.VehicleID {
				return fmt.Errorf("transfer_route_mismatch")
			}
			transfers[tIdx].VehicleID = req.ToVehicle
		}
		if manifest.VehicleID != "" && transfers[tIdx].VehicleID != "" && transfers[tIdx].VehicleID != manifest.VehicleID {
			return fmt.Errorf("transfer_route_mismatch")
		}
		transfers[tIdx].State = "REASSIGNED"
		transfers[tIdx].ReassignDepth++
		transfers[tIdx].UpdatedAt = s.now().Format(time.RFC3339Nano)
		s.manifestTransfers[req.ManifestID] = transfers

		s.transfers[globalTransferIdx].DriverID = transfers[tIdx].DriverID
		s.transfers[globalTransferIdx].VehicleID = transfers[tIdx].VehicleID
		s.transfers[globalTransferIdx].ReassignDepth = transfers[tIdx].ReassignDepth
		s.transfers[globalTransferIdx].State = transfers[tIdx].State
		s.transfers[globalTransferIdx].UpdatedAt = transfers[tIdx].UpdatedAt

		manifest.ReassignmentDepth++
		manifest.UpdatedAt = s.now().Format(time.RFC3339Nano)
		s.manifests[mIdx] = manifest

		reassign = ManifestReassignment{
			ManifestID:      req.ManifestID,
			TransferID:      req.TransferID,
			FromDriverID:    fromDriver,
			ToDriverID:      transfers[tIdx].DriverID,
			FromVehicleID:   fromVehicle,
			ToVehicleID:     transfers[tIdx].VehicleID,
			Depth:           transfers[tIdx].ReassignDepth,
			Reason:          strings.TrimSpace(req.Reason),
			ReassignedAt:    s.now().Format(time.RFC3339Nano),
			Recommendation:  "load_balanced",
			AppliedBy:       strings.TrimSpace(req.AppliedBy),
			CorrelationHint: req.ManifestID + ":" + req.TransferID,
		}
		s.manifestReassignments[req.ManifestID] = append(s.manifestReassignments[req.ManifestID], reassign)
		return nil
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, events.ManifestEvent{
			BaseEvent:     events.BaseEvent{Type: events.EventManifestRebalanced},
			ManifestID:    req.ManifestID,
			TransferID:    req.TransferID,
			SupplierID:    s.resolveSupplierScope(r.Context()),
			FactoryID:     s.factoryNodeID,
			FromDriverID:  reassign.FromDriverID,
			ToDriverID:    reassign.ToDriverID,
			FromVehicleID: reassign.FromVehicleID,
			ToVehicleID:   reassign.ToVehicleID,
			Depth:         reassign.Depth,
			Reason:        reassign.Reason,
		})
	})
	if err != nil {
		switch err.Error() {
		case "manifest_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
			return
		case "manifest_not_mutable":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "manifest_not_mutable"})
			return
		case "transfer_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "transfer_not_found"})
			return
		case "transfer_manifest_mismatch":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "transfer_manifest_mismatch"})
			return
		case "transfer_ledger_mismatch":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "transfer_ledger_mismatch"})
			return
		case "transfer_route_mismatch":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "transfer_route_mismatch"})
			return
		case "transfer_not_mutable":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "transfer_not_mutable"})
			return
		case "transfer_already_assigned":
			s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{"status": "already_assigned", "manifest_id": req.ManifestID, "transfer_id": req.TransferID})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "manifest_rebalance_failed"})
			return
		}
	}

	s.invalidateFactoryKeys(r.Context(), factoryManifestKey(req.ManifestID), factoryManifestListKey(s.resolveSupplierScope(r.Context())), factoryTransferListKey(s.resolveSupplierScope(r.Context())))
	s.broadcastFactoryEvent(r.Context(), events.EventManifestRebalanced, map[string]any{
		"manifest_id":            req.ManifestID,
		"transfer_id":            req.TransferID,
		"from_driver_id":         reassign.FromDriverID,
		"to_driver_id":           reassign.ToDriverID,
		"from_vehicle_id":        reassign.FromVehicleID,
		"to_vehicle_id":          reassign.ToVehicleID,
		"reassignment_depth":     reassign.Depth,
		"manifest_reassignments": manifest.ReassignmentDepth,
	})

	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"status":                 "manifest_rebalanced",
		"manifest_id":            req.ManifestID,
		"transfer_id":            req.TransferID,
		"from_driver_id":         reassign.FromDriverID,
		"to_driver_id":           reassign.ToDriverID,
		"from_vehicle_id":        reassign.FromVehicleID,
		"to_vehicle_id":          reassign.ToVehicleID,
		"reassignment_depth":     reassign.Depth,
		"manifest_reassignments": manifest.ReassignmentDepth,
	})
}

// HandleManifestCancelTransfer serves POST /v1/factory/manifests/cancel-transfer.
func (s *Service) HandleManifestCancelTransfer(w http.ResponseWriter, r *http.Request) {
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
	var req manifestCancelTransferRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.ManifestID = strings.TrimSpace(req.ManifestID)
	req.TransferID = strings.TrimSpace(req.TransferID)
	if req.ManifestID == "" || req.TransferID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_and_transfer_id_required"})
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
		if manifest.State == manifestStateCompleted || manifest.State == manifestStateCancelled {
			return fmt.Errorf("manifest_not_mutable")
		}

		transfers := s.manifestTransfers[req.ManifestID]
		tIdx := s.findTransferIndexLocked(transfers, req.TransferID)
		if tIdx < 0 {
			return fmt.Errorf("transfer_not_found")
		}
		if transfers[tIdx].State == "CANCELLED" {
			return fmt.Errorf("transfer_already_cancelled")
		}

		transfers[tIdx].State = "CANCELLED"
		transfers[tIdx].ExceptionCount++
		transfers[tIdx].UpdatedAt = s.now().Format(time.RFC3339Nano)
		s.manifestTransfers[req.ManifestID] = transfers

		for i := range s.transfers {
			if s.transfers[i].TransferID == req.TransferID {
				s.transfers[i].State = "CANCELLED"
				s.transfers[i].ExceptionCount++
				s.transfers[i].UpdatedAt = transfers[tIdx].UpdatedAt
				break
			}
		}

		if manifest.TransferCnt > 0 {
			manifest.TransferCnt--
		}
		if manifest.TotalVolumeVU >= transfers[tIdx].TotalVU {
			manifest.TotalVolumeVU -= transfers[tIdx].TotalVU
		}
		manifest.UpdatedAt = s.now().Format(time.RFC3339Nano)
		manifest.LastExceptionAt = manifest.UpdatedAt
		manifest.EscalatedException = transfers[tIdx].ExceptionCount >= manifestExceptionEscalationThreshold
		s.manifests[mIdx] = manifest

		exception = ManifestException{
			ExceptionID:   s.nextIDLocked("mex"),
			ManifestID:    req.ManifestID,
			TransferID:    req.TransferID,
			Reason:        coalesceReason(strings.TrimSpace(req.Reason), "CANCEL_TRANSFER"),
			Metadata:      strings.TrimSpace(req.Metadata),
			AttemptCount:  transfers[tIdx].ExceptionCount,
			Escalated:     transfers[tIdx].ExceptionCount >= manifestExceptionEscalationThreshold,
			CreatedAt:     s.now().Format(time.RFC3339Nano),
			CorrelationID: req.ManifestID + ":" + req.TransferID,
		}
		s.manifestExceptions = append(s.manifestExceptions, exception)
		return nil
	}, func(txn outbox.TxnBuffer) error {
		if err := outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, events.ManifestEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventManifestOrderException},
			ManifestID:   req.ManifestID,
			TransferID:   req.TransferID,
			SupplierID:   s.resolveSupplierScope(r.Context()),
			FactoryID:    s.factoryNodeID,
			Reason:       exception.Reason,
			AttemptCount: exception.AttemptCount,
			Escalated:    exception.Escalated,
		}); err != nil {
			return err
		}
		if exception.Escalated {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, events.ManifestEvent{
				BaseEvent:    events.BaseEvent{Type: events.EventManifestDLQEscalation},
				ManifestID:   req.ManifestID,
				TransferID:   req.TransferID,
				SupplierID:   s.resolveSupplierScope(r.Context()),
				FactoryID:    s.factoryNodeID,
				Reason:       exception.Reason,
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
		case "manifest_not_mutable":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "manifest_not_mutable"})
			return
		case "transfer_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "transfer_not_found"})
			return
		case "transfer_already_cancelled":
			s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{"status": "already_cancelled", "manifest_id": req.ManifestID, "transfer_id": req.TransferID})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cancel_transfer_failed"})
			return
		}
	}

	s.invalidateFactoryKeys(r.Context(), factoryManifestKey(req.ManifestID), factoryManifestListKey(s.resolveSupplierScope(r.Context())), factoryTransferListKey(s.resolveSupplierScope(r.Context())), factoryExceptionListKey(s.resolveSupplierScope(r.Context())))
	s.broadcastFactoryEvent(r.Context(), events.EventManifestOrderException, map[string]any{
		"manifest_id":   req.ManifestID,
		"transfer_id":   req.TransferID,
		"reason":        exception.Reason,
		"attempt_count": exception.AttemptCount,
		"escalated":     exception.Escalated,
		"exception_id":  exception.ExceptionID,
	})
	if exception.Escalated {
		s.broadcastFactoryEvent(r.Context(), events.EventManifestDLQEscalation, map[string]any{
			"manifest_id":   req.ManifestID,
			"transfer_id":   req.TransferID,
			"reason":        exception.Reason,
			"attempt_count": exception.AttemptCount,
			"exception_id":  exception.ExceptionID,
		})
	}

	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"status":        "transfer_cancelled",
		"manifest_id":   req.ManifestID,
		"transfer_id":   req.TransferID,
		"exception_id":  exception.ExceptionID,
		"attempt_count": exception.AttemptCount,
		"escalated":     exception.Escalated,
	})
}

// HandleManifestCancel serves POST /v1/factory/manifests/cancel.
func (s *Service) HandleManifestCancel(w http.ResponseWriter, r *http.Request) {
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
	var req manifestCancelRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.ManifestID = strings.TrimSpace(req.ManifestID)
	if req.ManifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}

	var manifest ManifestRow
	var now string
	err = s.apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		idx := s.findManifestIndexLocked(req.ManifestID)
		if idx < 0 {
			return fmt.Errorf("manifest_not_found")
		}
		manifest = s.manifests[idx]
		if manifest.State == manifestStateCompleted {
			return fmt.Errorf("manifest_already_completed")
		}
		if manifest.State == manifestStateCancelled {
			return fmt.Errorf("manifest_already_cancelled")
		}

		now = s.now().Format(time.RFC3339Nano)
		fromState := manifest.State
		manifest.State = manifestStateCancelled
		manifest.CancelledAt = now
		manifest.UpdatedAt = now
		s.manifests[idx] = manifest

		transfers := s.manifestTransfers[req.ManifestID]
		for i := range transfers {
			transfers[i].State = "CANCELLED"
			transfers[i].UpdatedAt = now
		}
		s.manifestTransfers[req.ManifestID] = transfers

		for i := range s.transfers {
			if s.transfers[i].ManifestID == req.ManifestID {
				s.transfers[i].State = "CANCELLED"
				s.transfers[i].UpdatedAt = now
			}
		}

		s.appendTransitionLocked(req.ManifestID, "CANCEL", fromState, manifestStateCancelled, req.Reason)
		return nil
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, events.ManifestEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventManifestCancelled},
			ManifestID: req.ManifestID,
			SupplierID: s.resolveSupplierScope(r.Context()),
			FactoryID:  s.factoryNodeID,
			Reason:     strings.TrimSpace(req.Reason),
		})
	})
	if err != nil {
		switch err.Error() {
		case "manifest_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
			return
		case "manifest_already_completed":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "manifest_already_completed"})
			return
		case "manifest_already_cancelled":
			s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{"status": "already_cancelled", "manifest_id": req.ManifestID})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "manifest_cancel_failed"})
			return
		}
	}

	s.invalidateFactoryKeys(r.Context(), factoryManifestKey(req.ManifestID), factoryManifestListKey(s.resolveSupplierScope(r.Context())), factoryTransferListKey(s.resolveSupplierScope(r.Context())))
	s.broadcastFactoryEvent(r.Context(), events.EventManifestCancelled, map[string]any{
		"manifest_id": req.ManifestID,
		"reason":      strings.TrimSpace(req.Reason),
		"updated_at":  now,
	})

	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"status":      "manifest_cancelled",
		"manifest_id": req.ManifestID,
		"reason":      strings.TrimSpace(req.Reason),
		"updated_at":  now,
	})
}

// HandleManifestExceptions serves GET /v1/factory/manifest-exceptions.
func (s *Service) HandleManifestExceptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	escalatedOnly := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("escalated")), "true")

	backend := s.exceptionBackend()
	if backend != nil {
		rows, err := backend.List(r.Context(), s.resolveSupplierScope(r.Context()), strings.TrimSpace(s.resolveFactoryNode(r.Context())))
		if err != nil {
			s.log.WarnContext(r.Context(), "factory exception list failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "exception_list_failed"})
			return
		}
		if len(rows) > 0 || !s.portalSeedEnabled() {
			writeFactoryExceptionList(w, rows, escalatedOnly)
			return
		}
	}

	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]ManifestException(nil), s.manifestExceptions...)
	s.mu.Unlock()
	if !s.portalSeedEnabled() && s.spannerClient == nil && backend == nil {
		rows = nil
	}
	writeFactoryExceptionList(w, rows, escalatedOnly)
}

// HandleResolveManifestException serves POST /v1/factory/manifest-exceptions/{exceptionID}/resolve.
func (s *Service) HandleResolveManifestException(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	exceptionID := strings.TrimSpace(chi.URLParam(r, "exceptionID"))
	if exceptionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "exception_id_required"})
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
		Resolution string `json:"resolution"`
		Note       string `json:"note"`
	}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}
	req.Resolution = strings.TrimSpace(req.Resolution)
	if req.Resolution == "" {
		req.Resolution = "RESOLVED"
	}

	row, fromMemory, found, lookupErr := s.lookupManifestException(r.Context(), exceptionID)
	if lookupErr != nil {
		s.log.ErrorContext(r.Context(), "factory exception lookup failed", "err", lookupErr)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "exception_resolve_failed"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "exception_not_found"})
		return
	}

	var orderID string
	err = s.repo.RunTx(r.Context(), func(ctx context.Context, tx FactoryTx) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		if fromMemory {
			s.manifestExceptions = removeMemoryExceptionLocked(s.manifestExceptions, exceptionID)
		}
		for i := range s.transfers {
			if s.transfers[i].TransferID == row.TransferID {
				orderID = s.transfers[i].OrderID
				break
			}
		}
		if strings.TrimSpace(orderID) == "" {
			orderID = row.TransferID
		}
		return tx.ResolveException(ctx, row, orderID)
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, row.ManifestID, events.TopicMain, events.ManifestEvent{
			BaseEvent:      events.BaseEvent{Type: events.EventManifestExceptionResolved},
			ManifestID:     row.ManifestID,
			ManifestDomain: events.ManifestDomainFactory,
			SupplierID:     s.resolveSupplierScope(r.Context()),
			FactoryID:      s.resolveFactoryNode(r.Context()),
			TransferID:     row.TransferID,
			Reason:         req.Resolution,
		})
	})
	if err != nil {
		s.log.ErrorContext(r.Context(), "factory exception resolve failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "exception_resolve_failed"})
		return
	}
	s.invalidateFactoryKeys(r.Context(), factoryExceptionListKey(s.resolveSupplierScope(r.Context())), factoryManifestKey(row.ManifestID))
	s.broadcastFactoryEvent(r.Context(), events.EventManifestExceptionResolved, map[string]any{
		"exception_id": row.ExceptionID,
		"manifest_id":  row.ManifestID,
		"resolution":   req.Resolution,
	})
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"exception_id": row.ExceptionID,
		"manifest_id":  row.ManifestID,
		"resolution":   req.Resolution,
		"note":         strings.TrimSpace(req.Note),
		"status":       "RESOLVED",
	})
}

// HandleSupplyRequests serves GET /v1/factory/supply-requests.
func (s *Service) HandleSupplyRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rows := s.loadFactorySupplyRequests(r.Context())
	now := s.now()
	mapped := make([]map[string]any, len(rows))
	for i := range rows {
		mapped[i] = supplyRequestToMap(rows[i], s.factoryNodeID, s.resolveSupplierScope(r.Context()), now)
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": mapped})
}

// HandleSLABoard GET /v1/factory/sla-board — open requests sorted breach → at_risk → on_time (G7.1).
func (s *Service) HandleSLABoard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rows := s.loadFactorySupplyRequests(r.Context())
	now := s.now()
	sid := s.resolveSupplierScope(r.Context())
	var summary SLABoardSummary
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		m := supplyRequestToMap(row, s.factoryNodeID, sid, now)
		status, _ := m["sla_status"].(string)
		switch status {
		case SLAStatusBreached:
			summary.Breached++
			summary.TotalOpen++
			items = append(items, m)
		case SLAStatusAtRisk:
			summary.AtRisk++
			summary.TotalOpen++
			items = append(items, m)
		case SLAStatusOnTime:
			summary.OnTime++
			summary.TotalOpen++
			items = append(items, m)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		si, _ := items[i]["sla_status"].(string)
		sj, _ := items[j]["sla_status"].(string)
		if slaStatusRank(si) != slaStatusRank(sj) {
			return slaStatusRank(si) < slaStatusRank(sj)
		}
		di, _ := items[i]["sla_due_at"].(string)
		dj, _ := items[j]["sla_due_at"].(string)
		return di < dj
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"summary": summary,
		"items":   items,
		"as_of":   now.UTC().Format(time.RFC3339Nano),
		"config": map[string]any{
			"default_hours": FactorySLADefaultHours(),
			"at_risk_hours": FactorySLAAtRiskHours(),
		},
	})
}

func (s *Service) loadFactorySupplyRequests(ctx context.Context) []SupplyRequest {
	var rows []SupplyRequest
	if s.spannerClient != nil {
		spRows, err := s.listSupplyRequestsFromSpanner(ctx)
		if err != nil {
			s.log.Warn("factory supply list spanner failed; falling back to memory", "err", err)
		} else {
			rows = spRows
		}
	}
	if len(rows) == 0 {
		s.mu.Lock()
		s.ensureDemoDataLocked()
		rows = append([]SupplyRequest(nil), s.supplyRequests...)
		s.mu.Unlock()
	}
	return rows
}

func supplyRequestToMap(row SupplyRequest, factoryID, supplierID string, now time.Time) map[string]any {
	items := make([]map[string]any, 0, len(row.Items))
	for _, item := range row.Items {
		items = append(items, map[string]any{
			"item_id":            item.ItemID,
			"product_id":         item.ProductID,
			"requested_quantity": item.RequestedQuantity,
			"recommended_qty":    item.RecommendedQty,
			"unit_volume_vu":     item.UnitVolumeVU,
		})
	}
	m := map[string]any{
		"request_id":              row.RequestID,
		"warehouse_id":            row.WarehouseID,
		"factory_id":              factoryID,
		"supplier_id":             supplierID,
		"state":                   row.Status,
		"priority":                row.Priority,
		"notes":                   row.Notes,
		"region_id":               row.RegionID,
		"requested_delivery_date": row.RequestedDeliveryDate,
		"total_volume_vu":         row.TotalVolumeVU,
		"item_count":              len(row.Items),
		"items":                   items,
		"transfer_order_id":       row.LinkedTransferID,
		"created_by":              "",
		"created_at":              row.CreatedAt,
		"updated_at":              row.UpdatedAt,
	}
	EnrichSupplyRequestSLA(m, row.Status, row.CreatedAt, row.RequestedDeliveryDate, now)
	return m
}

type acceptSupplyRequest struct {
	Reason string `json:"reason"`
}

// HandleAcceptSupplyRequest serves POST /v1/factory/supply-requests/{requestID}/accept.
func (s *Service) HandleAcceptSupplyRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	requestID := chi.URLParam(r, "requestID")
	if requestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing_request_id"})
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
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var reqBody acceptSupplyRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &reqBody); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	}

	nextState := "ACKNOWLEDGED"
	nowTS := s.now().UTC().Format(time.RFC3339Nano)

	if s.spannerClient != nil {
		rec, err := s.getSupplyRequestFromSpanner(r.Context(), requestID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "request_not_found"})
			return
		}
		if strings.ToUpper(rec.State) != "SUBMITTED" && strings.ToUpper(rec.State) != "OPEN" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_state_transition"})
			return
		}
		if err := s.requireSupplyQCPass(r.Context(), requestID); err != nil {
			if errors.Is(err, errQCPassRequired) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "qc_pass_required"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "qc_read_failed"})
			return
		}
		err = s.transitionSupplyRequestSpanner(r.Context(), requestID, nextState, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateWarehouse, requestID, events.TopicMain, events.WarehouseEvent{
				BaseEvent:   events.BaseEvent{Type: events.EventSupplyRequestAccepted, Timestamp: nowTS},
				RequestID:   requestID,
				WarehouseID: rec.WarehouseID,
				SupplierID:  rec.SupplierID,
				FactoryID:   s.factoryNodeID,
				Status:      nextState,
			})
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		s.broadcastFactorySupplyEvent(r.Context(), map[string]any{
			"type": events.EventFactorySupplyRequestUpdate,
			"data": map[string]any{"request_id": requestID, "state": nextState, "warehouse_id": rec.WarehouseID},
		})
		idemCommitted = true
		s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{"state": nextState, "updated_at": nowTS})
		return
	}

	s.mu.Lock()
	s.ensureDemoDataLocked()
	var req *SupplyRequest
	for i := range s.supplyRequests {
		if s.supplyRequests[i].RequestID == requestID {
			req = &s.supplyRequests[i]
			break
		}
	}
	if req == nil {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "request_not_found"})
		return
	}
	if req.Status != "SUBMITTED" {
		s.mu.Unlock()
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_state_transition"})
		return
	}
	s.mu.Unlock()
	if err := s.requireSupplyQCPass(r.Context(), requestID); err != nil {
		if errors.Is(err, errQCPassRequired) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "qc_pass_required"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "qc_read_failed"})
		return
	}
	s.mu.Lock()
	req.Status = nextState
	req.UpdatedAt = nowTS
	s.mu.Unlock()

	err = s.repo.UpdateSupplyRequestState(r.Context(), requestID, nextState, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateWarehouse, requestID, events.TopicMain, events.WarehouseEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventSupplyRequestAccepted, Timestamp: nowTS},
			RequestID:   requestID,
			WarehouseID: req.WarehouseID,
			SupplierID:  s.resolveSupplierScope(r.Context()),
			FactoryID:   s.factoryNodeID,
			Status:      nextState,
		})
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error", "detail": err.Error()})
		return
	}
	idemCommitted = true
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{"state": nextState, "updated_at": nowTS})
}

func coalesceReason(reason, fallback string) string {
	if reason != "" {
		return reason
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
