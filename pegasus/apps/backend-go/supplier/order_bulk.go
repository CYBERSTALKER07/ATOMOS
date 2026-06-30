package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/outbox"
	internalKafka "backend-go/kafka"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
)

type bulkDelayRequest struct {
	OrderIDs []string `json:"order_ids"`
	Reason   string   `json:"reason"`
}

type bulkDelayResult struct {
	OrderID string `json:"order_id"`
	State   string `json:"state"`
	Error   string `json:"error,omitempty"`
}

// HandleBulkOrderDelay marks active orders DELAYED or pushes scheduled delivery dates.
// POST /v1/supplier/orders/delay
func (s *OrderVettingService) HandleBulkOrderDelay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	var req bulkDelayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.OrderIDs) == 0 {
		http.Error(w, `{"error":"order_ids required"}`, http.StatusBadRequest)
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "SUPPLIER_BULK_DELAY"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	results := make([]bulkDelayResult, 0, len(req.OrderIDs))
	for _, orderID := range req.OrderIDs {
		orderID = strings.TrimSpace(orderID)
		if orderID == "" {
			continue
		}
		state, err := s.delayOrder(ctx, supplierID, orderID, reason)
		if err != nil {
			results = append(results, bulkDelayResult{OrderID: orderID, Error: err.Error()})
			continue
		}
		results = append(results, bulkDelayResult{OrderID: orderID, State: state})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results":   results,
		"delayed":   countBulkSuccess(results),
		"total":     len(results),
	})
}

func countBulkSuccess(results []bulkDelayResult) int {
	n := 0
	for _, r := range results {
		if r.Error == "" {
			n++
		}
	}
	return n
}

func (s *OrderVettingService) delayOrder(ctx context.Context, supplierID, orderID, reason string) (string, error) {
	var finalState string
	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID},
			[]string{"SupplierId", "State", "RequestedDeliveryDate", "RetailerId", "WarehouseId", "ManifestId"})
		if err != nil {
			return fmt.Errorf("order not found")
		}
		var sid, state string
		var reqDelivery spanner.NullTime
		var retailerID, warehouseID, manifestID spanner.NullString
		if err := row.Columns(&sid, &state, &reqDelivery, &retailerID, &warehouseID, &manifestID); err != nil {
			return err
		}
		if sid != supplierID {
			return fmt.Errorf("order not in supplier scope")
		}

		switch state {
		case "PENDING", "NO_CAPACITY", "LOADED":
			finalState = "DELAYED"
			if err := txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Orders",
					[]string{"OrderId", "State", "UpdatedAt"},
					[]interface{}{orderID, finalState, spanner.CommitTimestamp}),
			}); err != nil {
				return err
			}
			return outbox.EmitJSON(txn, "Order", orderID, internalKafka.EventOrderDelayed, internalKafka.TopicMain, internalKafka.OrderDelayedEvent{
				OrderID:     orderID,
				RetailerID:  retailerID.StringVal,
				SupplierID:  supplierID,
				WarehouseID: warehouseID.StringVal,
				ManifestID:  manifestID.StringVal,
				Reason:      reason,
				Timestamp:   time.Now().UTC(),
			}, telemetry.TraceIDFromContext(ctx))
		case "SCHEDULED":
			base := time.Now().UTC().Add(24 * time.Hour)
			if reqDelivery.Valid {
				base = reqDelivery.Time.Add(24 * time.Hour)
			}
			finalState = "SCHEDULED"
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Orders",
					[]string{"OrderId", "RequestedDeliveryDate", "UpdatedAt"},
					[]interface{}{orderID, base, spanner.CommitTimestamp}),
			})
		default:
			return fmt.Errorf("order %s in state %s cannot be delayed", orderID, state)
		}
	})
	return finalState, err
}
