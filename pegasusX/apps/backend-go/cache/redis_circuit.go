package cache

import (
	"context"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
	"github.com/redis/go-redis/v9"
)

// CircuitBreakerBackend wraps a primary backend (Redis) and optionally falls
// back to an in-memory backend when the circuit opens due to failures.
//
// When FailClosed is true (production / REQUIRE_INFRA_ADAPTERS), open-circuit
// and primary errors surface to callers instead of silently serving memory
// cache — preventing split-brain invalidation and stale multi-pod state.
type CircuitBreakerBackend struct {
	primary    Backend
	fallback   Backend
	breaker    *circuit.Breaker
	FailClosed bool
}

// NewCircuitBreakerBackend constructs a circuit breaker backend with memory fallback.
func NewCircuitBreakerBackend(primary, fallback Backend) *CircuitBreakerBackend {
	return NewCircuitBreakerBackendWithMode(primary, fallback, false)
}

// NewCircuitBreakerBackendWithMode constructs a circuit breaker backend.
// failClosed=true rejects operations when Redis is unavailable (enterprise default).
func NewCircuitBreakerBackendWithMode(primary, fallback Backend, failClosed bool) *CircuitBreakerBackend {
	cfg := circuit.Config{
		FailureThreshold: 5,
		FailureWindow:    10 * time.Second,
		OpenDuration:     30 * time.Second,
	}
	return &CircuitBreakerBackend{
		primary:    primary,
		fallback:   fallback,
		breaker:    circuit.New("redis-cache", cfg),
		FailClosed: failClosed,
	}
}

func (c *CircuitBreakerBackend) useFallback(err error) bool {
	if c == nil || c.FailClosed || c.fallback == nil {
		return false
	}
	return err == circuit.ErrUpstreamUnavailable
}

// Client exposes the underlying Redis client if available.
func (c *CircuitBreakerBackend) Client() *redis.Client {
	if adapter, ok := c.primary.(interface{ Client() *redis.Client }); ok {
		return adapter.Client()
	}
	return nil
}

// PoolStats exposes the underlying Redis pool stats if available.
func (c *CircuitBreakerBackend) PoolStats() *redis.PoolStats {
	if adapter, ok := c.primary.(interface{ PoolStats() *redis.PoolStats }); ok {
		return adapter.PoolStats()
	}
	return nil
}

// Ping checks the primary.
func (c *CircuitBreakerBackend) Ping(ctx context.Context) error {
	var err error
	circuitErr := c.breaker.Do(ctx, func(ctx context.Context) error {
		if pinger, ok := c.primary.(interface{ Ping(context.Context) error }); ok {
			err = pinger.Ping(ctx)
			return err
		}
		return nil
	})
	if circuitErr == circuit.ErrUpstreamUnavailable {
		return circuitErr
	}
	return err
}

// Get implements Backend.
func (c *CircuitBreakerBackend) Get(ctx context.Context, key string) ([]byte, bool, error) {
	var val []byte
	var ok bool
	var getErr error

	err := c.breaker.Do(ctx, func(ctx context.Context) error {
		val, ok, getErr = c.primary.Get(ctx, key)
		return getErr
	})

	if c.useFallback(err) {
		return c.fallback.Get(ctx, key)
	}
	if err == circuit.ErrUpstreamUnavailable {
		return nil, false, err
	}
	return val, ok, err
}

// Set implements Backend.
func (c *CircuitBreakerBackend) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	err := c.breaker.Do(ctx, func(ctx context.Context) error {
		return c.primary.Set(ctx, key, value, ttl)
	})

	if c.useFallback(err) {
		return c.fallback.Set(ctx, key, value, ttl)
	}
	return err
}

// Delete implements Backend.
func (c *CircuitBreakerBackend) Delete(ctx context.Context, keys ...string) error {
	err := c.breaker.Do(ctx, func(ctx context.Context) error {
		return c.primary.Delete(ctx, keys...)
	})

	if c.useFallback(err) {
		return c.fallback.Delete(ctx, keys...)
	}
	return err
}

// Publish implements Backend.
func (c *CircuitBreakerBackend) Publish(ctx context.Context, channel string, message []byte) error {
	err := c.breaker.Do(ctx, func(ctx context.Context) error {
		return c.primary.Publish(ctx, channel, message)
	})

	if c.useFallback(err) {
		return c.fallback.Publish(ctx, channel, message)
	}
	return err
}

// Subscribe implements Backend.
func (c *CircuitBreakerBackend) Subscribe(ctx context.Context, channel string) (<-chan []byte, func(), error) {
	var out <-chan []byte
	var cancel func()
	var subErr error

	err := c.breaker.Do(ctx, func(ctx context.Context) error {
		out, cancel, subErr = c.primary.Subscribe(ctx, channel)
		return subErr
	})

	if c.useFallback(err) {
		return c.fallback.Subscribe(ctx, channel)
	}
	return out, cancel, err
}

// Close implements Backend.
func (c *CircuitBreakerBackend) Close() error {
	if closer, ok := c.primary.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
	if closer, ok := c.fallback.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
	return nil
}

type perimeterSetPrimary interface {
	ReplaceSet(ctx context.Context, key string, members []string, ttl time.Duration) error
	SIsMember(ctx context.Context, key string, member string) (bool, error)
	Exists(ctx context.Context, key string) (bool, error)
}

// ReplaceSet delegates perimeter set writes to the primary Redis backend.
func (c *CircuitBreakerBackend) ReplaceSet(ctx context.Context, key string, members []string, ttl time.Duration) error {
	primary, ok := c.primary.(perimeterSetPrimary)
	if !ok {
		return circuit.ErrUpstreamUnavailable
	}
	var setErr error
	err := c.breaker.Do(ctx, func(ctx context.Context) error {
		setErr = primary.ReplaceSet(ctx, key, members, ttl)
		return setErr
	})
	if err == circuit.ErrUpstreamUnavailable {
		return err
	}
	return setErr
}

// SIsMember delegates perimeter membership checks to the primary Redis backend.
func (c *CircuitBreakerBackend) SIsMember(ctx context.Context, key string, member string) (bool, error) {
	primary, ok := c.primary.(perimeterSetPrimary)
	if !ok {
		return false, circuit.ErrUpstreamUnavailable
	}
	var memberOK bool
	var memberErr error
	err := c.breaker.Do(ctx, func(ctx context.Context) error {
		memberOK, memberErr = primary.SIsMember(ctx, key, member)
		return memberErr
	})
	if err == circuit.ErrUpstreamUnavailable {
		return false, err
	}
	return memberOK, memberErr
}

// Exists delegates perimeter key presence checks to the primary Redis backend.
func (c *CircuitBreakerBackend) Exists(ctx context.Context, key string) (bool, error) {
	primary, ok := c.primary.(perimeterSetPrimary)
	if !ok {
		return false, circuit.ErrUpstreamUnavailable
	}
	var exists bool
	var existsErr error
	err := c.breaker.Do(ctx, func(ctx context.Context) error {
		exists, existsErr = primary.Exists(ctx, key)
		return existsErr
	})
	if err == circuit.ErrUpstreamUnavailable {
		return false, err
	}
	return exists, existsErr
}

func (c *CircuitBreakerBackend) IncrBy(ctx context.Context, key string, amount int64) (int64, error) {
	if c == nil || c.primary == nil {
		return 0, nil
	}
	var res int64
	var innerErr error
	err := c.breaker.Do(ctx, func(ctx context.Context) error {
		res, innerErr = c.primary.IncrBy(ctx, key, amount)
		if innerErr == redis.Nil {
			return nil
		}
		return innerErr
	})
	if c.useFallback(err) {
		return c.fallback.IncrBy(ctx, key, amount)
	}
	if err != nil {
		return 0, err
	}
	return res, innerErr
}

func (c *CircuitBreakerBackend) DecrBy(ctx context.Context, key string, amount int64) (int64, error) {
	if c == nil || c.primary == nil {
		return 0, nil
	}
	var res int64
	var innerErr error
	err := c.breaker.Do(ctx, func(ctx context.Context) error {
		res, innerErr = c.primary.DecrBy(ctx, key, amount)
		if innerErr == redis.Nil {
			return nil
		}
		return innerErr
	})
	if c.useFallback(err) {
		return c.fallback.DecrBy(ctx, key, amount)
	}
	if err != nil {
		return 0, err
	}
	return res, innerErr
}
