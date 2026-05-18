// Package infraroutes mounts infrastructure-only routes (health, readiness).
package infraroutes

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Deps is intentionally empty; infra routes carry no domain dependencies.
type Deps struct{}

// RegisterRoutes mounts /v1/health.
func RegisterRoutes(r chi.Router, _ Deps) {
	r.Get("/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "pegasusx-backend"})
	})
}
