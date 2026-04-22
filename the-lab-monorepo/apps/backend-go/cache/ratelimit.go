package cache

import (
	"backend-go/auth"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// RateLimitConfig defines the request quota for a sliding window.
type RateLimitConfig struct {
	Window  time.Duration // Sliding window size
	MaxHits int64         // Max requests per window
	KeyFunc func(r *http.Request) string
}

// DefaultRateLimit applies a global per-IP limit.
func DefaultRateLimit() RateLimitConfig {
	return RateLimitConfig{
		Window:  1 * time.Minute,
		MaxHits: 120,
		KeyFunc: ipKey,
	}
}

// AuthRateLimit applies a stricter limit on auth endpoints (brute-force protection).
func AuthRateLimit() RateLimitConfig {
	return RateLimitConfig{
		Window:  1 * time.Minute,
		MaxHits: 10,
		KeyFunc: ipKey,
	}
}

// APIRateLimit applies a per-authenticated-user limit on general API endpoints.
// Falls back to IP-based limiting if no user ID is present.
func APIRateLimit() RateLimitConfig {
	return RateLimitConfig{
		Window:  1 * time.Minute,
		MaxHits: 60,
		KeyFunc: userOrIPKey,
	}
}

// AnalyticsRateLimit applies a stricter per-user limit on analytics/heavy-read endpoints.
func AnalyticsRateLimit() RateLimitConfig {
	return RateLimitConfig{
		Window:  1 * time.Minute,
		MaxHits: 30,
		KeyFunc: userOrIPKey,
	}
}

// ── Role-Tiered Rate Limits ─────────────────────────────────────────────────

// DriverGPSRateLimit enforces 1 request per 5 seconds per authenticated driver.
// GPS telemetry is high-frequency but low-priority — this prevents flood from
// malfunctioning clients while still allowing sufficient update resolution.
func DriverGPSRateLimit() RateLimitConfig {
	return RateLimitConfig{
		Window:  5 * time.Second,
		MaxHits: 1,
		KeyFunc: userOrIPKey,
	}
}

// RetailerRateLimit allows 10 requests/second per authenticated retailer.
// Retailers interact via order browsing, checkout, and payment confirmation.
func RetailerRateLimit() RateLimitConfig {
	return RateLimitConfig{
		Window:  1 * time.Second,
		MaxHits: 10,
		KeyFunc: userOrIPKey,
	}
}

// AdminRateLimit allows 50 requests/second per authenticated supplier/admin.
// Admin portal makes dense parallel requests (dashboard, fleet, analytics).
func AdminRateLimit() RateLimitConfig {
	return RateLimitConfig{
		Window:  1 * time.Second,
		MaxHits: 50,
		KeyFunc: userOrIPKey,
	}
}

// WebhookRateLimit allows 100 requests/second per IP for payment provider webhooks.
// Payment gateways (Click, Payme) may burst during settlement windows.
func WebhookRateLimit() RateLimitConfig {
	return RateLimitConfig{
		Window:  1 * time.Second,
		MaxHits: 100,
		KeyFunc: ipKey,
	}
}

// RateLimitMiddleware returns an HTTP middleware that enforces the given rate limit
// using Redis Lua-backed token bucket checks.
// If Redis is unavailable, requests pass through (fail-open for availability).
func RateLimitMiddleware(cfg RateLimitConfig) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if GetClient() == nil {
				next.ServeHTTP(w, r)
				return
			}

			key := fmt.Sprintf("%s%s", PrefixRateLimit, cfg.KeyFunc(r))
			ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
			defer cancel()
			windowSec := int64(cfg.Window / time.Second)
			if windowSec <= 0 {
				windowSec = 1
			}

			result := CheckTokenBucket(ctx, key, cfg.MaxHits, windowSec)

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.MaxHits))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))

			if result.TTL > 0 {
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Duration(result.TTL)*time.Second).Unix()))
			}

			if !result.Allowed {
				retryAfter := result.TTL
				if retryAfter <= 0 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
				http.Error(w, `{"error":"rate_limit_exceeded","message":"Too many requests"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		}
	}
}

func ipKey(r *http.Request) string {
	// Prefer X-Forwarded-For (Cloud Run / reverse proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip := strings.TrimSpace(strings.Split(xff, ",")[0])
		if ip != "" {
			return "ip:" + ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "ip:" + r.RemoteAddr
	}
	return "ip:" + host
}

// userOrIPKey extracts the authenticated user ID from the request context
// (set by auth middleware). Falls back to IP if unauthenticated.
func userOrIPKey(r *http.Request) string {
	if claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims); ok && claims != nil && claims.UserID != "" {
		return "uid:" + claims.UserID
	}
	return ipKey(r)
}
