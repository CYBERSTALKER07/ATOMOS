package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const redisKeyPrefix = "idem:"

// RedisStore persists idempotency records in Redis for cross-pod replay safety.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStoreFromAddr connects to Redis and returns a Store implementation.
func NewRedisStoreFromAddr(addr string) (*RedisStore, error) {
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return nil, fmt.Errorf("idempotency redis store: addr required")
	}
	var (
		opts *redis.Options
		err  error
	)
	if strings.HasPrefix(trimmed, "redis://") || strings.HasPrefix(trimmed, "rediss://") {
		opts, err = redis.ParseURL(trimmed)
		if err != nil {
			return nil, fmt.Errorf("idempotency redis store: parse url: %w", err)
		}
	} else {
		opts = &redis.Options{Addr: trimmed}
	}
	return &RedisStore{client: redis.NewClient(opts)}, nil
}

// Load implements Store.
func (s *RedisStore) Load(ctx context.Context, key string) (Record, bool, error) {
	if s == nil || s.client == nil || key == "" {
		return Record{}, false, nil
	}
	raw, err := s.client.Get(ctx, redisKeyPrefix+key).Bytes()
	if err == redis.Nil {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, err
	}
	var rec Record
	if err := json.Unmarshal(raw, &rec); err != nil {
		return Record{}, false, fmt.Errorf("idempotency redis load decode: %w", err)
	}
	return rec, true, nil
}

// Save implements Store.
func (s *RedisStore) Save(ctx context.Context, key string, rec Record, ttl time.Duration) error {
	if s == nil || s.client == nil || key == "" {
		return nil
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("idempotency redis save encode: %w", err)
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return s.client.Set(ctx, redisKeyPrefix+key, raw, ttl).Err()
}
