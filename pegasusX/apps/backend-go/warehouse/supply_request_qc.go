package warehouse

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

type supplyRequestQCResponse struct {
	RequestID   string `json:"request_id"`
	Result      string `json:"result"`
	Notes       string `json:"notes,omitempty"`
	InspectedBy string `json:"inspected_by,omitempty"`
	InspectedAt string `json:"inspected_at,omitempty"`
}

// HandleSupplyRequestQC serves GET/POST /v1/warehouse/supply-requests/{id}/qc.
func (s *Service) HandleSupplyRequestQC(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.readSupplyRequestQC(w, r)
	case http.MethodPost:
		s.upsertSupplyRequestQC(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) readSupplyRequestQC(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "qc_unavailable"})
		return
	}
	requestID := strings.TrimSpace(chi.URLParam(r, "id"))
	if requestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request_id_required"})
		return
	}
	warehouseID := warehouseIDFromRequest(r)
	if warehouseID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id_required"})
		return
	}
	found, err := s.warehouseOwnsSupplyRequest(r.Context(), requestID, warehouseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "qc_read_failed"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "request_not_found"})
		return
	}
	row, ok, err := s.readFactorySupplyQC(r.Context(), requestID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "qc_read_failed"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusOK, supplyRequestQCResponse{RequestID: requestID, Result: ""})
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Service) upsertSupplyRequestQC(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "qc_unavailable"})
		return
	}
	requestID := strings.TrimSpace(chi.URLParam(r, "id"))
	if requestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request_id_required"})
		return
	}
	warehouseID := warehouseIDFromRequest(r)
	if warehouseID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id_required"})
		return
	}
	body, ok := readMutationBody(w, r, 8*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseMutationReplay(r.Context(), key)
		}
	}()
	var req struct {
		Result string `json:"result"`
		Notes  string `json:"notes"`
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	}
	result := strings.ToUpper(strings.TrimSpace(req.Result))
	if result != "PASS" && result != "FAIL" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "result_must_be_pass_or_fail"})
		return
	}
	found, err := s.warehouseOwnsSupplyRequest(r.Context(), requestID, warehouseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "qc_write_failed"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "request_not_found"})
		return
	}
	inspector := strings.TrimSpace(auth.ActorFromContext(r.Context()))
	if inspector == "" || inspector == "unknown" {
		if claims, ok := auth.FromContext(r.Context()); ok {
			inspector = strings.TrimSpace(claims.Subject)
		}
	}
	notes := strings.TrimSpace(req.Notes)
	nowTS := s.now().UTC().Format(time.RFC3339Nano)
	supplierID := strings.TrimSpace(s.analyticsSupplierID(r.Context()))
	if supplierID == "" {
		if claims, ok := auth.FromContext(r.Context()); ok {
			supplierID = strings.TrimSpace(claims.SupplierID)
		}
	}
	_, err = s.spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if _, err := txn.ReadRow(ctx, "WarehouseSupplyRequests", spanner.Key{requestID}, []string{"RequestId", "State"}); err != nil {
			return err
		}
		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, requestID, events.TopicMain, map[string]any{
			"type":         events.EventFactorySupplyRequestUpdate,
			"request_id":   requestID,
			"warehouse_id": warehouseID,
			"supplier_id":  supplierID,
			"qc_result":    result,
			"qc_notes":     notes,
			"timestamp":    nowTS,
		}); err != nil {
			return err
		}
		muts := []*spanner.Mutation{spanner.InsertOrUpdateMap("FactorySupplyRequestQC", map[string]any{
			"RequestId":   requestID,
			"Result":      result,
			"Notes":       notes,
			"InspectedBy": inspector,
			"InspectedAt": spanner.CommitTimestamp,
		})}
		muts = append(muts, outboxMutations(buf.events)...)
		return txn.BufferWrite(muts)
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "request_not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "qc_write_failed"})
		return
	}
	resp := supplyRequestQCResponse{
		RequestID:   requestID,
		Result:      result,
		Notes:       notes,
		InspectedBy: inspector,
		InspectedAt: nowTS,
	}
	respBytes, _ := json.Marshal(resp)
	s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSON(w, http.StatusOK, resp)
}

func (s *Service) warehouseOwnsSupplyRequest(ctx context.Context, requestID, warehouseID string) (bool, error) {
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT RequestId FROM WarehouseSupplyRequests
		      WHERE RequestId = @rid AND WarehouseId = @wid LIMIT 1`,
		Params: map[string]any{"rid": requestID, "wid": warehouseID},
	})
	defer iter.Stop()
	_, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) readFactorySupplyQC(ctx context.Context, requestID string) (supplyRequestQCResponse, bool, error) {
	row, err := s.spannerClient.Single().ReadRow(ctx, "FactorySupplyRequestQC", spanner.Key{requestID},
		[]string{"RequestId", "Result", "Notes", "InspectedBy", "InspectedAt"})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return supplyRequestQCResponse{}, false, nil
		}
		return supplyRequestQCResponse{}, false, err
	}
	var resp supplyRequestQCResponse
	var notes, inspectedBy spanner.NullString
	var inspectedAt spanner.NullTime
	if err := row.Columns(&resp.RequestID, &resp.Result, &notes, &inspectedBy, &inspectedAt); err != nil {
		return supplyRequestQCResponse{}, false, err
	}
	if notes.Valid {
		resp.Notes = notes.StringVal
	}
	if inspectedBy.Valid {
		resp.InspectedBy = inspectedBy.StringVal
	}
	if inspectedAt.Valid {
		resp.InspectedAt = inspectedAt.Time.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return resp, true, nil
}
