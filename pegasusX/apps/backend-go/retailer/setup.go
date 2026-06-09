package retailer

import (
	"encoding/json"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleRetailerSetup serves POST /v1/retailer/setup
func (s *Service) HandleRetailerSetup(w http.ResponseWriter, r *http.Request) {
	if s.spannerClient == nil {
		web.JSONError(w, "Spanner not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Name        string  `json:"name"`
		Phone       string  `json:"phone"`
		CountryCode string  `json:"country_code"`
		Lat         float64 `json:"lat"`
		Lng         float64 `json:"lng"`
		RegionID    string  `json:"region_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if strings.TrimSpace(req.Phone) == "" || strings.TrimSpace(req.Name) == "" {
		web.JSONError(w, "phone and name are required", http.StatusBadRequest)
		return
	}
	if req.CountryCode == "" {
		req.CountryCode = "UZ"
	}

	retID := "ret-" + uuid.NewString()[:8]
	now := s.now().UTC()

	var regionID spanner.NullString
	if req.RegionID != "" {
		regionID = spanner.NullString{StringVal: req.RegionID, Valid: true}
	}

	m := spanner.Insert("Retailers",
		[]string{"RetailerId", "Phone", "Name", "CountryCode", "Lat", "Lng", "RegionId", "CreatedAt"},
		[]any{retID, strings.TrimSpace(req.Phone), strings.TrimSpace(req.Name), req.CountryCode, req.Lat, req.Lng, regionID, now},
	)

	if _, err := s.spannerClient.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
		s.log.ErrorContext(r.Context(), "failed to create retailer", "err", err)
		web.JSONError(w, "Failed to create retailer", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"retailer_id":  retID,
		"phone":        req.Phone,
		"name":         req.Name,
		"country_code": req.CountryCode,
	})
}
