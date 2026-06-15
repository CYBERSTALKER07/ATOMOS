package retailer

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

func readLimitedBody(w http.ResponseWriter, r *http.Request, limit int64) ([]byte, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, limit))
	_ = r.Body.Close()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return nil, false
	}
	return body, true
}

func idempotencyKeyFromRequest(r *http.Request) string {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		key = strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
	}
	return key
}

func (s *Service) guardIdempotency(w http.ResponseWriter, r *http.Request, body []byte) bool {
	key := idempotencyKeyFromRequest(r)
	if key == "" || s.idem == nil {
		return false
	}
	hash := sha256Hex(body)
	rec, hit, err := idempotency.Guard(r.Context(), s.idem, key, hash)
	switch {
	case errors.Is(err, idempotency.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency_key_payload_mismatch"})
		return true
	case errors.Is(err, idempotency.ErrInProgress):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "request_in_progress"})
		return true
	case err != nil:
		if s.log != nil {
			s.log.Warn("retailer idempotency guard failed", "err", err)
		}
	case hit:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rec.StatusCode)
		_, _ = w.Write(rec.Response)
		return true
	}
	return false
}

func (s *Service) saveIdempotency(ctx context.Context, r *http.Request, body []byte, status int, resp []byte) {
	key := idempotencyKeyFromRequest(r)
	if key == "" || s.idem == nil {
		return
	}
	_ = s.idem.Save(ctx, key, idempotency.Record{
		BodyHash:   sha256Hex(body),
		StatusCode: status,
		Response:   resp,
		StoredAt:   s.now(),
	}, 24*time.Hour)
}

func (s *Service) releaseIdempotency(ctx context.Context, r *http.Request) {
	key := idempotencyKeyFromRequest(r)
	if key == "" || s.idem == nil {
		return
	}
	_ = s.idem.Release(ctx, key)
}

func writeJSONBytes(w http.ResponseWriter, code int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(body)
}
