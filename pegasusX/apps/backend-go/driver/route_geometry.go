package driver

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
)

// RouteGeometryOptions controls route overlay resolution.
type RouteGeometryOptions struct {
	IncludeSteps bool
	RerouteFrom  *routing.LatLng
}

// RouteGeometryLookup resolves a driver's route overlay for map rendering.
type RouteGeometryLookup func(ctx context.Context, driverID, routeID string, opts RouteGeometryOptions) (routing.RouteGeometry, bool, error)

// HandleRouteGeometry serves GET /v1/fleet/route/{routeID}/geometry.
func (s *Service) HandleRouteGeometry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	routeID := strings.TrimSpace(chi.URLParam(r, "routeID"))
	if routeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "route_id_required"})
		return
	}
	opts := RouteGeometryOptions{
		IncludeSteps: strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("include_steps")), "true"),
	}
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("reroute")), "true") {
		lat, latErr := strconv.ParseFloat(strings.TrimSpace(r.URL.Query().Get("from_lat")), 64)
		lng, lngErr := strconv.ParseFloat(strings.TrimSpace(r.URL.Query().Get("from_lng")), 64)
		if latErr == nil && lngErr == nil {
			opts.RerouteFrom = &routing.LatLng{Lat: lat, Lng: lng}
		}
	}
	if s.routeGeometry == nil {
		if allowDriverDemoFallback() {
			writeJSON(w, http.StatusOK, demoRouteGeometry(routeID))
			return
		}
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "route_geometry_unavailable"})
		return
	}

	geometry, ok, err := s.routeGeometry(r.Context(), driverID, routeID, opts)
	if err != nil {
		s.log.ErrorContext(r.Context(), "route geometry lookup failed",
			"err", err, "driver_id", driverID, "route_id", routeID)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "route_geometry_unavailable"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "route_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, geometry)
}
