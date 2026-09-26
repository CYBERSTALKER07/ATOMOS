package payload

import (
	"encoding/json"
	"net/http"
)

// VehicleRoute models the CVRP optimized route returned by the Python sidecar.
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

// HandleGetLIFOSequence serves GET /v1/payload/sequence/{truckID}
func (s *Service) HandleGetLIFOSequence(w http.ResponseWriter, r *http.Request) {
	// Stub implementation. In production, this would fetch the VehicleRoute from Spanner or Redis
	// and return the reversed sequence to the terminal.
	
	route := VehicleRoute{
		VehicleID: "TRUCK_123",
		LocationIDs: []string{"STOP_1", "STOP_2", "STOP_3"},
	}

	reversed := SequenceLIFO(route)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"vehicle_id":      route.VehicleID,
		"loading_sequence": reversed,
	})
}
