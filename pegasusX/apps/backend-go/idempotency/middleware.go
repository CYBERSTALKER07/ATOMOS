package idempotency

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

const (
	maxIdempotencyBodyBytes         = 1 << 20 // 1 MiB (SO-4, CG-3)
	maxIdempotencyResponseBodyBytes = 1 << 20 // 1 MiB (SO-4, CG-3)
	lockTimeout                     = 2 * time.Second
)

// writeJSONError standardizes all error outputs on structured JSON with Content-Type: application/json (UA-6).
func writeJSONError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}

// detachedLockContext creates a context detached from client cancellation with a 2-second timeout (UA-3).
func detachedLockContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), lockTimeout)
}

// responseRecorder captures the response body and status code to save in the store.
type responseRecorder struct {
	http.ResponseWriter
	status      int
	body        *bytes.Buffer
	wroteHeader bool
	exceeded    bool
}

// WriteHeader captures the status code and sets headers if needed.
func (r *responseRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
		if status == http.StatusGatewayTimeout {
			r.ResponseWriter.Header().Set("Retry-After", "10")
		}
		r.ResponseWriter.WriteHeader(status)
	}
}

// Write captures the response body up to maxIdempotencyResponseBodyBytes.
func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	if !r.exceeded {
		if r.body.Len()+len(b) > maxIdempotencyResponseBodyBytes {
			r.exceeded = true
			r.body.Reset() // Release memory without caching
		} else {
			r.body.Write(b)
		}
	}
	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := r.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("idempotency: underlying ResponseWriter does not implement http.Hijacker")
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

			// 1. Cap request bodies at 1 MiB. Requests > 1 MiB must be rejected
			// with HTTP 413 ({"error":"payload_too_large"}) before acquiring any lock or touching Redis/Spanner.
			if r.ContentLength > maxIdempotencyBodyBytes {
				writeJSONError(w, http.StatusRequestEntityTooLarge, "payload_too_large")
				return
			}

			var bodyBytes []byte
			if r.Body != nil {
				lr := io.LimitReader(r.Body, maxIdempotencyBodyBytes+1)
				var err error
				bodyBytes, err = io.ReadAll(lr)
				if err != nil {
					writeJSONError(w, http.StatusBadRequest, "read_body_error")
					return
				}
				if int64(len(bodyBytes)) > maxIdempotencyBodyBytes {
					writeJSONError(w, http.StatusRequestEntityTooLarge, "payload_too_large")
					return
				}
				_ = r.Body.Close()
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			} else {
				r.Body = http.NoBody
			}

			principal := ""
			if claims, ok := auth.FromContext(r.Context()); ok && strings.TrimSpace(claims.Subject) != "" {
				principal = strings.TrimSpace(claims.Subject)
			} else if t, ok := auth.PartnerIdentityFromContext(r.Context()); ok && t.PrincipalIdentifier() != "" {
				principal = t.PrincipalIdentifier()
			} else if t, ok := auth.TenantFromContext(r.Context()); ok && strings.TrimSpace(t.SupplierID) != "" {
				principal = "supplier:" + strings.TrimSpace(t.SupplierID)
			}
			routePattern := r.Method + " " + r.URL.Path
			key := ScopeKey(principal, routePattern, rawKey)

			hash := sha256.Sum256(bodyBytes)
			hashHex := hex.EncodeToString(hash[:])

			// 2. Lock operations use a detached context with 2-second timeout (UA-3).
			guardCtx, cancelGuard := detachedLockContext(r.Context())
			rec, hit, err := Guard(guardCtx, store, key, hashHex)
			cancelGuard()

			if err != nil {
				if errors.Is(err, ErrConflict) {
					// Key payload mismatch must return HTTP 422 Unprocessable Entity (UA-4).
					writeJSONError(w, http.StatusUnprocessableEntity, "idempotency_key_payload_mismatch")
					return
				}
				if errors.Is(err, ErrInProgress) {
					// Concurrent in-flight requests return HTTP 409 Conflict with Retry-After: 1 (UA-4).
					w.Header().Set("Retry-After", "1")
					writeJSONError(w, http.StatusConflict, "request_in_progress")
					return
				}
				writeJSONError(w, http.StatusInternalServerError, "idempotency_guard_failed")
				return
			}

			if hit {
				// Replay cache hits must inject Idempotency-Replayed: true and preserve original Content-Type (UA-3).
				contentType := rec.ContentType
				if contentType == "" {
					contentType = "application/json"
				}
				w.Header().Set("Content-Type", contentType)
				w.Header().Set("Idempotency-Replayed", "true")
				w.WriteHeader(rec.StatusCode)
				_, _ = w.Write(rec.Response)
				return
			}

			r = r.WithContext(WithClaimed(r.Context()))

			recorder := &responseRecorder{
				ResponseWriter: w,
				status:         0,
				body:           &bytes.Buffer{},
			}

			panicked := true
			defer func() {
				if panicked {
					relCtx, cancelRel := detachedLockContext(r.Context())
					_ = store.Release(relCtx, key)
					cancelRel()
				}
			}()

			next.ServeHTTP(recorder, r)
			panicked = false

			if !recorder.wroteHeader {
				if errors.Is(r.Context().Err(), context.DeadlineExceeded) {
					recorder.status = http.StatusGatewayTimeout
					w.Header().Set("Retry-After", "10")
					writeJSONError(w, http.StatusGatewayTimeout, "gateway_timeout")
				} else {
					recorder.status = http.StatusOK
				}
			}

			// Differentiated lock retention (UA-5):
			// - On unconfirmed commit timeouts / 504s: hold lock for 10 seconds and return HTTP 504 with Retry-After: 10.
			// - On 4xx errors and confirmed Spanner rollback/validation errors: release lock immediately.
			// - On 2xx: cache response if <= 1 MiB, else release lock immediately without caching.
			if recorder.status == http.StatusGatewayTimeout {
				holdCtx, cancelHold := detachedLockContext(r.Context())
				_ = store.Hold(holdCtx, key, 10*time.Second)
				cancelHold()
				return
			}

			if recorder.status >= 200 && recorder.status < 300 {
				if recorder.exceeded {
					// Response exceeds 1 MiB: release lock immediately without caching
					relCtx, cancelRel := detachedLockContext(r.Context())
					_ = store.Release(relCtx, key)
					cancelRel()
					return
				}

				ct := recorder.Header().Get("Content-Type")
				if ct == "" {
					ct = "application/json"
				}
				saveCtx, cancelSave := detachedLockContext(r.Context())
				_ = store.Save(saveCtx, key, Record{
					BodyHash:    hashHex,
					StatusCode:  recorder.status,
					Response:    recorder.body.Bytes(),
					StoredAt:    time.Now(),
					ContentType: ct,
				}, 24*time.Hour)
				cancelSave()
			} else {
				// 4xx errors, 500s, confirmed Spanner rollback/validation errors: release lock immediately
				relCtx, cancelRel := detachedLockContext(r.Context())
				_ = store.Release(relCtx, key)
				cancelRel()
			}
		})
	}
}
