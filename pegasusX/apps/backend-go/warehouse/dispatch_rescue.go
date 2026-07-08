package warehouse

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

type RescuePreviewRequest struct {
	BrokenDriverID string `json:"broken_driver_id"`
}

func (s *Service) HandleOpsDispatchRescuePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id_required"})
		return
	}
	var req RescuePreviewRequest
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err := json.Unmarshal(body, &req); err != nil || req.BrokenDriverID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	drivers, err := s.loadDispatchDrivers(r.Context(), whID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_load_drivers"})
		return
	}

	sid := s.resolveDispatchSupplierID(r.Context(), whID)
	fleetCtx, err := s.loadFleetDispatchContext(r.Context(), sid, whID, collectWarehouseDriverIDs(drivers))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_load_fleet_context"})
		return
	}

	var brokenDriver *PortalDriver
	for i := range drivers {
		if drivers[i].DriverID == req.BrokenDriverID {
			brokenDriver = &drivers[i]
			break
		}
	}
	if brokenDriver == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "broken_driver_not_found"})
		return
	}

	var pendingVU float64
	stmt := spanner.Statement{
		SQL: `SELECT SUM(VolumeVU) FROM Orders WHERE DriverId = @did AND Status IN ('LOADED', 'IN_TRANSIT', 'ARRIVED', 'DISPATCHED', 'ARRIVING', 'EN_ROUTE', 'AWAITING_PAYMENT', 'PENDING_CASH_COLLECTION')`,
		Params: map[string]any{"did": brokenDriver.DriverID},
	}
	iter := s.spannerClient.Single().Query(r.Context(), stmt)
	row, err := iter.Next()
	iter.Stop()
	if err == nil {
		var sum spanner.NullFloat64
		if err := row.Columns(&sum); err == nil && sum.Valid {
			pendingVU = sum.Float64
		}
	}

	type RescueOption struct {
		DriverID          string  `json:"driver_id"`
		Name              string  `json:"name"`
		LicensePlate      string  `json:"license_plate"`
		TruckStatus       string  `json:"truck_status"`
		EffectiveCapacity float64 `json:"effective_capacity_vu"`
		IsCapacityExceeded bool   `json:"is_capacity_exceeded"`
	}

	var options []RescueOption
	for _, d := range drivers {
		if d.DriverID == brokenDriver.DriverID {
			continue
		}
		if !d.IsActive || !d.OnShift {
			continue
		}
		
		status, isUnavailable, _ := warehouseDriverAvailability(d, fleetCtx)
		if isUnavailable {
			continue
		}

		effCap := d.MaxVolumeVU
		top := fleetCtx.topOffFor(d.DriverID)
		if top != nil {
			if top.MaxVolumeVU > 0 {
				effCap = top.MaxVolumeVU
			}
			loadedRemaining := top.TotalVolumeVU // approximate
			effCap -= loadedRemaining
		}

		exceeded := effCap < pendingVU

		options = append(options, RescueOption{
			DriverID:           d.DriverID,
			Name:               d.Name,
			LicensePlate:       d.LicensePlate,
			TruckStatus:        status,
			EffectiveCapacity:  effCap,
			IsCapacityExceeded: exceeded,
		})
	}

	sort.Slice(options, func(i, j int) bool {
		if !options[i].IsCapacityExceeded && options[j].IsCapacityExceeded {
			return true
		}
		if options[i].IsCapacityExceeded && !options[j].IsCapacityExceeded {
			return false
		}
		return options[i].EffectiveCapacity > options[j].EffectiveCapacity
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"broken_driver_id": req.BrokenDriverID,
		"pending_volume_vu": pendingVU,
		"rescue_options": options,
	})
}

type RescueProposeRequest struct {
	RescueID       string `json:"rescue_id"`
	BrokenDriverID string `json:"broken_driver_id"`
	RescueDriverID string `json:"rescue_driver_id"`
	ForceCapacity  bool   `json:"force_capacity"`
}

func (s *Service) HandleOpsDispatchRescuePropose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id_required"})
		return
	}
	var req RescueProposeRequest
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err := json.Unmarshal(body, &req); err != nil || req.BrokenDriverID == "" || req.RescueDriverID == "" || req.RescueID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	
	drivers, err := s.loadDispatchDrivers(r.Context(), whID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_load_drivers"})
		return
	}
	sid := s.resolveDispatchSupplierID(r.Context(), whID)
	_, err = s.loadFleetDispatchContext(r.Context(), sid, whID, collectWarehouseDriverIDs(drivers))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_load_fleet_context"})
		return
	}

	var brokenDriver, rescueDriver *PortalDriver
	for i := range drivers {
		if drivers[i].DriverID == req.BrokenDriverID {
			brokenDriver = &drivers[i]
		}
		if drivers[i].DriverID == req.RescueDriverID {
			rescueDriver = &drivers[i]
		}
	}
	if brokenDriver == nil || rescueDriver == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "driver_not_found"})
		return
	}

	// Instead of mutating the database immediately, we emit a PROPOSED event for the rescue driver
	_, err = s.spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		ev := events.RescueEvent{
			RescueID:       req.RescueID,
			BrokenDriverID: req.BrokenDriverID,
			RescueDriverID: req.RescueDriverID,
			Status:         "PROPOSED",
			WarehouseID:    whID,
			SupplierID:     sid,
		}
		ev.Type = "RESCUE_PROPOSED"
		evJSON, _ := json.Marshal(ev)

		m := spanner.InsertMap("OutboxEvents", map[string]any{
			"EventId":       uuid.NewString(),
			"AggregateType": events.AggregateWarehouse,
			"AggregateId":   whID,
			"TopicName":     events.TopicMain,
			"Payload":       string(evJSON),
			"CreatedAt":     spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})

	if err != nil {
		s.log.ErrorContext(r.Context(), "rescue propose failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "rescue_proposal_failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "proposed",
	})
}
