package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Warehouse Scope Middleware ────────────────────────────────────────────────
//
// For SUPPLIER/ADMIN role endpoints, this middleware resolves the effective
// WarehouseID from the JWT claims:
//
//   GLOBAL_ADMIN (SupplierRole="GLOBAL_ADMIN" or empty):
//     - WarehouseID is empty → all warehouses visible
//     - Query parameter ?warehouse_id=X → scoped to X (voluntary scope)
//
//   NODE_ADMIN (SupplierRole="NODE_ADMIN"):
//     - WarehouseID is fixed from JWT → silently enforced on all queries
//     - Query parameter ?warehouse_id=X → REJECTED if X != JWT warehouse_id
//
//   FACTORY_ADMIN (SupplierRole="FACTORY_ADMIN"):
//     - WarehouseID must belong to warehouses linked to claims.FactoryID
//       via Warehouses.PrimaryFactoryId / SecondaryFactoryId
//     - Query parameter ?warehouse_id=X → REJECTED if X is outside linked set
//     - If one linked warehouse exists and query is omitted, scope auto-pins
//     - If multiple linked warehouses exist and query is omitted, request is rejected
//
// The effective scope is injected into context as WarehouseScopeKey.

type factoryWarehouseResolver func(ctx context.Context, supplierID, factoryID string) (map[string]struct{}, error)

type warehouseScopeKey string

const WarehouseScopeKey = warehouseScopeKey("warehouse_scope")

// WarehouseScope contains the resolved warehouse filtering parameters.
type WarehouseScope struct {
	// WarehouseID is the active warehouse filter. Empty = all warehouses (GLOBAL_ADMIN unscoped).
	WarehouseID string
	// IsNodeAdmin is true if the user is a NODE_ADMIN (forced scope, cannot be overridden).
	IsNodeAdmin bool
	// SupplierId is the supplier identity from the JWT.
	SupplierId string
}

// GetWarehouseScope extracts the WarehouseScope from the request context.
// Returns nil if not set (non-supplier endpoints).
func GetWarehouseScope(ctx context.Context) *WarehouseScope {
	s, _ := ctx.Value(WarehouseScopeKey).(*WarehouseScope)
	return s
}

// RequireWarehouseScope is middleware that resolves the effective warehouse scope
// from JWT claims and optional query parameter, then injects it into context.
// Must be placed AFTER RequireRole for SUPPLIER/ADMIN endpoints.
func RequireWarehouseScope(next http.HandlerFunc) http.HandlerFunc {
	return requireWarehouseScope(nil, next)
}

// RequireWarehouseScopeWithClient is a spanner-aware variant of
// RequireWarehouseScope that enforces FACTORY_ADMIN warehouse linkage.
func RequireWarehouseScopeWithClient(spannerClient *spanner.Client) func(http.HandlerFunc) http.HandlerFunc {
	resolver := factoryWarehouseResolver(nil)
	if spannerClient != nil {
		resolver = spannerFactoryWarehouseResolver(spannerClient)
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return requireWarehouseScope(resolver, next)
	}
}

func requireWarehouseScope(resolveFactoryWarehouses factoryWarehouseResolver, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(ClaimsContextKey).(*PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, "Missing authentication context", http.StatusUnauthorized)
			return
		}

		// Only apply warehouse scoping to SUPPLIER/ADMIN roles
		if claims.Role != "SUPPLIER" && claims.Role != "ADMIN" {
			// Non-supplier roles pass through without warehouse scope
			next.ServeHTTP(w, r)
			return
		}

		scope := &WarehouseScope{
			SupplierId: claims.ResolveSupplierID(),
		}

		// Determine effective warehouse scope
		qsWarehouseID := r.URL.Query().Get("warehouse_id")

		switch claims.SupplierRole {
		case "NODE_ADMIN":
			// NODE_ADMIN: warehouse scope is enforced from JWT
			scope.IsNodeAdmin = true
			scope.WarehouseID = claims.WarehouseID

			if scope.WarehouseID == "" {
				log.Printf("[AUTH] NODE_ADMIN %s has empty WarehouseID in JWT — rejecting", claims.UserID)
				http.Error(w, "Node admin must have assigned warehouse", http.StatusForbidden)
				return
			}

			// Reject if query param tries to override the enforced scope
			if qsWarehouseID != "" && qsWarehouseID != scope.WarehouseID {
				log.Printf("[AUTH] NODE_ADMIN %s attempted cross-warehouse access: jwt=%s qs=%s",
					claims.UserID, scope.WarehouseID, qsWarehouseID)
				http.Error(w, "Access denied: warehouse scope violation", http.StatusForbidden)
				return
			}

		case "FACTORY_ADMIN":
			if claims.FactoryID == "" {
				log.Printf("[AUTH] FACTORY_ADMIN %s has empty FactoryID in JWT — rejecting", claims.UserID)
				http.Error(w, "Factory admin must have assigned factory", http.StatusForbidden)
				return
			}
			if scope.SupplierId == "" {
				log.Printf("[AUTH] FACTORY_ADMIN %s has empty SupplierID in JWT — rejecting", claims.UserID)
				http.Error(w, "Factory admin must belong to a supplier", http.StatusForbidden)
				return
			}
			if resolveFactoryWarehouses == nil {
				log.Printf("[AUTH] FACTORY_ADMIN %s scope resolver unavailable", claims.UserID)
				http.Error(w, "Warehouse scope resolution unavailable", http.StatusInternalServerError)
				return
			}

			allowedWarehouses, err := resolveFactoryWarehouses(r.Context(), scope.SupplierId, claims.FactoryID)
			if err != nil {
				log.Printf("[AUTH] FACTORY_ADMIN %s warehouse scope resolve failed: %v", claims.UserID, err)
				http.Error(w, "Failed to resolve warehouse scope", http.StatusInternalServerError)
				return
			}
			if len(allowedWarehouses) == 0 {
				log.Printf("[AUTH] FACTORY_ADMIN %s has no linked warehouses for factory=%s", claims.UserID, claims.FactoryID)
				http.Error(w, "Access denied: no linked warehouses for factory scope", http.StatusForbidden)
				return
			}

			if qsWarehouseID != "" {
				if _, ok := allowedWarehouses[qsWarehouseID]; !ok {
					log.Printf("[AUTH] FACTORY_ADMIN %s attempted cross-warehouse access: factory=%s qs=%s", claims.UserID, claims.FactoryID, qsWarehouseID)
					http.Error(w, "Access denied: warehouse scope violation", http.StatusForbidden)
					return
				}
				scope.WarehouseID = qsWarehouseID
				break
			}

			if len(allowedWarehouses) != 1 {
				http.Error(w, "warehouse_id is required for this factory scope", http.StatusBadRequest)
				return
			}
			for warehouseID := range allowedWarehouses {
				scope.WarehouseID = warehouseID
				break
			}

		default:
			// GLOBAL_ADMIN: voluntary scoping via query parameter
			scope.IsNodeAdmin = false
			scope.WarehouseID = qsWarehouseID // empty = all warehouses
		}

		ctx := context.WithValue(r.Context(), WarehouseScopeKey, scope)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func spannerFactoryWarehouseResolver(spannerClient *spanner.Client) factoryWarehouseResolver {
	return func(ctx context.Context, supplierID, factoryID string) (map[string]struct{}, error) {
		if supplierID == "" {
			return nil, fmt.Errorf("supplier id is required for factory warehouse scope")
		}
		if factoryID == "" {
			return nil, fmt.Errorf("factory id is required for factory warehouse scope")
		}

		stmt := spanner.Statement{
			SQL: `SELECT WarehouseId
			      FROM Warehouses
			      WHERE SupplierId = @supplierId
			        AND IsActive = TRUE
			        AND (PrimaryFactoryId = @factoryId OR SecondaryFactoryId = @factoryId)`,
			Params: map[string]interface{}{
				"supplierId": supplierID,
				"factoryId":  factoryID,
			},
		}

		allowed := make(map[string]struct{})
		iter := spannerClient.Single().Query(ctx, stmt)
		defer iter.Stop()

		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("query linked warehouses: %w", err)
			}

			var warehouseID string
			if err := row.Columns(&warehouseID); err != nil {
				return nil, fmt.Errorf("decode linked warehouse id: %w", err)
			}
			if warehouseID != "" {
				allowed[warehouseID] = struct{}{}
			}
		}

		return allowed, nil
	}
}

// ─── Spanner Query Helpers ─────────────────────────────────────────────────────

// AppendWarehouseFilter adds a WarehouseId filter clause to SQL if the scope is active.
// Returns the updated SQL and params map. The caller's SQL should use a table alias
// that has a WarehouseId column.
//
// Usage:
//
//	sql := "SELECT ... FROM Orders o WHERE o.SupplierId = @supplierId"
//	params := map[string]interface{}{"supplierId": supplierID}
//	sql, params = auth.AppendWarehouseFilter(ctx, sql, params, "o")
func AppendWarehouseFilter(ctx context.Context, sql string, params map[string]interface{}, tableAlias string) (string, map[string]interface{}) {
	scope := GetWarehouseScope(ctx)
	if scope == nil || scope.WarehouseID == "" {
		return sql, params
	}

	sql += " AND (" + tableAlias + ".WarehouseId = @warehouseId OR (" + tableAlias + ".HomeNodeType = 'WAREHOUSE' AND " + tableAlias + ".HomeNodeId = @warehouseId))"
	if params == nil {
		params = make(map[string]interface{})
	}
	params["warehouseId"] = scope.WarehouseID
	return sql, params
}

// AppendWarehouseFilterStmt is a convenience wrapper that works with spanner.Statement.
func AppendWarehouseFilterStmt(ctx context.Context, stmt spanner.Statement, tableAlias string) spanner.Statement {
	scope := GetWarehouseScope(ctx)
	if scope == nil || scope.WarehouseID == "" {
		return stmt
	}

	stmt.SQL += " AND (" + tableAlias + ".WarehouseId = @warehouseId OR (" + tableAlias + ".HomeNodeType = 'WAREHOUSE' AND " + tableAlias + ".HomeNodeId = @warehouseId))"
	if stmt.Params == nil {
		stmt.Params = make(map[string]interface{})
	}
	stmt.Params["warehouseId"] = scope.WarehouseID
	return stmt
}

// EffectiveWarehouseID returns the warehouse ID from the scope, or empty string
// if the scope is not set or not filtered (GLOBAL_ADMIN viewing all).
func EffectiveWarehouseID(ctx context.Context) string {
	scope := GetWarehouseScope(ctx)
	if scope == nil {
		return ""
	}
	return scope.WarehouseID
}

// MustWarehouseID returns the warehouse ID from scope, writing a 400 error
// if no warehouse scope is active. Use for endpoints that REQUIRE a warehouse selection.
func MustWarehouseID(w http.ResponseWriter, ctx context.Context) (string, bool) {
	whID := EffectiveWarehouseID(ctx)
	if whID == "" {
		http.Error(w, "warehouse_id is required for this operation", http.StatusBadRequest)
		return "", false
	}
	return whID, true
}
