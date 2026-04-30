package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	kafkaEvents "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
)

// ─── Bifurcated Recovery ──────────────────────────────────────────────────────
// Two exception paths for order rejection:
//
//   Hard Kill (Admin Rejection / CANCELLED_BY_ORIGIN):
//     Warehouse admin or supplier admin cancels an order. State → CANCELLED_BY_ORIGIN.
//     Removed from manifest. Pending ledger entries voided by the Treasurer.
//     3-way notification: warehouse + supplier + retailer.
//
//   Soft Stop (Payload Overflow / RECOVERY_PENDING):
//     Payloader reports order doesn't fit. State → READY_FOR_DISPATCH.
//     ManifestId cleared. Order returns to the unassigned pool for redispatch.
//     Supplier-only notification.

// HandleOrderRejection handles POST /v1/warehouse/ops/orders/{id}/reject.
// Warehouse-scoped admin cancels an order — hard kill path.
func (s *OrderService) HandleOrderRejection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		ops := auth.GetWarehouseOps(r.Context())
		if ops == nil {
			http.Error(w, `{"error":"warehouse scope required"}`, http.StatusForbidden)
			return
		}

		parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/reject"), "/")
		orderID := parts[len(parts)-1]
		if orderID == "" {
			http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
			return
		}

		var body struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Reason == "" {
			http.Error(w, `{"error":"reason required"}`, http.StatusBadRequest)
			return
		}

		err := s.cancelByOrigin(r.Context(), orderID, ops.SupplierID, ops.WarehouseID, ops.UserID, body.Reason)
		if err != nil {
			var sc *ErrStateConflict
			if isStateConflict(err, &sc) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"error": sc.Error()})
				return
			}
			log.Printf("[RECOVERY] reject order %s failed: %v", orderID, err)
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "cancelled_by_origin", "order_id": orderID})
	}
}

// HandlePayloadOverflow handles POST /v1/warehouse/ops/orders/{id}/overflow.
// Payloader or warehouse ops reports a payload overflow — soft stop, order returns to pool.
func (s *OrderService) HandlePayloadOverflow() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		ops := auth.GetWarehouseOps(r.Context())
		if ops == nil {
			http.Error(w, `{"error":"warehouse scope required"}`, http.StatusForbidden)
			return
		}

		parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/overflow"), "/")
		orderID := parts[len(parts)-1]
		if orderID == "" {
			http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
			return
		}

		var body struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			body.Reason = "OVERFLOW"
		}
		if body.Reason == "" {
			body.Reason = "OVERFLOW"
		}

		err := s.recoverOverflow(r.Context(), orderID, ops.SupplierID, ops.WarehouseID, body.Reason)
		if err != nil {
			var sc *ErrStateConflict
			if isStateConflict(err, &sc) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"error": sc.Error()})
				return
			}
			log.Printf("[RECOVERY] overflow order %s failed: %v", orderID, err)
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ready_for_dispatch", "order_id": orderID})
	}
}

// cancelByOrigin — Hard Kill: CANCELLED_BY_ORIGIN + remove from manifest + outbox event.
// Treasurer voids pending ledger entries when it receives ORDER_CANCELLED_BY_ORIGIN.
func (s *OrderService) cancelByOrigin(ctx context.Context, orderID, supplierID, warehouseID, userID, reason string) error {
	txCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := s.Client.ReadWriteTransaction(txCtx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Read current state + scope verification.
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID},
			[]string{"State", "SupplierId", "WarehouseId", "RetailerId", "ManifestId", "TotalAmount"})
		if err != nil {
			return fmt.Errorf("read order %s: %w", orderID, err)
		}

		var state, sid, whid spanner.NullString
		var retailerID, manifestID spanner.NullString
		var amount spanner.NullInt64
		if err := row.Columns(&state, &sid, &whid, &retailerID, &manifestID, &amount); err != nil {
			return fmt.Errorf("parse order %s: %w", orderID, err)
		}

		// Scope guard: must belong to this supplier + warehouse.
		if sid.StringVal != supplierID || whid.StringVal != warehouseID {
			return fmt.Errorf("order %s does not belong to warehouse %s: %w",
				orderID, warehouseID, &ErrStateConflict{OrderID: orderID, CurrentState: "SCOPE_MISMATCH", AttemptedOp: "cancel"})
		}

		// State guard: only pre-delivery states can be killed.
		allowed := map[string]bool{
			"PENDING": true, "READY_FOR_DISPATCH": true, "DISPATCHED": true,
			"LOADING": true, "DELAYED": true,
		}
		if !allowed[state.StringVal] {
			return &ErrStateConflict{OrderID: orderID, CurrentState: state.StringVal, AttemptedOp: "cancel_by_origin"}
		}

		now := time.Now()
		mutations := []*spanner.Mutation{
			spanner.Update("Orders",
				[]string{"OrderId", "State", "ManifestId", "UpdatedAt"},
				[]interface{}{orderID, "CANCELLED_BY_ORIGIN", spanner.NullString{}, now}),
		}
		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		return outbox.EmitJSON(txn, "Order", orderID, kafkaEvents.EventOrderCancelledByOrigin, kafkaEvents.TopicMain,
			kafkaEvents.OrderCancelledByOriginEvent{
				OrderID:     orderID,
				SupplierId:  supplierID,
				WarehouseId: warehouseID,
				RetailerId:  retailerID.StringVal,
				ManifestID:  manifestID.StringVal,
				Reason:      reason,
				CancelledBy: userID,
				Amount:      amount.Int64,
				Timestamp:   now,
			}, telemetry.TraceIDFromContext(ctx))
	})
	return err
}

// recoverOverflow — Soft Stop: order → READY_FOR_DISPATCH, cleared from manifest, back to pool.
func (s *OrderService) recoverOverflow(ctx context.Context, orderID, supplierID, warehouseID, reason string) error {
	txCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := s.Client.ReadWriteTransaction(txCtx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID},
			[]string{"State", "SupplierId", "WarehouseId", "ManifestId"})
		if err != nil {
			return fmt.Errorf("read order %s: %w", orderID, err)
		}

		var state, sid, whid, manifestID spanner.NullString
		if err := row.Columns(&state, &sid, &whid, &manifestID); err != nil {
			return fmt.Errorf("parse order %s: %w", orderID, err)
		}

		if sid.StringVal != supplierID || whid.StringVal != warehouseID {
			return fmt.Errorf("order %s scope mismatch: %w",
				orderID, &ErrStateConflict{OrderID: orderID, CurrentState: "SCOPE_MISMATCH", AttemptedOp: "overflow"})
		}

		// Only LOADING / DISPATCHED / DELAYED can overflow.
		allowed := map[string]bool{"LOADING": true, "DISPATCHED": true, "DELAYED": true}
		if !allowed[state.StringVal] {
			return &ErrStateConflict{OrderID: orderID, CurrentState: state.StringVal, AttemptedOp: "payload_overflow"}
		}

		now := time.Now()
		mutations := []*spanner.Mutation{
			spanner.Update("Orders",
				[]string{"OrderId", "State", "ManifestId", "DriverId", "IsRecovery", "UpdatedAt"},
				[]interface{}{orderID, "READY_FOR_DISPATCH", spanner.NullString{}, spanner.NullString{}, true, now}),
		}
		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		return outbox.EmitJSON(txn, "Order", orderID, kafkaEvents.EventPayloadOverflow, kafkaEvents.TopicMain,
			kafkaEvents.PayloadOverflowEvent{
				OrderID:     orderID,
				SupplierId:  supplierID,
				WarehouseId: warehouseID,
				ManifestID:  manifestID.StringVal,
				Reason:      reason,
				Timestamp:   now,
			}, telemetry.TraceIDFromContext(ctx))
	})
	return err
}

// isStateConflict is a helper to unwrap ErrStateConflict from wrapped errors.
func isStateConflict(err error, target **ErrStateConflict) bool {
	var sc *ErrStateConflict
	if errors.As(err, &sc) {
		*target = sc
		return true
	}
	return false
}
