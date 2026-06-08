package factory

import (
	"net/http"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleFactorySetup serves POST /v1/factory/setup
func (s *Service) HandleFactorySetup(w http.ResponseWriter, r *http.Request) {
	// TODO: fully implement factory setup logic.
	web.JSONError(w, "Factory setup not implemented", http.StatusNotImplemented)
}
