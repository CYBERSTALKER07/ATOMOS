package idempotency

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"backend-go/cache"

	"github.com/redis/go-redis/v9"
)

const (
	headerKey = "Idempotency-Key"
	keyPrefix = cache.PrefixIdempotency // "idem:"
	idempTTL  = cache.TTLIdempotency    // 24h
)

// cachedResponse is stored in Redis for replay on duplicate requests.
type cachedResponse struct {
	StatusCode int               `json:"s"`
	Headers    map[string]string `json:"h"`
	Body       string            `json:"b"`
}

// responseRecorder captures the handler's response for caching.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	rr.body.Write(b)
	return rr.ResponseWriter.Write(b)
}

// Guard wraps an http.HandlerFunc with Redis-backed idempotency.
// If the Idempotency-Key header is missing or Redis is offline, the
// request passes through unmodified (graceful degradation).
func Guard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get(headerKey)
		rc := cache.GetClient()
		if key == "" || rc == nil {
			next(w, r)
			return
		}

		redisKey := keyPrefix + key
		ctx := r.Context()

		// Check for cached response
		cached, err := rc.Get(ctx, redisKey).Result()
		if err == nil {
			// Cache hit — replay the stored response
			var cr cachedResponse
			if json.Unmarshal([]byte(cached), &cr) == nil {
				for k, v := range cr.Headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(cr.StatusCode)
				w.Write([]byte(cr.Body))
				return
			}
		}

		// Acquire processing lock (NX = only if not exists, short TTL as guard)
		lockKey := redisKey + cache.SuffixIdempotencyLock
		acquired, err := rc.SetNX(ctx, lockKey, "1", cache.TTLIdempotencyLock).Result()
		if err != nil || !acquired {
			// Another request is processing this key — return 409
			if err == nil {
				http.Error(w, "Duplicate request in progress", http.StatusConflict)
				return
			}
			// Redis error — degrade gracefully and let the request through
			next(w, r)
			return
		}
		defer rc.Del(context.Background(), lockKey)

		// Execute the handler and capture the response
		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next(rec, r)

		// Only cache successful (2xx) responses
		if rec.statusCode >= 200 && rec.statusCode < 300 {
			cr := cachedResponse{
				StatusCode: rec.statusCode,
				Headers:    map[string]string{"Content-Type": w.Header().Get("Content-Type")},
				Body:       rec.body.String(),
			}
			if data, err := json.Marshal(cr); err == nil {
				rc.Set(ctx, redisKey, string(data), idempTTL)
			}
		}
	}
}

// Purge removes a cached idempotency key. Useful for testing.
func Purge(ctx context.Context, key string) error {
	rc := cache.GetClient()
	if rc == nil {
		return nil
	}
	return rc.Del(ctx, keyPrefix+key).Err()
}

// compile-time check that redis.Nil is used correctly
var _ = redis.Nil
