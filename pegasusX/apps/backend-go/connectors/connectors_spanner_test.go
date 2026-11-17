package connectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openConnectorTestSpanner(t *testing.T) *spanner.Client {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1" {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner connector test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, "projects/pegasusx-local/instances/pegasusx-instance/databases/pegasusx-db",
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("open spanner: %v", err)
	}
	ro := client.ReadOnlyTransaction()
	_, pErr := ro.ReadRow(ctx, "ExternalCredentials", spanner.Key{"__conn_probe__", "__probe__"}, []string{"Provider"})
	ro.Close()
	if pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
		client.Close()
		t.Fatalf("ExternalCredentials table unavailable: %v", pErr)
	}
	t.Cleanup(client.Close)
	return client
}

func TestLiveSpanner_Connectors_BillzShadowCompare_Onec_Idempotency(t *testing.T) {
	client := openConnectorTestSpanner(t)
	ctx := context.Background()

	tag := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	supID := "sup-conn-" + tag
	whID := "wh-conn-" + tag
	prod1 := "prod1-conn-" + tag
	prod2 := "prod2-conn-" + tag
	ord1 := "ord1-conn-" + tag
	inv1 := "inv1-conn-" + tag
	ret1 := "ret1-conn-" + tag

	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var outboxMuts []*spanner.Mutation
		ro := client.ReadOnlyTransaction()
		oIter := ro.Query(cleanCtx, spanner.Statement{
			SQL:    `SELECT EventId FROM OutboxEvents WHERE SupplierId = @tid`,
			Params: map[string]any{"tid": supID},
		})
		for {
			r, err := oIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var eid string
			if r.Columns(&eid) == nil {
				outboxMuts = append(outboxMuts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
			}
		}
		oIter.Stop()
		ro.Close()
		muts := append(outboxMuts,
			spanner.Delete("ConnectorEntityMaps", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ConnectorSyncRuns", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ExternalCredentials", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ArInvoices", spanner.Key{inv1}),
			spanner.Delete("Orders", spanner.Key{ord1}),
			spanner.Delete("SupplierInventoryV2", spanner.KeyRange{Start: spanner.Key{supID, whID}, End: spanner.Key{supID, whID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("Products", spanner.Key{prod1}),
			spanner.Delete("Products", spanner.Key{prod2}),
		)
		_, _ = client.Apply(cleanCtx, muts)
	})

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId": prod1, "SupplierId": supID, "CategoryId": "cat-1",
			"Name": "Live Tea", "Barcode": "BC-" + tag, "PriceMinor": int64(1500000), "Currency": "UZS",
			"StockQuantity": int64(20), "IsActive": true, "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId": prod2, "SupplierId": supID, "CategoryId": "cat-1",
			"Name": "Missing In Billz", "PriceMinor": int64(100000), "Currency": "UZS",
			"StockQuantity": int64(5), "IsActive": true, "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId": supID, "WarehouseId": whID, "ProductId": prod1,
			"QuantityOnHand": int64(20), "QuantityReserved": int64(0), "ReorderThreshold": int64(2),
			"UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("Orders", map[string]any{
			"OrderId": ord1, "SupplierId": supID, "RetailerId": ret1,
			"Status": "COMPLETED", "OrderSource": "DIRECT", "ConfirmationStatus": "CONFIRMED",
			"LineItemsJson": []byte("[]"), "TotalMinor": int64(400000), "OriginalTotalMinor": int64(400000),
			"Currency": "UZS", "Version": int64(1), "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("ArInvoices", map[string]any{
			"InvoiceId": inv1, "SupplierId": supID, "RetailerId": ret1, "OrderId": ord1,
			"Status": "OPEN", "PrincipalMinor": int64(400000), "BalanceMinor": int64(400000),
			"Currency": "UZS", "CreditLeaveAt": time.Now().UTC(), "DueAt": time.Now().UTC().Add(7 * 24 * time.Hour),
			"TermsDays": int64(7), "GracePeriodDays": int64(0), "DunningStep": int64(0),
			"Version": int64(1), "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed Spanner tables: %v", err)
	}

	billzSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/shops":
			_, _ = w.Write([]byte(`{"shops":[{"id":"` + whID + `","name":"WH"}]}`))
		case "/v1/products":
			_, _ = w.Write([]byte(`{"products":[{"id":"ext-p1","sku":"` + prod1 + `","barcode":"BC-` + tag + `","name":"Live Tea","price_minor":1500000,"currency":"UZS"}]}`))
		case "/v1/stocks":
			_, _ = w.Write([]byte(`{"stocks":[{"product_id":"ext-p1","warehouse_id":"` + whID + `","quantity_on_hand":35}]}`))
		case "/v1/orders":
			_, _ = w.Write([]byte(`{"orders":[{"id":"` + ord1 + `","order_id":"` + ord1 + `","client_id":"` + ret1 + `","total_minor":400000,"currency":"UZS"}]}`))
		case "/v1/clients":
			_, _ = w.Write([]byte(`{"clients":[{"id":"` + ret1 + `","name":"Shop","balance_minor":550000,"currency":"UZS"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer billzSrv.Close()

	t.Setenv("CONNECTOR_SECRET_LIVE_BILLZ_TOKEN_"+tag, "billz-live-tok")
	svc := NewService(client)
	svc.SetHTTPClient(billzSrv.Client())
	auditRec := &stubAuditRecorder{}
	svc.SetAuditRecorder(auditRec)
	var lineageUpdated int
	svc.SetOntologyLineageHook(func(_ context.Context, in OntologyLineageInput) error {
		lineageUpdated += len(in.EntityMaps)
		return nil
	})

	credSum, err := svc.UpsertCredential(ctx, UpsertCredentialInput{
		TenantID:     supID,
		Provider:     ProviderBillz,
		AccountLabel: "default",
		AuthType:     AuthTypeBearer,
		SecretRef:    "env://CONNECTOR_SECRET_LIVE_BILLZ_TOKEN_" + tag,
		BaseURL:      billzSrv.URL,
		CreatedBy:    "owner-live",
	})
	if err != nil {
		t.Fatalf("UpsertCredential: %v", err)
	}
	if strings.Contains(credSum.SecretRefMasked, tag) && len(tag) > 4 && !strings.HasPrefix(credSum.SecretRefMasked, "env://****") {
		t.Fatalf("unexpected masked ref: %s", credSum.SecretRefMasked)
	}

	hRes, err := svc.CheckHealth(ctx, TriggerConnectorInput{TenantID: supID, Provider: ProviderBillz, ActorID: "owner-live"})
	if err != nil || !hRes.Healthy {
		t.Fatalf("CheckHealth failed: res=%+v err=%v", hRes, err)
	}

	scRes, err := svc.ShadowCompare(ctx, TriggerConnectorInput{
		TenantID:       supID,
		Provider:       ProviderBillz,
		IdempotencyKey: "live-sc-" + tag,
		ActorID:        "owner-live",
	})
	if err != nil {
		t.Fatalf("ShadowCompare: %v", err)
	}
	if scRes.Summary.MissingExternal != 1 || scRes.Summary.StockDeltaCount != 1 || scRes.Summary.NetStockDelta != 15 ||
		scRes.Summary.BalanceDeltaCount != 1 || scRes.Summary.NetBalanceDeltaMinor != 150000 {
		t.Fatalf("unexpected live shadow summary: %+v", scRes.Summary)
	}
	if lineageUpdated == 0 {
		t.Fatal("expected ontology lineage callback with matched entity maps")
	}

	// Verify replay returns identical run.
	scReplay, err := svc.ShadowCompare(ctx, TriggerConnectorInput{
		TenantID:       supID,
		Provider:       ProviderBillz,
		IdempotencyKey: "live-sc-" + tag,
		ActorID:        "owner-live",
	})
	if err != nil || scReplay.Run.RunID != scRes.Run.RunID {
		t.Fatalf("idempotent replay mismatch: err=%v got=%s want=%s", err, scReplay.Run.RunID, scRes.Run.RunID)
	}

	// Verify OutboxEvents persisted on TopicConnectors.
	ro := client.ReadOnlyTransaction()
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TopicName, AggregateType, AggregateId
		      FROM OutboxEvents
		      WHERE SupplierId = @tid AND TopicName = @topic`,
		Params: map[string]any{"tid": supID, "topic": events.TopicConnectors},
	})
	defer iter.Stop()
	var outboxCount int
	for {
		_, nextErr := iter.Next()
		if nextErr == iterator.Done {
			break
		}
		if nextErr != nil {
			t.Fatalf("query OutboxEvents: %v", nextErr)
		}
		outboxCount++
	}
	if outboxCount < 3 {
		t.Fatalf("expected >=3 TopicConnectors OutboxEvents rows (health + sync.completed + shadow.mismatch), got %d", outboxCount)
	}

	// Verify GetRun fetches persisted summary and mismatches.
	fetched, err := svc.GetRun(ctx, supID, scRes.Run.RunID)
	if err != nil || fetched.Summary == nil || len(fetched.Mismatches) != 3 {
		t.Fatalf("GetRun failed: err=%v fetched=%+v", err, fetched)
	}
}

type spannerTestApprovalChecker struct {
	approvedID string
}

func (c *spannerTestApprovalChecker) VerifyPushApproval(_ context.Context, in PushApprovalInput) error {
	if in.ApprovalID == "" || in.ApprovalID != c.approvedID {
		return ErrApprovalRequired
	}
	return nil
}

func TestLiveSpanner_Connectors_TwoWayPush_And_CISAdapters(t *testing.T) {
	client := openConnectorTestSpanner(t)
	ctx := context.Background()

	tag := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	supID := "sup-conn-w4a-" + tag
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var outboxMuts []*spanner.Mutation
		ro := client.ReadOnlyTransaction()
		oIter := ro.Query(cleanCtx, spanner.Statement{
			SQL:    `SELECT EventId FROM OutboxEvents WHERE SupplierId = @tid`,
			Params: map[string]any{"tid": supID},
		})
		for {
			r, err := oIter.Next()
			if err != nil {
				break
			}
			var eid string
			if r.Columns(&eid) == nil {
				outboxMuts = append(outboxMuts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
			}
		}
		oIter.Stop()
		ro.Close()
		muts := append(outboxMuts,
			spanner.Delete("ConnectorEntityMaps", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ConnectorSyncRuns", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ExternalCredentials", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
		)
		_, _ = client.Apply(cleanCtx, muts)
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"document_id":"spn-push-doc-1"}`))
			return
		}
		switch r.URL.Path {
		case "/entity/organization", "/v1/warehouses", "/api/1/organizations":
			_, _ = w.Write([]byte(`{"rows":[{"id":"wh-1","name":"Main"}],"warehouses":[{"id":"wh-1","name":"Main"}],"organizations":[{"id":"wh-1","name":"Main"}]}`))
		default:
			_, _ = w.Write([]byte(`{"rows":[],"products":[],"stocks":[],"orders":[],"clients":[],"customers":[]}`))
		}
	}))
	defer srv.Close()

	t.Setenv("CONNECTOR_SECRET_W4A_SPN_TOK_"+tag, "spn-tok-val")
	svc := NewService(client)
	svc.SetHTTPClient(srv.Client())
	svc.SetApprovalChecker(&spannerTestApprovalChecker{approvedID: "apr-spn-" + tag})

	for _, prov := range []string{ProviderBillz, ProviderMoySklad, ProviderSmartup, ProviderIiko} {
		if _, err := svc.UpsertCredential(ctx, UpsertCredentialInput{
			TenantID: supID, Provider: prov, AccountLabel: "default",
			AuthType: AuthTypeBearer, SecretRef: "env://CONNECTOR_SECRET_W4A_SPN_TOK_" + tag, BaseURL: srv.URL, CreatedBy: "owner-spn",
		}); err != nil {
			t.Fatalf("UpsertCredential %s: %v", prov, err)
		}
		res, err := svc.Push(ctx, TriggerConnectorInput{
			TenantID: supID, Provider: prov, Stream: "goods_receipt", ApprovalID: "apr-spn-" + tag,
			IdempotencyKey: "idem-spn-" + prov + "-" + tag,
			Records:        []CanonicalRecord{{ExternalID: "SKU-SPN-1", SKU: "SKU-SPN-1", QuantityOnHand: 7}},
			ActorID:        "owner-spn", ActorRole: "OWNER", ActorType: "human",
		})
		if err != nil || res.PushResult.ExternalDocID != "spn-push-doc-1" {
			t.Fatalf("%s Push live Spanner failed: res=%+v err=%v", prov, res, err)
		}
	}

	// Verify 4 connector.push.completed events persisted on TopicConnectors in OutboxEvents.
	ro := client.ReadOnlyTransaction()
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT Payload FROM OutboxEvents WHERE SupplierId = @tid AND TopicName = @topic`,
		Params: map[string]any{"tid": supID, "topic": events.TopicConnectors},
	})
	defer iter.Stop()
	pushCount := 0
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("query OutboxEvents: %v", err)
		}
		var payload []byte
		if row.Columns(&payload) == nil && strings.Contains(string(payload), events.EventConnectorPushCompleted) {
			pushCount++
		}
	}
	if pushCount != 4 {
		t.Fatalf("expected 4 connector.push.completed OutboxEvents rows, got %d", pushCount)
	}
}
