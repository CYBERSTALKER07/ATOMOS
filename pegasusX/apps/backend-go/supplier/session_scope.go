package supplier

import (
	"net/http"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// scopedSupplierID returns the request-scoped supplier for portal routes.
// Uses PreferTenantSupplierID so enforced envs fail closed instead of seed fallback.
func (s *Service) scopedSupplierID(r *http.Request) string {
	if r == nil {
		return s.supplierID
	}
	return auth.PreferTenantSupplierID(r.Context(), s.supplierID)
}

// ScopedSupplierID exposes scopedSupplierID for other route packages.
func (s *Service) ScopedSupplierID(r *http.Request) string {
	return s.scopedSupplierID(r)
}
