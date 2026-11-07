package warehouse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// SupplyTransferDriverView is a factory-driver supply leg row.
type SupplyTransferDriverView struct {
	TransferID      string  `json:"transfer_id"`
	WarehouseID     string  `json:"warehouse_id"`
	SupplyRequestID string  `json:"supply_request_id,omitempty"`
	State           string  `json:"state"`
	TotalVolumeVU   float64 `json:"total_volume_vu"`
}

var factoryDriverArriveStates = map[string]struct{}{
	"IN_TRANSIT":              {},
	"IN_TRANSIT_TO_WAREHOUSE": {},
	"DISPATCHED":              {},
}

// ListSupplyTransfersForDriver returns assigned factory supply transfers.
func (s *Service) ListSupplyTransfersForDriver(ctx context.Context, driverID string) ([]SupplyTransferDriverView, error) {
	if s.spannerClient == nil || strings.TrimSpace(driverID) == "" {
		return []SupplyTransferDriverView{}, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT TransferId, WarehouseId, SupplyRequestId, State, TotalVolumeVU
		      FROM FactoryInternalTransfers
		      WHERE DriverId = @did AND State IN ('IN_TRANSIT','IN_TRANSIT_TO_WAREHOUSE','DISPATCHED','ARRIVED')
		      ORDER BY UpdatedAt DESC
		      LIMIT 50`,
		Params: map[string]any{"did": strings.TrimSpace(driverID)},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]SupplyTransferDriverView, 0, 8)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var view SupplyTransferDriverView
		var supplyID spanner.NullString
		if err := row.Columns(&view.TransferID, &view.WarehouseID, &supplyID, &view.State, &view.TotalVolumeVU); err != nil {
			continue
		}
		if supplyID.Valid {
			view.SupplyRequestID = supplyID.StringVal
		}
		out = append(out, view)
	}
	return out, nil
}

// ArriveSupplyTransfer marks IN_TRANSIT supply transfer ARRIVED with optional GPS proof.
func (s *Service) ArriveSupplyTransfer(ctx context.Context, driverID, transferID string, lat, lng float64) error {
	if s.spannerClient == nil {
		return errTransferNotFound
	}
	driverID = strings.TrimSpace(driverID)
	transferID = strings.TrimSpace(transferID)
	if driverID == "" || transferID == "" {
		return fmt.Errorf("driver_id and transfer_id required")
	}
	var warehouseID, supplierID string
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "FactoryInternalTransfers", spanner.Key{transferID},
			[]string{"DriverId", "State", "WarehouseId", "SupplierId"})
		if err != nil {
			return errTransferNotFound
		}
		var assignedDriver, state string
		var warehouseCol spanner.NullString
		if err := row.Columns(&assignedDriver, &state, &warehouseCol, &supplierID); err != nil {
			return err
		}
		if strings.TrimSpace(assignedDriver) != driverID {
			return errTransferForbidden
		}
		state = strings.ToUpper(strings.TrimSpace(state))
		if _, ok := factoryDriverArriveStates[state]; !ok {
			return fmt.Errorf("%w: state %s", errInvalidTransfer, state)
		}
		if warehouseCol.Valid {
			warehouseID = strings.TrimSpace(warehouseCol.StringVal)
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("FactoryInternalTransfers", map[string]any{
			"TransferId": transferID,
			"State":      "ARRIVED",
			"UpdatedAt":  spanner.CommitTimestamp,
		})}); err != nil {
			return err
		}
		// B2 M-P0-15: durable outbox for supply-transfer arrive (was silent Spanner write).
		buf := outbox.NewSpannerTxnBuffer(txn)
		agg := transferID
		if warehouseID != "" {
			agg = warehouseID
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, agg, events.TopicMain, map[string]any{
			"type":         events.EventSupplyTransferArrived,
			"transfer_id":  transferID,
			"driver_id":    driverID,
			"warehouse_id": warehouseID,
			"supplier_id":  supplierID,
			"latitude":     lat,
			"longitude":    lng,
			"state":        "ARRIVED",
			"timestamp":    time.Now().UTC().Format(time.RFC3339Nano),
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
	if err != nil {
		return err
	}
	if s.warehouseHub != nil && warehouseID != "" {
		payload, _ := json.Marshal(map[string]any{
			"type":         events.EventSupplyTransferArrived,
			"event_type":   events.EventSupplyTransferArrived,
			"transfer_id":  transferID,
			"driver_id":    driverID,
			"warehouse_id": warehouseID,
			"latitude":     lat,
			"longitude":    lng,
		})
		s.warehouseHub.Broadcast(ctx, "warehouse:"+warehouseID, payload)
	}
	return nil
}

// HandleDriverListSupplyTransfers serves GET /v1/driver/supply-transfers.
func (s *Service) HandleDriverListSupplyTransfers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Subject == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.HomeNodeType != auth.HomeNodeFactory {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "factory_driver_scope_required"})
		return
	}
	rows, err := s.ListSupplyTransfersForDriver(r.Context(), claims.Subject)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transfers": rows})
}

// HandleDriverArriveSupplyTransfer serves POST /v1/driver/supply-transfers/{id}/arrive.
func (s *Service) HandleDriverArriveSupplyTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Subject == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.HomeNodeType != auth.HomeNodeFactory {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "factory_driver_scope_required"})
		return
	}
	transferID := strings.TrimSpace(chi.URLParam(r, "id"))
	var req struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if err := s.ArriveSupplyTransfer(r.Context(), claims.Subject, transferID, req.Latitude, req.Longitude); err != nil {
		mapTransferError(w, r, transferID, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"transfer_id": transferID,
		"state":       "ARRIVED",
		"event_type":  events.EventSupplyTransferApproaching,
	})
}
