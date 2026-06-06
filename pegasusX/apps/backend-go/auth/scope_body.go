package auth

import (
	"encoding/json"
	"net/http"
	"strings"
)

// RejectBodyScopeOverrides returns true when the JSON object contains identity
// fields that conflict with JWT-bound scope (writes 403 to w).
func RejectBodyScopeOverrides(w http.ResponseWriter, r *http.Request, body []byte) bool {
	if len(body) == 0 {
		return false
	}
	claims, ok := FromContext(r.Context())
	if !ok {
		return false
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return false
	}

	pinnedType, pinnedID := ResolveHomeNode(claims)

	for _, field := range []string{"supplier_id", "warehouse_id", "factory_id", "home_node_id", "home_node_type"} {
		val, exists := raw[field]
		if !exists {
			continue
		}
		var s string
		if err := json.Unmarshal(val, &s); err != nil {
			continue
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		switch field {
		case "supplier_id":
			if claims.SupplierID != "" && s != claims.SupplierID {
				writeScopeError(w, http.StatusForbidden, "access denied: supplier scope violation")
				return true
			}
		case "warehouse_id":
			if (pinnedType == HomeNodeWarehouse && s != pinnedID) ||
				(claims.SupplierRole == RoleWarehouseAdmin && claims.HomeNodeID != "" && s != claims.HomeNodeID) {
				writeScopeError(w, http.StatusForbidden, "access denied: warehouse scope violation")
				return true
			}
		case "factory_id":
			if (pinnedType == HomeNodeFactory && s != pinnedID) ||
				(claims.SupplierRole == RoleFactoryAdmin && claims.HomeNodeID != "" && s != claims.HomeNodeID) {
				writeScopeError(w, http.StatusForbidden, "access denied: factory scope violation")
				return true
			}
		case "home_node_id":
			if pinnedType != "" && pinnedID != "" && s != pinnedID {
				writeScopeError(w, http.StatusForbidden, "access denied: home node scope violation")
				return true
			}
		case "home_node_type":
			if pinnedType != "" && HomeNodeType(s) != pinnedType {
				writeScopeError(w, http.StatusForbidden, "access denied: home node scope violation")
				return true
			}
		}
	}
	return false
}
