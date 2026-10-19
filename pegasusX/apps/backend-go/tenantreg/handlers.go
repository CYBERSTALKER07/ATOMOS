package tenantreg

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleRegister is POST /v1/platform/tenants/register (public).
func (s *Service) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		web.JSONError(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}
	if s == nil || s.repo == nil {
		web.JSONError(w, ErrRepoUnavailable.Error(), http.StatusServiceUnavailable)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 32*1024))
	if err != nil {
		web.JSONError(w, "read_body_failed", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if key := strings.TrimSpace(r.Header.Get("Idempotency-Key")); key != "" && s.idemStore != nil {
		rec, hit, gerr := idempotency.Guard(r.Context(), s.idemStore, key, sha256Hex(body))
		switch {
		case errors.Is(gerr, idempotency.ErrConflict):
			web.JSONError(w, "idempotency_key_payload_mismatch", http.StatusConflict)
			return
		case gerr != nil:
			s.log.Warn("tenantreg idempotency guard failed", "err", gerr)
		case hit:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(rec.StatusCode)
			_, _ = w.Write(rec.Response)
			return
		}
	}

	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		web.JSONError(w, "invalid_json", http.StatusBadRequest)
		return
	}

	resp, err := s.Register(r.Context(), req)
	if err != nil {
		writeRegisterError(w, err)
		return
	}

	if token, tokErr := s.IssueToken(resp.SupplierID, resp.MarketCode, resp.HomeCell); tokErr != nil {
		s.log.Warn("tenantreg issue token failed", "err", tokErr)
	} else {
		resp.Token = token
		auth.SetSessionCookie(w, token, s.jwtTTL, s.cookieSecure)
	}

	respBytes, _ := json.Marshal(resp)
	if key := strings.TrimSpace(r.Header.Get("Idempotency-Key")); key != "" && s.idemStore != nil {
		_ = s.idemStore.Save(r.Context(), key, idempotency.Record{
			BodyHash:   sha256Hex(body),
			StatusCode: http.StatusCreated,
			Response:   respBytes,
			StoredAt:   s.now(),
		}, 24*time.Hour)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

func writeRegisterError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrRepoUnavailable):
		web.JSONError(w, err.Error(), http.StatusServiceUnavailable)
	case errors.Is(err, ErrLegalNameRequired), errors.Is(err, ErrPhoneRequired),
		errors.Is(err, ErrPasswordRequired), errors.Is(err, ErrMarketCodeRequired):
		web.JSONError(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, ErrUnknownMarket), errors.Is(err, ErrMarketNotShipped):
		code := ""
		if parts := strings.Split(err.Error(), ": "); len(parts) == 2 {
			code = parts[1]
		}
		web.JSONResponse(w, http.StatusNotFound, map[string]string{
			"error": errKeyword(err),
			"code":  code,
		})
	case errors.Is(err, ErrPhoneTaken):
		web.JSONError(w, err.Error(), http.StatusConflict)
	default:
		web.JSONError(w, "register_failed", http.StatusUnprocessableEntity)
	}
}

func errKeyword(err error) string {
	switch {
	case errors.Is(err, ErrMarketNotShipped):
		return ErrMarketNotShipped.Error()
	case errors.Is(err, ErrUnknownMarket):
		return ErrUnknownMarket.Error()
	default:
		return "register_failed"
	}
}

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
