package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"time"
)

// cacheWriteJob represents a deferred cache write operation.
type cacheWriteJob struct {
	key string
	val string
	ttl time.Duration
}

// cacheWriteCh is a bounded channel that limits concurrent cache write goroutines.
// Buffer size caps the number of pending writes; excess writes are dropped (cache is advisory).
var cacheWriteCh = make(chan cacheWriteJob, 1024)

// StartCacheWorkers starts the pool of background write workers.
// Call once from bootstrap.NewApp AFTER cache.New so Redis is initialised.
// Workers use GetClient() on each iteration so they tolerate Redis restarts.
func StartCacheWorkers(ctx context.Context) {
	for i := 0; i < 8; i++ {
		go cacheWriteWorker(ctx)
	}
}

func cacheWriteWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-cacheWriteCh:
			// Always populate L1 — reads can serve from here if Redis is down.
			l1Set(job.key, job.val, job.ttl)
			c := GetClient()
			if c == nil {
				continue
			}
			writeCtx, writeCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			if setErr := c.Set(writeCtx, job.key, job.val, job.ttl).Err(); setErr != nil {
				log.Printf("[CACHE] Failed to write key %s: %v", job.key, setErr)
			}
			writeCancel()
		}
	}
}

// CacheHandler wraps an HTTP handler with read-through Redis caching backed by
// an in-process L1 layer. L1 is checked first (no network); on L1 miss it
// tries Redis (L2); on L2 miss it falls through to the inner handler and
// populates both tiers asynchronously via the write-worker pool.
// If Redis is unavailable the L1 still serves warm entries; cold misses fall
// through to the inner handler (fail-open).
func CacheHandler(keyPrefix string, ttl time.Duration, inner http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			inner(w, r)
			return
		}

		cacheKey := fmt.Sprintf("%s:%s", keyPrefix, r.URL.RawQuery)

		// ── L1 check (in-process, zero latency) ─────────────────────────────
		if cached, ok := l1Get(cacheKey); ok {
			etag := computeETag([]byte(cached))
			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "L1-HIT")
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(ttl.Seconds())))
			w.Header().Set("ETag", etag)
			w.Write([]byte(cached)) //nolint:errcheck
			return
		}

		// ── L2 check (Redis) ─────────────────────────────────────────────────
		if rc := GetClient(); rc != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
			cached, err := rc.Get(ctx, cacheKey).Result()
			cancel()
			if err == nil && cached != "" {
				l1Set(cacheKey, cached, ttl) // warm L1 on L2 hit
				etag := computeETag([]byte(cached))
				if r.Header.Get("If-None-Match") == etag {
					w.WriteHeader(http.StatusNotModified)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Cache", "HIT")
				w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(ttl.Seconds())))
				w.Header().Set("ETag", etag)
				w.Write([]byte(cached)) //nolint:errcheck
				return
			}
		}

		// Cache miss — call the inner handler and capture the response
		rec := &responseRecorder{ResponseWriter: w, statusCode: 200}
		inner(rec, r)

		// Only cache successful JSON responses
		if rec.statusCode == 200 && len(rec.body) > 0 {
			select {
			case cacheWriteCh <- cacheWriteJob{key: cacheKey, val: string(rec.body), ttl: ttl}:
				// queued
			default:
				// channel full — drop this cache write (cache is advisory)
			}
		}

		w.Header().Set("X-Cache", "MISS")
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(ttl.Seconds())))
		if len(rec.body) > 0 {
			w.Header().Set("ETag", computeETag(rec.body))
		}
	}
}

// computeETag generates a weak ETag from a SHA-256 prefix of the response body.
func computeETag(data []byte) string {
	h := sha256.Sum256(data)
	return `W/"` + hex.EncodeToString(h[:8]) + `"`
}

// Invalidate removes a specific cache key. Call on write operations.
func Invalidate(ctx context.Context, keys ...string) {
	(&Cache{}).Invalidate(ctx, keys...)
}

// InvalidatePrefix removes all keys matching a prefix pattern. Use sparingly.
func InvalidatePrefix(ctx context.Context, prefix string) {
	if err := (&Cache{}).InvalidatePrefix(ctx, prefix); err != nil {
		log.Printf("[CACHE] Prefix invalidation failed for %s: %v", prefix, err)
	}
}

// responseRecorder captures the HTTP response for caching.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.statusCode == 200 {
		r.body = append(r.body, b...)
	}
	return r.ResponseWriter.Write(b)
}
