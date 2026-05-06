package cache

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// ── Cooldown Jail (Repeat-Offender Escalation) ───────────────────────────────
// When an identity hits the rate limit repeatedly within a short window, we
// escalate from per-request 429s to a sustained "jail" — a Redis key with TTL
// that immediately rejects all requests from that identity until expiry.
//
// Tiered escalation:
//   3 strikes within JailWindowSec  → 60s jail
//   5 strikes within JailWindowSec  → 300s jail (5 min)
//
// Strike counts are themselves Redis keys with their own TTL so they self-heal
// after a quiet period. The jail key carries an X-Jail-Until response header
// so the frontend can render a precise countdown rather than a generic toast.

const (
	JailWindowSec      = 60   // sliding window during which strikes accumulate
	JailStrikesTier1   = 3    // first jail trigger
	JailStrikesTier2   = 5    // hard jail trigger
	JailDurationTier1  = 60   // seconds
	JailDurationTier2  = 300  // seconds
	jailKeyPrefix      = "jail:"
	jailStrikesPrefix  = "jstrike:"
)

// jailKey returns the Redis key holding the jail expiry for an actor+priority.
func jailKey(actor, priority string) string {
	return jailKeyPrefix + priority + ":" + actor
}

// strikesKey returns the Redis key counting recent rate-limit hits.
func strikesKey(actor, priority string) string {
	return jailStrikesPrefix + priority + ":" + actor
}

// CheckJail returns (jailedUntilUnix, true) if the actor is currently jailed
// for the given priority. Fail-open: any Redis error returns (0, false).
func CheckJail(ctx context.Context, actor, priority string) (int64, bool) {
	if Client == nil || actor == "" {
		return 0, false
	}
	ttl, err := Client.TTL(ctx, jailKey(actor, priority)).Result()
	if err != nil || ttl <= 0 {
		return 0, false
	}
	return time.Now().Add(ttl).Unix(), true
}

// RecordStrike increments the strike counter and, on threshold cross, opens
// a jail entry. Returns (jailedUntilUnix, true) when a jail was just opened
// (or extended), (0, false) otherwise. Fail-open on Redis errors.
func RecordStrike(ctx context.Context, actor, priority string) (int64, bool) {
	if Client == nil || actor == "" {
		return 0, false
	}
	sk := strikesKey(actor, priority)
	count, err := Client.Incr(ctx, sk).Result()
	if err != nil {
		return 0, false
	}
	// First strike sets the window TTL.
	if count == 1 {
		_ = Client.Expire(ctx, sk, time.Duration(JailWindowSec)*time.Second).Err()
	}

	var jailDuration int
	switch {
	case count >= JailStrikesTier2:
		jailDuration = JailDurationTier2
	case count >= JailStrikesTier1:
		jailDuration = JailDurationTier1
	default:
		return 0, false
	}

	until := time.Now().Add(time.Duration(jailDuration) * time.Second)
	if err := Client.Set(ctx, jailKey(actor, priority), "1", time.Duration(jailDuration)*time.Second).Err(); err != nil {
		return 0, false
	}
	return until.Unix(), true
}

// JailMiddleware wraps a downstream handler with cooldown enforcement. It is
// intended to compose AFTER PrioritySheddingMiddleware — when the underlying
// rate limiter signals 429 via its handler call chain we record a strike here
// (via RecordStrike from the priority middleware) and short-circuit any
// future request while the jail is active.
//
// This middleware itself only enforces existing jail entries; strike recording
// happens at the rate-limit decision point.
func JailMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			actor := userOrIPKey(r)
			priority := ClassifyRequest(r.URL.Path).String()
			if until, jailed := CheckJail(r.Context(), actor, priority); jailed {
				retryAfter := until - time.Now().Unix()
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
				w.Header().Set("X-Jail-Until", strconv.FormatInt(until, 10))
				w.Header().Set("X-Priority", priority)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				fmt.Fprintf(w, `{"error":"cooldown_active","priority":"%s","retry_after":%d,"jail_until":%d,"detail":"Repeated rate-limit violations triggered a cooldown. Slow down and retry after the cooldown expires."}`,
					priority, retryAfter, until)
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}
