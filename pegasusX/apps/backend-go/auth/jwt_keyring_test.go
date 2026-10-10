package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestJWTKeyring_SingleKey(t *testing.T) {
	kr := NewKeyring("secret-one")
	claims := Claims{
		Subject:    "user-1",
		Role:       RoleAdmin,
		SupplierID: "sup-1",
		MarketCode: "UZ",
		HomeCell:   "cell-uz-1",
	}

	token, err := Issue(claims, IssueOptions{Keyring: kr})
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	parsed, err := ParseWithKeyring(token, kr)
	if err != nil {
		t.Fatalf("parse with keyring failed: %v", err)
	}
	if parsed.Subject != "user-1" {
		t.Errorf("expected subject user-1, got %s", parsed.Subject)
	}

	// Verify legacy Parse also works
	parsedLegacy, err := Parse(token, "secret-one")
	if err != nil {
		t.Fatalf("legacy parse failed: %v", err)
	}
	if parsedLegacy.Subject != "user-1" {
		t.Errorf("expected subject user-1, got %s", parsedLegacy.Subject)
	}
}

func TestJWTKeyring_MultiKey_FallbackVerification(t *testing.T) {
	secretOld := "old-secret-phase0"
	secretNew := "new-secret-phase1"

	// Token issued under old secret
	claims := Claims{
		Subject:    "driver-42",
		Role:       RoleDriver,
		SupplierID: "sup-global",
		MarketCode: "UZ",
		HomeCell:   "cell-uz-1",
	}
	tokenOld, err := Issue(claims, IssueOptions{Secret: secretOld})
	if err != nil {
		t.Fatalf("issue old token failed: %v", err)
	}

	// Keyring has new secret as primary, old secret as fallback
	kr := NewKeyring(secretNew, secretOld)

	// Old token must still verify without error (zero session invalidation)
	parsedOld, err := ParseWithKeyring(tokenOld, kr)
	if err != nil {
		t.Fatalf("verification of old token with rotated keyring failed: %v", err)
	}
	if parsedOld.Subject != "driver-42" {
		t.Errorf("expected subject driver-42, got %s", parsedOld.Subject)
	}

	// New token issued under keyring
	tokenNew, err := Issue(claims, IssueOptions{Keyring: kr})
	if err != nil {
		t.Fatalf("issue new token failed: %v", err)
	}

	parsedNew, err := ParseWithKeyring(tokenNew, kr)
	if err != nil {
		t.Fatalf("verification of new token failed: %v", err)
	}
	if parsedNew.Subject != "driver-42" {
		t.Errorf("expected subject driver-42, got %s", parsedNew.Subject)
	}

	// Unrelated secret must fail
	_, err = Parse(tokenNew, "random-attacker-secret")
	if err == nil {
		t.Fatal("expected failure for invalid secret, got nil")
	}
}

func TestJWTKeyring_ZeroDowntimeRotation(t *testing.T) {
	kr := NewKeyringWithKID("secret-v1", "kid-v1")

	claims := Claims{
		Subject:    "cashier-99",
		Role:       RoleRetailer,
		SupplierID: "sup-store",
		MarketCode: "UZ",
		HomeCell:   "cell-uz-1",
	}

	// 1. Issue token under v1
	tokenV1, err := Issue(claims, IssueOptions{Keyring: kr})
	if err != nil {
		t.Fatalf("issue v1: %v", err)
	}

	// 2. Rotate to v2 at runtime
	kr.Rotate("secret-v2", "kid-v2")

	// 3. Issue token under v2
	tokenV2, err := Issue(claims, IssueOptions{Keyring: kr})
	if err != nil {
		t.Fatalf("issue v2: %v", err)
	}

	// 4. Both tokens verify successfully against active keyring
	pV1, err := ParseWithKeyring(tokenV1, kr)
	if err != nil {
		t.Fatalf("v1 token failed after rotation: %v", err)
	}
	if pV1.Subject != "cashier-99" {
		t.Errorf("unexpected subject: %s", pV1.Subject)
	}

	pV2, err := ParseWithKeyring(tokenV2, kr)
	if err != nil {
		t.Fatalf("v2 token failed: %v", err)
	}
	if pV2.Subject != "cashier-99" {
		t.Errorf("unexpected subject: %s", pV2.Subject)
	}
}

func TestJWTKeyring_MiddlewareContextAttachment(t *testing.T) {
	kr := NewKeyring("current-key", "old-key")

	claims := Claims{
		Subject:    "operator-1",
		Role:       RoleWarehouse,
		SupplierID: "wh-sup",
		MarketCode: "UZ",
		HomeCell:   "cell-uz-1",
	}

	// Token minted with old key
	oldToken, _ := Issue(claims, IssueOptions{Secret: "old-key"})

	// Handler verifying attached claims in context
	var attachedSub string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, ok := FromContext(r.Context()); ok {
			attachedSub = c.Subject
		}
		w.WriteHeader(http.StatusOK)
	})

	// 1. Test CookieAuthWithKeyring
	cookieMiddleware := CookieAuthWithKeyring(kr)(next)
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: oldToken})
	rec := httptest.NewRecorder()
	cookieMiddleware.ServeHTTP(rec, req)

	if attachedSub != "operator-1" {
		t.Errorf("CookieAuthWithKeyring did not attach claims, got sub: %s", attachedSub)
	}

	// 2. Test SessionAuthWithKeyring with Bearer token
	attachedSub = ""
	sessionMiddleware := SessionAuthWithKeyring(kr)(next)
	reqBearer := httptest.NewRequest("GET", "/test", nil)
	reqBearer.Header.Set("Authorization", "Bearer "+oldToken)
	recBearer := httptest.NewRecorder()
	sessionMiddleware.ServeHTTP(recBearer, reqBearer)

	if attachedSub != "operator-1" {
		t.Errorf("SessionAuthWithKeyring did not attach claims, got sub: %s", attachedSub)
	}
}

func TestNewKeyringFromEnv(t *testing.T) {
	_ = os.Setenv("JWT_SECRET_CURRENT", "secret-cur")
	_ = os.Setenv("JWT_KEY_ID", "kid-cur")
	_ = os.Setenv("JWT_SECRET_PREVIOUS", "secret-prev")
	_ = os.Setenv("JWT_SECRETS", "extra1,k2:extra2")
	defer func() {
		_ = os.Unsetenv("JWT_SECRET_CURRENT")
		_ = os.Unsetenv("JWT_KEY_ID")
		_ = os.Unsetenv("JWT_SECRET_PREVIOUS")
		_ = os.Unsetenv("JWT_SECRETS")
	}()

	kr := NewKeyringFromEnv()
	if !kr.HasKeys() {
		t.Fatal("expected keyring to have keys")
	}
	cur, kid := kr.CurrentKey()
	if cur != "secret-cur" || kid != "kid-cur" {
		t.Fatalf("unexpected current key: %s (kid %s)", cur, kid)
	}

	candidates := kr.VerifyCandidateKeys("")
	if len(candidates) < 4 {
		t.Fatalf("expected at least 4 candidate keys, got %d", len(candidates))
	}
}
