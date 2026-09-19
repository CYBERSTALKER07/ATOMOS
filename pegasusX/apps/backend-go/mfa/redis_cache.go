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
	key := fmt.Sprintf("mfa:used:%s:%d", subject, step)
	return c.client.Set(ctx, key, "1", 90*time.Second).Err()
}

func (c *RedisReplayCache) IsUsed(ctx context.Context, subject string, step uint64) (bool, error) {
	key := fmt.Sprintf("mfa:used:%s:%d", subject, step)
	res, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return res == "1", nil
}
