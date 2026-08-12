package auth

import (
	"context"
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
