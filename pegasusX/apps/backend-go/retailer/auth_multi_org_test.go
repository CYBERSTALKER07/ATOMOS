package retailer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"golang.org/x/crypto/bcrypt"
)

func multiOrgOn() *bool {
	v := true
	return &v
}

func multiOrgOff() *bool {
	v := false
	return &v
}

func seedTwoOrgOwners(t *testing.T, svc *Service, phone string) {
	t.Helper()
	ctx := context.Background()
	// Distinct NewID per call
	ids := []string{"user-org-a", "user-org-b"}
	i := 0
	svc.newID = func() string {
		id := ids[i%len(ids)]
		i++
		return id
	}
	if _, err := svc.EnsureOwnerUser(ctx, Retailer{RetailerID: "org-a", Phone: phone, Name: "Shop A"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.EnsureOwnerUser(ctx, Retailer{RetailerID: "org-b", Phone: phone, Name: "Shop B"}); err != nil {
		t.Fatal(err)
	}
}

func TestLogin_FlagOff_SingleOrgUnchanged(t *testing.T) {
	t.Parallel()
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass12"), bcrypt.MinCost)
	svc := NewService(ServiceConfig{
		JWTSecret:            "secret",
		JWTIssuer:            "test",
		Now:                  time.Now,
		NewID:                func() string { return "u-single" },
		MultiOrgLoginEnabled: multiOrgOff(),
	})
	ctx := context.Background()
	owner, err := svc.EnsureOwnerUser(ctx, Retailer{RetailerID: "org-1", Phone: "+998901111111", Name: "One"})
	if err != nil {
		t.Fatal(err)
	}
	owner.PasswordHash = string(hash)
	svc.mu.Lock()
	svc.ownerByRetailer["org-1"] = owner
	svc.mu.Unlock()

	body := `{"phone":"+998901111111","password":"pass12"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/login", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	svc.HandleRetailerLogin(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["token_type"] != "full" && resp["token_type"] != nil {
		// token_type full is set on marshal; tolerate absent for backward compat
	}
	if tt, _ := resp["token_type"].(string); tt != "" && tt != "full" {
		t.Fatalf("token_type=%v want full", resp["token_type"])
	}
	token, _ := resp["token"].(string)
	if token == "" {
		t.Fatal("expected full token")
	}
	claims, err := auth.Parse(token, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if auth.IsPendingOrgSelect(claims) {
		t.Fatal("flag off must not issue intermediate token")
	}
	if claims.RetailerOrgID != "org-1" {
		t.Fatalf("org=%s", claims.RetailerOrgID)
	}
}

func TestLogin_FlagOn_MultiOrg_PendingToken(t *testing.T) {
	t.Parallel()
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass12"), bcrypt.MinCost)
	phone := "+998902222222"
	svc := NewService(ServiceConfig{
		JWTSecret:            "secret",
		JWTIssuer:            "test",
		Now:                  time.Now,
		MultiOrgLoginEnabled: multiOrgOn(),
	})
	seedTwoOrgOwners(t, svc, phone)
	// Set password on both owners
	svc.mu.Lock()
	for rid, u := range svc.ownerByRetailer {
		u.PasswordHash = string(hash)
		svc.ownerByRetailer[rid] = u
	}
	svc.mu.Unlock()

	body := `{"phone":"` + phone + `","password":"pass12"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/login", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	svc.HandleRetailerLogin(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["token_type"] != "pending_org_select" {
		t.Fatalf("token_type=%v body=%s", resp["token_type"], rr.Body.String())
	}
	token, _ := resp["token"].(string)
	claims, err := auth.Parse(token, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if !auth.IsPendingOrgSelect(claims) {
		t.Fatal("expected PendingOrgSelect")
	}
	if claims.PhoneNumber != phone {
		t.Fatalf("phone claim=%s", claims.PhoneNumber)
	}
	if claims.RetailerOrgID != "" {
		t.Fatalf("intermediate must not bind org, got %s", claims.RetailerOrgID)
	}
	ms, ok := resp["memberships"].([]any)
	if !ok || len(ms) < 2 {
		t.Fatalf("memberships=%v", resp["memberships"])
	}
}

func TestSelectOrg_IssuesFullJWT(t *testing.T) {
	t.Parallel()
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass12"), bcrypt.MinCost)
	phone := "+998903333333"
	svc := NewService(ServiceConfig{
		JWTSecret:            "secret",
		JWTIssuer:            "test",
		Now:                  time.Now,
		MultiOrgLoginEnabled: multiOrgOn(),
	})
	seedTwoOrgOwners(t, svc, phone)
	svc.mu.Lock()
	for rid, u := range svc.ownerByRetailer {
		u.PasswordHash = string(hash)
		svc.ownerByRetailer[rid] = u
	}
	svc.mu.Unlock()

	// Login → intermediate
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/login",
		bytes.NewBufferString(`{"phone":"`+phone+`","password":"pass12"}`))
	rr := httptest.NewRecorder()
	svc.HandleRetailerLogin(rr, req)
	var login map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &login)
	inter, _ := login["token"].(string)
	if inter == "" {
		t.Fatalf("no intermediate: %s", rr.Body.String())
	}

	// select-org
	selReq := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/select-org",
		bytes.NewBufferString(`{"retailer_id":"org-b"}`))
	selReq = selReq.WithContext(auth.WithClaims(selReq.Context(), mustParse(t, inter, "secret")))
	selRR := httptest.NewRecorder()
	svc.HandleSelectOrg(selRR, selReq)
	if selRR.Code != http.StatusOK {
		t.Fatalf("select status=%d body=%s", selRR.Code, selRR.Body.String())
	}
	var sel map[string]any
	_ = json.Unmarshal(selRR.Body.Bytes(), &sel)
	if sel["token_type"] != "full" {
		t.Fatalf("token_type=%v", sel["token_type"])
	}
	full, _ := sel["token"].(string)
	claims, err := auth.Parse(full, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if auth.IsPendingOrgSelect(claims) {
		t.Fatal("full token must not be pending")
	}
	if claims.RetailerOrgID != "org-b" {
		t.Fatalf("org=%s", claims.RetailerOrgID)
	}
}

func TestSelectOrg_NotAMember(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{JWTSecret: "secret", JWTIssuer: "t", Now: time.Now, MultiOrgLoginEnabled: multiOrgOn()})
	phone := "+998904444444"
	seedTwoOrgOwners(t, svc, phone)
	claims := auth.Claims{
		Subject: "x", Role: auth.RoleRetailer, TokenUse: auth.TokenUsePendingOrgSelect, PhoneNumber: phone,
	}
	tok, err := auth.Issue(claims, auth.IssueOptions{Secret: "secret", Issuer: "t", TTL: 7 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/select-org",
		bytes.NewBufferString(`{"retailer_id":"org-unknown"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), mustParse(t, tok, "secret")))
	rr := httptest.NewRecorder()
	svc.HandleSelectOrg(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rr.Code)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("NOT_A_MEMBER")) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestRequireRole_RejectsPendingOrgSelect(t *testing.T) {
	t.Parallel()
	claims := auth.Claims{
		Subject: "u", Role: auth.RoleRetailer, TokenUse: auth.TokenUsePendingOrgSelect, PhoneNumber: "+1",
	}
	h := auth.RequireRole(auth.RoleRetailer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/me", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rr.Code)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("ORG_SELECT_REQUIRED")) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestSwitchOrg_FullToken(t *testing.T) {
	t.Parallel()
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass12"), bcrypt.MinCost)
	phone := "+998905555555"
	svc := NewService(ServiceConfig{
		JWTSecret: "secret", JWTIssuer: "test", Now: time.Now, MultiOrgLoginEnabled: multiOrgOn(),
	})
	seedTwoOrgOwners(t, svc, phone)
	svc.mu.Lock()
	for rid, u := range svc.ownerByRetailer {
		u.PasswordHash = string(hash)
		svc.ownerByRetailer[rid] = u
	}
	ua := svc.ownerByRetailer["org-a"]
	svc.mu.Unlock()

	// Full JWT for org-a
	fullClaims := auth.Claims{
		Subject: ua.UserID, Role: auth.RoleRetailer, RetailerOrgID: "org-a",
		RetailerUserID: ua.UserID, RetailerRole: "OWNER", PhoneNumber: phone,
	}
	tok, err := auth.Issue(fullClaims, auth.IssueOptions{Secret: "secret", Issuer: "test", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/switch-org",
		bytes.NewBufferString(`{"retailer_id":"org-b"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), mustParse(t, tok, "secret")))
	rr := httptest.NewRecorder()
	svc.HandleSwitchOrg(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	claims, err := auth.Parse(resp["token"].(string), "secret")
	if err != nil {
		t.Fatal(err)
	}
	if claims.RetailerOrgID != "org-b" {
		t.Fatalf("org=%s", claims.RetailerOrgID)
	}
}

func TestPendingToken_TTLInRange(t *testing.T) {
	t.Parallel()
	ttl := pendingOrgSelectTTL()
	if ttl < 5*time.Minute || ttl > 10*time.Minute {
		t.Fatalf("ttl=%v out of 5–10m", ttl)
	}
}

func mustParse(t *testing.T, token, secret string) auth.Claims {
	t.Helper()
	c, err := auth.Parse(token, secret)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
