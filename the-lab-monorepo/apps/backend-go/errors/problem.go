// Package errors — RFC 7807 Problem Details for HTTP APIs with three-tier
// Unified Response Strategy extensions.
//
// Tier 1 (retailer-facing): Title + Detail — human-readable, localizable.
// Tier 2 (operator-facing): Code — stable machine-readable status for admin dashboards.
// Tier 3 (system-level): Type — URI reference for engineering telemetry.
//
// Every error response from the API MUST use WriteProblem, WriteOperational,
// or one of the convenience wrappers. This ensures consistent error contracts
// across all endpoints, enabling deterministic frontend parsing, native app
// i18n lookups, and centralized tracing.
//
// Reference: https://www.rfc-editor.org/rfc/rfc7807
package errors

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
)

// ProblemDetail is the RFC 7807 response body.
type ProblemDetail struct {
	// Type is a URI reference identifying the problem type.
	// Example: "error/insufficient-stock", "error/auth/missing-token"
	Type string `json:"type"`

	// Title is a short, human-readable summary. SHOULD NOT change between
	// occurrences of the same problem type.
	Title string `json:"title"`

	// Status is the HTTP status code.
	Status int `json:"status"`

	// Detail is a human-readable explanation specific to this occurrence.
	Detail string `json:"detail,omitempty"`

	// TraceID is a UUID for correlating this error with server logs.
	TraceID string `json:"trace_id"`

	// Instance is the request path that generated this error.
	Instance string `json:"instance,omitempty"`

	// Code is a stable, machine-readable operational status code.
	// Used by operator dashboards and frontend switch statements.
	Code string `json:"code,omitempty"`

	// MessageKey is an i18n lookup key for native app string tables.
	// iOS: NSLocalizedString(key); Android: getString(R.string.<key>).
	MessageKey string `json:"message_key,omitempty"`

	// Retryable indicates whether the client should offer a retry action.
	Retryable bool `json:"retryable,omitempty"`

	// Action is a client hint for the recommended recovery UX.
	Action string `json:"action,omitempty"`
}

// WriteProblem writes a structured RFC 7807 JSON error response.
// Generates a unique TraceID automatically and logs it for correlation.
func WriteProblem(w http.ResponseWriter, r *http.Request, status int, problemType, title, detail string) {
	traceID := uuid.New().String()

	problem := ProblemDetail{
		Type:     problemType,
		Title:    title,
		Status:   status,
		Detail:   detail,
		TraceID:  traceID,
		Instance: r.URL.Path,
	}

	log.Printf("[ERROR] %d %s %s trace=%s detail=%s", status, r.Method, r.URL.Path, traceID, detail)

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(problem)
}

// WriteOperational writes a full Unified Response with all three message tiers:
// retailer-facing (title+detail), operator-facing (code), and system-level (type).
// Use this instead of WriteProblem when you need Code, MessageKey, Retryable, or Action.
func WriteOperational(w http.ResponseWriter, r *http.Request, p ProblemDetail) {
	if p.TraceID == "" {
		p.TraceID = uuid.New().String()
	}
	if p.Instance == "" {
		p.Instance = r.URL.Path
	}

	log.Printf("[ERROR] %d %s %s trace=%s code=%s detail=%s",
		p.Status, r.Method, r.URL.Path, p.TraceID, p.Code, p.Detail)

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	json.NewEncoder(w).Encode(p)
}

// ── Convenience Wrappers ────────────────────────────────────────────────────

// BadRequest writes a 400 error.
func BadRequest(w http.ResponseWriter, r *http.Request, detail string) {
	WriteProblem(w, r, http.StatusBadRequest, "error/bad-request", "Bad Request", detail)
}

// Unauthorized writes a 401 error.
func Unauthorized(w http.ResponseWriter, r *http.Request, detail string) {
	WriteProblem(w, r, http.StatusUnauthorized, "error/unauthorized", "Authentication Required", detail)
}

// Forbidden writes a 403 error.
func Forbidden(w http.ResponseWriter, r *http.Request, detail string) {
	WriteProblem(w, r, http.StatusForbidden, "error/forbidden", "Insufficient Permissions", detail)
}

// NotFound writes a 404 error.
func NotFound(w http.ResponseWriter, r *http.Request, detail string) {
	WriteProblem(w, r, http.StatusNotFound, "error/not-found", "Resource Not Found", detail)
}

// Conflict writes a 409 error.
func Conflict(w http.ResponseWriter, r *http.Request, detail string) {
	WriteProblem(w, r, http.StatusConflict, "error/conflict", "Conflict", detail)
}

// TooManyRequests writes a 429 error with Retry-After header.
func TooManyRequests(w http.ResponseWriter, r *http.Request, retryAfterSec int) {
	w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfterSec))
	WriteProblem(w, r, http.StatusTooManyRequests, "error/rate-limit", "Too Many Requests",
		fmt.Sprintf("Rate limit exceeded. Retry after %d seconds.", retryAfterSec))
}

// InternalError writes a 500 error. Does NOT expose internal error details —
// the detail field should be a safe user-facing message.
func InternalError(w http.ResponseWriter, r *http.Request, detail string) {
	WriteProblem(w, r, http.StatusInternalServerError, "error/internal", "Internal Server Error", detail)
}

// ServiceUnavailable writes a 503 error (used by circuit breaker).
func ServiceUnavailable(w http.ResponseWriter, r *http.Request, detail string) {
	WriteProblem(w, r, http.StatusServiceUnavailable, "error/service-unavailable", "Service Temporarily Unavailable", detail)
}

// MethodNotAllowed writes a 405 error.
func MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	WriteProblem(w, r, http.StatusMethodNotAllowed, "error/method-not-allowed", "Method Not Allowed",
		fmt.Sprintf("%s is not supported for this endpoint.", r.Method))
}
