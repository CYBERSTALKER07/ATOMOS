package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

type stubAuditRecorder struct {
	mu     sync.Mutex
	events []AuditRecordInput
}

func (a *stubAuditRecorder) RecordEvent(_ context.Context, ev AuditRecordInput) error {
	a.mu.Lock()
	a.events = append(a.events, ev)
	a.mu.Unlock()
	return nil
}

func TestSecretResolver_SchemesMaskingAndProductionFailClosed(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("CONNECTOR_SECRET_MY_BILLZ_TOKEN", "super-secret-billz-token-9876")
	t.Setenv("CONNECTOR_SECRET_ONEC_MAIN", "onec-pass-4321")
	t.Setenv("JWT_SECRET", "platform-jwt-signing-key-should-never-leak")

	resolver := NewDefaultSecretResolver(func(_ context.Context, secretName string) (string, error) {
		if secretName == "projects/p1/secrets/connector-sup1-billz/versions/latest" {
			return "gcp-resolved-token-5555", nil
		}
		return "", errors.New("not found")
	})

	if got, err := resolver.Resolve(context.Background(), "env://CONNECTOR_SECRET_MY_BILLZ_TOKEN"); err != nil || got != "super-secret-billz-token-9876" {
		t.Fatalf("env:// resolve got=%q err=%v", got, err)
	}
	if got, err := resolver.Resolve(context.Background(), "onec_main"); err != nil || got != "onec-pass-4321" {
		t.Fatalf("CONNECTOR_SECRET_* resolve got=%q err=%v", got, err)
	}
	if got, err := resolver.ResolveForTenant(context.Background(), "sup1", "gcp-secret://projects/p1/secrets/connector-sup1-billz/versions/latest"); err != nil || got != "gcp-resolved-token-5555" {
		t.Fatalf("gcp-secret:// resolve got=%q err=%v", got, err)
	}

	if _, err := resolver.Resolve(context.Background(), "env://CONNECTOR_SECRET_MISSING_BILLZ_TOKEN"); !errors.Is(err, ErrConnectorNotConfigured) {
		t.Fatalf("expected ErrConnectorNotConfigured on missing secret in prod, got %v", err)
	}

	// Security regressions: arbitrary env vars, cross-tenant GCP secrets, and paths must fail closed.
	for _, bad := range []struct {
		tenant string
		ref    string
	}{
		{"", "env://JWT_SECRET"},
		{"", "JWT_SECRET"},
		{"", "/etc/passwd"},
		{"", "gcp-secret://projects/p1/secrets/jwt-signing-key/versions/latest"},
		{"sup2", "gcp-secret://projects/p1/secrets/connector-sup1-billz/versions/latest"},
	} {
		if _, err := resolver.ResolveForTenant(context.Background(), bad.tenant, bad.ref); err == nil {
			t.Fatalf("expected secret ref %q (tenant=%q) to be rejected", bad.ref, bad.tenant)
		}
	}

	masked := MaskSecretRef("env://CONNECTOR_SECRET_MY_BILLZ_TOKEN")
	if masked != "env://****OKEN" || strings.Contains(masked, "MY_BILLZ") {
		t.Fatalf("MaskSecretRef unexpected: %q", masked)
	}
	if got := MaskSecretRef("raw-secret-123456"); got != "****3456" || strings.Contains(got, "raw-secret") {
		t.Fatalf("MaskSecretRef leaked prefix: %q", got)
	}
}

func TestBillzConnector_HTTP_HealthPullErrorsAndReadOnlyPush(t *testing.T) {
	t.Setenv("BILLZ_API_BASE_URL", "")
	t.Setenv("BILLZ_API_TOKEN", "")
	t.Setenv("PEGASUSX_ENV", "production")

	billz := NewBillzConnector(NewDefaultSecretResolver(nil), nil)

	// 1. Fail closed in production when BaseURL/token missing (zero mock rows).
	if _, err := billz.Health(context.Background(), ExternalCredential{Provider: ProviderBillz}); !errors.Is(err, ErrConnectorNotConfigured) {
		t.Fatalf("expected ErrConnectorNotConfigured on unconfigured health, got %v", err)
	}
	if _, err := billz.Pull(context.Background(), ExternalCredential{Provider: ProviderBillz}, PullRequest{Stream: StreamAll}); !errors.Is(err, ErrConnectorNotConfigured) {
		t.Fatalf("expected ErrConnectorNotConfigured on unconfigured pull, got %v", err)
	}

	// 2. Push also fails closed in production when BaseURL/token missing.
	if _, err := billz.Push(context.Background(), ExternalCredential{Provider: ProviderBillz}, PushRequest{
		Stream: "goods_receipt", ApprovalID: "apr-1", Records: []CanonicalRecord{{ExternalID: "SKU-1", QuantityOnHand: 5}},
	}); !errors.Is(err, ErrConnectorNotConfigured) {
		t.Fatalf("expected ErrConnectorNotConfigured on unconfigured Billz Push, got %v", err)
	}

	// 3. Full httptest.Server covering /v1/products, /v1/stocks, /v1/shops, /v1/orders, /v1/clients.
	var mu sync.Mutex
	hitPaths := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-billz-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid_token"}`))
			return
		}
		mu.Lock()
		hitPaths[r.URL.Path]++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/shops":
			_, _ = w.Write([]byte(`{"shops":[{"id":"shop-1","name":"Chilonzor Store","status":"ACTIVE"}]}`))
		case "/v1/products":
			_, _ = w.Write([]byte(`{"products":[{"id":"ext-sku-1","sku":"SKU-1","barcode":"460001","name":"Green Tea","price_minor":1500000,"currency":"UZS"}]}`))
		case "/v1/stocks":
			_, _ = w.Write([]byte(`{"stocks":[{"product_id":"ext-sku-1","warehouse_id":"WH-1","quantity_on_hand":45}]}`))
		case "/v1/orders":
			_, _ = w.Write([]byte(`{"orders":[{"id":"ext-ord-1","order_id":"ORD-1","client_id":"ret-1","total_minor":300000,"currency":"UZS","status":"COMPLETED"}]}`))
		case "/v1/clients":
			_, _ = w.Write([]byte(`{"clients":[{"id":"ret-1","name":"Baraka Market","balance_minor":750000,"currency":"UZS"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	billz = NewBillzConnector(NewDefaultSecretResolver(nil), srv.Client())

	t.Setenv("CONNECTOR_SECRET_BILLZ_TEST_TOKEN", "test-billz-token")
	cred := ExternalCredential{
		TenantID:     "sup-1",
		CredentialID: "cred-billz-1",
		Provider:     ProviderBillz,
		AccountLabel: "main",
		AuthType:     AuthTypeBearer,
		SecretRef:    "env://CONNECTOR_SECRET_BILLZ_TEST_TOKEN",
		BaseURL:      srv.URL,
	}

	hRes, err := billz.Health(context.Background(), cred)
	if err != nil || !hRes.Healthy || hRes.StatusCode != 200 {
		t.Fatalf("Billz Health failed: res=%+v err=%v", hRes, err)
	}

	pRes, err := billz.Pull(context.Background(), cred, PullRequest{Stream: StreamAll})
	if err != nil {
		t.Fatalf("Billz Pull failed: %v", err)
	}
	if pRes.StreamCounts[StreamProducts] != 1 || pRes.StreamCounts[StreamStocks] != 1 || pRes.StreamCounts[StreamShops] != 1 ||
		pRes.StreamCounts[StreamOrders] != 1 || pRes.StreamCounts[StreamClients] != 1 {
		t.Fatalf("unexpected stream counts: %+v", pRes.StreamCounts)
	}
	if pRes.RecordsCount != 5 {
		t.Fatalf("RecordsCount=%d want=5", pRes.RecordsCount)
	}

	// 4. Auth error (401) path.
	t.Setenv("CONNECTOR_SECRET_BILLZ_BAD_TOKEN", "wrong-token")
	badCred := cred
	badCred.SecretRef = "env://CONNECTOR_SECRET_BILLZ_BAD_TOKEN"
	if _, err := billz.Health(context.Background(), badCred); !errors.Is(err, ErrConnectorAuthFailed) {
		t.Fatalf("expected ErrConnectorAuthFailed on 401, got %v", err)
	}

	// 5. Upstream 500 error path.
	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errSrv.Close()
	errCred := cred
	errCred.BaseURL = errSrv.URL
	if _, err := billz.Pull(context.Background(), errCred, PullRequest{Stream: StreamProducts}); !errors.Is(err, ErrConnectorUpstreamError) {
		t.Fatalf("expected ErrConnectorUpstreamError on 500, got %v", err)
	}

	// 6. Timeout path.
	slowSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowSrv.Close()
	timeoutBillz := NewBillzConnector(NewDefaultSecretResolver(nil), &http.Client{Timeout: 30 * time.Millisecond})
	slowCred := cred
	slowCred.BaseURL = slowSrv.URL
	if _, err := timeoutBillz.Health(context.Background(), slowCred); !errors.Is(err, ErrConnectorTimeout) {
		t.Fatalf("expected ErrConnectorTimeout on slow server, got %v", err)
	}

	// 7. Malformed JSON payload path.
	badJSONSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not-valid-json`))
	}))
	defer badJSONSrv.Close()
	badJSONCred := cred
	badJSONCred.BaseURL = badJSONSrv.URL
	if _, err := billz.Pull(context.Background(), badJSONCred, PullRequest{Stream: StreamProducts}); !errors.Is(err, ErrConnectorBadPayload) {
		t.Fatalf("expected ErrConnectorBadPayload on malformed JSON, got %v", err)
	}
}

func TestOnecConnector_MultiAccount_CommerceML_PullAndPush(t *testing.T) {
	xmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<КоммерческаяИнформация ВерсияСхемы="2.10">
  <Каталог><Товары><Товар><Ид>1c-p1</Ид><Артикул>SKU-1</Артикул><Наименование>Tea 1C</Наименование><Штрихкод>478001</Штрихкод></Товар></Товары></Каталог>
  <ПакетПредложений><Предложения><Предложение><Ид>1c-p1</Ид><Артикул>SKU-1</Артикул><Цены><Цена><ЦенаЗаЕдиницу>12500</ЦенаЗаЕдиницу><Валюта>UZS</Валюта></Цена></Цены><Склад ИдСклада="WH-1" КоличествоНаСкладе="80"/></Предложение></Предложения></ПакетПредложений>
  <Контрагенты><Контрагент><Ид>ret-1</Ид><ОфициальноеНаименование>LLC Baraka</ОфициальноеНаименование><ИНН>301234567</ИНН></Контрагент></Контрагенты>
  <Документ><Ид>ORD-1</Ид><Номер>ORD-1</Номер><Сумма>250000</Сумма><Контрагенты><Контрагент><Ид>ret-1</Ид></Контрагент></Контрагенты></Документ>
</КоммерческаяИнформация>`

	var pushedBody string
	onecSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var b strings.Builder
			buf := make([]byte, 4096)
			n, _ := r.Body.Read(buf)
			b.Write(buf[:n])
			pushedBody = b.String()
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(xmlDoc))
	}))
	defer onecSrv.Close()

	t.Setenv("CONNECTOR_SECRET_ONEC_TASHKENT_AUTH", "admin:secret1")
	t.Setenv("CONNECTOR_SECRET_ONEC_SAMARKAND_AUTH", "admin:secret2")

	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	svc.SetHTTPClient(onecSrv.Client())

	if _, err := svc.UpsertCredential(context.Background(), UpsertCredentialInput{
		TenantID:     "sup-1",
		Provider:     ProviderOnec,
		AccountLabel: "tashkent",
		AuthType:     AuthTypeBasic,
		SecretRef:    "env://CONNECTOR_SECRET_ONEC_TASHKENT_AUTH",
		BaseURL:      onecSrv.URL,
		CreatedBy:    "owner-1",
	}); err != nil {
		t.Fatalf("UpsertCredential tashkent: %v", err)
	}
	if _, err := svc.UpsertCredential(context.Background(), UpsertCredentialInput{
		TenantID:     "sup-1",
		Provider:     ProviderOnec,
		AccountLabel: "samarkand",
		AuthType:     AuthTypeBasic,
		SecretRef:    "env://CONNECTOR_SECRET_ONEC_SAMARKAND_AUTH",
		BaseURL:      onecSrv.URL,
		CreatedBy:    "owner-1",
	}); err != nil {
		t.Fatalf("UpsertCredential samarkand: %v", err)
	}

	overview, err := svc.GetOverview(context.Background(), "sup-1")
	if err != nil {
		t.Fatalf("GetOverview: %v", err)
	}
	if len(overview.Credentials) != 2 {
		t.Fatalf("expected 2 multi-account 1C credentials, got %d", len(overview.Credentials))
	}
	for _, c := range overview.Credentials {
		raw, _ := json.Marshal(c)
		if strings.Contains(string(raw), "ONEC_TASHKENT_AUTH") || strings.Contains(string(raw), "secret1") {
			t.Fatalf("credential overview leaked secret: %s", raw)
		}
	}

	pullRes, err := svc.Pull(context.Background(), TriggerConnectorInput{
		TenantID:     "sup-1",
		Provider:     ProviderOnec,
		AccountLabel: "tashkent",
	})
	if err != nil {
		t.Fatalf("1C Pull failed: %v", err)
	}
	if pullRes.RecordsCount != 4 {
		t.Fatalf("1C RecordsCount=%d want=4 streamCounts=%+v", pullRes.RecordsCount, pullRes.StreamCounts)
	}

	onecConn := NewOnecConnector(NewDefaultSecretResolver(nil), onecSrv.Client())
	pushRes, err := onecConn.Push(context.Background(), ExternalCredential{
		Provider:     ProviderOnec,
		AccountLabel: "samarkand",
		AuthType:     AuthTypeBasic,
		SecretRef:    "env://CONNECTOR_SECRET_ONEC_SAMARKAND_AUTH",
		BaseURL:      onecSrv.URL,
	}, PushRequest{
		Records: []CanonicalRecord{{ExternalID: "SKU-1", Name: "Tea 1C", PriceMinor: 1250000, Currency: "UZS"}},
	})
	if err != nil || pushRes.RecordsPushed != 1 || !strings.Contains(pushedBody, "<Каталог>") {
		t.Fatalf("1C Push failed: res=%+v err=%v body=%s", pushRes, err, pushedBody)
	}
}

func TestShadowCompare_DeterministicDiffsOutboxAuditOntologyAndIdempotency(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/shops":
			_, _ = w.Write([]byte(`{"shops":[{"id":"WH-1","name":"Main Depot"}]}`))
		case "/v1/products":
			_, _ = w.Write([]byte(`{"products":[
				{"id":"SKU-1","sku":"SKU-1","barcode":"460001","name":"Butter","price_minor":2500000,"currency":"UZS"},
				{"id":"EXT-ONLY-SKU","sku":"EXT-ONLY-SKU","name":"External New Item","price_minor":100000,"currency":"UZS"}
			]}`))
		case "/v1/stocks":
			_, _ = w.Write([]byte(`{"stocks":[{"product_id":"SKU-1","warehouse_id":"WH-1","quantity_on_hand":45}]}`))
		case "/v1/orders":
			_, _ = w.Write([]byte(`{"orders":[{"id":"ORD-1","order_id":"ORD-1","client_id":"RET-1","total_minor":500000,"currency":"UZS"}]}`))
		case "/v1/clients":
			_, _ = w.Write([]byte(`{"clients":[{"id":"RET-1","name":"Baraka","balance_minor":750000,"currency":"UZS"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	t.Setenv("CONNECTOR_SECRET_BILLZ_SHADOW_TOKEN", "shadow-token-999")
	repo := NewMemoryRepository()
	repo.SeedInternalSnapshot("sup-shadow-1", InternalSnapshot{
		Products: []InternalProduct{
			{ProductID: "SKU-1", Name: "Butter", Barcode: "460001", PriceMinor: 2500000, StockQuantity: 30, Currency: "UZS", IsActive: true},
			{ProductID: "INT-ONLY-SKU", Name: "Internal Only", PriceMinor: 90000, StockQuantity: 10, Currency: "UZS", IsActive: true},
		},
		Stocks: []InternalStock{
			{WarehouseID: "WH-1", ProductID: "SKU-1", QuantityOnHand: 30},
		},
		Orders: []InternalOrder{
			{OrderID: "ORD-1", RetailerID: "RET-1", TotalMinor: 500000, Currency: "UZS", Status: "COMPLETED"},
			{OrderID: "ORD-INT-ONLY", RetailerID: "RET-2", TotalMinor: 120000, Currency: "UZS", Status: "OPEN"},
		},
		Invoices: []InternalInvoice{
			{InvoiceID: "INV-1", OrderID: "ORD-1", RetailerID: "RET-1", PrincipalMinor: 500000, BalanceMinor: 500000, Currency: "UZS", Status: "OPEN"},
		},
	})

	auditRec := &stubAuditRecorder{}
	var ontologyCalls int
	var ontologyMaps []ConnectorEntityMap

	svc := NewServiceWithRepository(repo)
	svc.SetHTTPClient(srv.Client())
	svc.SetAuditRecorder(auditRec)
	svc.SetOntologyLineageHook(func(_ context.Context, in OntologyLineageInput) error {
		ontologyCalls++
		ontologyMaps = append(ontologyMaps, in.EntityMaps...)
		return nil
	})

	if _, err := svc.UpsertCredential(context.Background(), UpsertCredentialInput{
		TenantID:     "sup-shadow-1",
		Provider:     ProviderBillz,
		AccountLabel: "default",
		AuthType:     AuthTypeBearer,
		SecretRef:    "env://CONNECTOR_SECRET_BILLZ_SHADOW_TOKEN",
		BaseURL:      srv.URL,
		CreatedBy:    "owner-1",
	}); err != nil {
		t.Fatalf("UpsertCredential: %v", err)
	}

	res, err := svc.ShadowCompare(context.Background(), TriggerConnectorInput{
		TenantID:       "sup-shadow-1",
		Provider:       ProviderBillz,
		IdempotencyKey: "shadow-idem-1",
		ActorID:        "owner-1",
	})
	if err != nil {
		t.Fatalf("ShadowCompare: %v", err)
	}

	if res.Run.Status != RunStatusMismatchFound {
		t.Fatalf("status=%s want=%s", res.Run.Status, RunStatusMismatchFound)
	}
	if res.Summary.MissingInternal != 1 || res.Summary.MissingExternal != 2 ||
		res.Summary.StockDeltaCount != 1 || res.Summary.BalanceDeltaCount != 1 ||
		res.Summary.NetStockDelta != 15 || res.Summary.NetBalanceDeltaMinor != 250000 {
		t.Fatalf("unexpected shadow summary: %+v", res.Summary)
	}
	if res.Summary.TotalMismatches != 5 || len(res.Mismatches) != 5 {
		t.Fatalf("mismatches=%d summaryTotal=%d list=%+v", len(res.Mismatches), res.Summary.TotalMismatches, res.Mismatches)
	}

	// Verify outbox emissions on TopicConnectors: connector.sync.completed + connector.shadow.mismatch.
	outboxEvs := repo.EmittedOutbox()
	if len(outboxEvs) != 2 || outboxEvs[0].Topic != events.TopicConnectors || outboxEvs[1].Topic != events.TopicConnectors {
		t.Fatalf("expected 2 TopicConnectors outbox emissions, got %+v", outboxEvs)
	}
	if ontologyCalls != 1 || len(ontologyMaps) == 0 {
		t.Fatalf("expected ontology lineage callback, calls=%d maps=%d", ontologyCalls, len(ontologyMaps))
	}
	if len(auditRec.events) < 2 {
		t.Fatalf("expected audit events recorded, got %d", len(auditRec.events))
	}

	// Verify idempotent replay returns same RunID without re-emitting.
	replay, err := svc.ShadowCompare(context.Background(), TriggerConnectorInput{
		TenantID:       "sup-shadow-1",
		Provider:       ProviderBillz,
		IdempotencyKey: "shadow-idem-1",
		ActorID:        "owner-1",
	})
	if err != nil {
		t.Fatalf("ShadowCompare replay: %v", err)
	}
	if replay.Run.RunID != res.Run.RunID {
		t.Fatalf("replay RunID=%s want=%s", replay.Run.RunID, res.Run.RunID)
	}
	if len(repo.EmittedOutbox()) != 2 {
		t.Fatalf("idempotent replay must not emit duplicate outbox events")
	}

	// Verify failed ShadowCompare persists FAILED run and emits connector.sync.failed on TopicConnectors.
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer failSrv.Close()
	_, _ = svc.UpsertCredential(context.Background(), UpsertCredentialInput{
		TenantID:     "sup-shadow-fail",
		Provider:     ProviderBillz,
		AccountLabel: "default",
		AuthType:     AuthTypeBearer,
		SecretRef:    "env://CONNECTOR_SECRET_BILLZ_SHADOW_TOKEN",
		BaseURL:      failSrv.URL,
		CreatedBy:    "owner-1",
	})
	if _, err := svc.ShadowCompare(context.Background(), TriggerConnectorInput{
		TenantID:       "sup-shadow-fail",
		Provider:       ProviderBillz,
		IdempotencyKey: "shadow-fail-1",
		ActorID:        "owner-1",
	}); !errors.Is(err, ErrConnectorAuthFailed) {
		t.Fatalf("expected ErrConnectorAuthFailed, got %v", err)
	}
	failRuns, _ := svc.ListRuns(context.Background(), "sup-shadow-fail", ProviderBillz, 10)
	if len(failRuns) != 1 || failRuns[0].Status != RunStatusFailed {
		t.Fatalf("expected FAILED run persisted, got %+v", failRuns)
	}
}

func TestOnecConnector_ShadowCompare_EndToEnd(t *testing.T) {
	xmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<КоммерческаяИнформация ВерсияСхемы="2.10">
  <Каталог><Товары><Товар><Ид>SKU-1C-1</Ид><Артикул>SKU-1C-1</Артикул><Наименование>Flour 1C</Наименование><Штрихкод>478999</Штрихкод></Товар></Товары></Каталог>
  <ПакетПредложений><Предложения><Предложение><Ид>SKU-1C-1</Ид><Артикул>SKU-1C-1</Артикул><Цены><Цена><ЦенаЗаЕдиницу>18000</ЦенаЗаЕдиницу><Валюта>UZS</Валюта></Цена></Цены><Склад ИдСклада="WH-1C" КоличествоНаСкладе="120"/></Предложение></Предложения></ПакетПредложений>
  <Контрагенты><Контрагент><Ид>RET-1C</Ид><ОфициальноеНаименование>1C Shop</ОфициальноеНаименование></Контрагент></Контрагенты>
  <Документ><Ид>ORD-1C-1</Ид><Номер>ORD-1C-1</Номер><Сумма>180000</Сумма><Контрагенты><Контрагент><Ид>RET-1C</Ид></Контрагент></Контрагенты></Документ>
</КоммерческаяИнформация>`

	onecSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(xmlDoc))
	}))
	defer onecSrv.Close()

	t.Setenv("CONNECTOR_SECRET_ONEC_SC_PASS", "u:p")
	repo := NewMemoryRepository()
	repo.SeedInternalSnapshot("sup-1c-sc", InternalSnapshot{
		Products: []InternalProduct{
			{ProductID: "SKU-1C-1", Name: "Flour 1C", Barcode: "478999", PriceMinor: 1800000, StockQuantity: 100, Currency: "UZS", IsActive: true},
		},
		Stocks: []InternalStock{
			{WarehouseID: "WH-1C", ProductID: "SKU-1C-1", QuantityOnHand: 100},
		},
		Orders: []InternalOrder{
			{OrderID: "ORD-1C-1", RetailerID: "RET-1C", TotalMinor: 18000000, Currency: "UZS", Status: "OPEN"},
		},
	})
	svc := NewServiceWithRepository(repo)
	svc.SetHTTPClient(onecSrv.Client())
	_, err := svc.UpsertCredential(context.Background(), UpsertCredentialInput{
		TenantID:     "sup-1c-sc",
		Provider:     ProviderOnec,
		AccountLabel: "branch-uz",
		AuthType:     AuthTypeBasic,
		SecretRef:    "env://CONNECTOR_SECRET_ONEC_SC_PASS",
		BaseURL:      onecSrv.URL,
		CreatedBy:    "owner-1",
	})
	if err != nil {
		t.Fatalf("UpsertCredential: %v", err)
	}
	res, err := svc.ShadowCompare(context.Background(), TriggerConnectorInput{
		TenantID:       "sup-1c-sc",
		Provider:       ProviderOnec,
		AccountLabel:   "branch-uz",
		IdempotencyKey: "1c-sc-1",
		ActorID:        "owner-1",
	})
	if err != nil {
		t.Fatalf("1C ShadowCompare: %v", err)
	}
	if res.Summary.StockDeltaCount != 1 || res.Summary.NetStockDelta != 20 {
		t.Fatalf("expected 1C stock delta +20, got %+v", res.Summary)
	}
}

func TestConnectors_SSRFBlockedByDefault(t *testing.T) {
	t.Setenv("CONNECTOR_ALLOW_PRIVATE_HOSTS", "")
	t.Setenv("BILLZ_API_BASE_URL", "")
	t.Setenv("BILLZ_API_TOKEN", "ssrf-tok")
	t.Setenv("CONNECTOR_SECRET_SSRF_TOK", "ssrf-tok")

	blockedURLs := []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://metadata.google.internal/computeMetadata/v1/",
		"https://kubernetes.default.svc/api",
		"http://127.0.0.1:8080/internal",
		"http://localhost:9010/",
		"http://10.0.0.5/erp",
		"http://192.168.1.10/api",
		"https://user:pass@erp.example.uz/api",
		"file:///etc/passwd",
	}

	svc := NewServiceWithRepository(NewMemoryRepository())
	billz := NewBillzConnector(NewDefaultSecretResolver(nil), nil)

	for _, rawURL := range blockedURLs {
		if _, err := svc.UpsertCredential(context.Background(), UpsertCredentialInput{
			TenantID:  "sup-ssrf",
			Provider:  ProviderBillz,
			AuthType:  AuthTypeBearer,
			SecretRef: "env://CONNECTOR_SECRET_SSRF_TOK",
			BaseURL:   rawURL,
		}); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("UpsertCredential(%q) err=%v, want ErrInvalidInput", rawURL, err)
		}
		if _, err := billz.Health(context.Background(), ExternalCredential{
			Provider:  ProviderBillz,
			AuthType:  AuthTypeBearer,
			SecretRef: "env://CONNECTOR_SECRET_SSRF_TOK",
			BaseURL:   rawURL,
		}); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("Billz Health(%q) err=%v, want ErrInvalidInput", rawURL, err)
		}
	}
}
