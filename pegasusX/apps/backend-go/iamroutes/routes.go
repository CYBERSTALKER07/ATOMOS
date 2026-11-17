// Package iamroutes mounts the ATOMOS M0.4 supplier RBAC & SoD HTTP endpoints onto chi.
package iamroutes

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/iam"
)

// Deps holds runtime dependencies required by the IAM route group.
type Deps struct {
	Service                      *iam.Service
	JWTSecret                    string
	JWTIssuer                    string
	SCIMBearerToken              string
	SCIMTenantID                 string
	SCIMTenantTokens             map[string]string
	AllowUnscopedStaticSCIMToken bool
	AllowAuthBypass              bool
}

// ParseSCIMTenantTokens parses a JSON object ({"tenant":"token"}) or CSV ("tenant=token,...")
// mapping tenant IDs to dedicated static SCIM bearer tokens.
func ParseSCIMTenantTokens(raw string) map[string]string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	out := make(map[string]string)
	if strings.HasPrefix(trimmed, "{") {
		var parsed map[string]string
		if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
			for tid, tok := range parsed {
				tid = strings.TrimSpace(tid)
				tok = strings.TrimSpace(tok)
				if tid != "" && tok != "" {
					out[tid] = tok
				}
			}
			if len(out) > 0 {
				return out
			}
		}
		return nil
	}
	for _, entry := range strings.Split(trimmed, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		var tid, tok string
		if k, v, ok := strings.Cut(entry, "="); ok {
			tid, tok = strings.TrimSpace(k), strings.TrimSpace(v)
		} else if k, v, ok := strings.Cut(entry, ":"); ok {
			tid, tok = strings.TrimSpace(k), strings.TrimSpace(v)
		}
		if tid != "" && tok != "" {
			out[tid] = tok
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// RegisterRoutes mounts the M0.4 IAM role, permission, SoD, M1.10 auditor-grant,
// Wave 4B custom roles, and SCIM 2.0 endpoints onto r.
func RegisterRoutes(r chi.Router, d Deps) {
	if r == nil || d.Service == nil {
		return
	}
	if strings.TrimSpace(d.JWTSecret) != "" {
		d.Service.SetJWTSecret(d.JWTSecret, d.JWTIssuer)
	}

	scimToken := strings.TrimSpace(d.SCIMBearerToken)
	if scimToken == "" {
		scimToken = strings.TrimSpace(os.Getenv("SCIM_BEARER_TOKEN"))
	}
	scimTenantID := strings.TrimSpace(d.SCIMTenantID)
	if scimTenantID == "" {
		scimTenantID = strings.TrimSpace(os.Getenv("SCIM_TENANT_ID"))
	}
	scimTenantTokens := make(map[string]string, len(d.SCIMTenantTokens))
	for tid, tok := range d.SCIMTenantTokens {
		tid = strings.TrimSpace(tid)
		tok = strings.TrimSpace(tok)
		if tid != "" && tok != "" {
			scimTenantTokens[tid] = tok
		}
	}
	if len(scimTenantTokens) == 0 {
		for tid, tok := range ParseSCIMTenantTokens(os.Getenv("SCIM_TENANT_TOKENS")) {
			scimTenantTokens[tid] = tok
		}
	}

	roleGuard := auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin)
	readPerm := auth.RequireAnySupplierPerm(auth.PermIAMManage, auth.PermAuditRead)
	managePerm := auth.RequireSupplierPerm(auth.PermIAMManage)

	mountIAM := func(gr chi.Router) {
		gr.With(roleGuard, readPerm).Get("/v1/iam/roles", d.Service.HandleListRoles)
		gr.With(roleGuard, readPerm).Get("/v1/iam/permissions", d.Service.HandleListPermissions)
		gr.With(roleGuard, managePerm).Post("/v1/iam/users/{userId}/roles", d.Service.HandleAssignUserRole)
		gr.With(roleGuard, readPerm).Get("/v1/iam/sod-policies", d.Service.HandleListSodPolicies)
		gr.With(roleGuard, managePerm).Put("/v1/iam/sod-policies/{id}", d.Service.HandleUpdateSodPolicy)
		gr.With(roleGuard, readPerm).Get("/v1/iam/sod-violations", d.Service.HandleListSodViolations)
		gr.With(roleGuard, readPerm).Get("/v1/iam/auditor-grants", d.Service.HandleListAuditorGrants)
		gr.With(roleGuard, managePerm).Post("/v1/iam/auditor-grants", d.Service.HandleGrantAuditorAccessWithJWT(d.JWTSecret, d.JWTIssuer))
		gr.With(roleGuard, managePerm).Post("/v1/iam/auditor-grants/{userID}/revoke", d.Service.HandleRevokeAuditorGrant)
		gr.With(roleGuard, readPerm).Get("/v1/iam/custom-roles", d.Service.HandleListCustomRoles)
		gr.With(roleGuard, managePerm).Post("/v1/iam/custom-roles", d.Service.HandleCreateCustomRole)
		gr.With(roleGuard, managePerm).Delete("/v1/iam/custom-roles/{roleName}", d.Service.HandleDeleteCustomRole)
	}

	r.Group(func(gr chi.Router) {
		if strings.TrimSpace(d.JWTSecret) != "" {
			gr.Use(auth.CookieAuth(d.JWTSecret))
		}
		gr.Use(auth.RequireAuditorActive)
		gr.Use(auth.AttachTenantFromClaims)
		auth.ProtectMutations(gr, auth.MutationGuardConfig{
			AllowBypass: d.AllowAuthBypass,
		}, mountIAM)
	})

	mountSCIM := func(gr chi.Router) {
		gr.With(roleGuard, readPerm).Get("/scim/v2/Users", d.Service.HandleListScimUsers)
		gr.With(roleGuard, managePerm).Post("/scim/v2/Users", d.Service.HandleCreateScimUser)
		gr.With(roleGuard, readPerm).Get("/scim/v2/Users/{id}", d.Service.HandleGetScimUser)
		gr.With(roleGuard, managePerm).Put("/scim/v2/Users/{id}", d.Service.HandlePutScimUser)
		gr.With(roleGuard, managePerm).Patch("/scim/v2/Users/{id}", d.Service.HandlePatchScimUser)
		gr.With(roleGuard, managePerm).Delete("/scim/v2/Users/{id}", d.Service.HandleDeleteScimUser)

		gr.With(roleGuard, readPerm).Get("/scim/v2/Groups", d.Service.HandleListScimGroups)
		gr.With(roleGuard, managePerm).Post("/scim/v2/Groups", d.Service.HandleCreateScimGroup)
		gr.With(roleGuard, readPerm).Get("/scim/v2/Groups/{id}", d.Service.HandleGetScimGroup)
		gr.With(roleGuard, managePerm).Put("/scim/v2/Groups/{id}", d.Service.HandlePutScimGroup)
		gr.With(roleGuard, managePerm).Patch("/scim/v2/Groups/{id}", d.Service.HandlePatchScimGroup)
		gr.With(roleGuard, managePerm).Delete("/scim/v2/Groups/{id}", d.Service.HandleDeleteScimGroup)
	}

	r.Group(func(gr chi.Router) {
		if strings.TrimSpace(d.JWTSecret) != "" {
			gr.Use(auth.CookieAuth(d.JWTSecret))
		}
		if scimToken != "" || len(scimTenantTokens) > 0 {
			gr.Use(scimBearerAuthMiddleware(scimToken, scimTenantID, scimTenantTokens, d.AllowUnscopedStaticSCIMToken))
		}
		gr.Use(auth.RequireAuditorActive)
		gr.Use(auth.AttachTenantFromClaims)
		auth.ProtectMutations(gr, auth.MutationGuardConfig{
			AllowBypass: d.AllowAuthBypass,
		}, mountSCIM)
	})
}

func scimBearerAuthMiddleware(expectedToken, boundTenantID string, tenantTokens map[string]string, allowUnscoped bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if _, hasClaims := auth.FromContext(ctx); hasClaims || auth.IsAuditorExpiredInContext(ctx) {
				next.ServeHTTP(w, r)
				return
			}
			authz := strings.TrimSpace(r.Header.Get("Authorization"))
			const prefix = "Bearer "
			if len(authz) <= len(prefix) || !strings.EqualFold(authz[:len(prefix)], prefix) {
				next.ServeHTTP(w, r)
				return
			}
			provided := strings.TrimSpace(authz[len(prefix):])
			if provided == "" {
				next.ServeHTTP(w, r)
				return
			}

			var matched bool
			var matchedTenant string
			// Compare against per-tenant tokens first without short-circuiting on error.
			for tid, tok := range tenantTokens {
				if subtle.ConstantTimeCompare([]byte(provided), []byte(tok)) == 1 {
					matched = true
					matchedTenant = tid
				}
			}
			if !matched && expectedToken != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(expectedToken)) == 1 {
				matched = true
				matchedTenant = boundTenantID
			}
			if !matched {
				next.ServeHTTP(w, r)
				return
			}

			reqTenant := firstNonEmptySCIM(
				r.Header.Get("X-Tenant-Id"),
				r.Header.Get("X-Supplier-Id"),
				r.URL.Query().Get("entity"),
				r.URL.Query().Get("tenant_id"),
			)
			tenantID := matchedTenant
			if tenantID != "" {
				if reqTenant != "" && reqTenant != tenantID {
					writeSCIMMiddlewareError(w, http.StatusForbidden, "scim_tenant_scope_mismatch", "SCIM bearer token is not authorized for requested tenant")
					return
				}
			} else {
				if !allowUnscoped {
					writeSCIMMiddlewareError(w, http.StatusForbidden, "scim_token_unscoped", "static SCIM bearer token requires SCIM_TENANT_ID or SCIM_TENANT_TOKENS tenant binding")
					return
				}
				tenantID = reqTenant
			}

			claims := auth.Claims{
				Subject:       "scim-connector",
				Role:          auth.RoleAdmin,
				SupplierID:    tenantID,
				SupplierRole:  auth.SupplierRoleOwner,
				SupplierPerms: []string{auth.PermIAMManage, auth.PermAuditRead},
				ActorType:     auth.ActorTypeConnector,
			}
			next.ServeHTTP(w, r.WithContext(auth.WithClaims(ctx, claims)))
		})
	}
}

func writeSCIMMiddlewareError(w http.ResponseWriter, status int, scimType, detail string) {
	w.Header().Set("Content-Type", "application/scim+json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"schemas":  []string{"urn:ietf:params:scim:api:messages:2.0:Error"},
		"status":   http.StatusText(status),
		"scimType": scimType,
		"detail":   detail,
	})
}

func firstNonEmptySCIM(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
