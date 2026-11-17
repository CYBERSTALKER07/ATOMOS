package e2ep0

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/connectors"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/exportpacks"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/multientity"
	"github.com/pegasusx/pegasusx/apps/backend-go/p2p"
	"github.com/xuri/excelize/v2"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openWave4ASpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping Wave 4A live Spanner E2E test")
	}
	envOr := func(k, def string) string {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
		return def
	}
	dsn := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		envOr("SPANNER_PROJECT", "pegasusx-local"),
		envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		envOr("SPANNER_DATABASE", "pegasusx-db"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dsn,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("spanner.NewClient: %v", err)
	}
	for _, spec := range []struct {
		tbl string
		col string
	}{
		{"LegalEntities", "Status"},
		{"IntercompanyTransfers", "Status"},
		{"FxRevaluationRuns", "PeriodId"},
		{"PurchaseOrders", "Status"},
		{"GoodsReceipts", "Status"},
		{"VendorInvoices", "MatchStatus"},
		{"ConnectorSyncRuns", "Status"},
		{"ExportPacks", "PackType"},
	} {
		ro := client.ReadOnlyTransaction()
		_, pErr := ro.ReadRow(ctx, spec.tbl, spanner.Key{"__w4a_e2e__", "__probe__"}, []string{spec.col})
		ro.Close()
		if pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
			client.Close()
			t.Fatalf("Wave 4A table %s unavailable: %v", spec.tbl, pErr)
		}
	}
	t.Cleanup(client.Close)
	return client
}

func issueWave4ASupplierToken(t *testing.T, secret, issuer string, role auth.Role, supRole, supID, subject string, auditorExpiry *time.Time) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:          subject,
		Role:             role,
		SupplierID:       supID,
		SupplierRole:     auth.Role(supRole),
		AuditorExpiresAt: auditorExpiry,
	}, auth.IssueOptions{
		Secret: secret,
		TTL:    time.Hour,
		Issuer: issuer,
	})
	if err != nil {
		t.Fatalf("auth.Issue(%s): %v", subject, err)
	}
	return tok
}

func doWave4AHTTP(t *testing.T, method, url, bearer string, body any, idemKey string) (int, http.Header, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		t.Fatalf("http.NewRequest(%s %s): %v", method, url, err)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("http.Do(%s %s): %v", method, url, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return resp.StatusCode, resp.Header, raw
}

func assertWave4AJSONNoFloatLiterals(t *testing.T, data []byte, label string) {
	t.Helper()
	var val any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&val); err != nil {
		t.Fatalf("%s: decode JSON: %v (raw=%s)", label, err, string(data))
	}
	var walk func(v any, path string)
	walk = func(v any, path string) {
		switch tv := v.(type) {
		case float64, float32:
			t.Fatalf("%s.%s contains unexpected float %T", label, path, tv)
		case json.Number:
			if strings.ContainsAny(tv.String(), ".eE") {
				t.Fatalf("%s.%s contains decimal/scientific float JSON number %q", label, path, tv.String())
			}
		case map[string]any:
			for k, elem := range tv {
				walk(elem, path+"."+k)
			}
		case []any:
			for i, elem := range tv {
				walk(elem, fmt.Sprintf("%s[%d]", path, i))
			}
		}
	}
	walk(val, "$")
}

func cleanupWave4ASpannerTenant(client *spanner.Client, tenantID string, extraRateIDs, extraRetailers, extraInvoices, extraLedgerIDs []string) {
	if client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var muts []*spanner.Mutation
	ro := client.ReadOnlyTransaction()
	oIter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT EventId FROM OutboxEvents WHERE SupplierId = @tid`,
		Params: map[string]any{"tid": tenantID},
	})
	for {
		r, err := oIter.Next()
		if err != nil {
			break
		}
		var eid string
		if err := r.Columns(&eid); err == nil {
			muts = append(muts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
		}
	}
	oIter.Stop()
	ro.Close()

	for _, rid := range extraRateIDs {
		muts = append(muts, spanner.Delete("FxRates", spanner.Key{rid}))
	}
	for _, r := range extraRetailers {
		muts = append(muts,
			spanner.Delete("CreditScoreHistory", spanner.KeyRange{Start: spanner.Key{r}, End: spanner.Key{r + "~"}}),
			spanner.Delete("RetailerCreditProfiles", spanner.Key{r, tenantID}),
			spanner.Delete("RetailerPaymentTerms", spanner.Key{r, tenantID}),
		)
	}
	for _, le := range extraLedgerIDs {
		muts = append(muts, spanner.Delete("ArLedgerEntries", spanner.Key{le}))
	}
	for _, inv := range extraInvoices {
		muts = append(muts, spanner.Delete("ArInvoices", spanner.Key{inv}))
	}

	kr := spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}
	muts = append(muts,
		spanner.Delete("ApprovalDecisions", kr),
		spanner.Delete("ApprovalRequests", kr),
		spanner.Delete("ApprovalPolicies", kr),
		spanner.Delete("VendorInvoices", kr),
		spanner.Delete("GoodsReceipts", kr),
		spanner.Delete("PurchaseOrderLines", kr),
		spanner.Delete("PurchaseOrders", kr),
		spanner.Delete("ProcessInstances", kr),
		spanner.Delete("ProcessTemplates", kr),
		spanner.Delete("EsfDocuments", kr),
		spanner.Delete("IntercompanyTransfers", kr),
		spanner.Delete("FxRevaluationRuns", kr),
		spanner.Delete("LegalEntities", kr),
		spanner.Delete("JournalPostings", kr),
		spanner.Delete("JournalEntries", kr),
		spanner.Delete("GlBalances", kr),
		spanner.Delete("GlAccounts", kr),
		spanner.Delete("Periods", kr),
		spanner.Delete("ConnectorSyncRuns", kr),
		spanner.Delete("ExternalCredentials", kr),
		spanner.Delete("ExportPacks", kr),
		spanner.Delete("AuditEvents", kr),
	)
	_, _ = client.Apply(ctx, muts)
}

func TestWave4A_MultiEntity_Intercompany_FxReval_Consolidation_LiveSpanner(t *testing.T) {
	client := openWave4ASpannerClient(t)
	h := NewTestHarnessWithOptions(t, HarnessOptions{SpannerClient: client})
	ctx := context.Background()

	tag := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	tenantID := "sup-w4a-me-" + tag
	rateCHFCloseID := "w4a-chf-uzs-" + tag
	rateKZTUZSID := "w4a-kzt-uzs-" + tag
	rateUZSKZTID := "w4a-uzs-kzt-" + tag

	t.Cleanup(func() {
		cleanupWave4ASpannerTenant(client, tenantID, []string{rateCHFCloseID, rateKZTUZSID, rateUZSKZTID}, nil, nil, nil)
	})

	if err := h.GLSvc.SeedDefaults(ctx, tenantID, "UZ", "UZS"); err != nil {
		t.Fatalf("GLSvc.SeedDefaults: %v", err)
	}

	futureExp := time.Now().UTC().Add(2 * time.Hour)
	pastExp := time.Now().UTC().Add(-15 * time.Minute)
	ownerTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleOwner, tenantID, "owner-me-"+tag, nil)
	auditorTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleAuditor, tenantID, "aud-me-"+tag, &futureExp)
	expiredAudTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleAuditor, tenantID, "exp-aud-me-"+tag, &pastExp)
	agentTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleAgent, tenantID, "agent-me-"+tag, nil)

	// 1. RBAC: AUDITOR/AGENT read-only on mutations; expired AUDITOR -> 401
	for _, mutReq := range []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodPost, "/v1/org/entities", map[string]any{"entity_id": "ent-deny", "name": "Denied", "country_code": "UZ", "base_currency": "UZS"}},
		{http.MethodPost, "/v1/finance/intercompany-transfers", map[string]any{"from_entity_id": "a", "to_entity_id": "b", "transfer_type": "LOAN", "amount_minor": 1000, "currency": "UZS", "idempotency_key": "ic-deny"}},
		{http.MethodPost, "/v1/finance/fx-revaluation", map[string]any{"period_id": "2026-11", "base_currency": "UZS", "idempotency_key": "fx-deny"}},
	} {
		for _, tok := range []string{auditorTok, agentTok} {
			code, _, b := doWave4AHTTP(t, mutReq.method, h.Server.URL+mutReq.path, tok, mutReq.body, "idem-deny-"+tag)
			if code != http.StatusForbidden {
				t.Fatalf("%s %s with AUDITOR/AGENT=%d want=403 body=%s", mutReq.method, mutReq.path, code, string(b))
			}
		}
		code, _, b := doWave4AHTTP(t, mutReq.method, h.Server.URL+mutReq.path, expiredAudTok, mutReq.body, "idem-exp-"+tag)
		if code != http.StatusUnauthorized || !strings.Contains(string(b), "auditor_access_expired") {
			t.Fatalf("%s %s expired AUDITOR=%d want=401 auditor_access_expired body=%s", mutReq.method, mutReq.path, code, string(b))
		}
	}
	if code, _, b := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/finance/consolidation?period=2026-11&currency=UZS", expiredAudTok, nil, ""); code != http.StatusUnauthorized || !strings.Contains(string(b), "auditor_access_expired") {
		t.Fatalf("expired AUDITOR GET /v1/finance/consolidation=%d want=401 body=%s", code, string(b))
	}

	// 2. LegalEntity CRUD & GL ownership scope
	entHQ := "ent-hq-" + tag
	entKZ := "ent-kz-" + tag
	code, _, b := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/org/entities", ownerTok, map[string]any{
		"entity_id":     entHQ,
		"name":          "Tashkent HQ LLC",
		"tin":           "301112233",
		"country_code":  "UZ",
		"base_currency": "UZS",
		"is_parent":     true,
	}, "")
	if code != http.StatusCreated {
		t.Fatalf("POST /v1/org/entities(HQ)=%d body=%s", code, string(b))
	}
	code, _, b = doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/org/entities", ownerTok, map[string]any{
		"entity_id":        entKZ,
		"name":             "Almaty Branch LLP",
		"tin":              "998877665544",
		"country_code":     "KZ",
		"base_currency":    "KZT",
		"parent_entity_id": entHQ,
	}, "")
	if code != http.StatusCreated {
		t.Fatalf("POST /v1/org/entities(KZ)=%d body=%s", code, string(b))
	}
	code, _, b = doWave4AHTTP(t, http.MethodPatch, h.Server.URL+"/v1/org/entities/"+entKZ, ownerTok, map[string]any{
		"name": "Almaty Logistics LLP",
	}, "")
	if code != http.StatusOK || !strings.Contains(string(b), "Almaty Logistics LLP") {
		t.Fatalf("PATCH /v1/org/entities(KZ)=%d body=%s", code, string(b))
	}
	if code, _, b := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/finance/trial-balance?entity=ent-foreign-other", ownerTok, nil, ""); code != http.StatusForbidden {
		t.Fatalf("GET /v1/finance/trial-balance unowned entity=%d want=403 body=%s", code, string(b))
	}

	t0 := time.Now().UTC().Truncate(time.Second).Add(-2 * time.Hour)
	t1 := time.Now().UTC().Truncate(time.Second).Add(-5 * time.Minute)
	for _, r := range []fxrates.Rate{
		{RateID: rateKZTUZSID, BaseCurrency: "KZT", QuoteCurrency: "UZS", RateScaled: 25 * fxrates.DefaultScale, Scale: fxrates.DefaultScale, EffectiveAt: t0, Source: "NBK-W4A"},
		{RateID: rateUZSKZTID, BaseCurrency: "UZS", QuoteCurrency: "KZT", RateScaled: 4_000_000, Scale: fxrates.DefaultScale, EffectiveAt: t0, Source: "NBK-W4A"},
	} {
		if err := h.FxRatesRepo.Upsert(ctx, r); err != nil {
			t.Fatalf("FxRatesRepo.Upsert(%s): %v", r.RateID, err)
		}
	}

	// 3. Intercompany Transfer: reject same-entity 422, post UZS->KZT transfer idempotently
	code, _, b = doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/finance/intercompany-transfers", ownerTok, map[string]any{
		"from_entity_id": entHQ,
		"to_entity_id":   entHQ,
		"transfer_type":  "LOAN",
		"amount_minor":   int64(100000),
		"currency":       "UZS",
		"period_id":      "2026-11",
	}, "ic-same-"+tag)
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("same-entity IC transfer=%d want=422 body=%s", code, string(b))
	}

	icIdem := "ic-hq-kz-" + tag
	code, _, b = doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/finance/intercompany-transfers", ownerTok, map[string]any{
		"from_entity_id": entHQ,
		"to_entity_id":   entKZ,
		"transfer_type":  "CASH",
		"amount_minor":   int64(1000000),
		"currency":       "UZS",
		"period_id":      "2026-11",
		"memo":           "HQ advance to Almaty branch",
	}, icIdem)
	if code != http.StatusCreated {
		t.Fatalf("POST /v1/finance/intercompany-transfers=%d body=%s", code, string(b))
	}
	var icRes1 multientity.CreateTransferResult
	if err := json.Unmarshal(b, &icRes1); err != nil {
		t.Fatalf("decode IC transfer: %v", err)
	}
	if icRes1.Transfer.DebitJournalID == "" || icRes1.Transfer.CreditJournalID == "" {
		t.Fatalf("expected DebitJournalID & CreditJournalID on IC transfer: %+v", icRes1)
	}
	// Idempotent replay returns 200 already_posted=true
	code, _, b = doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/finance/intercompany-transfers", ownerTok, map[string]any{
		"from_entity_id": entHQ,
		"to_entity_id":   entKZ,
		"transfer_type":  "CASH",
		"amount_minor":   int64(1000000),
		"currency":       "UZS",
		"period_id":      "2026-11",
	}, icIdem)
	if code != http.StatusOK {
		t.Fatalf("IC replay=%d want=200 body=%s", code, string(b))
	}

	// 4. FX Revaluation: fail closed (422 fx_rate_missing) without CHF/UZS rate, succeed once rate exists
	if _, err := h.GLSvc.PostJournal(ctx, gl.PostInput{
		TenantID:       tenantID,
		EntityID:       entHQ,
		PeriodID:       "2026-11",
		SourceType:     "MANUAL",
		SourceID:       "chf-ar-" + tag,
		Memo:           "CHF open receivable",
		Currency:       "CHF",
		CountryPack:    "UZ",
		IdempotencyKey: "gl-chf-" + tag,
		OccurredAt:     t0,
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountAccountsReceivable, Direction: gl.DirectionDebit, AmountMinor: 10000, Currency: "CHF", FunctionalAmountMinor: 140000000},
			{AccountCode: gl.AccountSalesRevenue, Direction: gl.DirectionCredit, AmountMinor: 10000, Currency: "CHF", FunctionalAmountMinor: 140000000},
		},
	}); err != nil {
		t.Fatalf("PostJournal(CHF AR): %v", err)
	}

	code, _, b = doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/finance/fx-revaluation", ownerTok, map[string]any{
		"entity_id":     entHQ,
		"period_id":     "2026-11",
		"base_currency": "UZS",
		"as_of":         t1.Format(time.RFC3339),
	}, "fxr-miss-"+tag)
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("FX reval missing rate=%d want=422 body=%s", code, string(b))
	}

	if err := h.FxRatesRepo.Upsert(ctx, fxrates.Rate{
		RateID: rateCHFCloseID, BaseCurrency: "CHF", QuoteCurrency: "UZS", RateScaled: 14500 * fxrates.DefaultScale, Scale: fxrates.DefaultScale, EffectiveAt: t1, Source: "CBU-W4A",
	}); err != nil {
		t.Fatalf("FxRatesRepo.Upsert(%s): %v", rateCHFCloseID, err)
	}

	code, _, b = doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/finance/fx-revaluation", ownerTok, map[string]any{
		"entity_id":     entHQ,
		"period_id":     "2026-11",
		"base_currency": "UZS",
		"as_of":         t1.Format(time.RFC3339),
	}, "fxr-ok-"+tag)
	if code != http.StatusCreated {
		t.Fatalf("POST /v1/finance/fx-revaluation=%d body=%s", code, string(b))
	}
	assertWave4AJSONNoFloatLiterals(t, b, "fx_revaluation")
	var revalRes multientity.FxRevalResult
	if err := json.Unmarshal(b, &revalRes); err != nil {
		t.Fatalf("decode FxRevalResult: %v", err)
	}
	if revalRes.Run.UnrealizedGainLossMinor != 5000000 || revalRes.Run.JournalEntryID == "" {
		t.Fatalf("unexpected FX revaluation gain/loss: %+v", revalRes)
	}

	// 5. Consolidation: active AUDITOR can read; IC eliminations balanced & zero floats
	code, _, b = doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/finance/consolidation?period=2026-11&currency=UZS", auditorTok, nil, "")
	if code != http.StatusOK {
		t.Fatalf("AUDITOR GET /v1/finance/consolidation=%d body=%s", code, string(b))
	}
	assertWave4AJSONNoFloatLiterals(t, b, "consolidation_report")
	var report multientity.ConsolidationReport
	if err := json.Unmarshal(b, &report); err != nil {
		t.Fatalf("decode ConsolidationReport: %v", err)
	}
	if !report.Balanced || report.ICVarianceMinor != 0 || report.TotalEliminationsMinor <= 0 || len(report.Entities) != 2 || len(report.FxRatesApplied) == 0 {
		t.Fatalf("unexpected ConsolidationReport: %+v", report)
	}
}

func TestWave4A_P2P_Lifecycle_3WayMatch_ProcessMining_LiveSpanner(t *testing.T) {
	client := openWave4ASpannerClient(t)
	h := NewTestHarnessWithOptions(t, HarnessOptions{SpannerClient: client})
	ctx := context.Background()

	tag := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	tenantID := "sup-w4a-p2p-" + tag
	entID := "ent-p2p-" + tag
	esfID := "esf-p2p-" + tag
	periodID := time.Now().UTC().Format("2006-01")

	t.Cleanup(func() {
		cleanupWave4ASpannerTenant(client, tenantID, nil, nil, nil, nil)
	})

	if err := h.GLSvc.SeedDefaults(ctx, tenantID, "UZ", "UZS"); err != nil {
		t.Fatalf("GLSvc.SeedDefaults: %v", err)
	}
	if _, err := h.MultiEntitySvc.CreateEntity(ctx, multientity.CreateEntityInput{
		TenantID: tenantID, EntityID: entID, Name: "P2P Entity", CountryCode: "UZ", BaseCurrency: "UZS", IsParent: true, ActorID: "owner-" + tag,
	}); err != nil {
		t.Fatalf("CreateEntity(p2p): %v", err)
	}

	futureExp := time.Now().UTC().Add(2 * time.Hour)
	pastExp := time.Now().UTC().Add(-15 * time.Minute)
	ownerTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleOwner, tenantID, "owner-p2p-"+tag, nil)
	accTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleAccountant, tenantID, "acc-p2p-"+tag, nil)
	whTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleWarehouseManager, tenantID, "wh-p2p-"+tag, nil)
	auditorTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleAuditor, tenantID, "aud-p2p-"+tag, &futureExp)
	expiredAudTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleAuditor, tenantID, "exp-aud-p2p-"+tag, &pastExp)
	agentTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleAgent, tenantID, "agent-p2p-"+tag, nil)

	// 1. RBAC blocks AUDITOR & AGENT mutations; expired AUDITOR -> 401
	for _, p := range []string{"/v1/p2p/purchase-orders", "/v1/p2p/receipts", "/v1/p2p/vendor-invoices", "/v1/p2p/three-way-match", "/v1/processes/templates"} {
		for _, tok := range []string{auditorTok, agentTok} {
			code, _, b := doWave4AHTTP(t, http.MethodPost, h.Server.URL+p, tok, map[string]any{}, "idem-p2p-deny-"+tag)
			if code != http.StatusForbidden {
				t.Fatalf("POST %s AUDITOR/AGENT=%d want=403 body=%s", p, code, string(b))
			}
		}
		code, _, b := doWave4AHTTP(t, http.MethodPost, h.Server.URL+p, expiredAudTok, map[string]any{}, "")
		if code != http.StatusUnauthorized || !strings.Contains(string(b), "auditor_access_expired") {
			t.Fatalf("POST %s expired AUDITOR=%d want=401 body=%s", p, code, string(b))
		}
	}
	if code, _, _ := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/processes/mining", expiredAudTok, nil, ""); code != http.StatusUnauthorized {
		t.Fatalf("GET /v1/processes/mining expired AUDITOR=%d want=401", code)
	}

	// 2. Shipped UZ templates & custom template + instance
	code, _, b := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/processes/templates?domain=P2P", auditorTok, nil, "")
	if code != http.StatusOK || !strings.Contains(string(b), "P2P-STD-UZ") {
		t.Fatalf("GET /v1/processes/templates=%d body=%s", code, string(b))
	}

	// 3. Configure CEL approval policy requiring OWNER approval on POs > 500k UZS
	code, _, b = doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/approvals/policies", ownerTok, map[string]any{
		"name":           "PO > 500k",
		"object_type":    "PurchaseOrder",
		"action":         "approve",
		"condition_expr": "amount_minor > 500000",
		"steps": []map[string]any{
			{"step_index": 0, "name": "Owner", "allowed_roles": []string{auth.SupplierRoleOwner}, "required_approvers": 1},
		},
	}, "")
	if code != http.StatusCreated {
		t.Fatalf("create PurchaseOrder CEL policy=%d body=%s", code, string(b))
	}

	// ACCOUNTANT creates PO (1,000,000 subtotal + 120,000 VAT = 1,120,000 -> 202 PENDING_APPROVAL)
	code, _, b = doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/p2p/purchase-orders", accTok, map[string]any{
		"entity_id":   entID,
		"vendor_id":   "ven-uz-" + tag,
		"vendor_name": "Fergana Silk Supply",
		"vendor_tin":  "309876543",
		"currency":    "UZS",
		"lines": []map[string]any{{
			"sku":             "SKU-P2P-01",
			"product_name":    "Cotton Yarn",
			"qty_ordered":     10,
			"unit_cost_minor": 100000,
			"vat_bps":         1200,
		}},
	}, "po-create-"+tag)
	if code != http.StatusAccepted {
		t.Fatalf("POST /v1/p2p/purchase-orders=%d want=202 body=%s", code, string(b))
	}
	var poCreateRes p2p.CreatePurchaseOrderResult
	if err := json.Unmarshal(b, &poCreateRes); err != nil {
		t.Fatalf("decode CreatePurchaseOrderResult: %v", err)
	}
	if !poCreateRes.ApprovalRequired || poCreateRes.ApprovalRequestID == "" || poCreateRes.PurchaseOrder.PoID == "" {
		t.Fatalf("expected ApprovalRequired & PoID, got %+v", poCreateRes)
	}
	poID := poCreateRes.PurchaseOrder.PoID

	// Creator cannot self-approve PO (SoD 403), and receipt blocked while PENDING_APPROVAL (409)
	if code, _, b := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/p2p/purchase-orders/"+poID+"/approve", accTok, map[string]any{
		"approval_id": poCreateRes.ApprovalRequestID,
	}, ""); code != http.StatusForbidden {
		t.Fatalf("creator self-approve PO=%d want=403 body=%s", code, string(b))
	}
	if code, _, b := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/p2p/receipts", whTok, map[string]any{
		"po_id":        poID,
		"warehouse_id": "wh-p2p-1",
		"lines":        []map[string]any{{"sku": "SKU-P2P-01", "qty_received": 10}},
	}, "gr-blocked-"+tag); code != http.StatusConflict {
		t.Fatalf("receipt on unapproved PO=%d want=409 body=%s", code, string(b))
	}

	// Distinct OWNER approves ApprovalRequest + PO
	if code, _, b := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/approvals/"+poCreateRes.ApprovalRequestID+"/approve", ownerTok, map[string]any{
		"comment": "verified vendor quote",
	}, ""); code != http.StatusOK {
		t.Fatalf("OWNER approve request=%d body=%s", code, string(b))
	}
	if code, _, b := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/p2p/purchase-orders/"+poID+"/approve", ownerTok, map[string]any{
		"approval_id": poCreateRes.ApprovalRequestID,
	}, ""); code != http.StatusOK {
		t.Fatalf("OWNER approve PO=%d body=%s", code, string(b))
	}

	// WAREHOUSE_MANAGER posts Goods Receipt -> posts Inventory DR / GRNI CR to GL
	code, _, b = doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/p2p/receipts", whTok, map[string]any{
		"po_id":        poID,
		"entity_id":    entID,
		"warehouse_id": "wh-p2p-1",
		"period_id":    periodID,
		"lines":        []map[string]any{{"sku": "SKU-P2P-01", "qty_received": 10}},
	}, "gr-ok-"+tag)
	if code != http.StatusCreated {
		t.Fatalf("POST /v1/p2p/receipts=%d body=%s", code, string(b))
	}
	var grRes p2p.PostGoodsReceiptResult
	_ = json.Unmarshal(b, &grRes)
	if grRes.Receipt.ReceiptID == "" || grRes.JournalEntryID == "" {
		t.Fatalf("expected ReceiptID & JournalEntryID, got %+v", grRes)
	}

	// Seed Inbound ESF document matching 1,120,000 UZS (12% VAT = 120,000)
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("EsfDocuments", map[string]any{
			"EntityId":        tenantID,
			"EsfId":           esfID,
			"Direction":       "INBOUND",
			"CounterpartyTin": "309876543",
			"TotalMinor":      int64(1120000),
			"VatMinor":        int64(120000),
			"VatRateBps":      int64(1200),
			"Status":          "ACCEPTED",
			"Operator":        "didox",
			"PeriodId":        periodID,
			"CreatedAt":       spanner.CommitTimestamp,
			"UpdatedAt":       spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("insert EsfDocuments: %v", err)
	}

	// Discrepant invoice -> DISCREPANCY
	code, _, b = doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/p2p/vendor-invoices", accTok, map[string]any{
		"po_id":          poID,
		"receipt_id":     grRes.Receipt.ReceiptID,
		"entity_id":      entID,
		"subtotal_minor": 1400000,
		"vat_minor":      168000,
		"total_minor":    1568000,
	}, "vi-bad-"+tag)
	if code != http.StatusCreated {
		t.Fatalf("create bad vendor invoice=%d body=%s", code, string(b))
	}
	var badInvWrap struct {
		VendorInvoice p2p.VendorInvoice `json:"vendor_invoice"`
	}
	_ = json.Unmarshal(b, &badInvWrap)

	code, _, b = doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/p2p/three-way-match", ownerTok, map[string]any{
		"vendor_invoice_id": badInvWrap.VendorInvoice.VendorInvoiceID,
		"po_id":             poID,
		"receipt_id":        grRes.Receipt.ReceiptID,
		"period_id":         periodID,
	}, "")
	if code != http.StatusOK {
		t.Fatalf("3-way match discrepant=%d body=%s", code, string(b))
	}
	var badMatch p2p.ThreeWayMatchResult
	_ = json.Unmarshal(b, &badMatch)
	if badMatch.Matched || badMatch.MatchStatus != p2p.MatchStatusDiscrepancy {
		t.Fatalf("expected DISCREPANCY, got %+v", badMatch)
	}

	// Matching invoice with ESF -> MATCHED_3WAY & clears GRNI -> AP + VAT_INPUT GL
	code, _, b = doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/p2p/vendor-invoices", accTok, map[string]any{
		"po_id":          poID,
		"receipt_id":     grRes.Receipt.ReceiptID,
		"esf_doc_id":     esfID,
		"entity_id":      entID,
		"subtotal_minor": 1000000,
		"vat_minor":      120000,
		"total_minor":    1120000,
	}, "vi-ok-"+tag)
	if code != http.StatusCreated {
		t.Fatalf("create good vendor invoice=%d body=%s", code, string(b))
	}
	var okInvWrap struct {
		VendorInvoice p2p.VendorInvoice `json:"vendor_invoice"`
	}
	_ = json.Unmarshal(b, &okInvWrap)

	code, _, b = doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/p2p/three-way-match", ownerTok, map[string]any{
		"vendor_invoice_id": okInvWrap.VendorInvoice.VendorInvoiceID,
		"po_id":             poID,
		"receipt_id":        grRes.Receipt.ReceiptID,
		"esf_doc_id":        esfID,
		"period_id":         periodID,
	}, "")
	if code != http.StatusOK {
		t.Fatalf("3-way match ok=%d body=%s", code, string(b))
	}
	assertWave4AJSONNoFloatLiterals(t, b, "three_way_match")
	var okMatch p2p.ThreeWayMatchResult
	_ = json.Unmarshal(b, &okMatch)
	if !okMatch.Matched || okMatch.MatchStatus != p2p.MatchStatusMatched3Way || okMatch.JournalEntryID == "" {
		t.Fatalf("expected MATCHED_3WAY with GL journal, got %+v", okMatch)
	}

	// Process Mining (auditor readable, pure integer durations)
	code, _, b = doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/processes/mining?domain=P2P", auditorTok, nil, "")
	if code != http.StatusOK {
		t.Fatalf("GET /v1/processes/mining=%d body=%s", code, string(b))
	}
	assertWave4AJSONNoFloatLiterals(t, b, "process_mining")
	var report p2p.ProcessMiningReport
	if err := json.Unmarshal(b, &report); err != nil {
		t.Fatalf("decode ProcessMiningReport: %v", err)
	}
	if report.TotalCases < 1 || report.TotalTransitions < 1 {
		t.Fatalf("expected live process mining cases & transitions, got %+v", report)
	}
}

func TestWave4A_TwoWayConnectors_ApprovalGated_CISAdapters_And_BankPack_LiveSpanner(t *testing.T) {
	client := openWave4ASpannerClient(t)
	h := NewTestHarnessWithOptions(t, HarnessOptions{SpannerClient: client})
	ctx := context.Background()

	tag := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	tenantID := "sup-w4a-conn-" + tag
	ret1 := "ret1-w4a-" + tag
	ret2 := "ret2-w4a-" + tag
	inv1 := "inv1-w4a-" + tag
	inv2 := "inv2-w4a-" + tag
	le1 := "le1-w4a-" + tag
	le2 := "le2-w4a-" + tag

	t.Cleanup(func() {
		cleanupWave4ASpannerTenant(client, tenantID, nil, []string{ret1, ret2}, []string{inv1, inv2}, []string{le1, le2})
	})

	futureExp := time.Now().UTC().Add(2 * time.Hour)
	pastExp := time.Now().UTC().Add(-15 * time.Minute)
	ownerTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleOwner, tenantID, "owner-conn-"+tag, nil)
	accTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleAccountant, tenantID, "acc-conn-"+tag, nil)
	auditorTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleAuditor, tenantID, "aud-conn-"+tag, &futureExp)
	expiredAudTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleAuditor, tenantID, "exp-aud-conn-"+tag, &pastExp)
	agentTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleAgent, tenantID, "agent-conn-"+tag, nil)

	t.Run("ProductionTwoWayConnectorsAndCISAdapters", func(t *testing.T) {
		t.Setenv("PEGASUSX_ENV", "production")

		// 1. Spin up mock external HTTP servers for all 5 ERP/POS providers
		type capturedCall struct {
			path       string
			authHeader string
			approval   string
			body       string
		}
		var callsMu sync.Mutex
		captured := make(map[string][]capturedCall)
		recordCall := func(provider string, r *http.Request) string {
			b, _ := io.ReadAll(r.Body)
			callsMu.Lock()
			defer callsMu.Unlock()
			captured[provider] = append(captured[provider], capturedCall{
				path:       r.URL.Path,
				authHeader: r.Header.Get("Authorization"),
				approval:   r.Header.Get("X-Atomos-Approval-Id"),
				body:       string(b),
			})
			return string(b)
		}

		makeSrv := func(provider, expectedBearer, docID string) *httptest.Server {
			return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if expectedBearer != "" && r.Header.Get("Authorization") != expectedBearer {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				_ = recordCall(provider, r)
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/commerceml") || strings.Contains(r.Header.Get("Content-Type"), "xml") {
					w.Header().Set("Content-Type", "application/xml")
					_, _ = w.Write([]byte(`<CommerceInformation><Status>success</Status><DocId>` + docID + `</DocId></CommerceInformation>`))
					return
				}
				_, _ = w.Write([]byte(`{"status":"ok","doc_id":"` + docID + `","items":[]}`))
			}))
		}

		billzSrv := makeSrv(connectors.ProviderBillz, "Bearer tok-billz-"+tag, "BLZ-DOC-1")
		defer billzSrv.Close()
		onecSrv := makeSrv(connectors.ProviderOnec, "Bearer tok-onec-"+tag, "1C-DOC-1")
		defer onecSrv.Close()
		moySrv := makeSrv(connectors.ProviderMoySklad, "Bearer tok-moy-"+tag, "MS-DOC-1")
		defer moySrv.Close()
		smartSrv := makeSrv(connectors.ProviderSmartup, "Bearer tok-smart-"+tag, "SU-DOC-1")
		defer smartSrv.Close()
		iikoSrv := makeSrv(connectors.ProviderIiko, "Bearer tok-iiko-"+tag, "IIKO-DOC-1")
		defer iikoSrv.Close()
		h.ConnectorsSvc.SetHTTPClient(billzSrv.Client())

		providerSpecs := []struct {
			provider string
			srvURL   string
			envKey   string
			secret   string
			stream   string
			payload  map[string]any
		}{
			{
				provider: connectors.ProviderBillz,
				srvURL:   billzSrv.URL,
				envKey:   "CONNECTOR_SECRET_E2E_BILLZ_TOK_" + tag,
				secret:   "tok-billz-" + tag,
				stream:   connectors.StreamStocks,
				payload:  map[string]any{"warehouse_id": "wh-1", "reference_id": "GR-100", "lines": []map[string]any{{"sku": "SKU-1", "qty": 10}}},
			},
			{
				provider: connectors.ProviderOnec,
				srvURL:   onecSrv.URL,
				envKey:   "CONNECTOR_SECRET_E2E_ONEC_TOK_" + tag,
				secret:   "tok-onec-" + tag,
				stream:   "journals",
				payload:  map[string]any{"entry_id": "JE-100", "period_id": "2026-11", "debit_account": "INVENTORY", "credit_account": "GRNI", "amount_minor": 500000, "currency": "UZS"},
			},
			{
				provider: connectors.ProviderMoySklad,
				srvURL:   moySrv.URL,
				envKey:   "CONNECTOR_SECRET_E2E_MOYSKLAD_TOK_" + tag,
				secret:   "tok-moy-" + tag,
				stream:   connectors.StreamStocks,
				payload:  map[string]any{"supply_id": "SUP-1", "store_id": "wh-1", "positions": []map[string]any{{"sku": "SKU-MS-1", "quantity": 12}}},
			},
			{
				provider: connectors.ProviderSmartup,
				srvURL:   smartSrv.URL,
				envKey:   "CONNECTOR_SECRET_E2E_SMARTUP_TOK_" + tag,
				secret:   "tok-smart-" + tag,
				stream:   "goods_receipt",
				payload:  map[string]any{"receipt_id": "GR-SU-1", "warehouse_id": "wh-1", "amount_minor": 250000},
			},
			{
				provider: connectors.ProviderIiko,
				srvURL:   iikoSrv.URL,
				envKey:   "CONNECTOR_SECRET_E2E_IIKO_TOK_" + tag,
				secret:   "tok-iiko-" + tag,
				stream:   connectors.StreamOrders,
				payload:  map[string]any{"invoice_id": "IIKO-INV-1", "department_id": "dep-1", "total_minor": 320000},
			},
		}

		for _, ps := range providerSpecs {
			t.Setenv(ps.envKey, ps.secret)
			code, _, b := doWave4AHTTP(t, http.MethodPut, h.Server.URL+"/v1/connectors/credentials", ownerTok, map[string]any{
				"provider":      ps.provider,
				"account_label": "default",
				"auth_type":     "bearer",
				"secret_ref":    "env://" + ps.envKey,
				"base_url":      ps.srvURL,
			}, "")
			if code != http.StatusOK {
				t.Fatalf("PUT credentials(%s)=%d body=%s", ps.provider, code, string(b))
			}
		}

		// 2. Fail closed on AUDITOR/AGENT (403), expired AUDITOR (401), missing approval_id (403)
		for _, tok := range []string{auditorTok, agentTok} {
			code, _, b := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/connectors/billz/push", tok, map[string]any{
				"stream": "stocks", "approval_id": "fake", "raw_payload": providerSpecs[0].payload,
			}, "push-deny-"+tag)
			if code != http.StatusForbidden {
				t.Fatalf("AUDITOR/AGENT POST /v1/connectors/billz/push=%d want=403 body=%s", code, string(b))
			}
		}
		if code, _, b := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/connectors/runs", expiredAudTok, nil, ""); code != http.StatusUnauthorized || !strings.Contains(string(b), "auditor_access_expired") {
			t.Fatalf("expired AUDITOR GET /v1/connectors/runs=%d want=401 body=%s", code, string(b))
		}
		if code, _, b := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/connectors/billz/push", ownerTok, map[string]any{
			"stream": "stocks", "raw_payload": providerSpecs[0].payload,
		}, "push-noapr-"+tag); code != http.StatusForbidden {
			t.Fatalf("missing approval_id push=%d want=403 body=%s", code, string(b))
		}

		// 3. Approval payload tamper test on Billz, then valid approved 2-way pushes across all 5 providers
		submitAndApprove := func(provider, stream string, rawPayload any, suffix string) string {
			code, _, b := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/approvals/requests", accTok, map[string]any{
				"object_type":     "ConnectorPush",
				"object_id":       provider,
				"action":          "push",
				"force_create":    true,
				"idempotency_key": "apr-push-" + provider + "-" + suffix + "-" + tag,
				"payload": map[string]any{
					"provider":      provider,
					"account_label": "default",
					"stream":        stream,
					"raw_payload":   rawPayload,
				},
			}, "")
			if code != http.StatusCreated && code != http.StatusOK {
				t.Fatalf("submit ConnectorPush approval(%s)=%d body=%s", provider, code, string(b))
			}
			var subWrap struct {
				Request approvals.ApprovalRequest `json:"request"`
			}
			if err := json.Unmarshal(b, &subWrap); err != nil || subWrap.Request.RequestID == "" {
				t.Fatalf("decode approval request: err=%v body=%s", err, string(b))
			}
			if code, _, b := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/approvals/"+subWrap.Request.RequestID+"/approve", ownerTok, map[string]any{
				"comment": "approved connector write-back",
			}, ""); code != http.StatusOK {
				t.Fatalf("approve ConnectorPush(%s)=%d body=%s", provider, code, string(b))
			}
			return subWrap.Request.RequestID
		}

		billzAprID := submitAndApprove(connectors.ProviderBillz, connectors.StreamStocks, providerSpecs[0].payload, "ok")
		if code, _, b := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/connectors/billz/push", ownerTok, map[string]any{
			"account_label": "default",
			"stream":        connectors.StreamStocks,
			"approval_id":   billzAprID,
			"raw_payload":   map[string]any{"warehouse_id": "wh-1", "reference_id": "TAMPERED", "lines": []map[string]any{{"sku": "SKU-1", "qty": 999}}},
		}, "push-tamper-"+tag); code != http.StatusConflict && code != http.StatusForbidden {
			t.Fatalf("tampered payload push=%d want=403/409 body=%s", code, string(b))
		}

		for idx, ps := range providerSpecs {
			aprID := billzAprID
			if idx > 0 {
				aprID = submitAndApprove(ps.provider, ps.stream, ps.payload, "ok")
			}
			code, _, b := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/connectors/"+ps.provider+"/push", ownerTok, map[string]any{
				"account_label": "default",
				"stream":        ps.stream,
				"approval_id":   aprID,
				"raw_payload":   ps.payload,
			}, "push-ok-"+ps.provider+"-"+tag)
			if code != http.StatusOK {
				t.Fatalf("POST /v1/connectors/%s/push=%d body=%s", ps.provider, code, string(b))
			}
			var pushResp connectors.PushResponse
			if err := json.Unmarshal(b, &pushResp); err != nil {
				t.Fatalf("decode PushResponse(%s): %v", ps.provider, err)
			}
			if pushResp.Run.RunID == "" || pushResp.PushResult.Status != connectors.RunStatusCompleted || pushResp.PushResult.ExternalDocID == "" {
				t.Fatalf("unexpected PushResponse(%s): %+v", ps.provider, pushResp)
			}
		}

		// Verify connector.push.completed outbox events emitted on TopicConnectors for all 5 providers
		{
			ro := client.ReadOnlyTransaction()
			oIter := ro.Query(ctx, spanner.Statement{
				SQL:    `SELECT TopicName, Payload FROM OutboxEvents WHERE SupplierId = @tid`,
				Params: map[string]any{"tid": tenantID},
			})
			pushOutboxCount := 0
			for {
				r, err := oIter.Next()
				if err != nil {
					break
				}
				var topic string
				var pl []byte
				if err := r.Columns(&topic, &pl); err == nil {
					var hdr struct {
						Type string `json:"type"`
					}
					_ = json.Unmarshal(pl, &hdr)
					if topic == events.TopicConnectors && hdr.Type == events.EventConnectorPushCompleted {
						pushOutboxCount++
					}
				}
			}
			oIter.Stop()
			ro.Close()
			if pushOutboxCount < 5 {
				t.Fatalf("expected >=5 %s events on %s, got %d", events.EventConnectorPushCompleted, events.TopicConnectors, pushOutboxCount)
			}
		}

		// Verify 1C CommerceML XML push in addition to OData JSON
		xmlPayload := `<CommerceInformation><Catalog><Product><Id>SKU-1C-1</Id></Product></Catalog></CommerceInformation>`
		xmlAprID := submitAndApprove(connectors.ProviderOnec, "catalog", xmlPayload, "xml")
		if code, _, b := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/connectors/onec/push", ownerTok, map[string]any{
			"account_label": "default",
			"stream":        "catalog",
			"approval_id":   xmlAprID,
			"raw_payload":   xmlPayload,
		}, "push-1c-xml-"+tag); code != http.StatusOK {
			t.Fatalf("POST /v1/connectors/onec/push(CommerceML XML)=%d body=%s", code, string(b))
		}

		callsMu.Lock()
		for _, ps := range providerSpecs {
			if len(captured[ps.provider]) == 0 || captured[ps.provider][0].approval == "" {
				t.Fatalf("provider %s did not receive real HTTP call with X-Atomos-Approval-Id: %+v", ps.provider, captured[ps.provider])
			}
		}
		callsMu.Unlock()
	})

	// 4. §6.8 Bank Credit Pack: seed RetailerPaymentTerms/CreditProfiles/ArInvoices/ArLedgerEntries/CreditScoreHistory
	now := time.Now().UTC().Add(-time.Hour)
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("RetailerPaymentTerms", map[string]any{
			"RetailerId": ret1, "SupplierId": tenantID, "CreditEnabled": true, "TermsDays": int64(30),
			"GracePeriodDays": int64(0), "CreditLimitMinor": int64(2000000), "UseGlobalDefaults": false,
			"ConsentVerified": true, "Version": int64(1), "UpdatedAt": now,
		}),
		spanner.InsertMap("RetailerPaymentTerms", map[string]any{
			"RetailerId": ret2, "SupplierId": tenantID, "CreditEnabled": true, "TermsDays": int64(15),
			"GracePeriodDays": int64(0), "CreditLimitMinor": int64(1000000), "UseGlobalDefaults": false,
			"ConsentVerified": false, "Version": int64(1), "UpdatedAt": now,
		}),
		spanner.InsertMap("RetailerCreditProfiles", map[string]any{
			"RetailerId": ret1, "SupplierId": tenantID, "CreditLimitMinor": int64(2000000),
			"CurrentBalanceMinor": int64(750000), "ReservedMinor": int64(0), "AvailableCreditMinor": int64(1250000),
			"RiskScore": int64(85), "DelinquencyCount": int64(0), "Status": "ACTIVE",
			"PrepayOnlyLocked": false, "Version": int64(1), "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("RetailerCreditProfiles", map[string]any{
			"RetailerId": ret2, "SupplierId": tenantID, "CreditLimitMinor": int64(1000000),
			"CurrentBalanceMinor": int64(250000), "ReservedMinor": int64(0), "AvailableCreditMinor": int64(750000),
			"RiskScore": int64(42), "DelinquencyCount": int64(3), "Status": "ACTIVE",
			"PrepayOnlyLocked": true, "Version": int64(1), "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("ArInvoices", map[string]any{
			"InvoiceId": inv1, "SupplierId": tenantID, "RetailerId": ret1, "OrderId": "ord1-" + tag,
			"Status": "PARTIAL", "PrincipalMinor": int64(1000000), "BalanceMinor": int64(750000),
			"Currency": "UZS", "CreditLeaveAt": now.AddDate(0, -1, 0), "DueAt": now.Add(-12 * 24 * time.Hour),
			"TermsDays": int64(30), "GracePeriodDays": int64(0), "CreatedAt": now.AddDate(0, -1, 0), "UpdatedAt": now,
		}),
		spanner.InsertMap("ArInvoices", map[string]any{
			"InvoiceId": inv2, "SupplierId": tenantID, "RetailerId": ret2, "OrderId": "ord2-" + tag,
			"Status": "OPEN", "PrincipalMinor": int64(250000), "BalanceMinor": int64(250000),
			"Currency": "UZS", "CreditLeaveAt": now.AddDate(0, -4, 0), "DueAt": now.Add(-95 * 24 * time.Hour),
			"TermsDays": int64(15), "GracePeriodDays": int64(0), "CreatedAt": now.AddDate(0, -4, 0), "UpdatedAt": now,
		}),
		spanner.InsertMap("ArLedgerEntries", map[string]any{
			"EntryId": le1, "InvoiceId": inv1, "SupplierId": tenantID, "RetailerId": ret1,
			"EntryType": "PAYMENT", "AmountMinor": int64(-250000), "IdempotencyKey": "ik-le1-" + tag, "CreatedAt": now,
		}),
		spanner.InsertMap("ArLedgerEntries", map[string]any{
			"EntryId": le2, "InvoiceId": inv2, "SupplierId": tenantID, "RetailerId": ret2,
			"EntryType": "OPEN", "AmountMinor": int64(250000), "IdempotencyKey": "ik-le2-" + tag, "CreatedAt": now.AddDate(0, -4, 0),
		}),
		spanner.InsertMap("CreditScoreHistory", map[string]any{
			"RetailerId": ret1, "ScoreId": "rel-" + tag, "Scope": "RELATIONSHIP", "SupplierId": tenantID,
			"ConsentVerified": true, "ModelVersion": "v2", "ScorerSource": "SCORECARD",
			"Score": int64(86), "PdBps": int64(140), "RiskTier": "LOW",
			"SuggestedLimitMinor": int64(2200000), "CurrentLimitMinor": int64(2000000),
			"ReasonCodesJson": "[]", "FeaturesHash": "fh1", "DisputesIsolatedCount": int64(0),
			"WindowStart": now.AddDate(0, -3, 0), "WindowEnd": now, "ComputedAt": now, "CreatedAt": now,
		}),
		spanner.InsertMap("CreditScoreHistory", map[string]any{
			"RetailerId": ret1, "ScoreId": "net-" + tag, "Scope": "NETWORK", "SupplierId": nil,
			"ConsentVerified": true, "ModelVersion": "v2", "ScorerSource": "SCORECARD",
			"Score": int64(91), "PdBps": int64(90), "RiskTier": "LOW",
			"SuggestedLimitMinor": int64(2500000), "CurrentLimitMinor": int64(2000000),
			"ReasonCodesJson": "[]", "FeaturesHash": "fh2", "DisputesIsolatedCount": int64(0),
			"WindowStart": now.AddDate(0, -3, 0), "WindowEnd": now, "ComputedAt": now, "CreatedAt": now,
		}),
	}); err != nil {
		t.Fatalf("seed Bank pack data: %v", err)
	}

	code, _, b := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/finance/packs", ownerTok, map[string]any{
		"pack_type": "BANK",
		"entity_id": tenantID,
		"period_id": time.Now().UTC().Format("2006-01"),
		"format":    "zip",
		"ttl_hours": int64(24),
	}, "")
	if code != http.StatusCreated {
		t.Fatalf("POST /v1/finance/packs(BANK)=%d body=%s", code, string(b))
	}
	var pack exportpacks.ExportPack
	if err := json.Unmarshal(b, &pack); err != nil {
		t.Fatalf("decode BANK ExportPack: %v", err)
	}

	// Active AUDITOR downloads ZIP/XLSX/CSV/JSON formats
	zipCode, _, zipBytes := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/finance/packs/"+pack.PackID+"/download?format=zip&token="+pack.DownloadToken, auditorTok, nil, "")
	if zipCode != http.StatusOK {
		t.Fatalf("download BANK zip=%d body=%s", zipCode, string(zipBytes))
	}
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	wantEntries := map[string]bool{
		"manifest.json":                false,
		"csv/bank_credit_tape.csv":     false,
		"csv/dpd_buckets.csv":          false,
		"csv/debtor_concentration.csv": false,
		"csv/loss_recovery_curves.csv": false,
	}
	for _, zf := range zr.File {
		if _, ok := wantEntries[zf.Name]; ok {
			wantEntries[zf.Name] = true
		}
	}
	for name, seen := range wantEntries {
		if !seen {
			t.Fatalf("BANK zip missing %q", name)
		}
	}

	xlsxCode, _, xlsxBytes := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/finance/packs/"+pack.PackID+"/download?format=xlsx&token="+pack.DownloadToken, auditorTok, nil, "")
	if xlsxCode != http.StatusOK {
		t.Fatalf("download BANK xlsx=%d", xlsxCode)
	}
	xl, err := excelize.OpenReader(bytes.NewReader(xlsxBytes))
	if err != nil {
		t.Fatalf("excelize.OpenReader: %v", err)
	}
	for _, sh := range []string{"Bank_Credit_Tape", "DPD_Buckets", "Debtor_Concentration", "Loss_LossRecovery_Curves"} {
		rows, rErr := xl.GetRows(sh)
		if rErr != nil || len(rows) < 2 {
			t.Fatalf("BANK xlsx sheet %s invalid: rows=%d err=%v", sh, len(rows), rErr)
		}
	}
	_ = xl.Close()

	csvCode, _, csvBytes := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/finance/packs/"+pack.PackID+"/download?format=csv&token="+pack.DownloadToken, auditorTok, nil, "")
	if csvCode != http.StatusOK {
		t.Fatalf("download BANK csv=%d", csvCode)
	}
	csvRows, err := csv.NewReader(bytes.NewReader(csvBytes)).ReadAll()
	if err != nil || len(csvRows) < 2 {
		t.Fatalf("invalid BANK csv: rows=%d err=%v", len(csvRows), err)
	}

	jsonCode, _, jsonBytes := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/finance/packs/"+pack.PackID+"/download?format=json&token="+pack.DownloadToken, auditorTok, nil, "")
	if jsonCode != http.StatusOK {
		t.Fatalf("download BANK json=%d", jsonCode)
	}
	assertWave4AJSONNoFloatLiterals(t, jsonBytes, "bank_pack_manifest")
	var manifest exportpacks.PackManifest
	if err := json.Unmarshal(jsonBytes, &manifest); err != nil {
		t.Fatalf("decode BANK PackManifest: %v", err)
	}
	if manifest.BankCredit == nil || manifest.BankCredit.TotalExposureMinor != 1000000 || manifest.BankCredit.Top1ConcentrationBps != 7500 || manifest.BankCredit.ConsentVerifiedRetailers != 1 || len(manifest.BankCredit.DPDBuckets) < 4 {
		t.Fatalf("unexpected BankCredit summary: %+v", manifest.BankCredit)
	}
}
