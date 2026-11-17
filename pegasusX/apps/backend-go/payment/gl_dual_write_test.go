package payment

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

func cleanupPaymentGLState(t *testing.T, client *spanner.Client, supplierID string) {
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
			{"SELECT LedgerEntryId FROM PaymentLedgerEntries WHERE SupplierId = @sid", "PaymentLedgerEntries", []string{"LedgerEntryId"}},
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
						case 2:
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

func TestPayment_GLDualWrite_SettlementChargebackAndShortfall_LiveSpanner(t *testing.T) {
	client := openLedgerSpannerTestClient(t)
	ctx := context.Background()
	sid := "sup_pay_gl_" + uuid.NewString()[:12]
	rid := "ret_pay_gl_" + uuid.NewString()[:12]
	cleanupPaymentGLState(t, client, sid)

	now := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond)
	repo := NewSpannerRepository(client)

	// 1. Payout settlement journal dual-write (gross 100k, fee 5k).
	batchID := "batch_gl_" + uuid.NewString()[:8]
	if err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return WriteSettlementJournal(txn, batchID, sid, "UZS", 100000, 5000)
	}); err != nil {
		t.Fatalf("WriteSettlementJournal: %v", err)
	}

	// 2. Claim chargeback dual-write (30k) + idempotent replay check.
	cb := ChargebackRecord{
		ChargebackID: "cb_gl_" + uuid.NewString()[:8],
		OrderID:      "ord_pay_cb_" + uuid.NewString()[:8],
		SupplierID:   sid,
		RetailerID:   rid,
		Gateway:      "LEDGER",
		AmountMinor:  30000,
		Currency:     "UZS",
		CreatedAt:    now,
		Source:       "claims.settle_chargeback",
	}
	cleanupPaymentLedgerTestState(t, client, nil, nil, []string{cb.ChargebackID}, nil, nil, nil)
	if err := repo.SaveChargeback(ctx, cb, func(buf outbox.TxnBuffer) error {
		return nil
	}); err != nil {
		t.Fatalf("SaveChargeback: %v", err)
	}
	if err := repo.SaveChargeback(ctx, cb, nil); err != nil {
		t.Fatalf("SaveChargeback idempotent retry: %v", err)
	}

	// 3. Doorstep cash shortfall dual-write (expected 50k, received 40k).
	sfOrd := "ord_pay_sf_" + uuid.NewString()[:8]
	if err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return WriteDoorstepShortfall(ctx, txn, sfOrd, sid, "drv_pay_gl", "UZS", 50000, 40000, now)
	}); err != nil {
		t.Fatalf("WriteDoorstepShortfall: %v", err)
	}

	glSvc := gl.NewService(client)
	// Settlement used time.Now() inside WriteSettlementJournal, so check current period.
	periodID := gl.ComputePeriodID(ctx, sid, "UZ", time.Now().UTC())
	tb, err := glSvc.TrialBalance(ctx, sid, periodID)
	if err != nil {
		t.Fatalf("gl.TrialBalance: %v", err)
	}
	if !tb.Balanced || tb.TotalDebitsMinor != 180000 || tb.TotalCreditsMinor != 180000 {
		t.Fatalf("payment GL trial balance mismatch: %+v", tb)
	}

	chain, err := gl.VerifyEntityHashChain(ctx, client, sid)
	if err != nil || !chain.Valid || chain.EntriesCount != 3 {
		t.Fatalf("VerifyEntityHashChain got (%+v, %v), want Valid=true EntriesCount=3", chain, err)
	}

	var glOutbox int64
	it := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM OutboxEvents WHERE SupplierId = @sid AND AggregateType = @agg AND TopicName = @topic`,
		Params: map[string]any{"sid": sid, "agg": events.AggregateJournalEntry, "topic": events.TopicLedger},
	})
	defer it.Stop()
	if r, err := it.Next(); err == nil {
		_ = r.Column(0, &glOutbox)
	} else if err != iterator.Done {
		t.Fatalf("query gl outbox: %v", err)
	}
	if glOutbox != 3 {
		t.Fatalf("gl.entry.posted outbox count = %d, want 3", glOutbox)
	}

	// 4. WriteDoorstepShortfall must NOT post gl:shortfall: if gl:order_delivered: already exists.
	preDelivOrd := "ord_pay_predeliv_" + uuid.NewString()[:8]
	if _, err := glSvc.PostEvent(ctx, gl.RuleEventOrderDelivered, gl.RuleEvalContext{
		EntityID: sid, SupplierID: sid, OrderID: preDelivOrd, SourceID: preDelivOrd,
		Currency: "UZS", PaymentMode: "CREDIT", NetMinor: 30000, GrossMinor: 30000,
		IdempotencyKey: "gl:order_delivered:" + preDelivOrd, OccurredAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("PostEvent(order_delivered): %v", err)
	}
	if err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return WriteDoorstepShortfall(ctx, txn, preDelivOrd, sid, "drv_pay_gl", "UZS", 30000, 20000, now)
	}); err != nil {
		t.Fatalf("WriteDoorstepShortfall(after order_delivered): %v", err)
	}
	tbAfter, err := glSvc.TrialBalance(ctx, sid, periodID)
	if err != nil {
		t.Fatalf("gl.TrialBalance(after preDeliv): %v", err)
	}
	if !tbAfter.Balanced || tbAfter.TotalDebitsMinor != 210000 {
		t.Fatalf("expected balanced 210k without gl:shortfall:, got %+v", tbAfter)
	}
	chainAfter, err := gl.VerifyEntityHashChain(ctx, client, sid)
	if err != nil || !chainAfter.Valid || chainAfter.EntriesCount != 4 {
		t.Fatalf("expected 4 entries (no gl:shortfall:), got (%+v, %v)", chainAfter, err)
	}
}
