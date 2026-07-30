package twin

import (
	"encoding/json"
	"net/http"
	"strings"
)

type HTTPHandler struct {
	repo Repository
}

func NewHTTPHandler(repo Repository) *HTTPHandler {
	return &HTTPHandler{repo: repo}
}

func (h *HTTPHandler) ListActiveRoutes(w http.ResponseWriter, r *http.Request) {
	zoneH3 := r.URL.Query().Get("zoneH3")

	routes, err := h.repo.ListActiveRouteTwins(r.Context(), zoneH3)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if routes == nil {
		routes = []RouteTwinView{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(routes)
}

func (h *HTTPHandler) GetRoute(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	id := parts[4]

	route, err := h.repo.GetRouteTwin(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if route == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(route)
}

func (h *HTTPHandler) GetRouteInventory(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	id := parts[4]

	inv, err := h.repo.GetVehicleInventory(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if inv == nil {
		inv = []VehicleInventory{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inv)
}

// ServeHTTP optionally routes requests internally if a router framework is not used.
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/inventory") {
		h.GetRouteInventory(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/active") {
		h.ListActiveRoutes(w, r)
		return
	}

	// Assume /v1/twin/routes/{id}
	h.GetRoute(w, r)
}
