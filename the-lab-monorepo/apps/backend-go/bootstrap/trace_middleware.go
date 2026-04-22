package bootstrap

import (
	"net/http"

	"backend-go/telemetry"
)

// TraceMiddleware is a chi-compatible middleware that ensures every request
// carries a trace_id (and optionally a span_id) in its context.
//
// Hybrid Protocol:
//
//	1. Internal callers send W3C traceparent ("00-<trace_id>-<span_id>-<flags>").
//	   The middleware extracts both trace_id and parent span_id.
//	2. External clients send X-Trace-Id (or X-Request-Id as fallback).
//	   The middleware promotes the value to trace_id and generates a fresh span_id.
//	3. When neither header is present, the middleware generates a new UUIDv7
//	   trace_id and a random span_id.
//
// Both the UUID-form trace_id (for structured logs / Kafka) and the W3C
// traceparent (for internal propagation) are echoed in the response headers
// so callers can correlate responses.
//
// Wire this as the FIRST middleware on the chi router so the trace_id is
// available to all subsequent layers (CORS, auth, logging, priority guard).
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var traceID, spanID string

		// ── Priority 1: W3C traceparent (internal backend-to-backend) ──
		if tp := r.Header.Get("Traceparent"); tp != "" {
			if parsed := telemetry.ParseTraceparent(tp); parsed != nil {
				traceID = parsed.TraceID
				spanID = parsed.SpanID
			}
		}

		// ── Priority 2: X-Trace-Id / X-Request-Id (external clients) ──
		if traceID == "" {
			traceID = r.Header.Get("X-Trace-Id")
		}
		if traceID == "" {
			traceID = r.Header.Get("X-Request-Id")
		}

		// ── Fallback: generate fresh identifiers ──
		if traceID == "" {
			traceID = telemetry.GenerateTraceID()
		}
		if spanID == "" {
			spanID = telemetry.GenerateSpanID()
		}

		ctx := telemetry.WithTraceID(r.Context(), traceID)
		ctx = telemetry.WithSpanID(ctx, spanID)

		// Echo both forms so callers can correlate:
		// - X-Trace-Id for simple client-side logging
		// - Traceparent for internal services that propagate W3C context
		w.Header().Set("X-Trace-Id", traceID)
		w.Header().Set("Traceparent", telemetry.FormatTraceparent(traceID, spanID))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
