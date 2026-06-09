package warehouse

import (
	"encoding/json"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleWarehouseSetup serves POST /v1/warehouse/setup
func (s *Service) HandleWarehouseSetup(w http.ResponseWriter, r *http.Request) {
	if s.spannerClient == nil {
		web.JSONError(w, "Spanner not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Name             string  `json:"name"`
		Lat              float64 `json:"lat"`
		Lng              float64 `json:"lng"`
		CoverageRadiusKm float64 `json:"coverage_radius_km"`
		PrimaryFactoryID string  `json:"primary_factory_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if strings.TrimSpace(req.Name) == "" {
		web.JSONError(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.CoverageRadiusKm <= 0 {
		req.CoverageRadiusKm = 25.0
	}

	whID := "wh-" + uuid.NewString()[:8]
	now := s.now().UTC()

	m := spanner.Insert("Warehouses",
		[]string{"WarehouseId", "SupplierId", "Name", "Lat", "Lng", "CoverageRadiusKm", "PrimaryFactoryId", "IsActive", "IsOnShift", "CreatedAt", "UpdatedAt"},
		[]any{whID, s.supplierID, strings.TrimSpace(req.Name), req.Lat, req.Lng, req.CoverageRadiusKm, req.PrimaryFactoryID, true, false, now, now},
	)

	if _, err := s.spannerClient.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
		s.log.ErrorContext(r.Context(), "failed to create warehouse", "err", err)
		web.JSONError(w, "Failed to create warehouse", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"warehouse_id":       whID,
		"supplier_id":        s.supplierID,
		"name":               req.Name,
		"coverage_radius_km": req.CoverageRadiusKm,
	})
}
