package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimitResult contains the outcome of a rate limit check.
type RateLimitResult struct {
	Allowed    bool          // True if the request is permitted under the limit.
	Remaining  int64         // Number of requests remaining in the sliding window.
	Limit      int64         // Maximum permitted requests within the window.
	ResetAfter time.Duration // Time until the full window resets assuming no further requests.
	RetryAfter time.Duration // Time the caller must wait before retrying (0 if Allowed is true).
}

// SlidingWindowLimiter implements a high-precision sliding-window rate limiter
// backed by Redis Sorted Sets (redis-core, redis-connections).
//
// Unlike fixed-window counters, the sliding window prevents burst traffic at
// window boundaries (where 2x the limit could pass within 2 epsilons of the boundary).
type SlidingWindowLimiter struct {
	cache *Cache
	lua   *redis.Script
	mu    sync.Mutex
}

const slidingWindowLua = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local windowNanos = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])
local memberPrefix = ARGV[5]

local clearBefore = now - windowNanos
redis.call('ZREMRANGEBYSCORE', key, 0, clearBefore)
local current = redis.call('ZCARD', key)

if current + cost <= limit then
    for i = 1, cost do
        local member = memberPrefix .. ":" .. tostring(i) .. ":" .. tostring(now)
        redis.call('ZADD', key, now, member)
    end
    local ttlMs = math.ceil(windowNanos / 1000000)
    redis.call('PEXPIRE', key, ttlMs)
    local remaining = limit - (current + cost)
    return {1, remaining, 0}
else
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local retryAfterNanos = 0
    if #oldest >= 2 then
        local oldestScore = tonumber(oldest[2])
        retryAfterNanos = oldestScore + windowNanos - now
        if retryAfterNanos < 0 then retryAfterNanos = 0 end
    else
        retryAfterNanos = windowNanos
    end
    return {0, 0, retryAfterNanos}
end
`

// NewSlidingWindowLimiter creates a new rate limiter attached to the given Cache.
func NewSlidingWindowLimiter(c *Cache) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		cache: c,
		lua:   redis.NewScript(slidingWindowLua),
	}
}

// Allow evaluates a single-request cost against the limit in the sliding window.
func (l *SlidingWindowLimiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (*RateLimitResult, error) {
	return l.AllowN(ctx, key, limit, window, 1)
}

// AllowN evaluates a batch of requests (cost) against the limit in the sliding window.
func (l *SlidingWindowLimiter) AllowN(ctx context.Context, key string, limit int64, window time.Duration, cost int64) (*RateLimitResult, error) {
	if l == nil || l.cache == nil {
		return &RateLimitResult{Allowed: true, Remaining: limit, Limit: limit}, nil
	}
	if cost <= 0 {
		cost = 1
	}
	if limit <= 0 {
		return &RateLimitResult{Allowed: false, Remaining: 0, Limit: limit, RetryAfter: window}, nil
	}

	// Try atomic Lua script execution if the backend wraps a live Redis client
	if client := l.extractRedisClient(); client != nil {
		return l.allowLua(ctx, client, key, limit, window, cost)
	}

	// Fallback to SortedSetBackend methods (e.g. for InMemoryBackend or custom adapters)
	return l.allowFallback(ctx, key, limit, window, cost)
}

func (l *SlidingWindowLimiter) allowLua(ctx context.Context, client *redis.Client, key string, limit int64, window time.Duration, cost int64) (*RateLimitResult, error) {
	nowNanos := time.Now().UnixNano()
	windowNanos := window.Nanoseconds()
	randBytes := make([]byte, 4)
	_, _ = rand.Read(randBytes)
	prefix := hex.EncodeToString(randBytes)

	res, err := l.lua.Run(ctx, client, []string{key}, nowNanos, windowNanos, limit, cost, prefix).Slice()
	if err != nil {
		return nil, fmt.Errorf("rate limiter lua execution failed: %w", err)
	}
	if len(res) < 3 {
		return nil, fmt.Errorf("rate limiter invalid lua response: %v", res)
	}

	allowedInt, _ := res[0].(int64)
	remainingInt, _ := res[1].(int64)
	retryAfterNanos, _ := res[2].(int64)

	allowed := allowedInt == 1
	retryAfter := time.Duration(retryAfterNanos)
	if !allowed && retryAfter <= 0 {
		retryAfter = window
	}

	return &RateLimitResult{
		Allowed:    allowed,
		Remaining:  remainingInt,
		Limit:      limit,
		ResetAfter: window,
		RetryAfter: retryAfter,
	}, nil
}

func (l *SlidingWindowLimiter) allowFallback(ctx context.Context, key string, limit int64, window time.Duration, cost int64) (*RateLimitResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	nowNanos := now.UnixNano()
	clearBefore := float64(nowNanos - window.Nanoseconds())

	// Remove stale events older than the sliding window
	_, err := l.cache.ZRemRangeByScore(ctx, key, 0, clearBefore)
	if err != nil {
		return nil, fmt.Errorf("rate limiter zremrange failed: %w", err)
	}

	// Count events within the current active window
	current, err := l.cache.ZCard(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("rate limiter zcard failed: %w", err)
	}

	if current+cost <= limit {
		for i := int64(0); i < cost; i++ {
			member := fmt.Sprintf("%d:%d:%d", nowNanos, i, now.Unix())
			if addErr := l.cache.ZAdd(ctx, key, float64(nowNanos), member); addErr != nil {
				return nil, fmt.Errorf("rate limiter zadd failed: %w", addErr)
			}
		}
		return &RateLimitResult{
			Allowed:    true,
			Remaining:  limit - (current + cost),
			Limit:      limit,
			ResetAfter: window,
			RetryAfter: 0,
		}, nil
	}

	// Rate limit exceeded: determine the oldest member in window to compute retryAfter
	oldestMembers, err := l.cache.ZRangeByScore(ctx, key, clearBefore, math.MaxFloat64, 0, 1)
	retryAfter := window
	if err == nil && len(oldestMembers) > 0 {
		// Extract timestamp prefix
		var oldestNanos int64
		if _, scanErr := fmt.Sscanf(oldestMembers[0], "%d:", &oldestNanos); scanErr == nil && oldestNanos > 0 {
			diff := time.Duration(oldestNanos + window.Nanoseconds() - nowNanos)
			if diff > 0 {
				retryAfter = diff
			}
		}
	}

	return &RateLimitResult{
		Allowed:    false,
		Remaining:  0,
		Limit:      limit,
		ResetAfter: window,
		RetryAfter: retryAfter,
	}, nil
}

// Reset clears the rate limit state for a key.
func (l *SlidingWindowLimiter) Reset(ctx context.Context, key string) error {
	if l == nil || l.cache == nil {
		return nil
	}
	return l.cache.backend.Delete(ctx, key)
}

func (l *SlidingWindowLimiter) extractRedisClient() *redis.Client {
	if l == nil || l.cache == nil || l.cache.backend == nil {
		return nil
	}
	type clientGetter interface {
		Client() *redis.Client
	}
	if cg, ok := l.cache.backend.(clientGetter); ok {
		return cg.Client()
	}
	return nil
}
