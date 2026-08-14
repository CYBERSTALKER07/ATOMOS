package countrycfg

import (
	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type Deps struct {
	Spanner             *spanner.Client
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
	AllowAuthBypass     bool
}

func RegisterRoutes(r chi.Router, d Deps) {
	h := &Handlers{Spanner: d.Spanner}
	mount := func(gr chi.Router) {
		gr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/country-configs/{code}", h.HandleCountryConfig)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/country-overrides/{code}", h.HandleCountryOverride)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Patch("/v1/supplier/country-overrides/{code}", h.HandleCountryOverride)
	}
	auth.ProtectMutations(r, auth.MutationGuardConfig{
		FirebaseEnabled:  d.FirebaseAuthEnabled,
		FirebaseVerifier: d.FirebaseVerifier,
		AllowBypass:      d.AllowAuthBypass,
	}, mount)
}
