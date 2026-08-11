package supplier

import (
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// scopedSupplierID returns the request-scoped supplier for portal routes.
// Prefers TenantContext, then JWT SupplierID, then bootstrap seed (tests only).
func (s *Service) scopedSupplierID(r *http.Request) string {
	if r == nil {
		return s.supplierID
	}
	if t, ok := auth.TenantFromContext(r.Context()); ok {
		return t.SupplierID
	}
	if claims, ok := auth.FromContext(r.Context()); ok {
		if sid := strings.TrimSpace(claims.SupplierID); sid != "" {
			return sid
		}
	}
	return s.supplierID
}

// ScopedSupplierID exposes scopedSupplierID for other route packages.
func (s *Service) ScopedSupplierID(r *http.Request) string {
	return s.scopedSupplierID(r)
}
