package supplier

import (
	"net/http"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleSupplierBusinessSetup serves POST /v1/supplier/business/setup
func (s *Service) HandleSupplierBusinessSetup(w http.ResponseWriter, r *http.Request) {
	// TODO: fully implement supplier business setup logic.
	web.JSONError(w, "Supplier business setup not implemented", http.StatusNotImplemented)
}
