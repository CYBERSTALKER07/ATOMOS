package bootstrap

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisRateLimiter_FallbackToInMemoryOnFailure(t *testing.T) {
	// Construct a redis rate limiter with an invalid address so network calls fail
	badClient := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:54321", // non-existent redis port
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
	})
	defer badClient.Close()

	limiter := newRedisRateLimiter(badClient)
	defer limiter.Close()

	now := time.Now()
	key := "test-actor-failover"
	maxRequests := 2
	window := time.Minute

	// First request: Redis will fail, falls back to in-memory -> allowed
	allowed, remaining, retry := limiter.Allow(key, maxRequests, window, now)
	if !allowed {
		t.Fatalf("first request should be allowed via fallback, remaining=%d, retry=%d", remaining, retry)
	}
	if remaining != 1 {
		t.Fatalf("expected remaining=1, got %d", remaining)
	}

	// Second request: allowed via fallback
	allowed2, remaining2, _ := limiter.Allow(key, maxRequests, window, now)
	if !allowed2 || remaining2 != 0 {
		t.Fatalf("second request should be allowed with remaining=0, got allowed=%v, remaining=%d", allowed2, remaining2)
	}

	// Third request: exceeded in-memory limit -> blocked
	allowed3, _, retry3 := limiter.Allow(key, maxRequests, window, now)
	if allowed3 {
		t.Fatalf("third request should be rejected via fallback rate limit, got allowed=true")
	}
	if retry3 < 1 {
		t.Fatalf("expected retry > 0, got %d", retry3)
	}
}

func TestRedisRateLimiter_CloseIdempotent(t *testing.T) {
	limiter := newRedisRateLimiter(nil)
	limiter.Close()
	limiter.Close()
}
