package warehouse

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

func readMutationBody(w http.ResponseWriter, r *http.Request, limit int64) ([]byte, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, limit))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body: " + err.Error()})
		return nil, false
	}
	defer r.Body.Close()
	return body, true
}

func idempotencyKeyFromRequest(r *http.Request) string {
	if key := strings.TrimSpace(r.Header.Get("Idempotency-Key")); key != "" {
		return key
	}
	return strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
}

func (s *Service) guardMutationReplay(w http.ResponseWriter, r *http.Request, body []byte) (string, bool) {
	key := idempotencyKeyFromRequest(r)
	if key == "" || s.idem == nil {
		return "", false
	}
	hash := sha256Hex(body)
	rec, hit, err := idempotency.Guard(r.Context(), s.idem, key, hash)
	switch {
	case errors.Is(err, idempotency.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency_key_payload_mismatch"})
		return "", true
	case err != nil:
		s.log.Warn("warehouse idempotency guard failed", "err", err)
		return key, false
	case hit:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rec.StatusCode)
		_, _ = w.Write(rec.Response)
		return "", true
	default:
		return key, false
	}
}

func (s *Service) storeMutationReplay(ctx context.Context, key string, body []byte, statusCode int, response []byte) {
	if key == "" || s.idem == nil {
		return
	}
	_ = s.idem.Save(ctx, key, idempotency.Record{
		BodyHash:   sha256Hex(body),
		StatusCode: statusCode,
		Response:   response,
		StoredAt:   s.now(),
	}, 24*time.Hour)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
