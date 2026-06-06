package auth

import "context"

// ResolveHomeNode derives (HomeNodeType, HomeNodeId) from JWT claims.
// Scoped callers are pinned to their node; unscoped supplier admins return ("", "").
func ResolveHomeNode(c Claims) (HomeNodeType, string) {
	if c.SupplierRole == RoleFactoryAdmin && c.HomeNodeID != "" {
		return HomeNodeFactory, c.HomeNodeID
	}
	if c.Role == RoleFactory && c.HomeNodeID != "" {
		return HomeNodeFactory, c.HomeNodeID
	}
	if c.SupplierRole == RoleWarehouseAdmin && c.HomeNodeID != "" {
		return HomeNodeWarehouse, c.HomeNodeID
	}
	if c.Role == RoleWarehouse && c.HomeNodeID != "" {
		return HomeNodeWarehouse, c.HomeNodeID
	}
	if c.HomeNodeType != "" && c.HomeNodeID != "" {
		return c.HomeNodeType, c.HomeNodeID
	}
	return "", ""
}

// ResolveHomeNodeFromContext pulls claims from ctx and delegates to ResolveHomeNode.
func ResolveHomeNodeFromContext(ctx context.Context) (HomeNodeType, string) {
	c, ok := FromContext(ctx)
	if !ok {
		return "", ""
	}
	return ResolveHomeNode(c)
}

// ApplyHomeNodeOverride lets unscoped supplier admins specify a target node in the
// request body while scoped callers remain pinned to JWT-bound nodes.
func ApplyHomeNodeOverride(c Claims, reqType HomeNodeType, reqID string) (HomeNodeType, string, bool) {
	resolvedType, resolvedID := ResolveHomeNode(c)
	if resolvedType != "" && resolvedID != "" {
		if reqType != "" && (reqType != resolvedType || reqID != resolvedID) {
			return "", "", false
		}
		return resolvedType, resolvedID, true
	}
	if reqType != "" && reqType != HomeNodeWarehouse && reqType != HomeNodeFactory {
		return "", "", false
	}
	return reqType, reqID, true
}
