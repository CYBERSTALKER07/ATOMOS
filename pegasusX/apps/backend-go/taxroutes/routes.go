package taxroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/tax"
)

// Mount attaches tax subsystem routes to the provided router.
func Mount(r chi.Router, svc *tax.Service) {
	if svc == nil {
		return
	}

	r.Route("/v1/admin/tax-regimes", func(r chi.Router) {
		r.Post("/", svc.HandleCreateRegime)
		r.Get("/", svc.HandleListRegimes)
		r.Get("/{regimeID}", svc.HandleGetRegime)
	})
}
