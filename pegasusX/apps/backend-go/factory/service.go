package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
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

// Repository is the mutation seam for factory write paths.
type Repository interface {
	Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error
}

// inMemoryRepository is the scaffold repository implementation.
type inMemoryRepository struct{}

// inMemoryTxnBuffer is the scaffold outbox buffer.
type inMemoryTxnBuffer struct {
	events []outbox.Event
}

func (b *inMemoryTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func (r *inMemoryRepository) Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error {
	if mutate != nil {
		if err := mutate(); err != nil {
			return err
		}
	}
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
	}
	_ = ctx
	return nil
}

// NewInMemoryRepository returns the default in-memory repository seam.
func NewInMemoryRepository() Repository {
	return &inMemoryRepository{}
}

// Service stores additive in-memory data for factory operational surfaces.
type Service struct {
	repo        Repository
	cache       *cache.Cache
	supplierHub *ws.Hub
	factoryHub  *ws.Hub
	log         *slog.Logger

	supplierID string
	currency   string
	now        func() time.Time

	mu                    sync.RWMutex
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

	SupplierID string
	Currency   string
	Now        func() time.Time
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
	ManifestID         string `json:"manifest_id"`
	State              string `json:"state"`
	TransferCnt        int    `json:"transfer_count"`
	TotalVolumeVU      int64  `json:"total_volume_vu"`
	MaxVolumeVU        int64  `json:"max_volume_vu"`
	DriverID           string `json:"driver_id,omitempty"`
	VehicleID          string `json:"vehicle_id,omitempty"`
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
	StaffID string `json:"staff_id"`
	Name    string `json:"name"`
	Role    string `json:"role"`
}

// SupplyRequest represents one supply request row.
type SupplyRequest struct {
	RequestID   string `json:"request_id"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	WarehouseID string `json:"warehouse_id,omitempty"`
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
	if c.Currency == "" {
		c.Currency = "UZS"
	}
	return &Service{
		repo:                  c.Repo,
		cache:                 c.Cache,
		supplierHub:           c.SupplierHub,
		factoryHub:            c.FactoryHub,
		log:                   c.Log,
		supplierID:            c.SupplierID,
		currency:              c.Currency,
		now:                   c.Now,
		manifestTransfers:     make(map[string][]TransferRow),
		manifestTransitions:   make(map[string][]ManifestTransition),
		manifestReassignments: make(map[string][]ManifestReassignment),
	}
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

func (s *Service) ensureDemoDataLocked() {
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
		now := s.now().Format(time.RFC3339Nano)
		s.supplyRequests = []SupplyRequest{
			{RequestID: "srq_factory_1", Status: "OPEN", WarehouseID: "wh_demo_1", CreatedAt: now, UpdatedAt: now},
		}
	}
	if len(s.transfers) == 0 {
		now := s.now().Format(time.RFC3339Nano)
		s.transfers = []TransferRow{
			{TransferID: "tr_factory_1", OrderID: "ord_factory_1", State: "CREATED", TotalVU: 42, CreatedAt: now, UpdatedAt: now},
			{TransferID: "tr_factory_2", OrderID: "ord_factory_2", State: "CREATED", TotalVU: 37, CreatedAt: now, UpdatedAt: now},
		}
	}
	if len(s.manifests) == 0 {
		now := s.now().Format(time.RFC3339Nano)
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
}

func (s *Service) findManifestIndexLocked(manifestID string) int {
	for i := range s.manifests {
		if s.manifests[i].ManifestID == manifestID {
			return i
		}
	}
	return -1
}

func (s *Service) findTransferIndexLocked(transfers []TransferRow, transferID string) int {
	for i := range transfers {
		if transfers[i].TransferID == transferID {
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
		s.supplierHub.Broadcast(ctx, "supplier:"+s.supplierID, payload)
	}
	if s.factoryHub != nil {
		s.factoryHub.Broadcast(ctx, "factory:"+s.supplierID, payload)
	}
}

func (s *Service) invalidateFactoryKeys(ctx context.Context, keys ...string) {
	if s.cache == nil || len(keys) == 0 {
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

// HandleAnalyticsOverview serves GET /v1/factory/analytics/overview.
func (s *Service) HandleAnalyticsOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.Lock()
	s.ensureDemoDataLocked()
	transfers := len(s.transfers)
	manifests := len(s.manifests)
	escalatedExceptions := 0
	for i := range s.manifestExceptions {
		if s.manifestExceptions[i].Escalated {
			escalatedExceptions++
		}
	}
	s.mu.Unlock()
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
	s.mu.Lock()
	s.ensureDemoDataLocked()
	resp := map[string]any{
		"supplier_id":     s.supplierID,
		"transfers_open":  len(s.transfers),
		"manifests_open":  len(s.manifests),
		"fleet_drivers":   len(s.fleetDrivers),
		"fleet_vehicles":  len(s.fleetVehicles),
		"staff_count":     len(s.staff),
		"exception_queue": len(s.manifestExceptions),
		"updated_at":      s.now().Format(time.RFC3339Nano),
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, resp)
}

// HandleProfile serves GET /v1/factory/profile.
func (s *Service) HandleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"factory_id":  strings.TrimSpace(r.URL.Query().Get("factory_id")),
		"supplier_id": s.supplierID,
		"currency":    s.currency,
		"updated_at":  s.now().Format(time.RFC3339Nano),
	})
}

// HandleTransfers serves GET /v1/factory/transfers and POST /v1/factory/transfers/create.
func (s *Service) HandleTransfers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.Lock()
		s.ensureDemoDataLocked()
		rows := append([]TransferRow(nil), s.transfers...)
		s.mu.Unlock()
		sort.Slice(rows, func(i, j int) bool { return rows[i].UpdatedAt > rows[j].UpdatedAt })
		writeJSON(w, http.StatusOK, map[string]any{"transfers": rows})
	case http.MethodPost:
		var req transferCreateRequest
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}
		if req.TotalVU <= 0 {
			req.TotalVU = 25
		}
		var row TransferRow
		now := ""
		err := s.repo.Apply(r.Context(), func() error {
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
		}, nil)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "transfer_create_failed"})
			return
		}
		s.invalidateFactoryKeys(r.Context(), factoryTransferListKey(s.supplierID))
		writeJSON(w, http.StatusCreated, row)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleManifests serves GET /v1/factory/manifests.
func (s *Service) HandleManifests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	state := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("state")))

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

	s.mu.Lock()
	s.ensureDemoDataLocked()
	idx := s.findManifestIndexLocked(manifestID)
	if idx < 0 {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
		return
	}
	manifest := s.manifests[idx]
	transfers := append([]TransferRow(nil), s.manifestTransfers[manifestID]...)
	transitions := append([]ManifestTransition(nil), s.manifestTransitions[manifestID]...)
	reassignments := append([]ManifestReassignment(nil), s.manifestReassignments[manifestID]...)
	exceptions := make([]ManifestException, 0)
	for i := range s.manifestExceptions {
		if s.manifestExceptions[i].ManifestID == manifestID {
			exceptions = append(exceptions, s.manifestExceptions[i])
		}
	}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"manifest":      manifest,
		"transfers":     transfers,
		"transitions":   transitions,
		"reassignments": reassignments,
		"exceptions":    exceptions,
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

	var req transitionRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
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
	err := s.repo.Apply(r.Context(), func() error {
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
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, manifest.ManifestID, events.TopicMain, map[string]any{
			"type":        eventType,
			"manifest_id": manifest.ManifestID,
			"supplier_id": s.supplierID,
			"state":       manifest.State,
			"reason":      strings.TrimSpace(req.Reason),
			"action":      action,
			"route_id":    routeIDForManifest(manifest),
			"driver_id":   manifest.DriverID,
			"vehicle_id":  manifest.VehicleID,
			"order_count": manifest.TransferCnt,
			"timestamp":   s.now().Format(time.RFC3339Nano),
		})
	})
	if err != nil {
		if strings.HasPrefix(err.Error(), "invalid_state:") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_state", "message": err.Error()})
			return
		}
		if err.Error() == "manifest_not_found" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "transition_failed"})
		return
	}

	s.invalidateFactoryKeys(r.Context(), factoryManifestKey(manifest.ManifestID), factoryManifestListKey(s.supplierID), factoryTransferListKey(s.supplierID))
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

	writeJSON(w, http.StatusOK, map[string]any{
		"status":      strings.ToLower(action) + "_applied",
		"manifest_id": manifest.ManifestID,
		"state":       manifest.State,
		"updated_at":  manifest.UpdatedAt,
	})
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
	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]FleetVehicle(nil), s.fleetVehicles...)
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"vehicles": rows})
}

// HandleStaff serves GET /v1/factory/staff.
func (s *Service) HandleStaff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]StaffRow(nil), s.staff...)
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"staff": rows})
}

// HandleDispatch serves POST /v1/factory/dispatch.
func (s *Service) HandleDispatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	var req dispatchRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	var manifest ManifestRow
	selectedCount := 0
	now := ""
	err := s.repo.Apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		now = s.now().Format(time.RFC3339Nano)

		available := make([]TransferRow, 0)
		for i := range s.transfers {
			if s.transfers[i].State == "CREATED" {
				available = append(available, s.transfers[i])
			}
		}
		if len(available) == 0 {
			transfer := TransferRow{
				TransferID: s.nextIDLocked("tr"),
				OrderID:    s.nextIDLocked("ord"),
				State:      "CREATED",
				TotalVU:    20,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			s.transfers = append(s.transfers, transfer)
			available = append(available, transfer)
		}

		if req.MaxVolumeVU <= 0 {
			req.MaxVolumeVU = 180
		}
		if strings.TrimSpace(req.DriverID) == "" {
			req.DriverID = s.fleetDrivers[0].DriverID
		}
		if strings.TrimSpace(req.VehicleID) == "" {
			req.VehicleID = s.fleetVehicles[0].VehicleID
		}

		selected := make([]TransferRow, 0)
		if len(req.TransferIDs) > 0 {
			for _, id := range req.TransferIDs {
				id = strings.TrimSpace(id)
				if id == "" {
					continue
				}
				for i := range available {
					if available[i].TransferID == id {
						selected = append(selected, available[i])
						break
					}
				}
			}
		}
		if len(selected) == 0 {
			limit := len(available)
			if limit > 2 {
				limit = 2
			}
			selected = append(selected, available[:limit]...)
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
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, manifest.ManifestID, events.TopicMain, map[string]any{
			"type":            events.EventManifestDraftCreated,
			"manifest_id":     manifest.ManifestID,
			"supplier_id":     s.supplierID,
			"route_id":        routeIDForManifest(manifest),
			"transfer_count":  manifest.TransferCnt,
			"total_volume_vu": manifest.TotalVolumeVU,
			"driver_id":       manifest.DriverID,
			"vehicle_id":      manifest.VehicleID,
			"timestamp":       now,
		})
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dispatch_failed"})
		return
	}

	s.invalidateFactoryKeys(r.Context(), factoryManifestKey(manifest.ManifestID), factoryManifestListKey(s.supplierID), factoryTransferListKey(s.supplierID))
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

	writeJSON(w, http.StatusOK, map[string]any{
		"status":                 "dispatch_planned",
		"created_manifest_count": 1,
		"manifest_id":            manifest.ManifestID,
		"transfer_count":         selectedCount,
		"updated_at":             now,
	})
}

// HandleManifestRebalance serves POST /v1/factory/manifests/rebalance.
func (s *Service) HandleManifestRebalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	var req manifestRebalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.ManifestID = strings.TrimSpace(req.ManifestID)
	req.TransferID = strings.TrimSpace(req.TransferID)
	req.ToDriverID = strings.TrimSpace(req.ToDriverID)
	req.ToVehicle = strings.TrimSpace(req.ToVehicle)
	if req.ManifestID == "" || req.TransferID == "" || (req.ToDriverID == "" && req.ToVehicle == "") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_transfer_id_and_target_required"})
		return
	}

	var manifest ManifestRow
	var reassign ManifestReassignment
	err := s.repo.Apply(r.Context(), func() error {
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

		fromDriver := transfers[tIdx].DriverID
		fromVehicle := transfers[tIdx].VehicleID
		if req.ToDriverID != "" {
			transfers[tIdx].DriverID = req.ToDriverID
		}
		if req.ToVehicle != "" {
			transfers[tIdx].VehicleID = req.ToVehicle
		}
		transfers[tIdx].State = "REASSIGNED"
		transfers[tIdx].ReassignDepth++
		transfers[tIdx].UpdatedAt = s.now().Format(time.RFC3339Nano)
		s.manifestTransfers[req.ManifestID] = transfers

		for i := range s.transfers {
			if s.transfers[i].TransferID == req.TransferID {
				s.transfers[i].DriverID = transfers[tIdx].DriverID
				s.transfers[i].VehicleID = transfers[tIdx].VehicleID
				s.transfers[i].ReassignDepth = transfers[tIdx].ReassignDepth
				s.transfers[i].State = transfers[tIdx].State
				s.transfers[i].UpdatedAt = transfers[tIdx].UpdatedAt
				break
			}
		}

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
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, map[string]any{
			"type":            events.EventManifestRebalanced,
			"manifest_id":     req.ManifestID,
			"transfer_id":     req.TransferID,
			"supplier_id":     s.supplierID,
			"from_driver_id":  reassign.FromDriverID,
			"to_driver_id":    reassign.ToDriverID,
			"from_vehicle_id": reassign.FromVehicleID,
			"to_vehicle_id":   reassign.ToVehicleID,
			"depth":           reassign.Depth,
			"reason":          reassign.Reason,
			"timestamp":       reassign.ReassignedAt,
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
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "manifest_rebalance_failed"})
			return
		}
	}

	s.invalidateFactoryKeys(r.Context(), factoryManifestKey(req.ManifestID), factoryManifestListKey(s.supplierID), factoryTransferListKey(s.supplierID))
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

	writeJSON(w, http.StatusOK, map[string]any{
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
	var req manifestCancelTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	err := s.repo.Apply(r.Context(), func() error {
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
		if err := outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, map[string]any{
			"type":          events.EventManifestOrderException,
			"manifest_id":   req.ManifestID,
			"transfer_id":   req.TransferID,
			"supplier_id":   s.supplierID,
			"reason":        exception.Reason,
			"attempt_count": exception.AttemptCount,
			"escalated":     exception.Escalated,
			"timestamp":     exception.CreatedAt,
		}); err != nil {
			return err
		}
		if exception.Escalated {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, map[string]any{
				"type":          events.EventManifestDLQEscalation,
				"manifest_id":   req.ManifestID,
				"transfer_id":   req.TransferID,
				"supplier_id":   s.supplierID,
				"reason":        exception.Reason,
				"attempt_count": exception.AttemptCount,
				"timestamp":     exception.CreatedAt,
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
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cancel_transfer_failed"})
			return
		}
	}

	s.invalidateFactoryKeys(r.Context(), factoryManifestKey(req.ManifestID), factoryManifestListKey(s.supplierID), factoryTransferListKey(s.supplierID), factoryExceptionListKey(s.supplierID))
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

	writeJSON(w, http.StatusOK, map[string]any{
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
	var req manifestCancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	err := s.repo.Apply(r.Context(), func() error {
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
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, map[string]any{
			"type":        events.EventManifestCancelled,
			"manifest_id": req.ManifestID,
			"supplier_id": s.supplierID,
			"reason":      strings.TrimSpace(req.Reason),
			"timestamp":   now,
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
			writeJSON(w, http.StatusOK, map[string]any{"status": "already_cancelled", "manifest_id": req.ManifestID})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "manifest_cancel_failed"})
			return
		}
	}

	s.invalidateFactoryKeys(r.Context(), factoryManifestKey(req.ManifestID), factoryManifestListKey(s.supplierID), factoryTransferListKey(s.supplierID))
	s.broadcastFactoryEvent(r.Context(), events.EventManifestCancelled, map[string]any{
		"manifest_id": req.ManifestID,
		"reason":      strings.TrimSpace(req.Reason),
		"updated_at":  now,
	})

	writeJSON(w, http.StatusOK, map[string]any{
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

	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]ManifestException(nil), s.manifestExceptions...)
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

// HandleSupplyRequests serves GET /v1/factory/supply-requests.
func (s *Service) HandleSupplyRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]SupplyRequest(nil), s.supplyRequests...)
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"requests": rows})
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
