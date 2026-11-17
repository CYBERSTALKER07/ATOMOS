package mfa

import (
	"context"
	"encoding/base32"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestTOTPRoundTrip(t *testing.T) {
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	code := hotpMust(t, secret, time.Now().UTC())
	if valid, _ := ValidateCode(secret, code, time.Now().UTC()); !valid {
		t.Fatal("expected valid code")
	}
	if valid, _ := ValidateCode(secret, "000000", time.Now().UTC()); valid {
		t.Fatal("expected invalid code")
	}
}

func hotpMust(t *testing.T, secret string, now time.Time) string {
	t.Helper()
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		t.Fatal(err)
	}
	return hotp(key, uint64(now.Unix()/totpPeriod))
}

func TestMFAEnrollConfirmVerifyAndStepUp(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, "Test", true, nil, nil)
	h := &Handlers{Svc: svc, JWTSecret: "test-secret-mfa", JWTIssuer: "test"}

	r := chi.NewRouter()
	RegisterRoutes(r, h)
	r.With(auth.RequireRole(auth.RolePlatformAdmin), RequireStepUp(svc)).Get("/v1/platform-admin/tenants", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	baseClaims := auth.Claims{Subject: "admin-a", Role: auth.RolePlatformAdmin}
	baseTok, err := auth.Issue(baseClaims, auth.IssueOptions{Secret: "test-secret-mfa", Issuer: "test"})
	if err != nil {
		t.Fatal(err)
	}

	rr := doAuth(r, http.MethodGet, "/v1/platform-admin/tenants", baseTok, nil)
	if rr.Code != http.StatusForbidden || !strings.Contains(rr.Body.String(), "mfa_enrollment_required") {
		t.Fatalf("expected enrollment required, got %d %s", rr.Code, rr.Body.String())
	}

	rr = doAuth(r, http.MethodPost, "/v1/platform-admin/mfa/enroll", baseTok, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("enroll: %d %s", rr.Code, rr.Body.String())
	}
	var enroll struct {
		Secret string `json:"secret"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &enroll)
	if enroll.Secret == "" {
		t.Fatal("missing secret")
	}

	code := hotpMust(t, enroll.Secret, time.Now().UTC())
	rr = doAuth(r, http.MethodPost, "/v1/platform-admin/mfa/confirm", baseTok, map[string]string{"code": code})
	if rr.Code != http.StatusOK {
		t.Fatalf("confirm: %d %s", rr.Code, rr.Body.String())
	}
	var confirmed struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &confirmed)
	if confirmed.Token == "" {
		t.Fatal("expected stepped-up token")
	}

	rr = doAuth(r, http.MethodGet, "/v1/platform-admin/tenants", confirmed.Token, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("tenants with mfa token: %d %s", rr.Code, rr.Body.String())
	}

	rr = doAuth(r, http.MethodGet, "/v1/platform-admin/tenants", baseTok, nil)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("unenforced base token should be forbidden after enroll, got %d", rr.Code)
	}

	code2 := hotpMust(t, enroll.Secret, time.Now().UTC())
	rr = doAuth(r, http.MethodPost, "/v1/platform-admin/mfa/verify", baseTok, map[string]string{"code": code2})
	if rr.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", rr.Code, rr.Body.String())
	}
}

func doAuth(r chi.Router, method, path, token string, body map[string]string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, strings.NewReader(string(b)))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	claims, _ := auth.Parse(token, "test-secret-mfa")
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestNeedsStepUpOptionalWhenNotRequired(t *testing.T) {
	svc := NewService(NewMemoryRepository(), "x", false, nil, nil)
	need, err := svc.NeedsStepUp(context.Background(), "anyone")
	if err != nil || need {
		t.Fatalf("need=%v err=%v", need, err)
	}
}

type failingReplayCache struct{}

func (failingReplayCache) IsUsed(context.Context, string, uint64) (bool, error) {
	return false, errors.New("redis connection refused")
}
func (failingReplayCache) MarkUsed(context.Context, string, uint64) error {
	return errors.New("redis connection refused")
}

func TestMFAReplayFailClosedAndLockout(t *testing.T) {
	repo := NewMemoryRepository()
	svcNoCache := NewService(repo, "Test", true, nil, nil)
	sec, _, err := svcNoCache.BeginEnroll(context.Background(), "adm-lock")
	if err != nil {
		t.Fatal(err)
	}
	validCode := hotpMust(t, sec, time.Now().UTC())
	if err := svcNoCache.ConfirmEnroll(context.Background(), "adm-lock", validCode); err != nil {
		t.Fatal(err)
	}

	// 1. Failing Redis cache fails closed instead of silently accepting replay.
	svcBadCache := NewService(repo, "Test", true, nil, failingReplayCache{})
	if err := svcBadCache.Verify(context.Background(), "adm-lock", validCode); err == nil || !strings.Contains(err.Error(), "mfa_replay_check_unavailable") {
		t.Fatalf("expected fail-closed replay error, got %v", err)
	}

	// 2. Five invalid TOTP attempts trigger temporary lockout even for a valid code.
	for i := 0; i < 5; i++ {
		if err := svcNoCache.Verify(context.Background(), "adm-lock", "000000"); err == nil {
			t.Fatal("expected invalid_totp")
		}
	}
	if err := svcNoCache.Verify(context.Background(), "adm-lock", validCode); err == nil || !strings.Contains(err.Error(), "mfa_temporarily_locked") {
		t.Fatalf("expected mfa_temporarily_locked after 5 bad codes, got %v", err)
	}
}
