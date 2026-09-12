package partner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIssueAndParsePartnerAccessToken(t *testing.T) {
	secret := ResolvePartnerJWTSecret("human-secret")
	tok, exp, err := IssuePartnerAccessToken(secret, PartnerAccessClaims{
		KeyID: "k1", TenantType: TenantSupplier, TenantID: "sup-1",
		Scopes: []string{ScopeOrdersRead, ScopeExportsRead},
	}, 0)
	if err != nil || exp <= 0 || tok == "" {
		t.Fatalf("issue: tok=%q exp=%d err=%v", tok, exp, err)
	}
	c, err := ParsePartnerAccessToken(tok, secret)
	if err != nil {
		t.Fatal(err)
	}
	if c.KeyID != "k1" || c.TenantID != "sup-1" || !HasScope(c.Scopes, ScopeOrdersRead) {
		t.Fatalf("%+v", c)
	}
	// Wrong secret must fail.
	if _, err := ParsePartnerAccessToken(tok, "other"); err == nil {
		t.Fatal("expected verify fail")
	}
}

func TestIntersectScopes(t *testing.T) {
	key := []string{ScopeOrdersRead, ScopeExportsRead}
	got, err := IntersectScopes(key, nil)
	if err != nil || len(got) != 2 {
		t.Fatalf("%v %v", got, err)
	}
	got, err = IntersectScopes(key, []string{ScopeOrdersRead})
	if err != nil || len(got) != 1 || got[0] != ScopeOrdersRead {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := IntersectScopes(key, []string{ScopeOrdersWrite}); err == nil {
		t.Fatal("expected invalid_scope")
	}
}

func TestClientCredentialsAndAuthMiddleware(t *testing.T) {
	keys := NewMemoryKeyRepository()
	svc := NewService(keys, NewMemoryWebhookRepository(), nil, nil, nil)
	secret := ResolvePartnerJWTSecret("test-jwt")
	svc.SetOAuthJWT(secret, "test-partner", partnerOAuthTTL())
	issued, err := svc.IssueKey(context.Background(), TenantRetailer, "ret-1", "tester", []string{ScopeOrdersRead, ScopeOrdersWrite})
	if err != nil {
		t.Fatal(err)
	}

	tok, err := svc.IssueClientCredentials(context.Background(), issued.KeyID, issued.Secret, ScopeOrdersRead)
	if err != nil {
		t.Fatal(err)
	}
	if tok.TokenType != "Bearer" || tok.AccessToken == "" || !strings.Contains(tok.Scope, ScopeOrdersRead) {
		t.Fatalf("%+v", tok)
	}

	h := AuthMiddlewareOpts(AuthOptions{Keys: keys, JWTSecret: secret})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFromContext(r.Context())
		if !ok || p.TenantID != "ret-1" || !HasScope(p.Scopes, ScopeOrdersRead) {
			t.Fatalf("principal=%+v ok=%v", p, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/partner/v1/orders/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	// Revoke key → JWT rejected
	_ = keys.Revoke(context.Background(), issued.KeyID, TenantRetailer, "ret-1")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req)
	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("revoked status=%d", rr2.Code)
	}
}

func TestHandleOAuthTokenJSON(t *testing.T) {
	keys := NewMemoryKeyRepository()
	svc := NewService(keys, NewMemoryWebhookRepository(), nil, nil, nil)
	secret := ResolvePartnerJWTSecret("test-jwt-2")
	svc.SetOAuthJWT(secret, "test", 0)
	issued, err := svc.IssueKey(context.Background(), TenantSupplier, "sup-1", "t", []string{"*"})
	if err != nil {
		t.Fatal(err)
	}
	h := &Handlers{Svc: svc}
	body, _ := json.Marshal(map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     issued.KeyID,
		"client_secret": issued.Secret,
	})
	req := httptest.NewRequest(http.MethodPost, "/partner/v1/oauth/token", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.HandleOAuthToken(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp TokenResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil || resp.AccessToken == "" {
		t.Fatalf("%s err=%v", rr.Body.String(), err)
	}
}
