package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
)

var (
	errFactoryScopeMissing     = errors.New("factory scope missing from token")
	errNoLinkedWarehouses      = errors.New("no linked warehouses for factory")
	errWarehouseScopeViolation = errors.New("warehouse scope violation")
	errWarehouseIDRequired     = errors.New("warehouse_id required for this factory scope")
)

// RequireReplenishmentInsightsScope gates the shared replenishment insights path for
// warehouse and factory role rows without registering duplicate chi routes.
func RequireReplenishmentInsightsScope(spannerClient *spanner.Client) func(http.Handler) http.Handler {
	resolve := factoryWarehouseResolver(nil)
	if spannerClient != nil {
		resolve = spannerFactoryWarehouseResolver(spannerClient)
	}
	return func(next http.Handler) http.Handler {
		warehouseOps := RequireWarehouseOpsScope(next)
		adminScope := RequireWarehouseScopeWithClient(spannerClient)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := FromContext(r.Context())
			if !ok {
				writeScopeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			switch claims.Role {
			case RoleWarehouse, RoleWarehouseAdmin:
				warehouseOps.ServeHTTP(w, r)
				return
			case RoleFactory, RoleFactoryAdmin:
				ctx, err := injectFactoryInsightScope(r.Context(), &claims, resolve, r.URL.Query().Get("warehouse_id"))
				if err != nil {
					writeFactoryInsightScopeError(w, err)
					return
				}
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			case RoleAdmin:
				adminScope.ServeHTTP(w, r)
				return
			default:
				writeScopeError(w, http.StatusForbidden, "forbidden")
			}
		})
	}
}

func injectFactoryInsightScope(
	ctx context.Context,
	claims *Claims,
	resolve factoryWarehouseResolver,
	queryWarehouseID string,
) (context.Context, error) {
	if resolve == nil {
		return ctx, nil
	}
	factoryID := claims.HomeNodeID
	if claims.HomeNodeType != HomeNodeFactory || factoryID == "" {
		return ctx, errFactoryScopeMissing
	}
	supplierID := claims.SupplierID
	if supplierID == "" {
		if sid, ok := ResolveSupplierID(ctx); ok {
			supplierID = sid
		}
	}
	allowed, err := resolve(ctx, supplierID, factoryID)
	if err != nil {
		return ctx, err
	}
	if len(allowed) == 0 {
		return ctx, errNoLinkedWarehouses
	}
	scope := &WarehouseScope{SupplierID: supplierID}
	queryWarehouseID = strings.TrimSpace(queryWarehouseID)
	if queryWarehouseID != "" {
		if _, ok := allowed[queryWarehouseID]; !ok {
			return ctx, errWarehouseScopeViolation
		}
		scope.WarehouseID = queryWarehouseID
		return context.WithValue(ctx, WarehouseScopeKey, scope), nil
	}
	if len(allowed) != 1 {
		return ctx, errWarehouseIDRequired
	}
	for warehouseID := range allowed {
		scope.WarehouseID = warehouseID
		break
	}
	return context.WithValue(ctx, WarehouseScopeKey, scope), nil
}

func writeFactoryInsightScopeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errFactoryScopeMissing),
		errors.Is(err, errNoLinkedWarehouses),
		errors.Is(err, errWarehouseScopeViolation):
		writeScopeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, errWarehouseIDRequired):
		writeScopeError(w, http.StatusBadRequest, err.Error())
	default:
		writeScopeError(w, http.StatusInternalServerError, "failed to resolve warehouse scope")
	}
}
