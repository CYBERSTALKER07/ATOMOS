package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/connectors"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/pegasusx/pegasusx/apps/backend-go/multientity"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/p2p"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func TestSetupWave4A_MemoryFallbackAndBridges(t *testing.T) {
	ctx := context.Background()
	auditSvc := audit.NewServiceWithRepository(audit.NewMemoryRepository())
	glSvc := gl.NewServiceWithRepository(gl.NewMemoryRepository())
	appSvc := approvals.NewServiceWithRepository(approvals.NewMemoryRepository())

	tenantID := "sup-w4a-mem-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	if err := glSvc.SeedDefaults(ctx, tenantID, "UZ", "UZS"); err != nil {
		t.Fatalf("glSvc.SeedDefaults: %v", err)
	}

	app := &App{
		Audit:     auditSvc,
		GL:        glSvc,
		Approvals: appSvc,
	}
	cfg := &Config{
		AllowMemoryFallback: true,
		JWTSecret:           "wave4a-wiring-memory-secret-123456",
		JWTIssuer:           "pegasusx-wave4a-test",
	}
	if err := SetupWave3C(ctx, cfg, nil, app, nil); err != nil {
		t.Fatalf("SetupWave3C(memory): %v", err)
	}
	if err := SetupWave4A(ctx, cfg, nil, app, nil); err != nil {
		t.Fatalf("SetupWave4A(memory): %v", err)
	}
	t.Cleanup(func() {
		glSvc.SetLegalEntityOwnershipChecker(nil)
	})

	if app.MultiEntity == nil || app.P2P == nil || app.Connectors == nil {
		t.Fatalf("expected MultiEntity, P2P, and Connectors initialized in memory fallback")
	}

	// 1. Multi-Entity + GL legal entity ownership hook
	ent, err := app.MultiEntity.CreateEntity(ctx, multientity.CreateEntityInput{
		TenantID:     tenantID,
		EntityID:     "ent-uz-1",
		Name:         "Tashkent Distribution LLC",
		TIN:          "301234567",
		CountryCode:  "UZ",
		BaseCurrency: "UZS",
		IsParent:     true,
		ActorID:      "owner-1",
		ActorType:    "human",
		Role:         auth.SupplierRoleOwner,
	})
	if err != nil {
		t.Fatalf("CreateEntity(memory): %v", err)
	}
	if ent.EntityID != "ent-uz-1" {
		t.Fatalf("unexpected entity: %+v", ent)
	}
	owned, err := app.MultiEntity.IsLegalEntityOwned(ctx, tenantID, "ent-uz-1")
	if err != nil || !owned {
		t.Fatalf("expected owned=true, err=nil; got owned=%v err=%v", owned, err)
	}
	if owned, err := glSvc.IsLegalEntityOwned(ctx, tenantID, "ent-uz-1"); err != nil || !owned {
		t.Fatalf("expected glSvc.IsLegalEntityOwned(ent-uz-1)=true, got (%v, %v)", owned, err)
	}
	if owned, err := glSvc.IsLegalEntityOwned(ctx, tenantID, "ent-other-tenant"); err != nil || owned {
		t.Fatalf("expected glSvc.IsLegalEntityOwned(ent-other-tenant)=false, got (%v, %v)", owned, err)
	}

	// 2. P2P + GL + Audit bridge (PO -> Receipt -> VendorInvoice -> 3-way match)
	poRes, err := app.P2P.CreatePurchaseOrder(ctx, p2p.CreatePurchaseOrderInput{
		TenantID:       tenantID,
		EntityID:       "ent-uz-1",
		VendorID:       "ven-w4a-1",
		VendorName:     "Samarqand Dairy",
		VendorTin:      "309998877",
		Currency:       "UZS",
		IdempotencyKey: "po-w4a-mem-1",
		CreatedBy:      "proc-1",
		ActorType:      "human",
		ActorRole:      auth.SupplierRoleAccountant,
		Lines: []p2p.PurchaseOrderLineInput{{
			Sku:           "SKU-DAIRY-1",
			ProductName:   "Kefir 1L",
			QtyOrdered:    10,
			UnitCostMinor: 100000,
			VatBps:        1200,
		}},
	})
	if err != nil {
		t.Fatalf("CreatePurchaseOrder(memory): %v", err)
	}
	grRes, err := app.P2P.PostGoodsReceipt(ctx, p2p.PostGoodsReceiptInput{
		TenantID:       tenantID,
		PoID:           poRes.PurchaseOrder.PoID,
		EntityID:       "ent-uz-1",
		WarehouseID:    "wh-1",
		IdempotencyKey: "gr-w4a-mem-1",
		ReceivedBy:     "wh-mgr-1",
		ActorType:      "human",
		ActorRole:      auth.SupplierRoleWarehouseManager,
		Lines:          []p2p.GoodsReceiptLineInput{{Sku: "SKU-DAIRY-1", QtyReceived: 10}},
	})
	if err != nil {
		t.Fatalf("PostGoodsReceipt(memory): %v", err)
	}
	if grRes.JournalEntryID == "" {
		t.Fatalf("expected GL JournalEntryID via NewP2PGLPoster on receipt")
	}
	inv, err := app.P2P.CreateVendorInvoice(ctx, p2p.CreateVendorInvoiceInput{
		TenantID:       tenantID,
		PoID:           poRes.PurchaseOrder.PoID,
		ReceiptID:      grRes.Receipt.ReceiptID,
		EntityID:       "ent-uz-1",
		VendorID:       "ven-w4a-1",
		Currency:       "UZS",
		SubtotalMinor:  1000000,
		VatMinor:       120000,
		TotalMinor:     1120000,
		IdempotencyKey: "vi-w4a-mem-1",
		CreatedBy:      "acc-1",
		ActorType:      "human",
		ActorRole:      auth.SupplierRoleAccountant,
	})
	if err != nil {
		t.Fatalf("CreateVendorInvoice(memory): %v", err)
	}
	matchRes, err := app.P2P.ThreeWayMatch(ctx, p2p.ThreeWayMatchInput{
		TenantID:        tenantID,
		VendorInvoiceID: inv.VendorInvoiceID,
		PoID:            poRes.PurchaseOrder.PoID,
		ReceiptID:       grRes.Receipt.ReceiptID,
		VerifiedBy:      "cfo-1",
		ActorType:       "human",
		ActorRole:       auth.SupplierRoleOwner,
	})
	if err != nil {
		t.Fatalf("ThreeWayMatch(memory): %v", err)
	}
	if !matchRes.Matched || matchRes.JournalEntryID == "" {
		t.Fatalf("expected MATCHED_3WAY with GL entry, got %+v", matchRes)
	}

	// 3. Connectors push approval bridge: fail closed without approval, succeed with approved ConnectorPush
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","doc_id":"BLZ-MEM-1"}`))
	}))
	defer srv.Close()
	app.Connectors.SetHTTPClient(srv.Client())

	t.Setenv("CONNECTOR_SECRET_MEM_W4A_BILLZ_TOKEN", "mem-billz-token")
	if _, err := app.Connectors.UpsertCredential(ctx, connectors.UpsertCredentialInput{
		TenantID:     tenantID,
		Provider:     connectors.ProviderBillz,
		AccountLabel: "default",
		AuthType:     connectors.AuthTypeBearer,
		SecretRef:    "env://CONNECTOR_SECRET_MEM_W4A_BILLZ_TOKEN",
		BaseURL:      srv.URL,
		CreatedBy:    "owner-1",
		ActorType:    "human",
		ActorRole:    auth.SupplierRoleOwner,
	}); err != nil {
		t.Fatalf("UpsertCredential: %v", err)
	}

	rawPush := `{"warehouse_id":"wh-1","reference_id":"GR-1","lines":[{"sku":"SKU-1","qty":5}]}`
	pushRecords := []connectors.CanonicalRecord{{
		ObjectType:     connectors.ObjectTypeInventory,
		ExternalID:     "SKU-1",
		SKU:            "SKU-1",
		QuantityOnHand: 5,
		WarehouseID:    "wh-1",
	}}
	// Without approval_id -> ErrApprovalRequired
	if _, err := app.Connectors.Push(ctx, connectors.TriggerConnectorInput{
		TenantID:   tenantID,
		Provider:   connectors.ProviderBillz,
		Stream:     connectors.StreamStocks,
		Records:    pushRecords,
		RawPayload: []byte(rawPush),
		ActorID:    "owner-1",
		ActorType:  "human",
		ActorRole:  auth.SupplierRoleOwner,
	}); !errors.Is(err, connectors.ErrApprovalRequired) {
		t.Fatalf("expected ErrApprovalRequired when approval_id empty, got %v", err)
	}

	approvalPayload := connectors.BuildPushApprovalPayloadFromRecords(
		connectors.ProviderBillz, "default", connectors.StreamStocks, pushRecords, rawPush,
	)
	subRes, err := appSvc.SubmitRequest(ctx, approvals.SubmitRequestInput{
		TenantID:       tenantID,
		ObjectType:     "ConnectorPush",
		ObjectID:       connectors.ProviderBillz,
		Action:         "push",
		RequestedBy:    "acc-1",
		ActorType:      "human",
		Payload:        approvalPayload,
		IdempotencyKey: "conn-push-apr-mem-1",
		ForceCreate:    true,
	})
	if err != nil || subRes.Request == nil {
		t.Fatalf("SubmitRequest(ConnectorPush): err=%v res=%+v", err, subRes)
	}
	if _, err := appSvc.Decide(ctx, approvals.DecideInput{
		TenantID:     tenantID,
		RequestID:    subRes.Request.RequestID,
		ApproverID:   "owner-1",
		ActorType:    "human",
		ApproverRole: auth.SupplierRoleOwner,
		Decision:     approvals.DecisionApprove,
		Comment:      "approved goods receipt push",
	}); err != nil {
		t.Fatalf("Decide(ConnectorPush): %v", err)
	}

	// Mutated payload must return ErrApprovalPayloadMismatch
	if _, err := app.Connectors.Push(ctx, connectors.TriggerConnectorInput{
		TenantID:   tenantID,
		Provider:   connectors.ProviderBillz,
		Stream:     connectors.StreamStocks,
		ApprovalID: subRes.Request.RequestID,
		Records:    pushRecords,
		RawPayload: []byte(`{"warehouse_id":"wh-1","reference_id":"TAMPERED","lines":[{"sku":"SKU-1","qty":99}]}`),
		ActorID:    "owner-1",
		ActorType:  "human",
		ActorRole:  auth.SupplierRoleOwner,
	}); !errors.Is(err, connectors.ErrApprovalPayloadMismatch) {
		t.Fatalf("expected ErrApprovalPayloadMismatch on tampered payload, got %v", err)
	}

	// Valid approved push -> succeeds
	pushOut, err := app.Connectors.Push(ctx, connectors.TriggerConnectorInput{
		TenantID:       tenantID,
		Provider:       connectors.ProviderBillz,
		Stream:         connectors.StreamStocks,
		ApprovalID:     subRes.Request.RequestID,
		Records:        pushRecords,
		RawPayload:     []byte(rawPush),
		IdempotencyKey: "push-exec-mem-1",
		ActorID:        "owner-1",
		ActorType:      "human",
		ActorRole:      auth.SupplierRoleOwner,
	})
	if err != nil {
		t.Fatalf("approved Connectors.Push failed: %v", err)
	}
	if pushOut.PushResult.Status != connectors.RunStatusCompleted {
		t.Fatalf("unexpected push status: %+v", pushOut)
	}
}

func TestNewApp_WiresAllWave4AModulesAndHooks_LiveSpanner(t *testing.T) {
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1" {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping bootstrap live Spanner Wave 4A wiring test")
	}

	stubRuntimeConstructors(t)
	newRedisRuntimeAdapter = func(_ cache.RedisConfig) (redisRuntimeAdapter, error) {
		return &fakeRedisAdapter{}, nil
	}
	newKafkaRuntimePublisher = func(_ string, _ outbox.KafkaPublisherConfig) (kafkaRuntimePublisher, error) {
		return &fakeKafkaPublisher{}, nil
	}
	newKafkaRuntimeDLQWriter = func(_ string, _ string, _ kafkautil.ClientAuth) (kafkaRuntimeDLQWriter, error) {
		return &fakeKafkaDLQWriter{}, nil
	}
	newSpannerRuntimeClient = func(ctx context.Context, dsn string) (*spanner.Client, error) {
		c, err := spanner.NewClient(ctx, dsn,
			option.WithEndpoint(host),
			option.WithoutAuthentication(),
			option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		)
		if err != nil {
			return nil, err
		}
		for _, tbl := range []string{"LegalEntities", "PurchaseOrders", "VendorInvoices", "ConnectorSyncRuns"} {
			ro := c.ReadOnlyTransaction()
			col := "Status"
			if tbl == "VendorInvoices" {
				col = "MatchStatus"
			}
			_, eErr := ro.ReadRow(ctx, tbl, spanner.Key{"__bs_w4a__", "__probe__"}, []string{col})
			ro.Close()
			if eErr != nil && spanner.ErrCode(eErr) != codes.NotFound {
				c.Close()
				return nil, eErr
			}
		}
		return c, nil
	}

	cfg := testConfig()
	cfg.RunMode = RunModeAPI
	cfg.RequireInfraAdapters = false
	cfg.JWTSecret = "wave4a-live-spanner-bootstrap-secret-123456"
	cfg.JWTIssuer = "pegasusx-bootstrap-test"

	app, err := NewApp(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewApp(api + live Spanner Wave 4A): %v", err)
	}
	defer app.Close()

	if app.MultiEntity == nil || app.MultiEntity.Repo() == nil {
		t.Fatalf("expected app.MultiEntity wired with Spanner repository")
	}
	if app.P2P == nil || app.P2P.Repo() == nil {
		t.Fatalf("expected app.P2P wired with Spanner repository")
	}
	if app.Connectors == nil || app.Approvals == nil || app.GL == nil || app.Audit == nil {
		t.Fatalf("expected Connectors, Approvals, GL, Audit wired on app")
	}

	ctx := context.Background()
	tenantID := "sup-bs-w4a-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:10]

	t.Cleanup(func() {
		if app.Spanner == nil {
			return
		}
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var outboxMuts []*spanner.Mutation
		ro := app.Spanner.ReadOnlyTransaction()
		oIter := ro.Query(cleanCtx, spanner.Statement{
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
				outboxMuts = append(outboxMuts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
			}
		}
		oIter.Stop()
		ro.Close()

		kr := spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}
		muts := []*spanner.Mutation{
			spanner.Delete("ApprovalDecisions", kr),
			spanner.Delete("ApprovalRequests", kr),
			spanner.Delete("VendorInvoices", kr),
			spanner.Delete("GoodsReceipts", kr),
			spanner.Delete("PurchaseOrderLines", kr),
			spanner.Delete("PurchaseOrders", kr),
			spanner.Delete("ProcessInstances", kr),
			spanner.Delete("JournalPostings", kr),
			spanner.Delete("JournalEntries", kr),
			spanner.Delete("GlBalances", kr),
			spanner.Delete("GlAccounts", kr),
			spanner.Delete("LegalEntities", kr),
			spanner.Delete("ConnectorSyncRuns", kr),
			spanner.Delete("ExternalCredentials", kr),
			spanner.Delete("AuditEvents", kr),
		}
		muts = append(muts, outboxMuts...)
		_, _ = app.Spanner.Apply(cleanCtx, muts)
	})

	if err := app.GL.SeedDefaults(ctx, tenantID, "UZ", "UZS"); err != nil {
		t.Fatalf("app.GL.SeedDefaults: %v", err)
	}

	// 1. Live Spanner MultiEntity + AuditEvents + GL ownership verification
	ent, err := app.MultiEntity.CreateEntity(ctx, multientity.CreateEntityInput{
		TenantID:     tenantID,
		EntityID:     "ent-live-1",
		Name:         "Apex UZ HQ LLC",
		TIN:          "304567890",
		CountryCode:  "UZ",
		BaseCurrency: "UZS",
		IsParent:     true,
		ActorID:      "owner-bs-1",
		ActorType:    "human",
		Role:         auth.SupplierRoleOwner,
	})
	if err != nil {
		t.Fatalf("app.MultiEntity.CreateEntity: %v", err)
	}
	entEvents, err := app.Audit.ListEvents(ctx, audit.Filter{
		TenantID:   tenantID,
		ObjectType: "LegalEntity",
		ObjectID:   ent.EntityID,
		ViewerRole: "ADMIN",
	})
	if err != nil || len(entEvents) == 0 {
		t.Fatalf("expected AuditEvents via NewMultiEntityAuditRecorder: len=%d err=%v", len(entEvents), err)
	}
	if owned, err := app.GL.IsLegalEntityOwned(ctx, tenantID, ent.EntityID); err != nil || !owned {
		t.Fatalf("app.GL.IsLegalEntityOwned(owned) = (%v, %v), want (true, nil)", owned, err)
	}
	if _, err := app.GL.TrialBalanceForEntity(ctx, tenantID, ent.EntityID, ""); err != nil {
		t.Fatalf("TrialBalanceForEntity(owned): %v", err)
	}
	if owned, err := app.GL.IsLegalEntityOwned(ctx, tenantID, "ent-not-owned"); err != nil || owned {
		t.Fatalf("expected app.GL.IsLegalEntityOwned to reject unowned legal entity, got (%v, %v)", owned, err)
	}

	// 2. Live Spanner P2P + Transactional GL bridge + AuditEvents
	poRes, err := app.P2P.CreatePurchaseOrder(ctx, p2p.CreatePurchaseOrderInput{
		TenantID:       tenantID,
		EntityID:       ent.EntityID,
		VendorID:       "ven-bs-1",
		VendorName:     "Bukhara Mills",
		VendorTin:      "309112233",
		Currency:       "UZS",
		IdempotencyKey: "po-bs-w4a-" + tenantID,
		CreatedBy:      "acc-bs-1",
		ActorType:      "human",
		ActorRole:      auth.SupplierRoleAccountant,
		Lines: []p2p.PurchaseOrderLineInput{{
			Sku:           "SKU-FLOUR-50",
			ProductName:   "Wheat Flour 50kg",
			QtyOrdered:    4,
			UnitCostMinor: 250000,
			VatBps:        1200,
		}},
	})
	if err != nil {
		t.Fatalf("app.P2P.CreatePurchaseOrder: %v", err)
	}
	grRes, err := app.P2P.PostGoodsReceipt(ctx, p2p.PostGoodsReceiptInput{
		TenantID:       tenantID,
		PoID:           poRes.PurchaseOrder.PoID,
		EntityID:       ent.EntityID,
		WarehouseID:    "wh-live-1",
		IdempotencyKey: "gr-bs-w4a-" + tenantID,
		ReceivedBy:     "wh-mgr-bs-1",
		ActorType:      "human",
		ActorRole:      auth.SupplierRoleWarehouseManager,
		Lines:          []p2p.GoodsReceiptLineInput{{Sku: "SKU-FLOUR-50", QtyReceived: 4}},
	})
	if err != nil || grRes.JournalEntryID == "" {
		t.Fatalf("app.P2P.PostGoodsReceipt: err=%v res=%+v", err, grRes)
	}
	p2pEvents, err := app.Audit.ListEvents(ctx, audit.Filter{
		TenantID:   tenantID,
		ObjectType: "GoodsReceipt",
		ObjectID:   grRes.Receipt.ReceiptID,
		ViewerRole: "ADMIN",
	})
	if err != nil || len(p2pEvents) == 0 {
		t.Fatalf("expected AuditEvents via NewP2PAuditRecorder: len=%d err=%v", len(p2pEvents), err)
	}

	// 3. Live Spanner Connector Push approval verification via approvals.Service
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","doc_id":"BLZ-LIVE-W4A"}`))
	}))
	defer srv.Close()
	app.Connectors.SetHTTPClient(srv.Client())

	t.Setenv("CONNECTOR_SECRET_LIVE_W4A_BILLZ_TOKEN", "live-w4a-billz-token")
	if _, err := app.Connectors.UpsertCredential(ctx, connectors.UpsertCredentialInput{
		TenantID:     tenantID,
		Provider:     connectors.ProviderBillz,
		AccountLabel: "default",
		AuthType:     connectors.AuthTypeBearer,
		SecretRef:    "env://CONNECTOR_SECRET_LIVE_W4A_BILLZ_TOKEN",
		BaseURL:      srv.URL,
		CreatedBy:    "owner-bs-1",
		ActorType:    "human",
		ActorRole:    auth.SupplierRoleOwner,
	}); err != nil {
		t.Fatalf("UpsertCredential: %v", err)
	}
	rawPush := `{"warehouse_id":"wh-live-1","reference_id":"GR-BS-1","lines":[{"sku":"SKU-FLOUR-50","qty":4}]}`
	pushRecords := []connectors.CanonicalRecord{{
		ObjectType:     connectors.ObjectTypeInventory,
		ExternalID:     "SKU-FLOUR-50",
		SKU:            "SKU-FLOUR-50",
		QuantityOnHand: 4,
		WarehouseID:    "wh-live-1",
	}}
	pushApprovalPayload := connectors.BuildPushApprovalPayloadFromRecords(
		connectors.ProviderBillz, "default", connectors.StreamStocks, pushRecords, rawPush,
	)
	subRes, err := app.Approvals.SubmitRequest(ctx, approvals.SubmitRequestInput{
		TenantID:       tenantID,
		ObjectType:     "ConnectorPush",
		ObjectID:       connectors.ProviderBillz,
		Action:         "push",
		RequestedBy:    "acc-bs-1",
		ActorType:      "human",
		Payload:        pushApprovalPayload,
		IdempotencyKey: "conn-push-live-apr-" + tenantID,
		ForceCreate:    true,
	})
	if err != nil || subRes.Request == nil {
		t.Fatalf("app.Approvals.SubmitRequest: err=%v", err)
	}
	if _, err := app.Approvals.Decide(ctx, approvals.DecideInput{
		TenantID:     tenantID,
		RequestID:    subRes.Request.RequestID,
		ApproverID:   "owner-bs-1",
		ActorType:    "human",
		ApproverRole: auth.SupplierRoleOwner,
		Decision:     approvals.DecisionApprove,
		Comment:      "approved billz sync",
	}); err != nil {
		t.Fatalf("app.Approvals.Decide: %v", err)
	}

	pushOut, err := app.Connectors.Push(ctx, connectors.TriggerConnectorInput{
		TenantID:       tenantID,
		Provider:       connectors.ProviderBillz,
		Stream:         connectors.StreamStocks,
		ApprovalID:     subRes.Request.RequestID,
		Records:        pushRecords,
		RawPayload:     []byte(rawPush),
		IdempotencyKey: "push-live-w4a-" + tenantID,
		ActorID:        "owner-bs-1",
		ActorType:      "human",
		ActorRole:      auth.SupplierRoleOwner,
	})
	if err != nil {
		t.Fatalf("app.Connectors.Push(live Spanner): %v", err)
	}
	if pushOut.PushResult.ExternalDocID != "BLZ-LIVE-W4A" {
		t.Fatalf("unexpected ExternalDocID: %+v", pushOut.PushResult)
	}
	// Verify JSON contains zero floating-point tokens on PushResponse
	b, _ := json.Marshal(pushOut)
	if strings.Contains(string(b), "live-w4a-billz-token") {
		t.Fatalf("PushResponse leaked raw secret token")
	}
}
