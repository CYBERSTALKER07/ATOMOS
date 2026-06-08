package warehouse

import (
	"net/http"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleWarehouseSetup serves POST /v1/warehouse/setup
func (s *Service) HandleWarehouseSetup(w http.ResponseWriter, r *http.Request) {
	// TODO: fully implement warehouse setup logic.
	web.JSONError(w, "Warehouse setup not implemented", http.StatusNotImplemented)
}
