package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const jwtRevokedKeyPrefix = "jwt:revoked:"
const jwtRevokedUserPrefix = "jwt:revoked_user:"

// RedisRevocationStore stores revoked jti and user values in Redis with TTL = remaining token life.
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

// RevokeUser marks userID revoked for ttl.
func (r *RedisRevocationStore) RevokeUser(ctx context.Context, userID string, ttl time.Duration) error {
	userID = trimJTI(userID)
	if userID == "" || r == nil || r.client == nil {
		return nil
	}
	if ttl < time.Second {
		ttl = time.Second
	}
	return r.client.Set(ctx, jwtRevokedUserPrefix+userID, "1", ttl).Err()
}

// IsUserRevoked reports whether userID is on the Redis denylist.
func (r *RedisRevocationStore) IsUserRevoked(ctx context.Context, userID string) (bool, error) {
	userID = trimJTI(userID)
	if userID == "" || r == nil || r.client == nil {
		return false, nil
	}
	n, err := r.client.Exists(ctx, jwtRevokedUserPrefix+userID).Result()
	if err != nil {
		return false, fmt.Errorf("jwt user revoke exists: %w", err)
	}
	return n > 0, nil
}
