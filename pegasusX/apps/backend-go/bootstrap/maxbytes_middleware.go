package bootstrap

import "net/http"

// DefaultMaxRequestBodyBytes is the global limit for incoming HTTP request bodies (10MB).
// 10MB comfortably accommodates batch order imports, catalog uploads, and telemetry,
// while defending against memory exhaustion and HTTP body denial-of-service attacks.
const DefaultMaxRequestBodyBytes = 10 << 20 // 10 MB

// MaxBytesMiddleware wraps r.Body in http.MaxBytesReader to enforce a strict ceiling.
func MaxBytesMiddleware(limit int64) func(http.Handler) http.Handler {
	if limit <= 0 {
		limit = DefaultMaxRequestBodyBytes
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			next.ServeHTTP(w, r)
		})
	}
}
