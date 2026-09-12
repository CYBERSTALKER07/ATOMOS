package planning

import "github.com/go-chi/chi/v5"

// RegisterAccuracyRoutes mounts the forecast-accuracy admin surface
// (read + ops trigger) onto r. Both handlers are admin-gated internally.
func RegisterAccuracyRoutes(r chi.Router, svc *AccuracyService) {
	if svc == nil {
		return
	}
	r.Get("/v1/admin/planning/accuracy", svc.HandleListAccuracy)
	r.Post("/v1/admin/planning/accuracy/run-once", svc.HandleRunAccuracyOnce)
}
