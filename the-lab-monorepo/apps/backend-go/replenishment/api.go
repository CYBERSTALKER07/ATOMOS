package replenishment

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	kafkaevents "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// InsightResponse is the JSON shape returned by the insights API.
type InsightResponse struct {
	InsightId         string  `json:"insight_id"`
	WarehouseId       string  `json:"warehouse_id"`
	ProductId         string  `json:"product_id"`
	ProductName       string  `json:"product_name,omitempty"`
	SupplierId        string  `json:"supplier_id"`
	CurrentStock      int64   `json:"current_stock"`
	DailyBurnRate     float64 `json:"daily_burn_rate"`
	TimeToEmptyDays   float64 `json:"time_to_empty_days"`
	SuggestedQuantity int64   `json:"suggested_quantity"`
	UrgencyLevel      string  `json:"urgency_level"`
	ReasonCode        string  `json:"reason_code"`
	Status            string  `json:"status"`
	TargetFactoryId   string  `json:"target_factory_id,omitempty"`
	DemandBreakdown   string  `json:"demand_breakdown,omitempty"`
	CreatedAt         string  `json:"created_at"`
}

// HandleInsights — GET /v1/warehouse/replenishment/insights
// Lists insights for the current warehouse scope. Defaults to PENDING status.
func HandleInsights(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		whID := auth.EffectiveWarehouseID(r.Context())
		if whID == "" {
			http.Error(w, `{"error":"warehouse scope required"}`, http.StatusBadRequest)
			return
		}

		status := r.URL.Query().Get("status")
		if status == "" {
			status = "PENDING"
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		stmt := spanner.Statement{
			SQL: `SELECT ri.InsightId, ri.WarehouseId, ri.ProductId,
			             COALESCE(sp.Name, ri.ProductId), ri.SupplierId,
			             ri.CurrentStock, ri.DailyBurnRate, ri.TimeToEmptyDays,
			             ri.SuggestedQuantity, ri.UrgencyLevel, ri.ReasonCode,
			             ri.Status, COALESCE(ri.TargetFactoryId, ''),
			             COALESCE(ri.DemandBreakdown, ''), ri.CreatedAt
			      FROM ReplenishmentInsights ri
			      LEFT JOIN SupplierProducts sp ON ri.ProductId = sp.SkuId
			      WHERE ri.WarehouseId = @whId AND ri.Status = @status
			      ORDER BY CASE ri.UrgencyLevel
			                 WHEN 'CRITICAL' THEN 1
			                 WHEN 'WARNING' THEN 2
			                 ELSE 3
			               END,
			               ri.CreatedAt DESC`,
			Params: map[string]interface{}{"whId": whID, "status": status},
		}
		iter := spannerClient.Single().Query(ctx, stmt)
		defer iter.Stop()

		var insights []InsightResponse
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[REPLENISHMENT API] query error: %v", err)
				break
			}
			var ins InsightResponse
			var createdAt time.Time
			if err := row.Columns(
				&ins.InsightId, &ins.WarehouseId, &ins.ProductId,
				&ins.ProductName, &ins.SupplierId,
				&ins.CurrentStock, &ins.DailyBurnRate, &ins.TimeToEmptyDays,
				&ins.SuggestedQuantity, &ins.UrgencyLevel, &ins.ReasonCode,
				&ins.Status, &ins.TargetFactoryId,
				&ins.DemandBreakdown, &createdAt,
			); err != nil {
				log.Printf("[REPLENISHMENT API] row parse error: %v", err)
				continue
			}
			ins.CreatedAt = createdAt.Format(time.RFC3339)
			insights = append(insights, ins)
		}

		if insights == nil {
			insights = []InsightResponse{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": insights})
	}
}

// HandleInsightAction — POST /v1/warehouse/replenishment/insights/{id}/approve
//
//	POST /v1/warehouse/replenishment/insights/{id}/dismiss
//
// One-click approve creates an InternalTransferOrder; dismiss marks it DISMISSED.
func HandleInsightAction(spannerClient *spanner.Client, producer *kafka.Writer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse path: /v1/warehouse/replenishment/insights/{id}/{action}
		path := strings.TrimPrefix(r.URL.Path, "/v1/warehouse/replenishment/insights/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 || parts[0] == "" {
			http.Error(w, `{"error":"path must be /insights/{id}/{approve|dismiss}"}`, http.StatusBadRequest)
			return
		}
		insightID := parts[0]
		action := parts[1]

		if action != "approve" && action != "dismiss" {
			http.Error(w, `{"error":"action must be 'approve' or 'dismiss'"}`, http.StatusBadRequest)
			return
		}

		whID := auth.EffectiveWarehouseID(r.Context())
		if whID == "" {
			http.Error(w, `{"error":"warehouse scope required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Read insight via query to safely handle nullable TargetFactoryId
		stmt := spanner.Statement{
			SQL: `SELECT WarehouseId, ProductId, SupplierId, SuggestedQuantity,
			             UrgencyLevel, Status, COALESCE(TargetFactoryId, '')
			      FROM ReplenishmentInsights WHERE InsightId = @iid`,
			Params: map[string]interface{}{"iid": insightID},
		}
		iter := spannerClient.Single().Query(ctx, stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err != nil {
			http.Error(w, `{"error":"insight not found"}`, http.StatusNotFound)
			return
		}

		var rowWhID, productId, supplierId, urgency, status, targetFactory string
		var suggestedQty int64
		if err := row.Columns(&rowWhID, &productId, &supplierId, &suggestedQty,
			&urgency, &status, &targetFactory); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if rowWhID != whID {
			http.Error(w, `{"error":"insight does not belong to this warehouse"}`, http.StatusForbidden)
			return
		}
		if status != "PENDING" {
			http.Error(w, `{"error":"insight already processed"}`, http.StatusConflict)
			return
		}

		if action == "dismiss" {
			_, err := spannerClient.Apply(ctx, []*spanner.Mutation{
				spanner.Update("ReplenishmentInsights",
					[]string{"InsightId", "Status"},
					[]interface{}{insightID, "DISMISSED"}),
			})
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":     "DISMISSED",
				"insight_id": insightID,
			})
			return
		}

		// APPROVE: create InternalTransferOrder from insight
		if targetFactory == "" {
			http.Error(w, `{"error":"no target factory assigned to this insight"}`, http.StatusBadRequest)
			return
		}

		// Get VU for this product
		var vu float64 = 1.0
		vuRow, vuErr := spannerClient.Single().ReadRow(ctx, "SupplierProducts",
			spanner.Key{productId}, []string{"VolumetricUnit"})
		if vuErr == nil {
			_ = vuRow.Columns(&vu)
		}

		transferID := uuid.New().String()
		itemID := uuid.New().String()
		totalVU := float64(suggestedQty) * vu

		mutations := []*spanner.Mutation{
			spanner.Insert("InternalTransferOrders",
				[]string{"TransferId", "FactoryId", "WarehouseId", "SupplierId",
					"State", "TotalVolumeVU", "Source", "CreatedAt"},
				[]interface{}{transferID, targetFactory, whID, supplierId,
					"DRAFT", totalVU, "SYSTEM_THRESHOLD", spanner.CommitTimestamp}),
			spanner.Insert("InternalTransferItems",
				[]string{"TransferId", "ItemId", "ProductId", "Quantity", "VolumeVU"},
				[]interface{}{transferID, itemID, productId, suggestedQty, totalVU}),
			spanner.Update("ReplenishmentInsights",
				[]string{"InsightId", "Status"},
				[]interface{}{insightID, "APPROVED"}),
		}

		if _, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}
			evt := map[string]interface{}{
				"event":        kafkaevents.EventInsightApprovedTransferCreated,
				"insight_id":   insightID,
				"transfer_id":  transferID,
				"factory_id":   targetFactory,
				"warehouse_id": whID,
				"product_id":   productId,
				"quantity":     suggestedQty,
				"timestamp":    time.Now().UTC().Format(time.RFC3339),
			}
			return outbox.EmitJSON(txn, "InternalTransferOrder", transferID,
				kafkaevents.EventInsightApprovedTransferCreated, kafkaevents.TopicMain, evt,
				telemetry.TraceIDFromContext(ctx))
		}); err != nil {
			log.Printf("[REPLENISHMENT API] approve failed: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "APPROVED",
			"insight_id":  insightID,
			"transfer_id": transferID,
		})
	}
}
