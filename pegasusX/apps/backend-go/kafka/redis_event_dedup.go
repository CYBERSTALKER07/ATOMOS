package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const eventDedupKeyPrefix = "dedup:event:"

// RedisEventDedup uses SETNX with TTL for cross-pod idempotency.
type RedisEventDedup struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisEventDedup builds a Redis-backed dedup store.
func NewRedisEventDedup(client *redis.Client, ttl time.Duration) *RedisEventDedup {
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	return &RedisEventDedup{client: client, ttl: ttl}
}

// ShouldProcess returns false when the event key already exists.
func (d *RedisEventDedup) ShouldProcess(ctx context.Context, key string) (bool, error) {
	if d == nil || d.client == nil || key == "" {
		return true, nil
	}
	ok, err := d.client.SetNX(ctx, eventDedupKeyPrefix+key, "1", d.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis event dedup: %w", err)
	}
	return ok, nil
}
