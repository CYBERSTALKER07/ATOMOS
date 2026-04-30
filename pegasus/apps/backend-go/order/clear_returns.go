// Package order — Clear Returns Endpoint (Phantom Cargo fix)
//
// POST /v1/vehicle/{vehicleId}/clear-returns
//
// When a driver returns to the depot with physically rejected items on the
// truck, the supplier (admin) confirms receipt by calling this endpoint.
// It timestamps ReturnClearedAt on all pending rejected line items for the
// vehicle, releasing the locked VU from GetTruckCapacity calculations.
package order

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"

	"backend-go/auth"
)

const (
	eventReturnsCleared = "RETURNS_CLEARED"
	topicReturnsCleared = "returns.cleared"
)

// ReturnsClearedEvent is emitted after a supplier confirms return receipt at depot.
type ReturnsClearedEvent struct {
	VehicleID   string    `json:"vehicle_id"`
	SupplierID  string    `json:"supplier_id"`
	RowsCleared int64     `json:"rows_cleared"`
	ClearedAt   time.Time `json:"cleared_at"`
}

// HandleClearReturns handles POST /v1/vehicle/{vehicleId}/clear-returns.
//
// Sets ReturnClearedAt = CURRENT_TIMESTAMP on all OrderLineItems where
// RejectedQty > 0 AND ReturnClearedAt IS NULL for the given vehicle's route.
func (s *OrderService) HandleClearReturns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract vehicleId from URL: /v1/vehicle/{vehicleId}/clear-returns
	// Expected segments: ["", "v1", "vehicle", "{vehicleId}", "clear-returns"]
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 || parts[len(parts)-1] != "clear-returns" {
		http.Error(w, `{"error":"invalid path"}`, http.StatusBadRequest)
		return
	}
	vehicleID := parts[len(parts)-2]
	if vehicleID == "" {
		http.Error(w, `{"error":"vehicle_id is required"}`, http.StatusBadRequest)
		return
	}

	// Extract supplier identity from JWT claims
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	var rowsCleared int64
	clearedAt := time.Now().UTC()

	_, err := s.Client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Verify the vehicle belongs to this supplier
		ownerStmt := spanner.Statement{
			SQL:    `SELECT SupplierId FROM Vehicles WHERE VehicleId = @vid LIMIT 1`,
			Params: map[string]interface{}{"vid": vehicleID},
		}
		ownerIter := txn.Query(ctx, ownerStmt)
		ownerRow, ownerErr := ownerIter.Next()
		ownerIter.Stop()
		if ownerErr != nil {
			return fmt.Errorf("vehicle %s not found", vehicleID)
		}
		var ownerSupplierId string
		if err := ownerRow.Columns(&ownerSupplierId); err != nil {
			return fmt.Errorf("vehicle owner scan failed: %w", err)
		}
		if ownerSupplierId != supplierID {
			return fmt.Errorf("vehicle does not belong to supplier")
		}

		// SET ReturnClearedAt on all rejected line items for this vehicle's active route
		clearStmt := spanner.Statement{
			SQL: `UPDATE OrderLineItems
			      SET ReturnClearedAt = PENDING_COMMIT_TIMESTAMP()
			      WHERE LineItemId IN (
			        SELECT li.LineItemId
			        FROM Orders o
			        JOIN OrderLineItems li ON o.OrderId = li.OrderId
			        JOIN Drivers d ON o.RouteId = d.DriverId
			        WHERE d.VehicleId = @vehicleId
			          AND li.RejectedQty > 0
			          AND li.ReturnClearedAt IS NULL
			      )`,
			Params: map[string]interface{}{"vehicleId": vehicleID},
		}
		n, updateErr := txn.Update(ctx, clearStmt)
		if updateErr != nil {
			return fmt.Errorf("clear returns update failed: %w", updateErr)
		}
		rowsCleared = n

		return outbox.EmitJSON(txn, "Vehicle", vehicleID, eventReturnsCleared, topicReturnsCleared, ReturnsClearedEvent{
			VehicleID:   vehicleID,
			SupplierID:  supplierID,
			RowsCleared: rowsCleared,
			ClearedAt:   clearedAt,
		}, telemetry.TraceIDFromContext(ctx))
	})

	if err != nil {
		if strings.Contains(err.Error(), "does not belong to supplier") {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		log.Printf("[CLEAR_RETURNS] VehicleId=%s err=%v", vehicleID, err)
		http.Error(w, `{"error":"clear returns failed"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("[CLEAR_RETURNS] VehicleId=%s cleared %d rejected line items", vehicleID, rowsCleared)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"vehicle_id":   vehicleID,
		"rows_cleared": rowsCleared,
		"cleared_at":   clearedAt,
	})
}
