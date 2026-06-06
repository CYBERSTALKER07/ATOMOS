package auth

import (
	"context"
	"log/slog"
	"net/http"
)

type factoryScopeKey struct{}

var FactoryScopeKey = factoryScopeKey{}

// FactoryScope is the resolved factory filter for factory-role endpoints.
type FactoryScope struct {
	FactoryID  string
	SupplierID string
	Subject    string
}

// GetFactoryScope returns factory scope injected by RequireFactoryScope.
func GetFactoryScope(ctx context.Context) *FactoryScope {
	s, _ := ctx.Value(FactoryScopeKey).(*FactoryScope)
	return s
}

// RequireFactoryScope pins factory staff to their JWT home node (factory id).
func RequireFactoryScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := FromContext(r.Context())
		if !ok {
			writeScopeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if claims.Role != RoleFactory && claims.Role != RoleFactoryAdmin {
			next.ServeHTTP(w, r)
			return
		}

		factoryID := claims.HomeNodeID
		if claims.HomeNodeType != HomeNodeFactory || factoryID == "" {
			slog.WarnContext(r.Context(), "factory scope missing home node",
				"subject", claims.Subject, "role", claims.Role)
			writeScopeError(w, http.StatusForbidden, "factory scope missing from token")
			return
		}

		qsFactory := r.URL.Query().Get("factory_id")
		if qsFactory != "" && qsFactory != factoryID {
			slog.WarnContext(r.Context(), "factory scope violation",
				"subject", claims.Subject, "jwt_factory", factoryID, "query_factory", qsFactory)
			writeScopeError(w, http.StatusForbidden, "access denied: factory scope violation")
			return
		}

		scope := &FactoryScope{
			FactoryID:  factoryID,
			SupplierID: claims.SupplierID,
			Subject:    claims.Subject,
		}
		ctx := context.WithValue(r.Context(), FactoryScopeKey, scope)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// EffectiveFactoryID returns the active factory id from scope, if any.
func EffectiveFactoryID(ctx context.Context) string {
	scope := GetFactoryScope(ctx)
	if scope == nil {
		return ""
	}
	return scope.FactoryID
}
