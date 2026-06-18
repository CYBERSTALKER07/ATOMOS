package geolocation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// Handler exposes geocode endpoints for all role clients.
type Handler struct {
	svc *Service
}

// NewHandler constructs HTTP handlers.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts public geocode helpers used during onboarding.
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

func (h *Handler) handleAutocomplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}
	predictions, err := h.svc.Autocomplete(r.Context(), r.URL.Query().Get("input"))
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
	loc, err := h.svc.ResolvePlaceID(r.Context(), r.URL.Query().Get("place_id"))
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
	loc, err := h.svc.ReverseGeocode(r.Context(), lat, lng)
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
	var req struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	loc, err := h.svc.ForwardGeocode(r.Context(), req.Address)
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
