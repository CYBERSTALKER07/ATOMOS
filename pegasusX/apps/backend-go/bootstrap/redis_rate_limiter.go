package bootstrap

import (
	"context"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisRateLimitScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[2])
end
if current > tonumber(ARGV[1]) then
  local ttl = redis.call('TTL', KEYS[1])
  if ttl < 0 then ttl = 0 end
  return {0, 0, ttl}
end
return {1, tonumber(ARGV[1]) - current, 0}
`)

type redisRateLimiter struct {
	client *redis.Client
}

func newRedisRateLimiter(client *redis.Client) *redisRateLimiter {
	return &redisRateLimiter{client: client}
}

func (l *redisRateLimiter) Allow(key string, max int, window time.Duration, _ time.Time) (bool, int, int) {
	if l == nil || l.client == nil || max <= 0 || window <= 0 {
		return true, max, 0
	}
	windowSec := int64(math.Ceil(window.Seconds()))
	if windowSec < 1 {
		windowSec = 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	res, err := redisRateLimitScript.Run(ctx, l.client, []string{"rl:" + key}, max, windowSec).Int64Slice()
	if err != nil || len(res) < 3 {
		return true, max, 0
	}
	allowed := res[0] == 1
	remaining := int(res[1])
	retryAfter := int(res[2])
	if retryAfter < 1 && !allowed {
		retryAfter = 1
	}
	return allowed, remaining, retryAfter
}
