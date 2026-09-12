package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMemoryRevocationStore_RevokeAndCheck(t *testing.T) {
	store := NewMemoryRevocationStore()
	SetRevocationStore(store)
	t.Cleanup(func() { SetRevocationStore(NewMemoryRevocationStore()) })

	if err := store.Revoke(context.Background(), "jti-1", time.Hour); err != nil {
		t.Fatal(err)
	}
	ok, err := store.IsRevoked(context.Background(), "jti-1")
	if err != nil || !ok {
		t.Fatalf("expected revoked, ok=%v err=%v", ok, err)
	}
	ok, err = store.IsRevoked(context.Background(), "other")
	if err != nil || ok {
		t.Fatalf("expected not revoked, ok=%v err=%v", ok, err)
	}
}

func TestIssueParse_IncludesJTI(t *testing.T) {
	tok, err := Issue(Claims{Subject: "u1", Role: RoleDriver, SupplierID: "s1"}, IssueOptions{
		Secret: "test-secret",
		Issuer: "test",
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Parse(tok, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(claims.JTI) == "" {
		t.Fatal("expected jti on issued token")
	}
	if claims.ExpiresAt.IsZero() {
		t.Fatal("expected expires_at")
	}
}

func TestLogoutRevokesRefresh(t *testing.T) {
	store := NewMemoryRevocationStore()
	SetRevocationStore(store)
	t.Cleanup(func() { SetRevocationStore(NewMemoryRevocationStore()) })

	secret := "test-secret"
	tok, err := Issue(Claims{Subject: "u1", Role: RoleDriver, SupplierID: "s1"}, IssueOptions{
		Secret: secret,
		Issuer: "test",
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+tok)
	logoutRR := httptest.NewRecorder()
	HandleLogout(secret)(logoutRR, logoutReq)
	if logoutRR.Code != http.StatusOK {
		t.Fatalf("logout status=%d body=%s", logoutRR.Code, logoutRR.Body.String())
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", nil)
	refreshReq.Header.Set("Authorization", "Bearer "+tok)
	refreshRR := httptest.NewRecorder()
	HandleTokenRefresh(secret, "test")(refreshRR, refreshReq)
	if refreshRR.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked refresh 401, got %d body=%s", refreshRR.Code, refreshRR.Body.String())
	}
	if !strings.Contains(refreshRR.Body.String(), "token_revoked") {
		t.Fatalf("expected token_revoked, got %s", refreshRR.Body.String())
	}
}

type errRevocationStore struct{}

func (errRevocationStore) Revoke(context.Context, string, time.Duration) error {
	return errors.New("store down")
}
func (errRevocationStore) IsRevoked(context.Context, string) (bool, error) {
	return false, errors.New("store down")
}

func TestTokenRevoked_StoreErrorFailsClosed(t *testing.T) {
	SetRevocationStore(errRevocationStore{})
	t.Cleanup(func() { SetRevocationStore(NewMemoryRevocationStore()) })
	if !tokenRevoked(context.Background(), Claims{JTI: "jti-x"}) {
		t.Fatal("store error must treat token as revoked")
	}
	if tokenRevoked(context.Background(), Claims{}) {
		t.Fatal("legacy empty jti still accepted")
	}
}

func TestSessionAuth_StoreErrorDoesNotAttach(t *testing.T) {
	SetRevocationStore(errRevocationStore{})
	t.Cleanup(func() { SetRevocationStore(NewMemoryRevocationStore()) })
	tok, err := Issue(Claims{Subject: "u1", Role: RoleAdmin, SupplierID: "s1"}, IssueOptions{
		Secret: "sec", TTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	SessionAuth("sec")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := FromContext(r.Context()); ok {
			t.Fatal("must not attach claims when revocation store is down")
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rr, req)
}

func TestRefresh_StoreError503(t *testing.T) {
	SetRevocationStore(errRevocationStore{})
	t.Cleanup(func() { SetRevocationStore(NewMemoryRevocationStore()) })
	tok, err := Issue(Claims{Subject: "u1", Role: RoleAdmin, SupplierID: "s1"}, IssueOptions{
		Secret: "sec", TTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	HandleTokenRefresh("sec", "test")(rr, req)
	if rr.Code != http.StatusServiceUnavailable || !strings.Contains(rr.Body.String(), "revocation_store_unavailable") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestEntitySupplierAllowed(t *testing.T) {
	ctx := WithClaims(context.Background(), Claims{Role: RoleAdmin, SupplierID: "sup-a"})
	ctx = WithTenant(ctx, TenantContext{SupplierID: "sup-a", Source: "jwt"})
	if !EntitySupplierAllowed(ctx, "sup-a") {
		t.Fatal("same supplier must allow")
	}
	if EntitySupplierAllowed(ctx, "sup-b") {
		t.Fatal("cross supplier must deny")
	}
	plat := WithClaims(context.Background(), Claims{Role: RolePlatformAdmin})
	if !EntitySupplierAllowed(plat, "sup-b") {
		t.Fatal("platform admin bypass")
	}
}
