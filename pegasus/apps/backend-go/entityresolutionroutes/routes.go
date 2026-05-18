// Package entityresolutionroutes owns supplier-scoped route composition for
// identity resolution and lineage explainability.
package entityresolutionroutes

import (
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	"backend-go/entityresolution"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware = func(http.HandlerFunc) http.HandlerFunc

// Deps bundles collaborators required to mount entity-resolution routes.
type Deps struct {
	Spanner     *spanner.Client
	Log         Middleware
	Idempotency Middleware
}

// RegisterRoutes mounts the supplier entity-resolution surface:
//
//	POST /v1/supplier/entity-resolution/resolve — deterministic and probabilistic identity resolution
//	POST /v1/supplier/entity-resolution/explain — semantic graph lineage projection for one source entity
func RegisterRoutes(r chi.Router, d Deps) {
	repo := entityresolution.NewRepository(d.Spanner)
	svc := entityresolution.NewService(repo)

	log := d.Log
	idem := d.Idempotency
	withRegionScope := auth.RequireRegionScopeWithClient(d.Spanner)
	supplierRole := []string{"SUPPLIER", "ADMIN"}

	r.HandleFunc("/v1/supplier/entity-resolution/resolve",
		auth.RequireRole(supplierRole, log(withRegionScope(idem(entityresolution.HandleResolve(svc))))))
	r.HandleFunc("/v1/supplier/entity-resolution/explain",
		auth.RequireRole(supplierRole, log(withRegionScope(idem(entityresolution.HandleExplain(svc))))))
}
