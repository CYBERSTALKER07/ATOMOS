package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const jwtRevokedKeyPrefix = "jwt:revoked:"

// RedisRevocationStore stores revoked jti values in Redis with TTL = remaining token life.
type RedisRevocationStore struct {
	client *redis.Client
}

// NewRedisRevocationStore wraps a go-redis client. client must be non-nil.
func NewRedisRevocationStore(client *redis.Client) *RedisRevocationStore {
	return &RedisRevocationStore{client: client}
}

// Revoke marks jti revoked for ttl.
func (r *RedisRevocationStore) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	jti = trimJTI(jti)
	if jti == "" || r == nil || r.client == nil {
		return nil
	}
	if ttl < time.Second {
		ttl = time.Second
	}
	return r.client.Set(ctx, jwtRevokedKeyPrefix+jti, "1", ttl).Err()
}

// IsRevoked reports whether jti is on the Redis denylist.
func (r *RedisRevocationStore) IsRevoked(ctx context.Context, jti string) (bool, error) {
	jti = trimJTI(jti)
	if jti == "" || r == nil || r.client == nil {
		return false, nil
	}
	n, err := r.client.Exists(ctx, jwtRevokedKeyPrefix+jti).Result()
	if err != nil {
		return false, fmt.Errorf("jwt revoke exists: %w", err)
	}
	return n > 0, nil
}
