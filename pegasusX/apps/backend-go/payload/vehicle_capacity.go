package payload

import (
	"encoding/json"
	"net/http"
)

// HandleVehicleCapacity serves GET /v1/payload/capacity/{vehicleID}.
// P4-P1: hardcoded 59% / v-752069247 was theatre. VU lives on the manifest.
// Never return 200 with fake metrics.
func (s *Service) HandleVehicleCapacity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusGone)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   "capacity_unwired",
		"message": "GET /v1/payload/capacity/{vehicleID} is not a live vehicle metric; volume units live on the manifest",
	})
}
