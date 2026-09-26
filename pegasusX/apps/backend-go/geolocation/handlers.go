package geolocation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Handler exposes geocode endpoints for all role clients.
type Handler struct {
	svc *Service
}

// NewHandler constructs HTTP handlers.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts geocode helpers.
func RegisterRoutes(r interface {
	Get(string, http.HandlerFunc)
	Post(string, http.HandlerFunc)
}, h *Handler) {
	if h == nil || h.svc == nil {
		return
	}
	r.Get("/v1/platform/geocode/autocomplete", h.handleAutocomplete)
	r.Get("/v1/platform/geocode/place", h.handlePlace)
	r.Get("/v1/platform/geocode/reverse", h.handleReverse)
	r.Post("/v1/platform/geocode/forward", h.handleForward)
}

func (h *Handler) checkAuth(w http.ResponseWriter, r *http.Request) bool {
	claims, ok := auth.FromContext(r.Context())
	if !ok || strings.TrimSpace(claims.Subject) == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return false
	}
	if auth.IsWSTicket(claims) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "code": "ws_ticket_not_allowed"})
		return false
	}
	return true
}

func queryCountry(r *http.Request) string {
	c := strings.TrimSpace(r.URL.Query().Get("country"))
	if c == "" {
		c = strings.TrimSpace(r.URL.Query().Get("country_code"))
	}
	return c
}

func (h *Handler) handleAutocomplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.checkAuth(w, r) {
		return
	}
	country := queryCountry(r)
	predictions, err := h.svc.Autocomplete(r.Context(), r.URL.Query().Get("input"), country)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"predictions": predictions})
}

func (h *Handler) handlePlace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.checkAuth(w, r) {
		return
	}
	country := queryCountry(r)
	loc, err := h.svc.ResolvePlaceID(r.Context(), r.URL.Query().Get("place_id"), country)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, loc)
}

func (h *Handler) handleReverse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.checkAuth(w, r) {
		return
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(r.URL.Query().Get("lat")), 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_lat"})
		return
	}
	lng, err := strconv.ParseFloat(strings.TrimSpace(r.URL.Query().Get("lng")), 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_lng"})
		return
	}
	country := queryCountry(r)
	loc, err := h.svc.ReverseGeocode(r.Context(), lat, lng, country)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, loc)
}

func (h *Handler) handleForward(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.checkAuth(w, r) {
		return
	}
	var req struct {
		Address     string `json:"address"`
		Country     string `json:"country,omitempty"`
		CountryCode string `json:"country_code,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	country := strings.TrimSpace(req.Country)
	if country == "" {
		country = strings.TrimSpace(req.CountryCode)
	}
	if country == "" {
		country = queryCountry(r)
	}
	loc, err := h.svc.ForwardGeocode(r.Context(), req.Address, country)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, loc)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

