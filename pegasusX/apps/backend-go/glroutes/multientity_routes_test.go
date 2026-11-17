package glroutes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/glroutes"
)

func TestGLRoutes_MultiEntityEndpoints_And_AuditorRBAC_LiveSpanner(t *testing.T) {
	client := openRoutesSpannerClient(t)
	ctx := context.Background()

	tenantID := fmt.Sprintf("sup_glr_me_%d", time.Now().UnixNano())
	cleanupEntity(t, client, tenantID, nil)
	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = client.Apply(cctx, []*spanner.Mutation{
			spanner.Delete("LegalEntities", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
		})
	})

	entUZ := tenantID + "_uz"
	entKZ := tenantID + "_kz"
	unownedEnt := "sup_foreign_unowned_entity"
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("LegalEntities", map[string]any{
			"TenantId":     tenantID,
			"EntityId":     entUZ,
			"Name":         "UZ Operating Co",
			"CountryCode":  "UZ",
			"BaseCurrency": "UZS",
			"IsParent":     true,
			"Status":       "ACTIVE",
			"CreatedAt":    spanner.CommitTimestamp,
			"UpdatedAt":    spanner.CommitTimestamp,
		}),
		spanner.InsertMap("LegalEntities", map[string]any{
			"TenantId":     tenantID,
			"EntityId":     entKZ,
			"Name":         "KZ Subsidiary LLP",
			"CountryCode":  "KZ",
			"BaseCurrency": "KZT",
			"IsParent":     false,
			"Status":       "ACTIVE",
			"CreatedAt":    spanner.CommitTimestamp,
			"UpdatedAt":    spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed LegalEntities: %v", err)
	}

	svc := gl.NewService(client)
	if err := svc.SeedDefaults(ctx, tenantID, "UZ", "UZS"); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}

	occ := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	if _, err := svc.PostJournal(ctx, gl.PostInput{
		TenantID:       tenantID,
		EntityID:       entUZ,
		PeriodID:       "2026-11",
		SourceType:     "MANUAL",
		SourceID:       "glr-uz-1",
		Memo:           "UZ legal entity revenue",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: tenantID + "-glr-uz-1",
		OccurredAt:     occ,
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountAccountsReceivable, Direction: gl.DirectionDebit, AmountMinor: 420000},
			{AccountCode: gl.AccountSalesRevenue, Direction: gl.DirectionCredit, AmountMinor: 420000},
		},
	}); err != nil {
		t.Fatalf("PostJournal(entUZ): %v", err)
	}
	if _, err := svc.PostJournal(ctx, gl.PostInput{
		TenantID:       tenantID,
		EntityID:       entKZ,
		PeriodID:       "2026-11",
		SourceType:     "INTERCOMPANY",
		SourceID:       "glr-kz-1",
		Memo:           "KZ legal entity IC posting",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: tenantID + "-glr-kz-1",
		OccurredAt:     occ.Add(time.Minute),
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountBankOperating, Direction: gl.DirectionDebit, AmountMinor: 180000},
			{AccountCode: gl.AccountIntercompanyDueTo, Direction: gl.DirectionCredit, AmountMinor: 180000},
		},
	}); err != nil {
		t.Fatalf("PostJournal(entKZ): %v", err)
	}

	const jwtSecret = "test-glr-me-secret-0123456789abcdef"
	r := chi.NewRouter()
	glroutes.RegisterRoutes(r, glroutes.Deps{
		Service:   svc,
		JWTSecret: jwtSecret,
	})

	adminCookie := issueTestCookie(t, jwtSecret, auth.Claims{
		Subject:    "admin-me-1",
		Role:       auth.RoleAdmin,
		SupplierID: tenantID,
	})
	future := time.Now().UTC().Add(30 * time.Minute)
	auditorCookie := issueTestCookie(t, jwtSecret, auth.Claims{
		Subject:          "auditor-me-1",
		Role:             auth.RoleAdmin,
		SupplierID:       tenantID,
		SupplierRole:     auth.SupplierRoleAuditor,
		AuditorExpiresAt: &future,
	})
	past := time.Now().UTC().Add(-10 * time.Minute)
	expiredAuditorTok, err := auth.Issue(auth.Claims{
		Subject:          "auditor-expired-1",
		Role:             auth.RoleAdmin,
		SupplierID:       tenantID,
		SupplierRole:     auth.SupplierRoleAuditor,
		AuditorExpiresAt: &past,
	}, auth.IssueOptions{Secret: jwtSecret})
	if err != nil {
		t.Fatalf("issue expired auditor jwt: %v", err)
	}
	agentCookie := issueTestCookie(t, jwtSecret, auth.Claims{
		Subject:       "agent-me-1",
		Role:          auth.RoleAdmin,
		SupplierID:    tenantID,
		SupplierRole:  auth.SupplierRoleAgent,
		ActorType:     "AGENT",
		SupplierPerms: []string{auth.PermJournalPost, auth.PermAuditRead},
	})

	// 1. GET /v1/finance/trial-balance?entity=<entUZ>&period=2026-11 -> 200 scoped to entUZ only.
	reqTBUZ := httptest.NewRequest(http.MethodGet, "/v1/finance/trial-balance?entity="+entUZ+"&period=2026-11", nil)
	reqTBUZ.AddCookie(adminCookie)
	recTBUZ := httptest.NewRecorder()
	r.ServeHTTP(recTBUZ, reqTBUZ)
	if recTBUZ.Code != http.StatusOK {
		t.Fatalf("GET /v1/finance/trial-balance?entity=entUZ status = %d, want 200 (%s)", recTBUZ.Code, recTBUZ.Body.String())
	}
	var tbUZ gl.TrialBalanceReport
	if err := json.Unmarshal(recTBUZ.Body.Bytes(), &tbUZ); err != nil {
		t.Fatal(err)
	}
	if !tbUZ.Balanced || tbUZ.TotalDebitsMinor != 420000 || tbUZ.LegalEntityID != entUZ {
		t.Fatalf("unexpected tbUZ: %+v", tbUZ)
	}

	// 2. GET /v1/finance/trial-balance?entity=<unowned> -> 403.
	reqTBUnowned := httptest.NewRequest(http.MethodGet, "/v1/finance/trial-balance?entity="+unownedEnt+"&period=2026-11", nil)
	reqTBUnowned.AddCookie(adminCookie)
	recTBUnowned := httptest.NewRecorder()
	r.ServeHTTP(recTBUnowned, reqTBUnowned)
	if recTBUnowned.Code != http.StatusForbidden {
		t.Fatalf("GET /v1/finance/trial-balance unowned entity status = %d, want 403 (%s)", recTBUnowned.Code, recTBUnowned.Body.String())
	}

	// 3. GET /v1/finance/gl?entity=<entKZ>&period=2026-11 -> 200 with KZ journal only.
	reqGLKZ := httptest.NewRequest(http.MethodGet, "/v1/finance/gl?entity="+entKZ+"&period=2026-11", nil)
	reqGLKZ.AddCookie(adminCookie)
	recGLKZ := httptest.NewRecorder()
	r.ServeHTTP(recGLKZ, reqGLKZ)
	if recGLKZ.Code != http.StatusOK {
		t.Fatalf("GET /v1/finance/gl?entity=entKZ status = %d, want 200 (%s)", recGLKZ.Code, recGLKZ.Body.String())
	}
	var glKZ struct {
		Count         int                     `json:"count"`
		LegalEntityID string                  `json:"legal_entity_id"`
		Items         []gl.JournalEntryRecord `json:"items"`
	}
	if err := json.Unmarshal(recGLKZ.Body.Bytes(), &glKZ); err != nil {
		t.Fatal(err)
	}
	if glKZ.Count != 1 || glKZ.LegalEntityID != entKZ || glKZ.Items[0].LegalEntityID != entKZ {
		t.Fatalf("unexpected GET /v1/finance/gl response: %+v", glKZ)
	}

	// 4. Active AUDITOR allowed on both read routes (200), blocked on POST /v1/finance/journal/manual (403).
	for _, path := range []string{
		"/v1/finance/trial-balance?period=2026-11",
		"/v1/finance/gl?entity=" + entUZ + "&period=2026-11",
	} {
		reqAud := httptest.NewRequest(http.MethodGet, path, nil)
		reqAud.AddCookie(auditorCookie)
		recAud := httptest.NewRecorder()
		r.ServeHTTP(recAud, reqAud)
		if recAud.Code != http.StatusOK {
			t.Fatalf("active AUDITOR GET %s status = %d, want 200 (%s)", path, recAud.Code, recAud.Body.String())
		}
	}

	manualBody := fmt.Sprintf(`{"entity_id":%q,"source_id":"aud-block","memo":"blocked","currency":"UZS","postings":[{"account_code":"CASH:OFFICE","direction":"DEBIT","amount_minor":1000},{"account_code":"REV:CASH:OVERAGE","direction":"CREDIT","amount_minor":1000}]}`, tenantID)
	for name, ck := range map[string]*http.Cookie{"AUDITOR": auditorCookie, "AGENT": agentCookie} {
		reqMut := httptest.NewRequest(http.MethodPost, "/v1/finance/journal/manual", bytes.NewBufferString(manualBody))
		reqMut.AddCookie(ck)
		reqMut.Header.Set("Idempotency-Key", "mut-block-"+name)
		recMut := httptest.NewRecorder()
		r.ServeHTTP(recMut, reqMut)
		if recMut.Code != http.StatusForbidden {
			t.Fatalf("%s POST /v1/finance/journal/manual status = %d, want 403 (%s)", name, recMut.Code, recMut.Body.String())
		}
	}

	// 5. Expired AUDITOR receives 401 auditor_access_expired.
	reqExp := httptest.NewRequest(http.MethodGet, "/v1/finance/trial-balance?period=2026-11", nil)
	reqExp.Header.Set("Authorization", "Bearer "+expiredAuditorTok)
	recExp := httptest.NewRecorder()
	r.ServeHTTP(recExp, reqExp)
	if recExp.Code != http.StatusUnauthorized || !strings.Contains(recExp.Body.String(), "auditor_access_expired") {
		t.Fatalf("expired AUDITOR status=%d body=%s, want 401 auditor_access_expired", recExp.Code, recExp.Body.String())
	}
}
