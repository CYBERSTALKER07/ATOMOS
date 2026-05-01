package auth

import "context"

// ─── Home-Node Resolution (V.O.I.D. Phase VII) ────────────────────────────────
//
// Drivers and Vehicles are home-based at a specific Warehouse OR Factory node.
// The canonical pair on those Spanner rows is (HomeNodeType, HomeNodeId) where
// HomeNodeType ∈ {"WAREHOUSE", "FACTORY"}. Per V.O.I.D. doctrine the home node
// is derived from the authenticated caller's JWT scope — NEVER from the
// request body — to prevent role-spoofing writes.
//
//   SupplierRole = NODE_ADMIN      → ("WAREHOUSE", claims.WarehouseID)
//   SupplierRole = FACTORY_ADMIN   → ("FACTORY",   claims.FactoryID)
//   Role         = FACTORY (staff) → ("FACTORY",   claims.FactoryID)
//   SupplierRole = GLOBAL_ADMIN    → ("", "")  — caller may pass an override
//                                                 via an explicit request body
//                                                 (treated as advisory only).

const (
	HomeNodeTypeWarehouse = "WAREHOUSE"
	HomeNodeTypeFactory   = "FACTORY"
)

// ResolveHomeNode derives (HomeNodeType, HomeNodeId) from the authenticated
// claims. Returns ("", "") for GLOBAL_ADMIN or any claim shape that does not
// bind the caller to a specific node.
func ResolveHomeNode(claims *PegasusClaims) (string, string) {
	if claims == nil {
		return "", ""
	}
	switch {
	case claims.SupplierRole == "FACTORY_ADMIN" && claims.FactoryID != "":
		return HomeNodeTypeFactory, claims.FactoryID
	case claims.Role == "FACTORY" && claims.FactoryID != "":
		return HomeNodeTypeFactory, claims.FactoryID
	case claims.SupplierRole == "NODE_ADMIN" && claims.WarehouseID != "":
		return HomeNodeTypeWarehouse, claims.WarehouseID
	case claims.WarehouseID != "":
		return HomeNodeTypeWarehouse, claims.WarehouseID
	}
	return "", ""
}

// ResolveHomeNodeFromContext is a convenience wrapper that pulls the claims
// out of the request context before delegating to ResolveHomeNode.
func ResolveHomeNodeFromContext(ctx context.Context) (string, string) {
	claims, _ := ctx.Value(ClaimsContextKey).(*PegasusClaims)
	return ResolveHomeNode(claims)
}

// ApplyHomeNodeOverride lets GLOBAL_ADMIN callers specify a target node in the
// request body while keeping scoped callers (NODE_ADMIN / FACTORY_ADMIN)
// pinned to their JWT-bound node. Returns (nodeType, nodeId, ok) — ok=false
// when the override violates the caller's scope (403-worthy).
func ApplyHomeNodeOverride(claims *PegasusClaims, reqType, reqID string) (string, string, bool) {
	resolvedType, resolvedID := ResolveHomeNode(claims)
	if resolvedType != "" && resolvedID != "" {
		// Scoped caller — override is rejected if it tries to escape scope.
		if reqType != "" && (reqType != resolvedType || reqID != resolvedID) {
			return "", "", false
		}
		return resolvedType, resolvedID, true
	}
	// GLOBAL_ADMIN (or unscoped) — body value wins (may still be empty).
	if reqType != "" && reqType != HomeNodeTypeWarehouse && reqType != HomeNodeTypeFactory {
		return "", "", false
	}
	return reqType, reqID, true
}
