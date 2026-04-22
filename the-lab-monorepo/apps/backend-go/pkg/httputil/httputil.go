// Package httputil provides standard HTTP response helpers used across handlers.
package httputil

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// AckResponse is the standard 202 Accepted response body for async operations.
// The client receives an acknowledgment immediately and polls or listens on
// WebSocket for the final result keyed by AckID.
type AckResponse struct {
	AckID       string `json:"ack_id"`
	Status      string `json:"status"`
	SubmittedAt string `json:"submitted_at"`
}

// WriteAccepted writes a 202 Accepted response with a generated AckID.
// The returned ack_id can be used by the caller for outbox event correlation.
func WriteAccepted(w http.ResponseWriter) string {
	ackID := uuid.NewString()
	resp := AckResponse{
		AckID:       ackID,
		Status:      "accepted",
		SubmittedAt: time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)
	return ackID
}

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// WriteError writes a JSON error response.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}
