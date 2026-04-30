package auth

import (
	"context"
	"log"
	"net/http"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Warehouse Ops Scope Middleware ────────────────────────────────────────────
//
// For WAREHOUSE role endpoints (warehouse portal / mobile apps).
// The WarehouseID is always enforced from the JWT — warehouse admins/staff
// can only operate within their assigned warehouse.
//
// The middleware also resolves the SupplierID from the WarehouseStaff row
// (since WAREHOUSE JWTs may not carry SupplierID directly).

type warehouseOpsKey string

const WarehouseOpsKey = warehouseOpsKey("warehouse_ops")

// WarehouseOps contains the resolved scope for a warehouse-role user.
type WarehouseOps struct {
	WarehouseID   string
	SupplierID    string
	UserID        string
	WarehouseRole string // WAREHOUSE_ADMIN | WAREHOUSE_STAFF | PAYLOADER
}

// GetWarehouseOps extracts the WarehouseOps from the request context.
func GetWarehouseOps(ctx context.Context) *WarehouseOps {
	s, _ := ctx.Value(WarehouseOpsKey).(*WarehouseOps)
	return s
}

// RequireWarehouseOpsScope is middleware for WAREHOUSE-role endpoints.
// Extracts WarehouseID from JWT, resolves SupplierID via WarehouseStaff lookup,
// and injects WarehouseOps into context. Rejects if no WarehouseID in JWT.
func RequireWarehouseOpsScope(spannerClient *spanner.Client, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(ClaimsContextKey).(*LabClaims)
		if !ok || claims == nil {
			http.Error(w, "Missing authentication context", http.StatusUnauthorized)
			return
		}

		if claims.Role != "WAREHOUSE" {
			http.Error(w, "Warehouse role required", http.StatusForbidden)
			return
		}

		if claims.WarehouseID == "" {
			log.Printf("[WAREHOUSE OPS] user %s has empty WarehouseID in JWT", claims.UserID)
			http.Error(w, "Warehouse scope missing from token", http.StatusForbidden)
			return
		}

		ops := &WarehouseOps{
			WarehouseID:   claims.WarehouseID,
			UserID:        claims.UserID,
			WarehouseRole: claims.WarehouseRole,
		}

		// Resolve SupplierID from WarehouseStaff table
		stmt := spanner.Statement{
			SQL:    `SELECT SupplierId FROM WarehouseStaff WHERE WorkerId = @uid LIMIT 1`,
			Params: map[string]interface{}{"uid": claims.UserID},
		}
		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err == iterator.Done {
			log.Printf("[WAREHOUSE OPS] no staff record for user %s", claims.UserID)
			http.Error(w, "Warehouse staff record not found", http.StatusForbidden)
			return
		}
		if err != nil {
			log.Printf("[WAREHOUSE OPS] spanner error resolving supplier: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if err := row.Columns(&ops.SupplierID); err != nil {
			log.Printf("[WAREHOUSE OPS] column parse error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), WarehouseOpsKey, ops)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RequireWarehouseAdmin is a guard that rejects non-WAREHOUSE_ADMIN roles.
// Must be placed AFTER RequireWarehouseOpsScope.
func RequireWarehouseAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ops := GetWarehouseOps(r.Context())
		if ops == nil || ops.WarehouseRole != "WAREHOUSE_ADMIN" {
			http.Error(w, `{"error":"warehouse admin privileges required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// ─── Query Helpers ─────────────────────────────────────────────────────────────

// AppendWarehouseOpsFilter adds WarehouseId + SupplierId filter to a raw SQL string.
func AppendWarehouseOpsFilter(ctx context.Context, sql string, params map[string]interface{}, tableAlias string) (string, map[string]interface{}) {
	ops := GetWarehouseOps(ctx)
	if ops == nil {
		return sql, params
	}
	if params == nil {
		params = make(map[string]interface{})
	}
	sql += " AND " + tableAlias + ".WarehouseId = @whOpsWarehouseId"
	params["whOpsWarehouseId"] = ops.WarehouseID
	sql += " AND " + tableAlias + ".SupplierId = @whOpsSupplierId"
	params["whOpsSupplierId"] = ops.SupplierID
	return sql, params
}

// AppendWarehouseOpsFilterStmt is the spanner.Statement variant.
func AppendWarehouseOpsFilterStmt(ctx context.Context, stmt spanner.Statement, tableAlias string) spanner.Statement {
	ops := GetWarehouseOps(ctx)
	if ops == nil {
		return stmt
	}
	if stmt.Params == nil {
		stmt.Params = make(map[string]interface{})
	}
	stmt.SQL += " AND " + tableAlias + ".WarehouseId = @whOpsWarehouseId"
	stmt.Params["whOpsWarehouseId"] = ops.WarehouseID
	stmt.SQL += " AND " + tableAlias + ".SupplierId = @whOpsSupplierId"
	stmt.Params["whOpsSupplierId"] = ops.SupplierID
	return stmt
}
