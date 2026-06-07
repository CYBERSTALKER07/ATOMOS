package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/optimizerclient"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/plan"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"google.golang.org/api/iterator"
)

// PortalOpsConfig wires optional Spanner + supplier hub for admin-parity endpoints.
type PortalOpsConfig struct {
	Spanner          *spanner.Client
	SupplierHub      *ws.Hub
	OptimizerClient  *optimizerclient.Client
	PlanCounters     *plan.SourceCounters
	FallbackDepotLat float64
	FallbackDepotLng float64
}

// SetPortalOps attaches cross-cutting deps used by supplier admin-parity handlers.
func (s *Service) SetPortalOps(cfg PortalOpsConfig) {
	s.portalSpanner = cfg.Spanner
	s.portalSupplierHub = cfg.SupplierHub
	s.optimizerClient = cfg.OptimizerClient
	s.planCounters = cfg.PlanCounters
	s.fallbackDepotLat = cfg.FallbackDepotLat
	s.fallbackDepotLng = cfg.FallbackDepotLng
}

// EmpathyAdoption is the supplier-scoped empathy metrics snapshot.
type EmpathyAdoption struct {
	TotalPredictions    int64 `json:"total_predictions"`
	PredictionsDormant  int64 `json:"predictions_dormant"`
	PredictionsWaiting  int64 `json:"predictions_waiting"`
	PredictionsFired    int64 `json:"predictions_fired"`
	PredictionsRejected int64 `json:"predictions_rejected"`
}

// HandleEmpathyAdoption serves GET /v1/supplier/empathy/adoption.
func (s *Service) HandleEmpathyAdoption(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "empathy_unavailable"})
		return
	}

	sid := s.scopedSupplierID(r)
	ctx := r.Context()
	adoption := EmpathyAdoption{}

	count := func(sql string, params map[string]any) int64 {
		iter := s.portalSpanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
		defer iter.Stop()
		row, err := iter.Next()
		if err != nil {
			return 0
		}
		var c int64
		if err := row.Columns(&c); err != nil {
			return 0
		}
		return c
	}

	adoption.TotalPredictions = count(
		`SELECT COUNT(*) FROM AIPredictions WHERE SupplierId = @sid`,
		map[string]any{"sid": sid},
	)

	statusSQL := `SELECT Status, COUNT(*) AS c FROM AIPredictions WHERE SupplierId = @sid GROUP BY Status`
	iter := s.portalSpanner.Single().Query(ctx, spanner.Statement{
		SQL: statusSQL, Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		var status string
		var c int64
		if err := row.Columns(&status, &c); err != nil {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(status)) {
		case "DORMANT":
			adoption.PredictionsDormant = c
		case "WAITING", "PENDING":
			adoption.PredictionsWaiting += c
		case "FIRED", "CONFIRMED":
			adoption.PredictionsFired += c
		case "REJECTED":
			adoption.PredictionsRejected = c
		}
	}

	writeJSON(w, http.StatusOK, adoption)
}

// HandleBroadcast serves POST /v1/supplier/broadcast (supplier-scoped WS fan-out).
func (s *Service) HandleBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()
	req.Title = strings.TrimSpace(req.Title)
	req.Body = strings.TrimSpace(req.Body)
	if req.Title == "" || req.Body == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title_and_body_required"})
		return
	}

	sid := s.scopedSupplierID(r)
	if s.portalSupplierHub == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "broadcast_unavailable"})
		return
	}

	payload, _ := json.Marshal(map[string]any{
		"type":        "SUPPLIER_BROADCAST",
		"title":       req.Title,
		"body":        req.Body,
		"target_role": strings.TrimSpace(req.Role),
		"supplier_id": sid,
		"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
	})
	s.portalSupplierHub.Broadcast(r.Context(), "supplier:"+sid, payload)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "broadcast_sent",
		"supplier_id": sid,
	})
}

// HandleReplenishmentTrigger serves POST /v1/supplier/replenishment/trigger.
func (s *Service) HandleReplenishmentTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "replenishment_unavailable"})
		return
	}

	sid := s.scopedSupplierID(r)
	ctx := r.Context()
	topology, err := s.repo.GetTopology(ctx, sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "topology_load_failed"})
		return
	}
	warehouseID := ""
	for _, wh := range topology.Warehouses {
		if wh.IsActive {
			warehouseID = wh.WarehouseID
			break
		}
	}
	if warehouseID == "" && len(topology.Warehouses) > 0 {
		warehouseID = topology.Warehouses[0].WarehouseID
	}
	if warehouseID == "" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "no_warehouse_configured"})
		return
	}

	claims, _ := auth.FromContext(r.Context())
	requestedBy := strings.TrimSpace(claims.Subject)
	now := s.now().UTC()
	requestID := uuid.NewString()
	start := now.Format("2006-01-02")

	eventPayload := events.WarehouseEvent{
		BaseEvent:         events.BaseEvent{Type: events.EventWarehouseSupplyRequestOpened},
		SupplierID:        sid,
		WarehouseID:       warehouseID,
		RequestID:         requestID,
		Status:            "OPEN",
		State:             "OPEN",
		RequestedBy:       requestedBy,
		CoverageStartDate: start,
		CoverageDays:      int64(7),
	}

	_, err = s.portalSpanner.ReadWriteTransaction(ctx, func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &supplierSpannerTxnBuf{}
		if err := outbox.EmitJSON(txnCtx, buf, events.AggregateWarehouse, warehouseID, events.TopicMain, eventPayload); err != nil {
			return err
		}
		mutations := []*spanner.Mutation{spanner.InsertOrUpdateMap("WarehouseSupplyRequests", map[string]any{
			"RequestId":                requestID,
			"SupplierId":               sid,
			"WarehouseId":              warehouseID,
			"State":                    "OPEN",
			"RequestedBy":              requestedBy,
			"CoverageStartDate":        start,
			"CoverageDays":             int64(7),
			"ProjectedUnits":           int64(0),
			"CommittedUnits":           int64(0),
			"PendingConfirmationUnits": int64(0),
			"CreatedAt":                now,
			"UpdatedAt":                now,
		})}
		for _, e := range buf.events {
			mutations = append(mutations, portalOutboxMutation(e))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		s.log.ErrorContext(ctx, "replenishment trigger failed", "err", err, "warehouse_id", warehouseID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_supply_request_failed"})
		return
	}

	if s.cache != nil {
		s.cache.Invalidate(ctx, fmt.Sprintf("warehouse:supply:%s:%s", sid, warehouseID))
	}
	if s.portalSupplierHub != nil {
		raw, _ := json.Marshal(eventPayload)
		s.portalSupplierHub.Broadcast(ctx, "supplier:"+sid, raw)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "triggered",
		"request_id":   requestID,
		"warehouse_id": warehouseID,
	})
}

// HandleSupplierFleetOrders serves GET /v1/supplier/fleet/orders.
func (s *Service) HandleSupplierFleetOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	orders, err := s.listSupplierOrders(r.Context(), sid, "", "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fleet_orders_failed"})
		return
	}

	rows := make([]map[string]any, 0, len(orders))
	for _, o := range orders {
		if o.Status == "COMPLETED" || o.Status == "CANCELLED" {
			continue
		}
		row := map[string]any{
			"id":          o.OrderID,
			"order_id":    o.OrderID,
			"retailer_id": o.RetailerID,
			"driver_id":   o.DriverID,
			"status":      o.Status,
			"state":       o.Status,
			"route_id":    o.RouteID,
			"total_minor": o.TotalMinor,
			"currency":    o.Currency,
			"updated_at":  o.UpdatedAt,
		}
		if o.DriverLocation != nil {
			row["driver_location"] = o.DriverLocation
		}
		rows = append(rows, row)
	}
	writeJSON(w, http.StatusOK, rows)
}

type supplierSpannerTxnBuf struct {
	events []outbox.Event
}

func (b *supplierSpannerTxnBuf) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func portalOutboxMutation(e outbox.Event) *spanner.Mutation {
	createdAt := e.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	row := map[string]any{
		"EventId":       e.EventID,
		"AggregateType": e.AggregateType,
		"AggregateId":   e.AggregateID,
		"TopicName":     e.TopicName,
		"Payload":       e.Payload,
		"CreatedAt":     createdAt,
		"PublishedAt":   nil,
	}
	if e.PublishedAt != nil {
		row["PublishedAt"] = e.PublishedAt.UTC()
	}
	return spanner.InsertOrUpdateMap("OutboxEvents", row)
}
