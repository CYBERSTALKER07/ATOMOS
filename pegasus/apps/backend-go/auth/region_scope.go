package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type regionScopeKey string

const RegionScopeKey = regionScopeKey("region_scope")

// RegionScope contains the resolved region filter parameters for role-scoped APIs.
type RegionScope struct {
	RegionID          string
	SupplierID        string
	WarehouseID       string
	RequestedRegionID string
}

// GetRegionScope extracts RegionScope from request context.
func GetRegionScope(ctx context.Context) *RegionScope {
	s, _ := ctx.Value(RegionScopeKey).(*RegionScope)
	return s
}

// EffectiveRegionID returns the resolved region from context, if available.
func EffectiveRegionID(ctx context.Context) string {
	s := GetRegionScope(ctx)
	if s == nil {
		return ""
	}
	return s.RegionID
}

// RequireRegionScopeWithClient resolves role-bound region scope and enforces
// query compatibility (region_id must match the caller's effective region).
func RequireRegionScopeWithClient(spannerClient *spanner.Client) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			claims, ok := r.Context().Value(ClaimsContextKey).(*PegasusClaims)
			if !ok || claims == nil {
				http.Error(w, "Missing authentication context", http.StatusUnauthorized)
				return
			}

			if claims.Role != "SUPPLIER" && claims.Role != "ADMIN" && claims.Role != "WAREHOUSE" {
				next.ServeHTTP(w, r)
				return
			}

			requestedRegionID := strings.TrimSpace(r.URL.Query().Get("region_id"))
			scope := &RegionScope{RequestedRegionID: requestedRegionID}
			if requestedRegionID == "" {
				ctx := context.WithValue(r.Context(), RegionScopeKey, scope)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if spannerClient == nil {
				http.Error(w, "Region scope resolution unavailable", http.StatusInternalServerError)
				return
			}

			var err error
			switch claims.Role {
			case "SUPPLIER", "ADMIN":
				scope.SupplierID = strings.TrimSpace(claims.ResolveSupplierID())
				if scope.SupplierID == "" {
					http.Error(w, "supplier scope missing from token", http.StatusForbidden)
					return
				}
				scope.RegionID, err = resolveSupplierRegionID(r.Context(), spannerClient, scope.SupplierID)
			case "WAREHOUSE":
				scope.WarehouseID = strings.TrimSpace(claims.WarehouseID)
				if scope.WarehouseID == "" {
					http.Error(w, "warehouse scope missing from token", http.StatusForbidden)
					return
				}
				scope.RegionID, scope.SupplierID, err = resolveWarehouseRegionID(r.Context(), spannerClient, scope.WarehouseID)
			}
			if err != nil {
				log.Printf("[AUTH] region scope resolution failed: role=%s user=%s err=%v", claims.Role, claims.UserID, err)
				http.Error(w, "failed to resolve region scope", http.StatusInternalServerError)
				return
			}

			if requestedRegionID != "" {
				if scope.RegionID == "" {
					http.Error(w, "region scope is unavailable for this tenant", http.StatusForbidden)
					return
				}
				if !strings.EqualFold(requestedRegionID, scope.RegionID) {
					log.Printf("[AUTH] region scope violation: role=%s user=%s requested=%s effective=%s", claims.Role, claims.UserID, requestedRegionID, scope.RegionID)
					http.Error(w, "Access denied: region scope violation", http.StatusForbidden)
					return
				}
				scope.RegionID = requestedRegionID
			}

			ctx := context.WithValue(r.Context(), RegionScopeKey, scope)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	}
}

func resolveSupplierRegionID(ctx context.Context, spannerClient *spanner.Client, supplierID string) (string, error) {
	row, err := spannerClient.Single().ReadRow(ctx, "Suppliers", spanner.Key{supplierID}, []string{"RegionId", "CountryCode"})
	if err != nil {
		return "", fmt.Errorf("read supplier region: %w", err)
	}

	var regionID spanner.NullString
	var countryCode spanner.NullString
	if err := row.Columns(&regionID, &countryCode); err != nil {
		return "", fmt.Errorf("decode supplier region: %w", err)
	}

	if regionID.Valid && strings.TrimSpace(regionID.StringVal) != "" {
		return strings.TrimSpace(regionID.StringVal), nil
	}

	if !countryCode.Valid || strings.TrimSpace(countryCode.StringVal) == "" {
		return "", nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT RegionId
		      FROM Regions
		      WHERE CountryCode = @countryCode
		        AND IsActive = TRUE
		        AND IsDefault = TRUE
		      LIMIT 1`,
		Params: map[string]interface{}{
			"countryCode": strings.ToUpper(strings.TrimSpace(countryCode.StringVal)),
		},
	}

	iter := spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	regionRow, err := iter.Next()
	if err == iterator.Done {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("query default region by country: %w", err)
	}

	var defaultRegionID string
	if err := regionRow.Columns(&defaultRegionID); err != nil {
		return "", fmt.Errorf("decode default region: %w", err)
	}
	return strings.TrimSpace(defaultRegionID), nil
}

func resolveWarehouseRegionID(ctx context.Context, spannerClient *spanner.Client, warehouseID string) (string, string, error) {
	row, err := spannerClient.Single().ReadRow(ctx, "Warehouses", spanner.Key{warehouseID}, []string{"RegionId", "SupplierId"})
	if err != nil {
		return "", "", fmt.Errorf("read warehouse region: %w", err)
	}

	var regionID spanner.NullString
	var supplierID string
	if err := row.Columns(&regionID, &supplierID); err != nil {
		return "", "", fmt.Errorf("decode warehouse region: %w", err)
	}
	if regionID.Valid && strings.TrimSpace(regionID.StringVal) != "" {
		return strings.TrimSpace(regionID.StringVal), strings.TrimSpace(supplierID), nil
	}

	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return "", "", nil
	}

	fallbackRegionID, err := resolveSupplierRegionID(ctx, spannerClient, supplierID)
	if err != nil {
		return "", supplierID, err
	}
	return fallbackRegionID, supplierID, nil
}

// AppendRegionFilter adds a RegionId filter clause to SQL if a region scope is active.
func AppendRegionFilter(ctx context.Context, sql string, params map[string]interface{}, tableAlias string) (string, map[string]interface{}) {
	scope := GetRegionScope(ctx)
	if scope == nil || scope.RegionID == "" {
		return sql, params
	}

	sql += " AND " + tableAlias + ".RegionId = @regionId"
	if params == nil {
		params = make(map[string]interface{})
	}
	params["regionId"] = scope.RegionID
	return sql, params
}

// AppendRegionFilterStmt is a convenience wrapper for spanner.Statement.
func AppendRegionFilterStmt(ctx context.Context, stmt spanner.Statement, tableAlias string) spanner.Statement {
	scope := GetRegionScope(ctx)
	if scope == nil || scope.RegionID == "" {
		return stmt
	}

	stmt.SQL += " AND " + tableAlias + ".RegionId = @regionId"
	if stmt.Params == nil {
		stmt.Params = make(map[string]interface{})
	}
	stmt.Params["regionId"] = scope.RegionID
	return stmt
}
