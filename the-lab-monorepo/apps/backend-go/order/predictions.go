package order

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/kafka"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// HandleListPredictions returns saved AI predictions for a retailer.
// GET /v1/ai/predictions?retailer_id=X
func HandleListPredictions(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		retailerID := r.URL.Query().Get("retailer_id")
		// Auto-fill from JWT claims when query param is missing (mobile clients)
		if retailerID == "" {
			if claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims); ok && claims.Role == "RETAILER" {
				retailerID = claims.UserID
			}
		}
		if retailerID == "" {
			http.Error(w, `{"error":"retailer_id required"}`, http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		stmt := spanner.Statement{
			SQL: `SELECT PredictionId, RetailerId, PredictedAmount, TriggerDate, Status
			      FROM AIPredictions
			      WHERE RetailerId = @retailerId
			      ORDER BY TriggerDate DESC
			      LIMIT 20`,
			Params: map[string]interface{}{
				"retailerId": retailerID,
			},
		}

		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()

		type PredictionResponse struct {
			ID                 string  `json:"id"`
			RetailerID         string  `json:"retailer_id"`
			ProductID          string  `json:"product_id"`
			ProductName        string  `json:"product_name"`
			PredictedAmount int64   `json:"predicted_amount"`
			PredictedQuantity  int     `json:"predicted_quantity"`
			TriggerDate        string  `json:"trigger_date"`
			Status             string  `json:"status"`
			Confidence         float64 `json:"confidence"`
			Reasoning          string  `json:"reasoning"`
			SuggestedOrderDate string  `json:"suggested_order_date"`
		}

		var predictions []PredictionResponse
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[ai] Failed to query predictions: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			var p PredictionResponse
			var triggerDate spanner.NullTime
			if err := row.Columns(&p.ID, &p.RetailerID, &p.PredictedAmount, &triggerDate, &p.Status); err != nil {
				log.Printf("[ai] Failed to parse prediction row: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if triggerDate.Valid {
				p.TriggerDate = triggerDate.Time.Format("2006-01-02T15:04:05Z")
				p.SuggestedOrderDate = p.TriggerDate
			}

			// Defaults until hydrated from prediction items below
			p.ProductName = "Predicted Order"
			p.PredictedQuantity = int(p.PredictedAmount / 50000)
			if p.PredictedQuantity < 1 {
				p.PredictedQuantity = 1
			}
			p.Confidence = 0.85
			p.Reasoning = "Based on your recent order patterns"

			predictions = append(predictions, p)
		}

		if predictions == nil {
			predictions = []PredictionResponse{}
		}

		// Hydrate product_id and product_name from AIPredictionItems
		if len(predictions) > 0 {
			predIDs := make([]string, len(predictions))
			for i, p := range predictions {
				predIDs[i] = p.ID
			}
			itemStmt := spanner.Statement{
				SQL: `SELECT pi.PredictionId, pi.SkuId, COALESCE(sp.Name, pi.SkuId) AS SkuName, pi.PredictedQuantity
				      FROM AIPredictionItems pi
				      LEFT JOIN SupplierProducts sp ON pi.SkuId = sp.SkuId
				      WHERE pi.PredictionId IN UNNEST(@predIds)
				      ORDER BY pi.PredictedQuantity DESC`,
				Params: map[string]interface{}{"predIds": predIDs},
			}
			itemIter := client.Single().Query(ctx, itemStmt)
			defer itemIter.Stop()

			// Map first (highest-quantity) item per prediction
			seenPred := map[string]bool{}
			predItemMap := map[string][2]string{} // predId -> [skuId, skuName]
			predQtyMap := map[string]int64{}      // predId -> quantity
			for {
				row, err := itemIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					break // non-fatal
				}
				var predId, skuId, skuName string
				var qty int64
				if err := row.Columns(&predId, &skuId, &skuName, &qty); err != nil {
					continue
				}
				if !seenPred[predId] {
					seenPred[predId] = true
					predItemMap[predId] = [2]string{skuId, skuName}
					predQtyMap[predId] = qty
				}
			}
			for i := range predictions {
				if item, ok := predItemMap[predictions[i].ID]; ok {
					predictions[i].ProductID = item[0]
					predictions[i].ProductName = item[1]
				}
				if qty, ok := predQtyMap[predictions[i].ID]; ok {
					predictions[i].PredictedQuantity = int(qty)
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(predictions)
	}
}

// HandlePatchPrediction allows a retailer to correct an AI_PLANNED prediction,
// emitting a Kafka event for the Empathy Engine's RLHF loop.
// PATCH /v1/ai/predictions?prediction_id=X
func HandlePatchPrediction(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		predictionID := r.URL.Query().Get("prediction_id")
		if predictionID == "" {
			http.Error(w, `{"error":"prediction_id required"}`, http.StatusBadRequest)
			return
		}

		var req struct {
			Amount   *int64  `json:"amount"`
			TriggerDate *string `json:"trigger_date"`
			Status      *string `json:"status"` // "REJECTED" to cancel
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		// Read current prediction state
		row, err := client.Single().ReadRow(ctx, "AIPredictions",
			spanner.Key{predictionID},
			[]string{"PredictionId", "RetailerId", "PredictedAmount", "TriggerDate", "Status", "WarehouseId"})
		if err != nil {
			http.Error(w, `{"error":"prediction not found"}`, http.StatusNotFound)
			return
		}

		var currentRetailer, currentStatus string
		var currentAmount int64
		var currentTrigger spanner.NullTime
		var currentWarehouse spanner.NullString
		if err := row.Columns(&predictionID, &currentRetailer, &currentAmount, &currentTrigger, &currentStatus, &currentWarehouse); err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		warehouseId := ""
		if currentWarehouse.Valid {
			warehouseId = currentWarehouse.StringVal
		}

		if currentStatus != "WAITING" {
			http.Error(w, fmt.Sprintf(`{"error":"prediction is %s, only WAITING can be corrected"}`, currentStatus), http.StatusConflict)
			return
		}

		// Build mutations and collect diffs for post-commit emission (outbox pattern)
		cols := []string{"PredictionId"}
		vals := []interface{}{predictionID}

		// Pending events — emitted ONLY after Spanner commit succeeds
		type pendingDateShift struct {
			oldDate string
			newDate string
		}
		type pendingSkuMod struct {
			field    string
			oldValue string
			newValue string
		}
		var dateShift *pendingDateShift
		var skuMods []pendingSkuMod

		if req.Status != nil && (*req.Status == "REJECTED" || *req.Status == "DISMISSED") {
			cols = append(cols, "Status")
			vals = append(vals, "REJECTED")
			skuMods = append(skuMods, pendingSkuMod{
				field: "rejected", oldValue: currentStatus, newValue: "REJECTED",
			})
		} else {
			if req.Amount != nil {
				cols = append(cols, "PredictedAmount")
				vals = append(vals, *req.Amount)
				skuMods = append(skuMods, pendingSkuMod{
					field:    "amount",
					oldValue: fmt.Sprintf("%d", currentAmount),
					newValue: fmt.Sprintf("%d", *req.Amount),
				})
			}
			if req.TriggerDate != nil {
				parsed, parseErr := time.Parse(time.RFC3339, *req.TriggerDate)
				if parseErr != nil {
					http.Error(w, `{"error":"invalid trigger_date format (RFC3339)"}`, http.StatusBadRequest)
					return
				}
				cols = append(cols, "TriggerDate")
				vals = append(vals, parsed)

				oldTD := ""
				if currentTrigger.Valid {
					oldTD = currentTrigger.Time.Format(time.RFC3339)
				}
				dateShift = &pendingDateShift{oldDate: oldTD, newDate: *req.TriggerDate}
			}
		}

		if len(cols) == 1 {
			http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
			return
		}

		_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("AIPredictions", cols, vals),
			})
		})
		if err != nil {
			log.Printf("[AI_FEEDBACK] Spanner update error: %v", err)
			http.Error(w, `{"error":"update failed"}`, http.StatusInternalServerError)
			return
		}

		// ── Outbox: emit granular events ONLY after Spanner commit ──
		now := time.Now().UnixMilli()

		if dateShift != nil {
			if emitErr := kafka.EmitDateShift(kafka.AIPlanDateShiftEvent{
				PredictionID: predictionID,
				RetailerID:   currentRetailer,
				WarehouseId:  warehouseId,
				OldDate:      dateShift.oldDate,
				NewDate:      dateShift.newDate,
				Timestamp:    now,
			}); emitErr != nil {
				log.Printf("[AI_FEEDBACK] EmitDateShift error: %v", emitErr)
			}
		}

		for _, mod := range skuMods {
			if emitErr := kafka.EmitSkuModified(kafka.AIPlanSkuModifiedEvent{
				PredictionID: predictionID,
				RetailerID:   currentRetailer,
				WarehouseId:  warehouseId,
				SkuID:        predictionID, // prediction-level (no line items yet)
				Field:        mod.field,
				OldValue:     mod.oldValue,
				NewValue:     mod.newValue,
				Timestamp:    now,
			}); emitErr != nil {
				log.Printf("[AI_FEEDBACK] EmitSkuModified error: %v", emitErr)
			}
		}

		// Legacy bulk event for backward compat
		for _, mod := range skuMods {
			if emitErr := kafka.EmitPredictionCorrected(kafka.PredictionCorrectedEvent{
				PredictionID: predictionID,
				RetailerID:   currentRetailer,
				WarehouseId:  warehouseId,
				FieldChanged: mod.field,
				OldValue:     mod.oldValue,
				NewValue:     mod.newValue,
				Timestamp:    now,
			}); emitErr != nil {
				log.Printf("[AI_FEEDBACK] EmitPredictionCorrected legacy error: %v", emitErr)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "corrected", "prediction_id": predictionID})
	}
}
