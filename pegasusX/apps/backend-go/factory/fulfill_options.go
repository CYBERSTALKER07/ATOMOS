package factory

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	"google.golang.org/api/iterator"
)

// SupplyFulfillOptions describes INTERNAL vs TRUCK outcomes before fulfilling a request.
type SupplyFulfillOptions struct {
	TransferMode    string  `json:"transfer_mode"`
	WarehouseID     string  `json:"warehouse_id"`
	WarehouseName   string  `json:"warehouse_name"`
	CoLocated       bool    `json:"co_located"`
	OutcomeInternal string  `json:"outcome_internal"`
	OutcomeTruck    string  `json:"outcome_truck"`
	LinkedDriverETA *string `json:"linked_driver_eta,omitempty"`
}

// HandleSupplyRequestFulfillOptions serves GET /v1/factory/supply-requests/{id}/fulfill-options.
func (s *Service) HandleSupplyRequestFulfillOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	requestID := strings.TrimSpace(chi.URLParam(r, "id"))
	if requestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing_request_id"})
		return
	}

	opts, err := s.buildSupplyFulfillOptions(r.Context(), requestID)
	if err != nil {
		if strings.Contains(err.Error(), "not_found") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "request_not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fulfill_options_failed"})
		return
	}
	writeJSON(w, http.StatusOK, opts)
}

func (s *Service) buildSupplyFulfillOptions(ctx context.Context, requestID string) (SupplyFulfillOptions, error) {
	if s.spannerClient == nil {
		return s.buildSupplyFulfillOptionsMemory(requestID)
	}

	stmt := spanner.Statement{
		SQL: `SELECT sr.WarehouseId, COALESCE(w.Name, ''), COALESCE(sr.TransferMode, w.TransferMode, 'TRUCK'),
		             COALESCE(w.CoLocateWithFactoryId, ''), COALESCE(w.PrimaryFactoryId, ''),
		             COALESCE(sr.LinkedTransferId, '')
		      FROM WarehouseSupplyRequests sr
		      INNER JOIN Warehouses w ON sr.WarehouseId = w.WarehouseId
		      WHERE sr.RequestId = @requestId`,
		Params: map[string]any{"requestId": requestID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return SupplyFulfillOptions{}, fmt.Errorf("request_not_found")
	}
	if err != nil {
		return SupplyFulfillOptions{}, err
	}

	var warehouseID, warehouseName, transferMode, coLocateFactoryID, primaryFactoryID, linkedTransferID string
	if err := row.Columns(&warehouseID, &warehouseName, &transferMode, &coLocateFactoryID, &primaryFactoryID, &linkedTransferID); err != nil {
		return SupplyFulfillOptions{}, err
	}

	mode := supplier.NormalizeTransferMode(transferMode)
	factoryID := strings.TrimSpace(s.factoryNodeID)
	coLocated := coLocateFactoryID == factoryID || primaryFactoryID == factoryID

	opts := SupplyFulfillOptions{
		TransferMode: mode,
		WarehouseID:  warehouseID,
		WarehouseName: strings.TrimSpace(warehouseName),
		CoLocated:    coLocated,
		OutcomeInternal: fmt.Sprintf("Creates RECEIVED transfer — stock available at %s without driver leg", strings.TrimSpace(warehouseName)),
		OutcomeTruck:    "Creates IN_TRANSIT transfer — assign driver on supply leg",
	}

	if eta := s.lookupLinkedDriverETA(ctx, linkedTransferID); eta != "" {
		opts.LinkedDriverETA = &eta
	}
	return opts, nil
}

func (s *Service) buildSupplyFulfillOptionsMemory(requestID string) (SupplyFulfillOptions, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureDemoDataLocked()
	for _, req := range s.supplyRequests {
		if req.RequestID != requestID {
			continue
		}
		mode := supplier.TransferModeTruck
		name := strings.TrimSpace(req.WarehouseID)
		return SupplyFulfillOptions{
			TransferMode:    mode,
			WarehouseID:     req.WarehouseID,
			WarehouseName:   name,
			CoLocated:       false,
			OutcomeInternal: fmt.Sprintf("Creates RECEIVED transfer — stock available at %s without driver leg", name),
			OutcomeTruck:    "Creates IN_TRANSIT transfer — assign driver on supply leg",
		}, nil
	}
	return SupplyFulfillOptions{}, fmt.Errorf("request_not_found")
}

func (s *Service) lookupLinkedDriverETA(ctx context.Context, transferID string) string {
	transferID = strings.TrimSpace(transferID)
	if transferID == "" || s.spannerClient == nil {
		return ""
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "FactoryInternalTransfers", spanner.Key{transferID},
		[]string{"DriverId", "State", "UpdatedAt"})
	if err != nil {
		return ""
	}
	var driverID, state string
	var updatedAt time.Time
	if err := row.Columns(&driverID, &state, &updatedAt); err != nil {
		return ""
	}
	if strings.TrimSpace(driverID) == "" || strings.ToUpper(state) == "RECEIVED" {
		return ""
	}
	// Only surface ETA when we have a live driver assignment on an active transfer.
	return updatedAt.UTC().Format(time.RFC3339)
}
