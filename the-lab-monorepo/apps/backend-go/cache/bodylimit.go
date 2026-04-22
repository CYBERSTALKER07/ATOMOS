package cache

import (
	"net/http"
)

// MaxBodySize is the global request body limit (1 MB).
// Individual endpoints (e.g., file uploads) can override by reading the body themselves.
const MaxBodySize = 1 << 20 // 1 MB

// LimitBodyMiddleware enforces a maximum request body size on all POST/PATCH/PUT requests.
// Returns 413 Payload Too Large if the client sends more than maxBytes.
func LimitBodyMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil && (r.Method == "POST" || r.Method == "PATCH" || r.Method == "PUT") {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
