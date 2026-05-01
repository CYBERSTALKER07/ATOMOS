package auth

import (
	"context"
	"log"
	"net/http"

	"cloud.google.com/go/spanner"
)

// ─── Factory Scope Middleware ─────────────────────────────────────────────────
//
// For FACTORY role endpoints, this middleware resolves the effective
// FactoryID from the JWT claims:
//
//   FACTORY_ADMIN:
//     - FactoryID is fixed from JWT → silently enforced on all queries
//
//   FACTORY_PAYLOADER:
//     - FactoryID is fixed from JWT → silently enforced on all queries
//     - More restricted action set (loading bay only)
//
// The effective scope is injected into context as FactoryScopeKey.

type factoryScopeKey string

const FactoryScopeKey = factoryScopeKey("factory_scope")

// FactoryScope contains the resolved factory filtering parameters.
type FactoryScope struct {
	// FactoryID is the active factory filter.
	FactoryID string
	// IsPayloader is true if the user is a FACTORY_PAYLOADER (restricted actions).
	IsPayloader bool
	// SupplierId is the supplier identity from the JWT (factory owner).
	SupplierId string
}

// GetFactoryScope extracts the FactoryScope from the request context.
// Returns nil if not set (non-factory endpoints).
func GetFactoryScope(ctx context.Context) *FactoryScope {
	s, _ := ctx.Value(FactoryScopeKey).(*FactoryScope)
	return s
}

// RequireFactoryScope is middleware that resolves the effective factory scope
// from JWT claims and injects it into context.
// Must be placed AFTER RequireRole for FACTORY endpoints.
func RequireFactoryScope(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(ClaimsContextKey).(*PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, "Missing authentication context", http.StatusUnauthorized)
			return
		}

		// Only apply factory scoping to FACTORY role
		if claims.Role != "FACTORY" {
			// Non-factory roles pass through without factory scope
			next.ServeHTTP(w, r)
			return
		}

		if claims.FactoryID == "" {
			log.Printf("[AUTH] FACTORY user %s has empty FactoryID in JWT — rejecting", claims.UserID)
			http.Error(w, "Factory staff must have assigned factory", http.StatusForbidden)
			return
		}

		scope := &FactoryScope{
			FactoryID:   claims.FactoryID,
			IsPayloader: claims.FactoryRole == "FACTORY_PAYLOADER",
			SupplierId:  "", // populated below from Spanner if needed, or from a claim extension
		}

		ctx := context.WithValue(r.Context(), FactoryScopeKey, scope)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// ─── Spanner Query Helpers ─────────────────────────────────────────────────────

// AppendFactoryFilter adds a FactoryId filter clause to SQL if the scope is active.
func AppendFactoryFilter(ctx context.Context, sql string, params map[string]interface{}, tableAlias string) (string, map[string]interface{}) {
	scope := GetFactoryScope(ctx)
	if scope == nil || scope.FactoryID == "" {
		return sql, params
	}

	sql += " AND " + tableAlias + ".FactoryId = @factoryId"
	if params == nil {
		params = make(map[string]interface{})
	}
	params["factoryId"] = scope.FactoryID
	return sql, params
}

// AppendFactoryFilterStmt is a convenience wrapper that works with spanner.Statement.
func AppendFactoryFilterStmt(ctx context.Context, stmt spanner.Statement, tableAlias string) spanner.Statement {
	scope := GetFactoryScope(ctx)
	if scope == nil || scope.FactoryID == "" {
		return stmt
	}

	stmt.SQL += " AND " + tableAlias + ".FactoryId = @factoryId"
	if stmt.Params == nil {
		stmt.Params = make(map[string]interface{})
	}
	stmt.Params["factoryId"] = scope.FactoryID
	return stmt
}

// EffectiveFactoryID returns the factory ID from the scope, or empty string.
func EffectiveFactoryID(ctx context.Context) string {
	scope := GetFactoryScope(ctx)
	if scope == nil {
		return ""
	}
	return scope.FactoryID
}

// MustFactoryID returns the factory ID from scope, writing a 400 error
// if no factory scope is active. Use for endpoints that REQUIRE a factory selection.
func MustFactoryID(w http.ResponseWriter, ctx context.Context) (string, bool) {
	fID := EffectiveFactoryID(ctx)
	if fID == "" {
		http.Error(w, "factory scope is required for this operation", http.StatusBadRequest)
		return "", false
	}
	return fID, true
}
