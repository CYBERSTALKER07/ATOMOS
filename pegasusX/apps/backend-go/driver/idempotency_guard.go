package driver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

func readLimitedBody(r *http.Request, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, limit))
	_ = r.Body.Close()
	return body, err
}

func idempotencyKeyFromRequest(r *http.Request) string {
	if key := strings.TrimSpace(r.Header.Get("Idempotency-Key")); key != "" {
		return key
	}
	return strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
}

func sha256HexBytes(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (s *Service) guardIdempotency(w http.ResponseWriter, r *http.Request, body []byte) bool {
	key := idempotencyKeyFromRequest(r)
	if key == "" || s.idem == nil {
		return false
	}
	hash := sha256HexBytes(body)
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
			s.log.Warn("driver idempotency guard failed", "err", err)
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
		BodyHash:   sha256HexBytes(body),
		StatusCode: status,
		Response:   resp,
		StoredAt:   time.Now().UTC(),
	}, 24*time.Hour)
}

func writeJSONBytes(w http.ResponseWriter, code int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(body)
}
