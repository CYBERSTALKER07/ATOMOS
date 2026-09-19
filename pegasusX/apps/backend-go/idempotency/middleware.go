package idempotency

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// responseRecorder captures the response body and status code to save in the store.
type responseRecorder struct {
	http.ResponseWriter
	status int
	body   *bytes.Buffer
}

// WriteHeader captures the status code.
func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Write captures the response body.
func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// Middleware provides an HTTP middleware that handles idempotency.
func Middleware(store Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if store == nil || !isMutatingMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			rawKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
			if rawKey == "" {
				rawKey = strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
			}
			if rawKey == "" {
				next.ServeHTTP(w, r)
				return
			}
			principal := ""
			if claims, ok := auth.FromContext(r.Context()); ok {
				principal = strings.TrimSpace(claims.Subject)
			}
			routePattern := r.Method + " " + r.URL.Path
			key := ScopeKey(principal, routePattern, rawKey)

			bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
			if err != nil {
				http.Error(w, `{"error":"read_body_error"}`, http.StatusBadRequest)
				return
			}
			_ = r.Body.Close()
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			hash := sha256.Sum256(bodyBytes)
			hashHex := hex.EncodeToString(hash[:])
			rec, hit, err := Guard(r.Context(), store, key, hashHex)
			if err != nil {
				if errors.Is(err, ErrConflict) {
					http.Error(w, `{"error":"idempotency_key_payload_mismatch"}`, http.StatusConflict)
					return
				}
				if errors.Is(err, ErrInProgress) {
					http.Error(w, `{"error":"request_in_progress"}`, http.StatusConflict)
					return
				}
				http.Error(w, `{"error":"idempotency_guard_failed"}`, http.StatusInternalServerError)
				return
			}
			if hit {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(rec.StatusCode)
				_, _ = w.Write(rec.Response)
				return
			}

			r = r.WithContext(WithClaimed(r.Context()))

			recorder := &responseRecorder{
				ResponseWriter: w,
				status:         http.StatusOK,
				body:           &bytes.Buffer{},
			}

			func() {
				defer func() {
					if rec := recover(); rec != nil {
						_ = store.Release(r.Context(), key)
						panic(rec)
					}
				}()
				next.ServeHTTP(recorder, r)
			}()

			if recorder.status >= 200 && recorder.status < 300 {
				_ = store.Save(r.Context(), key, Record{
					BodyHash:   hashHex,
					StatusCode: recorder.status,
					Response:   recorder.body.Bytes(),
					StoredAt:   time.Now(),
				}, 24*time.Hour)
			} else {
				_ = store.Release(r.Context(), key)
			}
		})
	}
}
