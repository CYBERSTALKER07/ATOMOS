package ws

import (
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// ReconnectDelay returns exponential backoff with full jitter for Desert Protocol
// client reconnect scheduling. attempt is zero-based; base/max bound the delay.
func ReconnectDelay(attempt int, base, max time.Duration) time.Duration {
	return ReconnectDelayWithRand(attempt, base, max, rand.New(rand.NewSource(time.Now().UnixNano())))
}

// ReconnectDelayWithRand is the injectable RNG variant for tests.
func ReconnectDelayWithRand(attempt int, base, max time.Duration, rng *rand.Rand) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	if base <= 0 {
		base = 2 * time.Second
	}
	if max <= 0 {
		max = 60 * time.Second
	}
	if max < base {
		max = base
	}
	delay := base << min(attempt, 10)
	if delay > max {
		delay = max
	}
	jitterMax := int64(delay / 2)
	if jitterMax <= 0 || rng == nil {
		return delay
	}
	return delay + time.Duration(rng.Int63n(jitterMax+1))
}

// ParseRetryAfterSeconds reads an HTTP Retry-After header (seconds form only).
func ParseRetryAfterSeconds(header string) (int, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0, false
	}
	seconds, err := strconv.Atoi(header)
	if err != nil || seconds < 0 {
		return 0, false
	}
	return seconds, true
}

// ReconnectDelayWithRetryAfter returns max(backoff+jitter, retryAfter) when the
// server signals capacity pressure via Retry-After.
func ReconnectDelayWithRetryAfter(attempt int, base, max, retryAfter time.Duration) time.Duration {
	delay := ReconnectDelay(attempt, base, max)
	if retryAfter > delay {
		return retryAfter
	}
	return delay
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
