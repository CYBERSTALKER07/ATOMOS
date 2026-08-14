// Package warehouse owns warehouse-role handlers, durable supply requests, and local operational scaffold state.
package warehouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/optimizerclient"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/plan"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"google.golang.org/api/iterator"
)

var errDispatchLockNotFound = errors.New("dispatch_lock_not_found")

// DemandPlanner exposes the warehouse-facing demand projection owned by the
// order aggregate.
type DemandPlanner interface {
	WarehouseDemandForecast(ctx context.Context, warehouseID string, start time.Time, days int) ([]order.WarehouseDemandDay, error)
}

// WarehouseAnalyticsCounts holds Spanner-backed aggregate counts.
type WarehouseAnalyticsCounts struct {
	TotalOrders     int64
	CompletedOrders int64
	CancelledOrders int64
	TotalRevenue    int64
}

// WarehouseAnalyticsQuery returns aggregate counts from Spanner.
type WarehouseAnalyticsQuery func(ctx context.Context, warehouseID string) (WarehouseAnalyticsCounts, error)

// OrderRow is the warehouse ops order read model.
type OrderRow struct {
	OrderID    string `json:"order_id"`
	RetailerID string `json:"retailer_id"`
	Status     string `json:"status"`
	TotalMinor int64  `json:"total_minor"`
	Currency   string `json:"currency"`
	UpdatedAt  string `json:"updated_at"`
}

// WarehouseOpsOrdersQuery returns orders scoped by warehouse from Spanner.
type WarehouseOpsOrdersQuery func(ctx context.Context, warehouseID string, limit int) ([]OrderRow, error)

// WarehouseOpsDriversQuery returns drivers by home-node warehouse.
type WarehouseOpsDriversQuery func(ctx context.Context, warehouseID string) ([]PortalDriver, error)

// WarehouseOpsVehiclesQuery returns vehicles by home-node warehouse.
type WarehouseOpsVehiclesQuery func(ctx context.Context, warehouseID string) ([]PortalVehicle, error)

// Service stores additive in-memory data for warehouse operational surfaces.
type Service struct {
	repo                 Repository
	planner              DemandPlanner
	analyticsQuery       WarehouseAnalyticsQuery
	opsOrders            WarehouseOpsOrdersQuery
	opsDrivers           WarehouseOpsDriversQuery
	opsVehicles            WarehouseOpsVehiclesQuery
	gatewayBreakdownQuery  func(ctx context.Context, warehouseID, period string) ([]map[string]any, bool)
	platformFeeQuery       func(ctx context.Context, warehouseID, period string) (int64, bool)
	cache                  *cache.Cache
	idem                 idempotency.Store
	spannerClient        *spanner.Client
	manifestStore        *manifest.Store
	routeGeometryBuilder *routing.GeometryBuilder
	locations            telemetry.LastLocationReader
	supplierHub          *ws.Hub
	warehouseHub         *ws.Hub
	driverHub            *ws.Hub
	retailerHub          *ws.Hub
	log                  *slog.Logger

	seedSupplierID string
	currency       string
	now            func() time.Time

	mu     sync.RWMutex
	orders []OrderRow

	jwtSecret        string
	jwtIssuer        string
	optimizerClient  *optimizerclient.Client
	planCounters     *plan.SourceCounters
	fallbackDepotLat float64
	fallbackDepotLng float64

	portalSeeded          bool
	drivers               []PortalDriver
	vehicles              []PortalVehicle
	staff                 []portalStaff
	products              []portalProduct
	manifests             []portalManifest
	retailers             []portalRetailer
	returns               []portalReturnItem
	insights              []replenishmentInsight
	internalTransfers     map[string]memoryTransferRow
	broadcastTemplatesMem map[string][]customBroadcastTemplateRow
	firebaseVerifier      auth.FirebaseVerifier
	orderStock            OrderStockReader
}

// ServiceConfig is the constructor input.
type ServiceConfig struct {
	Repo                 Repository
	Planner              DemandPlanner
	AnalyticsQuery       WarehouseAnalyticsQuery
	OpsOrders            WarehouseOpsOrdersQuery
	OpsDrivers           WarehouseOpsDriversQuery
	OpsVehicles          WarehouseOpsVehiclesQuery
	Cache                *cache.Cache
	Idem                 idempotency.Store
	Spanner              *spanner.Client
	ManifestStore        *manifest.Store
	RouteGeometryBuilder *routing.GeometryBuilder
	Locations            telemetry.LastLocationReader
	SupplierHub          *ws.Hub
	WarehouseHub         *ws.Hub
	DriverHub            *ws.Hub
	RetailerHub          *ws.Hub
	Log                  *slog.Logger

	// SeedSupplierID is bootstrap/fixture fallback only (Gate 5 Week 11).
	SeedSupplierID string
	// SupplierID is deprecated; use SeedSupplierID.
	SupplierID string
	Currency   string
	Now        func() time.Time

	JWTSecret        string
	JWTIssuer        string
	OptimizerClient  *optimizerclient.Client
	PlanCounters     *plan.SourceCounters
	FallbackDepotLat float64
	FallbackDepotLng float64
	FirebaseVerifier auth.FirebaseVerifier
}

// InventoryRow represents one stock row.
type InventoryRow struct {
	SKU              string `json:"sku"`
	ProductName      string `json:"product_name"`
	Quantity         int64  `json:"quantity"`
	QuantityOnHand   int64  `json:"quantity_on_hand"`
	ReorderThreshold int64  `json:"reorder_threshold"`
	OutOfStockPolicy string `json:"out_of_stock_policy"`
	EffectivePolicy  string `json:"effective_policy"`
	UpdatedAt        string `json:"updated_at"`
}

// SupplyRequest represents one replenishment request row.
type SupplyRequest struct {
	RequestID                string              `json:"request_id"`
	SupplierID               string              `json:"supplier_id,omitempty"`
	WarehouseID              string              `json:"warehouse_id,omitempty"`
	FactoryID                string              `json:"factory_id,omitempty"`
	TransferMode             string              `json:"transfer_mode,omitempty"`
	LinkedTransferID         string              `json:"linked_transfer_id,omitempty"`
	Status                   string              `json:"status"`
	State                    string              `json:"state,omitempty"`
	RequestedBy              string              `json:"requested_by,omitempty"`
	CoverageStartDate        string              `json:"coverage_start_date,omitempty"`
	CoverageDays             int                 `json:"coverage_days,omitempty"`
	ProjectedUnits           int64               `json:"projected_units,omitempty"`
	CommittedUnits           int64               `json:"committed_units,omitempty"`
	PendingConfirmationUnits int64               `json:"pending_confirmation_units,omitempty"`
	Priority                 string              `json:"priority,omitempty"`
	Notes                    string              `json:"notes,omitempty"`
	RegionID                 string              `json:"region_id,omitempty"`
	RequestedDeliveryDate    string              `json:"requested_delivery_date,omitempty"`
	TotalVolumeVU            float64             `json:"total_volume_vu,omitempty"`
	Items                    []SupplyRequestItem `json:"items,omitempty"`
	CreatedAt                string              `json:"created_at"`
	UpdatedAt                string              `json:"updated_at"`
}

// DispatchLock represents one active dispatch lock.
type DispatchLock struct {
	LockID     string `json:"lock_id"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Reason     string `json:"reason"`
	CreatedAt  string `json:"created_at"`
}

// NewService constructs the warehouse service.
func NewService(c ServiceConfig) *Service {
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	if c.Log == nil {
		c.Log = slog.Default()
	}
	if c.Currency == "" {
		c.Currency = "UZS"
	}
	seedID := strings.TrimSpace(c.SeedSupplierID)
	if seedID == "" {
		seedID = strings.TrimSpace(c.SupplierID)
	}
	return &Service{
		repo:                 c.Repo,
		planner:              c.Planner,
		analyticsQuery:       c.AnalyticsQuery,
		opsOrders:            c.OpsOrders,
		opsDrivers:           c.OpsDrivers,
		opsVehicles:          c.OpsVehicles,
		cache:                c.Cache,
		idem:                 c.Idem,
		spannerClient:        c.Spanner,
		manifestStore:        c.ManifestStore,
		routeGeometryBuilder: c.RouteGeometryBuilder,
		locations:            c.Locations,
		supplierHub:          c.SupplierHub,
		warehouseHub:         c.WarehouseHub,
		driverHub:            c.DriverHub,
		retailerHub:          c.RetailerHub,
		log:                  c.Log,
		seedSupplierID:       seedID,
		currency:             c.Currency,
		now:                  c.Now,
		jwtSecret:            c.JWTSecret,
		jwtIssuer:            c.JWTIssuer,
		optimizerClient:      c.OptimizerClient,
		planCounters:         c.PlanCounters,
		fallbackDepotLat:     c.FallbackDepotLat,
		fallbackDepotLng:     c.FallbackDepotLng,
		firebaseVerifier:     c.FirebaseVerifier,
	}
}

// HandleDashboard serves GET /v1/warehouse/ops/dashboard.
func (s *Service) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	s.handleOpsDashboard(w, r)
}

// HandleInventory serves GET/PATCH /v1/warehouse/ops/inventory.
func (s *Service) HandleInventory(w http.ResponseWriter, r *http.Request) {
	s.handleOpsInventory(w, r)
}

// HandleOrders serves GET /v1/warehouse/ops/orders and sub-paths.
func (s *Service) HandleOrders(w http.ResponseWriter, r *http.Request) {
	s.handleOpsOrders(w, r)
}

// HandleDispatchPreview serves GET/POST /v1/warehouse/ops/dispatch/preview.
func (s *Service) HandleDispatchPreview(w http.ResponseWriter, r *http.Request) {
	s.handleOpsDispatchPreview(w, r)
}

// HandleDispatchExecute serves POST /v1/warehouse/ops/dispatch/execute.
func (s *Service) HandleDispatchExecute(w http.ResponseWriter, r *http.Request) {
	s.handleOpsDispatchExecute(w, r)
}

// HandleDispatchSettings serves GET/PATCH /v1/warehouse/ops/dispatch/settings.
func (s *Service) HandleDispatchSettings(w http.ResponseWriter, r *http.Request) {
	s.handleOpsDispatchSettings(w, r)
}

// HandleDemandForecast serves GET /v1/warehouse/demand/forecast.
func (s *Service) HandleDemandForecast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	warehouseID := warehouseIDFromRequest(r)
	if warehouseID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id required"})
		return
	}
	start, days, err := parsePlanningWindow(r, s.now)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	series := make([]map[string]any, 0, days)
	if s.planner != nil {
		forecast, err := s.planner.WarehouseDemandForecast(r.Context(), warehouseID, start, days)
		if err != nil {
			s.log.Warn("warehouse demand forecast failed", "warehouse_id", warehouseID, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_demand_forecast_failed"})
			return
		}
		for _, day := range forecast {
			series = append(series, map[string]any{
				"date":                       day.Date,
				"projected_units":            day.ProjectedUnits,
				"projected_revenue":          day.ProjectedRevenue,
				"committed_units":            day.CommittedUnits,
				"pending_confirmation_units": day.PendingConfirmationUnits,
				"currency":                   day.Currency,
			})
		}
	} else {
		for i := 0; i < days; i++ {
			d := start.AddDate(0, 0, i)
			series = append(series, map[string]any{
				"date":                       d.Format("2006-01-02"),
				"projected_units":            0,
				"projected_revenue":          0,
				"committed_units":            0,
				"pending_confirmation_units": 0,
				"currency":                   s.currency,
			})
		}
	}
	products, source := s.productDemandForecast(r.Context(), warehouseID, days)
	writeJSON(w, http.StatusOK, map[string]any{
		"warehouse_id":  warehouseID,
		"forecast_days": days,
		"generated_at":  s.now().UTC().Format(time.RFC3339Nano),
		"series":        series,
		"products":      products,
		"source":        source,
	})
}

// HandleSupplyRequests serves GET/POST /v1/warehouse/supply-requests.
func (s *Service) HandleSupplyRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListSupplyRequests(w, r)
	case http.MethodPost:
		s.handleCreateSupplyRequest(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleListSupplyRequests(w http.ResponseWriter, r *http.Request) {
	warehouseID := warehouseIDFromRequest(r)
	if warehouseID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id required"})
		return
	}

	rows, err := s.repo.ListSupplyRequests(r.Context(), warehouseID, 100)
	if err != nil {
		s.log.Warn("warehouse supply request list failed", "warehouse_id", warehouseID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supply_requests_failed"})
		return
	}

	mapped := make([]map[string]any, len(rows))
	for i, row := range rows {
		mapped[i] = supplyRequestIOSPayload(row)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"requests":        mapped,
		"supply_requests": mapped,
	})
}

func (s *Service) handleCreateSupplyRequest(w http.ResponseWriter, r *http.Request) {
	warehouseID := warehouseIDFromRequest(r)
	if warehouseID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id required"})
		return
	}
	if s.handleCreateSupplyRequestFromBody(w, r, warehouseID) {
		return
	}

	start, days, err := parsePlanningWindow(r, s.now)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	claims, _ := auth.FromContext(r.Context())
	requestedBy := strings.TrimSpace(r.URL.Query().Get("requested_by"))
	if requestedBy == "" {
		requestedBy = strings.TrimSpace(claims.Subject)
	}

	projectedUnits, committedUnits, pendingConfirmationUnits, err := s.snapshotSupplyRequestForecast(r.Context(), warehouseID, start, days)
	if err != nil {
		s.log.Warn("warehouse supply request forecast snapshot failed", "warehouse_id", warehouseID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supply_request_forecast_failed"})
		return
	}

	topology, err := s.resolveWarehouseSupplyContext(r.Context(), warehouseID)
	if err != nil {
		s.log.Warn("warehouse supply topology resolve failed", "warehouse_id", warehouseID, "err", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_topology_unconfigured"})
		return
	}

	nowTS := s.now().UTC().Format(time.RFC3339Nano)
	req := SupplyRequest{
		RequestID:                uuid.NewString(),
		SupplierID:               s.resolveSupplierScope(r.Context()),
		WarehouseID:              warehouseID,
		FactoryID:                topology.FactoryID,
		TransferMode:             topology.TransferMode,
		Status:                   "SUBMITTED",
		State:                    "SUBMITTED",
		RequestedBy:              requestedBy,
		CoverageStartDate:        start.Format("2006-01-02"),
		CoverageDays:             days,
		ProjectedUnits:           projectedUnits,
		CommittedUnits:           committedUnits,
		PendingConfirmationUnits: pendingConfirmationUnits,
		TotalVolumeVU:            float64(projectedUnits),
		CreatedAt:                nowTS,
		UpdatedAt:                nowTS,
	}

	eventPayload := events.WarehouseEvent{
		BaseEvent:         events.BaseEvent{Type: events.EventWarehouseSupplyRequestOpened},
		RequestID:         req.RequestID,
		SupplierID:        s.resolveSupplierScope(r.Context()),
		WarehouseID:       req.WarehouseID,
		FactoryID:         req.FactoryID,
		TransferMode:      req.TransferMode,
		Status:            req.Status,
		Projected:         req.ProjectedUnits,
		Committed:         req.CommittedUnits,
		Pending:           req.PendingConfirmationUnits,
		RequestedBy:       req.RequestedBy,
		CoverageDays:      int64(req.CoverageDays),
		CoverageStartDate: req.CoverageStartDate,
	}

	if err := s.repo.CreateSupplyRequest(r.Context(), req, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateWarehouse, req.RequestID, events.TopicMain, eventPayload)
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_supply_request_failed"})
		return
	}

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), warehouseSupplyRequestsKey(s.resolveSupplierScope(r.Context()), warehouseID))
	}

	s.broadcastSupplyRequestUpdate(r.Context(), warehouseID, req)
	s.log.Info(
		"warehouse supply request submitted",
		"supplier_id", s.resolveSupplierScope(r.Context()),
		"warehouse_id", warehouseID,
		"request_id", req.RequestID,
		"projected_units", req.ProjectedUnits,
		"committed_units", req.CommittedUnits,
		"pending_confirmation_units", req.PendingConfirmationUnits,
	)
	writeJSON(w, http.StatusCreated, req)
}

// HandleSupplyRequestAccepted is called by the async event consumer when a factory accepts the supply request.
func (s *Service) HandleSupplyRequestAccepted(ctx context.Context, payloadBytes []byte) error {
	var payload events.WarehouseEvent
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		s.log.Warn("failed to parse supply request accepted payload", "err", err)
		return err
	}

	requestID := payload.RequestID
	if requestID == "" {
		s.log.Warn("missing request_id in supply request accepted payload")
		return nil
	}

	warehouseID := payload.WarehouseID
	if warehouseID == "" {
		warehouseID = "wh-1"
	}

	status := "ACKNOWLEDGED"
	if strings.TrimSpace(payload.Status) != "" {
		status = strings.ToUpper(strings.TrimSpace(payload.Status))
	}

	err := s.repo.UpdateSupplyRequestStatus(ctx, requestID, status, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateWarehouse, requestID, events.TopicMain, events.WarehouseEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventSupplyRequestUpdate, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)},
			RequestID:   requestID,
			WarehouseID: warehouseID,
			SupplierID:  s.resolveSupplierScope(ctx),
			FactoryID:   payload.FactoryID,
			Status:      status,
		})
	})
	if err != nil {
		s.log.Error("failed to update supply request status", "request_id", requestID, "err", err)
		return err
	}

	if s.cache != nil {
		s.cache.Invalidate(ctx, warehouseSupplyRequestsKey(s.resolveSupplierScope(ctx), warehouseID))
	}
	s.broadcastSupplyRequestUpdate(ctx, warehouseID, SupplyRequest{
		RequestID:   requestID,
		WarehouseID: warehouseID,
		SupplierID:  s.resolveSupplierScope(ctx),
		FactoryID:   payload.FactoryID,
		Status:      status,
		State:       status,
	})
	return nil
}

// HandleDispatchLocks serves GET /v1/warehouse/dispatch-locks.
func (s *Service) HandleDispatchLocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	warehouseID := warehouseIDFromRequest(r)
	if warehouseID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id_required"})
		return
	}
	lockMap, err := s.repo.GetLocks(r.Context(), warehouseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_fetch_locks"})
		return
	}
	locks := make([]map[string]any, 0, len(lockMap))
	for _, v := range lockMap {
		locks = append(locks, map[string]any{
			"lock_id":      v.LockID,
			"supplier_id":  s.resolveSupplierScope(r.Context()),
			"warehouse_id": warehouseID,
			"factory_id":   "",
			"lock_type":    "MANUAL_DISPATCH",
			"locked_at":    v.CreatedAt,
			"locked_by":    "warehouse_ops",
			"entity_type":  v.EntityType,
			"entity_id":    v.EntityID,
			"reason":       v.Reason,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"locks": locks})
}

// HandleDispatchLock serves POST/DELETE /v1/warehouse/dispatch-lock.
func (s *Service) HandleDispatchLock(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		body, ok := readMutationBody(w, r, 64*1024)
		if !ok {
			return
		}
		key, handled := s.guardMutationReplay(w, r, body)
		if handled {
			return
		}

		var payload struct {
			EntityType string `json:"entity_type"`
			EntityID   string `json:"entity_id"`
			Reason     string `json:"reason"`
			LockType   string `json:"lock_type"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		if strings.TrimSpace(payload.EntityType) == "" {
			payload.EntityType = "WAREHOUSE"
		}
		if strings.TrimSpace(payload.EntityID) == "" {
			payload.EntityID = strings.TrimSpace(warehouseIDFromRequest(r))
			if payload.EntityID == "" {
				payload.EntityID = "warehouse-scope"
			}
		}
		nowTS := s.now().Format(time.RFC3339Nano)
		claims, _ := auth.FromContext(r.Context())
		warehouseID, whErr := s.effectiveWarehouseID(r.Context(), r)
		if whErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "resolve_warehouse_scope_failed"})
			return
		}
		lockType := strings.TrimSpace(payload.LockType)
		if lockType == "" {
			lockType = "MANUAL_DISPATCH"
		}
		lock := DispatchLock{
			LockID:     "lock_" + strings.ReplaceAll(nowTS, ":", ""),
			EntityType: strings.TrimSpace(payload.EntityType),
			EntityID:   strings.TrimSpace(payload.EntityID),
			Reason:     encodeDispatchLockReason(lockType, payload.Reason),
			CreatedAt:  nowTS,
		}

		eventPayload := events.WarehouseEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventWarehouseDispatchLockChanged, Timestamp: nowTS},
			LockID:      lock.LockID,
			WarehouseID: warehouseID,
			SupplierID:  s.resolveSupplierScope(r.Context()),
			Status:      "ACTIVE",
			Action:      "ACQUIRED",
			RequestID:   lock.EntityID,
		}

		if err := s.repo.UpsertLock(r.Context(), warehouseID, lock, func(txn outbox.TxnBuffer) error {
			return emitDispatchLockAcquireOutbox(r.Context(), txn, lock.LockID, eventPayload, lock, lockType, warehouseID, claims.Subject, nowTS)
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_dispatch_lock_failed"})
			return
		}

		if s.cache != nil {
			s.cache.Invalidate(r.Context(), warehouseDispatchLocksKey(s.resolveSupplierScope(r.Context())))
		}

		s.broadcastWarehouseEvent(r.Context(), warehouseID, eventPayload)
		s.log.Info("warehouse dispatch lock acquired", "supplier_id", s.resolveSupplierScope(r.Context()), "warehouse_id", warehouseID, "lock_id", lock.LockID)
		resp := map[string]any{
			"lock_id":   lock.LockID,
			"lock_type": lockType,
			"status":    "ACTIVE",
		}
		respBytes, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(respBytes)
		s.storeMutationReplay(r.Context(), key, body, http.StatusCreated, respBytes)
	case http.MethodDelete:
		body, ok := readMutationBody(w, r, 64*1024)
		if !ok {
			return
		}
		key, handled := s.guardMutationReplay(w, r, body)
		if handled {
			return
		}

		lockID := strings.TrimSpace(r.URL.Query().Get("lock_id"))
		if lockID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lock_id required"})
			return
		}

		claims, _ := auth.FromContext(r.Context())
		warehouseID, whErr := s.effectiveWarehouseID(r.Context(), r)
		if whErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "resolve_warehouse_scope_failed"})
			return
		}

		var released DispatchLock
		// fetch to check exists
		lockMap, err := s.repo.GetLocks(r.Context(), warehouseID)
		releasedLock, exists := lockMap[lockID]
		if !exists || err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "lock_not_found"})
			return
		}
		released = releasedLock
		releasedLockType := decodeDispatchLockType(released.Reason)
		if err := s.repo.DeleteLock(r.Context(), warehouseID, lockID, func(txn outbox.TxnBuffer) error {
			eventPayload := events.WarehouseEvent{
				BaseEvent:   events.BaseEvent{Type: events.EventWarehouseDispatchLockChanged, Timestamp: s.now().Format(time.RFC3339Nano)},
				LockID:      lockID,
				WarehouseID: warehouseID,
				SupplierID:  s.resolveSupplierScope(r.Context()),
				Status:      "RELEASED",
				Action:      "RELEASED",
				RequestedBy: claims.Subject,
				RequestID:   released.EntityID,
			}
			return emitDispatchLockReleaseOutbox(r.Context(), txn, lockID, eventPayload, released, releasedLockType, warehouseID, claims.Subject)
		}); err != nil {
			if errors.Is(err, errDispatchLockNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "dispatch_lock_not_found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "release_dispatch_lock_failed"})
			return
		}

		if s.cache != nil {
			s.cache.Invalidate(r.Context(), warehouseDispatchLocksKey(s.resolveSupplierScope(r.Context())))
		}

		s.broadcastWarehouseEvent(r.Context(), warehouseID, map[string]any{
			"type":         events.EventWarehouseDispatchLockChanged,
			"supplier_id":  s.resolveSupplierScope(r.Context()),
			"warehouse_id": warehouseID,
			"lock_id":      lockID,
			"entity_type":  released.EntityType,
			"entity_id":    released.EntityID,
			"reason":       released.Reason,
			"action":       "RELEASED",
			"timestamp":    s.now().Format(time.RFC3339Nano),
		})
		s.log.Info("warehouse dispatch lock released", "supplier_id", s.resolveSupplierScope(r.Context()), "warehouse_id", warehouseID, "lock_id", lockID)
		resp := map[string]any{"status": "released", "lock_id": lockID}
		respBytes, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respBytes)
		s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// BroadcastFleetEvent pushes a real-time warehouse hub payload (dispatch board + notifications).
func (s *Service) BroadcastFleetEvent(ctx context.Context, warehouseID string, payload any) {
	s.broadcastWarehouseEvent(ctx, warehouseID, payload)
}

func (s *Service) broadcastWarehouseEvent(ctx context.Context, warehouseID string, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if s.supplierHub != nil && s.resolveSupplierScope(ctx) != "" {
		s.supplierHub.Broadcast(ctx, "supplier:"+s.resolveSupplierScope(ctx), raw)
	}
	if s.warehouseHub != nil && warehouseID != "" {
		s.warehouseHub.Broadcast(ctx, "warehouse:"+warehouseID, raw)
	}
}

func (s *Service) broadcastSupplyRequestUpdate(ctx context.Context, warehouseID string, req SupplyRequest) {
	state := strings.TrimSpace(req.State)
	if state == "" {
		state = strings.TrimSpace(req.Status)
	}
	s.broadcastWarehouseEvent(ctx, warehouseID, map[string]any{
		"type":      events.EventSupplyRequestUpdate,
		"timestamp": s.now().UTC().Format(time.RFC3339Nano),
		"data": map[string]any{
			"request_id":         req.RequestID,
			"warehouse_id":       req.WarehouseID,
			"factory_id":         req.FactoryID,
			"supplier_id":        req.SupplierID,
			"state":              state,
			"transfer_mode":      req.TransferMode,
			"linked_transfer_id": req.LinkedTransferID,
		},
	})
}

func (s *Service) snapshotSupplyRequestForecast(ctx context.Context, warehouseID string, start time.Time, days int) (int64, int64, int64, error) {
	if s.planner == nil {
		return 0, 0, 0, nil
	}

	forecast, err := s.planner.WarehouseDemandForecast(ctx, warehouseID, start, days)
	if err != nil {
		return 0, 0, 0, err
	}

	var projectedUnits int64
	var committedUnits int64
	var pendingConfirmationUnits int64
	for _, day := range forecast {
		projectedUnits += day.ProjectedUnits
		committedUnits += day.CommittedUnits
		pendingConfirmationUnits += day.PendingConfirmationUnits
	}

	return projectedUnits, committedUnits, pendingConfirmationUnits, nil
}

// portalSeedEnabled gates in-memory demo seed data. Disabled when Spanner is wired
// unless WAREHOUSE_PORTAL_SEED=true is set explicitly for local scaffold runs.
func (s *Service) portalSeedEnabled() bool {
	if s.spannerClient != nil {
		switch strings.ToLower(strings.TrimSpace(os.Getenv("WAREHOUSE_PORTAL_SEED"))) {
		case "1", "true", "yes":
			return true
		default:
			return false
		}
	}
	v := strings.ToLower(strings.TrimSpace(os.Getenv("WAREHOUSE_PORTAL_SEED")))
	return v == "1" || v == "true" || v == "yes"
}

func warehouseIDFromRequest(r *http.Request) string {
	if id := auth.EffectiveWarehouseOpsID(r.Context()); id != "" {
		return id
	}
	if id := auth.EffectiveWarehouseID(r.Context()); id != "" {
		return id
	}
	claims, ok := auth.FromContext(r.Context())
	if ok && (claims.Role == auth.RoleFactory || claims.Role == auth.RoleFactoryAdmin) {
		return ""
	}
	warehouseID := strings.TrimSpace(r.URL.Query().Get("warehouse_id"))
	if warehouseID == "" && ok && claims.HomeNodeType == auth.HomeNodeWarehouse {
		warehouseID = strings.TrimSpace(claims.HomeNodeID)
	}
	return warehouseID
}

func (s *Service) analyticsSupplierID(ctx context.Context) string {
	return s.resolveSupplierScope(ctx)
}

func (s *Service) resolveAnalyticsSupplierID(r *http.Request) string {
	if r == nil {
		return s.resolveSupplierScope(context.Background())
	}
	return s.resolveSupplierScope(r.Context())
}

// effectiveWarehouseID resolves warehouse scope for transfer mutations. Supplier
// ADMIN (portal cookie) may omit warehouse_id when the tenant has a default warehouse.
func (s *Service) effectiveWarehouseID(ctx context.Context, r *http.Request) (string, error) {
	if id := warehouseIDFromRequest(r); id != "" {
		return id, nil
	}
	claims, ok := auth.FromContext(ctx)
	if !ok || claims.Role != auth.RoleAdmin {
		return "", nil
	}
	supplierID := strings.TrimSpace(claims.SupplierID)
	if supplierID == "" {
		if sid, ok := auth.ResolveSupplierID(ctx); ok {
			supplierID = sid
		}
	}
	if supplierID == "" {
		return "", nil
	}
	return s.defaultWarehouseIDForSupplier(ctx, supplierID)
}

func (s *Service) defaultWarehouseIDForSupplier(ctx context.Context, supplierID string) (string, error) {
	if s.memoryTransfersEnabled() {
		return "ssmr-warehouse-1", nil
	}
	if s.spannerClient == nil {
		return "", errors.New("warehouse resolver unavailable")
	}
	stmt := spanner.Statement{
		SQL:    `SELECT WarehouseId FROM Warehouses WHERE SupplierId = @supplier_id LIMIT 1`,
		Params: map[string]any{"supplier_id": supplierID},
	}
	iter := s.spannerClient.Single().
		WithTimestampBound(spanner.MaxStaleness(15*time.Second)).
		Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return "", errors.New("no warehouse for supplier")
	}
	if err != nil {
		return "", fmt.Errorf("list warehouses for supplier %s: %w", supplierID, err)
	}
	var warehouseID string
	if err := row.Columns(&warehouseID); err != nil {
		return "", fmt.Errorf("decode warehouse id: %w", err)
	}
	return strings.TrimSpace(warehouseID), nil
}

func (s *Service) resolveWarehouseOps(ctx context.Context, r *http.Request) (*auth.WarehouseOps, error) {
	if ops := auth.GetWarehouseOps(r.Context()); ops != nil && strings.TrimSpace(ops.WarehouseID) != "" {
		return ops, nil
	}
	warehouseID, err := s.effectiveWarehouseID(ctx, r)
	if err != nil {
		return nil, err
	}
	if warehouseID == "" {
		return nil, errTransferForbidden
	}
	claims, _ := auth.FromContext(ctx)
	supplierID := strings.TrimSpace(claims.SupplierID)
	if supplierID == "" {
		if sid, ok := auth.ResolveSupplierID(ctx); ok {
			supplierID = sid
		}
	}
	return &auth.WarehouseOps{
		WarehouseID: warehouseID,
		SupplierID:  supplierID,
		Subject:     claims.Subject,
	}, nil
}

func parsePlanningWindow(r *http.Request, now func() time.Time) (time.Time, int, error) {
	days := 7
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return time.Time{}, 0, errors.New("days must be a positive integer")
		}
		days = parsed
	}
	start := now().UTC().Truncate(24 * time.Hour)
	if raw := strings.TrimSpace(r.URL.Query().Get("start_date")); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return time.Time{}, 0, errors.New("start_date must use YYYY-MM-DD")
		}
		start = parsed.UTC()
	}
	return start, days, nil
}

func warehouseSupplyRequestsKey(supplierID string, warehouseID string) string {
	return "warehouse:supply-requests:" + supplierID + ":" + warehouseID
}

func warehouseDispatchLocksKey(supplierID string) string {
	return "warehouse:dispatch-locks:" + supplierID
}

func encodeDispatchLockReason(lockType, reason string) string {
	lt := strings.TrimSpace(lockType)
	if lt == "" {
		lt = "MANUAL_DISPATCH"
	}
	r := strings.TrimSpace(reason)
	if r == "" {
		return "lock_type:" + lt
	}
	return "lock_type:" + lt + "|" + r
}

func decodeDispatchLockType(reason string) string {
	reason = strings.TrimSpace(reason)
	if strings.HasPrefix(reason, "lock_type:") {
		rest := strings.TrimPrefix(reason, "lock_type:")
		if idx := strings.Index(rest, "|"); idx >= 0 {
			return strings.TrimSpace(rest[:idx])
		}
		if rest != "" {
			return rest
		}
	}
	return "MANUAL_DISPATCH"
}

const defaultFreezeLockTTLSeconds int64 = 300

func emitDispatchLockAcquireOutbox(ctx context.Context, txn outbox.TxnBuffer, lockID string, warehouseEvent events.WarehouseEvent, lock DispatchLock, lockType, warehouseID, lockedBy, timestamp string) error {
	if err := outbox.EmitJSON(ctx, txn, events.AggregateWarehouse, lockID, events.TopicMain, warehouseEvent); err != nil {
		return err
	}
	if lockType != "MANUAL_DISPATCH" {
		return nil
	}
	freeze := events.DispatchLockEvent{
		BaseEvent:   events.BaseEvent{Type: events.EventFreezeLockAcquired, Timestamp: timestamp},
		LockID:      lockID,
		SupplierID:  warehouseEvent.SupplierID,
		WarehouseID: warehouseID,
		LockType:    lockType,
		LockedBy:    lockedBy,
		EntityType:  strings.TrimSpace(lock.EntityType),
		EntityID:    strings.TrimSpace(lock.EntityID),
		TTLSeconds:  defaultFreezeLockTTLSeconds,
	}
	if err := outbox.EmitJSON(ctx, txn, "DispatchLock", lockID, events.TopicFreezeLocks, freeze); err != nil {
		return err
	}
	return outbox.EmitJSON(ctx, txn, "DispatchLock", lockID, events.TopicMain, freeze)
}

func emitDispatchLockReleaseOutbox(ctx context.Context, txn outbox.TxnBuffer, lockID string, warehouseEvent events.WarehouseEvent, lock DispatchLock, lockType, warehouseID, lockedBy string) error {
	if err := outbox.EmitJSON(ctx, txn, events.AggregateWarehouse, lockID, events.TopicMain, warehouseEvent); err != nil {
		return err
	}
	if lockType != "MANUAL_DISPATCH" {
		return nil
	}
	freeze := events.DispatchLockEvent{
		BaseEvent:   events.BaseEvent{Type: events.EventFreezeLockReleased, Timestamp: warehouseEvent.BaseEvent.Timestamp},
		LockID:      lockID,
		SupplierID:  warehouseEvent.SupplierID,
		WarehouseID: warehouseID,
		LockType:    lockType,
		LockedBy:    lockedBy,
		EntityType:  strings.TrimSpace(lock.EntityType),
		EntityID:    strings.TrimSpace(lock.EntityID),
	}
	if err := outbox.EmitJSON(ctx, txn, "DispatchLock", lockID, events.TopicFreezeLocks, freeze); err != nil {
		return err
	}
	return outbox.EmitJSON(ctx, txn, "DispatchLock", lockID, events.TopicMain, freeze)
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
