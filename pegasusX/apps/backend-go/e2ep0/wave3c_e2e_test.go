package e2ep0

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/connectors"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/exportpacks"
	"github.com/pegasusx/pegasusx/apps/backend-go/iam"
	"github.com/pegasusx/pegasusx/apps/backend-go/ontology"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplierroutes"
	"github.com/xuri/excelize/v2"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openWave3CSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1" {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping Wave 3C live Spanner E2E test")
	}
	project := envOrDefault("SPANNER_PROJECT", "pegasusx-local")
	instance := envOrDefault("SPANNER_INSTANCE", "pegasusx-instance")
	database := envOrDefault("SPANNER_DATABASE", "pegasusx-db")
	dsn := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dsn,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("open spanner client: %v", err)
	}
	for _, tbl := range []string{"ConnectorSyncRuns", "ExportPacks", "SupplierUserRoles"} {
		ro := client.ReadOnlyTransaction()
		probeKey := spanner.Key{"__w3c_probe__", "__probe__"}
		if tbl == "SupplierUserRoles" {
			probeKey = spanner.Key{"__w3c_probe__", "__probe__", "__role__"}
		}
		_, err := ro.ReadRow(ctx, tbl, probeKey, []string{firstProbeCol(tbl)})
		ro.Close()
		if err != nil && spanner.ErrCode(err) != codes.NotFound {
			client.Close()
			t.Fatalf("required Wave 3C table %s unavailable: %v", tbl, err)
		}
	}
	t.Cleanup(client.Close)
	return client
}

func firstProbeCol(table string) string {
	switch table {
	case "ConnectorSyncRuns":
		return "Status"
	case "ExportPacks":
		return "Status"
	case "SupplierUserRoles":
		return "RoleName"
	default:
		return "CreatedAt"
	}
}

func envOrDefault(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func issueWave3CSupplierToken(t *testing.T, h *TestHarness, supID, subject string, supRole auth.Role, actorType string, audExp *time.Time, perms []string) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:          subject,
		Role:             auth.RoleAdmin,
		SupplierID:       supID,
		SupplierRole:     supRole,
		ActorType:        actorType,
		SupplierPerms:    perms,
		AuditorExpiresAt: audExp,
	}, auth.IssueOptions{
		Secret: h.JWTSecret,
		Issuer: h.JWTIssuer,
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatalf("issue token (%s): %v", subject, err)
	}
	return tok
}

func doWave3CHTTP(t *testing.T, method, url, bearer string, body any, idempotencyKey string) (int, http.Header, []byte) {
	t.Helper()
	var rdr io.Reader
	hasBody := false
	if body != nil {
		hasBody = true
		switch v := body.(type) {
		case []byte:
			rdr = bytes.NewReader(v)
		case string:
			rdr = strings.NewReader(v)
		default:
			b, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("marshal request body: %v", err)
			}
			rdr = bytes.NewReader(b)
		}
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		t.Fatalf("new request %s %s: %v", method, url, err)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("do request %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response %s %s: %v", method, url, err)
	}
	return resp.StatusCode, resp.Header, raw
}

func decodeJSONInt64Field(t *testing.T, raw []byte, keys ...string) int64 {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var cur any
	if err := dec.Decode(&cur); err != nil {
		t.Fatalf("decode json: %v body=%s", err, string(raw))
	}
	for _, k := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("expected object at key %s, got %T", k, cur)
		}
		cur = m[k]
	}
	num, ok := cur.(json.Number)
	if !ok {
		t.Fatalf("expected json.Number at %v, got %T (%v)", keys, cur, cur)
	}
	v, err := num.Int64()
	if err != nil {
		t.Fatalf("parse int64 at %v: %v", keys, err)
	}
	return v
}

func buildWave3CMultiSheetXLSX10500(t *testing.T, whID string) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheets := []struct {
		name  string
		start int
		count int
	}{
		{name: "Sheet1", start: 1, count: 5500},
		{name: "StockBatch2", start: 5501, count: 5000},
	}
	for idx, s := range sheets {
		if idx > 0 {
			if _, err := f.NewSheet(s.name); err != nil {
				t.Fatalf("NewSheet %s: %v", s.name, err)
			}
		}
		sw, err := f.NewStreamWriter(s.name)
		if err != nil {
			t.Fatalf("NewStreamWriter %s: %v", s.name, err)
		}
		if err := sw.SetRow("A1", []any{"sku", "warehouse_id", "quantity", "reserved"}); err != nil {
			t.Fatalf("SetRow header %s: %v", s.name, err)
		}
		for i := 0; i < s.count; i++ {
			n := s.start + i
			cell, err := excelize.CoordinatesToCellName(1, i+2)
			if err != nil {
				t.Fatalf("CoordinatesToCellName: %v", err)
			}
			row := []any{
				fmt.Sprintf("SKU-W3C-%05d", n),
				whID,
				int64(10 + (n % 50)),
				int64(1),
			}
			if err := sw.SetRow(cell, row); err != nil {
				t.Fatalf("SetRow %s: %v", s.name, err)
			}
		}
		if err := sw.Flush(); err != nil {
			t.Fatalf("Flush %s: %v", s.name, err)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("Write xlsx: %v", err)
	}
	return buf.Bytes()
}

func buildWave3CEncryptedOLE2Container() []byte {
	// OLE2 CDF magic header + UTF-16LE "EncryptedPackage" stream directory marker.
	buf := make([]byte, 1024)
	copy(buf[0:8], []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1})
	marker := []byte{
		'E', 0, 'n', 0, 'c', 0, 'r', 0, 'y', 0, 'p', 0, 't', 0, 'e', 0,
		'd', 0, 'P', 0, 'a', 0, 'c', 0, 'k', 0, 'a', 0, 'g', 0, 'e', 0,
	}
	copy(buf[512:], marker)
	return buf
}

func TestWave3C_FullStack_Connectors_ExportPacks_AuditorGrants_Excel_LiveSpanner(t *testing.T) {
	client := openWave3CSpannerClient(t)
	ctx := context.Background()

	tag := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	tenantID := "sup-w3c-" + tag
	whID := "wh-w3c-" + tag
	prod1 := "prod-w3c-1-" + tag
	prod2 := "prod-w3c-2-" + tag
	ord1 := "ord-w3c-" + tag
	inv1 := "inv-w3c-" + tag
	ret1 := "ret-w3c-" + tag
	je1 := "je-w3c-" + tag
	extAudUser := "ext-aud-w3c-" + tag

	t.Setenv("ENFORCE_TENANT_CONTEXT", "1")
	t.Setenv("CSRF_ENFORCE", "0")
	t.Setenv("EXPORT_PACKS_LOCAL_ROOT", t.TempDir())
	t.Setenv("PEGASUSX_ENV", "")

	now := time.Now().UTC().Add(-30 * time.Minute)
	props1, _ := json.Marshal(map[string]any{
		"product_id":  prod1,
		"sku":         prod1,
		"name":        "Silk Tea 500g",
		"price_minor": int64(1800000),
		"currency":    "UZS",
	})

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("Warehouses", map[string]any{
			"WarehouseId": whID, "SupplierId": tenantID, "Name": "Wave3C Hub", "CoverageRadiusKm": 25.0,
			"IsActive": true, "IsOnShift": true, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId": prod1, "SupplierId": tenantID, "CategoryId": "cat-tea",
			"Name": "Silk Tea 500g", "Barcode": "BC-W3C-" + tag, "PriceMinor": int64(1800000), "Currency": "UZS",
			"StockQuantity": int64(20), "IsActive": true, "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId": prod2, "SupplierId": tenantID, "CategoryId": "cat-tea",
			"Name": "Internal Only Tea", "Barcode": "BC-INT-" + tag, "PriceMinor": int64(900000), "Currency": "UZS",
			"StockQuantity": int64(5), "IsActive": true, "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId": tenantID, "WarehouseId": whID, "ProductId": prod1,
			"QuantityOnHand": int64(20), "QuantityReserved": int64(0), "ReorderThreshold": int64(2),
			"UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("Orders", map[string]any{
			"OrderId": ord1, "SupplierId": tenantID, "RetailerId": ret1,
			"Status": "COMPLETED", "OrderSource": "DIRECT", "ConfirmationStatus": "CONFIRMED",
			"LineItemsJson": []byte("[]"), "TotalMinor": int64(450000), "OriginalTotalMinor": int64(450000),
			"Currency": "UZS", "Version": int64(1), "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("ArInvoices", map[string]any{
			"InvoiceId": inv1, "SupplierId": tenantID, "RetailerId": ret1, "OrderId": ord1,
			"Status": "OPEN", "PrincipalMinor": int64(450000), "BalanceMinor": int64(450000),
			"Currency": "UZS", "CreditLeaveAt": now, "DueAt": now.Add(7 * 24 * time.Hour),
			"TermsDays": int64(7), "GracePeriodDays": int64(0), "DunningStep": int64(0),
			"Version": int64(1), "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("GlAccounts", map[string]any{
			"EntityId": tenantID, "AccountCode": "1010", "Class": "ASSET", "Name": "Cash", "Currency": "UZS",
			"CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("JournalEntries", map[string]any{
			"EntityId": tenantID, "EntryId": je1, "PeriodId": "2026-10", "SourceType": "ORDER", "SourceId": ord1,
			"Currency": "UZS", "IdempotencyKey": "ik-w3c-" + tag, "PrevHash": "0", "EntryHash": "h-w3c-" + tag,
			"PostedBy": "acct", "OccurredAt": now, "PostedAt": now,
		}),
		spanner.InsertMap("JournalPostings", map[string]any{
			"EntityId": tenantID, "EntryId": je1, "Line": int64(1), "AccountCode": "1010",
			"Direction": "DEBIT", "AmountMinor": int64(450000), "Currency": "UZS", "FunctionalAmountMinor": int64(450000),
		}),
		spanner.InsertOrUpdateMap("OntologyObjects", map[string]any{
			"CellId": ontology.DefaultCellID, "ObjectType": ontology.TypeSKU, "ObjectId": prod1,
			"SupplierId": tenantID, "Version": int64(1), "PropertiesJson": string(props1),
			"SecurityMarkings": []string{ontology.MarkingFinanceOnly}, "ValidFrom": now, "UpdatedAt": spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed Wave 3C Spanner data: %v", err)
	}

	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var outboxMuts []*spanner.Mutation
		ro := client.ReadOnlyTransaction()
		it := ro.Query(cleanCtx, spanner.Statement{
			SQL:    `SELECT EventId FROM OutboxEvents WHERE SupplierId = @tid OR AggregateId IN (@p1)`,
			Params: map[string]any{"tid": tenantID, "p1": prod1},
		})
		for {
			r, err := it.Next()
			if err == iterator.Done || err != nil {
				break
			}
			var eid string
			if r.Columns(&eid) == nil {
				outboxMuts = append(outboxMuts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
			}
		}
		it.Stop()
		ro.Close()

		muts := append(outboxMuts,
			spanner.Delete("ConnectorEntityMaps", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ConnectorSyncRuns", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ExternalCredentials", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ExportPacks", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("SupplierUserRoles", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID + "~"}}),
			spanner.Delete("OntologyObjects", spanner.Key{ontology.DefaultCellID, ontology.TypeSKU, prod1}),
			spanner.Delete("ArInvoices", spanner.Key{inv1}),
			spanner.Delete("Orders", spanner.Key{ord1}),
			spanner.Delete("SupplierInventoryV2", spanner.KeyRange{Start: spanner.Key{tenantID, whID}, End: spanner.Key{tenantID, whID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("Products", spanner.Key{prod1}),
			spanner.Delete("Products", spanner.Key{prod2}),
			spanner.Delete("Warehouses", spanner.Key{whID}),
			spanner.Delete("SupplierImportStagedRows", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("SupplierImportMapping", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("SupplierImportSessions", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
		)
		_, _ = client.Apply(cleanCtx, muts)
	})

	h := NewTestHarnessWithOptions(t, HarnessOptions{SpannerClient: client})
	srv := httptest.NewServer(h.Router)
	defer srv.Close()

	ownerTok := issueWave3CSupplierToken(t, h, tenantID, "owner-w3c-"+tag, auth.SupplierRoleOwner, auth.ActorTypeHuman, nil, nil)
	agentTok := issueWave3CSupplierToken(t, h, tenantID, "agent-w3c-"+tag, auth.SupplierRoleAgent, auth.ActorTypeAgent, nil, nil)
	auditorStaticTok := issueWave3CSupplierToken(t, h, tenantID, "aud-static-"+tag, auth.SupplierRoleAuditor, auth.ActorTypeHuman, nil, []string{auth.PermAuditRead})

	// 1. Fail-closed in PEGASUSX_ENV=production without configured external API token/base_url.
	t.Run("FailClosedProductionConnector", func(t *testing.T) {
		t.Setenv("PEGASUSX_ENV", "production")
		t.Setenv("BILLZ_API_BASE_URL", "")
		t.Setenv("BILLZ_API_TOKEN", "")
		code, _, body := doWave3CHTTP(t, http.MethodPost, srv.URL+"/v1/connectors/billz/health", ownerTok, map[string]any{}, "")
		if code != http.StatusServiceUnavailable {
			t.Fatalf("unconfigured prod billz health status=%d want=503 body=%s", code, string(body))
		}
	})

	// 2. RBAC 403 guards for AUDITOR and AGENT on mutations.
	for _, tok := range []string{agentTok, auditorStaticTok} {
		if code, _, body := doWave3CHTTP(t, http.MethodPut, srv.URL+"/v1/connectors/credentials", tok, map[string]any{
			"provider":   "billz",
			"auth_type":  "bearer",
			"secret_ref": "env://NONE",
			"base_url":   "http://127.0.0.1:1",
		}, ""); code != http.StatusForbidden {
			t.Fatalf("PUT credentials status=%d want=403 body=%s", code, string(body))
		}
		if code, _, _ := doWave3CHTTP(t, http.MethodPost, srv.URL+"/v1/connectors/billz/shadow-compare", tok, map[string]any{}, ""); code != http.StatusForbidden {
			t.Fatalf("POST shadow-compare status=%d want=403", code)
		}
		if code, _, _ := doWave3CHTTP(t, http.MethodPost, srv.URL+"/v1/finance/packs", tok, map[string]any{"pack_type": "AUDIT", "entity_id": tenantID}, ""); code != http.StatusForbidden {
			t.Fatalf("POST /v1/finance/packs status=%d want=403", code)
		}
	}

	// 3. Billz mock server -> credential upsert -> health -> pull -> shadow-compare (idempotent) + live Spanner/Outbox/Audit/Ontology check.
	billzSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer w3c-billz-secret-tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/shops":
			_, _ = w.Write([]byte(`{"shops":[{"id":"` + whID + `","name":"Wave3C Hub"}]}`))
		case "/v1/products":
			_, _ = w.Write([]byte(`{"products":[{"id":"ext-` + prod1 + `","sku":"` + prod1 + `","barcode":"BC-W3C-` + tag + `","name":"Silk Tea 500g","price_minor":1800000,"currency":"UZS"},{"id":"ext-billz-only-` + tag + `","sku":"BILLZ-ONLY-` + tag + `","name":"Billz Only Tea","price_minor":950000,"currency":"UZS"}]}`))
		case "/v1/stocks":
			_, _ = w.Write([]byte(`{"stocks":[{"product_id":"ext-` + prod1 + `","warehouse_id":"` + whID + `","quantity_on_hand":35}]}`))
		case "/v1/orders":
			_, _ = w.Write([]byte(`{"orders":[{"id":"` + ord1 + `","order_id":"` + ord1 + `","client_id":"` + ret1 + `","total_minor":450000,"currency":"UZS"}]}`))
		case "/v1/clients":
			_, _ = w.Write([]byte(`{"clients":[{"id":"` + ret1 + `","name":"Shop","balance_minor":600000,"currency":"UZS"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer billzSrv.Close()
	h.ConnectorsSvc.SetHTTPClient(billzSrv.Client())

	billzEnvKey := "CONNECTOR_SECRET_W3C_E2E_BILLZ_TOKEN_" + tag
	t.Setenv(billzEnvKey, "w3c-billz-secret-tok")

	code, _, credBody := doWave3CHTTP(t, http.MethodPut, srv.URL+"/v1/connectors/credentials", ownerTok, map[string]any{
		"provider":      "billz",
		"account_label": "primary",
		"auth_type":     "bearer",
		"secret_ref":    "env://" + billzEnvKey,
		"base_url":      billzSrv.URL,
	}, "")
	if code != http.StatusOK {
		t.Fatalf("OWNER PUT credentials=%d body=%s", code, string(credBody))
	}
	if strings.Contains(string(credBody), "w3c-billz-secret-tok") || strings.Contains(string(credBody), billzEnvKey) {
		t.Fatalf("PUT credentials leaked secret: %s", string(credBody))
	}

	if code, _, b := doWave3CHTTP(t, http.MethodPost, srv.URL+"/v1/connectors/billz/health", ownerTok, map[string]any{"account_label": "primary"}, ""); code != http.StatusOK {
		t.Fatalf("Billz health=%d body=%s", code, string(b))
	}
	if code, _, b := doWave3CHTTP(t, http.MethodPost, srv.URL+"/v1/connectors/billz/pull", ownerTok, map[string]any{"account_label": "primary", "stream": "products"}, "w3c-pull-"+tag); code != http.StatusOK {
		t.Fatalf("Billz pull=%d body=%s", code, string(b))
	}

	scIdem := "w3c-sc-" + tag
	code, _, scBody1 := doWave3CHTTP(t, http.MethodPost, srv.URL+"/v1/connectors/billz/shadow-compare", ownerTok, map[string]any{"account_label": "primary"}, scIdem)
	if code != http.StatusOK {
		t.Fatalf("Billz shadow-compare=%d body=%s", code, string(scBody1))
	}
	var scRes1 connectors.ShadowCompareResult
	if err := json.Unmarshal(scBody1, &scRes1); err != nil {
		t.Fatalf("decode shadow-compare: %v", err)
	}
	if scRes1.Summary.NetStockDelta != 15 || scRes1.Summary.NetBalanceDeltaMinor != 150000 || scRes1.Summary.MissingExternal != 1 || scRes1.Summary.MissingInternal != 1 || scRes1.Run.RunID == "" {
		t.Fatalf("unexpected shadow-compare summary: %+v", scRes1.Summary)
	}
	// Replay with identical Idempotency-Key returns same RunID without duplicating ConnectorSyncRuns.
	code, _, scBody2 := doWave3CHTTP(t, http.MethodPost, srv.URL+"/v1/connectors/billz/shadow-compare", ownerTok, map[string]any{"account_label": "primary"}, scIdem)
	if code != http.StatusOK {
		t.Fatalf("Billz shadow-compare replay=%d body=%s", code, string(scBody2))
	}
	var scRes2 connectors.ShadowCompareResult
	_ = json.Unmarshal(scBody2, &scRes2)
	if scRes2.Run.RunID != scRes1.Run.RunID {
		t.Fatalf("idempotency mismatch: %q != %q", scRes2.Run.RunID, scRes1.Run.RunID)
	}

	// Verify Spanner OutboxEvents (TopicConnectors) and OntologyObjects source_system lineage.
	{
		ro := client.ReadOnlyTransaction()
		oIter := ro.Query(ctx, spanner.Statement{
			SQL:    `SELECT TopicName, Payload FROM OutboxEvents WHERE SupplierId = @tid AND AggregateId = @rid`,
			Params: map[string]any{"tid": tenantID, "rid": scRes1.Run.RunID},
		})
		seenMismatch := false
		for {
			oRow, oErr := oIter.Next()
			if oErr == iterator.Done {
				break
			}
			if oErr != nil {
				oIter.Stop()
				ro.Close()
				t.Fatalf("query OutboxEvents for connector run %s: %v", scRes1.Run.RunID, oErr)
			}
			var top string
			var pl []byte
			if err := oRow.Columns(&top, &pl); err != nil {
				oIter.Stop()
				ro.Close()
				t.Fatalf("scan OutboxEvents row: %v", err)
			}
			var hdr struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(pl, &hdr)
			if top == events.TopicConnectors && hdr.Type == events.EventConnectorShadowMismatch {
				seenMismatch = true
			}
		}
		oIter.Stop()
		ro.Close()
		if !seenMismatch {
			t.Fatalf("expected OutboxEvents row on %s with type %s for run %s", events.TopicConnectors, events.EventConnectorShadowMismatch, scRes1.Run.RunID)
		}

		if h.OntologySvc != nil {
			ontObj, _, err := h.OntologySvc.GetObject(ctx, ontology.DefaultCellID, ontology.TypeSKU, prod1, "ADMIN")
			if err != nil {
				t.Fatalf("Ontology GetObject(%s): %v", prod1, err)
			}
			if ontObj.Properties["source_system"] != "billz" || ontObj.Properties["external_id"] != "ext-"+prod1 {
				t.Fatalf("Ontology lineage not updated by shadow-compare: %+v", ontObj.Properties)
			}
		}
	}

	// 4. Multi-account 1C CommerceML credential + inline CommerceML XML shadow-compare.
	onecEnvKey := "CONNECTOR_SECRET_W3C_E2E_ONEC_TOKEN_" + tag
	t.Setenv(onecEnvKey, "w3c-onec-secret-tok")
	if code, _, b := doWave3CHTTP(t, http.MethodPut, srv.URL+"/v1/connectors/credentials", ownerTok, map[string]any{
		"provider":      "onec",
		"account_label": "branch-uz-2",
		"auth_type":     "basic",
		"secret_ref":    "env://" + onecEnvKey,
		"base_url":      "https://1c.example.test",
		"metadata":      map[string]any{"exchange_mode": "commerceml"},
	}, ""); code != http.StatusOK {
		t.Fatalf("1C PUT credentials=%d body=%s", code, string(b))
	}
	commerceML := `<?xml version="1.0" encoding="UTF-8"?>
<КоммерческаяИнформация ВерсияСхемы="2.08">
  <Каталог><Товары>
    <Товар><Ид>` + prod1 + `</Ид><Артикул>` + prod1 + `</Артикул><Наименование>Silk Tea 500g</Наименование></Товар>
    <Товар><Ид>1c-only-sku</Ид><Артикул>1C-ONLY</Артикул><Наименование>Extra 1C Tea</Наименование></Товар>
  </Товары></Каталог>
</КоммерческаяИнформация>`
	code, _, onecBody := doWave3CHTTP(t, http.MethodPost, srv.URL+"/v1/connectors/onec/shadow-compare", ownerTok, map[string]any{
		"account_label": "branch-uz-2",
		"raw_xml":       commerceML,
	}, "w3c-onec-"+tag)
	if code != http.StatusOK {
		t.Fatalf("1C shadow-compare=%d body=%s", code, string(onecBody))
	}
	var onecRes connectors.ShadowCompareResult
	_ = json.Unmarshal(onecBody, &onecRes)
	if onecRes.Run.AccountLabel != "branch-uz-2" || onecRes.Summary.MissingExternal < 1 || onecRes.Summary.MissingInternal < 1 {
		t.Fatalf("unexpected 1C shadow result: %+v", onecRes)
	}

	// 5. ExportPacks: create AUDIT zip pack -> verify OutboxEvents (TopicLedger) -> download with headers/ZIP -> 403 bad token -> 410 expired.
	code, _, packBody := doWave3CHTTP(t, http.MethodPost, srv.URL+"/v1/finance/packs", ownerTok, map[string]any{
		"pack_type": "AUDIT",
		"entity_id": tenantID,
		"period_id": "2026-10",
		"format":    "zip",
		"ttl_hours": int64(24),
	}, "")
	if code != http.StatusCreated {
		t.Fatalf("POST /v1/finance/packs=%d want=201 body=%s", code, string(packBody))
	}
	var pack exportpacks.ExportPack
	if err := json.Unmarshal(packBody, &pack); err != nil {
		t.Fatalf("decode ExportPack: %v", err)
	}
	if pack.PackID == "" || pack.DownloadToken == "" || pack.Sha256Hex == "" || pack.WatermarkText == "" {
		t.Fatalf("incomplete ExportPack metadata: %+v", pack)
	}

	{
		ro := client.ReadOnlyTransaction()
		oIter := ro.Query(ctx, spanner.Statement{
			SQL:    `SELECT TopicName, Payload FROM OutboxEvents WHERE SupplierId = @tid AND AggregateId = @pid`,
			Params: map[string]any{"tid": tenantID, "pid": pack.PackID},
		})
		oRow, oErr := oIter.Next()
		oIter.Stop()
		ro.Close()
		if oErr != nil {
			t.Fatalf("query OutboxEvents for export pack %s: %v", pack.PackID, oErr)
		}
		var top string
		var pl []byte
		if err := oRow.Columns(&top, &pl); err != nil {
			t.Fatalf("scan export pack outbox row: %v", err)
		}
		var hdr struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(pl, &hdr)
		if top != events.TopicLedger || hdr.Type != events.EventExportPackCreated {
			t.Fatalf("unexpected export pack outbox: topic=%s type=%s", top, hdr.Type)
		}
	}

	if code, _, _ := doWave3CHTTP(t, http.MethodGet, srv.URL+"/v1/finance/packs/"+pack.PackID+"/download?format=zip&token=invalid-tok", ownerTok, nil, ""); code != http.StatusForbidden {
		t.Fatalf("bad download token status=%d want=403", code)
	}

	dlURL := srv.URL + "/v1/finance/packs/" + pack.PackID + "/download?format=zip&token=" + pack.DownloadToken
	code, dlHdr, dlBytes := doWave3CHTTP(t, http.MethodGet, dlURL, ownerTok, nil, "")
	if code != http.StatusOK {
		t.Fatalf("download pack=%d want=200 body=%s", code, string(dlBytes))
	}
	if dlHdr.Get("X-Atomos-Watermark") == "" || dlHdr.Get("X-Atomos-Pack-Sha256") != pack.Sha256Hex {
		t.Fatalf("missing/mismatched pack download headers: wm=%q sha=%q want=%q",
			dlHdr.Get("X-Atomos-Watermark"), dlHdr.Get("X-Atomos-Pack-Sha256"), pack.Sha256Hex)
	}
	zr, err := zip.NewReader(bytes.NewReader(dlBytes), int64(len(dlBytes)))
	if err != nil {
		t.Fatalf("invalid zip returned on download: %v", err)
	}
	foundManifest, foundXLSX := false, false
	for _, zf := range zr.File {
		if strings.HasSuffix(zf.Name, "manifest.json") {
			foundManifest = true
		}
		if strings.HasSuffix(zf.Name, ".xlsx") {
			foundXLSX = true
		}
	}
	if !foundManifest || !foundXLSX {
		t.Fatalf("expected manifest.json and .xlsx inside pack zip; got %d entries", len(zr.File))
	}

	// Expired pack -> 410 Gone.
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Update("ExportPacks", []string{"TenantId", "PackId", "ExpiresAt"}, []any{
			tenantID, pack.PackID, time.Now().UTC().Add(-time.Minute),
		}),
	}); err != nil {
		t.Fatalf("backdate ExportPacks.ExpiresAt: %v", err)
	}
	if code, _, b := doWave3CHTTP(t, http.MethodGet, dlURL, ownerTok, nil, ""); code != http.StatusGone {
		t.Fatalf("expired pack download=%d want=410 body=%s", code, string(b))
	}
	// Restore future expiry for auditor read verification next.
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Update("ExportPacks", []string{"TenantId", "PackId", "ExpiresAt"}, []any{
			tenantID, pack.PackID, time.Now().UTC().Add(12 * time.Hour),
		}),
	}); err != nil {
		t.Fatalf("restore ExportPacks.ExpiresAt: %v", err)
	}

	// 6. Time-boxed AUDITOR Grant lifecycle via HTTP: grant -> read 200 -> write 403 -> expired JWT 401 -> revoke -> 401.
	code, _, grantBody := doWave3CHTTP(t, http.MethodPost, srv.URL+"/v1/iam/auditor-grants", ownerTok, map[string]any{
		"entity_id": tenantID,
		"user_id":   extAudUser,
		"ttl_hours": int64(24),
		"reason":    "External tax/bank audit Wave 3C",
	}, "")
	if code != http.StatusCreated {
		t.Fatalf("POST /v1/iam/auditor-grants=%d want=201 body=%s", code, string(grantBody))
	}
	var grantRes iam.AuditorGrantResult
	if err := json.Unmarshal(grantBody, &grantRes); err != nil {
		t.Fatalf("decode grant result: %v", err)
	}
	if grantRes.Token == "" {
		t.Fatalf("expected non-empty minted AUDITOR JWT token")
	}

	for _, getPath := range []string{
		"/v1/connectors",
		"/v1/connectors/runs",
		"/v1/finance/packs",
		"/v1/finance/packs/" + pack.PackID,
		"/v1/finance/packs/" + pack.PackID + "/download?format=zip&token=" + pack.DownloadToken,
	} {
		if code, _, b := doWave3CHTTP(t, http.MethodGet, srv.URL+getPath, grantRes.Token, nil, ""); code != http.StatusOK {
			t.Fatalf("granted AUDITOR GET %s=%d want=200 body=%s", getPath, code, string(b))
		}
	}
	if code, _, _ := doWave3CHTTP(t, http.MethodPost, srv.URL+"/v1/finance/packs", grantRes.Token, map[string]any{"pack_type": "AUDIT"}, ""); code != http.StatusForbidden {
		t.Fatalf("granted AUDITOR POST /v1/finance/packs=%d want=403", code)
	}
	if code, _, _ := doWave3CHTTP(t, http.MethodPut, srv.URL+"/v1/connectors/credentials", grantRes.Token, map[string]any{"provider": "billz"}, ""); code != http.StatusForbidden {
		t.Fatalf("granted AUDITOR PUT /v1/connectors/credentials=%d want=403", code)
	}
	for _, tc := range []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodPost, "/v1/supplier/partner-exports", map[string]any{"resource": "orders", "format": "csv"}},
		{http.MethodPost, "/v1/supplier/partner-keys", map[string]any{"name": "auditor-blocked-key", "scopes": []string{"exports:read"}}},
		{http.MethodPost, "/v1/supplier/partner-webhooks", map[string]any{"url": "https://example.local/hook", "events": []string{"orders.created"}}},
		{http.MethodPut, "/v1/supplier/partner-sftp", map[string]any{"host": "sftp.example.local", "username": "aud"}},
	} {
		if code, _, b := doWave3CHTTP(t, tc.method, srv.URL+tc.path, grantRes.Token, tc.body, ""); code != http.StatusForbidden {
			t.Fatalf("granted AUDITOR %s %s=%d want=403 body=%s", tc.method, tc.path, code, string(b))
		}
	}

	pastAud := time.Now().UTC().Add(-2 * time.Minute)
	expiredAudTok := issueWave3CSupplierToken(t, h, tenantID, extAudUser, auth.Role(auth.SupplierRoleAuditor), auth.ActorTypeHuman, &pastAud, []string{auth.PermAuditRead})
	for _, expPath := range []string{"/v1/connectors", "/v1/connectors/runs", "/v1/finance/packs", "/v1/supplier/partner-keys", "/v1/supplier/partner-exports"} {
		code, _, expBody := doWave3CHTTP(t, http.MethodGet, srv.URL+expPath, expiredAudTok, nil, "")
		if code != http.StatusUnauthorized || !strings.Contains(string(expBody), "auditor_access_expired") {
			t.Fatalf("expired AUDITOR GET %s status=%d body=%s want=401 auditor_access_expired", expPath, code, string(expBody))
		}
	}

	if code, _, revBody := doWave3CHTTP(t, http.MethodPost, srv.URL+"/v1/iam/auditor-grants/"+extAudUser+"/revoke", ownerTok, map[string]any{"entity_id": tenantID}, ""); code != http.StatusOK {
		t.Fatalf("POST revoke=%d want=200 body=%s", code, string(revBody))
	}
	if code, _, b := doWave3CHTTP(t, http.MethodGet, srv.URL+"/v1/finance/packs", grantRes.Token, nil, ""); code != http.StatusUnauthorized {
		t.Fatalf("revoked AUDITOR GET /v1/finance/packs=%d want=401 body=%s", code, string(b))
	}

	// 7. Verify Wave 3C mutating endpoints registered in chi router.
	foundRoutes := map[string]bool{}
	_ = chi.Walk(h.Router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		foundRoutes[method+" "+route] = true
		return nil
	})
	for _, want := range []string{
		"PUT /v1/connectors/credentials",
		"POST /v1/connectors/{provider}/health",
		"POST /v1/connectors/{provider}/pull",
		"POST /v1/connectors/{provider}/shadow-compare",
		"POST /v1/finance/packs",
		"POST /v1/iam/auditor-grants",
		"POST /v1/iam/auditor-grants/{userID}/revoke",
	} {
		if !foundRoutes[want] {
			t.Fatalf("Wave 3C route %q missing from e2ep0 harness chi.Walk", want)
		}
	}

	// 8. Live Spanner 10,500-row .xlsx & encrypted 422 import pipeline via supplierroutes.
	t.Run("SupplierImport10500XLSXAndEncrypted422", func(t *testing.T) {
		sr := chi.NewRouter()
		sr.Use(auth.SessionAuth(h.JWTSecret))
		sr.Use(auth.AttachTenantFromClaims)
		supSvc := supplier.NewService(supplier.ServiceConfig{
			Repo:       supplier.NewSpannerRepository(client),
			SupplierID: tenantID,
			JWTSecret:  h.JWTSecret,
		})
		supplierroutes.RegisterRoutes(sr, supplierroutes.Deps{
			Service:   supSvc,
			JWTSecret: h.JWTSecret,
			Spanner:   client,
		})
		supSrv := httptest.NewServer(sr)
		defer supSrv.Close()

		xlsxBytes := buildWave3CMultiSheetXLSX10500(t, whID)
		code, _, s1Body := doWave3CHTTP(t, http.MethodPost, supSrv.URL+"/v1/supplier/inventory/imports", ownerTok, map[string]any{
			"file_name":       "wave3c_10500.xlsx",
			"file_size_bytes": int64(len(xlsxBytes)),
		}, "")
		if code != http.StatusOK && code != http.StatusCreated {
			t.Fatalf("create 10500 import session=%d body=%s", code, string(s1Body))
		}
		var s1Resp map[string]string
		_ = json.Unmarshal(s1Body, &s1Resp)
		sessBigID := s1Resp["session_id"]

		// Enforce AUDITOR & AGENT -> 403 across legacy /inventory/import and all 6 import session write routes.
		supWritePaths := []string{
			"/v1/supplier/inventory/import",
			"/v1/supplier/inventory/imports",
			"/v1/supplier/inventory/imports/" + sessBigID + "/uploaded",
			"/v1/supplier/inventory/imports/" + sessBigID + "/ingest",
			"/v1/supplier/inventory/imports/" + sessBigID + "/mapping",
			"/v1/supplier/inventory/imports/" + sessBigID + "/approve",
			"/v1/supplier/inventory/imports/" + sessBigID + "/apply",
		}
		for _, p := range supWritePaths {
			for _, tok := range []string{auditorStaticTok, agentTok} {
				if code, _, b := doWave3CHTTP(t, http.MethodPost, supSrv.URL+p, tok, map[string]any{}, ""); code != http.StatusForbidden {
					t.Fatalf("read-only/agent POST %s=%d want=403 body=%s", p, code, string(b))
				}
			}
			if code, _, b := doWave3CHTTP(t, http.MethodPost, supSrv.URL+p, expiredAudTok, map[string]any{}, ""); code != http.StatusUnauthorized || !strings.Contains(string(b), "auditor_access_expired") {
				t.Fatalf("expired AUDITOR POST %s=%d want=401 auditor_access_expired body=%s", p, code, string(b))
			}
		}
		if code, _, b := doWave3CHTTP(t, http.MethodGet, supSrv.URL+"/v1/supplier/inventory/imports/"+sessBigID, expiredAudTok, nil, ""); code != http.StatusUnauthorized || !strings.Contains(string(b), "auditor_access_expired") {
			t.Fatalf("expired AUDITOR GET import session=%d want=401 body=%s", code, string(b))
		}

		reqBig, err := http.NewRequest(http.MethodPost, supSrv.URL+"/v1/supplier/inventory/imports/"+sessBigID+"/ingest", bytes.NewReader(xlsxBytes))
		if err != nil {
			t.Fatalf("new ingest req: %v", err)
		}
		reqBig.Header.Set("Authorization", "Bearer "+ownerTok)
		reqBig.Header.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		respBig, err := (&http.Client{Timeout: 30 * time.Second}).Do(reqBig)
		if err != nil {
			t.Fatalf("do 10500 ingest: %v", err)
		}
		bigRaw, _ := io.ReadAll(respBig.Body)
		_ = respBig.Body.Close()
		if respBig.StatusCode != http.StatusOK {
			t.Fatalf("10500 xlsx ingest=%d want=200 body=%s", respBig.StatusCode, string(bigRaw))
		}
		if rowsStaged := decodeJSONInt64Field(t, bigRaw, "rows_staged"); rowsStaged != 10500 {
			t.Fatalf("rows_staged=%d want=10500 body=%s", rowsStaged, string(bigRaw))
		}

		// Encrypted OLE2 container -> 422.
		encBytes := buildWave3CEncryptedOLE2Container()
		code, _, s2Body := doWave3CHTTP(t, http.MethodPost, supSrv.URL+"/v1/supplier/inventory/imports", ownerTok, map[string]any{
			"file_name":       "locked.xlsx",
			"file_size_bytes": int64(len(encBytes)),
		}, "")
		if code != http.StatusOK && code != http.StatusCreated {
			t.Fatalf("create encrypted import session=%d body=%s", code, string(s2Body))
		}
		var s2Resp map[string]string
		_ = json.Unmarshal(s2Body, &s2Resp)
		sessEncID := s2Resp["session_id"]

		reqEnc, err := http.NewRequest(http.MethodPost, supSrv.URL+"/v1/supplier/inventory/imports/"+sessEncID+"/ingest", bytes.NewReader(encBytes))
		if err != nil {
			t.Fatalf("new enc ingest req: %v", err)
		}
		reqEnc.Header.Set("Authorization", "Bearer "+ownerTok)
		reqEnc.Header.Set("Content-Type", "application/octet-stream")
		respEnc, err := (&http.Client{Timeout: 10 * time.Second}).Do(reqEnc)
		if err != nil {
			t.Fatalf("do enc ingest: %v", err)
		}
		encRaw, _ := io.ReadAll(respEnc.Body)
		_ = respEnc.Body.Close()
		if respEnc.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("encrypted xlsx ingest=%d want=422 body=%s", respEnc.StatusCode, string(encRaw))
		}
	})
}
