package partner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

func TestIssueKeyAndAuthMiddleware(t *testing.T) {
	keys := NewMemoryKeyRepository()
	hooks := NewMemoryWebhookRepository()
	svc := NewService(keys, hooks, nil, nil, nil)
	issued, err := svc.IssueKey(context.Background(), TenantRetailer, "ret-1", "admin", nil)
	if err != nil {
		t.Fatal(err)
	}
	if issued.Secret == "" || issued.KeyID == "" {
		t.Fatal("missing secret")
	}

	var got Principal
	h := AuthMiddleware(keys)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFromContext(r.Context())
		if !ok {
			t.Fatal("no principal")
		}
		got = p
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/partner/v1/catalog", nil)
	req.Header.Set("Authorization", "Bearer "+issued.Secret)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rr.Code)
	}
	if got.TenantID != "ret-1" || got.TenantType != TenantRetailer {
		t.Fatalf("principal=%+v", got)
	}
}

func TestIDORGetOrder(t *testing.T) {
	// Minimal stub: GetOrder without real order service returns unavailable;
	// canAccessOrder unit-tested via service method with injected order.
	svc := &Service{}
	p := Principal{TenantType: TenantRetailer, TenantID: "ret-a"}
	o := order.Order{OrderID: "o1", RetailerID: "ret-b", SupplierID: "sup-1"}
	if svc.canAccessOrder(p, o) {
		t.Fatal("expected IDOR deny")
	}
	o.RetailerID = "ret-a"
	if !svc.canAccessOrder(p, o) {
		t.Fatal("expected allow")
	}
}

func TestWebhookEnqueueAndDeliver(t *testing.T) {
	keys := NewMemoryKeyRepository()
	hooks := NewMemoryWebhookRepository()
	svc := NewService(keys, hooks, nil, nil, nil)
	delivery := NewDeliveryWorker(hooks, nil)

	var received map[string]any
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-Pegasus-Signature")
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := Principal{TenantType: TenantSupplier, TenantID: "sup-1", Scopes: []string{ScopeWebhooksManage}}
	sub, secret, err := svc.CreateWebhookSubscription(context.Background(), p, srv.URL, []string{"ORDER_CREATED"})
	if err != nil {
		t.Fatal(err)
	}
	if secret == "" {
		t.Fatal("secret")
	}
	err = svc.EnqueueEvent(context.Background(), "evt-1", "ORDER_CREATED", map[string]any{
		"type": "ORDER_CREATED", "order_id": "o1", "supplier_id": "sup-1", "retailer_id": "ret-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err := delivery.RunOnce(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("delivered n=%d err=%v", n, err)
	}
	if received["order_id"] != "o1" {
		t.Fatalf("payload=%v", received)
	}
	if gotSig == "" || sub.SubscriptionID == "" {
		t.Fatal("missing signature")
	}
	// idempotent re-enqueue
	_ = svc.EnqueueEvent(context.Background(), "evt-1", "ORDER_CREATED", map[string]any{
		"type": "ORDER_CREATED", "order_id": "o1", "supplier_id": "sup-1",
	})
	n2, _ := delivery.RunOnce(context.Background())
	if n2 != 0 {
		t.Fatalf("expected no redelivery, got %d", n2)
	}
}

func TestPingWebhook(t *testing.T) {
	hooks := NewMemoryWebhookRepository()
	svc := NewService(NewMemoryKeyRepository(), hooks, nil, nil, nil)
	delivery := NewDeliveryWorker(hooks, nil)
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(200)
	}))
	defer srv.Close()
	p := Principal{TenantType: TenantRetailer, TenantID: "ret-1", Scopes: []string{"*"}}
	sub, _, err := svc.CreateWebhookSubscription(context.Background(), p, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.PingWebhook(context.Background(), p, sub.SubscriptionID, delivery.DeliverHTTP); err != nil {
		t.Fatal(err)
	}
	if !hit {
		t.Fatal("ping not delivered")
	}
	_ = time.Second
}

func TestWebhookReplayAndDeactivate(t *testing.T) {
	hooks := NewMemoryWebhookRepository()
	svc := NewService(NewMemoryKeyRepository(), hooks, nil, nil, nil)
	delivery := NewDeliveryWorker(hooks, nil)

	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := Principal{TenantType: TenantSupplier, TenantID: "sup-1", Scopes: []string{ScopeWebhooksManage}}
	sub, _, err := svc.CreateWebhookSubscription(context.Background(), p, srv.URL, []string{"ORDER_CREATED"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	att := DeliveryAttempt{
		AttemptID:      "att-dead-1",
		SubscriptionID: sub.SubscriptionID,
		EventID:        "evt-dead-1",
		EventType:      "ORDER_CREATED",
		PayloadJSON:    []byte(`{"type":"ORDER_CREATED","order_id":"o1","supplier_id":"sup-1"}`),
		Status:         DeliveryDead,
		AttemptCount:   8,
		LastError:      "gone",
		NextRetryAt:    &now,
		CreatedAt:      now,
	}
	if err := hooks.InsertAttempt(context.Background(), att); err != nil {
		t.Fatal(err)
	}

	replayed, err := svc.ReplayDeadLetter(context.Background(), p, att.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Status != DeliveryPending || replayed.AttemptCount != 0 {
		t.Fatalf("replay state=%+v", replayed)
	}
	n, err := delivery.RunOnce(context.Background())
	if err != nil || n != 1 || hits != 1 {
		t.Fatalf("deliver after replay n=%d hits=%d err=%v", n, hits, err)
	}

	if err := svc.DeactivateWebhook(context.Background(), p, sub.SubscriptionID); err != nil {
		t.Fatal(err)
	}
	subs, err := svc.ListWebhooks(context.Background(), p)
	if err != nil || len(subs) != 1 || subs[0].IsActive {
		t.Fatalf("expected inactive sub, got %+v err=%v", subs, err)
	}
}

func TestExportJobSkippedSFTP(t *testing.T) {
	t.Setenv("PARTNER_EXPORTS_ENABLED", "true")
	t.Setenv("PARTNER_SFTP_ENABLED", "false")
	root := t.TempDir()
	t.Setenv("PARTNER_EXPORT_LOCAL_ROOT", root)

	exports := NewMemoryExportRepository()
	sftp := NewMemorySftpConfigRepository()
	svc := NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, nil, nil)
	svc.SetExportRepos(exports, sftp)
	worker := NewExportWorker(exports, sftp, nil, nil)

	p := Principal{TenantType: TenantSupplier, TenantID: "sup-exp", Scopes: []string{ScopeExportsRead}}
	j, err := svc.CreateExportJob(context.Background(), p, ExportResourceOrders, ExportFormatCSV, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	n, err := worker.RunOnce(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("worker n=%d err=%v", n, err)
	}
	got, dl, err := svc.GetExportJob(context.Background(), p, j.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ExportStatusSucceeded || got.SftpStatus != SftpStatusSkipped {
		t.Fatalf("job=%+v", got)
	}
	if dl == "" {
		t.Fatal("expected download_url")
	}
}
