package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	kafkaEvents "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
)

// MarkDelayed transitions an order to the DELAYED state and emits
// EventOrderDelayed via the transactional outbox so the retailer + supplier
// admin are notified atomically with the state change.
//
// Allowed source states: PENDING, NO_CAPACITY, LOADED. Any other state returns
// an *ErrStateConflict-style error (409 at the handler layer).
func (s *OrderService) MarkDelayed(ctx context.Context, orderID, reason string) error {
	if orderID == "" {
		return fmt.Errorf("mark delayed: empty order id")
	}
	if reason == "" {
		reason = "MANUAL"
	}

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders",
			spanner.Key{orderID},
			[]string{"OrderId", "State", "RetailerId", "SupplierId", "WarehouseId", "ManifestId"})
		if err != nil {
			return fmt.Errorf("read order: %w", err)
		}

		var (
			id          string
			state       string
			retailerID  spanner.NullString
			supplierID  spanner.NullString
			warehouseID spanner.NullString
			manifestID  spanner.NullString
		)
		if err := row.Columns(&id, &state, &retailerID, &supplierID, &warehouseID, &manifestID); err != nil {
			return fmt.Errorf("decode order: %w", err)
		}

		switch state {
		case "PENDING", "NO_CAPACITY", "LOADED":
			// allowed
		default:
			return fmt.Errorf("mark delayed: order %s in state %s cannot be delayed", orderID, state)
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("Orders",
				[]string{"OrderId", "State", "UpdatedAt"},
				[]interface{}{orderID, "DELAYED", spanner.CommitTimestamp}),
		}); err != nil {
			return err
		}

		return outbox.EmitJSON(txn, "Order", orderID, kafkaEvents.EventOrderDelayed, kafkaEvents.TopicMain, kafkaEvents.OrderDelayedEvent{
			OrderID:     orderID,
			RetailerID:  retailerID.StringVal,
			SupplierID:  supplierID.StringVal,
			WarehouseID: warehouseID.StringVal,
			ManifestID:  manifestID.StringVal,
			Reason:      reason,
			Timestamp:   time.Now().UTC(),
		}, telemetry.TraceIDFromContext(ctx))
	})

	return err
}

// HandleMarkDelayed exposes MarkDelayed over HTTP for the warehouse ops surface.
//
//	POST /v1/warehouse/ops/orders/{id}/delay
//	Body: {"reason": "CAPACITY_OVERFLOW" | "OPS_HOLD" | ...}
//
// Auth/scope is enforced by the routes package (WAREHOUSE role + ops-scope).
// Returns 409 on invalid source state, 404 when the order is missing,
// 200 with {"order_id":"...","state":"DELAYED"} on success.
func (s *OrderService) HandleMarkDelayed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if _, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims); !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		orderID := chi.URLParam(r, "id")
		if orderID == "" {
			http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
			return
		}

		var body struct {
			Reason string `json:"reason"`
		}
		// Body is optional — empty body means MarkDelayed defaults reason to MANUAL.
		_ = json.NewDecoder(r.Body).Decode(&body)

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		traceID := r.Header.Get("X-Request-Id")

		if err := s.MarkDelayed(ctx, orderID, body.Reason); err != nil {
			switch {
			case errors.Is(err, spanner.ErrRowNotFound) || strings.Contains(err.Error(), "NotFound"):
				http.Error(w, fmt.Sprintf(`{"error":"order_not_found","order_id":%q}`, orderID), http.StatusNotFound)
				return
			case strings.Contains(err.Error(), "cannot be delayed"):
				http.Error(w, fmt.Sprintf(`{"error":"invalid_state","detail":%q}`, err.Error()), http.StatusConflict)
				return
			default:
				slog.ErrorContext(ctx, "mark delayed failed",
					"trace_id", traceID, "order_id", orderID, "reason", body.Reason, "err", err)
				http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
				return
			}
		}

		slog.InfoContext(ctx, "order marked DELAYED",
			"trace_id", traceID, "order_id", orderID, "reason", body.Reason)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"order_id": orderID,
			"state":    "DELAYED",
		})
	}
}
