// Package circuit implements outbound circuit breakers for external dependencies.
package circuit

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrUpstreamUnavailable is returned when the breaker is OPEN.
var ErrUpstreamUnavailable = errors.New("circuit: upstream unavailable")

// State is the circuit breaker state.
type State int

const (
	StateClosed   State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// Config tunes breaker behaviour.
type Config struct {
	FailureThreshold int
	FailureWindow    time.Duration
	OpenDuration     time.Duration
}

func (c *Config) applyDefaults() {
	if c.FailureThreshold <= 0 {
		c.FailureThreshold = 5
	}
	if c.FailureWindow <= 0 {
		c.FailureWindow = 30 * time.Second
	}
	if c.OpenDuration <= 0 {
		c.OpenDuration = 60 * time.Second
	}
}

// Breaker protects a single upstream dependency.
type Breaker struct {
	mu          sync.Mutex
	name        string
	cfg         Config
	state       State
	failures    int
	lastFailure time.Time
	openUntil   time.Time
	now         func() time.Time
}

// New constructs a Breaker in the CLOSED state.
func New(name string, cfg Config) *Breaker {
	cfg.applyDefaults()
	return &Breaker{name: name, cfg: cfg, state: StateClosed, now: time.Now}
}

// Name returns the upstream identifier.
func (b *Breaker) Name() string { return b.name }

// State returns the current state after applying time-based transitions.
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.maybeTransition(b.now())
	return b.state
}

// Do executes fn through the breaker.
func (b *Breaker) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := b.beforeCall(); err != nil {
		return err
	}
	err := fn(ctx)
	b.afterCall(err)
	return err
}

func (b *Breaker) beforeCall() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	b.maybeTransition(now)
	switch b.state {
	case StateOpen:
		return ErrUpstreamUnavailable
	default:
		return nil
	}
}

func (b *Breaker) afterCall(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	if err != nil {
		b.recordFailure(now)
		return
	}
	b.recordSuccess()
}

func (b *Breaker) recordFailure(now time.Time) {
	if !b.lastFailure.IsZero() && now.Sub(b.lastFailure) > b.cfg.FailureWindow {
		b.failures = 0
	}
	b.failures++
	b.lastFailure = now
	if b.state == StateHalfOpen || b.failures >= b.cfg.FailureThreshold {
		b.state = StateOpen
		b.openUntil = now.Add(b.cfg.OpenDuration)
		b.failures = 0
	}
}

func (b *Breaker) recordSuccess() {
	b.state = StateClosed
	b.failures = 0
	b.lastFailure = time.Time{}
}

func (b *Breaker) maybeTransition(now time.Time) {
	if b.state == StateOpen && !now.Before(b.openUntil) {
		b.state = StateHalfOpen
	}
}
