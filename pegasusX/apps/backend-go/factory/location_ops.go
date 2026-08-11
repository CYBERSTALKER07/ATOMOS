package factory

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
)

type factoryLocationResponse struct {
	FactoryID string  `json:"factory_id"`
	Name      string  `json:"name"`
	Address   string  `json:"address,omitempty"`
	PlaceID   string  `json:"place_id,omitempty"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	UpdatedAt string  `json:"updated_at,omitempty"`
}

type factoryLocationPatch struct {
	Address string  `json:"address"`
	PlaceID string  `json:"place_id,omitempty"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}

// HandleOpsLocation serves GET/PATCH /v1/factory/ops/location.
// Any authenticated factory-scoped caller (FACTORY, FACTORY_ADMIN, FACTORY_STAFF JWT)
// may read and update their assigned factory address — no admin-only gate.
func (s *Service) HandleOpsLocation(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleFactoryLocationGet(w, r)
	case http.MethodPatch:
		s.handleFactoryLocationPatch(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleFactoryLocationGet(w http.ResponseWriter, r *http.Request) {
	factoryID, ok := s.scopedFactoryID(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "factory_id_required"})
		return
	}
	loc, err := s.loadFactoryLocation(r.Context(), factoryID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_factory_location_failed"})
		return
	}
	writeJSON(w, http.StatusOK, loc)
}

func (s *Service) handleFactoryLocationPatch(w http.ResponseWriter, r *http.Request) {
	factoryID, ok := s.scopedFactoryID(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "factory_id_required"})
		return
	}

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

	var req factoryLocationPatch
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if strings.TrimSpace(req.Address) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "address_required"})
		return
	}
	if req.Lat < -90 || req.Lat > 90 || req.Lng < -180 || req.Lng > 180 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "coordinates_out_of_range"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "spanner_unavailable"})
		return
	}

	address := strings.TrimSpace(req.Address)
	placeID := strings.TrimSpace(req.PlaceID)
	h3Cell := proximity.H3CellFromLatLng(req.Lat, req.Lng)
	supplierID := strings.TrimSpace(s.resolveSupplierScope(r.Context()))

	update := map[string]any{
		"FactoryId": factoryID,
		"Address":   address,
		"PlaceId":   factoryNullableString(placeID),
		"Lat":       req.Lat,
		"Lng":       req.Lng,
		"UpdatedAt": spanner.CommitTimestamp,
	}
	if h3Cell != "" {
		update["H3Cell"] = h3Cell
	} else {
		update["H3Cell"] = nil
	}

	err = spannerutils.RunReadWriteTransaction(r.Context(), s.spannerClient, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		eventPayload := events.FactoryEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventFactoryLocationUpdated},
			FactoryID:  factoryID,
			SupplierID: supplierID,
			Lat:        req.Lat,
			Lng:        req.Lng,
			H3Cell:     h3Cell,
		}
		buf := &spannerTxnBuffer{}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregateFactory, factoryID, events.TopicMain, eventPayload); emitErr != nil {
			return emitErr
		}
		mutations := []*spanner.Mutation{spanner.UpdateMap("Factories", update)}
		mutations = append(mutations, outboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_factory_location_failed"})
		return
	}

	s.broadcastFactoryEvent(r.Context(), events.EventFactoryLocationUpdated, map[string]any{
		"factory_id": factoryID,
		"lat":        req.Lat,
		"lng":        req.Lng,
	})

	loc, err := s.loadFactoryLocation(r.Context(), factoryID)
	if err != nil {
		now := s.now().UTC()
		loc = factoryLocationResponse{
			FactoryID: factoryID,
			Address:   address,
			PlaceID:   placeID,
			Lat:       req.Lat,
			Lng:       req.Lng,
			UpdatedAt: now.Format(time.RFC3339Nano),
		}
	}
	respBytes, err := json.Marshal(loc)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encode_location_response_failed"})
		return
	}
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

func (s *Service) scopedFactoryID(r *http.Request) (string, bool) {
	if scope := auth.GetFactoryScope(r.Context()); scope != nil && strings.TrimSpace(scope.FactoryID) != "" {
		return scope.FactoryID, true
	}
	if fac := strings.TrimSpace(r.URL.Query().Get("factory_id")); fac != "" {
		if claims, ok := auth.FromContext(r.Context()); ok {
			if claims.HomeNodeID != "" && claims.HomeNodeID != fac {
				return "", false
			}
		}
		return fac, true
	}
	if claims, ok := auth.FromContext(r.Context()); ok && claims.HomeNodeID != "" {
		return claims.HomeNodeID, true
	}
	return "", false
}

func (s *Service) loadFactoryLocation(ctx context.Context, factoryID string) (factoryLocationResponse, error) {
	supplierID := strings.TrimSpace(s.resolveSupplierScope(ctx))
	sql := `SELECT FactoryId, Name, COALESCE(Address, ''), COALESCE(PlaceId, ''), COALESCE(Lat, 0), COALESCE(Lng, 0), UpdatedAt
		      FROM Factories WHERE FactoryId = @fid`
	params := map[string]any{"fid": factoryID}
	if supplierID != "" {
		sql += ` AND SupplierId = @sid`
		params["sid"] = supplierID
	}
	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return factoryLocationResponse{}, err
	}
	var resp factoryLocationResponse
	var updatedAt time.Time
	if err := row.Columns(&resp.FactoryID, &resp.Name, &resp.Address, &resp.PlaceID, &resp.Lat, &resp.Lng, &updatedAt); err != nil {
		return factoryLocationResponse{}, err
	}
	resp.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)
	return resp, nil
}

func factoryNullableString(v string) spanner.NullString {
	if strings.TrimSpace(v) == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: v, Valid: true}
}
