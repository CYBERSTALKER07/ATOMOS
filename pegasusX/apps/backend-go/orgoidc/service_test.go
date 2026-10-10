package orgoidc

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func signIDToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	h := base64.RawURLEncoding.EncodeToString(header)
	p := base64.RawURLEncoding.EncodeToString(payload)
	sum := sha256.Sum256([]byte(h + "." + p))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return h + "." + p + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func testSvc(t *testing.T, key *rsa.PrivateKey) *Service {
	t.Helper()
	return &Service{
		Store:     NewMemoryStore(),
		JWTSecret: "gs-i-test-secret",
		JWTIssuer: "pegasusx-test",
		JWTTTL:    time.Hour,
		Now:       func() time.Time { return time.Now().UTC() },
		Keys: func(ctx context.Context, iss, kid string) (*rsa.PublicKey, error) {
			return &key.PublicKey, nil
		},
	}
}

func TestAttachRejectsHTTPIssuer(t *testing.T) {
	svc := testSvc(t, mustKey(t))
	_, err := svc.Attach(context.Background(), Config{
		SupplierID: "sup-1",
		Issuer:     "http://evil.example/idp",
		ClientID:   "client",
	})
	if !errors.Is(err, ErrInvalidIssuer) {
		t.Fatalf("err=%v", err)
	}
}

func TestDiscoveryAndExchange(t *testing.T) {
	key := mustKey(t)
	svc := testSvc(t, key)
	cfg, err := svc.Attach(context.Background(), Config{
		SupplierID:  "sup-1",
		Issuer:      "https://idp.example/realms/org",
		ClientID:    "pegasusx-sup-1",
		RedirectURI: "https://supplier.pegasusx.app/auth/oidc/callback",
		AdminEmails: []string{"buyer@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	disc, err := svc.Discovery(context.Background(), "sup-1", "n1", "st", "")
	if err != nil {
		t.Fatal(err)
	}
	if disc["enabled"] != true || !strings.Contains(disc["authorization_url"].(string), "client_id=pegasusx-sup-1") {
		t.Fatalf("disc=%v", disc)
	}
	tok := signIDToken(t, key, map[string]any{
		"iss":   cfg.Issuer,
		"aud":   cfg.ClientID,
		"exp":   svc.now().Add(time.Hour).Unix(),
		"sub":   "user-99",
		"email": "buyer@example.com",
		"nonce": "n1",
	})
	access, refresh, err := svc.Exchange(context.Background(), "sup-1", tok, "n1")
	if err != nil {
		t.Fatal(err)
	}
	got, err := auth.Parse(access, "gs-i-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if got.Role != auth.RoleAdmin || got.SupplierID != "sup-1" || got.Subject != "buyer@example.com" {
		t.Fatalf("claims=%+v", got)
	}
	if refresh == "" {
		t.Fatal("missing refresh")
	}
}

func TestExchangeRejectsWrongIssuer(t *testing.T) {
	key := mustKey(t)
	svc := testSvc(t, key)
	_, err := svc.Attach(context.Background(), Config{
		SupplierID: "sup-1",
		Issuer:     "https://idp.example/a",
		ClientID:   "c1",
	})
	if err != nil {
		t.Fatal(err)
	}
	tok := signIDToken(t, key, map[string]any{
		"iss": "https://idp.example/other",
		"aud": "c1",
		"exp": svc.now().Add(time.Hour).Unix(),
		"sub": "u",
	})
	_, _, err = svc.Exchange(context.Background(), "sup-1", tok, "")
	if !errors.Is(err, ErrIssuerMismatch) {
		t.Fatalf("err=%v", err)
	}
}

func TestExchangeDoesNotWrapHS256(t *testing.T) {
	// Staff JWT still verifies with the cell secret after OIDC is attached.
	key := mustKey(t)
	svc := testSvc(t, key)
	staff, err := auth.Issue(auth.Claims{Subject: "driver-1", Role: auth.RoleDriver, SupplierID: "sup-1"}, auth.IssueOptions{
		Secret: "gs-i-test-secret", Issuer: "pegasusx-test", TTL: time.Hour, Now: svc.now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Parse(staff, "gs-i-test-secret"); err != nil {
		t.Fatalf("staff HS256 must still parse: %v", err)
	}
}

func TestHandleDiscoveryNotConfigured(t *testing.T) {
	svc := testSvc(t, mustKey(t))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/oidc/discovery?supplier_id=missing", nil)
	svc.HandleDiscovery(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestHandlePutRequiresClaims(t *testing.T) {
	svc := testSvc(t, mustKey(t))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/supplier/oidc", strings.NewReader(`{"issuer":"https://idp.example","client_id":"c"}`))
	svc.HandlePut(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rr.Code)
	}
}

func mustKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	return k
}
