package controltowerroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltower"
)

// Deps is the dependency contract for control tower routes.
type Deps struct {
	Handlers        *controltower.Handlers
	AllowAuthBypass bool
}

// RegisterRoutes mounts /v1/control-tower/* for supplier admins.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Handlers == nil {
		return
	}
	mount := func(gr chi.Router) {
		gr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/control-tower/exceptions/scored", d.Handlers.HandleScoredExceptions)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/control-tower/playbooks", d.Handlers.HandlePlaybooks)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/control-tower/playbooks", d.Handlers.HandlePlaybooks)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Patch("/v1/control-tower/playbooks/{id}", d.Handlers.HandlePlaybookByID)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/control-tower/playbooks/{id}/deactivate", d.Handlers.HandleDeactivatePlaybook)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/control-tower/runs", d.Handlers.HandleRuns)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/control-tower/runs/{id}/{action}", d.Handlers.HandleRunAction)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/control-tower/evaluate", d.Handlers.HandleEvaluate)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/control-tower/manifests/{id}/abort", d.Handlers.HandleManifestAbort)
	}
	auth.ProtectMutations(r, auth.MutationGuardConfig{
		AllowBypass: d.AllowAuthBypass,
	}, mount)
}
