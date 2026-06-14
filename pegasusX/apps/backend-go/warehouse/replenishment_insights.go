package warehouse

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

var errInsightNotFound = errors.New("insight_not_found")
var errInsightForbidden = errors.New("insight_forbidden")
var errInsightAlreadyProcessed = errors.New("insight_already_processed")
var errInsightNoFactory = errors.New("insight_no_target_factory")

type replenishmentInsight struct {
	ID                string  `json:"id"`
	WarehouseID       string  `json:"warehouse_id"`
	WarehouseName     string  `json:"warehouse_name"`
	ProductID         string  `json:"product_id"`
	ProductName       string  `json:"product_name"`
	Urgency           string  `json:"urgency"`
	CurrentStock      int64   `json:"current_stock"`
	AvgDailyVelocity  float64 `json:"avg_daily_velocity"`
	DaysUntilStockout int     `json:"days_until_stockout"`
	ReorderQuantity   int64   `json:"reorder_quantity"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"created_at"`
}

func insightWireStatus(dbStatus string) string {
	switch strings.ToUpper(strings.TrimSpace(dbStatus)) {
	case "PENDING":
		return "OPEN"
	default:
		return strings.ToUpper(strings.TrimSpace(dbStatus))
	}
}

func insightDBStatus(queryStatus string) string {
	switch strings.ToUpper(strings.TrimSpace(queryStatus)) {
	case "", "OPEN":
		return "PENDING"
	default:
		return strings.ToUpper(strings.TrimSpace(queryStatus))
	}
}

func insightWireUrgency(level string) string {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "WARNING":
		return "HIGH"
	case "CRITICAL":
		return "CRITICAL"
	default:
		if level == "" {
			return "STABLE"
		}
		return strings.ToUpper(strings.TrimSpace(level))
	}
}

// HandleReplenishmentInsights serves GET /v1/warehouse/replenishment/insights.
func (s *Service) HandleReplenishmentInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	statusFilter := insightDBStatus(r.URL.Query().Get("status"))

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	rows, err := s.listReplenishmentInsights(ctx, whID, statusFilter)
	if err != nil {
		s.log.ErrorContext(ctx, "replenishment insights list failed", "warehouse_id", whID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"insights": rows, "data": rows})
}

// HandleReplenishmentInsightAction serves POST /v1/warehouse/replenishment/insights/{id}/{action}.
func (s *Service) HandleReplenishmentInsightAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	insightID := strings.TrimSpace(chi.URLParam(r, "id"))
	action := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "action")))
	if insightID == "" || action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path must be /insights/{id}/{approve|dismiss}"})
		return
	}
	if action != "approve" && action != "dismiss" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action must be approve or dismiss"})
		return
	}

	whID := warehouseIDFromRequest(r)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	resp, err := s.applyReplenishmentInsightAction(ctx, whID, insightID, action)
	if errors.Is(err, errInsightNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "insight_not_found"})
		return
	}
	if errors.Is(err, errInsightForbidden) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if errors.Is(err, errInsightAlreadyProcessed) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "insight_already_processed"})
		return
	}
	if errors.Is(err, errInsightNoFactory) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insight_no_target_factory"})
		return
	}
	if err != nil {
		s.log.ErrorContext(ctx, "replenishment insight action failed", "insight_id", insightID, "action", action, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Service) listReplenishmentInsights(ctx context.Context, warehouseID, status string) ([]replenishmentInsight, error) {
	if s != nil && s.spannerClient != nil {
		rows, err := s.listReplenishmentInsightsSpanner(ctx, warehouseID, status)
		if err == nil {
			return rows, nil
		}
	}
	return s.listReplenishmentInsightsMemory(warehouseID), nil
}

func (s *Service) listReplenishmentInsightsSpanner(ctx context.Context, warehouseID, status string) ([]replenishmentInsight, error) {
	params := map[string]any{"status": status}
	sql := `SELECT ri.InsightId, ri.WarehouseId, COALESCE(w.Name, ri.WarehouseId),
	               ri.ProductId, COALESCE(p.Name, ri.ProductId),
	               ri.CurrentStock, ri.DailyBurnRate, ri.TimeToEmptyDays,
	               ri.SuggestedQuantity, ri.UrgencyLevel, ri.Status, ri.CreatedAt
	        FROM ReplenishmentInsights ri
	        LEFT JOIN Warehouses w ON ri.WarehouseId = w.WarehouseId
	        LEFT JOIN Products p ON ri.ProductId = p.ProductId
	        WHERE ri.Status = @status`
	if strings.TrimSpace(warehouseID) != "" {
		sql += ` AND ri.WarehouseId = @whId`
		params["whId"] = warehouseID
	}
	if strings.TrimSpace(s.supplierID) != "" {
		sql += ` AND ri.SupplierId = @sid`
		params["sid"] = s.supplierID
	}
	sql += ` ORDER BY ri.CreatedAt DESC`

	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	rows := make([]replenishmentInsight, 0)
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query replenishment insights: %w", err)
		}
		var item replenishmentInsight
		var createdAt time.Time
		var daysFloat float64
		if err := row.Columns(
			&item.ID, &item.WarehouseID, &item.WarehouseName,
			&item.ProductID, &item.ProductName,
			&item.CurrentStock, &item.AvgDailyVelocity, &daysFloat,
			&item.ReorderQuantity, &item.Urgency, &item.Status, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan replenishment insight: %w", err)
		}
		item.Urgency = insightWireUrgency(item.Urgency)
		item.Status = insightWireStatus(item.Status)
		item.DaysUntilStockout = int(math.Round(daysFloat))
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		rows = append(rows, item)
	}
	return rows, nil
}

func (s *Service) listReplenishmentInsightsMemory(warehouseID string) []replenishmentInsight {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureReplenishmentInsightsLocked(warehouseID)
	rows := make([]replenishmentInsight, 0, len(s.insights))
	for _, row := range s.insights {
		if warehouseID == "" || row.WarehouseID == warehouseID {
			rows = append(rows, row)
		}
	}
	return rows
}

func (s *Service) applyReplenishmentInsightAction(ctx context.Context, warehouseID, insightID, action string) (map[string]any, error) {
	if s == nil || s.spannerClient == nil {
		return s.applyReplenishmentInsightActionMemory(warehouseID, insightID, action)
	}
	return s.applyReplenishmentInsightActionSpanner(ctx, warehouseID, insightID, action)
}

func (s *Service) applyReplenishmentInsightActionMemory(warehouseID, insightID, action string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureReplenishmentInsightsLocked(warehouseID)
	idx := -1
	for i := range s.insights {
		if s.insights[i].ID == insightID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, errInsightNotFound
	}
	if warehouseID != "" && s.insights[idx].WarehouseID != warehouseID {
		return nil, errInsightForbidden
	}
	nextStatus := "DISMISSED"
	transferID := ""
	if action == "approve" {
		nextStatus = "APPROVED"
		transferID = uuid.NewString()
	}
	s.insights[idx].Status = nextStatus
	resp := map[string]any{
		"insight_id": insightID,
		"status":     nextStatus,
	}
	if transferID != "" {
		resp["transfer_id"] = transferID
	}
	return resp, nil
}

type insightActionRow struct {
	WarehouseID       string
	ProductID         string
	SupplierID        string
	SuggestedQuantity int64
	Status            string
	TargetFactoryID   string
}

func (s *Service) applyReplenishmentInsightActionSpanner(ctx context.Context, warehouseID, insightID, action string) (map[string]any, error) {
	row, err := s.loadInsightActionRow(ctx, insightID)
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return nil, errInsightNotFound
		}
		return nil, err
	}
	if warehouseID != "" && row.WarehouseID != warehouseID {
		return nil, errInsightForbidden
	}
	if strings.ToUpper(row.Status) != "PENDING" {
		return nil, errInsightAlreadyProcessed
	}

	if action == "dismiss" {
		_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.UpdateMap("ReplenishmentInsights", map[string]any{
					"InsightId": insightID,
					"Status":    "DISMISSED",
				}),
			})
		})
		if err != nil {
			return nil, fmt.Errorf("dismiss insight %s: %w", insightID, err)
		}
		return map[string]any{
			"insight_id": insightID,
			"status":     "DISMISSED",
		}, nil
	}

	targetFactory := strings.TrimSpace(row.TargetFactoryID)
	if targetFactory == "" {
		factoryID, _, resolveErr := s.resolveWarehouseFactory(ctx, row.WarehouseID)
		if resolveErr != nil {
			return nil, fmt.Errorf("resolve warehouse factory: %w", resolveErr)
		}
		targetFactory = factoryID
	}
	if targetFactory == "" {
		return nil, errInsightNoFactory
	}

	transferID := uuid.NewString()
	totalVU := float64(row.SuggestedQuantity)
	if totalVU <= 0 {
		totalVU = 1
	}

	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.UpdateMap("ReplenishmentInsights", map[string]any{
				"InsightId": insightID,
				"Status":    "APPROVED",
			}),
			spanner.InsertOrUpdateMap("FactoryInternalTransfers", map[string]any{
				"TransferId":    transferID,
				"FactoryId":     targetFactory,
				"SupplierId":    row.SupplierID,
				"WarehouseId":   row.WarehouseID,
				"State":         "APPROVED",
				"TotalVolumeVU": totalVU,
				"CreatedAt":     spanner.CommitTimestamp,
				"UpdatedAt":     spanner.CommitTimestamp,
			}),
		}
		payload := events.WarehouseEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventWarehouseTransferCreated, Timestamp: s.now().UTC().Format(time.RFC3339Nano)},
			TransferID:  transferID,
			WarehouseID: row.WarehouseID,
			SupplierID:  row.SupplierID,
			Units:       row.SuggestedQuantity,
		}
		buf := &spannerTxnBuffer{}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, row.WarehouseID, events.TopicMain, payload); emitErr != nil {
			return emitErr
		}
		mutations = append(mutations, outboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return nil, fmt.Errorf("approve insight %s: %w", insightID, err)
	}

	return map[string]any{
		"insight_id":  insightID,
		"status":      "APPROVED",
		"transfer_id": transferID,
	}, nil
}

func (s *Service) loadInsightActionRow(ctx context.Context, insightID string) (insightActionRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, ProductId, SupplierId, SuggestedQuantity, Status,
		             COALESCE(TargetFactoryId, '')
		      FROM ReplenishmentInsights WHERE InsightId = @iid`,
		Params: map[string]any{"iid": insightID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return insightActionRow{}, err
	}
	var out insightActionRow
	if err := row.Columns(
		&out.WarehouseID, &out.ProductID, &out.SupplierID,
		&out.SuggestedQuantity, &out.Status, &out.TargetFactoryID,
	); err != nil {
		return insightActionRow{}, err
	}
	return out, nil
}

func (s *Service) replenishmentInsightsByProduct(ctx context.Context, warehouseID string) map[string]replenishmentInsight {
	rows, err := s.listReplenishmentInsights(ctx, warehouseID, "PENDING")
	if err != nil {
		return nil
	}
	out := make(map[string]replenishmentInsight, len(rows))
	for _, row := range rows {
		if insightWireStatus(row.Status) != "OPEN" {
			continue
		}
		out[row.ProductID] = row
	}
	return out
}

func (s *Service) ensureReplenishmentInsightsLocked(warehouseID string) {
	if len(s.insights) > 0 {
		return
	}
	now := s.now().Format("2006-01-02T15:04:05.999999999Z07:00")
	s.insights = []replenishmentInsight{{
		ID:                "ins_wh_1",
		WarehouseID:       warehouseID,
		WarehouseName:     "Warehouse",
		ProductID:         "prod_demo_1",
		ProductName:       "Demo SKU",
		Urgency:           "HIGH",
		CurrentStock:      12,
		AvgDailyVelocity:  4.5,
		DaysUntilStockout: 3,
		ReorderQuantity:   48,
		Status:            "OPEN",
		CreatedAt:         now,
	}}
}
