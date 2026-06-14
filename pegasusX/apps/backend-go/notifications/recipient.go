package notifications

import (
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// RecipientIDFromClaims resolves the notification inbox key for the authenticated
// caller. Supplier-scoped roles read the org-wide supplier_id inbox; other roles
// use their subject (user / driver / retailer id).
func RecipientIDFromClaims(claims auth.Claims) string {
	switch claims.Role {
	case auth.RoleAdmin, auth.RolePayload, auth.RoleFactoryAdmin, auth.RoleWarehouseAdmin, auth.RoleFactory, auth.RoleWarehouse:
		if sid := strings.TrimSpace(claims.SupplierID); sid != "" {
			return sid
		}
	}
	if subject := strings.TrimSpace(claims.Subject); subject != "" {
		return subject
	}
	return ""
}
