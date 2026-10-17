package driver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

var (
	errRescueAlreadyAccepted = errors.New("rescue_already_accepted")
	errInsufficientCapacity  = errors.New("insufficient_vehicle_capacity")
	errBrokenDriverNotFound  = errors.New("broken_driver_not_found")
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

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	spannerRepo, ok := s.repo.(*SpannerRepository)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "spanner_not_configured"})
		return
	}

	rescueID := uuid.NewString()
	var outWarehouseID string
	var outSupplierID string

	_, err = spannerRepo.client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Look up SupplierID, WarehouseID and current state
		var supplierID, warehouseID string
		lookupStmt := spanner.Statement{
			SQL:    `SELECT SupplierId, HomeNodeId, UnavailableReason FROM Drivers WHERE DriverId = @id`,
			Params: map[string]any{"id": driverID},
		}
		iter := txn.Query(ctx, lookupStmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err != nil {
			return err
		}
		var sp, wh, reason spanner.NullString
		if err := row.Columns(&sp, &wh, &reason); err != nil {
			return err
		}
		supplierID = sp.StringVal
		warehouseID = wh.StringVal
		outWarehouseID = warehouseID
		outSupplierID = supplierID

		// Mark driver as needing rescue
		stmt := spanner.Statement{
			SQL:    `UPDATE Drivers SET UnavailableReason = 'NEEDS_RESCUE', IsActive = false, UpdatedAt = PENDING_COMMIT_TIMESTAMP() WHERE DriverId = @id`,
			Params: map[string]any{"id": driverID},
		}
		if _, err := txn.Update(ctx, stmt); err != nil {
			return err
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
			"SupplierId":    supplierID,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})

	if err != nil {
		s.log.Error("failed to request rescue", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Broadcast Rescue -> Notify nearby drivers (Post-Commit)
	if s.driverHub != nil && outWarehouseID != "" {
		payload, _ := json.Marshal(map[string]any{
			"type":          "RESCUE_BROADCAST",
			"rescue_id":     rescueID,
			"broken_driver": driverID,
			"warehouse_id":  outWarehouseID,
			"supplier_id":   outSupplierID,
		})
		s.driverHub.Broadcast(context.Background(), "warehouse:"+outWarehouseID, payload)
	}

	resp := map[string]string{"status": "requested", "rescue_id": rescueID}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// RescueRespondRequest is the JSON payload for a rescue driver's response.
type RescueRespondRequest struct {
	RescueID       string `json:"rescue_id"`
	BrokenDriverID string `json:"broken_driver_id"`
	Accept         bool   `json:"accept"`
	Force          bool   `json:"force,omitempty"`
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

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req RescueRespondRequest
	if err := json.Unmarshal(body, &req); err != nil || strings.TrimSpace(req.BrokenDriverID) == "" || strings.TrimSpace(req.RescueID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	spannerRepo, ok := s.repo.(*SpannerRepository)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "spanner_not_configured"})
		return
	}

	var affectedCount int
	_, txnErr := spannerRepo.client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var supplierID, warehouseID string
		lookupStmt := spanner.Statement{
			SQL:    `SELECT SupplierId, HomeNodeId FROM Drivers WHERE DriverId = @id`,
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

			// Step 1: Mutual Exclusion & Race Protection — Verify broken driver is in NEEDS_RESCUE
			brokenStmt := spanner.Statement{
				SQL:    `SELECT UnavailableReason, IsActive, SupplierId, HomeNodeId FROM Drivers WHERE DriverId = @brokenDriverId`,
				Params: map[string]any{"brokenDriverId": req.BrokenDriverID},
			}
			bIter := txn.Query(ctx, brokenStmt)
			bRow, bErr := bIter.Next()
			bIter.Stop()
			if bErr != nil {
				return errBrokenDriverNotFound
			}
			var bReason spanner.NullString
			var bActive bool
			var bSup, bWh spanner.NullString
			if err := bRow.Columns(&bReason, &bActive, &bSup, &bWh); err != nil {
				return err
			}
			if bReason.StringVal != "NEEDS_RESCUE" {
				return errRescueAlreadyAccepted
			}
			if supplierID == "" {
				supplierID = bSup.StringVal
			}
			if warehouseID == "" {
				warehouseID = bWh.StringVal
			}

			// Step 2: Find all active orders for the broken driver
			findOrdersStmt := spanner.Statement{
				SQL: `SELECT OrderId, RetailerId, COALESCE(VolumeVU, 0.0) FROM Orders 
				      WHERE DriverId = @oldDriverId AND Status IN ('LOADED', 'IN_TRANSIT', 'ARRIVED', 'DISPATCHED', 'ARRIVING', 'EN_ROUTE', 'AWAITING_PAYMENT', 'PENDING_CASH_COLLECTION')`,
				Params: map[string]any{"oldDriverId": req.BrokenDriverID},
			}
			iterOrders := txn.Query(ctx, findOrdersStmt)
			var affectedOrders []struct {
				OrderID    string
				RetailerID string
				VolumeVU   float64
			}
			var neededVU float64
			for {
				row, err := iterOrders.Next()
				if err != nil {
					break
				}
				var orderID, retailerID string
				var vol float64
				if err := row.Columns(&orderID, &retailerID, &vol); err == nil {
					affectedOrders = append(affectedOrders, struct {
						OrderID    string
						RetailerID string
						VolumeVU   float64
					}{orderID, retailerID, vol})
					neededVU += vol
				}
			}
			iterOrders.Stop()
			affectedCount = len(affectedOrders)

			// Step 3: Look up Rescue Driver's Vehicle & Active Manifest
			var rescueLicensePlate, rescueVehicleID string
			var rescueMaxVU float64
			lpStmt := spanner.Statement{
				SQL: `SELECT d.VehicleId, COALESCE(v.LicensePlate, ''), COALESCE(v.MaxVolumeVU, 0.0)
				      FROM Drivers d 
				      LEFT JOIN Vehicles v ON d.VehicleId = v.VehicleId 
				      WHERE d.DriverId = @id`,
				Params: map[string]any{"id": driverID},
			}
			iterLP := txn.Query(ctx, lpStmt)
			if row, err := iterLP.Next(); err == nil {
				var vID, lp spanner.NullString
				var maxV float64
				if err := row.Columns(&vID, &lp, &maxV); err == nil {
					rescueVehicleID = vID.StringVal
					rescueLicensePlate = lp.StringVal
					rescueMaxVU = maxV
				}
			}
			iterLP.Stop()

			// Check rescue driver's active manifest capacity
			var rescueManifestID string
			var currentLoadedVU, manifestMaxVU float64
			var stopCount int64
			manifestStmt := spanner.Statement{
				SQL: `SELECT ManifestId, TotalVolumeVU, MaxVolumeVU, StopCount
				      FROM SupplierTruckManifests
				      WHERE DriverId = @id AND State IN ('DISPATCHED', 'IN_TRANSIT', 'SEALED')
				      ORDER BY CreatedAt DESC LIMIT 1`,
				Params: map[string]any{"id": driverID},
			}
			mIter := txn.Query(ctx, manifestStmt)
			if mRow, mErr := mIter.Next(); mErr == nil {
				var mID string
				var tVU, mVU float64
				var sCount int64
				if err := mRow.Columns(&mID, &tVU, &mVU, &sCount); err == nil {
					rescueManifestID = mID
					currentLoadedVU = tVU
					manifestMaxVU = mVU
					stopCount = sCount
				}
			}
			mIter.Stop()

			// Capacity check (allow 5% buffer if not forced)
			effectiveMaxVU := manifestMaxVU
			if effectiveMaxVU == 0 {
				effectiveMaxVU = rescueMaxVU
			}
			if !req.Force && effectiveMaxVU > 0 && (currentLoadedVU+neededVU) > (effectiveMaxVU*1.05) {
				return errInsufficientCapacity
			}

			// Step 4: Reassign orders in Orders table
			updateStmt := spanner.Statement{
				SQL: `UPDATE Orders 
				      SET DriverId = @newDriverId, VehicleId = @newVehicleId, UpdatedAt = PENDING_COMMIT_TIMESTAMP() 
				      WHERE DriverId = @oldDriverId AND Status IN ('LOADED', 'IN_TRANSIT', 'ARRIVED', 'DISPATCHED', 'ARRIVING', 'EN_ROUTE', 'AWAITING_PAYMENT', 'PENDING_CASH_COLLECTION')`,
				Params: map[string]any{
					"newDriverId":  driverID,
					"newVehicleId": rescueVehicleID,
					"oldDriverId":  req.BrokenDriverID,
				},
			}
			if _, err := txn.Update(ctx, updateStmt); err != nil {
				return err
			}

			// Step 5: Update ManifestOrders and Manifests
			now := time.Now().UTC()
			if rescueManifestID != "" {
				for idx, ao := range affectedOrders {
					seq := stopCount + int64(idx+1)
					mutations = append(mutations, spanner.InsertOrUpdateMap("ManifestOrders", map[string]any{
						"ManifestId":    rescueManifestID,
						"OrderId":       ao.OrderID,
						"SequenceIndex": seq,
						"LoadingOrder":  seq,
						"VolumeVU":      ao.VolumeVU,
						"State":         "LOADED",
						"UpdatedAt":     spanner.CommitTimestamp,
					}))
				}
				mutations = append(mutations, spanner.UpdateMap("SupplierTruckManifests", map[string]any{
					"ManifestId":    rescueManifestID,
					"TotalVolumeVU": currentLoadedVU + neededVU,
					"StopCount":     stopCount + int64(len(affectedOrders)),
					"UpdatedAt":     spanner.CommitTimestamp,
				}))
			}

			// Mark broken driver truck as BROKEN_DOWN
			truckStmt := spanner.Statement{
				SQL:    `UPDATE Drivers SET UnavailableReason = 'BROKEN_DOWN', IsActive = false, UpdatedAt = PENDING_COMMIT_TIMESTAMP() WHERE DriverId = @oldDriverId`,
				Params: map[string]any{"oldDriverId": req.BrokenDriverID},
			}
			if _, err := txn.Update(ctx, truckStmt); err != nil {
				return err
			}

			// Mark broken driver's manifest as RESCUED
			brokenManifestStmt := spanner.Statement{
				SQL: `UPDATE SupplierTruckManifests SET State = 'RESCUED', UpdatedAt = PENDING_COMMIT_TIMESTAMP()
				      WHERE DriverId = @brokenDriverId AND State IN ('DISPATCHED', 'IN_TRANSIT', 'SEALED')`,
				Params: map[string]any{"brokenDriverId": req.BrokenDriverID},
			}
			_, _ = txn.Update(ctx, brokenManifestStmt)

			// Step 6: Emit EventOrderReassigned for each affected order
			for _, ao := range affectedOrders {
				evReassigned := events.OrderEvent{
					BaseEvent: events.BaseEvent{
						Type:      events.EventOrderReassigned,
						Timestamp: now.Format(time.RFC3339Nano),
						Version:   1,
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
					"SupplierId":    supplierID,
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
			"SupplierId":    supplierID,
		}))

		return txn.BufferWrite(mutations)
	})

	if txnErr != nil {
		switch {
		case errors.Is(txnErr, errRescueAlreadyAccepted):
			writeJSON(w, http.StatusConflict, map[string]string{
				"error":   "rescue_already_accepted",
				"message": "Rescue has already been accepted or resolved by another driver",
			})
			return
		case errors.Is(txnErr, errInsufficientCapacity):
			writeJSON(w, http.StatusConflict, map[string]string{
				"error":   "insufficient_vehicle_capacity",
				"message": "Your vehicle does not have enough remaining capacity for this rescue",
			})
			return
		case errors.Is(txnErr, errBrokenDriverNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "broken_driver_not_found"})
			return
		default:
			s.log.Error("failed to respond to rescue", "error", txnErr)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	}

	resp := map[string]any{
		"status":            "recorded",
		"rescue_id":         req.RescueID,
		"broken_driver_id":  req.BrokenDriverID,
		"rescue_driver_id":  driverID,
		"accepted":          req.Accept,
		"orders_reassigned": affectedCount,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}
