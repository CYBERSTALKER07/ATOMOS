package multientityroutes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/multientity"
	"github.com/pegasusx/pegasusx/apps/backend-go/multientityroutes"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type routeAuditAdapter struct {
	svc *audit.Service
}

func (a routeAuditAdapter) RecordEvent(ctx context.Context, ev multientity.AuditRecordInput) error {
	if a.svc == nil {
		return nil
	}
	_, err := a.svc.Record(ctx, audit.EventInput{
		TenantID:   ev.TenantID,
		ActorID:    ev.ActorID,
		ActorType:  audit.NormalizeActorType(ev.ActorType),
		Role:       ev.Role,
		Action:     ev.Action,
		ObjectType: ev.ObjectType,
		ObjectID:   ev.ObjectID,
		BeforeJSON: ev.BeforeJSON,
		AfterJSON:  ev.AfterJSON,
		ReasonCode: ev.ReasonCode,
		ApprovalID: ev.ApprovalID,
		TraceID:    ev.TraceID,
	})
	return err
}

func (a routeAuditAdapter) RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev multientity.AuditRecordInput) error {
	if a.svc == nil || txn == nil {
		return nil
	}
	_, err := a.svc.RecordInTxn(ctx, txn, audit.EventInput{
		TenantID:   ev.TenantID,
		ActorID:    ev.ActorID,
		ActorType:  audit.NormalizeActorType(ev.ActorType),
		Role:       ev.Role,
		Action:     ev.Action,
		ObjectType: ev.ObjectType,
		ObjectID:   ev.ObjectID,
		BeforeJSON: ev.BeforeJSON,
		AfterJSON:  ev.AfterJSON,
		ReasonCode: ev.ReasonCode,
		ApprovalID: ev.ApprovalID,
		TraceID:    ev.TraceID,
	})
	return err
}

func openSpannerRoutesClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping Spanner multientityroutes test")
	}

	project := envOrDefault("SPANNER_PROJECT", "pegasusx-local")
	instance := envOrDefault("SPANNER_INSTANCE", "pegasusx-instance")
	database := envOrDefault("SPANNER_DATABASE", "pegasusx-db")
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := spanner.NewClient(
		ctx,
		dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner emulator connect: %v", err)
		}
		t.Skipf("spanner unavailable: %v", err)
	}
	_, probeErr := client.Single().ReadRow(ctx, "LegalEntities", spanner.Key{"__mer_probe__", "__mer_probe__"}, []string{"EntityId"})
	if probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("LegalEntities unavailable: %v", probeErr)
		}
		t.Skipf("LegalEntities unavailable: %v", probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

func envOrDefault(k, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return fallback
}

func cleanupRoutesTenant(t *testing.T, client *spanner.Client, tenantID string, fxRateIDs []string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		muts := []*spanner.Mutation{
			spanner.Delete("JournalPostings", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("JournalEntries", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("GlBalances", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("GlAccounts", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("IntercompanyTransfers", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("FxRevaluationRuns", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("LegalEntities", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("AuditEvents", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
		}
		for _, rid := range fxRateIDs {
			muts = append(muts, spanner.Delete("FxRates", spanner.Key{rid}))
		}
		ro := client.ReadOnlyTransaction()
		iter := ro.Query(ctx, spanner.Statement{
			SQL:    `SELECT EventId FROM OutboxEvents WHERE SupplierId = @sid`,
			Params: map[string]any{"sid": tenantID},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done || err != nil {
				break
			}
			var eid string
			if row.Column(0, &eid) == nil {
				muts = append(muts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
			}
		}
		iter.Stop()
		ro.Close()
		_, _ = client.Apply(ctx, muts)
	})
}

func issueCookie(t *testing.T, secret string, c auth.Claims) *http.Cookie {
	t.Helper()
	tok, err := auth.Issue(c, auth.IssueOptions{Secret: secret, TTL: time.Hour})
	if err != nil {
		t.Fatalf("auth.Issue: %v", err)
	}
	return &http.Cookie{Name: auth.CookieName, Value: tok}
}

func TestMultientityRoutes_RBAC_Errors_And_Contract_Memory(t *testing.T) {
	ctx := context.Background()
	glSvc := gl.NewServiceWithRepository(gl.NewMemoryRepository())
	fxRepo := fxrates.NewMemoryRepository()
	meRepo := multientity.NewMemoryRepository(glSvc)
	svc := multientity.NewServiceWithDeps(meRepo, glSvc, fxrates.NewService(fxRepo), fxRepo)

	const jwtSecret = "test-mer-secret-0123456789abcdef"
	r := chi.NewRouter()
	multientityroutes.RegisterRoutes(r, multientityroutes.Deps{
		Service:   svc,
		JWTSecret: jwtSecret,
	})

	tenantID := "ten_mer_rbac"
	adminCookie := issueCookie(t, jwtSecret, auth.Claims{
		Subject:    "admin-1",
		Role:       auth.RoleAdmin,
		SupplierID: tenantID,
	})
	retailerCookie := issueCookie(t, jwtSecret, auth.Claims{
		Subject:    "ret-1",
		Role:       auth.RoleRetailer,
		SupplierID: tenantID,
	})
	future := time.Now().UTC().Add(30 * time.Minute)
	auditorCookie := issueCookie(t, jwtSecret, auth.Claims{
		Subject:          "aud-1",
		Role:             auth.RoleAdmin,
		SupplierID:       tenantID,
		SupplierRole:     auth.SupplierRoleAuditor,
		AuditorExpiresAt: &future,
	})
	past := time.Now().UTC().Add(-15 * time.Minute)
	expiredTok, err := auth.Issue(auth.Claims{
		Subject:          "aud-exp",
		Role:             auth.RoleAdmin,
		SupplierID:       tenantID,
		SupplierRole:     auth.SupplierRoleAuditor,
		AuditorExpiresAt: &past,
	}, auth.IssueOptions{Secret: jwtSecret})
	if err != nil {
		t.Fatal(err)
	}
	agentCookie := issueCookie(t, jwtSecret, auth.Claims{
		Subject:       "agent-1",
		Role:          auth.RoleAdmin,
		SupplierID:    tenantID,
		SupplierRole:  auth.SupplierRoleAgent,
		ActorType:     "AGENT",
		SupplierPerms: []string{auth.PermAuditRead, auth.PermConfigManage, auth.PermJournalPost},
	})

	// 1. Unauthenticated -> 401.
	reqUnauth := httptest.NewRequest(http.MethodGet, "/v1/org/entities", nil)
	recUnauth := httptest.NewRecorder()
	r.ServeHTTP(recUnauth, reqUnauth)
	if recUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", recUnauth.Code)
	}

	// 2. Non-admin role (RETAILER) -> 403.
	reqRet := httptest.NewRequest(http.MethodGet, "/v1/org/entities", nil)
	reqRet.AddCookie(retailerCookie)
	recRet := httptest.NewRecorder()
	r.ServeHTTP(recRet, reqRet)
	if recRet.Code != http.StatusForbidden {
		t.Fatalf("retailer status = %d, want 403", recRet.Code)
	}

	// 3. Cross-tenant query mismatch -> 403.
	reqCross := httptest.NewRequest(http.MethodGet, "/v1/org/entities?tenant_id=other_tenant", nil)
	reqCross.AddCookie(adminCookie)
	recCross := httptest.NewRecorder()
	r.ServeHTTP(recCross, reqCross)
	if recCross.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant status = %d, want 403", recCross.Code)
	}

	// 4. Expired AUDITOR -> 401 auditor_access_expired.
	reqExp := httptest.NewRequest(http.MethodGet, "/v1/finance/consolidation?period=2026-11", nil)
	reqExp.Header.Set("Authorization", "Bearer "+expiredTok)
	recExp := httptest.NewRecorder()
	r.ServeHTTP(recExp, reqExp)
	if recExp.Code != http.StatusUnauthorized || !strings.Contains(recExp.Body.String(), "auditor_access_expired") {
		t.Fatalf("expired AUDITOR got %d %s, want 401 auditor_access_expired", recExp.Code, recExp.Body.String())
	}

	// 5. AUDITOR allowed on ALL 4 GET routes (200), blocked on ALL 4 mutations (403); AGENT blocked on ALL 4 mutations (403).
	for _, path := range []string{
		"/v1/org/entities",
		"/v1/finance/intercompany-transfers",
		"/v1/finance/fx-revaluation",
		"/v1/finance/consolidation?period=2026-11",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(auditorCookie)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("active AUDITOR GET %s status = %d, want 200 (%s)", path, rec.Code, rec.Body.String())
		}
	}

	type mutSpec struct{ method, path, body string }
	mutations := []mutSpec{
		{http.MethodPost, "/v1/org/entities", `{"name":"Blocked"}`},
		{http.MethodPatch, "/v1/org/entities/e1", `{"name":"Blocked"}`},
		{http.MethodPost, "/v1/finance/intercompany-transfers", `{"from_entity_id":"e1","to_entity_id":"e2","amount_minor":100}`},
		{http.MethodPost, "/v1/finance/fx-revaluation", `{"period_id":"2026-11"}`},
	}
	for roleName, ck := range map[string]*http.Cookie{"AUDITOR": auditorCookie, "AGENT": agentCookie} {
		for _, m := range mutations {
			req := httptest.NewRequest(m.method, m.path, bytes.NewBufferString(m.body))
			req.AddCookie(ck)
			req.Header.Set("Idempotency-Key", "blocked-"+roleName)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("%s %s %s status = %d, want 403 (%s)", roleName, m.method, m.path, rec.Code, rec.Body.String())
			}
		}
	}

	// 6. HTTP 422 same_entity_transfer & HTTP 422 fx_rate_missing + 400 idempotency_key_required.
	reqNoIdem := httptest.NewRequest(http.MethodPost, "/v1/finance/fx-revaluation", bytes.NewBufferString(`{"period_id":"2026-11"}`))
	reqNoIdem.AddCookie(adminCookie)
	recNoIdem := httptest.NewRecorder()
	r.ServeHTTP(recNoIdem, reqNoIdem)
	if recNoIdem.Code != http.StatusBadRequest {
		t.Fatalf("missing Idempotency-Key status = %d, want 400", recNoIdem.Code)
	}

	reqSame := httptest.NewRequest(http.MethodPost, "/v1/finance/intercompany-transfers", bytes.NewBufferString(`{"from_entity_id":"e1","to_entity_id":"e1","amount_minor":1000}`))
	reqSame.AddCookie(adminCookie)
	reqSame.Header.Set("Idempotency-Key", "same-ent")
	recSame := httptest.NewRecorder()
	r.ServeHTTP(recSame, reqSame)
	if recSame.Code != http.StatusUnprocessableEntity || !strings.Contains(recSame.Body.String(), "same_entity_transfer") {
		t.Fatalf("same entity transfer got %d %s, want 422 same_entity_transfer", recSame.Code, recSame.Body.String())
	}

	// Seed CHF open position with no FX rate -> POST /v1/finance/fx-revaluation returns 422 fx_rate_missing.
	_, _ = svc.CreateEntity(ctx, multientity.CreateEntityInput{
		TenantID: tenantID, EntityID: "e_chf", Name: "CHF Ent", CountryCode: "UZ", BaseCurrency: "UZS", IsParent: true,
	})
	meRepo.SeedForeignPositions(tenantID, []multientity.FxRevalPosition{
		{SourceType: "AR_INVOICE", SourceID: "chf-1", AccountCode: gl.AccountAccountsReceivable, Direction: "ASSET", Currency: "CHF", ForeignAmountMinor: 5000},
	})
	reqMissFX := httptest.NewRequest(http.MethodPost, "/v1/finance/fx-revaluation", bytes.NewBufferString(`{"entity_id":"e_chf","period_id":"2026-11","base_currency":"UZS"}`))
	reqMissFX.AddCookie(adminCookie)
	reqMissFX.Header.Set("Idempotency-Key", "miss-fx-1")
	recMissFX := httptest.NewRecorder()
	r.ServeHTTP(recMissFX, reqMissFX)
	if recMissFX.Code != http.StatusUnprocessableEntity || !strings.Contains(recMissFX.Body.String(), "fx_rate_missing") {
		t.Fatalf("missing FX rate got %d %s, want 422 fx_rate_missing", recMissFX.Code, recMissFX.Body.String())
	}
}

func TestMultientityRoutes_EndToEnd_LiveSpanner(t *testing.T) {
	client := openSpannerRoutesClient(t)
	ctx := context.Background()

	tenantID := fmt.Sprintf("sup_mer_live_%d", time.Now().UnixNano())
	rateEUROpenID := fmt.Sprintf("fxr_eur_open_%d", time.Now().UnixNano())
	rateEURCloseID := fmt.Sprintf("fxr_eur_close_%d", time.Now().UnixNano())
	rateKZTUZSID := fmt.Sprintf("fxr_kzt_uzs_%d", time.Now().UnixNano())
	rateUZSKZTID := fmt.Sprintf("fxr_uzs_kzt_%d", time.Now().UnixNano())
	cleanupRoutesTenant(t, client, tenantID, []string{rateEUROpenID, rateEURCloseID, rateKZTUZSID, rateUZSKZTID})

	glSvc := gl.NewService(client)
	fxRepo := fxrates.NewSpannerRepository(client)
	fxSvc := fxrates.NewService(fxRepo)
	svc := multientity.NewService(client, glSvc, fxSvc, fxRepo)
	svc.SetAuditRecorder(routeAuditAdapter{svc: audit.NewService(client)})
	glSvc.SetLegalEntityOwnershipChecker(svc)

	if err := glSvc.SeedDefaults(ctx, tenantID, "UZ", "UZS"); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}

	const jwtSecret = "test-mer-live-secret-0123456789abc"
	r := chi.NewRouter()
	multientityroutes.RegisterRoutes(r, multientityroutes.Deps{
		Service:   svc,
		JWTSecret: jwtSecret,
	})
	adminCookie := issueCookie(t, jwtSecret, auth.Claims{
		Subject:    "cfo-http",
		Role:       auth.RoleAdmin,
		SupplierID: tenantID,
	})
	future := time.Now().UTC().Add(time.Hour)
	auditorCookie := issueCookie(t, jwtSecret, auth.Claims{
		Subject:          "aud-http",
		Role:             auth.RoleAdmin,
		SupplierID:       tenantID,
		SupplierRole:     auth.SupplierRoleAuditor,
		AuditorExpiresAt: &future,
	})

	// 1. POST /v1/org/entities (parent UZ + child KZ) & PATCH /v1/org/entities/{entityID}.
	uzID := tenantID + "_uz"
	kzID := tenantID + "_kz"
	for _, body := range []string{
		fmt.Sprintf(`{"entity_id":%q,"name":"HTTP UZ Parent","tin":"301001001","country_code":"UZ","base_currency":"UZS","is_parent":true}`, uzID),
		fmt.Sprintf(`{"entity_id":%q,"name":"HTTP KZ Sub","tin":"190140001122","country_code":"KZ","base_currency":"KZT","parent_entity_id":%q}`, kzID, uzID),
	} {
		req := httptest.NewRequest(http.MethodPost, "/v1/org/entities", bytes.NewBufferString(body))
		req.AddCookie(adminCookie)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("POST /v1/org/entities status = %d, want 201 (%s)", rec.Code, rec.Body.String())
		}
	}

	reqPatch := httptest.NewRequest(http.MethodPatch, "/v1/org/entities/"+kzID, bytes.NewBufferString(`{"name":"HTTP KZ Sub Updated"}`))
	reqPatch.AddCookie(adminCookie)
	recPatch := httptest.NewRecorder()
	r.ServeHTTP(recPatch, reqPatch)
	if recPatch.Code != http.StatusOK {
		t.Fatalf("PATCH /v1/org/entities/%s status = %d, want 200 (%s)", kzID, recPatch.Code, recPatch.Body.String())
	}

	// 2. Seed FX rates & foreign EUR open receivable.
	t0 := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 11, 16, 18, 0, 0, 0, time.UTC)
	for _, rate := range []fxrates.Rate{
		{RateID: rateEUROpenID, BaseCurrency: "EUR", QuoteCurrency: "UZS", RateScaled: 13500 * fxrates.DefaultScale, Scale: fxrates.DefaultScale, EffectiveAt: t0, Source: "CBU"},
		{RateID: rateEURCloseID, BaseCurrency: "EUR", QuoteCurrency: "UZS", RateScaled: 13800 * fxrates.DefaultScale, Scale: fxrates.DefaultScale, EffectiveAt: t1, Source: "CBU"},
		{RateID: rateKZTUZSID, BaseCurrency: "KZT", QuoteCurrency: "UZS", RateScaled: 25 * fxrates.DefaultScale, Scale: fxrates.DefaultScale, EffectiveAt: t0, Source: "NBK"},
		{RateID: rateUZSKZTID, BaseCurrency: "UZS", QuoteCurrency: "KZT", RateScaled: 4_000_000, Scale: fxrates.DefaultScale, EffectiveAt: t0, Source: "NBK"},
	} {
		if err := fxRepo.Upsert(ctx, rate); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := glSvc.PostJournal(ctx, gl.PostInput{
		TenantID: tenantID, EntityID: uzID, PeriodID: "2026-11",
		SourceType: "MANUAL", SourceID: "http-eur-1", Currency: "EUR", CountryPack: "UZ",
		IdempotencyKey: tenantID + "-http-eur", OccurredAt: t0.Add(time.Minute),
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountAccountsReceivable, Direction: gl.DirectionDebit, AmountMinor: 10000, Currency: "EUR", FunctionalAmountMinor: 135000000, FxRateID: rateEUROpenID},
			{AccountCode: gl.AccountSalesRevenue, Direction: gl.DirectionCredit, AmountMinor: 10000, Currency: "EUR", FunctionalAmountMinor: 135000000, FxRateID: rateEUROpenID},
		},
	}); err != nil {
		t.Fatal(err)
	}

	// 3. POST /v1/finance/intercompany-transfers -> 201 Created, replay -> 200 OK.
	icBody := fmt.Sprintf(`{"from_entity_id":%q,"to_entity_id":%q,"transfer_type":"CASH","amount_minor":500000,"currency":"UZS","period_id":"2026-11"}`, uzID, kzID)
	reqIC1 := httptest.NewRequest(http.MethodPost, "/v1/finance/intercompany-transfers", bytes.NewBufferString(icBody))
	reqIC1.AddCookie(adminCookie)
	reqIC1.Header.Set("Idempotency-Key", tenantID+"-http-ic-1")
	recIC1 := httptest.NewRecorder()
	r.ServeHTTP(recIC1, reqIC1)
	if recIC1.Code != http.StatusCreated {
		t.Fatalf("POST /v1/finance/intercompany-transfers status = %d, want 201 (%s)", recIC1.Code, recIC1.Body.String())
	}

	reqIC2 := httptest.NewRequest(http.MethodPost, "/v1/finance/intercompany-transfers", bytes.NewBufferString(icBody))
	reqIC2.AddCookie(adminCookie)
	reqIC2.Header.Set("Idempotency-Key", tenantID+"-http-ic-1")
	recIC2 := httptest.NewRecorder()
	r.ServeHTTP(recIC2, reqIC2)
	if recIC2.Code != http.StatusOK {
		t.Fatalf("replay POST /v1/finance/intercompany-transfers status = %d, want 200 (%s)", recIC2.Code, recIC2.Body.String())
	}

	// 4. POST /v1/finance/fx-revaluation -> 201 Created (+3,000,000 UZS unrealized gain).
	fxBody := fmt.Sprintf(`{"entity_id":%q,"period_id":"2026-11","base_currency":"UZS","as_of":%q}`, uzID, t1.Format(time.RFC3339))
	reqFX := httptest.NewRequest(http.MethodPost, "/v1/finance/fx-revaluation", bytes.NewBufferString(fxBody))
	reqFX.AddCookie(adminCookie)
	reqFX.Header.Set("Idempotency-Key", tenantID+"-http-fx-1")
	recFX := httptest.NewRecorder()
	r.ServeHTTP(recFX, reqFX)
	if recFX.Code != http.StatusCreated {
		t.Fatalf("POST /v1/finance/fx-revaluation status = %d, want 201 (%s)", recFX.Code, recFX.Body.String())
	}
	var fxResp multientity.FxRevalResult
	if err := json.Unmarshal(recFX.Body.Bytes(), &fxResp); err != nil {
		t.Fatal(err)
	}
	if fxResp.Run.UnrealizedGainLossMinor != 3000000 || fxResp.Run.JournalEntryID == "" {
		t.Fatalf("unexpected HTTP fx revaluation response: %+v", fxResp)
	}

	// 5. GET /v1/finance/consolidation via active AUDITOR -> 200 OK, balanced & eliminated!
	reqCons := httptest.NewRequest(http.MethodGet, "/v1/finance/consolidation?period=2026-11&currency=UZS&entities="+uzID+","+kzID, nil)
	reqCons.AddCookie(auditorCookie)
	recCons := httptest.NewRecorder()
	r.ServeHTTP(recCons, reqCons)
	if recCons.Code != http.StatusOK {
		t.Fatalf("GET /v1/finance/consolidation status = %d, want 200 (%s)", recCons.Code, recCons.Body.String())
	}
	var cons multientity.ConsolidationReport
	if err := json.Unmarshal(recCons.Body.Bytes(), &cons); err != nil {
		t.Fatal(err)
	}
	if !cons.Balanced || cons.TotalEliminationsMinor != 500000 || cons.ICVarianceMinor != 0 || len(cons.Entities) != 2 {
		t.Fatalf("unexpected HTTP consolidation report: %+v", cons)
	}
}
