package bootstrap

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// TraceMiddleware ensures every request carries a correlated trace identifier
// across W3C TraceContext headers ("traceparent"), legacy headers ("X-Trace-Id"),
// request context, and downstream outbox emission paths.
func TraceMiddleware(next http.Handler) http.Handler {
	if next == nil {
		return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawTP := strings.TrimSpace(r.Header.Get("traceparent"))
		var traceID, outboundTP, legacyTraceID string

		if tid, _, flags, ok := ParseTraceParent(rawTP); ok {
			traceID = tid
			legacyTraceID = tid
			childSpanID := newSpanID()
			outboundTP = fmt.Sprintf("00-%s-%s-%s", traceID, childSpanID, flags)
		} else {
			rawLegacy := strings.TrimSpace(r.Header.Get("X-Trace-Id"))
			if rawLegacy == "" {
				rawLegacy = strings.TrimSpace(r.Header.Get("X-Request-Id"))
			}

			if rawLegacy != "" {
				legacyTraceID = rawLegacy
				if isHex32(rawLegacy) && rawLegacy != "00000000000000000000000000000000" {
					traceID = rawLegacy
				} else {
					sum := sha256.Sum256([]byte(rawLegacy))
					traceID = hex.EncodeToString(sum[:16])
				}
			} else {
				traceID = newTraceID()
				legacyTraceID = traceID
			}

			spanID := newSpanID()
			outboundTP = fmt.Sprintf("00-%s-%s-01", traceID, spanID)
		}

		w.Header().Set("X-Trace-Id", legacyTraceID)
		w.Header().Set("traceparent", outboundTP)

		ctx := outbox.WithTraceID(r.Context(), legacyTraceID)
		ctx = outbox.WithTraceParent(ctx, outboundTP)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TraceParentFromContext extracts the W3C traceparent string if present in ctx.
func TraceParentFromContext(ctx context.Context) string {
	return outbox.TraceParentFromContext(ctx)
}

// ParseTraceParent parses a W3C traceparent string into its component parts:
// format: 00-{trace_id_32hex}-{parent_id_16hex}-{trace_flags_2hex}.
// Returns ok=false on any format or constraint violation.
func ParseTraceParent(raw string) (traceID, spanID, flags string, ok bool) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) != 55 {
		return "", "", "", false
	}
	parts := strings.Split(trimmed, "-")
	if len(parts) != 4 {
		return "", "", "", false
	}
	version := parts[0]
	tid := parts[1]
	sid := parts[2]
	flg := parts[3]

	if version != "00" {
		return "", "", "", false
	}
	if len(tid) != 32 || !isHex(tid) || tid == "00000000000000000000000000000000" {
		return "", "", "", false
	}
	if len(sid) != 16 || !isHex(sid) || sid == "0000000000000000" {
		return "", "", "", false
	}
	if len(flg) != 2 || !isHex(flg) {
		return "", "", "", false
	}

	return tid, sid, flg, true
}

// FormatTraceParent constructs a standard W3C traceparent header.
func FormatTraceParent(traceID, spanID string, sampled bool) string {
	tid := strings.TrimSpace(traceID)
	sid := strings.TrimSpace(spanID)
	if !isHex32(tid) || tid == "00000000000000000000000000000000" {
		tid = newTraceID()
	}
	if len(sid) != 16 || !isHex(sid) || sid == "0000000000000000" {
		sid = newSpanID()
	}
	flags := "00"
	if sampled {
		flags = "01"
	}
	return fmt.Sprintf("00-%s-%s-%s", tid, sid, flags)
}

func newTraceID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "10000000000000000000000000000001"
	}
	return hex.EncodeToString(buf)
}

func newSpanID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "1000000000000001"
	}
	return hex.EncodeToString(buf)
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

func isHex32(s string) bool {
	return len(s) == 32 && isHex(s)
}
