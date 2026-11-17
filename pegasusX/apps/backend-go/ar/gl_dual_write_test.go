package ar

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openARSpannerTestClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live AR GL Spanner test")
	}

	envOrDefault := func(k, fb string) string {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
		return fb
	}
	dbPath := fmt.Sprintf(
		"projects/%s/instances/%s/databases/%s",
		envOrDefault("SPANNER_PROJECT", "pegasusx-local"),
		envOrDefault("SPANNER_INSTANCE", "pegasusx-instance"),
		envOrDefault("SPANNER_DATABASE", "pegasusx-db"),
	)

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
	if _, probeErr := client.Single().ReadRow(ctx, "ArLedgerEntries", spanner.Key{"__ar_gl_probe__"}, []string{"EntryId"}); probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		t.Fatalf("spanner ArLedgerEntries probe failed: %v", probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

func cleanupARGLTestState(t *testing.T, client *spanner.Client, supplierID string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, q := range []struct {
			sql   string
			table string
			cols  []string
		}{
			{"SELECT EntityId, EntryId, Line FROM JournalPostings WHERE EntityId = @sid", "JournalPostings", []string{"EntityId", "EntryId", "Line"}},
			{"SELECT EntityId, EntryId FROM JournalEntries WHERE EntityId = @sid", "JournalEntries", []string{"EntityId", "EntryId"}},
			{"SELECT EntityId, PeriodId, AccountCode, PartyKey, Shard FROM GlBalances WHERE EntityId = @sid", "GlBalances", []string{"EntityId", "PeriodId", "AccountCode", "PartyKey", "Shard"}},
			{"SELECT EntryId FROM ArLedgerEntries WHERE SupplierId = @sid", "ArLedgerEntries", []string{"EntryId"}},
			{"SELECT InvoiceId FROM ArInvoices WHERE SupplierId = @sid", "ArInvoices", []string{"InvoiceId"}},
			{"SELECT RetailerId, SupplierId FROM ArAdvanceBalances WHERE SupplierId = @sid", "ArAdvanceBalances", []string{"RetailerId", "SupplierId"}},
			{"SELECT EventId FROM OutboxEvents WHERE SupplierId = @sid", "OutboxEvents", []string{"EventId"}},
		} {
			var muts []*spanner.Mutation
			it := client.Single().Query(ctx, spanner.Statement{SQL: q.sql, Params: map[string]any{"sid": supplierID}})
			for {
				row, err := it.Next()
				if err != nil {
					break
				}
				vals := make([]any, len(q.cols))
				ptrs := make([]any, len(q.cols))
				for i := range q.cols {
					var sv spanner.GenericColumnValue
					ptrs[i] = &sv
					vals[i] = &sv
				}
				if err := row.Columns(ptrs...); err == nil {
					k := make(spanner.Key, len(q.cols))
					for i := range q.cols {
						gv := vals[i].(*spanner.GenericColumnValue)
						switch gv.Type.Code {
						case 2: // INT64
							var iv int64
							_ = gv.Decode(&iv)
							k[i] = iv
						default:
							var s string
							_ = gv.Decode(&s)
							k[i] = s
						}
					}
					muts = append(muts, spanner.Delete(q.table, k))
				}
			}
			it.Stop()
			if len(muts) > 0 {
				_, _ = client.Apply(ctx, muts)
			}
		}
	})
}

func TestNormalizeARPaymentMode(t *testing.T) {
	cases := map[string]string{
		"":              "BANK",
		"bank_transfer": "BANK",
		"TRANSFER":      "TRANSFER",
		"cash_office":   "CASH_OFFICE",
		"CASH":          "CASH",
		"global_pay":    "CARD",
		"GLOBALPAY":     "CARD",
		"gp":            "CARD",
		"payme":         "PAYME",
		"click":         "CLICK",
		"adyen":         "ADYEN",
		"wallet":        "WALLET",
	}
	for in, want := range cases {
		if got := normalizeARPaymentMode(in); got != want {
			t.Fatalf("normalizeARPaymentMode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInferARPaymentModeFromKey(t *testing.T) {
	cases := map[string]string{
		"ar-cash-collect-ord123": "CASH",
		"ar-card-settle-ord123":  "CARD",
		"collect_cash_ord1":      "CASH",
		"cash:ord1":              "CASH",
		"card:ord1":              "CARD",
		"global-pay-1":           "CARD",
		"cash_office:1":          "CASH_OFFICE",
		"cash-office-1":          "CASH_OFFICE",
		"payme-tx-1":             "PAYME",
		"click-tx-1":             "CLICK",
		"adyen-tx-1":             "ADYEN",
		"wallet-tx-1":            "WALLET",
		"pay:sup:ret:inv:k1":     "BANK",
	}
	for in, want := range cases {
		if got := inferARPaymentModeFromKey(in); got != want {
			t.Fatalf("inferARPaymentModeFromKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestArLedgerEntries_AppendOnlyCollisionAndIdempotentRetry_LiveSpanner(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	client := openARSpannerTestClient(t)
	ctx := context.Background()
	sid := "sup_ar_coll_" + uuid.NewString()[:12]
	rid := "ret_ar_coll_" + uuid.NewString()[:12]
	cleanupARGLTestState(t, client, sid)

	now := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond)
	svc := NewService(NewSpannerRepository(client))
	svc.SetNow(func() time.Time { return now })

	// Pre-seed ArLedgerEntries row matching synthetic open key without ArInvoices row
	// to prove InsertMap fails closed with AlreadyExists when pre-read doesn't match ArInvoices.
	colOrder := "ord_col_" + uuid.NewString()[:12]
	colEntryID := ledgerEntryID("open", colOrder)
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("ArLedgerEntries", map[string]any{
			"EntryId":        colEntryID,
			"InvoiceId":      "ari_seed",
			"SupplierId":     sid,
			"RetailerId":     rid,
			"EntryType":      "OPEN",
			"AmountMinor":    int64(77777),
			"IdempotencyKey": "different-key-" + colOrder,
			"RefOrderId":     colOrder,
			"CreatedAt":      now,
		}),
	}); err != nil {
		t.Fatalf("seed colliding ArLedgerEntries: %v", err)
	}

	_, err := svc.OpenFromCreditLeave(ctx, OpenFromCreditLeaveRequest{
		SupplierID: sid, RetailerID: rid, OrderID: colOrder,
		AmountMinor: 99999, Currency: "UZS", CreditLeaveAt: now, TermsDays: 7,
	})
	if spanner.ErrCode(err) != codes.AlreadyExists {
		t.Fatalf("OpenFromCreditLeave on direct EntryId collision err = %v (code=%v), want AlreadyExists", err, spanner.ErrCode(err))
	}

	// Verify seeded row was NOT overwritten.
	row, rErr := client.Single().ReadRow(ctx, "ArLedgerEntries", spanner.Key{colEntryID}, []string{"AmountMinor"})
	if rErr != nil {
		t.Fatalf("read seeded ArLedgerEntries: %v", rErr)
	}
	var gotAmt int64
	_ = row.Column(0, &gotAmt)
	if gotAmt != 77777 {
		t.Fatalf("seeded ArLedgerEntries overwritten: got %d, want 77777", gotAmt)
	}
}

func TestAR_GLDualWrite_FIFOAdvanceWaiveVoidAndParity_LiveSpanner(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	client := openARSpannerTestClient(t)
	ctx := context.Background()
	sid := "sup_ar_gl_" + uuid.NewString()[:12]
	rid := "ret_ar_gl_" + uuid.NewString()[:12]
	cleanupARGLTestState(t, client, sid)

	now := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond)
	svc := NewService(NewSpannerRepository(client))
	svc.SetNow(func() time.Time { return now })
	glSvc := gl.NewService(client)

	ord1 := "ord_ar1_" + uuid.NewString()[:8]
	ord2 := "ord_ar2_" + uuid.NewString()[:8]
	ord3 := "ord_ar3_" + uuid.NewString()[:8]
	inv1, err := svc.OpenFromCreditLeave(ctx, OpenFromCreditLeaveRequest{
		SupplierID: sid, RetailerID: rid, OrderID: ord1,
		AmountMinor: 100000, Currency: "UZS", CreditLeaveAt: now.Add(-48 * time.Hour), TermsDays: 14,
	})
	if err != nil {
		t.Fatalf("open inv1: %v", err)
	}
	inv2, err := svc.OpenFromCreditLeave(ctx, OpenFromCreditLeaveRequest{
		SupplierID: sid, RetailerID: rid, OrderID: ord2,
		AmountMinor: 50000, Currency: "UZS", CreditLeaveAt: now.Add(-24 * time.Hour), TermsDays: 14,
	})
	if err != nil {
		t.Fatalf("open inv2: %v", err)
	}

	// Idempotent retry of OpenFromCreditLeave must succeed cleanly via pre-read without error.
	inv1Dup, err := svc.OpenFromCreditLeave(ctx, OpenFromCreditLeaveRequest{
		SupplierID: sid, RetailerID: rid, OrderID: ord1,
		AmountMinor: 100000, Currency: "UZS", CreditLeaveAt: now.Add(-48 * time.Hour), TermsDays: 14,
	})
	if err != nil || inv1Dup.InvoiceID != inv1.InvoiceID {
		t.Fatalf("idempotent OpenFromCreditLeave got (%v, %v), want %s", inv1Dup, err, inv1.InvoiceID)
	}

	// Apply FIFO 160k payment across inv1(100k) + inv2(50k) + advance credit(10k) using GLOBAL_PAY normalization.
	fifoKey := "fifo-gl-" + uuid.NewString()[:8]
	resp, err := svc.ApplyPaymentFIFO(ctx, PayFIFORequest{
		SupplierID:     sid,
		RetailerID:     rid,
		AmountMinor:    160000,
		Currency:       "UZS",
		PaymentMethod:  "GLOBAL_PAY",
		IdempotencyKey: fifoKey,
	})
	if err != nil {
		t.Fatalf("ApplyPaymentFIFO: %v", err)
	}
	if resp.PaidMinor != 150000 || resp.AdvanceAddedMinor != 10000 || len(resp.Allocations) != 2 {
		t.Fatalf("unexpected FIFO resp: %+v", resp)
	}
	_ = inv2

	// Replay same FIFO key idempotently — must not duplicate GL entries.
	if _, err := svc.ApplyPaymentFIFO(ctx, PayFIFORequest{
		SupplierID: sid, RetailerID: rid, AmountMinor: 160000, Currency: "UZS", PaymentMethod: "GLOBAL_PAY", IdempotencyKey: fifoKey,
	}); err != nil {
		t.Fatalf("idempotent ApplyPaymentFIFO retry: %v", err)
	}

	// Open inv3 (80k), waive 20k (80k -> 60k), then void remaining 60k.
	inv3, err := svc.OpenFromCreditLeave(ctx, OpenFromCreditLeaveRequest{
		SupplierID: sid, RetailerID: rid, OrderID: ord3,
		AmountMinor: 80000, Currency: "UZS", CreditLeaveAt: now, TermsDays: 14,
	})
	if err != nil {
		t.Fatalf("open inv3: %v", err)
	}
	err = spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		delta, err := AdjustInvoiceBalanceForOrderInTxn(ctx, txn, ord3, 80000, 60000, "adj-gl-"+ord3)
		if err != nil {
			return err
		}
		if delta != -20000 {
			return fmt.Errorf("expected delta -20000, got %d", delta)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("AdjustInvoiceBalanceForOrderInTxn: %v", err)
	}
	if err := svc.repo.VoidInvoice(ctx, inv3.InvoiceID, "void-gl-"+ord3); err != nil {
		t.Fatalf("VoidInvoice: %v", err)
	}

	// Verify GL Trial Balance & Entity Hash Chain.
	periodID := gl.ComputePeriodID(ctx, sid, "UZ", now)
	tb, err := glSvc.TrialBalance(ctx, sid, periodID)
	if err != nil {
		t.Fatalf("gl.TrialBalance: %v", err)
	}
	if !tb.Balanced || tb.TotalDebitsMinor != 240000 || tb.TotalCreditsMinor != 240000 {
		t.Fatalf("AR GL trial balance mismatch: %+v", tb)
	}
	chain, err := gl.VerifyEntityHashChain(ctx, client, sid)
	if err != nil || !chain.Valid || chain.EntriesCount != 5 {
		t.Fatalf("VerifyEntityHashChain got (%+v, %v), want Valid=true EntriesCount=5", chain, err)
	}

	// Verify OutboxEvents emitted JournalEntry events on TopicLedger.
	var glOutbox int64
	rowOut := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM OutboxEvents WHERE SupplierId = @sid AND AggregateType = @agg AND TopicName = @topic`,
		Params: map[string]any{"sid": sid, "agg": events.AggregateJournalEntry, "topic": events.TopicLedger},
	})
	defer rowOut.Stop()
	if r, err := rowOut.Next(); err == nil {
		_ = r.Column(0, &glOutbox)
	} else if err != iterator.Done {
		t.Fatalf("query gl outbox: %v", err)
	}
	if glOutbox != 5 {
		t.Fatalf("gl.entry.posted outbox count = %d, want 5", glOutbox)
	}
}

func TestAR_RetailerPayInvoiceAndOrderCashCardModes_LiveSpanner(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	client := openARSpannerTestClient(t)
	ctx := context.Background()
	suffix := uuid.NewString()[:8]
	sid := "sup_ar_modes_" + suffix
	rid := "ret_ar_modes_" + suffix
	cleanupARGLTestState(t, client, sid)

	now := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond)
	svc := NewService(NewSpannerRepository(client))
	svc.SetNow(func() time.Time { return now })
	glSvc := gl.NewService(client)

	openInv := func(ord string, amt int64) Invoice {
		inv, err := svc.OpenFromCreditLeave(ctx, OpenFromCreditLeaveRequest{
			SupplierID: sid, RetailerID: rid, OrderID: ord,
			AmountMinor: amt, Currency: "UZS", CreditLeaveAt: now, TermsDays: 14,
		})
		if err != nil {
			t.Fatalf("OpenFromCreditLeave(%s): %v", ord, err)
		}
		return inv
	}

	invCard := openInv("ord_mc_card_"+suffix, 10000)
	invCash := openInv("ord_mc_cash_"+suffix, 20000)
	invOffice := openInv("ord_mc_off_"+suffix, 30000)
	invWallet := openInv("ord_mc_wal_"+suffix, 40000)
	ordCash := "ord_mc_cc_" + suffix
	ordCard := "ord_mc_cs_" + suffix
	_ = openInv(ordCash, 15000)
	_ = openInv(ordCard, 25000)

	if _, err := svc.RetailerPayInvoice(ctx, rid, invCard.InvoiceID, RetailerPayInvoiceRequest{
		AmountMinor: 10000, PaymentMethod: "CARD",
	}, "rp-card-"+suffix); err != nil {
		t.Fatalf("RetailerPayInvoice(CARD): %v", err)
	}
	if _, err := svc.RetailerPayInvoice(ctx, rid, invCash.InvoiceID, RetailerPayInvoiceRequest{
		AmountMinor: 20000, PaymentMethod: "CASH",
	}, "rp-cash-"+suffix); err != nil {
		t.Fatalf("RetailerPayInvoice(CASH): %v", err)
	}
	if _, err := svc.RetailerPayInvoice(ctx, rid, invOffice.InvoiceID, RetailerPayInvoiceRequest{
		AmountMinor: 30000, PaymentMethod: "CASH_OFFICE",
	}, "rp-off-"+suffix); err != nil {
		t.Fatalf("RetailerPayInvoice(CASH_OFFICE): %v", err)
	}
	if _, err := svc.RetailerPayInvoice(ctx, rid, invWallet.InvoiceID, RetailerPayInvoiceRequest{
		AmountMinor: 40000, PaymentMethod: "WALLET",
	}, "rp-wal-"+suffix); err != nil {
		t.Fatalf("RetailerPayInvoice(WALLET): %v", err)
	}

	if err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return svc.RecordPaymentForOrderInTxn(ctx, txn, ordCash, 15000, "ar-cash-collect-"+ordCash, "UZS")
	}); err != nil {
		t.Fatalf("RecordPaymentForOrderInTxn(cash): %v", err)
	}
	if err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return svc.RecordPaymentForOrderInTxn(ctx, txn, ordCard, 25000, "ar-card-settle-"+ordCard, "UZS")
	}); err != nil {
		t.Fatalf("RecordPaymentForOrderInTxn(card): %v", err)
	}

	periodID := gl.ComputePeriodID(ctx, sid, "UZ", now)
	tb, err := glSvc.TrialBalance(ctx, sid, periodID)
	if err != nil {
		t.Fatalf("gl.TrialBalance: %v", err)
	}
	if !tb.Balanced || tb.TotalDebitsMinor != 140000 || tb.TotalCreditsMinor != 140000 {
		t.Fatalf("trial balance mismatch: %+v", tb)
	}
	byAcct := make(map[string]gl.TrialBalanceLine, len(tb.Accounts))
	for _, a := range tb.Accounts {
		byAcct[a.AccountCode] = a
	}
	if byAcct[gl.AccountBankOperating].DebitMinor != 0 || byAcct[gl.AccountBankOperating].CreditMinor != 0 {
		t.Fatalf("expected 0 posted to BANK:OPERATING, got %+v (accounts=%+v)", byAcct[gl.AccountBankOperating], byAcct)
	}
	if byAcct[gl.AccountCardPSPTransit].NetMinor != 35000 {
		t.Fatalf("PSP:GATEWAY = %+v, want NetMinor=35000 (10k CARD + 25k ar-card-settle)", byAcct[gl.AccountCardPSPTransit])
	}
	if byAcct[gl.AccountCashInTransit].NetMinor != 35000 {
		t.Fatalf("CASH:DRIVER = %+v, want NetMinor=35000 (20k CASH + 15k ar-cash-collect)", byAcct[gl.AccountCashInTransit])
	}
	if byAcct[gl.AccountCashOffice].NetMinor != 30000 {
		t.Fatalf("CASH:OFFICE = %+v, want NetMinor=30000", byAcct[gl.AccountCashOffice])
	}
	if byAcct[gl.AccountRetailerCredit].DebitMinor != 40000 || byAcct[gl.AccountRetailerCredit].CreditMinor != 0 || byAcct[gl.AccountRetailerCredit].NetMinor != -40000 {
		t.Fatalf("WALLET:RETAILER = %+v, want DebitMinor=40000 NetMinor=-40000", byAcct[gl.AccountRetailerCredit])
	}
	if byAcct[gl.AccountAccountsReceivable].NetMinor != -140000 {
		t.Fatalf("AR:RETAILER = %+v, want NetMinor=-140000", byAcct[gl.AccountAccountsReceivable])
	}

	chain, err := gl.VerifyEntityHashChain(ctx, client, sid)
	if err != nil || !chain.Valid || chain.EntriesCount != 6 {
		t.Fatalf("VerifyEntityHashChain got (%+v, %v), want Valid=true EntriesCount=6", chain, err)
	}
}
