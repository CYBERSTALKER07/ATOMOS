package iamroutes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/iam"
)

const testSecret = "test-iamroutes-secret-2026-0123456789"

func issueCookie(t *testing.T, c auth.Claims) *http.Cookie {
	t.Helper()
	tok, err := auth.Issue(c, auth.IssueOptions{Secret: testSecret, TTL: time.Hour})
	if err != nil {
		t.Fatalf("auth.Issue: %v", err)
	}
	return &http.Cookie{Name: auth.CookieName, Value: tok}
}

func doReq(t *testing.T, h http.Handler, method, path string, body any, c *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c != nil {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestIAMRoutes_RBAC_SoD_And_TenantIsolation(t *testing.T) {
	t.Parallel()
	svc := iam.NewServiceWithRepository(iam.NewMemoryRepository())
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{Service: svc, JWTSecret: testSecret})

	ownerA := issueCookie(t, auth.Claims{Subject: "own-a", Role: auth.RoleAdmin, SupplierID: "sup-a"})
	auditorA := issueCookie(t, auth.Claims{Subject: "aud-a", Role: auth.RoleAdmin, SupplierID: "sup-a", SupplierRole: auth.Role(auth.SupplierRoleAuditor)})
	agentA := issueCookie(t, auth.Claims{Subject: "agt-a", Role: auth.RoleAdmin, SupplierID: "sup-a", ActorType: auth.ActorTypeAgent})
	platformAdmin := issueCookie(t, auth.Claims{Subject: "pa-1", Role: auth.RolePlatformAdmin})

	// 1. Unauthenticated -> 401
	if res := doReq(t, r, http.MethodGet, "/v1/iam/roles", nil, nil); res.Code != http.StatusUnauthorized {
		t.Fatalf("unauth GET /v1/iam/roles = %d, want 401", res.Code)
	}

	// 2. AUDITOR can read GET /v1/iam/roles and GET /v1/iam/sod-policies, but CANNOT POST/PUT (403).
	if res := doReq(t, r, http.MethodGet, "/v1/iam/roles", nil, auditorA); res.Code != http.StatusOK {
		t.Fatalf("auditor GET /v1/iam/roles = %d, want 200 (%s)", res.Code, res.Body.String())
	}
	if res := doReq(t, r, http.MethodPost, "/v1/iam/users/u1/roles", map[string]any{"role_name": "ACCOUNTANT"}, auditorA); res.Code != http.StatusForbidden {
		t.Fatalf("auditor POST role = %d, want 403", res.Code)
	}

	// 3. AGENT cannot assign roles or modify policies -> 403.
	if res := doReq(t, r, http.MethodPut, "/v1/iam/sod-policies/sod_credit_note", map[string]any{"enabled": false}, agentA); res.Code != http.StatusForbidden {
		t.Fatalf("agent PUT sod-policy = %d, want 403", res.Code)
	}

	// 4. Cross-tenant ?entity=sup-b by ownerA -> 403.
	if res := doReq(t, r, http.MethodGet, "/v1/iam/sod-policies?entity=sup-b", nil, ownerA); res.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant GET sod-policies = %d, want 403", res.Code)
	}

	// 5. Owner assigns ACCOUNTANT -> 200, then conflicting FINANCE_MANAGER -> 403 sod_role_conflict.
	res1 := doReq(t, r, http.MethodPost, "/v1/iam/users/u1/roles", map[string]any{"role_name": "ACCOUNTANT"}, ownerA)
	if res1.Code != http.StatusOK {
		t.Fatalf("assign ACCOUNTANT = %d body=%s", res1.Code, res1.Body.String())
	}
	res2 := doReq(t, r, http.MethodPost, "/v1/iam/users/u1/roles", map[string]any{"role_name": "FINANCE_MANAGER"}, ownerA)
	if res2.Code != http.StatusForbidden {
		t.Fatalf("conflicting assign FINANCE_MANAGER = %d, want 403 (%s)", res2.Code, res2.Body.String())
	}

	// 6. Violations visible on GET /v1/iam/sod-violations and cross-tenant via PLATFORM_ADMIN.
	violRes := doReq(t, r, http.MethodGet, "/v1/iam/sod-violations", nil, ownerA)
	if violRes.Code != http.StatusOK {
		t.Fatalf("GET sod-violations = %d body=%s", violRes.Code, violRes.Body.String())
	}
	var vp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(violRes.Body.Bytes(), &vp); err != nil || vp.Count != 1 {
		t.Fatalf("violations payload = %s err=%v", violRes.Body.String(), err)
	}

	paRes := doReq(t, r, http.MethodGet, "/v1/iam/sod-violations?entity=sup-a", nil, platformAdmin)
	if paRes.Code != http.StatusOK {
		t.Fatalf("platform admin GET sod-violations = %d body=%s", paRes.Code, paRes.Body.String())
	}

	// 7. PUT /v1/iam/sod-policies/{id} toggles policy cleanly.
	putRes := doReq(t, r, http.MethodPut, "/v1/iam/sod-policies/sod_credit_note", map[string]any{"enabled": false}, ownerA)
	if putRes.Code != http.StatusOK {
		t.Fatalf("PUT sod-policy = %d body=%s", putRes.Code, putRes.Body.String())
	}

	// 8. GET /v1/iam/permissions returns 23 permissions.
	permRes := doReq(t, r, http.MethodGet, "/v1/iam/permissions", nil, ownerA)
	if permRes.Code != http.StatusOK {
		t.Fatalf("GET permissions = %d", permRes.Code)
	}
	var pp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(permRes.Body.Bytes(), &pp); err != nil || pp.Count != 23 {
		t.Fatalf("permissions payload = %s err=%v", permRes.Body.String(), err)
	}
}

func doBearerReq(t *testing.T, h http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestIAMRoutes_AuditorGrants_GrantUseExpireRevokeRegrant(t *testing.T) {
	t.Parallel()
	svc := iam.NewServiceWithRepository(iam.NewMemoryRepository())
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{Service: svc, JWTSecret: testSecret, JWTIssuer: "atomos-test"})

	ownerCookie := issueCookie(t, auth.Claims{Subject: "owner-1", Role: auth.RoleAdmin, SupplierID: "sup-1"})
	agentCookie := issueCookie(t, auth.Claims{Subject: "agent-1", Role: auth.RoleAdmin, SupplierID: "sup-1", ActorType: auth.ActorTypeAgent})

	// Past expires_at -> 400
	past := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)
	if res := doReq(t, r, http.MethodPost, "/v1/iam/auditor-grants", map[string]any{
		"user_id": "aud-1", "expires_at": past,
	}, ownerCookie); res.Code != http.StatusBadRequest {
		t.Fatalf("past expires_at = %d, want 400 (%s)", res.Code, res.Body.String())
	}

	// Self-grant -> 403
	if res := doReq(t, r, http.MethodPost, "/v1/iam/auditor-grants", map[string]any{
		"user_id": "owner-1",
	}, ownerCookie); res.Code != http.StatusForbidden {
		t.Fatalf("self-grant = %d, want 403 (%s)", res.Code, res.Body.String())
	}

	// AGENT caller -> 403
	if res := doReq(t, r, http.MethodPost, "/v1/iam/auditor-grants", map[string]any{
		"user_id": "aud-1",
	}, agentCookie); res.Code != http.StatusForbidden {
		t.Fatalf("agent grant = %d, want 403 (%s)", res.Code, res.Body.String())
	}

	// Valid short-lived grant (160ms) -> 201 + Bearer JWT works for read and is 403 on mutations.
	soon := time.Now().UTC().Add(160 * time.Millisecond).Format(time.RFC3339Nano)
	grantRes := doReq(t, r, http.MethodPost, "/v1/iam/auditor-grants", map[string]any{
		"user_id": "aud-short", "expires_at": soon, "reason": "short audit window",
	}, ownerCookie)
	if grantRes.Code != http.StatusCreated {
		t.Fatalf("POST /v1/iam/auditor-grants = %d body=%s", grantRes.Code, grantRes.Body.String())
	}
	var granted struct {
		Grant     iam.AuditorGrant `json:"grant"`
		Token     string           `json:"token"`
		ExpiresAt time.Time        `json:"expires_at"`
	}
	if err := json.Unmarshal(grantRes.Body.Bytes(), &granted); err != nil || granted.Token == "" {
		t.Fatalf("decode grant response: %v body=%s", err, grantRes.Body.String())
	}

	// Issued AUDITOR token can GET /v1/iam/auditor-grants and GET /v1/iam/roles...
	if res := doBearerReq(t, r, http.MethodGet, "/v1/iam/auditor-grants", nil, granted.Token); res.Code != http.StatusOK {
		t.Fatalf("active auditor GET grants = %d body=%s", res.Code, res.Body.String())
	}
	// ...and is 403 on write routes.
	if res := doBearerReq(t, r, http.MethodPost, "/v1/iam/auditor-grants", map[string]any{"user_id": "aud-x"}, granted.Token); res.Code != http.StatusForbidden {
		t.Fatalf("auditor POST grant = %d, want 403", res.Code)
	}

	// Wait past AuditorExpiresAt -> 401 auditor_access_expired!
	time.Sleep(190 * time.Millisecond)
	expRes := doBearerReq(t, r, http.MethodGet, "/v1/iam/auditor-grants", nil, granted.Token)
	if expRes.Code != http.StatusUnauthorized || !bytes.Contains(expRes.Body.Bytes(), []byte("auditor_access_expired")) {
		t.Fatalf("expired auditor GET = %d body=%s, want 401 auditor_access_expired", expRes.Code, expRes.Body.String())
	}

	// Grant 24h, verify active, revoke, verify old token rejected, then re-grant cleanly.
	grant2Res := doReq(t, r, http.MethodPost, "/v1/iam/auditor-grants", map[string]any{
		"user_id": "aud-rev", "ttl_hours": 24,
	}, ownerCookie)
	if grant2Res.Code != http.StatusCreated {
		t.Fatalf("POST 24h grant = %d body=%s", grant2Res.Code, grant2Res.Body.String())
	}
	var granted2 struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(grant2Res.Body.Bytes(), &granted2)
	if res := doBearerReq(t, r, http.MethodGet, "/v1/iam/roles", nil, granted2.Token); res.Code != http.StatusOK {
		t.Fatalf("24h auditor GET roles = %d body=%s", res.Code, res.Body.String())
	}

	revRes := doReq(t, r, http.MethodPost, "/v1/iam/auditor-grants/aud-rev/revoke", map[string]any{}, ownerCookie)
	if revRes.Code != http.StatusOK {
		t.Fatalf("POST revoke = %d body=%s", revRes.Code, revRes.Body.String())
	}
	if res := doBearerReq(t, r, http.MethodGet, "/v1/iam/roles", nil, granted2.Token); res.Code != http.StatusUnauthorized {
		t.Fatalf("revoked auditor JWT GET roles = %d, want 401", res.Code)
	}

	regrantRes := doReq(t, r, http.MethodPost, "/v1/iam/auditor-grants", map[string]any{
		"user_id": "aud-rev", "ttl_hours": 12,
	}, ownerCookie)
	if regrantRes.Code != http.StatusCreated {
		t.Fatalf("re-grant = %d body=%s", regrantRes.Code, regrantRes.Body.String())
	}
	var granted3 struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(regrantRes.Body.Bytes(), &granted3)
	if res := doBearerReq(t, r, http.MethodGet, "/v1/iam/roles", nil, granted3.Token); res.Code != http.StatusOK {
		t.Fatalf("re-granted auditor JWT GET roles = %d body=%s", res.Code, res.Body.String())
	}
}

func TestIAMRoutes_CustomRoles_And_SCIMv2_RBAC_BearerAndAuditorExpiry(t *testing.T) {
	t.Parallel()
	const scimToken = "scim-secret-token-2026"
	svc := iam.NewServiceWithRepository(iam.NewMemoryRepository())
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Service:          svc,
		JWTSecret:        testSecret,
		SCIMTenantTokens: map[string]string{"sup-a": scimToken},
	})

	ownerA := issueCookie(t, auth.Claims{Subject: "own-a", Role: auth.RoleAdmin, SupplierID: "sup-a", ActorType: auth.ActorTypeHuman})
	auditorA := issueCookie(t, auth.Claims{Subject: "aud-a", Role: auth.RoleAdmin, SupplierID: "sup-a", SupplierRole: auth.Role(auth.SupplierRoleAuditor), ActorType: auth.ActorTypeHuman})
	audExpAt := time.Now().UTC().Add(-time.Minute)
	expiredAuditorTok, err := auth.Issue(auth.Claims{
		Subject:          "aud-exp",
		Role:             auth.RoleAdmin,
		SupplierID:       "sup-a",
		SupplierRole:     auth.Role(auth.SupplierRoleAuditor),
		ActorType:        auth.ActorTypeHuman,
		AuditorExpiresAt: &audExpAt,
	}, auth.IssueOptions{Secret: testSecret, TTL: time.Hour})
	if err != nil {
		t.Fatalf("issue expired auditor jwt: %v", err)
	}
	agentA := issueCookie(t, auth.Claims{Subject: "agt-a", Role: auth.RoleAdmin, SupplierID: "sup-a", ActorType: auth.ActorTypeAgent})

	// 1. Custom roles RBAC: AUDITOR 403 POST/DELETE, 200 GET; AGENT 403 POST; OWNER 201 -> 200.
	if res := doReq(t, r, http.MethodPost, "/v1/iam/custom-roles", map[string]any{"role_name": "X", "permissions": []string{"audit.read"}}, auditorA); res.Code != http.StatusForbidden {
		t.Fatalf("auditor POST custom-roles = %d, want 403", res.Code)
	}
	if res := doReq(t, r, http.MethodPost, "/v1/iam/custom-roles", map[string]any{"role_name": "X", "permissions": []string{"audit.read"}}, agentA); res.Code != http.StatusForbidden {
		t.Fatalf("agent POST custom-roles = %d, want 403", res.Code)
	}
	if res := doReq(t, r, http.MethodPost, "/v1/iam/custom-roles", map[string]any{"role_name": "SCIM_CLERK", "permissions": []string{"stock.count", "audit.read"}}, ownerA); res.Code != http.StatusCreated {
		t.Fatalf("owner POST custom-roles = %d body=%s", res.Code, res.Body.String())
	}
	if res := doReq(t, r, http.MethodGet, "/v1/iam/custom-roles", nil, auditorA); res.Code != http.StatusOK {
		t.Fatalf("auditor GET custom-roles = %d body=%s", res.Code, res.Body.String())
	}
	if res := doReq(t, r, http.MethodDelete, "/v1/iam/custom-roles/SCIM_CLERK", nil, auditorA); res.Code != http.StatusForbidden {
		t.Fatalf("auditor DELETE custom-roles = %d, want 403", res.Code)
	}

	// 2. Unauthenticated SCIM -> 401; bad bearer token -> 401; static SCIM token cannot access /v1/iam/custom-roles -> 401.
	if res := doReq(t, r, http.MethodGet, "/scim/v2/Users", nil, nil); res.Code != http.StatusUnauthorized {
		t.Fatalf("unauth GET /scim/v2/Users = %d, want 401", res.Code)
	}
	if res := doBearerReq(t, r, http.MethodGet, "/scim/v2/Users", nil, "wrong-token"); res.Code != http.StatusUnauthorized {
		t.Fatalf("bad bearer GET /scim/v2/Users = %d, want 401", res.Code)
	}
	if res := doBearerReq(t, r, http.MethodGet, "/v1/iam/custom-roles", nil, scimToken); res.Code != http.StatusUnauthorized {
		t.Fatalf("SCIM static token must NOT access /v1/iam/custom-roles = %d, want 401", res.Code)
	}

	// 3. Expired AUDITOR JWT on SCIM GET -> 401 auditor_access_expired.
	if res := doBearerReq(t, r, http.MethodGet, "/scim/v2/Users", nil, expiredAuditorTok); res.Code != http.StatusUnauthorized || !bytes.Contains(res.Body.Bytes(), []byte("auditor_access_expired")) {
		t.Fatalf("expired auditor GET /scim/v2/Users = %d body=%s, want 401 auditor_access_expired", res.Code, res.Body.String())
	}

	// 4. Static SCIM Bearer token provisions User & Group when tenant is provided via X-Tenant-Id.
	doScimBearer := func(method, path string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Authorization", "Bearer "+scimToken)
		req.Header.Set("X-Tenant-Id", "sup-a")
		if body != nil {
			req.Header.Set("Content-Type", "application/scim+json")
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	uRes := doScimBearer(http.MethodPost, "/scim/v2/Users", map[string]any{
		"id":       "scim-br-1",
		"userName": "idp.user@sup-a.test",
		"active":   true,
		"roles":    []map[string]any{{"value": "SCIM_CLERK"}},
	})
	if uRes.Code != http.StatusCreated {
		t.Fatalf("SCIM bearer POST /scim/v2/Users = %d body=%s", uRes.Code, uRes.Body.String())
	}
	if ct := uRes.Header().Get("Content-Type"); !bytes.Contains([]byte(ct), []byte("application/scim+json")) {
		t.Fatalf("expected application/scim+json, got %q", ct)
	}

	gRes := doScimBearer(http.MethodPost, "/scim/v2/Groups", map[string]any{
		"id":          "scim-br-g1",
		"displayName": "SCIM Clerks",
		"roles":       []map[string]any{{"value": "SCIM_CLERK"}},
		"members":     []map[string]any{{"value": "scim-br-1"}},
	})
	if gRes.Code != http.StatusCreated {
		t.Fatalf("SCIM bearer POST /scim/v2/Groups = %d body=%s", gRes.Code, gRes.Body.String())
	}

	// 5. Active AUDITOR cookie can GET /scim/v2/Users & /scim/v2/Groups (200), but 403 on write.
	if res := doReq(t, r, http.MethodGet, "/scim/v2/Users", nil, auditorA); res.Code != http.StatusOK {
		t.Fatalf("auditor GET /scim/v2/Users = %d body=%s", res.Code, res.Body.String())
	}
	if res := doReq(t, r, http.MethodGet, "/scim/v2/Groups", nil, auditorA); res.Code != http.StatusOK {
		t.Fatalf("auditor GET /scim/v2/Groups = %d body=%s", res.Code, res.Body.String())
	}
	if res := doReq(t, r, http.MethodPatch, "/scim/v2/Users/scim-br-1", map[string]any{"Operations": []map[string]any{{"op": "replace", "path": "active", "value": false}}}, auditorA); res.Code != http.StatusForbidden {
		t.Fatalf("auditor PATCH /scim/v2/Users = %d, want 403", res.Code)
	}
	if res := doReq(t, r, http.MethodDelete, "/scim/v2/Users/scim-br-1", nil, agentA); res.Code != http.StatusForbidden {
		t.Fatalf("agent DELETE /scim/v2/Users = %d, want 403", res.Code)
	}

	// 6. Cross-tenant header spoofing with a tenant-bound SCIM token -> 403 Forbidden.
	spoofReq := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", nil)
	spoofReq.Header.Set("Authorization", "Bearer "+scimToken)
	spoofReq.Header.Set("X-Tenant-Id", "sup-b")
	spoofRec := httptest.NewRecorder()
	r.ServeHTTP(spoofRec, spoofReq)
	if spoofRec.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant SCIM spoof = %d body=%s, want 403", spoofRec.Code, spoofRec.Body.String())
	}

	// 7. Unscoped static SCIMBearerToken without SCIMTenantID / SCIMTenantTokens fails closed (403).
	rUnscoped := chi.NewRouter()
	RegisterRoutes(rUnscoped, Deps{
		Service:         iam.NewServiceWithRepository(iam.NewMemoryRepository()),
		JWTSecret:       testSecret,
		SCIMBearerToken: scimToken,
	})
	unscopedReq := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", nil)
	unscopedReq.Header.Set("Authorization", "Bearer "+scimToken)
	unscopedReq.Header.Set("X-Tenant-Id", "sup-a")
	unscopedRec := httptest.NewRecorder()
	rUnscoped.ServeHTTP(unscopedRec, unscopedReq)
	if unscopedRec.Code != http.StatusForbidden {
		t.Fatalf("unscoped static SCIM token = %d body=%s, want 403", unscopedRec.Code, unscopedRec.Body.String())
	}

	// 8. SCIM bearer cleanup DELETE returns 204.
	if res := doScimBearer(http.MethodDelete, "/scim/v2/Groups/scim-br-g1", nil); res.Code != http.StatusNoContent {
		t.Fatalf("SCIM bearer DELETE group = %d, want 204", res.Code)
	}
	if res := doScimBearer(http.MethodDelete, "/scim/v2/Users/scim-br-1", nil); res.Code != http.StatusNoContent {
		t.Fatalf("SCIM bearer DELETE user = %d, want 204", res.Code)
	}
}
