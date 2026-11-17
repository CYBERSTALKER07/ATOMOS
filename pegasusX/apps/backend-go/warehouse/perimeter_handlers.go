package warehouse

import (
	"net/http"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandlePublishPerimeter serves POST /v1/warehouses/publish-perimeter
func (s *Service) HandlePublishPerimeter(w http.ResponseWriter, r *http.Request) {
	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		web.JSONError(w, "missing supplier scope", http.StatusUnauthorized)
		return
	}

	if err := s.PublishSupplierPerimeter(r.Context(), supplierID); err != nil {
		web.JSONError(w, "failed to publish perimeter: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, map[string]string{"status": "published"})
}
