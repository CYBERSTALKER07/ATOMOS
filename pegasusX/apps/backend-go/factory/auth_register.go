package factory

import (
	"net/http"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleFactoryRegister serves POST /v1/auth/factory/register
// Registers a new factory instance.
func (s *Service) HandleFactoryRegister(w http.ResponseWriter, r *http.Request) {
	// TODO: fully implement factory registration logic.
	// For parity closure, returning Not Implemented.
	web.JSONError(w, "Factory registration not implemented", http.StatusNotImplemented)
}
