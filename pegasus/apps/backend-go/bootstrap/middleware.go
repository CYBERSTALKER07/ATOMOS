package bootstrap

import (
        "log/slog"
        "net/http"
        "strings"
        "time"

        "backend-go/analytics"
        "backend-go/auth"
        "backend-go/telemetry"
)

// EnableCORS returns a middleware that applies the App's origin allowlist.
// Origins matching a.CORSAllowlist, ngrok/expo/LAN patterns, or same-origin
// (empty Origin header) are accepted. All others fall through without a
// CORS header — which browsers will block by design.
func (a *App) EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		switch {
		case origin != "" && a.CORSAllowlist[origin]:
			w.Header().Set("Access-Control-Allow-Origin", origin)
		case origin != "" && isPatternAllowed(origin):
			w.Header().Set("Access-Control-Allow-Origin", origin)
		case origin == "":
			// Same-origin or non-browser clients (mobile apps)
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key, X-Internal-Key, X-Trace-Id, X-Request-Id, Traceparent, Tracestate")
		w.Header().Set("Access-Control-Expose-Headers", "X-Trace-Id, Traceparent")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isPatternAllowed keeps the development-tunnel and LAN-origin passthroughs
// that predate the explicit allowlist. These are not gated on environment
// because mobile-in-development traffic comes from unpredictable LAN IPs.
func isPatternAllowed(origin string) bool {
	return strings.HasSuffix(origin, ".ngrok-free.app") ||
		strings.HasSuffix(origin, ".expo.dev") ||
		strings.HasPrefix(origin, "http://192.168.") ||
		strings.HasPrefix(origin, "http://10.0.")
}

// LoggingMiddleware times every request, updates analytics counters, and
// emits a structured JSON log line with the request's trace_id. Signature
// matches the legacy main.go helper so domain routers can adopt it by name
// when the route migration happens.
func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                analytics.IncrementRequest()
                defer analytics.DecrementRequest()
                start := time.Now()
                
                next.ServeHTTP(w, r)
                
                supplierID := "unknown"
                if claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims); ok {
                        supplierID = claims.ResolveSupplierID()
                        if supplierID == "" {
                                supplierID = "anonymous"
                        }
                }
                
                // Track pass-through utility metric
                telemetry.HTTPRequestsTotal.WithLabelValues(supplierID, r.Method, r.URL.Path).Inc()
                
                slog.InfoContext(r.Context(), "http request",
                        "trace_id", telemetry.TraceIDFromContext(r.Context()),
                        "method", r.Method,
                        "path", r.URL.Path,
                        "supplier_id", supplierID,
                        "duration_ms", time.Since(start).Milliseconds(),
                )
        }
}
