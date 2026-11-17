package factory

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

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/optimizerclient"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"

	"cloud.google.com/go/spanner"
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
		if s.guardIdempotencyStrict(w, r, body) {
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
