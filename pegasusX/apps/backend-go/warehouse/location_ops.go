package warehouse

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type warehouseLocationResponse struct {
	WarehouseID string  `json:"warehouse_id"`
	Name        string  `json:"name"`
	Address     string  `json:"address,omitempty"`
	PlaceID     string  `json:"place_id,omitempty"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	UpdatedAt   string  `json:"updated_at,omitempty"`
}

type warehouseLocationPatch struct {
	Address string  `json:"address"`
	PlaceID string  `json:"place_id,omitempty"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
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
	writeJSON(w, http.StatusOK, loc)
}

func (s *Service) handleOpsLocationPatch(w http.ResponseWriter, r *http.Request) {
	warehouseID, ok := s.scopedWarehouseID(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id_required"})
		return
	}
	var req warehouseLocationPatch
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	now := time.Now().UTC()
	_, err := s.spannerClient.Apply(r.Context(), []*spanner.Mutation{
		spanner.UpdateMap("Warehouses", map[string]any{
			"WarehouseId": warehouseID,
			"Address":       strings.TrimSpace(req.Address),
			"PlaceId":       nullableString(strings.TrimSpace(req.PlaceID)),
			"Lat":           req.Lat,
			"Lng":           req.Lng,
			"UpdatedAt":     now,
		}),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_warehouse_location_failed"})
		return
	}
	loc, err := s.loadWarehouseLocation(r.Context(), warehouseID)
	if err != nil {
		writeJSON(w, http.StatusOK, warehouseLocationResponse{
			WarehouseID: warehouseID,
			Address:     strings.TrimSpace(req.Address),
			PlaceID:     strings.TrimSpace(req.PlaceID),
			Lat:         req.Lat,
			Lng:         req.Lng,
			UpdatedAt:   now.Format(time.RFC3339Nano),
		})
		return
	}
	writeJSON(w, http.StatusOK, loc)
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
		SQL: `SELECT WarehouseId, Name, COALESCE(Address, ''), COALESCE(PlaceId, ''), COALESCE(Lat, 0), COALESCE(Lng, 0), UpdatedAt
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
	if err := row.Columns(&resp.WarehouseID, &resp.Name, &resp.Address, &resp.PlaceID, &resp.Lat, &resp.Lng, &updatedAt); err != nil {
		return warehouseLocationResponse{}, err
	}
	resp.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)
	return resp, nil
}

func nullableString(v string) spanner.NullString {
	if strings.TrimSpace(v) == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: v, Valid: true}
}
