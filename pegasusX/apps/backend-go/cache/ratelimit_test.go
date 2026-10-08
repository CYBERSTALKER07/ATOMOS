package cache

import (
	"context"
	"testing"
	"time"
)

func TestSlidingWindowLimiter_Allow(t *testing.T) {
	backend := NewInMemoryBackend()
	c := New(backend, nil)
	limiter := NewSlidingWindowLimiter(c)
	ctx := context.Background()

	key := Key(PrefixRateLimit, "user_123", "orders")
	limit := int64(3)
	window := 100 * time.Millisecond

	// 1st request
	res, err := limiter.Allow(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Allowed || res.Remaining != 2 {
		t.Errorf("expected allowed=true, remaining=2, got allowed=%v, remaining=%d", res.Allowed, res.Remaining)
	}

	// 2nd request
	res, err = limiter.Allow(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Allowed || res.Remaining != 1 {
		t.Errorf("expected allowed=true, remaining=1, got allowed=%v, remaining=%d", res.Allowed, res.Remaining)
	}

	// 3rd request
	res, err = limiter.Allow(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Allowed || res.Remaining != 0 {
		t.Errorf("expected allowed=true, remaining=0, got allowed=%v, remaining=%d", res.Allowed, res.Remaining)
	}

	// 4th request (should be rejected)
	res, err = limiter.Allow(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Allowed || res.Remaining != 0 {
		t.Errorf("expected allowed=false, remaining=0, got allowed=%v, remaining=%d", res.Allowed, res.Remaining)
	}
	if res.RetryAfter <= 0 {
		t.Errorf("expected retryAfter > 0, got %v", res.RetryAfter)
	}

	// Wait for window to slide/expire
	time.Sleep(120 * time.Millisecond)

	// 5th request should be allowed again
	res, err = limiter.Allow(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Allowed || res.Remaining != 2 {
		t.Errorf("expected allowed=true after window slide, got allowed=%v, remaining=%d", res.Allowed, res.Remaining)
	}
}

func TestSlidingWindowLimiter_AllowN(t *testing.T) {
	backend := NewInMemoryBackend()
	c := New(backend, nil)
	limiter := NewSlidingWindowLimiter(c)
	ctx := context.Background()

	key := Key(PrefixRateLimit, "api_key_456", "batch")
	limit := int64(10)
	window := 100 * time.Millisecond

	// Consume 8 units
	res, err := limiter.AllowN(ctx, key, limit, window, 8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Allowed || res.Remaining != 2 {
		t.Errorf("expected allowed=true, remaining=2, got allowed=%v, remaining=%d", res.Allowed, res.Remaining)
	}

	// Attempt to consume 3 units (2 + 3 = 11 > 10, rejected)
	res, err = limiter.AllowN(ctx, key, limit, window, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Allowed {
		t.Errorf("expected allowed=false for cost exceeding limit, got %v", res.Allowed)
	}

	// Reset
	if err := limiter.Reset(ctx, key); err != nil {
		t.Fatalf("reset failed: %v", err)
	}

	// Now 10 units allowed immediately
	res, err = limiter.AllowN(ctx, key, limit, window, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Allowed || res.Remaining != 0 {
		t.Errorf("expected allowed=true after reset, got %v", res.Allowed)
	}
}
