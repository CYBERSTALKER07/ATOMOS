package bootstrap

import "net/http"

// DefaultMaxRequestBodyBytes is the global limit for incoming HTTP request bodies (10MB).
// 10MB comfortably accommodates batch order imports, catalog uploads, and telemetry,
// while defending against memory exhaustion and HTTP body denial-of-service attacks.
const DefaultMaxRequestBodyBytes = 10 << 20 // 10 MB

// MaxLocalMigrationUploadBytes bounds non-enforced local migration artifact PUT/chunked uploads (55 MiB).
const MaxLocalMigrationUploadBytes int64 = 55 << 20

// MaxBytesMiddleware wraps r.Body in http.MaxBytesReader to enforce a strict ceiling.
func MaxBytesMiddleware(limit int64) func(http.Handler) http.Handler {
	return MaxBytesMiddlewareWithLocalUpload(limit, false)
}

// MaxBytesMiddlewareWithLocalUpload enforces defaultLimit globally and raises the ceiling to
// MaxLocalMigrationUploadBytes only for non-enforced local migration upload requests.
func MaxBytesMiddlewareWithLocalUpload(defaultLimit int64, allowLocalUpload bool) func(http.Handler) http.Handler {
	if defaultLimit <= 0 {
		defaultLimit = DefaultMaxRequestBodyBytes
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limit := defaultLimit
			if allowLocalUpload &&
				r.URL.Path == "/v1/supplier/migrations/local-upload" &&
				(r.Method == http.MethodPut || r.Method == http.MethodPost) {
				limit = MaxLocalMigrationUploadBytes
			}
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			next.ServeHTTP(w, r)
		})
	}
}
