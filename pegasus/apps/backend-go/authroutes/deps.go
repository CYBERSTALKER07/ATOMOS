// Package authroutes owns the HTTP surface for every /v1/auth/* endpoint.
//
// The handlers themselves (admin login, driver PIN login, supplier register,
// warehouse login, etc.) live in the auth, supplier, factory, and warehouse
// packages respectively — this package is the router-level composition that
// glues them together behind one /v1/auth/* prefix with the consistent
// rate-limiter + logging middleware stack.
//
// Rationale for a dedicated package (rather than adding a Register function
// to backend-go/auth): auth is imported by supplier, factory, warehouse for
// its RequireRole/RequireWarehouseScope helpers. Adding handlers from those
// packages back into auth's Register surface would create a cycle.
package authroutes

import (
	"context"
	"net/http"

	"cloud.google.com/go/spanner"
)

// Middleware is the generic handler-wrap contract used by the logging and
// rate-limiter middlewares passed in from main.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// RetailerStatusProvider is the sole part of OrderService that the legacy
// /v1/auth/login handler needs — narrowed to an interface to keep this
// package free of an import cycle through the order service.
type RetailerStatusProvider interface {
	GetRetailerStatus(ctx context.Context, userID string) (string, error)
}

// Deps is the fully-populated set of collaborators Register needs. Every
// field is required unless documented otherwise.
type Deps struct {
	// Spanner is the shared data-plane client.
	Spanner *spanner.Client
	// RetailerStatus backs the legacy POST /v1/auth/login handler (the
	// pre-mobile-app web login that returns a JWT if KYC is VERIFIED).
	RetailerStatus RetailerStatusProvider
	// Log is the observability middleware (request counter + duration log).
	Log Middleware
	// RateLimit is the auth-specific rate-limit middleware (typically
	// cache.RateLimitMiddleware(cache.AuthRateLimit())).
	RateLimit Middleware
	// ActorRateLimit is a per-authenticated-actor limiter used on token-refresh
	// and role-gated auth management paths (typically APIRateLimit).
	ActorRateLimit Middleware
	// Idempotency protects replay-prone authenticated registration mutations.
	Idempotency Middleware
}
