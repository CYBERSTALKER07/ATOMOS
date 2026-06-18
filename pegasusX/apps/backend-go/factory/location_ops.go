package factory

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
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
	var req factoryLocationPatch
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
		spanner.UpdateMap("Factories", map[string]any{
			"FactoryId": factoryID,
			"Address":   strings.TrimSpace(req.Address),
			"PlaceId":   factoryNullableString(strings.TrimSpace(req.PlaceID)),
			"Lat":       req.Lat,
			"Lng":       req.Lng,
			"UpdatedAt": now,
		}),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_factory_location_failed"})
		return
	}
	loc, err := s.loadFactoryLocation(r.Context(), factoryID)
	if err != nil {
		writeJSON(w, http.StatusOK, factoryLocationResponse{
			FactoryID: factoryID,
			Address:   strings.TrimSpace(req.Address),
			PlaceID:   strings.TrimSpace(req.PlaceID),
			Lat:       req.Lat,
			Lng:       req.Lng,
			UpdatedAt: now.Format(time.RFC3339Nano),
		})
		return
	}
	writeJSON(w, http.StatusOK, loc)
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
	supplierID := strings.TrimSpace(s.supplierID)
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
