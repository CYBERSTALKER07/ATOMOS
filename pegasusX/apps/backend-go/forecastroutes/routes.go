// Package forecastroutes mounts ATOMOS M1.5 Forecast v2 and one-tap
// replenishment suggestion approval endpoints onto chi.
package forecastroutes

import (
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/forecastv2"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
)

// Deps is the narrow dependency contract for mounting M1.5 forecast routes.
type Deps struct {
	Service          *forecastv2.Service
	ReplenishmentAPI *replenishment.SuggestionsAPI
	JWTSecret        string
	AllowAuthBypass  bool
}

// RegisterRoutes mounts Forecast v2 and one-tap suggestion approval endpoints onto r.
func RegisterRoutes(r chi.Router, d Deps) {
	if r == nil || (d.Service == nil && d.ReplenishmentAPI == nil) {
		return
	}

	mount := func(gr chi.Router) {
		if d.Service != nil {
			gr.With(auth.RequireRole(
				auth.RoleAdmin,
				auth.RolePlatformAdmin,
				auth.RoleWarehouseAdmin,
				auth.RoleWarehouse,
				auth.RoleRetailer,
			)).Get("/v1/planning/forecast", d.Service.HandleGetForecast)

			gr.With(auth.RequireRole(
				auth.RoleAdmin,
				auth.RolePlatformAdmin,
				auth.RoleWarehouseAdmin,
			)).Post("/v1/planning/forecast/run", d.Service.HandleRunForecast)

			gr.With(auth.RequireRole(
				auth.RoleAdmin,
				auth.RolePlatformAdmin,
				auth.RoleWarehouseAdmin,
			)).Post("/v1/planning/forecast/explanations/{id}/approve", d.Service.HandleApproveExplanation)
		}
		if d.ReplenishmentAPI != nil {
			gr.With(auth.RequireRole(
				auth.RoleRetailer,
				auth.RoleAdmin,
				auth.RolePlatformAdmin,
				auth.RoleWarehouseAdmin,
			)).Post("/v1/replenishment/suggestions/{id}/approve", d.ReplenishmentAPI.HandleApprove)
		}
	}

	r.Group(func(gr chi.Router) {
		if strings.TrimSpace(d.JWTSecret) != "" {
			gr.Use(auth.CookieAuth(d.JWTSecret))
		}
		gr.Use(auth.AttachTenantFromClaims)
		auth.ProtectMutations(gr, auth.MutationGuardConfig{
			AllowBypass: d.AllowAuthBypass,
		}, mount)
	})
}
