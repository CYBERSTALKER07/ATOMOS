// Package exportpackroutes mounts ATOMOS M1.10 auditor/bank/tax export pack HTTP endpoints onto chi.
package exportpackroutes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/exportpacks"
)

// Deps holds runtime dependencies required by the M1.10 export pack route group.
type Deps struct {
	Service         *exportpacks.Service
	JWTSecret       string
	AllowAuthBypass bool
}

type ctxKeyOpaqueToken struct{}

// RegisterRoutes mounts POST/GET /v1/finance/packs and GET /v1/finance/packs/{packID}/download onto r.
func RegisterRoutes(r chi.Router, d Deps) {
	if r == nil || d.Service == nil {
		return
	}

	roleGuard := auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin)
	createPerm := auth.RequireAnySupplierPerm(
		auth.PermReconManage,
		auth.PermJournalPost,
		auth.PermJournalApprove,
		auth.PermPeriodClose,
		auth.PermAuditManage,
		auth.PermIAMManage,
	)
	readPerm := auth.RequireAnySupplierPerm(
		auth.PermAuditRead,
		auth.PermReconManage,
		auth.PermJournalPost,
		auth.PermJournalApprove,
		auth.PermPeriodClose,
		auth.PermAuditManage,
		auth.PermIAMManage,
	)

	mountProtected := func(gr chi.Router) {
		gr.With(auth.RejectAuditorOrAgentMutation(false), roleGuard, createPerm).
			Post("/v1/finance/packs", d.Service.HandleCreatePack)
		gr.With(roleGuard, readPerm).
			Get("/v1/finance/packs", d.Service.HandleListPacks)
		gr.With(roleGuard, readPerm).
			Get("/v1/finance/packs/{packID}", d.Service.HandleGetPack)
	}

	r.Group(func(gr chi.Router) {
		if strings.TrimSpace(d.JWTSecret) != "" {
			gr.Use(attachPackAuth(d.JWTSecret))
		}
		gr.Use(auth.RequireAuditorActive)
		gr.Use(auth.AttachTenantFromClaims)
		auth.ProtectMutations(gr, auth.MutationGuardConfig{
			AllowBypass: d.AllowAuthBypass,
		}, mountProtected)
		gr.With(downloadAuthGuard(roleGuard, readPerm)).
			Get("/v1/finance/packs/{packID}/download", d.Service.HandleDownloadPack)
	})
}

func attachPackAuth(secret string) func(http.Handler) http.Handler {
	cookieMW := auth.CookieAuth(secret)
	keyring := auth.NewKeyring(secret)
	return func(next http.Handler) http.Handler {
		return cookieMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := auth.FromContext(r.Context()); !ok && !auth.IsAuditorExpiredInContext(r.Context()) &&
				r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/download") {
				if qTok := strings.TrimSpace(r.URL.Query().Get("token")); qTok != "" {
					if strings.Contains(qTok, ".") {
						// Never accept session JWTs in query parameters; only report auditor_access_expired if expired.
						if _, err := auth.ParseTokenWithKeyring(qTok, keyring); errors.Is(err, auth.ErrAuditorAccessExpired) {
							r = r.WithContext(auth.WithAuditorExpired(r.Context()))
						}
					} else {
						// Carry opaque DownloadToken into downloadAuthGuard -> HandleDownloadPack (constant-time verify).
						r = r.WithContext(context.WithValue(r.Context(), ctxKeyOpaqueToken{}, true))
					}
				}
			}
			// Fail closed on Bearer/cookie revocation check (JTI + Subject user revocation).
			if claims, ok := auth.FromContext(r.Context()); ok && auth.IsTokenRevoked(r.Context(), claims) {
				http.Error(w, `{"error":"token_revoked","code":"token_revoked"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}

func downloadAuthGuard(roleGuard, readPerm func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		guarded := roleGuard(readPerm(next))
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := auth.FromContext(r.Context()); !ok {
				if hasOpaque, _ := r.Context().Value(ctxKeyOpaqueToken{}).(bool); hasOpaque {
					next.ServeHTTP(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "unauthorized",
					"code":    "unauthorized",
					"message": "authentication or valid download token required",
				})
				return
			}
			guarded.ServeHTTP(w, r)
		})
	}
}
