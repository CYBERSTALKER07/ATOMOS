package cache

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// ── Priority-Aware Load Shedder ─────────────────────────────────────────────
// Google-style traffic orchestration with request classification, adaptive
// backpressure, and Lua-backed atomic token buckets.
//
// Priority tiers:
//   P0 (CRITICAL)    — Checkout, Cancel, Driver Emergency → 100% guaranteed
//   P1 (OPERATIONAL) — State changes, Manifest generation → Shed at 90% load
//   P2 (TELEMETRY)   — GPS pings, Heartbeats, Logs       → Shed at 70% load
//
// When Redis latency exceeds thresholds, the system begins shedding lower-
// priority traffic and signals clients via X-Backpressure-Interval headers.

// Priority represents request criticality.
type Priority int

const (
	PriorityCritical    Priority = 0 // P0 — Never shed
	PriorityOperational Priority = 1 // P1 — Shed at high load
	PriorityTelemetry   Priority = 2 // P2 — Shed first
)

func (p Priority) String() string {
	switch p {
	case PriorityCritical:
		return "P0_CRITICAL"
	case PriorityOperational:
		return "P1_OPERATIONAL"
	case PriorityTelemetry:
		return "P2_TELEMETRY"
	default:
		return "UNKNOWN"
	}
}

// ── Route Classification ────────────────────────────────────────────────────

// p0Paths are critical paths that must never be shed.
var p0Paths = []string{
	"/v1/checkout/",
	"/v1/order/cash-checkout",
	"/v1/order/card-checkout",
	"/v1/orders/request-cancel",
	"/v1/admin/orders/approve-cancel",
	"/v1/order/cancel",
	"/v1/delivery/emergency",
	"/v1/webhooks/",
	"/v1/payment/",
}

// p2Paths are telemetry/low-priority paths shed first.
var p2Paths = []string{
	"/ws/telemetry",
	"/ws/fleet",
	"/v1/sync/batch",
	"/v1/telemetry/",
	"/v1/analytics/",
	"/v1/supplier/dashboard",
}

// ClassifyRequest returns the priority tier for a given request path.
func ClassifyRequest(path string) Priority {
	lower := strings.ToLower(path)
	for _, prefix := range p0Paths {
		if strings.HasPrefix(lower, prefix) {
			return PriorityCritical
		}
	}
	for _, prefix := range p2Paths {
		if strings.HasPrefix(lower, prefix) {
			return PriorityTelemetry
		}
	}
	return PriorityOperational
}

// ── Adaptive Backpressure Engine ────────────────────────────────────────────
// Monitors Redis latency to determine system health. When latency degrades,
// lower-priority traffic is shed and clients are told to back off.

// BackpressureConfig tunes the shedding thresholds.
type BackpressureConfig struct {
	P2ShedLatency time.Duration // Redis latency that triggers P2 shedding (default 50ms)
	P1ShedLatency time.Duration // Redis latency that triggers P1 shedding (default 150ms)
	ProbeInterval time.Duration // How often to probe Redis latency (default 2s)
}

// DefaultBackpressureConfig returns production-tuned defaults.
func DefaultBackpressureConfig() BackpressureConfig {
	return BackpressureConfig{
		P2ShedLatency: 50 * time.Millisecond,
		P1ShedLatency: 150 * time.Millisecond,
		ProbeInterval: 2 * time.Second,
	}
}

// BackpressureEngine tracks system health and determines shedding behavior.
type BackpressureEngine struct {
	config BackpressureConfig
	// latencyNanos is the last measured Redis PING latency in nanoseconds.
	// Accessed atomically for lock-free reads in the hot path.
	latencyNanos atomic.Int64
	done         chan struct{}
}

// NewBackpressureEngine creates and starts the engine. It probes Redis
// latency in a background goroutine and updates the atomic load factor.
func NewBackpressureEngine(cfg BackpressureConfig) *BackpressureEngine {
	eng := &BackpressureEngine{
		config: cfg,
		done:   make(chan struct{}),
	}
	go eng.probeLoop()
	log.Printf("[BACKPRESSURE] Engine started — P2 shed at %v, P1 shed at %v, probe every %v",
		cfg.P2ShedLatency, cfg.P1ShedLatency, cfg.ProbeInterval)
	return eng
}

// probeLoop periodically PINGs Redis and records latency.
func (e *BackpressureEngine) probeLoop() {
	ticker := time.NewTicker(e.config.ProbeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-e.done:
			return
		case <-ticker.C:
			e.probe()
		}
	}
}

func (e *BackpressureEngine) probe() {
	if Client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	err := Client.Ping(ctx).Err()
	elapsed := time.Since(start)

	if err != nil {
		// Redis unreachable — set very high latency to trigger maximum shedding
		e.latencyNanos.Store(int64(500 * time.Millisecond))
		return
	}
	e.latencyNanos.Store(int64(elapsed))
}

// Latency returns the last measured Redis latency.
func (e *BackpressureEngine) Latency() time.Duration {
	return time.Duration(e.latencyNanos.Load())
}

// ShouldShed returns true if the given priority should be shed under current load.
func (e *BackpressureEngine) ShouldShed(p Priority) bool {
	lat := e.Latency()
	switch p {
	case PriorityCritical:
		return false // Never shed P0
	case PriorityOperational:
		return lat >= e.config.P1ShedLatency
	case PriorityTelemetry:
		return lat >= e.config.P2ShedLatency
	default:
		return false
	}
}

// BackpressureInterval returns the number of seconds a client should wait
// before retrying telemetry/non-critical requests. Returns 0 if no backpressure.
func (e *BackpressureEngine) BackpressureInterval() int {
	lat := e.Latency()
	switch {
	case lat >= e.config.P1ShedLatency:
		return 30 // Severe — back off 30s
	case lat >= e.config.P2ShedLatency:
		return 10 // Moderate — back off 10s
	default:
		return 0 // Healthy
	}
}

// Stop gracefully stops the probe loop.
func (e *BackpressureEngine) Stop() {
	close(e.done)
}

// ── Lua-Backed Token Bucket ─────────────────────────────────────────────────
// Atomic "check-and-decrement" in a single Redis round-trip.
// Prevents race conditions during high-concurrency spikes where INCR+TTL
// could interleave between two requests.

// luaTokenBucket is the Redis Lua script for atomic rate limiting.
// KEYS[1] = rate limit key
// ARGV[1] = max tokens (bucket capacity)
// ARGV[2] = window in seconds
// Returns: [remaining_tokens, ttl_seconds]
//
// Algorithm: Sliding window counter with atomic check-and-decrement.
// If the key doesn't exist, it's created with capacity - 1 and a TTL.
// If the key exists and has tokens, decrement and return remaining.
// If the key exists with 0 tokens, reject with remaining = -1.
const luaTokenBucket = `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local window = tonumber(ARGV[2])

local current = redis.call('GET', key)
if current == false then
    redis.call('SET', key, capacity - 1, 'EX', window)
    return {capacity - 1, window}
end

local tokens = tonumber(current)
if tokens <= 0 then
    local ttl = redis.call('TTL', key)
    return {-1, ttl}
end

local remaining = redis.call('DECR', key)
local ttl = redis.call('TTL', key)
return {remaining, ttl}
`

// TokenBucketResult holds the result of a Lua-backed rate limit check.
type TokenBucketResult struct {
	Allowed   bool
	Remaining int64
	TTL       int64 // seconds until window resets
}

// CheckTokenBucket executes the atomic Lua token bucket against Redis.
// Returns a result indicating whether the request is allowed and remaining tokens.
// Fails open (allows request) if Redis is unavailable.
func CheckTokenBucket(ctx context.Context, key string, capacity int64, windowSec int64) TokenBucketResult {
	if Client == nil {
		return TokenBucketResult{Allowed: true, Remaining: capacity, TTL: 0}
	}

	result, err := Client.Eval(ctx, luaTokenBucket, []string{key}, capacity, windowSec).Int64Slice()
	if err != nil {
		// Redis error — fail open
		return TokenBucketResult{Allowed: true, Remaining: capacity, TTL: 0}
	}

	remaining := result[0]
	ttl := result[1]
	return TokenBucketResult{
		Allowed:   remaining >= 0,
		Remaining: max(0, remaining),
		TTL:       ttl,
	}
}

// ── Priority Middleware (Combining All Three Layers) ─────────────────────────

// PrioritySheddingMiddleware wraps a handler with the full traffic orchestration
// pipeline: classify → backpressure check → Lua token bucket → headers → serve.
func PrioritySheddingMiddleware(engine *BackpressureEngine, bucketCapacity int64, windowSec int64) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			priority := ClassifyRequest(r.URL.Path)

			// Layer 1: Adaptive backpressure — shed by priority under load
			if engine != nil && engine.ShouldShed(priority) {
				interval := engine.BackpressureInterval()
				w.Header().Set("X-Backpressure-Interval", fmt.Sprintf("%d", interval))
				w.Header().Set("X-Priority", priority.String())
				w.Header().Set("Retry-After", fmt.Sprintf("%d", interval))
				http.Error(w,
					fmt.Sprintf(`{"error":"load_shed","priority":"%s","retry_after":%d,"detail":"System under load. Priority %s traffic temporarily shed."}`,
						priority, interval, priority),
					http.StatusServiceUnavailable)
				return
			}

			// Layer 2: Lua-backed token bucket — atomic per-identity rate limit
			if Client != nil {
				key := fmt.Sprintf("tb:%s:%s", priority, userOrIPKey(r))
				result := CheckTokenBucket(r.Context(), key, bucketCapacity, windowSec)

				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", bucketCapacity))
				w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
				w.Header().Set("X-Priority", priority.String())

				if result.TTL > 0 {
					w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Duration(result.TTL)*time.Second).Unix()))
				}

				if !result.Allowed {
					retryAfter := result.TTL
					if retryAfter <= 0 {
						retryAfter = 1
					}
					w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
					http.Error(w,
						fmt.Sprintf(`{"error":"rate_limit_exceeded","priority":"%s","retry_after":%d}`, priority, retryAfter),
						http.StatusTooManyRequests)
					return
				}
			}

			// Layer 3: Set backpressure signal header (even when not shedding)
			if engine != nil {
				if interval := engine.BackpressureInterval(); interval > 0 {
					w.Header().Set("X-Backpressure-Interval", fmt.Sprintf("%d", interval))
				}
			}

			next.ServeHTTP(w, r)
		}
	}
}
