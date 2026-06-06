package auth

import (
	"context"
	"log/slog"
	"net/http"
)

type warehouseOpsKey struct{}

var WarehouseOpsKey = warehouseOpsKey{}

// WarehouseOps is the resolved scope for warehouse-role operational endpoints.
type WarehouseOps struct {
	WarehouseID string
	SupplierID  string
	Subject     string
}

// GetWarehouseOps returns warehouse ops scope from context.
func GetWarehouseOps(ctx context.Context) *WarehouseOps {
	s, _ := ctx.Value(WarehouseOpsKey).(*WarehouseOps)
	return s
}

// RequireWarehouseOpsScope pins warehouse staff to JWT home node and rejects query overrides.
func RequireWarehouseOpsScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := FromContext(r.Context())
		if !ok {
			writeScopeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		// Global supplier ADMIN (CEO) may call warehouse ops compat routes without a pinned home node.
		// Order mutations resolve scope from the target order in the service layer.
		if claims.Role == RoleAdmin && claims.SupplierRole != RoleWarehouseAdmin {
			next.ServeHTTP(w, r)
			return
		}

		allowedRole := claims.Role == RoleWarehouse ||
			claims.Role == RoleWarehouseAdmin ||
			(claims.Role == RoleAdmin && claims.SupplierRole == RoleWarehouseAdmin)
		if !allowedRole {
			writeScopeError(w, http.StatusForbidden, "warehouse role required")
			return
		}

		warehouseID := claims.HomeNodeID
		if claims.HomeNodeType != HomeNodeWarehouse || warehouseID == "" {
			slog.WarnContext(r.Context(), "warehouse ops missing home node",
				"subject", claims.Subject, "role", claims.Role)
			writeScopeError(w, http.StatusForbidden, "warehouse scope missing from token")
			return
		}

		qsWarehouse := r.URL.Query().Get("warehouse_id")
		if qsWarehouse != "" && qsWarehouse != warehouseID {
			slog.WarnContext(r.Context(), "warehouse ops scope violation",
				"subject", claims.Subject, "jwt_warehouse", warehouseID, "query_warehouse", qsWarehouse)
			writeScopeError(w, http.StatusForbidden, "access denied: warehouse scope violation")
			return
		}

		supplierID := claims.SupplierID
		if supplierID == "" {
			if sid, ok := ResolveSupplierID(r.Context()); ok {
				supplierID = sid
			}
		}

		ops := &WarehouseOps{
			WarehouseID: warehouseID,
			SupplierID:  supplierID,
			Subject:     claims.Subject,
		}
		ctx := context.WithValue(r.Context(), WarehouseOpsKey, ops)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// EffectiveWarehouseOpsID returns the warehouse id from ops scope.
func EffectiveWarehouseOpsID(ctx context.Context) string {
	ops := GetWarehouseOps(ctx)
	if ops == nil {
		return ""
	}
	return ops.WarehouseID
}
