package bootstrap

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/redis/go-redis/v9"
)

var redisRateLimitScript = redis.NewScript(`
local key = KEYS[1]
local max_requests = tonumber(ARGV[1])
local window_seconds = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local req_id = ARGV[4]

local window_ms = window_seconds * 1000
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
redis.call('EXPIRE', key, window_seconds)

local remaining = max_requests - (current_requests + 1)
return {1, remaining, 0}
`)

type redisRateLimiter struct {
	client *redis.Client
}

func newRedisRateLimiter(client *redis.Client) *redisRateLimiter {
	return &redisRateLimiter{client: client}
}

func (l *redisRateLimiter) Allow(key string, max int, window time.Duration, now time.Time) (bool, int, int) {
	if l == nil || l.client == nil || max <= 0 || window <= 0 {
		return true, max, 0
	}
	windowSec := int64(math.Ceil(window.Seconds()))
	if windowSec < 1 {
		windowSec = 1
	}
	if now.IsZero() {
		now = time.Now()
	}
	nowMs := now.UnixMilli()
	reqID := uuid.New().String()

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	res, err := redisRateLimitScript.Run(ctx, l.client, []string{"rl:" + key}, max, windowSec, nowMs, reqID).Int64Slice()
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
