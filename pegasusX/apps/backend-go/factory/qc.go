package factory

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

var errQCRequestNotFound = errors.New("request_not_found")
var errQCPassRequired = errors.New("qc_pass_required")

type SupplyRequestQCResponse struct {
	RequestID   string `json:"request_id"`
	Result      string `json:"result"`
	Notes       string `json:"notes,omitempty"`
	InspectedBy string `json:"inspected_by,omitempty"`
	InspectedAt string `json:"inspected_at,omitempty"`
}

type qcRequestMeta struct {
	RequestID   string
	WarehouseID string
	SupplierID  string
	FactoryID   string
	State       string
}

type qcUpsert struct {
	RequestID   string
	Result      string
	Notes       string
	InspectedBy string
}

type supplyQCRepo interface {
	GetVisibleRequest(ctx context.Context, requestID, supplierID, factoryID string) (qcRequestMeta, error)
	GetQC(ctx context.Context, requestID string) (SupplyRequestQCResponse, bool, error)
	UpsertQC(ctx context.Context, row qcUpsert, emit func(outbox.TxnBuffer) error) error
}

func validQCResult(result string) bool {
	switch strings.ToUpper(strings.TrimSpace(result)) {
	case "PASS", "FAIL":
		return true
	default:
		return false
	}
}

func emptyQCResponse(requestID string) SupplyRequestQCResponse {
	return SupplyRequestQCResponse{RequestID: requestID, Result: ""}
}

func writeQCLookupError(w http.ResponseWriter, err error) {
	if errors.Is(err, errQCRequestNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "request_not_found"})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "qc_read_failed"})
}

func (s *Service) qcBackend() supplyQCRepo {
	if s != nil && s.qcRepo != nil {
		return s.qcRepo
	}
	if s != nil && s.spannerClient != nil {
		return spannerQCRepo{client: s.spannerClient, now: s.now}
	}
	return nil
}

// HandleSupplyRequestQC serves GET/POST /v1/factory/supply-requests/{id}/qc.
func (s *Service) HandleSupplyRequestQC(w http.ResponseWriter, r *http.Request) {
	requestID := strings.TrimSpace(chi.URLParam(r, "id"))
	if requestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request_id_required"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.readSupplyRequestQC(w, r, requestID)
	case http.MethodPost:
		s.upsertSupplyRequestQC(w, r, requestID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) readSupplyRequestQC(w http.ResponseWriter, r *http.Request, requestID string) {
	backend := s.qcBackend()
	if backend == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "qc_unavailable"})
		return
	}
	meta, err := backend.GetVisibleRequest(r.Context(), requestID, s.resolveSupplierScope(r.Context()), strings.TrimSpace(s.factoryNodeID))
	if err != nil {
		writeQCLookupError(w, err)
		return
	}
	row, ok, err := backend.GetQC(r.Context(), meta.RequestID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "qc_read_failed"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusOK, emptyQCResponse(requestID))
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Service) upsertSupplyRequestQC(w http.ResponseWriter, r *http.Request, requestID string) {
	body, err := readLimitedBody(r, 8*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
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
	if !validQCResult(result) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "result_must_be_pass_or_fail"})
		return
	}
	backend := s.qcBackend()
	if backend == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "qc_unavailable"})
		return
	}
	meta, err := backend.GetVisibleRequest(r.Context(), requestID, s.resolveSupplierScope(r.Context()), strings.TrimSpace(s.factoryNodeID))
	if err != nil {
		writeQCLookupError(w, err)
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
	err = backend.UpsertQC(r.Context(), qcUpsert{
		RequestID:   requestID,
		Result:      result,
		Notes:       notes,
		InspectedBy: inspector,
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateWarehouse, requestID, events.TopicMain, map[string]any{
			"type":         events.EventFactorySupplyRequestUpdate,
			"request_id":   requestID,
			"warehouse_id": meta.WarehouseID,
			"supplier_id":  meta.SupplierID,
			"factory_id":   s.factoryNodeID,
			"state":        meta.State,
			"qc_result":    result,
			"qc_notes":     notes,
			"timestamp":    nowTS,
		})
	})
	if err != nil {
		if errors.Is(err, errQCRequestNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "request_not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "qc_write_failed"})
		return
	}
	s.broadcastFactorySupplyEvent(r.Context(), map[string]any{
		"type": events.EventFactorySupplyRequestUpdate,
		"data": map[string]any{
			"request_id":   requestID,
			"state":        meta.State,
			"qc_result":    result,
			"warehouse_id": meta.WarehouseID,
		},
	})
	idemCommitted = true
	s.writeIdempotentJSON(w, r, body, http.StatusOK, SupplyRequestQCResponse{
		RequestID:   requestID,
		Result:      result,
		Notes:       notes,
		InspectedBy: inspector,
		InspectedAt: nowTS,
	})
}

type spannerQCRepo struct {
	client *spanner.Client
	now    func() time.Time
}

func (r spannerQCRepo) GetVisibleRequest(ctx context.Context, requestID, supplierID, factoryID string) (qcRequestMeta, error) {
	stmt := spanner.Statement{
		SQL: `SELECT sr.RequestId, sr.WarehouseId, sr.SupplierId, sr.State,
		             COALESCE(sr.FactoryId, w.PrimaryFactoryId, '')
		      FROM WarehouseSupplyRequests sr
		      INNER JOIN Warehouses w ON sr.WarehouseId = w.WarehouseId
		      WHERE sr.RequestId = @requestId
		        AND sr.SupplierId = @supplierId
		        AND (COALESCE(sr.FactoryId, w.PrimaryFactoryId, w.CoLocateWithFactoryId, '') = @factoryId
		             OR w.PrimaryFactoryId = @factoryId
		             OR w.CoLocateWithFactoryId = @factoryId)
		      LIMIT 1`,
		Params: map[string]any{
			"requestId":  requestID,
			"supplierId": supplierID,
			"factoryId":  factoryID,
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return qcRequestMeta{}, errQCRequestNotFound
	}
	if err != nil {
		return qcRequestMeta{}, err
	}
	var meta qcRequestMeta
	if err := row.Columns(&meta.RequestID, &meta.WarehouseID, &meta.SupplierID, &meta.State, &meta.FactoryID); err != nil {
		return qcRequestMeta{}, err
	}
	return meta, nil
}

func (s *Service) requireSupplyQCPass(ctx context.Context, requestID string) error {
	backend := s.qcBackend()
	if backend == nil {
		return errQCPassRequired
	}
	row, ok, err := backend.GetQC(ctx, requestID)
	if err != nil {
		return err
	}
	if !ok || !strings.EqualFold(strings.TrimSpace(row.Result), "PASS") {
		return errQCPassRequired
	}
	return nil
}

func (r spannerQCRepo) GetQC(ctx context.Context, requestID string) (SupplyRequestQCResponse, bool, error) {
	row, err := r.client.Single().ReadRow(ctx, "FactorySupplyRequestQC", spanner.Key{requestID},
		[]string{"RequestId", "Result", "Notes", "InspectedBy", "InspectedAt"})
	if err != nil {
		if qcRowMissing(err) {
			return SupplyRequestQCResponse{}, false, nil
		}
		return SupplyRequestQCResponse{}, false, err
	}
	resp, err := scanQCRow(row)
	if err != nil {
		return SupplyRequestQCResponse{}, false, err
	}
	return resp, true, nil
}

func (r spannerQCRepo) UpsertQC(ctx context.Context, row qcUpsert, emit func(outbox.TxnBuffer) error) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := txn.ReadRow(ctx, "WarehouseSupplyRequests", spanner.Key{row.RequestID}, []string{"RequestId", "State"})
		if err != nil {
			if qcRowMissing(err) {
				return errQCRequestNotFound
			}
			return err
		}
		muts := []*spanner.Mutation{spanner.InsertOrUpdateMap("FactorySupplyRequestQC", map[string]any{
			"RequestId":   row.RequestID,
			"Result":      row.Result,
			"Notes":       row.Notes,
			"InspectedBy": nullableString(row.InspectedBy),
			"InspectedAt": spanner.CommitTimestamp,
		})}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts = append(muts, outboxMutations(buf.events)...)
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func qcRowMissing(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "row not found")
}

func scanQCRow(row *spanner.Row) (SupplyRequestQCResponse, error) {
	var resp SupplyRequestQCResponse
	var notes, inspectedBy spanner.NullString
	var inspectedAt spanner.NullTime
	if err := row.Columns(&resp.RequestID, &resp.Result, &notes, &inspectedBy, &inspectedAt); err != nil {
		return SupplyRequestQCResponse{}, err
	}
	if notes.Valid {
		resp.Notes = notes.StringVal
	}
	if inspectedBy.Valid {
		resp.InspectedBy = inspectedBy.StringVal
	}
	if inspectedAt.Valid {
		resp.InspectedAt = inspectedAt.Time.UTC().Format(time.RFC3339)
	}
	return resp, nil
}
