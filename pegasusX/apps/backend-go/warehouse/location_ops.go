package warehouse

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gs1"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
)

type warehouseLocationResponse struct {
	WarehouseID     string  `json:"warehouse_id"`
	Name            string  `json:"name"`
	Address         string  `json:"address,omitempty"`
	PlaceID         string  `json:"place_id,omitempty"`
	Lat             float64 `json:"lat"`
	Lng             float64 `json:"lng"`
	Gln             string  `json:"gln,omitempty"`
	CountryCode     string  `json:"country_code,omitempty"`
	PackCountryCode string  `json:"pack_country_code,omitempty"`
	CurrencyCode    string  `json:"currency_code,omitempty"`
	Timezone        string  `json:"timezone,omitempty"`
	UpdatedAt       string  `json:"updated_at,omitempty"`
}

type warehouseLocationPatch struct {
	Address string  `json:"address"`
	PlaceID string  `json:"place_id,omitempty"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Gln     *string `json:"gln,omitempty"`
}

// HandleOpsLocation serves GET/PATCH /v1/warehouse/ops/location for warehouse admins.
func (s *Service) HandleOpsLocation(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleOpsLocationGet(w, r)
	case http.MethodPatch:
		s.handleOpsLocationPatch(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleOpsLocationGet(w http.ResponseWriter, r *http.Request) {
	warehouseID, ok := s.scopedWarehouseID(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id_required"})
		return
	}
	loc, err := s.loadWarehouseLocation(r.Context(), warehouseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_warehouse_location_failed"})
		return
	}
	writeJSON(w, http.StatusOK, decorateLocationWithPack(r.Context(), loc))
}

func (s *Service) handleOpsLocationPatch(w http.ResponseWriter, r *http.Request) {
	warehouseID, ok := s.scopedWarehouseID(r)
	if !ok {
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

	var req warehouseLocationPatch
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
	var gln string
	if req.Gln != nil {
		raw := strings.TrimSpace(*req.Gln)
		if raw != "" {
			norm, err := gs1.NormalizeGLN(raw)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			gln = norm
		}
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "spanner_unavailable"})
		return
	}

	address := strings.TrimSpace(req.Address)
	placeID := strings.TrimSpace(req.PlaceID)
	geo, err := stampWarehouseCoords(r.Context(), req.Lat, req.Lng, "")
	if err != nil {
		if writeMarketLaw(w, err) {
			return
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	supplierID := s.resolveDispatchSupplierID(r.Context(), warehouseID)

	update := map[string]any{
		"WarehouseId": warehouseID,
		"Address":     address,
		"PlaceId":     nullableString(placeID),
		"Lat":         req.Lat,
		"Lng":         req.Lng,
		"CountryCode": geo.CountryCode,
		"H3Cell":      geo.H3Cell,
		"UpdatedAt":   spanner.CommitTimestamp,
	}
	if req.Gln != nil {
		update["Gln"] = nullableString(gln)
	}

	err = spannerutils.RunReadWriteTransaction(r.Context(), s.spannerClient, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		eventPayload := events.WarehouseEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventWarehouseLocationUpdated},
			WarehouseID: warehouseID,
			SupplierID:  supplierID,
		}
		buf := &spannerTxnBuffer{}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, warehouseID, events.TopicMain, eventPayload); emitErr != nil {
			return emitErr
		}
		mutations := []*spanner.Mutation{spanner.UpdateMap("Warehouses", update)}
		mutations = append(mutations, outboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_warehouse_location_failed"})
		return
	}

	s.InvalidateDispatchPlanCache(r.Context(), warehouseID)
	s.broadcastWarehouseEvent(r.Context(), warehouseID, map[string]any{
		"type":         events.EventWarehouseLocationUpdated,
		"warehouse_id": warehouseID,
		"lat":          req.Lat,
		"lng":          req.Lng,
	})

	loc, err := s.loadWarehouseLocation(r.Context(), warehouseID)
	if err != nil {
		now := s.now().UTC()
		loc = warehouseLocationResponse{
			WarehouseID: warehouseID,
			Address:     address,
			PlaceID:     placeID,
			Lat:         req.Lat,
			Lng:         req.Lng,
			UpdatedAt:   now.Format(time.RFC3339Nano),
		}
	}
	respBytes, err := json.Marshal(loc)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encode_location_response_failed"})
		return
	}
	s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	idemCommitted = true
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

func (s *Service) scopedWarehouseID(r *http.Request) (string, bool) {
	if wh := strings.TrimSpace(r.URL.Query().Get("warehouse_id")); wh != "" {
		if claims, ok := auth.FromContext(r.Context()); ok {
			if claims.SupplierRole == auth.RoleWarehouseAdmin && claims.HomeNodeID != "" && claims.HomeNodeID != wh {
				return "", false
			}
		}
		return wh, true
	}
	if claims, ok := auth.FromContext(r.Context()); ok && claims.HomeNodeID != "" {
		return claims.HomeNodeID, true
	}
	return "", false
}

func (s *Service) loadWarehouseLocation(ctx context.Context, warehouseID string) (warehouseLocationResponse, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, Name, COALESCE(Address, ''), COALESCE(PlaceId, ''), COALESCE(Lat, 0), COALESCE(Lng, 0),
		             COALESCE(Gln, ''), COALESCE(CountryCode, ''), UpdatedAt
		      FROM Warehouses WHERE WarehouseId = @wid`,
		Params: map[string]any{"wid": warehouseID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return warehouseLocationResponse{}, err
	}
	var resp warehouseLocationResponse
	var updatedAt time.Time
	if err := row.Columns(&resp.WarehouseID, &resp.Name, &resp.Address, &resp.PlaceID, &resp.Lat, &resp.Lng, &resp.Gln, &resp.CountryCode, &updatedAt); err != nil {
		return warehouseLocationResponse{}, err
	}
	resp.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)
	if tz, err := auth.TimezoneNameFromContext(ctx, ""); err == nil {
		resp.Timezone = tz
	}
	return resp, nil
}

func decorateLocationWithPack(ctx context.Context, loc warehouseLocationResponse) warehouseLocationResponse {
	pack, err := auth.CheckoutPackFromContext(ctx)
	if err != nil {
		return loc
	}
	if country, err := auth.PackCountryCode(pack); err == nil {
		loc.PackCountryCode = country
		if strings.TrimSpace(loc.CountryCode) == "" {
			loc.CountryCode = country
		}
	}
	if cur, err := auth.PackCurrency(pack); err == nil {
		loc.CurrencyCode = cur
	}
	return loc
}

func nullableString(v string) spanner.NullString {
	if strings.TrimSpace(v) == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: v, Valid: true}
}
