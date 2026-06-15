package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const redisKeyPrefix = "idem:"

// RedisStore persists idempotency records in Redis for cross-pod replay safety.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore returns a Store implementation using the provided Redis client.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
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

// Acquire implements Store.
func (s *RedisStore) Acquire(ctx context.Context, key, bodyHash string, ttl time.Duration) error {
	if s == nil || s.client == nil || key == "" {
		return nil
	}
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	rec := Record{
		BodyHash:   bodyHash,
		StatusCode: inProgressStatusCode,
		StoredAt:   time.Now(),
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("idempotency redis acquire encode: %w", err)
	}
	ok, err := s.client.SetNX(ctx, redisKeyPrefix+key, raw, ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrInProgress
	}
	return nil
}

// Release implements Store.
func (s *RedisStore) Release(ctx context.Context, key string) error {
	if s == nil || s.client == nil || key == "" {
		return nil
	}
	return s.client.Del(ctx, redisKeyPrefix+key).Err()
}
