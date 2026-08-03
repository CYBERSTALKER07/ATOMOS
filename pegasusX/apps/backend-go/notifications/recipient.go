package notifications

import (
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// RecipientIDFromClaims resolves the notification inbox key for the authenticated
// caller. Supplier-scoped roles read the org-wide supplier_id inbox.
// Retailers prefer JWT subject (staff user id for TEAM; org id for legacy owner) so
// dual-written inbox rows (org + user) are visible; FCM tokens also key on subject.
func RecipientIDFromClaims(claims auth.Claims) string {
	switch claims.Role {
	case auth.RoleAdmin, auth.RolePayload, auth.RoleFactoryAdmin, auth.RoleWarehouseAdmin, auth.RoleFactory, auth.RoleWarehouse:
		if sid := strings.TrimSpace(claims.SupplierID); sid != "" {
			return sid
		}
	case auth.RoleRetailer:
		// Prefer person id when TEAM; fall back to org for legacy single-owner tokens.
		if uid := auth.ResolveRetailerUserID(claims); uid != "" {
			return uid
		}
		if org := auth.ResolveRetailerOrgID(claims); org != "" {
			return org
		}
	}
	if subject := strings.TrimSpace(claims.Subject); subject != "" {
		return subject
	}
	return ""
}
