package retailer

import (
	"net/http"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleRetailerSetup serves POST /v1/retailer/setup
func (s *Service) HandleRetailerSetup(w http.ResponseWriter, r *http.Request) {
	// TODO: fully implement retailer setup logic.
	web.JSONError(w, "Retailer setup not implemented", http.StatusNotImplemented)
}
