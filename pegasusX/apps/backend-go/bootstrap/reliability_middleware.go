package bootstrap

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/redis/go-redis/v9"
)

const loadBootstrapHeader = "X-PegasusX-Load-Bootstrap"

type reliabilityClass string

const (
	reliabilityClassOperational reliabilityClass = "operational"
	reliabilityClassPayment     reliabilityClass = "payment"
	reliabilityClassAuth        reliabilityClass = "auth"
	reliabilityClassWebhook     reliabilityClass = "webhook"
	reliabilityClassTelemetry   reliabilityClass = "telemetry"
)

func (c reliabilityClass) isCritical() bool {
	return c == reliabilityClassPayment || c == reliabilityClassAuth || c == reliabilityClassWebhook
}

// ReliabilityConfig controls request-side safety layers: fixed-window rate
// limiting, priority-based load shedding, and per-class circuit breaking.
type ReliabilityConfig struct {
	Enabled bool

	PriorityMaxInFlight    int
	CriticalAcquireTimeout time.Duration

	RateLimitWindow       time.Duration
	RateLimitDefaultMax   int
	RateLimitPaymentMax   int
	RateLimitAuthMax      int
	RateLimitWebhookMax   int
	RateLimitTelemetryMax int

	CircuitFailureThreshold int
	CircuitOpenDuration     time.Duration
}

func (c *ReliabilityConfig) applyDefaults() {
	if c.PriorityMaxInFlight <= 0 {
		c.PriorityMaxInFlight = 64
	}
	if c.CriticalAcquireTimeout <= 0 {
		c.CriticalAcquireTimeout = 250 * time.Millisecond
	}
	if c.RateLimitWindow <= 0 {
		c.RateLimitWindow = time.Minute
	}
	if c.RateLimitDefaultMax <= 0 {
		c.RateLimitDefaultMax = 240
	}
	if c.RateLimitPaymentMax <= 0 {
		c.RateLimitPaymentMax = 60
	}
	if c.RateLimitAuthMax <= 0 {
		c.RateLimitAuthMax = 20
	}
	if c.RateLimitWebhookMax <= 0 {
		c.RateLimitWebhookMax = 240
	}
	if c.RateLimitTelemetryMax <= 0 {
		c.RateLimitTelemetryMax = 120
	}
	if c.CircuitFailureThreshold <= 0 {
		c.CircuitFailureThreshold = 5
	}
	if c.CircuitOpenDuration <= 0 {
		c.CircuitOpenDuration = 15 * time.Second
	}
}

// DefaultReliabilityConfig returns conservative defaults for pegasusX.
func DefaultReliabilityConfig() ReliabilityConfig {
	cfg := ReliabilityConfig{Enabled: true}
	cfg.applyDefaults()
	return cfg
}

// ReliabilityConfigFromEnv returns defaults with optional RELIABILITY_RATE_LIMIT_* overrides.
func ReliabilityConfigFromEnv() ReliabilityConfig {
	cfg := DefaultReliabilityConfig()
	if v := envInt("RELIABILITY_RATE_LIMIT_DEFAULT_MAX", 0); v > 0 {
		cfg.RateLimitDefaultMax = v
	}
	if v := envInt("RELIABILITY_RATE_LIMIT_PAYMENT_MAX", 0); v > 0 {
		cfg.RateLimitPaymentMax = v
	}
	if v := envInt("RELIABILITY_RATE_LIMIT_AUTH_MAX", 0); v > 0 {
		cfg.RateLimitAuthMax = v
	}
	if v := envInt("RELIABILITY_RATE_LIMIT_WEBHOOK_MAX", 0); v > 0 {
		cfg.RateLimitWebhookMax = v
	}
	if v := envInt("RELIABILITY_RATE_LIMIT_TELEMETRY_MAX", 0); v > 0 {
		cfg.RateLimitTelemetryMax = v
	}
	if v := envInt("RELIABILITY_PRIORITY_MAX_IN_FLIGHT", 0); v > 0 {
		cfg.PriorityMaxInFlight = v
	}
	return cfg
}

// RateLimiter enforces per-actor request limits.
type RateLimiter interface {
	Allow(key string, max int, window time.Duration, now time.Time) (allowed bool, remaining int, retryAfter int)
}

// ReliabilityMiddleware applies API-side resilience controls.
type ReliabilityMiddleware struct {
	cfg     ReliabilityConfig
	now     func() time.Time
	guard   *priorityGuard
	limiter RateLimiter
	breaker *requestCircuitBreaker
}

// NewReliabilityMiddleware constructs the resilience middleware stack.
func NewReliabilityMiddleware(cfg ReliabilityConfig) *ReliabilityMiddleware {
	return newReliabilityMiddlewareWithClock(cfg, time.Now)
}

func newReliabilityMiddlewareWithClock(cfg ReliabilityConfig, now func() time.Time) *ReliabilityMiddleware {
	cfg.applyDefaults()
	if now == nil {
		now = time.Now
	}
	return &ReliabilityMiddleware{
		cfg:     cfg,
		now:     now,
		guard:   newPriorityGuard(cfg.PriorityMaxInFlight, cfg.CriticalAcquireTimeout),
		limiter: newFixedWindowRateLimiter(),
		breaker: newRequestCircuitBreaker(cfg.CircuitFailureThreshold, cfg.CircuitOpenDuration),
	}
}

// SetRedisRateLimiter swaps the in-process limiter for a Redis-backed one.
func (m *ReliabilityMiddleware) SetRedisRateLimiter(client *redis.Client) {
	if m == nil || client == nil {
		return
	}
	m.limiter = newRedisRateLimiter(client)
}

// Middleware returns an http middleware ready for chi r.Use.
func (m *ReliabilityMiddleware) Middleware(next http.Handler) http.Handler {
	if m == nil || !m.cfg.Enabled {
		return next
	}
	if next == nil {
		return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		class := classifyReliabilityPath(r.URL.Path)
		now := m.now()

		if m.breaker.ShouldReject(class, now) {
			retryAfter := m.breaker.RetryAfterSeconds(class, now)
			w.Header().Set("X-Reliability-Class", string(class))
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			http.Error(w,
				fmt.Sprintf(`{"error":"circuit_open","class":"%s","retry_after":%d}`,
					class,
					retryAfter,
				),
				http.StatusServiceUnavailable,
			)
			return
		}

		w.Header().Set("X-Reliability-Class", string(class))
		if !isReliabilityRateLimitExempt(r.URL.Path, r) {
			actor := reliabilityActorKey(r)
			maxRequests := m.limitForClass(class)
			allowed, remaining, retryAfter := m.limiter.Allow(actor+":"+string(class), maxRequests, m.cfg.RateLimitWindow, now)
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(maxRequests))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(m.cfg.RateLimitWindow.Seconds())))
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				http.Error(w,
					fmt.Sprintf(`{"error":"rate_limit_exceeded","class":"%s","retry_after":%d}`,
						class,
						retryAfter,
					),
					http.StatusTooManyRequests,
				)
				return
			}
		}

		if !m.guard.Acquire(r.Context(), class) {
			w.Header().Set("Retry-After", "5")
			http.Error(w,
				fmt.Sprintf(`{"error":"load_shed","class":"%s","retry_after":5}`,
					class,
				),
				http.StatusServiceUnavailable,
			)
			return
		}
		defer m.guard.Release()

		sw := &statusCapturingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(sw, r)
		m.breaker.Observe(class, sw.statusCode, m.now())
	})
}

func (m *ReliabilityMiddleware) limitForClass(class reliabilityClass) int {
	switch class {
	case reliabilityClassPayment:
		return m.cfg.RateLimitPaymentMax
	case reliabilityClassAuth:
		return m.cfg.RateLimitAuthMax
	case reliabilityClassWebhook:
		return m.cfg.RateLimitWebhookMax
	case reliabilityClassTelemetry:
		return m.cfg.RateLimitTelemetryMax
	default:
		return m.cfg.RateLimitDefaultMax
	}
}

func isReliabilityRateLimitExempt(path string, r *http.Request) bool {
	lower := strings.ToLower(strings.TrimSpace(path))
	if lower == "/v1/health" || lower == "/health" || strings.HasPrefix(lower, "/v1/health/") {
		return true
	}
	secret := strings.TrimSpace(os.Getenv("LOAD_BOOTSTRAP_SECRET"))
	if secret == "" || r == nil {
		return false
	}
	if strings.TrimSpace(r.Header.Get(loadBootstrapHeader)) != secret {
		return false
	}
	return strings.HasPrefix(lower, "/v1/auth/")
}

func classifyReliabilityPath(path string) reliabilityClass {
	lower := strings.ToLower(strings.TrimSpace(path))
	switch {
	case strings.HasPrefix(lower, "/v1/webhooks/"):
		return reliabilityClassWebhook
	case strings.HasPrefix(lower, "/v1/payment/") || strings.HasPrefix(lower, "/v1/checkout/"):
		return reliabilityClassPayment
	case strings.HasPrefix(lower, "/v1/auth/"):
		return reliabilityClassAuth
	case strings.HasPrefix(lower, "/ws/telemetry") || strings.HasPrefix(lower, "/v1/telemetry/"):
		return reliabilityClassTelemetry
	default:
		return reliabilityClassOperational
	}
}

func reliabilityActorKey(r *http.Request) string {
	if r == nil {
		return "anonymous"
	}
	if claims, ok := auth.FromContext(r.Context()); ok {
		subject := strings.TrimSpace(claims.Subject)
		if subject != "" {
			return "sub:" + subject
		}
	}
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return "ip:" + ip
		}
	}
	if host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil && host != "" {
		return "ip:" + host
	}
	if strings.TrimSpace(r.RemoteAddr) != "" {
		return "ip:" + strings.TrimSpace(r.RemoteAddr)
	}
	return "anonymous"
}

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusCapturingResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusCapturingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("statusCapturingResponseWriter: underlying ResponseWriter does not implement http.Hijacker")
}

func (w *statusCapturingResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

type priorityGuard struct {
	sem             chan struct{}
	criticalTimeout time.Duration
}

func newPriorityGuard(maxInFlight int, criticalTimeout time.Duration) *priorityGuard {
	if maxInFlight <= 0 {
		maxInFlight = 1
	}
	if criticalTimeout <= 0 {
		criticalTimeout = 250 * time.Millisecond
	}
	return &priorityGuard{
		sem:             make(chan struct{}, maxInFlight),
		criticalTimeout: criticalTimeout,
	}
}

func (g *priorityGuard) Acquire(ctx context.Context, class reliabilityClass) bool {
	if g == nil {
		return true
	}
	if class.isCritical() {
		timer := time.NewTimer(g.criticalTimeout)
		defer timer.Stop()
		select {
		case g.sem <- struct{}{}:
			return true
		case <-ctx.Done():
			return false
		case <-timer.C:
			return false
		}
	}

	select {
	case g.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

func (g *priorityGuard) Release() {
	if g == nil {
		return
	}
	select {
	case <-g.sem:
	default:
	}
}

type fixedWindowRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]rateBucket
}

type rateBucket struct {
	windowStart time.Time
	count       int
}

func newFixedWindowRateLimiter() *fixedWindowRateLimiter {
	return &fixedWindowRateLimiter{buckets: make(map[string]rateBucket)}
}

func (l *fixedWindowRateLimiter) Allow(key string, max int, window time.Duration, now time.Time) (bool, int, int) {
	if l == nil {
		return true, max, 0
	}
	if max <= 0 || window <= 0 {
		return true, max, 0
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	bucket := l.buckets[key]
	if bucket.windowStart.IsZero() || now.Sub(bucket.windowStart) >= window {
		bucket = rateBucket{windowStart: now, count: 0}
	}

	if bucket.count >= max {
		retry := int(math.Ceil(window.Seconds() - now.Sub(bucket.windowStart).Seconds()))
		if retry < 1 {
			retry = 1
		}
		l.buckets[key] = bucket
		return false, 0, retry
	}

	bucket.count++
	l.buckets[key] = bucket
	return true, max - bucket.count, 0
}

type requestCircuitBreaker struct {
	mu             sync.Mutex
	threshold      int
	openDuration   time.Duration
	failureByClass map[reliabilityClass]int
	openUntil      map[reliabilityClass]time.Time
}

func newRequestCircuitBreaker(threshold int, openDuration time.Duration) *requestCircuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if openDuration <= 0 {
		openDuration = 15 * time.Second
	}
	return &requestCircuitBreaker{
		threshold:      threshold,
		openDuration:   openDuration,
		failureByClass: make(map[reliabilityClass]int),
		openUntil:      make(map[reliabilityClass]time.Time),
	}
}

func (b *requestCircuitBreaker) ShouldReject(class reliabilityClass, now time.Time) bool {
	if b == nil || class == reliabilityClassWebhook {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	until := b.openUntil[class]
	return now.Before(until)
}

func (b *requestCircuitBreaker) RetryAfterSeconds(class reliabilityClass, now time.Time) int {
	if b == nil {
		return 1
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	until := b.openUntil[class]
	retry := int(math.Ceil(until.Sub(now).Seconds()))
	if retry < 1 {
		retry = 1
	}
	return retry
}

func (b *requestCircuitBreaker) Observe(class reliabilityClass, statusCode int, now time.Time) {
	if b == nil || class == reliabilityClassWebhook {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	if statusCode >= http.StatusInternalServerError {
		b.failureByClass[class]++
		if b.failureByClass[class] >= b.threshold {
			b.openUntil[class] = now.Add(b.openDuration)
			b.failureByClass[class] = 0
		}
		return
	}

	b.failureByClass[class] = 0
}
