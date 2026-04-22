package analytics

import (
	"backend-go/auth"
)

// ApplyScopeFilter returns a Spanner WHERE clause fragment and params that
// restrict queries to the caller's RBAC scope.
//
// Scoping rules:
//   - GLOBAL_ADMIN: sees all data for their SupplierId.
//     If ?warehouse_id is provided, voluntarily narrows to that warehouse.
//   - NODE_ADMIN: sees only their assigned WarehouseId.
//
// The returned clause always starts with "AND" so it can be appended to an
// existing WHERE predicate. The caller must alias the supplier column as
// the first parameter (supplierCol) and optionally the warehouse column
// (warehouseCol). If warehouseCol is empty, warehouse scoping is skipped.
func ApplyScopeFilter(
	claims *auth.LabClaims,
	ws *auth.WarehouseScope,
	supplierCol string,
	warehouseCol string,
) (clause string, params map[string]interface{}) {
	params = make(map[string]interface{})

	// Supplier-level scope — every authenticated supplier user is scoped to
	// their own SupplierId.
	supplierID := ""
	if ws != nil {
		supplierID = ws.SupplierId
	}
	if supplierID == "" && claims != nil {
		supplierID = claims.ResolveSupplierID()
	}
	clause = " AND " + supplierCol + " = @_scopeSupplier"
	params["_scopeSupplier"] = supplierID

	// Warehouse-level scope — NODE_ADMIN always, GLOBAL_ADMIN only if explicit.
	if warehouseCol != "" && ws != nil && ws.WarehouseID != "" {
		clause += " AND " + warehouseCol + " = @_scopeWarehouse"
		params["_scopeWarehouse"] = ws.WarehouseID
	}

	return clause, params
}
