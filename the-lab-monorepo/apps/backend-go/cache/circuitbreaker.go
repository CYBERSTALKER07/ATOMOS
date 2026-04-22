package cache

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ── Circuit Breaker ─────────────────────────────────────────────────────────
// Prevents cascading failures by short-circuiting requests to an unhealthy
// downstream service. State transitions:
//
//	CLOSED (normal) → OPEN (tripped after N failures) → HALF_OPEN (probe)
//
// State is tracked in Redis for multi-pod consistency. If Redis is unavailable,
// the breaker operates in-memory only (pod-local fallback).

// CircuitState represents the breaker state.
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // Normal operation
	CircuitOpen                         // Tripped — reject immediately
	CircuitHalfOpen                     // Probing — allow 1 request
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "CLOSED"
	case CircuitOpen:
		return "OPEN"
	case CircuitHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker tracks failure state for a named service.
type CircuitBreaker struct {
	Name string // Service name (e.g., "fcm", "telegram", "payment-gateway")

	FailureThreshold int           // Consecutive failures before tripping (default 5)
	OpenDuration     time.Duration // Time to wait before half-open probe (default 30s)

	mu             sync.Mutex
	state          CircuitState
	failureCount   int
	lastFailureAt  time.Time
	lastTransition time.Time
}

// NewCircuitBreaker creates a breaker with sensible defaults.
func NewCircuitBreaker(name string) *CircuitBreaker {
	return &CircuitBreaker{
		Name:             name,
		FailureThreshold: 5,
		OpenDuration:     30 * time.Second,
		state:            CircuitClosed,
	}
}

// Allow checks if a request should proceed. Returns false if the circuit is
// open (the caller should return 503 immediately).
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Check if cooldown has elapsed → transition to half-open
		if time.Since(cb.lastTransition) >= cb.OpenDuration {
			cb.state = CircuitHalfOpen
			cb.lastTransition = time.Now()
			log.Printf("[CIRCUIT_BREAKER] %s: OPEN → HALF_OPEN (probing)", cb.Name)
			return true
		}
		return false

	case CircuitHalfOpen:
		// Only allow 1 probe request — subsequent requests are rejected
		// until the probe resolves via RecordSuccess or RecordFailure
		return false

	default:
		return true
	}
}

// RecordSuccess resets the breaker to CLOSED state.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state != CircuitClosed {
		log.Printf("[CIRCUIT_BREAKER] %s: %s → CLOSED (success)", cb.Name, cb.state)
	}
	cb.state = CircuitClosed
	cb.failureCount = 0
	cb.lastTransition = time.Now()

	// Sync to Redis (best-effort, non-blocking)
	cb.syncToRedis("CLOSED")
}

// RecordFailure increments the failure counter and trips the breaker if threshold is reached.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureAt = time.Now()

	if cb.state == CircuitHalfOpen {
		// Probe failed — back to open
		cb.state = CircuitOpen
		cb.lastTransition = time.Now()
		log.Printf("[CIRCUIT_BREAKER] %s: HALF_OPEN → OPEN (probe failed, failures=%d)", cb.Name, cb.failureCount)
		cb.syncToRedis("OPEN")
		return
	}

	if cb.failureCount >= cb.FailureThreshold {
		cb.state = CircuitOpen
		cb.lastTransition = time.Now()
		log.Printf("[CIRCUIT_BREAKER] %s: CLOSED → OPEN (threshold=%d reached)", cb.Name, cb.FailureThreshold)
		cb.syncToRedis("OPEN")
	}
}

// State returns the current breaker state (for observability headers).
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	// Auto-transition from OPEN if cooldown elapsed (lazy check)
	if cb.state == CircuitOpen && time.Since(cb.lastTransition) >= cb.OpenDuration {
		cb.state = CircuitHalfOpen
		cb.lastTransition = time.Now()
	}
	return cb.state
}

// syncToRedis writes the breaker state to Redis for cross-pod visibility.
// Best-effort: if Redis is unavailable, the breaker still works pod-locally.
func (cb *CircuitBreaker) syncToRedis(state string) {
	if Client == nil {
		return
	}
	key := fmt.Sprintf("cb:%s", cb.Name)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	// Store state with expiry matching 2× open duration (auto-cleanup if pod dies)
	Client.Set(ctx, key, state, 2*cb.OpenDuration)
}

// Do executes the given function through the circuit breaker. If the circuit
// is open, it returns an error immediately without calling fn.
func (cb *CircuitBreaker) Do(fn func() error) error {
	if !cb.Allow() {
		return fmt.Errorf("circuit breaker %s is %s", cb.Name, cb.State())
	}

	err := fn()
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}
