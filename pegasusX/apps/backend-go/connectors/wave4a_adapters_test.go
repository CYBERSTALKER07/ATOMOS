package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

type stubPushApprovalChecker struct {
	mu           sync.Mutex
	approvedID   string
	expectedHash string
	calls        int
}

func (c *stubPushApprovalChecker) VerifyPushApproval(_ context.Context, in PushApprovalInput) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if strings.TrimSpace(in.ApprovalID) == "" || in.ApprovalID != c.approvedID {
		return ErrApprovalRequired
	}
	if c.expectedHash != "" {
		raw, _ := json.Marshal(in.Payload)
		if sha256HexString(string(raw)) != c.expectedHash {
			return ErrApprovalPayloadMismatch
		}
	}
	return nil
}

func TestWave4A_BillzTwoWayPush_And_OnecODataJournalPush(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("BILLZ_API_BASE_URL", "")
	t.Setenv("BILLZ_API_TOKEN", "")
	t.Setenv("ONEC_API_BASE_URL", "")
	t.Setenv("ONEC_API_TOKEN", "")

	var billzMu sync.Mutex
	var billzHits []string
	var billzBodies []string
	billzSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		billzMu.Lock()
		billzHits = append(billzHits, r.Method+" "+r.URL.Path+"|"+r.Header.Get("X-Atomos-Approval-Id"))
		billzBodies = append(billzBodies, string(b))
		billzMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"document_id":"billz-doc-9001"}`))
	}))
	defer billzSrv.Close()

	var onecMu sync.Mutex
	var onecPaths []string
	var onecContentTypes []string
	var onecBodies []string
	onecSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		onecMu.Lock()
		onecPaths = append(onecPaths, r.URL.Path)
		onecContentTypes = append(onecContentTypes, r.Header.Get("Content-Type"))
		onecBodies = append(onecBodies, string(b))
		onecMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"external_doc_id":"1c-jrn-44"}`))
	}))
	defer onecSrv.Close()

	t.Setenv("CONNECTOR_SECRET_W4A_BILLZ_TOK", "tok-billz-push")
	t.Setenv("CONNECTOR_SECRET_W4A_ONEC_TOK", "u:p-onec-push")

	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	svc.SetHTTPClient(billzSrv.Client())
	auditRec := &stubAuditRecorder{}
	svc.SetAuditRecorder(auditRec)
	checker := &stubPushApprovalChecker{approvedID: "apr-valid-1"}
	svc.SetApprovalChecker(checker)

	_, _ = svc.UpsertCredential(context.Background(), UpsertCredentialInput{
		TenantID: "sup-w4a-push", Provider: ProviderBillz, AccountLabel: "default",
		AuthType: AuthTypeBearer, SecretRef: "env://CONNECTOR_SECRET_W4A_BILLZ_TOK", BaseURL: billzSrv.URL, CreatedBy: "owner-1",
	})
	_, _ = svc.UpsertCredential(context.Background(), UpsertCredentialInput{
		TenantID: "sup-w4a-push", Provider: ProviderOnec, AccountLabel: "hq",
		AuthType: AuthTypeBasic, SecretRef: "env://CONNECTOR_SECRET_W4A_ONEC_TOK", BaseURL: onecSrv.URL, CreatedBy: "owner-1",
	})

	// Missing approval_id fails closed -> ErrApprovalRequired.
	if _, err := svc.Push(context.Background(), TriggerConnectorInput{
		TenantID: "sup-w4a-push", Provider: ProviderBillz, Stream: "goods_receipt",
		Records: []CanonicalRecord{{ExternalID: "SKU-1", SKU: "SKU-1", WarehouseID: "WH-1", QuantityOnHand: 25}},
		ActorID: "owner-1", ActorRole: "OWNER", ActorType: "human",
	}); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("expected ErrApprovalRequired when approval_id missing, got %v", err)
	}

	// AUDITOR & AGENT rejected even with valid approval_id.
	if _, err := svc.Push(context.Background(), TriggerConnectorInput{
		TenantID: "sup-w4a-push", Provider: ProviderBillz, Stream: "goods_receipt", ApprovalID: "apr-valid-1",
		Records: []CanonicalRecord{{ExternalID: "SKU-1", QuantityOnHand: 25}},
		ActorID: "aud-1", ActorRole: "AUDITOR", ActorType: "human",
	}); !errors.Is(err, ErrAuditorReadOnly) {
		t.Fatalf("expected ErrAuditorReadOnly for AUDITOR push, got %v", err)
	}
	if _, err := svc.Push(context.Background(), TriggerConnectorInput{
		TenantID: "sup-w4a-push", Provider: ProviderBillz, Stream: "goods_receipt", ApprovalID: "apr-valid-1",
		Records: []CanonicalRecord{{ExternalID: "SKU-1", QuantityOnHand: 25}},
		ActorID: "agt-1", ActorRole: "AGENT", ActorType: "agent",
	}); !errors.Is(err, ErrAgentForbidden) {
		t.Fatalf("expected ErrAgentForbidden for AGENT push, got %v", err)
	}

	// Approved Billz goods-receipt write-back succeeds and emits connector.push.completed.
	bRes, err := svc.Push(context.Background(), TriggerConnectorInput{
		TenantID: "sup-w4a-push", Provider: ProviderBillz, Stream: "goods_receipt", ApprovalID: "apr-valid-1",
		IdempotencyKey: "push-billz-1",
		Records: []CanonicalRecord{{
			ObjectType: ObjectTypeInventory, ExternalID: "SKU-1", SKU: "SKU-1", WarehouseID: "WH-1", QuantityOnHand: 25,
		}},
		ActorID: "owner-1", ActorRole: "OWNER", ActorType: "human",
	})
	if err != nil {
		t.Fatalf("Billz Push failed: %v", err)
	}
	if bRes.PushResult.ExternalDocID != "billz-doc-9001" || bRes.PushResult.RecordsPushed != 1 || bRes.Run.Direction != DirectionPush {
		t.Fatalf("unexpected Billz PushResponse: %+v", bRes)
	}
	if len(billzHits) != 1 || billzHits[0] != "POST /v1/goods-receipts|apr-valid-1" {
		t.Fatalf("unexpected Billz upstream hits: %v", billzHits)
	}

	// Approved 1C OData journal pushback succeeds.
	oRes, err := svc.Push(context.Background(), TriggerConnectorInput{
		TenantID: "sup-w4a-push", Provider: ProviderOnec, AccountLabel: "hq", Stream: "odata_journals", ApprovalID: "apr-valid-1",
		IdempotencyKey: "push-onec-odata-1",
		Records: []CanonicalRecord{{
			ObjectType: ObjectTypeOrder, ExternalID: "JE-100", OrderID: "JE-100", TotalMinor: 550000, Currency: "UZS", Status: "POSTED",
		}},
		ActorID: "owner-1", ActorRole: "OWNER", ActorType: "human",
	})
	if err != nil {
		t.Fatalf("1C OData Push failed: %v", err)
	}
	if oRes.PushResult.ExternalDocID != "1c-jrn-44" || !strings.Contains(onecPaths[0], "/odata/journals") ||
		!strings.Contains(onecContentTypes[0], "application/json") || !strings.Contains(onecBodies[0], `"AmountMinor":550000`) {
		t.Fatalf("unexpected 1C OData push: res=%+v paths=%v ct=%v body=%v", oRes, onecPaths, onecContentTypes, onecBodies)
	}

	// Approved 1C CommerceML plural "goods_receipts" pushback emits <КоммерческаяИнформация><Документ>.
	xmlRes, err := svc.Push(context.Background(), TriggerConnectorInput{
		TenantID: "sup-w4a-push", Provider: ProviderOnec, AccountLabel: "hq", Stream: "goods_receipts", ApprovalID: "apr-valid-1",
		IdempotencyKey: "push-onec-goods-receipts-1",
		Records: []CanonicalRecord{{
			ObjectType: ObjectTypeOrder, ExternalID: "GR-101", OrderID: "GR-101", RetailerID: "sup-1", TotalMinor: 420000, Currency: "UZS", Status: "RECEIVED",
		}},
		ActorID: "owner-1", ActorRole: "OWNER", ActorType: "human",
	})
	if err != nil {
		t.Fatalf("1C goods_receipts CommerceML Push failed: %v", err)
	}
	if xmlRes.PushResult.Status != RunStatusCompleted || len(onecBodies) < 2 ||
		!strings.Contains(onecContentTypes[1], "application/xml") ||
		!strings.Contains(onecBodies[1], "<КоммерческаяИнформация") ||
		!strings.Contains(onecBodies[1], "<Документ>") {
		t.Fatalf("expected CommerceML <Документ> XML for goods_receipts push: res=%+v ct=%v bodies=%v", xmlRes, onecContentTypes, onecBodies)
	}

	// Verify TopicConnectors emitted connector.push.completed events.
	outboxEvs := repo.EmittedOutbox()
	pushEvents := 0
	for _, em := range outboxEvs {
		if em.Topic != events.TopicConnectors {
			t.Fatalf("expected TopicConnectors, got %s", em.Topic)
		}
		if pev, ok := em.Event.(events.ConnectorPushCompletedEvent); ok && pev.Type == events.EventConnectorPushCompleted {
			pushEvents++
			if pev.ApprovalID != "apr-valid-1" {
				t.Fatalf("missing approval_id in ConnectorPushCompletedEvent: %+v", pev)
			}
		}
	}
	if pushEvents != 3 {
		t.Fatalf("expected 3 connector.push.completed events, got %d", pushEvents)
	}
}

func TestWave4A_CIS_HORECA_Adapters_MoySklad_Smartup_Iiko_HealthPullPushShadowCompare(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	for _, k := range []string{
		"MOYSKLAD_API_BASE_URL", "MOYSKLAD_API_TOKEN",
		"SMARTUP_API_BASE_URL", "SMARTUP_API_TOKEN",
		"IIKO_API_BASE_URL", "IIKO_API_TOKEN",
	} {
		t.Setenv(k, "")
	}

	// 1. All 3 fail closed in PEGASUSX_ENV=production when URL/token unset.
	for _, conn := range []Connector{
		NewMoySkladConnector(NewDefaultSecretResolver(nil), nil),
		NewSmartupConnector(NewDefaultSecretResolver(nil), nil),
		NewIikoConnector(NewDefaultSecretResolver(nil), nil),
	} {
		if _, err := conn.Health(context.Background(), ExternalCredential{Provider: conn.Provider()}); !errors.Is(err, ErrConnectorNotConfigured) {
			t.Fatalf("%s Health expected ErrConnectorNotConfigured, got %v", conn.Provider(), err)
		}
		if _, err := conn.Pull(context.Background(), ExternalCredential{Provider: conn.Provider()}, PullRequest{Stream: StreamAll}); !errors.Is(err, ErrConnectorNotConfigured) {
			t.Fatalf("%s Pull expected ErrConnectorNotConfigured, got %v", conn.Provider(), err)
		}
		if _, err := conn.Push(context.Background(), ExternalCredential{Provider: conn.Provider()}, PushRequest{Records: []CanonicalRecord{{ExternalID: "X"}}}); !errors.Is(err, ErrConnectorNotConfigured) {
			t.Fatalf("%s Push expected ErrConnectorNotConfigured, got %v", conn.Provider(), err)
		}
	}

	// 2. Real httptest.Server fixtures for MoySklad, Smartup, and iiko.
	var pushedMu sync.Mutex
	pushedPaths := map[string]string{}

	msSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			b, _ := io.ReadAll(r.Body)
			pushedMu.Lock()
			pushedPaths["moysklad"] = r.URL.Path + "|" + string(b)
			pushedMu.Unlock()
			_, _ = w.Write([]byte(`{"id":"ms-supply-77"}`))
			return
		}
		switch r.URL.Path {
		case "/entity/organization":
			_, _ = w.Write([]byte(`{"rows":[{"id":"org-1","name":"Tashkent Main"}]}`))
		case "/entity/assortment":
			_, _ = w.Write([]byte(`{"rows":[{"id":"SKU-MS-1","article":"SKU-MS-1","name":"Sugar 1kg","barcodes":[{"ean13":"478001122"}],"salePrices":[{"value":1400000}]}]}`))
		case "/report/stock/all":
			_, _ = w.Write([]byte(`{"rows":[{"assortmentId":"SKU-MS-1","storeId":"WH-MS","stock":55}]}`))
		case "/entity/store":
			_, _ = w.Write([]byte(`{"rows":[{"id":"WH-MS","name":"Central Warehouse"}]}`))
		case "/entity/customerorder":
			_, _ = w.Write([]byte(`{"rows":[{"id":"ORD-MS-1","name":"ORD-MS-1","sum":2800000,"agent":{"id":"RET-MS-1"}}]}`))
		case "/entity/counterparty":
			_, _ = w.Write([]byte(`{"rows":[{"id":"RET-MS-1","name":"Korzinka","inn":"301112223","balance":400000}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer msSrv.Close()

	suSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			b, _ := io.ReadAll(r.Body)
			pushedMu.Lock()
			pushedPaths["smartup"] = r.URL.Path + "|" + string(b)
			pushedMu.Unlock()
			_, _ = w.Write([]byte(`{"document_id":"su-gr-88"}`))
			return
		}
		switch r.URL.Path {
		case "/v1/warehouses":
			_, _ = w.Write([]byte(`{"warehouses":[{"warehouse_id":"WH-SU","name":"Samarkand Hub"}]}`))
		case "/v1/products":
			_, _ = w.Write([]byte(`{"products":[{"product_id":"SKU-SU-1","sku":"SKU-SU-1","barcode":"47800999","name":"Oil 5L","price_minor":8500000,"currency":"UZS"}]}`))
		case "/v1/stocks":
			_, _ = w.Write([]byte(`{"stocks":[{"product_id":"SKU-SU-1","warehouse_id":"WH-SU","balance":30}]}`))
		case "/v1/orders":
			_, _ = w.Write([]byte(`{"orders":[{"deal_id":"ORD-SU-1","person_id":"RET-SU-1","total_minor":8500000,"currency":"UZS","status":"OPEN"}]}`))
		case "/v1/clients":
			_, _ = w.Write([]byte(`{"clients":[{"person_id":"RET-SU-1","name":"Havas","balance_minor":1200000,"currency":"UZS"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer suSrv.Close()

	iiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			b, _ := io.ReadAll(r.Body)
			pushedMu.Lock()
			pushedPaths["iiko"] = r.URL.Path + "|" + string(b)
			pushedMu.Unlock()
			_, _ = w.Write([]byte(`{"data":{"id":"iiko-inv-99"}}`))
			return
		}
		switch r.URL.Path {
		case "/api/1/organizations":
			_, _ = w.Write([]byte(`{"organizations":[{"id":"WH-IIKO","name":"Bukhara Kitchen"}]}`))
		case "/api/1/nomenclature":
			_, _ = w.Write([]byte(`{"products":[{"id":"SKU-II-1","sku":"SKU-II-1","barcode":"47800777","name":"Coffee Beans 1kg","price_minor":19000000,"currency":"UZS"}]}`))
		case "/api/1/stocks":
			_, _ = w.Write([]byte(`{"stocks":[{"productId":"SKU-II-1","organizationId":"WH-IIKO","amount":18}]}`))
		case "/api/1/orders":
			_, _ = w.Write([]byte(`{"orders":[{"id":"ORD-II-1","customerId":"RET-II-1","sum":190000,"currency":"UZS"}]}`))
		case "/api/1/customers":
			_, _ = w.Write([]byte(`{"customers":[{"id":"RET-II-1","name":"Caravan Cafe","balance_minor":500000,"currency":"UZS"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer iiSrv.Close()

	t.Setenv("CONNECTOR_SECRET_MS_TEST_TOK", "ms-tok")
	t.Setenv("CONNECTOR_SECRET_SU_TEST_TOK", "su-tok")
	t.Setenv("CONNECTOR_SECRET_IIKO_TEST_TOK", "ii-tok")

	repo := NewMemoryRepository()
	repo.SeedInternalSnapshot("sup-cis-1", InternalSnapshot{
		Products: []InternalProduct{
			{ProductID: "SKU-MS-1", Name: "Sugar 1kg", Barcode: "478001122", PriceMinor: 1400000, StockQuantity: 40, Currency: "UZS", IsActive: true},
			{ProductID: "SKU-SU-1", Name: "Oil 5L", Barcode: "47800999", PriceMinor: 8500000, StockQuantity: 30, Currency: "UZS", IsActive: true},
			{ProductID: "SKU-II-1", Name: "Coffee Beans 1kg", Barcode: "47800777", PriceMinor: 19000000, StockQuantity: 10, Currency: "UZS", IsActive: true},
		},
		Stocks: []InternalStock{
			{WarehouseID: "WH-MS", ProductID: "SKU-MS-1", QuantityOnHand: 40},
			{WarehouseID: "WH-SU", ProductID: "SKU-SU-1", QuantityOnHand: 30},
			{WarehouseID: "WH-IIKO", ProductID: "SKU-II-1", QuantityOnHand: 10},
		},
	})

	svc := NewServiceWithRepository(repo)
	svc.SetHTTPClient(msSrv.Client())
	svc.SetApprovalChecker(&stubPushApprovalChecker{approvedID: "apr-cis-1"})

	// Verify all 5 adapters registered in ListAdapters with ReadOnly=false.
	adapters := svc.ListAdapters()
	if len(adapters) != 5 {
		t.Fatalf("expected 5 adapters, got %d (%+v)", len(adapters), adapters)
	}
	for _, ad := range adapters {
		if ad.ReadOnly {
			t.Fatalf("adapter %s unexpectedly marked ReadOnly=true", ad.Provider)
		}
	}

	cases := []struct {
		provider       string
		secretEnv      string
		baseURL        string
		pushStream     string
		wantPushPrefix string
		wantDocID      string
		wantStockDelta int64
	}{
		{ProviderMoySklad, "env://CONNECTOR_SECRET_MS_TEST_TOK", msSrv.URL, "goods_receipt", "/entity/supply", "ms-supply-77", 15},
		{ProviderSmartup, "env://CONNECTOR_SECRET_SU_TEST_TOK", suSrv.URL, "goods_receipt", "/v1/goods-receipts", "su-gr-88", 0},
		{ProviderIiko, "env://CONNECTOR_SECRET_IIKO_TEST_TOK", iiSrv.URL, "goods_receipt", "/api/1/documents/incoming_invoice", "iiko-inv-99", 8},
	}

	for _, tc := range cases {
		if _, err := svc.UpsertCredential(context.Background(), UpsertCredentialInput{
			TenantID: "sup-cis-1", Provider: tc.provider, AccountLabel: "default",
			AuthType: AuthTypeBearer, SecretRef: tc.secretEnv, BaseURL: tc.baseURL, CreatedBy: "owner-1",
		}); err != nil {
			t.Fatalf("UpsertCredential %s: %v", tc.provider, err)
		}

		hRes, err := svc.CheckHealth(context.Background(), TriggerConnectorInput{
			TenantID: "sup-cis-1", Provider: tc.provider, ActorID: "owner-1",
		})
		if err != nil || !hRes.Healthy {
			t.Fatalf("%s CheckHealth failed: res=%+v err=%v", tc.provider, hRes, err)
		}

		pullRes, err := svc.Pull(context.Background(), TriggerConnectorInput{
			TenantID: "sup-cis-1", Provider: tc.provider, ActorID: "owner-1",
		})
		if err != nil || pullRes.RecordsCount != 5 {
			t.Fatalf("%s Pull failed: res=%+v err=%v", tc.provider, pullRes, err)
		}

		scRes, err := svc.ShadowCompare(context.Background(), TriggerConnectorInput{
			TenantID: "sup-cis-1", Provider: tc.provider, ActorID: "owner-1",
		})
		if err != nil {
			t.Fatalf("%s ShadowCompare failed: %v", tc.provider, err)
		}
		if scRes.Summary.NetStockDelta != tc.wantStockDelta {
			t.Fatalf("%s ShadowCompare NetStockDelta=%d want=%d", tc.provider, scRes.Summary.NetStockDelta, tc.wantStockDelta)
		}

		pRes, err := svc.Push(context.Background(), TriggerConnectorInput{
			TenantID: "sup-cis-1", Provider: tc.provider, Stream: tc.pushStream, ApprovalID: "apr-cis-1",
			Records: []CanonicalRecord{{ExternalID: "SKU-1", SKU: "SKU-1", QuantityOnHand: 12}},
			ActorID: "owner-1", ActorRole: "OWNER", ActorType: "human",
		})
		if err != nil {
			t.Fatalf("%s Push failed: %v", tc.provider, err)
		}
		if pRes.PushResult.ExternalDocID != tc.wantDocID {
			t.Fatalf("%s ExternalDocID=%q want=%q", tc.provider, pRes.PushResult.ExternalDocID, tc.wantDocID)
		}
		if !strings.HasPrefix(pushedPaths[tc.provider], tc.wantPushPrefix+"|") {
			t.Fatalf("%s push hit %q, want prefix %q", tc.provider, pushedPaths[tc.provider], tc.wantPushPrefix)
		}
	}

	// Also verify plural "inventory_adjustments" stream routes cleanly across all 3 CIS/HORECA adapters.
	for _, adjCase := range []struct {
		provider   string
		wantPrefix string
	}{
		{ProviderMoySklad, "/entity/enter"},
		{ProviderSmartup, "/v1/inventory-adjustments"},
		{ProviderIiko, "/api/1/inventory/writeoff"},
	} {
		adjRes, err := svc.Push(context.Background(), TriggerConnectorInput{
			TenantID: "sup-cis-1", Provider: adjCase.provider, Stream: "inventory_adjustments", ApprovalID: "apr-cis-1",
			Records: []CanonicalRecord{{ExternalID: "SKU-ADJ-1", SKU: "SKU-ADJ-1", QuantityOnHand: 3}},
			ActorID: "owner-1", ActorRole: "OWNER", ActorType: "human",
		})
		if err != nil {
			t.Fatalf("%s inventory_adjustments Push failed: %v", adjCase.provider, err)
		}
		if adjRes.PushResult.Status != RunStatusCompleted || !strings.HasPrefix(pushedPaths[adjCase.provider], adjCase.wantPrefix+"|") {
			t.Fatalf("%s inventory_adjustments hit %q, want prefix %q (res=%+v)", adjCase.provider, pushedPaths[adjCase.provider], adjCase.wantPrefix, adjRes)
		}
	}
}
