package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testWSTicket(t *testing.T, secret string) string {
	t.Helper()
	tok, _, err := IssueWSTicket(Claims{
		Subject:    "u1",
		Role:       RoleAdmin,
		SupplierID: "sup-1",
	}, IssueOptions{Secret: secret, TTL: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestSessionAuthDoesNotAttachWSTicket(t *testing.T) {
	secret := "ws-ticket-secret"
	tok := testWSTicket(t, secret)
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/orders", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	SessionAuth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := FromContext(r.Context()); ok {
			t.Fatal("ws ticket must not attach as session claims")
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestParseBearerClaimsRejectsWSTicket(t *testing.T) {
	secret := "ws-ticket-secret"
	tok := testWSTicket(t, secret)
	req := httptest.NewRequest(http.MethodGet, "/v1/ws", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	if _, ok := ParseBearerClaims(req, secret); ok {
		t.Fatal("ParseBearerClaims must reject ws tickets")
	}
}

func TestRequireRoleRejectsWSTicket(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/orders", nil)
	req = req.WithContext(WithClaims(req.Context(), Claims{
		Subject: "u1", Role: RoleAdmin, SupplierID: "sup-1", TokenUse: TokenUseWS,
	}))
	rr := httptest.NewRecorder()
	RequireRole(RoleAdmin)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("must not reach handler")
	})).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden || !strings.Contains(rr.Body.String(), "ws_ticket_not_allowed") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRequireAnyAuthenticatedRejectsWSTicket(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/session", nil)
	req = req.WithContext(WithClaims(req.Context(), Claims{
		Subject: "u1", Role: RoleAdmin, TokenUse: TokenUseWS,
	}))
	rr := httptest.NewRecorder()
	RequireAnyAuthenticated()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("must not reach handler")
	})).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRefreshRejectsWSTicket(t *testing.T) {
	secret := "ws-ticket-secret"
	tok := testWSTicket(t, secret)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	HandleTokenRefresh(secret, "test")(rr, req)
	if rr.Code != http.StatusUnauthorized || !strings.Contains(rr.Body.String(), "ws_ticket_not_refreshable") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestIssueWSTicketRejectsTicketInput(t *testing.T) {
	_, _, err := IssueWSTicket(Claims{Subject: "u1", Role: RoleAdmin, TokenUse: TokenUseWS}, IssueOptions{
		Secret: "s", TTL: time.Minute,
	})
	if err == nil {
		t.Fatal("must not mint ticket from ticket")
	}
}
