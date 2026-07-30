package order

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

func idempotencyKeyFromRequest(r *http.Request, body []byte) string {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		key = strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
	}
	if key == "" && len(body) > 0 {
		var partial struct {
			IdempotencyKey string `json:"idempotencyKey"`
		}
		_ = json.Unmarshal(body, &partial)
		key = partial.IdempotencyKey
	}
	return key
}

func sha256HexBytes(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// guardIdempotency replays a prior response when the key was seen with the same body.
// Returns true when the handler should stop.
func (s *Service) guardIdempotency(w http.ResponseWriter, r *http.Request, body []byte) bool {
	key := idempotencyKeyFromRequest(r, body)
	if key == "" || s.idem == nil {
		return false
	}

	// Boss rule: Telemetry timestamp skewed > 5 min -> Reject
	if len(body) > 0 {
		var partial struct {
			ClientTimestamp *time.Time `json:"clientTimestamp"`
		}
		if err := json.Unmarshal(body, &partial); err == nil && partial.ClientTimestamp != nil {
			skew := s.now().Sub(*partial.ClientTimestamp)
			if skew < 0 {
				skew = -skew
			}
			if skew > 5*time.Minute {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_timestamp_skewed"})
				return true
			}
		}
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
			s.log.Warn("idempotency guard failed", "err", err)
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

func (s *Service) releaseIdempotency(ctx context.Context, r *http.Request) {
	key := idempotencyKeyFromRequest(r)
	if key == "" || s.idem == nil {
		return
	}
	_ = s.idem.Release(ctx, key)
}

func (s *Service) writeIdempotentJSON(w http.ResponseWriter, r *http.Request, reqBody []byte, code int, resp any) {
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, reqBody, code, respBytes)
	writeJSONBytes(w, code, respBytes)
}

func writeJSONBytes(w http.ResponseWriter, code int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(body)
}
