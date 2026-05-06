// Package telemetryroutes owns telemetry websocket route composition.
package telemetryroutes

import (
	"time"

	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	"backend-go/telemetry"
)

// Deps bundles collaborators required to mount telemetry websocket routes.
type Deps struct {
	FleetHub *telemetry.Hub
}

// RegisterRoutes mounts websocket telemetry endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.FleetHub == nil {
		return
	}

	r.HandleFunc("/ws/telemetry",
		auth.RequireRoleWithGrace([]string{"DRIVER", "ADMIN", "SUPPLIER"}, 2*time.Hour, d.FleetHub.HandleConnection))
	r.HandleFunc("/ws/fleet",
		auth.RequireRoleWithGrace([]string{"DRIVER", "ADMIN", "SUPPLIER"}, 2*time.Hour, d.FleetHub.HandleConnection))
}
