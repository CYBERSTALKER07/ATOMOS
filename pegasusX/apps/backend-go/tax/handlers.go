package tax

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// HandleCreateRegime serves POST /v1/admin/tax-regimes.
func (s *Service) HandleCreateRegime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req CreateRegimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	regime, err := s.CreateRegime(r.Context(), claims, req)
	if err != nil {
		writeTaxError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, regime)
}

// HandleGetRegime serves GET /v1/admin/tax-regimes/{regimeID}.
func (s *Service) HandleGetRegime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	_, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	regimeID := strings.TrimSpace(chi.URLParam(r, "regimeID"))
	regime, found, err := s.GetRegime(r.Context(), regimeID)
	if err != nil {
		writeTaxError(w, r, err)
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "regime_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, regime)
}

// HandleListRegimes serves GET /v1/admin/tax-regimes?country=UZ&limit=50.
func (s *Service) HandleListRegimes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	_, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
	if country == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "country_required"})
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	regimes, err := s.ListRegimes(r.Context(), country, limit)
	if err != nil {
		writeTaxError(w, r, err)
		return
	}
	if regimes == nil {
		regimes = []TaxRegimeVersion{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"regimes": regimes})
}

func writeTaxError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrRegimeNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "regime_not_found"})
	case errors.Is(err, ErrRegimeForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	case errors.Is(err, ErrRegimeOverlap):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "regime_overlap", "message": err.Error()})
	case errors.Is(err, ErrRegimeInvalid), errors.Is(err, ErrMissingCountryCode):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": err.Error()})
	default:
		slog.ErrorContext(r.Context(), "tax regime operation failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	}
}
