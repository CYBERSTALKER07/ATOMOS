// Package platformroutes mounts version policy and device-token routes.
package platformroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/geolocation"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
)

// Deps supplies platform handlers.
type Deps struct {
	Handler        *platform.Handler
	GeocodeHandler *geolocation.Handler
	JWTSecret      string
	JWTIssuer      string
}

// RegisterRoutes mounts /v1/platform/* and shared device-token registration.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Handler == nil {
		return
	}
	r.Get("/v1/platform/client-policy", d.Handler.HandleClientPolicy)
	r.Get("/v1/platform/client-config", d.Handler.HandleClientConfig)
	r.With(auth.RequireRole(auth.RoleAdmin)).Put("/v1/platform/client-policy", d.Handler.HandleUpsertPolicy)
	r.Post("/v1/user/device-token", d.Handler.HandleDeviceToken)
	if d.GeocodeHandler != nil {
		geolocation.RegisterRoutes(r, d.GeocodeHandler)
	}
	if d.JWTSecret != "" {
		r.Post("/v1/auth/refresh", auth.HandleTokenRefresh(d.JWTSecret, d.JWTIssuer))
	}
}
