package mfa

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisReplayCache struct {
	client *redis.Client
}

func NewRedisReplayCache(client *redis.Client) *RedisReplayCache {
	if client == nil {
		return nil
	}
	return &RedisReplayCache{client: client}
}

func (c *RedisReplayCache) MarkUsed(ctx context.Context, subject string, step uint64) error {
	key := fmt.Sprintf("{mfa:used:%s}:%d", subject, step)
	return c.client.Set(ctx, key, "1", 90*time.Second).Err()
}

func (c *RedisReplayCache) IsUsed(ctx context.Context, subject string, step uint64) (bool, error) {
	key := fmt.Sprintf("{mfa:used:%s}:%d", subject, step)
	res, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return res == "1", nil
}

// IncrFailedAttempt increments failed TOTP count in a rolling window and sets expiry on first fail.
func (c *RedisReplayCache) IncrFailedAttempt(ctx context.Context, subject string, window time.Duration) (int64, error) {
	key := fmt.Sprintf("{mfa:fail:%s}", subject)
	n, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if n == 1 {
		_ = c.client.Expire(ctx, key, window).Err()
	}
	return n, nil
}

// ResetFailedAttempts clears the failure counter upon successful TOTP verification.
func (c *RedisReplayCache) ResetFailedAttempts(ctx context.Context, subject string) error {
	return c.client.Del(ctx, fmt.Sprintf("{mfa:fail:%s}", subject)).Err()
}

// GetFailedAttempts returns the current failed TOTP count for subject.
func (c *RedisReplayCache) GetFailedAttempts(ctx context.Context, subject string) (int64, error) {
	n, err := c.client.Get(ctx, fmt.Sprintf("{mfa:fail:%s}", subject)).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return n, err
}
