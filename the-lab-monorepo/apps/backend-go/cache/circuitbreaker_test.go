package cache

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedAllowsRequests(t *testing.T) {
	cb := NewCircuitBreaker("test-svc")
	if !cb.Allow() {
		t.Fatal("expected CLOSED breaker to allow requests")
	}
	if cb.State() != CircuitClosed {
		t.Fatalf("state = %s, want CLOSED", cb.State())
	}
}

func TestCircuitBreaker_TripsAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker("test-svc")
	cb.FailureThreshold = 3

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	if cb.State() != CircuitOpen {
		t.Fatalf("state = %s, want OPEN after %d failures", cb.State(), cb.FailureThreshold)
	}
	if cb.Allow() {
		t.Fatal("expected OPEN breaker to reject requests")
	}
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker("test-svc")
	cb.FailureThreshold = 1
	cb.OpenDuration = 10 * time.Millisecond

	cb.RecordFailure() // → OPEN
	if cb.State() != CircuitOpen {
		t.Fatalf("state = %s, want OPEN", cb.State())
	}

	time.Sleep(15 * time.Millisecond)
	if cb.State() != CircuitHalfOpen {
		t.Fatalf("state = %s, want HALF_OPEN after cooldown", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenProbeSuccess(t *testing.T) {
	cb := NewCircuitBreaker("test-svc")
	cb.FailureThreshold = 1
	cb.OpenDuration = 10 * time.Millisecond

	cb.RecordFailure()                // → OPEN
	time.Sleep(15 * time.Millisecond) // → HALF_OPEN on next check

	if !cb.Allow() {
		t.Fatal("expected HALF_OPEN breaker to allow one probe")
	}
	cb.RecordSuccess() // → CLOSED
	if cb.State() != CircuitClosed {
		t.Fatalf("state = %s, want CLOSED after successful probe", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenProbeFailure(t *testing.T) {
	cb := NewCircuitBreaker("test-svc")
	cb.FailureThreshold = 1
	cb.OpenDuration = 10 * time.Millisecond

	cb.RecordFailure()                // → OPEN
	time.Sleep(15 * time.Millisecond) // → HALF_OPEN on next check

	if !cb.Allow() {
		t.Fatal("expected HALF_OPEN breaker to allow probe")
	}
	cb.RecordFailure() // → OPEN again
	if cb.State() != CircuitOpen {
		t.Fatalf("state = %s, want OPEN after failed probe", cb.State())
	}
}

func TestCircuitBreaker_SuccessResetsCounter(t *testing.T) {
	cb := NewCircuitBreaker("test-svc")
	cb.FailureThreshold = 3

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess() // Reset

	// Two more failures should NOT trip (counter was reset)
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != CircuitClosed {
		t.Fatalf("state = %s, want CLOSED (counter was reset)", cb.State())
	}
}

func TestCircuitBreaker_Do(t *testing.T) {
	cb := NewCircuitBreaker("test-svc")
	cb.FailureThreshold = 2

	// Successful call
	err := cb.Do(func() error { return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Two failing calls → trips
	cb.Do(func() error { return errors.New("fail1") })
	cb.Do(func() error { return errors.New("fail2") })

	// Third call should be rejected by breaker
	err = cb.Do(func() error { return nil })
	if err == nil {
		t.Fatal("expected error from open breaker")
	}
}
