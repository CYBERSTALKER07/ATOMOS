// Package driverroutes mounts driver-role endpoints.
package driverroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/driver"
)

// Deps is the narrow dependency contract for driver routes.
type Deps struct {
	Service             *driver.Service
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
}

// RegisterRoutes mounts driver role-row endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	mountProtected := func(rr chi.Router) {
		rr.Get("/v1/driver/profile", d.Service.HandleProfile)
		rr.Get("/v1/driver/history", d.Service.HandleHistory)
		rr.Get("/v1/driver/earnings", d.Service.HandleEarnings)
		rr.Get("/v1/driver/availability", d.Service.HandleAvailability)
		rr.Patch("/v1/driver/availability", d.Service.HandleAvailability)
		rr.Get("/v1/driver/pending-collections", d.Service.HandlePendingCollections)
		rr.Get("/v1/driver/manifest-gate", d.Service.HandleManifestGate)
		rr.Get("/v1/driver/manifest", d.Service.HandleManifest)
		rr.Get("/v1/fleet/manifest", d.Service.HandleManifest)
	}

	if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
		r.Group(func(gr chi.Router) {
			gr.Use(auth.FirebaseAuth(d.FirebaseVerifier))
			gr.Use(auth.RequireRole(auth.RoleDriver))
			mountProtected(gr)
		})
		return
	}

	// Local scaffold fallback when Firebase is disabled.
	mountProtected(r)
}
