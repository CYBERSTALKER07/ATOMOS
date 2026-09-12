package warehouse

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

var (
	errTransferNotFound  = errors.New("transfer_not_found")
	errTransferForbidden = errors.New("transfer_forbidden")
	errInvalidTransfer   = errors.New("invalid_transfer_state")
)

var receiveableTransferStates = map[string]struct{}{
	"IN_TRANSIT":              {},
	"IN_TRANSIT_TO_WAREHOUSE": {},
	"DISPATCHED":              {},
	"ARRIVED":                 {},
	"ASSIGNED":                {},
}

// HandleEmergencyTransfer serves POST /v1/warehouse/transfers/emergency.
func (s *Service) HandleEmergencyTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	body, ok := readMutationBody(w, r, 64*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}

	whID, err := s.effectiveWarehouseID(ctx, r)
	if err != nil || whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	var req struct {
		TotalVolumeVU float64 `json:"total_volume_vu"`
		Notes         string  `json:"notes"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if req.TotalVolumeVU <= 0 {
		req.TotalVolumeVU = 1
	}

	factoryID, supplierID, err := s.resolveWarehouseFactory(ctx, whID)
	if err != nil {
		writeFactoryResolveError(w, err)
		return
	}

	if s.memoryTransfersEnabled() {
		row := s.memoryCreateEmergencyTransfer(whID, factoryID, supplierID, req.TotalVolumeVU, req.Notes)
		resp := map[string]any{
			"transfer_id": row.TransferID,
			"state":       row.State,
			"notes":       row.Notes,
		}
		respBytes, _ := json.Marshal(resp)
		s.storeMutationReplay(ctx, key, body, http.StatusCreated, respBytes)
		writeJSON(w, http.StatusCreated, resp)
		return
	}

	transferID := uuid.NewString()
	err = s.repo.CreateTransfer(ctx, transferID, factoryID, supplierID, whID, req.TotalVolumeVU, func(txn outbox.TxnBuffer) error {
		payload := events.WarehouseEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventWarehouseTransferCreated},
			TransferID:  transferID,
			WarehouseID: whID, // maps factory_id locally in context
			SupplierID:  supplierID,
			Units:       int64(req.TotalVolumeVU),
		}
		return outbox.EmitJSON(ctx, txn, events.AggregateWarehouse, whID, events.TopicMain, payload)
	})
	if err != nil {
		slog.ErrorContext(ctx, "emergency transfer create failed", "warehouse_id", whID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	resp := map[string]any{
		"transfer_id": transferID,
		"state":       "APPROVED",
		"notes":       strings.TrimSpace(req.Notes),
	}
	respBytes, _ := json.Marshal(resp)
	s.storeMutationReplay(ctx, key, body, http.StatusCreated, respBytes)
	writeJSON(w, http.StatusCreated, resp)
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

	body, ok := readMutationBody(w, r, 64*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}

	ops, err := s.resolveWarehouseOps(ctx, r)
	if err != nil || ops == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}

	if err := s.receiveTransfer(ctx, ops, transferID, parseReceiveItems(body)); err != nil {
		mapTransferError(w, r, transferID, err)
		return
	}
	resp := map[string]string{"transfer_id": transferID, "state": "RECEIVED"}
	respBytes, _ := json.Marshal(resp)
	s.storeMutationReplay(ctx, key, body, http.StatusOK, respBytes)
	writeJSON(w, http.StatusOK, resp)
}

// HandleForceReceive serves POST /v1/warehouse/transfers/force-receive.
func (s *Service) HandleForceReceive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	body, ok := readMutationBody(w, r, 64*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}

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
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if req.TotalVolumeVU <= 0 {
		req.TotalVolumeVU = 1
	}

	// Factory + supplier are always derived from the JWT-scoped warehouse, never
	// trusted from the body. A body factory_id is honored only when it matches the
	// engine-resolved factory; any mismatch is a scope-spoofing attempt.
	factoryID, supplierID, err := s.resolveWarehouseFactory(ctx, whID)
	if err != nil {
		writeFactoryResolveError(w, err)
		return
	}
	if bodyFactory := strings.TrimSpace(req.FactoryID); bodyFactory != "" && bodyFactory != factoryID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "factory_scope_violation"})
		return
	}

	if s.memoryTransfersEnabled() {
		row := s.memoryForceReceiveTransfer(factoryID, supplierID, req.TotalVolumeVU, req.Notes)
		resp := map[string]any{
			"transfer_id": row.TransferID,
			"state":       row.State,
			"notes":       row.Notes,
		}
		respBytes, _ := json.Marshal(resp)
		s.storeMutationReplay(ctx, key, body, http.StatusCreated, respBytes)
		writeJSON(w, http.StatusCreated, resp)
		return
	}

	transferID := uuid.NewString()
	err = s.repo.CreateTransfer(ctx, transferID, factoryID, supplierID, whID, req.TotalVolumeVU, nil)
	if err == nil {
		err = s.repo.UpdateTransferState(ctx, transferID, supplierID, "RECEIVED", func(txn outbox.TxnBuffer) error {
			payload := events.WarehouseEvent{
				BaseEvent:   events.BaseEvent{Type: events.EventWarehouseTransferReceived},
				TransferID:  transferID,
				WarehouseID: whID, // replacing factory_id for standardization
				SupplierID:  supplierID,
				Units:       int64(req.TotalVolumeVU),
			}
			return outbox.EmitJSON(ctx, txn, events.AggregateWarehouse, whID, events.TopicMain, payload)
		})
	}
	if err != nil {
		slog.ErrorContext(ctx, "force receive failed", "warehouse_id", whID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	resp := map[string]any{
		"transfer_id": transferID,
		"state":       "RECEIVED",
		"notes":       strings.TrimSpace(req.Notes),
	}
	respBytes, _ := json.Marshal(resp)
	s.storeMutationReplay(ctx, key, body, http.StatusCreated, respBytes)
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Service) receiveTransfer(ctx context.Context, ops *auth.WarehouseOps, transferID string, lines []receiveLineInput) error {
	if s.memoryTransfersEnabled() {
		s.mu.Lock()
		s.ensureMemoryDemoReceiveTransferLocked()
		s.mu.Unlock()
		return s.memoryReceiveTransfer(ops, transferID)
	}
	if len(lines) > 0 && s.spannerClient != nil {
		return s.receiveTransferWithItems(ctx, ops, transferID, lines)
	}
	err := s.repo.UpdateTransferState(ctx, transferID, ops.SupplierID, "RECEIVED", func(txn outbox.TxnBuffer) error {
		payload := events.WarehouseEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventWarehouseTransferReceived},
			TransferID: transferID,
			SupplierID: ops.SupplierID,
		}
		// assuming ops is warehouse scope, we might not have warehouse_id here, but we can emit for aggregate
		return outbox.EmitJSON(ctx, txn, events.AggregateWarehouse, transferID, events.TopicMain, payload)
	})
	return err
}

func (s *Service) resolveWarehouseFactory(ctx context.Context, warehouseID string) (factoryID, supplierID string, err error) {
	if s.memoryTransfersEnabled() {
		return s.memoryResolveWarehouseFactory(ctx, warehouseID)
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "Warehouses", spanner.Key{warehouseID},
		[]string{"SupplierId", "PrimaryFactoryId", "CountryCode", "Lat", "Lng"})
	if err != nil {
		return "", "", err
	}
	var primaryFactory, country spanner.NullString
	var lat, lng spanner.NullFloat64
	if err := row.Columns(&supplierID, &primaryFactory, &country, &lat, &lng); err != nil {
		return "", "", err
	}
	factoryID, err = s.engineFactoryID(ctx, supplierID, warehouseID, country.StringVal, lat.Float64, lng.Float64, primaryFactory.StringVal)
	if err != nil {
		return "", "", err
	}
	return factoryID, supplierID, nil
}

func writeFactoryResolveError(w http.ResponseWriter, err error) {
	if errors.Is(err, proximity.ErrFactoryUnassigned) ||
		errors.Is(err, auth.ErrGeographyIncomplete) ||
		errors.Is(err, auth.ErrCrossMarketDeferred) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "warehouse_not_found"})
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
