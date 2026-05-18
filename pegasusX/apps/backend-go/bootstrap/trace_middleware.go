package bootstrap

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// TraceMiddleware ensures every request carries a trace identifier through
// headers, request context, and downstream outbox emission paths.
func TraceMiddleware(next http.Handler) http.Handler {
	if next == nil {
		return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := strings.TrimSpace(r.Header.Get("X-Trace-Id"))
		if traceID == "" {
			traceID = strings.TrimSpace(r.Header.Get("X-Request-Id"))
		}
		if traceID == "" {
			traceID = newTraceID()
		}
		w.Header().Set("X-Trace-Id", traceID)
		ctx := outbox.WithTraceID(r.Context(), traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newTraceID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "trace-fallback"
	}
	return hex.EncodeToString(buf)
}
