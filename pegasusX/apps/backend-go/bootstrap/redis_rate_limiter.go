package bootstrap

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var redisRateLimitScript = redis.NewScript(`
local key = KEYS[1]
local max_requests = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local req_id = ARGV[4]

local clear_before = now_ms - window_ms

redis.call('ZREMRANGEBYSCORE', key, 0, clear_before)
local current_requests = redis.call('ZCARD', key)

if current_requests >= max_requests then
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local retry_after_ms = window_ms
    if oldest and oldest[2] then
        retry_after_ms = (tonumber(oldest[2]) + window_ms) - now_ms
    end
    local retry_after_sec = math.ceil(retry_after_ms / 1000)
    if retry_after_sec < 1 then retry_after_sec = 1 end
    return {0, 0, retry_after_sec}
end

redis.call('ZADD', key, now_ms, req_id)
redis.call('PEXPIRE', key, window_ms)

local remaining = max_requests - (current_requests + 1)
return {1, remaining, 0}
`)

type redisRateLimiter struct {
	client   *redis.Client
	fallback *fixedWindowRateLimiter
}

func newRedisRateLimiter(client *redis.Client) *redisRateLimiter {
	return &redisRateLimiter{
		client:   client,
		fallback: newFixedWindowRateLimiter(),
	}
}

// Close cleanly stops the fallback limiter background sweep.
func (l *redisRateLimiter) Close() {
	if l == nil {
		return
	}
	if l.fallback != nil {
		l.fallback.Close()
	}
}

func (l *redisRateLimiter) Allow(key string, max int, window time.Duration, now time.Time) (bool, int, int) {
	if l == nil || max <= 0 || window <= 0 {
		return true, max, 0
	}
	if l.client == nil {
		if l.fallback != nil {
			return l.fallback.Allow(key, max, window, now)
		}
		return true, max, 0
	}

	windowMs := window.Milliseconds()
	if windowMs < 1000 {
		windowMs = 1000
	}
	if now.IsZero() {
		now = time.Now()
	}
	nowMs := now.UnixMilli()
	reqID := uuid.New().String()

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	res, err := redisRateLimitScript.Run(ctx, l.client, []string{"{rl:" + key + "}"}, max, windowMs, nowMs, reqID).Int64Slice()
	if err != nil || len(res) < 3 {
		// Clean fallback to local in-memory rate limiter on Redis failure or timeout (SO-5, CG-4)
		if l.fallback != nil {
			return l.fallback.Allow(key, max, window, now)
		}
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
