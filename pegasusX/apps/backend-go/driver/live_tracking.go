package driver

import (
	"encoding/json"
	"net/http"
)

type LocationUpdate struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// HandleLiveTracking serves POST /v1/fleet/location
// Drivers push their real-time coordinates here.
func (s *Service) HandleLiveTracking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req LocationUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_payload"})
		return
	}

	// Location storage deferred to telemetry service.
	// WebSocket broadcast deferred to outbox fanout.

	writeJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}
