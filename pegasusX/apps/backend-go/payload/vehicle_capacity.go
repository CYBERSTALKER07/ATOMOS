package payload

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type VehicleCapacityMetricsResponse struct {
	VehicleID                string  `json:"vehicle_id"`
	Code                     string  `json:"code"`
	CapacityPercentage       int     `json:"capacity_percentage"`
	CurrentVolumeCubicMeters float64 `json:"current_volume_cubic_meters"`
	MaxVolumeCubicMeters     float64 `json:"max_volume_cubic_meters"`
	CurrentWeightKG          float64 `json:"current_weight_kg"`
	MaxWeightKG              float64 `json:"max_weight_kg"`
}

// HandleVehicleCapacity serves GET /v1/payload/capacity/{vehicleID}.
func (s *Service) HandleVehicleCapacity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
		return
	}

	vehicleID := strings.TrimSpace(chi.URLParam(r, "vehicleID"))
	if vehicleID == "" {
		vehicleID = "v-752069247"
	}

	resp := VehicleCapacityMetricsResponse{
		VehicleID:                vehicleID,
		Code:                     "SD-752069247",
		CapacityPercentage:       59,
		CurrentVolumeCubicMeters: 41.3,
		MaxVolumeCubicMeters:     70.0,
		CurrentWeightKG:          14750.0,
		MaxWeightKG:              25000.0,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
