package glroutes_test

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
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/glroutes"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openRoutesSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping glroutes live emulator test")
	}

	project := envDefault("SPANNER_PROJECT", "pegasusx-local")
	instance := envDefault("SPANNER_INSTANCE", "pegasusx-instance")
	database := envDefault("SPANNER_DATABASE", "pegasusx-db")
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := spanner.NewClient(
		ctx, dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner connect: %v", err)
		}
		t.Skipf("spanner unavailable: %v", err)
	}
	_, probeErr := client.Single().ReadRow(ctx, "JournalEntries", spanner.Key{"__glr_probe__", "__glr_probe__"}, []string{"EntryId"})
	if probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("JournalEntries unavailable: %v", probeErr)
		}
		t.Skipf("JournalEntries unavailable: %v", probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

func envDefault(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}

func cleanupEntity(t *testing.T, client *spanner.Client, entityID string, pleIDs []string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		muts := []*spanner.Mutation{
			spanner.Delete("JournalPostings", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("JournalEntries", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("GlBalances", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("GlAccounts", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
		}
		for _, id := range pleIDs {
			muts = append(muts, spanner.Delete("PaymentLedgerEntries", spanner.Key{id}))
		}
		iter := client.Single().Query(ctx, spanner.Statement{
			SQL:    `SELECT EventId FROM OutboxEvents WHERE SupplierId = @sid`,
			Params: map[string]any{"sid": entityID},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var eid string
			if row.Column(0, &eid) == nil {
				muts = append(muts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
			}
		}
		iter.Stop()
		_, _ = client.Apply(ctx, muts)
	})
}

func issueTestCookie(t *testing.T, secret string, c auth.Claims) *http.Cookie {
	t.Helper()
	tok, err := auth.Issue(c, auth.IssueOptions{Secret: secret, TTL: time.Hour})
	if err != nil {
		t.Fatalf("auth.Issue: %v", err)
	}
	return &http.Cookie{Name: auth.CookieName, Value: tok}
}

func TestGLRoutes_EndToEndLifecycle_And_AuthTenantIsolation(t *testing.T) {
	client := openRoutesSpannerClient(t)
	ctx := context.Background()
	entityID := fmt.Sprintf("sup_glr_%d", time.Now().UnixNano())
	otherEntityID := entityID + "_other"
	pleID := entityID + "_ple_1"
	cleanupEntity(t, client, entityID, []string{pleID})

	// Seed one PaymentLedgerEntries row for POST /v1/finance/gl/backfill.
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("PaymentLedgerEntries", map[string]any{
			"LedgerEntryId": pleID, "OrderId": "ord-route-1", "SupplierId": entityID,
			"Gateway": "PAYME", "EntryType": "WEBHOOK_PAID", "AmountMinor": int64(35000),
			"Currency": "UZS", "ReferenceId": "ref_" + entityID, "Source": "payment.webhook",
			"OccurredAt": time.Now().UTC().Add(-5 * time.Second).Truncate(time.Microsecond), "CreatedAt": spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed PaymentLedgerEntries: %v", err)
	}

	const jwtSecret = "test-gl-secret-0123456789abcdef"
	svc := gl.NewService(client)
	r := chi.NewRouter()
	glroutes.RegisterRoutes(r, glroutes.Deps{
		Service:   svc,
		JWTSecret: jwtSecret,
	})

	adminCookie := issueTestCookie(t, jwtSecret, auth.Claims{
		Subject:    "admin-1",
		Role:       auth.RoleAdmin,
		SupplierID: entityID,
	})
	retailerCookie := issueTestCookie(t, jwtSecret, auth.Claims{
		Subject:    "ret-1",
		Role:       auth.RoleRetailer,
		SupplierID: entityID,
	})
	platformAdminCookie := issueTestCookie(t, jwtSecret, auth.Claims{
		Subject: "pa-1",
		Role:    auth.RolePlatformAdmin,
	})

	// 1. Unauthenticated request must return 401.
	reqUnauth := httptest.NewRequest(http.MethodGet, "/v1/finance/gl/trial-balance", nil)
	recUnauth := httptest.NewRecorder()
	r.ServeHTTP(recUnauth, reqUnauth)
	if recUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", recUnauth.Code)
	}

	// 2. Non-admin role (RETAILER) must return 403.
	reqRet := httptest.NewRequest(http.MethodGet, "/v1/finance/gl/trial-balance", nil)
	reqRet.AddCookie(retailerCookie)
	recRet := httptest.NewRecorder()
	r.ServeHTTP(recRet, reqRet)
	if recRet.Code != http.StatusForbidden {
		t.Fatalf("retailer status = %d, want 403", recRet.Code)
	}

	// 3. Cross-entity request by tenant ADMIN must fail closed (403).
	reqCross := httptest.NewRequest(http.MethodGet, "/v1/finance/gl/trial-balance?entity="+otherEntityID, nil)
	reqCross.AddCookie(adminCookie)
	recCross := httptest.NewRecorder()
	r.ServeHTTP(recCross, reqCross)
	if recCross.Code != http.StatusForbidden {
		t.Fatalf("cross-entity status = %d, want 403", recCross.Code)
	}

	// 4. Unbalanced manual journal must return 422.
	unbalBody := `{"source_id":"man-bad","currency":"UZS","memo":"bad","postings":[{"account_code":"CASH:OFFICE","direction":"DEBIT","amount_minor":500},{"account_code":"REV:SALES","direction":"CREDIT","amount_minor":400}]}`
	reqBad := httptest.NewRequest(http.MethodPost, "/v1/finance/journal/manual", bytes.NewBufferString(unbalBody))
	reqBad.AddCookie(adminCookie)
	reqBad.Header.Set("Idempotency-Key", "man-bad-1")
	recBad := httptest.NewRecorder()
	r.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unbalanced manual status = %d, want 422 (%s)", recBad.Code, recBad.Body.String())
	}

	// 5. Balanced manual journal -> 201 Created.
	manBody := `{"source_id":"man-ok","currency":"UZS","memo":"Opening cash injection","postings":[{"account_code":"CASH:OFFICE","direction":"DEBIT","amount_minor":120000},{"account_code":"EQUITY:OPENING_BALANCE","direction":"CREDIT","amount_minor":120000}]}`
	reqMan := httptest.NewRequest(http.MethodPost, "/v1/finance/journal/manual", bytes.NewBufferString(manBody))
	reqMan.AddCookie(adminCookie)
	reqMan.Header.Set("Idempotency-Key", "man-ok-1")
	recMan := httptest.NewRecorder()
	r.ServeHTTP(recMan, reqMan)
	if recMan.Code != http.StatusCreated {
		t.Fatalf("manual create status = %d, want 201 (%s)", recMan.Code, recMan.Body.String())
	}
	var createdRes gl.JournalEntryResult
	if err := json.Unmarshal(recMan.Body.Bytes(), &createdRes); err != nil {
		t.Fatal(err)
	}
	if createdRes.EntryID == "" || createdRes.TotalDebit != 120000 || createdRes.AlreadyPosted {
		t.Fatalf("unexpected manual create response: %+v", createdRes)
	}

	// 6. Idempotent replay -> 200 OK with already_posted=true.
	reqManDup := httptest.NewRequest(http.MethodPost, "/v1/finance/journal/manual", bytes.NewBufferString(manBody))
	reqManDup.AddCookie(adminCookie)
	reqManDup.Header.Set("Idempotency-Key", "man-ok-1")
	recManDup := httptest.NewRecorder()
	r.ServeHTTP(recManDup, reqManDup)
	if recManDup.Code != http.StatusOK {
		t.Fatalf("manual replay status = %d, want 200 (%s)", recManDup.Code, recManDup.Body.String())
	}

	// 7. GET /v1/finance/journal/{entryId} -> 200 OK.
	reqGet := httptest.NewRequest(http.MethodGet, "/v1/finance/journal/"+createdRes.EntryID, nil)
	reqGet.AddCookie(adminCookie)
	recGet := httptest.NewRecorder()
	r.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("get journal status = %d, want 200 (%s)", recGet.Code, recGet.Body.String())
	}
	var gotEntry gl.JournalEntryRecord
	if err := json.Unmarshal(recGet.Body.Bytes(), &gotEntry); err != nil {
		t.Fatal(err)
	}
	if gotEntry.EntryHash != createdRes.Hash || len(gotEntry.Postings) != 2 {
		t.Fatalf("unexpected get journal payload: %+v", gotEntry)
	}

	// 8. GET /v1/finance/gl/accounts/CASH:OFFICE/ledger -> 200 OK.
	reqAcct := httptest.NewRequest(http.MethodGet, "/v1/finance/gl/accounts/CASH:OFFICE/ledger?period="+createdRes.PeriodID+"&limit=20", nil)
	reqAcct.AddCookie(adminCookie)
	recAcct := httptest.NewRecorder()
	r.ServeHTTP(recAcct, reqAcct)
	if recAcct.Code != http.StatusOK {
		t.Fatalf("account ledger status = %d, want 200 (%s)", recAcct.Code, recAcct.Body.String())
	}
	var acctBody struct {
		Count int                   `json:"count"`
		Items []gl.AccountLedgerRow `json:"items"`
	}
	if err := json.Unmarshal(recAcct.Body.Bytes(), &acctBody); err != nil {
		t.Fatal(err)
	}
	if acctBody.Count != 1 || acctBody.Items[0].AmountMinor != 120000 {
		t.Fatalf("unexpected account ledger response: %+v", acctBody)
	}

	// 9. GET /v1/finance/gl/trial-balance via PLATFORM_ADMIN with ?entity= -> 200 OK.
	reqTB := httptest.NewRequest(http.MethodGet, "/v1/finance/gl/trial-balance?entity="+entityID+"&period="+createdRes.PeriodID, nil)
	reqTB.AddCookie(platformAdminCookie)
	recTB := httptest.NewRecorder()
	r.ServeHTTP(recTB, reqTB)
	if recTB.Code != http.StatusOK {
		t.Fatalf("trial balance status = %d, want 200 (%s)", recTB.Code, recTB.Body.String())
	}
	var tb gl.TrialBalanceReport
	if err := json.Unmarshal(recTB.Body.Bytes(), &tb); err != nil {
		t.Fatal(err)
	}
	if !tb.Balanced || tb.TotalDebitsMinor != 120000 || len(tb.Accounts) != 2 {
		t.Fatalf("unexpected trial balance: %+v", tb)
	}

	// 10. POST /v1/finance/journal/{id}/reverse -> 201 Created, second key -> 409 Conflict.
	reqRev := httptest.NewRequest(http.MethodPost, "/v1/finance/journal/"+createdRes.EntryID+"/reverse", bytes.NewBufferString(`{"reason":"audit test"}`))
	reqRev.AddCookie(adminCookie)
	reqRev.Header.Set("Idempotency-Key", "rev-route-1")
	recRev := httptest.NewRecorder()
	r.ServeHTTP(recRev, reqRev)
	if recRev.Code != http.StatusCreated {
		t.Fatalf("reverse status = %d, want 201 (%s)", recRev.Code, recRev.Body.String())
	}
	reqRev2 := httptest.NewRequest(http.MethodPost, "/v1/finance/journal/"+createdRes.EntryID+"/reverse", bytes.NewBufferString(`{"reason":"second"}`))
	reqRev2.AddCookie(adminCookie)
	reqRev2.Header.Set("Idempotency-Key", "rev-route-2")
	recRev2 := httptest.NewRecorder()
	r.ServeHTTP(recRev2, reqRev2)
	if recRev2.Code != http.StatusConflict {
		t.Fatalf("second reverse status = %d, want 409 (%s)", recRev2.Code, recRev2.Body.String())
	}

	// 11. POST /v1/finance/gl/backfill -> 200 OK, idempotent on second call.
	reqBF1 := httptest.NewRequest(http.MethodPost, "/v1/finance/gl/backfill", bytes.NewBufferString(`{"limit":50}`))
	reqBF1.AddCookie(adminCookie)
	recBF1 := httptest.NewRecorder()
	r.ServeHTTP(recBF1, reqBF1)
	if recBF1.Code != http.StatusOK {
		t.Fatalf("backfill 1 status = %d, want 200 (%s)", recBF1.Code, recBF1.Body.String())
	}
	var bf1 gl.BackfillReport
	if err := json.Unmarshal(recBF1.Body.Bytes(), &bf1); err != nil {
		t.Fatal(err)
	}
	if bf1.Posted != 1 {
		t.Fatalf("backfill 1 posted = %d, want 1 (%+v)", bf1.Posted, bf1)
	}

	reqBF2 := httptest.NewRequest(http.MethodPost, "/v1/finance/gl/backfill", bytes.NewBufferString(`{"limit":50}`))
	reqBF2.AddCookie(adminCookie)
	recBF2 := httptest.NewRecorder()
	r.ServeHTTP(recBF2, reqBF2)
	if recBF2.Code != http.StatusOK {
		t.Fatalf("backfill 2 status = %d, want 200 (%s)", recBF2.Code, recBF2.Body.String())
	}
	var bf2 gl.BackfillReport
	if err := json.Unmarshal(recBF2.Body.Bytes(), &bf2); err != nil {
		t.Fatal(err)
	}
	if bf2.Posted != 0 || bf2.AlreadyPosted != 1 {
		t.Fatalf("backfill 2 report unexpected: %+v", bf2)
	}

	// 12. Enforced tenant isolation (TENANT_CONTEXT_ENFORCED=true) + Bearer token auth.
	t.Setenv("TENANT_CONTEXT_ENFORCED", "true")
	noTenantToken, err := auth.Issue(auth.Claims{
		Subject: "admin-no-tenant",
		Role:    auth.RoleAdmin,
	}, auth.IssueOptions{Secret: jwtSecret, TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	reqNoTenant := httptest.NewRequest(http.MethodGet, "/v1/finance/gl/trial-balance", nil)
	reqNoTenant.Header.Set("Authorization", "Bearer "+noTenantToken)
	recNoTenant := httptest.NewRecorder()
	r.ServeHTTP(recNoTenant, reqNoTenant)
	if recNoTenant.Code != http.StatusUnauthorized {
		t.Fatalf("enforced missing tenant status = %d, want 401 (%s)", recNoTenant.Code, recNoTenant.Body.String())
	}

	// Platform admin without ?entity= must fail closed (400).
	reqPANoEntity := httptest.NewRequest(http.MethodGet, "/v1/finance/gl/trial-balance", nil)
	reqPANoEntity.AddCookie(platformAdminCookie)
	recPANoEntity := httptest.NewRecorder()
	r.ServeHTTP(recPANoEntity, reqPANoEntity)
	if recPANoEntity.Code != http.StatusBadRequest {
		t.Fatalf("platform admin missing entity status = %d, want 400", recPANoEntity.Code)
	}

	// Admin Bearer token with valid SupplierID succeeds under TENANT_CONTEXT_ENFORCED=true.
	reqBearer := httptest.NewRequest(http.MethodGet, "/v1/finance/gl/trial-balance?period="+createdRes.PeriodID, nil)
	reqBearer.Header.Set("Authorization", "Bearer "+adminCookie.Value)
	recBearer := httptest.NewRecorder()
	r.ServeHTTP(recBearer, reqBearer)
	if recBearer.Code != http.StatusOK {
		t.Fatalf("bearer enforced tenant status = %d, want 200 (%s)", recBearer.Code, recBearer.Body.String())
	}

	// 13. Supplier RBAC: AUDITOR can read trial-balance but cannot post/reverse; AGENT cannot post/reverse;
	// ACCOUNTANT can post manual journal but cannot reverse; FINANCE_MANAGER can reverse but cannot post manual.
	issueSupCookie := func(sub string, sr string, actorType string) *http.Cookie {
		return issueTestCookie(t, jwtSecret, auth.Claims{
			Subject:      sub,
			Role:         auth.RoleAdmin,
			SupplierID:   entityID,
			SupplierRole: auth.Role(sr),
			ActorType:    actorType,
		})
	}

	auditorCookie := issueSupCookie("aud-gl", auth.SupplierRoleAuditor, auth.ActorTypeHuman)
	agentCookie := issueSupCookie("agent-gl", auth.SupplierRoleAgent, auth.ActorTypeAgent)
	acctCookie := issueSupCookie("acct-gl", auth.SupplierRoleAccountant, auth.ActorTypeHuman)
	finMgrCookie := issueSupCookie("finmgr-gl", auth.SupplierRoleFinanceManager, auth.ActorTypeHuman)

	reqAudRead := httptest.NewRequest(http.MethodGet, "/v1/finance/gl/trial-balance", nil)
	reqAudRead.AddCookie(auditorCookie)
	recAudRead := httptest.NewRecorder()
	r.ServeHTTP(recAudRead, reqAudRead)
	if recAudRead.Code != http.StatusOK {
		t.Fatalf("auditor GL GET status = %d, want 200 (%s)", recAudRead.Code, recAudRead.Body.String())
	}

	reqAudPost := httptest.NewRequest(http.MethodPost, "/v1/finance/journal/manual", bytes.NewBufferString(manBody))
	reqAudPost.AddCookie(auditorCookie)
	reqAudPost.Header.Set("Idempotency-Key", "aud-post-denied")
	recAudPost := httptest.NewRecorder()
	r.ServeHTTP(recAudPost, reqAudPost)
	if recAudPost.Code != http.StatusForbidden {
		t.Fatalf("auditor GL POST status = %d, want 403", recAudPost.Code)
	}

	reqAgentRev := httptest.NewRequest(http.MethodPost, "/v1/finance/journal/"+createdRes.EntryID+"/reverse", bytes.NewBufferString(`{"reason":"agent"}`))
	reqAgentRev.AddCookie(agentCookie)
	reqAgentRev.Header.Set("Idempotency-Key", "agent-rev-denied")
	recAgentRev := httptest.NewRecorder()
	r.ServeHTTP(recAgentRev, reqAgentRev)
	if recAgentRev.Code != http.StatusForbidden {
		t.Fatalf("agent GL reverse status = %d, want 403", recAgentRev.Code)
	}

	// ACCOUNTANT lacks journal.reverse -> 403.
	reqAcctRev := httptest.NewRequest(http.MethodPost, "/v1/finance/journal/"+createdRes.EntryID+"/reverse", bytes.NewBufferString(`{"reason":"acct"}`))
	reqAcctRev.AddCookie(acctCookie)
	reqAcctRev.Header.Set("Idempotency-Key", "acct-rev-denied")
	recAcctRev := httptest.NewRecorder()
	r.ServeHTTP(recAcctRev, reqAcctRev)
	if recAcctRev.Code != http.StatusForbidden {
		t.Fatalf("accountant GL reverse status = %d, want 403", recAcctRev.Code)
	}

	// FINANCE_MANAGER lacks journal.post -> 403 on manual journal.
	reqFinPost := httptest.NewRequest(http.MethodPost, "/v1/finance/journal/manual", bytes.NewBufferString(manBody))
	reqFinPost.AddCookie(finMgrCookie)
	reqFinPost.Header.Set("Idempotency-Key", "fin-post-denied")
	recFinPost := httptest.NewRecorder()
	r.ServeHTTP(recFinPost, reqFinPost)
	if recFinPost.Code != http.StatusForbidden {
		t.Fatalf("finance_manager GL manual post status = %d, want 403", recFinPost.Code)
	}
}
