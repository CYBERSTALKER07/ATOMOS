package auth

import "github.com/go-chi/chi/v5"

// MutationGuardConfig controls how mutation routes authenticate callers.
type MutationGuardConfig struct {
	FirebaseEnabled  bool
	FirebaseVerifier FirebaseVerifier
	// AllowBypass mounts protected handlers without auth (e2e/smoke only).
	AllowBypass bool
}

// ProtectMutations mounts a protected route subtree. When AllowBypass is false,
// callers must present a valid session (Bearer or supplier cookie via
// SessionAuth) and per-route RequireRole middleware enforces role scope.
func ProtectMutations(r chi.Router, cfg MutationGuardConfig, mount func(chi.Router)) {
	if cfg.AllowBypass {
		mount(r)
		return
	}
	r.Group(func(gr chi.Router) {
		if cfg.FirebaseEnabled && cfg.FirebaseVerifier != nil {
			gr.Use(FirebaseAuth(cfg.FirebaseVerifier))
		}
		mount(gr)
	})
}
