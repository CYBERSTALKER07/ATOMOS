// Package simroutes owns the stealth simulation harness routes under
// /v1/internal/sim/*. Handler implementations live in backend-go/simulation.
package simroutes

import (
	"net/http"

	"backend-go/auth"
	"backend-go/simulation"
	"github.com/go-chi/chi/v5"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

const (
	// PathSimStart starts the simulation harness.
	PathSimStart = "/v1/internal/sim/start"
	// PathSimStop stops the simulation harness.
	PathSimStop = "/v1/internal/sim/stop"
	// PathSimStatus returns the current simulation snapshot.
	PathSimStatus = "/v1/internal/sim/status"
)

// Deps bundles the collaborators required to mount /v1/internal/sim/*.
type Deps struct {
	Engine *simulation.Engine
	Log    Middleware
}

// RegisterRoutes mounts the simulation harness surface when the engine exists:
//
//	POST /v1/internal/sim/start
//	POST /v1/internal/sim/stop
//	GET  /v1/internal/sim/status
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Engine == nil {
		return
	}

	admin := []string{"ADMIN"}
	log := d.Log

	r.HandleFunc(PathSimStart, auth.RequireRole(admin, log(simulation.HandleStart(d.Engine))))
	r.HandleFunc(PathSimStop, auth.RequireRole(admin, log(simulation.HandleStop(d.Engine))))
	r.HandleFunc(PathSimStatus, auth.RequireRole(admin, log(simulation.HandleStatus(d.Engine))))
}
