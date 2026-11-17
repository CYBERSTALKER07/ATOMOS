package exportpackroutes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/exportpacks"
)

const testPackSecret = "test-exportpackroutes-secret-2026-0123456789"

type spyPackAudit struct {
	mu      sync.Mutex
	actions []string
}

func (s *spyPackAudit) RecordEvent(_ context.Context, ev exportpacks.AuditRecordInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.actions = append(s.actions, ev.Action)
	return nil
}

func (s *spyPackAudit) RecordEventInTxn(_ context.Context, _ *spanner.ReadWriteTransaction, ev exportpacks.AuditRecordInput) error {
	return s.RecordEvent(context.Background(), ev)
}

func issueTestJWT(t *testing.T, c auth.Claims) string {
	t.Helper()
	tok, err := auth.Issue(c, auth.IssueOptions{Secret: testPackSecret, TTL: time.Hour})
	if err != nil {
		t.Fatalf("auth.Issue: %v", err)
	}
	return tok
}

func doPackReq(t *testing.T, h http.Handler, method, path string, body any, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestExportPackRoutes_RBAC_DownloadFormats_TokenAndExpiry(t *testing.T) {
	t.Parallel()
	const eid = "sup-packs-1"
	now := time.Now().UTC()

	ds := &exportpacks.PackDatasets{
		GlAccounts: []exportpacks.GlAccountRow{{
			EntityID: eid, AccountCode: "1010", Class: "ASSET", Name: "Cash", Currency: "UZS", CreatedAt: now, UpdatedAt: now,
		}},
		JournalEntries: []exportpacks.JournalEntryRow{{
			EntityID: eid, EntryID: "je-r1", PeriodID: "2026-10", SourceType: "ORDER", SourceID: "o-1", Currency: "UZS", PrevHash: "0", EntryHash: "h1", PostedBy: "acct-1", OccurredAt: now, PostedAt: now,
		}},
		JournalPostings: []exportpacks.JournalPostingRow{{
			EntityID: eid, EntryID: "je-r1", Line: 1, AccountCode: "1010", Direction: "DEBIT", AmountMinor: 100000, Currency: "UZS", FunctionalAmountMinor: 100000,
		}},
	}

	auditSpy := &spyPackAudit{}
	svc := exportpacks.NewServiceWithDeps(
		exportpacks.NewMemoryRepository(),
		&exportpacks.MemoryCollector{Datasets: ds},
		exportpacks.NewMemoryPackStorage(),
	)
	svc.SetAuditRecorder(auditSpy)
	svc.SetSigningSecret(testPackSecret)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{Service: svc, JWTSecret: testPackSecret})

	fmTok := issueTestJWT(t, auth.Claims{Subject: "fm-1", Role: auth.RoleAdmin, SupplierID: eid, SupplierRole: auth.SupplierRoleFinanceManager})
	acctTok := issueTestJWT(t, auth.Claims{Subject: "acct-1", Role: auth.RoleAdmin, SupplierID: eid, SupplierRole: auth.SupplierRoleAccountant})
	audExp := now.Add(2 * time.Hour)
	audTok := issueTestJWT(t, auth.Claims{
		Subject: "aud-1", Role: auth.RoleAdmin, SupplierID: eid, SupplierRole: auth.SupplierRoleAuditor,
		SupplierPerms: []string{auth.PermAuditRead}, AuditorExpiresAt: &audExp,
	})
	pastExp := now.Add(-10 * time.Millisecond)
	expiredAudTok := issueTestJWT(t, auth.Claims{
		Subject: "aud-exp", Role: auth.RoleAdmin, SupplierID: eid, SupplierRole: auth.SupplierRoleAuditor,
		SupplierPerms: []string{auth.PermAuditRead}, AuditorExpiresAt: &pastExp,
	})
	agentTok := issueTestJWT(t, auth.Claims{Subject: "agt-1", Role: auth.RoleAdmin, SupplierID: eid, SupplierRole: auth.SupplierRoleAgent, ActorType: auth.ActorTypeAgent})
	dispTok := issueTestJWT(t, auth.Claims{Subject: "dsp-1", Role: auth.RoleAdmin, SupplierID: eid, SupplierRole: auth.SupplierRoleDispatcher})

	// 1. Unauthenticated rejected -> 401.
	if res := doPackReq(t, r, http.MethodGet, "/v1/finance/packs", nil, ""); res.Code != http.StatusUnauthorized {
		t.Fatalf("unauth GET packs = %d, want 401", res.Code)
	}
	if res := doPackReq(t, r, http.MethodPost, "/v1/finance/packs", map[string]any{"pack_type": "AUDIT"}, ""); res.Code != http.StatusUnauthorized {
		t.Fatalf("unauth POST pack = %d, want 401", res.Code)
	}

	// 2. AUDITOR, AGENT, and DISPATCHER blocked from POST -> 403.
	for name, tok := range map[string]string{"AUDITOR": audTok, "AGENT": agentTok, "DISPATCHER": dispTok} {
		res := doPackReq(t, r, http.MethodPost, "/v1/finance/packs", map[string]any{"pack_type": "AUDIT"}, tok)
		if res.Code != http.StatusForbidden {
			t.Fatalf("%s POST /v1/finance/packs = %d, want 403 (%s)", name, res.Code, res.Body.String())
		}
	}

	// 3. Invalid pack_type or date range -> 400.
	if res := doPackReq(t, r, http.MethodPost, "/v1/finance/packs", map[string]any{"pack_type": "BOGUS"}, fmTok); res.Code != http.StatusBadRequest {
		t.Fatalf("invalid pack_type = %d, want 400", res.Code)
	}

	// 4. FINANCE_MANAGER & ACCOUNTANT create packs -> 201 Created.
	createRes := doPackReq(t, r, http.MethodPost, "/v1/finance/packs", map[string]any{
		"pack_type": "TAX",
		"entity_id": eid,
		"period_id": "2026-10",
		"from_date": now.Add(-24 * time.Hour).Format(time.RFC3339),
		"to_date":   now.Add(time.Minute).Format(time.RFC3339),
		"format":    "zip",
		"ttl_hours": 12,
	}, fmTok)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("FM POST /v1/finance/packs = %d body=%s", createRes.Code, createRes.Body.String())
	}
	var created exportpacks.ExportPack
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil || created.PackID == "" || created.DownloadToken == "" {
		t.Fatalf("decode created pack: %v body=%s", err, createRes.Body.String())
	}

	acctCreate := doPackReq(t, r, http.MethodPost, "/v1/finance/packs", map[string]any{"pack_type": "BANK", "format": "xlsx"}, acctTok)
	if acctCreate.Code != http.StatusCreated {
		t.Fatalf("ACCOUNTANT POST /v1/finance/packs = %d body=%s", acctCreate.Code, acctCreate.Body.String())
	}

	// 5. AUDITOR can list and get pack metadata (redacted DownloadToken).
	if res := doPackReq(t, r, http.MethodGet, "/v1/finance/packs", nil, audTok); res.Code != http.StatusOK {
		t.Fatalf("AUDITOR GET /v1/finance/packs = %d body=%s", res.Code, res.Body.String())
	}
	getRes := doPackReq(t, r, http.MethodGet, "/v1/finance/packs/"+created.PackID, nil, audTok)
	if getRes.Code != http.StatusOK {
		t.Fatalf("AUDITOR GET pack = %d body=%s", getRes.Code, getRes.Body.String())
	}
	var fetched exportpacks.ExportPack
	_ = json.Unmarshal(getRes.Body.Bytes(), &fetched)
	if fetched.DownloadToken != "" {
		t.Fatalf("expected DownloadToken redacted for AUDITOR view, got non-empty")
	}

	// Cross-tenant access -> 403.
	if res := doPackReq(t, r, http.MethodGet, "/v1/finance/packs/"+created.PackID+"?entity=sup-other", nil, audTok); res.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant GET pack = %d, want 403", res.Code)
	}

	// 6. AUDITOR downloads across formats via Bearer header & via ?token=<jwt> & opaque ?token=<DownloadToken>.
	for _, fmtName := range []string{"zip", "xlsx", "csv", "xml", "json"} {
		dl := doPackReq(t, r, http.MethodGet, "/v1/finance/packs/"+created.PackID+"/download?format="+fmtName, nil, audTok)
		if dl.Code != http.StatusOK {
			t.Fatalf("AUDITOR download %s = %d body=%s", fmtName, dl.Code, dl.Body.String())
		}
		if dl.Header().Get("X-Atomos-Watermark") != created.WatermarkText {
			t.Fatalf("missing X-Atomos-Watermark on %s", fmtName)
		}
		if dl.Header().Get("X-Atomos-Pack-Sha256") == "" {
			t.Fatalf("missing X-Atomos-Pack-Sha256 on %s", fmtName)
		}
	}

	// Session JWT passed in ?token= query param is rejected (401).
	if dl := doPackReq(t, r, http.MethodGet, "/v1/finance/packs/"+created.PackID+"/download?format=csv&token="+audTok, nil, ""); dl.Code != http.StatusUnauthorized {
		t.Fatalf("download ?token=<session_jwt> = %d, want 401 body=%s", dl.Code, dl.Body.String())
	}

	// Query ?token=<DownloadToken> succeeds without Authorization header.
	if dl := doPackReq(t, r, http.MethodGet, "/v1/finance/packs/"+created.PackID+"/download?format=json&token="+created.DownloadToken, nil, ""); dl.Code != http.StatusOK {
		t.Fatalf("download ?token=<download_token> = %d body=%s", dl.Code, dl.Body.String())
	}

	// Wrong opaque ?token= returns 403 Forbidden.
	if dl := doPackReq(t, r, http.MethodGet, "/v1/finance/packs/"+created.PackID+"/download?token=wrong-token-123", nil, ""); dl.Code != http.StatusForbidden {
		t.Fatalf("wrong ?token= download = %d, want 403 (%s)", dl.Code, dl.Body.String())
	}

	// Expired AUDITOR JWT returns 401 auditor_access_expired via header or query.
	expHeader := doPackReq(t, r, http.MethodGet, "/v1/finance/packs", nil, expiredAudTok)
	if expHeader.Code != http.StatusUnauthorized || !bytes.Contains(expHeader.Body.Bytes(), []byte("auditor_access_expired")) {
		t.Fatalf("expired auditor GET /packs = %d body=%s, want 401 auditor_access_expired", expHeader.Code, expHeader.Body.String())
	}
	expQuery := doPackReq(t, r, http.MethodGet, "/v1/finance/packs/"+created.PackID+"/download?token="+expiredAudTok, nil, "")
	if expQuery.Code != http.StatusUnauthorized || !bytes.Contains(expQuery.Body.Bytes(), []byte("auditor_access_expired")) {
		t.Fatalf("expired auditor ?token= download = %d body=%s, want 401 auditor_access_expired", expQuery.Code, expQuery.Body.String())
	}

	// Verify AuditEvents captured creations and downloads.
	auditSpy.mu.Lock()
	dlCount := 0
	for _, a := range auditSpy.actions {
		if a == "EXPORT_PACK_DOWNLOADED" {
			dlCount++
		}
	}
	auditSpy.mu.Unlock()
	if dlCount != 6 {
		t.Fatalf("EXPORT_PACK_DOWNLOADED count = %d, want 6", dlCount)
	}
}

func TestExportPackRoutes_ExpiredPackReturns410(t *testing.T) {
	t.Parallel()
	const eid = "sup-exp-pack"
	baseNow := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	repo := exportpacks.NewMemoryRepository()
	storage := exportpacks.NewMemoryPackStorage()
	pack := &exportpacks.ExportPack{
		TenantID:      eid,
		PackID:        "pack-already-expired",
		PackType:      exportpacks.PackTypeAudit,
		EntityID:      eid,
		FromDate:      baseNow.Add(-24 * time.Hour),
		ToDate:        baseNow,
		Format:        exportpacks.FormatJSON,
		Status:        exportpacks.StatusReady,
		DownloadToken: "dl-tok-exp-1",
		WatermarkText: "Prepared for owner | Expired",
		Artifacts: []exportpacks.ArtifactMeta{{
			Format:      exportpacks.FormatJSON,
			Filename:    "pack-already-expired.json",
			ContentType: exportpacks.ContentTypeForFormat(exportpacks.FormatJSON),
			StorageURI:  "mem://expired.json",
		}},
		CreatedBy: "owner-1",
		ExpiresAt: time.Now().UTC().Add(-time.Minute),
		CreatedAt: baseNow,
	}
	_, _ = storage.PutObject(context.Background(), "expired.json", "application/json", []byte(`{}`))
	if err := repo.CreatePack(context.Background(), pack, nil); err != nil {
		t.Fatalf("seed expired pack: %v", err)
	}

	svc := exportpacks.NewServiceWithDeps(repo, &exportpacks.MemoryCollector{}, storage)
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{Service: svc, JWTSecret: testPackSecret})

	audTok := issueTestJWT(t, auth.Claims{
		Subject: "aud-1", Role: auth.RoleAdmin, SupplierID: eid, SupplierRole: auth.SupplierRoleAuditor,
		SupplierPerms: []string{auth.PermAuditRead},
	})

	res := doPackReq(t, r, http.MethodGet, "/v1/finance/packs/"+pack.PackID+"/download", nil, audTok)
	if res.Code != http.StatusGone {
		t.Fatalf("expired pack download = %d, want 410 Gone (%s)", res.Code, res.Body.String())
	}
}
