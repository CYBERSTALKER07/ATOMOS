package esf_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/esf"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/fiscal"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
	"github.com/pegasusx/pegasusx/apps/backend-go/soliq"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func openESFSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner ESF test")
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
		t.Fatalf("open spanner client (%s): %v", dbPath, err)
	}
	t.Cleanup(client.Close)
	return client
}

func envOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func TestESF_FailClosedWhenUnconfigured(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := esf.NewMemoryRepository()
	repo.SeedOrderFiscalSource(esf.OrderFiscalSource{
		OrderID:         "ord-fail-1",
		SupplierID:      "sup-fail",
		RetailerID:      "ret-1",
		CounterpartyTIN: "300111222",
		TotalMinor:      112_000,
		TotalNetMinor:   100_000,
		TotalVatMinor:   12_000,
		TotalGrossMinor: 112_000,
		DominantVatBps:  1200,
	})

	signer, err := fiscal.NewDevHMACSigner([]byte("test-esf-eds-hmac-secret-32-bytes!"))
	if err != nil {
		t.Fatalf("NewDevHMACSigner: %v", err)
	}

	// Missing signer -> fail closed with ErrEDSSignerNotConfigured.
	noSignerSvc := esf.NewService(esf.ServiceConfig{Repo: repo})
	if _, err := noSignerSvc.CreateOutbound(ctx, esf.CreateOutboundRequest{
		EntityID: "sup-fail",
		OrderID:  "ord-fail-1",
	}); !errors.Is(err, fiscal.ErrEDSSignerNotConfigured) {
		t.Fatalf("missing signer err = %v, want ErrEDSSignerNotConfigured", err)
	}

	// Signer present, SoliqClient absent -> fail closed with ErrSoliqGatewayNotConfigured.
	noClientSvc := esf.NewService(esf.ServiceConfig{
		Repo:   repo,
		Signer: signer,
	})
	if _, err := noClientSvc.CreateOutbound(ctx, esf.CreateOutboundRequest{
		EntityID: "sup-fail",
		OrderID:  "ord-fail-1",
	}); !errors.Is(err, fiscal.ErrSoliqGatewayNotConfigured) {
		t.Fatalf("missing soliq client err = %v, want ErrSoliqGatewayNotConfigured", err)
	}

	// Ensure zero fake documents were persisted.
	docs, err := repo.ListDocuments(ctx, esf.ListFilter{EntityID: "sup-fail"})
	if err != nil || len(docs) != 0 {
		t.Fatalf("expected 0 docs after fail-closed errors, got (%d, %v)", len(docs), err)
	}
}

func TestESF_Memory_SignSubmit_CircuitBreaker_Resend_RejectVATReversal_And_IKPU(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	repo := esf.NewMemoryRepository()
	now := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	periodID := gl.ComputePeriodID(ctx, "sup-m1", "UZ", now)

	repo.SeedOrderFiscalSource(esf.OrderFiscalSource{
		OrderID:          "ord-m1",
		SupplierID:       "sup-m1",
		RetailerID:       "ret-m1",
		SupplierTIN:      "200999888",
		CounterpartyTIN:  "300777666",
		CounterpartyName: "Korzinka Retail",
		Currency:         "UZS",
		TotalMinor:       112_000,
		CreatedAt:        now,
		TotalNetMinor:    100_000,
		TotalVatMinor:    12_000,
		TotalGrossMinor:  112_000,
		DominantVatBps:   1200,
		Lines: []esf.EsfCanonicalLine{
			{OrderLineID: "line-1", RegimeID: "reg-uz", VatRateBps: 1200, NetMinor: 100_000, VatMinor: 12_000, GrossMinor: 112_000},
		},
	})
	repo.SeedGLVATBalance("sup-m1", periodID, "UZS", 12_000, 6_000)

	var mu sync.Mutex
	var capturedPayloads []map[string]any
	var httpCalls atomic.Int32
	var return500 atomic.Bool
	return500.Store(true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := httpCalls.Add(1)
		raw, _ := io.ReadAll(r.Body)
		var decoded map[string]any
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		_ = dec.Decode(&decoded)
		mu.Lock()
		capturedPayloads = append(capturedPayloads, decoded)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if return500.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": map[string]any{"code": "SOLIQ_500", "message": "upstream outage"}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"ehf_id": fmt.Sprintf("ehf-ok-%d", n)},
		})
	}))
	defer srv.Close()

	signer, err := fiscal.NewDevHMACSigner([]byte("test-esf-eds-hmac-secret-32-bytes!"))
	if err != nil {
		t.Fatalf("NewDevHMACSigner: %v", err)
	}
	breaker := circuit.New("soliq-esf-test", circuit.Config{
		FailureThreshold: 2,
		OpenDuration:     60 * time.Millisecond,
	})
	sc := soliq.NewClientWithBreaker(soliq.SoliqConfig{
		BaseURL:  srv.URL,
		Operator: "didox",
		TIN:      "200999888",
		Timeout:  3 * time.Second,
	}, breaker)

	svc := esf.NewService(esf.ServiceConfig{
		Repo:        repo,
		SoliqClient: sc,
		Signer:      signer,
		Now:         func() time.Time { return now },
	})

	// 1. Outage triggers failure 1 & 2, opening circuit breaker and persisting FAILED doc.
	failedDoc1, err := svc.CreateOutbound(ctx, esf.CreateOutboundRequest{
		EntityID: "sup-m1", OrderID: "ord-m1", Operator: "didox", ActorID: "usr-fin",
	})
	if !errors.Is(err, esf.ErrUpstreamSubmission) || failedDoc1.Status != esf.StatusFailed {
		t.Fatalf("failedDoc1 = (%+v, %v), want FAILED + ErrUpstreamSubmission", failedDoc1, err)
	}
	_, err = svc.CreateOutbound(ctx, esf.CreateOutboundRequest{
		EntityID: "sup-m1", OrderID: "ord-m1", Operator: "didox",
	})
	if !errors.Is(err, esf.ErrUpstreamSubmission) {
		t.Fatalf("attempt 2 err = %v, want ErrUpstreamSubmission", err)
	}
	if httpCalls.Load() != 2 {
		t.Fatalf("expected 2 HTTP calls so far, got %d", httpCalls.Load())
	}

	// 3rd call while breaker is OPEN must NOT hit HTTP server.
	_, err = svc.Resend(ctx, "sup-m1", failedDoc1.EsfID, "didox", "usr-fin")
	if !errors.Is(err, esf.ErrUpstreamSubmission) {
		t.Fatalf("open-breaker resend err = %v, want ErrUpstreamSubmission", err)
	}
	if httpCalls.Load() != 2 {
		t.Fatalf("circuit breaker failed to short-circuit: httpCalls=%d, want 2", httpCalls.Load())
	}

	// 2. Recover Soliq & wait for half-open window -> Resend succeeds with EDS signature!
	return500.Store(false)
	time.Sleep(80 * time.Millisecond)

	submittedDoc, err := svc.Resend(ctx, "sup-m1", failedDoc1.EsfID, "didox", "usr-fin")
	if err != nil {
		t.Fatalf("Resend after recovery: %v", err)
	}
	if submittedDoc.Status != esf.StatusSubmitted || submittedDoc.OperatorRef == "" || submittedDoc.PayloadHash == "" {
		t.Fatalf("unexpected submitted doc: %+v", submittedDoc)
	}
	mu.Lock()
	lastSent := capturedPayloads[len(capturedPayloads)-1]
	mu.Unlock()
	if sig, _ := lastSent["signature"].(string); sig == "" {
		t.Fatalf("expected non-empty EDS signature envelope in submitted payload: %+v", lastSent)
	}

	// 3. Record inbound purchase ESF (input VAT = 6,000) and verify VATSummary DiffMinor == 0!
	if _, err := svc.CreateInbound(ctx, esf.CreateInboundRequest{
		EntityID:         "sup-m1",
		PoID:             "po-in-1",
		CounterpartyTIN:  "400555111",
		CounterpartyName: "Factory Supplier LLC",
		TotalMinor:       56_000,
		VatMinor:         6_000,
		VatRateBps:       1200,
		PeriodID:         periodID,
		ActorID:          "usr-fin",
	}); err != nil {
		t.Fatalf("CreateInbound: %v", err)
	}

	vatBeforeReject, err := svc.GetVATSummary(ctx, "sup-m1", periodID)
	if err != nil {
		t.Fatalf("GetVATSummary before reject: %v", err)
	}
	if !vatBeforeReject.Reconciled || vatBeforeReject.DiffMinor != 0 || vatBeforeReject.EsfNetVatMinor != 6_000 {
		t.Fatalf("VAT summary before reject = %+v, want Reconciled=true DiffMinor=0 Net=6000", vatBeforeReject)
	}

	// 4. Reject outbound ESF -> emits esf.rejected + triggers GL DR TAX:VAT:OUTPUT reversal (12,000) -> DiffMinor remains 0!
	rejRes, err := svc.TransitionStatus(ctx, "sup-m1", submittedDoc.EsfID, esf.TransitionStatusRequest{
		Action:  esf.ActionReject,
		Reason:  "wrong TIN on delivery",
		ActorID: "usr-fin",
	})
	if err != nil {
		t.Fatalf("TransitionStatus REJECT: %v", err)
	}
	if rejRes.Document.Status != esf.StatusRejected || rejRes.VatReversedMinor != 12_000 || rejRes.GlReversalEntryID == "" {
		t.Fatalf("unexpected reject result: %+v", rejRes)
	}
	if _, err := svc.TransitionStatus(ctx, "sup-m1", submittedDoc.EsfID, esf.TransitionStatusRequest{Action: esf.ActionAccept}); !errors.Is(err, esf.ErrInvalidTransition) {
		t.Fatalf("transition out of REJECTED err = %v, want ErrInvalidTransition", err)
	}

	vatAfterReject, err := svc.GetVATSummary(ctx, "sup-m1", periodID)
	if err != nil {
		t.Fatalf("GetVATSummary after reject: %v", err)
	}
	if !vatAfterReject.Reconciled || vatAfterReject.DiffMinor != 0 || vatAfterReject.RejectedCount != 1 || vatAfterReject.EsfOutputVatMinor != 0 || vatAfterReject.GlOutputVatMinor != 0 {
		t.Fatalf("VAT summary after reject = %+v, want output=0 gl_output=0 diff=0 rejected=1", vatAfterReject)
	}

	// Verify SyncOrderAcceptanceStatus propagates ACCEPT cleanly on another SUBMITTED ESF.
	doc2, err := svc.CreateOutbound(ctx, esf.CreateOutboundRequest{
		EntityID: "sup-m1", OrderID: "ord-m1", Operator: "direct",
	})
	if err != nil {
		t.Fatalf("CreateOutbound doc2: %v", err)
	}
	if err := svc.SyncOrderAcceptanceStatus(ctx, "", doc2.OperatorRef, "ACCEPTED", "", []byte(`{"status":"ACCEPTED"}`)); err != nil {
		t.Fatalf("SyncOrderAcceptanceStatus: %v", err)
	}
	detail2, err := svc.Get(ctx, "sup-m1", doc2.EsfID)
	if err != nil || detail2.Document.Status != esf.StatusAccepted {
		t.Fatalf("doc2 after SyncOrderAcceptanceStatus = (%+v, %v), want ACCEPTED", detail2, err)
	}

	// Verify CANCEL transition on a fresh SUBMITTED ESF.
	doc3, err := svc.CreateOutbound(ctx, esf.CreateOutboundRequest{
		EntityID: "sup-m1", OrderID: "ord-m1", Operator: "direct",
	})
	if err != nil {
		t.Fatalf("CreateOutbound doc3: %v", err)
	}
	canRes, err := svc.TransitionStatus(ctx, "sup-m1", doc3.EsfID, esf.TransitionStatusRequest{
		Action: esf.ActionCancel, Reason: "cancelled draft order",
	})
	if err != nil || canRes.Document.Status != esf.StatusCancelled {
		t.Fatalf("cancel doc3 = (%+v, %v), want CANCELLED", canRes, err)
	}

	// Verify emitted event types include submitted, rejected, accepted, cancelled.
	emitted := make(map[string]bool)
	for _, ev := range repo.EmittedEvents() {
		emitted[ev.EventType] = true
	}
	for _, wantEv := range []string{events.EventEsfSubmitted, events.EventEsfAccepted, events.EventEsfRejected, events.EventEsfCancelled} {
		if !emitted[wantEv] {
			t.Fatalf("missing emitted ESF event %q (emitted=%v)", wantEv, emitted)
		}
	}

	// 5. Missing IKPU audit checks compliant vs missing SKUs vs inactive SKUs.
	repo.SeedProductCompliance(esf.MemoryProductCompliance{
		ProductID: "sku-ok", SupplierID: "sup-m1", Name: "Sugar 1kg", Barcode: "4780001", IKPUCode: "01701001001000000", PackageCode: "11002", IsActive: true,
	})
	repo.SeedProductCompliance(esf.MemoryProductCompliance{
		ProductID: "sku-bad", SupplierID: "sup-m1", Name: "Tea 250g", Barcode: "", IKPUCode: "", PackageCode: "", IsActive: true,
	})
	repo.SeedProductCompliance(esf.MemoryProductCompliance{
		ProductID: "sku-inactive", SupplierID: "sup-m1", Name: "Archived SKU", IsActive: false,
	})
	ikpuResp, err := svc.ListMissingIKPU(ctx, "sup-m1")
	if err != nil {
		t.Fatalf("ListMissingIKPU: %v", err)
	}
	if ikpuResp.Count != 1 || ikpuResp.Items[0].ProductID != "sku-bad" || len(ikpuResp.Items[0].MissingFields) != 3 {
		t.Fatalf("unexpected ListMissingIKPU = %+v, want only sku-bad with 3 missing fields", ikpuResp)
	}
}

func TestESF_SpannerLiveE2E_Snapshots_GLVatSummary_And_IKPU(t *testing.T) {
	client := openESFSpannerClient(t)
	ctx := context.Background()

	suffix := uuid.NewString()[:8]
	supID := "sup-esf-" + suffix
	retID := "ret-esf-" + suffix
	ordID := "ord-esf-" + suffix
	regID := "reg-esf-" + suffix
	prodOK := "prod-ok-" + suffix
	prodMiss := "prod-miss-" + suffix
	now := time.Now().UTC()
	periodID := gl.ComputePeriodID(ctx, supID, "UZ", now)

	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, q := range []struct {
			tbl, col, val string
		}{
			{"EsfStatusHistory", "EntityId", supID},
			{"EsfDocuments", "EntityId", supID},
			{"JournalPostings", "EntityId", supID},
			{"JournalEntries", "EntityId", supID},
			{"GlBalances", "EntityId", supID},
			{"OutboxEvents", "SupplierId", supID},
			{"OntologyObjects", "SupplierId", supID},
			{"Products", "SupplierId", supID},
		} {
			_, _ = client.PartitionedUpdate(cleanCtx, spanner.Statement{
				SQL:    fmt.Sprintf("DELETE FROM %s WHERE %s = @v", q.tbl, q.col),
				Params: map[string]any{"v": q.val},
			})
		}
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("Orders", spanner.Key{ordID}),
			spanner.Delete("TaxRegimeVersions", spanner.Key{regID}),
			spanner.Delete("Retailers", spanner.Key{retID}),
			spanner.Delete("SupplierProfiles", spanner.Key{supID}),
		})
	})

	// Seed SupplierProfiles, Retailers, TaxRegimeVersions, Orders + OrderLineFiscalSnapshots, Products, OntologyObjects.
	skuOKProps, _ := json.Marshal(map[string]any{
		"product_id":   prodOK,
		"name":         "Water 1.5L",
		"barcode":      "478000999001",
		"ikpu_code":    "02201100001000000",
		"package_code": "123456",
	})
	skuMissProps, _ := json.Marshal(map[string]any{
		"product_id": prodMiss,
		"name":       "Juice 1L",
	})

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("SupplierProfiles", map[string]any{
			"SupplierId": supID, "TaxId": "201112223", "WarehouseName": "Central ESF WH",
			"RegisteredAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("Retailers", map[string]any{
			"RetailerId": retID, "Phone": "+99890" + suffix, "Name": "Havas Mart",
			"CountryCode": "UZ", "TaxId": "309888777", "CreatedAt": now,
		}),
		spanner.InsertMap("TaxRegimeVersions", map[string]any{
			"Id": regID, "CountryCode": "UZ", "EffectiveFrom": now.Add(-24 * time.Hour),
			"Currency": "UZS", "VatRatesBps": []int64{1200},
			"CreatedAt": now, "CreatedBy": "esf-test", "UpdatedAt": now,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ordID, "SupplierId": supID, "RetailerId": retID,
			"Status": "COMPLETED", "OrderSource": "MANUAL", "ConfirmationStatus": "CONFIRMED",
			"LineItemsJson": []byte(`[{"sku":"Water","qty":10}]`),
			"TotalMinor":    int64(224_000), "Currency": "UZS",
			"Version": int64(1), "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("OrderLineFiscalSnapshots", map[string]any{
			"OrderId": ordID, "OrderLineId": "line-esf-1", "RegimeId": regID,
			"VatRateBps": int64(1200), "NetMinor": int64(200_000), "VatMinor": int64(24_000), "GrossMinor": int64(224_000),
			"SnapshotAt": now, "CreatedAt": now,
		}),
		spanner.InsertMap("Products", map[string]any{
			"ProductId": prodOK, "SupplierId": supID, "CategoryId": "cat-1",
			"Name": "Water 1.5L", "Barcode": "478000999001", "PriceMinor": int64(22400),
			"Currency": "UZS", "IsActive": true, "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("Products", map[string]any{
			"ProductId": prodMiss, "SupplierId": supID, "CategoryId": "cat-1",
			"Name": "Juice 1L", "PriceMinor": int64(15000),
			"Currency": "UZS", "IsActive": true, "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("OntologyObjects", map[string]any{
			"CellId": "cell-uz", "ObjectType": "SKU", "ObjectId": prodOK, "SupplierId": supID,
			"Version": int64(1), "PropertiesJson": string(skuOKProps), "ValidFrom": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("OntologyObjects", map[string]any{
			"CellId": "cell-uz", "ObjectType": "SKU", "ObjectId": prodMiss, "SupplierId": supID,
			"Version": int64(1), "PropertiesJson": string(skuMissProps), "ValidFrom": now, "UpdatedAt": now,
		}),
	}); err != nil {
		t.Fatalf("seed live Spanner ESF fixtures: %v", err)
	}

	// Post GL entries matching order output VAT (24,000) and inbound purchase input VAT (6,000).
	glSvc := gl.NewService(client)
	if _, err := glSvc.PostEntry(ctx, gl.JournalEntryInput{
		EntityID: supID, SourceType: "ORDER_DELIVERED", SourceID: ordID,
		Memo: "Live ESF order VAT", Currency: "UZS", CountryPack: "UZ",
		IdempotencyKey: "esf-test-out-" + suffix, PostedBy: "esf-test", OccurredAt: now,
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountAccountsReceivable, Direction: gl.DirectionDebit, AmountMinor: 224_000, Currency: "UZS", PartyType: "RETAILER", PartyID: retID},
			{AccountCode: gl.AccountSalesRevenue, Direction: gl.DirectionCredit, AmountMinor: 200_000, Currency: "UZS"},
			{AccountCode: gl.AccountVatOutput, Direction: gl.DirectionCredit, AmountMinor: 24_000, Currency: "UZS"},
		},
	}); err != nil {
		t.Fatalf("seed GL output VAT: %v", err)
	}
	if _, err := glSvc.PostEntry(ctx, gl.JournalEntryInput{
		EntityID: supID, SourceType: "INVENTORY_RECEIPT", SourceID: "po-" + suffix,
		Memo: "Live ESF purchase VAT", Currency: "UZS", CountryPack: "UZ",
		IdempotencyKey: "esf-test-in-" + suffix, PostedBy: "esf-test", OccurredAt: now,
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountInventory, Direction: gl.DirectionDebit, AmountMinor: 50_000, Currency: "UZS"},
			{AccountCode: gl.AccountVatInput, Direction: gl.DirectionDebit, AmountMinor: 6_000, Currency: "UZS"},
			{AccountCode: gl.AccountVendorPayable, Direction: gl.DirectionCredit, AmountMinor: 56_000, Currency: "UZS"},
		},
	}); err != nil {
		t.Fatalf("seed GL input VAT: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"ehf_id": "ehf-live-" + suffix},
		})
	}))
	defer srv.Close()

	signer, err := fiscal.NewDevHMACSigner([]byte("live-spanner-esf-hmac-key-32bytes!"))
	if err != nil {
		t.Fatalf("NewDevHMACSigner: %v", err)
	}
	sc := soliq.NewClient(soliq.SoliqConfig{BaseURL: srv.URL, Operator: "direct", TIN: "201112223", Timeout: 5 * time.Second})
	svc := esf.NewService(esf.ServiceConfig{
		Repo:        esf.NewSpannerRepository(client),
		SoliqClient: sc,
		Signer:      signer,
		Now:         func() time.Time { return now },
	})

	// 1. Outbound ESF built from Orders + OrderLineFiscalSnapshots + TaxRegimeVersions.
	outDoc, err := svc.CreateOutbound(ctx, esf.CreateOutboundRequest{
		EntityID: supID, OrderID: ordID, Operator: "direct", ActorID: "usr-esf",
	})
	if err != nil {
		t.Fatalf("Spanner CreateOutbound: %v", err)
	}
	if outDoc.TotalMinor != 224_000 || outDoc.VatMinor != 24_000 || outDoc.VatRateBps != 1200 || outDoc.CounterpartyTIN != "309888777" || outDoc.Status != esf.StatusSubmitted {
		t.Fatalf("unexpected live outbound ESF: %+v", outDoc)
	}

	// 2. Inbound ESF -> VATSummary proves DiffMinor == 0 against GLBalances!
	if _, err := svc.CreateInbound(ctx, esf.CreateInboundRequest{
		EntityID: supID, PoID: "po-" + suffix, CounterpartyTIN: "200555444",
		TotalMinor: 56_000, VatMinor: 6_000, VatRateBps: 1200, PeriodID: periodID, ActorID: "usr-esf",
	}); err != nil {
		t.Fatalf("Spanner CreateInbound: %v", err)
	}

	vat1, err := svc.GetVATSummary(ctx, supID, periodID)
	if err != nil {
		t.Fatalf("Spanner GetVATSummary 1: %v", err)
	}
	if !vat1.Reconciled || vat1.DiffMinor != 0 || vat1.EsfOutputVatMinor != 24_000 || vat1.GlOutputVatMinor != 24_000 || vat1.EsfInputVatMinor != 6_000 || vat1.GlInputVatMinor != 6_000 {
		t.Fatalf("Spanner VATSummary 1 = %+v, want Reconciled=true DiffMinor=0", vat1)
	}

	// 3. Reject outbound ESF -> triggers atomic GL DR TAX:VAT:OUTPUT 24,000 reversal + esf.rejected on TopicLedger!
	rej, err := svc.TransitionStatus(ctx, supID, outDoc.EsfID, esf.TransitionStatusRequest{
		Action: esf.ActionReject, Reason: "buyer rejected quantity", ActorID: "usr-esf",
	})
	if err != nil {
		t.Fatalf("Spanner TransitionStatus REJECT: %v", err)
	}
	if rej.Document.Status != esf.StatusRejected || len(rej.History) != 2 || rej.VatReversedMinor != 24_000 || rej.GlReversalEntryID == "" {
		t.Fatalf("unexpected Spanner reject result: %+v", rej)
	}

	// VAT Summary STILL reconciles with DiffMinor == 0 after rejection because GL TAX:VAT:OUTPUT was debited 24,000!
	vat2, err := svc.GetVATSummary(ctx, supID, periodID)
	if err != nil {
		t.Fatalf("Spanner GetVATSummary 2: %v", err)
	}
	if !vat2.Reconciled || vat2.DiffMinor != 0 || vat2.EsfOutputVatMinor != 0 || vat2.GlOutputVatMinor != 0 || vat2.RejectedCount != 1 {
		t.Fatalf("Spanner VATSummary 2 = %+v, want Reconciled=true output=0 gl_output=0 diff=0 rejected=1", vat2)
	}

	// Verify OutboxEvents on TopicLedger contain esf.submitted and esf.rejected.
	iter := client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    `SELECT TopicName, Payload FROM OutboxEvents WHERE SupplierId = @eid`,
		Params: map[string]any{"eid": supID},
	})
	defer iter.Stop()
	esfTopics := make(map[string]string)
	for {
		row, rErr := iter.Next()
		if errors.Is(rErr, iterator.Done) {
			break
		}
		if rErr != nil {
			t.Fatalf("query OutboxEvents: %v", rErr)
		}
		var topic string
		var payloadBytes []byte
		if sErr := row.Columns(&topic, &payloadBytes); sErr != nil {
			t.Fatalf("scan OutboxEvent: %v", sErr)
		}
		var ev struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(payloadBytes, &ev)
		if strings.HasPrefix(ev.Type, "esf.") {
			esfTopics[ev.Type] = topic
		}
	}
	if esfTopics[events.EventEsfSubmitted] != events.TopicLedger || esfTopics[events.EventEsfRejected] != events.TopicLedger {
		t.Fatalf("missing TopicLedger ESF events: %v", esfTopics)
	}

	// 4. ListMissingIKPU on live Spanner returns only prodMiss (Juice 1L), excluding compliant Water 1.5L!
	ikpu, err := svc.ListMissingIKPU(ctx, supID)
	if err != nil {
		t.Fatalf("Spanner ListMissingIKPU: %v", err)
	}
	if ikpu.Count != 1 || ikpu.Items[0].ProductID != prodMiss {
		t.Fatalf("Spanner ListMissingIKPU = %+v, want only %s", ikpu, prodMiss)
	}
}
