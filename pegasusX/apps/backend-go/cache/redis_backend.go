package cache

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisBackend binds Cache Backend to a Redis client.
type RedisBackend struct {
	client *redis.Client
}

// NewRedisBackend constructs a Redis-backed cache backend from either a raw
// host:port address or redis:// URL.
func NewRedisBackend(addr string) (*RedisBackend, error) {
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return nil, fmt.Errorf("redis backend: addr required")
	}
	var (
		opts *redis.Options
		err  error
	)
	if strings.HasPrefix(trimmed, "redis://") || strings.HasPrefix(trimmed, "rediss://") {
		opts, err = redis.ParseURL(trimmed)
		if err != nil {
			return nil, fmt.Errorf("redis backend: parse url: %w", err)
		}
	} else {
		opts = &redis.Options{Addr: trimmed}
	}
	return &RedisBackend{client: redis.NewClient(opts)}, nil
}

// Ping verifies connectivity.
func (r *RedisBackend) Ping(ctx context.Context) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("redis backend: nil client")
	}
	return r.client.Ping(ctx).Err()
}

// Close closes underlying client resources.
func (r *RedisBackend) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

func (r *RedisBackend) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if r == nil || r.client == nil {
		return nil, false, nil
	}
	value, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func (r *RedisBackend) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisBackend) Delete(ctx context.Context, keys ...string) error {
	if r == nil || r.client == nil || len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisBackend) Publish(ctx context.Context, channel string, payload []byte) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Publish(ctx, channel, payload).Err()
}

func (r *RedisBackend) Subscribe(ctx context.Context, channel string) (<-chan []byte, func(), error) {
	if r == nil || r.client == nil {
		return nil, func() {}, fmt.Errorf("redis backend: nil client")
	}
	pubsub := r.client.Subscribe(ctx, channel)
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, func() {}, fmt.Errorf("redis backend: subscribe receive: %w", err)
	}
	in := pubsub.Channel()
	out := make(chan []byte, 128)
	var once sync.Once
	cancel := func() {
		once.Do(func() {
			_ = pubsub.Close()
		})
	}
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				cancel()
				return
			case msg, ok := <-in:
				if !ok {
					return
				}
				payload := []byte(msg.Payload)
				select {
				case out <- payload:
				default:
				}
			}
		}
	}()
	return out, cancel, nil
}
