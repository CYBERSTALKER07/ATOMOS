package circuit

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestBreaker_DefaultConfig(t *testing.T) {
	b := New("test-dep", Config{})
	if b.Name() != "test-dep" {
		t.Fatalf("expected name test-dep, got %s", b.Name())
	}
	if b.State() != StateClosed {
		t.Fatalf("expected StateClosed, got %v", b.State())
	}
	if b.cfg.FailureThreshold != 5 {
		t.Fatalf("expected FailureThreshold 5, got %d", b.cfg.FailureThreshold)
	}
	if b.cfg.FailureWindow != 30*time.Second {
		t.Fatalf("expected FailureWindow 30s, got %v", b.cfg.FailureWindow)
	}
	if b.cfg.OpenDuration != 60*time.Second {
		t.Fatalf("expected OpenDuration 60s, got %v", b.cfg.OpenDuration)
	}
}

func TestBreaker_SuccessKeepsClosed(t *testing.T) {
	b := New("test-dep", Config{FailureThreshold: 3})
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		err := b.Do(ctx, func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error on success call: %v", err)
		}
	}

	if b.State() != StateClosed {
		t.Fatalf("expected StateClosed after successes, got %v", b.State())
	}
}

func TestBreaker_TripToOpenAndRejectCalls(t *testing.T) {
	fakeNow := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	b := New("test-dep", Config{
		FailureThreshold: 3,
		FailureWindow:    10 * time.Second,
		OpenDuration:     30 * time.Second,
	})
	b.now = func() time.Time { return fakeNow }

	ctx := context.Background()
	testErr := errors.New("upstream failed")

	// 2 failures: still closed
	for i := 0; i < 2; i++ {
		err := b.Do(ctx, func(ctx context.Context) error {
			return testErr
		})
		if !errors.Is(err, testErr) {
			t.Fatalf("expected testErr, got %v", err)
		}
		if b.State() != StateClosed {
			t.Fatalf("expected StateClosed at %d failures, got %v", i+1, b.State())
		}
	}

	// 3rd failure: trips to open
	err := b.Do(ctx, func(ctx context.Context) error {
		return testErr
	})
	if !errors.Is(err, testErr) {
		t.Fatalf("expected testErr, got %v", err)
	}
	if b.State() != StateOpen {
		t.Fatalf("expected StateOpen after threshold failures, got %v", b.State())
	}

	// Subsequent calls must be fast-rejected with ErrUpstreamUnavailable without calling fn
	called := false
	err = b.Do(ctx, func(ctx context.Context) error {
		called = true
		return nil
	})
	if !errors.Is(err, ErrUpstreamUnavailable) {
		t.Fatalf("expected ErrUpstreamUnavailable, got %v", err)
	}
	if called {
		t.Fatalf("fn must not be called when circuit breaker is open")
	}
}

func TestBreaker_FailureWindowExpiryResetsCount(t *testing.T) {
	fakeNow := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	b := New("test-dep", Config{
		FailureThreshold: 2,
		FailureWindow:    5 * time.Second,
		OpenDuration:     30 * time.Second,
	})
	b.now = func() time.Time { return fakeNow }

	ctx := context.Background()
	failErr := errors.New("transient fail")

	// 1 failure at T=0
	_ = b.Do(ctx, func(ctx context.Context) error { return failErr })
	if b.State() != StateClosed {
		t.Fatalf("expected StateClosed, got %v", b.State())
	}

	// Advance time past FailureWindow
	fakeNow = fakeNow.Add(6 * time.Second)

	// Another failure at T=6s should reset previous failure, total count = 1
	_ = b.Do(ctx, func(ctx context.Context) error { return failErr })
	if b.State() != StateClosed {
		t.Fatalf("expected StateClosed because window expired, got %v", b.State())
	}
}

func TestBreaker_HalfOpenTransitions(t *testing.T) {
	fakeNow := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	b := New("test-dep", Config{
		FailureThreshold: 1,
		FailureWindow:    10 * time.Second,
		OpenDuration:     20 * time.Second,
	})
	b.now = func() time.Time { return fakeNow }
	ctx := context.Background()

	// Trip breaker
	_ = b.Do(ctx, func(ctx context.Context) error { return errors.New("boom") })
	if b.State() != StateOpen {
		t.Fatalf("expected StateOpen, got %v", b.State())
	}

	// Advance time by 19s (not expired yet)
	fakeNow = fakeNow.Add(19 * time.Second)
	if b.State() != StateOpen {
		t.Fatalf("expected StateOpen before open duration, got %v", b.State())
	}

	// Advance time to 21s (expired)
	fakeNow = fakeNow.Add(2 * time.Second)
	if b.State() != StateHalfOpen {
		t.Fatalf("expected StateHalfOpen after open duration, got %v", b.State())
	}

	// Case A: failure in half-open immediately reopens breaker
	_ = b.Do(ctx, func(ctx context.Context) error { return errors.New("boom again") })
	if b.State() != StateOpen {
		t.Fatalf("expected StateOpen after failure in half-open, got %v", b.State())
	}

	// Advance time past open duration again
	fakeNow = fakeNow.Add(21 * time.Second)
	if b.State() != StateHalfOpen {
		t.Fatalf("expected StateHalfOpen, got %v", b.State())
	}

	// Case B: success in half-open closes breaker
	err := b.Do(ctx, func(ctx context.Context) error { return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.State() != StateClosed {
		t.Fatalf("expected StateClosed after recovery, got %v", b.State())
	}
}

func TestBreaker_ConcurrentSafe(t *testing.T) {
	b := New("test-concurrent", Config{
		FailureThreshold: 50,
		FailureWindow:    5 * time.Second,
		OpenDuration:     5 * time.Second,
	})
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = b.Do(ctx, func(ctx context.Context) error {
				if id%2 == 0 {
					return errors.New("err")
				}
				return nil
			})
			_ = b.State().String()
		}(i)
	}
	wg.Wait()
}
