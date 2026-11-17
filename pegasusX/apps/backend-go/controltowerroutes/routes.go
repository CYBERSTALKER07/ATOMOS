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

// RegisterRoutes mounts /v1/control-tower/* for supplier and platform admins.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Handlers == nil {
		return
	}
	mount := func(gr chi.Router) {
		roleGuard := auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin)

		gr.With(roleGuard).Get("/v1/control-tower/exceptions/scored", d.Handlers.HandleScoredExceptions)
		gr.With(roleGuard).Get("/v1/control-tower/playbooks", d.Handlers.HandlePlaybooks)
		gr.With(roleGuard).Post("/v1/control-tower/playbooks", d.Handlers.HandlePlaybooks)
		gr.With(roleGuard).Patch("/v1/control-tower/playbooks/{id}", d.Handlers.HandlePlaybookByID)
		gr.With(roleGuard).Post("/v1/control-tower/playbooks/{id}/deactivate", d.Handlers.HandleDeactivatePlaybook)
		gr.With(roleGuard).Get("/v1/control-tower/runs", d.Handlers.HandleRuns)
		gr.With(roleGuard).Post("/v1/control-tower/runs/{id}/{action}", d.Handlers.HandleRunAction)
		gr.With(roleGuard).Post("/v1/control-tower/evaluate", d.Handlers.HandleEvaluate)
		gr.With(roleGuard).Post("/v1/control-tower/manifests/{id}/abort", d.Handlers.HandleManifestAbort)

		// Control Tower macOS observability & analytics endpoints
		gr.With(roleGuard).Get("/v1/control-tower/cells", d.Handlers.HandleCells)
		gr.With(roleGuard).Get("/v1/control-tower/metrics/spanner", d.Handlers.HandleSpannerMetrics)
		gr.With(roleGuard).Get("/v1/control-tower/metrics/kafka", d.Handlers.HandleKafkaMetrics)
		gr.With(roleGuard).Get("/v1/control-tower/metrics/redis", d.Handlers.HandleRedisMetrics)
		gr.With(roleGuard).Get("/v1/control-tower/waterfall", d.Handlers.HandleWaterfall)
		gr.With(roleGuard).Get("/v1/control-tower/costs/attributed", d.Handlers.HandleAttributedCosts)
		gr.With(roleGuard).Get("/v1/control-tower/scorecards", d.Handlers.HandleScorecards)
	}
	auth.ProtectMutations(r, auth.MutationGuardConfig{
		AllowBypass: d.AllowAuthBypass,
	}, mount)
}
