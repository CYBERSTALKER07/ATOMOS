package billing

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"google.golang.org/api/iterator"
)

func cleanupBillingGLTestState(t *testing.T, client *spanner.Client, supplierID string) {
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
			{"SELECT FeeScheduleId FROM BillingFeeSchedules WHERE SupplierId = @sid", "BillingFeeSchedules", []string{"FeeScheduleId"}},
			{"SELECT EventId FROM BillingMeterEvents WHERE SupplierId = @sid", "BillingMeterEvents", []string{"EventId"}},
			{"SELECT SupplierId, ShardId FROM BillingSupplierMeters WHERE SupplierId = @sid", "BillingSupplierMeters", []string{"SupplierId", "ShardId"}},
			{"SELECT EventId FROM OutboxEvents WHERE SupplierId = @sid", "OutboxEvents", []string{"EventId"}},
		} {
			var muts []*spanner.Mutation
			it := client.Single().Query(ctx, spanner.Statement{SQL: q.sql, Params: map[string]any{"sid": supplierID}})
			for {
				row, err := it.Next()
				if err != nil {
					break
				}
				vals := make([]*spanner.GenericColumnValue, len(q.cols))
				ptrs := make([]any, len(q.cols))
				for i := range q.cols {
					var sv spanner.GenericColumnValue
					vals[i] = &sv
					ptrs[i] = &sv
				}
				if row.Columns(ptrs...) == nil {
					k := make(spanner.Key, len(q.cols))
					for i, gv := range vals {
						if gv.Type.Code == 2 {
							var iv int64
							_ = gv.Decode(&iv)
							k[i] = iv
						} else {
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

func TestMeterWorker_PerOrderFee_GLDualWriteAndIdempotency_LiveSpanner(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	suffix := uuid.NewString()[:8]
	supID := "sup_bil_gl_" + suffix
	ordID := "ord_bil_gl_" + suffix
	feeID := "fee_bil_gl_" + suffix
	now := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond)
	cleanupBillingGLTestState(t, client, supID)

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("BillingFeeSchedules", map[string]any{
			"FeeScheduleId":            feeID,
			"SupplierId":               supID,
			"Tier":                     "STANDARD",
			"PerOrderMinor":            int64(3500),
			"GmvBps":                   int64(0),
			"MonthlySubscriptionMinor": int64(0),
			"Currency":                 "UZS",
			"EffectiveFrom":            now.Add(-time.Hour),
			"CreatedAt":                now,
		}),
	}); err != nil {
		t.Fatalf("seed BillingFeeSchedules: %v", err)
	}

	w := NewMeterWorker(client)
	if err := w.ProcessOrderFinalized(ctx, ordID, 120000, supID); err != nil {
		t.Fatalf("ProcessOrderFinalized: %v", err)
	}
	// Replay must be idempotent for both meter and GL.
	if err := w.ProcessOrderFinalized(ctx, ordID, 120000, supID); err != nil {
		t.Fatalf("ProcessOrderFinalized replay: %v", err)
	}

	glSvc := gl.NewService(client)
	periodID := gl.ComputePeriodID(ctx, supID, "UZ", time.Now().UTC())
	tb, err := glSvc.TrialBalance(ctx, supID, periodID)
	if err != nil {
		t.Fatalf("gl.TrialBalance: %v", err)
	}
	if !tb.Balanced || tb.TotalDebitsMinor != 3500 || tb.TotalCreditsMinor != 3500 {
		t.Fatalf("billing per-order fee GL mismatch: %+v", tb)
	}

	chain, err := gl.VerifyEntityHashChain(ctx, client, supID)
	if err != nil || !chain.Valid || chain.EntriesCount != 1 {
		t.Fatalf("VerifyEntityHashChain got (%+v, %v), want Valid=true EntriesCount=1", chain, err)
	}

	var glOutbox int64
	it := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM OutboxEvents WHERE SupplierId = @sid AND AggregateType = @agg AND TopicName = @topic`,
		Params: map[string]any{"sid": supID, "agg": events.AggregateJournalEntry, "topic": events.TopicLedger},
	})
	defer it.Stop()
	if r, err := it.Next(); err == nil {
		_ = r.Column(0, &glOutbox)
	} else if err != iterator.Done {
		t.Fatalf("query gl outbox: %v", err)
	}
	if glOutbox != 1 {
		t.Fatalf("gl outbox count = %d, want 1", glOutbox)
	}
}
