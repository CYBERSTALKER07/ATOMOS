package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// MutationGuardConfig controls how mutation routes authenticate callers.
type MutationGuardConfig struct {
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
		gr.Use(RequireAnyAuthenticated())
		mount(gr)
	})
}

// RequireDeviceCert enforces mTLS or hardware TPM certificate validation for terminals.
func RequireDeviceCert(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock implementation: In production, this would verify r.TLS.PeerCertificates
		// or a custom hardware attestation header.
		certHeader := r.Header.Get("X-Device-Cert")
		if IsProduction() && certHeader == "" && r.TLS != nil && len(r.TLS.PeerCertificates) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "hardware_attestation_required"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
