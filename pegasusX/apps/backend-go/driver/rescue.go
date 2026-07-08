package driver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// HandleRescueRequest handles a driver reporting that their truck is broken and needs rescue.
func (s *Service) HandleRescueRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	spannerRepo, ok := s.repo.(*SpannerRepository)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "spanner_not_configured"})
		return
	}

	rescueID := uuid.NewString()

	_, err := spannerRepo.client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Update driver's truck status to NEEDS_RESCUE
		stmt := spanner.Statement{
			SQL:    `UPDATE Drivers SET TruckStatus = 'NEEDS_RESCUE' WHERE Id = @id`,
			Params: map[string]any{"id": driverID},
		}
		if _, err := txn.Update(ctx, stmt); err != nil {
			return err
		}

		// Look up SupplierID and WarehouseID for the driver to route the event
		var supplierID, warehouseID string
		lookupStmt := spanner.Statement{
			SQL:    `SELECT SupplierId, AssignedWarehouseId FROM Drivers WHERE Id = @id`,
			Params: map[string]any{"id": driverID},
		}
		iter := txn.Query(ctx, lookupStmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err == nil {
			var sp, wh spanner.NullString
			if err := row.Columns(&sp, &wh); err == nil {
				supplierID = sp.StringVal
				warehouseID = wh.StringVal
			}
		}

		ev := events.RescueEvent{
			RescueID:       rescueID,
			BrokenDriverID: driverID,
			Status:         "REQUESTED",
			WarehouseID:    warehouseID,
			SupplierID:     supplierID,
		}
		ev.Type = "RESCUE_REQUESTED"
		evJSON, _ := json.Marshal(ev)

		m := spanner.InsertMap("OutboxEvents", map[string]any{
			"EventId":       uuid.NewString(),
			"AggregateType": events.AggregateDriver,
			"AggregateId":   driverID,
			"TopicName":     events.TopicMain,
			"Payload":       string(evJSON),
			"CreatedAt":     spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})

	if err != nil {
		s.log.Error("failed to request rescue", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "requested", "rescue_id": rescueID})
}

// RescueRespondRequest is the JSON payload for a rescue driver's response.
type RescueRespondRequest struct {
	RescueID       string `json:"rescue_id"`
	BrokenDriverID string `json:"broken_driver_id"`
	Accept         bool   `json:"accept"`
}

// HandleRescueRespond handles the accept/reject response from a rescue driver.
func (s *Service) HandleRescueRespond(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req RescueRespondRequest
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err := json.Unmarshal(body, &req); err != nil || req.BrokenDriverID == "" || req.RescueID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	spannerRepo, ok := s.repo.(*SpannerRepository)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "spanner_not_configured"})
		return
	}

	_, err := spannerRepo.client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var supplierID, warehouseID string
		lookupStmt := spanner.Statement{
			SQL:    `SELECT SupplierId, AssignedWarehouseId FROM Drivers WHERE Id = @id`,
			Params: map[string]any{"id": driverID},
		}
		iterLookup := txn.Query(ctx, lookupStmt)
		defer iterLookup.Stop()
		row, err := iterLookup.Next()
		if err == nil {
			var sp, wh spanner.NullString
			if err := row.Columns(&sp, &wh); err == nil {
				supplierID = sp.StringVal
				warehouseID = wh.StringVal
			}
		}

		status := "REJECTED"
		var mutations []*spanner.Mutation

		if req.Accept {
			status = "ACCEPTED"

			// Find all active orders for the broken driver to reassign
			findOrdersStmt := spanner.Statement{
				SQL: `SELECT Id, RetailerId FROM Orders 
				      WHERE DriverId = @oldDriverId AND Status IN ('LOADED', 'IN_TRANSIT', 'ARRIVED', 'DISPATCHED', 'ARRIVING', 'EN_ROUTE', 'AWAITING_PAYMENT', 'PENDING_CASH_COLLECTION')`,
				Params: map[string]any{"oldDriverId": req.BrokenDriverID},
			}
			iterOrders := txn.Query(ctx, findOrdersStmt)
			var affectedOrders []struct {
				OrderID    string
				RetailerID string
			}
			for {
				row, err := iterOrders.Next()
				if err != nil { // handles iterator.Done internally if we check later or just use iterOrders.Stop()
					break 
				}
				var orderID, retailerID string
				if err := row.Columns(&orderID, &retailerID); err == nil {
					affectedOrders = append(affectedOrders, struct {
						OrderID    string
						RetailerID string
					}{orderID, retailerID})
				}
			}
			iterOrders.Stop()

			// Reassign orders to the Rescue Driver
			updateStmt := spanner.Statement{
				SQL: `UPDATE Orders 
				      SET DriverId = @newDriverId 
				      WHERE DriverId = @oldDriverId AND Status IN ('LOADED', 'IN_TRANSIT', 'ARRIVED', 'DISPATCHED', 'ARRIVING', 'EN_ROUTE', 'AWAITING_PAYMENT', 'PENDING_CASH_COLLECTION')`,
				Params: map[string]any{
					"newDriverId": driverID,
					"oldDriverId": req.BrokenDriverID,
				},
			}
			if _, err := txn.Update(ctx, updateStmt); err != nil {
				return err
			}

			// Mark the old driver's truck as BROKEN_DOWN instead of NEEDS_RESCUE
			truckStmt := spanner.Statement{
				SQL:    `UPDATE Drivers SET TruckStatus = 'BROKEN_DOWN' WHERE Id = @oldDriverId`,
				Params: map[string]any{"oldDriverId": req.BrokenDriverID},
			}
			if _, err := txn.Update(ctx, truckStmt); err != nil {
				return err
			}

			// Look up Rescue Driver's License Plate for the event
			var rescueLicensePlate string
			lpStmt := spanner.Statement{
				SQL:    `SELECT LicensePlate FROM Drivers WHERE Id = @id`,
				Params: map[string]any{"id": driverID},
			}
			iterLP := txn.Query(ctx, lpStmt)
			if row, err := iterLP.Next(); err == nil {
				var lp spanner.NullString
				if err := row.Columns(&lp); err == nil {
					rescueLicensePlate = lp.StringVal
				}
			}
			iterLP.Stop()

			// Emit EventOrderReassigned for each affected order
			for _, ao := range affectedOrders {
				evReassigned := events.OrderEvent{
					BaseEvent: events.BaseEvent{
						Type: events.EventOrderReassigned,
					},
					OrderID:      ao.OrderID,
					SupplierID:   supplierID,
					RetailerID:   ao.RetailerID,
					WarehouseID:  warehouseID,
					ToDriverID:   driverID,
					FromDriverID: req.BrokenDriverID,
					LicensePlate: rescueLicensePlate,
				}
				evJSON, _ := json.Marshal(evReassigned)
				mutations = append(mutations, spanner.InsertMap("OutboxEvents", map[string]any{
					"EventId":       uuid.NewString(),
					"AggregateType": events.AggregateOrder,
					"AggregateId":   ao.OrderID,
					"TopicName":     events.TopicMain,
					"Payload":       string(evJSON),
					"CreatedAt":     spanner.CommitTimestamp,
				}))
			}
		}

		ev := events.RescueEvent{
			RescueID:       req.RescueID,
			BrokenDriverID: req.BrokenDriverID,
			RescueDriverID: driverID,
			Status:         status,
			WarehouseID:    warehouseID,
			SupplierID:     supplierID,
		}
		ev.Type = "RESCUE_" + status
		evJSON, _ := json.Marshal(ev)
		mutations = append(mutations, spanner.InsertMap("OutboxEvents", map[string]any{
			"EventId":       uuid.NewString(),
			"AggregateType": events.AggregateDriver,
			"AggregateId":   driverID,
			"TopicName":     events.TopicMain,
			"Payload":       string(evJSON),
			"CreatedAt":     spanner.CommitTimestamp,
		}))

		return txn.BufferWrite(mutations)
	})

	if err != nil {
		s.log.Error("failed to respond to rescue", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}
