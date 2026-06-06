package warehouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/grpc/codes"
)

var (
	errTransferNotFound  = errors.New("transfer_not_found")
	errTransferForbidden = errors.New("transfer_forbidden")
	errInvalidTransfer   = errors.New("invalid_transfer_state")
)

var receiveableTransferStates = map[string]struct{}{
	"IN_TRANSIT":            {},
	"IN_TRANSIT_TO_WAREHOUSE": {},
	"DISPATCHED":            {},
	"ARRIVED":               {},
	"ASSIGNED":              {},
}

// HandleEmergencyTransfer serves POST /v1/warehouse/transfers/emergency.
func (s *Service) HandleEmergencyTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	whID, err := s.effectiveWarehouseID(ctx, r)
	if err != nil || whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	var req struct {
		TotalVolumeVU float64 `json:"total_volume_vu"`
		Notes         string  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if req.TotalVolumeVU <= 0 {
		req.TotalVolumeVU = 1
	}

	factoryID, supplierID, err := s.resolveWarehouseFactory(ctx, whID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "warehouse_not_found"})
		return
	}

	if s.memoryTransfersEnabled() {
		row := s.memoryCreateEmergencyTransfer(whID, factoryID, supplierID, req.TotalVolumeVU, req.Notes)
		writeJSON(w, http.StatusCreated, map[string]any{
			"transfer_id": row.TransferID,
			"state":       row.State,
			"notes":       row.Notes,
		})
		return
	}

	transferID := uuid.NewString()
	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdateMap("FactoryInternalTransfers", map[string]any{
				"TransferId":    transferID,
				"FactoryId":     factoryID,
				"SupplierId":    supplierID,
				"State":         "APPROVED",
				"TotalVolumeVU": req.TotalVolumeVU,
				"CreatedAt":     spanner.CommitTimestamp,
				"UpdatedAt":     spanner.CommitTimestamp,
			}),
		})
	})
	if err != nil {
		slog.ErrorContext(ctx, "emergency transfer create failed", "warehouse_id", whID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"transfer_id": transferID,
		"state":       "APPROVED",
		"notes":       strings.TrimSpace(req.Notes),
	})
}

// HandleReceiveTransfer serves POST /v1/warehouse/transfers/{id}/receive.
func (s *Service) HandleReceiveTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	transferID := strings.TrimSpace(chi.URLParam(r, "id"))
	if transferID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "transfer_id_required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	ops, err := s.resolveWarehouseOps(ctx, r)
	if err != nil || ops == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}

	if err := s.receiveTransfer(ctx, ops, transferID); err != nil {
		mapTransferError(w, r, transferID, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"transfer_id": transferID, "state": "RECEIVED"})
}

// HandleForceReceive serves POST /v1/warehouse/transfers/force-receive.
func (s *Service) HandleForceReceive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	whID, err := s.effectiveWarehouseID(ctx, r)
	if err != nil || whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	var req struct {
		FactoryID     string  `json:"factory_id"`
		TotalVolumeVU float64 `json:"total_volume_vu"`
		Notes         string  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if req.TotalVolumeVU <= 0 {
		req.TotalVolumeVU = 1
	}

	// Factory + supplier are always derived from the JWT-scoped warehouse, never
	// trusted from the body. A body factory_id is honored only when it matches the
	// warehouse's own primary factory; any mismatch is a scope-spoofing attempt.
	factoryID, supplierID, err := s.resolveWarehouseFactory(ctx, whID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "warehouse_not_found"})
		return
	}
	if bodyFactory := strings.TrimSpace(req.FactoryID); bodyFactory != "" && bodyFactory != factoryID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "factory_scope_violation"})
		return
	}

	if s.memoryTransfersEnabled() {
		row := s.memoryForceReceiveTransfer(factoryID, supplierID, req.TotalVolumeVU, req.Notes)
		writeJSON(w, http.StatusCreated, map[string]any{
			"transfer_id": row.TransferID,
			"state":       row.State,
			"notes":       row.Notes,
		})
		return
	}

	transferID := uuid.NewString()
	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdateMap("FactoryInternalTransfers", map[string]any{
				"TransferId":    transferID,
				"FactoryId":     factoryID,
				"SupplierId":    supplierID,
				"State":         "RECEIVED",
				"TotalVolumeVU": req.TotalVolumeVU,
				"CreatedAt":     spanner.CommitTimestamp,
				"UpdatedAt":     spanner.CommitTimestamp,
			}),
		})
	})
	if err != nil {
		slog.ErrorContext(ctx, "force receive failed", "warehouse_id", whID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"transfer_id": transferID,
		"state":       "RECEIVED",
		"notes":       strings.TrimSpace(req.Notes),
	})
}

func (s *Service) receiveTransfer(ctx context.Context, ops *auth.WarehouseOps, transferID string) error {
	if s.memoryTransfersEnabled() {
		s.mu.Lock()
		s.ensureMemoryDemoReceiveTransferLocked()
		s.mu.Unlock()
		return s.memoryReceiveTransfer(ops, transferID)
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "FactoryInternalTransfers", spanner.Key{transferID},
			[]string{"TransferId", "SupplierId", "State"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return errTransferNotFound
			}
			return fmt.Errorf("read transfer: %w", err)
		}
		var id, supplierID, state string
		if err := row.Columns(&id, &supplierID, &state); err != nil {
			return fmt.Errorf("decode transfer: %w", err)
		}
		if ops.SupplierID != "" && supplierID != ops.SupplierID {
			return errTransferForbidden
		}
		if _, ok := receiveableTransferStates[strings.ToUpper(state)]; !ok {
			return fmt.Errorf("%w: %s", errInvalidTransfer, state)
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("FactoryInternalTransfers", map[string]any{
				"TransferId": transferID,
				"State":      "RECEIVED",
				"UpdatedAt":  spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

func (s *Service) resolveWarehouseFactory(ctx context.Context, warehouseID string) (factoryID, supplierID string, err error) {
	if s.memoryTransfersEnabled() {
		return s.memoryResolveWarehouseFactory(ctx, warehouseID)
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "Warehouses", spanner.Key{warehouseID},
		[]string{"SupplierId", "PrimaryFactoryId"})
	if err != nil {
		return "", "", err
	}
	var primaryFactory spanner.NullString
	if err := row.Columns(&supplierID, &primaryFactory); err != nil {
		return "", "", err
	}
	if !primaryFactory.Valid || primaryFactory.StringVal == "" {
		return "", "", errors.New("primary factory missing")
	}
	return primaryFactory.StringVal, supplierID, nil
}

func mapTransferError(w http.ResponseWriter, r *http.Request, transferID string, err error) {
	switch {
	case errors.Is(err, errTransferNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "transfer_not_found", "transfer_id": transferID})
	case errors.Is(err, errTransferForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	case errors.Is(err, errInvalidTransfer):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	default:
		slog.ErrorContext(r.Context(), "transfer receive failed", "transfer_id", transferID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	}
}
