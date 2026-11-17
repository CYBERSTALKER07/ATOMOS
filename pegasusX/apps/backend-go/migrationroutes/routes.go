// Package migrationroutes mounts Universal Smart Ingestion & Cutover APIs.
package migrationroutes

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/migration"
)

// Deps configures dependencies for migration HTTP routes.
type Deps struct {
	Handlers        *migration.Handlers
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts `/v1/supplier/migrations/*` routes behind fail-closed session auth,
// supplier admin RBAC, and strict Idempotency-Key enforcement on POST/PATCH.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Handlers == nil {
		return
	}

	// Non-enforced HMAC-signed local upload receiver (handler enforces fail-closed guards).
	r.Put("/v1/supplier/migrations/local-upload", d.Handlers.HandleLocalArtifactUpload)
	r.Post("/v1/supplier/migrations/local-upload", d.Handlers.HandleLocalArtifactUpload)

	r.Group(func(gr chi.Router) {
		gr.Use(auth.CookieAuth(d.JWTSecret))
		gr.Use(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin))
		gr.Use(failClosedOnBypassMiddleware(d.AllowAuthBypass))
		gr.Use(requireIdempotencyKeyForMutations)

		gr.Post("/v1/supplier/migrations/projects", d.Handlers.HandleCreateProject)
		gr.Get("/v1/supplier/migrations/projects", d.Handlers.HandleListProjects)
		gr.Get("/v1/supplier/migrations/projects/{projectId}", d.Handlers.HandleGetProject)
		gr.Post("/v1/supplier/migrations/projects/{projectId}/bundles", d.Handlers.HandleCreateBundle)
		gr.Post("/v1/supplier/migrations/projects/{projectId}/bundles/{bundleId}/uploaded", d.Handlers.HandleBundleUploaded)
		gr.Patch("/v1/supplier/migrations/projects/{projectId}/bundles/{bundleId}/sort", d.Handlers.HandleOverrideBundleSort)

		gr.Get("/v1/supplier/migrations/sessions/{sessionId}", d.Handlers.HandleGetSession)
		gr.Get("/v1/supplier/migrations/sessions/{sessionId}/rows", d.Handlers.HandleGetSessionRows)
		gr.Get("/v1/supplier/migrations/sessions/{sessionId}/mapping", d.Handlers.HandleGetSessionMapping)
		gr.Post("/v1/supplier/migrations/sessions/{sessionId}/mapping", d.Handlers.HandleConfirmSessionMapping)
		gr.Post("/v1/supplier/migrations/projects/{projectId}/dry-run", d.Handlers.HandleDryRunProject)
		gr.Post("/v1/supplier/migrations/sessions/{sessionId}/dry-run", d.Handlers.HandleDryRunSession)
		gr.Get("/v1/supplier/migrations/reports/{reportId}", d.Handlers.HandleGetReport)
		gr.Post("/v1/supplier/migrations/sessions/{sessionId}/approve", d.Handlers.HandleApproveSession)
		gr.Post("/v1/supplier/migrations/sessions/{sessionId}/apply", d.Handlers.HandleApplySession)
		gr.Post("/v1/supplier/migrations/sessions/{sessionId}/rollback", d.Handlers.HandleRollbackSession)
		gr.Post("/v1/supplier/migrations/projects/{projectId}/open-orders/release", d.Handlers.HandleReleaseOpenOrders)
		gr.Post("/v1/supplier/migrations/sessions/{sessionId}/open-orders/release", d.Handlers.HandleReleaseOpenOrders)
		gr.Post("/v1/supplier/migrations/projects/{projectId}/cutover", d.Handlers.HandleCutoverProject)
	})
}

func failClosedOnBypassMiddleware(allowAuthBypass bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if migration.IsAuthBypassActive(allowAuthBypass) {
				migration.WriteAuthzError(w, migration.ErrAuthBypassForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requireIdempotencyKeyForMutations(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPatch || r.Method == http.MethodPut {
			k := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
			if k == "" {
				k = strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
			}
			if k == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "idempotency_key_required",
					"message": "Idempotency-Key header is required on mutating migration requests",
				})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
