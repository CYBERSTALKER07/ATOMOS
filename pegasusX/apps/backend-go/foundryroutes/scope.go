package foundryroutes

import (
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// requireFoundrySession rejects calls that have no authenticated principal.
// The package must fail closed even when a caller mounts it without the
// process-wide session middleware.
func requireFoundrySession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.FromContext(r.Context())
		if !ok || (strings.TrimSpace(claims.Subject) == "" && claims.Role == "") {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// bindSupplierID returns the supplier the caller is allowed to mutate.
// Platform admins may name a supplier. Every other role is pinned to the
// session supplier. A body supplier that disagrees with the session is rejected.
func bindSupplierID(w http.ResponseWriter, r *http.Request, bodySupplier string) (string, bool) {
	bodySupplier = strings.TrimSpace(bodySupplier)
	if auth.IsPlatformAdmin(r.Context()) {
		if bodySupplier != "" {
			return bodySupplier, true
		}
		if caller, ok := auth.CallerSupplierID(r.Context()); ok && strings.TrimSpace(caller) != "" {
			return strings.TrimSpace(caller), true
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "supplier_id_required"})
		return "", false
	}
	caller, ok := auth.CallerSupplierID(r.Context())
	caller = strings.TrimSpace(caller)
	if !ok || caller == "" {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "supplier_scope_required"})
		return "", false
	}
	if bodySupplier != "" && bodySupplier != caller {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "supplier_scope_mismatch"})
		return "", false
	}
	return caller, true
}

func objectVisible(r *http.Request, supplierID string) bool {
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return auth.IsPlatformAdmin(r.Context())
	}
	return auth.EntitySupplierAllowed(r.Context(), supplierID)
}
