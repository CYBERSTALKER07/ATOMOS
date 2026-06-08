package warehouse

import (
	"net/http"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleWarehouseRegister serves POST /v1/auth/warehouse/register
func (s *Service) HandleWarehouseRegister(w http.ResponseWriter, r *http.Request) {
	// TODO: fully implement warehouse registration logic.
	// For parity closure, returning Not Implemented.
	web.JSONError(w, "Warehouse registration not implemented", http.StatusNotImplemented)
}
