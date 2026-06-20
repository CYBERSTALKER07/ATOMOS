package bootstrap

import (
	"net/http"
	"os"
	"strings"
)

var defaultDevCORSOrigins = []string{
	"http://localhost:3000",
	"http://localhost:3001",
	"http://localhost:3002",
	"http://localhost:3003",
	"http://localhost:3004",
	"http://127.0.0.1:3000",
	"http://127.0.0.1:3001",
	"http://127.0.0.1:3002",
	"http://127.0.0.1:3003",
	"http://127.0.0.1:3004",
	"tauri://localhost",
	"https://tauri.localhost",
}

// DevCORSMiddleware enables browser portal clients to call the API from local dev origins.
// Disabled when PEGASUSX_ENV=production unless CORS_ALLOWED_ORIGINS is explicitly set.
func DevCORSMiddleware() func(http.Handler) http.Handler {
	allowed := parseCORSOrigins()
	if len(allowed) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	allowHeaders := "Accept, Authorization, Content-Type, X-Trace-Id, X-Idempotency-Key, X-Request-Id"
	allowMethods := "GET, POST, PUT, PATCH, DELETE, OPTIONS"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" && corsOriginAllowed(origin, allowed) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
				w.Header().Set("Access-Control-Allow-Methods", allowMethods)
				w.Header().Set("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func parseCORSOrigins() map[string]struct{} {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw != "" {
		return originSet(strings.Split(raw, ","))
	}
	if isProductionEnv() {
		return nil
	}
	return originSet(defaultDevCORSOrigins)
}

func originSet(origins []string) map[string]struct{} {
	set := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		set[origin] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

func corsOriginAllowed(origin string, allowed map[string]struct{}) bool {
	_, ok := allowed[origin]
	return ok
}
