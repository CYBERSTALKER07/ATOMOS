package partner

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestPartnerRoutes_AuditorReadOnly403AndExpiry401(t *testing.T) {
	t.Setenv("PARTNER_EXPORTS_ENABLED", "true")
	t.Setenv("PARTNER_WEBHOOKS_ENABLED", "true")
	t.Setenv("PARTNER_SFTP_ENABLED", "true")
	t.Setenv("PARTNER_AS2_ENABLED", "true")
	t.Setenv("PARTNER_COA_ENABLED", "true")

	keys := NewMemoryKeyRepository()
	webhooks := NewMemoryWebhookRepository()
	exports := NewMemoryExportRepository()
	svc := NewService(keys, webhooks, nil, nil, nil)
	svc.SetExportRepos(exports, NewMemorySftpConfigRepository())
	h := &Handlers{Svc: svc}

	r := chi.NewRouter()
	const jwtSecret = "partner-auditor-rbac-test-secret-32b!"
	r.Use(auth.SessionAuth(jwtSecret))
	RegisterAdminKeyRoutes(r, h)

	now := time.Now().UTC()
	activeExp := now.Add(2 * time.Hour)
	expiredExp := now.Add(-10 * time.Millisecond)

	const supID = "sup-rbac-partner-1"
	ownerTok, err := auth.Issue(auth.Claims{
		Subject: "owner-1", Role: auth.RoleAdmin, SupplierID: supID, SupplierRole: auth.SupplierRoleOwner,
	}, auth.IssueOptions{Secret: jwtSecret, TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	auditorTok, err := auth.Issue(auth.Claims{
		Subject: "aud-1", Role: auth.RoleAdmin, SupplierID: supID, SupplierRole: auth.SupplierRoleAuditor,
		SupplierPerms: []string{auth.PermAuditRead}, AuditorExpiresAt: &activeExp,
	}, auth.IssueOptions{Secret: jwtSecret, TTL: 2 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	expiredAuditorTok, err := auth.Issue(auth.Claims{
		Subject: "aud-exp", Role: auth.RoleAdmin, SupplierID: supID, SupplierRole: auth.SupplierRoleAuditor,
		SupplierPerms: []string{auth.PermAuditRead}, AuditorExpiresAt: &expiredExp,
	}, auth.IssueOptions{Secret: jwtSecret, TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	agentTok, err := auth.Issue(auth.Claims{
		Subject: "agt-1", Role: auth.RoleAdmin, SupplierID: supID, SupplierRole: auth.SupplierRoleAgent,
		ActorType: auth.ActorTypeAgent,
	}, auth.IssueOptions{Secret: jwtSecret, TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}

	call := func(method, path, body, tok string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// 1. Valid non-auditor RoleAdmin succeeds on key issue and export creation.
	if res := call(http.MethodPost, "/v1/supplier/partner-keys", `{"scopes":["*"]}`, ownerTok); res.Code != http.StatusCreated {
		t.Fatalf("owner POST /v1/supplier/partner-keys = %d body=%s", res.Code, res.Body.String())
	}
	if res := call(http.MethodPost, "/v1/supplier/partner-exports", `{"resource":"orders","format":"xlsx"}`, ownerTok); res.Code != http.StatusAccepted {
		t.Fatalf("owner POST /v1/supplier/partner-exports = %d body=%s", res.Code, res.Body.String())
	}

	// 2. Active AUDITOR JWT can read GET routes (not 401/403).
	for _, path := range []string{
		"/v1/supplier/partner-keys",
		"/v1/supplier/partner-webhooks",
		"/v1/supplier/partner-exports",
		"/v1/supplier/partner-edi/profile",
	} {
		res := call(http.MethodGet, path, "", auditorTok)
		if res.Code == http.StatusUnauthorized || res.Code == http.StatusForbidden {
			t.Fatalf("active AUDITOR GET %s got %d body=%s", path, res.Code, res.Body.String())
		}
	}

	// 3. Active AUDITOR JWT MUST get 403 Forbidden across every mutating /v1/supplier/partner-* & /v1/admin/partner-* route.
	mutatingCases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/v1/supplier/partner-keys", `{"scopes":["*"]}`},
		{http.MethodPost, "/v1/supplier/partner-keys/k-1/revoke", `{}`},
		{http.MethodPost, "/v1/admin/partner-keys", `{"tenant_type":"SUPPLIER","tenant_id":"` + supID + `","scopes":["*"]}`},
		{http.MethodPost, "/v1/admin/partner-keys/k-1/revoke", `{}`},
		{http.MethodPost, "/v1/supplier/partner-webhooks", `{"url":"https://example.com/hook","events":["order.created"]}`},
		{http.MethodDelete, "/v1/supplier/partner-webhooks/sub-1", ``},
		{http.MethodPost, "/v1/supplier/partner-webhooks/sub-1/ping", `{}`},
		{http.MethodPost, "/v1/supplier/partner-webhooks/sub-1/rotate-secret", `{}`},
		{http.MethodPost, "/v1/supplier/partner-webhooks/dead-letter/att-1/replay", `{}`},
		{http.MethodPut, "/v1/supplier/partner-sftp", `{"host":"sftp.example.com","port":22,"username":"u"}`},
		{http.MethodPost, "/v1/supplier/partner-exports", `{"resource":"orders","format":"csv"}`},
		{http.MethodPost, "/v1/supplier/partner-edi/documents/doc-1/replay", `{}`},
		{http.MethodPut, "/v1/supplier/partner-edi/profile", `{"pack_name":"EDIFACT_LITE_V1"}`},
		{http.MethodPut, "/v1/supplier/partner-coa", `{"mappings":{}}`},
		{http.MethodPut, "/v1/supplier/partner-as2", `{"as2_id":"AS2-TEST","partner_url":"https://example.com/as2"}`},
		{http.MethodPost, "/v1/supplier/partner-onec/ack", `{"posting_id":"post-1","onec_ref":"1c-1"}`},
		{http.MethodPost, "/v1/supplier/partner-onec/reject", `{"posting_id":"post-1","reason":"bad"}`},
		{http.MethodPut, "/v1/admin/partner-sftp", `{}`},
		{http.MethodPut, "/v1/admin/partner-coa", `{}`},
		{http.MethodPut, "/v1/admin/partner-as2", `{}`},
	}
	for _, tc := range mutatingCases {
		res := call(tc.method, tc.path, tc.body, auditorTok)
		if res.Code != http.StatusForbidden {
			t.Fatalf("AUDITOR %s %s = %d, want 403 (%s)", tc.method, tc.path, res.Code, res.Body.String())
		}
	}

	// 4. AGENT JWT blocked on partner mutations -> 403.
	for _, path := range []string{"/v1/supplier/partner-keys", "/v1/supplier/partner-exports", "/v1/supplier/partner-webhooks"} {
		res := call(http.MethodPost, path, `{}`, agentTok)
		if res.Code != http.StatusForbidden {
			t.Fatalf("AGENT POST %s = %d, want 403 (%s)", path, res.Code, res.Body.String())
		}
	}

	// 5. Expired AUDITOR JWT gets 401 auditor_access_expired on BOTH GET reads and POST/PUT writes.
	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/supplier/partner-keys"},
		{http.MethodGet, "/v1/supplier/partner-exports"},
		{http.MethodGet, "/v1/supplier/partner-edi/profile"},
		{http.MethodPost, "/v1/supplier/partner-exports"},
		{http.MethodPost, "/v1/supplier/partner-keys"},
	} {
		res := call(tc.method, tc.path, `{}`, expiredAuditorTok)
		if res.Code != http.StatusUnauthorized || !bytes.Contains(res.Body.Bytes(), []byte("auditor_access_expired")) {
			t.Fatalf("expired AUDITOR %s %s = %d body=%s, want 401 auditor_access_expired", tc.method, tc.path, res.Code, res.Body.String())
		}
	}

	// 6. M2M Partner API key caller keeps full access on /partner/v1/exports while direct handler calls still enforce AUDITOR -> 403.
	issuedKey, err := svc.IssueKey(t.Context(), TenantSupplier, supID, "owner-1", []string{"*"})
	if err != nil {
		t.Fatalf("IssueKey for M2M check: %v", err)
	}
	pr := chi.NewRouter()
	RegisterPartnerRoutes(pr, keys, h)
	m2mReq := httptest.NewRequest(http.MethodPost, "/partner/v1/exports", strings.NewReader(`{"resource":"orders","format":"csv"}`))
	m2mReq.Header.Set("Authorization", "Bearer "+issuedKey.Secret)
	m2mReq.Header.Set("Content-Type", "application/json")
	m2mRec := httptest.NewRecorder()
	pr.ServeHTTP(m2mRec, m2mReq)
	if m2mRec.Code != http.StatusAccepted {
		t.Fatalf("M2M API-key POST /partner/v1/exports = %d body=%s, want 202", m2mRec.Code, m2mRec.Body.String())
	}

	// Direct handler invocation without chi middleware rejects active AUDITOR (403) and expired AUDITOR (401).
	audClaims := auth.Claims{
		Subject: "aud-direct", Role: auth.RoleAdmin, SupplierID: supID, SupplierRole: auth.SupplierRoleAuditor,
		SupplierPerms: []string{auth.PermAuditRead}, AuditorExpiresAt: &activeExp,
	}
	directCases := []struct {
		name string
		fn   http.HandlerFunc
		meth string
		body string
	}{
		{"HandleIssueKey", h.HandleIssueKey, http.MethodPost, `{"scopes":["*"]}`},
		{"HandleRevokeKey", h.HandleRevokeKey, http.MethodPost, `{}`},
		{"HandleCreateExport", h.HandleCreateExport, http.MethodPost, `{"resource":"orders","format":"csv"}`},
		{"HandlePutEdiProfile", h.HandlePutEdiProfile, http.MethodPut, `{}`},
		{"HandleOneCAck", h.HandleOneCAck, http.MethodPost, `{"posting_id":"p1"}`},
		{"HandleOneCReject", h.HandleOneCReject, http.MethodPost, `{"posting_id":"p1"}`},
	}
	for _, dc := range directCases {
		req := httptest.NewRequest(dc.meth, "/direct", strings.NewReader(dc.body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithClaims(req.Context(), audClaims))
		rec := httptest.NewRecorder()
		dc.fn(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("direct %s as AUDITOR = %d, want 403 (%s)", dc.name, rec.Code, rec.Body.String())
		}

		expClaims := audClaims
		expClaims.AuditorExpiresAt = &expiredExp
		reqExp := httptest.NewRequest(dc.meth, "/direct", strings.NewReader(dc.body))
		reqExp = reqExp.WithContext(auth.WithClaims(reqExp.Context(), expClaims))
		recExp := httptest.NewRecorder()
		dc.fn(recExp, reqExp)
		var ep map[string]any
		_ = json.Unmarshal(recExp.Body.Bytes(), &ep)
		if recExp.Code != http.StatusUnauthorized || ep["error"] != "auditor_access_expired" {
			t.Fatalf("direct %s as expired AUDITOR = %d body=%s, want 401 auditor_access_expired", dc.name, recExp.Code, recExp.Body.String())
		}
	}
}
