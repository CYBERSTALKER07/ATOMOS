package supplier

import (
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// scopedSupplierID returns the JWT-bound supplier for portal routes, falling back
// to the bootstrap seed supplier when no session is present (local tests).
func (s *Service) scopedSupplierID(r *http.Request) string {
	if r != nil {
		if claims, ok := auth.FromContext(r.Context()); ok {
			if claims.Role == auth.RoleAdmin {
				if sid := strings.TrimSpace(claims.SupplierID); sid != "" {
					return sid
				}
			}
		}
	}
	return s.supplierID
}
