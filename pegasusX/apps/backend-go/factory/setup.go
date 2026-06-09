package factory

import (
	"encoding/json"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleFactorySetup serves POST /v1/factory/setup
func (s *Service) HandleFactorySetup(w http.ResponseWriter, r *http.Request) {
	if s.spannerClient == nil {
		web.JSONError(w, "Spanner not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Name string  `json:"name"`
		Lat  float64 `json:"lat"`
		Lng  float64 `json:"lng"`
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

	facID := "fac-" + uuid.NewString()[:8]
	now := s.now().UTC()

	m := spanner.Insert("Factories",
		[]string{"FactoryId", "SupplierId", "Name", "Lat", "Lng", "IsActive", "CreatedAt", "UpdatedAt"},
		[]any{facID, s.supplierID, strings.TrimSpace(req.Name), req.Lat, req.Lng, true, now, now},
	)

	if _, err := s.spannerClient.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
		s.log.ErrorContext(r.Context(), "failed to create factory", "err", err)
		web.JSONError(w, "Failed to create factory", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"factory_id":  facID,
		"supplier_id": s.supplierID,
		"name":        req.Name,
	})
}
