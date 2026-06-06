package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type warehouseScopeKey struct{}

var WarehouseScopeKey = warehouseScopeKey{}

// WarehouseScope is the supplier-portal warehouse filter for ADMIN callers.
type WarehouseScope struct {
	WarehouseID string
	SupplierID  string
	IsPinned    bool
}

// GetWarehouseScope returns supplier warehouse scope when set.
func GetWarehouseScope(ctx context.Context) *WarehouseScope {
	s, _ := ctx.Value(WarehouseScopeKey).(*WarehouseScope)
	return s
}

type factoryWarehouseResolver func(ctx context.Context, supplierID, factoryID string) (map[string]struct{}, error)

// RequireWarehouseScope resolves warehouse scope for supplier ADMIN endpoints.
func RequireWarehouseScope(next http.Handler) http.Handler {
	return requireWarehouseScope(nil, next)
}

// RequireWarehouseScopeWithClient enforces FACTORY_ADMIN linkage via Spanner.
func RequireWarehouseScopeWithClient(spannerClient *spanner.Client) func(http.Handler) http.Handler {
	resolver := factoryWarehouseResolver(nil)
	if spannerClient != nil {
		resolver = spannerFactoryWarehouseResolver(spannerClient)
	}
	return func(next http.Handler) http.Handler {
		return requireWarehouseScope(resolver, next)
	}
}

func requireWarehouseScope(resolveFactoryWarehouses factoryWarehouseResolver, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := FromContext(r.Context())
		if !ok {
			writeScopeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if claims.Role != RoleAdmin {
			next.ServeHTTP(w, r)
			return
		}

		supplierID := claims.SupplierID
		if supplierID == "" {
			if sid, ok := ResolveSupplierID(r.Context()); ok {
				supplierID = sid
			}
		}

		scope := &WarehouseScope{SupplierID: supplierID}
		qsWarehouseID := r.URL.Query().Get("warehouse_id")

		switch claims.SupplierRole {
		case RoleWarehouseAdmin:
			scope.IsPinned = true
			scope.WarehouseID = claims.HomeNodeID
			if scope.WarehouseID == "" {
				writeScopeError(w, http.StatusForbidden, "warehouse admin must have assigned warehouse")
				return
			}
			if qsWarehouseID != "" && qsWarehouseID != scope.WarehouseID {
				writeScopeError(w, http.StatusForbidden, "access denied: warehouse scope violation")
				return
			}

		case RoleFactoryAdmin:
			factoryID := claims.HomeNodeID
			if claims.HomeNodeType != HomeNodeFactory || factoryID == "" {
				writeScopeError(w, http.StatusForbidden, "factory admin must have assigned factory")
				return
			}
			if resolveFactoryWarehouses == nil {
				writeScopeError(w, http.StatusInternalServerError, "warehouse scope resolution unavailable")
				return
			}
			allowed, err := resolveFactoryWarehouses(r.Context(), supplierID, factoryID)
			if err != nil {
				slog.ErrorContext(r.Context(), "factory admin warehouse scope resolve failed",
					"err", err, "factory_id", factoryID)
				writeScopeError(w, http.StatusInternalServerError, "failed to resolve warehouse scope")
				return
			}
			if len(allowed) == 0 {
				writeScopeError(w, http.StatusForbidden, "access denied: no linked warehouses for factory")
				return
			}
			if qsWarehouseID != "" {
				if _, ok := allowed[qsWarehouseID]; !ok {
					writeScopeError(w, http.StatusForbidden, "access denied: warehouse scope violation")
					return
				}
				scope.WarehouseID = qsWarehouseID
				break
			}
			if len(allowed) != 1 {
				writeScopeError(w, http.StatusBadRequest, "warehouse_id is required for this factory scope")
				return
			}
			for warehouseID := range allowed {
				scope.WarehouseID = warehouseID
				break
			}

		default:
			scope.WarehouseID = qsWarehouseID
		}

		ctx := context.WithValue(r.Context(), WarehouseScopeKey, scope)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func spannerFactoryWarehouseResolver(spannerClient *spanner.Client) factoryWarehouseResolver {
	return func(ctx context.Context, supplierID, factoryID string) (map[string]struct{}, error) {
		if supplierID == "" || factoryID == "" {
			return nil, fmt.Errorf("supplier and factory id required")
		}
		stmt := spanner.Statement{
			SQL: `SELECT WarehouseId FROM Warehouses
				WHERE SupplierId = @supplierId AND IsActive = TRUE
				AND (PrimaryFactoryId = @factoryId OR SecondaryFactoryId = @factoryId)`,
			Params: map[string]any{
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
				return nil, err
			}
			if warehouseID != "" {
				allowed[warehouseID] = struct{}{}
			}
		}
		return allowed, nil
	}
}

// EffectiveWarehouseID returns warehouse scope for supplier reads (ops scope takes precedence).
func EffectiveWarehouseID(ctx context.Context) string {
	if ops := GetWarehouseOps(ctx); ops != nil && ops.WarehouseID != "" {
		return ops.WarehouseID
	}
	if scope := GetWarehouseScope(ctx); scope != nil {
		return scope.WarehouseID
	}
	return ""
}
