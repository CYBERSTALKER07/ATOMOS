import re

with open('pegasus/apps/backend-go/bootstrap/middleware.go', 'r') as f:
    content = f.read()

import_block = """import (
        "log/slog"
        "net/http"
        "strings"
        "time"

        "backend-go/analytics"
        "backend-go/auth"
        "backend-go/telemetry"
)"""

content = re.sub(r'import \([\s\S]*?"backend-go/telemetry"\n\)', import_block, content)

middleware_func = """func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
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
}"""

content = re.sub(r'func LoggingMiddleware[\s\S]*?^}$', middleware_func, content, flags=re.MULTILINE)

with open('pegasus/apps/backend-go/bootstrap/middleware.go', 'w') as f:
    f.write(content)
