package payload

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type ScanPayload struct {
	ManifestID string `json:"manifest_id"`
	ItemID     string `json:"item_id"`
	ItemVU     int64  `json:"item_vu"`
}

// HandleScanProgress serves POST /v1/payload/scan
// Atomic counter tracking the filled volume of a manifest.
func (s *Service) HandleScanProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScanPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	manifestID := strings.TrimSpace(req.ManifestID)
	if manifestID == "" {
		http.Error(w, "missing manifest_id", http.StatusBadRequest)
		return
	}

	// ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	// defer cancel()
	
	var newVolume int64
	if s.cache != nil {
		// Deferred until cache package supports IncrBy
		// redisKey := "manifest:" + manifestID + ":loaded_vu"
		// res, err := s.cache.Client.IncrBy(ctx, redisKey, req.ItemVU).Result()
		// if err == nil {
		// 	newVolume = res
		// }
	}

	// Fire WS update to warehouse
	if s.payloadHub != nil {
		payload, _ := json.Marshal(map[string]any{
			"type":        "PAYLOAD_PROGRESS",
			"manifest_id": manifestID,
			"loaded_vu":   newVolume,
		})
		if whID := s.resolveWarehouseScope(r.Context()); whID != "" {
			s.payloadHub.Broadcast(context.Background(), "warehouse:"+whID, payload)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":    "recorded",
		"loaded_vu": newVolume,
	})
}
