package payload

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

var (
	// ErrRouteNotFound indicates no manifest or route twin stops exist for the requested vehicle/route.
	ErrRouteNotFound = errors.New("payload: route not found")
	// ErrLIFORepoUnavailable indicates no Spanner/route repository is available to resolve stop sequence.
	ErrLIFORepoUnavailable = errors.New("payload: repository unavailable")
)

// LIFORouteLookup carries the resolved delivery stop sequence for a vehicle/route/manifest.
type LIFORouteLookup struct {
	VehicleID  string
	RouteID    string
	ManifestID string
	StopIDs    []string
}

// LIFORouteReader resolves ordered delivery stops for a truck, route, or manifest.
type LIFORouteReader interface {
	LookupVehicleRoute(ctx context.Context, supplierID, truckOrRouteID string) (LIFORouteLookup, error)
}

// VehicleRoute models the CVRP optimized route returned by the routing/dispatch pipeline.
type VehicleRoute struct {
	VehicleID   string   `json:"vehicle_id"`
	LocationIDs []string `json:"location_ids"` // The original delivery order
}

// SequenceLIFO calculates the reverse-route loading sequence.
// LIFO (Last-In, First-Out) ensures the last stop is loaded first so it's at the back of the truck.
func SequenceLIFO(route VehicleRoute) []string {
	if len(route.LocationIDs) == 0 {
		return nil
	}

	seq := make([]string, len(route.LocationIDs))
	// Reverse the array
	for i, j := 0, len(route.LocationIDs)-1; i < len(route.LocationIDs); i, j = i+1, j-1 {
		seq[i] = route.LocationIDs[j]
	}
	return seq
}

// HandleGetLIFOSequence serves GET /v1/payload/sequence/{truckID}.
func (s *Service) HandleGetLIFOSequence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s == nil || s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "payload_repository_unavailable"})
		return
	}
	reader, ok := s.repo.(LIFORouteReader)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "payload_repository_unavailable"})
		return
	}

	targetID := extractSequenceTargetID(r)
	if targetID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "truck_id_required"})
		return
	}

	lookup, err := reader.LookupVehicleRoute(r.Context(), s.resolveSupplierScope(r.Context()), targetID)
	if err != nil {
		switch {
		case errors.Is(err, ErrLIFORepoUnavailable):
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "payload_repository_unavailable"})
		case errors.Is(err, ErrRouteNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "route_not_found"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "sequence_lookup_failed"})
		}
		return
	}
	if len(lookup.StopIDs) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "route_not_found"})
		return
	}

	vehicleID := strings.TrimSpace(lookup.VehicleID)
	if vehicleID == "" {
		vehicleID = targetID
	}
	route := VehicleRoute{
		VehicleID:   vehicleID,
		LocationIDs: lookup.StopIDs,
	}
	reversed := SequenceLIFO(route)

	resp := map[string]any{
		"vehicle_id":       route.VehicleID,
		"stops":            route.LocationIDs,
		"loading_sequence": reversed,
	}
	if lookup.RouteID != "" {
		resp["route_id"] = lookup.RouteID
	}
	if lookup.ManifestID != "" {
		resp["manifest_id"] = lookup.ManifestID
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func extractSequenceTargetID(r *http.Request) string {
	for _, key := range []string{"truckID", "vehicleID", "routeID", "id"} {
		if v := strings.TrimSpace(chi.URLParam(r, key)); v != "" {
			return v
		}
	}
	q := r.URL.Query()
	for _, key := range []string{"truck_id", "vehicle_id", "route_id"} {
		if v := strings.TrimSpace(q.Get(key)); v != "" {
			return v
		}
	}
	const prefix = "/v1/payload/sequence/"
	if strings.HasPrefix(r.URL.Path, prefix) {
		rest := strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/")
		if rest != "" && !strings.Contains(rest, "/") {
			return rest
		}
	}
	return ""
}
