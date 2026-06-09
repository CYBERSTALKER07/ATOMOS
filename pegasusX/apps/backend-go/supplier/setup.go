package supplier

import (
	"encoding/json"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleSupplierBusinessSetup serves POST /v1/supplier/business/setup
func (s *Service) HandleSupplierBusinessSetup(w http.ResponseWriter, r *http.Request) {
	if s.portalSpanner == nil {
		web.JSONError(w, "Spanner not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Name        string `json:"name"`
		CountryCode string `json:"country_code"`
		Currency    string `json:"currency"`
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
	if req.CountryCode == "" {
		req.CountryCode = "UZ"
	}
	if req.Currency == "" {
		req.Currency = "UZS"
	}

	supID := "sup-" + uuid.NewString()[:8]
	now := s.now().UTC()

	m := spanner.Insert("Suppliers",
		[]string{"SupplierId", "Name", "CountryCode", "Currency", "IsConfigured", "CreatedAt", "UpdatedAt"},
		[]any{supID, strings.TrimSpace(req.Name), req.CountryCode, req.Currency, true, now, now},
	)

	if _, err := s.portalSpanner.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
		s.log.ErrorContext(r.Context(), "failed to create supplier", "err", err)
		web.JSONError(w, "Failed to create supplier", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"supplier_id":  supID,
		"name":         req.Name,
		"country_code": req.CountryCode,
		"currency":     req.Currency,
	})
}
