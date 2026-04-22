// Package telemetry owns GPS buffering, fleet hub communication, and
// distributed trace propagation. This file provides the canonical trace-id
// threading primitives referenced throughout the V.O.I.D. doctrine.
//
// Hybrid Protocol:
//
//	Internal (Backend-to-Backend): W3C traceparent header.
//	External (Client-to-Backend):  X-Trace-Id for simplicity in the 13 apps.
//	The middleware normalises both into a unified span context stored in ctx.
//
// Every inbound request is tagged with a trace_id. The id propagates through
// context.Context via WithTraceID and must appear on every structured log
// line, every Kafka event, and every WebSocket broadcast payload triggered
// by that request.
package telemetry

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

type traceKey struct{}
type spanIDKey struct{}

// WithTraceID returns a child context carrying the given trace identifier.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceKey{}, id)
}

// TraceIDFromContext extracts the trace identifier previously stored via
// WithTraceID. Returns empty string when no trace is attached — callers
// should treat empty as "untraced", not as an error.
func TraceIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(traceKey{}).(string)
	return v
}

// WithSpanID stores a W3C parent-id (span) alongside the trace.
func WithSpanID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, spanIDKey{}, id)
}

// SpanIDFromContext extracts the span-id. Empty when the request arrived
// via X-Trace-Id (external client) rather than traceparent (internal).
func SpanIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(spanIDKey{}).(string)
	return v
}

// GenerateTraceID produces a new UUIDv7-formatted identifier suitable for
// cross-service correlation. UUIDv7 is time-ordered so log grep and Kafka
// partition scans benefit from monotonic prefixes.
func GenerateTraceID() string {
	id, err := uuid.NewV7()
	if err != nil {
		// uuid.NewV7 only errors on broken crypto/rand; fall back to v4.
		return uuid.NewString()
	}
	return id.String()
}

// GenerateSpanID produces a random 16-hex-character span identifier
// compatible with the W3C trace-context parent-id field.
func GenerateSpanID() string {
	id := trace.SpanID{}
	// Use a random UUID as entropy source (avoids importing crypto/rand directly).
	raw := uuid.New()
	copy(id[:], raw[:8])
	return id.String()
}

// ParsedTraceparent holds the fields extracted from a W3C traceparent header.
type ParsedTraceparent struct {
	TraceID string // 32 hex lowercase
	SpanID  string // 16 hex lowercase
	Flags   string // 2 hex lowercase (e.g. "01" for sampled)
}

// ParseTraceparent extracts trace-id, parent-id, and flags from a W3C
// traceparent header value. Returns nil when the header is absent or
// malformed. Format: "00-<trace-id>-<parent-id>-<flags>".
func ParseTraceparent(header string) *ParsedTraceparent {
	parts := strings.Split(header, "-")
	if len(parts) != 4 {
		return nil
	}
	if len(parts[1]) != 32 || len(parts[2]) != 16 || len(parts[3]) != 2 {
		return nil
	}
	return &ParsedTraceparent{
		TraceID: parts[1],
		SpanID:  parts[2],
		Flags:   parts[3],
	}
}

// FormatTraceparent builds a W3C traceparent header value from a trace-id
// and span-id. Flags default to "01" (sampled). The trace-id is normalised
// to 32 hex (UUIDs are stripped of dashes and zero-padded).
func FormatTraceparent(traceID, spanID string) string {
	hex := strings.ReplaceAll(traceID, "-", "")
	if len(hex) < 32 {
		hex = strings.Repeat("0", 32-len(hex)) + hex
	}
	if len(hex) > 32 {
		hex = hex[:32]
	}
	return fmt.Sprintf("00-%s-%s-01", hex, spanID)
}
