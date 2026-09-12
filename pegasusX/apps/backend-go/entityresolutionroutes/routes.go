package entityresolutionroutes

import (
	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/entityresolution"
)

type Deps struct {
	Spanner         *spanner.Client
	AllowAuthBypass bool
}

func RegisterRoutes(r chi.Router, d Deps) {
	if d.Spanner == nil {
		return
	}
	svc := entityresolution.NewService(entityresolution.NewRepository(d.Spanner))
	mount := func(gr chi.Router) {
		gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/entity-resolution/resolve", entityresolution.HandleResolve(svc))
		gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/entity-resolution/explain", entityresolution.HandleExplain(svc))
	}
	auth.ProtectMutations(r, auth.MutationGuardConfig{
		AllowBypass: d.AllowAuthBypass,
	}, mount)
}
